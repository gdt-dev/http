// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	nethttp "net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gdt-dev/gdt/api"
	gdtcontext "github.com/gdt-dev/gdt/context"
	gdtdebug "github.com/gdt-dev/gdt/debug"
)

// RunData is data stored in the context about the run. It is fetched from the
// gdtcontext.PriorRun() function and evaluated for things like the special
// `$LOCATION` URL value.
type RunData struct {
	Response *nethttp.Response
}

// priorRunData returns any prior run cached data in the context.
func priorRunData(ctx context.Context) *RunData {
	prData := gdtcontext.PriorRun(ctx)
	httpData, ok := prData[pluginName]
	if !ok {
		return nil
	}
	if data, ok := httpData.(*RunData); ok {
		return data
	}
	return nil
}

// getURL returns the URL to use for the test's HTTP request. The test's url
// field is first queried to see if it is the special $LOCATION string. If it
// is, then we return the previous HTTP response's Location header. Otherwise,
// we construct the URL from the httpFile's base URL and the test's url field.
func (s *Spec) getURL(ctx context.Context) (string, error) {
	if strings.ToUpper(s.URL) == "$LOCATION" {
		pr := priorRunData(ctx)
		if pr == nil || pr.Response == nil {
			panic("test unit referenced $LOCATION before executing an HTTP request")
		}
		url, err := pr.Response.Location()
		if err != nil {
			return "", ErrExpectedLocationHeader
		}
		return url.String(), nil
	}

	d := fromBaseDefaults(s.Defaults)
	base := d.BaseURLFromContext(ctx)

	// Apply variable substitution to URL
	url := substituteVariables(ctx, s.URL)
	return base + url, nil
}

// processRequestData looks through the raw data interface{} that was
// unmarshaled during parse for any string values that look like JSONPath
// expressions. If we find any, we query the fixture registry to see if any
// fixtures have a value that matches the JSONPath expression. See
// gdt.fixtures:jsonFixture for more information on how this works
func (s *Spec) processRequestData(ctx context.Context) {
	if s.Data == nil {
		return
	}
	// Get a pointer to the unmarshaled interface{} so we can mutate the
	// contents pointed to
	p := reflect.ValueOf(&s.Data)

	// We're interested in the value pointed to by the interface{}, which is
	// why we do a double Elem() here.
	v := p.Elem().Elem()
	vt := v.Type()

	switch vt.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i).Elem()
			it := item.Type()
			s.preprocessMap(ctx, item, it.Key(), it.Elem())
		}
		//	ht.f.preprocessSliceValue(v, vt.Key(), vt.Elem())
	case reflect.Map:
		s.preprocessMap(ctx, v, vt.Key(), vt.Elem())
	}
}

// client returns the HTTP client to use when executing HTTP requests. If any
// fixture provides a state with key "http.client", the fixture is asked for
// the HTTP client. Otherwise, we use the net/http.DefaultClient
func (s *Spec) client(ctx context.Context) *nethttp.Client {
	// query the fixture registry to determine if any of them contain an
	// http.client state attribute.
	fixtures := gdtcontext.Fixtures(ctx)
	for _, f := range fixtures {
		if f.HasState(StateKeyClient) {
			c, ok := f.State(StateKeyClient).(*nethttp.Client)
			if !ok {
				panic("fixture failed to return a *net/http.Client")
			}
			return c
		}
	}
	return nethttp.DefaultClient
}

// processRequestDataMap processes a map pointed to by v, transforming any
// string keys or values of the map into the results of calling the fixture
// set's State() method.
func (s *Spec) preprocessMap(
	ctx context.Context,
	m reflect.Value,
	kt reflect.Type,
	vt reflect.Type,
) error {
	it := m.MapRange()
	for it.Next() {
		if kt.Kind() == reflect.String {
			keyStr := it.Key().String()
			fixtures := gdtcontext.Fixtures(ctx)
			for _, f := range fixtures {
				if !f.HasState(keyStr) {
					continue
				}
				trKeyStr := f.State(keyStr)
				keyStr = trKeyStr.(string)
			}

			val := it.Value()
			err := s.preprocessMapValue(ctx, m, reflect.ValueOf(keyStr), val, val.Type())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Spec) preprocessMapValue(
	ctx context.Context,
	m reflect.Value,
	k reflect.Value,
	v reflect.Value,
	vt reflect.Type,
) error {
	if vt.Kind() == reflect.Interface {
		v = v.Elem()
		vt = v.Type()
	}

	switch vt.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			fmt.Println(item)
		}
		fmt.Printf("map element is an array.\n")
	case reflect.Map:
		return s.preprocessMap(ctx, v, vt.Key(), vt.Elem())
	case reflect.String:
		valStr := v.String()

		// First check if it's a fixture reference
		fixtures := gdtcontext.Fixtures(ctx)
		for _, f := range fixtures {
			if !f.HasState(valStr) {
				continue
			}
			trValStr := f.State(valStr)
			m.SetMapIndex(k, reflect.ValueOf(trValStr))
			return nil
		}

		// If not a fixture reference, apply variable substitution
		substitutedValue := substituteVariables(ctx, valStr)
		if substitutedValue != valStr {
			m.SetMapIndex(k, reflect.ValueOf(substitutedValue))
		}
	default:
		return nil
	}
	return nil
}

