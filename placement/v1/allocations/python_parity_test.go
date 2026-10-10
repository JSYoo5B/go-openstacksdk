package allocations_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonAllocationTransport func(*http.Request) (*http.Response, error)

func (transport pythonAllocationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonAllocationWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type pythonAllocationCall struct{ method, path, query, body, version string }

// Allocation declares _max_microversion 1.38, so the helper pins that version.
func pythonAllocationAPI(t *testing.T, calls *[]pythonAllocationCall, reply func(*http.Request) *http.Response) *allocations.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonAllocationTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonAllocationCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = "1.38"
	return allocations.New(client)
}

func pythonAllocationJSON(t *testing.T, raw string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(raw, err)
	}
	return value
}

func pythonAllocationInt(v int) *int { return &v }

func TestPythonAllocationProxyCalls(t *testing.T) {
	ctx := context.Background()
	var calls []pythonAllocationCall
	api := pythonAllocationAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet {
			return pythonAllocationWire(200, `{"allocations":{"rp-1":{"generation":3,"resources":{"VCPU":1,"MEMORY_MB":512}}},"consumer_generation":2,"consumer_type":"INSTANCE","project_id":"p-1","user_id":"u-1"}`)
		}
		return pythonAllocationWire(204, "")
	})
	// get_allocation("c-1")
	got, err := api.Get(ctx, "c-1")
	if err != nil || got.Allocations["rp-1"].Generation != 3 || got.Allocations["rp-1"].Resources["MEMORY_MB"] != 512 ||
		*got.ConsumerGeneration != 2 || *got.ConsumerType != "INSTANCE" || *got.ProjectID != "p-1" || *got.UserID != "u-1" {
		t.Fatal(got, err)
	}
	// update_allocation("c-1", allocations={...}, project_id="p-1", user_id="u-1", consumer_generation=None, consumer_type="INSTANCE")
	update := allocations.UpdateOpts{
		Allocations: map[string]allocations.ProviderAllocationsOpts{"rp-1": {Resources: map[string]int{"VCPU": 1}}},
		ProjectID:   "p-1", UserID: "u-1", ConsumerType: "INSTANCE",
	}
	if err := api.Update(ctx, "c-1", update); err != nil {
		t.Fatal(err)
	}
	// create_allocations({"c-1": {...}, "c-2": {..., "allocations": {}}}) removes c-2 with an empty non-nil map.
	existing := update
	existing.ConsumerGeneration = pythonAllocationInt(1)
	if err := api.Manage(ctx, allocations.ManageOpts{
		"c-1": existing,
		"c-2": {Allocations: map[string]allocations.ProviderAllocationsOpts{}, ProjectID: "p-1", UserID: "u-1", ConsumerGeneration: pythonAllocationInt(4), ConsumerType: "MIGRATION"},
	}); err != nil {
		t.Fatal(err)
	}
	// delete_allocation("c-1")
	if err := api.Delete(ctx, "c-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 4 {
		t.Fatalf("%+v", calls)
	}
	for i, want := range []struct{ method, path, body string }{
		{http.MethodGet, "/placement/allocations/c-1", ""},
		{http.MethodPut, "/placement/allocations/c-1", `{"allocations":{"rp-1":{"resources":{"VCPU":1}}},"consumer_generation":null,"consumer_type":"INSTANCE","project_id":"p-1","user_id":"u-1"}`},
		{http.MethodPost, "/placement/allocations", `{"c-1":{"allocations":{"rp-1":{"resources":{"VCPU":1}}},"consumer_generation":1,"consumer_type":"INSTANCE","project_id":"p-1","user_id":"u-1"},"c-2":{"allocations":{},"consumer_generation":4,"consumer_type":"MIGRATION","project_id":"p-1","user_id":"u-1"}}`},
		{http.MethodDelete, "/placement/allocations/c-1", ""},
	} {
		call := calls[i]
		if call.method != want.method || call.path != want.path || call.query != "" || call.version != "placement 1.38" {
			t.Fatalf("%d %+v", i, call)
		}
		if want.body == "" {
			if call.body != "" {
				t.Fatal(i, call.body)
			}
		} else if !reflect.DeepEqual(pythonAllocationJSON(t, call.body), pythonAllocationJSON(t, want.body)) {
			t.Fatal(i, call.body)
		}
	}
}

// delete_allocation swallows 404 by default; Go Delete reports the native status.
func TestPythonAllocationDeleteMissing(t *testing.T) {
	var calls []pythonAllocationCall
	api := pythonAllocationAPI(t, &calls, func(*http.Request) *http.Response { return pythonAllocationWire(404, `{}`) })
	err := api.Delete(context.Background(), "c-1")
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 404 || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}

// create_allocations can put per-consumer keys such as the 1.34 mappings into every consumer.
// With two consumers Go places an extension field at the top level instead.
func TestPythonAllocationManageExtensionPlacement(t *testing.T) {
	var calls []pythonAllocationCall
	api := pythonAllocationAPI(t, &calls, func(*http.Request) *http.Response { return pythonAllocationWire(204, "") })
	consumer := allocations.UpdateOpts{Allocations: map[string]allocations.ProviderAllocationsOpts{}, ProjectID: "p-1", UserID: "u-1", ConsumerType: "INSTANCE"}
	mappings := map[string][]string{"": {"rp-1"}}
	if err := api.Manage(context.Background(), allocations.ManageOpts{"c-1": consumer, "c-2": consumer}, allocations.WithManageField("mappings", mappings)); err != nil {
		t.Fatal(err)
	}
	body, ok := pythonAllocationJSON(t, calls[0].body).(map[string]any)
	if !ok || body["mappings"] == nil {
		t.Fatal(calls[0].body)
	}
	for _, key := range []string{"c-1", "c-2"} {
		if _, found := body[key].(map[string]any)["mappings"]; found {
			t.Fatal(key, calls[0].body)
		}
	}
}
