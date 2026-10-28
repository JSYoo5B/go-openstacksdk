package services_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonComputeServiceStatusRecordsFromMicroversion253(t *testing.T) {
	ctx := context.Background()
	listed := `{"services":[` + pythonComputeServiceRow + `]}`
	reply := `{"service":` + pythonComputeServiceRow + `}`
	api, calls := pythonComputeServiceAPI(t, "2.69", func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet {
			return pythonComputeServiceWire(http.StatusOK, listed)
		}
		return pythonComputeServiceWire(http.StatusOK, reply)
	})
	// enable_service("svc-uuid") updates by id; host and binary are ignored.
	got, err := api.EnableRecord(ctx, services.ServiceActionTarget{ID: "svc-uuid", Host: "ignored"})
	if err != nil || !got.Sent || got.StatusCode != 200 || got.Service == nil || got.Service.ID != "svc-uuid" || string(got.Fields["zone"]) != `"nova"` {
		t.Fatal(got, err)
	}
	// disable_service("svc-uuid") omits an empty reason, and a reason is sent
	// with the status.
	if _, err := api.DisableRecord(ctx, services.ServiceActionTarget{ID: "svc-uuid"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := api.DisableRecord(ctx, services.ServiceActionTarget{ID: "svc-uuid"}, "maint"); err != nil {
		t.Fatal(err)
	}
	// enable_service(None, host="cmp1", binary="nova-compute") lists by host
	// and binary; the disabled row makes status dirty.
	got, err = api.EnableRecord(ctx, services.ServiceActionTarget{Host: "cmp1", Binary: "nova-compute"})
	if err != nil || !got.Sent || got.Service.Host != "cmp1" {
		t.Fatal(got, err)
	}
	// disable_service(None, ..., disabled_reason="new") sends only the reason
	// because the listed row is already disabled.
	if _, err := api.DisableRecord(ctx, services.ServiceActionTarget{Host: "cmp1", Binary: "nova-compute"}, "new"); err != nil {
		t.Fatal(err)
	}
	// With status and reason unchanged Resource.commit sends nothing and the
	// listed Service is returned.
	got, err = api.DisableRecord(ctx, services.ServiceActionTarget{Host: "cmp1", Binary: "nova-compute"}, "maint")
	if err != nil || got.Sent || got.Service == nil || got.Service.ID != "svc-uuid" || got.StatusCode != 200 || string(got.Fields["status"]) != `"disabled"` {
		t.Fatal(got, err)
	}
	query := "binary=nova-compute&host=cmp1"
	want := []pythonComputeServiceCall{
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"status":"enabled"}`, "2.69"},
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"status":"disabled"}`, "2.69"},
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"disabled_reason":"maint","status":"disabled"}`, "2.69"},
		{http.MethodGet, pythonComputeServiceBase, query, "", "2.69"},
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"status":"enabled"}`, "2.69"},
		{http.MethodGet, pythonComputeServiceBase, query, "", "2.69"},
		{http.MethodPut, pythonComputeServiceBase + "/svc-uuid", "", `{"disabled_reason":"new"}`, "2.69"},
		{http.MethodGet, pythonComputeServiceBase, query, "", "2.69"},
	}
	if !reflect.DeepEqual(pythonComputeServiceCalls(calls), want) {
		t.Fatalf("%+v", pythonComputeServiceCalls(calls))
	}
}

