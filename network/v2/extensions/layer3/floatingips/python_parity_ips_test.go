package floatingips_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonIPsCall struct{ method, path, query, accept string }

func pythonIPsAPI(t *testing.T, calls *[]pythonIPsCall, reply func(*http.Request) (int, string)) *floatingips.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeFloatingIPTransport(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			_, _ = io.Copy(io.Discard, req.Body)
		}
		*calls = append(*calls, pythonIPsCall{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("Accept")})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}, Request: req}, nil
	})
	return floatingips.New(nativeFloatingIPClient(cloud))
}

func TestPythonIPsFindIPOnlyReproducesTheIDLookup(t *testing.T) {
	ctx := context.Background()
	var calls []pythonIPsCall
	api := pythonIPsAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == "/neutron/v2.0/floatingips/id-1" {
			return 200, `{"floatingip":` + nativeFloatingIPRow + `}`
		}
		return 404, `{"NeutronError":{"type":"FloatingIPNotFound","message":"missing"}}`
	})

	// find_ip(id) succeeds with the first GET /floatingips/{id}.
	got, err := api.Find(ctx, resource.ID("id-1"), resource.WithIgnoreMissing())
	if err != nil || got == nil || got.ID != "id-1" || got.FloatingIP != "203.0.113.5" || got.FloatingNetworkID != "ext" || got.Status != "ACTIVE" {
		t.Fatal(got, err)
	}
	// Explicit Ref ID lookup stops after this 404; FindIdentity continues.
	missing, err := api.Find(ctx, resource.ID("203.0.113.5"), resource.WithIgnoreMissing())
	if err != nil || missing != nil {
		t.Fatal(missing, err)
	}
	if _, err := api.Find(ctx, resource.ID("203.0.113.5")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	want := []pythonIPsCall{
		{http.MethodGet, "/neutron/v2.0/floatingips/id-1", "", "application/json"},
		{http.MethodGet, "/neutron/v2.0/floatingips/203.0.113.5", "", "application/json"},
		{http.MethodGet, "/neutron/v2.0/floatingips/203.0.113.5", "", "application/json"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}

	// Explicit Ref name lookup stays unsupported; API.FindIdentity owns the
	// floating_ip_address fallback (python_find_identity_test.go).
	calls = nil
	if _, err := api.Find(ctx, resource.Name("203.0.113.5"), resource.WithIgnoreMissing()); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("unsupported lookups sent %+v", calls)
	}
}

func TestPythonIPsListServerQueryAndPagesButNoLocalBodyFilters(t *testing.T) {
	ctx := context.Background()
	var calls []pythonIPsCall
	api := pythonIPsAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"floatingips":[` + nativeFloatingIPRow + `],"floatingips_links":[{"rel":"next","href":"` + "http://" + req.Host + `/neutron/v2.0/floatingips?marker=id-1"}]}`
		}
		return 200, `{"floatingips":[{"id":"id-2","floating_ip_address":"203.0.113.6","status":"DOWN"}],"floatingips_links":[]}`
	})

	// ips() sends no query and follows floatingips_links.
	rows, err := api.All(ctx)
	if err != nil || len(rows) != 2 || rows[0].FloatingIP != "203.0.113.5" || rows[1].ID != "id-2" || rows[1].Status != "DOWN" {
		t.Fatal(rows, err)
	}
	// ips(floating_network_id=..., status=...) maps both keys to the server query.
	if _, err := api.All(ctx, resource.WithQuery("floating_network_id", "ext"), resource.WithQuery("status", "ACTIVE")); err != nil {
		t.Fatal(err)
	}
	want := []pythonIPsCall{
		{http.MethodGet, "/neutron/v2.0/floatingips", "", "application/json"},
		{http.MethodGet, "/neutron/v2.0/floatingips", "marker=id-1", "application/json"},
		{http.MethodGet, "/neutron/v2.0/floatingips", "floating_network_id=ext&status=ACTIVE", "application/json"},
		{http.MethodGet, "/neutron/v2.0/floatingips", "marker=id-1", "application/json"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}

	// Python filters declared non-query Body attributes such as name
	// (floating_ip_address) and dns_name locally; this collection rejects both.
	calls = nil
	if _, err := api.All(ctx, resource.WithName("203.0.113.5")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := api.All(ctx, resource.WithBodyFilter("dns_name", "web")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("unsupported filters sent %+v", calls)
	}
}
