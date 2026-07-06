package cmd

//spellchecker:words context log slog testing time github quickpid backend authentication
import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend/authentication"
)

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func fixedNow() func() time.Time {
	t := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func TestEnsureRootUser_EmptyBackend(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	auth := authentication.NewInMemoryBackend()

	if err := ensureRootUser(ctx, auth, testLogger()); err != nil {
		t.Fatalf("ensureRootUser() error = %v", err)
	}

	user, err := auth.GetUser(ctx, "root")
	if err != nil {
		t.Fatalf("GetUser(root) error = %v", err)
	}
	if !user.Superuser {
		t.Fatalf("GetUser(root) superuser = false, want true")
	}

	keys, err := auth.ListKeys(ctx, "root", api.ListKeysParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListKeys(root) error = %v", err)
	}
	if len(keys.Items) != 1 || keys.Items[0].ID != "bootstrap" {
		t.Fatalf("ListKeys(root) = %+v", keys)
	}
}

func TestEnsureRootUser_ExistingUser(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	auth := authentication.NewInMemoryBackend()

	if _, err := auth.CreateUser(ctx, api.UserCreateRequest{Username: "alice"}, fixedNow()); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if err := ensureRootUser(ctx, auth, testLogger()); err != nil {
		t.Fatalf("ensureRootUser() error = %v", err)
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
	t.Helper()
	ctx := context.Background()
	auth := authentication.NewInMemoryBackend()

	if _, err := auth.CreateUser(ctx, api.UserCreateRequest{Username: "root", Superuser: true}, fixedNow()); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if err := ensureRootUser(ctx, auth, testLogger()); err != nil {
		t.Fatalf("ensureRootUser() error = %v", err)
	}
}

func TestEnsureRootUser_Idempotent(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	auth := authentication.NewInMemoryBackend()
	logger := testLogger()

	if err := ensureRootUser(ctx, auth, logger); err != nil {
		t.Fatalf("first ensureRootUser() error = %v", err)
	}
	if err := ensureRootUser(ctx, auth, logger); err != nil {
		t.Fatalf("second ensureRootUser() error = %v", err)
	}

	page, err := auth.ListUsers(ctx, api.ListUsersParams{Limit: 100})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("ListUsers() total = %d, want 1", page.Total)
	}
}
