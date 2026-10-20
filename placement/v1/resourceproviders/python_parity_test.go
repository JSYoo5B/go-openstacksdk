package resourceproviders_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonRPTransport func(*http.Request) (*http.Response, error)

func (transport pythonRPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonRPWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type pythonRPCall struct{ method, path, query, body, version string }

type pythonRPRecorder struct {
	mu    sync.Mutex
	calls []pythonRPCall
}

func (r *pythonRPRecorder) snapshot() []pythonRPCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pythonRPCall(nil), r.calls...)
}

// pythonRPAPI pins the microversion that the Python resource class would negotiate.
func pythonRPAPI(t *testing.T, microversion string, recorder *pythonRPRecorder, reply func(*http.Request) *http.Response) *resourceproviders.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonRPTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, pythonRPCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		recorder.mu.Unlock()
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = microversion
	return resourceproviders.New(client)
}

func pythonRPString(v string) *string { return &v }
func pythonRPInt(v int) *int          { return &v }

const pythonRPBase = "/placement/resource_providers"

const pythonRPRow = `{"generation":1,"uuid":"rp-1","name":"compute-1","parent_provider_uuid":"root-1","root_provider_uuid":"root-1","links":[{"href":"/resource_providers/rp-1","rel":"self"}]}`

func pythonRPExpect(t *testing.T, got, want []pythonRPCall) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("calls\n got %+v\nwant %+v", got, want)
	}
}

// ResourceProvider declares _max_microversion 1.20, so every proxy call below is pinned to placement 1.20.
func TestPythonResourceProviderProxyCRUD(t *testing.T) {
	ctx := context.Background()
	t.Run("create_get_update_list", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, func(req *http.Request) *http.Response {
			if req.Method == http.MethodGet && req.URL.Path == pythonRPBase {
				return pythonRPWire(200, `{"resource_providers":[`+pythonRPRow+`]}`)
			}
			return pythonRPWire(200, pythonRPRow)
		})
		// create_resource_provider(name="compute-1", id="rp-1", parent_provider_id="root-1")
		created, err := api.Create(ctx, resourceproviders.CreateOpts{Name: "compute-1", UUID: "rp-1", ParentProviderUUID: "root-1"})
		if err != nil || created.UUID != "rp-1" || created.Generation != 1 || created.RootProviderUUID != "root-1" || created.Links[0].Rel != "self" {
			t.Fatal(created, err)
		}
		// get_resource_provider("rp-1")
		got, err := api.Get(ctx, "rp-1")
		if err != nil || got.Name != "compute-1" || got.ParentProviderUUID != "root-1" {
			t.Fatal(got, err)
		}
		// update_resource_provider("rp-1", name="renamed"); Python strips the dirty uuid from the PUT body.
		if updated, err := api.Update(ctx, "rp-1", resourceproviders.UpdateOpts{Name: pythonRPString("renamed")}); err != nil || updated.UUID != "rp-1" {
			t.Fatal(updated, err)
		}
		// update_resource_provider("rp-1", name="renamed", parent_provider_id=None)
		if _, err := api.Update(ctx, "rp-1", resourceproviders.UpdateOpts{Name: pythonRPString("renamed"), ParentProviderUUID: pythonRPString("")}); err != nil {
			t.Fatal(err)
		}
		// resource_providers(name=..., member_of=..., resources=..., in_tree=..., required=..., id="rp-1")
		var names []string
		for value, err := range api.List(ctx, resourceproviders.WithListOptions(resourceproviders.ListOpts{
			Name: "compute-1", UUID: "rp-1", MemberOf: "in:agg-1,agg-2", Resources: "VCPU:1", InTree: "root-1", Required: "CUSTOM_A",
		})) {
			if err != nil {
				t.Fatal(err)
			}
			names = append(names, value.Name)
		}
		if !reflect.DeepEqual(names, []string{"compute-1"}) {
			t.Fatal(names)
		}
		// resource_providers() sends no query at all.
		all, err := api.All(ctx)
		if err != nil || len(all) != 1 || all[0].UUID != "rp-1" {
			t.Fatal(all, err)
		}
		pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{
			{http.MethodPost, pythonRPBase, "", `{"name":"compute-1","parent_provider_uuid":"root-1","uuid":"rp-1"}`, "placement 1.20"},
			{http.MethodGet, pythonRPBase + "/rp-1", "", "", "placement 1.20"},
			{http.MethodPut, pythonRPBase + "/rp-1", "", `{"name":"renamed"}`, "placement 1.20"},
			{http.MethodPut, pythonRPBase + "/rp-1", "", `{"name":"renamed","parent_provider_uuid":null}`, "placement 1.20"},
			{http.MethodGet, pythonRPBase, "in_tree=root-1&member_of=in%3Aagg-1%2Cagg-2&name=compute-1&required=CUSTOM_A&resources=VCPU%3A1&uuid=rp-1", "", "placement 1.20"},
			{http.MethodGet, pythonRPBase, "", "", "placement 1.20"},
		})
	})
	t.Run("delete_resource_provider ignore_missing", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			code    int
			options []resource.LookupOption
			missing bool
		}{
			{"default ignores 404", 404, nil, false},
			{"ignore_missing=True", 404, []resource.LookupOption{resource.WithIgnoreMissing()}, false},
			{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, true},
			{"deleted", 204, nil, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var recorder pythonRPRecorder
				api := pythonRPAPI(t, "1.20", &recorder, func(*http.Request) *http.Response { return pythonRPWire(tc.code, `{}`) })
				err := api.Remove(ctx, resource.ID("rp-1"), tc.options...)
				if tc.missing != errors.Is(err, resource.ErrNotFound) || (!tc.missing && err != nil) {
					t.Fatal(err)
				}
				pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{{http.MethodDelete, pythonRPBase + "/rp-1", "", "", "placement 1.20"}})
			})
		}
	})
	t.Run("get_resource_provider missing", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, func(*http.Request) *http.Response { return pythonRPWire(404, `{}`) })
		_, err := api.Get(ctx, "rp-1")
		if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
	})
}

