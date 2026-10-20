package aggregates_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeAggregateTransport func(*http.Request) (*http.Response, error)

func (transport nativeAggregateTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeAggregateWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeAggregateCall struct{ method, path, query, body string }

func nativeAggregateAPI(t *testing.T, calls *[]nativeAggregateCall, reply func(*http.Request) *http.Response) *aggregates.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeAggregateTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeAggregateCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return aggregates.New(client)
}

func nativeAggregateOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "aggregates" {
		t.Fatal("generated aggregates context", err, wrapped)
	}
}

const nativeAggregateRow = `{"id":7,"uuid":"agg-uuid","name":"rack1","availability_zone":"az1","hosts":["cmp1"],"metadata":{"availability_zone":"az1","ssd":"true"},"created_at":"2026-10-01T01:02:03.000000","updated_at":null,"deleted_at":null,"deleted":false}`

func TestNativeAggregateRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeAggregateCall
	api := nativeAggregateAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet && req.URL.Path == "/nova/v2.1/os-aggregates" {
			// A single page: embedded links are never followed.
			return nativeAggregateWire(200, `{"aggregates":[`+nativeAggregateRow+`,{"id":8,"name":"rack2","hosts":[]}],"aggregates_links":[{"rel":"next","href":"http://ignored/"}]}`)
		}
		return nativeAggregateWire(200, `{"aggregate":`+nativeAggregateRow+`}`)
	})
	created, err := api.Create(ctx, aggregates.CreateOpts{Name: "rack1", AvailabilityZone: "az1"}, aggregates.WithCreateField("x_extension", 1))
	created2 := time.Date(2026, 10, 1, 1, 2, 3, 0, time.UTC)
	if err != nil || created.ID != 7 || created.UUID != "agg-uuid" || !created.CreatedAt.Equal(created2) || !created.UpdatedAt.IsZero() || !reflect.DeepEqual(created.Hosts, []string{"cmp1"}) || created.Metadata["ssd"] != "true" {
		t.Fatal(created, err)
	}
	if _, err := api.Create(ctx, aggregates.CreateOpts{Name: "rack1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Get(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, 7, aggregates.UpdateOpts{Name: "rack9"}, aggregates.WithUpdateField("x_extension", true)); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, 7, aggregates.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.AddHost(ctx, 7, aggregates.AddHostOpts{Host: "cmp2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.RemoveHost(ctx, 7, aggregates.RemoveHostOpts{Host: "cmp2"}); err != nil {
		t.Fatal(err)
	}
	// A nil metadata value asks Nova to remove that key.
	if _, err := api.SetMetadata(ctx, 7, aggregates.SetMetadataOpts{Metadata: map[string]any{"ssd": "false", "old": nil}}); err != nil {
		t.Fatal(err)
	}
	var ids []int
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if err := api.Delete(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []int{7, 8}) {
		t.Fatal(ids)
	}
	base := "/nova/v2.1/os-aggregates"
	want := []nativeAggregateCall{
		{http.MethodPost, base, "", `{"aggregate":{"availability_zone":"az1","name":"rack1","x_extension":1}}`},
		{http.MethodPost, base, "", `{"aggregate":{"name":"rack1"}}`},
		{http.MethodGet, base + "/7", "", ""},
		{http.MethodPut, base + "/7", "", `{"aggregate":{"name":"rack9","x_extension":true}}`},
		{http.MethodPut, base + "/7", "", `{"aggregate":{}}`},
		{http.MethodPost, base + "/7/action", "", `{"add_host":{"host":"cmp2"}}`},
		{http.MethodPost, base + "/7/action", "", `{"remove_host":{"host":"cmp2"}}`},
		{http.MethodPost, base + "/7/action", "", `{"set_metadata":{"metadata":{"old":null,"ssd":"false"}}}`},
		{http.MethodGet, base, "", ""},
		{http.MethodDelete, base + "/7", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeAggregateStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name string
		call func(*aggregates.API) error
	}{
		// Every call, including Delete, accepts only 200.
		{"Create", func(api *aggregates.API) error {
			_, err := api.Create(ctx, aggregates.CreateOpts{Name: "a"})
			return err
		}},
		{"Get", func(api *aggregates.API) error { _, err := api.Get(ctx, 7); return err }},
		{"Update", func(api *aggregates.API) error { _, err := api.Update(ctx, 7, aggregates.UpdateOpts{}); return err }},
		{"Delete", func(api *aggregates.API) error { return api.Delete(ctx, 7) }},
		{"AddHost", func(api *aggregates.API) error {
			_, err := api.AddHost(ctx, 7, aggregates.AddHostOpts{Host: "h"})
			return err
		}},
		{"RemoveHost", func(api *aggregates.API) error {
			_, err := api.RemoveHost(ctx, 7, aggregates.RemoveHostOpts{Host: "h"})
			return err
		}},
		{"SetMetadata", func(api *aggregates.API) error {
			_, err := api.SetMetadata(ctx, 7, aggregates.SetMetadataOpts{Metadata: map[string]any{"k": "v"}})
			return err
		}},
	} {
		for _, code := range []int{201, 202, 204, 404} {
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeAggregateCall
				api := nativeAggregateAPI(t, &calls, func(*http.Request) *http.Response { return nativeAggregateWire(code, `{}`) })
				err := call.call(api)
				nativeAggregateOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// The envelope is a pointer, so a missing or null aggregate is nil without error.
			`{}`:                       false,
			`{"aggregate":null}`:       false,
			`{"aggregate":[]}`:         true,
			`{"aggregate":{"id":"7"}}`: true,
			// Metadata values must be strings and timestamps carry no zone.
			`{"aggregate":{"metadata":{"k":1}}}`:                     true,
			`{"aggregate":{"created_at":"2026-10-01T01:02:03Z"}}`:    true,
			`{"aggregate":{"created_at":"2026-10-01T01:02:03.123"}}`: false,
			`{"aggregate":{"deleted_at":""}}`:                        false,
		} {
			var calls []nativeAggregateCall
			api := nativeAggregateAPI(t, &calls, func(*http.Request) *http.Response { return nativeAggregateWire(200, body) })
			got, err := api.Get(ctx, 7)
			switch {
			case wantErr:
				nativeAggregateOperation(t, err, "Get")
			case err != nil:
				t.Fatal(body, err)
			case !strings.Contains(body, "_at") && got != nil:
				t.Fatal(body, got)
			}
		}
	})
	t.Run("list pager status, empty page, bad row and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"aggregates":[]}`}, {200, `{}`}, {200, `{"aggregates":[{"id":"x"}]}`}, {204, ""}} {
			var calls []nativeAggregateCall
			api := nativeAggregateAPI(t, &calls, func(*http.Request) *http.Response { return nativeAggregateWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var wrapped *resource.OperationError
			var typeErr *json.UnmarshalTypeError
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}) && !errors.As(errs[0], &wrapped):
			case tc.code == 200 && !strings.Contains(tc.body, `"x"`) && len(errs) == 0:
			case tc.code == 200 && len(errs) == 1 && errors.As(errs[0], &typeErr):
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, tc.body, errs)
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeAggregateCall
		api := nativeAggregateAPI(t, &calls, func(*http.Request) *http.Response { return nativeAggregateWire(200, `{}`) })
		check := func(operation string, err error) {
			t.Helper()
			nativeAggregateOperation(t, err, operation)
		}
		_, err := api.Create(ctx, aggregates.CreateOpts{AvailabilityZone: "az1"})
		check("Create", err)
		// An omitted core key is still reserved.
		_, err = api.Create(ctx, aggregates.CreateOpts{Name: "a"}, aggregates.WithCreateField("availability_zone", "az"))
		check("Create", err)
		_, err = api.Create(ctx, aggregates.CreateOpts{Name: "a"}, nil)
		check("Create", err)
		_, err = api.Update(ctx, 7, aggregates.UpdateOpts{}, aggregates.WithUpdateField("name", "x"))
		check("Update", err)
		_, err = api.AddHost(ctx, 7, aggregates.AddHostOpts{})
		check("AddHost", err)
		_, err = api.RemoveHost(ctx, 7, aggregates.RemoveHostOpts{})
		check("RemoveHost", err)
		_, err = api.SetMetadata(ctx, 7, aggregates.SetMetadataOpts{})
		check("SetMetadata", err)
		_, err = api.AddHost(ctx, 7, aggregates.AddHostOpts{Host: "h"}, nil)
		check("AddHost", err)
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
