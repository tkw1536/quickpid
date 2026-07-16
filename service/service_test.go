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

func TestEnsureRootUser_EmptyBackend(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	auth := memory.NewStore()
	svc := testService(auth)

	if err := svc.EnsureRootUser(ctx, testLogger()); err != nil {
		t.Fatalf("EnsureRootUser() error = %v", err)
	}

	user, err := auth.GetUser(ctx, "root")
	if err != nil {
		t.Fatalf("GetUser(root) error = %v", err)
	}
	if !user.Superuser {
		t.Fatalf("GetUser(root) superuser = false, want true")
	}

	keys, err := auth.ListKeys(ctx, apikey.Default, "root", api.ListKeysParams{Limit: 100})
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

	if _, err := auth.CreateUser(ctx, api.UserCreateRequest{Username: "alice"}, fixedNow()); err != nil {
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

	if _, err := auth.CreateUser(ctx, api.UserCreateRequest{Username: "root", Superuser: true}, fixedNow()); err != nil {
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

func TestValidateNamespaceID(t *testing.T) {
	t.Parallel()
	testIdentifierValidation(t, service.ValidateNamespaceID, "invalid namespace id")
}

func TestValidatePID(t *testing.T) {
	t.Parallel()
	testIdentifierValidation(t, service.ValidatePID, "invalid pid")
}

func TestValidateUsername(t *testing.T) {
	t.Parallel()
	testIdentifierValidation(t, service.ValidateUsername, "invalid username")
}

func testIdentifierValidation(t *testing.T, validate func(string) error, wantInvalidMsg string) {
	t.Helper()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:  "valid_alphanumeric",
			input: "abc123",
		},
		{
			name:  "valid_hyphen_underscore",
			input: "a-b_c",
		},
		{
			name:    "invalid_empty",
			input:   "",
			wantErr: wantInvalidMsg,
		},
		{
			name:    "invalid_uppercase",
			input:   "Alice",
			wantErr: wantInvalidMsg,
		},
		{
			name:    "invalid_special_characters",
			input:   "a.b",
			wantErr: wantInvalidMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validate(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate(%q) error = %v, want nil", tt.input, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validate(%q) error = nil, want %q", tt.input, tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("validate(%q) error = %q, want %q", tt.input, err.Error(), tt.wantErr)
			}
		})
	}
}
