//spellchecker:words auth
package auth

//spellchecker:words errors slog http strings
import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// Authorization handles authorization for a specific request.
//
// Multiple functions may be called concurrently.
type Authorization interface {
	// CanListNamespaces checks if the user associated with the given request can list namespaces.
	CanListNamespaces(logger *slog.Logger, r *http.Request) (bool, error)

	// CanCreate checks if the given user is authorized to create new namespaces.
	CanCreate(logger *slog.Logger, r *http.Request) (bool, error)

	// OnCreateNamespace is called immediately after a new namespace is created by a user associated with the given request.
	// In case permissions are dynamic, this function can be used to record new permissions for the user.
	OnCreateNamespace(logger *slog.Logger, r *http.Request, namespace string) error

	// GetNamespaceLevel returns the level for the user associated with the given request and namespace.
	GetNamespaceLevel(logger *slog.Logger, r *http.Request, namespace string) (Level, error)

	// OnUnauthorized is called once a request is marked as unauthorized.
	// It should prompt the user to re-authenticate, e.g. via a WWW-Authenticate header.
	OnUnauthorized(logger *slog.Logger, w http.ResponseWriter, r *http.Request)
}

// Level represents the level an authorized user has for a namespace.
type Level string

const (
	// LevelNone indicates that only reading data of specific PIDs is permitted.
	LevelNone Level = "none"
	// LevelContributor indicates that on top of [LevelNone], the user can create new resources in the namespace.
	// Listing all existing PIDs is not permitted.
	LevelContributor = "contributor"
	// LevelEditor indicates that on top of [LevelContributor], the user can also update and list resources in a namespace.
	LevelEditor = "editor"
)

var errInvalidLevel = errors.New("invalid level")

func (l *Level) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "none":
		*l = LevelNone
	case "contributor":
		*l = LevelContributor
	case "editor":
		*l = LevelEditor
	default:
		return fmt.Errorf("%w: %s", errInvalidLevel, string(text))
	}
	return nil
}

// Int turns this level into an integer, for easy comparison.
func (l Level) Int() int {
	switch l {
	case LevelNone:
		return 0
	case LevelContributor:
		return 1
	case LevelEditor:
		return 2
	}
	return -1
}

// NoAuthentication is an [Authorization] that performs no checks.
type NoAuthentication struct{}

func (NoAuthentication) CanListNamespaces(l *slog.Logger, r *http.Request) (bool, error) {
	return true, nil
}
func (NoAuthentication) GetNamespaceLevel(l *slog.Logger, r *http.Request, namespace string) (Level, error) {
	return LevelEditor, nil
}
func (NoAuthentication) CanCreate(l *slog.Logger, r *http.Request) (bool, error) {
	return true, nil
}
func (NoAuthentication) OnCreateNamespace(l *slog.Logger, r *http.Request, namespace string) error {
	return nil
}
func (NoAuthentication) OnUnauthorized(l *slog.Logger, w http.ResponseWriter, r *http.Request) {}
