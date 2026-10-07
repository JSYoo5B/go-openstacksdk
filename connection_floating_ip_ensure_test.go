package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

func TestConnectionFloatingIPEnsureReadsCurrentAuthAndResolvesServer(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, lists, updates atomic.Int32
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if r.URL.Query().Get("name") != `^web\[1\]$` {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"servers":[{"id":"other","name":"web[1]copy"},{"id":"server","name":"web[1]"}]}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("device_id") != "server" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"ports":[{"id":"port","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.10"}]}]}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		owner := r.URL.Query().Get("project_id")
		if owner != "first" && owner != "second" || r.Header.Get("X-Auth-Token") != "token-"+owner {
			t.Error("stale or guessed owner/token", owner, r.Header)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"floatingips":[{"id":%q,"project_id":%q,"floating_network_id":"external","floating_ip_address":"198.51.100.10","revision_number":0}]}`, owner, owner))
	})
	cloud.Mux.HandleFunc("PUT /network/v2.0/floatingips/{id}", func(w http.ResponseWriter, r *http.Request) {
		updates.Add(1)
		owner := r.PathValue("id")
		if r.Header.Get("X-Auth-Token") != "token-"+owner || r.Header.Get("If-Match") != "revision_number=0" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"floatingip":{"id":%q,"project_id":%q,"floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","status":"ACTIVE"}}`, owner, owner))
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected endpoint/lookup/allocation/cleanup", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	service, err := connection(t, cloud).Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Prepare once; reuse of the same policy must still read current token scope.
	policy, err := network.PrepareEnsureFloatingIPOptions(context.Background(), network.WithEnsureReuse(false), network.WithEnsureReuse(true))
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"first", "second"} {
		auth := tokens.CreateResult{}
		auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": owner}}}
		auth.Header = http.Header{"X-Subject-Token": []string{"token-" + owner}}
		if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
			t.Fatal(err)
		}
		result, err := service.FloatingIPs.Ensure(context.Background(), network.EnsureFloatingIPRequest{Server: resource.Name("web[1]"), Network: resource.ID("external")}, network.WithEnsureFloatingIPPolicy(policy))
		if err != nil || result == nil || !result.Reused || result.FloatingIP.ID != owner || result.FloatingIP.ProjectID != owner {
			t.Fatalf("owner=%s result=%+v err=%v", owner, result, err)
		}
	}
	if lookups.Load() != 2 || lists.Load() != 2 || updates.Load() != 2 {
		t.Fatalf("lookups=%d lists=%d updates=%d", lookups.Load(), lists.Load(), updates.Load())
	}
}

func TestConnectionFloatingIPEnsureWithoutReuseLetsNeutronChooseProject(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"ports":[{"id":"port","device_id":"server","fixed_ips":[{"ip_address":"10.0.0.10"}]}]}`)
	})
	cloud.Mux.HandleFunc("POST /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			FloatingIP map[string]any `json:"floatingip"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if _, present := body.FloatingIP["project_id"]; present {
			t.Error("guessed project from endpoint", body.FloatingIP)
		}
		testcloud.JSON(w, 201, `{"floatingip":{"id":"fip","project_id":"actual","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","status":"DOWN"}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected auth/lookup/reuse/cleanup", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	service, err := connection(t, cloud).Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.FloatingIPs.Ensure(context.Background(), network.EnsureFloatingIPRequest{Server: resource.ID("server"), Network: resource.ID("external")}, network.WithEnsureReuse(false))
	if err != nil || result == nil || !result.Allocated || result.Reused || result.FloatingIP.ProjectID != "actual" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
