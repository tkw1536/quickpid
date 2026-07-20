//spellchecker:words service
package service_test

//spellchecker:words context testing time github quickpid backend memory internal apikey service
import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/backend/memory"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/service"
)

var testNS api.ValidNamespaceID

func init() {
	ns, err := api.NewNamespaceID("test-ns")
	if err != nil {
		panic(err)
	}
	testNS = ns
}

type fixedRuntime struct {
	now time.Time
}

func (r *fixedRuntime) NewNamespaceID() (string, error)         { return testNS.String(), nil }
func (r *fixedRuntime) NewPID(pid.Format) (string, error)       { return "aaa-bbb", nil }
func (r *fixedRuntime) Now() time.Time                          { return r.now }
func (r *fixedRuntime) NewAPIKeyID() (string, error)            { return "api-key-id", nil }
func (r *fixedRuntime) NewAPIKey(apikey.Format) (string, error) { return "api-key", nil }

var (
	ownerUsername       api.ValidUsername
	editorUsername      api.ValidUsername
	contributorUsername api.ValidUsername
	readerUsername      api.ValidUsername
	someoneUsername     api.ValidUsername
)

func init() {
	var err error

	ownerUsername, err = api.NewUsername("owner")
	if err != nil {
		panic(err)
	}
	editorUsername, err = api.NewUsername("editor")
	if err != nil {
		panic(err)
	}
	contributorUsername, err = api.NewUsername("contributor")
	if err != nil {
		panic(err)
	}
	readerUsername, err = api.NewUsername("reader")
	if err != nil {
		panic(err)
	}
	someoneUsername, err = api.NewUsername("someone")
	if err != nil {
		panic(err)
	}
}

func newTestService(t *testing.T) (*service.Service, backend.Store) {
	t.Helper()
	store := memory.NewStore()
	ctx := t.Context()
	now := fixedNow()
	runtime := &fixedRuntime{now: now()}

	if _, err := store.CreateUser(ctx, api.ValidUserCreateRequest{Username: ownerUsername}, now); err != nil {
		t.Fatalf("CreateUser(owner) error = %v", err)
	}
	if _, err := store.CreateUser(ctx, api.ValidUserCreateRequest{Username: editorUsername}, now); err != nil {
		t.Fatalf("CreateUser(editor) error = %v", err)
	}
	if _, err := store.CreateUser(ctx, api.ValidUserCreateRequest{Username: contributorUsername}, now); err != nil {
		t.Fatalf("CreateUser(contributor) error = %v", err)
	}
	if _, err := store.CreateUser(ctx, api.ValidUserCreateRequest{Username: readerUsername}, now); err != nil {
		t.Fatalf("CreateUser(reader) error = %v", err)
	}

	req := api.NamespaceCreateRequest{
		Tag:       "tag",
		PIDFormat: pid.Format{Pattern: "***-***", Characters: pid.Full},
	}
	if _, err := store.CreateNamespace(ctx, testNS, req, &ownerUsername, now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}
	if err := store.SetNamespacePermission(ctx, testNS, editorUsername, api.PermissionLevelEditor); err != nil {
		t.Fatalf("SetNamespacePermission(editor) error = %v", err)
	}
	if err := store.SetNamespacePermission(ctx, testNS, contributorUsername, api.PermissionLevelContributor); err != nil {
		t.Fatalf("SetNamespacePermission(contributor) error = %v", err)
	}
	if err := store.SetNamespacePermission(ctx, testNS, readerUsername, api.PermissionLevelNone); err != nil {
		t.Fatalf("SetNamespacePermission(reader) error = %v", err)
	}

	return service.New(store, runtime, service.Options{}), store
}

func newAnonymousTestService(t *testing.T) (*service.Service, backend.Store) {
	t.Helper()
	store := memory.NewStore()
	now := fixedNow()
	runtime := &fixedRuntime{now: now()}
	return service.New(store, runtime, service.Options{Anonymous: true}), store
}

func userInfo(username string) *api.ValidUserInfo {
	valid, err := (&api.UserInfo{Username: username}).Validate()
	if err != nil {
		panic(fmt.Sprintf("NewValidUserInfo(%q) error = %v", username, err))
	}
	return &valid
}

