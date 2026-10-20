package services_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonComputeServiceUpdateRecordSendsFalseAndEmptyValues(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusOK, `{"service":`+pythonComputeServiceRow+`}`
	api, calls := pythonComputeServiceAPI(t, "2.69", func(*http.Request) *http.Response { return pythonComputeServiceWire(status, reply) })
	// update_service("svc-uuid", forced_down=False, status="", disabled_reason="")
	got, err := api.UpdateRecord(ctx, "svc-uuid",
		services.WithUpdateRecordForcedDown(request.Present(false)),
		services.WithUpdateRecordStatus(request.Present("")),
		services.WithUpdateRecordDisabledReason(request.Present("")))
	if err != nil || !got.Sent || got.StatusCode != 200 || got.Service == nil || got.Service.ID != "svc-uuid" || !got.Service.ForcedDown || string(got.Fields["zone"]) != `"nova"` {
		t.Fatal(got, err)
	}
	// update_service("svc-uuid", status="disabled", disabled_reason=None,
	// availability_zone="az2") with a flat 202 response.
	status, reply = http.StatusAccepted, `{"id":"svc-uuid","binary":"nova-compute","host":"cmp1","zone":"az2","status":"disabled","state":"up","disabled_reason":null,"forced_down":false}`
	got, err = api.UpdateRecord(ctx, "svc-uuid",
		services.WithUpdateRecordOptions(services.UpdateRecordOpts{Status: request.Present("disabled"), DisabledReason: request.Null[string]()}),
		services.WithUpdateRecordField("zone", "az2"), services.WithUpdateRecordHeader("X-Trace", "t"))
	if err != nil || got.StatusCode != 202 || got.Service.Zone != "az2" || got.Service.ForcedDown || got.Service.DisabledReason != "" || len(got.Fields) != 8 {
		t.Fatal(got, err)
	}
	// update_service("svc-uuid") has nothing dirty, so Python sends no request.
	if got, err := api.UpdateRecord(ctx, "svc-uuid"); err != nil || got.Sent || got.Service != nil {
		t.Fatal(got, err)
	}
	want := []pythonComputeServiceCall{
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"disabled_reason":"","forced_down":false,"status":""}`, "2.69"},
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"disabled_reason":null,"status":"disabled","zone":"az2"}`, "2.69"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonComputeServiceUpdateRecordRejectsInputsVersionsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	reply := func(*http.Request) *http.Response {
		return pythonComputeServiceWire(http.StatusConflict, `{"conflictingRequest":{"message":"busy"}}`)
	}
	api, calls := pythonComputeServiceAPI(t, "2.53", reply)
	for _, option := range []services.UpdateRecordOption{
		services.WithUpdateRecordField("forced_down", false),
		services.WithUpdateRecordField("id", "other"),
		nil,
	} {
		var operation *resource.OperationError
		if _, err := api.UpdateRecord(ctx, "svc-uuid", option); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "UpdateRecord" {
			t.Fatal(err)
		}
	}
	if _, err := api.UpdateRecord(ctx, "", services.WithUpdateRecordStatus(request.Present("enabled"))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	// Python raises SDKException without a request below 2.53.
	for _, version := range []string{"2.52", "", "latest"} {
		old, oldCalls := pythonComputeServiceAPI(t, version, reply)
		if _, err := old.UpdateRecord(ctx, "svc-uuid", services.WithUpdateRecordStatus(request.Present("enabled"))); !errors.Is(err, resource.ErrUnsupported) || len(*oldCalls) != 0 {
			t.Fatal(version, err)
		}
	}
	if len(*calls) != 0 {
		t.Fatalf("%+v", *calls)
	}
	var operation *resource.OperationError
	if _, err := api.UpdateRecord(ctx, "svc-uuid", services.WithUpdateRecordStatus(request.Present("enabled"))); !gophercloud.ResponseCodeIs(err, http.StatusConflict) || !errors.As(err, &operation) || operation.Resource != "services" {
		t.Fatal(err)
	}
	var nilAPI *services.API
	if _, err := nilAPI.UpdateRecord(ctx, "svc-uuid"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestPythonComputeServiceUpdateForcedDownRecordFromMicroversion253(t *testing.T) {
	ctx := context.Background()
	listed := `{"services":[` + pythonComputeServiceRow + `]}`
	api, calls := pythonComputeServiceAPI(t, "2.69", func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet {
			return pythonComputeServiceWire(http.StatusOK, listed)
		}
		return pythonComputeServiceWire(http.StatusOK, `{"service":`+pythonComputeServiceRow+`}`)
	})
	// update_service_forced_down("svc-uuid", forced=False) needs no lookup.
	got, err := api.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{ID: "svc-uuid", Host: "ignored"}, false)
	if err != nil || !got.Sent || got.Service == nil || got.Service.ID != "svc-uuid" {
		t.Fatal(got, err)
	}
	// update_service_forced_down(None, host="cmp1", binary="nova-compute")
	// lists by host and binary and updates the single match.
	if got, err = api.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{Host: "cmp1", Binary: "nova-compute"}, true); err != nil || got.Service.Host != "cmp1" {
		t.Fatal(got, err)
	}
	// Python raises SDKException for no match or several matches.
	listed = `{"services":[]}`
	if _, err := api.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{Host: "cmp1", Binary: "nova-compute"}, true); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	listed = `{"services":[` + pythonComputeServiceRow + `,` + pythonComputeServiceRow + `]}`
	var operation *resource.OperationError
	if _, err := api.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{Host: "cmp1", Binary: "nova-compute"}, true); !errors.Is(err, resource.ErrAmbiguous) || !errors.As(err, &operation) || operation.Operation != "UpdateForcedDownRecord" {
		t.Fatal(err)
	}
	// Python raises ValueError without an id or both host and binary.
	if _, err := api.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{Host: "cmp1"}, true); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	query := "binary=nova-compute&host=cmp1"
	want := []pythonComputeServiceCall{
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"forced_down":false}`, "2.69"},
		{http.MethodGet, pythonComputeServiceBase, query, "", "2.69"},
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"forced_down":true}`, "2.69"},
		{http.MethodGet, pythonComputeServiceBase, query, "", "2.69"},
		{http.MethodGet, pythonComputeServiceBase, query, "", "2.69"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonComputeServiceUpdateForcedDownRecordBelowMicroversion253(t *testing.T) {
	ctx := context.Background()
	status := http.StatusOK
	reply := func(*http.Request) *http.Response {
		if status >= 400 {
			return pythonComputeServiceWire(status, `{"itemNotFound":{"message":"gone"}}`)
		}
		return pythonComputeServiceWire(status, `{"service":{"host":"cmp1","binary":"nova-compute","forced_down":false}}`)
	}
	// From 2.11 set_forced_down adds forced_down and pins 2.11.
	api, calls := pythonComputeServiceAPI(t, "2.30", reply)
	got, err := api.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{ID: "svc-1", Host: "cmp1", Binary: "nova-compute"}, false)
	if err != nil || !got.Sent || got.Service != nil || string(got.Fields["forced_down"]) != "false" || got.StatusCode != 200 {
		t.Fatal(got, err)
	}
	// Python raises ValueError without host and binary below 2.53.
	if _, err := api.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{ID: "svc-1"}, true); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	status = http.StatusNotFound
	var operation *resource.OperationError
	if _, err := api.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{Host: "cmp1", Binary: "nova-compute"}, true); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) || !errors.As(err, &operation) || operation.Resource != "services" {
		t.Fatal(err)
	}
	// Before 2.11 the body has no forced_down and the client version is kept.
	status = http.StatusOK
	old, oldCalls := pythonComputeServiceAPI(t, "2.10", reply)
	if got, err := old.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{Host: "cmp1", Binary: "nova-compute"}, true); err != nil || got.Service != nil {
		t.Fatal(got, err)
	}
	want := []pythonComputeServiceCall{
		{http.MethodPut, pythonComputeServiceBase + "/force-down", "", `{"binary":"nova-compute","forced_down":false,"host":"cmp1"}`, "2.11"},
		{http.MethodPut, pythonComputeServiceBase + "/force-down", "", `{"binary":"nova-compute","forced_down":true,"host":"cmp1"}`, "2.11"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
	wantOld := []pythonComputeServiceCall{
		{http.MethodPut, pythonComputeServiceBase + "/force-down", "", `{"binary":"nova-compute","host":"cmp1"}`, "2.10"},
	}
	if !reflect.DeepEqual(*oldCalls, wantOld) {
		t.Fatalf("%+v", *oldCalls)
	}
	var nilAPI *services.API
	if _, err := nilAPI.UpdateForcedDownRecord(ctx, services.ForcedDownRecordTarget{Host: "cmp1", Binary: "nova-compute"}, true); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
