// Package openapi provides a helper for replacing specific configuration within the openapi spec.
//
//spellchecker:words openapi
package openapi

//spellchecker:words bytes errors gopkg yaml
import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// SetServersPath loads an openapi yaml specification and replaces the server url with the given one.
func SetServersPath(input []byte, serverUrl string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(input, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	root, err := mappingRoot(&doc)
	if err != nil {
		return nil, fmt.Errorf("failed to find root mapping: %w", err)
	}

	var servers yaml.Node
	if err := servers.Encode([]map[string]string{{"url": serverUrl}}); err != nil {
		return nil, fmt.Errorf("failed to encode servers: %w", err)
	}
	setMappingKey(root, "servers", &servers)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(&doc); err != nil {
		return nil, fmt.Errorf("failed to encode yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encoder: %w", err)
	}
	return buf.Bytes(), nil
}

var (
	errEmptyDocument   = errors.New("empty yaml document")
	errExpectedMapping = errors.New("expected a yaml mapping")
)

// mappingRoot returns the top-level mapping node of a decoded yaml document.
func mappingRoot(doc *yaml.Node) (*yaml.Node, error) {
	n := doc
	if n.Kind == yaml.DocumentNode {
		if len(n.Content) == 0 {
			return nil, errEmptyDocument
		}
		n = n.Content[0]
	}
	if n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	if n == nil || n.Kind != yaml.MappingNode {
		return nil, errExpectedMapping
	}
	return n, nil
}

// setMappingKey sets key to value in a yaml mapping node, replacing it if present.
func setMappingKey(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		k := mapping.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			mapping.Content[i+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}