func TestService_GetNamespacePermission(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	perm, err := svc.GetNamespacePermission(ctx, userInfo("reader"), testNS, readerUsername)
	if err != nil {
		t.Fatalf("GetNamespacePermission(self) = %v, %v", perm, err)
	}
	if perm.Level != api.PermissionLevelNone {
		t.Fatalf("level = %q, want none", perm.Level)
	}

	_, err = svc.GetNamespacePermission(ctx, userInfo("reader"), testNS, editorUsername)
	if !service.IsForbidden(err) {
		t.Fatalf("GetNamespacePermission(other) = %v, want forbidden", err)
	}

	perm, err = svc.GetNamespacePermission(ctx, userInfo("owner"), testNS, editorUsername)
	if err != nil {
		t.Fatalf("GetNamespacePermission(manager) = %v, %v", perm, err)
	}
	if perm.Level != api.PermissionLevelEditor {
		t.Fatalf("level = %q, want editor", perm.Level)
	}
}

func TestService_ListUserPermissions(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()
	now := fixedNow()

	ns2, err := api.NewNamespaceID("test-ns-2")
	if err != nil {
		t.Fatalf("NewNamespaceID() error = %v", err)
	}
	req := api.NamespaceCreateRequest{
		Tag:       "tag-2",
		PIDFormat: pid.Format{Pattern: "***-***", Characters: pid.Full},
	}
	if _, err := store.CreateNamespace(ctx, ns2, req, &ownerUsername, now); err != nil {
		t.Fatalf("CreateNamespace(ns2) error = %v", err)
	}
	if err := store.SetNamespacePermission(ctx, ns2, editorUsername, api.PermissionLevelContributor); err != nil {
		t.Fatalf("SetNamespacePermission(editor, ns2) error = %v", err)
	}

	page, err := svc.ListUserPermissions(ctx, userInfo("editor"), nil, api.ListUserPermissionsParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUserPermissions(self) = %v, %v", page, err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("ListUserPermissions(self) = %+v, want 2 items", page)
	}
	if page.Items[0].Namespace != testNS.String() || page.Items[0].Level != api.PermissionLevelEditor {
		t.Fatalf("ListUserPermissions(self)[0] = %+v", page.Items[0])
	}
	if page.Items[1].Namespace != ns2.String() || page.Items[1].Level != api.PermissionLevelContributor {
		t.Fatalf("ListUserPermissions(self)[1] = %+v", page.Items[1])
	}

	other := editorUsername
	_, err = svc.ListUserPermissions(ctx, userInfo("owner"), &other, api.ListUserPermissionsParams{Limit: 100})
	if !service.IsForbidden(err) {
		t.Fatalf("ListUserPermissions(other as non-superuser) = %v, want forbidden", err)
	}

	page, err = svc.ListUserPermissions(ctx, &api.ValidUserInfo{Username: ownerUsername, Superuser: true}, &other, api.ListUserPermissionsParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUserPermissions(superuser) = %v, %v", page, err)
	}
	if page.Total != 2 {
		t.Fatalf("ListUserPermissions(superuser) total = %d, want 2", page.Total)
	}

	missing, err := api.NewUsername("missing")
	if err != nil {
		t.Fatalf("NewUsername(missing) error = %v", err)
	}
	_, err = svc.ListUserPermissions(ctx, &api.ValidUserInfo{Username: ownerUsername, Superuser: true}, &missing, api.ListUserPermissionsParams{Limit: 100})
	if code, ok := api.GetErrorString(err); !ok || code != api.UserNotFound {
		t.Fatalf("ListUserPermissions(missing) = %v, want user_not_found", err)
	}

	_, err = svc.ListUserPermissions(ctx, nil, nil, api.ListUserPermissionsParams{Limit: 100})
	if !service.IsUnauthorized(err) {
		t.Fatalf("ListUserPermissions(nil caller) = %v, want unauthorized", err)
	}
}

