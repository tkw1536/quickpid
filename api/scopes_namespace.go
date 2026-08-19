package api

//spellchecker:words errors slices strings sync
import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// NamespaceScope describes a permission that can be granted to a user for a specific namespace.
// Each scope typically corresponds to the appropriate operation id in the spec.
type NamespaceScope string

var errUnknownNamespaceScope error = errors.New("unknown namespace scope")

// Definition returns the definition of the given scope.
// If the scope is not defined, it returns an error.
func (scope NamespaceScope) Definition() (NamespaceScopeDefinition, error) {
	action, ok := namespaceDefs[scope]
	if !ok {
		return NamespaceScopeDefinition{}, errUnknownNamespaceScope
	}
	return action, nil
}

// NamespaceScope names.
//
// They (and their definitions below) should be kept in the same order as in the spec.
// Related operations (those using the same path) should be grouped together, with a blank line between them.
// Their formatting is kept similar to the [server.NewServer] route definitions.
const (
	ScopeGetNamespace    NamespaceScope = "getNamespace"
	ScopeUpdateNamespace NamespaceScope = "updateNamespace"

	ScopeListNamespaceMounts NamespaceScope = "listNamespaceMounts"

	ScopeListNamespaceRoles NamespaceScope = "listNamespaceRoles"

	ScopeGetNamespaceRoleSelf     NamespaceScope = "getNamespaceRoleSelf"
	ScopeGetNamespaceRoleOther    NamespaceScope = "getNamespaceRoleOther"
	ScopeSetNamespaceRoleSelf     NamespaceScope = "setNamespaceRoleSelf"
	ScopeSetNamespaceRoleOther    NamespaceScope = "setNamespaceRoleOther"
	ScopeRemoveNamespaceRoleSelf  NamespaceScope = "removeNamespaceRoleSelf"
	ScopeRemoveNamespaceRoleOther NamespaceScope = "removeNamespaceRoleOther"

	ScopeListResources       NamespaceScope = "listResources"
	ScopeCreateResource      NamespaceScope = "createResource"
	ScopeCreateResourceBatch NamespaceScope = "createResourceBatch"

	ScopeGetResource    NamespaceScope = "getResource"
	ScopeUpdateResource NamespaceScope = "updateResource"

	// SeeDeletedResource is special because it does not correspond to a dedicated operation in the spec.
	// It gates whether a deleted resource is returned un-redacted when reading a resource.
	ScopeSeeDeletedResource NamespaceScope = "seeDeletedResource"

	// ScopeSeeInNamespaceList is special because it does not correspond to a dedicated operation in the spec.
	// Instead it gates whether a namespace is returned in the list of namespaces.
	ScopeSeeInNamespaceList NamespaceScope = "seeInNamespaceList"
)

// NamespaceScopeDefinition defines a [NamespaceScope] and its associated permissions.
type NamespaceScopeDefinition struct {
	Scope       NamespaceScope `json:"scope"`
	Description string         `json:"description"`

	// AnonymousMode indicates if the given action is available in anonymous mode.
	AnonymousMode bool `json:"anonymousMode"`

	// AllowUnauthenticated indicates if the given action is available to unauthenticated users.
	AllowUnauthenticated bool `json:"allowUnauthenticated"`

	// MinRole is the minimum explicit namespace role that grants access.
	// [RoleNone] means no role grants access by itself.
	MinRole Role `json:"minRole"`

	// RequireSuperuser indicates if only a superuser may perform the action.
	RequireSuperuser bool `json:"requireSuperuser"`
}

