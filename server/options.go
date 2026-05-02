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
	o.Limits = o.Limits.withDefaults()
	return o
}

// Limits represents limits for the server server.
type Limits struct {
	MaxBodyBytes int64 // maximum size of request body

	DefaultPageLimit int // default number of items per page
	MaxPageLimit     int // maximum number of items per page

	MaxBatchItems int // maximum number of items in a batch

	MaxNamespaceIDAttempts int // maximum number of attempts to allocate a namespace ID
	MaxPIDAttempts         int // maximum number of attempts to allocate a PID
}

func (o Limits) withDefaults() Limits {
	if o.MaxBodyBytes <= 0 {
		o.MaxBodyBytes = 1 << 20
	}
	if o.MaxBatchItems <= 0 {
		o.MaxBatchItems = 100
	}
	if o.DefaultPageLimit <= 0 {
		o.DefaultPageLimit = 100
	}
	if o.MaxPageLimit <= 1 {
		o.MaxPageLimit = 1000
	}
	if o.MaxNamespaceIDAttempts <= 0 {
		o.MaxNamespaceIDAttempts = 100
	}
	if o.MaxPIDAttempts <= 0 {
		o.MaxPIDAttempts = 100
	}
	return o
}
