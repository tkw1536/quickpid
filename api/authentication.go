package api

//spellchecker:words time github quickpid internal strict
import (
	"fmt"
	"time"

	"github.com/tkw1536/quickpid/internal/strict"
)

// UserCreateRequest is the JSON body for createUser.
type UserCreateRequest struct {
	Username  string `json:"username"`
	Superuser bool   `json:"superuser"`
}

func (r *UserCreateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Username  strict.Optional[strict.String] `json:"username"`
		Superuser strict.Optional[strict.Bool]   `json:"superuser"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if !decoded.Username.Present {
		return missingRequiredFieldError("username")
	}
	r.Username = string(decoded.Username.Value)
	if decoded.Superuser.Present {
		r.Superuser = bool(decoded.Superuser.Value)
	}
	return nil
}

// Validate checks if the given request is valid.
func (r *UserCreateRequest) Validate() (ValidUserCreateRequest, error) {
	username, err := NewUsername(r.Username)
	if err != nil {
		return ValidUserCreateRequest{}, err
	}
	return ValidUserCreateRequest{
		Username:  username,
		Superuser: r.Superuser,
	}, nil
}

// ValidUserCreateRequest is like a [UserCreateRequest] but with a validated username.
type ValidUserCreateRequest struct {
	Username  ValidUsername
	Superuser bool
}

// UserInfo is returned for user operations.
type UserInfo struct {
	Username  string `json:"username"`
	Superuser bool   `json:"superuser"`
	Password  bool   `json:"password"`
}

func (u *UserInfo) Validate() (ValidUserInfo, error) {
	username, err := NewUsername(u.Username)
	if err != nil {
		return ValidUserInfo{}, err
	}
	return ValidUserInfo{
		Username:  username,
		Superuser: u.Superuser,
		Password:  u.Password,
	}, nil
}

// ValidUserInfo is information about a user with a validated username.
type ValidUserInfo struct {
	Username  ValidUsername
	Superuser bool
	Password  bool
}

// UserUpdateRequest updates fields on an existing user account.
//
// A nil pointer indicates that no update should be performed on that field.
// The target account is selected with the username query parameter, not the request body.
type UserUpdateRequest struct {
	Superuser *bool `json:"superuser"`
}

func (r *UserUpdateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Superuser strict.Optional[strict.Bool] `json:"superuser"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	r.Superuser = strict.OptionalBoolToPointer(decoded.Superuser)
	return nil
}

// SetPasswordRequest is the JSON body for setUserPassword.
//
// The target account is selected with the username query parameter, not the request body.
type SetPasswordRequest struct {
	Password **string `json:"password"`
}

func (r *SetPasswordRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Password strict.Optional[*string] `json:"password"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if !decoded.Password.Present {
		return missingRequiredFieldError("password")
	}
	r.Password = decoded.Password.ToPointer()
	return nil
}

// Validate checks if the given request is valid.
func (r *SetPasswordRequest) Validate() (ValidSetPasswordRequest, error) {
	if r.Password == nil {
		return ValidSetPasswordRequest{}, missingRequiredFieldError("password")
	}
	if *r.Password == nil {
		return ValidSetPasswordRequest{}, nil
	}
	password, err := NewPassword(**r.Password)
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

type ListUsersParams struct {
	Superuser *bool
	Password  *bool
	Limit     int
	Offset    int
}

type PaginatedUsersResponse struct {
	Total  int        `json:"total"`
	Offset int        `json:"offset"`
	Items  []UserInfo `json:"items"`
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

// KeyIssueRequest is the JSON body for issueKey.
//
// The target account is selected with the username query parameter, not the request body.
type KeyIssueRequest struct {
	Comment   string     `json:"comment"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (r *KeyIssueRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Comment   strict.Optional[strict.String] `json:"comment"`
		ExpiresAt strict.Optional[*strict.Time]  `json:"expires_at"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.Comment.Present {
		return missingRequiredFieldError("comment")
	}
	r.Comment = string(decoded.Comment.Value)
	if !decoded.ExpiresAt.Present {
		return missingRequiredFieldError("expires_at")
	}
	if decoded.ExpiresAt.Value == nil {
		r.ExpiresAt = nil
	} else {
		ts := decoded.ExpiresAt.Value.Time()
		r.ExpiresAt = &ts
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
	decoded, err := strict.UnmarshalStruct[internal](data)
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
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt optionally holds the time at which this key will expire.
	// If it is nil, the key never expires.
	// If it is a pointer to the zero time, the key is revoked.
	ExpiresAt *time.Time `json:"expires_at"`
}

// Valid checks if this key is valid for the given time.
//
// The time function is only called when ExpiresAt is not nil and not the zero time.
func (k *APIKeyInfo) Valid(now func() time.Time) bool {
	return k.ExpiresAt == nil || (!k.ExpiresAt.IsZero() && k.ExpiresAt.After(now()))
}
