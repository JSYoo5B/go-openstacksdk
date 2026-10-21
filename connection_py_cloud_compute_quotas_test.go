package openstack_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	blockquotas "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pyCloudComputeQuotaLog struct {
	mu    sync.Mutex
	lines []string
	body  string
}

func (l *pyCloudComputeQuotaLog) add(r *http.Request) {
	data, _ := io.ReadAll(r.Body)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, r.Method+" "+r.URL.RequestURI())
	if len(data) > 0 {
		l.body = string(data)
	}
}

func (l *pyCloudComputeQuotaLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.lines...)
}

func pyCloudComputeQuotaConnection(t *testing.T, projects http.HandlerFunc) (*sdk.Connection, *pyCloudComputeQuotaLog) {
	t.Helper()
	cloud := testcloud.New(t)
	log := &pyCloudComputeQuotaLog{}
	cloud.Mux.HandleFunc("/keystone/v3/projects", func(w http.ResponseWriter, r *http.Request) { log.add(r); projects(w, r) })
	cloud.Mux.HandleFunc("/keystone/v3/projects/", func(w http.ResponseWriter, r *http.Request) { log.add(r); projects(w, r) })
	cloud.Mux.HandleFunc("/nova/v2.1/admin/os-quota-sets/", func(w http.ResponseWriter, r *http.Request) {
		log.add(r)
		switch r.Method {
		case http.MethodDelete:
			w.Header().Set("X-Trace", "nova")
			w.WriteHeader(http.StatusAccepted)
		default:
			testcloud.JSON(w, 200, `{"quota_set":{"id":"wire","cores":8}}`)
		}
	})
	cloud.Mux.HandleFunc("/cinder/v3/admin/os-quota-sets/", func(w http.ResponseWriter, r *http.Request) {
		log.add(r)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		testcloud.JSON(w, 200, `{"quota_set":{"id":"wire","volumes":5}}`)
	})
	c, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/admin"),
		sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cinder/v3/admin"),
		sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/keystone/v3"))
	if err != nil {
		t.Fatal(err)
	}
	return c, log
}

func pyCloudComputeProjectByID(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/keystone/v3/projects/p1":
		testcloud.JSON(w, 200, `{"project":{"id":"p1","name":"tenant"}}`)
	case "/keystone/v3/projects":
		if r.URL.Query().Get("name") == "tenant" {
			testcloud.JSON(w, 200, `{"projects":[{"id":"p1","name":"tenant"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"projects":[]}`)
	default:
		testcloud.JSON(w, 404, `{"error":{"message":"missing"}}`)
	}
}

func TestPyCloudComputeQuotaHelpersLookUpIDThenName(t *testing.T) {
	ctx := context.Background()
	c, log := pyCloudComputeQuotaConnection(t, pyCloudComputeProjectByID)
	got, err := c.GetComputeQuotas(ctx, "p1")
	if err != nil || got.ProjectID != "p1" {
		t.Fatalf("by id: %+v %v", got, err)
	}
	if lines := strings.Join(log.snapshot(), "|"); lines != "GET /keystone/v3/projects/p1|GET /nova/v2.1/admin/os-quota-sets/p1" {
		t.Fatalf("ID path still looks the project up first: %s", lines)
	}
	c, log = pyCloudComputeQuotaConnection(t, pyCloudComputeProjectByID)
	if got, err = c.GetComputeQuotas(ctx, "tenant"); err != nil || got.ProjectID != "p1" {
		t.Fatalf("by name: %+v %v", got, err)
	}
	if lines := strings.Join(log.snapshot(), "|"); !strings.Contains(lines, "GET /keystone/v3/projects?name=tenant") || !strings.HasSuffix(lines, "GET /nova/v2.1/admin/os-quota-sets/p1") {
		t.Fatalf("name path: %s", lines)
	}
}

func TestPyCloudComputeQuotaHelpersMissingProjectStopsBeforeService(t *testing.T) {
	ctx := context.Background()
	c, log := pyCloudComputeQuotaConnection(t, pyCloudComputeProjectByID)
	cores := 1
	if _, err := c.GetComputeQuotas(ctx, "absent"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("get: %v", err)
	}
	if _, err := c.SetComputeQuotas(ctx, "absent", quotasets.UpdateOpts{Cores: &cores}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("set: %v", err)
	}
	if _, err := c.DeleteComputeQuotas(ctx, "absent"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("delete: %v", err)
	}
	if _, err := c.GetBlockStorageQuotas(ctx, "absent"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("volume get: %v", err)
	}
	for _, line := range log.snapshot() {
		if strings.Contains(line, "os-quota-sets") {
			t.Fatalf("service request after failed lookup: %s", line)
		}
	}
}

func TestPyCloudComputeSetQuotasForcesTrueAndKeepsZero(t *testing.T) {
	ctx := context.Background()
	c, log := pyCloudComputeQuotaConnection(t, pyCloudComputeProjectByID)
	zero := 0
	if _, err := c.SetComputeQuotas(ctx, "tenant", quotasets.UpdateOpts{Cores: &zero}, quotasets.WithUpdateForce(false)); err != nil {
		t.Fatal(err)
	}
	log.mu.Lock()
	body := log.body
	log.mu.Unlock()
	if !strings.Contains(body, `"force":true`) || !strings.Contains(body, `"cores":0`) || strings.Contains(body, "ram") {
		t.Fatalf("body: %s", body)
	}
}

func TestPyCloudComputeDeleteQuotasReturnsResetAndVolumeSideWorks(t *testing.T) {
	ctx := context.Background()
	c, log := pyCloudComputeQuotaConnection(t, pyCloudComputeProjectByID)
	reset, err := c.DeleteComputeQuotas(ctx, "p1")
	if err != nil || reset.ProjectID != "p1" || reset.Header.Get("X-Trace") != "nova" {
		t.Fatalf("delete: %+v %v", reset, err)
	}
	vol := 3
	got, err := c.SetBlockStorageQuotas(ctx, "p1", blockquotas.UpdateOpts{Volumes: &vol})
	if err != nil || got == nil {
		t.Fatalf("volume set: %+v %v", got, err)
	}
	if _, err := c.GetBlockStorageQuotas(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DeleteBlockStorageQuotas(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	lines := strings.Join(log.snapshot(), "|")
	for _, want := range []string{"DELETE /nova/v2.1/admin/os-quota-sets/p1", "PUT /cinder/v3/admin/os-quota-sets/p1", "GET /cinder/v3/admin/os-quota-sets/p1", "DELETE /cinder/v3/admin/os-quota-sets/p1"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("missing %s in %s", want, lines)
		}
	}
	if strings.Count(lines, "GET /keystone/v3/projects/p1") != 4 {
		t.Fatalf("each helper call should look the project up once: %s", lines)
	}
}

func TestPyCloudComputeQuotaHelpersHonorCanceledContext(t *testing.T) {
	c, log := pyCloudComputeQuotaConnection(t, pyCloudComputeProjectByID)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.GetComputeQuotas(ctx, "p1"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := c.DeleteBlockStorageQuotas(ctx, "p1"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(log.snapshot()) != 0 {
		t.Fatalf("requests sent: %v", log.snapshot())
	}
}
