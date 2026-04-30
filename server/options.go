package server

import (
	"crypto/rand"
	"time"

	"github.com/google/uuid"
	"github.com/tkw1536/quickpid/pid"
)

// Options represents options for the handler.
type Options struct {
	// MountPath is the URL prefix where the handler will be mounted.
	// It must not have a trailing slash, e.g. "/api/v2".
	MountPath string

	// Disable swagger UI and spec file being served.
	DisableSwaggerUI bool

	// NewNamespaceID returns a new namespace identifier.
	// If nil, a v4 UUID is generated using [rand.Reader].
	NewNamespaceID func() (string, error)

	// NewPID returns a new PID for the given PID format.
	// If nil, this calls [pid.Format.Generate] with [rand.Reader].
	NewPID func(format pid.Format) (string, error)

	// Now returns the current time.
	// If nil, time.Now is used.
	Now func() time.Time

	// Limits for various internal server behavior.
	Limits Limits
}

func (o Options) withDefaults() Options {
	o.Limits = o.Limits.withDefaults()
	if o.NewNamespaceID == nil {
		o.NewNamespaceID = defaultNewNamespaceID
	}
	if o.NewPID == nil {
		o.NewPID = defaultNewPID
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return o
}

var defaultNewNamespaceID = func() (string, error) {
	id, err := uuid.NewRandomFromReader(rand.Reader)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

var defaultNewPID = func(format pid.Format) (string, error) {
	return format.Generate(rand.Reader)
}

// Limits represents limits for the server server.
type Limits struct {
	MaxBodyBytes int64 // maximum size of request body

	DefaultPageLimit int // default number of items per page
	MaxPageLimit     int // maximum number of items per page

	MaxBatchItems int // maximum number of items in a batch

	MaxPIDAttempts int // maximum number of attempts to allocate a PID
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
	if o.MaxPIDAttempts <= 0 {
		o.MaxPIDAttempts = 100
	}
	return o
}
