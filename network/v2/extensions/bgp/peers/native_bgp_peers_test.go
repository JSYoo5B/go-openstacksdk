package peers_test

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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeBGPPeerTransport func(*http.Request) (*http.Response, error)

func (transport nativeBGPPeerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeBGPPeerWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeBGPPeerCall struct{ method, path, query, body string }

func nativeBGPPeerAPI(t *testing.T, calls *[]nativeBGPPeerCall, reply func(*http.Request) *http.Response) (*peers.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeBGPPeerTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeBGPPeerCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return peers.New(client), cloud
}

func nativeBGPPeerOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "peers" {
		t.Fatal("generated peers context", err, wrapped)
	}
}

const nativeBGPPeerRow = `{"id":"bp-1","name":"edge","auth_type":"md5","peer_ip":"192.0.2.1","remote_as":65001,"tenant_id":"p","project_id":"p"}`

func TestNativeBGPPeerRoutesBodiesListAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeBGPPeerCall
	var cloud *testcloud.Cloud
	api, cloud := nativeBGPPeerAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeBGPPeerWire(204, "")
		case req.Method == http.MethodPost:
			return nativeBGPPeerWire(201, `{"bgp_peer":`+nativeBGPPeerRow+`}`)
		case req.Method == http.MethodPut:
			return nativeBGPPeerWire(200, `{"bgp_peer":{"id":"bp-1","name":"edge-2"}}`)
		case req.URL.Path == "/neutron/v2.0/bgp-peers":
			// The single-page pager ignores both Neutron and generic next links.
			return nativeBGPPeerWire(200, `{"bgp_peers":[`+nativeBGPPeerRow+`],"bgp_peers_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}],"links":{"next":"`+cloud.Server.URL+`/never"}}`)
		}
		return nativeBGPPeerWire(200, `{"bgp_peer":`+nativeBGPPeerRow+`}`)
	})
	created, err := api.Create(ctx, peers.CreateOpts{Name: "edge", AuthType: "md5", Password: "secret", PeerIP: "192.0.2.1", RemoteAS: 65001}, peers.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "bp-1" && created.RemoteAS == 65001 && created.PeerIP == "192.0.2.1" && created.ProjectID == "p") {
		t.Fatal(created, err)
	}
	// No field is required and only password is omitted when empty.
	if _, err := api.Create(ctx, peers.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "bp-1")
	if err != nil || got.AuthType != "md5" {
		t.Fatal(got, err)
	}
	updated, err := api.Update(ctx, "bp-1", peers.UpdateOpts{Name: "edge-2"}, peers.WithUpdateField("x_extension", 1))
	if err != nil || updated.Name != "edge-2" {
		t.Fatal(updated, err)
	}
	if _, err := api.Update(ctx, "bp-1", peers.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if err := api.Delete(ctx, "bp-1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"bp-1"}) {
		t.Fatal(ids)
	}
	base := "/neutron/v2.0/bgp-peers"
	want := []nativeBGPPeerCall{
		{http.MethodPost, base, "", `{"bgp_peer":{"auth_type":"md5","name":"edge","password":"secret","peer_ip":"192.0.2.1","remote_as":65001,"x_extension":1}}`},
		{http.MethodPost, base, "", `{"bgp_peer":{"auth_type":"","name":"","peer_ip":"","remote_as":0}}`},
		{http.MethodGet, base + "/bp-1", "", ""},
		{http.MethodPut, base + "/bp-1", "", `{"bgp_peer":{"name":"edge-2","x_extension":1}}`},
		{http.MethodPut, base + "/bp-1", "", `{"bgp_peer":{}}`},
		{http.MethodGet, base, "", ""},
		{http.MethodDelete, base + "/bp-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeBGPPeerStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*peers.API) error
	}{
		{"Create", []int{201, 202}, func(api *peers.API) error { _, err := api.Create(ctx, peers.CreateOpts{Name: "edge"}); return err }},
		{"Get", []int{200}, func(api *peers.API) error { _, err := api.Get(ctx, "bp-1"); return err }},
		// Update narrows the PUT default to 200 only.
		{"Update", []int{200}, func(api *peers.API) error { _, err := api.Update(ctx, "bp-1", peers.UpdateOpts{Name: "x"}); return err }},
		{"Delete", []int{202, 204}, func(api *peers.API) error { return api.Delete(ctx, "bp-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeBGPPeerCall
				api, _ := nativeBGPPeerAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPPeerWire(code, `{"bgp_peer":{}}`) })
				err := call.call(api)
				nativeBGPPeerOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:                                 false,
			`{"bgp_peer":null}`:                  false,
			`{"other":{"id":"bp-1"}}`:            true,
			`{"bgp_peer":[]}`:                    true,
			`{"bgp_peer":{"remote_as":"65001"}}`: true,
		} {
			var calls []nativeBGPPeerCall
			api, _ := nativeBGPPeerAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPPeerWire(200, body) })
			got, err := api.Get(ctx, "bp-1")
			if wantErr {
				nativeBGPPeerOperation(t, err, "Get")
			} else if err != nil || got == nil || got.ID != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"bgp_peers":[]}`}, {204, ""}} {
			var calls []nativeBGPPeerCall
			api, _ := nativeBGPPeerAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPPeerWire(tc.code, tc.body) })
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
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeBGPPeerCall
		api, _ := nativeBGPPeerAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPPeerWire(201, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create extension": {"Create", func() error {
				_, err := api.Create(ctx, peers.CreateOpts{Name: "edge"}, peers.WithCreateField("peer_ip", "x"))
				return err
			}()},
			"create nil option": {"Create", func() error { _, err := api.Create(ctx, peers.CreateOpts{}, nil); return err }()},
			"update extension": {"Update", func() error {
				_, err := api.Update(ctx, "bp-1", peers.UpdateOpts{}, peers.WithUpdateField("password", "x"))
				return err
			}()},
			"update nil option": {"Update", func() error { _, err := api.Update(ctx, "bp-1", peers.UpdateOpts{}, nil); return err }()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeBGPPeerOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
