// Package quickpid holds specification data for the PID Resolver API.
//
//spellchecker:words quickpid
package quickpid

//spellchecker:words embed
import (
	"embed"
	"io/fs"
	"runtime/debug"
)

//go:embed spec/openapi.yaml
var apiSpec string

// Spec returns the OpenAPI specification as a string.
func Spec() string {
	return apiSpec
}

//go:embed spec/tests
var testDataFS embed.FS

// GetTestData returns an fs.FS containing only .json files with test data.
func GetTestData() fs.FS {
	data, err := fs.Sub(testDataFS, "spec/tests")
	if err != nil {
		panic(err)
	}
	return data
}

// CopyrightNotice is the copyright notice for the project.
const CopyrightNotice = "© Tom Wiesing. All rights reserved."

// License returns the full text of the license file.
//
// There currently is no license and this function returns the empty string.
func License() string {
	return ""
}

// Version returns the version of this module as reported by the Go runtime
// build info. It returns the empty string when build info is unavailable.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}
