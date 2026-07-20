// Package api holds type definitions for the PID Resolver API.
package api

//spellchecker:words encoding json github quickpid internal strict embed
import (
	"encoding/json"
	"fmt"

	"github.com/tkw1536/quickpid/internal/strict"
	"github.com/tkw1536/quickpid/pid"

	_ "embed"
)

// NamespaceCreateRequest is the JSON body for createNamespace.
type NamespaceCreateRequest struct {
	Tag       string     `json:"tag"`
	PIDFormat pid.Format `json:"pid_format"`
}

func (r *NamespaceCreateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Tag       strict.Optional[strict.String] `json:"tag"`
		PIDFormat strict.Optional[pid.Format]    `json:"pid_format"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.Tag.Present {
		return fmt.Errorf("%w: tag", errMissingRequiredField)
	}
	r.Tag = string(decoded.Tag.Value)
	if !decoded.PIDFormat.Present {
		return fmt.Errorf("%w: pid_format", errMissingRequiredField)
	}
	r.PIDFormat = decoded.PIDFormat.Value
	if err := r.PIDFormat.Validate(); err != nil {
		return fmt.Errorf("%w: %w", errInvalidPIDFormat, err)
	}

	return nil
}

// NamespaceResponse is returned for namespace operations.
type NamespaceResponse struct {
	ID          string     `json:"id"`
	Tag         string     `json:"tag"`
	PIDFormat   pid.Format `json:"pid_format"`
	DateCreated string     `json:"date_created"`
}

type PaginatedNamespacesResponse struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`

	Items []NamespaceResponse `json:"items"`
}

// ResourceCreateRequest is the JSON body for createResource and batchCreateResources items.
type ResourceCreateRequest struct {
	URL      string
	Metadata *string
	Tags     []string
}

func (r *ResourceCreateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		URL      strict.Optional[strict.String]      `json:"url"`
		Metadata strict.Optional[*string]            `json:"metadata"`
		Tag      strict.Optional[strict.String]      `json:"tag"`
		Tags     strict.Optional[strict.StringSlice] `json:"tags"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}

	if !decoded.URL.Present {
		return fmt.Errorf("%w: url", errMissingRequiredField)
	}
	r.URL = string(decoded.URL.Value)

	if !decoded.Metadata.Present {
		return fmt.Errorf("%w: metadata", errMissingRequiredField)
	}
	r.Metadata = decoded.Metadata.Value

	tags, _, err := unmarshalResourceTags(decoded.Tag, decoded.Tags, true)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	r.Tags = tags

	return nil
}

// ResourceResponse is returned for resource operations.
type ResourceResponse struct {
	PID         string
	URL         string
	Metadata    *string
	DateCreated string
	DateUpdated string
	Tags        []string
	Deleted     bool
}

func (r ResourceResponse) MarshalJSON() ([]byte, error) {
	type out struct {
		PID         string   `json:"pid"`
		URL         string   `json:"url"`
		Metadata    *string  `json:"metadata"`
		DateCreated string   `json:"date_created"`
		DateUpdated string   `json:"date_updated"`
		Tag         *string  `json:"tag,omitempty"`
		Tags        []string `json:"tags,omitempty"`
		Deleted     bool     `json:"deleted"`
	}
	encoded := out{
		PID:         r.PID,
		URL:         r.URL,
		Metadata:    r.Metadata,
		DateCreated: r.DateCreated,
		DateUpdated: r.DateUpdated,
		Deleted:     r.Deleted,
	}
	switch len(r.Tags) {
	case 0:
		return nil, fmt.Errorf("%w: empty tags", errInvalidResourceTags)
	case 1:
		encoded.Tag = &r.Tags[0]
	default:
		encoded.Tags = r.Tags
	}
	bytes, err := json.Marshal(encoded)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}
	return bytes, nil
}

type PaginatedResourcesResponse struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`

	Items []ResourceResponse `json:"items"`
}

// ResourceCountResponse is returned for countAllResources.
type ResourceCountResponse struct {
	Total int `json:"total"`
}

// ResourceUpdateRequest is the JSON body for updateResource.
//
// A nil pointer (or nil Tags slice) indicates that no update should be performed on that field.
// When Tags is non-nil, it always has length >= 1.
type ResourceUpdateRequest struct {
	URL      *string
	Metadata **string
	Tags     []string
	Deleted  *bool
}

func (r *ResourceUpdateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		URL      strict.Optional[strict.String]      `json:"url"`
		Metadata strict.Optional[*string]            `json:"metadata"`
		Tag      strict.Optional[strict.String]      `json:"tag"`
		Tags     strict.Optional[strict.StringSlice] `json:"tags"`
		Deleted  strict.Optional[strict.Bool]        `json:"deleted"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	r.URL = strict.OptionalStringToPointer(decoded.URL)
	r.Metadata = decoded.Metadata.ToPointer()
	r.Deleted = strict.OptionalBoolToPointer(decoded.Deleted)

	tags, present, err := unmarshalResourceTags(decoded.Tag, decoded.Tags, false)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if present {
		r.Tags = tags
	} else {
		r.Tags = nil
	}

	return nil
}

// ListResourcesParams carries path and query parameters for listResources.
type ListResourcesParams struct {
	Tag     *string // optionally filter by tag membership
	Deleted *bool   // optionally filter by deletion status

	Limit  int
	Offset int
}

type ListNamespacesParams struct {
	Tag *string // optionally filter by tag

	Limit  int
	Offset int
}

// InfoResponse provides information about the resolver.
type InfoResponse struct {
	MaxBodyBytes         int64 `json:"max_body_bytes"`
	DefaultPageLimit     int64 `json:"default_page_limit"`
	MaxPageLimit         int64 `json:"max_page_limit"`
	MaxBatchItems        int64 `json:"max_batch_items"`
	MaxAutocompleteUsers int64 `json:"max_autocomplete_users,omitzero"`
	Authentication       bool  `json:"authentication,omitzero"`
}

// unmarshalResourceTags decodes exclusive tag / tags fields into a non-empty []string.
//
// If required is true, exactly one of tag or tags must be present.
// If required is false and neither is present, ok is false and tags is nil.
func unmarshalResourceTags(tag strict.Optional[strict.String], tagsField strict.Optional[strict.StringSlice], required bool) (tags []string, ok bool, err error) {
	if tag.Present && tagsField.Present {
		return nil, false, fmt.Errorf("%w: %w", errInvalidResourceTags, errTagAndTagsMutuallyExclusive)
	}
	if tag.Present {
		return []string{string(tag.Value)}, true, nil
	}
	if tagsField.Present {
		if len(tagsField.Value) < 2 {
			return nil, false, fmt.Errorf("%w: %w", errInvalidResourceTags, errTagsMustHaveAtLeastTwo)
		}
		return tagsField.Value.Strings(), true, nil
	}
	if required {
		return nil, false, errMissingTagOrTags
	}
	return nil, false, nil
}
