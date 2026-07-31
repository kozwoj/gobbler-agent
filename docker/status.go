package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// infoResponse is the subset of fields used from Docker's GET /info response.
type infoResponse struct {
	ID                string `json:"ID"`
	Containers        int    `json:"Containers"`
	ContainersRunning int    `json:"ContainersRunning"`
	ContainersPaused  int    `json:"ContainersPaused"`
	ContainersStopped int    `json:"ContainersStopped"`
}

/*
	Status returns selected fields from the Docker daemon's GET /info response.

- returns ErrDaemonUnreachable if the Docker daemon is not reachable
*/
func (c *Docker) Status() (*DaemonStatus, error) {
	resp, err := c.client.Get(c.baseURL + "/info")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: unexpected status %d from /info", ErrDaemonUnreachable, resp.StatusCode)
	}

	var info infoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decoding /info response: %w", err)
	}

	return &DaemonStatus{
		ID:                info.ID,
		Containers:        info.Containers,
		ContainersRunning: info.ContainersRunning,
		ContainersPaused:  info.ContainersPaused,
		ContainersStopped: info.ContainersStopped,
	}, nil
}
