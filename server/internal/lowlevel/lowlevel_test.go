//spellchecker:words lowlevel
package lowlevel_test

//spellchecker:words context encoding base errors slog http httptest strings testing time github quickpid backend memory internal apikey httpfixture server lowlevel service
import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/backend/memory"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/internal/httpfixture"
	"github.com/tkw1536/quickpid/server/internal/lowlevel"
	"github.com/tkw1536/quickpid/service"
)

type authProbeResponse struct {
	Username *string       `json:"username"`
	User     *api.UserInfo `json:"user"`
}

var (
	errLookupFailure = errors.New("lookup failure")
	errUserFailure   = errors.New("user failure")
)

func TestAuthHandlerAuthVariants(t *testing.T) {
	t.Parallel()

	const (
		validKnownToken    = "valid-known-token"
		validMissingToken  = "valid-missing-token"
		invalidToken       = "invalid-token"
		lookupFailToken    = "lookup-fail-token" /* #nosec G101 -- hard-coded test token never talks to a real API  */
		userFailToken      = "user-fail-token"
		validBasicPassword = "basic-secret"
		invalidBasicUser   = "mallory"
		invalidBasicPass   = "wrong-secret"
		userFailBasicUser  = "charlie"
		userFailBasicPass  = "charlie-secret"
	)

	validUser := mustValidUser(t, "alice", true)

	tests := []struct {
		name string
		run  func(*testing.T, *lowlevel.AuthHandler, scenario, *api.ValidUserInfo)
	}{
		{
			name: "HandleNoAuth",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.ValidUserInfo) {
				t.Helper()

				var gotCalled bool
				handler := lowlevel.HandleNoAuth(
					h,
					func(w http.ResponseWriter, r *http.Request) (authProbeResponse, error) {
						gotCalled = true
						return authProbeResponse{}, nil
					},
					lowlevel.FixedStatusCode[authProbeResponse](http.StatusOK),
					[]api.ErrorString{api.DatabaseError},
				)

				rec := runHandler(t, handler, scenario.authHeaders...)
				assertStatusAndCalled(t, rec, gotCalled, http.StatusOK, true)
			},
		},
		{
			name: "HandleRequiredUsername",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.ValidUserInfo) {
				t.Helper()

				var (
					gotCalled bool
					gotUser   string
				)
				handler := lowlevel.HandleRequiredUsername(
					h,
					func(w http.ResponseWriter, r *http.Request, username *api.ValidUsername) (authProbeResponse, error) {
						gotCalled = true
						gotUser = username.String()
						return authProbeResponse{Username: stringPtr(username.String())}, nil
					},
					lowlevel.FixedStatusCode[authProbeResponse](http.StatusOK),
					[]api.ErrorString{api.Unauthorized, api.DatabaseError},
				)

				rec := runHandler(t, handler, scenario.authHeaders...)
				wantStatus, wantCalled, wantUser := expectedUsernameOutcome(true, scenario, validUser)
				assertStatusAndCalled(t, rec, gotCalled, wantStatus, wantCalled)
				if wantCalled && gotUser != wantUser {
					t.Fatalf("username = %q, want %q", gotUser, wantUser)
				}
			},
		},
		{
			name: "HandleOptionalUsername",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.ValidUserInfo) {
				t.Helper()

				var (
					gotCalled bool
					gotUser   *string
				)
				handler := lowlevel.HandleOptionalUsername(
					h,
					func(w http.ResponseWriter, r *http.Request, username *api.ValidUsername) (authProbeResponse, error) {
						gotCalled = true
						gotUser = usernameStringPtr(username)
						return authProbeResponse{Username: usernameStringPtr(username)}, nil
					},
					lowlevel.FixedStatusCode[authProbeResponse](http.StatusOK),
					[]api.ErrorString{api.Unauthorized, api.DatabaseError},
				)

				rec := runHandler(t, handler, scenario.authHeaders...)
				wantStatus, wantCalled, wantUser := expectedOptionalUsernameOutcome(scenario, validUser)
				assertStatusAndCalled(t, rec, gotCalled, wantStatus, wantCalled)
				if wantCalled {
					assertOptionalStringEqual(t, gotUser, wantUser)
				}
			},
		},
		{
			name: "HandleRequiredUser",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.ValidUserInfo) {
				t.Helper()

				var (
					gotCalled bool
					gotUser   *api.ValidUserInfo
				)
				handler := lowlevel.HandleRequiredUser(
					h,
					func(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (authProbeResponse, error) {
						gotCalled = true
						gotUser = user
						return authProbeResponse{Username: stringPtr(user.Username.String()), User: userInfoFromValid(user)}, nil
					},
					lowlevel.FixedStatusCode[authProbeResponse](http.StatusOK),
					[]api.ErrorString{api.Unauthorized, api.DatabaseError},
				)

				rec := runHandler(t, handler, scenario.authHeaders...)
				wantStatus, wantCalled, wantUser := expectedUserOutcome(true, scenario, validUser)
				assertStatusAndCalled(t, rec, gotCalled, wantStatus, wantCalled)
				if wantCalled {
					assertOptionalUserEqual(t, gotUser, wantUser)
				}
			},
		},
		{
			name: "HandleOptionalUser",
			run: func(t *testing.T, h *lowlevel.AuthHandler, scenario scenario, validUser *api.ValidUserInfo) {
				t.Helper()

				var (
					gotCalled bool
					gotUser   *api.ValidUserInfo
				)
				handler := lowlevel.HandleOptionalUser(
					h,
					func(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (authProbeResponse, error) {
						gotCalled = true
						gotUser = user
						var username *string
						var userInfo *api.UserInfo
						if user != nil {
							username = stringPtr(user.Username.String())
							userInfo = userInfoFromValid(user)
						}
						return authProbeResponse{Username: username, User: userInfo}, nil
					},
					lowlevel.FixedStatusCode[authProbeResponse](http.StatusOK),
					[]api.ErrorString{api.Unauthorized, api.DatabaseError},
				)

				rec := runHandler(t, handler, scenario.authHeaders...)
				wantStatus, wantCalled, wantUser := expectedUserOutcome(false, scenario, validUser)
				assertStatusAndCalled(t, rec, gotCalled, wantStatus, wantCalled)
				if wantCalled {
					assertOptionalUserEqual(t, gotUser, wantUser)
				}
			},
		},
	}

	scenarios := []scenario{
		{id: "no-auth", name: "no user passed"},
		{id: "bearer-valid-known", name: "bearer user given", authHeaders: []string{"Bearer " + validKnownToken}},
		{id: "bearer-valid-missing", name: "bearer user given but backend can't find it", authHeaders: []string{"Bearer " + validMissingToken}},
		{id: "bearer-invalid", name: "invalid bearer user passed", authHeaders: []string{"Bearer " + invalidToken}},
		{id: "bearer-lookup-fail", name: "bearer lookup backend failure", authHeaders: []string{"Bearer " + lookupFailToken}},
		{id: "bearer-user-fail", name: "bearer user backend failure", authHeaders: []string{"Bearer " + userFailToken}},
		{id: "basic-valid-known", name: "basic user given", authHeaders: []string{basicAuthHeader(validUser.Username.String(), validBasicPassword)}},
		{id: "basic-valid-missing", name: "basic user given but backend can't find it", authHeaders: []string{basicAuthHeader(validUser.Username.String(), "missing-secret")}},
		{id: "basic-invalid", name: "invalid basic user passed", authHeaders: []string{basicAuthHeader(invalidBasicUser, invalidBasicPass)}},
		{id: "basic-user-fail", name: "basic user backend failure", authHeaders: []string{basicAuthHeader(userFailBasicUser, userFailBasicPass)}},
		{id: "basic-malformed", name: "malformed basic auth", authHeaders: []string{"Basic !!!"}},
		{id: "mixed-auth", name: "mixed bearer and basic auth", authHeaders: []string{"Bearer " + validKnownToken, basicAuthHeader(validUser.Username.String(), validBasicPassword)}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			for _, scenario := range scenarios {
				t.Run(scenario.name, func(t *testing.T) {
					t.Parallel()

					var (
						lastAPIKey   string
						lastPassword string
					)
					h := lowlevel.NewAuthHandler(&mockAuthService{
						authenticateAPIKey: func(_ context.Context, apiKey string) (api.ValidUsername, error) {
							lastAPIKey = apiKey
							lastPassword = ""
							switch apiKey {
							case validKnownToken, validMissingToken, userFailToken:
								return validUser.Username, nil
							case invalidToken:
								return api.ValidUsername{}, unauthorizedError{}
							case lookupFailToken:
								return api.ValidUsername{}, errLookupFailure
							default:
								return api.ValidUsername{}, unauthorizedError{}
							}
						},
						authenticatePassword: func(_ context.Context, username string, password string) (api.ValidUsername, error) {
							lastAPIKey = ""
							lastPassword = password
							switch {
							case username == validUser.Username.String() && password == validBasicPassword:
								return validUser.Username, nil
							case username == validUser.Username.String() && password == "missing-secret":
								return validUser.Username, nil
							case username == userFailBasicUser && password == userFailBasicPass:
								name, err := api.NewUsername(username)
								if err != nil {
									t.Fatalf("NewUsername(%q) error = %v", username, err)
								}
								return name, nil
							default:
								return api.ValidUsername{}, unauthorizedError{}
							}
						},
						loadUser: func(_ context.Context, username api.ValidUsername) (api.ValidUserInfo, error) {
							switch lastAPIKey {
							case validKnownToken:
								return *validUser, nil
							case validMissingToken:
								return api.ValidUserInfo{}, unauthorizedError{}
							case userFailToken:
								return api.ValidUserInfo{}, errUserFailure
							}
							switch lastPassword {
							case validBasicPassword:
								return *validUser, nil
							case "missing-secret":
								return api.ValidUserInfo{}, unauthorizedError{}
							case userFailBasicPass:
								return api.ValidUserInfo{}, errUserFailure
							default:
								return api.ValidUserInfo{}, unauthorizedError{}
							}
						},
					}, nil)

					test.run(t, h, scenario, validUser)
				})
			}
		})
	}
}

