// Package openapi provides a helper for replacing specific configuration within the openapi spec.
//
//spellchecker:words openapi
package openapi

//spellchecker:words bytes pkglib yamlx gopkg yaml
import (
	"bytes"
	"fmt"

	"go.tkw01536.de/pkglib/yamlx"
	"gopkg.in/yaml.v3"
)

// SetServersPath loads an openapi yaml specification and replaces the server url with the given one.
func SetServersPath(input []byte, serverUrl string) ([]byte, error) {
	// decode the yaml node
	var node yaml.Node
	if err := yaml.Unmarshal(input, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	root, err := yamlx.Find(&node)
	if err != nil {
		return nil, fmt.Errorf("failed to find root node: %w", err)
	}

	serverYaml, err := yamlx.Marshal([]map[string]string{{"url": serverUrl}})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal yaml: %w", err)
	}
	serversNode, err := yamlx.Find(serverYaml)
	if err != nil {
		return nil, fmt.Errorf("failed to find root node: %w", err)
	}
	if err := yamlx.Assign(root, "servers", *serversNode); err != nil {
		return nil, fmt.Errorf("failed to assign servers node: %w", err)
	}

	// re en-code the document into bytes.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(&node); err != nil {
		return nil, fmt.Errorf("failed to encode yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encoder: %w", err)
	}
	return buf.Bytes(), nil
}
