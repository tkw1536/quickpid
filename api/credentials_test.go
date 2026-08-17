package api_test

//spellchecker:words encoding json errors reflect strings testing time github quickpid
import (
	"encoding/json/v2"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
)

func TestSetPasswordRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		wantNil   bool
		wantValue string
	}{
		{name: "string", body: `{"password":"secret"}`, wantValue: "secret"},
		{name: "null", body: `{"password":null}`, wantNil: true},
		{name: "missing", body: `{}`, wantErr: true, wantErrIn: []string{"missing required field", "password"}},
		{name: "unknown", body: `{"password":"secret","unknown":1}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "unknown"}},
		{name: "number", body: `{"password":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var req api.SetPasswordRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				es := err.Error()
				for _, wantIn := range tt.wantErrIn {
					if !strings.Contains(es, wantIn) {
						t.Fatalf("error: got %q want substring %q", es, wantIn)
					}
				}
				return
			}
			if tt.wantNil {
				if req.Password != nil {
					t.Fatalf("req.Password = %v, want nil payload", *req.Password)
				}
				return
			}
			if req.Password == nil || *req.Password != tt.wantValue {
				t.Fatalf("req.Password = %v, want %q", req.Password, tt.wantValue)
			}
		})
	}
}

func TestSetPasswordRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		req := api.SetPasswordRequest{Password: new("secret")}
		valid, err := req.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if valid.Password == nil || valid.Password.String() != "secret" {
			t.Fatalf("Validate() = %+v, want password secret", valid)
		}
	})

	t.Run("null", func(t *testing.T) {
		t.Parallel()

		req := api.SetPasswordRequest{Password: (*string)(nil)}
		valid, err := req.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if valid.Password != nil {
			t.Fatalf("Validate() = %+v, want nil password", valid)
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		req := api.SetPasswordRequest{Password: new("")}
		_, err := req.Validate()
		if err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "invalid password") {
			t.Fatalf("Validate() error = %q, want invalid password", err.Error())
		}
	})
}

func TestKeyIssueRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.KeyIssueRequest
	}{
		{
			name:      "fail_missingExpiresAt",
			body:      `{"comment":"dev key","userScopes":[],"namespaceScopes":[]}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "expiresAt"},
		},
		{
			name:    "ok_expiresAtSet",
			body:    `{"comment":"dev key","expiresAt":"2026-12-31T00:00:00Z","userScopes":[],"namespaceScopes":[]}`,
			wantErr: false,
			want: api.KeyIssueRequest{
				Comment:         "dev key",
				ExpiresAt:       new(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)),
				UserScopes:      []api.UserScope{},
				NamespaceScopes: []api.APIKeyNamespaceScope{},
			},
		},
		{
			name:      "fail_usernameInBody",
			body:      `{"comment":"dev key","expiresAt":"2026-12-31T00:00:00Z","userScopes":[],"namespaceScopes":[],"username":"alice"}`,
			wantErr:   true,
			wantErrIn: []string{"unknown object member name", "username"},
		},
		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"json is null"},
		},
		{
			name:      "fail_missingComment",
			body:      `{}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "comment"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"comment":"dev key","expiresAt":null,"userScopes":[],"namespaceScopes":[],"unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "unknown"},
		},
		{
			name:      "fail_commentNull",
			body:      `{"comment":null,"expiresAt":null,"userScopes":[],"namespaceScopes":[]}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:    "ok_expiresAtNull",
			body:    `{"comment":"dev key","expiresAt":null,"userScopes":[],"namespaceScopes":[]}`,
			wantErr: false,
			want: api.KeyIssueRequest{
				Comment:         "dev key",
				ExpiresAt:       nil,
				UserScopes:      []api.UserScope{},
				NamespaceScopes: []api.APIKeyNamespaceScope{},
			},
		},
		{
			name:    "ok_withScopes",
			body:    `{"comment":"dev key","expiresAt":null,"userScopes":["listOwnKeys","getUserInfo"],"namespaceScopes":[{"namespace":"ns1","scope":"getResource"}]}`,
			wantErr: false,
			want: api.KeyIssueRequest{
				Comment:    "dev key",
				ExpiresAt:  nil,
				UserScopes: []api.UserScope{api.ScopeListUserKeys, api.ScopeGetUserInfo},
				NamespaceScopes: []api.APIKeyNamespaceScope{
					{Namespace: "ns1", Scope: api.ScopeGetResource},
				},
			},
		},
		{
			name:      "fail_missingUserScopes",
			body:      `{"comment":"dev key","expiresAt":null,"namespaceScopes":[]}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "userScopes"},
		},
		{
			name:      "fail_missingNamespaceScopes",
			body:      `{"comment":"dev key","expiresAt":null,"userScopes":[]}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "namespaceScopes"},
		},
		{
			name:      "fail_userScopesNull",
			body:      `{"comment":"dev key","expiresAt":null,"userScopes":null,"namespaceScopes":[]}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_namespaceScopesNull",
			body:      `{"comment":"dev key","expiresAt":null,"userScopes":[],"namespaceScopes":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_expiresAtMalformed",
			body:      `{"comment":"dev key","expiresAt":"not-a-time","userScopes":[],"namespaceScopes":[]}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_usernameNull",
			body:      `{"comment":"dev key","expiresAt":null,"userScopes":[],"namespaceScopes":[],"username":null}`,
			wantErr:   true,
			wantErrIn: []string{"unknown object member name", "username"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.KeyIssueRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err != nil && len(tt.wantErrIn) > 0 {
				es := err.Error()
				for _, wantIn := range tt.wantErrIn {
					if !strings.Contains(es, wantIn) {
						t.Fatalf("error: got %q want substring %q", es, wantIn)
					}
				}
			}
			if err == nil && !tt.wantErr && !reflect.DeepEqual(req, tt.want) {
				t.Fatalf("req: got %+v want %+v", req, tt.want)
			}
		})
	}
}

func TestKeyIssueRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		req       api.KeyIssueRequest
		wantErrIn []string
		wantIs    error
	}{
		{
			name: "ok_empty",
			req: api.KeyIssueRequest{
				UserScopes:      []api.UserScope{},
				NamespaceScopes: []api.APIKeyNamespaceScope{},
			},
		},
		{
			name: "ok_known",
			req: api.KeyIssueRequest{
				UserScopes: []api.UserScope{api.ScopeListUserKeys},
				NamespaceScopes: []api.APIKeyNamespaceScope{
					{Namespace: "ns1", Scope: api.ScopeGetResource},
				},
			},
		},
		{
			name: "fail_unknownUserScope",
			req: api.KeyIssueRequest{
				UserScopes:      []api.UserScope{"notARealScope"},
				NamespaceScopes: []api.APIKeyNamespaceScope{},
			},
			wantErrIn: []string{"unknown scope", "notARealScope"},
			wantIs:    api.ErrUnknownScope,
		},
		{
			name: "fail_unknownNamespaceScope",
			req: api.KeyIssueRequest{
				UserScopes: []api.UserScope{},
				NamespaceScopes: []api.APIKeyNamespaceScope{
					{Namespace: "ns1", Scope: "notARealScope"},
				},
			},
			wantErrIn: []string{"unknown scope", "notARealScope"},
			wantIs:    api.ErrUnknownScope,
		},
		{
			name: "fail_invalidNamespace",
			req: api.KeyIssueRequest{
				UserScopes: []api.UserScope{},
				NamespaceScopes: []api.APIKeyNamespaceScope{
					{Namespace: "BAD!", Scope: api.ScopeGetResource},
				},
			},
			wantErrIn: []string{"invalid namespace id"},
			wantIs:    api.ErrInvalidNamespaceID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.req.Validate()
			if len(tt.wantErrIn) == 0 {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			es := err.Error()
			for _, wantIn := range tt.wantErrIn {
				if !strings.Contains(es, wantIn) {
					t.Fatalf("Validate() error = %q, want substring %q", es, wantIn)
				}
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("Validate() error = %v, want errors.Is(..., %v)", err, tt.wantIs)
			}
		})
	}
}

func TestKeyRevokeRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.KeyRevokeRequest
	}{
		{
			name:    "ok",
			body:    `{"id":"key-123"}`,
			wantErr: false,
			want: api.KeyRevokeRequest{
				ID: "key-123",
			},
		},
		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"json is null"},
		},
		{
			name:      "fail_missingID",
			body:      `{}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "id"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"id":"key-123","unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "unknown"},
		},
		{
			name:      "fail_idNull",
			body:      `{"id":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.KeyRevokeRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err != nil && len(tt.wantErrIn) > 0 {
				es := err.Error()
				for _, wantIn := range tt.wantErrIn {
					if !strings.Contains(es, wantIn) {
						t.Fatalf("error: got %q want substring %q", es, wantIn)
					}
				}
			}
			if err == nil && !tt.wantErr && !reflect.DeepEqual(req, tt.want) {
				t.Fatalf("req: got %+v want %+v", req, tt.want)
			}
		})
	}
}

func TestNewPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:  "valid_simple",
			input: "secret",
		},
		{
			name:  "valid_with_spaces",
			input: "secret phrase",
		},
		{
			name:    "invalid_empty",
			input:   "",
			wantErr: "invalid password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			password, err := api.NewPassword(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("NewPassword(%q) error = %v, want nil", tt.input, err)
				}
				if password.String() != tt.input {
					t.Fatalf("password.String() = %q, want %q", password.String(), tt.input)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewPassword(%q) error = nil, want %q", tt.input, tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("NewPassword(%q) error = %q, want %q", tt.input, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestAPIKeyInfo_HasUserScope(t *testing.T) {
	t.Parallel()

	nsGrant := []api.APIKeyNamespaceScope{{Namespace: "ns1", Scope: api.ScopeGetResource}}

	tests := []struct {
		name            string
		userScopes      []api.UserScope
		namespaceScopes []api.APIKeyNamespaceScope
		query           api.UserScope
		want            bool
	}{
		{
			name:            "nil_both_allows_all",
			userScopes:      nil,
			namespaceScopes: nil,
			query:           api.ScopeGetUserInfo,
			want:            true,
		},
		{
			name:            "empty_both_allows_all",
			userScopes:      []api.UserScope{},
			namespaceScopes: []api.APIKeyNamespaceScope{},
			query:           api.ScopeListUserKeys,
			want:            true,
		},
		{
			name:            "empty_user_with_namespace_grants_none",
			userScopes:      []api.UserScope{},
			namespaceScopes: nsGrant,
			query:           api.ScopeGetUserInfo,
			want:            false,
		},
		{
			name:            "nil_user_with_namespace_grants_none",
			userScopes:      nil,
			namespaceScopes: nsGrant,
			query:           api.ScopeGetUserInfo,
			want:            false,
		},
		{
			name:            "present",
			userScopes:      []api.UserScope{api.ScopeListUserKeys, api.ScopeGetUserInfo},
			namespaceScopes: nil,
			query:           api.ScopeGetUserInfo,
			want:            true,
		},
		{
			name:            "present_with_namespace_scopes",
			userScopes:      []api.UserScope{api.ScopeListUserKeys, api.ScopeGetUserInfo},
			namespaceScopes: nsGrant,
			query:           api.ScopeGetUserInfo,
			want:            true,
		},
		{
			name:            "absent",
			userScopes:      []api.UserScope{api.ScopeListUserKeys},
			namespaceScopes: nil,
			query:           api.ScopeGetUserInfo,
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key := &api.APIKeyInfo{UserScopes: tt.userScopes, NamespaceScopes: tt.namespaceScopes}
			if got := key.HasUserScope(tt.query); got != tt.want {
				t.Fatalf("HasUserScope(%q) = %v, want %v", tt.query, got, tt.want)
			}
			// Second call exercises the cached index.
			if got := key.HasUserScope(tt.query); got != tt.want {
				t.Fatalf("HasUserScope(%q) cached = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestAPIKeyInfo_HasNamespaceScope(t *testing.T) {
	t.Parallel()

	grant := api.APIKeyNamespaceScope{Namespace: "ns1", Scope: api.ScopeGetResource}
	userGrant := []api.UserScope{api.ScopeGetUserInfo}

	tests := []struct {
		name            string
		userScopes      []api.UserScope
		namespaceScopes []api.APIKeyNamespaceScope
		query           api.APIKeyNamespaceScope
		want            bool
	}{
		{
			name:            "nil_both_allows_all",
			userScopes:      nil,
			namespaceScopes: nil,
			query:           grant,
			want:            true,
		},
		{
			name:            "empty_both_allows_all",
			userScopes:      []api.UserScope{},
			namespaceScopes: []api.APIKeyNamespaceScope{},
			query:           grant,
			want:            true,
		},
		{
			name:            "empty_namespace_with_user_grants_none",
			userScopes:      userGrant,
			namespaceScopes: []api.APIKeyNamespaceScope{},
			query:           grant,
			want:            false,
		},
		{
			name:            "nil_namespace_with_user_grants_none",
			userScopes:      userGrant,
			namespaceScopes: nil,
			query:           grant,
			want:            false,
		},
		{
			name:            "exact_match",
			userScopes:      nil,
			namespaceScopes: []api.APIKeyNamespaceScope{grant},
			query:           grant,
			want:            true,
		},
		{
			name:            "exact_match_with_user_scopes",
			userScopes:      userGrant,
			namespaceScopes: []api.APIKeyNamespaceScope{grant},
			query:           grant,
			want:            true,
		},
		{
			name:            "wrong_namespace",
			userScopes:      nil,
			namespaceScopes: []api.APIKeyNamespaceScope{grant},
			query:           api.APIKeyNamespaceScope{Namespace: "ns2", Scope: api.ScopeGetResource},
			want:            false,
		},
		{
			name:            "wrong_scope",
			userScopes:      nil,
			namespaceScopes: []api.APIKeyNamespaceScope{grant},
			query:           api.APIKeyNamespaceScope{Namespace: "ns1", Scope: api.ScopeListResources},
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key := &api.APIKeyInfo{UserScopes: tt.userScopes, NamespaceScopes: tt.namespaceScopes}
			if got := key.HasNamespaceScope(tt.query); got != tt.want {
				t.Fatalf("HasNamespaceScope(%+v) = %v, want %v", tt.query, got, tt.want)
			}
			if got := key.HasNamespaceScope(tt.query); got != tt.want {
				t.Fatalf("HasNamespaceScope(%+v) cached = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

