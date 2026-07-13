// Package storetest provides shared backend test suites.
//
//spellchecker:words storetest
package storetest

//spellchecker:words context errors strings testing time github bicpid backend internal apikey
import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/apikey"
	"github.com/tkw1536/bicpid/pid"
)

const TestNamespaceOwner = "test-owner"

// TestAPIKey returns a 32-character lowercase alphanumeric key for use in tests.
func TestAPIKey(suffix string) string {
	if len(suffix) >= 32 {
		return suffix[:32]
	}
	return strings.Repeat("a", 32-len(suffix)) + suffix
}

// FixedNow returns a fixed clock for tests.
func FixedNow() func() time.Time {
	t := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func userReq(username string) api.UserCreateRequest {
	return api.UserCreateRequest{Username: username}
}

func namespaceReq() api.NamespaceCreateRequest {
	return api.NamespaceCreateRequest{
		Tag:       "test-tag",
		PIDFormat: pid.Format{Pattern: "***-***", Characters: pid.Full},
	}
}

// RunAuthUserCRUD runs user CRUD tests against an auth backend.
func RunAuthUserCRUD(t *testing.T, newBackend func() backend.AuthenticationBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := FixedNow()

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

// RunAuthKeyLifecycle runs API key lifecycle tests.
func RunAuthKeyLifecycle(t *testing.T, newBackend func() backend.AuthenticationBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := FixedNow()

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	expires := "2027-01-01T00:00:00Z"
	rawKey := TestAPIKey("lifecyclekey000000000000000")
	created, err := b.CreateKey(ctx, apikey.Default, "alice", "key-1", rawKey, api.KeyIssueRequest{
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

	page, err := b.ListKeys(ctx, apikey.Default, "alice", api.ListKeysParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "key-1" {
		t.Fatalf("ListKeys() = %+v", page)
	}

	got, err := b.GetKey(ctx, apikey.Default, "alice", "key-1")
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	if got.Comment != "laptop" {
		t.Fatalf("GetKey() = %+v", got)
	}

	updatedComment := "desktop"
	updated, err := b.UpdateKey(ctx, apikey.Default, "alice", "key-1", api.KeyUpdateRequest{Comment: &updatedComment}, now)
	if err != nil {
		t.Fatalf("UpdateKey() error = %v", err)
	}
	if updated.Comment != "desktop" {
		t.Fatalf("UpdateKey() = %+v", updated)
	}

	username, err := b.LookupUserByKey(ctx, apikey.Default, rawKey)
	if err != nil {
		t.Fatalf("LookupUserByKey() error = %v", err)
	}
	if username != "alice" {
		t.Fatalf("LookupUserByKey() = %q, want alice", username)
	}

	revoked, err := b.RevokeKey(ctx, apikey.Default, "alice", "key-1")
	if err != nil {
		t.Fatalf("RevokeKey() error = %v", err)
	}
	if revoked.ID != "key-1" || revoked.Comment != "desktop" {
		t.Fatalf("RevokeKey() = %+v", revoked)
	}

	if _, err := b.GetKey(ctx, apikey.Default, "alice", "key-1"); !errors.Is(err, backend.ErrKeyNotFound) {
		t.Fatalf("GetKey() after revoke error = %v, want ErrKeyNotFound", err)
	}

	if _, err := b.LookupUserByKey(ctx, apikey.Default, rawKey); !errors.Is(err, backend.ErrInvalidKey) {
		t.Fatalf("LookupUserByKey() after revoke error = %v, want ErrInvalidKey", err)
	}
}

// RunAuthNotFoundErrors runs auth not-found error tests.
func RunAuthNotFoundErrors(t *testing.T, newBackend func() backend.AuthenticationBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := FixedNow()

	if _, err := b.GetUser(ctx, "missing"); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("GetUser() error = %v, want ErrUserNotFound", err)
	}
	if err := b.DeleteUser(ctx, "missing"); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("DeleteUser() error = %v, want ErrUserNotFound", err)
	}
	if _, err := b.UpdateUser(ctx, "missing", api.UserUpdateRequest{}); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("UpdateUser() error = %v, want ErrUserNotFound", err)
	}
	if _, err := b.CreateKey(ctx, apikey.Default, "missing", "key-1", "secret", api.KeyIssueRequest{Comment: "x"}, now); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("CreateKey() error = %v, want ErrUserNotFound", err)
	}

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := b.GetKey(ctx, apikey.Default, "alice", "missing"); !errors.Is(err, backend.ErrKeyNotFound) {
		t.Fatalf("GetKey() error = %v, want ErrKeyNotFound", err)
	}
	if _, err := b.RevokeKey(ctx, apikey.Default, "alice", "missing"); !errors.Is(err, backend.ErrKeyNotFound) {
		t.Fatalf("RevokeKey() error = %v, want ErrKeyNotFound", err)
	}
	if _, err := b.LookupUserByKey(ctx, apikey.Default, TestAPIKey("unknown000000000000000000")); !errors.Is(err, backend.ErrInvalidKey) {
		t.Fatalf("LookupUserByKey() error = %v, want ErrInvalidKey", err)
	}
}

// RunAuthShutdown runs shutdown tests.
func RunAuthShutdown(t *testing.T, newBackend func() backend.AuthenticationBackend) {
	t.Helper()
	b := newBackend()
	ctx := context.Background()

	if _, err := b.CreateUser(ctx, userReq("alice"), FixedNow()); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := b.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

// RunAuthListKeysSorted runs list keys ordering tests.
func RunAuthListKeysSorted(t *testing.T, newBackend func() backend.AuthenticationBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := FixedNow()

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	for _, id := range []string{"key-b", "key-a", "key-c"} {
		raw := TestAPIKey(strings.ReplaceAll(id, "-", ""))
		if _, err := b.CreateKey(ctx, apikey.Default, "alice", id, raw, api.KeyIssueRequest{Comment: id}, now); err != nil {
			t.Fatalf("CreateKey(%q) error = %v", id, err)
		}
	}

	page, err := b.ListKeys(ctx, apikey.Default, "alice", api.ListKeysParams{Limit: 100})
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

// RunAuthListUsers runs list users tests.
func RunAuthListUsers(t *testing.T, newBackend func() backend.AuthenticationBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := FixedNow()

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

// RunAuthAutocompleteUsers runs autocomplete users tests.
func RunAuthAutocompleteUsers(t *testing.T, newBackend func() backend.AuthenticationBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := FixedNow()

	for _, username := range []string{"alice", "alex", "bob", "carol"} {
		if _, err := b.CreateUser(ctx, userReq(username), now); err != nil {
			t.Fatalf("CreateUser(%q) error = %v", username, err)
		}
	}

	usernames, err := b.AutocompleteUsers(ctx, "al", 10)
	if err != nil {
		t.Fatalf("AutocompleteUsers() error = %v", err)
	}
	want := []string{"alex", "alice"}
	if len(usernames) != len(want) {
		t.Fatalf("AutocompleteUsers() = %v, want %v", usernames, want)
	}
	for i, username := range usernames {
		if username != want[i] {
			t.Fatalf("AutocompleteUsers()[%d] = %q, want %q", i, username, want[i])
		}
	}

	usernames, err = b.AutocompleteUsers(ctx, "al", 1)
	if err != nil {
		t.Fatalf("AutocompleteUsers() error = %v", err)
	}
	if len(usernames) != 1 || usernames[0] != "alex" {
		t.Fatalf("AutocompleteUsers() with limit 1 = %v, want [alex]", usernames)
	}

	usernames, err = b.AutocompleteUsers(ctx, "zzz", 10)
	if err != nil {
		t.Fatalf("AutocompleteUsers() error = %v", err)
	}
	if len(usernames) != 0 {
		t.Fatalf("AutocompleteUsers() = %v, want []", usernames)
	}
}

// RunAuthSuperuser runs superuser tests.
func RunAuthSuperuser(t *testing.T, newBackend func() backend.AuthenticationBackend) {
	t.Helper()
	ctx := context.Background()
	b := newBackend()
	now := FixedNow()

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

	superuser := false
	updated, err := b.UpdateUser(ctx, "admin", api.UserUpdateRequest{Superuser: &superuser})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if updated.Superuser {
		t.Fatalf("UpdateUser() superuser = true, want false")
	}
}

// RunAuthorizationCRUD runs namespace permission CRUD tests.
func RunAuthorizationCRUD(t *testing.T, newStore func() backend.Store) {
	t.Helper()
	ctx := context.Background()
	s := newStore()
	now := FixedNow()

	if _, err := s.CreateUser(ctx, userReq(TestNamespaceOwner), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := s.CreateNamespace(ctx, "ns-1", namespaceReq(), new(TestNamespaceOwner), now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	level, err := s.GetNamespacePermission(ctx, "ns-1", "other")
	if err != nil {
		t.Fatalf("GetNamespacePermission() error = %v", err)
	}
	if level != api.PermissionLevelNone {
		t.Fatalf("GetNamespacePermission() = %q, want none", level)
	}

	level, err = s.GetNamespacePermission(ctx, "ns-1", TestNamespaceOwner)
	if err != nil {
		t.Fatalf("GetNamespacePermission(owner) error = %v", err)
	}
	if level != api.PermissionLevelManager {
		t.Fatalf("GetNamespacePermission(owner) = %q, want manager", level)
	}

	if err := s.SetNamespacePermission(ctx, "ns-1", "bob", api.PermissionLevelEditor); err != nil {
		t.Fatalf("SetNamespacePermission() error = %v", err)
	}
	level, err = s.GetNamespacePermission(ctx, "ns-1", "bob")
	if err != nil {
		t.Fatalf("GetNamespacePermission(bob) error = %v", err)
	}
	if level != api.PermissionLevelEditor {
		t.Fatalf("GetNamespacePermission(bob) = %q, want editor", level)
	}

	if err := s.SetNamespacePermission(ctx, "ns-1", "bob", api.PermissionLevelNone); err != nil {
		t.Fatalf("SetNamespacePermission(none) error = %v", err)
	}
	level, err = s.GetNamespacePermission(ctx, "ns-1", "bob")
	if err != nil {
		t.Fatalf("GetNamespacePermission(bob) after none error = %v", err)
	}
	if level != api.PermissionLevelNone {
		t.Fatalf("GetNamespacePermission(bob) after none = %q, want none", level)
	}

	if err := s.SetNamespacePermission(ctx, "ns-1", "carol", api.PermissionLevelContributor); err != nil {
		t.Fatalf("SetNamespacePermission(carol) error = %v", err)
	}
	if err := s.DeleteNamespacePermission(ctx, "ns-1", "carol"); err != nil {
		t.Fatalf("DeleteNamespacePermission() error = %v", err)
	}
	if err := s.DeleteNamespacePermission(ctx, "ns-1", "carol"); !errors.Is(err, backend.ErrPermissionNotFound) {
		t.Fatalf("DeleteNamespacePermission() twice error = %v, want ErrPermissionNotFound", err)
	}

	page, err := s.ListNamespacePermissions(ctx, "ns-1", api.ListNamespacePermissionsParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListNamespacePermissions() error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Username != TestNamespaceOwner {
		t.Fatalf("ListNamespacePermissions() = %+v", page)
	}

	if err := s.SetNamespacePermission(ctx, "ns-1", "dave", api.PermissionLevel("invalid")); !errors.Is(err, backend.ErrInvalidPermissionLevel) {
		t.Fatalf("SetNamespacePermission(invalid) error = %v, want ErrInvalidPermissionLevel", err)
	}
}

// RunCreateNamespaceWithOwner runs namespace creation with owner tests.
func RunCreateNamespaceWithOwner(t *testing.T, newStore func() backend.Store) {
	t.Helper()
	ctx := context.Background()
	s := newStore()
	now := FixedNow()

	if _, err := s.CreateNamespace(ctx, "ns-missing-owner", namespaceReq(), new("nobody"), now); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("CreateNamespace() missing owner error = %v, want ErrUserNotFound", err)
	}

	if _, err := s.CreateUser(ctx, userReq("owner"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := s.CreateNamespace(ctx, "ns-owned", namespaceReq(), new("owner"), now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	level, err := s.GetNamespacePermission(ctx, "ns-owned", "owner")
	if err != nil {
		t.Fatalf("GetNamespacePermission() error = %v", err)
	}
	if level != api.PermissionLevelManager {
		t.Fatalf("GetNamespacePermission() = %q, want manager", level)
	}
}

// RunDeleteUserCascadesPermissions runs user deletion cascade tests.
func RunDeleteUserCascadesPermissions(t *testing.T, newStore func() backend.Store) {
	t.Helper()
	ctx := context.Background()
	s := newStore()
	now := FixedNow()

	if _, err := s.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
	}
	if _, err := s.CreateUser(ctx, userReq("bob"), now); err != nil {
		t.Fatalf("CreateUser(bob) error = %v", err)
	}
	if _, err := s.CreateNamespace(ctx, "ns-1", namespaceReq(), new("alice"), now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}
	if err := s.SetNamespacePermission(ctx, "ns-1", "bob", api.PermissionLevelEditor); err != nil {
		t.Fatalf("SetNamespacePermission() error = %v", err)
	}

	if err := s.DeleteUser(ctx, "bob"); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	level, err := s.GetNamespacePermission(ctx, "ns-1", "bob")
	if err != nil {
		t.Fatalf("GetNamespacePermission(bob) error = %v", err)
	}
	if level != api.PermissionLevelNone {
		t.Fatalf("GetNamespacePermission(bob) after delete = %q, want none", level)
	}

	page, err := s.ListNamespacePermissions(ctx, "ns-1", api.ListNamespacePermissionsParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListNamespacePermissions() error = %v", err)
	}
	if page.Total != 1 || page.Items[0].Username != "alice" {
		t.Fatalf("ListNamespacePermissions() = %+v", page)
	}
}

// SeedNamespaceOwner creates the standard test owner user on a store.
func SeedNamespaceOwner(t *testing.T, s backend.AuthenticationBackend) {
	t.Helper()
	ctx := context.Background()
	if _, err := s.CreateUser(ctx, userReq(TestNamespaceOwner), FixedNow()); err != nil {
		t.Fatalf("SeedNamespaceOwner() error = %v", err)
	}
}
