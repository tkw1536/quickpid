// Package pidtest provides shared backend test suites.
//
//spellchecker:words pidtest
package pidtest

//spellchecker:words context errors strings testing time github quickpid backend internal apikey
import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/pid"
)

// RunStoreTests tests various store operations, without creating an external server.
func RunStoreTests(t *testing.T, newStore StoreFactory) {
	t.Helper()

	for _, test := range []struct {
		Name string
		Test func(t *testing.T, newStore StoreFactory)
	}{
		{
			Name: "AuthUserCRUD",
			Test: testAuthUserCRUD,
		},
		{
			Name: "AuthKeyLifecycle",
			Test: testAuthKeyLifecycle,
		},
		{
			Name: "AuthNotFoundErrors",
			Test: testAuthNotFoundErrors,
		},
		{
			Name: "AuthListKeysSorted",
			Test: testAuthListKeysSorted,
		},
		{
			Name: "AuthListUsers",
			Test: testAuthListUsers,
		},
		{
			Name: "AuthAutocompleteUsers",
			Test: testAuthAutocompleteUsers,
		},
		{
			Name: "AuthSuperuser",
			Test: testAuthSuperuser,
		},
		{
			Name: "AuthPasswordLifecycle",
			Test: testAuthPasswordLifecycle,
		},
		{
			Name: "AuthorizationCRUD",
			Test: testAuthorizationCRUD,
		},
		{
			Name: "ListUserRoles",
			Test: testListUserRoles,
		},
		{
			Name: "CreateNamespaceWithOwner",
			Test: testCreateNamespaceWithOwner,
		},
		{
			Name: "DeleteUserCascadesRoles",
			Test: testDeleteUserCascadesRoles,
		},
		{
			Name: "MountCRUD",
			Test: testMountCRUD,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			test.Test(t, newStore)
		})
	}
}

const testNamespaceOwner = "test-owner"

// testAPIKey returns a 32-character lowercase alphanumeric key for use in tests.
func testAPIKey(suffix string) string {
	if len(suffix) >= 32 {
		return suffix[:32]
	}
	return strings.Repeat("a", 32-len(suffix)) + suffix
}

// fixedNow returns a fixed clock for tests.
func fixedNow() func() time.Time {
	t := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func user(username string) api.ValidUsername {
	user, err := api.NewUsername(username)
	if err != nil {
		panic("cannot create user" + username)
	}
	return user
}

func userPtr(username string) *api.ValidUsername {
	u := user(username)
	return &u
}

func userReq(username string) api.ValidUserCreateRequest {
	return api.ValidUserCreateRequest{Username: user(username), Superuser: false}
}

func validPassword(t *testing.T, value string) api.ValidPassword {
	t.Helper()
	password, err := api.NewPassword(value)
	if err != nil {
		t.Fatalf("NewPassword(%q) error = %v", value, err)
	}
	return password
}

func namespaceReq() api.NamespaceCreateRequest {
	return api.NamespaceCreateRequest{
		Tags:      []string{"test-tag"},
		PIDFormat: pid.Format{Pattern: "***-***", Characters: pid.Full},
	}
}

// testAuthUserCRUD runs user CRUD tests against an auth backend.
func testAuthUserCRUD(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	b := newStore(t)
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

	got, err := b.GetUser(ctx, user("alice"))
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if got.Username != "alice" || got.Superuser {
		t.Fatalf("GetUser() = %+v", got)
	}

	if err := b.DeleteUser(ctx, user("alice")); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if _, err := b.GetUser(ctx, user("alice")); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("GetUser() after delete error = %v, want ErrUserNotFound", err)
	}
}

