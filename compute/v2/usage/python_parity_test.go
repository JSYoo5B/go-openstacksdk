package usage_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/usage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// pythonUsageCall records the wire request openstacksdk would also send.
type pythonUsageCall struct{ method, path, query, version string }

type pythonUsageTransport func(*http.Request) (*http.Response, error)

func (transport pythonUsageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonUsageWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

const pythonUsageRow = `{"tenant_id":"p1","total_hours":1.5,"total_local_gb_usage":15,"total_memory_mb_usage":768,"total_vcpus_usage":3,"start":"2026-10-01T00:00:00.000000","stop":"2026-10-10T00:00:00","server_usages":[{"instance_id":"s1","name":"vm","flavor":"m1","vcpus":2,"memory_mb":512,"local_gb":10,"hours":1.5,"state":"active","uptime":5400,"tenant_id":"p1","started_at":"2026-10-01T01:00:00.000000","ended_at":null}]}`

func pythonUsageAPI(t *testing.T, calls *[]pythonUsageCall, cloud *testcloud.Cloud, reply func(*http.Request) *http.Response) *usage.API {
	t.Helper()
	cloud.Provider.HTTPClient.Transport = pythonUsageTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pythonUsageCall{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("X-OpenStack-Nova-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.75"
	return usage.New(client)
}

func pythonUsageCheck(t *testing.T, value *usage.TenantUsage) {
	t.Helper()
	if value.TenantID != "p1" || value.TotalHours != 1.5 || value.TotalLocalGBUsage != 15 || value.TotalMemoryMBUsage != 768 || value.TotalVCPUsUsage != 3 || value.Start.IsZero() || value.Stop.IsZero() || len(value.ServerUsages) != 1 {
		t.Fatal(value)
	}
	server := value.ServerUsages[0]
	if server.InstanceID != "s1" || server.Name != "vm" || server.Flavor != "m1" || server.VCPUs != 2 || server.MemoryMB != 512 || server.LocalGB != 10 || server.Hours != 1.5 || server.State != "active" || server.Uptime != 5400 || server.TenantID != "p1" || server.StartedAt.IsZero() || !server.EndedAt.IsZero() {
		t.Fatal(server)
	}
}

func TestPythonUsageAllTenantsMatchesProxyQuery(t *testing.T) {
	ctx := context.Background()
	var calls []pythonUsageCall
	cloud := testcloud.New(t)
	api := pythonUsageAPI(t, &calls, cloud, func(*http.Request) *http.Response {
		return pythonUsageWire(200, `{"tenant_usages":[`+pythonUsageRow+`]}`)
	})
	// usages() and usages(start=datetime(2026, 10, 1), end=datetime(2026, 10, 10), detailed=1)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	for _, opts := range []usage.AllTenantsOpts{{}, {Start: &start, End: &end, Detailed: true}} {
		var rows []*usage.TenantUsage
		for value, err := range api.AllTenants(ctx, opts) {
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, value)
		}
		if len(rows) != 1 {
			t.Fatal(rows)
		}
		pythonUsageCheck(t, rows[0])
	}
	want := []pythonUsageCall{
		{http.MethodGet, "/nova/v2.1/os-simple-tenant-usage", "", "2.75"},
		{http.MethodGet, "/nova/v2.1/os-simple-tenant-usage", "detailed=1&end=2026-10-10T00%3A00%3A00&start=2026-10-01T00%3A00%3A00", "2.75"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestPythonUsageSingleProjectFirstPageMatchesFetch(t *testing.T) {
	ctx := context.Background()
	var calls []pythonUsageCall
	cloud := testcloud.New(t)
	api := pythonUsageAPI(t, &calls, cloud, func(*http.Request) *http.Response {
		return pythonUsageWire(200, `{"tenant_usage":`+pythonUsageRow+`,"tenant_usage_links":[{"rel":"next","href":"`+cloud.Server.URL+`/nova/v2.1/os-simple-tenant-usage/p1?marker=s1"}]}`)
	})
	// get_usage("p1", start=..., end=...) fetches once; stopping after the first
	// value keeps Go from following the next link.
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	var first *usage.TenantUsage
	for value, err := range api.SingleTenant(ctx, "p1", usage.SingleTenantOpts{Start: &start, End: &end}) {
		if err != nil {
			t.Fatal(err)
		}
		first = value
		break
	}
	pythonUsageCheck(t, first)
	want := []pythonUsageCall{
		{http.MethodGet, "/nova/v2.1/os-simple-tenant-usage/p1", "end=2026-10-10T00%3A00%3A00&start=2026-10-01T00%3A00%3A00", "2.75"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
