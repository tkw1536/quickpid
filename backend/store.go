// Package backend provides the [Store] interface.
//
//spellchecker:words backend
package backend

//spellchecker:words context
import (
	"context"
)

// Store represents a backend for the bicpid application.
type Store interface {
	ResolverBackend
	AuthenticationBackend
	AuthorizationBackend
}

type Shutdowner interface {
	// Shutdown instructs to stop any in-fight operations, and permanently closes any internal resources.
	// No other methods may be called after Shutdown.
	//
	// Shutdown should return if ctx closes.
	Shutdown(ctx context.Context) error
}
