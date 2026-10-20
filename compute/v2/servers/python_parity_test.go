package servers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonServerCall records the wire request openstacksdk would also send.
type pythonServerCall struct{ method, path, query, body, version string }

type pythonServerTransport func(*http.Request) (*http.Response, error)

func (transport pythonServerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonServerWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

// pythonServerCloud returns a Nova client (microversion 2.96) and a Glance
// client that share one recording transport.
func pythonServerCloud(t *testing.T, reply func(*http.Request) *http.Response) (*gophercloud.ServiceClient, *gophercloud.ServiceClient, *[]pythonServerCall) {
	t.Helper()
	calls := &[]pythonServerCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonServerTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
			req.Body = io.NopCloser(strings.NewReader(raw))
		}
		*calls = append(*calls, pythonServerCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("X-OpenStack-Nova-API-Version")})
		return reply(req), nil
	})
	compute := cloud.Client("compute", "/catalog/unused/")
	compute.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	compute.Microversion = "2.96"
	glance := cloud.Client("image", "/catalog/glance/")
	glance.ResourceBase = cloud.Server.URL + "/glance/v2/"
	return compute, glance, calls
}

const pythonServerBase = "/nova/v2.1/servers/s1"

const pythonServerRow = `{"id":"s1","name":"vm","status":"ACTIVE","tenant_id":"project","metadata":{"k":"v"},"accessIPv4":"10.0.0.5"}`

func TestPythonServerGetUpdateDeleteMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	status := http.StatusNotFound
	compute, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			return pythonServerWire(status, `{"itemNotFound":{"message":"gone"}}`)
		case http.MethodPost:
			return pythonServerWire(202, "")
		}
		return pythonServerWire(200, `{"server":`+pythonServerRow+`}`)
	})
	api := servers.New(compute)
	// get_server(server)
	got, err := api.Get(ctx, "s1")
	if err != nil || got.ID != "s1" || got.Name != "vm" || got.Status != "ACTIVE" || got.TenantID != "project" || got.Metadata["k"] != "v" || got.AccessIPv4 != "10.0.0.5" {
		t.Fatal(got, err)
	}
	// update_server(server, name="vm2", description="d") sends only the passed attributes.
	if _, err := api.Update(ctx, "s1", servers.UpdateOpts{Name: "vm2"}, servers.WithUpdateField("description", "d")); err != nil {
		t.Fatal(err)
	}
	// delete_server(server) ignores a missing server; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("s1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("s1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "s1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	status = http.StatusNoContent
	if err := api.Remove(ctx, resource.ID("s1")); err != nil {
		t.Fatal(err)
	}
	// delete_server(server, force=True) posts forceDelete; Python sends null, Gophercloud "".
	if err := api.ForceDelete(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonServerCall{
		{http.MethodGet, pythonServerBase, "", "", "2.96"},
		{http.MethodPut, pythonServerBase, "", `{"server":{"description":"d","name":"vm2"}}`, "2.96"},
		{http.MethodDelete, pythonServerBase, "", "", "2.96"},
		{http.MethodDelete, pythonServerBase, "", "", "2.96"},
		{http.MethodDelete, pythonServerBase, "", "", "2.96"},
		{http.MethodDelete, pythonServerBase, "", "", "2.96"},
		{http.MethodPost, pythonServerBase + "/action", "", `{"forceDelete":""}`, "2.96"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonServerActionsMatchProxyBodies(t *testing.T) {
	ctx := context.Background()
	compute, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		data, _ := io.ReadAll(req.Body)
		if strings.HasPrefix(string(data), `{"rescue"`) {
			return pythonServerWire(200, `{"adminPass":"rescue-pass"}`)
		}
		if strings.HasPrefix(string(data), `{"confirmResize"`) {
			return pythonServerWire(204, "")
		}
		return pythonServerWire(202, "")
	})
	api := servers.New(compute)
	steps := []struct {
		python, body string
		call         func() error
	}{
		{"change_server_password(server, 'pw')", `{"changePassword":{"adminPass":"pw"}}`, func() error { return api.ChangeAdminPassword(ctx, "s1", "pw") }},
		{"reboot_server(server, 'HARD')", `{"reboot":{"type":"HARD"}}`, func() error {
			return api.Reboot(ctx, "s1", servers.RebootOpts{Type: servers.RebootMethod("HARD")})
		}},
		{"resize_server(server, flavor)", `{"resize":{"flavorRef":"f2"}}`, func() error { return api.Resize(ctx, "s1", servers.ResizeOpts{FlavorRef: "f2"}) }},
		{"confirm_server_resize(server)", `{"confirmResize":null}`, func() error { return api.ConfirmResize(ctx, "s1") }},
		{"revert_server_resize(server)", `{"revertResize":null}`, func() error { return api.RevertResize(ctx, "s1") }},
		{"pause_server(server)", `{"pause":null}`, func() error { return api.Pause(ctx, "s1") }},
		{"unpause_server(server)", `{"unpause":null}`, func() error { return api.Unpause(ctx, "s1") }},
		{"suspend_server(server)", `{"suspend":null}`, func() error { return api.Suspend(ctx, "s1") }},
		{"resume_server(server)", `{"resume":null}`, func() error { return api.Resume(ctx, "s1") }},
		{"unlock_server(server)", `{"unlock":null}`, func() error { return api.Unlock(ctx, "s1") }},
		{"rescue_server(server)", `{"rescue":{}}`, func() error { _, err := api.Rescue(ctx, "s1", servers.RescueOpts{}); return err }},
		{"rescue_server(server, admin_pass='p', image='ri')", `{"rescue":{"adminPass":"p","rescue_image_ref":"ri"}}`, func() error {
			pass, err := api.Rescue(ctx, "s1", servers.RescueOpts{AdminPass: "p", RescueImageRef: "ri"})
			if err == nil && pass != "rescue-pass" {
				t.Errorf("rescue returned %q", pass)
			}
			return err
		}},
		{"unrescue_server(server)", `{"unrescue":null}`, func() error { return api.Unrescue(ctx, "s1") }},
		{"start_server(server)", `{"os-start":null}`, func() error { return api.Start(ctx, "s1") }},
		{"stop_server(server)", `{"os-stop":null}`, func() error { return api.Stop(ctx, "s1") }},
		{"shelve_server(server)", `{"shelve":null}`, func() error { return api.Shelve(ctx, "s1") }},
		{"shelve_offload_server(server)", `{"shelveOffload":null}`, func() error { return api.ShelveOffload(ctx, "s1") }},
	}
	for _, step := range steps {
		before := len(*calls)
		if err := step.call(); err != nil {
			t.Fatal(step.python, err)
		}
		if len(*calls) != before+1 {
			t.Fatalf("%s sent %d requests", step.python, len(*calls)-before)
		}
		got := (*calls)[before]
		if got != (pythonServerCall{http.MethodPost, pythonServerBase + "/action", "", step.body, "2.96"}) {
			t.Fatalf("%s: %+v", step.python, got)
		}
	}
}

func TestPythonServerRebuildSendsPassedKeysAndReturnsServer(t *testing.T) {
	ctx := context.Background()
	compute, _, calls := pythonServerCloud(t, func(*http.Request) *http.Response {
		return pythonServerWire(202, `{"server":{"id":"s1","name":"vm2","status":"REBUILD","adminPass":"np"}}`)
	})
	api := servers.New(compute)
	// rebuild_server(server, image, name=..., admin_password=..., preserve_ephemeral=True,
	// metadata=..., key_name=None, description=None, user_data=..., hostname=...)
	got, err := api.Rebuild(ctx, "s1", servers.RebuildOpts{ImageRef: "img", Name: "vm2", AdminPass: "p", Metadata: map[string]string{"k": "v"}},
		servers.WithRebuildField("preserve_ephemeral", true),
		servers.WithRebuildField("key_name", nil),
		servers.WithRebuildField("description", nil),
		servers.WithRebuildField("user_data", "dXNlcg=="),
		servers.WithRebuildField("hostname", "host2"))
	if err != nil || got.ID != "s1" || got.Name != "vm2" || got.Status != "REBUILD" || got.AdminPass != "np" {
		t.Fatal(got, err)
	}
	// rebuild_server(server, image) sends only imageRef.
	if _, err := api.Rebuild(ctx, "s1", servers.RebuildOpts{ImageRef: "img"}); err != nil {
		t.Fatal(err)
	}
	// A typed key cannot carry an explicit null, so name=None has no Go spelling.
	if _, err := api.Rebuild(ctx, "s1", servers.RebuildOpts{ImageRef: "img"}, servers.WithRebuildField("name", nil)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	want := []pythonServerCall{
		{http.MethodPost, pythonServerBase + "/action", "", `{"rebuild":{"adminPass":"p","description":null,"hostname":"host2","imageRef":"img","key_name":null,"metadata":{"k":"v"},"name":"vm2","preserve_ephemeral":true,"user_data":"dXNlcg=="}}`, "2.96"},
		{http.MethodPost, pythonServerBase + "/action", "", `{"rebuild":{"imageRef":"img"}}`, "2.96"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonServerLockAndUnshelveArgumentGaps(t *testing.T) {
	ctx := context.Background()
	compute, _, calls := pythonServerCloud(t, func(*http.Request) *http.Response { return pythonServerWire(202, "") })
	api := servers.New(compute)
	// lock_server(server) matches; lock_server(server, locked_reason=...) has no Go input.
	if err := api.Lock(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	// unshelve_server(server) and unshelve_server(server, availability_zone="az", host="h") match.
	if err := api.Unshelve(ctx, "s1", servers.UnshelveOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.Unshelve(ctx, "s1", servers.UnshelveOpts{AvailabilityZone: "az"}, servers.WithUnshelveField("host", "h")); err != nil {
		t.Fatal(err)
	}
	// unshelve_server(server, host="h") nests host in the action; Go puts it beside a null action.
	if err := api.Unshelve(ctx, "s1", servers.UnshelveOpts{}, servers.WithUnshelveField("host", "h")); err != nil {
		t.Fatal(err)
	}
	// unshelve_server(server, availability_zone=None) sends {"availability_zone": null}; Go rejects it.
	if err := api.Unshelve(ctx, "s1", servers.UnshelveOpts{}, servers.WithUnshelveField("availability_zone", nil)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	action := pythonServerBase + "/action"
	want := []pythonServerCall{
		{http.MethodPost, action, "", `{"lock":null}`, "2.96"},
		{http.MethodPost, action, "", `{"unshelve":null}`, "2.96"},
		{http.MethodPost, action, "", `{"unshelve":{"availability_zone":"az","host":"h"}}`, "2.96"},
		{http.MethodPost, action, "", `{"host":"h","unshelve":null}`, "2.96"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonServerCreateImageFetchesAndWaitsForGlanceImage(t *testing.T) {
	ctx := context.Background()
	statuses := []string{"queued", "saving", "active"}
	compute, glance, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			if req.Header.Get("X-OpenStack-Nova-API-Version") == "2.1" {
				response := pythonServerWire(202, "")
				response.Header.Set("X-OpenStack-Nova-API-Version", "2.1")
				response.Header.Set("Location", "http://glance/v2/images/img0")
				return response
			}
			response := pythonServerWire(202, `{"image_id":"img1"}`)
			response.Header.Set("X-OpenStack-Nova-API-Version", "2.45")
			return response
		}
		status := statuses[0]
		if len(statuses) > 1 {
			statuses = statuses[1:]
		}
		return pythonServerWire(200, `{"id":"img1","name":"snap","status":"`+status+`"}`)
	})
	compute.Microversion = "2.45"
	api, glanceAPI := servers.New(compute), images.New(glance)
	// create_server_image(server, "snap", metadata={"a": "b"}) with 2.45 reads image_id from the body,
	// then get_image(image_id) looks the image up in Glance.
	id, err := api.CreateImage(ctx, "s1", servers.CreateImageOpts{Name: "snap", Metadata: map[string]string{"a": "b"}})
	if err != nil || id != "img1" {
		t.Fatal(id, err)
	}
	image, err := glanceAPI.FindIdentity(ctx, id)
	if err != nil || image.ID != "img1" || image.Status != "queued" {
		t.Fatal(image, err)
	}
	// wait=True polls until active, failing on error, within the timeout.
	image, err = glanceAPI.WaitFor(ctx, resource.ID(id), "active", resource.WithTimeout(120*time.Second), resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("error"))
	if err != nil || image.Status != "active" {
		t.Fatal(image, err)
	}
	// Below 2.45 the image ID comes from the Location header, as in Python.
	compute.Microversion = "2.1"
	if id, err := api.CreateImage(ctx, "s1", servers.CreateImageOpts{Name: "snap"}); err != nil || id != "img0" {
		t.Fatal(id, err)
	}
	want := []pythonServerCall{
		{http.MethodPost, pythonServerBase + "/action", "", `{"createImage":{"metadata":{"a":"b"},"name":"snap"}}`, "2.45"},
		{http.MethodGet, "/glance/v2/images/img1", "", "", ""},
		{http.MethodGet, "/glance/v2/images/img1", "", "", ""},
		{http.MethodGet, "/glance/v2/images/img1", "", "", ""},
		{http.MethodPost, pythonServerBase + "/action", "", `{"createImage":{"name":"snap"}}`, "2.1"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}

	statuses = []string{"error"}
	if _, err := glanceAPI.WaitFor(ctx, resource.ID("img1"), "active", resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("error")); !errors.Is(err, resource.ErrFailedState) {
		t.Fatal(err)
	}
	statuses = []string{"saving"}
	if _, err := glanceAPI.WaitFor(ctx, resource.ID("img1"), "active", resource.WithTimeout(20*time.Millisecond), resource.WithPollInterval(time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestPythonServerMetadataSetAndDeleteMatchMixinRequests(t *testing.T) {
	ctx := context.Background()
	compute, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodDelete {
			return pythonServerWire(204, "")
		}
		return pythonServerWire(200, `{"metadata":{"old":"1","k":"v"}}`)
	})
	api := servers.New(compute)
	// set_server_metadata(server, k="v") merges with POST.
	merged, err := api.UpdateMetadata(ctx, "s1", servers.MetadataOpts{"k": "v"})
	if err != nil || !reflect.DeepEqual(merged, map[string]string{"old": "1", "k": "v"}) {
		t.Fatal(merged, err)
	}
	// delete_server_metadata(server, keys=["old"]) deletes each key.
	if err := api.DeleteMetadatum(ctx, "s1", "old"); err != nil {
		t.Fatal(err)
	}
	// delete_server_metadata(server) replaces the whole set with {}.
	if _, err := api.ResetMetadata(ctx, "s1", servers.MetadataOpts{}); err != nil {
		t.Fatal(err)
	}
	want := []pythonServerCall{
		{http.MethodPost, pythonServerBase + "/metadata", "", `{"metadata":{"k":"v"}}`, "2.96"},
		{http.MethodDelete, pythonServerBase + "/metadata/old", "", "", "2.96"},
		{http.MethodPut, pythonServerBase + "/metadata", "", `{"metadata":{}}`, "2.96"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonServerIPsListAllAndByNetworkLabel(t *testing.T) {
	ctx := context.Background()
	compute, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/ips") {
			return pythonServerWire(200, `{"addresses":{"private":[{"addr":"10.0.0.5","version":4}],"public":[{"addr":"2001:db8::5","version":6}]}}`)
		}
		return pythonServerWire(200, `{"private":[{"addr":"10.0.0.5","version":4}]}`)
	})
	api := servers.New(compute)
	// server_ips(server) yields every address with its network label.
	var all []map[string][]servers.Address
	for value, err := range api.ListAddresses(ctx, "s1") {
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, value)
	}
	if len(all) != 1 || !reflect.DeepEqual(all[0], map[string][]servers.Address{"private": {{Version: 4, Address: "10.0.0.5"}}, "public": {{Version: 6, Address: "2001:db8::5"}}}) {
		t.Fatalf("%+v", all)
	}
	// server_ips(server, network_label="private")
	var rows []servers.Address
	for value, err := range api.ListAddressesByNetwork(ctx, "s1", "private") {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	if !reflect.DeepEqual(rows, []servers.Address{{Version: 4, Address: "10.0.0.5"}}) {
		t.Fatalf("%+v", rows)
	}
	want := []pythonServerCall{
		{http.MethodGet, pythonServerBase + "/ips", "", "", "2.96"},
		{http.MethodGet, pythonServerBase + "/ips/private", "", "", "2.96"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
