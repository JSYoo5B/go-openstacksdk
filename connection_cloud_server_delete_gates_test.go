package openstack_test

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

const cloudDeleteGateTwoFloatingServer = `{"server":{"id":"s1","name":"web-1","status":"ACTIVE","addresses":{` +
	`"private":[{"addr":"10.0.0.5","version":4,"OS-EXT-IPS:type":"fixed"},{"addr":"198.51.100.7","version":4,"OS-EXT-IPS:type":"floating"}],` +
	`"public":[{"addr":"203.0.113.9","version":4,"OS-EXT-IPS:type":"floating"}]}}}`

func cloudDeleteGateDeletes(cloud *r2CloudComputeCloud) string {
	var paths []string
	for _, call := range r2CloudComputeMethods(cloud.Calls(), http.MethodDelete) {
		paths = append(paths, call.Path)
	}
	return strings.Join(paths, ",")
}

// _has_floating_ips: floating_ip_source none skips the floating cleanup even
// with delete_ips, and the server is still deleted.
func TestCloudDeleteServerFloatingSourceNoneSkipsFloatingCleanup(t *testing.T) {
	cloud := r2CloudComputeNew(t, sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNone)))
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, cloudDeleteGateTwoFloatingServer)
	cloud.JSON("GET /network/v2.0/floatingips", 500, `{}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/os-floating-ips", 500, `{}`)
	cloud.JSON("DELETE "+r2CloudComputeBase+"/servers/s1", 204, ``)
	ok, err := cloud.Conn.DeleteServer(context.Background(), "s1", sdk.WithCloudServerDeleteIPs(true))
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	for _, call := range cloud.Calls() {
		if strings.HasPrefix(call.Path, "/network/") || strings.Contains(call.Path, "os-floating-ips") {
			t.Fatalf("floating backend contacted: %+v", call)
		}
	}
	if got := cloudDeleteGateDeletes(cloud); got != r2CloudComputeBase+"/servers/s1" {
		t.Fatal(got)
	}
}

// _delete_server_floating_ips: a lookup that raises NotFound for one address
// continues with the next address; the remaining floating IP and then the
// server are deleted.
func TestCloudDeleteServerFloatingLookupNotFoundContinues(t *testing.T) {
	cloud := r2CloudComputeNew(t, sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNeutron)))
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, cloudDeleteGateTwoFloatingServer)
	var deleted atomic.Bool
	cloud.Handle("GET /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request, _ string) {
		q := r.URL.Query()
		switch {
		case q.Get("floating_ip_address") == "198.51.100.7":
			testcloud.JSON(w, 404, `{"NeutronError":{"type":"HTTPNotFound","message":"gone"}}`)
		case q.Get("floating_ip_address") == "203.0.113.9" || q.Get("id") == "fip2" && !deleted.Load():
			testcloud.JSON(w, 200, `{"floatingips":[{"id":"fip2","floating_ip_address":"203.0.113.9","status":"ACTIVE"}]}`)
		default:
			testcloud.JSON(w, 200, `{"floatingips":[]}`)
		}
	})
	cloud.Handle("DELETE /network/v2.0/floatingips/fip2", func(w http.ResponseWriter, _ *http.Request, _ string) {
		deleted.Store(true)
		w.WriteHeader(204)
	})
	cloud.JSON("DELETE "+r2CloudComputeBase+"/servers/s1", 204, ``)
	ok, err := cloud.Conn.DeleteServer(context.Background(), "s1", sdk.WithCloudServerDeleteIPs(true))
	if err != nil || !ok {
		t.Fatal(ok, err, cloud.Calls())
	}
	if got := cloudDeleteGateDeletes(cloud); got != "/network/v2.0/floatingips/fip2,"+r2CloudComputeBase+"/servers/s1" {
		t.Fatal(got)
	}
}

// delete_server uses find_server(name_or_id, ignore_missing=True) with its
// defaults: GET by identity, then the detail listing filtered by name only
// (no all_projects), then one DELETE of the found ID.
func TestCloudDeleteServerFindServerDefaults(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/web-1", 404, `{"itemNotFound":{"message":"nf","code":404}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, `{"servers":[{"id":"s1","name":"web-1","status":"ACTIVE","addresses":{}}]}`)
	cloud.JSON("DELETE "+r2CloudComputeBase+"/servers/s1", 204, ``)
	ok, err := cloud.Conn.DeleteServer(context.Background(), "web-1")
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	want := []r2CloudComputeCall{
		{http.MethodGet, r2CloudComputeBase + "/servers/web-1", "", ""},
		{http.MethodGet, r2CloudComputeBase + "/servers/detail", "name=web-1", ""},
		{http.MethodDelete, r2CloudComputeBase + "/servers/s1", "", ""},
	}
	got := cloud.Calls()
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d: %+v want %+v", i, got[i], want[i])
		}
	}
}

// With floating_ip_source nova, delete_ips lists Nova floating IPs, filters by
// address locally, deletes through os-floating-ips and never contacts Neutron.
func TestCloudDeleteServerNovaFloatingSourceDeletesThroughNova(t *testing.T) {
	// The legacy os-floating-ips proxy needs a microversion below 2.36.
	cloud := r2CloudComputeNew(t, sdk.WithMicroversion(sdk.Compute, "2.1"), sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNova)))
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, r2CloudComputeFloatingServer)
	var deleted atomic.Bool
	cloud.Handle("GET "+r2CloudComputeBase+"/os-floating-ips", func(w http.ResponseWriter, _ *http.Request, _ string) {
		if deleted.Load() {
			testcloud.JSON(w, 200, `{"floating_ips":[]}`)
			return
		}
		testcloud.JSON(w, 200, `{"floating_ips":[{"id":"nfip1","ip":"198.51.100.7","fixed_ip":"10.0.0.5","instance_id":"s1","pool":"public"},`+
			`{"id":"nfip2","ip":"203.0.113.50","fixed_ip":null,"instance_id":null,"pool":"public"}]}`)
	})
	cloud.Handle("GET "+r2CloudComputeBase+"/os-floating-ips/nfip1", func(w http.ResponseWriter, _ *http.Request, _ string) {
		if deleted.Load() {
			testcloud.JSON(w, 404, `{"itemNotFound":{"message":"nf","code":404}}`)
			return
		}
		testcloud.JSON(w, 200, `{"floating_ip":{"id":"nfip1","ip":"198.51.100.7","fixed_ip":"10.0.0.5","instance_id":"s1","pool":"public"}}`)
	})
	cloud.Handle("DELETE "+r2CloudComputeBase+"/os-floating-ips/nfip1", func(w http.ResponseWriter, _ *http.Request, _ string) {
		deleted.Store(true)
		w.WriteHeader(202)
	})
	cloud.JSON("DELETE "+r2CloudComputeBase+"/servers/s1", 204, ``)
	ok, err := cloud.Conn.DeleteServer(context.Background(), "s1", sdk.WithCloudServerDeleteIPs(true))
	if err != nil || !ok {
		t.Fatal(ok, err, cloud.Calls())
	}
	if got := cloudDeleteGateDeletes(cloud); got != r2CloudComputeBase+"/os-floating-ips/nfip1,"+r2CloudComputeBase+"/servers/s1" {
		t.Fatal(got)
	}
	for _, call := range cloud.Calls() {
		if strings.HasPrefix(call.Path, "/network/") {
			t.Fatalf("Neutron contacted: %+v", call)
		}
	}
}
