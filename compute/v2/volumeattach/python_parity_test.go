package volumeattach_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// pythonVolumeAttachCall records the wire request openstacksdk would also send.
type pythonVolumeAttachCall struct{ method, path, query, body string }

type pythonVolumeAttachTransport func(*http.Request) (*http.Response, error)

func (transport pythonVolumeAttachTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonVolumeAttachWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonVolumeAttachAPI(t *testing.T, reply func(*http.Request) *http.Response) (*volumeattach.API, *[]pythonVolumeAttachCall) {
	t.Helper()
	calls := &[]pythonVolumeAttachCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonVolumeAttachTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonVolumeAttachCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.89"
	return volumeattach.New(client), calls
}

const pythonVolumeAttachBase = "/nova/v2.1/servers/s1/os-volume_attachments"

const pythonVolumeAttachRow = `{"id":"v1","volumeId":"v1","serverId":"s1","device":"/dev/vdb","tag":"data","delete_on_termination":true,"attachment_id":"a1","bdm_uuid":"b1"}`

func TestPythonVolumeAttachmentCreateGetListDeleteMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	deleteStatus := http.StatusNotFound
	api, calls := pythonVolumeAttachAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return pythonVolumeAttachWire(deleteStatus, `{"itemNotFound":{"message":"gone"}}`)
		case req.URL.Path == pythonVolumeAttachBase && req.Method == http.MethodGet:
			return pythonVolumeAttachWire(200, `{"volumeAttachments":[`+pythonVolumeAttachRow+`]}`)
		}
		return pythonVolumeAttachWire(200, `{"volumeAttachment":`+pythonVolumeAttachRow+`}`)
	})
	check := func(value *volumeattach.VolumeAttachment) {
		t.Helper()
		if value == nil || value.ID != "v1" || value.VolumeID != "v1" || value.ServerID != "s1" || value.Device != "/dev/vdb" ||
			value.Tag == nil || *value.Tag != "data" || value.DeleteOnTermination == nil || !*value.DeleteOnTermination {
			t.Fatalf("%+v", value)
		}
	}
	// create_volume_attachment(server, volume, device=..., tag=..., delete_on_termination=True)
	created, err := api.Create(ctx, "s1", volumeattach.CreateOpts{VolumeID: "v1", Device: "/dev/vdb", Tag: "data", DeleteOnTermination: true})
	if err != nil {
		t.Fatal(err)
	}
	check(created)
	// create_volume_attachment(server, volume)
	if _, err := api.Create(ctx, "s1", volumeattach.CreateOpts{VolumeID: "v1"}); err != nil {
		t.Fatal(err)
	}
	// get_volume_attachment(server, volume)
	got, err := api.Get(ctx, "s1", "v1")
	if err != nil {
		t.Fatal(err)
	}
	check(got)
	// volume_attachments(server)
	var listed []*volumeattach.VolumeAttachment
	for value, err := range api.List(ctx, "s1") {
		if err != nil {
			t.Fatal(err)
		}
		listed = append(listed, value)
	}
	if len(listed) != 1 {
		t.Fatal(listed)
	}
	check(listed[0])
	// delete_volume_attachment(server, volume) ignores a missing attachment by default.
	scope, err := api.InServer(ctx, resource.ID("s1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID("v1")); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID("v1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	deleteStatus = http.StatusAccepted
	if err := scope.Delete(ctx, resource.ID("v1")); err != nil {
		t.Fatal(err)
	}
	want := []pythonVolumeAttachCall{
		{http.MethodPost, pythonVolumeAttachBase, "", `{"volumeAttachment":{"delete_on_termination":true,"device":"/dev/vdb","tag":"data","volumeId":"v1"}}`},
		{http.MethodPost, pythonVolumeAttachBase, "", `{"volumeAttachment":{"volumeId":"v1"}}`},
		{http.MethodGet, pythonVolumeAttachBase + "/v1", "", ""},
		{http.MethodGet, pythonVolumeAttachBase, "", ""},
		{http.MethodDelete, pythonVolumeAttachBase + "/v1", "", ""},
		{http.MethodDelete, pythonVolumeAttachBase + "/v1", "", ""},
		{http.MethodDelete, pythonVolumeAttachBase + "/v1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonVolumeAttachmentListCannotSendLimitOrOffset(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonVolumeAttachAPI(t, func(*http.Request) *http.Response {
		return pythonVolumeAttachWire(200, `{"volumeAttachments":[]}`)
	})
	scope, err := api.InServer(ctx, resource.ID("s1"))
	if err != nil {
		t.Fatal(err)
	}
	// volume_attachments(server, limit=1, offset=1) sends both; the Go list rejects any query.
	rejected := false
	for _, err := range scope.List(ctx, resource.WithQuery("limit", "1"), resource.WithQuery("offset", "1")) {
		if !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
		rejected = true
	}
	if !rejected || len(*calls) != 0 {
		t.Fatalf("%+v", *calls)
	}
}
