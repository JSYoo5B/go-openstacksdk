package tapmirrors_test

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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTapTransport func(*http.Request) (*http.Response, error)

func (transport nativeTapTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeTapWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeTapCall struct{ method, path, query, body string }

func nativeTapAPI(t *testing.T, calls *[]nativeTapCall, reply func(*http.Request) *http.Response) (*tapmirrors.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTapTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTapCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return tapmirrors.New(client), cloud
}

func nativeTapOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "tapmirrors" {
		t.Fatal("generated tapmirrors context", err, wrapped)
	}
}

func nativeTapStatuses(t *testing.T, name string, accepted []int, envelope string, call func(*tapmirrors.API) error) {
	t.Helper()
	for _, code := range []int{200, 201, 202, 204, 404} {
		if slices.Contains(accepted, code) {
			continue
		}
		t.Run(fmt.Sprintf("%s/%d", name, code), func(t *testing.T) {
			var calls []nativeTapCall
			api, _ := nativeTapAPI(t, &calls, func(*http.Request) *http.Response { return nativeTapWire(code, `{"`+envelope+`":{}}`) })
			err := call(api)
			nativeTapOperation(t, err, name)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, accepted) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
}

func nativeTapListStatuses(t *testing.T, collection string, list func(*tapmirrors.API) []error) {
	t.Helper()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"` + collection + `":[]}`}, {204, ""}} {
		var calls []nativeTapCall
		api, _ := nativeTapAPI(t, &calls, func(*http.Request) *http.Response { return nativeTapWire(tc.code, tc.body) })
		errs := list(api)
		var native gophercloud.ErrUnexpectedResponseCode
		switch {
		case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
		case tc.code == 200 && len(errs) == 0:
		case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
		default:
			t.Fatal(tc.code, errs)
		}
	}
}

const nativeTapRow = `{"id":"tm-1","name":"mirror","port_id":"p1","mirror_type":"erspanv1","remote_ip":"192.0.2.9","directions":{"IN":"1","OUT":"2"}}`

func TestNativeTapMirrorRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTapCall
	var cloud *testcloud.Cloud
	api, cloud := nativeTapAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeTapWire(204, "")
		case req.Method == http.MethodPost:
			return nativeTapWire(201, `{"tap_mirror":`+nativeTapRow+`}`)
		case req.URL.Path == "/neutron/v2.0/taas/tap_mirrors":
			return nativeTapWire(200, `{"tap_mirrors":[`+nativeTapRow+`],"tap_mirrors_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/page?marker=x"}]}`)
		case req.URL.Path == "/other/page":
			return nativeTapWire(200, `{"tap_mirrors":[{"id":"tm-2","directions":{}}]}`)
		}
		return nativeTapWire(200, `{"tap_mirror":`+nativeTapRow+`}`)
	})
	created, err := api.Create(ctx, tapmirrors.CreateOpts{Name: "mirror", PortID: "p1", MirrorType: tapmirrors.MirrorTypeErspanv1, RemoteIP: "192.0.2.9", Directions: tapmirrors.Directions{In: 1, Out: 2}}, tapmirrors.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "tm-1" && created.Directions.In == 1 && created.Directions.Out == 2) {
		t.Fatal(created, err)
	}
	// Name, port, type, remote IP and directions have no omitempty; zero directions are {}.
	if _, err := api.Create(ctx, tapmirrors.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	if got, err := api.Get(ctx, "tm-1"); err != nil || got.RemoteIP != "192.0.2.9" {
		t.Fatal(got, err)
	}
	empty := ""
	if _, err := api.Update(ctx, "tm-1", tapmirrors.UpdateOpts{Description: &empty}, tapmirrors.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var rows []*tapmirrors.TapMirror
	for value, err := range api.List(ctx, tapmirrors.WithListOptions(tapmirrors.ListOpts{PortID: "p1", MirrorType: tapmirrors.MirrorTypeGre}), tapmirrors.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "tm-1" && rows[1].ID == "tm-2" && rows[1].Directions == (tapmirrors.Directions{})) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "tm-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 7 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[4].query)
	base := "/neutron/v2.0/taas/tap_mirrors"
	want := []nativeTapCall{
		// Direction tunnel IDs are sent as quoted strings.
		{http.MethodPost, base, "", `{"tap_mirror":{"directions":{"IN":"1","OUT":"2"},"mirror_type":"erspanv1","name":"mirror","port_id":"p1","remote_ip":"192.0.2.9","x_extension":1}}`},
		{http.MethodPost, base, "", `{"tap_mirror":{"directions":{},"mirror_type":"","name":"","port_id":"","remote_ip":""}}`},
		{http.MethodGet, base + "/tm-1", "", ""},
		{http.MethodPut, base + "/tm-1", "", `{"tap_mirror":{"description":"","x_extension":1}}`},
		{http.MethodGet, base, calls[4].query, ""},
		{http.MethodGet, "/other/page", "marker=x", ""},
		{http.MethodDelete, base + "/tm-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"port_id": {"p1"}, "mirror_type": {"gre"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeTapMirrorStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	nativeTapStatuses(t, "Create", []int{201, 202}, "tap_mirror", func(api *tapmirrors.API) error {
		_, err := api.Create(ctx, tapmirrors.CreateOpts{})
		return err
	})
	nativeTapStatuses(t, "Get", []int{200}, "tap_mirror", func(api *tapmirrors.API) error { _, err := api.Get(ctx, "tm-1"); return err })
	nativeTapStatuses(t, "Update", []int{200}, "tap_mirror", func(api *tapmirrors.API) error {
		_, err := api.Update(ctx, "tm-1", tapmirrors.UpdateOpts{})
		return err
	})
	nativeTapStatuses(t, "Delete", []int{202, 204}, "tap_mirror", func(api *tapmirrors.API) error { return api.Delete(ctx, "tm-1") })
	nativeTapListStatuses(t, "tap_mirrors", func(api *tapmirrors.API) (errs []error) {
		for _, err := range api.List(ctx) {
			errs = append(errs, err)
		}
		return errs
	})
	t.Run("unquoted direction IDs fail decode", func(t *testing.T) {
		var calls []nativeTapCall
		api, _ := nativeTapAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeTapWire(200, `{"tap_mirror":{"id":"tm-1","directions":{"IN":1}}}`)
		})
		_, err := api.Get(ctx, "tm-1")
		nativeTapOperation(t, err, "Get")
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeTapCall
		api, _ := nativeTapAPI(t, &calls, func(*http.Request) *http.Response { return nativeTapWire(201, `{}`) })
		for operation, err := range map[string]error{
			"Create": func() error {
				_, err := api.Create(ctx, tapmirrors.CreateOpts{}, tapmirrors.WithCreateField("directions", nil))
				return err
			}(),
			"Update": func() error { _, err := api.Update(ctx, "tm-1", tapmirrors.UpdateOpts{}, nil); return err }(),
		} {
			if err == nil {
				t.Fatal(operation, "accepted")
			}
			nativeTapOperation(t, err, operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeTapOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
