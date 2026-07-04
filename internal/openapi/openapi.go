// Package openapi provides a helper for replacing specific configuration within the openapi spec.
package openapi

import (
	"bytes"

	"go.tkw01536.de/pkglib/yamlx"
	"gopkg.in/yaml.v3"
)

// Server represents options for an openapi spec 'server' section.
type Server struct {
	MountPath string
	BasicAuth bool
}

// Rewrite loads input as an openapi yaml, updates the server according to the options, and then serializes it back as yaml.
func Rewrite(input []byte, server Server) ([]byte, error) {
	// decode the yaml node
	var node yaml.Node
	if err := yaml.Unmarshal(input, &node); err != nil {
		return nil, err
	}

	root, err := yamlx.Find(&node)
	if err != nil {
		return nil, err
	}

	serversNode, err := openapiYAMLNode([]map[string]string{{"url": server.MountPath}})
	if err != nil {
		return nil, err
	}
	if err := yamlx.Assign(root, "servers", *serversNode); err != nil {
		return nil, err
	}

	if server.BasicAuth {
		schemeNode, err := openapiYAMLNode(map[string]any{
			"type":   "http",
			"scheme": "basic",
		})
		if err != nil {
			return nil, err
		}
		securitySchemes := ensureOpenAPIMapValue(ensureOpenAPIMapValue(root, "components"), "securitySchemes")
		if err := yamlx.Assign(securitySchemes, "BasicAuth", *schemeNode); err != nil {
			return nil, err
		}

		securityNode, err := openapiYAMLNode([]map[string][]string{{"BasicAuth": {}}})
		if err != nil {
			return nil, err
		}
		if err := yamlx.Assign(root, "security", *securityNode); err != nil {
			return nil, err
		}
	}

	// re en-code the document into bytes.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(&node); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func openapiYAMLNode(value any) (*yaml.Node, error) {
	node, err := yamlx.Marshal(value)
	if err != nil {
		return nil, err
	}
	root, err := yamlx.Find(node)
	if err != nil {
		return nil, err
	}
	return root, nil
}

func ensureOpenAPIMapValue(node *yaml.Node, key string) *yaml.Node {
	if child, err := yamlx.Child(node, key); err == nil {
		return child
	}

	child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	node.Content = append(
		node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		child,
	)
	return child
}