// find_resource_provider tries GET by ID, then lists with name=<value> and returns None when ignore_missing.
func TestPythonResourceProviderFindIDThenName(t *testing.T) {
	ctx := context.Background()
	find := func(api *resourceproviders.API, value string) (*resourceproviders.ResourceProvider, error) {
		found, err := api.Find(ctx, resource.ID(value), resource.WithIgnoreMissing())
		if err != nil && !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) && !gophercloud.ResponseCodeIs(err, http.StatusForbidden) {
			return nil, err
		}
		if found != nil {
			return found, nil
		}
		return api.Find(ctx, resource.Name(value), resource.WithIgnoreMissing())
	}
	for _, tc := range []struct {
		name     string
		getCode  int
		listBody string
		wantUUID string
		calls    int
	}{
		{"id hit", 200, "", "rp-1", 1},
		{"name fallback after 404", 404, `{"resource_providers":[` + pythonRPRow + `]}`, "rp-1", 2},
		// Python also falls back after 400 and 403; Go Find(ID) reports them, so the caller continues explicitly.
		{"name fallback after 400", 400, `{"resource_providers":[` + pythonRPRow + `]}`, "rp-1", 2},
		{"name fallback after 403", 403, `{"resource_providers":[` + pythonRPRow + `]}`, "rp-1", 2},
		{"missing returns nil", 404, `{"resource_providers":[]}`, "", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var recorder pythonRPRecorder
			api := pythonRPAPI(t, "1.20", &recorder, func(req *http.Request) *http.Response {
				if req.URL.Path == pythonRPBase {
					return pythonRPWire(200, tc.listBody)
				}
				return pythonRPWire(tc.getCode, pythonRPRow)
			})
			found, err := find(api, "compute-1")
			if err != nil || (tc.wantUUID == "") != (found == nil) || (found != nil && found.UUID != tc.wantUUID) {
				t.Fatal(found, err)
			}
			want := []pythonRPCall{
				{http.MethodGet, pythonRPBase + "/compute-1", "", "", "placement 1.20"},
				{http.MethodGet, pythonRPBase, "name=compute-1", "", "placement 1.20"},
			}
			pythonRPExpect(t, recorder.snapshot(), want[:tc.calls])
		})
	}
	t.Run("ignore_missing=False", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, func(req *http.Request) *http.Response {
			if req.URL.Path == pythonRPBase {
				return pythonRPWire(200, `{"resource_providers":[]}`)
			}
			return pythonRPWire(404, `{}`)
		})
		if _, err := api.Find(ctx, resource.ID("compute-1")); !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
		if _, err := api.Find(ctx, resource.Name("compute-1")); !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
	})
}

