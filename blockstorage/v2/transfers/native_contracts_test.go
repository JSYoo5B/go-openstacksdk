package transfers_test

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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTransferTransport func(*http.Request) (*http.Response, error)

func (transport nativeTransferTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeTransferWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeTransferCall struct{ method, path, query, body string }

func nativeTransferAPI(t *testing.T, calls *[]nativeTransferCall, reply func(*http.Request) *http.Response) (*transfers.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTransferTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTransferCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return transfers.New(client), cloud
}

func nativeTransferOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "transfers" {
		t.Fatal("generated transfers context", err, wrapped)
	}
}

const nativeTransferRow = `{"id":"tr-1","name":"handoff","volume_id":"vol-1","auth_key":"secret","links":[{"rel":"self","href":"x"}],"created_at":"2026-10-11T01:02:03.000000"}`

func TestNativeTransferRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTransferCall
	var cloud *testcloud.Cloud
	api, cloud := nativeTransferAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeTransferWire(202, "")
		case req.Method == http.MethodPost:
			return nativeTransferWire(202, `{"transfer":`+nativeTransferRow+`}`)
		case req.URL.Path == "/cinder/v2/project/os-volume-transfer/detail":
			return nativeTransferWire(200, `{"transfers":[`+nativeTransferRow+`],"transfers_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/transfers?marker=x"}]}`)
		case req.URL.Path == "/other/transfers":
			return nativeTransferWire(200, `{"transfers":[{"id":"tr-2"}]}`)
		}
		return nativeTransferWire(200, `{"transfer":`+nativeTransferRow+`}`)
	})
	created, err := api.Create(ctx, transfers.CreateOpts{VolumeID: "vol-1", Name: "handoff"}, transfers.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "tr-1" && created.AuthKey == "secret" && created.Links[0]["rel"] == "self" && created.CreatedAt.Second() == 3) {
		t.Fatal(created, err)
	}
	accepted, err := api.Accept(ctx, "tr-1", transfers.AcceptOpts{AuthKey: "secret"}, transfers.WithAcceptField("x_extension", 1))
	if err != nil || accepted.VolumeID != "vol-1" {
		t.Fatal(accepted, err)
	}
	got, err := api.Get(ctx, "tr-1")
	if err != nil || got.Name != "handoff" {
		t.Fatal(got, err)
	}
	var ids []string
	for value, err := range api.List(ctx, transfers.WithListOptions(transfers.ListOpts{AllTenants: true, Limit: 1}), transfers.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"tr-1", "tr-2"}) {
		t.Fatal(ids)
	}
	if err := api.Delete(ctx, "tr-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 6 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[3].query)
	base := "/cinder/v2/project/os-volume-transfer"
	want := []nativeTransferCall{
		{http.MethodPost, base, "", `{"transfer":{"name":"handoff","volume_id":"vol-1","x_extension":1}}`},
		{http.MethodPost, base + "/tr-1/accept", "", `{"accept":{"auth_key":"secret","x_extension":1}}`},
		{http.MethodGet, base + "/tr-1", "", ""},
		// The list uses the detail route.
		{http.MethodGet, base + "/detail", calls[3].query, ""},
		{http.MethodGet, "/other/transfers", "marker=x", ""},
		{http.MethodDelete, base + "/tr-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"all_tenants": {"true"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeTransferStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*transfers.API) error
	}{
		// Create and Accept accept only 202.
		{"Create", []int{202}, func(api *transfers.API) error {
			_, err := api.Create(ctx, transfers.CreateOpts{VolumeID: "vol-1"})
			return err
		}},
		{"Accept", []int{202}, func(api *transfers.API) error {
			_, err := api.Accept(ctx, "tr-1", transfers.AcceptOpts{AuthKey: "secret"})
			return err
		}},
		{"Get", []int{200}, func(api *transfers.API) error { _, err := api.Get(ctx, "tr-1"); return err }},
		{"Delete", []int{202, 204}, func(api *transfers.API) error { return api.Delete(ctx, "tr-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeTransferCall
				api, _ := nativeTransferAPI(t, &calls, func(*http.Request) *http.Response { return nativeTransferWire(code, `{"transfer":{}}`) })
				err := call.call(api)
				nativeTransferOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope and timestamp decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:                false,
			`{"transfer":null}`: false,
			`{"other":{}}`:      true,
			`{"transfer":{"created_at":"2026-10-11T01:02:03Z"}}`: true,
		} {
			var calls []nativeTransferCall
			api, _ := nativeTransferAPI(t, &calls, func(*http.Request) *http.Response { return nativeTransferWire(200, body) })
			got, err := api.Get(ctx, "tr-1")
			if wantErr {
				nativeTransferOperation(t, err, "Get")
			} else if err != nil || got == nil || got.ID != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"transfers":[]}`}, {204, ""}} {
			var calls []nativeTransferCall
			api, _ := nativeTransferAPI(t, &calls, func(*http.Request) *http.Response { return nativeTransferWire(tc.code, tc.body) })
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
		var calls []nativeTransferCall
		api, _ := nativeTransferAPI(t, &calls, func(*http.Request) *http.Response { return nativeTransferWire(202, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"volume id": {"Create", func() error { _, err := api.Create(ctx, transfers.CreateOpts{}); return err }()},
			"auth key":  {"Accept", func() error { _, err := api.Accept(ctx, "tr-1", transfers.AcceptOpts{}); return err }()},
			"core extension": {"Create", func() error {
				_, err := api.Create(ctx, transfers.CreateOpts{VolumeID: "vol-1"}, transfers.WithCreateField("name", "x"))
				return err
			}()},
			"accept nil option": {"Accept", func() error {
				_, err := api.Accept(ctx, "tr-1", transfers.AcceptOpts{AuthKey: "secret"}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeTransferOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeTransferOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
