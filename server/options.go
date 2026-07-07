//spellchecker:words server
package server

//spellchecker:words github bicpid service
import (
	"github.com/tkw1536/bicpid/service"
)

// Options represents HTTP options for the handler.
type Options struct {
	// MountPath is the URL prefix where the handler will be mounted.
	// It must not have a trailing slash, e.g. "/api/v2".
	MountPath string

	// Disable swagger UI and spec file being served.
	DisableSwaggerUI bool

	// Limits and InfoEnabled are delegated to the service via [Handler.SetOptions].
	Limits service.Limits

	// InfoEnabled enables or disables the info endpoint.
	InfoEnabled bool
}

func (o Options) withValidValues() Options {
	o.Limits = o.Limits.WithValidValues()
	return o
}
