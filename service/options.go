// Package service provides high-level PID resolver and auth operations.
//
//spellchecker:words service
package service

// Options represents operational configuration for the service.
type Options struct {
	// InfoEnabled enables the generic info endpoint.
	InfoEnabled bool

	// DefaultNamespaceOwner is used as the namespace owner when CreateNamespace
	// is called without an explicit owner (e.g. before auth is enforced on that route).
	DefaultNamespaceOwner string

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

	MaxNamespaceIDAttempts int // maximum number of attempts to allocate a namespace ID, must be at least 1.
	MaxPIDAttempts         int // maximum number of attempts to allocate a PID, must be at least 1.
}

const (
	defaultMaxBodyBytes           = 1 << 20 // 1 MiB
	defaultDefaultPageLimit       = 100
	defaultMaxPageLimit           = 1000
	defaultMaxBatchItems          = 100
	defaultMaxNamespaceIDAttempts = 100
	defaultMaxPIDAttempts         = 100
)

// DefaultLimits returns a new Limits struct with default values.
func DefaultLimits() Limits {
	return Limits{
		MaxBodyBytes:           defaultMaxBodyBytes,
		DefaultPageLimit:       defaultDefaultPageLimit,
		MaxPageLimit:           defaultMaxPageLimit,
		MaxBatchItems:          defaultMaxBatchItems,
		MaxNamespaceIDAttempts: defaultMaxNamespaceIDAttempts,
		MaxPIDAttempts:         defaultMaxPIDAttempts,
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

	if o.MaxNamespaceIDAttempts < 1 {
		o.MaxNamespaceIDAttempts = 1
	}
	if o.MaxPIDAttempts < 1 {
		o.MaxPIDAttempts = 1
	}

	return o
}
