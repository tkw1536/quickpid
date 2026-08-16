package api

//spellchecker:words errors github quickpid internal strict regexp
import (
	"errors"
	"fmt"
	"regexp"

	"github.com/tkw1536/quickpid/internal/strict"
)

type ValidUsername struct {
	valid bool
	value string
}

func (username ValidUsername) String() string {
	if !username.valid {
		panic("invalid username")
	}
	return username.value
}

var (
	usernameRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	errInvalidUsername = errors.New("invalid username")
)

// NewUsername creates a new Username.
func NewUsername(value string) (ValidUsername, error) {
	if !usernameRE.MatchString(value) {
		return ValidUsername{}, errInvalidUsername
	}
	return ValidUsername{valid: true, value: value}, nil
}

var (
	autocompleteQueryRE = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

var errInvalidAutocompleteQuery = errors.New("invalid autocomplete query")

func NewAutocompleteQuery(value string) (ValidAutocompleteQuery, error) {
	if !autocompleteQueryRE.MatchString(value) {
		return ValidAutocompleteQuery{}, errInvalidAutocompleteQuery
	}
	return ValidAutocompleteQuery{valid: true, value: value}, nil
}

// ValidAutocompleteQuery represents a valid query for autocomplete.
//
// Use [NewAutocompleteQuery] to create a new autocomplete query.
type ValidAutocompleteQuery struct {
	valid bool
	value string
}

func (query ValidAutocompleteQuery) String() string {
	if !query.valid {
		panic("invalid autocomplete query")
	}
	return query.value
}

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
	decoded, err := strict.UnmarshalStrict[internal](data)
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
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	r.Superuser = strict.OptionalBoolToPointer(decoded.Superuser)
	return nil
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
