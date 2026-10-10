package volumetypes_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTypeTransport func(*http.Request) (*http.Response, error)

func (transport nativeTypeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeTypeWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeTypeCall struct{ method, path, query, body string }

func nativeTypeAPI(t *testing.T, calls *[]nativeTypeCall, reply func(*http.Request) *http.Response) (*volumetypes.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTypeTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTypeCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return volumetypes.New(client), cloud
}

func nativeTypeOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "volumetypes" {
		t.Fatal("generated volumetypes context", err, wrapped)
	}
}

const nativeTypeRow = `{"id":"vt-1","name":"ssd","description":null,"is_public":true,"os-volume-type-access:is_public":true,"qos_specs_id":null,"extra_specs":{"volume_backend_name":"fast"}}`

func TestNativeVolumeTypeReadRoutesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTypeCall
	var cloud *testcloud.Cloud
	api, cloud := nativeTypeAPI(t, &calls, func(req *http.Request) *http.Response {
		switch req.URL.Path {
		case "/cinder/v3/project/types":
			// The native page reads the singular volume_type_links key.
			return nativeTypeWire(200, `{"volume_types":[`+nativeTypeRow+`],"volume_type_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/types?marker=x"}]}`)
		case "/other/types":
			return nativeTypeWire(200, `{"volume_types":[{"id":"vt-2","extra_specs":null}],"volume_types_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}]}`)
		case "/cinder/v3/project/types/vt-1/extra_specs":
			return nativeTypeWire(200, `{"extra_specs":{"volume_backend_name":"fast","multiattach":"<is> True"}}`)
		case "/cinder/v3/project/types/vt-1/extra_specs/volume_backend_name":
			// A single extra spec is the bare key/value object without an envelope.
			return nativeTypeWire(200, `{"volume_backend_name":"fast"}`)
		}
		return nativeTypeWire(200, `{"volume_type":`+nativeTypeRow+`}`)
	})
	got, err := api.Get(ctx, "vt-1")
	if err != nil || !(got.Name == "ssd" && got.IsPublic && got.PublicAccess && got.Description == "" && got.ExtraSpecs["volume_backend_name"] == "fast") {
		t.Fatal(got, err)
	}
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	for _, err := range api.List(ctx, volumetypes.WithListOptions(volumetypes.ListOpts{Name: "ssd", IsPublic: volumetypes.VisibilityPrivate, Limit: 1}), volumetypes.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
	}
	specs, err := api.ListExtraSpecs(ctx, "vt-1")
	if err != nil || !reflect.DeepEqual(specs, map[string]string{"volume_backend_name": "fast", "multiattach": "<is> True"}) {
		t.Fatal(specs, err)
	}
	spec, err := api.GetExtraSpec(ctx, "vt-1", "volume_backend_name")
	if err != nil || !reflect.DeepEqual(spec, map[string]string{"volume_backend_name": "fast"}) {
		t.Fatal(spec, err)
	}
	if !reflect.DeepEqual(ids, []string{"vt-1", "vt-2"}) || len(calls) != 7 {
		t.Fatal(ids, calls)
	}
	defaultQuery, _ := url.ParseQuery(calls[1].query)
	filteredQuery, _ := url.ParseQuery(calls[3].query)
	base := "/cinder/v3/project/types"
	want := []nativeTypeCall{
		{http.MethodGet, base + "/vt-1", "", ""},
		{http.MethodGet, base, calls[1].query, ""},
		{http.MethodGet, "/other/types", "marker=x", ""},
		{http.MethodGet, base, calls[3].query, ""},
		{http.MethodGet, "/other/types", "marker=x", ""},
		{http.MethodGet, base + "/vt-1/extra_specs", "", ""},
		{http.MethodGet, base + "/vt-1/extra_specs/volume_backend_name", "", ""},
	}
	// An unset visibility is always sent as is_public=None.
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(defaultQuery, url.Values{"is_public": {"None"}}) || !reflect.DeepEqual(filteredQuery, url.Values{"name": {"ssd"}, "is_public": {"false"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v %v", calls, defaultQuery, filteredQuery)
	}
}

func TestNativeVolumeTypeReadStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name string
		call func(*volumetypes.API) error
	}{
		{"Get", func(api *volumetypes.API) error { _, err := api.Get(ctx, "vt-1"); return err }},
		{"ListExtraSpecs", func(api *volumetypes.API) error { _, err := api.ListExtraSpecs(ctx, "vt-1"); return err }},
		{"GetExtraSpec", func(api *volumetypes.API) error { _, err := api.GetExtraSpec(ctx, "vt-1", "k"); return err }},
	} {
		for _, code := range []int{201, 202, 204, 404} {
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeTypeCall
				api, _ := nativeTypeAPI(t, &calls, func(*http.Request) *http.Response { return nativeTypeWire(code, `{}`) })
				err := call.call(api)
				nativeTypeOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("decode", func(t *testing.T) {
		var calls []nativeTypeCall
		api, _ := nativeTypeAPI(t, &calls, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/k") {
				// Non-string spec values fail the string map decode.
				return nativeTypeWire(200, `{"k":1}`)
			}
			if strings.HasSuffix(req.URL.Path, "/extra_specs") {
				return nativeTypeWire(200, `{}`)
			}
			return nativeTypeWire(200, `{}`)
		})
		if got, err := api.Get(ctx, "vt-1"); err != nil || got == nil || got.ID != "" {
			t.Fatal(got, err)
		}
		if specs, err := api.ListExtraSpecs(ctx, "vt-1"); err != nil || specs != nil {
			t.Fatal(specs, err)
		}
		_, err := api.GetExtraSpec(ctx, "vt-1", "k")
		nativeTypeOperation(t, err, "GetExtraSpec")
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"volume_types":[]}`}, {204, ""}} {
			var calls []nativeTypeCall
			api, _ := nativeTypeAPI(t, &calls, func(*http.Request) *http.Response { return nativeTypeWire(tc.code, tc.body) })
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
	t.Run("list nil option", func(t *testing.T) {
		var calls []nativeTypeCall
		api, _ := nativeTypeAPI(t, &calls, func(*http.Request) *http.Response { return nativeTypeWire(200, `{}`) })
		for _, err := range api.List(ctx, nil) {
			nativeTypeOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
