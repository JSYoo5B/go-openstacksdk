package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type r2MiscServers struct {
	mu      sync.Mutex
	queries []string
	calls   int
}

func r2MiscInventoryCloud(t *testing.T, body string, status int) (*sdk.Connection, *r2MiscServers) {
	t.Helper()
	cloud := testcloud.New(t)
	state := &r2MiscServers{}
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		state.queries = append(state.queries, r.URL.RawQuery)
		state.calls++
		state.mu.Unlock()
		testcloud.JSON(w, status, body)
	})
	return connection(t, cloud), state
}

const r2MiscServersA = `{"servers":[{"id":"a-1","name":"web-1","status":"ACTIVE"},{"id":"a-2","name":"db-1","status":"SHUTOFF"}]}`
const r2MiscServersB = `{"servers":[{"id":"b-1","name":"web-2","status":"ACTIVE"},{"id":"b-2","name":"web-1","status":"ACTIVE"}]}`

func r2MiscInventory(t *testing.T, clouds ...sdk.InventoryCloud) *sdk.Inventory {
	t.Helper()
	inventory, err := sdk.NewInventory(clouds...)
	if err != nil {
		t.Fatal(err)
	}
	return inventory
}

func TestR2MiscInventoryListsEveryCloudInOrderWithAllProjectsQuery(t *testing.T) {
	a, stateA := r2MiscInventoryCloud(t, r2MiscServersA, 200)
	b, stateB := r2MiscInventoryCloud(t, r2MiscServersB, 200)
	inventory := r2MiscInventory(t, sdk.InventoryCloud{Name: "a", Connection: a}, sdk.InventoryCloud{Name: "b", Connection: b})
	ctx := context.Background()
	hosts, err := inventory.ListHosts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, host := range hosts.Hosts {
		ids = append(ids, host.Cloud+"/"+host.Server.ID)
	}
	if want := []string{"a/a-1", "a/a-2", "b/b-1", "b/b-2"}; len(ids) != 4 || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] || ids[3] != want[3] {
		t.Fatalf("hosts %v", ids)
	}
	if _, err := inventory.ListHosts(ctx, sdk.WithInventoryAllProjects()); err != nil {
		t.Fatal(err)
	}
	if stateA.queries[0] != "" || stateA.queries[1] != "all_tenants=true" || stateB.calls != 2 {
		t.Fatalf("queries %v %v", stateA.queries, stateB.queries)
	}
}

func TestR2MiscInventoryFailurePolicyKeepsEarlierHostsAndNeverRetries(t *testing.T) {
	a, _ := r2MiscInventoryCloud(t, r2MiscServersA, 200)
	bad, badState := r2MiscInventoryCloud(t, `{"badRequest":{"message":"no"}}`, 503)
	c, _ := r2MiscInventoryCloud(t, r2MiscServersB, 200)
	inventory := r2MiscInventory(t, sdk.InventoryCloud{Name: "a", Connection: a}, sdk.InventoryCloud{Name: "bad", Connection: bad}, sdk.InventoryCloud{Name: "c", Connection: c})
	ctx := context.Background()
	partial, err := inventory.ListHosts(ctx)
	if err == nil || partial == nil || len(partial.Hosts) != 2 || partial.Hosts[0].Cloud != "a" {
		t.Fatalf("default policy: %v %+v", err, partial)
	}
	if badState.calls != 1 {
		t.Fatalf("failing cloud called %d times", badState.calls)
	}
	all, err := inventory.ListHosts(ctx, sdk.WithInventoryContinueOnCloudError())
	if err != nil || len(all.Hosts) != 4 || len(all.Failures) != 1 || all.Failures[0].Cloud != "bad" || all.Failures[0].Err == nil {
		t.Fatalf("continue policy: %v %+v", err, all)
	}
	if badState.calls != 2 {
		t.Fatalf("a second call is a new operation, calls=%d", badState.calls)
	}
}

func TestR2MiscInventoryCancellationAlwaysAborts(t *testing.T) {
	a, stateA := r2MiscInventoryCloud(t, r2MiscServersA, 200)
	inventory := r2MiscInventory(t, sdk.InventoryCloud{Name: "a", Connection: a})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, options := range [][]sdk.InventoryOption{nil, {sdk.WithInventoryContinueOnCloudError()}} {
		if _, err := inventory.ListHosts(ctx, options...); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled: %v", err)
		}
	}
	if stateA.calls != 0 {
		t.Fatal("request made after cancellation")
	}
}

