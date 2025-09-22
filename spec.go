// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package http

import (
	api "github.com/gdt-dev/gdt/api"
)

// Capture describes how to capture response values into variables using JSONPath
type Capture map[string]string

// Spec describes a test of a single HTTP request and response
type Spec struct {
	api.Spec
	// URL being called by HTTP client
	URL string `yaml:"url,omitempty"`
	// HTTP Method specified by HTTP client
	Method string `yaml:"method,omitempty"`
	// Shortcut for URL and Method of "GET"
	GET string `yaml:"GET,omitempty"`
	// Shortcut for URL and Method of "POST"
	POST string `yaml:"POST,omitempty"`
	// Shortcut for URL and Method of "PUT"
	PUT string `yaml:"PUT,omitempty"`
	// Shortcut for URL and Method of "PATCH"
	PATCH string `yaml:"PATCH,omitempty"`
	// Shortcut for URL and Method of "DELETE"
	DELETE string `yaml:"DELETE,omitempty"`
	// Headers contains HTTP headers to be sent with the request
	Headers map[string]string `yaml:"headers,omitempty"`
	// JSON payload to send along in request
	Data interface{} `yaml:"data,omitempty"`
	// Capture contains JSONPath expressions to extract values from response
	Capture Capture `yaml:"capture,omitempty"`
	// Assert is the assertions for the HTTP response
	Assert *Expect `yaml:"assert,omitempty"`
}

// Title returns a good name for the Spec
func (s *Spec) Title() string {
	// If the user did not specify a name for the test spec, just default
	// it to the method and URL
	if s.Name != "" {
		return s.Name
	}
	return s.Method + ":" + s.URL
}

func (s *Spec) SetBase(b api.Spec) {
	s.Spec = b
}

func (s *Spec) Base() *api.Spec {
	return &s.Spec
}

// Retry returns the Evaluable's Retry override, if any
func (s *Spec) Retry() *api.Retry {
	return s.Spec.Retry
}

// Timeout returns the Evaluable's Timeout override, if any
func (s *Spec) Timeout() *api.Timeout {
	return s.Spec.Timeout
}
