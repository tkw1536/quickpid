// Package memory provides an in-memory [backend.Backend].
//
//spellchecker:words memory
package memory

//spellchecker:words context errors sync github quickpid
import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/tkw1536/quickpid/api"
)

// MemoryBackend is an in-memory implementation of [backend.Backend].
type MemoryBackend struct {
	mu sync.RWMutex

	users      map[string]userRecord
	namespaces map[string]api.NamespaceResponse
	resources  map[string]map[string]api.ResourceResponse
	roles      map[string]map[string]api.Role
	mounts     map[string]string // baseURI -> namespaceID
}

// NewMemoryBackend returns a new in-memory backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		users:      make(map[string]userRecord),
		namespaces: make(map[string]api.NamespaceResponse),
		resources:  make(map[string]map[string]api.ResourceResponse),
		roles:      make(map[string]map[string]api.Role),
		mounts:     make(map[string]string),
	}
}

var errShutdownMemoryBackend = errors.New("stopped waiting for shutdown to complete")

func (s *MemoryBackend) Shutdown(ctx context.Context) error {
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)

		s.mu.Lock()
		defer s.mu.Unlock()

		if err := ctx.Err(); err != nil {
			return
		}

		s.users = nil
		s.namespaces = nil
		s.resources = nil
		s.roles = nil
		s.mounts = nil
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", errShutdownMemoryBackend, ctx.Err())
	}
}
