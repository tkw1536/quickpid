//spellchecker:words memory
package memory

//spellchecker:words bytes context sort strings time github quickpid backend internal apikey password
import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/internal/password"
)

func (s *Store) CreateUser(_ context.Context, req api.ValidUserCreateRequest, _ func() time.Time) (*api.UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[req.Username.String()]; exists {
		return nil, backend.ErrDuplicateUsername
	}
	s.users[req.Username.String()] = &userRecord{
		superuser: req.Superuser,
		keys:      make(map[string]*keyRecord),
		revoked:   make(map[string]*api.APIKeyInfo),
	}
	return s.users[req.Username.String()].toSpec(req.Username.String()), nil
}

func (s *Store) GetUser(_ context.Context, username api.ValidUsername) (*api.UserInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, err := s.userLocked(username.String())
	if err != nil {
		return nil, err
	}
	return user.toSpec(username.String()), nil
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

func (s *Store) AutocompleteUsers(_ context.Context, query string, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matches := make([]string, 0)
	for username := range s.users {
		if strings.HasPrefix(username, query) {
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

	user, err := s.userLocked(username.String())
	if err != nil {
		return nil, err
	}
	if req.Superuser != nil {
		user.superuser = *req.Superuser
	}
	return user.toSpec(username.String()), nil
}

func (s *Store) SetPassword(_ context.Context, username api.ValidUsername, newPassword *api.ValidPassword) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.userLocked(username.String())
	if err != nil {
		return false, err
	}

	if newPassword == nil {
		user.passwordHash = nil
		return false, nil
	}

	hash, err := password.Hash(newPassword.String())
	if err != nil {
		return false, fmt.Errorf("password.Hash: %w", err)
	}
	user.passwordHash = hash
	return true, nil
}

func (s *Store) CheckPassword(_ context.Context, username api.ValidUsername, candidate api.ValidPassword) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, err := s.userLocked(username.String())
	if err != nil {
		return false, err
	}
	if len(user.passwordHash) == 0 {
		return false, nil
	}
	return password.Verify(candidate.String(), user.passwordHash), nil
}

func (s *Store) CreateKey(_ context.Context, format apikey.Format, username api.ValidUsername, keyID string, key string, req api.KeyIssueRequest, now func() time.Time) (*api.APIKeyInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.userLocked(username.String())
	if err != nil {
		return nil, err
	}

	info := api.APIKeyInfo{
		ID:        keyID,
		Comment:   req.Comment,
		CreatedAt: now().UTC(),
		ExpiresAt: cloneTimePtr(req.ExpiresAt),
	}
	hashed, err := format.Hash(key)
	if err != nil {
		return nil, fmt.Errorf("apikey.Format.Hash: %w", err)
	}
	if user.keys[keyID] != nil {
		return nil, backend.ErrKeyCollision
	}
	for _, otherUser := range s.users {
		for _, existing := range otherUser.keys {
			if existing.prefix == hashed.Prefix && bytes.Equal(existing.digest, hashed.Digest) {
				return nil, backend.ErrKeyCollision
			}
		}
	}
	user.keys[keyID] = &keyRecord{
		info:   info,
		prefix: hashed.Prefix,
		digest: hashed.Digest,
	}
	return cloneAPIKeyInfo(&info), nil
}

func (s *Store) ListKeys(_ context.Context, _ apikey.Format, username api.ValidUsername, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, err := s.userLocked(username.String())
	if err != nil {
		return nil, err
	}

	all := make([]api.APIKeyInfo, 0, len(user.keys))
	for _, key := range user.keys {
		all = append(all, *cloneAPIKeyInfo(&key.info))
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedAPIKeysResponse{Total: total, Offset: offset, Items: []api.APIKeyInfo{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.APIKeyInfo(nil), all[offset:end]...)
	if items == nil {
		items = []api.APIKeyInfo{}
	}
	return &api.PaginatedAPIKeysResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *Store) RevokeKey(_ context.Context, _ apikey.Format, username api.ValidUsername, keyID string) (*api.APIKeyInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.userLocked(username.String())
	if err != nil {
		return nil, err
	}
	if revoked, ok := user.revoked[keyID]; ok {
		return cloneAPIKeyInfo(revoked), nil
	}
	key, ok := user.keys[keyID]
	if !ok {
		return nil, backend.ErrKeyNotFound
	}
	info := cloneAPIKeyInfo(&key.info)
	user.revoked[keyID] = info
	delete(user.keys, keyID)
	return info, nil
}

func (s *Store) LookupUserByKey(_ context.Context, format apikey.Format, key string) (string, *api.APIKeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lookupPrefix, err := format.Prefix(key)
	if err != nil {
		return "", nil, fmt.Errorf("%w: invalid key prefix", backend.ErrInvalidKey)
	}

	for username, user := range s.users {
		for _, record := range user.keys {
			if record.prefix != lookupPrefix {
				continue
			}
			if format.Verify(key, apikey.Stored{Prefix: record.prefix, Digest: record.digest}) {
				return username, new(record.info), nil
			}
		}
	}
	return "", nil, fmt.Errorf("%w: no matching key found", backend.ErrInvalidKey)
}

func (s *Store) userLocked(username string) (*userRecord, error) {
	user, ok := s.users[username]
	if !ok {
		return nil, backend.ErrUserNotFound
	}
	return user, nil
}

func cloneTimePtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func cloneAPIKeyInfo(info *api.APIKeyInfo) *api.APIKeyInfo {
	if info == nil {
		return nil
	}
	out := *info
	out.ExpiresAt = cloneTimePtr(info.ExpiresAt)
	return &out
}