// testAuthKeyLifecycle runs API key lifecycle tests.
func testAuthKeyLifecycle(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	b := newStore(t)
	now := fixedNow()

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	expires := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	//spellchecker:words lifecyclekey
	rawKey := testAPIKey("lifecyclekey000000000000000")
	created, err := b.CreateKey(ctx, apikey.Default, user("alice"), "key-1", rawKey, api.KeyIssueRequest{
		Comment:   "laptop",
		ExpiresAt: &expires,
	}, now)
	if err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}
	if created.ID != "key-1" || created.Comment != "laptop" || !created.CreatedAt.Equal(now()) {
		t.Fatalf("CreateKey() = %+v", created)
	}
	if created.ExpiresAt == nil || !created.ExpiresAt.Equal(expires) {
		t.Fatalf("CreateKey() expires_at = %+v", created.ExpiresAt)
	}

	page, err := b.ListKeys(ctx, apikey.Default, user("alice"), api.ListKeysParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListKeys() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "key-1" {
		t.Fatalf("ListKeys() = %+v", page)
	}

	username, key, err := b.LookupUserByKey(ctx, apikey.Default, rawKey)
	if err != nil {
		t.Fatalf("LookupUserByKey() error = %v", err)
	}
	if username != "alice" {
		t.Fatalf("LookupUserByKey() = %q, want alice", username)
	}
	if key == nil || key.ID != "key-1" || key.Comment != "laptop" || !key.CreatedAt.Equal(now()) {
		t.Fatalf("LookupUserByKey() = %+v", key)
	}

	if err := b.RevokeKey(ctx, apikey.Default, user("alice"), "key-1"); err != nil {
		t.Fatalf("RevokeKey() error = %v", err)
	}

	if _, _, err := b.LookupUserByKey(ctx, apikey.Default, rawKey); !errors.Is(err, backend.ErrInvalidKey) {
		t.Fatalf("LookupUserByKey() after revoke error = %v, want ErrInvalidKey", err)
	}
}

// testAuthNotFoundErrors runs auth not-found error tests.
func testAuthNotFoundErrors(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	b := newStore(t)
	now := fixedNow()

	if _, err := b.GetUser(ctx, user("missing")); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("GetUser() error = %v, want ErrUserNotFound", err)
	}
	if err := b.DeleteUser(ctx, user("missing")); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("DeleteUser() error = %v, want ErrUserNotFound", err)
	}
	if _, err := b.UpdateUser(ctx, user("missing"), api.UserUpdateRequest{}); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("UpdateUser() error = %v, want ErrUserNotFound", err)
	}
	if _, err := b.CreateKey(ctx, apikey.Default, user("missing"), "key-1", "secret", api.KeyIssueRequest{Comment: "x"}, now); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("CreateKey() error = %v, want ErrUserNotFound", err)
	}

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if err := b.RevokeKey(ctx, apikey.Default, user("alice"), "missing"); !errors.Is(err, backend.ErrKeyNotFound) {
		t.Fatalf("RevokeKey() error = %v, want ErrKeyNotFound", err)
	}
	if _, _, err := b.LookupUserByKey(ctx, apikey.Default, testAPIKey("unknown000000000000000000")); !errors.Is(err, backend.ErrInvalidKey) {
		t.Fatalf("LookupUserByKey() error = %v, want ErrInvalidKey", err)
	}
}

