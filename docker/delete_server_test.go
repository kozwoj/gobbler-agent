package docker

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

// TestDeleteServer requires a Docker daemon reachable at 127.0.0.1:2375 with the
// gobbler:latest image already loaded/pulled on the host (see docker-REST.md), and
// the /gobbler directory already created on the host (see README.md). It creates a
// real container via CreateServer, deletes it via DeleteServer, and verifies the
// container no longer exists. It also verifies DeleteServer on an unknown name
// returns ErrNotFound.
func TestDeleteServer(t *testing.T) {
	if info, err := os.Stat(gobblerRoot); err != nil || !info.IsDir() {
		t.Fatalf("%s must exist on the host as part of host preparation (see README.md): %v", gobblerRoot, err)
	}

	c, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	const instanceName = "gobbler-agent-test-deleteserver"
	const hostPort = 19003
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
	// Belt-and-suspenders: force-remove if DeleteServer fails partway through.
	t.Cleanup(func() { removeContainer(t, c, rec.ContainerID) })

	if err := c.DeleteServer(instanceName); err != nil {
		t.Fatalf("DeleteServer() returned error: %v", err)
	}

	id, err := c.containerID(instanceName)
	if err != nil {
		t.Fatalf("containerID() returned error: %v", err)
	}
	if id != "" {
		t.Errorf("container %q still exists after DeleteServer()", instanceName)
	}

	if err := c.DeleteServer("gobbler-agent-test-does-not-exist"); !errors.Is(err, ErrNotFound) {
		t.Errorf("DeleteServer(nonexistent) error = %v, want ErrNotFound", err)
	}
}
