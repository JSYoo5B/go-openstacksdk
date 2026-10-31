package ports_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonParityCorePortCall struct{ method, path, query string }

func pythonParityCorePortAPI(t *testing.T, calls *[]pythonParityCorePortCall, reply func(*http.Request) (int, string)) (*ports.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pythonParityCorePortCall{req.Method, req.URL.Path, req.URL.RawQuery})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return ports.New(client), cloud
}

const pythonParityCorePortBase = "/neutron/v2.0/ports"

// Python ports(**query) transposes is_admin_state_up and security_group_ids,
// repeats fixed_ips and security_groups, and follows ports_links. Go sends the
// same wire keys through ListOpts and WithListQuery.
func TestPythonParityCorePortListQueryAndLinks(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCorePortCall
	var cloud *testcloud.Cloud
	api, cloud := pythonParityCorePortAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"ports":[{"id":"p-1","network_id":"net-1","status":"ACTIVE","fixed_ips":[{"subnet_id":"sub-1","ip_address":"10.0.0.5"}],"security_groups":["sg-1"]}],"ports_links":[{"rel":"next","href":"` + cloud.Server.URL + pythonParityCorePortBase + `?marker=p-1"}]}`
		}
		return 200, `{"ports":[{"id":"p-2","network_id":"net-1","status":"DOWN"}]}`
	})
	up := true
	var ids []string
	for value, err := range api.List(ctx,
		ports.WithListOptions(ports.ListOpts{
			NetworkID:      "net-1",
			AdminStateUp:   &up,
			SecurityGroups: []string{"sg-1", "sg-2"},
			FixedIPs:       []ports.FixedIPOpts{{SubnetID: "sub-1"}, {IPAddress: "10.0.0.5"}},
		}),
		ports.WithListQuery("binding:host_id", "compute-1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"p-1", "p-2"}) || len(calls) != 2 || calls[1].query != "marker=p-1" {
		t.Fatalf("%v %+v", ids, calls)
	}
	query, _ := url.ParseQuery(calls[0].query)
	want := url.Values{
		"network_id":      {"net-1"},
		"admin_state_up":  {"true"},
		"security_groups": {"sg-1", "sg-2"},
		"fixed_ips":       {"subnet_id=sub-1", "ip_address=10.0.0.5"},
		"binding:host_id": {"compute-1"},
	}
	if !reflect.DeepEqual(query, want) {
		t.Fatalf("%v", query)
	}
}

// Python get_subnet_ports lists every port without a query and keeps those
// with a fixed IP on the subnet. Go sends the same unfiltered list and the
// caller keeps the same rows from FixedIPs.
func TestPythonParityCorePortSubnetPortsSelection(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCorePortCall
	api, _ := pythonParityCorePortAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"ports":[` +
			`{"id":"p-1","fixed_ips":[{"subnet_id":"sub-1","ip_address":"10.0.0.5"}]},` +
			`{"id":"p-2","fixed_ips":[{"subnet_id":"sub-2","ip_address":"10.1.0.5"}]},` +
			`{"id":"p-3","fixed_ips":[{"subnet_id":"sub-2","ip_address":"10.1.0.6"},{"subnet_id":"sub-1","ip_address":"10.0.0.6"}]}]}`
	})
	all, err := api.All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var selected []string
	for _, port := range all {
		if slices.ContainsFunc(port.FixedIPs, func(ip ports.IP) bool { return ip.SubnetID == "sub-1" }) {
			selected = append(selected, port.ID)
		}
	}
	if !reflect.DeepEqual(selected, []string{"p-1", "p-3"}) || !reflect.DeepEqual(calls, []pythonParityCorePortCall{{http.MethodGet, pythonParityCorePortBase, ""}}) {
		t.Fatalf("%v %+v", selected, calls)
	}
}

// Python wait_for_status polls GET on the resource until the status matches
// case-insensitively, fails on failures (default ERROR) and times out after
// wait. wait_for_delete polls until 404. Go offers the same polling on the
// collections that have WaitFor and WaitForDeletion.
func TestPythonParityCorePortWaiters(t *testing.T) {
	ctx := context.Background()
	t.Run("status reaches target", func(t *testing.T) {
		var calls []pythonParityCorePortCall
		var polls atomic.Int32
		api, _ := pythonParityCorePortAPI(t, &calls, func(*http.Request) (int, string) {
			if polls.Add(1) < 3 {
				return 200, `{"port":{"id":"p-1","status":"BUILD"}}`
			}
			return 200, `{"port":{"id":"p-1","status":"active"}}`
		})
		var progress []int
		got, err := api.WaitFor(ctx, resource.ID("p-1"), "ACTIVE",
			resource.WithFailureStates("ERROR"), resource.WithUnlimitedWait(),
			resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(p int) { progress = append(progress, p) }))
		if err != nil || got.Status != "active" || len(calls) != 3 || !reflect.DeepEqual(progress, []int{0, 0}) {
			t.Fatal(got, err, calls, progress)
		}
		for _, call := range calls {
			if call != (pythonParityCorePortCall{http.MethodGet, pythonParityCorePortBase + "/p-1", ""}) {
				t.Fatalf("%+v", calls)
			}
		}
	})
	t.Run("failure state", func(t *testing.T) {
		var calls []pythonParityCorePortCall
		api, _ := pythonParityCorePortAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"port":{"id":"p-1","status":"ERROR"}}`
		})
		_, err := api.WaitFor(ctx, resource.ID("p-1"), "ACTIVE", resource.WithFailureStates("ERROR"), resource.WithPollInterval(time.Millisecond))
		if !errors.Is(err, resource.ErrFailedState) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
	t.Run("timeout", func(t *testing.T) {
		var calls []pythonParityCorePortCall
		api, _ := pythonParityCorePortAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"port":{"id":"p-1","status":"DOWN"}}`
		})
		_, err := api.WaitFor(ctx, resource.ID("p-1"), "ACTIVE", resource.WithTimeout(20*time.Millisecond), resource.WithPollInterval(time.Millisecond))
		if !errors.Is(err, context.DeadlineExceeded) || len(calls) < 2 {
			t.Fatal(err, len(calls))
		}
	})
	t.Run("deletion", func(t *testing.T) {
		var calls []pythonParityCorePortCall
		var polls atomic.Int32
		api, _ := pythonParityCorePortAPI(t, &calls, func(*http.Request) (int, string) {
			if polls.Add(1) < 3 {
				return 200, `{"port":{"id":"p-1","status":"ACTIVE"}}`
			}
			return 404, `{"NeutronError":{"type":"PortNotFound"}}`
		})
		err := api.WaitForDeletion(ctx, resource.ID("p-1"), resource.WithTimeout(120*time.Second), resource.WithPollInterval(time.Millisecond))
		if err != nil || len(calls) != 3 || calls[2] != (pythonParityCorePortCall{http.MethodGet, pythonParityCorePortBase + "/p-1", ""}) {
			t.Fatal(err, calls)
		}
	})
}
