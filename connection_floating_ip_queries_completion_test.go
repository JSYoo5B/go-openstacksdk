package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type floatingQueryClosureEvidence struct {
	pages                []*compute.FloatingIPQueryResponse
	failure              *compute.FloatingIPQueryResponse
	fallback, suppressed error
	value                json.RawMessage
	selected             int
}

// get-dict/search-dict consume the network Resource path; get-local/search-local
// consume unfiltered public List before ordinary identifier filtering.
func queryClosureCall(t *testing.T, conn *sdk.Connection, consumer string, options []compute.FloatingIPQueryOption) (floatingQueryClosureEvidence, error) {
	t.Helper()
	if strings.HasPrefix(consumer, "get") {
		result, err := conn.GetFloatingIP(context.Background(), compute.GetFloatingIPRequest{ID: "absent"}, options...)
		if result == nil {
			t.Fatalf("Get returned no operation evidence: %v", err)
		}
		selected := 0
		if result.FloatingIP != nil {
			selected = 1
		}
		return floatingQueryClosureEvidence{result.Pages, result.Failure, result.FallbackError, result.SuppressedNotFound, result.Value, selected}, err
	}
	var result *compute.FloatingIPQueryResult
	var err error
	if strings.HasPrefix(consumer, "search") {
		result, err = conn.SearchFloatingIPs(context.Background(), compute.SearchFloatingIPsRequest{ID: "absent"}, options...)
	} else {
		result, err = conn.ListFloatingIPs(context.Background(), options...)
	}
	if result == nil {
		t.Fatalf("query returned no operation evidence: %v", err)
	}
	return floatingQueryClosureEvidence{result.Pages, result.Failure, result.FallbackError, result.SuppressedNotFound, result.Value, len(result.FloatingIPs)}, err
}

func TestFloatingIPQueryNeutronRowsFailBeforeContinuation(t *testing.T) {
	for _, consumer := range []string{"list", "list-dict", "search-dict", "get-dict", "search-local", "get-local"} {
		for _, failure := range []string{"descriptor", "local-shape"} {
			if failure == "local-shape" && strings.HasSuffix(consumer, "local") {
				continue
			}
			t.Run(consumer+"/"+failure, func(t *testing.T) {
				_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
				row := `{"id":"physical","revision_number":"²"}`
				filters := `{}`
				if failure == "local-shape" {
					row = `{"id":"physical","dns_name":"present"}`
					filters = `{"dns_name":{"x":1}}`
				}
				firstBody := `{"floatingips":[` + row + `],"floatingips_links":[{"rel":"next","href":"?marker=second"}]}`
				state.reply = func(r *http.Request) (int, string) {
					if r.URL.Query().Get("marker") != "" {
						return 404, `{"missing":"next"}`
					}
					if strings.Contains(r.URL.Path, "compute") {
						return 200, `{"floating_ips":[{"id":"incorrect-fallback"}]}`
					}
					return 200, firstBody
				}
				var options []compute.FloatingIPQueryOption
				if failure == "local-shape" || strings.HasSuffix(consumer, "dict") {
					options = []compute.FloatingIPQueryOption{compute.WithFloatingIPQueryFilters(json.RawMessage(filters))}
				}
				got, err := queryClosureCall(t, conn, consumer, options)
				var proof *resource.ResponseError
				if err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != firstBody {
					t.Fatal(got, err, proof)
				}
				if got.value != nil || got.selected != 0 || got.fallback != nil || got.suppressed != nil || got.failure == nil || got.failure.StatusCode != 200 {
					t.Fatal(got, err)
				}
				if len(got.pages) != 1 || string(got.pages[0].Envelope) != firstBody || !reflect.DeepEqual(state.locators, []string{"network"}) || !reflect.DeepEqual(state.paths, []string{"/query-network/v2.0/floatingips"}) {
					t.Fatal(got, state)
				}
			})
		}
	}
}

func TestFloatingIPQueryNeutronEmptyNestedFilterUsesTruthinessBeforeShape(t *testing.T) {
	for _, consumer := range []string{"list", "search-dict", "get-dict"} {
		for name, value := range map[string]string{"string": `"present"`, "number": "7", "array": `["x"]`, "object": `{"x":1}`, "false": "false", "zero": "0", "empty-string": `""`, "null": "null", "empty-array": "[]", "empty-object": "{}"} {
			t.Run(consumer+"/"+name, func(t *testing.T) {
				_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
				state.reply = func(*http.Request) (int, string) {
					return 200, `{"floatingips":[{"id":"physical","dns_name":` + value + `}]}`
				}
				got, err := queryClosureCall(t, conn, consumer, []compute.FloatingIPQueryOption{compute.WithFloatingIPQueryFilters(json.RawMessage(`{"dns_name":{}}`))})
				truthy := name == "string" || name == "number" || name == "array" || name == "object"
				want := 0
				if truthy {
					want = 1
				}
				if err != nil || got.selected != want || got.failure != nil || got.fallback != nil || got.suppressed != nil || len(got.pages) != 1 || len(state.paths) != 1 || len(state.queries[0]) != 0 {
					t.Fatal(got, err, state)
				}
				if truthy && !strings.Contains(string(got.value), `"physical"`) {
					t.Fatal(got)
				}
				if !truthy && strings.HasPrefix(consumer, "get") && got.value != nil {
					t.Fatal(got)
				}
				if !truthy && !strings.HasPrefix(consumer, "get") && string(got.value) != "[]" {
					t.Fatal(got)
				}
			})
		}
	}
}
func TestFloatingIPQueryNeutronEagerRowsCacheLocationAndCountPhysicalRows(t *testing.T) {
	for _, mode := range []string{"continuation", "cap", "marker"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
			facade, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			state.locators = nil
			calls := 0
			direct := compute.New(facade.RawClient(), compute.Dependencies{
				AddressNetworks: conn.Network,
				CloudLocation:   func() (resource.CloudLocation, error) { calls++; return resource.CloudLocation{}, nil },
			})
			state.reply = func(r *http.Request) (int, string) {
				if r.URL.Query().Get("marker") != "" {
					return 200, `{"floatingips":[]}`
				}
				if mode == "marker" {
					return 200, `{"floatingips":[{"id":"a","dns_name":"keep"},{"id":"b","dns_name":"drop"}]}`
				}
				return 200, `{"floatingips":[{"id":"a","dns_name":"drop"},{"id":"b","dns_name":"keep"}],"floatingips_links":[{"rel":"next","href":"?marker=b"}]}`
			}
			filter := `{"dns_name":"keep"}`
			if mode == "cap" {
				filter = `{"dns_name":"keep","max_items":1}`
			}
			if mode == "marker" {
				filter = `{"dns_name":"keep","limit":2}`
			}
			result, err := direct.ListFloatingIPs(context.Background(), compute.WithFloatingIPQueryFilters(json.RawMessage(filter)))
			if err != nil {
				t.Fatal(result, err, state)
			}
			if mode == "cap" {
				if calls != 1 || len(state.paths) != 1 || len(result.FloatingIPs) != 0 || string(result.Value) != "[]" {
					t.Fatal(result, calls, state)
				}
				return
			}
			if calls != 2 || len(state.paths) != 2 || len(result.FloatingIPs) != 1 || state.queries[1].Get("marker") != "b" {
				t.Fatal(result, calls, state)
			}
			id := `"b"`
			if mode == "marker" {
				id = `"a"`
			}
			queryRaw(t, result.FloatingIPs[0].Resource, "id", id)
		})
	}
}
