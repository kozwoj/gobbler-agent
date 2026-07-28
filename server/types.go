package server

import "github.com/kozwoj/gobbler-agent/docker"

// CreateInstanceRequest is the body of POST /agent/instance.
// It is the full Gobbler pipeline config; the Agent extracts instanceName, mode,
// and outputDir from it and passes the raw bytes through to /gobbler/pipeline/configure.
type CreateInstanceRequest struct {
	InstanceName string `json:"instanceName"`
	Mode         string `json:"mode"`
	OutputDir    string `json:"outputDir"`
}

// CreateInstanceResponse is returned by POST /agent/instance on success (201).
type CreateInstanceResponse struct {
	InstanceName string `json:"instanceName"`
	URL          string `json:"url"`
	HostPort     int    `json:"hostPort"`
	Status       string `json:"status"`
}

// ListInstancesResponse is returned by GET /agent/instances.
type ListInstancesResponse struct {
	Instances []docker.InstanceRecord `json:"instances"`
}
