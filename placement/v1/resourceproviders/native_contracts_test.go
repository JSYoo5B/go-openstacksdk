package resourceproviders_test

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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRPTransport func(*http.Request) (*http.Response, error)

func (transport nativeRPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeRPWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeRPCall struct{ method, path, query, body, version string }

func nativeRPAPI(t *testing.T, microversion string, calls *[]nativeRPCall, reply func(*http.Request) *http.Response) *resourceproviders.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeRPTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeRPCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = microversion
	return resourceproviders.New(client)
}

func nativeRPOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "resourceproviders" {
		t.Fatal("generated resourceproviders context", err, wrapped)
	}
}

func nativeRPInt(v int) *int          { return &v }
func nativeRPString(v string) *string { return &v }

const nativeRPRow = `{"generation":1,"uuid":"rp-1","name":"compute-1","parent_provider_uuid":"root-1","root_provider_uuid":"root-1","links":[{"href":"/resource_providers/rp-1","rel":"self"}]}`

func TestNativeResourceProvidersRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRPCall
	api := nativeRPAPI(t, "1.39", &calls, func(req *http.Request) *http.Response {
		path := strings.TrimPrefix(req.URL.Path, "/placement/resource_providers")
		switch {
		case req.Method == http.MethodDelete:
			return nativeRPWire(204, "")
		case path == "":
			if req.Method == http.MethodGet {
				return nativeRPWire(200, `{"resource_providers":[`+nativeRPRow+`,{"uuid":"rp-2","name":"compute-2"}]}`)
			}
			return nativeRPWire(200, nativeRPRow)
		case path == "/rp-1":
			return nativeRPWire(200, nativeRPRow)
		case path == "/rp-1/usages":
			return nativeRPWire(200, `{"resource_provider_generation":4,"usages":{"VCPU":2,"MEMORY_MB":512}}`)
		case path == "/rp-1/inventories":
			return nativeRPWire(200, `{"resource_provider_generation":4,"inventories":{"VCPU":{"allocation_ratio":16.0,"max_unit":8,"min_unit":1,"reserved":0,"step_size":1,"total":8}}}`)
		case path == "/rp-1/inventories/VCPU":
			return nativeRPWire(200, `{"resource_provider_generation":5,"allocation_ratio":1.5,"max_unit":8,"min_unit":1,"reserved":1,"step_size":1,"total":8}`)
		case path == "/rp-1/allocations":
			return nativeRPWire(200, `{"resource_provider_generation":4,"allocations":{"c-1":{"resources":{"VCPU":1}}}}`)
		case path == "/rp-1/traits":
			return nativeRPWire(200, `{"resource_provider_generation":6,"traits":["CUSTOM_A","HW_CPU_X86_AVX"]}`)
		case path == "/rp-1/aggregates":
			return nativeRPWire(200, `{"resource_provider_generation":7,"aggregates":["agg-1"]}`)
		}
		return nativeRPWire(500, `{}`)
	})
	created, err := api.Create(ctx, resourceproviders.CreateOpts{Name: "compute-1", UUID: "rp-1", ParentProviderUUID: "root-1"}, resourceproviders.WithCreateField("x_extension", 1))
	if err != nil || !(created.UUID == "rp-1" && created.Generation == 1 && created.RootProviderUUID == "root-1" && created.Links[0].Rel == "self") {
		t.Fatal(created, err)
	}
	// Name has no omitempty, so an empty create still sends it.
	if _, err := api.Create(ctx, resourceproviders.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "rp-1")
	if err != nil || got.Name != "compute-1" || got.ParentProviderUUID != "root-1" {
		t.Fatal(got, err)
	}
	if _, err := api.Update(ctx, "rp-1", resourceproviders.UpdateOpts{Name: nativeRPString("renamed"), ParentProviderUUID: nativeRPString("root-2")}); err != nil {
		t.Fatal(err)
	}
	// An empty parent pointer is rewritten to JSON null to make the provider a root.
	if _, err := api.Update(ctx, "rp-1", resourceproviders.UpdateOpts{ParentProviderUUID: nativeRPString("")}); err != nil {
		t.Fatal(err)
	}
	usage, err := api.GetUsages(ctx, "rp-1")
	if err != nil || usage.ResourceProviderGeneration != 4 || usage.Usages["MEMORY_MB"] != 512 {
		t.Fatal(usage, err)
	}
	inventories, err := api.GetInventories(ctx, "rp-1")
	if err != nil || inventories.Inventories["VCPU"].AllocationRatio != 16 || inventories.Inventories["VCPU"].Total != 8 {
		t.Fatal(inventories, err)
	}
	updatedInventories, err := api.UpdateInventories(ctx, "rp-1", resourceproviders.UpdateInventoriesOpts{
		ResourceProviderGeneration: 3,
		Inventories:                map[string]resourceproviders.Inventory{"VCPU": {AllocationRatio: 16, MaxUnit: 8, MinUnit: 1, StepSize: 1, Total: 8}},
	})
	if err != nil || updatedInventories.ResourceProviderGeneration != 4 {
		t.Fatal(updatedInventories, err)
	}
	inventory, err := api.GetInventory(ctx, "rp-1", "VCPU")
	if err != nil || inventory.ResourceProviderGeneration != 5 || inventory.AllocationRatio != 1.5 || inventory.Reserved != 1 {
		t.Fatal(inventory, err)
	}
	updatedInventory, err := api.UpdateInventory(ctx, "rp-1", "VCPU", resourceproviders.UpdateInventoryOpts{
		ResourceProviderGeneration: 4,
		Inventory:                  resourceproviders.Inventory{AllocationRatio: 1.5, MaxUnit: 8, MinUnit: 1, Reserved: 1, StepSize: 1, Total: 8},
	})
	if err != nil || updatedInventory.ResourceProviderGeneration != 5 {
		t.Fatal(updatedInventory, err)
	}
	allocations, err := api.GetAllocations(ctx, "rp-1")
	if err != nil || allocations.Allocations["c-1"].Resources["VCPU"] != 1 {
		t.Fatal(allocations, err)
	}
	traits, err := api.GetTraits(ctx, "rp-1")
	if err != nil || !reflect.DeepEqual(traits.Traits, []string{"CUSTOM_A", "HW_CPU_X86_AVX"}) || traits.ResourceProviderGeneration != 6 {
		t.Fatal(traits, err)
	}
	if _, err := api.UpdateTraits(ctx, "rp-1", resourceproviders.UpdateTraitsOpts{ResourceProviderGeneration: 5, Traits: []string{"CUSTOM_A"}}); err != nil {
		t.Fatal(err)
	}
	// Nil traits are sent as null rather than an empty list.
	if _, err := api.UpdateTraits(ctx, "rp-1", resourceproviders.UpdateTraitsOpts{ResourceProviderGeneration: 6}); err != nil {
		t.Fatal(err)
	}
	aggregates, err := api.GetAggregates(ctx, "rp-1")
	if err != nil || *aggregates.ResourceProviderGeneration != 7 || !reflect.DeepEqual(aggregates.Aggregates, []string{"agg-1"}) {
		t.Fatal(aggregates, err)
	}
	if _, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{ResourceProviderGeneration: nativeRPInt(6), Aggregates: []string{"agg-1"}}); err != nil {
		t.Fatal(err)
	}
	var names []string
	for value, err := range api.List(ctx, resourceproviders.WithListOptions(resourceproviders.ListOpts{
		Name: "compute-1", UUID: "rp-1", MemberOf: "in:agg-1,agg-2", Resources: "VCPU:1,MEMORY_MB:512", InTree: "root-1", Required: "CUSTOM_A,!CUSTOM_B",
	}), resourceproviders.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name)
	}
	if !reflect.DeepEqual(names, []string{"compute-1", "compute-2"}) {
		t.Fatal(names)
	}
	for name, err := range map[string]error{
		"DeleteInventory":   api.DeleteInventory(ctx, "rp-1", "VCPU"),
		"DeleteInventories": api.DeleteInventories(ctx, "rp-1"),
		"DeleteTraits":      api.DeleteTraits(ctx, "rp-1"),
		"Delete":            api.Delete(ctx, "rp-1"),
	} {
		if err != nil {
			t.Fatal(name, err)
		}
	}
	base := "/placement/resource_providers"
	want := []nativeRPCall{
		{http.MethodPost, base, "", `{"name":"compute-1","parent_provider_uuid":"root-1","uuid":"rp-1","x_extension":1}`, "placement 1.39"},
		{http.MethodPost, base, "", `{"name":""}`, "placement 1.39"},
		{http.MethodGet, base + "/rp-1", "", "", "placement 1.39"},
		{http.MethodPut, base + "/rp-1", "", `{"name":"renamed","parent_provider_uuid":"root-2"}`, "placement 1.39"},
		{http.MethodPut, base + "/rp-1", "", `{"parent_provider_uuid":null}`, "placement 1.39"},
		{http.MethodGet, base + "/rp-1/usages", "", "", "placement 1.39"},
		{http.MethodGet, base + "/rp-1/inventories", "", "", "placement 1.39"},
		// Inventory fields have no omitempty, so zero reserved is sent.
		{http.MethodPut, base + "/rp-1/inventories", "", `{"inventories":{"VCPU":{"allocation_ratio":16,"max_unit":8,"min_unit":1,"reserved":0,"step_size":1,"total":8}},"resource_provider_generation":3}`, "placement 1.39"},
		{http.MethodGet, base + "/rp-1/inventories/VCPU", "", "", "placement 1.39"},
		// The embedded Inventory is flattened next to the generation.
		{http.MethodPut, base + "/rp-1/inventories/VCPU", "", `{"allocation_ratio":1.5,"max_unit":8,"min_unit":1,"reserved":1,"resource_provider_generation":4,"step_size":1,"total":8}`, "placement 1.39"},
		{http.MethodGet, base + "/rp-1/allocations", "", "", "placement 1.39"},
		{http.MethodGet, base + "/rp-1/traits", "", "", "placement 1.39"},
		{http.MethodPut, base + "/rp-1/traits", "", `{"resource_provider_generation":5,"traits":["CUSTOM_A"]}`, "placement 1.39"},
		{http.MethodPut, base + "/rp-1/traits", "", `{"resource_provider_generation":6,"traits":null}`, "placement 1.39"},
		{http.MethodGet, base + "/rp-1/aggregates", "", "", "placement 1.39"},
		{http.MethodPut, base + "/rp-1/aggregates", "", `{"aggregates":["agg-1"],"resource_provider_generation":6}`, "placement 1.39"},
		{http.MethodGet, base, "extra=1&in_tree=root-1&member_of=in%3Aagg-1%2Cagg-2&name=compute-1&required=CUSTOM_A%2C%21CUSTOM_B&resources=VCPU%3A1%2CMEMORY_MB%3A512&uuid=rp-1", "", "placement 1.39"},
	}
	deletes := calls[len(want):]
	if !reflect.DeepEqual(calls[:len(want)], want) {
		t.Fatalf("%+v", calls)
	}
	var deletePaths []string
	for _, call := range deletes {
		if call.method != http.MethodDelete || call.body != "" || call.version != "placement 1.39" {
			t.Fatalf("%+v", call)
		}
		deletePaths = append(deletePaths, call.path)
	}
	slices.Sort(deletePaths)
	if !reflect.DeepEqual(deletePaths, []string{base + "/rp-1", base + "/rp-1/inventories", base + "/rp-1/inventories/VCPU", base + "/rp-1/traits"}) {
		t.Fatal(deletePaths)
	}
}