func TestPythonResourceProviderAggregatesUsagesAllocationsTraits(t *testing.T) {
	ctx := context.Background()
	t.Run("aggregates at 1.20", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, func(req *http.Request) *http.Response {
			if req.Method == http.MethodPut {
				return pythonRPWire(200, `{"resource_provider_generation":8,"aggregates":["agg-2"]}`)
			}
			return pythonRPWire(200, `{"resource_provider_generation":7,"aggregates":["agg-1"]}`)
		})
		// fetch_resource_provider_aggregates and the deprecated get_resource_provider_aggregates
		got, err := api.GetAggregates(ctx, "rp-1")
		if err != nil || got.ResourceProviderGeneration == nil || *got.ResourceProviderGeneration != 7 || !reflect.DeepEqual(got.Aggregates, []string{"agg-1"}) {
			t.Fatal(got, err)
		}
		// set_resource_provider_aggregates(rp, "agg-2") with rp.generation == 7
		set, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{ResourceProviderGeneration: pythonRPInt(7), Aggregates: []string{"agg-2"}})
		if err != nil || *set.ResourceProviderGeneration != 8 || !reflect.DeepEqual(set.Aggregates, []string{"agg-2"}) {
			t.Fatal(set, err)
		}
		// set_resource_provider_aggregates(rp) sends an empty list; Go needs a non-nil empty slice.
		if _, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{ResourceProviderGeneration: pythonRPInt(8), Aggregates: []string{}}); err != nil {
			t.Fatal(err)
		}
		pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{
			{http.MethodGet, pythonRPBase + "/rp-1/aggregates", "", "", "placement 1.20"},
			{http.MethodPut, pythonRPBase + "/rp-1/aggregates", "", `{"aggregates":["agg-2"],"resource_provider_generation":7}`, "placement 1.20"},
			{http.MethodPut, pythonRPBase + "/rp-1/aggregates", "", `{"aggregates":[],"resource_provider_generation":8}`, "placement 1.20"},
		})
	})
	t.Run("aggregates below 1.19", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.18", &recorder, func(*http.Request) *http.Response { return pythonRPWire(200, `{"aggregates":["agg-1"]}`) })
		got, err := api.UpdateAggregates(ctx, "rp-1", resourceproviders.UpdateAggregatesOpts{Aggregates: []string{"agg-1"}})
		if err != nil || got.ResourceProviderGeneration != nil {
			t.Fatal(got, err)
		}
		pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{{http.MethodPut, pythonRPBase + "/rp-1/aggregates", "", `["agg-1"]`, "placement 1.18"}})
	})
	t.Run("usages", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, func(*http.Request) *http.Response {
			return pythonRPWire(200, `{"resource_provider_generation":4,"usages":{"VCPU":2,"MEMORY_MB":512}}`)
		})
		// fetch_resource_provider_usages("rp-1")
		got, err := api.GetUsages(ctx, "rp-1")
		if err != nil || !reflect.DeepEqual(got.Usages, map[string]int{"VCPU": 2, "MEMORY_MB": 512}) {
			t.Fatal(got, err)
		}
		pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{{http.MethodGet, pythonRPBase + "/rp-1/usages", "", "", "placement 1.20"}})
	})
	t.Run("allocations without microversion", func(t *testing.T) {
		var recorder pythonRPRecorder
		// ResourceProviderAllocation declares no _max_microversion, so Python sends no version header.
		api := pythonRPAPI(t, "", &recorder, func(*http.Request) *http.Response {
			return pythonRPWire(200, `{"resource_provider_generation":4,"allocations":{"c-1":{"resources":{"VCPU":1}},"c-2":{"resources":{"VCPU":2}}}}`)
		})
		// resource_provider_allocations("rp-1") yields one row per consumer.
		got, err := api.GetAllocations(ctx, "rp-1")
		if err != nil || got.ResourceProviderGeneration != 4 || len(got.Allocations) != 2 || got.Allocations["c-2"].Resources["VCPU"] != 2 {
			t.Fatal(got, err)
		}
		pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{{http.MethodGet, pythonRPBase + "/rp-1/allocations", "", "", ""}})
	})
	t.Run("traits at 1.6", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.6", &recorder, func(req *http.Request) *http.Response {
			switch req.Method {
			case http.MethodDelete:
				return pythonRPWire(204, "")
			case http.MethodPut:
				return pythonRPWire(200, `{"resource_provider_generation":7,"traits":["CUSTOM_B"]}`)
			}
			return pythonRPWire(200, `{"resource_provider_generation":6,"traits":["CUSTOM_A"]}`)
		})
		// get_resource_provider_trait("rp-1")
		got, err := api.GetTraits(ctx, "rp-1")
		if err != nil || got.ResourceProviderGeneration != 6 || !reflect.DeepEqual(got.Traits, []string{"CUSTOM_A"}) {
			t.Fatal(got, err)
		}
		// set_resource_provider_trait(rp_trait, traits=["CUSTOM_B"]) forces the generation into the PUT body.
		set, err := api.UpdateTraits(ctx, "rp-1", resourceproviders.UpdateTraitsOpts{ResourceProviderGeneration: 6, Traits: []string{"CUSTOM_B"}})
		if err != nil || set.ResourceProviderGeneration != 7 || !reflect.DeepEqual(set.Traits, []string{"CUSTOM_B"}) {
			t.Fatal(set, err)
		}
		// set_resource_provider_trait(rp_trait, traits=[]) needs a non-nil empty slice in Go.
		if _, err := api.UpdateTraits(ctx, "rp-1", resourceproviders.UpdateTraitsOpts{ResourceProviderGeneration: 7, Traits: []string{}}); err != nil {
			t.Fatal(err)
		}
		// delete_resource_provider_trait("rp-1")
		if err := api.DeleteTraits(ctx, "rp-1"); err != nil {
			t.Fatal(err)
		}
		pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{
			{http.MethodGet, pythonRPBase + "/rp-1/traits", "", "", "placement 1.6"},
			{http.MethodPut, pythonRPBase + "/rp-1/traits", "", `{"resource_provider_generation":6,"traits":["CUSTOM_B"]}`, "placement 1.6"},
			{http.MethodPut, pythonRPBase + "/rp-1/traits", "", `{"resource_provider_generation":7,"traits":[]}`, "placement 1.6"},
			{http.MethodDelete, pythonRPBase + "/rp-1/traits", "", "", "placement 1.6"},
		})
	})
	t.Run("delete traits keeps 404", func(t *testing.T) {
		// Python's default ignore_missing=True swallows this 404; Go has no option for it.
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.6", &recorder, func(*http.Request) *http.Response { return pythonRPWire(404, `{}`) })
		err := api.DeleteTraits(ctx, "rp-1")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 404 {
			t.Fatal(err)
		}
	})
}

