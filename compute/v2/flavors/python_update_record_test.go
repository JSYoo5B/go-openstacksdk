package flavors_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonFlavorUpdateRecordSendsNullAndEmptyDescription(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusOK, `{"flavor":`+pythonFlavorRow+`}`
	api, calls := pythonFlavorAPI(t, func(*http.Request) *http.Response { return pythonFlavorWire(status, reply) })
	// update_flavor("f1", description=None) clears the description in Nova.
	got, err := api.UpdateRecord(ctx, "f1", flavors.WithUpdateRecordDescription(request.Null[string]()))
	if err != nil || !got.Sent || got.StatusCode != 200 || got.Flavor == nil || got.Flavor.ID != "f1" || got.Flavor.Ephemeral != 5 || string(got.Fields["description"]) != `"d"` {
		t.Fatal(got, err)
	}
	// update_flavor("f1", description="", x_extension=1) with a flat 202
	// response; Python accepts every status below 400.
	status, reply = http.StatusAccepted, `{"id":"f1","name":"m1","ram":512,"vcpus":1,"disk":10,"description":""}`
	got, err = api.UpdateRecord(ctx, "f1",
		flavors.WithUpdateRecordOptions(flavors.UpdateRecordOpts{Description: request.Present("")}),
		flavors.WithUpdateRecordField("x_extension", 1), flavors.WithUpdateRecordHeader("X-Trace", "t"))
	if err != nil || got.StatusCode != 202 || got.Flavor.Description != "" || got.Flavor.RAM != 512 || len(got.Fields) != 6 {
		t.Fatal(got, err)
	}
	// update_flavor("f1", description="new") on an empty 200 body.
	status, reply = http.StatusOK, ``
	if got, err = api.UpdateRecord(ctx, "f1", flavors.WithUpdateRecordDescription(request.Present("new"))); err != nil || !got.Sent || got.Flavor != nil || got.Fields != nil {
		t.Fatal(got, err)
	}
	// update_flavor("f1") has nothing dirty, so Python sends no request.
	if got, err := api.UpdateRecord(ctx, "f1"); err != nil || got.Sent || got.Flavor != nil {
		t.Fatal(got, err)
	}
	want := []pythonFlavorCall{
		{http.MethodPut, pythonFlavorBase + "/f1", "", `{"flavor":{"description":null}}`, "2.61"},
		{http.MethodPut, pythonFlavorBase + "/f1", "", `{"flavor":{"description":"","x_extension":1}}`, "2.61"},
		{http.MethodPut, pythonFlavorBase + "/f1", "", `{"flavor":{"description":"new"}}`, "2.61"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonFlavorUpdateRecordRejectsInputsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonFlavorAPI(t, func(*http.Request) *http.Response {
		return pythonFlavorWire(http.StatusConflict, `{"conflictingRequest":{"message":"busy"}}`)
	})
	for _, option := range []flavors.UpdateRecordOption{
		flavors.WithUpdateRecordField("description", nil),
		flavors.WithUpdateRecordField("id", "f2"),
		nil,
	} {
		var operation *resource.OperationError
		if _, err := api.UpdateRecord(ctx, "f1", option); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "UpdateRecord" {
			t.Fatal(err)
		}
	}
	if _, err := api.UpdateRecord(ctx, " ", flavors.WithUpdateRecordDescription(request.Present("x"))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatalf("%+v", *calls)
	}
	var operation *resource.OperationError
	if _, err := api.UpdateRecord(ctx, "f1", flavors.WithUpdateRecordDescription(request.Present("x"))); !gophercloud.ResponseCodeIs(err, http.StatusConflict) || !errors.As(err, &operation) || operation.Resource != "flavors" {
		t.Fatal(err)
	}
	var nilAPI *flavors.API
	if _, err := nilAPI.UpdateRecord(ctx, "f1", flavors.WithUpdateRecordDescription(request.Present("x"))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
