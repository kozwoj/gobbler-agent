package docker

import (
	"fmt"
	"os"
	"testing"
)

// TestListServers requires a Docker daemon reachable at 127.0.0.1:2375 with the
// gobbler:latest image already loaded/pulled on the host (see docker-REST.md), and
// the /gobbler directory already created on the host (see README.md). It creates a
// real container via CreateServer, then verifies ListServers reports it both
// unfiltered and filtered by name, and returns nothing for other names.
func TestListServers(t *testing.T) {
	if info, err := os.Stat(gobblerRoot); err != nil || !info.IsDir() {
		t.Fatalf("%s must exist on the host as part of host preparation (see README.md): %v", gobblerRoot, err)
	}

	c, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	const instanceName = "gobbler-agent-test-listservers"
	const hostPort = 19002
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

	all, err := c.ListServers("")
	if err != nil {
		t.Fatalf(`ListServers("") returned error: %v`, err)
	}
	found := findInstance(all, instanceName)
	if found == nil {
		t.Fatalf(`ListServers("") did not include %q; got %+v`, instanceName, all)
	}
	if found.HostPort != hostPort {
		t.Errorf("HostPort = %d, want %d", found.HostPort, hostPort)
	}
	if found.ContainerID != rec.ContainerID {
		t.Errorf("ContainerID = %q, want %q", found.ContainerID, rec.ContainerID)
	}
	if found.Status != "running" {
		t.Errorf("Status = %q, want %q", found.Status, "running")
	}
	if found.URL != rec.URL {
		t.Errorf("URL = %q, want %q", found.URL, rec.URL)
	}

	filtered, err := c.ListServers(instanceName)
	if err != nil {
		t.Fatalf("ListServers(%q) returned error: %v", instanceName, err)
	}
	if len(filtered) != 1 {
		t.Fatalf("ListServers(%q) returned %d records, want 1", instanceName, len(filtered))
	}

	none, err := c.ListServers("gobbler-agent-test-does-not-exist")
	if err != nil {
		t.Fatalf("ListServers(nonexistent) returned error: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("ListServers(nonexistent) returned %d records, want 0", len(none))
	}
}

// findInstance returns a pointer to the record in records with the given
// InstanceName, or nil if not found.
func findInstance(records []InstanceRecord, name string) *InstanceRecord {
	for i := range records {
		if records[i].InstanceName == name {
			return &records[i]
		}
	}
	return nil
}
