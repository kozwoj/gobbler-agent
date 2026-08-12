package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// containerSummary is the subset of fields used from Docker's GET /containers/json response.
type containerSummary struct {
	ID    string   `json:"Id"`
	Names []string `json:"Names"`
	State string   `json:"State"`
	Ports []struct {
		PrivatePort int `json:"PrivatePort"`
		PublicPort  int `json:"PublicPort"`
	} `json:"Ports"`
}

/*
	ListServers lists all, or selected, Gobbler container(s) running on the host

- if name is provided filters the list to include only the container with the given name
- if name is empty, returns the list of all Gobbler containers
- for each container returns
  - container name
  - host port assigned to the container (9000 + NumOfServers)
*/
func (c *Docker) ListServers(name string) ([]InstanceRecord, error) {
	// all=true so stopped/exited Gobbler containers are also reported, not just running ones.
	filters := url.QueryEscape(fmt.Sprintf(`{"ancestor":["%s"]}`, gobblerImage))
	resp, err := c.client.Get(fmt.Sprintf("%s/containers/json?all=true&filters=%s", c.baseURL, filters))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: unexpected status %d from /containers/json", ErrDaemonUnreachable, resp.StatusCode)
	}

	var containers []containerSummary
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decoding /containers/json response: %w", err)
	}

	records := make([]InstanceRecord, 0, len(containers))
	for _, ct := range containers {
		if len(ct.Names) == 0 {
			continue
		}
		// Container names are reported with a leading "/".
		instanceName := strings.TrimPrefix(ct.Names[0], "/")
		if name != "" && instanceName != name {
			continue
		}

		var hostPort int
		for _, p := range ct.Ports {
			if p.PrivatePort == gobblerContainerPort {
				hostPort = p.PublicPort
				break
			}
		}

		rec := InstanceRecord{
			InstanceName: instanceName,
			HostPort:     hostPort,
			ContainerID:  ct.ID,
			Status:       containerStatus(ct.State),
		}
		if hostPort != 0 {
			rec.URL = fmt.Sprintf("http://%s:%d", hostIP(), hostPort)
		}
		if rec.Status == "running" && hostPort != 0 {
			if cfg, err := fetchGobblerConfig(hostPort); err == nil {
				rec.Mode = cfg.Mode
				rec.OutputDir = cfg.OutputDir
				rec.AccountName = cfg.AccountName
				rec.WriterQueueSize = cfg.WriterQueueSize
				rec.WriterBatchSize = cfg.WriterBatchSize
			}
		}
		records = append(records, rec)
	}
	return records, nil
}

// containerStatus maps Docker's container State field to the Agent's
// simplified status vocabulary ("running" or "stopped").
func containerStatus(state string) string {
	if state == "running" {
		return "running"
	}
	return "stopped"
}

// gobblerConfigResult holds the subset of pipeline/status fields we care about.
type gobblerConfigResult struct {
	Configured      bool   `json:"configured"`
	Mode            string `json:"mode"`
	OutputDir       string `json:"outputDir"`
	AccountName     string `json:"accountName"`
	WriterQueueSize int    `json:"writerQueueSize"`
	WriterBatchSize int    `json:"writerBatchSize"`
}

// fetchGobblerConfig calls GET /gobbler/pipeline/status on the Gobbler server
// running at 127.0.0.1:{hostPort} and returns the config fields.
// Returns an error if the request fails, times out, or the pipeline is not configured.
func fetchGobblerConfig(hostPort int) (*gobblerConfigResult, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/gobbler/pipeline/status", hostPort))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pipeline/status returned %d", resp.StatusCode)
	}
	var cfg gobblerConfigResult
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, err
	}
	if !cfg.Configured {
		return nil, fmt.Errorf("pipeline not configured")
	}
	return &cfg, nil
}
