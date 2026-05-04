// Package quickpid holds specification data for the PID Resolver API.
package quickpid

import (
	"embed"
	"io/fs"
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
const CopyrightNotice = "© Tom Wiesing. Available under AGPL 3.0."

//go:embed LICENSE
var license string

// License returns the full text of the license file.
func License() string {
	return license
}
