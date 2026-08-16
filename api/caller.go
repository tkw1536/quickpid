package api

//spellchecker:words encoding json errors
import (
	"encoding/json/v2"
	"fmt"
)

// Caller returns a [Caller] struct for the given user info and authentication method.
func (u *ValidUserInfo) Caller(method AuthenticationMethod) Caller {
	return Caller{
		username:  u.Username,
		superuser: u.Superuser,
		password:  u.Password,
		method:    method,
	}
}

// Caller represents an caller of the api.
//
// It holds information like a [ValidUserInfo], but also includes the authentication method used.
type Caller struct {
	username  ValidUsername
	superuser bool
	password  bool
	method    AuthenticationMethod
}

func (c *Caller) Method() AuthenticationMethod {
	if c == nil || c.method == nil {
		return NoAuthentication{}
	}
	return c.method
}

func (c *Caller) Username() ValidUsername {
	if c == nil {
		return ValidUsername{}
	}
	return c.username
}

func (c *Caller) Superuser() bool {
	if c == nil {
		return false
	}
	return c.superuser
}

// CompleteInfo returns the complete [UserInfo] for this authenticated user.
//
// This info does not reflect any limitations in this authenticated user info, and retains full permissions.
// It MUST NOT be used for authorization decisions.
func (c *Caller) PlainInfo() UserInfo {
	return UserInfo{
		Username:  c.username.String(),
		Superuser: c.superuser,
		Password:  c.password,
	}
}

// AuthenticationMethod holds a reason why a user was authenticated.
//
// These are implemented by [Impersonation], [BasicAuthentication], [TokenAuthentication], or [NoAuthentication].
type AuthenticationMethod interface {
	isAuthenticationMethod()

	// The following two methods check if this AuthenticationMethod is valid for the given scope.
	// These do not determine eligibileity to perform the action, but are only intended to perform an additional retriction.
	// For example, a token might only be valid for a specific namespace.

	AllowsUserScope(scope UserScope) error
	AllowsNamespaceScope(namespace ValidNamespaceID, scope NamespaceScope) error

	json.Marshaler
}

var (
	_ AuthenticationMethod = NoAuthentication{}
	_ AuthenticationMethod = Impersonation{}
	_ AuthenticationMethod = BasicAuthentication{}
	_ AuthenticationMethod = TokenAuthentication{}
)

// NoAuthentication implies that a user is not authenticated.
type NoAuthentication struct{}

func (NoAuthentication) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"none"}`), nil
}

func (NoAuthentication) AllowsUserScope(scope UserScope) error {
	return nil
}

func (NoAuthentication) AllowsNamespaceScope(namespace ValidNamespaceID, scope NamespaceScope) error {
	return nil
}

func (NoAuthentication) isAuthenticationMethod() {}

// Impersonation implies that a user is authenticated because a user was impersonated.
type Impersonation struct {
	// The original user that was impersonated prior to impersonation.
	User ValidUsername
	// The reason
	Reason AuthenticationMethod
}

func (u Impersonation) MarshalJSON() ([]byte, error) {
	bytes, err := json.Marshal(struct {
		Type   string               `json:"type"`
		User   string               `json:"user"`
		Reason AuthenticationMethod `json:"reason"`
	}{
		Type:   "impersonate",
		User:   u.User.String(),
		Reason: u.Reason,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal impersonate method: %w", err)
	}
	return bytes, nil
}
func (Impersonation) isAuthenticationMethod() {}

func (u Impersonation) AllowsUserScope(scope UserScope) error {
	return nil
}

func (u Impersonation) AllowsNamespaceScope(namespace ValidNamespaceID, scope NamespaceScope) error {
	return nil
}

// BasicAuthentication implies that a user is authenticated because they provided a username and password.
type BasicAuthentication struct{}

func (BasicAuthentication) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"basic"}`), nil
}

func (BasicAuthentication) isAuthenticationMethod() {}

func (BasicAuthentication) AllowsUserScope(scope UserScope) error {
	return nil
}

func (BasicAuthentication) AllowsNamespaceScope(namespace ValidNamespaceID, scope NamespaceScope) error {
	return nil
}

// TokenAuthentication implies that a user is authenticated because they provided a token.
type TokenAuthentication struct {
	Token APIKeyInfo
}

func (TokenAuthentication) isAuthenticationMethod() {}

func (TokenAuthentication) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"token"}`), nil
}

func (TokenAuthentication) AllowsUserScope(scope UserScope) error {
	return nil
}

func (TokenAuthentication) AllowsNamespaceScope(namespace ValidNamespaceID, scope NamespaceScope) error {
	return nil
}

// ImpersonateHeader is the name of the header field used to impersonate a user.
const ImpersonateHeader = "X-Impersonate-User"
