package api

//spellchecker:words encoding json
import (
	"encoding/json/v2"
	"fmt"
)

func (u *ValidUserInfo) Authenticate(method AuthenticationMethod) AuthenticatedUser {
	return AuthenticatedUser{
		username:  u.Username,
		superuser: u.Superuser,
		password:  u.Password,
		method:    method,
	}
}

// AuthenticatedUser represents an authenticated user.
//
// It holds information like a [ValidUserInfo], but also includes the authentication method used.
type AuthenticatedUser struct {
	username  ValidUsername
	superuser bool
	password  bool
	method    AuthenticationMethod
}

func (u *AuthenticatedUser) Method() AuthenticationMethod {
	return u.method
}

func (u *AuthenticatedUser) Username() ValidUsername {
	return u.username
}

func (u *AuthenticatedUser) Superuser() bool {
	return u.superuser
}

// CompleteInfo returns the complete [UserInfo] for this authenticated user.
//
// This info does not reflect any limitations in this authenticated user info, and retains full permissions.
// It MUST NOT be used for authorization decisions.
func (u *AuthenticatedUser) PlainInfo() UserInfo {
	return UserInfo{
		Username:  u.username.String(),
		Superuser: u.superuser,
		Password:  u.password,
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
