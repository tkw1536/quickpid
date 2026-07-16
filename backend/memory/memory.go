// Package memory provides an in-memory [backend.Store].
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

// NewStore returns a new in-memory backend store.
func NewStore() *Store {
	return &Store{
		users:       make(map[string]*userRecord),
		namespaces:  make(map[string]api.NamespaceResponse),
		resources:   make(map[string]map[string]api.ResourceResponse),
		permissions: make(map[string]map[string]api.PermissionLevel),
	}
}

// Store is an in-memory implementation of [backend.Store].
type Store struct {
	mu sync.RWMutex

	users       map[string]*userRecord
	namespaces  map[string]api.NamespaceResponse
	resources   map[string]map[string]api.ResourceResponse
	permissions map[string]map[string]api.PermissionLevel
}

type userRecord struct {
	superuser bool
	keys      map[string]*keyRecord
	revoked   map[string]*api.APIKeyInfo
}

func (u *userRecord) toSpec(username string) *api.UserInfo {
	return &api.UserInfo{
		Username:  username,
		Superuser: u.superuser,
	}
}

type keyRecord struct {
	info   api.APIKeyInfo
	prefix string
	digest []byte
}

var errShutdownMemoryStore = errors.New("stopped waiting for shutdown to complete")

func (s *Store) Shutdown(ctx context.Context) error {
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
		s.permissions = nil
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", errShutdownMemoryStore, ctx.Err())
	}
}
