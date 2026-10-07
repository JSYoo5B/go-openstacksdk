package network_test

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

func TestFloatingIPPrepareActiveBindsCopyAndPreservesWaitPolicy(t *testing.T) {
	cloud := testcloud.New(t)
	ensurePortFixture(t, cloud)
	setOwner := func(id string) {
		auth := tokens.CreateResult{}
		auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": id}}}
		auth.Header = http.Header{"X-Subject-Token": []string{"test-token"}}
		if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
			t.Fatal(err)
		}
	}
	var owner atomic.Value
	var ipGets, callbacks atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		owner.Store(r.URL.Query().Get("project_id"))
		testcloud.JSON(w, 200, fmt.Sprintf(`{"floatingips":[{"id":"fip","project_id":%q,"floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","status":"DOWN"}]}`, owner.Load()))
	})
	cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
		get := ipGets.Add(1)
		status := "DOWN"
		if get == 2 {
			status = "ACTIVE"
		}
		respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", status, map[string]any{"project_id": owner.Load()})
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected allocation/mutation/cleanup", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	service := network.New(cloud.Client("network", "/v2.0"))
	plain, err := network.PrepareEnsureFloatingIPOptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	setOwner("first")
	bound, err := service.FloatingIPs.PrepareEnsureActive(context.Background(), network.WithEnsureFloatingIPPolicy(plain), network.WithEnsureWait(resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second), resource.WithProgressCallback(func(int) { callbacks.Add(1) })))
	if err != nil {
		t.Fatal(err)
	}
	setOwner("second")
	result, err := service.FloatingIPs.Ensure(context.Background(), ensureFloatingRequest(), network.WithEnsureFloatingIPPolicy(bound))
	if err != nil || result == nil || result.FloatingIP.ProjectID != "first" || result.FloatingIP.Status != "ACTIVE" || callbacks.Load() != 1 || ipGets.Load() != 2 {
		t.Fatalf("bound result=%+v err=%v callback=%d GETs=%d", result, err, callbacks.Load(), ipGets.Load())
	}
	result, err = service.FloatingIPs.Ensure(context.Background(), ensureFloatingRequest(), network.WithEnsureFloatingIPPolicy(plain))
	if err != nil || result == nil || result.FloatingIP.ProjectID != "second" || result.FloatingIP.Status != "DOWN" || ipGets.Load() != 2 {
		t.Fatalf("plain policy mutated: result=%+v err=%v GETs=%d", result, err, ipGets.Load())
	}
}
