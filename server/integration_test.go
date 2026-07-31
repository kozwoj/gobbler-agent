package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kozwoj/gobbler-agent/docker"
)

const gobblerRoot = "/gobbler"

// doRequest performs an HTTP request against the httptest server and returns the
// status code and decoded JSON body (as raw bytes for the caller to unmarshal).
func doRequest(t *testing.T, method, url string, body []byte) (int, []byte) {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("NewRequest(%s %s): %v", method, url, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body for %s %s: %v", method, url, err)
	}
	return resp.StatusCode, respBody
}

// TestInstanceLifecycle requires a Docker daemon reachable at 127.0.0.1:2375 with the
// gobbler:latest image already loaded/pulled on the host (see docker-REST.md), and the
// /gobbler directory already created on the host as part of host preparation (see
// README.md). It exercises the full REST lifecycle against a real httptest.Server
// wrapping the Agent's mux: create, list, get, delete, and status.
func TestInstanceLifecycle(t *testing.T) {
	if info, err := os.Stat(gobblerRoot); err != nil || !info.IsDir() {
		t.Fatalf("%s must exist on the host as part of host preparation (see README.md): %v", gobblerRoot, err)
	}

	s, err := New(9000)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	httpSrv := httptest.NewServer(s.mux)
	t.Cleanup(httpSrv.Close)

	const instanceName = "gobbler-agent-test-lifecycle"
	outputDir := gobblerRoot + "/" + instanceName
	t.Cleanup(func() { os.RemoveAll(outputDir) })

	config := []byte(fmt.Sprintf(`{
		"instanceName": %q,
		"mode": "file",
		"outputDir": %q,
		"writerQueueSize": 100,
		"writerBatchSize": 10
	}`, instanceName, outputDir))

	// POST /agent/instance
	status, body := doRequest(t, http.MethodPost, httpSrv.URL+"/agent/instance", config)
	if status != http.StatusOK {
		t.Fatalf("POST /agent/instance status = %d, body = %s", status, body)
	}
	var created docker.InstanceRecord
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode create response: %v, body = %s", err, body)
	}
	// Belt-and-suspenders cleanup in case a later assertion fails before the
	// explicit DELETE step runs.
	t.Cleanup(func() {
		doRequest(t, http.MethodDelete, httpSrv.URL+"/agent/instances/"+instanceName, nil)
	})
	if created.InstanceName != instanceName {
		t.Errorf("InstanceName = %q, want %q", created.InstanceName, instanceName)
	}
	if created.Status != "running" {
		t.Errorf("Status = %q, want %q", created.Status, "running")
	}
	if created.ContainerID == "" {
		t.Error("ContainerID is empty")
	}

	// POST /agent/instance again with the same name should conflict.
	status, body = doRequest(t, http.MethodPost, httpSrv.URL+"/agent/instance", config)
	if status != http.StatusConflict {
		t.Errorf("second POST /agent/instance status = %d, want %d, body = %s", status, http.StatusConflict, body)
	}

	// GET /agent/instances
	status, body = doRequest(t, http.MethodGet, httpSrv.URL+"/agent/instances", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /agent/instances status = %d, body = %s", status, body)
	}
	var list ListInstancesResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode list response: %v, body = %s", err, body)
	}
	found := false
	for _, rec := range list.Instances {
		if rec.InstanceName == instanceName {
			found = true
		}
	}
	if !found {
		t.Errorf("GET /agent/instances did not contain %q: %+v", instanceName, list.Instances)
	}

	// GET /agent/instances/{name}
	status, body = doRequest(t, http.MethodGet, httpSrv.URL+"/agent/instances/"+instanceName, nil)
	if status != http.StatusOK {
		t.Fatalf("GET /agent/instances/{name} status = %d, body = %s", status, body)
	}
	var single docker.InstanceRecord
	if err := json.Unmarshal(body, &single); err != nil {
		t.Fatalf("decode single response: %v, body = %s", err, body)
	}
	if single.InstanceName != instanceName {
		t.Errorf("single InstanceName = %q, want %q", single.InstanceName, instanceName)
	}

	// GET /agent/instances/{name} for a nonexistent instance.
	status, _ = doRequest(t, http.MethodGet, httpSrv.URL+"/agent/instances/does-not-exist", nil)
	if status != http.StatusNotFound {
		t.Errorf("GET nonexistent instance status = %d, want %d", status, http.StatusNotFound)
	}

	// GET /agent/status
	status, body = doRequest(t, http.MethodGet, httpSrv.URL+"/agent/status", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /agent/status status = %d, body = %s", status, body)
	}
	var daemonStatus docker.DaemonStatus
	if err := json.Unmarshal(body, &daemonStatus); err != nil {
		t.Fatalf("decode status response: %v, body = %s", err, body)
	}
	if daemonStatus.ID == "" {
		t.Error("GET /agent/status: ID is empty")
	}

	// DELETE /agent/instances/{name}
	status, body = doRequest(t, http.MethodDelete, httpSrv.URL+"/agent/instances/"+instanceName, nil)
	if status != http.StatusNoContent {
		t.Errorf("DELETE /agent/instances/{name} status = %d, want %d, body = %s", status, http.StatusNoContent, body)
	}

	// DELETE again should now 404.
	status, _ = doRequest(t, http.MethodDelete, httpSrv.URL+"/agent/instances/"+instanceName, nil)
	if status != http.StatusNotFound {
		t.Errorf("second DELETE status = %d, want %d", status, http.StatusNotFound)
	}
}
