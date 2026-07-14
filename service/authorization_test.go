//spellchecker:words service
package service_test

//spellchecker:words context errors testing time github bicpid backend memory storetest internal apikey
import (
	"context"
	"testing"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/memory"
	"github.com/tkw1536/bicpid/backend/storetest"
	"github.com/tkw1536/bicpid/internal/apikey"
	"github.com/tkw1536/bicpid/pid"
	"github.com/tkw1536/bicpid/service"
)

type fixedRuntime struct {
	now time.Time
}

func (r *fixedRuntime) NewNamespaceID() (string, error)         { return "test-ns", nil }
func (r *fixedRuntime) NewPID(pid.Format) (string, error)       { return "aaa-bbb", nil }
func (r *fixedRuntime) Now() time.Time                          { return r.now }
func (r *fixedRuntime) NewAPIKeyID() (string, error)            { return "api-key-id", nil }
func (r *fixedRuntime) NewAPIKey(apikey.Format) (string, error) { return "api-key", nil }

func newTestService(t *testing.T) (*service.Service, backend.Store) {
	t.Helper()
	store := memory.NewStore()
	ctx := t.Context()
	now := storetest.FixedNow()
	runtime := &fixedRuntime{now: now()}

	if _, err := store.CreateUser(ctx, api.UserCreateRequest{Username: "owner"}, now); err != nil {
		t.Fatalf("CreateUser(owner) error = %v", err)
	}
	if _, err := store.CreateUser(ctx, api.UserCreateRequest{Username: "editor"}, now); err != nil {
		t.Fatalf("CreateUser(editor) error = %v", err)
	}
	if _, err := store.CreateUser(ctx, api.UserCreateRequest{Username: "contributor"}, now); err != nil {
		t.Fatalf("CreateUser(contributor) error = %v", err)
	}
	if _, err := store.CreateUser(ctx, api.UserCreateRequest{Username: "reader"}, now); err != nil {
		t.Fatalf("CreateUser(reader) error = %v", err)
	}

	req := api.NamespaceCreateRequest{
		Tag:       "tag",
		PIDFormat: pid.Format{Pattern: "***-***", Characters: pid.Full},
	}
	if _, err := store.CreateNamespace(ctx, "test-ns", req, new("owner"), now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}
	if err := store.SetNamespacePermission(ctx, "test-ns", "editor", api.PermissionLevelEditor); err != nil {
		t.Fatalf("SetNamespacePermission(editor) error = %v", err)
	}
	if err := store.SetNamespacePermission(ctx, "test-ns", "contributor", api.PermissionLevelContributor); err != nil {
		t.Fatalf("SetNamespacePermission(contributor) error = %v", err)
	}
	if err := store.SetNamespacePermission(ctx, "test-ns", "reader", api.PermissionLevelNone); err != nil {
		t.Fatalf("SetNamespacePermission(reader) error = %v", err)
	}

	return service.New(store, runtime, service.Options{}), store
}

func newAnonymousTestService(t *testing.T) (*service.Service, backend.Store) {
	t.Helper()
	store := memory.NewStore()
	now := storetest.FixedNow()
	runtime := &fixedRuntime{now: now()}
	return service.New(store, runtime, service.Options{Anonymous: true}), store
}

func userInfo(username string) *api.UserInfo {
	return &api.UserInfo{Username: username}
}

func TestService_GetNamespacePermission(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	perm, specError, err := svc.GetNamespacePermission(ctx, userInfo("reader"), "test-ns", "reader")
	if err != nil || specError != "" {
		t.Fatalf("GetNamespacePermission(self) = %v, %q, %v", perm, specError, err)
	}
	if perm.Level != api.PermissionLevelNone {
		t.Fatalf("level = %q, want none", perm.Level)
	}

	_, specError, err = svc.GetNamespacePermission(ctx, userInfo("reader"), "test-ns", "editor")
	if !service.IsForbidden(err) || specError != "" {
		t.Fatalf("GetNamespacePermission(other) = %q, %v, want forbidden", specError, err)
	}

	perm, specError, err = svc.GetNamespacePermission(ctx, userInfo("owner"), "test-ns", "editor")
	if err != nil || specError != "" {
		t.Fatalf("GetNamespacePermission(manager) = %v, %q, %v", perm, specError, err)
	}
	if perm.Level != api.PermissionLevelEditor {
		t.Fatalf("level = %q, want editor", perm.Level)
	}
}

