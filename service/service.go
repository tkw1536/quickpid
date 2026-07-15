//spellchecker:words service
package service

//spellchecker:words regexp sync github bicpid backend
import (
	"regexp"
	"sync"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

// Service provides high-level PID resolver and authentication operations.
//
// The zero value is not ready to use; use [New] instead.
type Service struct {
	mu      sync.RWMutex
	opts    Options
	runtime Runtime
	store   backend.Store
}

// New returns a new Service.
func New(store backend.Store, runtime Runtime, opts Options) *Service {
	return &Service{
		opts:    opts.withValidValues(),
		runtime: runtime,
		store:   store,
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

// GetResolverInfo returns information about the resolver.
//
// It can return the following errors:
//
// - [api.InfoUnavailable].
func (s *Service) GetResolverInfo() (*api.InfoResponse, error) {
	opts := s.Options()

	if !opts.InfoEnabled {
		return nil, api.WithErrorString(errSpecInfoPrivate, api.InfoUnavailable)
	}
	resp := &api.InfoResponse{
		MaxBodyBytes:     opts.Limits.MaxBodyBytes,
		DefaultPageLimit: int64(opts.Limits.DefaultPageLimit),
		MaxPageLimit:     int64(opts.Limits.MaxPageLimit),
		MaxBatchItems:    int64(opts.Limits.MaxBatchItems),
		Authentication:   !opts.Anonymous,
	}
	if !opts.Anonymous {
		resp.MaxAutocompleteUsers = int64(opts.Limits.MaxAutocompleteUsers)
	}
	return resp, nil
}

// regular expressions to validate various identifiers.
var (
	namespaceIDRE = regexp.MustCompile(`^[a-z0-9_-]+$`)
	pidRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	usernameRE    = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

// ValidateNamespaceID reports whether id is a valid namespace identifier.
func ValidateNamespaceID(id string) error {
	if !namespaceIDRE.MatchString(id) {
		return errInvalidNamespaceID
	}
	return nil
}

// ValidatePID reports whether id is a valid pid.
func ValidatePID(id string) error {
	if !pidRE.MatchString(id) {
		return errInvalidPID
	}
	return nil
}

// ValidateUsername reports whether username is a valid username.
func ValidateUsername(username string) error {
	if !usernameRE.MatchString(username) {
		return errInvalidUsername
	}
	return nil
}
