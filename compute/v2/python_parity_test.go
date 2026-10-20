package v2_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	v2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonCloudCall records the wire request the openstacksdk cloud helper would also send.
type pythonCloudCall struct{ method, path, query, body string }

type pythonCloudTransport func(*http.Request) (*http.Response, error)

func (transport pythonCloudTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonCloudWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

// pythonCloudService returns the Nova v2 service (microversion 2.96) and a
// Glance images API sharing one recording transport.
func pythonCloudService(t *testing.T, reply func(*http.Request) *http.Response) (*v2.Service, *images.API, *[]pythonCloudCall) {
	t.Helper()
	calls := &[]pythonCloudCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonCloudTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
			req.Body = io.NopCloser(strings.NewReader(raw))
		}
		*calls = append(*calls, pythonCloudCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	compute := cloud.Client("compute", "/catalog/unused/")
	compute.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	compute.Microversion = "2.96"
	glance := cloud.Client("image", "/catalog/glance/")
	glance.ResourceBase = cloud.Server.URL + "/glance/v2/"
	return v2.New(compute), images.New(glance), calls
}

func pythonCloudCheck(t *testing.T, calls *[]pythonCloudCall, want []pythonCloudCall) {
	t.Helper()
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("got  %+v\nwant %+v", *calls, want)
	}
}

func pythonCloudInt(value int) *int { return &value }

func pythonCloudBool(value bool) *bool { return &value }

const pythonCloudNova = "/nova/v2.1"

func TestPythonCloudFlavorAccessAndSpecHelpers(t *testing.T) {
	ctx := context.Background()
	svc, _, calls := pythonCloudService(t, func(req *http.Request) *http.Response {
		switch {
		case strings.HasSuffix(req.URL.Path, "/os-extra_specs/gone"):
			return pythonCloudWire(404, `{"itemNotFound":{"message":"missing"}}`)
		case strings.HasSuffix(req.URL.Path, "/os-extra_specs/busy"):
			return pythonCloudWire(409, `{"conflictingRequest":{"message":"busy"}}`)
		case req.Method == http.MethodDelete:
			return pythonCloudWire(200, "")
		case strings.HasSuffix(req.URL.Path, "/os-extra_specs"):
			return pythonCloudWire(200, `{"extra_specs":{"hw:cpu_policy":"dedicated"}}`)
		}
		return pythonCloudWire(200, `{"flavor_access":[{"flavor_id":"f1","tenant_id":"p1"}]}`)
	})
	// add_flavor_access("f1", "p1") and remove_flavor_access("f1", "p1")
	if _, err := svc.Flavors.AddAccess(ctx, "f1", flavors.AddAccessOpts{Tenant: "p1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Flavors.RemoveAccess(ctx, "f1", flavors.RemoveAccessOpts{Tenant: "p1"}); err != nil {
		t.Fatal(err)
	}
	// list_flavor_access("f1") returns the flavor_access rows.
	var rows []flavors.FlavorAccess
	for row, err := range svc.Flavors.ListAccesses(ctx, "f1") {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *row)
	}
	if !reflect.DeepEqual(rows, []flavors.FlavorAccess{{FlavorID: "f1", TenantID: "p1"}}) {
		t.Fatal(rows)
	}
	// set_flavor_specs("f1", {"hw:cpu_policy": "dedicated"})
	if specs, err := svc.Flavors.CreateExtraSpecs(ctx, "f1", flavors.ExtraSpecsOpts{"hw:cpu_policy": "dedicated"}); err != nil || specs["hw:cpu_policy"] != "dedicated" {
		t.Fatal(specs, err)
	}
	// unset_flavor_specs("f1", ["old", "gone", "busy"]) ignores 404 per key and stops on other errors.
	var unsetErr error
	for _, key := range []string{"old", "gone", "busy", "never"} {
		if unsetErr = resource.IgnoreMissing(svc.Flavors.DeleteExtraSpec(ctx, "f1", key)); unsetErr != nil {
			break
		}
	}
	if !gophercloud.ResponseCodeIs(unsetErr, http.StatusConflict) {
		t.Fatal(unsetErr)
	}
	base := pythonCloudNova + "/flavors/f1"
	pythonCloudCheck(t, calls, []pythonCloudCall{
		{http.MethodPost, base + "/action", "", `{"addTenantAccess":{"tenant":"p1"}}`},
		{http.MethodPost, base + "/action", "", `{"removeTenantAccess":{"tenant":"p1"}}`},
		{http.MethodGet, base + "/os-flavor-access", "", ""},
		{http.MethodPost, base + "/os-extra_specs", "", `{"extra_specs":{"hw:cpu_policy":"dedicated"}}`},
		{http.MethodDelete, base + "/os-extra_specs/old", "", ""},
		{http.MethodDelete, base + "/os-extra_specs/gone", "", ""},
		{http.MethodDelete, base + "/os-extra_specs/busy", "", ""},
	})
}

