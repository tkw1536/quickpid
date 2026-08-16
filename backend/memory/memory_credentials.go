//spellchecker:words memory
package memory

//spellchecker:words bytes context slices sort time github quickpid backend internal apikey password
import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/internal/password"
)

type keyRecord struct {
	info   *api.APIKeyInfo
	prefix string
	digest []byte
}

func (s *MemoryBackend) SetPassword(_ context.Context, username api.ValidUsername, newPassword *api.ValidPassword) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return false, backend.ErrUserNotFound
	}

	user := s.users[usernameString]

	if newPassword == nil {
		user.passwordHash = nil
		s.users[usernameString] = user
		return false, nil
	}

	hash, err := password.Hash(newPassword.String())
	if err != nil {
		return false, fmt.Errorf("failed to hash password: %w", err)
	}
	user.passwordHash = hash

	s.users[usernameString] = user
	return true, nil
}

func (s *MemoryBackend) CheckPassword(_ context.Context, username api.ValidUsername, candidate api.ValidPassword) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return false, backend.ErrUserNotFound
	}

	user := s.users[usernameString]
	if len(user.passwordHash) == 0 {
		return false, nil
	}
	return password.Verify(candidate.String(), user.passwordHash), nil
}

func (s *MemoryBackend) CreateKey(_ context.Context, format apikey.Format, username api.ValidUsername, keyID string, key string, req api.KeyIssueRequest, now func() time.Time) (*api.APIKeyInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return nil, backend.ErrUserNotFound
	}

	user := s.users[usernameString]

	info := &api.APIKeyInfo{
		ID:              keyID,
		Comment:         req.Comment,
		CreatedAt:       now().UTC(),
		ExpiresAt:       req.ExpiresAt,
		UserScopes:      slices.Clone(req.UserScopes),
		NamespaceScopes: slices.Clone(req.NamespaceScopes),
	}
	info.NormalizeScopes()

	hashed, err := format.Hash(key)
	if err != nil {
		return nil, fmt.Errorf("failed to hash key: %w", err)
	}
	if _, exists := user.keys[keyID]; exists {
		return nil, backend.ErrKeyCollision
	}
	for _, otherUser := range s.users {
		for _, existing := range otherUser.keys {
			if existing.prefix == hashed.Prefix && bytes.Equal(existing.digest, hashed.Digest) {
				return nil, backend.ErrKeyCollision
			}
		}
	}

	user.keys[keyID] = keyRecord{
		info:   info,
		prefix: hashed.Prefix,
		digest: hashed.Digest,
	}

	s.users[usernameString] = user

	return info, nil
}

func (s *MemoryBackend) ListKeys(_ context.Context, _ apikey.Format, username api.ValidUsername, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return nil, backend.ErrUserNotFound
	}

	user := s.users[usernameString]

	all := make([]*api.APIKeyInfo, 0, len(user.keys))
	for _, key := range user.keys {
		info := key.info
		info.NormalizeScopes()
		all = append(all, info)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedAPIKeysResponse{Total: total, Offset: offset, Items: []*api.APIKeyInfo{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]*api.APIKeyInfo(nil), all[offset:end]...)
	if items == nil {
		items = []*api.APIKeyInfo{}
	}
	return &api.PaginatedAPIKeysResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *MemoryBackend) RevokeKey(_ context.Context, _ apikey.Format, username api.ValidUsername, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return backend.ErrUserNotFound
	}

	user := s.users[usernameString]
	if _, ok := user.keys[keyID]; !ok {
		return backend.ErrKeyNotFound
	}
	delete(s.users[usernameString].keys, keyID)
	return nil
}

func (s *MemoryBackend) LookupUserByKey(_ context.Context, format apikey.Format, key string) (string, *api.APIKeyInfo, error) {
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
				info := record.info
				info.NormalizeScopes()
				return username, info, nil
			}
		}
	}
	return "", nil, fmt.Errorf("%w: no matching key found", backend.ErrInvalidKey)
}
