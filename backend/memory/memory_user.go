//spellchecker:words memory
package memory

//spellchecker:words context sort strings time github quickpid backend
import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

type userRecord struct {
	superuser    bool
	passwordHash []byte
	keys         map[string]keyRecord
}

func (u userRecord) toSpec(username string) *api.UserInfo {
	return &api.UserInfo{
		Username:  username,
		Superuser: u.superuser,
		Password:  len(u.passwordHash) > 0,
	}
}

func (s *Store) CreateUser(_ context.Context, req api.ValidUserCreateRequest, _ func() time.Time) (*api.UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username := req.Username.String()
	if _, exists := s.users[username]; exists {
		return nil, backend.ErrDuplicateUsername
	}

	s.users[username] = userRecord{
		superuser: req.Superuser,
		keys:      make(map[string]keyRecord),
	}
	return s.users[username].toSpec(username), nil
}

func (s *Store) GetUser(_ context.Context, username api.ValidUsername) (*api.UserInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	usernameString := username.String()

	if _, exists := s.users[usernameString]; !exists {
		return nil, backend.ErrUserNotFound
	}
	return s.users[usernameString].toSpec(usernameString), nil
}

func (s *Store) ListUsers(_ context.Context, params api.ListUsersParams) (*api.PaginatedUsersResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]api.UserInfo, 0, len(s.users))
	for username, user := range s.users {
		if params.Superuser != nil && user.superuser != *params.Superuser {
			continue
		}
		if params.Password != nil && (len(user.passwordHash) > 0) != *params.Password {
			continue
		}
		all = append(all, *user.toSpec(username))
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Username < all[j].Username })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedUsersResponse{Total: total, Offset: offset, Items: []api.UserInfo{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.UserInfo(nil), all[offset:end]...)
	if items == nil {
		items = []api.UserInfo{}
	}
	return &api.PaginatedUsersResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *Store) AutocompleteUsers(_ context.Context, query api.ValidAutocompleteQuery, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queryString := query.String()
	matches := make([]string, 0)
	for username := range s.users {
		if strings.HasPrefix(username, queryString) {
			matches = append(matches, username)
		}
	}
	sort.Strings(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	if matches == nil {
		matches = []string{}
	}
	return matches, nil
}

func (s *Store) DeleteUser(_ context.Context, username api.ValidUsername) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username.String()]; !exists {
		return backend.ErrUserNotFound
	}

	delete(s.users, username.String())
	for namespace, byUser := range s.roles {
		delete(byUser, username.String())
		if len(byUser) == 0 {
			delete(s.roles, namespace)
		}
	}
	return nil
}

func (s *Store) UpdateUser(_ context.Context, username api.ValidUsername, req api.UserUpdateRequest) (*api.UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return nil, backend.ErrUserNotFound
	}

	// update the superuser flag
	if req.Superuser != nil {
		oldUser := s.users[usernameString]
		oldUser.superuser = *req.Superuser
		s.users[usernameString] = oldUser
	}

	// and return the updated users
	return s.users[usernameString].toSpec(usernameString), nil
}
