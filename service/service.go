//spellchecker:words service
package service

//spellchecker:words sync github quickpid backend
import (
	"sync"

	"github.com/tkw1536/quickpid/backend"
)

// Service provides high-level PID resolver and authentication operations.
//
// The zero value is not ready to use; use [New] instead.
//
// Service itself does not perform any authentication or authorization checks;
// These should always be performed by the caller, and possibly provided by means of a callback.
type Service struct {
	mu      sync.RWMutex
	opts    Options
	runtime Runtime
	backend backend.Backend
}

// New returns a new Service.
func New(backend backend.Backend, runtime Runtime, opts Options) *Service {
	return &Service{
		opts:    opts.withValidValues(),
		runtime: runtime,
		backend: backend,
	}
}

// SetOptions updates the options for this service.
// It is safe to call this method concurrently with other methods.
func (s *Service) SetOptions(opts Options) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.opts = opts.withValidValues()
}

// Options returns a copy of the current options.
func (s *Service) Options() Options {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.opts
}

// AnonymousMode reports whether the service runs without authentication.
func (s *Service) AnonymousMode() bool {
	return s.Options().Anonymous
}
