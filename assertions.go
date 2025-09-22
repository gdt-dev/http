// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package http

import (
	"context"
	nethttp "net/http"
	"strings"

	api "github.com/gdt-dev/gdt/api"
	gdtjson "github.com/gdt-dev/gdt/assertion/json"
)

// PollConfig contains configuration for polling assertions
type PollConfig struct {
	// Interval is the time to wait between polling attempts
	Interval string `yaml:"interval,omitempty"`
	// Timeout is the maximum time to wait for the condition to be met
	Timeout string `yaml:"timeout,omitempty"`
	// Condition contains the assertions that must be met to stop polling
	Condition *Expect `yaml:"condition,omitempty"`
}

// Expect contains one or more assertions about an HTTP response
type Expect struct {
	// JSON contains the assertions about JSON data in the response
	JSON *gdtjson.Expect `yaml:"json,omitempty"`
	// Headers contains a list of HTTP headers that should be in the response
	Headers []string `yaml:"headers,omitempty"`
	// Strings contains a list of strings that should be present in the
	// response content
	Strings []string `yaml:"strings,omitempty"`
	// Status contains the numeric HTTP status code (e.g. 200 or 404) that
	// should be returned in the HTTP response
	Status *int `yaml:"status,omitempty"`
	// Poll contains configuration for polling assertions
	Poll *PollConfig `yaml:"poll,omitempty"`
}

// headerEqual returns true if the supplied http.Response contains an expected
// HTTP header
func headerEqual(r *nethttp.Response, exp string) bool {
	key, val, wantVal := strings.Cut(exp, ":")
	_, ok := r.Header[key]
	if !ok {
		return false
	}
	if !wantVal {
		// We were only looking for the Header key, didn't care about value.
		return true
	}
	valGot := r.Header.Get(key)
	if valGot == "" {
		return false
	}
	// If the string being compared is of the form Key: Value,
	// then we check for both existence and the value of the
	// header
	return strings.EqualFold(val, valGot)
}

// assertions contains all assertions made for the exec test
type assertions struct {
	// failures contains the set of error messages for failed assertions
	failures []error
	// terminal indicates there was a failure in evaluating the assertions that
	// should be considered a terminal condition (and therefore the test action
	// should not be retried).
	terminal bool
	// exp contains the expected conditions to assert against
	exp *Expect
	// r is the `nethttp.Response` we will evaluate
	r *nethttp.Response
	// b is the body of the `nethttp.Response` we've read into a buffer
	b []byte
}

// Fail appends a supplied error to the set of failed assertions
func (a *assertions) Fail(err error) {
	a.failures = append(a.failures, err)
}

// Failures returns a slice of errors for all failed assertions
func (a *assertions) Failures() []error {
	if a == nil {
		return []error{}
	}
	return a.failures
}

// Terminal returns a bool indicating the assertions failed in a way that is
// not retryable.
func (a *assertions) Terminal() bool {
	if a == nil {
		return false
	}
	return a.terminal
}

// OK checks all the assertions against the supplied arguments and returns true
// if all assertions pass.
func (a *assertions) OK(ctx context.Context) bool {
	if a.exp == nil {
		return true
	}
	exp := a.exp
	res := true
	if exp.Status != nil {
		got := a.r.StatusCode
		if *exp.Status != got {
			a.Fail(HTTPStatusNotEqual(*exp.Status, got))
			return false
		}
	}
	if exp.JSON != nil {
		ja := gdtjson.New(exp.JSON, a.b)
		jsonOK := ja.OK(ctx)
		if !jsonOK {
			for _, f := range ja.Failures() {
				a.Fail(f)
			}
			return false
		}
	}

	if len(exp.Strings) > 0 {
		for _, s := range exp.Strings {
			if !strings.Contains(string(a.b), s) {
				a.Fail(HTTPNotInBody(s))
				return false
			}
		}
	}

	if len(exp.Headers) > 0 {
		for _, header := range exp.Headers {
			if !headerEqual(a.r, header) {
				a.Fail(HTTPHeaderNotIn(header, a.r.Header))
				return false
			}
		}
	}
	return res
}

// newAssertions returns an assertions object populated with the supplied http
// spec assertions
func newAssertions(
	exp *Expect,
	r *nethttp.Response,
	b []byte,
) api.Assertions {
	return &assertions{
		failures: []error{},
		exp:      exp,
		r:        r,
		b:        b,
	}
}