// ResourceProviderInventory declares no _max_microversion, so Python sends these requests without a version header.
func TestPythonResourceProviderInventories(t *testing.T) {
	ctx := context.Background()
	full := resourceproviders.Inventory{AllocationRatio: 16, MaxUnit: 8, MinUnit: 1, Reserved: 0, StepSize: 1, Total: 8}
	t.Run("get list update", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "", &recorder, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/inventories") {
				return pythonRPWire(200, `{"resource_provider_generation":4,"inventories":{"VCPU":{"allocation_ratio":16.0,"max_unit":8,"min_unit":1,"reserved":0,"step_size":1,"total":8},"DISK_GB":{"allocation_ratio":1.0,"max_unit":100,"min_unit":1,"reserved":0,"step_size":1,"total":100}}}`)
			}
			return pythonRPWire(200, `{"resource_provider_generation":5,"allocation_ratio":16.0,"max_unit":8,"min_unit":1,"reserved":0,"step_size":1,"total":8}`)
		})
		// get_resource_provider_inventory("VCPU", "rp-1")
		one, err := api.GetInventory(ctx, "rp-1", "VCPU")
		if err != nil || one.ResourceProviderGeneration != 5 || one.Inventory != full {
			t.Fatal(one, err)
		}
		// resource_provider_inventories("rp-1") yields one row per resource class.
		all, err := api.GetInventories(ctx, "rp-1")
		if err != nil || all.ResourceProviderGeneration != 4 || len(all.Inventories) != 2 || all.Inventories["VCPU"] != full || all.Inventories["DISK_GB"].Total != 100 {
			t.Fatal(all, err)
		}
		// update_resource_provider_inventory("VCPU", "rp-1", resource_provider_generation=4, total=8, ...)
		updated, err := api.UpdateInventory(ctx, "rp-1", "VCPU", resourceproviders.UpdateInventoryOpts{ResourceProviderGeneration: 4, Inventory: full})
		if err != nil || updated.ResourceProviderGeneration != 5 || updated.Total != 8 {
			t.Fatal(updated, err)
		}
		pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{
			{http.MethodGet, pythonRPBase + "/rp-1/inventories/VCPU", "", "", ""},
			{http.MethodGet, pythonRPBase + "/rp-1/inventories", "", "", ""},
			{http.MethodPut, pythonRPBase + "/rp-1/inventories/VCPU", "", `{"allocation_ratio":16,"max_unit":8,"min_unit":1,"reserved":0,"resource_provider_generation":4,"step_size":1,"total":8}`, ""},
		})
	})
	t.Run("set and delete all at 1.20", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, func(req *http.Request) *http.Response {
			if req.Method == http.MethodDelete {
				return pythonRPWire(204, "")
			}
			return pythonRPWire(200, `{"resource_provider_generation":9,"inventories":{"VCPU":{"allocation_ratio":16.0,"max_unit":8,"min_unit":1,"reserved":0,"step_size":1,"total":8}}}`)
		})
		// set_resource_provider_inventories("rp-1", {"VCPU": {...}}, 8) uses ResourceProvider's 1.20.
		set, err := api.UpdateInventories(ctx, "rp-1", resourceproviders.UpdateInventoriesOpts{ResourceProviderGeneration: 8, Inventories: map[string]resourceproviders.Inventory{"VCPU": full}})
		if err != nil || set.ResourceProviderGeneration != 9 {
			t.Fatal(set, err)
		}
		// set_resource_provider_inventories("rp-1", {}, 9) clears everything; Go needs a non-nil empty map.
		if _, err := api.UpdateInventories(ctx, "rp-1", resourceproviders.UpdateInventoriesOpts{ResourceProviderGeneration: 9, Inventories: map[string]resourceproviders.Inventory{}}); err != nil {
			t.Fatal(err)
		}
		// delete_resource_provider_inventories("rp-1")
		if err := api.DeleteInventories(ctx, "rp-1"); err != nil {
			t.Fatal(err)
		}
		pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{
			{http.MethodPut, pythonRPBase + "/rp-1/inventories", "", `{"inventories":{"VCPU":{"allocation_ratio":16,"max_unit":8,"min_unit":1,"reserved":0,"step_size":1,"total":8}},"resource_provider_generation":8}`, "placement 1.20"},
			{http.MethodPut, pythonRPBase + "/rp-1/inventories", "", `{"inventories":{},"resource_provider_generation":9}`, "placement 1.20"},
			{http.MethodDelete, pythonRPBase + "/rp-1/inventories", "", "", "placement 1.20"},
		})
	})
	t.Run("delete one keeps 404", func(t *testing.T) {
		// delete_resource_provider_inventory("VCPU", "rp-1") swallows 404 by default; Go DeleteInventory reports it.
		for code, wantErr := range map[int]bool{204: false, 404: true} {
			var recorder pythonRPRecorder
			api := pythonRPAPI(t, "", &recorder, func(*http.Request) *http.Response { return pythonRPWire(code, "") })
			err := api.DeleteInventory(ctx, "rp-1", "VCPU")
			var native gophercloud.ErrUnexpectedResponseCode
			if wantErr != (errors.As(err, &native) && native.Actual == 404) || (!wantErr && err != nil) {
				t.Fatal(code, err)
			}
			pythonRPExpect(t, recorder.snapshot(), []pythonRPCall{{http.MethodDelete, pythonRPBase + "/rp-1/inventories/VCPU", "", "", ""}})
		}
	})
}