func TestHandleNoAuthHonorsSuccessCode(t *testing.T) {
	t.Parallel()

	h := lowlevel.NewAuthHandler(&mockAuthService{}, nil)
	handler := lowlevel.HandleNoAuth(
		h,
		func(w http.ResponseWriter, r *http.Request) (*api.RedactedResourceResponse, error) {
			return &api.RedactedResourceResponse{
				PID:         "abc-def",
				DateCreated: "2020-01-02T03:04:05Z",
				DateUpdated: "2020-01-02T03:04:05Z",
			}, nil
		},
		lowlevel.FixedStatusCode[*api.RedactedResourceResponse](http.StatusGone),
		nil,
	)

	rec := runHandler(t, handler)
	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusGone)
	}
	wantBody := `{"pid":"abc-def","date_created":"2020-01-02T03:04:05Z","date_updated":"2020-01-02T03:04:05Z","deleted":true}` + "\n"
	if got := rec.Body.String(); got != wantBody {
		t.Fatalf("body = %q, want %q", got, wantBody)
	}
}

func TestHandleRequiredUsernamePanicsWhenUnauthorizedNotAllowed(t *testing.T) {
	t.Parallel()

	h := lowlevel.NewAuthHandler(&mockAuthService{}, nil)
	handler := lowlevel.HandleRequiredUsername(
		h,
		func(w http.ResponseWriter, r *http.Request, username *api.ValidUsername) (struct{}, error) {
			t.Fatal("impl should not be called")
			return struct{}{}, nil
		},
		lowlevel.FixedStatusCode[struct{}](http.StatusOK),
		[]api.ErrorString{api.DatabaseError},
	)

	defer expectPanic(t)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/probe", nil))
}

