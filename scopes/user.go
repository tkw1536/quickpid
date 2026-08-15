//spellchecker:words scopes
package scopes

import "fmt"

type userActionDefinition struct {
	Scope       UserScope
	Description string

	// AnonymousMode indicates if the given action is available in anonymous mode.
	AnonymousMode bool

	// RequireAuthentication indicates if the given action is available to unauthenticated users.
	AllowUnauthenticated bool

	// RequireSuperuser indicates if the given action is available to superuser users.
	RequireSuperuser bool
}

// getUserActionDefinition returns the action with the given scope.
// If the action does not exist, it panics.
func getUserActionDefinition(scope UserScope) userActionDefinition {
	action, ok := userActions[scope]
	if !ok {
		panic(fmt.Sprintf("action %q not found", scope))
	}
	return action
}

// UserScope names a user action.
type UserScope string

// Scopes not related to a specific namespace.
const (
	ScopeImpersonate       UserScope = "impersonate"
	ScopeSetPassword       UserScope = "set_password"
	ScopeCreateUser        UserScope = "create_user"
	ScopeDeleteUser        UserScope = "delete_user"
	ScopeListUsers         UserScope = "list_users"
	ScopeListOwnKeys       UserScope = "list_own_keys"
	ScopeIssueKey          UserScope = "issue_key"
	ScopeRevokeKey         UserScope = "revoke_key"
	ScopeUpdateUser        UserScope = "update_user"
	ScopeListUserRoles     UserScope = "list_user_roles"
	ScopeGetMount          UserScope = "get_mount"
	ScopeListMounts        UserScope = "list_mounts"
	ScopeSetMount          UserScope = "set_mount"
	ScopeDeleteMount       UserScope = "delete_mount"
	ScopeListNamespaces    UserScope = "list_namespaces"
	ScopeCountAllResources UserScope = "count_all_resources"
	ScopeCreateNamespace   UserScope = "create_namespace"
)

var userActions = func(actions ...userActionDefinition) map[UserScope]userActionDefinition {
	m := make(map[UserScope]userActionDefinition, len(actions))
	for _, action := range actions {
		m[action.Scope] = action
	}
	return m
}(
	userActionDefinition{
		Scope:                ScopeImpersonate,
		Description:          "Impersonate another user",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	userActionDefinition{
		Scope:                ScopeSetPassword,
		Description:          "Set or clear own password",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
	userActionDefinition{
		Scope:                ScopeCreateUser,
		Description:          "Create a new user",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	userActionDefinition{
		Scope:                ScopeDeleteUser,
		Description:          "Delete a user",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	userActionDefinition{
		Scope:                ScopeListUsers,
		Description:          "List all users",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	userActionDefinition{
		Scope:                ScopeListOwnKeys,
		Description:          "List own API keys",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
	userActionDefinition{
		Scope:                ScopeIssueKey,
		Description:          "Issue a new API key for oneself",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
	userActionDefinition{
		Scope:                ScopeRevokeKey,
		Description:          "Revoke an API key for oneself",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
	userActionDefinition{
		Scope:                ScopeUpdateUser,
		Description:          "Update another user",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	userActionDefinition{
		Scope:                ScopeListUserRoles,
		Description:          "List own roles across all namespaces",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
	userActionDefinition{
		Scope:                ScopeGetMount,
		Description:          "Resolve a mount by base URI",
		AnonymousMode:        true,
		AllowUnauthenticated: true,
		RequireSuperuser:     false,
	},
	userActionDefinition{
		Scope:                ScopeListMounts,
		Description:          "List all mounts",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	userActionDefinition{
		Scope:                ScopeSetMount,
		Description:          "Set a mount by base URI",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	userActionDefinition{
		Scope:                ScopeDeleteMount,
		Description:          "Delete a mount by base URI",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     true,
	},
	userActionDefinition{
		Scope:                ScopeListNamespaces,
		Description:          "List namespaces",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
	userActionDefinition{
		Scope:                ScopeCountAllResources,
		Description:          "Count all resources across all namespaces",
		AnonymousMode:        true,
		AllowUnauthenticated: true,
		RequireSuperuser:     false,
	},
	userActionDefinition{
		Scope:                ScopeCreateNamespace,
		Description:          "Create a namespace",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		RequireSuperuser:     false,
	},
)
