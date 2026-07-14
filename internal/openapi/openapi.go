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

// Server represents options for an openapi spec 'server' section.
type Server struct {
	MountPath string
}

// Rewrite loads input as an openapi yaml, updates the server according to the options, and then serializes it back as yaml.
func Rewrite(input []byte, server Server) ([]byte, error) {
	// decode the yaml node
	var node yaml.Node
	if err := yaml.Unmarshal(input, &node); err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal: %w", err)
	}

	root, err := yamlx.Find(&node)
	if err != nil {
		return nil, fmt.Errorf("yamlx.Find: %w", err)
	}

	serversNode, err := openapiYAMLNode([]map[string]string{{"url": server.MountPath}})
	if err != nil {
		return nil, err
	}
	if err := yamlx.Assign(root, "servers", *serversNode); err != nil {
		return nil, fmt.Errorf("yamlx.Assign: %w", err)
	}

	// re en-code the document into bytes.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(&node); err != nil {
		return nil, fmt.Errorf("encoder.Encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("Encoder.Close: %w", err)
	}
	return buf.Bytes(), nil
}

func openapiYAMLNode(value any) (*yaml.Node, error) {
	node, err := yamlx.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("yamlx.Marshal: %w", err)
	}
	root, err := yamlx.Find(node)
	if err != nil {
		return nil, fmt.Errorf("yamlx.Find: %w", err)
	}
	return root, nil
}
