package openstack_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Connection.CreateRefreshedServer performs Python's wait=False sequence on
// the Connection-owned Compute service: one POST, one GET and expansion.
func TestConnectionCreateRefreshedServerPostsOnceAndExpands(t *testing.T) {
	cloud := testcloud.New(t)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"))
	if err != nil {
		t.Fatal(err)
	}
	var posts, gets atomic.Int32
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		testcloud.JSON(w, 202, `{"server":{"id":"server","status":"BUILD","adminPass":"generated"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/servers/server", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, `{"server":{"id":"server","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"203.0.113.9","OS-EXT-IPS:type":"floating"}]}}}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"networks":[]}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	result, err := conn.CreateRefreshedServer(context.Background(), compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")},
		[]compute.CreateServerOption{compute.WithNetworks(resource.ID("private"))}, compute.WithAddressReachability(false))
	if err != nil || result.Server.AdminPass != "generated" || result.Addresses == nil || result.Addresses.PublicIPv4 != "203.0.113.9" || posts.Load() != 1 || gets.Load() != 1 {
		t.Fatal(result, err, posts.Load(), gets.Load())
	}
}
