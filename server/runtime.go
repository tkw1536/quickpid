package server

import (
	"crypto/rand"
	"time"

	"github.com/google/uuid"
	"github.com/tkw1536/quickpid/pid"
)

// Runtime is used by server to interact with specific system functions.
type Runtime interface {
	// NewNamespaceID returns a new namespace identifier.
	// If nil, a v4 UUID is generated using [rand.Reader].
	NewNamespaceID() (string, error)

	// NewPID returns a new PID for the given PID format.
	// If nil, this calls [pid.Format.Generate] with [rand.Reader].
	NewPID(format pid.Format) (string, error)

	// Now returns the current time.
	// If nil, time.Now is used.
	Now() time.Time
}

// NewRuntime returns a new [Runtime] implementation, that uses [rand.Reader] to generate both namespace IDs and PIDs.
// It furthermore returns the real current time.
func NewRuntime() Runtime {
	return runtime{}
}

type runtime struct{}

func (runtime) NewNamespaceID() (string, error) {
	id, err := uuid.NewRandomFromReader(rand.Reader)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func (runtime) NewPID(format pid.Format) (string, error) {
	return format.Generate(rand.Reader)
}

func (runtime) Now() time.Time {
	return time.Now()
}