func TestPythonCloudFlavorCreateNameAndDelete(t *testing.T) {
	ctx := context.Background()
	svc, _, calls := pythonCloudService(t, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodPost:
			return pythonCloudWire(200, `{"flavor":{"id":"f1","name":"small","ram":512,"vcpus":1,"disk":10,"swap":0,"OS-FLV-EXT-DATA:ephemeral":0,"rxtx_factor":1.0,"os-flavor-access:is_public":true}}`)
		case req.Method == http.MethodDelete:
			return pythonCloudWire(202, "")
		case req.URL.Path == pythonCloudNova+"/flavors/f1":
			return pythonCloudWire(200, `{"flavor":{"id":"f1","name":"small","ram":512,"vcpus":1,"disk":10}}`)
		case req.URL.Path == pythonCloudNova+"/flavors/detail":
			return pythonCloudWire(200, `{"flavors":[{"id":"f1","name":"small","ram":512,"vcpus":1,"disk":10}]}`)
		}
		return pythonCloudWire(404, `{"itemNotFound":{"message":"missing"}}`)
	})
	// create_flavor("small", 512, 1, 10) with flavorid="auto", ephemeral=0, swap=0,
	// rxtx_factor=1.0 and is_public=True.
	created, err := svc.Flavors.Create(ctx, flavors.CreateOpts{Name: "small", RAM: 512, VCPUs: 1, Disk: pythonCloudInt(10), Swap: pythonCloudInt(0), Ephemeral: pythonCloudInt(0), RxTxFactor: 1.0, IsPublic: pythonCloudBool(true)})
	if err != nil || created.ID != "f1" || created.Name != "small" || created.RAM != 512 || !created.IsPublic {
		t.Fatal(created, err)
	}
	// An explicit flavorid and description are sent as given.
	if _, err := svc.Flavors.Create(ctx, flavors.CreateOpts{Name: "small", RAM: 512, VCPUs: 1, Disk: pythonCloudInt(10), ID: "f1", Description: "d", Swap: pythonCloudInt(0), Ephemeral: pythonCloudInt(0), RxTxFactor: 1.0, IsPublic: pythonCloudBool(false)}); err != nil {
		t.Fatal(err)
	}
	// get_flavor_name("f1") reads the flavor without extra specs.
	if flavor, err := svc.Flavors.FindIdentity(ctx, "f1"); err != nil || flavor == nil || flavor.Name != "small" {
		t.Fatal(flavor, err)
	}
	// get_flavor_name("missing") returns None after the list fallback.
	if flavor, err := svc.Flavors.FindIdentity(ctx, "missing"); err != nil || flavor != nil {
		t.Fatal(flavor, err)
	}
	// delete_flavor("small") finds by name, then deletes with ignore_missing=True.
	flavor, err := svc.Flavors.FindIdentity(ctx, "small")
	if err != nil || flavor == nil || flavor.ID != "f1" {
		t.Fatal(flavor, err)
	}
	if err := svc.Flavors.Remove(ctx, resource.ID(flavor.ID)); err != nil {
		t.Fatal(err)
	}
	base := pythonCloudNova + "/flavors"
	pythonCloudCheck(t, calls, []pythonCloudCall{
		{http.MethodPost, base, "", `{"flavor":{"OS-FLV-EXT-DATA:ephemeral":0,"disk":10,"name":"small","os-flavor-access:is_public":true,"ram":512,"rxtx_factor":1,"swap":0,"vcpus":1}}`},
		{http.MethodPost, base, "", `{"flavor":{"OS-FLV-EXT-DATA:ephemeral":0,"description":"d","disk":10,"id":"f1","name":"small","os-flavor-access:is_public":false,"ram":512,"rxtx_factor":1,"swap":0,"vcpus":1}}`},
		{http.MethodGet, base + "/f1", "", ""},
		{http.MethodGet, base + "/missing", "", ""},
		{http.MethodGet, base + "/detail", "is_public=None", ""},
		{http.MethodGet, base + "/small", "", ""},
		{http.MethodGet, base + "/detail", "is_public=None", ""},
		{http.MethodDelete, base + "/f1", "", ""},
	})
}

