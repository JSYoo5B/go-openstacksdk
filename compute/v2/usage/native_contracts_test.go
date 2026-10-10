package usage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/usage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeUsageTransport func(*http.Request) (*http.Response, error)

func (transport nativeUsageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeUsageWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeUsageOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "usage" {
		t.Fatal("generated usage context", err, wrapped)
	}
}

type nativeUsageCall struct{ method, path, query string }

func nativeUsageRecorder(cloud *testcloud.Cloud, calls *[]nativeUsageCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeUsageTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeUsageCall{req.Method, req.URL.Path, req.URL.RawQuery})
		return reply(req), nil
	})
}

func nativeUsageStatus(t *testing.T, err error, operation string, code int, expected []int) {
	t.Helper()
	nativeUsageOperation(t, err, operation)
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, expected) {
		t.Fatal(err, native)
	}
}

func nativeUsagePagerStatus(t *testing.T, errs []error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if len(errs) != 1 || !errors.As(errs[0], &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200, 204, 300}) {
		t.Fatal(errs)
	}
}

func nativeUsageClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return client
}

func TestNativeSingleTenantUsageQueryPagingAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeUsageCall
	nativeUsageRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "" {
			return nativeUsageWire(200, `{"tenant_usage":{"tenant_id":"p1","total_hours":1.5,"start":"2026-10-01T00:00:00.000000","stop":"2026-10-10T00:00:00","server_usages":[{"instance_id":"s1","flavor":"m1","vcpus":2,"ended_at":null,"started_at":"2026-10-01T01:00:00.000000"}]},"tenant_usage_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/os-simple-tenant-usage/p1?marker=s1"}]}`)
		}
		return nativeUsageWire(200, `{"tenant_usage":{}}`)
	})
	api := usage.New(nativeUsageClient(cloud))
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.FixedZone("KST", 9*3600))
	end := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	var pages []*usage.TenantUsage
	for value, err := range api.SingleTenant(context.Background(), "p1", usage.SingleTenantOpts{Start: &start, End: &end, Limit: 1}, usage.WithSingleTenantQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, value)
	}
	first := pages[0]
	if len(pages) != 2 || first.TenantID != "p1" || first.TotalHours != 1.5 || len(first.ServerUsages) != 1 || !first.ServerUsages[0].EndedAt.IsZero() || !first.Stop.Equal(end) || pages[1].TenantID != "" {
		t.Fatal(pages)
	}
	// start/end are formatted as wall-clock time without the caller's offset.
	want := []nativeUsageCall{
		{http.MethodGet, "/nova/v2.1/os-simple-tenant-usage/p1", "end=2026-10-10T00%3A00%3A00&extra=1&limit=1&start=2026-10-01T09%3A00%3A00"},
		{http.MethodGet, "/other/os-simple-tenant-usage/p1", "marker=s1"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeSingleTenantUsageStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"other":{}}`}, {204, ""}} {
		cloud := testcloud.New(t)
		var calls []nativeUsageCall
		nativeUsageRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeUsageWire(tc.code, tc.body) })
		var errs []error
		for _, err := range usage.New(nativeUsageClient(cloud)).SingleTenant(ctx, "p1", usage.SingleTenantOpts{}) {
			errs = append(errs, err)
		}
		switch tc.code {
		case 404:
			nativeUsagePagerStatus(t, errs, 404)
		case 200:
			// A response without tenant_usage is an empty page.
			if len(errs) != 0 {
				t.Fatal(errs)
			}
		case 204:
			if len(errs) != 1 || !errors.Is(errs[0], io.EOF) {
				t.Fatal(errs)
			}
		}
	}
	cloud := testcloud.New(t)
	var calls []nativeUsageCall
	nativeUsageRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeUsageWire(200, `{}`) })
	for _, err := range usage.New(nativeUsageClient(cloud)).SingleTenant(ctx, "p1", usage.SingleTenantOpts{}, nil) {
		nativeUsageOperation(t, err, "SingleTenant")
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}
