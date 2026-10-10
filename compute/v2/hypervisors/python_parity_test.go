package hypervisors_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonHypervisorCall records the wire request openstacksdk would also send.
type pythonHypervisorCall struct{ method, path, query, version string }

type pythonHypervisorTransport func(*http.Request) (*http.Response, error)

func (transport pythonHypervisorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonHypervisorWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonHypervisorAPI(t *testing.T, microversion string, reply func(*http.Request) *http.Response) (*hypervisors.API, *[]pythonHypervisorCall) {
	t.Helper()
	calls := &[]pythonHypervisorCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonHypervisorTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pythonHypervisorCall{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("X-OpenStack-Nova-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = microversion
	return hypervisors.New(client), calls
}

const pythonHypervisorBase = "/nova/v2.1/os-hypervisors"

const pythonHypervisorRow = `{"id":"hv-uuid","hypervisor_hostname":"cmp1","hypervisor_type":"QEMU","hypervisor_version":8002000,"host_ip":"192.0.2.10","state":"up","status":"enabled","uptime":" 10:00:00 up 1 day","service":{"id":"svc-uuid","host":"cmp1","disabled_reason":null}}`

func TestPythonHypervisorGetAndUptimeMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonHypervisorAPI(t, "2.88", func(*http.Request) *http.Response {
		return pythonHypervisorWire(200, `{"hypervisor":`+pythonHypervisorRow+`}`)
	})
	// get_hypervisor("hv-uuid")
	got, err := api.Get(ctx, "hv-uuid")
	if err != nil || got.ID != "hv-uuid" || got.HypervisorHostname != "cmp1" || got.HypervisorType != "QEMU" || got.HypervisorVersion != 8002000 || got.HostIP != "192.0.2.10" || got.State != "up" || got.Status != "enabled" || got.Service.ID != "svc-uuid" || got.Service.Host != "cmp1" {
		t.Fatal(got, err)
	}
	// get_hypervisor_uptime only sends a request below 2.88.
	older, olderCalls := pythonHypervisorAPI(t, "2.87", func(*http.Request) *http.Response {
		return pythonHypervisorWire(200, `{"hypervisor":{"id":"hv-uuid","hypervisor_hostname":"cmp1","state":"up","status":"enabled","uptime":" 10:00:00 up 1 day"}}`)
	})
	uptime, err := older.GetUptime(ctx, "hv-uuid")
	if err != nil || uptime.ID != "hv-uuid" || uptime.HypervisorHostname != "cmp1" || uptime.Uptime != " 10:00:00 up 1 day" || uptime.State != "up" || uptime.Status != "enabled" {
		t.Fatal(uptime, err)
	}
	// Go does not refuse the call at 2.88 like Python; Nova answers 404 there.
	failing, _ := pythonHypervisorAPI(t, "2.88", func(*http.Request) *http.Response {
		return pythonHypervisorWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	if _, err := failing.GetUptime(ctx, "hv-uuid"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	if want := []pythonHypervisorCall{{http.MethodGet, pythonHypervisorBase + "/hv-uuid", "", "2.88"}}; !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
	if want := []pythonHypervisorCall{{http.MethodGet, pythonHypervisorBase + "/hv-uuid/uptime", "", "2.87"}}; !reflect.DeepEqual(*olderCalls, want) {
		t.Fatalf("%+v", *olderCalls)
	}
}

func TestPythonHypervisorListAlwaysUsesDetailRoute(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonHypervisorAPI(t, "2.53", func(*http.Request) *http.Response {
		return pythonHypervisorWire(200, `{"hypervisors":[`+pythonHypervisorRow+`]}`)
	})
	// hypervisors() lists os-hypervisors without /detail; Go has only the detail list.
	// hypervisors(details=True, hypervisor_hostname_pattern="cmp", with_servers=True)
	// at 2.53 sends both keys to /detail.
	pattern, withServers := "cmp", true
	for _, options := range [][]hypervisors.ListOption{
		nil,
		{hypervisors.WithListOptions(hypervisors.ListOpts{HypervisorHostnamePattern: &pattern, WithServers: &withServers})},
	} {
		var names []string
		for value, err := range api.List(ctx, options...) {
			if err != nil {
				t.Fatal(err)
			}
			names = append(names, value.HypervisorHostname)
		}
		if !reflect.DeepEqual(names, []string{"cmp1"}) {
			t.Fatal(names)
		}
	}
	want := []pythonHypervisorCall{
		{http.MethodGet, pythonHypervisorBase + "/detail", "", "2.53"},
		{http.MethodGet, pythonHypervisorBase + "/detail", "hypervisor_hostname_pattern=cmp&with_servers=true", "2.53"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonHypervisorFindHasNoNameFallback(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonHypervisorAPI(t, "2.88", func(req *http.Request) *http.Response {
		if req.URL.Path == pythonHypervisorBase+"/hv-uuid" {
			return pythonHypervisorWire(200, `{"hypervisor":`+pythonHypervisorRow+`}`)
		}
		return pythonHypervisorWire(404, `{"itemNotFound":{"message":"missing"}}`)
	})
	if got, err := api.Find(ctx, resource.ID("hv-uuid")); err != nil || got.HypervisorHostname != "cmp1" {
		t.Fatal(got, err)
	}
	// find_hypervisor("cmp1") falls back to the detail list after a 404;
	// Find by ID stops at the 404 and the collection has no name binding.
	if got, err := api.Find(ctx, resource.ID("cmp1"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if _, err := api.Find(ctx, resource.Name("cmp1")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	want := []pythonHypervisorCall{
		{http.MethodGet, pythonHypervisorBase + "/hv-uuid", "", "2.88"},
		{http.MethodGet, pythonHypervisorBase + "/cmp1", "", "2.88"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
