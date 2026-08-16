package api

//spellchecker:words errors slices strings sync
import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// UserScope describes a permission that can be granted to a user.
// Each scope typically corresponds to the appropriate operation id in the spec.
type UserScope string

var errUnknownUserScope error = errors.New("unknown user scope")

// Definition returns the definition of the given scope.
// If the scope is not defined, it returns an error.
func (scope UserScope) Definition() (UserScopeDefinition, error) {
	def, ok := userDefs[scope]
	if !ok {
		return UserScopeDefinition{}, errUnknownUserScope
	}
	return def, nil
}

// UserScopeDefinition defines a [UserScope] and its associated permissions.
type UserScopeDefinition struct {
	Scope       UserScope `json:"scope"`
	Description string    `json:"description"`

	// AnonymousMode indicates if the given action is available in anonymous mode.
	AnonymousMode bool `json:"anonymousMode"`

	// RequireAuthentication indicates if the given action is available to unauthenticated users.
	AllowUnauthenticated bool `json:"allowUnauthenticated"`

	// RequireSuperuser indicates if the given action is available to superuser users.
	RequireSuperuser bool `json:"requireSuperuser"`
}

// UserScope names.
//
// They (and their definitions below) should be kept in the same order as in the spec.
// Related operations (those using the same path) should be grouped together, with a blank line between them.
// Their formatting is kept similar to the [server.NewServer] route definitions.
const (
	ScopeListNamespaces  UserScope = "listNamespaces"
	ScopeCreateNamespace UserScope = "createNamespace"

	ScopeCountAllResources UserScope = "countAllResources"

	ScopeListMounts UserScope = "listMounts"

	ScopeResolveMountByBaseUri UserScope = "resolveMountResource"
	ScopeSetMount              UserScope = "setMount"
	ScopeDeleteMount           UserScope = "deleteMount"

	ScopeGetMount UserScope = "getMount"

	ScopeGetUserInfo UserScope = "getUserInfo"
	ScopeUpdateUser  UserScope = "updateUser"
	ScopeCreateUser  UserScope = "createUser"
	ScopeDeleteUser  UserScope = "deleteUser"

	ScopeAutocompleteUsers UserScope = "autocompleteUsers"

	ScopeListUsers UserScope = "listUsers"

	ScopeGetUserRoles UserScope = "listUserRoles"

	ScopeListUserKeys UserScope = "listOwnKeys"
	ScopeIssueUserKey UserScope = "issueKey"

	ScopeSetUserPassword UserScope = "setPassword"

	ScopeRevokeUserKey UserScope = "revokeKey"

	// The impersonate scope is special because it does not correspond to a specific operation in the spec.
	// But can be used on any route to impersonate another user.
	// When impersonating another user, the call implicitly inherits all scopes of the impersonated user.
	ScopeImpersonate UserScope = "impersonate"
)

// user definitions keyed by their scope.
var userDefs = func(actions ...UserScopeDefinition) map[UserScope]UserScopeDefinition {
	m := make(map[UserScope]UserScopeDefinition, len(actions))
	for _, action := range actions {
		if _, ok := m[action.Scope]; ok {
			panic(fmt.Sprintf("duplicate scope definition: %s", action.Scope))
		}
		m[action.Scope] = action
	}
	return m
}(
	UserScopeDefinition{
		Scope:                ScopeListNamespaces,
		Description:          "List namespaces available on the resolver.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
	UserScopeDefinition{
		Scope:                ScopeCreateNamespace,
		Description:          "Creates a new namespace.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},

	UserScopeDefinition{
		Scope:                ScopeCountAllResources,
		Description:          "Return the total number of resources (issued PIDs) stored across all namespaces.",
		AnonymousMode:        true,
		AllowUnauthenticated: true,
		RequireSuperuser:     false,
	},

	UserScopeDefinition{
		Scope:                ScopeListMounts,
		Description:          "List all namespace mounts.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},

	UserScopeDefinition{
		Scope:                ScopeResolveMountByBaseUri,
		Description:          "Resolve a namespace mount by base URI.",
		AnonymousMode:        true,
		AllowUnauthenticated: true,
		RequireSuperuser:     false,
	},
	UserScopeDefinition{
		Scope:                ScopeSetMount,
		Description:          "Create or replace a namespace mount.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	UserScopeDefinition{
		Scope:                ScopeDeleteMount,
		Description:          "Delete a namespace mount.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},

	UserScopeDefinition{
		Scope:                ScopeGetMount,
		Description:          "Resolve a resource using a mounted base URI and PID.",
		AnonymousMode:        true,
		AllowUnauthenticated: true,
		RequireSuperuser:     false,
	},

	UserScopeDefinition{
		Scope:                ScopeGetUserInfo,
		Description:          "Return information about a user account.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
	UserScopeDefinition{
		Scope:                ScopeUpdateUser,
		Description:          "Update a user account.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	UserScopeDefinition{
		Scope:                ScopeCreateUser,
		Description:          "Create a new user account.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	UserScopeDefinition{
		Scope:                ScopeDeleteUser,
		Description:          "Delete a user account.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},

	UserScopeDefinition{
		Scope:                ScopeAutocompleteUsers,
		Description:          "Autocomplete usernames by prefix.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},

	UserScopeDefinition{
		Scope:                ScopeListUsers,
		Description:          "List all user accounts.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},

	UserScopeDefinition{
		Scope:                ScopeGetUserRoles,
		Description:          "List explicit namespace roles for a user.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},

	UserScopeDefinition{
		Scope:                ScopeListUserKeys,
		Description:          "List API keys for a user account.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
	UserScopeDefinition{
		Scope:                ScopeIssueUserKey,
		Description:          "Issue a new API key.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},

	UserScopeDefinition{
		Scope:                ScopeSetUserPassword,
		Description:          "Set or clear a user password.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},

	UserScopeDefinition{
		Scope:                ScopeRevokeUserKey,
		Description:          "Revoke an API key for a user account.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},

	UserScopeDefinition{
		Scope:                ScopeImpersonate,
		Description:          "Impersonate another user.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
)

// GetUserScopes returns an array of all defined scopes.
// The scopes are sorted alphabetically.
//
// The returned slice is safe for manipulation.
func GetUserScopes() []UserScope {
	return slices.Clone(scopesInternal())
}

var scopesInternal = sync.OnceValue(func() []UserScope {
	scopes := make([]UserScope, 0, len(userDefs))
	for _, def := range userDefs {
		scopes = append(scopes, def.Scope)
	}
	slices.Sort(scopes)
	return scopes
})

// GetUserScopeDefinitions returns all user scope definitions sorted alphabetically by scope.
//
// The returned slice is safe for manipulation.
func GetUserScopeDefinitions() []UserScopeDefinition {
	return slices.Clone(userScopeDefinitionsInternal())
}

var userScopeDefinitionsInternal = sync.OnceValue(func() []UserScopeDefinition {
	defs := make([]UserScopeDefinition, 0, len(userDefs))
	for _, def := range userDefs {
		defs = append(defs, def)
	}
	slices.SortFunc(defs, func(a, b UserScopeDefinition) int {
		return strings.Compare(string(a.Scope), string(b.Scope))
	})
	return defs
})
