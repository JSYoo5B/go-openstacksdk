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
)

func TestNativeAllTenantsUsageQueryPagingAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeUsageCall
	nativeUsageRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == "/nova/v2.1/os-simple-tenant-usage" {
			return nativeUsageWire(200, `{"tenant_usages":[{"tenant_id":"p1","total_hours":2,"total_vcpus_usage":4.5,"start":"2026-10-01T00:00:00.000000","stop":"2026-10-10T00:00:00","server_usages":[{"instance_id":"s1","hours":2,"vcpus":2,"started_at":"2026-10-01T01:00:00","ended_at":null}]}],"tenant_usages_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/os-simple-tenant-usage?marker=s1"}]}`)
		}
		return nativeUsageWire(200, `{"tenant_usages":[{"tenant_id":"p2"}]}`)
	})
	api := usage.New(nativeUsageClient(cloud))
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.FixedZone("KST", 9*3600))
	end := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	var rows []*usage.TenantUsage
	for value, err := range api.AllTenants(context.Background(), usage.AllTenantsOpts{Detailed: true, Start: &start, End: &end, Limit: 1}, usage.WithAllTenantsQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if len(rows) != 2 || rows[0].TenantID != "p1" || rows[0].TotalVCPUsUsage != 4.5 || !rows[0].Stop.Equal(end) || len(rows[0].ServerUsages) != 1 || !rows[0].ServerUsages[0].EndedAt.IsZero() || rows[1].TenantID != "p2" {
		t.Fatal(rows)
	}
	// Detailed=false sends nothing; true sends detailed=1. start/end lose the caller's offset.
	if _, err := collectNativeAllTenants(api.AllTenants(context.Background(), usage.AllTenantsOpts{})); err != nil {
		t.Fatal(err)
	}
	want := []nativeUsageCall{
		{http.MethodGet, "/nova/v2.1/os-simple-tenant-usage", "detailed=1&end=2026-10-10T00%3A00%3A00&extra=1&limit=1&start=2026-10-01T09%3A00%3A00"},
		{http.MethodGet, "/other/os-simple-tenant-usage", "marker=s1"},
		{http.MethodGet, "/nova/v2.1/os-simple-tenant-usage", ""},
		{http.MethodGet, "/other/os-simple-tenant-usage", "marker=s1"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func collectNativeAllTenants(stream func(func(*usage.TenantUsage, error) bool)) ([]*usage.TenantUsage, error) {
	var rows []*usage.TenantUsage
	for value, err := range stream {
		if err != nil {
			return rows, err
		}
		rows = append(rows, value)
	}
	return rows, nil
}

func TestNativeAllTenantsUsageStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"tenant_usages":[]}`}, {200, `{}`}, {200, `{"tenant_usages":[{"start":"2026-10-01T00:00:00Z"}]}`}, {204, ""}} {
		cloud := testcloud.New(t)
		var calls []nativeUsageCall
		nativeUsageRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeUsageWire(tc.code, tc.body) })
		var errs []error
		for _, err := range usage.New(nativeUsageClient(cloud)).AllTenants(ctx, usage.AllTenantsOpts{}) {
			errs = append(errs, err)
		}
		var wrapped *resource.OperationError
		switch {
		case tc.code == 404:
			nativeUsagePagerStatus(t, errs, 404)
		case tc.code == 204:
			if len(errs) != 1 || !errors.Is(errs[0], io.EOF) {
				t.Fatal(errs)
			}
		case strings.Contains(tc.body, "Z\""):
			// A zoned timestamp fails the page decode without operation context.
			if len(errs) != 1 || errs[0] == nil || errors.As(errs[0], &wrapped) {
				t.Fatal(errs)
			}
		case len(errs) != 0:
			t.Fatal(tc.body, errs)
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
	cloud := testcloud.New(t)
	var calls []nativeUsageCall
	nativeUsageRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeUsageWire(200, `{}`) })
	for _, err := range usage.New(nativeUsageClient(cloud)).AllTenants(ctx, usage.AllTenantsOpts{}, nil) {
		nativeUsageOperation(t, err, "AllTenants")
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}
