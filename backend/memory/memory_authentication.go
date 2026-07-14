//spellchecker:words memory
package memory

//spellchecker:words bytes context sort strings time github bicpid backend internal apikey
import (
	"bytes"
	"context"
	"sort"
	"strings"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/apikey"
)

func (s *Store) CreateUser(_ context.Context, req api.UserCreateRequest, _ func() time.Time) (*api.UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[req.Username]; exists {
		return nil, backend.ErrDuplicateUsername
	}
	s.users[req.Username] = &userRecord{
		superuser: req.Superuser,
		keys:      make(map[string]*keyRecord),
		revoked:   make(map[string]*api.APIKeyInfo),
	}
	return s.users[req.Username].toSpec(req.Username), nil
}

func (s *Store) GetUser(_ context.Context, username string) (*api.UserInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, err := s.userLocked(username)
	if err != nil {
		return nil, err
	}
	return user.toSpec(username), nil
}

func (s *Store) ListUsers(_ context.Context, params api.ListUsersParams) (*api.PaginatedUsersResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]api.UserInfo, 0, len(s.users))
	for username, user := range s.users {
		if params.Superuser != nil && user.superuser != *params.Superuser {
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

func (s *Store) DeleteUser(_ context.Context, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; !exists {
		return backend.ErrUserNotFound
	}

	delete(s.users, username)
	for namespace, byUser := range s.permissions {
		delete(byUser, username)
		if len(byUser) == 0 {
			delete(s.permissions, namespace)
		}
	}
	return nil
}

func (s *Store) UpdateUser(_ context.Context, username string, req api.UserUpdateRequest) (*api.UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.userLocked(username)
	if err != nil {
		return nil, err
	}
	if req.Superuser != nil {
		user.superuser = *req.Superuser
	}
	return user.toSpec(username), nil
}

func (s *Store) CreateKey(_ context.Context, format apikey.Format, username, keyID string, key string, req api.KeyIssueRequest, now func() time.Time) (*api.APIKeyInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.userLocked(username)
	if err != nil {
		return nil, err
	}

	info := api.APIKeyInfo{
		ID:        keyID,
		Comment:   req.Comment,
		CreatedAt: now().UTC().Format(time.RFC3339),
		ExpiresAt: cloneStringPtr(req.ExpiresAt),
	}
	hashed, err := format.Hash(key)
	if err != nil {
		return nil, err
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

func (s *Store) ListKeys(_ context.Context, _ apikey.Format, username string, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, err := s.userLocked(username)
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

func (s *Store) GetKey(_ context.Context, _ apikey.Format, username, keyID string) (*api.APIKeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, err := s.userLocked(username)
	if err != nil {
		return nil, err
	}
	key, ok := user.keys[keyID]
	if !ok {
		return nil, backend.ErrKeyNotFound
	}
	return cloneAPIKeyInfo(&key.info), nil
}

func (s *Store) UpdateKey(_ context.Context, _ apikey.Format, username, keyID string, req api.KeyUpdateRequest, _ func() time.Time) (*api.APIKeyInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.userLocked(username)
	if err != nil {
		return nil, err
	}
	key, ok := user.keys[keyID]
	if !ok {
		return nil, backend.ErrKeyNotFound
	}

	if req.Comment != nil {
		key.info.Comment = *req.Comment
	}
	if req.ExpiresAt != nil {
		key.info.ExpiresAt = cloneStringPtr(*req.ExpiresAt)
	}
	return cloneAPIKeyInfo(&key.info), nil
}

func (s *Store) RevokeKey(_ context.Context, _ apikey.Format, username, keyID string) (*api.APIKeyInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.userLocked(username)
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

func (s *Store) LookupUserByKey(_ context.Context, format apikey.Format, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lookupPrefix, err := format.Prefix(key)
	if err != nil {
		return "", backend.ErrInvalidKey
	}

	for username, user := range s.users {
		for _, record := range user.keys {
			if record.prefix != lookupPrefix {
				continue
			}
			if format.Verify(key, apikey.Stored{Prefix: record.prefix, Digest: record.digest}) {
				return username, nil
			}
		}
	}
	return "", backend.ErrInvalidKey
}

func (s *Store) userLocked(username string) (*userRecord, error) {
	user, ok := s.users[username]
	if !ok {
		return nil, backend.ErrUserNotFound
	}
	return user, nil
}

func cloneStringPtr(v *string) *string {
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
	out.ExpiresAt = cloneStringPtr(info.ExpiresAt)
	return &out
}
