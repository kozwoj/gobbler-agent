package docker

import "testing"

// TestStatus requires a Docker daemon reachable at 127.0.0.1:2375 (see docker-REST.md).
func TestStatus(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	status, err := c.Status()
	if err != nil {
		t.Fatalf("Status() returned error: %v", err)
	}
	if status == nil {
		t.Fatal("Status() returned nil DaemonStatus with no error")
	}
	if status.ID == "" {
		t.Error("Status().ID is empty")
	}
	if status.Containers < 0 {
		t.Errorf("Status().Containers = %d, want >= 0", status.Containers)
	}
}
