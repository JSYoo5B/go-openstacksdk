package instanceactions_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeActionTransport func(*http.Request) (*http.Response, error)

func (transport nativeActionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeActionWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeActionOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "instanceactions" {
		t.Fatal("generated instanceactions context", err, wrapped)
	}
}

type nativeActionCall struct{ method, path, query string }

func nativeActionRecorder(cloud *testcloud.Cloud, calls *[]nativeActionCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeActionTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeActionCall{req.Method, req.URL.Path, req.URL.RawQuery})
		return reply(req), nil
	})
}

func nativeActionStatus(t *testing.T, err error, operation string, code int, expected []int) {
	t.Helper()
	nativeActionOperation(t, err, operation)
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, expected) {
		t.Fatal(err, native)
	}
}

func nativeActionPagerStatus(t *testing.T, errs []error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if len(errs) != 1 || !errors.As(errs[0], &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200, 204, 300}) {
		t.Fatal(errs)
	}
}

func nativeActionClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return client
}

func TestNativeInstanceActionsRoutesQueryAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeActionCall
	nativeActionRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/os-instance-actions") {
			// Links are ignored: the native list is a single page.
			return nativeActionWire(200, `{"instanceActions":[{"action":"reboot","request_id":"req-1","start_time":"2026-10-10T01:02:03.000000","user_id":"u"},{"action":"create","request_id":"req-0","start_time":""}],"links":[{"rel":"next","href":"http://other/next"}]}`)
		}
		return nativeActionWire(200, `{"instanceAction":{"action":"reboot","request_id":"req-1","instance_uuid":"s1","start_time":"2026-10-10T01:02:03.5","updated_at":"2026-10-10T01:02:04.000000","events":[{"event":"compute_reboot","result":"Success","host":null,"start_time":"2026-10-10T01:02:03.000000","finish_time":null}]}}`)
	})
	api := instanceactions.New(nativeActionClient(cloud))
	ctx := context.Background()
	since := time.Date(2026, 10, 10, 10, 0, 0, 0, time.FixedZone("KST", 9*3600))
	var rows []instanceactions.InstanceAction
	for value, err := range api.List(ctx, "s1", instanceactions.WithListOptions(instanceactions.ListOpts{Limit: 2, Marker: "req-9", ChangesSince: &since, ChangesBefore: &since}), instanceactions.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	if len(rows) != 2 || rows[0].RequestID != "req-1" || !rows[0].StartTime.Equal(time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC)) || !rows[1].StartTime.IsZero() {
		t.Fatal(rows)
	}
	detail, err := api.Get(ctx, "s1", "req-1")
	if err != nil || detail.InstanceUUID != "s1" || detail.UpdatedAt == nil || detail.Events == nil || len(*detail.Events) != 1 || (*detail.Events)[0].Host != nil || !(*detail.Events)[0].FinishTime.IsZero() || detail.StartTime.Nanosecond() != 500000000 {
		t.Fatal(detail, err)
	}
	// changes-since/before keep the caller's offset in RFC3339 form.
	want := []nativeActionCall{
		{http.MethodGet, "/nova/v2.1/servers/s1/os-instance-actions", "changes-before=2026-10-10T10%3A00%3A00%2B09%3A00&changes-since=2026-10-10T10%3A00%3A00%2B09%3A00&extra=1&limit=2&marker=req-9"},
		{http.MethodGet, "/nova/v2.1/servers/s1/os-instance-actions/req-1", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeInstanceActionsStatusesAndDecodeErrors(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 203, 204, 404} {
		cloud := testcloud.New(t)
		var calls []nativeActionCall
		nativeActionRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeActionWire(code, `{"instanceAction":{}}`) })
		_, err := instanceactions.New(nativeActionClient(cloud)).Get(ctx, "s1", "req-1")
		nativeActionStatus(t, err, "Get", code, []int{200})
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
	t.Run("zoned timestamps are rejected", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls []nativeActionCall
		nativeActionRecorder(cloud, &calls, func(*http.Request) *http.Response {
			return nativeActionWire(200, `{"instanceAction":{"start_time":"2026-10-10T01:02:03Z"}}`)
		})
		if _, err := instanceactions.New(nativeActionClient(cloud)).Get(ctx, "s1", "req-1"); err == nil {
			t.Fatal("RFC3339 with zone accepted")
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"instanceActions":[]}`}, {204, ""}} {
			cloud := testcloud.New(t)
			var calls []nativeActionCall
			nativeActionRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeActionWire(tc.code, tc.body) })
			var errs []error
			for _, err := range instanceactions.New(nativeActionClient(cloud)).List(ctx, "s1") {
				errs = append(errs, err)
			}
			switch tc.code {
			case 404:
				nativeActionPagerStatus(t, errs, 404)
			case 200:
				if len(errs) != 0 {
					t.Fatal(errs)
				}
			case 204:
				if len(errs) != 1 || !errors.Is(errs[0], io.EOF) {
					t.Fatal(errs)
				}
			}
		}
	})
	t.Run("list preflight", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls []nativeActionCall
		nativeActionRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeActionWire(200, `{}`) })
		for _, err := range instanceactions.New(nativeActionClient(cloud)).List(ctx, "s1", nil) {
			nativeActionOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
