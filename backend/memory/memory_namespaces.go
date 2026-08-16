//spellchecker:words memory
package memory

//spellchecker:words context slices sort time github quickpid backend
import (
	"context"
	"slices"
	"sort"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

func (s *MemoryBackend) ListNamespaces(_ context.Context, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]api.NamespaceResponse, 0, len(s.namespaces))
	for _, ns := range s.namespaces {
		if params.Tag != nil && !slices.Contains(ns.Tags, *params.Tag) {
			continue
		}
		all = append(all, ns)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: []api.NamespaceResponse{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.NamespaceResponse(nil), all[offset:end]...)
	if items == nil {
		items = []api.NamespaceResponse{}
	}
	return &api.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *MemoryBackend) CreateNamespace(_ context.Context, namespace api.ValidNamespaceID, req api.NamespaceCreateRequest, owner *api.ValidUsername, now func() time.Time) (*api.NamespaceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if owner != nil {
		if _, exists := s.users[owner.String()]; !exists {
			return nil, backend.ErrUserNotFound
		}
	}
	if _, exists := s.namespaces[namespace.String()]; exists {
		return nil, backend.ErrDuplicateNamespaceID
	}

	ts := now().UTC()
	ns := api.NamespaceResponse{
		ID:          namespace.String(),
		Tags:        append([]string(nil), req.Tags...),
		PIDFormat:   req.PIDFormat,
		DateCreated: ts,
		DateUpdated: ts,
	}
	s.namespaces[namespace.String()] = ns
	s.resources[namespace.String()] = make(map[string]api.ResourceResponse)

	if owner != nil {
		if s.roles[namespace.String()] == nil {
			s.roles[namespace.String()] = make(map[string]api.Role)
		}
		s.roles[namespace.String()][owner.String()] = api.RoleManager
	}

	return &ns, nil
}

func (s *MemoryBackend) GetNamespace(_ context.Context, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.namespaces[namespace.String()]
	if !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	return &ns, nil
}

func (s *MemoryBackend) UpdateNamespace(_ context.Context, namespace api.ValidNamespaceID, req api.ValidNamespaceUpdateRequest, now func() time.Time) (*api.NamespaceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ns, ok := s.namespaces[namespace.String()]
	if !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	if req.Tags != nil {
		ns.Tags = append([]string(nil), req.Tags...)
	}
	ns.DateUpdated = now().UTC()
	s.namespaces[namespace.String()] = ns
	return &ns, nil
}