func TestNativeResourceProvidersMicroversionAndAggregatesBody(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		microversion, body, version string
	}{
		// Below 1.19 the native call sends the bare aggregate list and drops the generation and extensions.
		{"1.18", `["agg-1"]`, "placement 1.18"},
		{"1.19", `{"aggregates":["agg-1"],"resource_provider_generation":2,"x_extension":1}`, "placement 1.19"},
		// Without a microversion no header is sent and the enveloped body is kept.
		{"", `{"aggregates":["agg-1"],"resource_provider_generation":2,"x_extension":1}`, ""},
	} {
		var calls []nativeRPCall
		api := nativeRPAPI(t, tc.microversion, &calls, func(*http.Request) *http.Response {
			return nativeRPWire(200, `{"aggregates":["agg-1"]}`)
		})
		got, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{ResourceProviderGeneration: nativeRPInt(2), Aggregates: []string{"agg-1"}}, resourceproviders.WithUpdateAggregatesField("x_extension", 1))
		// The pre-1.19 response has no generation, which decodes as a nil pointer.
		if err != nil || got.ResourceProviderGeneration != nil || len(calls) != 1 || calls[0].body != tc.body || calls[0].version != tc.version {
			t.Fatal(tc.microversion, got, err, calls)
		}
	}
	t.Run("nil aggregates below 1.19 send no body", func(t *testing.T) {
		var calls []nativeRPCall
		api := nativeRPAPI(t, "1.1", &calls, func(*http.Request) *http.Response { return nativeRPWire(200, `{"aggregates":[]}`) })
		if _, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{}); err != nil || len(calls) != 1 || calls[0].body != "" {
			t.Fatal(err, calls)
		}
	})
	t.Run("unparsable microversion fails before HTTP", func(t *testing.T) {
		for _, microversion := range []string{"latest", "1", "1.x"} {
			var calls []nativeRPCall
			api := nativeRPAPI(t, microversion, &calls, func(*http.Request) *http.Response { return nativeRPWire(200, `{}`) })
			_, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{Aggregates: []string{"agg-1"}})
			nativeRPOperation(t, err, "UpdateAggregates")
			if len(calls) != 0 {
				t.Fatal(microversion, calls)
			}
		}
	})
	t.Run("other calls forward any microversion", func(t *testing.T) {
		var calls []nativeRPCall
		api := nativeRPAPI(t, "latest", &calls, func(*http.Request) *http.Response { return nativeRPWire(200, nativeRPRow) })
		if _, err := api.Get(ctx, "rp-1"); err != nil || calls[0].version != "placement latest" {
			t.Fatal(err, calls)
		}
	})
}

func TestNativeResourceProvidersStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*resourceproviders.API) error
	}{
		// Create accepts only 200, so the pre-1.20 bodyless 201 is an error.
		{"Create", []int{200}, func(api *resourceproviders.API) error {
			_, err := api.Create(ctx, resourceproviders.CreateOpts{Name: "rp"})
			return err
		}},
		{"Get", []int{200}, func(api *resourceproviders.API) error { _, err := api.Get(ctx, "rp-1"); return err }},
		{"Update", []int{200}, func(api *resourceproviders.API) error {
			_, err := api.Update(ctx, "rp-1", resourceproviders.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *resourceproviders.API) error { return api.Delete(ctx, "rp-1") }},
		{"GetUsages", []int{200}, func(api *resourceproviders.API) error { _, err := api.GetUsages(ctx, "rp-1"); return err }},
		{"GetInventories", []int{200}, func(api *resourceproviders.API) error { _, err := api.GetInventories(ctx, "rp-1"); return err }},
		{"UpdateInventories", []int{200}, func(api *resourceproviders.API) error {
			_, err := api.UpdateInventories(ctx, "rp-1", resourceproviders.UpdateInventoriesOpts{})
			return err
		}},
		{"DeleteInventories", []int{204}, func(api *resourceproviders.API) error { return api.DeleteInventories(ctx, "rp-1") }},
		{"GetInventory", []int{200}, func(api *resourceproviders.API) error {
			_, err := api.GetInventory(ctx, "rp-1", "VCPU")
			return err
		}},
		{"UpdateInventory", []int{200}, func(api *resourceproviders.API) error {
			_, err := api.UpdateInventory(ctx, "rp-1", "VCPU", resourceproviders.UpdateInventoryOpts{})
			return err
		}},
		{"DeleteInventory", []int{204}, func(api *resourceproviders.API) error { return api.DeleteInventory(ctx, "rp-1", "VCPU") }},
		{"GetAllocations", []int{200}, func(api *resourceproviders.API) error { _, err := api.GetAllocations(ctx, "rp-1"); return err }},
		{"GetTraits", []int{200}, func(api *resourceproviders.API) error { _, err := api.GetTraits(ctx, "rp-1"); return err }},
		{"UpdateTraits", []int{200}, func(api *resourceproviders.API) error {
			_, err := api.UpdateTraits(ctx, "rp-1", resourceproviders.UpdateTraitsOpts{})
			return err
		}},
		{"DeleteTraits", []int{204}, func(api *resourceproviders.API) error { return api.DeleteTraits(ctx, "rp-1") }},
		{"GetAggregates", []int{200}, func(api *resourceproviders.API) error { _, err := api.GetAggregates(ctx, "rp-1"); return err }},
		{"UpdateAggregates", []int{200}, func(api *resourceproviders.API) error {
			_, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{})
			return err
		}},
	} {
		for _, code := range []int{200, 201, 202, 204, 404, 409} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeRPCall
				api := nativeRPAPI(t, "1.39", &calls, func(*http.Request) *http.Response { return nativeRPWire(code, `{}`) })
				err := call.call(api)
				nativeRPOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("plain object decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// Responses are not enveloped, so {} and null decode as a zero provider.
			`{}`:              false,
			`null`:            false,
			`{"uuid":5}`:      true,
			`["not-an-item"]`: true,
		} {
			var calls []nativeRPCall
			api := nativeRPAPI(t, "1.39", &calls, func(*http.Request) *http.Response { return nativeRPWire(200, body) })
			got, err := api.Get(ctx, "rp-1")
			if wantErr {
				nativeRPOperation(t, err, "Get")
			} else if err != nil || got == nil || got.UUID != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"resource_providers":[]}`}, {200, `{}`}, {204, ""}, {200, `{"resource_providers":{}}`}} {
			var calls []nativeRPCall
			api := nativeRPAPI(t, "1.39", &calls, func(*http.Request) *http.Response { return nativeRPWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && tc.body != `{"resource_providers":{}}` && len(errs) == 0:
			case tc.code == 200 && len(errs) == 1 && errs[0] != nil:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, tc.body, errs)
			}
			if len(calls) != 1 || calls[0].path != "/placement/resource_providers" || calls[0].query != "" {
				t.Fatal(calls)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeRPCall
		api := nativeRPAPI(t, "1.39", &calls, func(*http.Request) *http.Response { return nativeRPWire(200, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, resourceproviders.CreateOpts{Name: "rp"}, resourceproviders.WithCreateField("uuid", "x"))
				return err
			}()},
			"create nil option": {"Create", func() error {
				_, err := api.Create(ctx, resourceproviders.CreateOpts{Name: "rp"}, nil)
				return err
			}()},
			// Omitted pointer fields are still reserved.
			"update core extension": {"Update", func() error {
				_, err := api.Update(ctx, "rp-1", resourceproviders.UpdateOpts{}, resourceproviders.WithUpdateField("parent_provider_uuid", nil))
				return err
			}()},
			"inventories core extension": {"UpdateInventories", func() error {
				_, err := api.UpdateInventories(ctx, "rp-1", resourceproviders.UpdateInventoriesOpts{}, resourceproviders.WithUpdateInventoriesField("inventories", nil))
				return err
			}()},
			// Embedded Inventory keys are reserved as well.
			"inventory embedded extension": {"UpdateInventory", func() error {
				_, err := api.UpdateInventory(ctx, "rp-1", "VCPU", resourceproviders.UpdateInventoryOpts{}, resourceproviders.WithUpdateInventoryField("total", 1))
				return err
			}()},
			"traits core extension": {"UpdateTraits", func() error {
				_, err := api.UpdateTraits(ctx, "rp-1", resourceproviders.UpdateTraitsOpts{}, resourceproviders.WithUpdateTraitsField("traits", nil))
				return err
			}()},
			"aggregates core extension": {"UpdateAggregates", func() error {
				_, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{}, resourceproviders.WithUpdateAggregatesField("resource_provider_generation", 1))
				return err
			}()},
			"aggregates nil option": {"UpdateAggregates", func() error {
				_, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeRPOperation(t, check.err, check.operation)
		}
		for name, options := range map[string][]resourceproviders.ListOption{
			"empty query key": {resourceproviders.WithListQuery(" ", "x")},
			"nil option":      {nil},
		} {
			var errs []error
			for _, err := range api.List(ctx, options...) {
				errs = append(errs, err)
			}
			if len(errs) != 1 {
				t.Fatal(name, errs)
			}
			nativeRPOperation(t, errs[0], "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
	t.Run("list query replaces native keys", func(t *testing.T) {
		var calls []nativeRPCall
		api := nativeRPAPI(t, "1.39", &calls, func(*http.Request) *http.Response { return nativeRPWire(200, `{"resource_providers":[]}`) })
		// ListOpts carries one member_of and one required string; WithListQuery replaces rather than appends.
		for range api.List(ctx, resourceproviders.WithListOptions(resourceproviders.ListOpts{MemberOf: "agg-1", Required: "CUSTOM_A"}), resourceproviders.WithListQuery("member_of", "agg-2")) {
		}
		if len(calls) != 1 || calls[0].query != "member_of=agg-2&required=CUSTOM_A" {
			t.Fatal(calls)
		}
	})
}
