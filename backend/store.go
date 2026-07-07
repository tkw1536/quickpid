package backend

// Store combines resolver, authentication, and authorization backends backed by one store.
type Store interface {
	ResolverBackend
	AuthBackend
	AuthorizationBackend
}
