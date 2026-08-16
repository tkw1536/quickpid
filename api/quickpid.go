// Package api holds type definitions for the PID Resolver API.
package api

// InfoResponse provides information about the resolver.
type InfoResponse struct {
	MaxBodyBytes         int64 `json:"maxBodyBytes"`
	DefaultPageLimit     int64 `json:"defaultPageLimit"`
	MaxPageLimit         int64 `json:"maxPageLimit"`
	MaxBatchItems        int64 `json:"maxBatchItems"`
	MaxAutocompleteUsers int64 `json:"maxAutocompleteUsers,omitzero"`
	Authentication       bool  `json:"authentication,omitzero"`
}

// MetaResponse provides optional public meta information about the server.
// All fields are optional; when present they must be non-empty.
type MetaResponse struct {
	About    string `json:"about,omitempty"`
	Comment  string `json:"comment,omitempty"`
	Software string `json:"software,omitempty"`
	Version  string `json:"version,omitempty"`
}

// ScopesResponse is the catalogue of user and namespace scope definitions.
// Both arrays are empty when the server runs in anonymous mode.
type ScopesResponse struct {
	User      []UserScopeDefinition      `json:"user"`
	Namespace []NamespaceScopeDefinition `json:"namespace"`
}
