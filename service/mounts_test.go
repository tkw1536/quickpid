//spellchecker:words service
package service_test

//spellchecker:words context testing github quickpid api service
import (
	"context"
	"testing"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/service"
)

func validMountUpsert(t *testing.T, namespace api.ValidNamespaceID) api.ValidMountUpsertRequest {
	t.Helper()
	req := api.MountUpsertRequest{Namespace: namespace.String()}
	valid, err := req.Validate()
	if err != nil {
		t.Fatalf("MountUpsertRequest.Validate() error = %v", err)
	}
	return valid
}

func TestService_Mounts_Anonymous(t *testing.T) {
	t.Parallel()

	svc, store := newAnonymousTestService(t)
	ctx := context.Background()
	now := fixedNow()

	ns, err := api.NewNamespaceID("anon-ns")
	if err != nil {
		t.Fatalf("NewNamespaceID() error = %v", err)
	}
	if _, err := store.CreateNamespace(ctx, ns, api.NamespaceCreateRequest{
		Tag:       "mount",
		PIDFormat: pid.Format{Pattern: "***-***", Characters: pid.Full},
	}, nil, now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	baseURI, err := api.NewBaseURI("https://example.com/anon/")
	if err != nil {
		t.Fatalf("NewBaseURI() error = %v", err)
	}

	mount, err := svc.SetMount(ctx, nil, baseURI, validMountUpsert(t, ns))
	if err != nil {
		t.Fatalf("SetMount() error = %v", err)
	}
	if mount.Namespace != ns.String() || mount.BaseURI != baseURI.String() {
		t.Fatalf("SetMount() = %+v", mount)
	}

	got, err := svc.GetMount(ctx, baseURI)
	if err != nil {
		t.Fatalf("GetMount() error = %v", err)
	}
	if got.Namespace != ns.String() {
		t.Fatalf("GetMount() = %+v, want namespace %q", got, ns.String())
	}

	if err := svc.DeleteMount(ctx, nil, baseURI); err != nil {
		t.Fatalf("DeleteMount() error = %v", err)
	}
	if _, err := svc.GetMount(ctx, baseURI); err == nil {
		t.Fatal("GetMount() after delete error = nil, want error")
	} else if code, ok := api.GetErrorString(err); !ok || code != api.MountNotFound {
		t.Fatalf("GetMount() after delete error = %v, want mount_not_found", err)
	}
}

func TestService_Mounts_Authenticated(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	baseURI, err := api.NewBaseURI("https://example.com/auth/")
	if err != nil {
		t.Fatalf("NewBaseURI() error = %v", err)
	}

	upsert := validMountUpsert(t, testNS)

	_, err = svc.SetMount(ctx, nil, baseURI, upsert)
	if !service.IsUnauthorized(err) {
		t.Fatalf("SetMount(nil) error = %v, want unauthorized", err)
	}

	_, err = svc.SetMount(ctx, userInfo("reader"), baseURI, upsert)
	if !service.IsForbidden(err) {
		t.Fatalf("SetMount(non-superuser) error = %v, want forbidden", err)
	}

	super := &api.ValidUserInfo{Username: ownerUsername, Superuser: true}
	mount, err := svc.SetMount(ctx, super, baseURI, upsert)
	if err != nil {
		t.Fatalf("SetMount(superuser) error = %v", err)
	}
	if mount.Namespace != testNS.String() {
		t.Fatalf("SetMount(superuser) = %+v", mount)
	}

	got, err := svc.GetMount(ctx, baseURI)
	if err != nil {
		t.Fatalf("GetMount() error = %v", err)
	}
	if got.Namespace != testNS.String() {
		t.Fatalf("GetMount() = %+v", got)
	}

	err = svc.DeleteMount(ctx, userInfo("reader"), baseURI)
	if !service.IsForbidden(err) {
		t.Fatalf("DeleteMount(non-superuser) error = %v, want forbidden", err)
	}

	if err := svc.DeleteMount(ctx, super, baseURI); err != nil {
		t.Fatalf("DeleteMount(superuser) error = %v", err)
	}
}

func TestService_ListMounts(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()
	baseURI, err := api.NewBaseURI("https://example.com/list/")
	if err != nil {
		t.Fatalf("NewBaseURI() error = %v", err)
	}
	super := &api.ValidUserInfo{Username: ownerUsername, Superuser: true}
	if _, err := svc.SetMount(ctx, super, baseURI, validMountUpsert(t, testNS)); err != nil {
		t.Fatalf("SetMount() error = %v", err)
	}

	_, err = svc.ListMounts(ctx, nil, api.ListMountsParams{Limit: 10})
	if !service.IsUnauthorized(err) {
		t.Fatalf("ListMounts(nil) error = %v, want unauthorized", err)
	}
	_, err = svc.ListMounts(ctx, userInfo("reader"), api.ListMountsParams{Limit: 10})
	if !service.IsForbidden(err) {
		t.Fatalf("ListMounts(non-superuser) error = %v, want forbidden", err)
	}

	page, err := svc.ListMounts(ctx, super, api.ListMountsParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListMounts(superuser) error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].BaseURI != baseURI.String() {
		t.Fatalf("ListMounts() = %+v", page)
	}

	anon, store := newAnonymousTestService(t)
	now := fixedNow()
	if _, err := store.CreateNamespace(ctx, testNS, api.NamespaceCreateRequest{
		Tag:       "t",
		PIDFormat: pid.Format{Pattern: "***-***", Characters: pid.Full},
	}, nil, now); err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}
	if _, err := anon.SetMount(ctx, nil, baseURI, validMountUpsert(t, testNS)); err != nil {
		t.Fatalf("anonymous SetMount() error = %v", err)
	}
	page, err = anon.ListMounts(ctx, nil, api.ListMountsParams{Limit: 10})
	if err != nil {
		t.Fatalf("anonymous ListMounts() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("anonymous ListMounts() total = %d, want 1", page.Total)
	}
}

func TestService_ListNamespaceMounts(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()
	baseURI, err := api.NewBaseURI("https://example.com/ns-list/")
	if err != nil {
		t.Fatalf("NewBaseURI() error = %v", err)
	}
	super := &api.ValidUserInfo{Username: ownerUsername, Superuser: true}
	if _, err := svc.SetMount(ctx, super, baseURI, validMountUpsert(t, testNS)); err != nil {
		t.Fatalf("SetMount() error = %v", err)
	}

	_, err = svc.ListNamespaceMounts(ctx, nil, testNS, api.ListNamespaceMountsParams{Limit: 10})
	if !service.IsUnauthorized(err) {
		t.Fatalf("ListNamespaceMounts(nil) error = %v, want unauthorized", err)
	}
	_, err = svc.ListNamespaceMounts(ctx, userInfo("reader"), testNS, api.ListNamespaceMountsParams{Limit: 10})
	if !service.IsForbidden(err) {
		t.Fatalf("ListNamespaceMounts(reader) error = %v, want forbidden", err)
	}

	page, err := svc.ListNamespaceMounts(ctx, userInfo("contributor"), testNS, api.ListNamespaceMountsParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListNamespaceMounts(contributor) error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0] != baseURI.String() {
		t.Fatalf("ListNamespaceMounts() = %+v", page)
	}
}