func TestPythonCloudAggregateCreateAndList(t *testing.T) {
	ctx := context.Background()
	svc, _, calls := pythonCloudService(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet {
			return pythonCloudWire(200, `{"aggregates":[{"id":7,"name":"rack1","hosts":[]},{"id":8,"name":"rack2","availability_zone":"az1","hosts":["cmp1"]}]}`)
		}
		return pythonCloudWire(200, `{"aggregate":{"id":7,"name":"rack1","availability_zone":"az1","hosts":[]}}`)
	})
	// create_aggregate("rack1") and create_aggregate("rack1", availability_zone="az1")
	if created, err := svc.Aggregates.Create(ctx, aggregates.CreateOpts{Name: "rack1"}); err != nil || created.ID != 7 {
		t.Fatal(created, err)
	}
	if _, err := svc.Aggregates.Create(ctx, aggregates.CreateOpts{Name: "rack1", AvailabilityZone: "az1"}); err != nil {
		t.Fatal(err)
	}
	// list_aggregates() and list_aggregates(filters={"name": "rack2"}), filtered locally.
	all, err := svc.Aggregates.All(ctx)
	if err != nil || len(all) != 2 {
		t.Fatal(all, err)
	}
	named, err := svc.Aggregates.All(ctx, resource.WithName("rack2"))
	if err != nil || len(named) != 1 || named[0].ID != 8 || !reflect.DeepEqual(named[0].Hosts, []string{"cmp1"}) {
		t.Fatal(named, err)
	}
	base := pythonCloudNova + "/os-aggregates"
	pythonCloudCheck(t, calls, []pythonCloudCall{
		{http.MethodPost, base, "", `{"aggregate":{"name":"rack1"}}`},
		{http.MethodPost, base, "", `{"aggregate":{"availability_zone":"az1","name":"rack1"}}`},
		{http.MethodGet, base, "", ""},
		{http.MethodGet, base, "", ""},
	})
}

