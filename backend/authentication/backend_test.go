//spellchecker:words backend authentication
package authentication_test

//spellchecker:words context errors strings testing time github quickpid backend
import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

func userReq(username string) api.UserCreateRequest {
	return api.UserCreateRequest{Username: username}
}

func testBackendUserCRUD(t *testing.T, newBackend func() backend.AuthBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := fixedNow()

	created, err := b.CreateUser(ctx, userReq("alice"), now)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if created.Username != "alice" || created.Superuser {
		t.Fatalf("CreateUser() = %+v, want username alice with superuser false", created)
	}

	if _, err := b.CreateUser(ctx, userReq("alice"), now); !errors.Is(err, backend.ErrDuplicateUsername) {
		t.Fatalf("duplicate CreateUser() error = %v, want ErrDuplicateUsername", err)
	}

	got, err := b.GetUser(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if got.Username != "alice" || got.Superuser {
		t.Fatalf("GetUser() = %+v", got)
	}

	if err := b.DeleteUser(ctx, "alice"); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if _, err := b.GetUser(ctx, "alice"); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("GetUser() after delete error = %v, want ErrUserNotFound", err)
	}
}

func testBackendKeyLifecycle(t *testing.T, newBackend func() backend.AuthBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := fixedNow()

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	expires := "2027-01-01T00:00:00Z"
	rawKey := testAPIKey("lifecyclekey000000000000000")
	created, err := b.CreateKey(ctx, "alice", "key-1", rawKey, api.IssueKeyRequest{
		Comment:   "laptop",
		ExpiresAt: &expires,
	}, now)
	if err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}
	if created.ID != "key-1" || created.Comment != "laptop" || created.CreatedAt != "2026-07-05T12:00:00Z" {
		t.Fatalf("CreateKey() = %+v", created)
	}
	if created.ExpiresAt == nil || *created.ExpiresAt != expires {
		t.Fatalf("CreateKey() expires_at = %+v", created.ExpiresAt)
	}

	page, err := b.ListKeys(ctx, "alice", api.ListKeysParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "key-1" {
		t.Fatalf("ListKeys() = %+v", page)
	}

	got, err := b.GetKey(ctx, "alice", "key-1")
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	if got.Comment != "laptop" {
		t.Fatalf("GetKey() = %+v", got)
	}

	updatedComment := "desktop"
	updated, err := b.UpdateKey(ctx, "alice", "key-1", api.UpdateKeyRequest{Comment: &updatedComment}, now)
	if err != nil {
		t.Fatalf("UpdateKey() error = %v", err)
	}
	if updated.Comment != "desktop" {
		t.Fatalf("UpdateKey() = %+v", updated)
	}

	username, err := b.LookupUserByKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("LookupUserByKey() error = %v", err)
	}
	if username != "alice" {
		t.Fatalf("LookupUserByKey() = %q, want alice", username)
	}

	revoked, err := b.RevokeKey(ctx, "alice", "key-1")
	if err != nil {
		t.Fatalf("RevokeKey() error = %v", err)
	}
	if revoked.ID != "key-1" || revoked.Comment != "desktop" {
		t.Fatalf("RevokeKey() = %+v", revoked)
	}

	if _, err := b.GetKey(ctx, "alice", "key-1"); !errors.Is(err, backend.ErrKeyNotFound) {
		t.Fatalf("GetKey() after revoke error = %v, want ErrKeyNotFound", err)
	}

	if _, err := b.LookupUserByKey(ctx, rawKey); !errors.Is(err, backend.ErrInvalidKey) {
		t.Fatalf("LookupUserByKey() after revoke error = %v, want ErrInvalidKey", err)
	}
}

func testBackendNotFoundErrors(t *testing.T, newBackend func() backend.AuthBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := fixedNow()

	if _, err := b.GetUser(ctx, "missing"); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("GetUser() error = %v, want ErrUserNotFound", err)
	}
	if err := b.DeleteUser(ctx, "missing"); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("DeleteUser() error = %v, want ErrUserNotFound", err)
	}
	if _, err := b.UpdateUser(ctx, "missing", api.UpdateUserRequest{}); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("UpdateUser() error = %v, want ErrUserNotFound", err)
	}
	if _, err := b.CreateKey(ctx, "missing", "key-1", "secret", api.IssueKeyRequest{Comment: "x"}, now); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("CreateKey() error = %v, want ErrUserNotFound", err)
	}

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := b.GetKey(ctx, "alice", "missing"); !errors.Is(err, backend.ErrKeyNotFound) {
		t.Fatalf("GetKey() error = %v, want ErrKeyNotFound", err)
	}
	if _, err := b.RevokeKey(ctx, "alice", "missing"); !errors.Is(err, backend.ErrKeyNotFound) {
		t.Fatalf("RevokeKey() error = %v, want ErrKeyNotFound", err)
	}
	if _, err := b.LookupUserByKey(ctx, testAPIKey("unknown000000000000000000")); !errors.Is(err, backend.ErrInvalidKey) {
		t.Fatalf("LookupUserByKey() error = %v, want ErrInvalidKey", err)
	}
}

