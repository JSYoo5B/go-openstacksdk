package connectiontest_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func pyCloudComputeReadConnection(t *testing.T) (*sdk.Connection, *pyCloudComputeQuotaLog) {
	t.Helper()
	cloud := testcloud.New(t)
	log := &pyCloudComputeQuotaLog{}
	cloud.Mux.HandleFunc("/keystone/v3/projects/", func(w http.ResponseWriter, r *http.Request) {
		log.add(r)
		switch r.URL.Path {
		case "/keystone/v3/projects/tenant":
			testcloud.JSON(w, 404, `{}`)
		case "/keystone/v3/projects/p1":
			testcloud.JSON(w, 200, `{"project":{"id":"p1","name":"other"}}`)
		default:
			testcloud.JSON(w, 404, `{}`)
		}
	})
	cloud.Mux.HandleFunc("/keystone/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		log.add(r)
		if r.URL.Query().Get("name") == "tenant" {
			testcloud.JSON(w, 200, `{"projects":[{"id":"p1","name":"tenant"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"projects":[]}`)
	})
	cloud.Mux.HandleFunc("/nova/v2.1/admin/limits", func(w http.ResponseWriter, r *http.Request) {
		log.add(r)
		testcloud.JSON(w, 200, `{"limits":{"rate":[],"absolute":{"maxTotalCores":20,"totalCoresUsed":0}}}`)
	})
	cloud.Mux.HandleFunc("/nova/v2.1/admin/os-simple-tenant-usage/", func(w http.ResponseWriter, r *http.Request) {
		log.add(r)
		testcloud.JSON(w, 200, `{"tenant_usage":{"tenant_id":"p1","total_hours":2.5,"server_usages":[]}}`)
	})
	c, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/admin"),
		sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/keystone/v3"))
	if err != nil {
		t.Fatal(err)
	}
	return c, log
}

func TestPyCloudComputeGetLimitsWithoutProjectSendsNoIdentityOrTenant(t *testing.T) {
	c, log := pyCloudComputeReadConnection(t)
	got, err := c.GetComputeLimits(context.Background(), "")
	if err != nil || got.Absolute.MaxTotalCores != 20 {
		t.Fatalf("limits: %+v %v", got, err)
	}
	if lines := log.snapshot(); len(lines) != 1 || lines[0] != "GET /nova/v2.1/admin/limits" {
		t.Fatalf("requests: %v", lines)
	}
}

func TestPyCloudComputeGetLimitsResolvesProjectToTenantQuery(t *testing.T) {
	c, log := pyCloudComputeReadConnection(t)
	got, err := c.GetComputeLimits(context.Background(), "tenant")
	if err != nil || got.ProjectID != "p1" {
		t.Fatalf("limits: %+v %v", got, err)
	}
	want := "GET /keystone/v3/projects/tenant|GET /keystone/v3/projects?name=tenant|GET /nova/v2.1/admin/limits?tenant_id=p1"
	if s := strings.Join(log.snapshot(), "|"); s != want {
		t.Fatalf("requests: %s", s)
	}
	if _, err := c.GetComputeLimits(context.Background(), "ghost"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing project: %v", err)
	}
}

func TestPyCloudComputeGetUsageResolvesProjectAndSendsUTCRange(t *testing.T) {
	c, log := pyCloudComputeReadConnection(t)
	start, err := sdk.ParseComputeUsageDate("2026-10-01T09:00:00+09:00")
	if err != nil {
		t.Fatal(err)
	}
	end, err := sdk.ParseComputeUsageDate("2026-10-10")
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.GetComputeUsage(context.Background(), "p1", sdk.ComputeUsageRange{Start: start, End: end})
	if err != nil || got.TenantID != "p1" || got.TotalHours != 2.5 {
		t.Fatalf("usage: %+v %v", got, err)
	}
	lines := log.snapshot()
	if len(lines) != 2 || lines[0] != "GET /keystone/v3/projects/p1" || !strings.HasPrefix(lines[1], "GET /nova/v2.1/admin/os-simple-tenant-usage/p1?") ||
		!strings.Contains(lines[1], "start=2026-10-01T00%3A00%3A00") || !strings.Contains(lines[1], "end=2026-10-10T00%3A00%3A00") {
		t.Fatalf("requests: %v", lines)
	}
	c, log = pyCloudComputeReadConnection(t)
	if _, err := c.GetComputeUsage(context.Background(), "p1", sdk.ComputeUsageRange{}); err != nil {
		t.Fatal(err)
	}
	if last := log.snapshot()[1]; strings.Contains(last, "?") {
		t.Fatalf("zero range sent a query: %s", last)
	}
}

func TestPyCloudComputeUsageDateParsingAndFailures(t *testing.T) {
	for _, bad := range []string{"", "yesterday", "2026-13-01", "10/10/2026"} {
		if _, err := sdk.ParseComputeUsageDate(bad); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%q: %v", bad, err)
		}
	}
	if v, err := sdk.ParseComputeUsageDate("2026-10-01 12:30:00"); err != nil || !v.Equal(time.Date(2026, 10, 1, 12, 30, 0, 0, time.UTC)) {
		t.Fatalf("space separator: %v %v", v, err)
	}
	c, log := pyCloudComputeReadConnection(t)
	if _, err := c.GetComputeUsage(context.Background(), "ghost", sdk.ComputeUsageRange{}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing project: %v", err)
	}
	if _, err := c.GetComputeUsage(context.Background(), "", sdk.ComputeUsageRange{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("blank project: %v", err)
	}
	for _, line := range log.snapshot() {
		if strings.Contains(line, "tenant-usage") {
			t.Fatalf("usage request after failed lookup: %s", line)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.GetComputeLimits(ctx, ""); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