func TestService_ResolverPermissions(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()
	now := storetest.FixedNow()

	_, specError, err := svc.GetNamespace(ctx, userInfo("contributor"), "test-ns")
	if err != nil || specError != "" {
		t.Fatalf("contributor GetNamespace = %q, %v", specError, err)
	}

	_, specError, err = svc.GetNamespace(ctx, userInfo("reader"), "test-ns")
	if !service.IsForbidden(err) || specError != api.Forbidden {
		t.Fatalf("reader GetNamespace = %q, %v, want forbidden", specError, err)
	}

	_, err = store.CreateResource(ctx, "test-ns", "existing", api.ResourceCreateRequest{
		URL: "https://example.com",
	}, now)
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}

	_, specError, err = svc.GetResource(ctx, userInfo("reader"), "test-ns", "existing")
	if err != nil || specError != "" {
		t.Fatalf("reader GetResource = %q, %v", specError, err)
	}

	_, specError, err = svc.GetResource(ctx, userInfo("contributor"), "test-ns", "existing")
	if err != nil || specError != "" {
		t.Fatalf("contributor GetResource = %q, %v", specError, err)
	}

	_, specError, err = svc.ListResources(ctx, userInfo("contributor"), api.ListResourcesParams{Namespace: "test-ns", Limit: 10})
	if !service.IsForbidden(err) || specError != api.Forbidden {
		t.Fatalf("contributor ListResources = %q, %v, want forbidden", specError, err)
	}

	_, specError, err = svc.CreateResource(ctx, userInfo("contributor"), "test-ns", api.ResourceCreateRequest{
		URL: "https://example.com/new",
	})
	if err != nil || specError != "" {
		t.Fatalf("contributor CreateResource = %q, %v", specError, err)
	}

	_, specError, err = svc.SetNamespacePermission(ctx, userInfo("editor"), "test-ns", "reader", api.SetNamespacePermissionRequest{
		Level: api.PermissionLevelContributor,
	})
	if !service.IsForbidden(err) || specError != "" {
		t.Fatalf("editor SetNamespacePermission = %q, %v, want forbidden", specError, err)
	}
}

func TestService_SetNamespacePermissionCannotEscalate(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := t.Context()

	_, specError, err := svc.SetNamespacePermission(ctx, userInfo("editor"), "test-ns", "reader", api.SetNamespacePermissionRequest{
		Level: api.PermissionLevelManager,
	})
	if !service.IsForbidden(err) || specError != "" {
		t.Fatalf("editor grant manager = %q, %v, want forbidden", specError, err)
	}
}

func TestService_SetNamespacePermissionSelfRules(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()

	_, specError, err := svc.SetNamespacePermission(ctx, userInfo("owner"), "test-ns", "owner", api.SetNamespacePermissionRequest{
		Level: api.PermissionLevelEditor,
	})
	if !service.IsForbidden(err) || specError != "" {
		t.Fatalf("owner modify own permission = %q, %v, want forbidden", specError, err)
	}

	level, err := store.GetNamespacePermission(ctx, "test-ns", "owner")
	if err != nil {
		t.Fatalf("GetNamespacePermission(owner) error = %v", err)
	}
	if level != api.PermissionLevelManager {
		t.Fatalf("owner level after forbidden update = %q, want manager", level)
	}

	perm, specError, err := svc.SetNamespacePermission(ctx, &api.UserInfo{Username: "owner", Superuser: true}, "test-ns", "owner", api.SetNamespacePermissionRequest{
		Level: api.PermissionLevelEditor,
	})
	if err != nil || specError != "" {
		t.Fatalf("superuser modify own permission = %v, %q, %v", perm, specError, err)
	}
	if perm.Level != api.PermissionLevelEditor {
		t.Fatalf("superuser level = %q, want editor", perm.Level)
	}
}