// testAuthListKeysSorted runs list keys ordering tests.
func testAuthListKeysSorted(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	b := newStore(t)
	now := fixedNow()

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	for _, id := range []string{"key-b", "key-a", "key-c"} {
		raw := testAPIKey(strings.ReplaceAll(id, "-", ""))
		if _, err := b.CreateKey(ctx, apikey.Default, user("alice"), id, raw, api.KeyIssueRequest{Comment: id}, now); err != nil {
			t.Fatalf("CreateKey(%q) error = %v", id, err)
		}
	}

	page, err := b.ListKeys(ctx, apikey.Default, user("alice"), api.ListKeysParams{Limit: 100})
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

// testAuthListUsers runs list users tests.
func testAuthListUsers(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	b := newStore(t)
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

	secret := validPassword(t, "secret")
	if _, err := b.SetPassword(ctx, user("bob"), &secret); err != nil {
		t.Fatalf("SetPassword(bob) error = %v", err)
	}

	withPassword := true
	page, err = b.ListUsers(ctx, api.ListUsersParams{Password: &withPassword, Limit: 100})
	if err != nil {
		t.Fatalf("ListUsers(password=true) error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Username != "bob" || !page.Items[0].Password {
		t.Fatalf("ListUsers(password=true) = %+v, want [bob]", page)
	}

	withoutPassword := false
	page, err = b.ListUsers(ctx, api.ListUsersParams{Password: &withoutPassword, Limit: 100})
	if err != nil {
		t.Fatalf("ListUsers(password=false) error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("ListUsers(password=false) = %+v, want 2 users", page)
	}
	wantWithout := []string{"alice", "carol"}
	for i, listed := range page.Items {
		if listed.Username != wantWithout[i] || listed.Password {
			t.Fatalf("ListUsers(password=false) items[%d] = %+v, want %q without password", i, listed, wantWithout[i])
		}
	}
}

// testAuthAutocompleteUsers runs autocomplete users tests.
func testAuthAutocompleteUsers(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	b := newStore(t)
	now := fixedNow()

	for _, username := range []string{"alice", "alex", "bob", "carol"} {
		if _, err := b.CreateUser(ctx, userReq(username), now); err != nil {
			t.Fatalf("CreateUser(%q) error = %v", username, err)
		}
	}

	al, err := api.NewAutocompleteQuery("al")
	if err != nil {
		t.Fatalf("NewAutocompleteQuery() error = %v", err)
	}

	usernames, err := b.AutocompleteUsers(ctx, al, 10)
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

	usernames, err = b.AutocompleteUsers(ctx, al, 1)
	if err != nil {
		t.Fatalf("AutocompleteUsers() error = %v", err)
	}
	if len(usernames) != 1 || usernames[0] != "alex" {
		t.Fatalf("AutocompleteUsers() with limit 1 = %v, want [alex]", usernames)
	}

	zzz, err := api.NewAutocompleteQuery("zzz")
	if err != nil {
		t.Fatalf("NewAutocompleteQuery() error = %v", err)
	}

	usernames, err = b.AutocompleteUsers(ctx, zzz, 10)
	if err != nil {
		t.Fatalf("AutocompleteUsers() error = %v", err)
	}
	if len(usernames) != 0 {
		t.Fatalf("AutocompleteUsers() = %v, want []", usernames)
	}
}

// testAuthSuperuser runs superuser tests.
func testAuthSuperuser(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	b := newStore(t)
	now := fixedNow()

	created, err := b.CreateUser(ctx, api.ValidUserCreateRequest{Username: user("admin"), Superuser: true}, now)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if !created.Superuser {
		t.Fatalf("CreateUser() superuser = false, want true")
	}

	got, err := b.GetUser(ctx, user("admin"))
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if !got.Superuser {
		t.Fatalf("GetUser() superuser = false, want true")
	}

	superuser := false
	updated, err := b.UpdateUser(ctx, user("admin"), api.UserUpdateRequest{Superuser: &superuser})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if updated.Superuser {
		t.Fatalf("UpdateUser() superuser = true, want false")
	}
}

func testAuthPasswordLifecycle(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	b := newStore(t)
	now := fixedNow()

	if _, err := b.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	initial, err := b.GetUser(ctx, user("alice"))
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if initial.Password {
		t.Fatal("GetUser() password = true, want false")
	}

	secret := validPassword(t, "secret")
	hasPassword, err := b.SetPassword(ctx, user("alice"), &secret)
	if err != nil {
		t.Fatalf("SetPassword() error = %v", err)
	}
	if !hasPassword {
		t.Fatal("SetPassword() = false, want true")
	}

	updated, err := b.GetUser(ctx, user("alice"))
	if err != nil {
		t.Fatalf("GetUser() after set error = %v", err)
	}
	if !updated.Password {
		t.Fatal("GetUser() after set password = false, want true")
	}

	matches, err := b.CheckPassword(ctx, user("alice"), secret)
	if err != nil {
		t.Fatalf("CheckPassword() error = %v", err)
	}
	if !matches {
		t.Fatal("CheckPassword() = false, want true")
	}

	wrong := validPassword(t, "wrong")
	matches, err = b.CheckPassword(ctx, user("alice"), wrong)
	if err != nil {
		t.Fatalf("CheckPassword(wrong) error = %v", err)
	}
	if matches {
		t.Fatal("CheckPassword(wrong) = true, want false")
	}

	hasPassword, err = b.SetPassword(ctx, user("alice"), nil)
	if err != nil {
		t.Fatalf("SetPassword(nil) error = %v", err)
	}
	if hasPassword {
		t.Fatal("SetPassword(nil) = true, want false")
	}

	updated, err = b.GetUser(ctx, user("alice"))
	if err != nil {
		t.Fatalf("GetUser() after unset error = %v", err)
	}
	if updated.Password {
		t.Fatal("GetUser() after unset password = true, want false")
	}

	matches, err = b.CheckPassword(ctx, user("alice"), secret)
	if err != nil {
		t.Fatalf("CheckPassword() after unset error = %v", err)
	}
	if matches {
		t.Fatal("CheckPassword() after unset = true, want false")
	}

	if _, err := api.NewPassword(""); err == nil {
		t.Fatal("NewPassword(\"\") error = nil, want error")
	}
	if _, err := b.SetPassword(ctx, user("missing"), &secret); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("SetPassword(missing) error = %v, want ErrUserNotFound", err)
	}
	if _, err := b.CheckPassword(ctx, user("missing"), secret); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("CheckPassword(missing) error = %v, want ErrUserNotFound", err)
	}
}

// testAuthorizationCRUD runs namespace role CRUD tests.
func testAuthorizationCRUD(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	now := fixedNow()

	ns1, err := api.NewNamespaceID("ns-1")
	if err != nil {
		t.Fatalf("NewNamespaceID() error = %v", err)
	}

	if _, err := s.CreateUser(ctx, userReq(testNamespaceOwner), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := s.CreateNamespace(ctx, ns1, namespaceReq(), userPtr(testNamespaceOwner), now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	role, err := s.GetNamespaceRole(ctx, ns1, user("other"))
	if err != nil {
		t.Fatalf("GetNamespaceRole() error = %v", err)
	}
	if role != api.RoleNone {
		t.Fatalf("GetNamespaceRole() = %q, want none", role)
	}

	role, err = s.GetNamespaceRole(ctx, ns1, user(testNamespaceOwner))
	if err != nil {
		t.Fatalf("GetNamespaceRole(owner) error = %v", err)
	}
	if role != api.RoleManager {
		t.Fatalf("GetNamespaceRole(owner) = %q, want manager", role)
	}

	if err := s.SetNamespaceRole(ctx, ns1, user("bob"), api.RoleEditor); err != nil {
		t.Fatalf("SetNamespaceRole() error = %v", err)
	}
	role, err = s.GetNamespaceRole(ctx, ns1, user("bob"))
	if err != nil {
		t.Fatalf("GetNamespaceRole(bob) error = %v", err)
	}
	if role != api.RoleEditor {
		t.Fatalf("GetNamespaceRole(bob) = %q, want editor", role)
	}

	if err := s.SetNamespaceRole(ctx, ns1, user("bob"), api.RoleNone); err != nil {
		t.Fatalf("SetNamespaceRole(none) error = %v", err)
	}
	role, err = s.GetNamespaceRole(ctx, ns1, user("bob"))
	if err != nil {
		t.Fatalf("GetNamespaceRole(bob) after none error = %v", err)
	}
	if role != api.RoleNone {
		t.Fatalf("GetNamespaceRole(bob) after none = %q, want none", role)
	}

	if err := s.SetNamespaceRole(ctx, ns1, user("carol"), api.RoleContributor); err != nil {
		t.Fatalf("SetNamespaceRole(carol) error = %v", err)
	}
	if err := s.DeleteNamespaceRole(ctx, ns1, user("carol")); err != nil {
		t.Fatalf("DeleteNamespaceRole() error = %v", err)
	}
	if err := s.DeleteNamespaceRole(ctx, ns1, user("carol")); !errors.Is(err, backend.ErrRoleNotFound) {
		t.Fatalf("DeleteNamespaceRole() twice error = %v, want ErrRoleNotFound", err)
	}

	page, err := s.ListNamespaceRoles(ctx, ns1, api.ListNamespaceRolesParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListNamespaceRoles() error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Username != testNamespaceOwner {
		t.Fatalf("ListNamespaceRoles() = %+v", page)
	}

	if err := s.SetNamespaceRole(ctx, ns1, user("dave"), api.Role("invalid")); !errors.Is(err, backend.ErrInvalidRole) {
		t.Fatalf("SetNamespaceRole(invalid) error = %v, want ErrInvalidRole", err)
	}
}

// testListUserRoles runs list-roles-by-user tests.
func testListUserRoles(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	now := fixedNow()

	nsB, err := api.NewNamespaceID("ns-b")
	if err != nil {
		t.Fatalf("NewNamespaceID(ns-b) error = %v", err)
	}
	nsA, err := api.NewNamespaceID("ns-a")
	if err != nil {
		t.Fatalf("NewNamespaceID(ns-a) error = %v", err)
	}

	if _, err := s.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
	}
	if _, err := s.CreateUser(ctx, userReq("bob"), now); err != nil {
		t.Fatalf("CreateUser(bob) error = %v", err)
	}
	if _, err := s.CreateUser(ctx, userReq("carol"), now); err != nil {
		t.Fatalf("CreateUser(carol) error = %v", err)
	}

	if _, err := s.CreateNamespace(ctx, nsB, namespaceReq(), userPtr("alice"), now); err != nil {
		t.Fatalf("CreateNamespace(ns-b) error = %v", err)
	}
	if _, err := s.CreateNamespace(ctx, nsA, namespaceReq(), userPtr("alice"), now); err != nil {
		t.Fatalf("CreateNamespace(ns-a) error = %v", err)
	}
	if err := s.SetNamespaceRole(ctx, nsB, user("bob"), api.RoleEditor); err != nil {
		t.Fatalf("SetNamespaceRole(bob, ns-b) error = %v", err)
	}
	if err := s.SetNamespaceRole(ctx, nsA, user("bob"), api.RoleContributor); err != nil {
		t.Fatalf("SetNamespaceRole(bob, ns-a) error = %v", err)
	}

	page, err := s.ListUserRoles(ctx, user("alice"), api.ListUserRolesParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUserRoles(alice) error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("ListUserRoles(alice) = %+v, want 2 items", page)
	}
	if page.Items[0].Namespace != "ns-a" || page.Items[0].Role != api.RoleManager {
		t.Fatalf("ListUserRoles(alice)[0] = %+v, want ns-a manager", page.Items[0])
	}
	if page.Items[1].Namespace != "ns-b" || page.Items[1].Role != api.RoleManager {
		t.Fatalf("ListUserRoles(alice)[1] = %+v, want ns-b manager", page.Items[1])
	}

	page, err = s.ListUserRoles(ctx, user("bob"), api.ListUserRolesParams{Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("ListUserRoles(bob, limit=1) error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Namespace != "ns-a" || page.Items[0].Role != api.RoleContributor {
		t.Fatalf("ListUserRoles(bob, limit=1) = %+v", page)
	}

	page, err = s.ListUserRoles(ctx, user("bob"), api.ListUserRolesParams{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("ListUserRoles(bob, offset=1) error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Namespace != "ns-b" || page.Items[0].Role != api.RoleEditor {
		t.Fatalf("ListUserRoles(bob, offset=1) = %+v", page)
	}

	page, err = s.ListUserRoles(ctx, user("carol"), api.ListUserRolesParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUserRoles(carol) error = %v", err)
	}
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("ListUserRoles(carol) = %+v, want empty", page)
	}

	if _, err := s.ListUserRoles(ctx, user("missing"), api.ListUserRolesParams{Limit: 100}); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("ListUserRoles(missing) error = %v, want ErrUserNotFound", err)
	}
}

// testCreateNamespaceWithOwner runs namespace creation with owner tests.
func testCreateNamespaceWithOwner(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	now := fixedNow()

	nsMissingOwner, err := api.NewNamespaceID("ns-missing-owner")
	if err != nil {
		t.Fatalf("NewNamespaceID() error = %v", err)
	}
	nsOwned, err := api.NewNamespaceID("ns-owned")
	if err != nil {
		t.Fatalf("NewNamespaceID() error = %v", err)
	}

	if _, err := s.CreateNamespace(ctx, nsMissingOwner, namespaceReq(), userPtr("nobody"), now); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("CreateNamespace() missing owner error = %v, want ErrUserNotFound", err)
	}

	if _, err := s.CreateUser(ctx, userReq("owner"), now); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := s.CreateNamespace(ctx, nsOwned, namespaceReq(), userPtr("owner"), now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	role, err := s.GetNamespaceRole(ctx, nsOwned, user("owner"))
	if err != nil {
		t.Fatalf("GetNamespaceRole() error = %v", err)
	}
	if role != api.RoleManager {
		t.Fatalf("GetNamespaceRole() = %q, want manager", role)
	}
}

// testDeleteUserCascadesRoles runs user deletion cascade tests.
func testDeleteUserCascadesRoles(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	now := fixedNow()

	ns1, err := api.NewNamespaceID("ns-1")
	if err != nil {
		t.Fatalf("NewNamespaceID() error = %v", err)
	}

	if _, err := s.CreateUser(ctx, userReq("alice"), now); err != nil {
		t.Fatalf("CreateUser(alice) error = %v", err)
	}
	if _, err := s.CreateUser(ctx, userReq("bob"), now); err != nil {
		t.Fatalf("CreateUser(bob) error = %v", err)
	}
	if _, err := s.CreateNamespace(ctx, ns1, namespaceReq(), userPtr("alice"), now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}
	if err := s.SetNamespaceRole(ctx, ns1, user("bob"), api.RoleEditor); err != nil {
		t.Fatalf("SetNamespaceRole() error = %v", err)
	}

	if err := s.DeleteUser(ctx, user("bob")); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	role, err := s.GetNamespaceRole(ctx, ns1, user("bob"))
	if err != nil {
		t.Fatalf("GetNamespaceRole(bob) error = %v", err)
	}
	if role != api.RoleNone {
		t.Fatalf("GetNamespaceRole(bob) after delete = %q, want none", role)
	}

	page, err := s.ListNamespaceRoles(ctx, ns1, api.ListNamespaceRolesParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListNamespaceRoles() error = %v", err)
	}
	if page.Total != 1 || page.Items[0].Username != "alice" {
		t.Fatalf("ListNamespaceRoles() = %+v", page)
	}
}

// SeedNamespaceOwner creates the standard test owner user on a store.
func SeedNamespaceOwner(t *testing.T, s backend.AuthenticationBackend) {
	t.Helper()
	ctx := context.Background()
	if _, err := s.CreateUser(ctx, userReq(testNamespaceOwner), fixedNow()); err != nil {
		t.Fatalf("SeedNamespaceOwner() error = %v", err)
	}
}

// testMountCRUD runs namespace mount CRUD tests.
func testMountCRUD(t *testing.T, newStore StoreFactory) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	now := fixedNow()

	ns1, err := api.NewNamespaceID("ns-1")
	if err != nil {
		t.Fatalf("NewNamespaceID(ns-1) error = %v", err)
	}
	ns2, err := api.NewNamespaceID("ns-2")
	if err != nil {
		t.Fatalf("NewNamespaceID(ns-2) error = %v", err)
	}
	baseURI, err := api.NewBaseURI("https://example.com/prefix/")
	if err != nil {
		t.Fatalf("NewBaseURI() error = %v", err)
	}
	otherURI, err := api.NewBaseURI("https://other.example.com/")
	if err != nil {
		t.Fatalf("NewBaseURI(other) error = %v", err)
	}

	if _, err := s.CreateNamespace(ctx, ns1, namespaceReq(), nil, now); err != nil {
		t.Fatalf("CreateNamespace(ns-1) error = %v", err)
	}
	if _, err := s.CreateNamespace(ctx, ns2, namespaceReq(), nil, now); err != nil {
		t.Fatalf("CreateNamespace(ns-2) error = %v", err)
	}

	if _, err := s.GetMount(ctx, baseURI); !errors.Is(err, backend.ErrMountNotFound) {
		t.Fatalf("GetMount() empty error = %v, want ErrMountNotFound", err)
	}
	if err := s.DeleteMount(ctx, baseURI); !errors.Is(err, backend.ErrMountNotFound) {
		t.Fatalf("DeleteMount() empty error = %v, want ErrMountNotFound", err)
	}

	missingNS, err := api.NewNamespaceID("missing")
	if err != nil {
		t.Fatalf("NewNamespaceID(missing) error = %v", err)
	}
	if err := s.SetMount(ctx, baseURI, missingNS); !errors.Is(err, backend.ErrNamespaceNotFound) {
		t.Fatalf("SetMount(missing ns) error = %v, want ErrNamespaceNotFound", err)
	}

	if err := s.SetMount(ctx, baseURI, ns1); err != nil {
		t.Fatalf("SetMount(ns-1) error = %v", err)
	}
	got, err := s.GetMount(ctx, baseURI)
	if err != nil {
		t.Fatalf("GetMount() error = %v", err)
	}
	if got != ns1.String() {
		t.Fatalf("GetMount() = %q, want %q", got, ns1.String())
	}

	if err := s.SetMount(ctx, baseURI, ns2); err != nil {
		t.Fatalf("SetMount(upsert ns-2) error = %v", err)
	}
	got, err = s.GetMount(ctx, baseURI)
	if err != nil {
		t.Fatalf("GetMount() after upsert error = %v", err)
	}
	if got != ns2.String() {
		t.Fatalf("GetMount() after upsert = %q, want %q", got, ns2.String())
	}

	if err := s.SetMount(ctx, otherURI, ns1); err != nil {
		t.Fatalf("SetMount(otherURI) error = %v", err)
	}
	got, err = s.GetMount(ctx, otherURI)
	if err != nil {
		t.Fatalf("GetMount(otherURI) error = %v", err)
	}
	if got != ns1.String() {
		t.Fatalf("GetMount(otherURI) = %q, want %q", got, ns1.String())
	}

	page, err := s.ListMounts(ctx, api.ListMountsParams{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("ListMounts() error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("ListMounts() = %+v, want 2 items", page)
	}
	if page.Items[0].Namespace != ns1.String() || page.Items[0].BaseURI != otherURI.String() {
		t.Fatalf("ListMounts()[0] = %+v", page.Items[0])
	}
	if page.Items[1].Namespace != ns2.String() || page.Items[1].BaseURI != baseURI.String() {
		t.Fatalf("ListMounts()[1] = %+v", page.Items[1])
	}

	page, err = s.ListMounts(ctx, api.ListMountsParams{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("ListMounts(page) error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Namespace != ns2.String() {
		t.Fatalf("ListMounts(page) = %+v", page)
	}

	uris, err := s.ListNamespaceMounts(ctx, ns1, api.ListNamespaceMountsParams{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("ListNamespaceMounts(ns1) error = %v", err)
	}
	if uris.Total != 1 || len(uris.Items) != 1 || uris.Items[0] != otherURI.String() {
		t.Fatalf("ListNamespaceMounts(ns1) = %+v", uris)
	}

	uris, err = s.ListNamespaceMounts(ctx, ns2, api.ListNamespaceMountsParams{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("ListNamespaceMounts(ns2) error = %v", err)
	}
	if uris.Total != 1 || len(uris.Items) != 1 || uris.Items[0] != baseURI.String() {
		t.Fatalf("ListNamespaceMounts(ns2) = %+v", uris)
	}

	if _, err := s.ListNamespaceMounts(ctx, missingNS, api.ListNamespaceMountsParams{Limit: 10}); !errors.Is(err, backend.ErrNamespaceNotFound) {
		t.Fatalf("ListNamespaceMounts(missing) error = %v, want ErrNamespaceNotFound", err)
	}

	if err := s.DeleteMount(ctx, baseURI); err != nil {
		t.Fatalf("DeleteMount() error = %v", err)
	}
	if _, err := s.GetMount(ctx, baseURI); !errors.Is(err, backend.ErrMountNotFound) {
		t.Fatalf("GetMount() after delete error = %v, want ErrMountNotFound", err)
	}
	got, err = s.GetMount(ctx, otherURI)
	if err != nil {
		t.Fatalf("GetMount(otherURI) after delete error = %v", err)
	}
	if got != ns1.String() {
		t.Fatalf("GetMount(otherURI) after delete = %q, want %q", got, ns1.String())
	}
}
