//spellchecker:words backend
package backend

//spellchecker:words context errors github quickpid
import (
	"context"
	"errors"

	"github.com/tkw1536/quickpid/api"
)

// MountBackend represents storage for namespace mounts.
//
// A mount is a unidirectional mapping from a base URI to a namespace id.
// There may only be one mapping per base URI; multiple base URIs may map to the same namespace.
//
// See [memory.NewStore] and [gorm.NewStore] for implementations.
type MountBackend interface {
	// Gets the namespace id mapped from the given base URI.
	//
	// Should return [ErrMountNotFound] if no mount exists for the base URI.
	GetMount(ctx context.Context, baseURI api.ValidBaseURI) (string, error)

	// Creates or replaces the mount for the given base URI.
	//
	// Should return [ErrNamespaceNotFound] if the namespace does not exist.
	SetMount(ctx context.Context, baseURI api.ValidBaseURI, namespace api.ValidNamespaceID) error

	// Deletes the mount for the given base URI.
	//
	// Should return [ErrMountNotFound] if no mount exists for the base URI.
	DeleteMount(ctx context.Context, baseURI api.ValidBaseURI) error

	// Lists all mounts, ordered ascending by namespace id then base URI.
	//
	// Has no specific error conditions.
	ListMounts(ctx context.Context, params api.ListMountsParams) (*api.PaginatedMountsResponse, error)

	// Lists base URIs mounted to the given namespace, ordered ascending by base URI.
	//
	// Should return [ErrNamespaceNotFound] if the namespace does not exist.
	ListNamespaceMounts(ctx context.Context, namespace api.ValidNamespaceID, params api.ListNamespaceMountsParams) (*api.PaginatedBaseURIResponse, error)

	WithShutdownMethod
}

// Sentinel errors to be returned by [MountBackend] implementations.
var (
	ErrMountNotFound = errors.New("mount not found")
)
