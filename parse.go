// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package http

import (
	"strings"

	"github.com/gdt-dev/gdt/api"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

func (s *Spec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return api.ExpectedMapAt(node)
	}
	// maps/structs are stored in a top-level Node.Content field which is a
	// concatenated slice of Node pointers in pairs of key/values.
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Kind != yaml.ScalarNode {
			return api.ExpectedScalarAt(keyNode)
		}
		key := keyNode.Value
		valNode := node.Content[i+1]
		switch key {
		case "url":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			s.URL = strings.TrimSpace(valNode.Value)
		case "method":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			s.Method = strings.TrimSpace(valNode.Value)
		case "GET":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			s.GET = strings.TrimSpace(valNode.Value)
		case "POST":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			s.POST = strings.TrimSpace(valNode.Value)
		case "PUT":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			s.PUT = strings.TrimSpace(valNode.Value)
		case "DELETE":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			s.DELETE = strings.TrimSpace(valNode.Value)
		case "PATCH":
			if valNode.Kind != yaml.ScalarNode {
				return api.ExpectedScalarAt(valNode)
			}
			s.PATCH = strings.TrimSpace(valNode.Value)
		case "headers":
			if valNode.Kind != yaml.MappingNode {
				return api.ExpectedMapAt(valNode)
			}
			headers := make(map[string]string)
			if err := valNode.Decode(&headers); err != nil {
				return err
			}
			s.Headers = headers
		case "data":
			var data interface{}
			if err := valNode.Decode(&data); err != nil {
				return err
			}
			s.Data = data
		case "capture":
			if valNode.Kind != yaml.MappingNode {
				return api.ExpectedMapAt(valNode)
			}
			var capture Capture
			if err := valNode.Decode(&capture); err != nil {
				return err
			}
			s.Capture = capture
		case "assert":
			if valNode.Kind != yaml.MappingNode {
				return api.ExpectedMapAt(valNode)
			}
			var exp *Expect
			if err := valNode.Decode(&exp); err != nil {
				return err
			}
			s.Assert = exp
		default:
			if lo.Contains(api.BaseSpecFields, key) {
				continue
			}
			return api.UnknownFieldAt(key, keyNode)
		}
	}
	if err := validateMethodAndURL(s); err != nil {
		return err
	}
	return nil
}

func validateMethodAndURL(s *Spec) error {
	if s.URL == "" {
		if s.GET != "" {
			s.Method = "GET"
			s.URL = s.GET
			return nil
		} else if s.POST != "" {
			s.Method = "POST"
			s.URL = s.POST
			return nil
		} else if s.PUT != "" {
			s.Method = "PUT"
			s.URL = s.PUT
			return nil
		} else if s.DELETE != "" {
			s.Method = "DELETE"
			s.URL = s.DELETE
			return nil
		} else if s.PATCH != "" {
			s.Method = "PATCH"
			s.URL = s.PATCH
			return nil
		} else {
			return ErrAliasOrURL
		}
	}
	if s.Method == "" {
		return ErrAliasOrURL
	}
	return nil
}
