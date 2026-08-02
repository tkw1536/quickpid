//spellchecker:words memory
package memory

//spellchecker:words context sort github quickpid backend
import (
	"context"
	"sort"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

func (s *Store) GetNamespaceRole(_ context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) (api.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	namespaceString := namespace.String()

	if _, ok := s.namespaces[namespaceString]; !ok {
		return api.RoleNone, backend.ErrNamespaceNotFound
	}
	if byUser, ok := s.roles[namespaceString]; ok {
		if role, ok := byUser[username.String()]; ok {
			return role, nil
		}
	}
	return api.RoleNone, nil
}

func (s *Store) SetNamespaceRole(_ context.Context, namespace api.ValidNamespaceID, username api.ValidUsername, role api.Role) error {
	if role.Validate() != nil {
		return backend.ErrInvalidRole
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	namespaceString := namespace.String()
	usernameString := username.String()

	if _, ok := s.namespaces[namespaceString]; !ok {
		return backend.ErrNamespaceNotFound
	}

	if role == api.RoleNone {
		if byUser, ok := s.roles[namespaceString]; ok {
			delete(byUser, usernameString)
			if len(byUser) == 0 {
				delete(s.roles, namespaceString)
			}
		}
		return nil
	}

	if s.roles[namespaceString] == nil {
		s.roles[namespaceString] = make(map[string]api.Role)
	}
	s.roles[namespaceString][usernameString] = role
	return nil
}

func (s *Store) DeleteNamespaceRole(_ context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	namespaceString := namespace.String()
	usernameString := username.String()

	byUser, ok := s.roles[namespaceString]
	if !ok {
		return backend.ErrRoleNotFound
	}
	if _, ok := byUser[usernameString]; !ok {
		return backend.ErrRoleNotFound
	}
	delete(byUser, usernameString)
	if len(byUser) == 0 {
		delete(s.roles, namespaceString)
	}
	return nil
}

func (s *Store) ListNamespaceRoles(_ context.Context, namespace api.ValidNamespaceID, params api.ListNamespaceRolesParams) (*api.PaginatedNamespaceRolesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	namespaceString := namespace.String()

	if _, ok := s.namespaces[namespaceString]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}

	byUser := s.roles[namespaceString]
	all := make([]api.NamespaceRole, 0, len(byUser))
	for username, role := range byUser {
		all = append(all, api.NamespaceRole{
			Username: username,
			Role:     role,
		})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Username < all[j].Username })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedNamespaceRolesResponse{Total: total, Offset: offset, Items: []api.NamespaceRole{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.NamespaceRole(nil), all[offset:end]...)
	if items == nil {
		items = []api.NamespaceRole{}
	}
	return &api.PaginatedNamespaceRolesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *Store) ListUserRoles(_ context.Context, username api.ValidUsername, params api.ListUserRolesParams) (*api.PaginatedUserRolesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return nil, backend.ErrUserNotFound
	}

	all := make([]api.UserRole, 0)
	for namespace, byUser := range s.roles {
		if role, ok := byUser[username.String()]; ok {
			all = append(all, api.UserRole{
				Namespace: namespace,
				Role:      role,
			})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Namespace < all[j].Namespace })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedUserRolesResponse{Total: total, Offset: offset, Items: []api.UserRole{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.UserRole(nil), all[offset:end]...)
	if items == nil {
		items = []api.UserRole{}
	}
	return &api.PaginatedUserRolesResponse{Total: total, Offset: offset, Items: items}, nil
}
