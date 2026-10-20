package segments_test

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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeSegmentTransport func(*http.Request) (*http.Response, error)

func (transport nativeSegmentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeSegmentWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeSegmentCall struct{ method, path, query, body string }

func nativeSegmentAPI(t *testing.T, calls *[]nativeSegmentCall, reply func(*http.Request) *http.Response) (*segments.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeSegmentTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeSegmentCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return segments.New(client), cloud
}

func nativeSegmentOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "segments" {
		t.Fatal("generated segments context", err, wrapped)
	}
}

const nativeSegmentRow = `{"id":"seg-1","name":"rack-1","network_id":"net-1","network_type":"vlan","physical_network":"physnet1","segmentation_id":2016,"revision_number":3,"created_at":"2026-10-10T01:02:03Z","updated_at":"2026-10-11T04:05:06Z"}`

var nativeSegmentCreate = segments.CreateOpts{NetworkID: "net-1", NetworkType: "vlan", PhysicalNetwork: "physnet1", SegmentationID: 2016}

func TestNativeSegmentRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeSegmentCall
	var cloud *testcloud.Cloud
	api, cloud := nativeSegmentAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeSegmentWire(204, "")
		case req.Method == http.MethodPost:
			return nativeSegmentWire(201, `{"segment":`+nativeSegmentRow+`}`)
		case req.URL.Path == "/neutron/v2.0/segments":
			// SegmentPage has no NextPageURL override: segments_links is ignored and only links.next is followed.
			return nativeSegmentWire(200, `{"segments":[`+nativeSegmentRow+`],"segments_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}],"links":{"next":"`+cloud.Server.URL+`/other/segments"}}`)
		case req.URL.Path == "/other/segments":
			return nativeSegmentWire(200, `{"segments":[{"id":"seg-2","network_type":"flat","segmentation_id":null}]}`)
		}
		return nativeSegmentWire(200, `{"segment":`+nativeSegmentRow+`}`)
	})
	created, err := api.Create(ctx, nativeSegmentCreate, segments.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "seg-1" && created.SegmentationID == 2016 && created.RevisionNumber == 3 &&
		created.CreatedAt.Equal(time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC))) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "seg-1")
	if err != nil || got.PhysicalNetwork != "physnet1" || !got.UpdatedAt.Equal(time.Date(2026, 10, 11, 4, 5, 6, 0, time.UTC)) {
		t.Fatal(got, err)
	}
	name, zero := "", 0
	if _, err := api.Update(ctx, "seg-1", segments.UpdateOpts{Name: &name, SegmentationID: &zero}, segments.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var rows []*segments.Segment
	for value, err := range api.List(ctx, segments.WithListOptions(segments.ListOpts{NetworkID: "net-1", NetworkType: "vlan", SegmentationID: 0, SortKey: "name"}), segments.WithListQuery("limit", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "seg-1" && rows[1].ID == "seg-2" && rows[1].SegmentationID == 0) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "seg-1"); err != nil {
		t.Fatal(err)
	}
	query, _ := url.ParseQuery(calls[3].query)
	want := []nativeSegmentCall{
		{http.MethodPost, "/neutron/v2.0/segments", "", `{"segment":{"network_id":"net-1","network_type":"vlan","physical_network":"physnet1","segmentation_id":2016,"x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/segments/seg-1", "", ""},
		{http.MethodPut, "/neutron/v2.0/segments/seg-1", "", `{"segment":{"name":"","segmentation_id":0,"x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/segments", calls[3].query, ""},
		{http.MethodGet, "/other/segments", "", ""},
		{http.MethodDelete, "/neutron/v2.0/segments/seg-1", "", ""},
	}
	// The integer segmentation_id filter omits 0.
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"network_id": {"net-1"}, "network_type": {"vlan"}, "sort_key": {"name"}, "limit": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeSegmentStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*segments.API) error
	}{
		{"Create", []int{201, 202}, func(api *segments.API) error { _, err := api.Create(ctx, nativeSegmentCreate); return err }},
		{"Get", []int{200}, func(api *segments.API) error { _, err := api.Get(ctx, "seg-1"); return err }},
		{"Update", []int{200}, func(api *segments.API) error {
			_, err := api.Update(ctx, "seg-1", segments.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *segments.API) error { return api.Delete(ctx, "seg-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeSegmentCall
				api, _ := nativeSegmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeSegmentWire(code, `{"segment":{}}`) })
				err := call.call(api)
				nativeSegmentOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope and timestamp decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// An empty object or null envelope yields a zero value; other keys without the envelope fail.
			`{}`:                   false,
			`{"segment":null}`:     false,
			`{"other":{"id":"x"}}`: true,
			`{"segment":[]}`:       true,
			`{"segment":{"created_at":"2026-10-10 01:02:03"}}`: true,
		} {
			var calls []nativeSegmentCall
			api, _ := nativeSegmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeSegmentWire(200, body) })
			got, err := api.Get(ctx, "seg-1")
			if wantErr {
				nativeSegmentOperation(t, err, "Get")
			} else if err != nil || got == nil || got.ID != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"segments":[]}`}, {200, `{}`}, {204, ""}} {
			var calls []nativeSegmentCall
			api, _ := nativeSegmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeSegmentWire(tc.code, tc.body) })
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
		var calls []nativeSegmentCall
		api, _ := nativeSegmentAPI(t, &calls, func(*http.Request) *http.Response { return nativeSegmentWire(201, `{}`) })
		noType := nativeSegmentCreate
		noType.NetworkType = ""
		noNetwork := nativeSegmentCreate
		noNetwork.NetworkID = ""
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create network type": {"Create", func() error { _, err := api.Create(ctx, noType); return err }()},
			"create network id":   {"Create", func() error { _, err := api.Create(ctx, noNetwork); return err }()},
			"create extension": {"Create", func() error {
				_, err := api.Create(ctx, nativeSegmentCreate, segments.WithCreateField("description", "x"))
				return err
			}()},
			"update extension": {"Update", func() error {
				_, err := api.Update(ctx, "seg-1", segments.UpdateOpts{}, segments.WithUpdateField("segmentation_id", 1))
				return err
			}()},
			"create nil option": {"Create", func() error { _, err := api.Create(ctx, nativeSegmentCreate, nil); return err }()},
			"update nil option": {"Update", func() error { _, err := api.Update(ctx, "seg-1", segments.UpdateOpts{}, nil); return err }()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeSegmentOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeSegmentOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
