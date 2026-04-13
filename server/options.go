package server

import "time"

// Options represents options for the handler.
type Options struct {
	// MountPath is the URL prefix where the handler will be mounted.
	// It must not have a trailing slash, e.g. "/api/v2".
	MountPath string

	// Disable swagger UI and spec file being served.
	DisableSwaggerUI bool

	// GeneratePID generates new PID strings for create resource operations.
	// If nil, a default generator is used.
	GeneratePID func() (string, error)

	// Now returns the current time.
	// If nil, time.Now is used.
	Now func() time.Time

	Limits Limits
}

// Limits represents limits for the server server.
type Limits struct {
	MaxBodyBytes  int64
	MaxBatchItems int

	DefaultPageLimit int
	MaxPageLimit     int
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
	return o
}

func (o Options) withDefaults() Options {
	o.Limits = o.Limits.withDefaults()
	if o.GeneratePID == nil {
		o.GeneratePID = RandomAlphanumericPID
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return o
}
