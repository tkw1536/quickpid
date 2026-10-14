//spellchecker:words service
package service

//spellchecker:words crypto rand iter sync atomic time uuid github quickpid internal apikey idgen
import (
	"crypto/rand"
	"fmt"
	"iter"
	"sync/atomic"
	"time"
	"uuid"

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
//
// The parameter total is the approximate total number of identifiers that have been generated thus far, if known.
// If a single operation allocates multiple identifiers, the total might change during calls to this function,
// thus using the sync.atomic.Int64 tracks the total.
// Note that total never switches between nil and non-nil during a single operation.
//
// The [NewAPIKey] method intentionally omits this parameter, as API keys *must be* secure, and thus should not be predictable.
type Runtime interface {
	// NewNamespaceID is an identifier generator function for a new namespace identifier.
	// total is the approximate total number of namespaces that have been generated so far, if known.
	// maxAttempts is the maximum number of results that will be made by the caller during this operation.
	NewNamespaceID(total *atomic.Int64, maxAttempts int, round uint64) iter.Seq2[string, error]

	// NewAPIKeyID is an identifier generator function for a new API key id.
	// total is the approximate total number of API keys that have been generated so far, if known.
	// maxAttempts is the maximum number of results that will be made by the caller during this operation.
	NewAPIKeyID(total *atomic.Int64, maxAttempts int, round uint64) iter.Seq2[string, error]

	// NewAPIKeyID is an identifier generator function for a new API key in the given format.
	// maxAttempts is the maximum number of results that will be made by the caller during this operation.
	NewAPIKey(format apikey.Format, maxAttempts int, round uint64) iter.Seq2[string, error]

	// NewPID is an identifier generator function that returns a sequence of possible PIDs for the given format.
	// counter is the approximate total number of PIDs that have been generated in the namespace so far, if known.
	// maxAttempts is the maximum number of results that will be made by the caller during this operation.
	NewPID(format pid.Format, total *atomic.Int64, maxAttempts int, round uint64) iter.Seq2[string, error]

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

func (r runtime) NewNamespaceID(total *atomic.Int64, maxAttempts int, round uint64) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for {
			if !yield(uuid.NewV4().String(), nil) {
				return
			}
		}
	}
}

func (r runtime) NewAPIKeyID(total *atomic.Int64, maxAttempts int, round uint64) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		yield(uuid.NewV4().String(), nil)
	}
}

func (r runtime) NewAPIKey(format apikey.Format, maxAttempts int, round uint64) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		key, err := format.Generate(rand.Reader)
		if err != nil {
			yield("", fmt.Errorf("failed to generate API key in specified format: %w", err))
			return
		}
		yield(key, nil)
	}
}

func (r runtime) NewPID(format pid.Format, total *atomic.Int64, maxAttempts int, round uint64) iter.Seq2[string, error] {
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
