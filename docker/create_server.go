package docker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	// gobblerRoot is the host directory that must be created as part of host
	// preparation (see README.md); each instance's outputDir is a subdirectory
	// of it, created by the running Gobbler instance itself — the Agent never
	// creates directories under it.
	gobblerRoot = "/gobbler"

	// readyPollInterval/readyPollAttempts bound how long CreateServer waits for a
	// freshly started Gobbler container to start responding to HTTP requests.
	readyPollInterval = 250 * time.Millisecond
	readyPollAttempts = 20
)

// containerPortSpec is gobblerContainerPort in Docker's "<port>/<protocol>" format,
// as used in ExposedPorts/PortBindings keys.
var containerPortSpec = fmt.Sprintf("%d/tcp", gobblerContainerPort)

// createContainerRequest is the JSON body for POST /containers/create.
type createContainerRequest struct {
	Image        string                    `json:"Image"`
	ExposedPorts map[string]struct{}       `json:"ExposedPorts"`
	HostConfig   createContainerHostConfig `json:"HostConfig"`
}

type createContainerHostConfig struct {
	PortBindings map[string][]portBinding `json:"PortBindings"`
	Binds        []string                 `json:"Binds,omitempty"`
}

type portBinding struct {
	HostPort string `json:"HostPort"`
}

/*
		CreateServer creates, starts and configures a new Gobbler container. It

	  - verifies that no container with instanceName is already running (returns ErrAlreadyExists if so)
	  - if mode == file, verifies that /gobbler exists on the host (returns ErrHostNotPrepared if not) and
	    that outputDir follows the host path convention /gobbler/<instanceName> — it never creates
	    directories itself; the running Gobbler instance creates its own outputDir subdirectory
	  - creates the container (POST /containers/create) with instanceName, -v bindings (file mode) and -p hostPort:8080
	  - starts the container (POST /containers/{id}/start)
	  - polls GET /gobbler/pipeline/status until the container responds
	  - configures the container (POST /gobbler/pipeline/configure) with the full config payload
	  - returns the populated InstanceRecord for the new instance
*/
func (c *Docker) CreateServer(config []byte, hostPort int) (InstanceRecord, error) {
	var req struct {
		InstanceName string `json:"instanceName"`
		Mode         string `json:"mode"`
		OutputDir    string `json:"outputDir"`
	}
	if err := json.Unmarshal(config, &req); err != nil {
		return InstanceRecord{}, fmt.Errorf("decoding config: %w", err)
	}
	if req.InstanceName == "" {
		return InstanceRecord{}, errors.New("instanceName is required")
	}
	if req.Mode == "file" {
		if info, err := os.Stat(gobblerRoot); err != nil || !info.IsDir() {
			return InstanceRecord{}, ErrHostNotPrepared
		}
		expected := gobblerRoot + "/" + req.InstanceName
		if req.OutputDir != expected {
			return InstanceRecord{}, fmt.Errorf("outputDir must be %s for file mode", expected)
		}
	}

	existingID, err := c.containerID(req.InstanceName)
	if err != nil {
		return InstanceRecord{}, err
	}
	if existingID != "" {
		return InstanceRecord{}, ErrAlreadyExists
	}

	createReq := createContainerRequest{
		Image:        gobblerImage,
		ExposedPorts: map[string]struct{}{containerPortSpec: {}},
		HostConfig: createContainerHostConfig{
			PortBindings: map[string][]portBinding{
				containerPortSpec: {{HostPort: strconv.Itoa(hostPort)}},
			},
		},
	}
	if req.Mode == "file" {
		createReq.HostConfig.Binds = []string{fmt.Sprintf("%s:%s", req.OutputDir, req.OutputDir)}
	}

	payload, err := json.Marshal(createReq)
	if err != nil {
		return InstanceRecord{}, fmt.Errorf("encoding create request: %w", err)
	}

	createURL := fmt.Sprintf("%s/containers/create?name=%s", c.baseURL, url.QueryEscape(req.InstanceName))
	containerID, err := c.createContainer(createURL, payload)
	if err != nil {
		return InstanceRecord{}, err
	}

	if err := c.startContainer(containerID); err != nil {
		return InstanceRecord{}, err
	}

	localURL := fmt.Sprintf("http://127.0.0.1:%d", hostPort)
	if err := waitForGobblerReady(localURL); err != nil {
		return InstanceRecord{}, fmt.Errorf("gobbler container did not become ready: %w", err)
	}

	configResp, err := c.client.Post(localURL+"/gobbler/pipeline/configure", "application/json", bytes.NewReader(config))
	if err != nil {
		return InstanceRecord{}, fmt.Errorf("configuring gobbler pipeline: %w", err)
	}
	defer configResp.Body.Close()
	if configResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(configResp.Body)
		return InstanceRecord{}, fmt.Errorf("unexpected status %d from /gobbler/pipeline/configure: %s", configResp.StatusCode, body)
	}

	return InstanceRecord{
		InstanceName: req.InstanceName,
		URL:          fmt.Sprintf("http://%s:%d", hostIP(), hostPort),
		HostPort:     hostPort,
		ContainerID:  containerID,
		Status:       "running",
	}, nil
}