func TestHandleRequiredUserPanicsWhenForbiddenNotAllowed(t *testing.T) {
	t.Parallel()

	store := memory.NewStore()
	svc := service.New(store, service.NewRuntime(), service.Options{})
	key := createUserWithKey(t, store, "alice", false)

	h := lowlevel.NewAuthHandler(svc, nil)
	handler := lowlevel.HandleRequiredUser(
		h,
		func(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (struct{}, error) {
			bobUsername, err := api.NewUsername("bob")
			if err != nil {
				return struct{}{}, fmt.Errorf("api.NewUsername: %w", err)
			}
			if _, err := svc.CreateUser(r.Context(), user, api.ValidUserCreateRequest{Username: bobUsername, Superuser: true}); err != nil {
				return struct{}{}, fmt.Errorf("svc.CreateUser: %w", err)
			}
			return struct{}{}, nil
		},
		lowlevel.FixedStatusCode[struct{}](http.StatusOK),
		[]api.ErrorString{api.Unauthorized, api.DatabaseError},
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+key)

	defer expectPanic(t)
	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestHandleRequiredUserInAuthModeUnavailableInAnonymousMode(t *testing.T) {
	t.Parallel()

	var authenticateAPIKeyCalled bool
	h := lowlevel.NewAuthHandler(&mockAuthService{
		anonymousMode: true,
		authenticateAPIKey: func(context.Context, string) (api.ValidUsername, error) {
			authenticateAPIKeyCalled = true
			return api.ValidUsername{}, unauthorizedError{}
		},
	}, nil)
	handler := lowlevel.HandleRequiredUserInAuthMode(
		h,
		func(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (struct{}, error) {
			t.Fatal("handler must not be called in anonymous mode")
			return struct{}{}, nil
		},
		lowlevel.FixedStatusCode[struct{}](http.StatusOK),
		[]api.ErrorString{api.Unauthorized, api.UnavailableInAnonymousMode, api.DatabaseError},
	)

	rec := runHandler(t, handler, "any-token")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if authenticateAPIKeyCalled {
		t.Fatal("AuthenticateAPIKey was called, want anonymous-mode gate before auth")
	}
	if err := (httpfixture.Response{
		Code: http.StatusNotFound,
		Body: []byte(`{"error":"unavailable_in_anonymous_mode"}`),
	}).Compare(rec); err != nil {
		t.Fatalf("response = %v", err)
	}
}

func TestAuthHandlerLog(t *testing.T) {
	t.Parallel()

	handler := &captureHandler{}
	logger := slog.New(handler)
	authHandler := lowlevel.NewAuthHandler(&mockAuthService{}, logger)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/probe?x=1", nil)
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
	id          string
	name        string
	authHeaders []string
}

type mockAuthService struct {
	anonymousMode        bool
	authenticateAPIKey   func(context.Context, string) (api.ValidUsername, error)
	authenticatePassword func(context.Context, string, string) (api.ValidUsername, error)
	loadUser             func(context.Context, api.ValidUsername) (api.ValidUserInfo, error)
}

func (m *mockAuthService) AnonymousMode() bool {
	return m.anonymousMode
}

func (m *mockAuthService) AuthenticateAPIKey(ctx context.Context, apiKey string) (api.ValidUsername, error) {
	if m.authenticateAPIKey != nil {
		return m.authenticateAPIKey(ctx, apiKey)
	}
	return api.ValidUsername{}, unauthorizedError{}
}

func (m *mockAuthService) AuthenticatePassword(ctx context.Context, username string, password string) (api.ValidUsername, error) {
	if m.authenticatePassword != nil {
		return m.authenticatePassword(ctx, username, password)
	}
	return api.ValidUsername{}, unauthorizedError{}
}

func (m *mockAuthService) LoadUser(ctx context.Context, username api.ValidUsername) (api.ValidUserInfo, error) {
	if m.loadUser != nil {
		return m.loadUser(ctx, username)
	}
	return api.ValidUserInfo{}, unauthorizedError{}
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

func runHandler(t *testing.T, handler http.Handler, authHeaders ...string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/probe", nil)
	for _, authHeader := range authHeaders {
		req.Header.Add("Authorization", authHeader)
	}
	handler.ServeHTTP(rec, req)
	return rec
}

func expectedUsernameOutcome(required bool, scenario scenario, validUser *api.ValidUserInfo) (status int, called bool, username string) {
	if scenario.id == "no-auth" {
		if required {
			return http.StatusUnauthorized, false, ""
		}
		return http.StatusOK, true, ""
	}

	switch scenario.id {
	case "bearer-valid-known", "bearer-valid-missing", "bearer-user-fail", "basic-valid-known", "basic-valid-missing":
		return http.StatusOK, true, validUser.Username.String()
	case "basic-user-fail":
		return http.StatusOK, true, "charlie"
	case "bearer-invalid", "basic-invalid", "basic-malformed", "mixed-auth":
		return http.StatusUnauthorized, false, ""
	case "bearer-lookup-fail":
		return http.StatusInternalServerError, false, ""
	default:
		panic("unexpected scenario")
	}
}

func expectedOptionalUsernameOutcome(scenario scenario, validUser *api.ValidUserInfo) (status int, called bool, username *string) {
	status, called, value := expectedUsernameOutcome(false, scenario, validUser)
	if !called || value == "" {
		return status, called, nil
	}
	return status, called, stringPtr(value)
}

func expectedUserOutcome(required bool, scenario scenario, validUser *api.ValidUserInfo) (status int, called bool, user *api.ValidUserInfo) {
	if scenario.id == "no-auth" {
		if required {
			return http.StatusUnauthorized, false, nil
		}
		return http.StatusOK, true, nil
	}

	switch scenario.id {
	case "bearer-valid-known", "basic-valid-known":
		return http.StatusOK, true, validUser
	case "bearer-valid-missing", "bearer-invalid", "basic-valid-missing", "basic-invalid", "basic-malformed", "mixed-auth":
		return http.StatusUnauthorized, false, nil
	case "bearer-lookup-fail", "bearer-user-fail", "basic-user-fail":
		return http.StatusInternalServerError, false, nil
	default:
		panic("unexpected scenario")
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

func createUserWithKey(t *testing.T, auth backend.AuthenticationBackend, username string, superuser bool) string {
	t.Helper()

	ctx := context.Background()
	now := func() time.Time { return time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC) }

	valid, err := api.NewUsername(username)
	if err != nil {
		t.Fatalf("NewUsername(%q) error = %v", username, err)
	}
	if _, err := auth.CreateUser(ctx, api.ValidUserCreateRequest{Username: valid, Superuser: superuser}, now); err != nil {
		t.Fatalf("CreateUser(%q) error = %v", username, err)
	}

	rawKey := strings.Repeat("a", 32-len(username)) + username
	if _, err := auth.CreateKey(ctx, apikey.Default, valid, "key-1", rawKey, api.KeyIssueRequest{Comment: "test"}, now); err != nil {
		t.Fatalf("CreateKey(%q) error = %v", username, err)
	}
	return rawKey
}

func userInfoFromValid(user *api.ValidUserInfo) *api.UserInfo {
	return &api.UserInfo{
		Username:  user.Username.String(),
		Superuser: user.Superuser,
	}
}

func expectPanic(t *testing.T) {
	t.Helper()
	if recover() == nil {
		t.Fatal("expected panic")
	}
}

func mustValidUser(t *testing.T, username string, superuser bool) *api.ValidUserInfo {
	t.Helper()
	name, err := api.NewUsername(username)
	if err != nil {
		t.Fatalf("NewUsername(%q) error = %v", username, err)
	}
	return &api.ValidUserInfo{Username: name, Superuser: superuser}
}

func stringPtr(value string) *string {
	return &value
}

func basicAuthHeader(username string, password string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return "Basic " + encoded
}

func usernameStringPtr(username *api.ValidUsername) *string {
	if username == nil {
		return nil
	}
	return stringPtr(username.String())
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

func assertOptionalUserEqual(t *testing.T, got, want *api.ValidUserInfo) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("user pointer mismatch: got %v, want %v", got, want)
	case got.Username.String() != want.Username.String() || got.Superuser != want.Superuser:
		t.Fatalf("user = %+v, want %+v", *got, *want)
	}
}
