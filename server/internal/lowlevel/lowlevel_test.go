//spellchecker:words lowlevel
package lowlevel_test

//spellchecker:words context errors slog http httptest strings testing time github bicpid backend memory server internal lowlevel service
import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/memory"
	"github.com/tkw1536/bicpid/server/internal/lowlevel"
	"github.com/tkw1536/bicpid/service"
)

type authProbeResponse struct {
	Username *string       `json:"username"`
	User     *api.UserInfo `json:"user"`
}

func TestAuthHandlerAuthVariants(t *testing.T) {
	t.Parallel()

	const (
		validKnownToken   = "valid-known-token"
		validMissingToken = "valid-missing-token"
		invalidToken      = "invalid-token"
		lookupFailToken   = "lookup-fail-token"
		userFailToken     = "user-fail-token"
	)

	validUser := &api.UserInfo{Username: "alice", Superuser: true}
	lookupFailure := errors.New("lookup failure")
	userFailure := errors.New("user failure")

	tests := []struct {
		name string
		run  func(*testing.T, *lowlevel.AuthHandler, scenario, *api.UserInfo)
	}{
		{
			name: "HandleNoAuth",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.UserInfo) {
				t.Helper()

				var gotCalled bool
				handler := lowlevel.HandleNoAuth(
					h,
					func(w http.ResponseWriter, r *http.Request) (authProbeResponse, api.Error, error) {
						gotCalled = true
						return authProbeResponse{}, "", nil
					},
					http.StatusOK,
					[]api.Error{api.DatabaseError},
				)

				rec := runHandler(handler, scenario.token)
				assertStatusAndCalled(t, rec, gotCalled, http.StatusOK, true)
			},
		},
		{
			name: "HandleRequiredUsername",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.UserInfo) {
				t.Helper()

				var (
					gotCalled bool
					gotUser   string
				)
				handler := lowlevel.HandleRequiredUsername(
					h,
					func(w http.ResponseWriter, r *http.Request, username string) (authProbeResponse, api.Error, error) {
						gotCalled = true
						gotUser = username
						return authProbeResponse{Username: new(username)}, "", nil
					},
					http.StatusOK,
					[]api.Error{api.Unauthorized, api.DatabaseError},
				)

				rec := runHandler(handler, scenario.token)
				wantStatus, wantCalled, wantUser := expectedUsernameOutcome(true, scenario.token, validUser)
				assertStatusAndCalled(t, rec, gotCalled, wantStatus, wantCalled)
				if wantCalled && gotUser != wantUser {
					t.Fatalf("username = %q, want %q", gotUser, wantUser)
				}
			},
		},
		{
			name: "HandleOptionalUsername",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.UserInfo) {
				t.Helper()

				var (
					gotCalled bool
					gotUser   *string
				)
				handler := lowlevel.HandleOptionalUsername(
					h,
					func(w http.ResponseWriter, r *http.Request, username *string) (authProbeResponse, api.Error, error) {
						gotCalled = true
						gotUser = username
						return authProbeResponse{Username: username}, "", nil
					},
					http.StatusOK,
					[]api.Error{api.Unauthorized, api.DatabaseError},
				)

				rec := runHandler(handler, scenario.token)
				wantStatus, wantCalled, wantUser := expectedOptionalUsernameOutcome(scenario.token, validUser)
				assertStatusAndCalled(t, rec, gotCalled, wantStatus, wantCalled)
				if wantCalled {
					assertOptionalStringEqual(t, gotUser, wantUser)
				}
			},
		},
		{
			name: "HandleRequiredUser",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.UserInfo) {
				t.Helper()

				var (
					gotCalled bool
					gotUser   *api.UserInfo
				)
				handler := lowlevel.HandleRequiredUser(
					h,
					func(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (authProbeResponse, api.Error, error) {
						gotCalled = true
						gotUser = user
						return authProbeResponse{Username: &user.Username, User: user}, "", nil
					},
					http.StatusOK,
					[]api.Error{api.Unauthorized, api.DatabaseError},
				)

				rec := runHandler(handler, scenario.token)
				wantStatus, wantCalled, wantUser := expectedUserOutcome(true, scenario.token, validUser)
				assertStatusAndCalled(t, rec, gotCalled, wantStatus, wantCalled)
				if wantCalled {
					assertOptionalUserEqual(t, gotUser, wantUser)
				}
			},
		},
		{
			name: "HandleOptionalUser",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.UserInfo) {
				t.Helper()

				var (
					gotCalled bool
					gotUser   *api.UserInfo
				)
				handler := lowlevel.HandleOptionalUser(
					h,
					func(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (authProbeResponse, api.Error, error) {
						gotCalled = true
						gotUser = user
						var username *string
						if user != nil {
							username = &user.Username
						}
						return authProbeResponse{Username: username, User: user}, "", nil
					},
					http.StatusOK,
					[]api.Error{api.Unauthorized, api.DatabaseError},
				)

				rec := runHandler(handler, scenario.token)
				wantStatus, wantCalled, wantUser := expectedUserOutcome(false, scenario.token, validUser)
				assertStatusAndCalled(t, rec, gotCalled, wantStatus, wantCalled)
				if wantCalled {
					assertOptionalUserEqual(t, gotUser, wantUser)
				}
			},
		},
	}

	scenarios := []scenario{
		{name: "no user passed"},
		{name: "user given", token: validKnownToken},
		{name: "user given but backend can't find it", token: validMissingToken},
		{name: "invalid user passed", token: invalidToken},
		{name: "lookup backend failure", token: lookupFailToken},
		{name: "user backend failure", token: userFailToken},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			for _, scenario := range scenarios {
				t.Run(scenario.name, func(t *testing.T) {
					t.Parallel()

					h := lowlevel.NewAuthHandler(&mockAuthService{
						authenticate: func(_ context.Context, apiKey string) (string, error) {
							switch apiKey {
							case validKnownToken, validMissingToken, userFailToken:
								return validUser.Username, nil
							case invalidToken:
								return "", unauthorizedError{}
							case lookupFailToken:
								return "", lookupFailure
							default:
								return "", unauthorizedError{}
							}
						},
						currentUser: func(_ context.Context, apiKey string) (*api.UserInfo, error) {
							switch apiKey {
							case validKnownToken:
								return validUser, nil
							case validMissingToken:
								return nil, unauthorizedError{}
							case invalidToken:
								return nil, unauthorizedError{}
							case lookupFailToken:
								return nil, lookupFailure
							case userFailToken:
								return nil, userFailure
							default:
								return nil, unauthorizedError{}
							}
						},
					}, nil)

					test.run(t, h, scenario, validUser)
				})
			}
		})
	}
}

