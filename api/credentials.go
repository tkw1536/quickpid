package api

//spellchecker:words bytes encoding json jsontext errors time github quickpid internal strict
import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"

	"github.com/tkw1536/quickpid/internal/strict"
)

// ValidPassword represents a valid password.
type ValidPassword struct {
	valid bool
	value string
}

func (password ValidPassword) String() string {
	if !password.valid {
		panic("invalid password")
	}
	return password.value
}

var errInvalidPassword = errors.New("invalid password")

// NewPassword creates a new password.
func NewPassword(value string) (ValidPassword, error) {
	if value == "" {
		return ValidPassword{}, errInvalidPassword
	}
	return ValidPassword{valid: true, value: value}, nil
}

// SetPasswordRequest is the JSON body for setUserPassword.
type SetPasswordRequest struct {
	Password *string `json:"password"`
}

func (r *SetPasswordRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Password strict.Optional[*string] `json:"password"`
	}
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if !decoded.Password.Present {
		return missingRequiredFieldError("password")
	}
	r.Password = decoded.Password.Value
	return nil
}

// Validate checks if the given request is valid.
func (r *SetPasswordRequest) Validate() (ValidSetPasswordRequest, error) {
	if r.Password == nil {
		return ValidSetPasswordRequest{}, nil
	}
	password, err := NewPassword(*r.Password)
	if err != nil {
		return ValidSetPasswordRequest{}, err
	}
	return ValidSetPasswordRequest{
		Password: &password,
	}, nil
}

// ValidSetPasswordRequest is like a [SetPasswordRequest] but with a validated password.
type ValidSetPasswordRequest struct {
	Password *ValidPassword
}

// SetPasswordResponse is returned when a password is set or unset.
type SetPasswordResponse struct {
	Password bool `json:"password"`
}

type ListKeysParams struct {
	Limit  int
	Offset int
}

type PaginatedAPIKeysResponse struct {
	Total  int          `json:"total"`
	Offset int          `json:"offset"`
	Items  []APIKeyInfo `json:"items"`
}

// APIKeyNamespaceScope is a single (namespace, NamespaceScope) grant on an API key.
type APIKeyNamespaceScope struct {
	Namespace string         `json:"namespace"`
	Scope     NamespaceScope `json:"scope"`
}

// KeyIssueRequest is the JSON body for issueKey.
type KeyIssueRequest struct {
	Comment         string                 `json:"comment"`
	ExpiresAt       *time.Time             `json:"expiresAt"`
	UserScopes      []UserScope            `json:"userScopes"`
	NamespaceScopes []APIKeyNamespaceScope `json:"namespaceScopes"`
}

func (r *KeyIssueRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Comment         strict.Optional[strict.String]             `json:"comment"`
		ExpiresAt       strict.Optional[*strict.Time]              `json:"expiresAt"`
		UserScopes      strict.Optional[strict.StringSlice]        `json:"userScopes"`
		NamespaceScopes strict.Optional[apiKeyNamespaceScopeSlice] `json:"namespaceScopes"`
	}
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.Comment.Present {
		return missingRequiredFieldError("comment")
	}
	r.Comment = string(decoded.Comment.Value)
	if !decoded.ExpiresAt.Present {
		return missingRequiredFieldError("expiresAt")
	}
	if decoded.ExpiresAt.Value == nil {
		r.ExpiresAt = nil
	} else {
		ts := decoded.ExpiresAt.Value.Time()
		r.ExpiresAt = &ts
	}
	if !decoded.UserScopes.Present {
		return missingRequiredFieldError("userScopes")
	}
	userScopes := make([]UserScope, len(decoded.UserScopes.Value))
	for i, s := range decoded.UserScopes.Value {
		userScopes[i] = UserScope(s)
	}
	r.UserScopes = userScopes
	if !decoded.NamespaceScopes.Present {
		return missingRequiredFieldError("namespaceScopes")
	}
	r.NamespaceScopes = []APIKeyNamespaceScope(decoded.NamespaceScopes.Value)
	return nil
}

// ErrUnknownScope is returned when an API key request references an undefined scope.
var ErrUnknownScope = errors.New("unknown scope")

