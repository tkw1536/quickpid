//spellchecker:words backend authentication
package authentication

//spellchecker:words bytes context errors fmt sort sync time github quickpid
import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

// NewInMemoryBackend returns a new backend backed by in-memory maps.
func NewInMemoryBackend() backend.AuthBackend {
	return &inMemoryBackend{
		users: make(map[string]*userRecord),
	}
}

type inMemoryBackend struct {
	mu    sync.RWMutex
	users map[string]*userRecord
}

type userRecord struct {
	keys map[string]*keyRecord
}

type keyRecord struct {
	info api.APIKeyInfo
	hash []byte
}

func (s *inMemoryBackend) CreateUser(_ context.Context, username string, _ func() time.Time) (*api.UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, backend.ErrDuplicateUsername
	}
	s.users[username] = &userRecord{keys: make(map[string]*keyRecord)}
	return &api.UserInfo{Username: username}, nil
}

func (s *inMemoryBackend) GetUser(_ context.Context, username string) (*api.UserInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.userLocked(username); err != nil {
		return nil, err
	}
	return &api.UserInfo{Username: username}, nil
}

func (s *inMemoryBackend) ListUsers(_ context.Context, params api.ListUsersParams) (*api.PaginatedUsersResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]api.UserInfo, 0, len(s.users))
	for username := range s.users {
		all = append(all, api.UserInfo{Username: username})
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

func (s *inMemoryBackend) DeleteUser(_ context.Context, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; !exists {
		return backend.ErrUserNotFound
	}
	delete(s.users, username)
	return nil
}

func (s *inMemoryBackend) CreateKey(_ context.Context, username, keyID string, keyHash []byte, req api.IssueKeyRequest, now func() time.Time) (*api.APIKeyInfo, error) {
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
	user.keys[keyID] = &keyRecord{info: info, hash: append([]byte(nil), keyHash...)}
	return cloneAPIKeyInfo(&info), nil
}

func (s *inMemoryBackend) ListKeys(_ context.Context, username string, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
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

func (s *inMemoryBackend) GetKey(_ context.Context, username, keyID string) (*api.APIKeyInfo, error) {
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

func (s *inMemoryBackend) UpdateKey(_ context.Context, username, keyID string, req api.UpdateKeyRequest, _ func() time.Time) (*api.APIKeyInfo, error) {
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

func (s *inMemoryBackend) RevokeKey(_ context.Context, username, keyID string) (*api.APIKeyInfo, error) {
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
	info := cloneAPIKeyInfo(&key.info)
	delete(user.keys, keyID)
	return info, nil
}

func (s *inMemoryBackend) LookupUserByKeyHash(_ context.Context, keyHash []byte) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for username, user := range s.users {
		for _, key := range user.keys {
			if bytes.Equal(key.hash, keyHash) {
				return username, nil
			}
		}
	}
	return "", backend.ErrInvalidKey
}

var errShutdownInMemoryBackend = errors.New("stopped waiting for shutdown to complete")

func (s *inMemoryBackend) Shutdown(ctx context.Context) error {
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)

		s.mu.Lock()
		defer s.mu.Unlock()

		if err := ctx.Err(); err != nil {
			return
		}

		s.users = nil
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", errShutdownInMemoryBackend, ctx.Err())
	}
}

func (s *inMemoryBackend) userLocked(username string) (*userRecord, error) {
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
