//spellchecker:words service
package service

//spellchecker:words crypto rand iter time uuid github quickpid internal apikey idgen
import (
	"crypto/rand"
	"fmt"
	"iter"
	"time"
	"uuid"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/internal/idgen"
	"github.com/tkw1536/quickpid/pid"
)

// imported so that we can link to the docs below.
var _ idgen.IDGenerationFunction = nil

// Runtime is used by the service to interact with specific system functions.
//
// Runtime contains several "identifier generator functions" which are used during operations.
// Each is called New* and has the same semantics as an [idgen.IDGenerationFunction], except that it is passed extra parameters.
type Runtime interface {
	// NewNamespaceID is an identifier generator function for a new namespace identifier.
	NewNamespaceID(exhausted uint64) iter.Seq2[string, error]

	// NewAPIKeyID is an identifier generator function for a new API key id.
	NewAPIKeyID(exhausted uint64) iter.Seq2[string, error]

	// NewAPIKeyID is an identifier generator function for a new API key in the given format.
	NewAPIKey(format apikey.Format, exhausted uint64) iter.Seq2[string, error]

	// NewPID is an identifier generator function that returns a sequence of possible PIDs for the given format.
	// It is optionally passed a namespace id, hinting the namespace for which the PID will be used.
	NewPID(format pid.Format, namespace *api.ValidNamespaceID, exhausted uint64) iter.Seq2[string, error]

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

func (r runtime) NewNamespaceID(exhausted uint64) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for {
			if !yield(uuid.NewV4().String(), nil) {
				return
			}
		}
	}
}

func (r runtime) NewAPIKeyID(exhausted uint64) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		yield(uuid.NewV4().String(), nil)
	}
}

func (r runtime) NewAPIKey(format apikey.Format, exhausted uint64) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		key, err := format.Generate(rand.Reader)
		if err != nil {
			yield("", fmt.Errorf("failed to generate API key in specified format: %w", err))
			return
		}
		yield(key, nil)
	}
}

func (r runtime) NewPID(format pid.Format, namespace *api.ValidNamespaceID, exhausted uint64) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		pid, err := format.Generate(rand.Reader)
		if err != nil {
			yield(pid, fmt.Errorf("failed to generate PID in specified format: %w", err))
			return
		}
		yield(pid, nil)
	}
}

func (runtime) Now() time.Time {
	return time.Now()
}