func testBackendKeyNotStoredInPlaintext(t *testing.T, newBackend func() backend.AuthBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := fixedNow()

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	rawKey := testAPIKey("plaintextmustnotbestored00000")
	if _, err := b.CreateKey(ctx, "alice", "key-1", rawKey, api.IssueKeyRequest{Comment: "one"}, now); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	if _, err := b.LookupUserByKey(ctx, rawKey); err != nil {
		t.Fatalf("LookupUserByKey() error = %v", err)
	}
	if _, err := b.LookupUserByKey(ctx, testAPIKey("wrongkey000000000000000000")); !errors.Is(err, backend.ErrInvalidKey) {
		t.Fatalf("LookupUserByKey() wrong key error = %v, want ErrInvalidKey", err)
	}
}

func testBackendShutdown(t *testing.T, newBackend func() backend.AuthBackend) {
	t.Helper()
	b := newBackend()
	ctx := context.Background()

	if _, err := b.CreateUser(ctx, userReq("alice"), fixedNow()); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := b.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func testBackendListKeysSorted(t *testing.T, newBackend func() backend.AuthBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := fixedNow()

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	for _, id := range []string{"key-b", "key-a", "key-c"} {
		raw := testAPIKey(strings.ReplaceAll(id, "-", ""))
		if _, err := b.CreateKey(ctx, "alice", id, raw, api.IssueKeyRequest{Comment: id}, now); err != nil {
			t.Fatalf("CreateKey(%q) error = %v", id, err)
		}
	}

	page, err := b.ListKeys(ctx, "alice", api.ListKeysParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	if len(page.Items) != 3 {
		t.Fatalf("ListKeys() len = %d, want 3", len(page.Items))
	}
	want := []string{"key-a", "key-b", "key-c"}
	for i, key := range page.Items {
		if key.ID != want[i] {
			t.Fatalf("keys[%d].ID = %q, want %q", i, key.ID, want[i])
		}
	}
}

func testBackendListUsers(t *testing.T, newBackend func() backend.AuthBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := fixedNow()

	for _, username := range []string{"carol", "alice", "bob"} {
		if _, err := b.CreateUser(ctx, userReq(username), now); err != nil {
			t.Fatalf("CreateUser(%q) error = %v", username, err)
		}
	}

	page, err := b.ListUsers(ctx, api.ListUsersParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 3 || len(page.Items) != 3 {
		t.Fatalf("ListUsers() = %+v", page)
	}
	want := []string{"alice", "bob", "carol"}
	for i, user := range page.Items {
		if user.Username != want[i] {
			t.Fatalf("items[%d].Username = %q, want %q", i, user.Username, want[i])
		}
	}
}

func testBackendUpdateKeyExpiresAt(t *testing.T, newBackend func() backend.AuthBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := fixedNow()

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := b.CreateKey(ctx, "alice", "key-1", testAPIKey("updatekeyexpires000000000"), api.IssueKeyRequest{Comment: "x"}, now); err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	expires := "2028-01-01T00:00:00Z"
	updated, err := b.UpdateKey(ctx, "alice", "key-1", api.UpdateKeyRequest{ExpiresAt: ptrStringPtr(&expires)}, now)
	if err != nil {
		t.Fatalf("UpdateKey() error = %v", err)
	}
	if updated.ExpiresAt == nil || *updated.ExpiresAt != expires {
		t.Fatalf("UpdateKey() expires_at = %+v", updated.ExpiresAt)
	}

	var nilExpires *string
	updated, err = b.UpdateKey(ctx, "alice", "key-1", api.UpdateKeyRequest{ExpiresAt: &nilExpires}, now)
	if err != nil {
		t.Fatalf("UpdateKey() clear expires error = %v", err)
	}
	if updated.ExpiresAt != nil {
		t.Fatalf("UpdateKey() expires_at = %+v, want nil", updated.ExpiresAt)
	}
}

func testBackendSuperuser(t *testing.T, newBackend func() backend.AuthBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := fixedNow()

	created, err := b.CreateUser(ctx, api.UserCreateRequest{Username: "admin", Superuser: true}, now)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if !created.Superuser {
		t.Fatalf("CreateUser() superuser = false, want true")
	}

	got, err := b.GetUser(ctx, "admin")
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if !got.Superuser {
		t.Fatalf("GetUser() superuser = false, want true")
	}

	page, err := b.ListUsers(ctx, api.ListUsersParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(page.Items) != 1 || !page.Items[0].Superuser {
		t.Fatalf("ListUsers() = %+v", page)
	}

	superuser := false
	updated, err := b.UpdateUser(ctx, "admin", api.UpdateUserRequest{Superuser: &superuser})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if updated.Superuser {
		t.Fatalf("UpdateUser() superuser = true, want false")
	}

	got, err = b.GetUser(ctx, "admin")
	if err != nil {
		t.Fatalf("GetUser() after update error = %v", err)
	}
	if got.Superuser {
		t.Fatalf("GetUser() after update superuser = true, want false")
	}

	unchanged, err := b.UpdateUser(ctx, "admin", api.UpdateUserRequest{})
	if err != nil {
		t.Fatalf("UpdateUser() empty request error = %v", err)
	}
	if unchanged.Superuser {
		t.Fatalf("UpdateUser() empty request changed superuser to true")
	}
}
