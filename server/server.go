package server

import (
	"fmt"
	"net/http"

	"github.com/kozwoj/gobbler-agent/docker"
	"github.com/kozwoj/gobbler-agent/ports"
)

const basePort = 9001

// Server is the Gobbler Agent HTTP server.
type Server struct {
	port   int
	mux    *http.ServeMux
	ports  *ports.Allocator
	docker *docker.Docker
}

func New(port int) (*Server, error) {
	dc, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	s := &Server{
		port:   port,
		mux:    http.NewServeMux(),
		ports:  ports.NewAllocator(basePort),
		docker: dc,
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /agent/instance", s.handleCreateInstance)
	s.mux.HandleFunc("GET /agent/instances", s.handleListInstances)
	s.mux.HandleFunc("GET /agent/instances/{name}", s.handleGetInstance)
	s.mux.HandleFunc("DELETE /agent/instances/{name}", s.handleDeleteInstance)
	s.mux.HandleFunc("GET /agent/status", s.handleStatus)
}

func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), s.mux)
}
