package volumeattach_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeVolumeTransport func(*http.Request) (*http.Response, error)

func (transport nativeVolumeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeVolumeWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeVolumeClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return client
}

func nativeVolumeOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "volumeattach" {
		t.Fatal("generated volume attachment context", err, wrapped)
	}
}

type nativeVolumeCall struct{ method, path, query, body string }

const nativeVolumeRow = `{"id":"v1","device":"/dev/vdb","volumeId":"v1","serverId":"s1","tag":"data","delete_on_termination":true}`

func TestNativeVolumeAttachRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeVolumeCall
	cloud.Provider.HTTPClient.Transport = nativeVolumeTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		calls = append(calls, nativeVolumeCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		switch {
		case req.Method == http.MethodDelete:
			return nativeVolumeWire(202, ""), nil
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/os-volume_attachments"):
			// Older rows without tag or delete_on_termination decode to nil pointers.
			return nativeVolumeWire(200, `{"volumeAttachments":[`+nativeVolumeRow+`,{"id":"v2","volumeId":"v2"}],"volumeAttachments_links":[{"rel":"next","href":"http://other/next"}]}`), nil
		}
		return nativeVolumeWire(200, `{"volumeAttachment":`+nativeVolumeRow+`}`), nil
	})
	api := volumeattach.New(nativeVolumeClient(cloud))
	ctx := context.Background()
	created, err := api.Create(ctx, "s1", volumeattach.CreateOpts{VolumeID: "v1", Device: "/dev/vdb", Tag: "data", DeleteOnTermination: true}, volumeattach.WithCreateField("x_extension", 1))
	if err != nil || created.ID != "v1" || created.ServerID != "s1" || *created.Tag != "data" || !*created.DeleteOnTermination {
		t.Fatal(created, err)
	}
	if _, err := api.Create(ctx, "s1", volumeattach.CreateOpts{VolumeID: "v1"}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "s1", "v1")
	if err != nil || got.Device != "/dev/vdb" {
		t.Fatal(got, err)
	}
	var rows []volumeattach.VolumeAttachment
	for value, err := range api.List(ctx, "s1") {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	if len(rows) != 2 || rows[0].ID != "v1" || rows[1].Tag != nil || rows[1].DeleteOnTermination != nil {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "s1", "v1"); err != nil {
		t.Fatal(err)
	}
	want := []nativeVolumeCall{
		{http.MethodPost, "/nova/v2.1/servers/s1/os-volume_attachments", "", `{"volumeAttachment":{"delete_on_termination":true,"device":"/dev/vdb","tag":"data","volumeId":"v1","x_extension":1}}`},
		// Empty device, tag and false delete_on_termination are omitted.
		{http.MethodPost, "/nova/v2.1/servers/s1/os-volume_attachments", "", `{"volumeAttachment":{"volumeId":"v1"}}`},
		{http.MethodGet, "/nova/v2.1/servers/s1/os-volume_attachments/v1", "", ""},
		{http.MethodGet, "/nova/v2.1/servers/s1/os-volume_attachments", "", ""},
		{http.MethodDelete, "/nova/v2.1/servers/s1/os-volume_attachments/v1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeVolumeAttachStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*volumeattach.API) error
	}{
		{"Create", []int{200}, func(api *volumeattach.API) error {
			_, err := api.Create(ctx, "s1", volumeattach.CreateOpts{VolumeID: "v1"})
			return err
		}},
		{"Get", []int{200}, func(api *volumeattach.API) error { _, err := api.Get(ctx, "s1", "v1"); return err }},
		{"Delete", []int{202, 204}, func(api *volumeattach.API) error { return api.Delete(ctx, "s1", "v1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeVolumeTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeVolumeWire(code, `{"volumeAttachment":{"id":"v1"}}`), nil
				})
				err := call.call(volumeattach.New(nativeVolumeClient(cloud)))
				nativeVolumeOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("List native pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
			want func([]error) bool
		}{
			{404, `{}`, func(errs []error) bool {
				var native gophercloud.ErrUnexpectedResponseCode
				return len(errs) == 1 && errors.As(errs[0], &native) && native.Actual == 404 && reflect.DeepEqual(native.Expected, []int{200, 204, 300})
			}},
			{200, `{"volumeAttachments":[]}`, func(errs []error) bool { return len(errs) == 0 }},
			{204, "", func(errs []error) bool { return len(errs) == 1 && errors.Is(errs[0], io.EOF) }},
		} {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = nativeVolumeTransport(func(req *http.Request) (*http.Response, error) {
				return nativeVolumeWire(tc.code, tc.body), nil
			})
			var errs []error
			for value, err := range volumeattach.New(nativeVolumeClient(cloud)).List(ctx, "s1") {
				if value != nil {
					t.Fatal(value)
				}
				errs = append(errs, err)
			}
			if !tc.want(errs) {
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeVolumeTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeVolumeWire(200, `{}`), nil
		})
		api := volumeattach.New(nativeVolumeClient(cloud))
		for name, check := range map[string]func() error{
			"missing volume": func() error { _, err := api.Create(ctx, "s1", volumeattach.CreateOpts{}); return err },
			"volumeId extension": func() error {
				_, err := api.Create(ctx, "s1", volumeattach.CreateOpts{VolumeID: "v"}, volumeattach.WithCreateField("volumeId", "x"))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, "s1", volumeattach.CreateOpts{VolumeID: "v"}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeVolumeOperation(t, err, "Create")
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
}

func contains(values []int, value int) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
