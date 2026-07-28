package docker

import (
	"errors"
	"time"
)

// Sentinel errors returned by Docker methods.
var (
	ErrNotFound      = errors.New("instance not found")
	ErrAlreadyExists = errors.New("instance already exists")
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
	starTime string // time when the Docker wrapper was created
}

/* New creates a new Docker wrapper instance. At minimum it
- verifies that the Docker daemon is running and is accessible on port 2375 (or 2376 for TLS)
- verifies that the Gobbler image is available

Note: we are assuming that part of the host configuration is to start
Docker daemon and pull the Gobbler image.
*/
func New() (*Docker, error) {
	// TODO: check if Docker daemon is running and Gobbler image is available
	return &Docker{
		starTime: time.Now().Format(time.RFC3339),
	}, nil
}

/* CreateServer creates, starts and configures a new Gobbler container. It
- verifies that no container with instanceName is already running (returns ErrAlreadyExists if so)
- if mode == file, verifies that outputDir follows the host path convention /gobbler/<instanceName>
- creates the container (POST /containers/create) with instanceName, -v bindings (file mode) and -p hostPort:8080
- starts the container (POST /containers/{id}/start)
- polls GET /gobbler/pipeline/status until the container responds
- configures the container (POST /gobbler/pipeline/configure) with the full config payload
- returns the populated InstanceRecord for the new instance
*/
func (c *Docker) CreateServer(config []byte, hostPort int) (InstanceRecord, error) {
	// TODO: implement the logic to create, start and configure a new Gobbler container
	return InstanceRecord{}, nil
}

/* ListServers lists all, or selected, Gobbler container(s) running on the host
- if name is provided filters the list to include only the container with the given name
- if name is empty, returns the list of all Gobbler containers
- for each container returns
	- container name
	- host port assigned to the container (9000 + NumOfServers)
*/
func (c *Docker) ListServers(name string) ([]InstanceRecord, error) {
	// TODO: query GET /containers/json, filter by gobbler label/name, map to InstanceRecord
	return nil, nil
}

/* DeleteServer stops and removes a Gobbler container with the given name
- verifies that a container with the given name exists
- stops the container (POST /containers/{id}/stop)
- removes the container (DELETE /containers/{id})

Note: we are assuming that the Agent's endpoint corresponding to deleting the container verifies
that the server's pipeline has been stopped -- Gobbler is not ingesting any data and all files/blobs have been flushed.
*/
func (c *Docker) DeleteServer(name string) error {
	// TODO: implement the logic to stop and remove a Gobbler container
	return nil
}

/* Status returns selected fields from the Docker daemon's GET /info response.
- returns ErrNotFound if the Docker daemon is not reachable
*/
func (c *Docker) Status() (*DaemonStatus, error) {
	// TODO: call GET /info on the Docker daemon and map to DaemonStatus
	return nil, nil
}
