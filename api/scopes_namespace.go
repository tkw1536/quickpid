package api

//spellchecker:words errors
import "errors"

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

type namespaceActionDefinition struct {
	Scope       NamespaceScope
	Description string

	// AnonymousMode indicates if the given action is available in anonymous mode.
	AnonymousMode bool

	// AllowUnauthenticated indicates if the given action is available to unauthenticated users.
	AllowUnauthenticated bool

	// MinRole is the minimum explicit namespace role that grants access.
	// [RoleNone] means no role grants access by itself.
	MinRole Role

	// RequireSuperuser indicates if only a superuser may perform the action.
	RequireSuperuser bool
}

var namespaceActions = func(actions ...namespaceActionDefinition) map[NamespaceScope]namespaceActionDefinition {
	m := make(map[NamespaceScope]namespaceActionDefinition, len(actions))
	for _, action := range actions {
		m[action.Scope] = action
	}
	return m
}(
	namespaceActionDefinition{
		Scope:                ScopeReadMetadata,
		Description:          "Read namespace metadata and have it appear in the list of namespaces",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleContributor,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeListNamespaceMounts,
		Description:          "List mounts for a namespace",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleContributor,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopePatchMetadata,
		Description:          "Patch namespace metadata",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeListResources,
		Description:          "List resources in a namespace",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleEditor,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeCreateResource,
		Description:          "Create a resource in a namespace",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleContributor,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeGetResource,
		Description:          "Get a resource by namespace and pid",
		AnonymousMode:        true,
		AllowUnauthenticated: true,
		MinRole:              RoleNone,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeSeeDeletedResource,
		Description:          "See an un-redacted deleted resource",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleEditor,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeUpdateResource,
		Description:          "Update a resource by namespace and pid",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleEditor,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeGetOwnNamespaceRole,
		Description:          "Get own role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleNone,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeGetOtherNamespaceRole,
		Description:          "Get another user's role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeListNamespaceRoles,
		Description:          "List all roles in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeSetOtherNamespaceRole,
		Description:          "Set another user's role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeSetOwnNamespaceRole,
		Description:          "Set own role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleNone,
		RequireSuperuser:     true,
	},
	namespaceActionDefinition{
		Scope:                ScopeClearOtherNamespaceRole,
		Description:          "Clear another user's role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},
	namespaceActionDefinition{
		Scope:                ScopeClearOwnNamespaceRole,
		Description:          "Clear own role in a namespace",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleNone,
		RequireSuperuser:     true,
	},
)

var errUnknownNamespaceScope error = errors.New("unknown namespace scope")

// getNamespaceScopeDefinition returns the namespace action with the given scope.
// If the action does not exist, it panics.
func getNamespaceScopeDefinition(name NamespaceScope) (namespaceActionDefinition, error) {
	action, ok := namespaceActions[name]
	if !ok {
		return namespaceActionDefinition{}, errUnknownNamespaceScope
	}
	return action, nil
}

// roleAtLeast reports whether have is at least as privileged as want.
// [RoleNone] never meets any minimum.
func roleAtLeast(have, want Role) bool {
	if want == RoleNone {
		return false
	}
	return roleRank(have) >= roleRank(want)
}

func roleRank(role Role) int {
	switch role {
	case RoleContributor:
		return 1
	case RoleEditor:
		return 2
	case RoleManager:
		return 3
	case RoleNone:
		return 0
	}
	panic("never reached")
}
