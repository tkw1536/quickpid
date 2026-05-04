package server

// Options represents options for the handler.
type Options struct {
	// MountPath is the URL prefix where the handler will be mounted.
	// It must not have a trailing slash, e.g. "/api/v2".
	MountPath string

	// Disable swagger UI and spec file being served.
	DisableSwaggerUI bool

	// InfoEnabled disable the generic info endpoint.
	InfoEnabled bool

	// Limits for various internal server behavior.
	Limits Limits
}

func (o Options) withDefaults() Options {
	o.Limits = o.Limits.WithDefaults()
	return o
}

// Limits represent limits for the server.
type Limits struct {
	MaxBodyBytes int64 // maximum size of request body

	DefaultPageLimit int // default number of items per page
	MaxPageLimit     int // maximum number of items per page

	MaxBatchItems int // maximum number of items in a batch

	MaxNamespaceIDAttempts int // maximum number of attempts to allocate a namespace ID
	MaxPIDAttempts         int // maximum number of attempts to allocate a PID
}

const (
	defaultMaxBodyBytes           = 1 << 20 // 1 MiB
	defaultDefaultPageLimit       = 100
	defaultMaxPageLimit           = 1000
	defaultMaxBatchItems          = 100
	defaultMaxNamespaceIDAttempts = 100
	defaultMaxPIDAttempts         = 100
)

// WithDefaults returns a copy of the limits with default values applied for unset fields.
func (o Limits) WithDefaults() Limits {
	if o.MaxBodyBytes <= 0 {
		o.MaxBodyBytes = defaultMaxBodyBytes
	}
	if o.MaxBatchItems <= 0 {
		o.MaxBatchItems = defaultMaxBatchItems
	}
	if o.DefaultPageLimit <= 0 {
		o.DefaultPageLimit = defaultDefaultPageLimit
	}
	if o.MaxPageLimit <= 1 {
		o.MaxPageLimit = defaultMaxPageLimit
	}
	if o.MaxNamespaceIDAttempts <= 0 {
		o.MaxNamespaceIDAttempts = defaultMaxNamespaceIDAttempts
	}
	if o.MaxPIDAttempts <= 0 {
		o.MaxPIDAttempts = defaultMaxPIDAttempts
	}
	return o
}
