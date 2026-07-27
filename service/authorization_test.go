//spellchecker:words service
package service_test

//spellchecker:words context errors testing time github quickpid backend memory internal apikey service
import (
	"context"
	"errors"
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
	if err := store.SetNamespaceRole(ctx, testNS, editorUsername, api.RoleEditor); err != nil {
		t.Fatalf("SetNamespaceRole(editor) error = %v", err)
	}
	if err := store.SetNamespaceRole(ctx, testNS, contributorUsername, api.RoleContributor); err != nil {
		t.Fatalf("SetNamespaceRole(contributor) error = %v", err)
	}
	if err := store.SetNamespaceRole(ctx, testNS, readerUsername, api.RoleNone); err != nil {
		t.Fatalf("SetNamespaceRole(reader) error = %v", err)
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

func TestService_GetNamespaceRole(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	perm, err := svc.GetNamespaceRole(ctx, userInfo("reader"), testNS, readerUsername)
	if err != nil {
		t.Fatalf("GetNamespaceRole(self) = %v, %v", perm, err)
	}
	if perm.Role != api.RoleNone {
		t.Fatalf("role = %q, want none", perm.Role)
	}

	_, err = svc.GetNamespaceRole(ctx, userInfo("reader"), testNS, editorUsername)
	if !service.IsForbidden(err) {
		t.Fatalf("GetNamespaceRole(other) = %v, want forbidden", err)
	}

	perm, err = svc.GetNamespaceRole(ctx, userInfo("owner"), testNS, editorUsername)
	if err != nil {
		t.Fatalf("GetNamespaceRole(manager) = %v, %v", perm, err)
	}
	if perm.Role != api.RoleEditor {
		t.Fatalf("role = %q, want editor", perm.Role)
	}
}

func TestService_ListUserRoles(t *testing.T) {
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
	if err := store.SetNamespaceRole(ctx, ns2, editorUsername, api.RoleContributor); err != nil {
		t.Fatalf("SetNamespaceRole(editor, ns2) error = %v", err)
	}

	page, err := svc.ListUserRoles(ctx, userInfo("editor"), nil, api.ListUserRolesParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUserRoles(self) = %v, %v", page, err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("ListUserRoles(self) = %+v, want 2 items", page)
	}
	if page.Items[0].Namespace != testNS.String() || page.Items[0].Role != api.RoleEditor {
		t.Fatalf("ListUserRoles(self)[0] = %+v", page.Items[0])
	}
	if page.Items[1].Namespace != ns2.String() || page.Items[1].Role != api.RoleContributor {
		t.Fatalf("ListUserRoles(self)[1] = %+v", page.Items[1])
	}

	other := editorUsername
	_, err = svc.ListUserRoles(ctx, userInfo("owner"), &other, api.ListUserRolesParams{Limit: 100})
	if !service.IsForbidden(err) {
		t.Fatalf("ListUserRoles(other as non-superuser) = %v, want forbidden", err)
	}

	page, err = svc.ListUserRoles(ctx, &api.ValidUserInfo{Username: ownerUsername, Superuser: true}, &other, api.ListUserRolesParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUserRoles(superuser) = %v, %v", page, err)
	}
	if page.Total != 2 {
		t.Fatalf("ListUserRoles(superuser) total = %d, want 2", page.Total)
	}

	missing, err := api.NewUsername("missing")
	if err != nil {
		t.Fatalf("NewUsername(missing) error = %v", err)
	}
	_, err = svc.ListUserRoles(ctx, &api.ValidUserInfo{Username: ownerUsername, Superuser: true}, &missing, api.ListUserRolesParams{Limit: 100})
	if code, ok := api.GetErrorString(err); !ok || code != api.UserNotFound {
		t.Fatalf("ListUserRoles(missing) = %v, want user_not_found", err)
	}

	_, err = svc.ListUserRoles(ctx, nil, nil, api.ListUserRolesParams{Limit: 100})
	if !service.IsUnauthorized(err) {
		t.Fatalf("ListUserRoles(nil caller) = %v, want unauthorized", err)
	}
}

func TestService_DeleteUser(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()

	if err := svc.DeleteUser(ctx, userInfo("owner"), contributorUsername); !service.IsForbidden(err) {
		t.Fatalf("DeleteUser(non-superuser) = %v, want forbidden", err)
	}

	superuserCaller := &api.ValidUserInfo{Username: ownerUsername, Superuser: true}
	if err := svc.DeleteUser(ctx, superuserCaller, ownerUsername); !service.IsForbidden(err) {
		t.Fatalf("DeleteUser(self as superuser) = %v, want forbidden", err)
	}

	if err := svc.DeleteUser(ctx, superuserCaller, contributorUsername); err != nil {
		t.Fatalf("DeleteUser(other as superuser) = %v", err)
	}

	if _, err := store.GetUser(ctx, contributorUsername); !errors.Is(err, backend.ErrUserNotFound) {
		t.Fatalf("GetUser(deleted contributor) = %v, want backend.ErrUserNotFound", err)
	}
}

func TestService_ResolverRoles(t *testing.T) {
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

	_, err = svc.SetNamespaceRole(ctx, userInfo("editor"), testNS, readerUsername, api.SetNamespaceRoleRequest{
		Role: api.RoleContributor,
	})
	if !service.IsForbidden(err) {
		t.Fatalf("editor SetNamespaceRole = %v, want forbidden", err)
	}
}

func TestService_GetResourceDeletedRedacted(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()
	now := fixedNow()

	existingPID, err := api.NewPID("existing")
	if err != nil {
		t.Fatalf("NewPID(existing) error = %v", err)
	}
	created, err := store.CreateResource(ctx, testNS, existingPID, api.ResourceCreateRequest{
		URL: "https://example.com",
	}, now)
	if err != nil {
		t.Fatalf("CreateResource() error = %v", err)
	}
	deleted := true
	if _, err := store.UpdateResource(ctx, testNS, existingPID, api.ResourceUpdateRequest{Deleted: &deleted}, now); err != nil {
		t.Fatalf("UpdateResource() error = %v", err)
	}

	assertRedacted := func(t *testing.T, caller *api.ValidUserInfo) {
		t.Helper()
		got, err := svc.GetResource(ctx, caller, testNS, existingPID)
		if err != nil {
			t.Fatalf("GetResource() = %v, %v", got, err)
		}
		redacted, ok := got.(api.RedactedResourceResponse)
		if !ok {
			t.Fatalf("GetResource() type = %T, want api.RedactedResourceResponse", got)
		}
		if redacted.PID != created.PID || redacted.DateCreated != created.DateCreated || redacted.DateUpdated != created.DateUpdated {
			t.Fatalf("redacted = %+v, want pid/dates from %+v", redacted, *created)
		}
		if redacted.StatusCode() != 410 {
			t.Fatalf("StatusCode() = %d, want 410", redacted.StatusCode())
		}
	}

	assertRedacted(t, nil)
	assertRedacted(t, userInfo("reader"))
	assertRedacted(t, userInfo("contributor"))

	full, err := svc.GetResource(ctx, userInfo("editor"), testNS, existingPID)
	if err != nil {
		t.Fatalf("editor GetResource = %v, %v", full, err)
	}
	if _, ok := full.(*api.ResourceResponse); !ok {
		t.Fatalf("editor GetResource type = %T, want *api.ResourceResponse", full)
	}
}

func TestService_SetNamespaceRoleCannotEscalate(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := t.Context()

	_, err := svc.SetNamespaceRole(ctx, userInfo("editor"), testNS, readerUsername, api.SetNamespaceRoleRequest{
		Role: api.RoleManager,
	})
	if !service.IsForbidden(err) {
		t.Fatalf("editor grant manager = %v, want forbidden", err)
	}
}

func TestService_SetNamespaceRoleSelfRules(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()

	_, err := svc.SetNamespaceRole(ctx, userInfo("owner"), testNS, ownerUsername, api.SetNamespaceRoleRequest{
		Role: api.RoleEditor,
	})
	if !service.IsForbidden(err) {
		t.Fatalf("owner modify own role = %v, want forbidden", err)
	}

	role, err := store.GetNamespaceRole(ctx, testNS, ownerUsername)
	if err != nil {
		t.Fatalf("GetNamespaceRole(owner) error = %v", err)
	}
	if role != api.RoleManager {
		t.Fatalf("owner role after forbidden update = %q, want manager", role)
	}

	perm, err := svc.SetNamespaceRole(ctx, &api.ValidUserInfo{Username: ownerUsername, Superuser: true}, testNS, ownerUsername, api.SetNamespaceRoleRequest{
		Role: api.RoleEditor,
	})
	if err != nil {
		t.Fatalf("superuser modify own role = %v, %v", perm, err)
	}
	if perm.Role != api.RoleEditor {
		t.Fatalf("superuser role = %q, want editor", perm.Role)
	}
}

func TestService_DeleteNamespaceRoleSelfRules(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := t.Context()

	err := svc.DeleteNamespaceRole(ctx, userInfo("owner"), testNS, ownerUsername)
	if !service.IsForbidden(err) {
		t.Fatalf("owner delete own role = %v, want forbidden", err)
	}

	role, err := store.GetNamespaceRole(ctx, testNS, ownerUsername)
	if err != nil {
		t.Fatalf("GetNamespaceRole(owner) error = %v", err)
	}
	if role != api.RoleManager {
		t.Fatalf("owner role after forbidden delete = %q, want manager", role)
	}

	err = svc.DeleteNamespaceRole(ctx, &api.ValidUserInfo{Username: ownerUsername, Superuser: true}, testNS, ownerUsername)
	if err != nil {
		t.Fatalf("superuser delete own role = %v", err)
	}

	role, err = store.GetNamespaceRole(ctx, testNS, ownerUsername)
	if err != nil {
		t.Fatalf("GetNamespaceRole(owner) error = %v", err)
	}
	if role != api.RoleNone {
		t.Fatalf("owner role after delete = %q, want none", role)
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

	perm, err := store.GetNamespaceRole(ctx, testNS, someoneUsername)
	if err != nil {
		t.Fatalf("GetNamespaceRole() error = %v", err)
	}
	if perm != api.RoleNone {
		t.Fatalf("GetNamespaceRole() = %q, want none", perm)
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
	full, ok := got.(*api.ResourceResponse)
	if !ok {
		t.Fatalf("GetResource() type = %T, want *api.ResourceResponse", got)
	}
	if full.URL != "https://example.com/anon" {
		t.Fatalf("GetResource() url = %q, want https://example.com/anon", full.URL)
	}

	count, err := svc.CountAllResources(ctx)
	if err != nil {
		t.Fatalf("CountAllResources(nil) = %v, %v", count, err)
	}
	if count.Total != 1 {
		t.Fatalf("CountAllResources() total = %d, want 1", count.Total)
	}
}

func TestService_AnonymousDisablesUserAndRoleManagement(t *testing.T) {
	t.Parallel()

	svc, _ := newAnonymousTestService(t)
	ctx := t.Context()

	if _, err := svc.AuthenticateAPIKey(ctx, "anything"); !service.IsUnauthorized(err) {
		t.Fatalf("AuthenticateAPIKey() error = %v, want unauthorized", err)
	}
	rootUsername, err := api.NewUsername("root")
	if err != nil {
		t.Fatalf("NewUsername(root) error = %v", err)
	}
	if _, err := svc.LoadUser(ctx, rootUsername); !service.IsUnauthorized(err) {
		t.Fatalf("LoadUser() error = %v, want unauthorized", err)
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
	if err := svc.DeleteUser(ctx, userInfo("owner"), aliceUsername); !service.IsUnavailableInAnonymousMode(err) {
		t.Fatalf("DeleteUser() = %v, want anonymous mode unavailable", err)
	}
	if _, err := svc.GetNamespaceRole(ctx, userInfo("owner"), testNS, ownerUsername); !service.IsUnavailableInAnonymousMode(err) {
		t.Fatalf("GetNamespaceRole() = %v, want anonymous mode unavailable", err)
	}
	if _, err := svc.ListUserRoles(ctx, userInfo("owner"), nil, api.ListUserRolesParams{Limit: 10}); !service.IsUnavailableInAnonymousMode(err) {
		t.Fatalf("ListUserRoles() = %v, want anonymous mode unavailable", err)
	}
}