func TestPythonComputeServiceStatusRecordsLookupFailures(t *testing.T) {
	ctx := context.Background()
	listed := `{"services":[]}`
	api, calls := pythonComputeServiceAPI(t, "2.53", func(*http.Request) *http.Response {
		return pythonComputeServiceWire(http.StatusOK, listed)
	})
	target := services.ServiceActionTarget{Host: "cmp1", Binary: "nova-compute"}
	// Python raises SDKException for no match or several matches.
	if _, err := api.EnableRecord(ctx, target); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	listed = `{"services":[` + pythonComputeServiceRow + `,` + pythonComputeServiceRow + `]}`
	var operation *resource.OperationError
	if _, err := api.DisableRecord(ctx, target, ""); !errors.Is(err, resource.ErrAmbiguous) || !errors.As(err, &operation) || operation.Operation != "DisableRecord" {
		t.Fatal(err)
	}
	// Python raises ValueError without an id or both host and binary.
	for _, partial := range []services.ServiceActionTarget{{}, {Host: "cmp1"}, {Binary: "nova-compute"}} {
		if _, err := api.EnableRecord(ctx, partial); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(partial, err)
		}
	}
	if calls.Len() != 2 {
		t.Fatalf("%+v", pythonComputeServiceCalls(calls))
	}
	var nilAPI *services.API
	if _, err := nilAPI.EnableRecord(ctx, target); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	bad, badCalls := pythonComputeServiceAPI(t, "two", nil)
	if _, err := bad.DisableRecord(ctx, target, ""); !errors.Is(err, resource.ErrInvalidOption) || badCalls.Len() != 0 {
		t.Fatal(err)
	}
}

func TestPythonComputeServiceStatusRecordsBelowMicroversion253(t *testing.T) {
	ctx := context.Background()
	status := http.StatusOK
	reply := func(*http.Request) *http.Response {
		if status >= 400 {
			return pythonComputeServiceWire(status, `{"itemNotFound":{"message":"gone"}}`)
		}
		return pythonComputeServiceWire(status, `{"service":{"host":"cmp1","binary":"nova-compute","status":"enabled"}}`)
	}
	api, calls := pythonComputeServiceAPI(t, "2.11", reply)
	target := services.ServiceActionTarget{ID: "svc-1", Host: "cmp1", Binary: "nova-compute"}
	// The id-less legacy response is kept in Fields without a typed Service.
	got, err := api.EnableRecord(ctx, target)
	if err != nil || !got.Sent || got.Service != nil || string(got.Fields["status"]) != `"enabled"` || string(got.Fields["host"]) != `"cmp1"` || got.StatusCode != 200 {
		t.Fatal(got, err)
	}
	status = http.StatusAccepted
	if got, err := api.DisableRecord(ctx, target, ""); err != nil || got.StatusCode != 202 {
		t.Fatal(got, err)
	}
	if _, err := api.DisableRecord(ctx, target, "maint"); err != nil {
		t.Fatal(err)
	}
	// Python raises ValueError without host and binary below 2.53, even with
	// an id.
	if _, err := api.EnableRecord(ctx, services.ServiceActionTarget{ID: "svc-1", Host: "cmp1"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	status = http.StatusNotFound
	var operation *resource.OperationError
	if _, err := api.EnableRecord(ctx, target); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) || !errors.As(err, &operation) || operation.Operation != "EnableRecord" || operation.Resource != "services" {
		t.Fatal(err)
	}
	// An unset client microversion is Nova's 2.1 default and sends no header.
	status = http.StatusOK
	old, oldCalls := pythonComputeServiceAPI(t, "", reply)
	if _, err := old.DisableRecord(ctx, target, ""); err != nil {
		t.Fatal(err)
	}
	host := `{"binary":"nova-compute","host":"cmp1"}`
	want := []pythonComputeServiceCall{
		{http.MethodPut, pythonComputeServiceBase + "/enable", "", host, "2.11"},
		{http.MethodPut, pythonComputeServiceBase + "/disable", "", host, "2.11"},
		{http.MethodPut, pythonComputeServiceBase + "/disable-log-reason", "", `{"binary":"nova-compute","disabled_reason":"maint","host":"cmp1"}`, "2.11"},
		{http.MethodPut, pythonComputeServiceBase + "/enable", "", host, "2.11"},
	}
	if !reflect.DeepEqual(pythonComputeServiceCalls(calls), want) {
		t.Fatalf("%+v", pythonComputeServiceCalls(calls))
	}
	if wantOld := []pythonComputeServiceCall{{http.MethodPut, pythonComputeServiceBase + "/disable", "", host, ""}}; !reflect.DeepEqual(pythonComputeServiceCalls(oldCalls), wantOld) {
		t.Fatalf("%+v", pythonComputeServiceCalls(oldCalls))
	}
}
