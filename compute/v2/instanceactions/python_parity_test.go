package instanceactions_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// pythonActionCall records the wire request openstacksdk would also send.
type pythonActionCall struct{ method, path, version string }

type pythonActionTransport func(*http.Request) (*http.Response, error)

func (transport pythonActionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func TestPythonServerActionGetReturnsDetailWithEvents(t *testing.T) {
	ctx := context.Background()
	var calls []pythonActionCall
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonActionTransport(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, pythonActionCall{req.Method, req.URL.Path, req.Header.Get("X-OpenStack-Nova-API-Version")})
		code, body := 200, `{"instanceAction":{"action":"reboot","request_id":"req-1","instance_uuid":"s1","user_id":"u","project_id":"p","message":null,"start_time":"2026-10-10T01:02:03.000000","updated_at":"2026-10-10T01:02:04.000000","events":[{"event":"compute_reboot_instance","result":"Error","host":"h1","hostId":"hid","traceback":"tb","details":"boom","start_time":"2026-10-10T01:02:03.000000","finish_time":"2026-10-10T01:02:04.000000"}]}}`
		if strings.HasSuffix(req.URL.Path, "/req-0") {
			code, body = 404, `{"itemNotFound":{"message":"gone"}}`
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.84"
	scope, err := instanceactions.New(client).InServer(ctx, resource.ID("s1"))
	if err != nil {
		t.Fatal(err)
	}
	// get_server_action(request_id, server)
	got, err := scope.Get(ctx, "req-1")
	if err != nil || got.RequestID != "req-1" || got.Action != "reboot" || got.InstanceUUID != "s1" || got.UserID != "u" || got.ProjectID != "p" || got.ServerID != "s1" {
		t.Fatal(got, err)
	}
	if got.Events == nil || len(*got.Events) != 1 {
		t.Fatal(got.Events)
	}
	event := (*got.Events)[0]
	if event.Event != "compute_reboot_instance" || event.Result != "Error" || event.Host == nil || *event.Host != "h1" || event.HostID == nil || *event.HostID != "hid" || event.Traceback != "tb" || event.Details == nil || *event.Details != "boom" {
		t.Fatalf("%+v", event)
	}
	// Python's ignore_missing argument is not applied by _get, so a missing action raises.
	if _, err := scope.Get(ctx, "req-0"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	want := []pythonActionCall{
		{http.MethodGet, "/nova/v2.1/servers/s1/os-instance-actions/req-1", "2.84"},
		{http.MethodGet, "/nova/v2.1/servers/s1/os-instance-actions/req-0", "2.84"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
