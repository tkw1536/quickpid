//spellchecker:words service
package service

//spellchecker:words sync github bicpid backend
import (
	"sync"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

// Service provides high-level PID resolver and authentication operations.
//
// The zero value is not ready to use; use [New] instead.
type Service struct {
	mu       sync.RWMutex
	opts     Options
	runtime  Runtime
	resolver backend.ResolverBackend
	auth     backend.AuthBackend
}

// New returns a new Service.
func New(resolver backend.ResolverBackend, auth backend.AuthBackend, runtime Runtime, opts Options) *Service {
	return &Service{
		opts:     opts.withValidValues(),
		runtime:  runtime,
		resolver: resolver,
		auth:     auth,
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

// GetResolverInfo returns information about the resolver.
//
// It can return the following errors:
//
// - [api.InfoUnavailable]
func (s *Service) GetResolverInfo() (*api.InfoResponse, api.Error, error) {
	s.mu.RLock()
	opts := s.opts
	s.mu.RUnlock()

	if !opts.InfoEnabled {
		return nil, api.InfoUnavailable, errSpecInfoPrivate
	}
	return &api.InfoResponse{
		MaxBodyBytes:     opts.Limits.MaxBodyBytes,
		DefaultPageLimit: int64(opts.Limits.DefaultPageLimit),
		MaxPageLimit:     int64(opts.Limits.MaxPageLimit),
		MaxBatchItems:    int64(opts.Limits.MaxBatchItems),
	}, "", nil
}
