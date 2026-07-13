//spellchecker:words server
package server

//spellchecker:words context encoding json http httptest strings testing time github bicpid backend memory internal apikey service
import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/memory"
	"github.com/tkw1536/bicpid/internal/apikey"
	"github.com/tkw1536/bicpid/service"
)

func testHandler(t *testing.T, store backend.Store) *Server {
	t.Helper()
	svc := service.New(
		store,
		store,
		store,
		service.NewRuntime(),
		service.Options{},
	)
	return NewServer(
		Options{DisableSwaggerUI: true},
		svc,
		nil,
	)
}

func testAnonymousHandler(t *testing.T, store backend.Store) *Server {
	t.Helper()
	svc := service.New(
		store,
		store,
		store,
		service.NewRuntime(),
		service.Options{Anonymous: true},
	)
	return NewServer(
		Options{DisableSwaggerUI: true, Anonymous: true},
		svc,
		nil,
	)
}

func createUserWithKey(t *testing.T, auth backend.AuthenticationBackend, username string, superuser bool) string {
	t.Helper()
	ctx := context.Background()
	now := func() time.Time { return time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC) }

	if _, err := auth.CreateUser(ctx, api.UserCreateRequest{Username: username, Superuser: superuser}, now); err != nil {
		t.Fatalf("CreateUser(%q) error = %v", username, err)
	}
	rawKey := strings.Repeat("a", 32-len(username)) + username
	if _, err := auth.CreateKey(ctx, apikey.Default, username, "key-1", rawKey, api.KeyIssueRequest{Comment: "test"}, now); err != nil {
		t.Fatalf("CreateKey(%q) error = %v", username, err)
	}
	return rawKey
}

func TestGetCurrentUser_UnauthorizedJSON(t *testing.T) {
	h := testHandler(t, memory.NewStore())

	t.Run("missing authorization header", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/user/", nil)
		h.ServeHTTP(rec, req)
		assertUnauthorizedJSON(t, rec)
	})

	t.Run("invalid bearer token", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/user/", nil)
		req.Header.Set("Authorization", "Bearer invalidtoken0000000000000000")
		h.ServeHTTP(rec, req)
		assertUnauthorizedJSON(t, rec)
	})
}

func assertUnauthorizedJSON(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	var resp api.ErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body = %q", err, body)
	}
	if resp.Error != string(api.Unauthorized) {
		t.Fatalf("error = %q, want %q", resp.Error, api.Unauthorized)
	}
}

func assertForbiddenJSON(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	var resp api.ErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body = %q", err, body)
	}
	if resp.Error != string(api.Forbidden) {
		t.Fatalf("error = %q, want %q", resp.Error, api.Forbidden)
	}
}

func TestUpdateCurrentUser_Forbidden(t *testing.T) {
	auth := memory.NewStore()
	aliceKey := createUserWithKey(t, auth, "alice", false)
	rootKey := createUserWithKey(t, auth, "root", true)
	h := testHandler(t, auth)

	t.Run("non-superuser cannot patch", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/user/?username=alice", strings.NewReader(`{"superuser":true}`))
		req.Header.Set("Authorization", "Bearer "+aliceKey)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rec, req)
		assertForbiddenJSON(t, rec)
	})

	t.Run("non-superuser cannot update another user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/user/?username=root", strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer "+aliceKey)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rec, req)
		assertForbiddenJSON(t, rec)
	})

	t.Run("superuser can update another user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/user/?username=alice", strings.NewReader(`{"superuser":true}`))
		req.Header.Set("Authorization", "Bearer "+rootKey)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		user, err := auth.GetUser(context.Background(), "alice")
		if err != nil {
			t.Fatalf("GetUser(alice) error = %v", err)
		}
		if !user.Superuser {
			t.Fatalf("alice superuser = false, want true")
		}
	})
}

func TestUpdateCurrentUser_ForbiddenSelfUpdate(t *testing.T) {
	auth := memory.NewStore()
	rootKey := createUserWithKey(t, auth, "root", true)
	h := testHandler(t, auth)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/user/?username=root", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+rootKey)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	assertForbiddenJSON(t, rec)
}

func TestCreateUser_RequiresSuperuser(t *testing.T) {
	auth := memory.NewStore()
	aliceKey := createUserWithKey(t, auth, "alice", false)
	rootKey := createUserWithKey(t, auth, "root", true)
	h := testHandler(t, auth)

	t.Run("unauthenticated", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/user/", strings.NewReader(`{"username":"bob"}`))
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rec, req)
		assertUnauthorizedJSON(t, rec)
	})

	t.Run("non-superuser forbidden", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/user/", strings.NewReader(`{"username":"bob"}`))
		req.Header.Set("Authorization", "Bearer "+aliceKey)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rec, req)
		assertForbiddenJSON(t, rec)
	})

	t.Run("superuser creates user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/user/", strings.NewReader(`{"username":"bob"}`))
		req.Header.Set("Authorization", "Bearer "+rootKey)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}
	})
}

func TestListUsers_RequiresSuperuser(t *testing.T) {
	auth := memory.NewStore()
	aliceKey := createUserWithKey(t, auth, "alice", false)
	rootKey := createUserWithKey(t, auth, "root", true)
	h := testHandler(t, auth)

	t.Run("non-superuser forbidden", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/users/", nil)
		req.Header.Set("Authorization", "Bearer "+aliceKey)
		h.ServeHTTP(rec, req)
		assertForbiddenJSON(t, rec)
	})

	t.Run("superuser lists users", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/users/", nil)
		req.Header.Set("Authorization", "Bearer "+rootKey)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var page api.PaginatedUsersResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if page.Total < 2 {
			t.Fatalf("total = %d, want at least 2", page.Total)
		}
	})
}

func TestIssueAndRevokeKey(t *testing.T) {
	auth := memory.NewStore()
	aliceKey := createUserWithKey(t, auth, "alice", false)
	h := testHandler(t, auth)

	rec := httptest.NewRecorder()
	issueReq := httptest.NewRequest(http.MethodPost, "/user/key", strings.NewReader(`{"comment":"new key"}`))
	issueReq.Header.Set("Authorization", "Bearer "+aliceKey)
	issueReq.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, issueReq)

	if rec.Code != http.StatusCreated {
		t.Fatalf("issue status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var issued api.IssueKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &issued); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if issued.Key == "" || issued.ID == "" {
		t.Fatalf("issue response = %+v", issued)
	}

	rec = httptest.NewRecorder()
	revokeReq := httptest.NewRequest(http.MethodPost, "/user/key/revoke", strings.NewReader(`{"id":"`+issued.ID+`"}`))
	revokeReq.Header.Set("Authorization", "Bearer "+aliceKey)
	revokeReq.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, revokeReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestResolverAnonymousMode_AllowsRequestsWithoutAuth(t *testing.T) {
	store := memory.NewStore()
	h := testAnonymousHandler(t, store)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/resolver/namespaces", strings.NewReader(`{"tag":"anon","pid_format":{"pattern":"***-***","characters":"full"}}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create namespace status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/resolver/namespaces", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list namespaces status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
