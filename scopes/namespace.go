//spellchecker:words scopes
package scopes

//spellchecker:words github quickpid
import (
	"fmt"

	"github.com/tkw1536/quickpid/api"
)

type NamespaceScope string

// Scopes related to a specific namespace.
const (
	ScopeReadMetadata            NamespaceScope = "read_metadata"
	ScopeListNamespaceMounts     NamespaceScope = "list_namespace_mounts"
	ScopePatchMetadata           NamespaceScope = "patch_metadata"
	ScopeListResources           NamespaceScope = "list_resources"
	ScopeCreateResource          NamespaceScope = "create_resource"
	ScopeGetResource             NamespaceScope = "get_resource"
	ScopeSeeDeletedResource      NamespaceScope = "see_deleted_resource"
	ScopeUpdateResource          NamespaceScope = "update_resource"
	ScopeGetOwnNamespaceRole     NamespaceScope = "get_own_namespace_role"
	ScopeGetOtherNamespaceRole   NamespaceScope = "get_other_namespace_role"
	ScopeListNamespaceRoles      NamespaceScope = "list_namespace_roles"
	ScopeSetOtherNamespaceRole   NamespaceScope = "set_other_namespace_role"
	ScopeSetOwnNamespaceRole     NamespaceScope = "set_own_namespace_role"
	ScopeClearOtherNamespaceRole NamespaceScope = "clear_other_namespace_role"
	ScopeClearOwnNamespaceRole   NamespaceScope = "clear_own_namespace_role"
)

type namespaceAction struct {
	Scope       NamespaceScope
	Description string

	// AnonymousMode indicates if the given action is available in anonymous mode.
	AnonymousMode bool

	// AllowUnauthenticated indicates if the given action is available to unauthenticated users.
	AllowUnauthenticated bool

	// MinRole is the minimum explicit namespace role that grants access.
	// [api.RoleNone] means no role grants access by itself.
	MinRole api.Role

	// RequireSuperuser indicates if only a superuser may perform the action.
	RequireSuperuser bool
}

var namespaceActions = func(actions ...namespaceAction) map[NamespaceScope]namespaceAction {
	m := make(map[NamespaceScope]namespaceAction, len(actions))
	for _, action := range actions {
		m[action.Scope] = action
	}
	return m
}(
	namespaceAction{
		Scope:                ScopeReadMetadata,
		Description:          "Read namespace metadata",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              api.RoleContributor,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeListNamespaceMounts,
		Description:          "List mounts for a namespace",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              api.RoleContributor,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopePatchMetadata,
		Description:          "Patch namespace metadata",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              api.RoleManager,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeListResources,
		Description:          "List resources in a namespace",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              api.RoleEditor,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeCreateResource,
		Description:          "Create a resource in a namespace",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              api.RoleContributor,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeGetResource,
		Description:          "Get a resource by namespace and pid",
		AnonymousMode:        true,
		AllowUnauthenticated: true,
		MinRole:              api.RoleNone,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeSeeDeletedResource,
		Description:          "See an un-redacted deleted resource",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              api.RoleEditor,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeUpdateResource,
		Description:          "Update a resource by namespace and pid",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              api.RoleEditor,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeGetOwnNamespaceRole,
		Description:          "Get own role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              api.RoleNone,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeGetOtherNamespaceRole,
		Description:          "Get another user's role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              api.RoleManager,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeListNamespaceRoles,
		Description:          "List all roles in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              api.RoleManager,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeSetOtherNamespaceRole,
		Description:          "Set another user's role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              api.RoleManager,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeSetOwnNamespaceRole,
		Description:          "Set own role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              api.RoleNone,
		RequireSuperuser:     true,
	},
	namespaceAction{
		Scope:                ScopeClearOtherNamespaceRole,
		Description:          "Clear another user's role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              api.RoleManager,
		RequireSuperuser:     false,
	},
	namespaceAction{
		Scope:                ScopeClearOwnNamespaceRole,
		Description:          "Clear own role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              api.RoleNone,
		RequireSuperuser:     true,
	},
)

// getNamespace returns the namespace action with the given scope.
// If the action does not exist, it panics.
func getNamespace(name NamespaceScope) namespaceAction {
	action, ok := namespaceActions[name]
	if !ok {
		panic(fmt.Sprintf("namespace action %q not found", name))
	}
	return action
}

// roleAtLeast reports whether have is at least as privileged as want.
// [api.RoleNone] never meets any minimum.
func roleAtLeast(have, want api.Role) bool {
	if want == api.RoleNone {
		return false
	}
	return roleRank(have) >= roleRank(want)
}

func roleRank(role api.Role) int {
	switch role {
	case api.RoleContributor:
		return 1
	case api.RoleEditor:
		return 2
	case api.RoleManager:
		return 3
	case api.RoleNone:
		return 0
	}
	panic("never reached")
}