// Run executes the test described by the HTTP test. A new HTTP request and
// response pair is created during this call.
func (s *Spec) Eval(ctx context.Context) (*api.Result, error) {
	runData := &RunData{}
	var rerr error
	fails := []error{}

	url, err := s.getURL(ctx)
	if err != nil {
		return nil, err
	}

	gdtdebug.Println(ctx, "http: > %s %s", s.Method, url)
	var body io.Reader
	if s.Data != nil {
		s.processRequestData(ctx)
		jsonBody, err := json.Marshal(s.Data)
		if err != nil {
			return nil, err
		}
		b := bytes.NewReader(jsonBody)
		if b.Size() > 0 {
			sendData, _ := io.ReadAll(b)
			gdtdebug.Println(ctx, "http: > %s", sendData)
			b.Seek(0, 0)
		}
		body = b
	}

	req, err := nethttp.NewRequest(s.Method, url, body)
	if err != nil {
		return nil, err
	}

	// Set custom headers if provided
	if s.Headers != nil {
		for key, value := range s.Headers {
			// Apply variable substitution to header values
			substitutedValue := substituteVariables(ctx, value)
			req.Header.Set(key, substitutedValue)
		}
	}

	// TODO(jaypipes): Allow customization of the HTTP client for proxying,
	// TLS, etc
	c := s.client(ctx)

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	gdtdebug.Println(ctx, "http: < %d", resp.StatusCode)

	// Make sure we drain and close our response body...
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	// Only read the response body contents once and pass the byte
	// buffer to the assertion functions
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(b) > 0 {
		gdtdebug.Println(ctx, "http: < %s", b)
	}

	// Process capture rules before assertions
	if err := s.processCaptureRules(ctx, b); err != nil {
		return nil, err
	}

	exp := s.Assert
	if exp != nil {
		// Check if polling is required
		if exp.Poll != nil {
			pollErr := s.handlePolling(ctx, exp)
			if pollErr != nil {
				return nil, pollErr
			}
		}

		a := newAssertions(exp, resp, b)
		if !a.OK(ctx) {
			fails = a.Failures()
		}
	}
	runData.Response = resp

	if rerr != nil {
		return nil, rerr
	}

	return api.NewResult(
		api.WithFailures(fails...),
		api.WithData(pluginName, runData),
	), nil
}

// handlePolling implements the polling logic for HTTP assertions
func (s *Spec) handlePolling(ctx context.Context, exp *Expect) error {
	if exp.Poll == nil {
		return nil
	}

	// Parse polling configuration
	interval, err := time.ParseDuration(exp.Poll.Interval)
	if err != nil {
		interval = 5 * time.Second // default interval
	}

	timeout, err := time.ParseDuration(exp.Poll.Timeout)
	if err != nil {
		timeout = 60 * time.Second // default timeout
	}

	gdtdebug.Println(ctx, "http: starting polling with interval=%v, timeout=%v", interval, timeout)

	// Start polling
	startTime := time.Now()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// Check timeout
		if time.Since(startTime) > timeout {
			return fmt.Errorf("polling timeout exceeded after %v", timeout)
		}

		// Execute HTTP request
		url, err := s.getURL(ctx)
		if err != nil {
			return err
		}

		var body io.Reader
		if s.Data != nil {
			s.processRequestData(ctx)
			jsonBody, err := json.Marshal(s.Data)
			if err != nil {
				return err
			}
			body = bytes.NewReader(jsonBody)
		}

		req, err := nethttp.NewRequest(s.Method, url, body)
		if err != nil {
			return err
		}

		// Set custom headers if provided
		if s.Headers != nil {
			for key, value := range s.Headers {
				// Apply variable substitution to header values
				substitutedValue := substituteVariables(ctx, value)
				req.Header.Set(key, substitutedValue)
			}
		}

		c := s.client(ctx)

		resp, err := c.Do(req)
		if err != nil {
			return err
		}

		// Read response body
		b, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		gdtdebug.Println(ctx, "http: polling attempt - status: %d", resp.StatusCode)
		gdtdebug.Println(ctx, "http: polling response body: %s", string(b))

		// Check if poll condition is met
		if exp.Poll.Condition != nil {
			gdtdebug.Println(ctx, "http: polling condition - expected status: %v", exp.Poll.Condition.Status)
			if exp.Poll.Condition.JSON != nil {
				gdtdebug.Println(ctx, "http: polling condition - has JSON assertion")
			} else {
				gdtdebug.Println(ctx, "http: polling condition - NO JSON assertion")
			}

			a := newAssertions(exp.Poll.Condition, resp, b)
			isOK := a.OK(ctx)
			gdtdebug.Println(ctx, "http: polling condition check result: %v", isOK)
			if isOK {
				gdtdebug.Println(ctx, "http: polling condition met, stopping")
				return nil
			} else {
				failures := a.Failures()
				if len(failures) > 0 {
					gdtdebug.Println(ctx, "http: polling condition not met, failures: %v", failures)
				}
			}
		}

		// Wait for next polling interval
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
