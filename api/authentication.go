package api

//spellchecker:words github bicpid internal strict
import (
	"fmt"

	"github.com/tkw1536/bicpid/internal/strict"
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
		return fmt.Errorf("missing required field: username")
	}
	r.Username = string(decoded.Username.Value)
	if decoded.Superuser.Present {
		r.Superuser = bool(decoded.Superuser.Value)
	}
	return nil
}

// UserInfo is returned for user operations.
type UserInfo struct {
	Username  string `json:"username"`
	Superuser bool   `json:"superuser"`
}

// UserUpdateRequest updates fields on an existing user account.
//
// A nil pointer indicates that no update should be performed on that field.
// When username is omitted, the authenticated user account is updated.
type UserUpdateRequest struct {
	Username  *string `json:"username"`
	Superuser *bool   `json:"superuser"`
}

func (r *UserUpdateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Username  strict.Optional[strict.String] `json:"username"`
		Superuser strict.Optional[strict.Bool]   `json:"superuser"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	r.Username = strict.OptionalStringToPointer(decoded.Username)
	r.Superuser = strict.OptionalBoolToPointer(decoded.Superuser)
	return nil
}

type ListUsersParams struct {
	Limit  int
	Offset int
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
// When username is omitted, a key is issued for the authenticated user.
// A superuser may set username to issue a key for another account.
type KeyIssueRequest struct {
	Comment   string  `json:"comment"`
	ExpiresAt *string `json:"expires_at"`
	Username  *string `json:"username"`
}

func (r *KeyIssueRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Comment   strict.Optional[strict.String] `json:"comment"`
		ExpiresAt strict.Optional[*string]       `json:"expires_at"`
		Username  strict.Optional[strict.String] `json:"username"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if !decoded.Comment.Present {
		return fmt.Errorf("missing required field: comment")
	}
	r.Comment = string(decoded.Comment.Value)
	if decoded.ExpiresAt.Present {
		r.ExpiresAt = decoded.ExpiresAt.Value
	}
	r.Username = strict.OptionalStringToPointer(decoded.Username)
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
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if !decoded.ID.Present {
		return fmt.Errorf("missing required field: id")
	}
	r.ID = string(decoded.ID.Value)
	return nil
}

// RevokeKeyResponse is returned when an API key is revoked.
type RevokeKeyResponse struct {
	APIKeyInfo
}

// KeyUpdateRequest updates metadata for an existing API key.
//
// A nil pointer indicates that no update should be performed on that field.
type KeyUpdateRequest struct {
	Comment   *string  `json:"comment"`
	ExpiresAt **string `json:"expires_at"`
}

func (r *KeyUpdateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Comment   strict.Optional[strict.String] `json:"comment"`
		ExpiresAt strict.Optional[*string]       `json:"expires_at"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	r.Comment = strict.OptionalStringToPointer(decoded.Comment)
	r.ExpiresAt = decoded.ExpiresAt.ToPointer()
	return nil
}

// APIKeyInfo describes an API key without the secret value.
type APIKeyInfo struct {
	ID        string  `json:"id"`
	Comment   string  `json:"comment"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt *string `json:"expires_at"`
}