func TestService_ResolverPermissions(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()
	now := fixedNow()

	_, err := svc.GetNamespace(ctx, userInfo("contributor"), testNS)
	if err != nil {
		t.Fatalf("contributor GetNamespace = %v", err)
	}

	_, err = svc.GetNamespace(ctx, userInfo("reader"), testNS)
	if !service.IsForbidden(err) {
		t.Fatalf("reader GetNamespace = %v, want forbidden", err)
	}
	code, ok := api.GetErrorString(err)
	if !ok || code != api.Forbidden {
		t.Fatalf("reader GetNamespace error code = %q, %v, want %q", code, ok, api.Forbidden)
	}
	existingPID, err := api.NewPID("existing")
	if err != nil {
		t.Fatalf("NewPID(existing) error = %v", err)
	}
	_, err = store.CreateResource(ctx, testNS, existingPID, api.ResourceCreateRequest{
		URL: "https://example.com",
	}, now)
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	_, err = svc.GetResource(ctx, userInfo("reader"), testNS, existingPID)
	if err != nil {
		t.Fatalf("reader GetResource = %v", err)
	}

	_, err = svc.GetResource(ctx, userInfo("contributor"), testNS, existingPID)
	if err != nil {
		t.Fatalf("contributor GetResource = %v", err)
	}

	_, err = svc.ListResources(ctx, userInfo("contributor"), testNS, api.ListResourcesParams{Limit: 10})
	if !service.IsForbidden(err) {
		t.Fatalf("contributor ListResources = %v, want forbidden", err)
	}
	code, ok = api.GetErrorString(err)
	if !ok || code != api.Forbidden {
		t.Fatalf("contributor ListResources error code = %q, %v, want %q", code, ok, api.Forbidden)
	}

	_, err = svc.CreateResource(ctx, userInfo("contributor"), testNS, api.ResourceCreateRequest{
		URL: "https://example.com/new",
	})
	if err != nil {
		t.Fatalf("contributor CreateResource = %v", err)
	}

	_, err = svc.SetNamespacePermission(ctx, userInfo("editor"), testNS, readerUsername, api.SetNamespacePermissionRequest{
		Level: api.PermissionLevelContributor,
	})
	if !service.IsForbidden(err) {
		t.Fatalf("editor SetNamespacePermission = %v, want forbidden", err)
	}
}

func TestService_SetNamespacePermissionCannotEscalate(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := t.Context()

	_, err := svc.SetNamespacePermission(ctx, userInfo("editor"), testNS, readerUsername, api.SetNamespacePermissionRequest{
		Level: api.PermissionLevelManager,
	})
	if !service.IsForbidden(err) {
		t.Fatalf("editor grant manager = %v, want forbidden", err)
	}
}

func TestService_SetNamespacePermissionSelfRules(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()

	_, err := svc.SetNamespacePermission(ctx, userInfo("owner"), testNS, ownerUsername, api.SetNamespacePermissionRequest{
		Level: api.PermissionLevelEditor,
	})
	if !service.IsForbidden(err) {
		t.Fatalf("owner modify own permission = %v, want forbidden", err)
	}

	level, err := store.GetNamespacePermission(ctx, testNS, ownerUsername)
	if err != nil {
		t.Fatalf("GetNamespacePermission(owner) error = %v", err)
	}
	if level != api.PermissionLevelManager {
		t.Fatalf("owner level after forbidden update = %q, want manager", level)
	}

	perm, err := svc.SetNamespacePermission(ctx, &api.ValidUserInfo{Username: ownerUsername, Superuser: true}, testNS, ownerUsername, api.SetNamespacePermissionRequest{
		Level: api.PermissionLevelEditor,
	})
	if err != nil {
		t.Fatalf("superuser modify own permission = %v, %v", perm, err)
	}
	if perm.Level != api.PermissionLevelEditor {
		t.Fatalf("superuser level = %q, want editor", perm.Level)
	}
}

func TestService_DeleteNamespacePermissionSelfRules(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()

	err := svc.DeleteNamespacePermission(ctx, userInfo("owner"), testNS, ownerUsername)
	if !service.IsForbidden(err) {
		t.Fatalf("owner delete own permission = %v, want forbidden", err)
	}

	level, err := store.GetNamespacePermission(ctx, testNS, ownerUsername)
	if err != nil {
		t.Fatalf("GetNamespacePermission(owner) error = %v", err)
	}
	if level != api.PermissionLevelManager {
		t.Fatalf("owner level after forbidden delete = %q, want manager", level)
	}

	err = svc.DeleteNamespacePermission(ctx, &api.ValidUserInfo{Username: ownerUsername, Superuser: true}, testNS, ownerUsername)
	if err != nil {
		t.Fatalf("superuser delete own permission = %v", err)
	}

	level, err = store.GetNamespacePermission(ctx, testNS, ownerUsername)
	if err != nil {
		t.Fatalf("GetNamespacePermission(owner) error = %v", err)
	}
	if level != api.PermissionLevelNone {
		t.Fatalf("owner level after delete = %q, want none", level)
	}
}

func TestService_Unauthenticated(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := t.Context()

	_, err := svc.ListNamespaces(ctx, nil, api.ListNamespacesParams{})
	if !service.IsUnauthorized(err) {
		t.Fatalf("ListNamespaces(nil) = %v, want unauthorized", err)
	}
}