func TestR2MiscInventorySearchAndGetHost(t *testing.T) {
	a, _ := r2MiscInventoryCloud(t, r2MiscServersA, 200)
	b, _ := r2MiscInventoryCloud(t, r2MiscServersB, 200)
	inventory := r2MiscInventory(t, sdk.InventoryCloud{Name: "a", Connection: a}, sdk.InventoryCloud{Name: "b", Connection: b})
	ctx := context.Background()
	glob, err := inventory.SearchHosts(ctx, "web-*")
	if err != nil || len(glob.Hosts) != 3 {
		t.Fatalf("glob %v %+v", err, glob)
	}
	filtered, err := inventory.SearchHosts(ctx, "web-*", sdk.WithInventoryFilters(json.RawMessage(`{"status":"ACTIVE"}`)))
	if err != nil || len(filtered.Hosts) != 3 {
		t.Fatalf("filter %v %+v", err, filtered)
	}
	none, err := inventory.SearchHosts(ctx, "db-*", sdk.WithInventoryFilters(json.RawMessage(`{"status":"ACTIVE"}`)))
	if err != nil || len(none.Hosts) != 0 {
		t.Fatalf("none %v %+v", err, none)
	}
	expression, err := inventory.SearchHosts(ctx, "", sdk.WithInventoryFilters(json.RawMessage(`"[?status=='ACTIVE'].name"`)))
	if err != nil || !expression.Expression || string(expression.Value) != `["web-1","web-2","web-1"]` || len(expression.Hosts) != 0 {
		t.Fatalf("expression %v %+v %s", err, expression, expression.Value)
	}
	host, err := inventory.GetHost(ctx, "db-1")
	if err != nil || host == nil || host.Cloud != "a" || host.Server.ID != "a-2" {
		t.Fatalf("get %v %+v", err, host)
	}
	host, err = inventory.GetHost(ctx, "b-1")
	if err != nil || host == nil || host.Cloud != "b" {
		t.Fatalf("get by id %v %+v", err, host)
	}
	if host, err = inventory.GetHost(ctx, "missing"); err != nil || host != nil {
		t.Fatalf("missing %v %+v", err, host)
	}
	var ambiguous *resource.AmbiguousError
	if _, err = inventory.GetHost(ctx, "web-1"); !errors.As(err, &ambiguous) || len(ambiguous.IDs) != 2 {
		t.Fatalf("ambiguous %v", err)
	}
	if _, err = inventory.GetHost(ctx, "", sdk.WithInventoryFilters(json.RawMessage(`"[].name"`))); err == nil {
		t.Fatal("expression result accepted as a host")
	}
	if _, err = inventory.SearchHosts(ctx, "x", sdk.WithInventoryFilters(json.RawMessage(`{`))); err == nil {
		t.Fatal("malformed filters accepted")
	}
}

func TestR2MiscInventoryValidationAndCloudNames(t *testing.T) {
	if _, err := sdk.NewInventory(sdk.InventoryCloud{}); err == nil {
		t.Error("empty name accepted")
	}
	if _, err := sdk.NewInventory(sdk.InventoryCloud{Name: "x"}, sdk.InventoryCloud{Name: "x"}); err == nil {
		t.Error("duplicate name accepted")
	}
	inventory := r2MiscInventory(t)
	if _, err := inventory.ListHosts(context.Background(), nil); err == nil {
		t.Error("nil option accepted")
	}
	if hosts, err := inventory.ListHosts(context.Background()); err != nil || len(hosts.Hosts) != 0 {
		t.Errorf("empty inventory: %v", err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "clouds.yaml")
	if err := os.WriteFile(file, []byte("clouds:\n  zeta:\n    auth: {auth_url: 'https://z.example'}\n  alpha:\n    auth: {auth_url: 'https://a.example'}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	names, err := sdk.CloudNames(sdk.WithCloudFiles(file))
	if err != nil || len(names) != 2 || names[0] != "zeta" || names[1] != "alpha" {
		t.Fatalf("file order not preserved: %v %v", names, err)
	}
	if _, err := sdk.CloudNames(sdk.WithCloudFiles(filepath.Join(dir, "missing.yaml"))); err == nil {
		t.Error("missing file accepted")
	}
	built, err := sdk.NewInventoryFromCloudFiles(nil, sdk.WithCloudFiles(file))
	if err != nil || built == nil {
		t.Fatalf("from files: %v", err)
	}
	// Lazy connection failures follow the inventory failure policy.
	hosts, err := built.ListHosts(context.Background(), sdk.WithInventoryContinueOnCloudError())
	if err != nil || len(hosts.Failures) != 2 {
		t.Fatalf("lazy failures: %v %+v", err, hosts)
	}
}
