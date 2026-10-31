package networkipavailabilities_test

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities"
)

type pythonParityCoreIPAvailabilityCall struct{ method, path, query string }

func pythonParityCoreIPAvailabilityAPI(t *testing.T, calls *[]pythonParityCoreIPAvailabilityCall, reply func(*http.Request) string) (*networkipavailabilities.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pythonParityCoreIPAvailabilityCall{req.Method, req.URL.Path, req.URL.RawQuery})
		return testcloud.Response(200, reply(req), http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return networkipavailabilities.New(client), cloud
}

const pythonParityCoreIPAvailabilityRow = `{"network_id":"net-1","network_name":"private","project_id":"p-1","tenant_id":"p-1","total_ips":253,"used_ips":3,"subnet_ip_availability":[{"subnet_id":"sub-1","subnet_name":"s","cidr":"10.0.0.0/24","ip_version":4,"total_ips":253,"used_ips":3}]}`

// Python network_ip_availabilities(**query) sends the QueryParameters keys to
// GET network-ip-availabilities. The native page is single, so Go does not
// follow network_ip_availabilities_links; the caller sends the next marker.
// get_network_ip_availability without fields sends a bare member GET.
func TestPythonParityCoreIPAvailabilityListAndGet(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreIPAvailabilityCall
	var cloud *testcloud.Cloud
	api, cloud := pythonParityCoreIPAvailabilityAPI(t, &calls, func(req *http.Request) string {
		if strings.HasSuffix(req.URL.Path, "/net-1") {
			return `{"network_ip_availability":` + pythonParityCoreIPAvailabilityRow + `}`
		}
		return `{"network_ip_availabilities":[` + pythonParityCoreIPAvailabilityRow + `],"network_ip_availabilities_links":[{"rel":"next","href":"` + cloud.Server.URL + `/neutron/v2.0/network-ip-availabilities?marker=net-1"}]}`
	})
	var rows []*networkipavailabilities.NetworkIPAvailability
	for value, err := range api.List(ctx,
		networkipavailabilities.WithListOptions(networkipavailabilities.ListOpts{NetworkName: "private", IPVersion: "4", ProjectID: "p-1"}),
		networkipavailabilities.WithListQuery("sort_key", "network_name"),
		networkipavailabilities.WithListQuery("fields", "network_id")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if len(rows) != 1 || rows[0].NetworkID != "net-1" || rows[0].TotalIPs != "253" || rows[0].UsedIPs != "3" || len(rows[0].SubnetIPAvailabilities) != 1 || rows[0].SubnetIPAvailabilities[0].CIDR != "10.0.0.0/24" {
		t.Fatalf("%+v", rows)
	}
	query, _ := url.ParseQuery(calls[0].query)
	if len(calls) != 1 || calls[0].path != "/neutron/v2.0/network-ip-availabilities" || !reflect.DeepEqual(query, url.Values{"network_name": {"private"}, "ip_version": {"4"}, "project_id": {"p-1"}, "sort_key": {"network_name"}, "fields": {"network_id"}}) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	got, err := api.Get(ctx, "net-1")
	if err != nil || got.NetworkName != "private" || !reflect.DeepEqual(calls, []pythonParityCoreIPAvailabilityCall{{http.MethodGet, "/neutron/v2.0/network-ip-availabilities/net-1", ""}}) {
		t.Fatalf("%+v %v %v", calls, got, err)
	}
}
