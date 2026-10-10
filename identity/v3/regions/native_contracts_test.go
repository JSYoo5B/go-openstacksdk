package regions_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRegionTransport func(*http.Request) (*http.Response, error)

func (transport nativeRegionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeRegionWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeRegionCall struct{ method, path, query, body string }

func nativeRegionAPI(t *testing.T, calls *[]nativeRegionCall, reply func(*http.Request) *http.Response) (*regions.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeRegionTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeRegionCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return regions.New(client), cloud
}

func nativeRegionOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "regions" {
		t.Fatal("generated regions context", err, wrapped)
	}
}

const nativeRegionRow = `{"id":"RegionOne","description":"main","parent_region_id":"Root","links":{"self":"x"},"location":"seoul"}`

func TestNativeRegionRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRegionCall
	var cloud *testcloud.Cloud
	api, cloud := nativeRegionAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeRegionWire(204, "")
		case req.Method == http.MethodPost:
			return nativeRegionWire(201, `{"region":`+nativeRegionRow+`}`)
		case req.URL.Path == "/keystone/v3/regions":
			return nativeRegionWire(200, `{"regions":[`+nativeRegionRow+`],"links":{"next":"`+cloud.Server.URL+`/other/regions?page=2"}}`)
		case req.URL.Path == "/other/regions":
			// An explicit extra object replaces the unknown-key scan.
			return nativeRegionWire(200, `{"regions":[{"id":"RegionTwo","extra":{"k":"v"},"zone":"ignored"}],"links":{"next":null}}`)
		}
		return nativeRegionWire(200, `{"region":`+nativeRegionRow+`}`)
	})
	created, err := api.Create(ctx, regions.CreateOpts{ID: "RegionOne", Description: "main", ParentRegionID: "Root", Extra: map[string]any{"location": "seoul"}}, regions.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "RegionOne" && created.ParentRegionID == "Root" && reflect.DeepEqual(created.Extra, map[string]any{"location": "seoul"})) {
		t.Fatal(created, err)
	}
	// An empty CreateOpts still sends the region envelope so Keystone generates the ID.
	if _, err := api.Create(ctx, regions.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "RegionOne")
	if err != nil || got.Description != "main" {
		t.Fatal(got, err)
	}
	empty := ""
	if _, err := api.Update(ctx, "RegionOne", regions.UpdateOpts{Description: &empty, ParentRegionID: "Root"}, regions.WithUpdateField("location", "busan")); err != nil {
		t.Fatal(err)
	}
	var rows []*regions.Region
	for value, err := range api.List(ctx, regions.WithListOptions(regions.ListOpts{ParentRegionID: "Root"}), regions.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "RegionOne" && rows[1].ID == "RegionTwo" && reflect.DeepEqual(rows[1].Extra, map[string]any{"k": "v"})) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "RegionOne"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/regions"
	want := []nativeRegionCall{
		{http.MethodPost, base, "", `{"region":{"description":"main","id":"RegionOne","location":"seoul","parent_region_id":"Root","x_extension":1}}`},
		{http.MethodPost, base, "", `{"region":{}}`},
		{http.MethodGet, base + "/RegionOne", "", ""},
		// Update is a PATCH; the description pointer sends an empty string.
		{http.MethodPatch, base + "/RegionOne", "", `{"region":{"description":"","location":"busan","parent_region_id":"Root"}}`},
		{http.MethodGet, base, "extra=1&parent_region_id=Root", ""},
		{http.MethodGet, "/other/regions", "page=2", ""},
		{http.MethodDelete, base + "/RegionOne", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeRegionStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*regions.API) error
	}{
		{"Create", []int{201}, func(api *regions.API) error { _, err := api.Create(ctx, regions.CreateOpts{ID: "r"}); return err }},
		{"Get", []int{200}, func(api *regions.API) error { _, err := api.Get(ctx, "r"); return err }},
		{"Update", []int{200}, func(api *regions.API) error { _, err := api.Update(ctx, "r", regions.UpdateOpts{}); return err }},
		{"Delete", []int{202, 204}, func(api *regions.API) error { return api.Delete(ctx, "r") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeRegionCall
				api, _ := nativeRegionAPI(t, &calls, func(*http.Request) *http.Response { return nativeRegionWire(code, `{}`) })
				err := call.call(api)
				nativeRegionOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// The pointer envelope makes {} and null decode to a nil region without an error.
			`{}`:              false,
			`{"region":null}`: false,
			`{"region":[]}`:   true,
		} {
			var calls []nativeRegionCall
			api, _ := nativeRegionAPI(t, &calls, func(*http.Request) *http.Response { return nativeRegionWire(200, body) })
			got, err := api.Get(ctx, "r")
			if wantErr {
				nativeRegionOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"regions":[]}`}, {204, ""}} {
			var calls []nativeRegionCall
			api, _ := nativeRegionAPI(t, &calls, func(*http.Request) *http.Response { return nativeRegionWire(tc.code, tc.body) })
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
		var calls []nativeRegionCall
		api, _ := nativeRegionAPI(t, &calls, func(*http.Request) *http.Response { return nativeRegionWire(201, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, regions.CreateOpts{}, regions.WithCreateField("id", "x"))
				return err
			}()},
			"create extra collision": {"Create", func() error {
				_, err := api.Create(ctx, regions.CreateOpts{Extra: map[string]any{"location": "a"}}, regions.WithCreateField("location", "b"))
				return err
			}()},
			"update core extension": {"Update", func() error {
				_, err := api.Update(ctx, "r", regions.UpdateOpts{}, regions.WithUpdateField("description", "x"))
				return err
			}()},
			"update nil option": {"Update", func() error {
				_, err := api.Update(ctx, "r", regions.UpdateOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeRegionOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeRegionOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