// Validate checks that scopes and namespace ids in this request are well-formed.
func (r *KeyIssueRequest) Validate() error {
	for _, scope := range r.UserScopes {
		if _, err := scope.Definition(); err != nil {
			return fmt.Errorf("%w: %q", ErrUnknownScope, scope)
		}
	}
	for _, grant := range r.NamespaceScopes {
		if _, err := NewNamespaceID(grant.Namespace); err != nil {
			return err
		}
		if _, err := grant.Scope.Definition(); err != nil {
			return fmt.Errorf("%w: %q", ErrUnknownScope, grant.Scope)
		}
	}
	return nil
}

// IssueKeyResponse is returned when a new API key is issued.
type IssueKeyResponse struct {
	APIKeyInfo
	Key string `json:"key"`
}

// KeyRevokeRequest is the JSON body for revokeKey.
type KeyRevokeRequest struct {
	ID string `json:"id"`
}

func (r *KeyRevokeRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		ID strict.Optional[strict.String] `json:"id"`
	}
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.ID.Present {
		return missingRequiredFieldError("id")
	}
	r.ID = string(decoded.ID.Value)
	return nil
}

// APIKeyInfo describes an API key without the secret value.
type APIKeyInfo struct {
	ID        string    `json:"id"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`

	// ExpiresAt optionally holds the time at which this key will expire.
	// If it is nil, the key never expires.
	// If it is a pointer to the zero time, the key is revoked.
	ExpiresAt *time.Time `json:"expiresAt"`

	// UserScopes lists user-level scopes granted to this key.
	// An empty array means all user scopes are granted.
	UserScopes []UserScope `json:"userScopes"`

	// NamespaceScopes lists (namespace, scope) pairs granted to this key.
	// An empty array means all namespace scopes are granted for all namespaces.
	NamespaceScopes []APIKeyNamespaceScope `json:"namespaceScopes"`
}

// NormalizeScopes ensures UserScopes and NamespaceScopes are non-nil empty slices.
func (k *APIKeyInfo) NormalizeScopes() {
	if k.UserScopes == nil {
		k.UserScopes = []UserScope{}
	}
	if k.NamespaceScopes == nil {
		k.NamespaceScopes = []APIKeyNamespaceScope{}
	}
}

// Valid checks if this key is valid for the given time.
//
// The time function is only called when ExpiresAt is not nil and not the zero time.
func (k *APIKeyInfo) Valid(now func() time.Time) bool {
	return k.ExpiresAt == nil || (!k.ExpiresAt.IsZero() && k.ExpiresAt.After(now()))
}

// apiKeyNamespaceScopeSlice rejects JSON null and non-arrays.
type apiKeyNamespaceScopeSlice []APIKeyNamespaceScope

func (s *apiKeyNamespaceScopeSlice) UnmarshalJSON(data []byte) error {
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}
	if tok.Kind() != jsontext.KindBeginArray {
		return errNotAnArray
	}

	var out []APIKeyNamespaceScope
	for {
		kind := dec.PeekKind()
		if kind == jsontext.KindEndArray || kind == jsontext.KindInvalid {
			break
		}
		var elem apiKeyNamespaceScopeJSON
		if err := json.UnmarshalDecode(dec, &elem, json.RejectUnknownMembers(true)); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}
		if !elem.Namespace.Present {
			return missingRequiredFieldError("namespace")
		}
		if !elem.Scope.Present {
			return missingRequiredFieldError("scope")
		}
		out = append(out, APIKeyNamespaceScope{
			Namespace: string(elem.Namespace.Value),
			Scope:     NamespaceScope(elem.Scope.Value),
		})
	}
	tok, err = dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read last token: %w", err)
	}
	if tok.Kind() != jsontext.KindEndArray {
		return errNotAnArray
	}
	if out == nil {
		out = []APIKeyNamespaceScope{}
	}
	*s = out
	return nil
}

type apiKeyNamespaceScopeJSON struct {
	Namespace strict.Optional[strict.String] `json:"namespace"`
	Scope     strict.Optional[strict.String] `json:"scope"`
}

var errNotAnArray = errors.New("can only unmarshal JSON array")
