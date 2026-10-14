package allocations_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonAllocationCreateRecordSendsPerConsumerKeys(t *testing.T) {
	ctx := context.Background()
	status := http.StatusNoContent
	var calls []pythonAllocationCall
	var traces []string
	api := pythonAllocationAPI(t, &calls, func(req *http.Request) *http.Response {
		traces = append(traces, req.Header.Get("X-Trace"))
		return pythonAllocationWire(status, "")
	})
	// create_allocations({c1: {..., "mappings": {...}}, c2: {"allocations": {}, ...}})
	consumers := map[string]any{
		"c1": map[string]any{
			"allocations": map[string]any{"rp-1": map[string]any{"resources": map[string]any{"VCPU": 1}}},
			"mappings":    map[string]any{"": []string{"rp-1"}},
			"project_id":  "p", "user_id": "u", "consumer_generation": nil, "consumer_type": "INSTANCE",
		},
		"c2": map[string]any{"allocations": map[string]any{}, "project_id": "p", "user_id": "u", "consumer_generation": 3, "consumer_type": "MIGRATION"},
		"c3": nil,
	}
	got, err := api.CreateRecord(ctx, consumers, allocations.WithCreateRecordHeader("X-Trace", "t"))
	if err != nil || got.StatusCode != 204 {
		t.Fatal(got, err)
	}
	// create_allocations({}) and a nil map both post an empty object.
	status = http.StatusOK
	for _, value := range []map[string]any{{}, nil} {
		if got, err := api.CreateRecord(ctx, value); err != nil || got.StatusCode != 200 {
			t.Fatal(got, err)
		}
	}
	if len(calls) != 3 || !reflect.DeepEqual(traces, []string{"t", "", ""}) {
		t.Fatalf("%+v %q", calls, traces)
	}
	want := `{"c1":{"allocations":{"rp-1":{"resources":{"VCPU":1}}},"consumer_generation":null,"consumer_type":"INSTANCE","mappings":{"":["rp-1"]},"project_id":"p","user_id":"u"},` +
		`"c2":{"allocations":{},"consumer_generation":3,"consumer_type":"MIGRATION","project_id":"p","user_id":"u"},"c3":null}`
	for i, body := range []string{want, `{}`, `{}`} {
		call := calls[i]
		if call.method != http.MethodPost || call.path != "/placement/allocations" || call.version != "placement 1.38" || !reflect.DeepEqual(pythonAllocationJSON(t, call.body), pythonAllocationJSON(t, body)) {
			t.Fatalf("call %d: %+v", i, call)
		}
	}
}

func TestPythonAllocationCreateRecordRejections(t *testing.T) {
	ctx := context.Background()
	var calls []pythonAllocationCall
	api := pythonAllocationAPI(t, &calls, func(*http.Request) *http.Response {
		return pythonAllocationWire(http.StatusConflict, `{"errors":[{"status":409,"code":"placement.concurrent_update"}]}`)
	})
	var operation *resource.OperationError
	if _, err := api.CreateRecord(ctx, map[string]any{"c1": map[string]any{"allocations": map[string]any{}}}); !gophercloud.ResponseCodeIs(err, http.StatusConflict) || !errors.As(err, &operation) || operation.Operation != "CreateRecord" || operation.Resource != "allocations" {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("%+v", calls)
	}
	for _, call := range []func() error{
		func() error { _, err := api.CreateRecord(ctx, map[string]any{"c1": func() {}}); return err },
		func() error { _, err := api.CreateRecord(ctx, nil, nil); return err },
		func() error {
			var nilAPI *allocations.API
			_, err := nilAPI.CreateRecord(ctx, nil)
			return err
		},
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 1 {
		t.Fatalf("%+v", calls)
	}
}
