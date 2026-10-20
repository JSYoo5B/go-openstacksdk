package attachments_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeAttachmentTransport func(*http.Request) (*http.Response, error)

func (transport nativeAttachmentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeAttachmentWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeAttachmentCall struct{ method, path, query, body, version string }

func nativeAttachmentAPI(t *testing.T, calls *[]nativeAttachmentCall, reply func(*http.Request) *http.Response) (*attachments.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeAttachmentTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeAttachmentCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	// The attachment API needs a caller-selected microversion; the native call does not choose one.
	client.Microversion = "3.44"
	return attachments.New(client), cloud
}

func nativeAttachmentOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "attachments" {
		t.Fatal("generated attachments context", err, wrapped)
	}
}

const nativeAttachmentRow = `{"id":"att-1","volume_id":"vol-1","instance":"srv","status":"reserved","attach_mode":"rw","connection_info":{"driver_volume_type":"iscsi"},"attached_at":"2026-10-11T01:02:03.000000","detached_at":null}`

func TestNativeAttachmentRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeAttachmentCall
	var cloud *testcloud.Cloud
	api, cloud := nativeAttachmentAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeAttachmentWire(200, `{}`)
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/action"):
			return nativeAttachmentWire(204, "")
		case req.Method == http.MethodPost:
			return nativeAttachmentWire(200, `{"attachment":`+nativeAttachmentRow+`}`)
		case req.URL.Path == "/cinder/v3/project/attachments/detail":
			// attachments_links is ignored; only a links.next string is followed.
			return nativeAttachmentWire(200, `{"attachments":[`+nativeAttachmentRow+`],"attachments_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}],"links":{"next":"`+cloud.Server.URL+`/other/attachments"}}`)
		case req.URL.Path == "/other/attachments":
			return nativeAttachmentWire(200, `{"attachments":[{"id":"att-2","connection_info":null}]}`)
		}
		return nativeAttachmentWire(200, `{"attachment":`+nativeAttachmentRow+`}`)
	})
	created, err := api.Create(ctx, attachments.CreateOpts{VolumeUUID: "vol-1", InstanceUUID: "srv", Connector: map[string]any{"host": "compute-1"}, Mode: "rw"}, attachments.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "att-1" && created.ConnectionInfo["driver_volume_type"] == "iscsi" && created.AttachedAt.Second() == 3 && created.DetachedAt.IsZero()) {
		t.Fatal(created, err)
	}
	// Volume and instance UUIDs have no omitempty and are sent empty.
	if _, err := api.Create(ctx, attachments.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "att-1")
	if err != nil || got.Instance != "srv" {
		t.Fatal(got, err)
	}
	if _, err := api.Update(ctx, "att-1", attachments.UpdateOpts{Connector: map[string]any{"host": "compute-2"}}, attachments.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	// A nil connector is sent as null.
	if _, err := api.Update(ctx, "att-1", attachments.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.Complete(ctx, "att-1"); err != nil {
		t.Fatal(err)
	}
	var rows []*attachments.Attachment
	for value, err := range api.List(ctx, attachments.WithListOptions(attachments.ListOpts{VolumeID: "vol-1", InstanceID: "srv", AllTenants: true, Limit: 1}), attachments.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "att-1" && rows[1].ID == "att-2" && rows[1].ConnectionInfo == nil) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "att-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 9 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[6].query)
	base := "/cinder/v3/project/attachments"
	v := "volume 3.44"
	want := []nativeAttachmentCall{
		{http.MethodPost, base, "", `{"attachment":{"connector":{"host":"compute-1"},"instance_uuid":"srv","mode":"rw","volume_uuid":"vol-1","x_extension":1}}`, v},
		{http.MethodPost, base, "", `{"attachment":{"instance_uuid":"","volume_uuid":""}}`, v},
		{http.MethodGet, base + "/att-1", "", "", v},
		{http.MethodPut, base + "/att-1", "", `{"attachment":{"connector":{"host":"compute-2"},"x_extension":1}}`, v},
		{http.MethodPut, base + "/att-1", "", `{"attachment":{"connector":null}}`, v},
		{http.MethodPost, base + "/att-1/action", "", `{"os-complete":null}`, v},
		{http.MethodGet, base + "/detail", calls[6].query, "", v},
		{http.MethodGet, "/other/attachments", "", "", v},
		{http.MethodDelete, base + "/att-1", "", "", v},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"volume_id": {"vol-1"}, "instance_id": {"srv"}, "all_tenants": {"true"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeAttachmentStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*attachments.API) error
	}{
		{"Create", []int{200, 202}, func(api *attachments.API) error {
			_, err := api.Create(ctx, attachments.CreateOpts{})
			return err
		}},
		{"Get", []int{200}, func(api *attachments.API) error { _, err := api.Get(ctx, "att-1"); return err }},
		{"Update", []int{200}, func(api *attachments.API) error {
			_, err := api.Update(ctx, "att-1", attachments.UpdateOpts{})
			return err
		}},
		// Delete accepts only 200 and Complete only 204.
		{"Delete", []int{200}, func(api *attachments.API) error { return api.Delete(ctx, "att-1") }},
		{"Complete", []int{204}, func(api *attachments.API) error { return api.Complete(ctx, "att-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeAttachmentCall
				api, _ := nativeAttachmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAttachmentWire(code, `{"attachment":{}}`) })
				err := call.call(api)
				nativeAttachmentOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope and timestamp decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// An empty object or null envelope yields a zero attachment.
			`{}`:                  false,
			`{"attachment":null}`: false,
			`{"other":{}}`:        true,
			`{"attachment":{"attached_at":"2026-10-11T01:02:03Z"}}`: true,
		} {
			var calls []nativeAttachmentCall
			api, _ := nativeAttachmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAttachmentWire(200, body) })
			got, err := api.Get(ctx, "att-1")
			if wantErr {
				nativeAttachmentOperation(t, err, "Get")
			} else if err != nil || got == nil || got.ID != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"attachments":[]}`}, {204, ""}} {
			var calls []nativeAttachmentCall
			api, _ := nativeAttachmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAttachmentWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeAttachmentCall
		api, _ := nativeAttachmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAttachmentWire(200, `{}`) })
		for operation, err := range map[string]error{
			"Create": func() error {
				_, err := api.Create(ctx, attachments.CreateOpts{}, attachments.WithCreateField("volume_uuid", "x"))
				return err
			}(),
			"Update": func() error { _, err := api.Update(ctx, "att-1", attachments.UpdateOpts{}, nil); return err }(),
		} {
			if err == nil {
				t.Fatal(operation, "accepted")
			}
			nativeAttachmentOperation(t, err, operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeAttachmentOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}

func TestNativeAttachmentWaitForStatus(t *testing.T) {
	ctx := context.Background()
	t.Run("immediate match", func(t *testing.T) {
		var calls []nativeAttachmentCall
		api, _ := nativeAttachmentAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeAttachmentWire(200, `{"attachment":{"id":"att-1","status":"attached"}}`)
		})
		if err := api.WaitForStatus(ctx, "att-1", "attached"); err != nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
	t.Run("stops on Get errors", func(t *testing.T) {
		var calls []nativeAttachmentCall
		api, _ := nativeAttachmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAttachmentWire(404, `{}`) })
		err := api.WaitForStatus(ctx, "att-1", "attached")
		nativeAttachmentOperation(t, err, "WaitForStatus")
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	})
	t.Run("missing envelope keeps polling until the deadline", func(t *testing.T) {
		var calls []nativeAttachmentCall
		api, _ := nativeAttachmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAttachmentWire(200, `{}`) })
		deadline, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		err := api.WaitForStatus(deadline, "att-1", "attached")
		nativeAttachmentOperation(t, err, "WaitForStatus")
		if !errors.Is(err, context.DeadlineExceeded) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}
