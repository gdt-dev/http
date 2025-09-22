// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	gdtcontext "github.com/gdt-dev/gdt/context"
)

const (
	// captureFixtureName is the name of the fixture that stores captured variables
	captureFixtureName = "http_capture"
)

// CaptureFixture implements api.Fixture interface for storing captured variables
type CaptureFixture struct {
	variables map[string]interface{}
}

// NewCaptureFixture creates a new capture fixture
func NewCaptureFixture() *CaptureFixture {
	return &CaptureFixture{
		variables: make(map[string]interface{}),
	}
}

// HasState returns true if the fixture has state for the given key
func (f *CaptureFixture) HasState(key string) bool {
	_, exists := f.variables[key]
	return exists
}

// State returns the state for the given key
func (f *CaptureFixture) State(key string) interface{} {
	return f.variables[key]
}

// SetState sets the state for the given key
func (f *CaptureFixture) SetState(key string, value interface{}) {
	f.variables[key] = value
}

// Start is called when the fixture is started (no-op for capture fixture)
func (f *CaptureFixture) Start(ctx context.Context) error {
	return nil
}

// Stop is called when the fixture is stopped (no-op for capture fixture)
func (f *CaptureFixture) Stop(ctx context.Context) {
}

// getCaptureFixture returns the capture fixture from the context, creating one if it doesn't exist
func getCaptureFixture(ctx context.Context) *CaptureFixture {
	fixtures := gdtcontext.Fixtures(ctx)
	for _, f := range fixtures {
		if captureFixture, ok := f.(*CaptureFixture); ok {
			return captureFixture
		}
	}
	// If no capture fixture exists, this means variables from capture are not available
	// Return a new one anyway to avoid nil pointer errors
	return NewCaptureFixture()
}

// processCaptureRules processes the capture rules and extracts values from the response body
func (s *Spec) processCaptureRules(ctx context.Context, responseBody []byte) error {
	if s.Capture == nil || len(s.Capture) == 0 {
		return nil
	}

	// Parse the response body as JSON
	var jsonData interface{}
	if err := json.Unmarshal(responseBody, &jsonData); err != nil {
		return fmt.Errorf("failed to parse response as JSON for capture: %w", err)
	}

	// Get or create the capture fixture
	captureFixture := getCaptureFixture(ctx)

	// Process each capture rule
	for varName, jsonPathExpr := range s.Capture {
		// Extract value using JSONPath
		value, err := jsonpath.Get(jsonPathExpr, jsonData)
		if err != nil {
			return fmt.Errorf("failed to extract value for variable '%s' using JSONPath '%s': %w", varName, jsonPathExpr, err)
		}

		// Store the captured value
		captureFixture.SetState(varName, value)
	}

	return nil
}

// substituteVariables substitutes captured variables in a string value
func substituteVariables(ctx context.Context, value string) string {
	captureFixture := getCaptureFixture(ctx)

	// Simple variable substitution using {variable_name} syntax
	result := value

	// Get all captured variables and replace them
	for varName, varValue := range captureFixture.variables {
		placeholder := fmt.Sprintf("{%s}", varName)
		var replacement string
		if stringValue, ok := varValue.(string); ok {
			replacement = stringValue
		} else {
			// Convert non-string values to string representation
			replacement = fmt.Sprintf("%v", varValue)
		}
		result = strings.Replace(result, placeholder, replacement, -1)
	}

	return result
}