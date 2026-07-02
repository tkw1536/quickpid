//spellchecker:words auth
package auth_test

//spellchecker:words slog http httptest testing github quickpid auth
import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tkw1536/quickpid/auth"
)

type trackingAuth struct {
	canListNamespaces    bool
	canListNamespacesErr error
	level                auth.Level
	levelErr             error
	canCreate            bool
	canCreateErr         error
	onCreateNamespaceErr error

	canListNamespacesCalls int
	levelCalls             int
	canCreateCalls         int
	onCreateNamespaceCalls int
	onUnauthorizedCalls    int
	lastLevelNamespace     string
	lastOnCreateNamespace  string
}

func (a *trackingAuth) CanListNamespaces(_ *slog.Logger, _ *http.Request) (bool, error) {
	a.canListNamespacesCalls++
	return a.canListNamespaces, a.canListNamespacesErr
}

func (a *trackingAuth) GetNamespaceLevel(_ *slog.Logger, _ *http.Request, namespace string) (auth.Level, error) {
	a.levelCalls++
	a.lastLevelNamespace = namespace
	return a.level, a.levelErr
}

func (a *trackingAuth) CanCreate(_ *slog.Logger, _ *http.Request) (bool, error) {
	a.canCreateCalls++
	return a.canCreate, a.canCreateErr
}

func (a *trackingAuth) OnCreateNamespace(_ *slog.Logger, _ *http.Request, namespace string) error {
	a.onCreateNamespaceCalls++
	a.lastOnCreateNamespace = namespace
	return a.onCreateNamespaceErr
}

func (a *trackingAuth) OnUnauthorized(_ *slog.Logger, w http.ResponseWriter, _ *http.Request) {
	a.onUnauthorizedCalls++
	w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

type forwardRecorder struct {
	calls  int
	status int
	body   string
}

func (f *forwardRecorder) handler(w http.ResponseWriter, _ *http.Request) {
	f.calls++
	w.WriteHeader(f.status)
	if f.body != "" {
		_, _ = w.Write([]byte(f.body))
	}
}

func newAuthProxy(authImpl auth.Authorization, forward *forwardRecorder) *auth.Proxy {
	return &auth.Proxy{
		Auth:    authImpl,
		Forward: forward.handler,
	}
}

func TestAuthProxy_UnauthenticatedRouteForwardsWithoutAuth(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{canListNamespaces: true, level: auth.LevelEditor, canCreate: true}
	forward := &forwardRecorder{status: http.StatusOK, body: "resolver-list"}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.canListNamespacesCalls != 0 || mockAuth.levelCalls != 0 || mockAuth.canCreateCalls != 0 {
		t.Fatalf("auth calls = canListNamespaces %d level %d canCreate %d, want 0 for unauthenticated route", mockAuth.canListNamespacesCalls, mockAuth.levelCalls, mockAuth.canCreateCalls)
	}
	if forward.calls != 1 {
		t.Fatalf("forward calls = %d, want 1", forward.calls)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "resolver-list" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "resolver-list")
	}
}

