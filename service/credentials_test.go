//spellchecker:words service
package service_test

//spellchecker:words context errors testing time github quickpid backend memory internal apikey service
import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/backend/memory"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/service"
)

//spellchecker:words aaaaaaaaaaaaaaaaaaaaaaaaaaexpire aaaaaaaaaaaaaaaaaaaaaaaaaaaakeep aaaaaaaaaaaaaaaaaaaaaaaaaaanever aaaaaaaaaaaaaaaaaaaaaaaaaaaaakey

func TestCleanupExpiredAPIKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	b := memory.NewMemoryBackend()

	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	rt := fixedRuntime{now: now}
	svc := service.New(b, rt, service.Options{})

	alice, err := api.NewUsername("alice")
	if err != nil {
		t.Fatalf("NewUsername() error = %v", err)
	}
	if _, err := b.CreateUser(ctx, api.ValidUserCreateRequest{Username: alice}, rt.Now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	expiredAt := now.Add(-time.Hour)
	futureAt := now.Add(time.Hour)

	expiredKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaexpire"
	keepKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaaakeep"

	if _, err := b.CreateKey(ctx, apikey.Default, alice, "expired-key", expiredKey, api.KeyIssueRequest{
		Comment:         "expired",
		ExpiresAt:       &expiredAt,
		UserScopes:      []api.UserScope{},
		NamespaceScopes: []api.APIKeyNamespaceScope{},
	}, rt.Now); err != nil {
		t.Fatalf("CreateKey(expired) error = %v", err)
	}
	if _, err := b.CreateKey(ctx, apikey.Default, alice, "keep-key", keepKey, api.KeyIssueRequest{
		Comment:         "keep",
		ExpiresAt:       &futureAt,
		UserScopes:      []api.UserScope{},
		NamespaceScopes: []api.APIKeyNamespaceScope{},
	}, rt.Now); err != nil {
		t.Fatalf("CreateKey(keep) error = %v", err)
	}
	if _, err := b.CreateKey(ctx, apikey.Default, alice, "never-key", "aaaaaaaaaaaaaaaaaaaaaaaaaaanever", api.KeyIssueRequest{
		Comment:         "never",
		ExpiresAt:       nil,
		UserScopes:      []api.UserScope{},
		NamespaceScopes: []api.APIKeyNamespaceScope{},
	}, rt.Now); err != nil {
		t.Fatalf("CreateKey(never) error = %v", err)
	}

	if err := svc.CleanupExpiredAPIKeys(ctx); err != nil {
		t.Fatalf("CleanupExpiredAPIKeys() error = %v", err)
	}

	page, err := b.ListKeys(ctx, apikey.Default, alice, api.ListKeysParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("ListKeys() = %+v, want 2 remaining keys", page)
	}
	ids := map[string]struct{}{}
	for _, item := range page.Items {
		ids[item.ID] = struct{}{}
	}
	if _, ok := ids["keep-key"]; !ok {
		t.Fatalf("ListKeys() missing keep-key: %+v", page.Items)
	}
	if _, ok := ids["never-key"]; !ok {
		t.Fatalf("ListKeys() missing never-key: %+v", page.Items)
	}
	if _, ok := ids["expired-key"]; ok {
		t.Fatalf("ListKeys() still has expired-key: %+v", page.Items)
	}

	if _, _, err := b.LookupUserByKey(ctx, apikey.Default, expiredKey); !errors.Is(err, backend.ErrInvalidKey) {
		t.Fatalf("LookupUserByKey(expired) error = %v, want ErrInvalidKey", err)
	}
	if username, key, err := b.LookupUserByKey(ctx, apikey.Default, keepKey); err != nil || username != "alice" || key.ID != "keep-key" {
		t.Fatalf("LookupUserByKey(keep) = %q %+v %v", username, key, err)
	}
}

func TestCleanupExpiredAPIKeys_AnonymousMode(t *testing.T) {
	t.Parallel()

	svc := service.New(memory.NewMemoryBackend(), fixedRuntime{now: time.Now().UTC()}, service.Options{Anonymous: true})
	if err := svc.CleanupExpiredAPIKeys(context.Background()); err != nil {
		t.Fatalf("CleanupExpiredAPIKeys() error = %v", err)
	}
}

type fixedRuntime struct {
	now time.Time
}

func (r fixedRuntime) NewNamespaceID() (string, error) { return "ns", nil }
func (r fixedRuntime) NewAPIKeyID() (string, error)    { return "id", nil }
func (r fixedRuntime) NewAPIKey(apikey.Format) (string, error) {
	return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaakey", nil
}
func (r fixedRuntime) NewPID(pid.Format) (string, error) { return "pid", nil }
func (r fixedRuntime) Now() time.Time                    { return r.now }
