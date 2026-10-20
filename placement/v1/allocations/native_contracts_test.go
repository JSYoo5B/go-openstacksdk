package allocations_test

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
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeAllocTransport func(*http.Request) (*http.Response, error)

func (transport nativeAllocTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeAllocWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeAllocCall struct{ method, path, query, body, version string }

func nativeAllocAPI(t *testing.T, microversion string, calls *[]nativeAllocCall, reply func(*http.Request) *http.Response) *allocations.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeAllocTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeAllocCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = microversion
	return allocations.New(client)
}

func nativeAllocOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "allocations" {
		t.Fatal("generated allocations context", err, wrapped)
	}
}

func nativeAllocInt(v int) *int { return &v }

func TestNativeAllocationsRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeAllocCall
	api := nativeAllocAPI(t, "1.38", &calls, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodGet:
			return nativeAllocWire(200, `{"allocations":{"rp-1":{"generation":2,"resources":{"VCPU":1,"MEMORY_MB":512}}},"consumer_generation":3,"project_id":"p-1","user_id":"u-1","consumer_type":"INSTANCE"}`)
		}
		return nativeAllocWire(204, "")
	})
	got, err := api.Get(ctx, "c-1")
	if err != nil || got.Allocations["rp-1"].Generation != 2 || got.Allocations["rp-1"].Resources["MEMORY_MB"] != 512 ||
		*got.ConsumerGeneration != 3 || *got.ProjectID != "p-1" || *got.UserID != "u-1" || *got.ConsumerType != "INSTANCE" {
		t.Fatal(got, err)
	}
	allocation := map[string]allocations.ProviderAllocationsOpts{"rp-1": {Resources: map[string]int{"VCPU": 1}}}
	// A nil consumer generation is sent as null to claim a new consumer.
	if err := api.Update(ctx, "c-1", allocations.UpdateOpts{Allocations: allocation, ProjectID: "p-1", UserID: "u-1", ConsumerType: "INSTANCE"}, allocations.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if err := api.Update(ctx, "c-1", allocations.UpdateOpts{Allocations: map[string]allocations.ProviderAllocationsOpts{}, ConsumerGeneration: nativeAllocInt(3)}); err != nil {
		t.Fatal(err)
	}
	if err := api.Manage(ctx, allocations.ManageOpts{
		"c-1": {Allocations: allocation, ProjectID: "p-1", UserID: "u-1", ConsumerGeneration: nativeAllocInt(1)},
		"c-2": {Allocations: map[string]allocations.ProviderAllocationsOpts{}, ProjectID: "p-1", UserID: "u-1", ConsumerGeneration: nativeAllocInt(4)},
	}); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "c-1"); err != nil {
		t.Fatal(err)
	}
	base := "/placement/allocations"
	want := []nativeAllocCall{
		{http.MethodGet, base + "/c-1", "", "", "placement 1.38"},
		{http.MethodPut, base + "/c-1", "", `{"allocations":{"rp-1":{"resources":{"VCPU":1}}},"consumer_generation":null,"consumer_type":"INSTANCE","project_id":"p-1","user_id":"u-1","x_extension":1}`, "placement 1.38"},
		// project_id and user_id have no omitempty, so empty strings are sent.
		{http.MethodPut, base + "/c-1", "", `{"allocations":{},"consumer_generation":3,"project_id":"","user_id":""}`, "placement 1.38"},
		{http.MethodPost, base, "", `{"c-1":{"allocations":{"rp-1":{"resources":{"VCPU":1}}},"consumer_generation":1,"project_id":"p-1","user_id":"u-1"},"c-2":{"allocations":{},"consumer_generation":4,"project_id":"p-1","user_id":"u-1"}}`, "placement 1.38"},
		{http.MethodDelete, base + "/c-1", "", "", "placement 1.38"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	t.Run("decode without consumer fields", func(t *testing.T) {
		var calls []nativeAllocCall
		api := nativeAllocAPI(t, "1.38", &calls, func(*http.Request) *http.Response { return nativeAllocWire(200, `{"allocations":{}}`) })
		got, err := api.Get(ctx, "unknown")
		if err != nil || got.ProjectID != nil || got.UserID != nil || got.ConsumerGeneration != nil || got.ConsumerType != nil || len(got.Allocations) != 0 {
			t.Fatal(got, err)
		}
	})
	t.Run("manage extension placement", func(t *testing.T) {
		var calls []nativeAllocCall
		api := nativeAllocAPI(t, "1.28", &calls, func(*http.Request) *http.Response { return nativeAllocWire(204, "") })
		one := allocations.ManageOpts{"c-1": {ProjectID: "p", UserID: "u"}}
		two := allocations.ManageOpts{"c-1": {ProjectID: "p", UserID: "u"}, "c-2": {ProjectID: "p", UserID: "u"}}
		// With exactly one consumer the extension lands inside that consumer object; otherwise it is a root key.
		for _, opts := range []allocations.ManageOpts{one, two} {
			if err := api.Manage(ctx, opts, allocations.WithManageField("x_extension", 1)); err != nil {
				t.Fatal(err)
			}
		}
		// A nil ManageOpts map is sent as a JSON null body.
		if err := api.Manage(ctx, nil); err != nil {
			t.Fatal(err)
		}
		// A single consumer protects its own keys from extensions.
		err := api.Manage(ctx, one, allocations.WithManageField("project_id", "x"))
		nativeAllocOperation(t, err, "Manage")
		bodies := []string{}
		for _, call := range calls {
			bodies = append(bodies, call.body)
		}
		if !reflect.DeepEqual(bodies, []string{
			`{"c-1":{"allocations":null,"consumer_generation":null,"project_id":"p","user_id":"u","x_extension":1}}`,
			`{"c-1":{"allocations":null,"consumer_generation":null,"project_id":"p","user_id":"u"},"c-2":{"allocations":null,"consumer_generation":null,"project_id":"p","user_id":"u"},"x_extension":1}`,
			`null`,
		}) {
			t.Fatal(bodies)
		}
	})
}

func TestNativeAllocationsStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*allocations.API) error
	}{
		{"Get", []int{200}, func(api *allocations.API) error { _, err := api.Get(ctx, "c-1"); return err }},
		// Update and Manage accept only 204, so a consumer generation conflict surfaces as 409.
		{"Update", []int{204}, func(api *allocations.API) error { return api.Update(ctx, "c-1", allocations.UpdateOpts{}) }},
		{"Manage", []int{204}, func(api *allocations.API) error { return api.Manage(ctx, allocations.ManageOpts{}) }},
		{"Delete", []int{202, 204}, func(api *allocations.API) error { return api.Delete(ctx, "c-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404, 409} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeAllocCall
				api := nativeAllocAPI(t, "1.28", &calls, func(*http.Request) *http.Response { return nativeAllocWire(code, `{}`) })
				err := call.call(api)
				nativeAllocOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 || calls[0].version != "placement 1.28" {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("plain object decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:                 false,
			`null`:               false,
			`{"allocations":[]}`: true,
		} {
			var calls []nativeAllocCall
			api := nativeAllocAPI(t, "1.28", &calls, func(*http.Request) *http.Response { return nativeAllocWire(200, body) })
			got, err := api.Get(ctx, "c-1")
			if wantErr {
				nativeAllocOperation(t, err, "Get")
			} else if err != nil || got == nil || got.Allocations != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeAllocCall
		api := nativeAllocAPI(t, "1.28", &calls, func(*http.Request) *http.Response { return nativeAllocWire(204, "") })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"update core extension": {"Update", api.Update(ctx, "c-1", allocations.UpdateOpts{}, allocations.WithUpdateField("allocations", nil))},
			// consumer_type is omitted when empty but still reserved.
			"update omitted core extension": {"Update", api.Update(ctx, "c-1", allocations.UpdateOpts{}, allocations.WithUpdateField("consumer_type", "x"))},
			"update nil option":             {"Update", api.Update(ctx, "c-1", allocations.UpdateOpts{}, nil)},
			"update empty extension key":    {"Update", api.Update(ctx, "c-1", allocations.UpdateOpts{}, allocations.WithUpdateField(" ", 1))},
			"manage nil option":             {"Manage", api.Manage(ctx, allocations.ManageOpts{}, nil)},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeAllocOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
