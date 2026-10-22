package openstack_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

const inventoryHostvarsServers = `{"servers":[` +
	`{"id":"h1","name":"web-1","status":"ACTIVE","flavor":{"original_name":"small"},"image":"","metadata":{},` +
	`"addresses":{"private":[{"version":4,"addr":"10.0.0.7","OS-EXT-IPS:type":"fixed"},{"version":4,"addr":"8.8.4.4","OS-EXT-IPS:type":"floating"}]}}]}`

type inventoryHostvarsLog struct {
	mu    sync.Mutex
	paths []string
}

func (l *inventoryHostvarsLog) record(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.paths = append(l.paths, path)
}

func (l *inventoryHostvarsLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.paths...)
}

func inventoryHostvarsCloud(t *testing.T, securityGroupStatus int) (*sdk.Inventory, *inventoryHostvarsLog) {
	t.Helper()
	cloud := testcloud.New(t)
	log := &inventoryHostvarsLog{}
	serve := func(pattern string, status int, body string) {
		cloud.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			log.record(r.URL.Path)
			testcloud.JSON(w, status, body)
		})
	}
	serve("GET /compute/v2.1/project/servers/detail", 200, inventoryHostvarsServers)
	serve("GET /network/v2.0/networks", 200, `{"networks":[]}`)
	serve("GET /compute/v2.1/project/servers/h1", securityGroupStatus, `{"server":{"id":"h1"}}`)
	serve("GET /compute/v2.1/project/servers/h1/os-security-groups", 200, `{"security_groups":[]}`)
	serve("GET /volume/v3/project/volumes/detail", 200, `{"volumes":[{"id":"v1","attachments":[{"server_id":"h1","device":"/dev/vdb"}]}]}`)
	inventory, err := sdk.NewInventory(sdk.InventoryCloud{Name: "a", Connection: connection(t, cloud)})
	if err != nil {
		t.Fatal(err)
	}
	return inventory, log
}

func TestInventoryHostsDefaultToHostvarsExpansion(t *testing.T) {
	inventory, log := inventoryHostvarsCloud(t, 200)
	ctx := context.Background()
	hosts, err := inventory.ListHosts(ctx)
	if err != nil || len(hosts.Hosts) != 1 {
		t.Fatalf("%+v %v", hosts, err)
	}
	host := hosts.Hosts[0]
	if host.Hostvars == nil || host.Interfaces != host.Hostvars.Interfaces || host.Interfaces.PublicIPv4 != "8.8.4.4" || host.Interfaces.InterfaceIP != "8.8.4.4" {
		t.Fatalf("%+v", host)
	}
	if string(host.Hostvars.Flavor.Name) != `"small"` || !host.Hostvars.Image.BootFromVolume || len(host.Hostvars.Volumes) != 1 {
		t.Fatalf("%+v", host.Hostvars)
	}
	paths := strings.Join(log.snapshot(), " ")
	for _, want := range []string{"/servers/detail", "/network/v2.0/networks", "/servers/h1/os-security-groups", "/volumes/detail"} {
		if !strings.Contains(paths, want) {
			t.Fatalf("missing %s in %s", want, paths)
		}
	}
	// Filters see the hostvars mapping.
	found, err := inventory.SearchHosts(ctx, "web-*", sdk.WithInventoryFilters(json.RawMessage(`{"public_v4":"8.8.4.4","flavor":{"name":"small"}}`)))
	if err != nil || len(found.Hosts) != 1 {
		t.Fatalf("%+v %v", found, err)
	}
	host2, err := inventory.GetHost(ctx, "h1", sdk.WithInventoryFilters(json.RawMessage(`"[?volumes[0].device=='/dev/vdb']"`)))
	if err == nil || host2 != nil {
		t.Fatalf("an expression result is not a host: %+v %v", host2, err)
	}
	got, err := inventory.GetHost(ctx, "web-1", sdk.WithInventoryFilters(json.RawMessage(`{"image":{"id":""}}`)))
	if err != nil || got == nil || got.Hostvars == nil {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestInventoryExpandFalseBareAndPrivate(t *testing.T) {
	inventory, log := inventoryHostvarsCloud(t, 200)
	ctx := context.Background()
	hosts, err := inventory.ListHosts(ctx, sdk.WithInventoryExpand(false))
	if err != nil || len(hosts.Hosts) != 1 || hosts.Hosts[0].Hostvars != nil || hosts.Hosts[0].Interfaces == nil {
		t.Fatalf("%+v %v", hosts, err)
	}
	for _, path := range log.snapshot() {
		if strings.Contains(path, "security-groups") || strings.Contains(path, "volumes") {
			t.Fatalf("expand=false must not build hostvars: %s", path)
		}
	}
	before := len(log.snapshot())
	bare, err := inventory.ListHosts(ctx, sdk.WithInventoryBare())
	if err != nil || bare.Hosts[0].Interfaces != nil || bare.Hosts[0].Hostvars != nil || len(log.snapshot()) != before+1 {
		t.Fatalf("%+v %v %v", bare, err, log.snapshot()[before:])
	}
	private, err := inventory.SearchHosts(ctx, "web-1", sdk.WithInventoryExpand(false), sdk.WithInventoryPrivate(true),
		sdk.WithInventoryFilters(json.RawMessage(`{"interface_ip":"10.0.0.7","access_ipv4":"10.0.0.7","public_v4":"8.8.4.4"}`)))
	if err != nil || len(private.Hosts) != 1 {
		t.Fatalf("%+v %v", private, err)
	}
	public, err := inventory.SearchHosts(ctx, "web-1", sdk.WithInventoryExpand(false),
		sdk.WithInventoryFilters(json.RawMessage(`{"interface_ip":"10.0.0.7"}`)))
	if err != nil || len(public.Hosts) != 0 {
		t.Fatalf("%+v %v", public, err)
	}
}

func TestInventoryHostvarsSwallowSecurityGroupFailureButFailCloudOnLookupError(t *testing.T) {
	inventory, _ := inventoryHostvarsCloud(t, 500)
	hosts, err := inventory.ListHosts(context.Background())
	if err != nil || len(hosts.Hosts) != 1 || hosts.Hosts[0].Hostvars.SecurityGroupsError == nil {
		t.Fatalf("%+v %v", hosts, err)
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/detail", func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"servers":[{"id":"x1","name":"x","status":"SHUTOFF","flavor":{"id":"f9"},"image":"","addresses":{}}]}`)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/flavors/f9", func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 500, `{"computeFault":{"message":"boom","code":500}}`)
	})
	broken, err := sdk.NewInventory(sdk.InventoryCloud{Name: "broken", Connection: connection(t, cloud)})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := broken.ListHosts(context.Background()); err == nil || len(result.Hosts) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
	result, err := broken.ListHosts(context.Background(), sdk.WithInventoryContinueOnCloudError())
	if err != nil || len(result.Hosts) != 0 || len(result.Failures) != 1 || result.Failures[0].Cloud != "broken" {
		t.Fatalf("%+v %v", result, err)
	}
}