func TestAuthProxy_ListNamespacesCallsCanListNamespacesAndForwardsWhenAuthorized(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{canListNamespaces: true}
	forward := &forwardRecorder{status: http.StatusOK, body: "namespace-list"}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.canListNamespacesCalls != 1 {
		t.Fatalf("canListNamespaces calls = %d, want 1", mockAuth.canListNamespacesCalls)
	}
	if forward.calls != 1 {
		t.Fatalf("forward calls = %d, want 1", forward.calls)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthProxy_ListNamespacesDoesNotForwardWhenCanListNamespacesReturnsFalse(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{canListNamespaces: false}
	forward := &forwardRecorder{status: http.StatusOK, body: "should-not-reach-client"}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.canListNamespacesCalls != 1 {
		t.Fatalf("canListNamespaces calls = %d, want 1", mockAuth.canListNamespacesCalls)
	}
	if forward.calls != 0 {
		t.Fatalf("forward calls = %d, want 0 when CanListNamespaces returns false", forward.calls)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAuthProxy_ListNamespacesDoesNotForwardWhenCanListNamespacesFails(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{canListNamespacesErr: errUnauthorized}
	forward := &forwardRecorder{status: http.StatusOK}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.canListNamespacesCalls != 1 {
		t.Fatalf("canListNamespaces calls = %d, want 1", mockAuth.canListNamespacesCalls)
	}
	if forward.calls != 0 {
		t.Fatalf("forward calls = %d, want 0 when CanListNamespaces returns an error", forward.calls)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if mockAuth.onUnauthorizedCalls != 1 {
		t.Fatalf("onUnauthorized calls = %d, want 1", mockAuth.onUnauthorizedCalls)
	}
}

func TestAuthProxy_MinLevelRouteCallsLevelAndForwardsWhenAuthorized(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{level: auth.LevelEditor}
	forward := &forwardRecorder{status: http.StatusOK, body: "namespace-resources"}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces/my-ns/resources", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.levelCalls != 1 {
		t.Fatalf("level calls = %d, want 1", mockAuth.levelCalls)
	}
	if mockAuth.lastLevelNamespace != "my-ns" {
		t.Fatalf("level namespace = %q, want %q", mockAuth.lastLevelNamespace, "my-ns")
	}
	if forward.calls != 1 {
		t.Fatalf("forward calls = %d, want 1", forward.calls)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthProxy_MinLevelRouteDoesNotForwardWhenLevelInsufficient(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{level: auth.LevelContributor}
	forward := &forwardRecorder{status: http.StatusOK, body: "should-not-reach-client"}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces/my-ns/resources", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.levelCalls != 1 {
		t.Fatalf("level calls = %d, want 1", mockAuth.levelCalls)
	}
	if forward.calls != 0 {
		t.Fatalf("forward calls = %d, want 0 when authorization level is insufficient", forward.calls)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAuthProxy_MinLevelRouteDoesNotForwardWhenLevelFails(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{levelErr: errUnauthorized}
	forward := &forwardRecorder{status: http.StatusOK}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces/my-ns/resources", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.levelCalls != 1 {
		t.Fatalf("level calls = %d, want 1", mockAuth.levelCalls)
	}
	if forward.calls != 0 {
		t.Fatalf("forward calls = %d, want 0 when authorization fails", forward.calls)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if mockAuth.onUnauthorizedCalls != 1 {
		t.Fatalf("onUnauthorized calls = %d, want 1", mockAuth.onUnauthorizedCalls)
	}
}

func TestAuthProxy_CreateNamespaceCallsCanCreateAndForwardsWhenAuthorized(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{canCreate: true}
	forward := &forwardRecorder{
		status: http.StatusCreated,
		body:   `{"id":"new-ns","tag":"new-ns","pid_format":{"pattern":"***","characters":"full"},"date_created":"2026-01-01T00:00:00Z"}`,
	}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/resolver/namespaces", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.canCreateCalls != 1 {
		t.Fatalf("canCreate calls = %d, want 1", mockAuth.canCreateCalls)
	}
	if forward.calls != 1 {
		t.Fatalf("forward calls = %d, want 1", forward.calls)
	}
	if mockAuth.onCreateNamespaceCalls != 1 {
		t.Fatalf("onCreateNamespace calls = %d, want 1 after 201 response", mockAuth.onCreateNamespaceCalls)
	}
	if mockAuth.lastOnCreateNamespace != "new-ns" {
		t.Fatalf("onCreateNamespace namespace = %q, want %q", mockAuth.lastOnCreateNamespace, "new-ns")
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestAuthProxy_CreateNamespaceDoesNotForwardWhenCanCreateReturnsFalse(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{canCreate: false}
	forward := &forwardRecorder{status: http.StatusCreated, body: `{"id":"new-ns"}`}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/resolver/namespaces", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.canCreateCalls != 1 {
		t.Fatalf("canCreate calls = %d, want 1", mockAuth.canCreateCalls)
	}
	if forward.calls != 0 {
		t.Fatalf("forward calls = %d, want 0 when CanCreate returns false", forward.calls)
	}
	if mockAuth.onCreateNamespaceCalls != 0 {
		t.Fatalf("onCreateNamespace calls = %d, want 0 when request is not forwarded", mockAuth.onCreateNamespaceCalls)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAuthProxy_CreateNamespaceDoesNotForwardWhenCanCreateFails(t *testing.T) {
	t.Parallel()

	mockAuth := &trackingAuth{canCreateErr: errUnauthorized}
	forward := &forwardRecorder{status: http.StatusCreated}

	proxy := newAuthProxy(mockAuth, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/resolver/namespaces", nil)

	proxy.ServeHTTP(rec, req)

	if mockAuth.canCreateCalls != 1 {
		t.Fatalf("canCreate calls = %d, want 1", mockAuth.canCreateCalls)
	}
	if forward.calls != 0 {
		t.Fatalf("forward calls = %d, want 0 when CanCreate returns an error", forward.calls)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if mockAuth.onUnauthorizedCalls != 1 {
		t.Fatalf("onUnauthorized calls = %d, want 1", mockAuth.onUnauthorizedCalls)
	}
}

//spellchecker:disable-next-line
const testBcryptSecret = "$2y$05$y6RJtugGZOvBPuDg/f8.7O8GqTZAvUD8nNm1ipNrS8LiAm28fbdk2"

func TestAuthProxy_BasicAuthorizationPromptsOnProtectedRoute(t *testing.T) {
	t.Parallel()

	authz := &auth.BasicAuthorization{
		Authentication: auth.BasicAuthentication{
			Users: []string{"alice:" + testBcryptSecret},
		},
		Superusers: []string{"alice"},
	}
	forward := &forwardRecorder{status: http.StatusOK}

	proxy := newAuthProxy(authz, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces/my-ns/resources", nil)

	proxy.ServeHTTP(rec, req)

	if forward.calls != 0 {
		t.Fatalf("forward calls = %d, want 0 without credentials", forward.calls)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	wantAuth := `Basic realm="quickpid"`
	if got := rec.Header().Get("WWW-Authenticate"); got != wantAuth {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, wantAuth)
	}
}

func TestAuthProxy_BasicAuthorizationPromptsOnListNamespacesWithoutCredentials(t *testing.T) {
	t.Parallel()

	authz := &auth.BasicAuthorization{
		Authentication: auth.BasicAuthentication{
			Users: []string{"alice:" + testBcryptSecret},
		},
	}
	forward := &forwardRecorder{status: http.StatusOK}

	proxy := newAuthProxy(authz, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces", nil)

	proxy.ServeHTTP(rec, req)

	if forward.calls != 0 {
		t.Fatalf("forward calls = %d, want 0 without credentials", forward.calls)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	wantAuth := `Basic realm="quickpid"`
	if got := rec.Header().Get("WWW-Authenticate"); got != wantAuth {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, wantAuth)
	}
}

func TestAuthProxy_BasicAuthorizationAllowsListNamespacesForAuthenticatedUser(t *testing.T) {
	t.Parallel()

	authz := &auth.BasicAuthorization{
		Authentication: auth.BasicAuthentication{
			Users: []string{"alice:" + testBcryptSecret},
		},
	}
	forward := &forwardRecorder{status: http.StatusOK, body: "namespace-list"}

	proxy := newAuthProxy(authz, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces", nil)
	req.SetBasicAuth("alice", "secret")

	proxy.ServeHTTP(rec, req)

	if forward.calls != 1 {
		t.Fatalf("forward calls = %d, want 1 with valid credentials", forward.calls)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthProxy_BasicAuthorizationAllowsAnonymousPIDRead(t *testing.T) {
	t.Parallel()

	authz := &auth.BasicAuthorization{
		Authentication: auth.BasicAuthentication{
			Users: []string{"alice:" + testBcryptSecret},
		},
	}
	forward := &forwardRecorder{status: http.StatusOK, body: "pid-resource"}

	proxy := newAuthProxy(authz, forward)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resolver/namespaces/my-ns/resources/abc123", nil)

	proxy.ServeHTTP(rec, req)

	if forward.calls != 1 {
		t.Fatalf("forward calls = %d, want 1 for LevelNone route without credentials", forward.calls)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

var errUnauthorized = &authTestError{msg: "unauthorized"}

type authTestError struct {
	msg string
}

func (e *authTestError) Error() string { return e.msg }