func TestService_AnonymousResolverMode(t *testing.T) {
	t.Parallel()

	svc, store := newAnonymousTestService(t)
	ctx := t.Context()

	req := api.NamespaceCreateRequest{
		Tag:       "anon-tag",
		PIDFormat: pid.Format{Pattern: "***-***", Characters: pid.Full},
	}
	ns, err := svc.CreateNamespace(ctx, nil, req)
	if err != nil {
		t.Fatalf("CreateNamespace(nil) = %v, %v", ns, err)
	}
	if ns.ID != testNS.String() {
		t.Fatalf("namespace id = %q, want test-ns", ns.ID)
	}

	perm, err := store.GetNamespacePermission(ctx, testNS, someoneUsername)
	if err != nil {
		t.Fatalf("GetNamespacePermission() error = %v", err)
	}
	if perm != api.PermissionLevelNone {
		t.Fatalf("GetNamespacePermission() = %q, want none", perm)
	}

	page, err := svc.ListNamespaces(ctx, nil, api.ListNamespacesParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListNamespaces(nil) = %v, %v", page, err)
	}
	if page.Total != 1 {
		t.Fatalf("ListNamespaces() total = %d, want 1", page.Total)
	}

	created, err := svc.CreateResource(ctx, nil, testNS, api.ResourceCreateRequest{
		URL: "https://example.com/anon",
	})
	if err != nil {
		t.Fatalf("CreateResource(nil) = %v, %v", created, err)
	}
	if created.PID != "aaa-bbb" {
		t.Fatalf("CreateResource() pid = %q, want aaa-bbb", created.PID)
	}

	aaaBBB, err := api.NewPID("aaa-bbb")
	if err != nil {
		t.Fatalf("NewPID(aaa-bbb) error = %v", err)
	}
	got, err := svc.GetResource(ctx, nil, testNS, aaaBBB)
	if err != nil {
		t.Fatalf("GetResource(nil) = %v, %v", got, err)
	}
	if got.URL != "https://example.com/anon" {
		t.Fatalf("GetResource() url = %q, want https://example.com/anon", got.URL)
	}

	count, err := svc.CountAllResources(ctx)
	if err != nil {
		t.Fatalf("CountAllResources(nil) = %v, %v", count, err)
	}
	if count.Total != 1 {
		t.Fatalf("CountAllResources() total = %d, want 1", count.Total)
	}
}

func TestService_AnonymousDisablesUserAndPermissionManagement(t *testing.T) {
	t.Parallel()

	svc, _ := newAnonymousTestService(t)
	ctx := t.Context()

	if _, err := svc.Authenticate(ctx, "anything"); !service.IsUnauthorized(err) {
		t.Fatalf("Authenticate() error = %v, want unauthorized", err)
	}
	if _, err := svc.CurrentUser(ctx, "anything"); !service.IsUnauthorized(err) {
		t.Fatalf("CurrentUser() error = %v, want unauthorized", err)
	}

	aliceUsername, err := api.NewUsername("alice")
	if err != nil {
		t.Fatalf("NewUsername(alice) error = %v", err)
	}
	_, err = svc.CreateUser(ctx, userInfo("owner"), api.ValidUserCreateRequest{Username: aliceUsername})
	if !service.IsUnavailableInAnonymousMode(err) {
		t.Fatalf("CreateUser() = %v, want anonymous mode unavailable", err)
	}
	if code, ok := api.GetErrorString(err); !ok || code != api.UnavailableInAnonymousMode {
		t.Fatalf("CreateUser() error code = %q, %v, want %q", code, ok, api.UnavailableInAnonymousMode)
	}
	if _, err := svc.ListUsers(ctx, userInfo("owner"), api.ListUsersParams{Limit: 10}); !service.IsUnavailableInAnonymousMode(err) {
		t.Fatalf("ListUsers() = %v, want anonymous mode unavailable", err)
	}
	if _, err := svc.GetNamespacePermission(ctx, userInfo("owner"), testNS, ownerUsername); !service.IsUnavailableInAnonymousMode(err) {
		t.Fatalf("GetNamespacePermission() = %v, want anonymous mode unavailable", err)
	}
	if _, err := svc.ListUserPermissions(ctx, userInfo("owner"), nil, api.ListUserPermissionsParams{Limit: 10}); !service.IsUnavailableInAnonymousMode(err) {
		t.Fatalf("ListUserPermissions() = %v, want anonymous mode unavailable", err)
	}
}
