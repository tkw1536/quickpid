//spellchecker:words service
package service

//spellchecker:words errors github quickpid
import (
	"errors"

	"github.com/tkw1536/quickpid/api"
)

var errSpecInfoPrivate = errors.New("info is private")

// GetResolverInfo returns information about the resolver.
//
// It can return the following errors:
//
// - [api.InfoUnavailable].
func (s *Service) GetResolverInfo() (*api.InfoResponse, error) {
	opts := s.Options()

	if !opts.InfoEnabled {
		return nil, api.WithErrorCode(errSpecInfoPrivate, api.InfoUnavailable)
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
