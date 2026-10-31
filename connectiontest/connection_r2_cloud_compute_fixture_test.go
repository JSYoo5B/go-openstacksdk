package connectiontest_test

import (
	"io"
	"net/http"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

// r2CloudComputeCall records one request seen by the fixture.
type r2CloudComputeCall struct{ Method, Path, Query, Body string }

type r2CloudComputeCloud struct {
	*testcloud.Cloud
	Conn  *sdk.Connection
	mu    sync.Mutex
	calls []r2CloudComputeCall
}

func (c *r2CloudComputeCloud) Calls() []r2CloudComputeCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]r2CloudComputeCall(nil), c.calls...)
}

// r2CloudComputeNew serves Nova under /compute/v2.1/p/ and Neutron under
// /network/ and records every request.
func r2CloudComputeNew(t *testing.T, options ...sdk.ConnectionOption) *r2CloudComputeCloud {
	t.Helper()
	cloud := testcloud.New(t)
	fixture := &r2CloudComputeCloud{Cloud: cloud}
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		switch o.Type {
		case "compute":
			return cloud.Server.URL + "/compute/v2.1/p/", nil
		case "network":
			return cloud.Server.URL + "/network/", nil
		}
		return "", gophercloud.ErrEndpointNotFound{}
	}
	conn, err := sdk.FromProvider(cloud.Provider, append([]sdk.ConnectionOption{sdk.WithMicroversion(sdk.Compute, "2.96")}, options...)...)
	if err != nil {
		t.Fatal(err)
	}
	fixture.Conn = conn
	return fixture
}

// Handle registers a handler that records the request before delegating.
func (c *r2CloudComputeCloud) Handle(pattern string, handler func(w http.ResponseWriter, r *http.Request, body string)) {
	c.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.calls = append(c.calls, r2CloudComputeCall{r.Method, r.URL.Path, r.URL.RawQuery, string(data)})
		c.mu.Unlock()
		handler(w, r, string(data))
	})
}

// JSON registers a fixed JSON response.
func (c *r2CloudComputeCloud) JSON(pattern string, status int, body string) {
	c.Handle(pattern, func(w http.ResponseWriter, _ *http.Request, _ string) { testcloud.JSON(w, status, body) })
}

const r2CloudComputeBase = "/compute/v2.1/p"
