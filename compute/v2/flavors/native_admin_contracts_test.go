package flavors_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeFlavorAdminTransport func(*http.Request) (*http.Response, error)

func (transport nativeFlavorAdminTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeFlavorAdminWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeFlavorAdminCall struct{ method, path, query, body string }

func nativeFlavorAdminAPI(t *testing.T, calls *[]nativeFlavorAdminCall, reply func(*http.Request) *http.Response) *flavors.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeFlavorAdminTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeFlavorAdminCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return flavors.New(client)
}

func nativeFlavorAdminOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "flavors" {
		t.Fatal("generated flavors context", err, wrapped)
	}
}

func nativeFlavorAdminInt(value int) *int { return &value }

const nativeFlavorAdminRow = `{"id":"f1","name":"m1","ram":512,"vcpus":1,"disk":10,"swap":"","rxtx_factor":1.5,"os-flavor-access:is_public":false,"OS-FLV-EXT-DATA:ephemeral":0,"description":"d","extra_specs":{"hw:cpu_policy":"dedicated"}}`

func TestNativeFlavorAdminRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeFlavorAdminCall
	api := nativeFlavorAdminAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete && strings.Contains(req.URL.Path, "/os-extra_specs/"):
			return nativeFlavorAdminWire(200, "")
		case req.Method == http.MethodDelete:
			return nativeFlavorAdminWire(202, "")
		case strings.HasSuffix(req.URL.Path, "/action") && strings.Contains(calls[len(calls)-1].body, "removeTenantAccess"):
			return nativeFlavorAdminWire(200, `{"flavor_access":[]}`)
		case strings.HasSuffix(req.URL.Path, "/action") || strings.HasSuffix(req.URL.Path, "/os-flavor-access"):
			return nativeFlavorAdminWire(200, `{"flavor_access":[{"flavor_id":"f1","tenant_id":"p1"},{"flavor_id":"f1","tenant_id":"p2"}]}`)
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/os-extra_specs"):
			return nativeFlavorAdminWire(200, `{"extra_specs":{"hw:cpu_policy":"dedicated","hw:numa_nodes":"1"}}`)
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/os-extra_specs/"):
			// A single extra spec answers with the bare key/value object.
			return nativeFlavorAdminWire(200, `{"hw:cpu_policy":"shared"}`)
		case req.Method == http.MethodPut:
			return nativeFlavorAdminWire(200, `{"flavor":{"id":"f1","name":"m1","swap":512,"description":"new"}}`)
		}
		return nativeFlavorAdminWire(201, `{"flavor":`+nativeFlavorAdminRow+`}`)
	})
	public := false
	created, err := api.Create(ctx, flavors.CreateOpts{
		Name: "m1", RAM: 512, VCPUs: 1, Disk: nativeFlavorAdminInt(10), ID: "f1", Swap: nativeFlavorAdminInt(0),
		RxTxFactor: 1.5, IsPublic: &public, Ephemeral: nativeFlavorAdminInt(0), Description: "d",
	}, flavors.WithCreateField("x_extension", 1))
	// The string swap "" decodes to 0 through Flavor.UnmarshalJSON.
	if err != nil || !(created.ID == "f1" && created.RAM == 512 && created.Disk == 10 && created.Swap == 0 && !created.IsPublic &&
		created.RxTxFactor == 1.5 && created.ExtraSpecs["hw:cpu_policy"] == "dedicated") {
		t.Fatal(created, err)
	}
	// The required tags reject only an empty Name and a nil Disk; zero RAM and VCPUs are sent.
	if _, err := api.Create(ctx, flavors.CreateOpts{Name: "m0", Disk: nativeFlavorAdminInt(0)}); err != nil {
		t.Fatal(err)
	}
	updated, err := api.Update(ctx, "f1", flavors.UpdateOpts{Description: "new"}, flavors.WithUpdateField("x_extension", true))
	if err != nil || updated.Description != "new" || updated.Swap != 512 {
		t.Fatal(updated, err)
	}
	// An empty description is omitted, leaving an empty flavor envelope.
	if _, err := api.Update(ctx, "f1", flavors.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "f1"); err != nil {
		t.Fatal(err)
	}
	added, err := api.AddAccess(ctx, "f1", flavors.AddAccessOpts{Tenant: "p1"}, flavors.WithAddAccessField("x_extension", "a"))
	if err != nil || !reflect.DeepEqual(added, []flavors.FlavorAccess{{FlavorID: "f1", TenantID: "p1"}, {FlavorID: "f1", TenantID: "p2"}}) {
		t.Fatal(added, err)
	}
	// Tenant has no required tag, so an empty tenant is sent as "".
	if _, err := api.AddAccess(ctx, "f1", flavors.AddAccessOpts{}); err != nil {
		t.Fatal(err)
	}
	removed, err := api.RemoveAccess(ctx, "f1", flavors.RemoveAccessOpts{Tenant: "p1"})
	if err != nil || removed == nil || len(removed) != 0 {
		t.Fatal(removed, err)
	}
	var accesses []string
	for value, err := range api.ListAccesses(ctx, "f1") {
		if err != nil {
			t.Fatal(err)
		}
		accesses = append(accesses, value.FlavorID+"/"+value.TenantID)
	}
	if !reflect.DeepEqual(accesses, []string{"f1/p1", "f1/p2"}) {
		t.Fatal(accesses)
	}
	specs, err := api.CreateExtraSpecs(ctx, "f1", flavors.ExtraSpecsOpts{"hw:cpu_policy": "dedicated", "hw:numa_nodes": "1"}, flavors.WithCreateExtraSpecsField("x_extension", "1"))
	if err != nil || !reflect.DeepEqual(specs, map[string]string{"hw:cpu_policy": "dedicated", "hw:numa_nodes": "1"}) {
		t.Fatal(specs, err)
	}
	spec, err := api.UpdateExtraSpec(ctx, "f1", flavors.ExtraSpecsOpts{"hw:cpu_policy": "shared"})
	if err != nil || !reflect.DeepEqual(spec, map[string]string{"hw:cpu_policy": "shared"}) {
		t.Fatal(spec, err)
	}
	if err := api.DeleteExtraSpec(ctx, "f1", "hw:cpu_policy"); err != nil {
		t.Fatal(err)
	}
	base := "/nova/v2.1/flavors"
	want := []nativeFlavorAdminCall{
		{http.MethodPost, base, "", `{"flavor":{"OS-FLV-EXT-DATA:ephemeral":0,"description":"d","disk":10,"id":"f1","name":"m1","os-flavor-access:is_public":false,"ram":512,"rxtx_factor":1.5,"swap":0,"vcpus":1,"x_extension":1}}`},
		{http.MethodPost, base, "", `{"flavor":{"disk":0,"name":"m0","ram":0,"vcpus":0}}`},
		{http.MethodPut, base + "/f1", "", `{"flavor":{"description":"new","x_extension":true}}`},
		{http.MethodPut, base + "/f1", "", `{"flavor":{}}`},
		{http.MethodDelete, base + "/f1", "", ""},
		{http.MethodPost, base + "/f1/action", "", `{"addTenantAccess":{"tenant":"p1","x_extension":"a"}}`},
		{http.MethodPost, base + "/f1/action", "", `{"addTenantAccess":{"tenant":""}}`},
		{http.MethodPost, base + "/f1/action", "", `{"removeTenantAccess":{"tenant":"p1"}}`},
		{http.MethodGet, base + "/f1/os-flavor-access", "", ""},
		// The extra_specs value is a typed map, so the extension lands beside it at the top level.
		{http.MethodPost, base + "/f1/os-extra_specs", "", `{"extra_specs":{"hw:cpu_policy":"dedicated","hw:numa_nodes":"1"},"x_extension":"1"}`},
		{http.MethodPut, base + "/f1/os-extra_specs/hw:cpu_policy", "", `{"hw:cpu_policy":"shared"}`},
		{http.MethodDelete, base + "/f1/os-extra_specs/hw:cpu_policy", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeFlavorAdminStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	createOpts := flavors.CreateOpts{Name: "m1", RAM: 1, VCPUs: 1, Disk: nativeFlavorAdminInt(1)}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*flavors.API) error
	}{
		{"Create", []int{200, 201}, func(api *flavors.API) error { _, err := api.Create(ctx, createOpts); return err }},
		{"Update", []int{200}, func(api *flavors.API) error { _, err := api.Update(ctx, "f1", flavors.UpdateOpts{}); return err }},
		{"Delete", []int{202, 204}, func(api *flavors.API) error { return api.Delete(ctx, "f1") }},
		{"AddAccess", []int{200}, func(api *flavors.API) error {
			_, err := api.AddAccess(ctx, "f1", flavors.AddAccessOpts{Tenant: "p"})
			return err
		}},
		{"RemoveAccess", []int{200}, func(api *flavors.API) error {
			_, err := api.RemoveAccess(ctx, "f1", flavors.RemoveAccessOpts{Tenant: "p"})
			return err
		}},
		{"CreateExtraSpecs", []int{200}, func(api *flavors.API) error {
			_, err := api.CreateExtraSpecs(ctx, "f1", flavors.ExtraSpecsOpts{"k": "v"})
			return err
		}},
		{"UpdateExtraSpec", []int{200}, func(api *flavors.API) error {
			_, err := api.UpdateExtraSpec(ctx, "f1", flavors.ExtraSpecsOpts{"k": "v"})
			return err
		}},
		// Deleting one extra spec accepts only 200, unlike the flavor Delete defaults.
		{"DeleteExtraSpec", []int{200}, func(api *flavors.API) error { return api.DeleteExtraSpec(ctx, "f1", "k") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeFlavorAdminCall
				api := nativeFlavorAdminAPI(t, &calls, func(*http.Request) *http.Response { return nativeFlavorAdminWire(code, `{}`) })
				err := call.call(api)
				nativeFlavorAdminOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeFlavorAdminCall
		api := nativeFlavorAdminAPI(t, &calls, func(*http.Request) *http.Response { return nativeFlavorAdminWire(200, `{}`) })
		create := func(opts flavors.CreateOpts, options ...flavors.CreateOption) error {
			_, err := api.Create(ctx, opts, options...)
			return err
		}
		updateSpec := func(opts flavors.ExtraSpecsOpts, options ...flavors.UpdateExtraSpecOption) error {
			_, err := api.UpdateExtraSpec(ctx, "f1", opts, options...)
			return err
		}
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create name": {"Create", create(flavors.CreateOpts{RAM: 1, VCPUs: 1, Disk: nativeFlavorAdminInt(1)})},
			"create disk": {"Create", create(flavors.CreateOpts{Name: "m", RAM: 1, VCPUs: 1})},
			// Omitted optional keys are still reserved for typed input.
			"create core extension": {"Create", create(createOpts, flavors.WithCreateField("swap", 1))},
			"create nil option":     {"Create", create(createOpts, nil)},
			"update core extension": {"Update", func() error {
				_, err := api.Update(ctx, "f1", flavors.UpdateOpts{}, flavors.WithUpdateField("description", "x"))
				return err
			}()},
			"add tenant extension": {"AddAccess", func() error {
				_, err := api.AddAccess(ctx, "f1", flavors.AddAccessOpts{Tenant: "p"}, flavors.WithAddAccessField("tenant", "q"))
				return err
			}()},
			"remove tenant extension": {"RemoveAccess", func() error {
				_, err := api.RemoveAccess(ctx, "f1", flavors.RemoveAccessOpts{Tenant: "p"}, flavors.WithRemoveAccessField("tenant", "q"))
				return err
			}()},
			"extra_specs extension": {"CreateExtraSpecs", func() error {
				_, err := api.CreateExtraSpecs(ctx, "f1", flavors.ExtraSpecsOpts{"k": "v"}, flavors.WithCreateExtraSpecsField("extra_specs", nil))
				return err
			}()},
			"update spec none": {"UpdateExtraSpec", updateSpec(flavors.ExtraSpecsOpts{})},
			"update spec two":  {"UpdateExtraSpec", updateSpec(flavors.ExtraSpecsOpts{"a": "1", "b": "2"})},
			"update spec nil":  {"UpdateExtraSpec", updateSpec(flavors.ExtraSpecsOpts{"a": "1"}, nil)},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeFlavorAdminOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}

func TestNativeFlavorAdminAccessListStatuses(t *testing.T) {
	ctx := context.Background()
	collect := func(api *flavors.API) ([]flavors.FlavorAccess, error) {
		var values []flavors.FlavorAccess
		for value, err := range api.ListAccesses(ctx, "f1") {
			if err != nil {
				return values, err
			}
			values = append(values, *value)
		}
		return values, nil
	}
	t.Run("single page ignores links", func(t *testing.T) {
		var calls []nativeFlavorAdminCall
		api := nativeFlavorAdminAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeFlavorAdminWire(200, `{"flavor_access":[{"flavor_id":"f1","tenant_id":"p1"}],"flavor_access_links":[{"rel":"next","href":"http://other/next"}]}`)
		})
		values, err := collect(api)
		if err != nil || len(values) != 1 || len(calls) != 1 {
			t.Fatal(values, err, calls)
		}
	})
	t.Run("empty", func(t *testing.T) {
		var calls []nativeFlavorAdminCall
		api := nativeFlavorAdminAPI(t, &calls, func(*http.Request) *http.Response { return nativeFlavorAdminWire(200, `{"flavor_access":[]}`) })
		values, err := collect(api)
		if err != nil || len(values) != 0 || len(calls) != 1 {
			t.Fatal(values, err, calls)
		}
	})
	t.Run("bodyless 204", func(t *testing.T) {
		var calls []nativeFlavorAdminCall
		api := nativeFlavorAdminAPI(t, &calls, func(*http.Request) *http.Response { return nativeFlavorAdminWire(204, "") })
		// The JSON content type makes the native page parser fail on the empty body.
		if _, err := collect(api); !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
	})
	for _, code := range []int{201, 202, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeFlavorAdminCall
			api := nativeFlavorAdminAPI(t, &calls, func(*http.Request) *http.Response { return nativeFlavorAdminWire(code, `{}`) })
			_, err := collect(api)
			var native gophercloud.ErrUnexpectedResponseCode
			var wrapped *resource.OperationError
			// List streams carry no request.Wrap context.
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200, 204, 300}) || errors.As(err, &wrapped) {
				t.Fatal(err)
			}
		})
	}
}
