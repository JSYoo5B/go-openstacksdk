package openstack_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	networkquotas "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pyCloudNetQuotaServer struct {
	mu       sync.Mutex
	requests []string
	bodies   []string
	project  string
	status   int
}

func newPyCloudNetQuotaConnection(t *testing.T, projects string) (*sdk.Connection, *pyCloudNetQuotaServer) {
	t.Helper()
	cloud := testcloud.New(t)
	server := &pyCloudNetQuotaServer{}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		server.mu.Lock()
		server.requests = append(server.requests, r.Method+" "+r.URL.Path)
		server.bodies = append(server.bodies, string(raw))
		server.mu.Unlock()
		switch {
		case r.URL.Path == "/keystone/v3/projects":
			testcloud.JSON(w, 200, projects)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/details.json"):
			testcloud.JSON(w, 200, `{"quota":{"network":{"limit":8,"used":2,"reserved":0}}}`)
		case r.Method == http.MethodGet:
			testcloud.JSON(w, 200, `{"quota":{"network":8,"id":"wire-project"}}`)
		case r.Method == http.MethodPut:
			testcloud.JSON(w, 200, `{"quota":{"network":0,"port":-1}}`)
		case r.Method == http.MethodDelete:
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
		}
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/neutron/v2.0"), sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/keystone/v3"))
	if err != nil {
		t.Fatal(err)
	}
	return c, server
}

func (s *pyCloudNetQuotaServer) snapshot() ([]string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...), append([]string(nil), s.bodies...)
}

func pyCloudNetEqualStrings(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %q want %q", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %q want %q", got, want)
		}
	}
}

func TestPyCloudNetNetworkQuotasGetDetailAndNamedProjectLookup(t *testing.T) {
	c, server := newPyCloudNetQuotaConnection(t, `{"projects":[{"id":"proj-1","name":"tenant"}]}`)
	scope, err := c.NetworkProjectQuotas(context.Background(), resource.Name("tenant"))
	if err != nil || scope.ProjectID() != "proj-1" {
		t.Fatal(scope, err)
	}
	quota, err := scope.Get(context.Background())
	if err != nil || quota == nil {
		t.Fatal(quota, err)
	}
	if _, err := scope.Detail(context.Background()); err != nil {
		t.Fatal(err)
	}
	requests, _ := server.snapshot()
	pyCloudNetEqualStrings(t, requests, "GET /keystone/v3/projects", "GET /neutron/v2.0/quotas/proj-1", "GET /neutron/v2.0/quotas/proj-1/details.json")
}

func TestPyCloudNetNetworkQuotasUpdateBodyKeepsZeroUnlimitedAndCheckLimit(t *testing.T) {
	c, server := newPyCloudNetQuotaConnection(t, `{"projects":[]}`)
	scope, err := c.NetworkProjectQuotas(context.Background(), resource.ID("proj-2"))
	if err != nil {
		t.Fatal(err)
	}
	zero, unlimited := 0, -1
	if _, err := scope.Update(context.Background(), networkquotas.UpdateOpts{Network: &zero, Port: &unlimited}, networkquotas.WithUpdateCheckLimit(false)); err != nil {
		t.Fatal(err)
	}
	requests, bodies := server.snapshot()
	pyCloudNetEqualStrings(t, requests, "PUT /neutron/v2.0/quotas/proj-2")
	pyCloudNetEqualStrings(t, bodies, `{"quota":{"check_limit":false,"network":0,"port":-1}}`)
}

func TestPyCloudNetNetworkQuotasMissingNamedProjectFailsBeforeNeutron(t *testing.T) {
	c, server := newPyCloudNetQuotaConnection(t, `{"projects":[]}`)
	scope, err := c.NetworkProjectQuotas(context.Background(), resource.Name("ghost"))
	if err == nil || scope != nil {
		t.Fatal(scope, err)
	}
	requests, _ := server.snapshot()
	pyCloudNetEqualStrings(t, requests, "GET /keystone/v3/projects")
}

func TestPyCloudNetNetworkQuotasDeleteSendsNoBody(t *testing.T) {
	c, server := newPyCloudNetQuotaConnection(t, `{"projects":[]}`)
	scope, err := c.NetworkProjectQuotas(context.Background(), resource.ID("proj-3"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	requests, bodies := server.snapshot()
	pyCloudNetEqualStrings(t, requests, "DELETE /neutron/v2.0/quotas/proj-3")
	pyCloudNetEqualStrings(t, bodies, "")
}

// The pinned Python update_quota commits only dirty fields, so no kwargs sends
// no request. The Go scope sends an explicit empty quota envelope instead.
func TestPyCloudNetNetworkQuotasEmptyUpdateSendsEmptyEnvelope(t *testing.T) {
	c, server := newPyCloudNetQuotaConnection(t, `{"projects":[]}`)
	scope, err := c.NetworkProjectQuotas(context.Background(), resource.ID("proj-4"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Update(context.Background(), networkquotas.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	requests, bodies := server.snapshot()
	pyCloudNetEqualStrings(t, requests, "PUT /neutron/v2.0/quotas/proj-4")
	pyCloudNetEqualStrings(t, bodies, `{"quota":{}}`)
}