// Placement resources have no status field; Python callers pass attribute= and Go selects it with WithStatusAttribute.
func TestPythonResourceProviderWaiters(t *testing.T) {
	ctx := context.Background()
	rows := func(names ...string) func(*http.Request) *http.Response {
		var mu sync.Mutex
		index := 0
		return func(*http.Request) *http.Response {
			mu.Lock()
			defer mu.Unlock()
			name := names[min(index, len(names)-1)]
			index++
			if name == "" {
				return pythonRPWire(404, `{}`)
			}
			return pythonRPWire(200, `{"uuid":"rp-1","generation":1,"name":"`+name+`"}`)
		}
	}
	fast := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithStatusAttribute("name"), resource.WithFailureStates("ERROR")}
	t.Run("wait_for_status reaches target", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, rows("building", "building", "ready"))
		got, err := api.WaitFor(ctx, resource.ID("rp-1"), "READY", fast...)
		if err != nil || got.Name != "ready" || len(recorder.snapshot()) != 3 {
			t.Fatal(got, err, recorder.snapshot())
		}
		for _, call := range recorder.snapshot() {
			if call != (pythonRPCall{http.MethodGet, pythonRPBase + "/rp-1", "", "", "placement 1.20"}) {
				t.Fatal(call)
			}
		}
	})
	t.Run("wait_for_status failure state", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, rows("building", "error"))
		if _, err := api.WaitFor(ctx, resource.ID("rp-1"), "ready", fast...); !errors.Is(err, resource.ErrFailedState) {
			t.Fatal(err)
		}
	})
	t.Run("wait_for_status timeout", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, rows("building"))
		_, err := api.WaitFor(ctx, resource.ID("rp-1"), "ready", append(fast, resource.WithTimeout(20*time.Millisecond))...)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	})
	t.Run("wait_for_delete", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, rows("compute-1", "compute-1", ""))
		if err := api.WaitForDeletion(ctx, resource.ID("rp-1"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(120*time.Second)); err != nil || len(recorder.snapshot()) != 3 {
			t.Fatal(err, recorder.snapshot())
		}
	})
	t.Run("wait_for_delete timeout", func(t *testing.T) {
		var recorder pythonRPRecorder
		api := pythonRPAPI(t, "1.20", &recorder, rows("compute-1"))
		err := api.WaitForDeletion(ctx, resource.ID("rp-1"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(20*time.Millisecond))
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	})
}
