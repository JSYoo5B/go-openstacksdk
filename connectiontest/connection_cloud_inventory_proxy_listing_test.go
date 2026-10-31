package connectiontest_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// TestInventoryListsThroughCloudListServersProxyPath checks that list_hosts
// consumes the same Proxy.servers rows as cloud list_servers: the Source
// Record view travels with every host, the Python Accept header is sent and
// a caller-independent listing continues by marker when Nova advertises a
// next link.
func TestInventoryListsThroughCloudListServersProxyPath(t *testing.T) {
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var queries []string
	var accepts []string
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		queries = append(queries, r.URL.RawQuery)
		accepts = append(accepts, r.Header.Get("Accept"))
		mu.Unlock()
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"servers":[{"id":"a-1","name":"web-1","tenant_id":"p1","locked":true}],"servers_links":[{"rel":"next","href":"`+cloud.Server.URL+`/compute/v2.1/project/servers/detail?marker=a-1"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"servers":[{"id":"a-2","name":"db-1","tenant_id":"p1"}]}`)
	})
	r2MiscInventoryExpansionEndpoints(cloud)
	inventory := r2MiscInventory(t, sdk.InventoryCloud{Name: "a", Connection: connection(t, cloud)})
	hosts, err := inventory.ListHosts(context.Background(), sdk.WithInventoryBare())
	if err != nil || len(hosts.Hosts) != 2 {
		t.Fatal(hosts, err)
	}
	first := hosts.Hosts[0]
	if first.Record == nil || first.Record.Resource == nil || string(first.Record.Resource.Body["project_id"]) != `"p1"` || string(first.Record.Resource.Body["is_locked"]) != "true" || first.Server.ID != "a-1" || first.Interfaces != nil || first.Hostvars != nil {
		t.Fatalf("%+v", first)
	}
	if hosts.Hosts[1].Server.ID != "a-2" || hosts.Hosts[1].Record == nil {
		t.Fatalf("%+v", hosts.Hosts[1])
	}
	mu.Lock()
	defer mu.Unlock()
	if len(queries) != 2 || queries[0] != "" || queries[1] != "marker=a-1" || accepts[0] != "application/json" || accepts[1] != "application/json" {
		t.Fatal(queries, accepts)
	}
}
