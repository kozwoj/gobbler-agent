package docker

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestNew requires a Docker daemon reachable at 127.0.0.1:2375 with the
// gobbler:latest image already loaded/pulled on the host (see docker-REST.md).
func TestNew(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if c == nil {
		t.Fatal("New() returned nil Docker instance with no error")
	}
	if c.baseURL != baseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, baseURL)
	}
}

// TestCreateServer requires a Docker daemon reachable at 127.0.0.1:2375 with the
// gobbler:latest image already loaded/pulled on the host (see docker-REST.md), and
// the /gobbler directory already created on the host as part of host preparation
// (see README.md). It creates a real container, verifies the returned InstanceRecord,
// and stops + removes the container via the Docker Engine API when done. The Agent
// never creates directories itself — the container creates its own outputDir
// subdirectory, so it is removed here only for test cleanup.
func TestCreateServer(t *testing.T) {
	if info, err := os.Stat(gobblerRoot); err != nil || !info.IsDir() {
		t.Fatalf("%s must exist on the host as part of host preparation (see README.md): %v", gobblerRoot, err)
	}

	c, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	const instanceName = "gobbler-agent-test-createserver"
	const hostPort = 19001
	outputDir := gobblerRoot + "/" + instanceName
	t.Cleanup(func() { os.RemoveAll(outputDir) })

	config := []byte(fmt.Sprintf(`{
		"instanceName": %q,
		"mode": "file",
		"outputDir": %q,
		"writerQueueSize": 100,
		"writerBatchSize": 10
	}`, instanceName, outputDir))

	rec, err := c.CreateServer(config, hostPort)
	if err != nil {
		t.Fatalf("CreateServer() returned error: %v", err)
	}
	t.Cleanup(func() { removeContainer(t, c, rec.ContainerID) })

	if rec.InstanceName != instanceName {
		t.Errorf("InstanceName = %q, want %q", rec.InstanceName, instanceName)
	}
	if rec.HostPort != hostPort {
		t.Errorf("HostPort = %d, want %d", rec.HostPort, hostPort)
	}
	if rec.ContainerID == "" {
		t.Error("ContainerID is empty")
	}
	if rec.Status != "running" {
		t.Errorf("Status = %q, want %q", rec.Status, "running")
	}
	if wantSuffix := ":" + strconv.Itoa(hostPort); !strings.HasSuffix(rec.URL, wantSuffix) {
		t.Errorf("URL = %q, want suffix %q", rec.URL, wantSuffix)
	}

	// Creating again with the same instanceName must fail with ErrAlreadyExists.
	if _, err := c.CreateServer(config, hostPort+1); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("second CreateServer() error = %v, want ErrAlreadyExists", err)
	}
}

// removeContainer stops and force-removes a container via the Docker Engine API.
// Used to clean up containers created by integration tests.
func removeContainer(t *testing.T, c *Docker, id string) {
	t.Helper()
	if id == "" {
		return
	}
	if resp, err := c.client.Post(fmt.Sprintf("%s/containers/%s/stop", c.baseURL, id), "application/json", nil); err == nil {
		resp.Body.Close()
	}
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/containers/%s?force=true", c.baseURL, id), nil)
	if err != nil {
		t.Errorf("building DELETE request: %v", err)
		return
	}
	resp, err := c.client.Do(req)
	if err != nil {
		t.Errorf("removing container %s: %v", id, err)
		return
	}
	resp.Body.Close()
}
