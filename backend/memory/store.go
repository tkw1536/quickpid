// Package memory provides an in-memory [backend.Store].
package memory

//spellchecker:words sync github bicpid backend internal apikey
import (
	"sync"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/internal/apikey"
)

// NewStore returns a new in-memory backend store.
func NewStore() *Store {
	return &Store{
		users:       make(map[string]*userRecord),
		namespaces:  make(map[string]api.NamespaceResponse),
		resources:   make(map[string]map[string]api.ResourceResponse),
		permissions: make(map[string]map[string]api.PermissionLevel),
		keyFormat:   apikey.Default,
	}
}

// Store is an in-memory implementation of [backend.Store].
type Store struct {
	mu sync.RWMutex

	users       map[string]*userRecord
	namespaces  map[string]api.NamespaceResponse
	resources   map[string]map[string]api.ResourceResponse
	permissions map[string]map[string]api.PermissionLevel

	keyFormat apikey.Format
}

type userRecord struct {
	superuser bool
	keys      map[string]*keyRecord
}

type keyRecord struct {
	info   api.APIKeyInfo
	prefix string
	digest []byte
}
