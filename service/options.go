// Package service provides high-level PID resolver and auth operations.
//
//spellchecker:words service
package service

// Options represents operational configuration for the service.
type Options struct {
	// InfoEnabled enables the generic info endpoint.
	InfoEnabled bool

	// Anonymous disables user management and authorization checks for resolver operations.
	Anonymous bool

	// Limits for various internal behavior.
	Limits Limits
}

// withValidValues returns a copy of the Options where each field is set to a valid value.
func (o Options) withValidValues() Options {
	o.Limits = o.Limits.WithValidValues()
	return o
}

// Limits represent limits for the service.
type Limits struct {
	MaxBodyBytes int64 // maximum size of request body, 0 or negative means no limit.

	DefaultPageLimit int // default number of items per page, must be at least 1.
	MaxPageLimit     int // maximum number of items per page, 0 or negative means no limit.

	MaxBatchItems int // maximum number of items in a batch, 0 or negative means no limit.

	MaxAutocompleteUsers int // maximum number of usernames returned by autocomplete, must be at least 1.

	MaxNamespaceIDAttempts int // maximum number of attempts to allocate a namespace ID, must be at least 1.
	MaxPIDAttempts         int // maximum number of attempts to allocate a PID, must be at least 1.
	MaxAPIKeyAttempts      int // maximum number of attempts to issue an API key, must be at least 1.
}

// InternalPageLimit returns a page limit that can be used for internal operations.
// It is guaranteed to be at least 1 and at most the maximum page limit.
func (limit Limits) InternalPageLimit() int {
	if limit.MaxPageLimit > 0 && limit.DefaultPageLimit > limit.MaxPageLimit {
		return limit.MaxPageLimit
	}
	return limit.DefaultPageLimit
}

const (
	defaultMaxBodyBytes           = 1 << 20 // 1 MiB
	defaultDefaultPageLimit       = 100
	defaultMaxPageLimit           = 1000
	defaultMaxBatchItems          = 100
	defaultMaxAutocompleteUsers   = 10
	defaultMaxNamespaceIDAttempts = 100
	defaultMaxPIDAttempts         = 100
	defaultMaxAPIKeyAttempts      = 100
)

// DefaultLimits returns a new Limits struct with default values.
func DefaultLimits() Limits {
	return Limits{
		MaxBodyBytes:           defaultMaxBodyBytes,
		DefaultPageLimit:       defaultDefaultPageLimit,
		MaxPageLimit:           defaultMaxPageLimit,
		MaxBatchItems:          defaultMaxBatchItems,
		MaxAutocompleteUsers:   defaultMaxAutocompleteUsers,
		MaxNamespaceIDAttempts: defaultMaxNamespaceIDAttempts,
		MaxPIDAttempts:         defaultMaxPIDAttempts,
		MaxAPIKeyAttempts:      defaultMaxAPIKeyAttempts,
	}
}

// WithValidValues returns a copy of the Limits where each field is set to a valid value.
// All fields are set to at least zero, or their minimum valid value.
func (o Limits) WithValidValues() Limits {
	if o.MaxBodyBytes < 0 {
		o.MaxBodyBytes = 0
	}

	if o.DefaultPageLimit < 1 {
		o.DefaultPageLimit = 1
	}
	if o.MaxPageLimit < 0 {
		o.MaxPageLimit = 0
	}

	if o.MaxBatchItems < 0 {
		o.MaxBatchItems = 0
	}

	if o.MaxAutocompleteUsers < 1 {
		o.MaxAutocompleteUsers = 1
	}

	if o.MaxNamespaceIDAttempts < 1 {
		o.MaxNamespaceIDAttempts = 1
	}
	if o.MaxPIDAttempts < 1 {
		o.MaxPIDAttempts = 1
	}
	if o.MaxAPIKeyAttempts < 1 {
		o.MaxAPIKeyAttempts = 1
	}

	return o
}
