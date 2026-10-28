package services_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonComputeServiceCall records the wire request openstacksdk would also send.
type pythonComputeServiceCall struct{ method, path, query, body, version string }

func pythonComputeServiceWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

func pythonComputeServiceAPI(t *testing.T, microversion string, reply func(*http.Request) *http.Response) (*services.API, *testcloud.Recorder) {
	t.Helper()
	calls := testcloud.NewRecorder(func(req *http.Request) (*http.Response, error) { return reply(req), nil })
	cloud := testcloud.New(t)
	cloud.SetTransport(calls)
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = microversion
	return services.New(client), calls
}

// pythonComputeServiceCalls projects the recorded calls onto the local call fields.
func pythonComputeServiceCalls(rec *testcloud.Recorder) []pythonComputeServiceCall {
	calls := []pythonComputeServiceCall{}
	for _, call := range rec.Calls() {
		calls = append(calls, pythonComputeServiceCall{call.Method, call.Path, call.RawQuery, call.Body, call.Header.Get("X-OpenStack-Nova-API-Version")})
	}
	return calls
}

const pythonComputeServiceBase = "/nova/v2.1/os-services"

const pythonComputeServiceRow = `{"id":"svc-uuid","binary":"nova-compute","host":"cmp1","zone":"nova","status":"disabled","state":"up","disabled_reason":"maint","forced_down":true,"updated_at":"2026-10-01T01:02:03.000000"}`

func TestPythonComputeServiceListMapsNameToBinary(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonComputeServiceAPI(t, "2.69", func(*http.Request) *http.Response {
		return pythonComputeServiceWire(200, `{"services":[`+pythonComputeServiceRow+`]}`)
	})
	// services() and services(name="nova-compute", host="cmp1"); Python maps name to binary.
	for _, options := range [][]services.ListOption{
		nil,
		{services.WithListOptions(services.ListOpts{Binary: "nova-compute", Host: "cmp1"})},
	} {
		var rows []*services.Service
		for value, err := range api.List(ctx, options...) {
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, value)
		}
		if len(rows) != 1 || rows[0].ID != "svc-uuid" || rows[0].Binary != "nova-compute" || rows[0].Host != "cmp1" || rows[0].Zone != "nova" || rows[0].Status != "disabled" || rows[0].State != "up" || rows[0].DisabledReason != "maint" || !rows[0].ForcedDown || rows[0].UpdatedAt.IsZero() {
			t.Fatal(rows)
		}
	}
	want := []pythonComputeServiceCall{
		{http.MethodGet, pythonComputeServiceBase, "", "", "2.69"},
		{http.MethodGet, pythonComputeServiceBase, "binary=nova-compute&host=cmp1", "", "2.69"},
	}
	if !reflect.DeepEqual(pythonComputeServiceCalls(calls), want) {
		t.Fatalf("%+v", pythonComputeServiceCalls(calls))
	}
}

func TestPythonComputeServiceUpdateByIDCannotClearForcedDown(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonComputeServiceAPI(t, "2.69", func(*http.Request) *http.Response {
		return pythonComputeServiceWire(200, `{"service":`+pythonComputeServiceRow+`}`)
	})
	steps := []struct {
		python string
		opts   services.UpdateOpts
	}{
		{"enable_service('svc-uuid')", services.UpdateOpts{Status: services.ServiceEnabled}},
		{"disable_service('svc-uuid', disabled_reason='maint')", services.UpdateOpts{Status: services.ServiceDisabled, DisabledReason: "maint"}},
		{"update_service_forced_down('svc-uuid')", services.UpdateOpts{ForcedDown: true}},
	}
	for _, step := range steps {
		if got, err := api.Update(ctx, "svc-uuid", step.opts); err != nil || got.ID != "svc-uuid" {
			t.Fatal(step.python, got, err)
		}
	}
	// update_service_forced_down('svc-uuid', forced=False) sends forced_down false;
	// Go omits the false typed key and refuses it as an extension.
	if _, err := api.Update(ctx, "svc-uuid", services.UpdateOpts{}, services.WithUpdateField("forced_down", false)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	want := []pythonComputeServiceCall{
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"status":"enabled"}`, "2.69"},
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"disabled_reason":"maint","status":"disabled"}`, "2.69"},
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"forced_down":true}`, "2.69"},
	}
	if !reflect.DeepEqual(pythonComputeServiceCalls(calls), want) {
		t.Fatalf("%+v", pythonComputeServiceCalls(calls))
	}
}

func TestPythonComputeServiceLegacyActionsSendBodyButFailDecode(t *testing.T) {
	ctx := context.Background()
	// Before 2.53 Nova answers the action routes without a service id.
	api, calls := pythonComputeServiceAPI(t, "2.11", func(*http.Request) *http.Response {
		return pythonComputeServiceWire(200, `{"service":{"host":"cmp1","binary":"nova-compute","status":"disabled"}}`)
	})
	// disable_service(None, host="cmp1", binary="nova-compute") puts os-services/disable.
	_, err := api.Update(ctx, "disable", services.UpdateOpts{}, services.WithUpdateField("host", "cmp1"), services.WithUpdateField("binary", "nova-compute"))
	if err == nil || !strings.Contains(err.Error(), "ID has unexpected type") {
		t.Fatal("legacy service action response must fail decoding without an id", err)
	}
	want := []pythonComputeServiceCall{
		{http.MethodPut, pythonComputeServiceBase + "/disable", "", `{"binary":"nova-compute","host":"cmp1"}`, "2.11"},
	}
	if !reflect.DeepEqual(pythonComputeServiceCalls(calls), want) {
		t.Fatalf("%+v", pythonComputeServiceCalls(calls))
	}
}

func TestPythonComputeServiceDeleteReturnsMissingError(t *testing.T) {
	ctx := context.Background()
	status := http.StatusNotFound
	api, calls := pythonComputeServiceAPI(t, "2.69", func(*http.Request) *http.Response {
		return pythonComputeServiceWire(status, `{"itemNotFound":{"message":"gone"}}`)
	})
	// delete_service("svc-uuid") ignores 404 by default; Delete has no such option.
	if err := api.Delete(ctx, "svc-uuid"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	status = http.StatusNoContent
	if err := api.Delete(ctx, "svc-uuid"); err != nil {
		t.Fatal(err)
	}
	want := []pythonComputeServiceCall{
		{http.MethodDelete, pythonComputeServiceBase + "/svc-uuid", "", "", "2.69"},
		{http.MethodDelete, pythonComputeServiceBase + "/svc-uuid", "", "", "2.69"},
	}
	if !reflect.DeepEqual(pythonComputeServiceCalls(calls), want) {
		t.Fatalf("%+v", pythonComputeServiceCalls(calls))
	}
}
