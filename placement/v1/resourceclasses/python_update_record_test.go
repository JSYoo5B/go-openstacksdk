package resourceclasses_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonResourceClassUpdateRecordRenames(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusOK, `{"name":"CUSTOM_NEW","links":[{"rel":"self","href":"/resource_classes/CUSTOM_NEW"}]}`
	var recorder pythonRCRecorder
	api := pythonRCAPI(t, &recorder, func(*http.Request) *http.Response { return pythonRCWire(status, reply) })
	// The client selects 1.39 for other calls; the rename still uses 1.2.
	api.RawClient().Microversion = "1.39"
	api.RawClient().MoreHeaders = map[string]string{"OpenStack-API-Version": "placement 1.39"}
	// update_resource_class("CUSTOM_OLD", name="CUSTOM_NEW")
	got, err := api.UpdateRecord(ctx, "CUSTOM_OLD", resourceclasses.WithUpdateRecordName("CUSTOM_NEW"))
	if err != nil || got.StatusCode != 200 || got.ResourceClass == nil || got.ResourceClass.Name != "CUSTOM_NEW" || len(got.ResourceClass.Links) != 1 || string(got.Fields["name"]) != `"CUSTOM_NEW"` {
		t.Fatal(got, err)
	}
	// update_resource_class("CUSTOM_OLD") commits an empty body
	// (allow_empty_commit), and a 204 leaves no decoded fields.
	status, reply = http.StatusNoContent, ``
	if got, err := api.UpdateRecord(ctx, "CUSTOM_OLD"); err != nil || got.StatusCode != 204 || got.ResourceClass != nil || got.Fields != nil {
		t.Fatal(got, err)
	}
	// update_resource_class("CUSTOM_OLD", name=None)
	status, reply = http.StatusCreated, `{"name":"CUSTOM_OLD"}`
	if got, err := api.UpdateRecord(ctx, "CUSTOM_OLD", resourceclasses.WithUpdateRecordOptions(resourceclasses.UpdateRecordOpts{Name: request.Null[string]()}), resourceclasses.WithUpdateRecordHeader("X-Trace", "t")); err != nil || got.StatusCode != 201 {
		t.Fatal(got, err)
	}
	want := []pythonRCCall{
		{http.MethodPut, pythonRCBase + "/CUSTOM_OLD", "", `{"name":"CUSTOM_NEW"}`, "placement 1.2"},
		{http.MethodPut, pythonRCBase + "/CUSTOM_OLD", "", `{}`, "placement 1.2"},
		{http.MethodPut, pythonRCBase + "/CUSTOM_OLD", "", `{"name":null}`, "placement 1.2"},
	}
	calls := recorder.snapshot()
	if len(calls) != len(want) {
		t.Fatalf("%+v", calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("call %d: %+v", i, calls[i])
		}
	}
	if api.RawClient().Microversion != "1.39" {
		t.Fatal(api.RawClient().Microversion)
	}
}

func TestPythonResourceClassUpdateRecordRejections(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusConflict, `{"errors":[{"status":409,"title":"Conflict"}]}`
	var recorder pythonRCRecorder
	api := pythonRCAPI(t, &recorder, func(*http.Request) *http.Response { return pythonRCWire(status, reply) })
	var operation *resource.OperationError
	if _, err := api.UpdateRecord(ctx, "CUSTOM_OLD", resourceclasses.WithUpdateRecordName("CUSTOM_TAKEN")); !gophercloud.ResponseCodeIs(err, http.StatusConflict) || !errors.As(err, &operation) || operation.Operation != "UpdateRecord" || operation.Resource != "resourceclasses" {
		t.Fatal(err)
	}
	status, reply = http.StatusOK, `["not","an","object"]`
	var receipt *resource.ResponseError
	if _, err := api.UpdateRecord(ctx, "CUSTOM_OLD"); !errors.As(err, &receipt) || receipt.StatusCode != 200 {
		t.Fatal(err)
	}
	if len(recorder.snapshot()) != 2 {
		t.Fatal(recorder.snapshot())
	}
	var nilAPI *resourceclasses.API
	for _, call := range []func() error{
		func() error { _, err := api.UpdateRecord(ctx, " "); return err },
		func() error { _, err := api.UpdateRecord(ctx, "bad\nname"); return err },
		func() error { _, err := api.UpdateRecord(ctx, "CUSTOM_OLD", nil); return err },
		func() error {
			_, err := api.UpdateRecord(ctx, "CUSTOM_OLD", resourceclasses.WithUpdateRecordHeader("OpenStack-API-Version", "placement 1.39"))
			return err
		},
		func() error { _, err := nilAPI.UpdateRecord(ctx, "CUSTOM_OLD"); return err },
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(recorder.snapshot()) != 2 {
		t.Fatal(recorder.snapshot())
	}
}