func TestHandleRequiredUsernamePanicsWhenUnauthorizedNotAllowed(t *testing.T) {
	t.Parallel()

	h := lowlevel.NewAuthHandler(&mockAuthService{}, nil)
	handler := lowlevel.HandleRequiredUsername(
		h,
		func(w http.ResponseWriter, r *http.Request, username string) (struct{}, api.Error, error) {
			t.Fatal("impl should not be called")
			return struct{}{}, "", nil
		},
		http.StatusOK,
		[]api.Error{api.DatabaseError},
	)

	defer expectPanic(t)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/probe", nil))
}

func TestHandleRequiredUserPanicsWhenForbiddenNotAllowed(t *testing.T) {
	t.Parallel()

	store := memory.NewStore()
	svc := service.New(
		store,
		store,
		store,
		service.NewRuntime(),
		service.Options{},
	)
	key := createUserWithKey(t, store, "alice", false)

	h := lowlevel.NewAuthHandler(svc, nil)
	handler := lowlevel.HandleRequiredUser(
		h,
		func(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (struct{}, api.Error, error) {
			_, specError, err := svc.CreateUser(r.Context(), user, api.UserCreateRequest{Username: "bob", Superuser: true})
			return struct{}{}, specError, err
		},
		http.StatusOK,
		[]api.Error{api.Unauthorized, api.DatabaseError},
	)

	req := httptest.NewRequest(http.MethodPost, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+key)

	defer expectPanic(t)
	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestAuthHandlerLog(t *testing.T) {
	t.Parallel()

	handler := &captureHandler{}
	logger := slog.New(handler)
	authHandler := lowlevel.NewAuthHandler(&mockAuthService{}, logger)
	req := httptest.NewRequest(http.MethodGet, "/probe?x=1", nil)
	req.RemoteAddr = "127.0.0.1:9999"

	authHandler.Log(context.Background(), req, 10*time.Millisecond, http.StatusForbidden, slog.String("error", "forbidden"))

	if len(handler.records) != 1 {
		t.Fatalf("record count = %d, want 1", len(handler.records))
	}
	record := handler.records[0]
	if record.Message != "request client error" {
		t.Fatalf("message = %q, want %q", record.Message, "request client error")
	}
	if record.Level != slog.LevelWarn {
		t.Fatalf("level = %v, want %v", record.Level, slog.LevelWarn)
	}
	if record.Attrs["method"] != http.MethodGet {
		t.Fatalf("method = %v, want %q", record.Attrs["method"], http.MethodGet)
	}
	if record.Attrs["uri"] != "/probe?x=1" {
		t.Fatalf("uri = %v, want %q", record.Attrs["uri"], "/probe?x=1")
	}
	if record.Attrs["status"] != int64(http.StatusForbidden) {
		t.Fatalf("status = %v, want %d", record.Attrs["status"], http.StatusForbidden)
	}
	if record.Attrs["error"] != "forbidden" {
		t.Fatalf("error = %v, want %q", record.Attrs["error"], "forbidden")
	}
}

type scenario struct {
	name  string
	token string
}

type mockAuthService struct {
	authenticate func(context.Context, string) (string, error)
	currentUser  func(context.Context, string) (*api.UserInfo, error)
}

func (m *mockAuthService) Authenticate(ctx context.Context, apiKey string) (string, error) {
	if m.authenticate != nil {
		return m.authenticate(ctx, apiKey)
	}
	return "", unauthorizedError{}
}

func (m *mockAuthService) CurrentUser(ctx context.Context, apiKey string) (*api.UserInfo, error) {
	if m.currentUser != nil {
		return m.currentUser(ctx, apiKey)
	}
	return nil, unauthorizedError{}
}

type unauthorizedError struct{}

func (unauthorizedError) Error() string {
	return "unauthorized"
}

func (unauthorizedError) Is(target error) bool {
	return service.IsUnauthorized(target)
}

type capturedRecord struct {
	Level   slog.Level
	Message string
	Attrs   map[string]any
}

type captureHandler struct {
	records []capturedRecord
}

func (*captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := make(map[string]any)
	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})
	h.records = append(h.records, capturedRecord{
		Level:   record.Level,
		Message: record.Message,
		Attrs:   attrs,
	})
	return nil
}

