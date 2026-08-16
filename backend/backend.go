// Package backend provides the [Backend] interface.
//
//spellchecker:words backend
package backend

//spellchecker:words context
import (
	"context"
)

// Backend represents a backend for the quickpid application.
//
// See the [memory] and [gorm] packages for implementations.
type Backend interface {
	UserBackend
	CredentialsBackend
	NamespaceBackend
	ResourceBackend
	AuthorizationBackend
	MountBackend

	// Shutdown instructs to stop any in-fight operations, and permanently closes any internal resources.
	// No other methods may be called after Shutdown.
	//
	// Shutdown should return if ctx closes.
	Shutdown(ctx context.Context) error
}
