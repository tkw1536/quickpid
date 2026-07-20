//spellchecker:words memory
package memory

//spellchecker:words context sort github quickpid backend
import (
	"context"
	"sort"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

func (s *Store) GetNamespacePermission(_ context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) (api.PermissionLevel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[namespace.String()]; !ok {
		return api.PermissionLevelNone, backend.ErrNamespaceNotFound
	}
	if byUser, ok := s.permissions[namespace.String()]; ok {
		if level, ok := byUser[username.String()]; ok {
			return level, nil
		}
	}
	return api.PermissionLevelNone, nil
}

func (s *Store) SetNamespacePermission(_ context.Context, namespace api.ValidNamespaceID, username api.ValidUsername, level api.PermissionLevel) error {
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
			delete(byUser, username.String())
			if len(byUser) == 0 {
				delete(s.permissions, namespace.String())
			}
		}
		return nil
	}

	if s.permissions[namespace.String()] == nil {
		s.permissions[namespace.String()] = make(map[string]api.PermissionLevel)
	}
	s.permissions[namespace.String()][username.String()] = level
	return nil
}

func (s *Store) DeleteNamespacePermission(_ context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	byUser, ok := s.permissions[namespace.String()]
	if !ok {
		return backend.ErrPermissionNotFound
	}
	if _, ok := byUser[username.String()]; !ok {
		return backend.ErrPermissionNotFound
	}
	delete(byUser, username.String())
	if len(byUser) == 0 {
		delete(s.permissions, namespace.String())
	}
	return nil
}

func (s *Store) ListNamespacePermissions(_ context.Context, namespace api.ValidNamespaceID, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, error) {
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

func (s *Store) ListUserPermissions(_ context.Context, username api.ValidUsername, params api.ListUserPermissionsParams) (*api.PaginatedUserPermissionsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.userLocked(username.String()); err != nil {
		return nil, err
	}

	all := make([]api.UserPermission, 0)
	for namespace, byUser := range s.permissions {
		if level, ok := byUser[username.String()]; ok {
			all = append(all, api.UserPermission{
				Namespace: namespace,
				Level:     level,
			})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Namespace < all[j].Namespace })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedUserPermissionsResponse{Total: total, Offset: offset, Items: []api.UserPermission{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.UserPermission(nil), all[offset:end]...)
	if items == nil {
		items = []api.UserPermission{}
	}
	return &api.PaginatedUserPermissionsResponse{Total: total, Offset: offset, Items: items}, nil
}
