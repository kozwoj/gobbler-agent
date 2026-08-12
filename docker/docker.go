package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Docker Engine API connection settings.
// The daemon is configured to listen on localhost only (see docker-REST.md) —
// no TLS/auth is needed because the traffic never leaves the host.
const (
	baseURL        = "http://127.0.0.1:2375"
	gobblerImage   = "gobbler:latest"
	requestTimeout = 5 * time.Second

	// gobblerContainerPort is the port the Gobbler image listens on inside the container (see Dockerfile).
	gobblerContainerPort = 8080
)

// Sentinel errors returned by Docker methods.
var (
	ErrNotFound          = errors.New("instance not found")
	ErrAlreadyExists     = errors.New("instance already exists")
	ErrDaemonUnreachable = errors.New("docker daemon unreachable")
	ErrImageNotFound     = errors.New("gobbler image not found")
	ErrHostNotPrepared   = errors.New("/gobbler directory not found on host; see host preparation steps in README.md")
)

// DaemonStatus is a projection of selected fields from Docker's GET /info response.
type DaemonStatus struct {
	ID                string `json:"id"`
	Containers        int    `json:"containers"`
	ContainersRunning int    `json:"containersRunning"`
	ContainersPaused  int    `json:"containersPaused"`
	ContainersStopped int    `json:"containersStopped"`
}

// InstanceRecord is a projection of a Docker container's state for a Gobbler instance.
type InstanceRecord struct {
	InstanceName string `json:"instanceName"`
	URL          string `json:"url"`
	HostPort     int    `json:"hostPort"`
	ContainerID  string `json:"containerId"`
	Status       string `json:"status"`
	// Config fields — populated for running containers only; zero-value when stopped
	// or when the Gobbler server inside the container is not yet configured.
	Mode            string `json:"mode,omitempty"`
	OutputDir       string `json:"outputDir,omitempty"`
	AccountName     string `json:"accountName,omitempty"`
	WriterQueueSize int    `json:"writerQueueSize,omitempty"`
	WriterBatchSize int    `json:"writerBatchSize,omitempty"`
}

/* Docker is a wrapper class that implements operations exposed by Agent
endpoints. It uses REST Docker Engine APIs to manage Gobbler containers
- Create + start + configure a new Gobbler instances
- List Gobbler instances with their names and URLs
- Get a single Gobbler instance's details
- Stop and remove a Gobbler instance container
- Return Agent's health status (Docker daemon + Gobbler container health)
*/

type Docker struct {
	client    *http.Client
	baseURL   string // base URL for Docker Engine API
	startTime string // time when the Docker wrapper was created
}

/*
	New creates a new Docker wrapper instance. At minimum it

- verifies that the Docker daemon is running and is accessible on port 2375 (or 2376 for TLS)
- verifies that the Gobbler image is available

Note: we are assuming that part of the host configuration is to start
Docker daemon and pull the Gobbler image.
*/
func New() (*Docker, error) {
	c := &Docker{
		client:    &http.Client{Timeout: requestTimeout},
		baseURL:   baseURL,
		startTime: time.Now().Format(time.RFC3339),
	}

	if err := c.checkDaemon(); err != nil {
		return nil, err
	}
	if err := c.checkImage(); err != nil {
		return nil, err
	}

	return c, nil
}

// checkDaemon verifies that the Docker daemon is reachable by calling GET /info.
func (c *Docker) checkDaemon() error {
	resp, err := c.client.Get(c.baseURL + "/info")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: unexpected status %d from /info", ErrDaemonUnreachable, resp.StatusCode)
	}
	return nil
}

// checkImage verifies that the Gobbler image has been loaded/pulled on the host
// by calling GET /images/json filtered to the gobbler:latest reference.
func (c *Docker) checkImage() error {
	filters := url.QueryEscape(fmt.Sprintf(`{"reference":["%s"]}`, gobblerImage))
	resp, err := c.client.Get(fmt.Sprintf("%s/images/json?filters=%s", c.baseURL, filters))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: unexpected status %d from /images/json", ErrDaemonUnreachable, resp.StatusCode)
	}

	var images []struct {
		RepoTags []string `json:"RepoTags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		return fmt.Errorf("decoding /images/json response: %w", err)
	}

	if len(images) == 0 {
		return fmt.Errorf("%w: %s (run `docker load` or `docker pull` on the host)", ErrImageNotFound, gobblerImage)
	}
	return nil
}
