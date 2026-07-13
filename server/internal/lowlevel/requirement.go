//spellchecker:words lowlevel
package lowlevel

// authConfig describes an authentication configuration.
type authConfig struct {
	requirement authRequirement
	loadUser    bool
}

// requirement describes a low-level authentication requirement.
type authRequirement uint8

const (
	authRequirementNone authRequirement = iota
	authRequirementOptional
	authRequirementRequired
	authRequirementAuthMode
)

// authNone disables authentication entirely.
func authNone() authConfig {
	return authConfig{requirement: authRequirementNone}
}

// optionalUsernameAuth resolves the username if credentials are present.
func optionalUsernameAuth() authConfig {
	return authConfig{requirement: authRequirementOptional}
}

// requiredUsernameAuth requires valid credentials and resolves only the username.
func requiredUsernameAuth() authConfig {
	return authConfig{requirement: authRequirementRequired}
}

// optionalUserAuth loads the full user object if credentials are present.
func optionalUserAuth() authConfig {
	return authConfig{requirement: authRequirementOptional, loadUser: true}
}

// requiredUserAuth requires valid credentials and loads the full user object.
func requiredUserAuth() authConfig {
	return authConfig{requirement: authRequirementRequired, loadUser: true}
}

// requiredUserAuthManagement requires valid credentials in authenticated mode and
// returns anonymous_mode_unavailable in anonymous mode before checking credentials.
func requiredUserAuthManagement() authConfig {
	return authConfig{requirement: authRequirementAuthMode, loadUser: true}
}