func TestService_DeleteNamespacePermissionSelfRules(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()

	specError, err := svc.DeleteNamespacePermission(ctx, userInfo("owner"), "test-ns", "owner")
	if !service.IsForbidden(err) || specError != "" {
		t.Fatalf("owner delete own permission = %q, %v, want forbidden", specError, err)
	}

	level, err := store.GetNamespacePermission(ctx, "test-ns", "owner")
	if err != nil {
		t.Fatalf("GetNamespacePermission(owner) error = %v", err)
	}
	if level != api.PermissionLevelManager {
		t.Fatalf("owner level after forbidden delete = %q, want manager", level)
	}

	specError, err = svc.DeleteNamespacePermission(ctx, &api.UserInfo{Username: "owner", Superuser: true}, "test-ns", "owner")
	if err != nil || specError != "" {
		t.Fatalf("superuser delete own permission = %q, %v", specError, err)
	}

	level, err = store.GetNamespacePermission(ctx, "test-ns", "owner")
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

	_, specError, err := svc.ListNamespaces(ctx, nil, api.ListNamespacesParams{})
	if !service.IsUnauthorized(err) || specError != "" {
		t.Fatalf("ListNamespaces(nil) = %q, %v, want unauthorized", specError, err)
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
	ns, specError, err := svc.CreateNamespace(ctx, nil, req)
	if err != nil || specError != "" {
		t.Fatalf("CreateNamespace(nil) = %v, %q, %v", ns, specError, err)
	}
	if ns.ID != "test-ns" {
		t.Fatalf("namespace id = %q, want test-ns", ns.ID)
	}

	perm, err := store.GetNamespacePermission(ctx, "test-ns", "someone")
	if err != nil {
		t.Fatalf("GetNamespacePermission() error = %v", err)
	}
	if perm != api.PermissionLevelNone {
		t.Fatalf("GetNamespacePermission() = %q, want none", perm)
	}

	page, specError, err := svc.ListNamespaces(ctx, nil, api.ListNamespacesParams{Limit: 10})
	if err != nil || specError != "" {
		t.Fatalf("ListNamespaces(nil) = %v, %q, %v", page, specError, err)
	}
	if page.Total != 1 {
		t.Fatalf("ListNamespaces() total = %d, want 1", page.Total)
	}

	created, specError, err := svc.CreateResource(ctx, nil, "test-ns", api.ResourceCreateRequest{
		URL: "https://example.com/anon",
	})
	if err != nil || specError != "" {
		t.Fatalf("CreateResource(nil) = %v, %q, %v", created, specError, err)
	}
	if created.PID != "aaa-bbb" {
		t.Fatalf("CreateResource() pid = %q, want aaa-bbb", created.PID)
	}

	got, specError, err := svc.GetResource(ctx, nil, "test-ns", "aaa-bbb")
	if err != nil || specError != "" {
		t.Fatalf("GetResource(nil) = %v, %q, %v", got, specError, err)
	}
	if got.URL != "https://example.com/anon" {
		t.Fatalf("GetResource() url = %q, want https://example.com/anon", got.URL)
	}

	count, specError, err := svc.CountAllResources(ctx)
	if err != nil || specError != "" {
		t.Fatalf("CountAllResources(nil) = %v, %q, %v", count, specError, err)
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

	if _, specError, err := svc.CreateUser(ctx, userInfo("owner"), api.UserCreateRequest{Username: "alice"}); !service.IsUnavailableInAnonymousMode(err) || specError != api.UnavailableInAnonymousMode {
		t.Fatalf("CreateUser() = %q, %v, want anonymous mode unavailable", specError, err)
	}
	if _, specError, err := svc.ListUsers(ctx, userInfo("owner"), api.ListUsersParams{Limit: 10}); !service.IsUnavailableInAnonymousMode(err) || specError != api.UnavailableInAnonymousMode {
		t.Fatalf("ListUsers() = %q, %v, want anonymous mode unavailable", specError, err)
	}
	if _, specError, err := svc.GetNamespacePermission(ctx, userInfo("owner"), "test-ns", "owner"); !service.IsUnavailableInAnonymousMode(err) || specError != api.UnavailableInAnonymousMode {
		t.Fatalf("GetNamespacePermission() = %q, %v, want anonymous mode unavailable", specError, err)
	}
}
