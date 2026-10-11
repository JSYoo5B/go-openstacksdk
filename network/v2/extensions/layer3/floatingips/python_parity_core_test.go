package floatingips_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
)

type pythonParityCoreFloatingIPTransport func(*http.Request) (*http.Response, error)

func (transport pythonParityCoreFloatingIPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonParityCoreFloatingIPCall struct{ method, path, query, body string }

func pythonParityCoreFloatingIPAPI(t *testing.T, calls *[]pythonParityCoreFloatingIPCall, reply func(*http.Request) string) (*floatingips.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonParityCoreFloatingIPTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonParityCoreFloatingIPCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(reply(req))), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return floatingips.New(client), cloud
}

const pythonParityCoreFloatingIPBase = "/neutron/v2.0/floatingips"

// Python add_ip_to_port sets port_id on a fetched FloatingIP and commits only
// that dirty field; remove_ip_from_port commits port_id=None. Go Update sends
// the same PUT bodies, turning an empty PortID into null.
func TestPythonParityCoreFloatingIPPortAssociation(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreFloatingIPCall
	api, _ := pythonParityCoreFloatingIPAPI(t, &calls, func(req *http.Request) string {
		if strings.Contains(pythonParityCoreFloatingIPLastBody(calls), `"p-1"`) {
			return `{"floatingip":{"id":"fip-1","floating_ip_address":"203.0.113.5","port_id":"p-1","status":"DOWN"}}`
		}
		return `{"floatingip":{"id":"fip-1","floating_ip_address":"203.0.113.5","port_id":null,"status":"DOWN"}}`
	})
	portID, none := "p-1", ""
	attached, err := api.Update(ctx, "fip-1", floatingips.UpdateOpts{PortID: &portID})
	if err != nil || attached.PortID != "p-1" {
		t.Fatal(attached, err)
	}
	detached, err := api.Update(ctx, "fip-1", floatingips.UpdateOpts{PortID: &none})
	if err != nil || detached.PortID != "" {
		t.Fatal(detached, err)
	}
	want := []pythonParityCoreFloatingIPCall{
		{http.MethodPut, pythonParityCoreFloatingIPBase + "/fip-1", "", `{"floatingip":{"port_id":"p-1"}}`},
		{http.MethodPut, pythonParityCoreFloatingIPBase + "/fip-1", "", `{"floatingip":{"port_id":null}}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func pythonParityCoreFloatingIPLastBody(calls []pythonParityCoreFloatingIPCall) string {
	if len(calls) == 0 {
		return ""
	}
	return calls[len(calls)-1].body
}

// Python find_available_ip iterates GET floatingips without a query and
// returns the first row whose port_id is empty, stopping the generator there.
// The lazy Go List stops fetching pages when the caller breaks at that row.
func TestPythonParityCoreFloatingIPFindAvailableStopsAtFirstFreeRow(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		second string
		want   string
		calls  int
	}{
		{"free row on first page", "", "fip-2", 1},
		{"free row on second page", `{"id":"fip-3","port_id":null}`, "fip-3", 2},
		{"no free row", `{"id":"fip-3","port_id":"p-3"}`, "", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonParityCoreFloatingIPCall
			var cloud *testcloud.Cloud
			api, cloud := pythonParityCoreFloatingIPAPI(t, &calls, func(req *http.Request) string {
				if req.URL.Query().Get("marker") != "" {
					return `{"floatingips":[` + tc.second + `]}`
				}
				free := `{"id":"fip-2","port_id":""}`
				if tc.second != "" {
					free = `{"id":"fip-2","port_id":"p-2"}`
				}
				return `{"floatingips":[{"id":"fip-1","port_id":"p-1"},` + free + `],"floatingips_links":[{"rel":"next","href":"` + cloud.Server.URL + pythonParityCoreFloatingIPBase + `?marker=fip-2"}]}`
			})
			var found *floatingips.FloatingIP
			for ip, err := range api.List(ctx) {
				if err != nil {
					t.Fatal(err)
				}
				if ip.PortID == "" {
					found = ip
					break
				}
			}
			got := ""
			if found != nil {
				got = found.ID
			}
			if got != tc.want || len(calls) != tc.calls || calls[0] != (pythonParityCoreFloatingIPCall{http.MethodGet, pythonParityCoreFloatingIPBase, "", ""}) {
				t.Fatalf("%q %+v", got, calls)
			}
		})
	}
}
