package docker

import (
	"fmt"
	"io"
	"net/http"
)

/*
	DeleteServer stops and removes a Gobbler container with the given name

- verifies that a container with the given name exists
- stops the container (POST /containers/{id}/stop)
- removes the container (DELETE /containers/{id})

Note: we are assuming that the Agent's endpoint corresponding to deleting the container verifies
that the server's pipeline has been stopped -- Gobbler is not ingesting any data and all files/blobs have been flushed.
*/
func (c *Docker) DeleteServer(name string) error {
	id, err := c.containerID(name)
	if err != nil {
		return err
	}
	if id == "" {
		return ErrNotFound
	}

	if err := c.stopContainer(id); err != nil {
		return err
	}
	return c.deleteContainer(id)
}

// stopContainer calls POST /containers/{id}/stop. A 304 response (container
// already stopped) is treated as success.
func (c *Docker) stopContainer(id string) error {
	resp, err := c.client.Post(fmt.Sprintf("%s/containers/%s/stop", c.baseURL, id), "application/json", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d from /containers/%s/stop: %s", resp.StatusCode, id, body)
	}
	return nil
}

// deleteContainer calls DELETE /containers/{id}.
func (c *Docker) deleteContainer(id string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/containers/%s", c.baseURL, id), nil)
	if err != nil {
		return fmt.Errorf("building DELETE request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d from DELETE /containers/%s: %s", resp.StatusCode, id, body)
	}
	return nil
}
