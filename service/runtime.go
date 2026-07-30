//spellchecker:words service
package service

//spellchecker:words crypto rand time github google uuid quickpid internal apikey
import (
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/pid"
)

// Runtime is used by the service to interact with specific system functions.
type Runtime interface {
	// NewNamespaceID returns a new namespace identifier.
	NewNamespaceID() (string, error)

	// NewAPIKeyID returns a new API key id.
	NewAPIKeyID() (string, error)

	// NewAPIKeyID returns a new API key.
	NewAPIKey(format apikey.Format) (string, error)

	// NewPID returns a new PID for the given PID format.
	NewPID(format pid.Format) (string, error)

	// Now returns the current time.
	Now() time.Time
}

// NewRuntime returns a new [Runtime] implementation that uses [rand.Reader] to generate
// namespace IDs, API key IDs, and PIDs, and returns the real current time.
func NewRuntime() Runtime {
	return runtime{
		random: rand.Reader,
	}
}

type runtime struct {
	random io.Reader
}

func (r runtime) NewNamespaceID() (string, error) {
	id, err := uuid.NewRandomFromReader(r.random)
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}
	return id.String(), nil
}

func (r runtime) NewAPIKeyID() (string, error) {
	id, err := uuid.NewRandomFromReader(r.random)
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}
	return id.String(), nil
}

func (r runtime) NewAPIKey(format apikey.Format) (string, error) {
	key, err := format.Generate(r.random)
	if err != nil {
		return "", fmt.Errorf("failed to generate API key in specified format: %w", err)
	}
	return key, nil
}

func (r runtime) NewPID(format pid.Format) (string, error) {
	pid, err := format.Generate(r.random)
	if err != nil {
		return "", fmt.Errorf("failed to generate PID in specified format: %w", err)
	}
	return pid, nil
}

func (runtime) Now() time.Time {
	return time.Now()
}
