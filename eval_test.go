// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package http_test

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"testing"

	gdthttp "github.com/doingdd/http"
	"github.com/doingdd/http/test/server"
	"github.com/gdt-dev/gdt"
	api "github.com/gdt-dev/gdt/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dataFilePath = "testdata/fixtures.json"
)

type dataset struct {
	Authors    interface{}
	Publishers interface{}
	Books      []*server.Book
}

func data() *dataset {
	f, err := os.Open(dataFilePath)
	if err != nil {
		panic(err)
	}
	data := &dataset{}
	if err = json.NewDecoder(f).Decode(&data); err != nil {
		panic(err)
	}
	return data
}

func dataFixture() api.Fixture {
	f, err := os.Open(dataFilePath)
	if err != nil {
		panic(err)
	}
	fix, err := gdt.NewJSONFixture(f)
	if err != nil {
		panic(err)
	}
	return fix
}

func TestFixturesNotSetup(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	fp := filepath.Join("testdata", "create-then-get.yaml")
	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	err = s.Run(context.TODO(), t)
	require.NotNil(err)
	assert.ErrorIs(err, api.RuntimeError)
}

func setup(ctx context.Context) context.Context {
	// Register an HTTP server fixture that spins up the API service on a
	// random port on localhost
	logger := log.New(os.Stdout, "books_api_http: ", log.LstdFlags)
	srv := server.NewControllerWithBooks(logger, data().Books)
	serverFixture := gdthttp.NewServerFixture(srv.Router(), false /* useTLS */)
	ctx = gdt.RegisterFixture(ctx, "books_api", serverFixture)
	ctx = gdt.RegisterFixture(ctx, "books_data", dataFixture())

	// Register capture fixture for variable storage
	captureFixture := gdthttp.NewCaptureFixture()
	ctx = gdt.RegisterFixture(ctx, "http_capture", captureFixture)

	return ctx
}

func TestCreateThenGet(t *testing.T) {
	require := require.New(t)

	fp := filepath.Join("testdata", "create-then-get.yaml")

	ctx := gdt.NewContext()
	ctx = setup(ctx)

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	s.Run(ctx, t)
}

func TestFailures(t *testing.T) {
	require := require.New(t)

	fp := filepath.Join("testdata", "failures.yaml")

	ctx := gdt.NewContext()
	ctx = setup(ctx)

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	s.Run(ctx, t)
}

func TestGetBooks(t *testing.T) {
	require := require.New(t)

	fp := filepath.Join("testdata", "get-books.yaml")

	ctx := gdt.NewContext()
	ctx = setup(ctx)

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	s.Run(ctx, t)
}

func TestPutMultipleBooks(t *testing.T) {
	require := require.New(t)

	fp := filepath.Join("testdata", "put-multiple-books.yaml")

	ctx := gdt.NewContext()
	ctx = setup(ctx)

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	s.Run(ctx, t)
}

func TestCaptureFeature(t *testing.T) {
	require := require.New(t)

	fp := filepath.Join("testdata", "capture-test.yaml")

	ctx := gdt.NewContext()
	ctx = setup(ctx)

	s, err := gdt.From(fp)
	require.Nil(err)
	require.NotNil(s)

	s.Run(ctx, t)
}
