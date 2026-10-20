package flavors_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonFlavorCall records the wire request openstacksdk would also send.
type pythonFlavorCall struct{ method, path, query, body, version string }

type pythonFlavorTransport func(*http.Request) (*http.Response, error)

func (transport pythonFlavorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonFlavorWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonFlavorAPI(t *testing.T, reply func(*http.Request) *http.Response) (*flavors.API, *[]pythonFlavorCall) {
	t.Helper()
	calls := &[]pythonFlavorCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonFlavorTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonFlavorCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("X-OpenStack-Nova-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.61"
	return flavors.New(client), calls
}

func pythonFlavorInt(value int) *int { return &value }

const pythonFlavorBase = "/nova/v2.1/flavors"

const pythonFlavorRow = `{"id":"f1","name":"m1","ram":512,"vcpus":1,"disk":10,"swap":0,"rxtx_factor":1.0,"os-flavor-access:is_public":false,"OS-FLV-EXT-DATA:ephemeral":5,"OS-FLV-DISABLED:disabled":false,"description":"d","extra_specs":{}}`

func TestPythonFlavorCreateDeleteMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	deleteStatus := http.StatusNotFound
	api, calls := pythonFlavorAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodDelete {
			return pythonFlavorWire(deleteStatus, `{"itemNotFound":{"message":"gone"}}`)
		}
		return pythonFlavorWire(200, `{"flavor":`+pythonFlavorRow+`}`)
	})
	public := false
	// create_flavor(name="m1", ram=512, vcpus=1, disk=10, is_public=False, ephemeral=5, description="d", id="f1")
	created, err := api.Create(ctx, flavors.CreateOpts{Name: "m1", RAM: 512, VCPUs: 1, Disk: pythonFlavorInt(10), IsPublic: &public, Ephemeral: pythonFlavorInt(5), Description: "d", ID: "f1"})
	if err != nil || created.ID != "f1" || created.Name != "m1" || created.RAM != 512 || created.VCPUs != 1 || created.Disk != 10 || created.IsPublic || created.Ephemeral != 5 || created.RxTxFactor != 1.0 || created.Description != "d" {
		t.Fatal(created, err)
	}
	// Keys outside CreateOpts go through WithCreateField.
	if _, err := api.Create(ctx, flavors.CreateOpts{Name: "m2", RAM: 256, VCPUs: 2, Disk: pythonFlavorInt(0)}, flavors.WithCreateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	// delete_flavor("f1") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("f1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("f1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "f1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	deleteStatus = http.StatusAccepted
	if err := api.Remove(ctx, resource.ID("f1")); err != nil {
		t.Fatal(err)
	}
	want := []pythonFlavorCall{
		{http.MethodPost, pythonFlavorBase, "", `{"flavor":{"OS-FLV-EXT-DATA:ephemeral":5,"description":"d","disk":10,"id":"f1","name":"m1","os-flavor-access:is_public":false,"ram":512,"vcpus":1}}`, "2.61"},
		{http.MethodPost, pythonFlavorBase, "", `{"flavor":{"disk":0,"name":"m2","ram":256,"vcpus":2,"x_extension":1}}`, "2.61"},
		{http.MethodDelete, pythonFlavorBase + "/f1", "", "", "2.61"},
		{http.MethodDelete, pythonFlavorBase + "/f1", "", "", "2.61"},
		{http.MethodDelete, pythonFlavorBase + "/f1", "", "", "2.61"},
		{http.MethodDelete, pythonFlavorBase + "/f1", "", "", "2.61"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonFlavorUpdateCannotClearDescription(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonFlavorAPI(t, func(*http.Request) *http.Response {
		return pythonFlavorWire(200, `{"flavor":`+pythonFlavorRow+`}`)
	})
	// update_flavor("f1", description="new")
	if got, err := api.Update(ctx, "f1", flavors.UpdateOpts{Description: "new"}); err != nil || got.ID != "f1" {
		t.Fatal(got, err)
	}
	// update_flavor("f1", description=None) or "" clears the description;
	// Go omits the empty typed key and refuses it as an extension.
	if _, err := api.Update(ctx, "f1", flavors.UpdateOpts{}, flavors.WithUpdateField("description", nil)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	want := []pythonFlavorCall{
		{http.MethodPut, pythonFlavorBase + "/f1", "", `{"flavor":{"description":"new"}}`, "2.61"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonFlavorTenantAccessMatchesProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonFlavorAPI(t, func(*http.Request) *http.Response {
		return pythonFlavorWire(200, `{"flavor_access":[{"flavor_id":"f1","tenant_id":"p1"}]}`)
	})
	// flavor_add_tenant_access("f1", "p1") and flavor_remove_tenant_access("f1", "p1")
	if _, err := api.AddAccess(ctx, "f1", flavors.AddAccessOpts{Tenant: "p1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.RemoveAccess(ctx, "f1", flavors.RemoveAccessOpts{Tenant: "p1"}); err != nil {
		t.Fatal(err)
	}
	// get_flavor_access("f1") returns the flavor_access rows.
	var rows []flavors.FlavorAccess
	for row, err := range api.ListAccesses(ctx, "f1") {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *row)
	}
	if !reflect.DeepEqual(rows, []flavors.FlavorAccess{{FlavorID: "f1", TenantID: "p1"}}) {
		t.Fatal(rows)
	}
	want := []pythonFlavorCall{
		{http.MethodPost, pythonFlavorBase + "/f1/action", "", `{"addTenantAccess":{"tenant":"p1"}}`, "2.61"},
		{http.MethodPost, pythonFlavorBase + "/f1/action", "", `{"removeTenantAccess":{"tenant":"p1"}}`, "2.61"},
		{http.MethodGet, pythonFlavorBase + "/f1/os-flavor-access", "", "", "2.61"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonFlavorExtraSpecsWriteRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonFlavorAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodPost:
			return pythonFlavorWire(200, `{"extra_specs":{"hw:cpu_policy":"dedicated","hw:numa_nodes":"1"}}`)
		case http.MethodPut:
			return pythonFlavorWire(200, `{"hw:numa_nodes":"2"}`)
		}
		return pythonFlavorWire(404, `{"itemNotFound":{"message":"missing"}}`)
	})
	// create_flavor_extra_specs("f1", {"hw:cpu_policy": "dedicated"}) merges and returns all specs.
	specs, err := api.CreateExtraSpecs(ctx, "f1", flavors.ExtraSpecsOpts{"hw:cpu_policy": "dedicated"})
	if err != nil || !reflect.DeepEqual(specs, map[string]string{"hw:cpu_policy": "dedicated", "hw:numa_nodes": "1"}) {
		t.Fatal(specs, err)
	}
	// update_flavor_extra_specs_property("f1", "hw:numa_nodes", "2") returns the value under the key.
	updated, err := api.UpdateExtraSpec(ctx, "f1", flavors.ExtraSpecsOpts{"hw:numa_nodes": "2"})
	if err != nil || updated["hw:numa_nodes"] != "2" {
		t.Fatal(updated, err)
	}
	// delete_flavor_extra_specs_property("f1", "gone") ignores 404 by default;
	// DeleteExtraSpec has no ignore-missing option and returns it.
	if err := api.DeleteExtraSpec(ctx, "f1", "gone"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonFlavorCall{
		{http.MethodPost, pythonFlavorBase + "/f1/os-extra_specs", "", `{"extra_specs":{"hw:cpu_policy":"dedicated"}}`, "2.61"},
		{http.MethodPut, pythonFlavorBase + "/f1/os-extra_specs/hw:numa_nodes", "", `{"hw:numa_nodes":"2"}`, "2.61"},
		{http.MethodDelete, pythonFlavorBase + "/f1/os-extra_specs/gone", "", "", "2.61"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