var namespaceDefs = func(actions ...NamespaceScopeDefinition) map[NamespaceScope]NamespaceScopeDefinition {
	m := make(map[NamespaceScope]NamespaceScopeDefinition, len(actions))
	for _, action := range actions {
		if _, ok := m[action.Scope]; ok {
			panic(fmt.Sprintf("duplicate scope definition: %s", action.Scope))
		}
		m[action.Scope] = action
	}
	return m
}(
	NamespaceScopeDefinition{
		Scope:                ScopeGetNamespace,
		Description:          "Retrieve a single namespace by id.",
		AnonymousMode:        true,
		AllowUnauthenticated: true,
		MinRole:              RoleNone,
		RequireSuperuser:     false,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeUpdateNamespace,
		Description:          "Update a namespace by id.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},

	NamespaceScopeDefinition{
		Scope:                ScopeListNamespaceMounts,
		Description:          "List mounts for a namespace.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleContributor,
		RequireSuperuser:     false,
	},

	NamespaceScopeDefinition{
		Scope:                ScopeListNamespaceRoles,
		Description:          "List explicit namespace roles.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},

	NamespaceScopeDefinition{
		Scope:                ScopeGetNamespaceRoleSelf,
		Description:          "Get own role in a namespace.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleNone,
		RequireSuperuser:     false,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeGetNamespaceRoleOther,
		Description:          "Get another user's role in a namespace.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeSetNamespaceRoleSelf,
		Description:          "Set own role in a namespace.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleNone,
		RequireSuperuser:     true,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeSetNamespaceRoleOther,
		Description:          "Set another user's role in a namespace.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeRemoveNamespaceRoleSelf,
		Description:          "Remove own explicit role in a namespace.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleNone,
		RequireSuperuser:     true,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeRemoveNamespaceRoleOther,
		Description:          "Remove another user's explicit role in a namespace.",
		AnonymousMode:        false,
		AllowUnauthenticated: false,
		MinRole:              RoleManager,
		RequireSuperuser:     false,
	},

	NamespaceScopeDefinition{
		Scope:                ScopeListResources,
		Description:          "List resources within a namespace.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleEditor,
		RequireSuperuser:     false,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeCreateResource,
		Description:          "Create a resource within a namespace.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleContributor,
		RequireSuperuser:     false,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeCreateResourceBatch,
		Description:          "Create several resources within a namespace.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleContributor,
		RequireSuperuser:     false,
	},

	NamespaceScopeDefinition{
		Scope:                ScopeGetResource,
		Description:          "Retrieve a resource by PID within a namespace.",
		AnonymousMode:        true,
		AllowUnauthenticated: true,
		MinRole:              RoleNone,
		RequireSuperuser:     false,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeUpdateResource,
		Description:          "Update a resource by PID within a namespace.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleEditor,
		RequireSuperuser:     false,
	},

	NamespaceScopeDefinition{
		Scope:                ScopeSeeDeletedResource,
		Description:          "See an un-redacted deleted resource.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleEditor,
		RequireSuperuser:     false,
	},
	NamespaceScopeDefinition{
		Scope:                ScopeSeeInNamespaceList,
		Description:          "See a namespace in the list of namespaces.",
		AnonymousMode:        true,
		AllowUnauthenticated: false,
		MinRole:              RoleContributor,
		RequireSuperuser:     false,
	},
)

// GetNamespaceScopes returns an array of all defined namespace scopes.
// The scopes are sorted alphabetically.
//
// The returned slice is safe for manipulation.
func GetNamespaceScopes() []NamespaceScope {
	return slices.Clone(namespaceScopesInternal())
}

var namespaceScopesInternal = sync.OnceValue(func() []NamespaceScope {
	scopes := make([]NamespaceScope, 0, len(namespaceDefs))
	for _, def := range namespaceDefs {
		scopes = append(scopes, def.Scope)
	}
	slices.Sort(scopes)
	return scopes
})

// GetNamespaceScopeDefinitions returns all namespace scope definitions sorted alphabetically by scope.
//
// The returned slice is safe for manipulation.
func GetNamespaceScopeDefinitions() []NamespaceScopeDefinition {
	return slices.Clone(namespaceScopeDefinitionsInternal())
}

var namespaceScopeDefinitionsInternal = sync.OnceValue(func() []NamespaceScopeDefinition {
	defs := make([]NamespaceScopeDefinition, 0, len(namespaceDefs))
	for _, def := range namespaceDefs {
		defs = append(defs, def)
	}
	slices.SortFunc(defs, func(a, b NamespaceScopeDefinition) int {
		return strings.Compare(string(a.Scope), string(b.Scope))
	})
	return defs
})
