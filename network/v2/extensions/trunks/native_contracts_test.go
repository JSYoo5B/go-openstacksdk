package trunks_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTrunkTransport func(*http.Request) (*http.Response, error)

func (transport nativeTrunkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeTrunkWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeTrunkCall struct{ method, path, query, body string }

func nativeTrunkAPI(t *testing.T, calls *[]nativeTrunkCall, reply func(*http.Request) *http.Response) (*trunks.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTrunkTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTrunkCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return trunks.New(client), cloud
}

func nativeTrunkOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "trunks" {
		t.Fatal("generated trunks context", err, wrapped)
	}
}

const nativeTrunkRow = `{"id":"tr-1","name":"trunk","port_id":"parent","status":"ACTIVE","admin_state_up":true,"revision_number":2,"sub_ports":[{"port_id":"child","segmentation_type":"vlan","segmentation_id":100}],"created_at":"2026-10-11T01:02:03Z"}`

func TestNativeTrunkRoutesBodiesSinglePageAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTrunkCall
	var cloud *testcloud.Cloud
	api, cloud := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeTrunkWire(204, "")
		case req.Method == http.MethodPost:
			return nativeTrunkWire(201, `{"trunk":`+nativeTrunkRow+`}`)
		case req.URL.Path == "/neutron/v2.0/trunks":
			return nativeTrunkWire(200, `{"trunks":[`+nativeTrunkRow+`],"trunks_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/trunks?marker=x"}]}`)
		}
		return nativeTrunkWire(200, `{"trunk":`+nativeTrunkRow+`}`)
	})
	created, err := api.Create(ctx, trunks.CreateOpts{PortID: "parent", Name: "trunk", AdminStateUp: gophercloud.Disabled, Subports: []trunks.Subport{{PortID: "child", SegmentationType: "vlan", SegmentationID: 100}}}, trunks.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "tr-1" && created.Subports[0].SegmentationID == 100 && created.AdminStateUp && created.CreatedAt.Year() == 2026) {
		t.Fatal(created, err)
	}
	// A nil subport list is sent as [] rather than null.
	if _, err := api.Create(ctx, trunks.CreateOpts{PortID: "parent"}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "tr-1")
	if err != nil || got.Status != trunks.StatusActive || got.RevisionNumber != 2 {
		t.Fatal(got, err)
	}
	var rows []*trunks.Trunk
	for value, err := range api.List(ctx, trunks.WithListOptions(trunks.ListOpts{Name: "trunk", AdminStateUp: gophercloud.Enabled, RevisionNumber: "0"}), trunks.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	// TrunkPage has no NextPageURL for trunks_links, so only the first page is read.
	if !(len(rows) == 1 && rows[0].ID == "tr-1") {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "tr-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 5 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[3].query)
	want := []nativeTrunkCall{
		{http.MethodPost, "/neutron/v2.0/trunks", "", `{"trunk":{"admin_state_up":false,"name":"trunk","port_id":"parent","sub_ports":[{"port_id":"child","segmentation_id":100,"segmentation_type":"vlan"}],"x_extension":1}}`},
		{http.MethodPost, "/neutron/v2.0/trunks", "", `{"trunk":{"port_id":"parent","sub_ports":[]}}`},
		{http.MethodGet, "/neutron/v2.0/trunks/tr-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/trunks", calls[3].query, ""},
		{http.MethodDelete, "/neutron/v2.0/trunks/tr-1", "", ""},
	}
	// The string revision filter keeps an explicit "0".
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"trunk"}, "admin_state_up": {"true"}, "revision_number": {"0"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeTrunkSubportOperations(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTrunkCall
	api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/get_subports") {
			return nativeTrunkWire(200, `{"sub_ports":[{"port_id":"child","segmentation_type":"inherit","segmentation_id":0}]}`)
		}
		// Subport actions return the trunk object without an envelope.
		return nativeTrunkWire(200, nativeTrunkRow)
	})
	subports, err := api.GetSubports(ctx, "tr-1")
	if err != nil || !reflect.DeepEqual(subports, []trunks.Subport{{PortID: "child", SegmentationType: "inherit"}}) {
		t.Fatal(subports, err)
	}
	added, err := api.AddSubports(ctx, "tr-1", trunks.AddSubportsOpts{Subports: []trunks.Subport{{PortID: "child", SegmentationType: "inherit"}}}, trunks.WithAddSubportsField("x_extension", 1))
	if err != nil || added.ID != "tr-1" || added.Subports[0].PortID != "child" {
		t.Fatal(added, err)
	}
	if _, err := api.AddSubports(ctx, "tr-1", trunks.AddSubportsOpts{Subports: []trunks.Subport{}}); err != nil {
		t.Fatal(err)
	}
	removed, err := api.RemoveSubports(ctx, "tr-1", trunks.RemoveSubportsOpts{Subports: []trunks.RemoveSubport{{PortID: "child"}}})
	if err != nil || removed.ID != "tr-1" {
		t.Fatal(removed, err)
	}
	if _, err := api.RemoveSubports(ctx, "tr-1", trunks.RemoveSubportsOpts{}); err != nil {
		t.Fatal(err)
	}
	want := []nativeTrunkCall{
		{http.MethodGet, "/neutron/v2.0/trunks/tr-1/get_subports", "", ""},
		// Segmentation ID zero passes the native required check and is sent.
		{http.MethodPut, "/neutron/v2.0/trunks/tr-1/add_subports", "", `{"sub_ports":[{"port_id":"child","segmentation_id":0,"segmentation_type":"inherit"}],"x_extension":1}`},
		{http.MethodPut, "/neutron/v2.0/trunks/tr-1/add_subports", "", `{"sub_ports":[]}`},
		{http.MethodPut, "/neutron/v2.0/trunks/tr-1/remove_subports", "", `{"sub_ports":[{"port_id":"child"}]}`},
		{http.MethodPut, "/neutron/v2.0/trunks/tr-1/remove_subports", "", `{"sub_ports":null}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	for name, check := range map[string]struct {
		operation string
		err       error
	}{
		"nil add list": {"AddSubports", func() error { _, err := api.AddSubports(ctx, "tr-1", trunks.AddSubportsOpts{}); return err }()},
		"add segmentation type": {"AddSubports", func() error {
			_, err := api.AddSubports(ctx, "tr-1", trunks.AddSubportsOpts{Subports: []trunks.Subport{{PortID: "child", SegmentationID: 5}}})
			return err
		}()},
		"remove port": {"RemoveSubports", func() error {
			_, err := api.RemoveSubports(ctx, "tr-1", trunks.RemoveSubportsOpts{Subports: []trunks.RemoveSubport{{}}})
			return err
		}()},
		"add extension": {"AddSubports", func() error {
			_, err := api.AddSubports(ctx, "tr-1", trunks.AddSubportsOpts{Subports: []trunks.Subport{}}, trunks.WithAddSubportsField("sub_ports", nil))
			return err
		}()},
		"remove nil option": {"RemoveSubports", func() error {
			_, err := api.RemoveSubports(ctx, "tr-1", trunks.RemoveSubportsOpts{}, nil)
			return err
		}()},
	} {
		if check.err == nil {
			t.Fatal(name, "accepted")
		}
		nativeTrunkOperation(t, check.err, check.operation)
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestNativeTrunkStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	add := trunks.AddSubportsOpts{Subports: []trunks.Subport{}}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*trunks.API) error
	}{
		{"Create", []int{201, 202}, func(api *trunks.API) error {
			_, err := api.Create(ctx, trunks.CreateOpts{PortID: "parent"})
			return err
		}},
		{"Get", []int{200}, func(api *trunks.API) error { _, err := api.Get(ctx, "tr-1"); return err }},
		{"Delete", []int{202, 204}, func(api *trunks.API) error { return api.Delete(ctx, "tr-1") }},
		{"GetSubports", []int{200}, func(api *trunks.API) error { _, err := api.GetSubports(ctx, "tr-1"); return err }},
		{"AddSubports", []int{200}, func(api *trunks.API) error { _, err := api.AddSubports(ctx, "tr-1", add); return err }},
		{"RemoveSubports", []int{200}, func(api *trunks.API) error {
			_, err := api.RemoveSubports(ctx, "tr-1", trunks.RemoveSubportsOpts{})
			return err
		}},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeTrunkCall
				api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response { return nativeTrunkWire(code, `{"trunk":{}}`) })
				err := call.call(api)
				nativeTrunkOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("timestamps accept RFC3339 only", func(t *testing.T) {
		var calls []nativeTrunkCall
		api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeTrunkWire(200, `{"trunk":{"id":"tr-1","created_at":"2026-10-11T01:02:03"}}`)
		})
		_, err := api.Get(ctx, "tr-1")
		nativeTrunkOperation(t, err, "Get")
	})
	t.Run("list follows only a links.next string", func(t *testing.T) {
		var calls []nativeTrunkCall
		var cloud *testcloud.Cloud
		api, cloud := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.Path == "/other/trunks" {
				return nativeTrunkWire(200, `{"trunks":[{"id":"tr-2"}]}`)
			}
			return nativeTrunkWire(200, `{"trunks":[{"id":"tr-1"}],"links":{"next":"`+cloud.Server.URL+`/other/trunks"}}`)
		})
		var ids []string
		for value, err := range api.List(ctx) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
		}
		if !reflect.DeepEqual(ids, []string{"tr-1", "tr-2"}) || len(calls) != 2 {
			t.Fatal(ids, calls)
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"trunks":[]}`}, {204, ""}} {
			var calls []nativeTrunkCall
			api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response { return nativeTrunkWire(tc.code, tc.body) })
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
		var calls []nativeTrunkCall
		api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response { return nativeTrunkWire(201, `{}`) })
		for name, check := range map[string]func() error{
			"parent port": func() error { _, err := api.Create(ctx, trunks.CreateOpts{}); return err },
			"subport type": func() error {
				_, err := api.Create(ctx, trunks.CreateOpts{PortID: "parent", Subports: []trunks.Subport{{PortID: "child"}}})
				return err
			},
			"sub_ports extension": func() error {
				_, err := api.Create(ctx, trunks.CreateOpts{PortID: "parent"}, trunks.WithCreateField("sub_ports", nil))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, trunks.CreateOpts{PortID: "parent"}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeTrunkOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeTrunkOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
