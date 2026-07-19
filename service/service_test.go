//spellchecker:words service
package service_test

//spellchecker:words context slog testing time github quickpid backend memory internal apikey service
import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/backend/memory"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/service"
)

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func fixedNow() func() time.Time {
	t := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func testService(store backend.Store) *service.Service {
	return service.New(store, service.NewRuntime(), service.Options{})
}

var rootUsername *api.ValidUsername

func init() {
	user, err := api.NewUsername("root")
	if err != nil {
		panic(err)
	}
	rootUsername = user
}

func TestEnsureRootUser_EmptyBackend(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	auth := memory.NewStore()
	svc := testService(auth)

	if err := svc.EnsureRootUser(ctx, testLogger()); err != nil {
		t.Fatalf("EnsureRootUser() error = %v", err)
	}

	user, err := auth.GetUser(ctx, rootUsername)
	if err != nil {
		t.Fatalf("GetUser(root) error = %v", err)
	}
	if !user.Superuser {
		t.Fatalf("GetUser(root) superuser = false, want true")
	}

	keys, err := auth.ListKeys(ctx, apikey.Default, rootUsername, api.ListKeysParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListKeys(root) error = %v", err)
	}
	if len(keys.Items) != 1 {
		t.Fatalf("ListKeys(root) = %+v, want 1 key", keys)
	}
}

func TestEnsureRootUser_ExistingUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	auth := memory.NewStore()

	aliceUsername, err := api.NewUsername("alice")
	if err != nil {
		t.Fatalf("NewUsername(alice) error = %v", err)
	}
	if _, err := auth.CreateUser(ctx, &api.ValidUserCreateRequest{Username: aliceUsername}, fixedNow()); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	svc := testService(auth)
	if err := svc.EnsureRootUser(ctx, testLogger()); err != nil {
		t.Fatalf("EnsureRootUser() error = %v", err)
	}

	page, err := auth.ListUsers(ctx, api.ListUsersParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("ListUsers() total = %d, want 1", page.Total)
	}
	if page.Items[0].Username != "alice" {
		t.Fatalf("ListUsers() = %+v", page)
	}
}

func TestEnsureRootUser_ExistingRootRace(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	auth := memory.NewStore()

	if _, err := auth.CreateUser(ctx, &api.ValidUserCreateRequest{Username: rootUsername, Superuser: true}, fixedNow()); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	svc := testService(auth)
	if err := svc.EnsureRootUser(ctx, testLogger()); err != nil {
		t.Fatalf("EnsureRootUser() error = %v", err)
	}
}

func TestEnsureRootUser_Idempotent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	auth := memory.NewStore()
	logger := testLogger()
	svc := testService(auth)

	if err := svc.EnsureRootUser(ctx, logger); err != nil {
		t.Fatalf("first EnsureRootUser() error = %v", err)
	}
	if err := svc.EnsureRootUser(ctx, logger); err != nil {
		t.Fatalf("second EnsureRootUser() error = %v", err)
	}

	page, err := auth.ListUsers(ctx, api.ListUsersParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("ListUsers() total = %d, want 1", page.Total)
	}
}

func TestEnsureRootUser_AnonymousModeSkipsBootstrap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	auth := memory.NewStore()
	svc := service.New(auth, service.NewRuntime(), service.Options{Anonymous: true})

	if err := svc.EnsureRootUser(ctx, testLogger()); err != nil {
		t.Fatalf("EnsureRootUser() error = %v", err)
	}

	page, err := auth.ListUsers(ctx, api.ListUsersParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("ListUsers() total = %d, want 0", page.Total)
	}
}