// containerID returns the ID of the container with the given exact name
// (as created via /containers/create?name=<name>), or "" if none exists.
func (c *Docker) containerID(name string) (string, error) {
	filters := url.QueryEscape(fmt.Sprintf(`{"name":["%s"]}`, name))
	resp, err := c.client.Get(fmt.Sprintf("%s/containers/json?all=true&filters=%s", c.baseURL, filters))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: unexpected status %d from /containers/json", ErrDaemonUnreachable, resp.StatusCode)
	}

	var containers []struct {
		ID    string   `json:"Id"`
		Names []string `json:"Names"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return "", fmt.Errorf("decoding /containers/json response: %w", err)
	}

	// Docker's name filter does a substring match, so confirm an exact match
	// ourselves (container names are reported with a leading "/").
	want := "/" + name
	for _, ct := range containers {
		for _, n := range ct.Names {
			if n == want {
				return ct.ID, nil
			}
		}
	}
	return "", nil
}

// createContainer calls POST /containers/create and returns the new container's ID.
func (c *Docker) createContainer(createURL string, payload []byte) (string, error) {
	resp, err := c.client.Post(createURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return "", ErrAlreadyExists
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d from /containers/create: %s", resp.StatusCode, body)
	}

	var created struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("decoding /containers/create response: %w", err)
	}
	return created.ID, nil
}

// startContainer calls POST /containers/{id}/start.
func (c *Docker) startContainer(id string) error {
	resp, err := c.client.Post(fmt.Sprintf("%s/containers/%s/start", c.baseURL, id), "application/json", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d from /containers/%s/start: %s", resp.StatusCode, id, body)
	}
	return nil
}

// waitForGobblerReady polls a freshly started Gobbler container's
// /gobbler/pipeline/status endpoint until it responds, giving the process
// inside the container time to start listening.
func waitForGobblerReady(instanceURL string) error {
	client := &http.Client{Timeout: time.Second}
	var lastErr error
	for i := 0; i < readyPollAttempts; i++ {
		resp, err := client.Get(instanceURL + "/gobbler/pipeline/status")
		if err != nil {
			lastErr = err
		} else {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("unexpected status %d from /gobbler/pipeline/status", resp.StatusCode)
		}
		time.Sleep(readyPollInterval)
	}
	return lastErr
}

// hostIP returns the host machine's outbound IP address, used to build each
// instance's externally reachable URL. It relies on the OS routing table by
// "connecting" a UDP socket to a well-known address — no packets are actually
// sent, so this works even without real network connectivity. Falls back to
// "127.0.0.1" if the lookup fails (e.g. no network interfaces configured).
func hostIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
