package connectiontest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const hostvarsServerS1 = `{"id":"s1","name":"web-1","status":"ACTIVE","tenant_id":"p",` +
	`"flavor":{"id":"f1"},"image":{"id":"i1"},` +
	`"metadata":{"group":"web","groups":"blue,,green","role":"front"},` +
	`"addresses":{"private":[{"version":4,"addr":"10.0.0.5","OS-EXT-IPS:type":"fixed","OS-EXT-IPS-MAC:mac_addr":"fa:16:3e:00:00:01"}]}}`

func hostvarsServer(t *testing.T, raw string) *servers.Server {
	t.Helper()
	var server servers.Server
	if err := json.Unmarshal([]byte(raw), &server); err != nil {
		t.Fatal(err)
	}
	return &server
}

// hostvarsCloud serves the Neutron, flavor, security group, image and volume
// reads of one fully expanded ACTIVE server s1. volumeStatus 0 removes the
// block-storage endpoint; imageStatus selects the image list response.
func hostvarsCloud(t *testing.T, volumeStatus, imageStatus int) *r2CloudComputeCloud {
	t.Helper()
	cloud := r2CloudComputeNew(t)
	options := []sdk.ConnectionOption{sdk.WithMicroversion(sdk.Compute, "2.96"),
		sdk.WithEndpoint(sdk.Image, cloud.Server.URL+"/image/v2"),
		sdk.WithServerAddressPolicy(compute.WithAddressReachability(false))}
	if volumeStatus != 0 {
		options = append(options, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/volume/v3/p"))
	}
	conn, err := sdk.FromProvider(cloud.Provider, options...)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Conn = conn
	cloud.JSON("GET /network/v2.0/networks", 200, `{"networks":[]}`)
	cloud.JSON("GET /network/v2.0/ports", 200, `{"ports":[{"id":"port1","device_id":"s1","mac_address":"fa:16:3e:00:00:01"}]}`)
	cloud.JSON("GET /network/v2.0/floatingips", 200, `{"floatingips":[{"id":"fip1","port_id":"port1","fixed_ip_address":"10.0.0.5","floating_ip_address":"8.8.8.8"}]}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/flavors/f1", 200, `{"flavor":{"id":"f1","name":"small","ram":512,"vcpus":1,"disk":1}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1/os-security-groups", 200, `{"security_groups":[{"id":"sg1","name":"default","rules":[]}]}`)
	images := `{"images":[{"id":"i1","name":"ubuntu","status":"active"}]}`
	if imageStatus != 200 {
		images = `{"message":"boom"}`
	}
	cloud.JSON("GET /image/v2/images", imageStatus, images)
	volumes := `{"volumes":[` +
		`{"id":"v1","display_name":"data","attachments":[{"server_id":"s1","device":"/dev/vdb"}]},` +
		`{"id":"v2","attachments":[{"server_id":"other","device":"/dev/vdc"}]}]}`
	if volumeStatus != 200 {
		volumes = `{"badRequest":{"message":"boom"}}`
	}
	cloud.JSON("GET /volume/v3/p/volumes/detail", volumeStatus, volumes)
	return cloud
}

func hostvarsCallPaths(cloud *r2CloudComputeCloud) []string {
	var paths []string
	for _, call := range cloud.Calls() {
		if call.Method != http.MethodGet {
			paths = append(paths, call.Method+" "+call.Path)
			continue
		}
		paths = append(paths, call.Path)
	}
	return paths
}

func TestCloudServerHostvarsExpandInPythonOrderWithoutMutatingServer(t *testing.T) {
	cloud := hostvarsCloud(t, 200, 200)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":`+hostvarsServerS1+`}`)
	server := hostvarsServer(t, hostvarsServerS1)
	before, _ := json.Marshal(server)
	ctx := context.Background()
	meta, err := cloud.Conn.GetServerMeta(ctx, server)
	if err != nil || meta == nil || meta.Hostvars == nil {
		t.Fatalf("%+v %v", meta, err)
	}
	if after, _ := json.Marshal(server); string(after) != string(before) {
		t.Fatalf("supplied server mutated: %s", after)
	}
	hostvars := meta.Hostvars
	view := hostvars.Interfaces
	if view == nil || view.PublicIPv4 != "8.8.8.8" || view.PrivateIPv4 != "10.0.0.5" || view.InterfaceIP != "8.8.8.8" || view.AccessIPv4 != "8.8.8.8" {
		t.Fatalf("%+v", view)
	}
	if string(hostvars.Flavor.Name) != `"small"` || hostvars.Flavor.NameSource != "lookup" || hostvars.Flavor.ID != "f1" {
		t.Fatalf("%+v", hostvars.Flavor)
	}
	if string(hostvars.Image.Name) != `"ubuntu"` || hostvars.Image.ID != "i1" || hostvars.Image.BootFromVolume {
		t.Fatalf("%+v", hostvars.Image)
	}
	if len(hostvars.SecurityGroups) != 1 || hostvars.SecurityGroups[0].Name != "default" || hostvars.SecurityGroupsError != nil {
		t.Fatalf("%+v %v", hostvars.SecurityGroups, hostvars.SecurityGroupsError)
	}
	if !hostvars.VolumeService || hostvars.VolumesError != nil || len(hostvars.Volumes) != 1 || string(hostvars.Volumes[0].Device) != `"/dev/vdb"` {
		t.Fatalf("%+v", hostvars.Volumes)
	}
	want := []string{
		"/network/v2.0/ports", "/network/v2.0/floatingips", "/network/v2.0/networks",
		r2CloudComputeBase + "/flavors/f1",
		r2CloudComputeBase + "/servers/s1", r2CloudComputeBase + "/servers/s1/os-security-groups",
		"/image/v2/images", "/volume/v3/p/volumes/detail",
	}
	if got := hostvarsCallPaths(cloud); !slices.Equal(got, want) {
		t.Fatalf("calls %v", got)
	}
	var mapping map[string]json.RawMessage
	raw, err := json.Marshal(hostvars)
	if err != nil || json.Unmarshal(raw, &mapping) != nil {
		t.Fatal(string(raw), err)
	}
	for key, value := range map[string]string{
		"public_v4": `"8.8.8.8"`, "private_v4": `"10.0.0.5"`, "interface_ip": `"8.8.8.8"`,
		"access_ipv4": `"8.8.8.8"`, "public_v6": `""`, "access_ipv6": `""`, "id": `"s1"`,
	} {
		if string(mapping[key]) != value {
			t.Fatalf("%s=%s in %s", key, mapping[key], raw)
		}
	}
	for _, fragment := range []string{
		`"flavor":{"id":"f1","name":"small"}`, `"image":{"id":"i1","name":"ubuntu"}`,
		`"addr":"8.8.8.8"`, `"device":"/dev/vdb"`, `"location":{`,
	} {
		if !strings.Contains(string(raw), fragment) {
			t.Fatalf("missing %s in %s", fragment, raw)
		}
	}
	if strings.Contains(string(raw), `"sg1"`) || strings.Contains(string(raw), `"v2"`) {
		t.Fatalf("security groups and foreign volumes are not hostvars: %s", raw)
	}
	wantGroups := []string{"", "", "_", "web", "blue", "green", "instance-s1", "flavor-small", "image-ubuntu",
		"meta-group_web", "meta-groups_blue,,green", "meta-role_front"}
	if !slices.Equal(meta.Groups, wantGroups) {
		t.Fatalf("groups %q", meta.Groups)
	}
}

func TestCloudServerHostvarsOriginalNameBootFromVolumeAndRecordedFailures(t *testing.T) {
	cloud := hostvarsCloud(t, 200, 200)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s2", 500, `{"computeFault":{"message":"boom","code":500}}`)
	server := hostvarsServer(t, `{"id":"s2","name":"bfv","status":"SHUTOFF","flavor":{"original_name":"m1.tiny","vcpus":1},"image":"","metadata":{}}`)
	hostvars, err := cloud.Conn.GetOpenStackVars(context.Background(), server)
	if err != nil || hostvars == nil {
		t.Fatal(hostvars, err)
	}
	if string(hostvars.Flavor.Name) != `"m1.tiny"` || hostvars.Flavor.NameSource != "original_name" || hostvars.Flavor.Lookup != nil {
		t.Fatalf("%+v", hostvars.Flavor)
	}
	if !hostvars.Image.BootFromVolume || hostvars.Image.Name != nil || hostvars.Image.Lookup != nil {
		t.Fatalf("%+v", hostvars.Image)
	}
	if hostvars.SecurityGroupsError == nil || hostvars.SecurityGroups != nil {
		t.Fatalf("security group failure must be recorded: %+v", hostvars)
	}
	if !hostvars.VolumeService || hostvars.VolumesError != nil || len(hostvars.Volumes) != 0 {
		t.Fatalf("%+v", hostvars)
	}
	for _, path := range hostvarsCallPaths(cloud) {
		if strings.Contains(path, "/flavors/") || strings.Contains(path, "/images") || strings.Contains(path, "ports") {
			t.Fatalf("unexpected lookup %s", path)
		}
	}
	raw, _ := json.Marshal(hostvars)
	if !strings.Contains(string(raw), `"image":{"id":""}`) || !strings.Contains(string(raw), `"name":"m1.tiny"`) {
		t.Fatalf("%s", raw)
	}
	groups := sdk.CloudServerGroups(hostvars)
	if !slices.Contains(groups, "flavor-m1.tiny") || slices.ContainsFunc(groups, func(g string) bool { return strings.HasPrefix(g, "image-") }) {
		t.Fatalf("%q", groups)
	}
}

func TestCloudServerHostvarsVolumeServiceAndLookupFailures(t *testing.T) {
	ctx := context.Background()
	// No block-storage endpoint: the volume step is skipped without error.
	cloud := hostvarsCloud(t, 0, 200)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":`+hostvarsServerS1+`}`)
	hostvars, err := cloud.Conn.GetOpenStackVars(ctx, hostvarsServer(t, hostvarsServerS1))
	if err != nil || hostvars.VolumeService || hostvars.VolumesError != nil || hostvars.Volumes == nil || len(hostvars.Volumes) != 0 {
		t.Fatalf("%+v %v", hostvars, err)
	}

	// A Cinder failure is recorded, not returned (Python swallows SDKException).
	failing := hostvarsCloud(t, 500, 200)
	failing.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":`+hostvarsServerS1+`}`)
	hostvars, err = failing.Conn.GetOpenStackVars(ctx, hostvarsServer(t, hostvarsServerS1))
	if err != nil || !hostvars.VolumeService || hostvars.VolumesError == nil || len(hostvars.Volumes) != 0 {
		t.Fatalf("%+v %v", hostvars, err)
	}

	// A missing image or flavor sets no name; a non-string flavor ID is invalid.
	server := hostvarsServer(t, hostvarsServerS1)
	server.Image = map[string]any{"id": "i-missing"}
	server.Flavor = map[string]any{"id": "missing", "original_name": "ignored"}
	failing.JSON("GET "+r2CloudComputeBase+"/flavors/missing", 404, `{"itemNotFound":{"message":"nf","code":404}}`)
	failing.JSON("GET "+r2CloudComputeBase+"/flavors/detail", 200, `{"flavors":[]}`)
	hostvars, err = failing.Conn.GetOpenStackVars(ctx, server)
	if err != nil || hostvars.Image.Name != nil || hostvars.Image.Lookup == nil || hostvars.Flavor.Name != nil || hostvars.Flavor.Lookup != nil {
		t.Fatalf("missing names: %+v %v", hostvars, err)
	}
	server.Flavor = map[string]any{"id": 7}
	if _, err = failing.Conn.GetOpenStackVars(ctx, server); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}

	// An image lookup failure is returned beside the partial hostvars.
	broken := hostvarsCloud(t, 200, 500)
	broken.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":`+hostvarsServerS1+`}`)
	hostvars, err = broken.Conn.GetOpenStackVars(ctx, hostvarsServer(t, hostvarsServerS1))
	if err == nil || hostvars == nil || hostvars.Interfaces == nil || hostvars.Flavor.Name == nil || hostvars.Volumes != nil {
		t.Fatalf("%+v %v", hostvars, err)
	}
	for _, path := range hostvarsCallPaths(broken) {
		if strings.Contains(path, "volumes") {
			t.Fatalf("volumes listed after a failed image lookup: %s", path)
		}
	}
}

func TestCloudServerHostvarsCancellationAndInvalidInputSendNothing(t *testing.T) {
	cloud := hostvarsCloud(t, 200, 200)
	server := hostvarsServer(t, hostvarsServerS1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cloud.Conn.GetOpenStackVars(ctx, server); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := cloud.Conn.GetServerMeta(ctx, server); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := cloud.Conn.GetOpenStackVars(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls := cloud.Calls(); len(calls) != 0 {
		t.Fatalf("%+v", calls)
	}
}

func TestCloudServerGetServerFindsWithDetailsAndExpands(t *testing.T) {
	cloud := hostvarsCloud(t, 200, 200)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":`+hostvarsServerS1+`}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/gone", 404, `{"itemNotFound":{"message":"nf","code":404}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, `{"servers":[`+hostvarsServerS1+`,`+
		`{"id":"s2","name":"web-2","status":"SHUTOFF","flavor":{},"image":{},"metadata":{},"addresses":{}}]}`)
	ctx := context.Background()
	if got, err := cloud.Conn.GetServer(ctx, "gone"); got != nil || err != nil {
		t.Fatal(got, err)
	}
	bare, err := cloud.Conn.GetServer(ctx, "s1", sdk.WithCloudServerBare(true))
	if err != nil || bare.Server.ID != "s1" || bare.Record == nil || bare.Interfaces != nil || bare.Hostvars != nil {
		t.Fatalf("%+v %v", bare, err)
	}
	plain, err := cloud.Conn.GetServer(ctx, "s1")
	if err != nil || plain.Interfaces == nil || plain.Interfaces.PublicIPv4 != "8.8.8.8" || plain.Hostvars != nil {
		t.Fatalf("%+v %v", plain, err)
	}
	detailed, err := cloud.Conn.GetServer(ctx, "s1", sdk.WithCloudServerDetailed(true))
	if err != nil || detailed.Hostvars == nil || detailed.Interfaces != detailed.Hostvars.Interfaces || string(detailed.Hostvars.Flavor.Name) != `"small"` {
		t.Fatalf("%+v %v", detailed, err)
	}
	view, err := sdk.CloudServerResultView(detailed)
	if err != nil || !strings.Contains(string(view), `"access_ipv4":"8.8.8.8"`) || !strings.Contains(string(view), `"flavor":{`) {
		t.Fatal(string(view), err)
	}
	// A name falls back from GET servers/{name} to the detailed list (details=True).
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/web-2", 404, `{"itemNotFound":{"message":"nf","code":404}}`)
	byName, err := cloud.Conn.GetServer(ctx, "web-2", sdk.WithCloudServerBare(true))
	if err != nil || byName == nil || byName.Server.ID != "s2" {
		t.Fatalf("%+v %v", byName, err)
	}
	calls := cloud.Calls()
	if last := calls[len(calls)-1]; last.Path != r2CloudComputeBase+"/servers/detail" {
		t.Fatalf("%+v", last)
	}
	// Deprecated filters: local selection over a bare list, one match required.
	one, err := cloud.Conn.GetServer(ctx, "web-*", sdk.WithCloudServerFilters(json.RawMessage(`{"status":"SHUTOFF"}`)))
	if err != nil || one == nil || one.Server.ID != "s2" || one.Interfaces == nil {
		t.Fatalf("%+v %v", one, err)
	}
	var ambiguous *resource.AmbiguousError
	if _, err := cloud.Conn.GetServer(ctx, "web-*", sdk.WithCloudServerFilters(json.RawMessage(`{}`))); !errors.As(err, &ambiguous) || len(ambiguous.IDs) != 2 {
		t.Fatal(err)
	}
	if got, err := cloud.Conn.GetServer(ctx, "none-*", sdk.WithCloudServerFilters(json.RawMessage(`{}`))); got != nil || err != nil {
		t.Fatal(got, err)
	}
	if _, err := cloud.Conn.GetServer(ctx, "", sdk.WithCloudServerExpression("[].name")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := cloud.Conn.GetServerByID(ctx, "s1", sdk.WithCloudServerDetailed(true)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestCloudServerListAndSearchServersExpandEveryRow(t *testing.T) {
	cloud := hostvarsCloud(t, 200, 200)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":`+hostvarsServerS1+`}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s2", 200, `{"server":{"id":"s2"}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s2/os-security-groups", 200, `{"security_groups":[]}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, `{"servers":[`+hostvarsServerS1+`,`+
		`{"id":"s2","name":"web-2","status":"SHUTOFF","flavor":{},"image":{},"metadata":{},"addresses":{}}]}`)
	ctx := context.Background()
	listed, err := cloud.Conn.ListServers(ctx)
	if err != nil || len(listed) != 2 || listed[0].Interfaces == nil || listed[1].Interfaces == nil || listed[0].Hostvars != nil {
		t.Fatalf("%+v %v", listed, err)
	}
	if listed[0].Interfaces.PublicIPv4 != "8.8.8.8" || listed[0].Record == nil {
		t.Fatalf("%+v", listed[0])
	}
	bare, err := cloud.Conn.ListServers(ctx, sdk.WithCloudServerBare(true), sdk.WithCloudServerListOptions(servers.WithRecordFindAllProjects(true)))
	if err != nil || len(bare) != 2 || bare[0].Interfaces != nil {
		t.Fatalf("%+v %v", bare, err)
	}
	detailed, err := cloud.Conn.ListServers(ctx, sdk.WithCloudServerDetailed(true))
	if err != nil || len(detailed) != 2 || detailed[0].Hostvars == nil || detailed[1].Hostvars == nil || detailed[1].Hostvars.Image.BootFromVolume || detailed[1].Hostvars.Image.ID != "" {
		t.Fatalf("%+v %v", detailed, err)
	}
	search, err := cloud.Conn.SearchServers(ctx, "web-*", sdk.WithCloudServerFilters(json.RawMessage(`{"public_v4":"8.8.8.8"}`)))
	if err != nil || len(search.Inventory) != 2 || len(search.Servers) != 1 || search.Servers[0].Server.ID != "s1" {
		t.Fatalf("%+v %v", search, err)
	}
	expression, err := cloud.Conn.SearchServers(ctx, "", sdk.WithCloudServerDetailed(true), sdk.WithCloudServerExpression("[].flavor.name"))
	if err != nil || !expression.Expression || string(expression.Value) != `["small"]` || len(expression.Servers) != 0 {
		t.Fatalf("%+v %s %v", expression, expression.Value, err)
	}
	var allProjects int
	for _, call := range cloud.Calls() {
		if call.Path == r2CloudComputeBase+"/servers/detail" {
			if call.Query == "all_tenants=True" {
				allProjects++
			} else if call.Query != "" {
				t.Fatalf("filters reached Nova: %+v", call)
			}
		}
	}
	if allProjects != 1 {
		t.Fatal(allProjects)
	}
}

func TestCloudServerUpdateAndRebuildApplyDefaultAndDetailedExpansion(t *testing.T) {
	cloud := hostvarsCloud(t, 200, 200)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":`+hostvarsServerS1+`}`)
	cloud.Handle("PUT "+r2CloudComputeBase+"/servers/s1", func(w http.ResponseWriter, _ *http.Request, _ string) {
		renamed := strings.Replace(hostvarsServerS1, `"web-1"`, `"renamed"`, 1)
		testcloud.JSON(w, 200, `{"server":`+renamed+`}`)
	})
	cloud.Handle("POST "+r2CloudComputeBase+"/servers/s1/action", func(w http.ResponseWriter, _ *http.Request, _ string) {
		testcloud.JSON(w, 202, `{"server":`+strings.Replace(hostvarsServerS1, `"id":"s1"`, `"id":"s1","adminPass":"pw"`, 1)+`}`)
	})
	ctx := context.Background()
	updated, err := cloud.Conn.UpdateServer(ctx, "s1", sdk.CloudServerUpdate{Name: request.Present("renamed")}, sdk.WithCloudServerDetailed(true))
	if err != nil || updated.Hostvars == nil || updated.Server.Name != "renamed" {
		t.Fatalf("%+v %v", updated, err)
	}
	raw, _ := json.Marshal(updated.Hostvars)
	if !strings.Contains(string(raw), `"name":"renamed"`) || strings.Contains(string(raw), `"name":"web-1"`) {
		t.Fatalf("hostvars must describe the PUT response: %s", raw)
	}
	plain, err := cloud.Conn.UpdateServer(ctx, "s1", sdk.CloudServerUpdate{Name: request.Present("renamed")})
	if err != nil || plain.Interfaces == nil || plain.Hostvars != nil || plain.Interfaces.PublicIPv4 != "8.8.8.8" {
		t.Fatalf("%+v %v", plain, err)
	}
	rebuilt, err := cloud.Conn.RebuildServer(ctx, "s1", "i1")
	if err != nil || rebuilt.Interfaces == nil || rebuilt.Interfaces.PublicIPv4 != "8.8.8.8" || rebuilt.Hostvars != nil || rebuilt.AdminPassword != "pw" {
		t.Fatalf("%+v %v", rebuilt, err)
	}
	detailed, err := cloud.Conn.RebuildServer(ctx, "s1", "i1", sdk.WithCloudServerActionDetailed(true))
	if err != nil || detailed.Hostvars == nil || string(detailed.Hostvars.Image.Name) != `"ubuntu"` || detailed.Interfaces != detailed.Hostvars.Interfaces {
		t.Fatalf("%+v %v", detailed, err)
	}
	var posts int
	for _, call := range cloud.Calls() {
		if call.Method == http.MethodPost {
			posts++
		}
	}
	if posts != 2 {
		t.Fatalf("rebuild must be sent once per call, got %d", posts)
	}
}
