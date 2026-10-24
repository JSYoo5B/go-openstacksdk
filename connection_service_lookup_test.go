package openstack_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

// TestServiceLookupFailureAsUnavailableMapsHasServiceFalse covers Python's
// has_service('network') returning False when the endpoint lookup raises: with
// the opt-in policy role discovery is empty (and retried on the next call)
// and the floating IP helpers take the Nova lane; by default the lookup error
// is returned, and cancellation is returned under either policy.
func TestServiceLookupFailureAsUnavailableMapsHasServiceFalse(t *testing.T) {
	sentinel := errors.New("catalog failed")
	newConn := func(t *testing.T, enabled bool, handle func(r2CloudNetCall) (int, string)) (*sdk.Connection, *atomic.Int32, *r2CloudNetFixture) {
		t.Helper()
		cloud := testcloud.New(t)
		var lookups atomic.Int32
		cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
			if opts.Type == "compute" {
				return cloud.Server.URL + "/compute/", nil
			}
			lookups.Add(1)
			return "", sentinel
		}
		f := &r2CloudNetFixture{handle: handle}
		cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
			var raw []byte
			if r.Body != nil {
				raw, _ = io.ReadAll(r.Body)
			}
			call := r2CloudNetCall{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, body: string(raw)}
			f.mu.Lock()
			f.calls = append(f.calls, call)
			f.mu.Unlock()
			code, body := 0, ""
			if handle != nil {
				code, body = handle(call)
			}
			if code == 0 {
				t.Errorf("unexpected request %s", call.key())
				return nil, errors.New("unexpected request")
			}
			return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		})
		conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.Compute, "2.35"), sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNeutron)), sdk.WithServiceLookupFailureAsUnavailable(enabled))
		if err != nil {
			t.Fatal(err)
		}
		return conn, &lookups, f
	}
	t.Run("roles", func(t *testing.T) {
		for _, enabled := range []bool{false, true} {
			conn, lookups, _ := newConn(t, enabled, nil)
			for i := 0; i < 2; i++ {
				roles, err := conn.GetNetworkRoles(context.Background())
				if enabled != (err == nil) || enabled && (roles == nil || roles.NATDestination != nil) || !enabled && !errors.Is(err, sentinel) {
					t.Fatal(enabled, roles, err)
				}
			}
			if lookups.Load() != 2 {
				t.Fatal(enabled, lookups.Load())
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := conn.GetNATDestination(ctx); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		}
	})
	t.Run("floating IP helper uses Nova", func(t *testing.T) {
		attached := false
		body := func() string {
			return `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"198.51.100.21","OS-EXT-IPS:type":"floating"}]}}}`
		}
		conn, lookups, f := newConn(t, true, r2CloudNetNovaAutoHandler(&attached, body))
		result, err := conn.AddIPList(context.Background(), &compute.Server{ID: "srv-1", Status: "ACTIVE"}, []string{"198.51.100.21"})
		if err != nil || result.Decision == nil || result.Decision.Backend != compute.FloatingIPNova || f.count("POST /compute/servers/srv-1/action") != 1 || lookups.Load() == 0 {
			t.Fatal(result, err, f.keys())
		}
		conn, _, f = newConn(t, false, r2CloudNetNovaAutoHandler(&attached, body))
		if _, err := conn.AddIPList(context.Background(), &compute.Server{ID: "srv-1", Status: "ACTIVE"}, []string{"198.51.100.21"}); !errors.Is(err, sentinel) || f.count("POST /compute/servers/srv-1/action") != 0 {
			t.Fatal(err, f.keys())
		}
	})
}
