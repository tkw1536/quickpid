//spellchecker:words service
package service

//spellchecker:words crypto rand time github google uuid bicpid
import (
	"crypto/rand"
	"time"

	"github.com/google/uuid"
	"github.com/tkw1536/bicpid/pid"
)

// Runtime is used by the service to interact with specific system functions.
type Runtime interface {
	// NewNamespaceID returns a new namespace identifier.
	NewNamespaceID() (string, error)

	// NewPID returns a new PID for the given PID format.
	NewPID(format pid.Format) (string, error)

	// Now returns the current time.
	Now() time.Time
}

// NewRuntime returns a new [Runtime] implementation that uses [rand.Reader] to generate
// namespace IDs and PIDs, and returns the real current time.
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