func (*captureHandler) WithAttrs([]slog.Attr) slog.Handler {
	return &captureHandler{}
}

func (*captureHandler) WithGroup(string) slog.Handler {
	return &captureHandler{}
}

func runHandler(handler http.Handler, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	handler.ServeHTTP(rec, req)
	return rec
}

func expectedUsernameOutcome(required bool, token string, validUser *api.UserInfo) (status int, called bool, username string) {
	if token == "" {
		if required {
			return http.StatusUnauthorized, false, ""
		}
		return http.StatusOK, true, ""
	}

	switch token {
	case "valid-known-token", "valid-missing-token", "user-fail-token":
		return http.StatusOK, true, validUser.Username
	case "invalid-token":
		return http.StatusUnauthorized, false, ""
	case "lookup-fail-token":
		return http.StatusInternalServerError, false, ""
	default:
		panic("unexpected token")
	}
}

func expectedOptionalUsernameOutcome(token string, validUser *api.UserInfo) (status int, called bool, username *string) {
	status, called, value := expectedUsernameOutcome(false, token, validUser)
	if !called || value == "" {
		return status, called, nil
	}
	return status, called, new(value)
}

func expectedUserOutcome(required bool, token string, validUser *api.UserInfo) (status int, called bool, user *api.UserInfo) {
	if token == "" {
		if required {
			return http.StatusUnauthorized, false, nil
		}
		return http.StatusOK, true, nil
	}

	switch token {
	case "valid-known-token":
		return http.StatusOK, true, validUser
	case "valid-missing-token", "invalid-token":
		return http.StatusUnauthorized, false, nil
	case "lookup-fail-token", "user-fail-token":
		return http.StatusInternalServerError, false, nil
	default:
		panic("unexpected token")
	}
}

func assertStatusAndCalled(t *testing.T, rec *httptest.ResponseRecorder, gotCalled bool, wantStatus int, wantCalled bool) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, wantStatus, rec.Body.String())
	}
	if gotCalled != wantCalled {
		t.Fatalf("impl called = %v, want %v", gotCalled, wantCalled)
	}
}

func createUserWithKey(t *testing.T, auth backend.AuthBackend, username string, superuser bool) string {
	t.Helper()

	ctx := context.Background()
	now := func() time.Time { return time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC) }
	if _, err := auth.CreateUser(ctx, api.UserCreateRequest{Username: username, Superuser: superuser}, now); err != nil {
		t.Fatalf("CreateUser(%q) error = %v", username, err)
	}

	rawKey := strings.Repeat("a", 32-len(username)) + username
	if _, err := auth.CreateKey(ctx, username, "key-1", rawKey, api.IssueKeyRequest{Comment: "test"}, now); err != nil {
		t.Fatalf("CreateKey(%q) error = %v", username, err)
	}
	return rawKey
}

func expectPanic(t *testing.T) {
	t.Helper()
	if recover() == nil {
		t.Fatal("expected panic")
	}
}

func assertOptionalStringEqual(t *testing.T, got, want *string) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("string pointer mismatch: got %v, want %v", got, want)
	case *got != *want:
		t.Fatalf("string value = %q, want %q", *got, *want)
	}
}

func assertOptionalUserEqual(t *testing.T, got, want *api.UserInfo) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("user pointer mismatch: got %v, want %v", got, want)
	case got.Username != want.Username || got.Superuser != want.Superuser:
		t.Fatalf("user = %+v, want %+v", *got, *want)
	}
}
