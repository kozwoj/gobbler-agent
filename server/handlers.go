package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kozwoj/gobbler-agent/docker"
)

// handleStatus maps to GET /agent/status.
// Calls docker.Status() and returns selected Docker daemon info.
func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	info, err := s.docker.Status()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, info)
}

// handleCreateInstance maps to POST /agent/instance.
// Validates the Gobbler config, allocates a host port, then calls docker.CreateServer.
func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read request body"})
		return
	}

	// Partially decode to validate required fields before touching Docker.
	var req CreateInstanceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InstanceName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "instanceName is required"})
		return
	}
	if req.Mode != "file" && req.Mode != "blob" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `mode must be "file" or "blob"`})
		return
	}
	if req.Mode == "file" {
		expected := "/gobbler/" + req.InstanceName
		if req.OutputDir != expected {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("outputDir must be %s for file mode", expected),
			})
			return
		}
	}

	port, err := s.ports.Allocate()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	rec, err := s.docker.CreateServer(body, port)
	if err != nil {
		if errors.Is(err, docker.ErrAlreadyExists) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, rec)
}

// handleListInstances maps to GET /agent/instances.
// Calls docker.ListServers("") to return all Gobbler containers.
func (s *Server) handleListInstances(w http.ResponseWriter, _ *http.Request) {
	records, err := s.docker.ListServers("")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ListInstancesResponse{Instances: records})
}

// handleGetInstance maps to GET /agent/instances/{name}.
// Calls docker.ListServers(name) and returns the single matching record.
func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	records, err := s.docker.ListServers(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if len(records) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "instance not found"})
		return
	}
	writeJSON(w, http.StatusOK, records[0])
}

// handleDeleteInstance maps to DELETE /agent/instances/{name}.
// Calls docker.DeleteServer(name); returns 204 on success.
func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	err := s.docker.DeleteServer(name)
	if err != nil {
		if errors.Is(err, docker.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