func TestPythonCloudServerGroupHelpers(t *testing.T) {
	ctx := context.Background()
	deleteStatus := http.StatusNoContent
	svc, _, calls := pythonCloudService(t, func(req *http.Request) *http.Response {
		base := pythonCloudNova + "/os-server-groups"
		switch {
		case req.Method == http.MethodPost:
			return pythonCloudWire(200, `{"server_group":{"id":"g1","name":"web","policy":"affinity","members":[]}}`)
		case req.Method == http.MethodDelete:
			return pythonCloudWire(deleteStatus, `{"itemNotFound":{"message":"gone"}}`)
		case req.URL.Path == base:
			return pythonCloudWire(200, `{"server_groups":[{"id":"g1","name":"web","policy":"affinity","members":[]}]}`)
		case req.URL.Path == base+"/g1":
			return pythonCloudWire(200, `{"server_group":{"id":"g1","name":"web","policy":"affinity","members":[]}}`)
		}
		return pythonCloudWire(404, `{"itemNotFound":{"message":"missing"}}`)
	})
	// create_server_group("web", policies=["affinity"]) at 2.64 or later sends the first policy as policy.
	created, err := svc.ServerGroups.Create(ctx, servergroups.CreateOpts{Name: "web", Policy: "affinity"})
	if err != nil || created.ID != "g1" || created.Policy == nil || *created.Policy != "affinity" {
		t.Fatal(created, err)
	}
	// list_server_groups()
	var names []string
	for group, err := range svc.ServerGroups.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, group.Name)
	}
	if !reflect.DeepEqual(names, []string{"web"}) {
		t.Fatal(names)
	}
	// get_server_group("g1") without filters is find_server_group.
	if group, err := svc.ServerGroups.FindRecord(ctx, "g1"); err != nil || group == nil || string(group.Resource.Body["name"]) != `"web"` {
		t.Fatal(group, err)
	}
	// delete_server_group("web"): find by name, then a strict DELETE.
	group, err := svc.ServerGroups.FindRecord(ctx, "web")
	if err != nil || group == nil || string(group.Resource.Body["id"]) != `"g1"` {
		t.Fatal(group, err)
	}
	if err := svc.ServerGroups.Delete(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
	// ignore_missing=False: a 404 on DELETE is an error, as NotFoundException is in Python.
	deleteStatus = http.StatusNotFound
	if err := svc.ServerGroups.Delete(ctx, "g1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	// A missing group returns False before any DELETE.
	if group, err := svc.ServerGroups.FindRecord(ctx, "missing"); err != nil || group != nil {
		t.Fatal(group, err)
	}
	base := pythonCloudNova + "/os-server-groups"
	pythonCloudCheck(t, calls, []pythonCloudCall{
		{http.MethodPost, base, "", `{"server_group":{"name":"web","policy":"affinity"}}`},
		{http.MethodGet, base, "", ""},
		{http.MethodGet, base + "/g1", "", ""},
		{http.MethodGet, base + "/web", "", ""},
		{http.MethodGet, base, "", ""},
		{http.MethodDelete, base + "/g1", "", ""},
		{http.MethodDelete, base + "/g1", "", ""},
		{http.MethodGet, base + "/missing", "", ""},
		{http.MethodGet, base, "", ""},
	})
}

// pythonCloudServerReply answers find_server(details=True) with a name fallback
// and the follow-up server, metadata and security group requests.
func pythonCloudServerReply(req *http.Request) *http.Response {
	switch {
	case req.URL.Path == pythonCloudNova+"/servers/detail":
		if req.URL.Query().Get("name") == "vm" {
			return pythonCloudWire(200, `{"servers":[{"id":"s1","name":"vm","status":"ACTIVE"}]}`)
		}
		return pythonCloudWire(200, `{"servers":[]}`)
	case req.URL.Path == pythonCloudNova+"/servers/s1":
		if req.Method == http.MethodDelete {
			return pythonCloudWire(404, `{"itemNotFound":{"message":"gone"}}`)
		}
		return pythonCloudWire(200, `{"server":{"id":"s1","name":"vm","status":"ACTIVE"}}`)
	case strings.HasPrefix(req.URL.Path, pythonCloudNova+"/servers/s1/metadata"):
		if req.Method == http.MethodDelete {
			return pythonCloudWire(204, "")
		}
		return pythonCloudWire(200, `{"metadata":{"k":"v","old":"1"}}`)
	case req.URL.Path == pythonCloudNova+"/servers/s1/os-security-groups":
		return pythonCloudWire(200, `{"security_groups":[{"id":"sg-1","name":"web","description":"d","tenant_id":"p","rules":[]}]}`)
	case req.URL.Path == pythonCloudNova+"/servers/s1/action":
		if strings.Contains(pythonCloudBody(req), "os-getConsoleOutput") {
			return pythonCloudWire(400, `{"badRequest":{"message":"unsupported"}}`)
		}
		response := pythonCloudWire(202, `{"image_id":"img1"}`)
		response.Header.Set("X-OpenStack-Nova-API-Version", "2.96")
		return response
	}
	return pythonCloudWire(404, `{"itemNotFound":{"message":"missing"}}`)
}

func pythonCloudBody(req *http.Request) string {
	if req.Body == nil {
		return ""
	}
	data, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(strings.NewReader(string(data)))
	return string(data)
}

func pythonCloudServerID(t *testing.T, svc *v2.Service, nameOrID string) string {
	t.Helper()
	record, err := svc.Servers.FindRecord(context.Background(), nameOrID)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil {
		return ""
	}
	id := string(record.Resource.Body["id"])
	return strings.Trim(id, `"`)
}

func TestPythonCloudServerLookupHelpers(t *testing.T) {
	ctx := context.Background()
	svc, _, calls := pythonCloudService(t, pythonCloudServerReply)
	// get_server_id("vm") is get_server(bare=True): GET by ID, then the detailed list by name.
	if id := pythonCloudServerID(t, svc, "vm"); id != "s1" {
		t.Fatal(id)
	}
	// get_server_id("missing") returns None.
	if id := pythonCloudServerID(t, svc, "missing"); id != "" {
		t.Fatal(id)
	}
	// set_server_metadata("vm", {"k": "v"}) merges with POST on the found ID.
	id := pythonCloudServerID(t, svc, "vm")
	if merged, err := svc.Servers.UpdateMetadata(ctx, id, servers.MetadataOpts{"k": "v"}); err != nil || merged["k"] != "v" {
		t.Fatal(merged, err)
	}
	// delete_server_metadata("vm", ["old"]) deletes each key on the found ID.
	id = pythonCloudServerID(t, svc, "vm")
	if err := svc.Servers.DeleteMetadatum(ctx, id, "old"); err != nil {
		t.Fatal(err)
	}
	// list_server_security_groups("s1"): get_server by ID, then its os-security-groups.
	server, err := svc.Servers.Get(ctx, "s1")
	if err != nil || server.ID != "s1" {
		t.Fatal(server, err)
	}
	var groups []string
	for group, err := range svc.SecurityGroups.ListByServer(ctx, server.ID) {
		if err != nil {
			t.Fatal(err)
		}
		groups = append(groups, group.Name)
	}
	if !reflect.DeepEqual(groups, []string{"web"}) {
		t.Fatal(groups)
	}
	serverBase := pythonCloudNova + "/servers"
	lookup := []pythonCloudCall{
		{http.MethodGet, serverBase + "/vm", "", ""},
		{http.MethodGet, serverBase + "/detail", "name=vm", ""},
	}
	want := append([]pythonCloudCall{}, lookup...)
	want = append(want,
		pythonCloudCall{http.MethodGet, serverBase + "/missing", "", ""},
		pythonCloudCall{http.MethodGet, serverBase + "/detail", "name=missing", ""},
	)
	want = append(want, lookup...)
	want = append(want, pythonCloudCall{http.MethodPost, serverBase + "/s1/metadata", "", `{"metadata":{"k":"v"}}`})
	want = append(want, lookup...)
	want = append(want,
		pythonCloudCall{http.MethodDelete, serverBase + "/s1/metadata/old", "", ""},
		pythonCloudCall{http.MethodGet, serverBase + "/s1", "", ""},
		pythonCloudCall{http.MethodGet, serverBase + "/s1/os-security-groups", "", ""},
	)
	pythonCloudCheck(t, calls, want)
}

func TestPythonCloudCreateImageSnapshotFindsServerThenImage(t *testing.T) {
	ctx := context.Background()
	statuses := []string{"queued", "saving", "active"}
	svc, glance, calls := pythonCloudService(t, func(req *http.Request) *http.Response {
		if strings.HasPrefix(req.URL.Path, "/glance/") {
			status := statuses[0]
			if len(statuses) > 1 {
				statuses = statuses[1:]
			}
			return pythonCloudWire(200, `{"id":"img1","name":"snap","status":"`+status+`"}`)
		}
		return pythonCloudServerReply(req)
	})
	// create_image_snapshot("snap", "vm", wait=True, owner="ops"): get_server(bare=True),
	// createImage, get_image(image_id) and wait_for_image with a 3600 second budget.
	id := pythonCloudServerID(t, svc, "vm")
	imageID, err := svc.Servers.CreateImage(ctx, id, servers.CreateImageOpts{Name: "snap", Metadata: map[string]string{"owner": "ops"}})
	if err != nil || imageID != "img1" {
		t.Fatal(imageID, err)
	}
	image, err := glance.FindIdentity(ctx, imageID)
	if err != nil || image == nil || image.Status != "queued" {
		t.Fatal(image, err)
	}
	image, err = glance.WaitFor(ctx, resource.ID(imageID), "active", resource.WithTimeout(3600*time.Second), resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("error"))
	if err != nil || image.Status != "active" {
		t.Fatal(image, err)
	}
	// The image error state fails the wait, as SDKException does in Python.
	statuses = []string{"error"}
	if _, err := glance.WaitFor(ctx, resource.ID(imageID), "active", resource.WithTimeout(3600*time.Second), resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("error")); !errors.Is(err, resource.ErrFailedState) {
		t.Fatal(err)
	}
	pythonCloudCheck(t, calls, []pythonCloudCall{
		{http.MethodGet, pythonCloudNova + "/servers/vm", "", ""},
		{http.MethodGet, pythonCloudNova + "/servers/detail", "name=vm", ""},
		{http.MethodPost, pythonCloudNova + "/servers/s1/action", "", `{"createImage":{"metadata":{"owner":"ops"},"name":"snap"}}`},
		{http.MethodGet, "/glance/v2/images/img1", "", ""},
		{http.MethodGet, "/glance/v2/images/img1", "", ""},
		{http.MethodGet, "/glance/v2/images/img1", "", ""},
		{http.MethodGet, "/glance/v2/images/img1", "", ""},
	})
}

// The following pin the parts of unresolved helpers that Go reproduces and the
// point where the Python helper adds behavior Go does not have.
func TestPythonCloudServerDeleteAndConsolePartialPaths(t *testing.T) {
	ctx := context.Background()
	svc, _, calls := pythonCloudService(t, pythonCloudServerReply)
	// delete_server("vm") with wait=False and delete_ips=False: find, then DELETE ignoring 404.
	id := pythonCloudServerID(t, svc, "vm")
	if err := svc.Servers.Remove(ctx, resource.ID(id)); err != nil {
		t.Fatal(err)
	}
	// get_server_console("vm"): Python returns "" on 400; Go returns the 400 error.
	id = pythonCloudServerID(t, svc, "vm")
	if output, err := svc.Servers.ConsoleOutput(ctx, id); output != "" || !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) {
		t.Fatal(output, err)
	}
	serverBase := pythonCloudNova + "/servers"
	pythonCloudCheck(t, calls, []pythonCloudCall{
		{http.MethodGet, serverBase + "/vm", "", ""},
		{http.MethodGet, serverBase + "/detail", "name=vm", ""},
		{http.MethodDelete, serverBase + "/s1", "", ""},
		{http.MethodGet, serverBase + "/vm", "", ""},
		{http.MethodGet, serverBase + "/detail", "name=vm", ""},
		{http.MethodPost, serverBase + "/s1/action", "", `{"os-getConsoleOutput":{}}`},
	})
}

func TestPythonCloudComputeLimitsCurrentProject(t *testing.T) {
	ctx := context.Background()
	svc, _, calls := pythonCloudService(t, func(*http.Request) *http.Response {
		return pythonCloudWire(200, `{"limits":{"absolute":{"maxTotalCores":20,"totalCoresUsed":4,"maxTotalInstances":10},"rate":[]}}`)
	})
	// get_compute_limits() without name_or_id: GET /limits and return the absolute limits.
	limits, err := svc.Limits.Fetch(ctx)
	if err != nil || limits.Absolute.MaxTotalCores != 20 || limits.Absolute.TotalCoresUsed != 4 || limits.Absolute.MaxTotalInstances != 10 {
		t.Fatal(limits, err)
	}
	pythonCloudCheck(t, calls, []pythonCloudCall{{http.MethodGet, pythonCloudNova + "/limits", "", ""}})
}
