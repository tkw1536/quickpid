package memory

//spellchecker:words context sort github bicpid backend
import (
	"context"
	"sort"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

func (s *Store) GetNamespacePermission(_ context.Context, namespace, username string) (api.PermissionLevel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if byUser, ok := s.permissions[namespace]; ok {
		if level, ok := byUser[username]; ok {
			return level, nil
		}
	}
	return api.PermissionLevelNone, nil
}

func (s *Store) SetNamespacePermission(_ context.Context, namespace, username string, level api.PermissionLevel) error {
	if !api.IsValidPermissionLevel(level) {
		return backend.ErrInvalidPermissionLevel
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace]; !ok {
		return backend.ErrNamespaceNotFound
	}

	if level == api.PermissionLevelNone {
		if byUser, ok := s.permissions[namespace]; ok {
			delete(byUser, username)
			if len(byUser) == 0 {
				delete(s.permissions, namespace)
			}
		}
		return nil
	}

	if s.permissions[namespace] == nil {
		s.permissions[namespace] = make(map[string]api.PermissionLevel)
	}
	s.permissions[namespace][username] = level
	return nil
}

func (s *Store) DeleteNamespacePermission(_ context.Context, namespace, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	byUser, ok := s.permissions[namespace]
	if !ok {
		return backend.ErrPermissionNotFound
	}
	if _, ok := byUser[username]; !ok {
		return backend.ErrPermissionNotFound
	}
	delete(byUser, username)
	if len(byUser) == 0 {
		delete(s.permissions, namespace)
	}
	return nil
}

func (s *Store) ListNamespacePermissions(_ context.Context, namespace string, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[namespace]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}

	byUser := s.permissions[namespace]
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
