//spellchecker:words lowlevel
package lowlevel_test

//spellchecker:words context encoding base errors slog http httptest testing time github quickpid internal httpfixture server lowlevel
import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/internal/httpfixture"
	"github.com/tkw1536/quickpid/server/internal/lowlevel"
)

type authProbeResponse struct {
	Username *string       `json:"username"`
	User     *api.UserInfo `json:"user"`
}

var (
	errLookupFailure = errors.New("lookup failure")
	errUserFailure   = errors.New("user failure")
	errUserNotFound  = errors.New("user not found")
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
		run  func(*testing.T, *lowlevel.Handler, scenario, *api.ValidUserInfo)
	}{
		{
			name: "HandleNoAuth",
			run: func(t *testing.T, h *lowlevel.Handler, scenario scenario, validUser *api.ValidUserInfo) {
				t.Helper()

				var gotCalled bool
				handler := h.Public(
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
			name: "HandleOptionalUser",
			run: func(t *testing.T, h *lowlevel.Handler, scenario scenario, validUser *api.ValidUserInfo) {
				t.Helper()

				var (
					gotCalled bool
					gotUser   *api.ValidUserInfo
				)
				handler := h.Open(
					func(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (authProbeResponse, error) {
						gotCalled = true
						gotUser = user
						var username *string
						var userInfo *api.UserInfo
						if user != nil {
							username = new(user.Username.String())
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
					h := lowlevel.NewHandler(&mockAuthService{
						authenticateAPIKey: func(_ context.Context, apiKey string) (api.ValidUsername, error) {
							lastAPIKey = apiKey
							lastPassword = ""
							switch apiKey {
							case validKnownToken, validMissingToken, userFailToken:
								return validUser.Username, nil
							case invalidToken:
								return api.ValidUsername{}, errUnauthorized
							case lookupFailToken:
								return api.ValidUsername{}, errLookupFailure
							default:
								return api.ValidUsername{}, errUnauthorized
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
								return api.ValidUsername{}, errUnauthorized
							}
						},
						loadUser: func(_ context.Context, username api.ValidUsername) (api.ValidUserInfo, error) {
							switch lastAPIKey {
							case validKnownToken:
								return *validUser, nil
							case validMissingToken:
								return api.ValidUserInfo{}, errUnauthorized
							case userFailToken:
								return api.ValidUserInfo{}, errUserFailure
							}
							switch lastPassword {
							case validBasicPassword:
								return *validUser, nil
							case "missing-secret":
								return api.ValidUserInfo{}, errUnauthorized
							case userFailBasicPass:
								return api.ValidUserInfo{}, errUserFailure
							default:
								return api.ValidUserInfo{}, errUnauthorized
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

	h := lowlevel.NewHandler(&mockAuthService{}, nil)
	handler := h.Public(
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
	wantBody := `{"pid":"abc-def","dateCreated":"2020-01-02T03:04:05Z","dateUpdated":"2020-01-02T03:04:05Z","deleted":true}`
	if got := rec.Body.String(); got != wantBody {
		t.Fatalf("body = %q, want %q", got, wantBody)
	}
}

func TestHandleRequiredUserInAuthModeUnavailableInAnonymousMode(t *testing.T) {
	t.Parallel()

	var authenticateAPIKeyCalled bool
	h := lowlevel.NewHandler(&mockAuthService{
		anonymousMode: true,
		authenticateAPIKey: func(context.Context, string) (api.ValidUsername, error) {
			authenticateAPIKeyCalled = true
			return api.ValidUsername{}, errUnauthorized
		},
	}, nil)
	handler := h.Restricted(
		func(w http.ResponseWriter, r *http.Request, _ api.ValidUserInfo) (struct{}, error) {
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
		Body: []byte(`{"error":"unavailableInAnonymousMode"}`),
	}).Compare(rec); err != nil {
		t.Fatalf("response = %v", err)
	}
}

func TestImpersonation(t *testing.T) {
	t.Parallel()

	const (
		rootToken  = "root-token"
		aliceToken = "alice-token"
	)

	rootUser := mustValidUser(t, "root", true)
	aliceUser := mustValidUser(t, "alice", false)
	users := map[string]api.ValidUserInfo{
		rootUser.Username.String():  *rootUser,
		aliceUser.Username.String(): *aliceUser,
	}

	newHandler := func() *lowlevel.Handler {
		return lowlevel.NewHandler(&mockAuthService{
			authenticateAPIKey: func(_ context.Context, apiKey string) (api.ValidUsername, error) {
				switch apiKey {
				case rootToken:
					return rootUser.Username, nil
				case aliceToken:
					return aliceUser.Username, nil
				default:
					return api.ValidUsername{}, errUnauthorized
				}
			},
			loadUser: func(_ context.Context, username api.ValidUsername) (api.ValidUserInfo, error) {
				user, ok := users[username.String()]
				if !ok {
					return api.ValidUserInfo{}, api.WithErrorString(errUserNotFound, api.UserNotFound)
				}
				return user, nil
			},
		}, nil)
	}

	restrictedProbe := func(h *lowlevel.Handler) (http.HandlerFunc, *bool, **api.ValidUserInfo) {
		var (
			gotCalled bool
			gotUser   *api.ValidUserInfo
		)
		handler := h.Restricted(
			func(w http.ResponseWriter, r *http.Request, user api.ValidUserInfo) (authProbeResponse, error) {
				gotCalled = true
				gotUser = &user
				return authProbeResponse{
					Username: new(user.Username.String()),
					User:     userInfoFromValid(&user),
				}, nil
			},
			lowlevel.FixedStatusCode[authProbeResponse](http.StatusOK),
			[]api.ErrorString{api.Unauthorized, api.DatabaseError},
		)
		return handler, &gotCalled, &gotUser
	}

	t.Run("superuser impersonates other user", func(t *testing.T) {
		t.Parallel()
		handler, gotCalled, gotUser := restrictedProbe(newHandler())
		rec := runHandlerWithHeaders(t, handler, map[string][]string{
			"Authorization":       {"Bearer " + rootToken},
			api.ImpersonateHeader: {"alice"},
		})
		assertStatusAndCalled(t, rec, *gotCalled, http.StatusOK, true)
		assertOptionalUserEqual(t, *gotUser, aliceUser)
	})

	t.Run("no impersonate header keeps caller", func(t *testing.T) {
		t.Parallel()
		handler, gotCalled, gotUser := restrictedProbe(newHandler())
		rec := runHandler(t, handler, "Bearer "+rootToken)
		assertStatusAndCalled(t, rec, *gotCalled, http.StatusOK, true)
		assertOptionalUserEqual(t, *gotUser, rootUser)
	})

	t.Run("non-superuser cannot impersonate", func(t *testing.T) {
		t.Parallel()
		handler, gotCalled, _ := restrictedProbe(newHandler())
		rec := runHandlerWithHeaders(t, handler, map[string][]string{
			"Authorization":       {"Bearer " + aliceToken},
			api.ImpersonateHeader: {"root"},
		})
		assertStatusAndCalled(t, rec, *gotCalled, http.StatusUnauthorized, false)
	})

	t.Run("multiple impersonate headers unauthorized", func(t *testing.T) {
		t.Parallel()
		handler, gotCalled, _ := restrictedProbe(newHandler())
		rec := runHandlerWithHeaders(t, handler, map[string][]string{
			"Authorization":       {"Bearer " + rootToken},
			api.ImpersonateHeader: {"alice", "root"},
		})
		assertStatusAndCalled(t, rec, *gotCalled, http.StatusUnauthorized, false)
	})

	t.Run("invalid impersonate username unauthorized", func(t *testing.T) {
		t.Parallel()
		handler, gotCalled, _ := restrictedProbe(newHandler())
		rec := runHandlerWithHeaders(t, handler, map[string][]string{
			"Authorization":       {"Bearer " + rootToken},
			api.ImpersonateHeader: {"BADUPPER"},
		})
		assertStatusAndCalled(t, rec, *gotCalled, http.StatusUnauthorized, false)
	})

	t.Run("missing impersonate user unauthorized", func(t *testing.T) {
		t.Parallel()
		handler, gotCalled, _ := restrictedProbe(newHandler())
		rec := runHandlerWithHeaders(t, handler, map[string][]string{
			"Authorization":       {"Bearer " + rootToken},
			api.ImpersonateHeader: {"nobody"},
		})
		assertStatusAndCalled(t, rec, *gotCalled, http.StatusUnauthorized, false)
	})
}

func TestAuthHandlerLog(t *testing.T) {
	t.Parallel()

	handler := &captureHandler{}
	logger := slog.New(handler)
	authHandler := lowlevel.NewHandler(&mockAuthService{}, logger)
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
	if record.Level != slog.LevelDebug {
		t.Fatalf("level = %v, want %v", record.Level, slog.LevelDebug)
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
	return api.ValidUsername{}, errUnauthorized
}

func (m *mockAuthService) AuthenticatePassword(ctx context.Context, username string, password string) (api.ValidUsername, error) {
	if m.authenticatePassword != nil {
		return m.authenticatePassword(ctx, username, password)
	}
	return api.ValidUsername{}, errUnauthorized
}

func (m *mockAuthService) LoadUser(ctx context.Context, username api.ValidUsername) (api.ValidUserInfo, error) {
	if m.loadUser != nil {
		return m.loadUser(ctx, username)
	}
	return api.ValidUserInfo{}, errUnauthorized
}

var errUnauthorized = errors.New("unauthorized")

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

	headers := make(map[string][]string)
	if len(authHeaders) > 0 {
		headers["Authorization"] = append([]string{}, authHeaders...)
	}
	return runHandlerWithHeaders(t, handler, headers)
}

func runHandlerWithHeaders(t *testing.T, handler http.Handler, headers map[string][]string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/probe", nil)
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	handler.ServeHTTP(rec, req)
	return rec
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
	case "bearer-invalid", "basic-invalid", "basic-malformed", "mixed-auth", "bearer-lookup-fail":
		return http.StatusUnauthorized, false, nil
	case "bearer-valid-missing", "basic-valid-missing", "bearer-user-fail", "basic-user-fail":
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

func userInfoFromValid(user *api.ValidUserInfo) *api.UserInfo {
	return &api.UserInfo{
		Username:  user.Username.String(),
		Superuser: user.Superuser,
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

func basicAuthHeader(username string, password string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return "Basic " + encoded
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
