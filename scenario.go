// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package http

import (
	"context"

	"github.com/gdt-dev/gdt"
)

// NewContextWithCapture creates a new GDT context with CaptureFixture pre-registered.
// This ensures that HTTP capture functionality works without manual setup.
func NewContextWithCapture() context.Context {
	ctx := gdt.NewContext()

	// Register a CaptureFixture for HTTP variable capture
	captureFixture := NewCaptureFixture()
	ctx = gdt.RegisterFixture(ctx, captureFixtureName, captureFixture)

	return ctx
}