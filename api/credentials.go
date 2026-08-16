package api

//spellchecker:words errors time github quickpid internal strict
import (
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

// KeyIssueRequest is the JSON body for issueKey.
type KeyIssueRequest struct {
	Comment   string     `json:"comment"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

func (r *KeyIssueRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Comment   strict.Optional[strict.String] `json:"comment"`
		ExpiresAt strict.Optional[*strict.Time]  `json:"expiresAt"`
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
}

// Valid checks if this key is valid for the given time.
//
// The time function is only called when ExpiresAt is not nil and not the zero time.
func (k *APIKeyInfo) Valid(now func() time.Time) bool {
	return k.ExpiresAt == nil || (!k.ExpiresAt.IsZero() && k.ExpiresAt.After(now()))
}
