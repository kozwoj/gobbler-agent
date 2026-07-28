package ports

import (
	"fmt"
	"net"
	"sync"
)

// Allocator finds free TCP ports starting from a base port.
type Allocator struct {
	mu   sync.Mutex
	next int
}

func NewAllocator(base int) *Allocator {
	return &Allocator{next: base}
}

// Allocate returns the next free TCP port at or above the base port.
func (a *Allocator) Allocate() (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for port := a.next; port < a.next+100; port++ {
		if isFree(port) {
			a.next = port + 1
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found in range %d–%d", a.next, a.next+100)
}

func isFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
