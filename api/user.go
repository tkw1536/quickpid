package api

//spellchecker:words encoding json
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
	if c == nil {
		return nil
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
// These are implemented by [Impersonation], [BasicAuthentication], or [TokenAuthentication].
type AuthenticationMethod interface {
	isAuthenticationMethod()
	json.Marshaler
}

var (
	_ AuthenticationMethod = Impersonation{}
	_ AuthenticationMethod = BasicAuthentication{}
	_ AuthenticationMethod = TokenAuthentication{}
)

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

// BasicAuthentication implies that a user is authenticated because they provided a username and password.
type BasicAuthentication struct{}

func (BasicAuthentication) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"basic"}`), nil
}

func (BasicAuthentication) isAuthenticationMethod() {}

// TokenAuthentication implies that a user is authenticated because they provided a token.
type TokenAuthentication struct {
	Token APIKeyInfo
}

func (TokenAuthentication) isAuthenticationMethod() {}

func (TokenAuthentication) MarshalJSON() ([]byte, error) {
	return []byte(`{"type":"token"}`), nil
}
