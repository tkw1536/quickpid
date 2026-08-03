//spellchecker:words service
package service

//spellchecker:words crypto rand time uuid github quickpid internal apikey
import (
	"crypto/rand"
	"fmt"
	"time"
	"uuid"

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
	return runtime{}
}

type runtime struct {
}

func (r runtime) NewNamespaceID() (string, error) {
	return uuid.NewV4().String(), nil
}

func (r runtime) NewAPIKeyID() (string, error) {
	return uuid.NewV4().String(), nil
}

func (r runtime) NewAPIKey(format apikey.Format) (string, error) {
	key, err := format.Generate(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate API key in specified format: %w", err)
	}
	return key, nil
}

func (r runtime) NewPID(format pid.Format) (string, error) {
	pid, err := format.Generate(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate PID in specified format: %w", err)
	}
	return pid, nil
}

func (runtime) Now() time.Time {
	return time.Now()
}
