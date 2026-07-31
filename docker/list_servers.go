package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
