package services_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeCinderServiceTransport func(*http.Request) (*http.Response, error)

func (transport nativeCinderServiceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeCinderServiceWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeCinderServiceCall struct{ method, path, query, body string }

func nativeCinderServiceAPI(t *testing.T, calls *[]nativeCinderServiceCall, reply func(*http.Request) *http.Response) (*services.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeCinderServiceTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeCinderServiceCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return services.New(client), cloud
}

func TestNativeCinderServiceListRouteAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeCinderServiceCall
	var cloud *testcloud.Cloud
	api, cloud := nativeCinderServiceAPI(t, &calls, func(*http.Request) *http.Response {
		// The service list is a single page, so the link is never followed.
		return nativeCinderServiceWire(200, `{"services":[
			{"binary":"cinder-volume","host":"h@lvm","zone":"nova","status":"enabled","state":"up","updated_at":"2017-06-29T05:50:35.000000","disabled_reason":null,"frozen":false,"cluster":"c","replication_status":"disabled","active_backend_id":null},
			{"binary":"cinder-scheduler","host":"h","updated_at":null}],
			"services_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}]}`)
	})
	var rows []services.Service
	for value, err := range api.List(ctx, services.WithListOptions(services.ListOpts{Binary: "cinder-volume", Host: "h@lvm"}), services.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	for _, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(rows) != 2 {
		t.Fatal(rows)
	}
	// updated_at uses the millisecond layout without a zone; null leaves a zero time.
	if !(rows[0].Binary == "cinder-volume" && rows[0].State == "up" && rows[0].Status == "enabled" && rows[0].Zone == "nova" && rows[0].Cluster == "c" &&
		rows[0].ReplicationStatus == "disabled" && rows[0].UpdatedAt.Equal(time.Date(2017, 6, 29, 5, 50, 35, 0, time.UTC)) && rows[1].UpdatedAt.IsZero()) {
		t.Fatalf("%+v", rows)
	}
	path := "/cinder/v3/project/os-services"
	want := []nativeCinderServiceCall{
		{http.MethodGet, path, "binary=cinder-volume&extra=1&host=h%40lvm", ""},
		{http.MethodGet, path, "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeCinderServiceListStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	t.Run("pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {201, `{}`}, {200, `{"services":[]}`}, {204, ""}} {
			var calls []nativeCinderServiceCall
			api, _ := nativeCinderServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeCinderServiceWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case (tc.code == 404 || tc.code == 201) && len(errs) == 1 && errors.As(errs[0], &native) && native.Actual == tc.code && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		}
	})
	t.Run("decode", func(t *testing.T) {
		var calls []nativeCinderServiceCall
		api, _ := nativeCinderServiceAPI(t, &calls, func(*http.Request) *http.Response {
			// A zone suffix does not match the native timestamp layout.
			return nativeCinderServiceWire(200, `{"services":[{"binary":"cinder-volume","updated_at":"2017-06-29T05:50:35Z"}]}`)
		})
		var errs []error
		for _, err := range api.List(ctx) {
			errs = append(errs, err)
		}
		if len(errs) != 1 || errs[0] == nil {
			t.Fatal(errs)
		}
	})
	t.Run("nil option", func(t *testing.T) {
		var calls []nativeCinderServiceCall
		api, _ := nativeCinderServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeCinderServiceWire(200, `{}`) })
		for _, err := range api.List(ctx, nil) {
			var wrapped *resource.OperationError
			if !errors.As(err, &wrapped) || wrapped.Operation != "List" || wrapped.Resource != "services" {
				t.Fatal(err)
			}
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
