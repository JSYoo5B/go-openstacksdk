package aggregates_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonAggregateUpdateRecordSendsNullAndEmptyValues(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusOK, `{"aggregate":`+pythonAggregateRow+`}`
	api, calls := pythonAggregateAPI(t, func(*http.Request) *http.Response { return pythonAggregateWire(status, reply) })
	// update_aggregate(7, availability_zone=None) removes the zone in Nova.
	got, err := api.UpdateRecord(ctx, 7, aggregates.WithUpdateRecordAvailabilityZone(request.Null[string]()))
	if err != nil || !got.Sent || got.StatusCode != 200 || got.Aggregate == nil || got.Aggregate.ID != 7 || got.Aggregate.UUID != "agg-uuid" || string(got.Fields["name"]) != `"rack1"` {
		t.Fatal(got, err)
	}
	// update_aggregate(7, name="", availability_zone="az2", uuid="u") with a
	// flat 202 response; Python accepts every status below 400.
	status, reply = http.StatusAccepted, `{"id":7,"name":"","availability_zone":"az2"}`
	got, err = api.UpdateRecord(ctx, 7,
		aggregates.WithUpdateRecordOptions(aggregates.UpdateRecordOpts{Name: request.Present(""), AvailabilityZone: request.Present("az2")}),
		aggregates.WithUpdateRecordField("uuid", "u"), aggregates.WithUpdateRecordHeader("X-Trace", "t"))
	if err != nil || got.StatusCode != 202 || got.Aggregate.Name != "" || got.Aggregate.AvailabilityZone != "az2" || len(got.Fields) != 3 {
		t.Fatal(got, err)
	}
	// update_aggregate(7) has nothing dirty, so Python sends no request.
	if got, err := api.UpdateRecord(ctx, 7); err != nil || got.Sent || got.Aggregate != nil {
		t.Fatal(got, err)
	}
	want := []pythonAggregateCall{
		{http.MethodPut, pythonAggregateBase + "/7", "", `{"aggregate":{"availability_zone":null}}`, "2.81"},
		{http.MethodPut, pythonAggregateBase + "/7", "", `{"aggregate":{"availability_zone":"az2","name":"","uuid":"u"}}`, "2.81"},
	}
	if !reflect.DeepEqual(pythonAggregateCalls(calls), want) {
		t.Fatalf("%+v", pythonAggregateCalls(calls))
	}
}

func TestPythonAggregateUpdateRecordRejectsInputsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonAggregateAPI(t, func(*http.Request) *http.Response {
		return pythonAggregateWire(http.StatusConflict, `{"conflictingRequest":{"message":"busy"}}`)
	})
	for _, option := range []aggregates.UpdateRecordOption{
		aggregates.WithUpdateRecordField("availability_zone", nil),
		aggregates.WithUpdateRecordField("id", 8),
		nil,
	} {
		var operation *resource.OperationError
		if _, err := api.UpdateRecord(ctx, 7, option); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "UpdateRecord" {
			t.Fatal(err)
		}
	}
	if calls.Len() != 0 {
		t.Fatalf("%+v", pythonAggregateCalls(calls))
	}
	var operation *resource.OperationError
	if _, err := api.UpdateRecord(ctx, 7, aggregates.WithUpdateRecordName(request.Present("x"))); !gophercloud.ResponseCodeIs(err, http.StatusConflict) || !errors.As(err, &operation) || operation.Resource != "aggregates" {
		t.Fatal(err)
	}
	var nilAPI *aggregates.API
	if _, err := nilAPI.UpdateRecord(ctx, 7, aggregates.WithUpdateRecordName(request.Present("x"))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
