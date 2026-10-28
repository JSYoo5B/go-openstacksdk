package attachments_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// pythonAttachmentCall records the wire request openstacksdk would also send.
type pythonAttachmentCall struct{ method, path, query, body string }

func pythonAttachmentWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

func pythonAttachmentAPI(t *testing.T, reply func(*http.Request) *http.Response) (*attachments.API, *testcloud.Recorder) {
	t.Helper()
	calls := testcloud.NewRecorder(func(req *http.Request) (*http.Response, error) { return reply(req), nil })
	cloud := testcloud.New(t)
	cloud.SetTransport(calls)
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = "3.54"
	return attachments.New(client), calls
}

// pythonAttachmentCalls projects recorded calls onto the local call fields.
func pythonAttachmentCalls(rec *testcloud.Recorder) []pythonAttachmentCall {
	calls := []pythonAttachmentCall{}
	for _, call := range rec.Calls() {
		calls = append(calls, pythonAttachmentCall{call.Method, call.Path, call.RawQuery, call.Body})
	}
	return calls
}

const pythonAttachmentBase = "/cinder/v3/project/attachments"

const pythonAttachmentRow = `{"id":"att-1","volume_id":"vol-1","instance":"srv","status":"attached","attach_mode":"rw","connection_info":{"driver_volume_type":"iscsi"},"attached_at":"2026-10-11T01:02:03.000000"}`

func TestPythonAttachmentCrudAndCompleteMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonAttachmentAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return pythonAttachmentWire(200, `{}`)
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/action"):
			return pythonAttachmentWire(204, "")
		}
		return pythonAttachmentWire(200, `{"attachment":`+pythonAttachmentRow+`}`)
	})
	// create_attachment(volume, connector=..., instance=..., mode="rw") renames volume_id and instance.
	created, err := api.Create(ctx, attachments.CreateOpts{VolumeUUID: "vol-1", InstanceUUID: "srv", Connector: map[string]any{"host": "compute-1", "nqn": "nqn.x"}, Mode: "rw"})
	if err != nil || created.ID != "att-1" || created.ConnectionInfo["driver_volume_type"] != "iscsi" {
		t.Fatal(created, err)
	}
	// create_attachment(volume) without instance omits instance_uuid in Python; Go always sends it.
	if _, err := api.Create(ctx, attachments.CreateOpts{VolumeUUID: "vol-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Create(ctx, attachments.CreateOpts{VolumeUUID: "vol-1"}, attachments.WithCreateField("instance_uuid", nil)); err == nil {
		t.Fatal("core instance_uuid was replaced")
	}
	// get_attachment(attachment)
	got, err := api.Get(ctx, "att-1")
	if err != nil || got.Instance != "srv" || got.VolumeID != "vol-1" || got.Status != "attached" || got.AttachMode != "rw" {
		t.Fatal(got, err)
	}
	// update_attachment(attachment, connector=...)
	if _, err := api.Update(ctx, "att-1", attachments.UpdateOpts{Connector: map[string]any{"host": "compute-2"}}); err != nil {
		t.Fatal(err)
	}
	// complete_attachment(attachment): Python sends the attachment ID as the action value.
	if err := api.Complete(ctx, "att-1"); err != nil {
		t.Fatal(err)
	}
	// delete_attachment(attachment)
	if err := api.Remove(ctx, resource.ID("att-1")); err != nil {
		t.Fatal(err)
	}
	want := []pythonAttachmentCall{
		{http.MethodPost, pythonAttachmentBase, "", `{"attachment":{"connector":{"host":"compute-1","nqn":"nqn.x"},"instance_uuid":"srv","mode":"rw","volume_uuid":"vol-1"}}`},
		{http.MethodPost, pythonAttachmentBase, "", `{"attachment":{"instance_uuid":"","volume_uuid":"vol-1"}}`},
		{http.MethodGet, pythonAttachmentBase + "/att-1", "", ""},
		{http.MethodPut, pythonAttachmentBase + "/att-1", "", `{"attachment":{"connector":{"host":"compute-2"}}}`},
		{http.MethodPost, pythonAttachmentBase + "/att-1/action", "", `{"os-complete":null}`},
		{http.MethodDelete, pythonAttachmentBase + "/att-1", "", ""},
	}
	if !reflect.DeepEqual(pythonAttachmentCalls(calls), want) {
		t.Fatalf("%+v", pythonAttachmentCalls(calls))
	}
}

func TestPythonAttachmentDeleteIgnoresMissing(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response {
		return pythonAttachmentWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	// delete_attachment(attachment) ignores 404 by default; ignore_missing=False raises.
	if err := api.Remove(ctx, resource.ID("att-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("att-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if calls.Len() != 2 {
		t.Fatalf("%+v", pythonAttachmentCalls(calls))
	}
}

func TestPythonAttachmentListUsesDetailFirstPage(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonAttachmentAPI(t, func(req *http.Request) *http.Response {
		return pythonAttachmentWire(200, `{"attachments":[`+pythonAttachmentRow+`],"attachments_links":[{"rel":"next","href":"http://`+req.Host+pythonAttachmentBase+`?marker=att-1"}]}`)
	})
	// attachments(volume_id="vol-1", all_projects=True): Python lists GET /attachments and follows attachments_links.
	var ids []string
	for value, err := range api.List(ctx, attachments.WithListOptions(attachments.ListOpts{VolumeID: "vol-1"}), attachments.WithListQuery("all_tenants", "True")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	query, _ := url.ParseQuery(pythonAttachmentCalls(calls)[0].query)
	if !reflect.DeepEqual(ids, []string{"att-1"}) || calls.Len() != 1 || pythonAttachmentCalls(calls)[0].path != pythonAttachmentBase+"/detail" || !reflect.DeepEqual(query, url.Values{"volume_id": {"vol-1"}, "all_tenants": {"True"}}) {
		t.Fatalf("%v %+v", ids, pythonAttachmentCalls(calls))
	}
}
