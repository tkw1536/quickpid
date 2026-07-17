//spellchecker:words memory
package memory

//spellchecker:words context sort github quickpid backend
import (
	"context"
	"sort"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

func (s *Store) GetNamespacePermission(_ context.Context, namespace *api.NamespaceID, username string) (api.PermissionLevel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[namespace.String()]; !ok {
		return api.PermissionLevelNone, backend.ErrNamespaceNotFound
	}
	if byUser, ok := s.permissions[namespace.String()]; ok {
		if level, ok := byUser[username]; ok {
			return level, nil
		}
	}
	return api.PermissionLevelNone, nil
}

func (s *Store) SetNamespacePermission(_ context.Context, namespace *api.NamespaceID, username string, level api.PermissionLevel) error {
	if level.Validate() != nil {
		return backend.ErrInvalidPermissionLevel
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace.String()]; !ok {
		return backend.ErrNamespaceNotFound
	}

	if level == api.PermissionLevelNone {
		if byUser, ok := s.permissions[namespace.String()]; ok {
			delete(byUser, username)
			if len(byUser) == 0 {
				delete(s.permissions, namespace.String())
			}
		}
		return nil
	}

	if s.permissions[namespace.String()] == nil {
		s.permissions[namespace.String()] = make(map[string]api.PermissionLevel)
	}
	s.permissions[namespace.String()][username] = level
	return nil
}

func (s *Store) DeleteNamespacePermission(_ context.Context, namespace *api.NamespaceID, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	byUser, ok := s.permissions[namespace.String()]
	if !ok {
		return backend.ErrPermissionNotFound
	}
	if _, ok := byUser[username]; !ok {
		return backend.ErrPermissionNotFound
	}
	delete(byUser, username)
	if len(byUser) == 0 {
		delete(s.permissions, namespace.String())
	}
	return nil
}

func (s *Store) ListNamespacePermissions(_ context.Context, namespace *api.NamespaceID, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[namespace.String()]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}

	byUser := s.permissions[namespace.String()]
	all := make([]api.NamespacePermission, 0, len(byUser))
	for username, level := range byUser {
		all = append(all, api.NamespacePermission{
			Username: username,
			Level:    level,
		})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Username < all[j].Username })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedNamespacePermissionsResponse{Total: total, Offset: offset, Items: []api.NamespacePermission{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.NamespacePermission(nil), all[offset:end]...)
	if items == nil {
		items = []api.NamespacePermission{}
	}
	return &api.PaginatedNamespacePermissionsResponse{Total: total, Offset: offset, Items: items}, nil
}
