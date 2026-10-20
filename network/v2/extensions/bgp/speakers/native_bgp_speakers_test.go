package speakers_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeBGPSpeakerTransport func(*http.Request) (*http.Response, error)

func (transport nativeBGPSpeakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeBGPSpeakerWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeBGPSpeakerCall struct{ method, path, query, body string }

func nativeBGPSpeakerAPI(t *testing.T, calls *[]nativeBGPSpeakerCall, reply func(*http.Request) *http.Response) (*speakers.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeBGPSpeakerTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeBGPSpeakerCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return speakers.New(client), cloud
}

func nativeBGPSpeakerOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "speakers" {
		t.Fatal("generated speakers context", err, wrapped)
	}
}

const nativeBGPSpeakerRow = `{"id":"bs-1","name":"spk","ip_version":4,"local_as":65000,"advertise_floating_ip_host_routes":true,"advertise_tenant_networks":true,"networks":["n-1"],"peers":["bp-1"],"tenant_id":"p","project_id":"p"}`

func TestNativeBGPSpeakerRoutesBodiesActionsAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeBGPSpeakerCall
	var cloud *testcloud.Cloud
	api, cloud := nativeBGPSpeakerAPI(t, &calls, func(req *http.Request) *http.Response {
		path := req.URL.Path
		switch {
		case req.Method == http.MethodDelete:
			return nativeBGPSpeakerWire(204, "")
		case req.Method == http.MethodPost:
			return nativeBGPSpeakerWire(201, `{"bgp_speaker":`+nativeBGPSpeakerRow+`}`)
		case strings.HasSuffix(path, "/add_bgp_peer"):
			return nativeBGPSpeakerWire(200, `{"bgp_peer_id":"bp-1"}`)
		case strings.HasSuffix(path, "/add_gateway_network"):
			return nativeBGPSpeakerWire(200, `{"network_id":"n-1"}`)
		case strings.HasSuffix(path, "/remove_bgp_peer") || strings.HasSuffix(path, "/remove_gateway_network"):
			// The remove calls never read the response body.
			return nativeBGPSpeakerWire(200, `not json`)
		case req.Method == http.MethodPut:
			return nativeBGPSpeakerWire(200, `{"bgp_speaker":`+nativeBGPSpeakerRow+`}`)
		case strings.HasSuffix(path, "/get_advertised_routes"):
			return nativeBGPSpeakerWire(200, `{"advertised_routes":[{"destination":"10.0.0.0/24","next_hop":"192.0.2.10"}],"links":{"next":"`+cloud.Server.URL+`/never"}}`)
		case path == "/neutron/v2.0/bgp-speakers":
			// The single-page pager ignores both Neutron and generic next links.
			return nativeBGPSpeakerWire(200, `{"bgp_speakers":[`+nativeBGPSpeakerRow+`],"bgp_speakers_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}],"links":{"next":"`+cloud.Server.URL+`/never"}}`)
		}
		return nativeBGPSpeakerWire(200, `{"bgp_speaker":`+nativeBGPSpeakerRow+`}`)
	})
	// LocalAS is a string on input and an integer on output.
	created, err := api.Create(ctx, speakers.CreateOpts{Name: "spk", IPVersion: 4, LocalAS: "65000", AdvertiseTenantNetworks: true, Networks: []string{"n-1"}}, speakers.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "bs-1" && created.LocalAS == 65000 && created.IPVersion == 4 && reflect.DeepEqual(created.Peers, []string{"bp-1"})) {
		t.Fatal(created, err)
	}
	if _, err := api.Create(ctx, speakers.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "bs-1")
	if err != nil || !reflect.DeepEqual(got.Networks, []string{"n-1"}) {
		t.Fatal(got, err)
	}
	// The advertise flags have no omitempty, so a name-only update also sends false for both.
	if _, err := api.Update(ctx, "bs-1", speakers.UpdateOpts{Name: "spk-2"}, speakers.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	peer, err := api.AddBGPPeer(ctx, "bs-1", speakers.AddBGPPeerOpts{BGPPeerID: "bp-1"}, speakers.WithAddBGPPeerField("x_extension", 1))
	if err != nil || peer.BGPPeerID != "bp-1" {
		t.Fatal(peer, err)
	}
	if err := api.RemoveBGPPeer(ctx, "bs-1", speakers.RemoveBGPPeerOpts{BGPPeerID: "bp-1"}, speakers.WithRemoveBGPPeerField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	network, err := api.AddGatewayNetwork(ctx, "bs-1", speakers.AddGatewayNetworkOpts{NetworkID: "n-1"}, speakers.WithAddGatewayNetworkField("x_extension", 1))
	if err != nil || network.NetworkID != "n-1" {
		t.Fatal(network, err)
	}
	if err := api.RemoveGatewayNetwork(ctx, "bs-1", speakers.RemoveGatewayNetworkOpts{NetworkID: "n-1"}, speakers.WithRemoveGatewayNetworkField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	// Empty IDs are sent as empty strings; Neutron decides validity.
	if _, err := api.AddBGPPeer(ctx, "bs-1", speakers.AddBGPPeerOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveGatewayNetwork(ctx, "bs-1", speakers.RemoveGatewayNetworkOpts{}); err != nil {
		t.Fatal(err)
	}
	var routes []string
	for value, err := range api.GetAdvertisedRoutes(ctx, "bs-1") {
		if err != nil {
			t.Fatal(err)
		}
		routes = append(routes, value.Destination+"@"+value.NextHop)
	}
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if err := api.Delete(ctx, "bs-1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(routes, []string{"10.0.0.0/24@192.0.2.10"}) || !reflect.DeepEqual(ids, []string{"bs-1"}) {
		t.Fatal(routes, ids)
	}
	base := "/neutron/v2.0/bgp-speakers"
	want := []nativeBGPSpeakerCall{
		{http.MethodPost, base, "", `{"bgp_speaker":{"advertise_floating_ip_host_routes":false,"advertise_tenant_networks":true,"ip_version":4,"local_as":"65000","name":"spk","networks":["n-1"],"x_extension":1}}`},
		{http.MethodPost, base, "", `{"bgp_speaker":{"advertise_floating_ip_host_routes":false,"advertise_tenant_networks":false,"ip_version":0,"local_as":"","name":""}}`},
		{http.MethodGet, base + "/bs-1", "", ""},
		{http.MethodPut, base + "/bs-1", "", `{"bgp_speaker":{"advertise_floating_ip_host_routes":false,"advertise_tenant_networks":false,"name":"spk-2","x_extension":1}}`},
		// Speaker actions send envelope-free bodies, so extensions sit at the top level.
		{http.MethodPut, base + "/bs-1/add_bgp_peer", "", `{"bgp_peer_id":"bp-1","x_extension":1}`},
		{http.MethodPut, base + "/bs-1/remove_bgp_peer", "", `{"bgp_peer_id":"bp-1","x_extension":1}`},
		{http.MethodPut, base + "/bs-1/add_gateway_network", "", `{"network_id":"n-1","x_extension":1}`},
		{http.MethodPut, base + "/bs-1/remove_gateway_network", "", `{"network_id":"n-1","x_extension":1}`},
		{http.MethodPut, base + "/bs-1/add_bgp_peer", "", `{"bgp_peer_id":""}`},
		{http.MethodPut, base + "/bs-1/remove_gateway_network", "", `{"network_id":""}`},
		{http.MethodGet, base + "/bs-1/get_advertised_routes", "", ""},
		{http.MethodGet, base, "", ""},
		{http.MethodDelete, base + "/bs-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeBGPSpeakerStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*speakers.API) error
	}{
		{"Create", []int{201, 202}, func(api *speakers.API) error { _, err := api.Create(ctx, speakers.CreateOpts{Name: "spk"}); return err }},
		{"Get", []int{200}, func(api *speakers.API) error { _, err := api.Get(ctx, "bs-1"); return err }},
		// Update and every speaker action narrow PUT to 200 only.
		{"Update", []int{200}, func(api *speakers.API) error {
			_, err := api.Update(ctx, "bs-1", speakers.UpdateOpts{Name: "x"})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *speakers.API) error { return api.Delete(ctx, "bs-1") }},
		{"AddBGPPeer", []int{200}, func(api *speakers.API) error {
			_, err := api.AddBGPPeer(ctx, "bs-1", speakers.AddBGPPeerOpts{BGPPeerID: "bp-1"})
			return err
		}},
		{"RemoveBGPPeer", []int{200}, func(api *speakers.API) error {
			return api.RemoveBGPPeer(ctx, "bs-1", speakers.RemoveBGPPeerOpts{BGPPeerID: "bp-1"})
		}},
		{"AddGatewayNetwork", []int{200}, func(api *speakers.API) error {
			_, err := api.AddGatewayNetwork(ctx, "bs-1", speakers.AddGatewayNetworkOpts{NetworkID: "n-1"})
			return err
		}},
		{"RemoveGatewayNetwork", []int{200}, func(api *speakers.API) error {
			return api.RemoveGatewayNetwork(ctx, "bs-1", speakers.RemoveGatewayNetworkOpts{NetworkID: "n-1"})
		}},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeBGPSpeakerCall
				api, _ := nativeBGPSpeakerAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPSpeakerWire(code, `{}`) })
				err := call.call(api)
				nativeBGPSpeakerOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:                                   false,
			`{"bgp_speaker":null}`:                 false,
			`{"other":{"id":"bs-1"}}`:              true,
			`{"bgp_speaker":[]}`:                   true,
			`{"bgp_speaker":{"local_as":"65000"}}`: true,
		} {
			var calls []nativeBGPSpeakerCall
			api, _ := nativeBGPSpeakerAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPSpeakerWire(200, body) })
			got, err := api.Get(ctx, "bs-1")
			if wantErr {
				nativeBGPSpeakerOperation(t, err, "Get")
			} else if err != nil || got == nil || got.ID != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("action response decode", func(t *testing.T) {
		for _, tc := range []struct {
			body    string
			wantErr bool
		}{
			// The action responses have no envelope; an enveloped body silently decodes to an empty ID.
			{`{"bgp_peer_id":"bp-1"}`, false},
			{`{"bgp_peer":{"bgp_peer_id":"bp-1"}}`, false},
			{`{}`, false},
			{`[]`, true},
			{``, true},
		} {
			var calls []nativeBGPSpeakerCall
			api, _ := nativeBGPSpeakerAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPSpeakerWire(200, tc.body) })
			peer, err := api.AddBGPPeer(ctx, "bs-1", speakers.AddBGPPeerOpts{BGPPeerID: "bp-1"})
			network, netErr := api.AddGatewayNetwork(ctx, "bs-1", speakers.AddGatewayNetworkOpts{NetworkID: "n-1"})
			if tc.wantErr {
				nativeBGPSpeakerOperation(t, err, "AddBGPPeer")
				nativeBGPSpeakerOperation(t, netErr, "AddGatewayNetwork")
				continue
			}
			wantPeer := ""
			if tc.body == `{"bgp_peer_id":"bp-1"}` {
				wantPeer = "bp-1"
			}
			if err != nil || netErr != nil || peer.BGPPeerID != wantPeer || network.NetworkID != "" {
				t.Fatal(tc.body, peer, err, network, netErr)
			}
		}
	})
	t.Run("list pagers status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"bgp_speakers":[],"advertised_routes":[]}`}, {204, ""}} {
			var calls []nativeBGPSpeakerCall
			api, _ := nativeBGPSpeakerAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPSpeakerWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			for _, err := range api.GetAdvertisedRoutes(ctx, "bs-1") {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 2 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}) && errors.As(errs[1], &native):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 2 && errors.Is(errs[0], io.EOF) && errors.Is(errs[1], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
			if len(calls) != 2 {
				t.Fatal(calls)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeBGPSpeakerCall
		api, _ := nativeBGPSpeakerAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPSpeakerWire(200, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create extension": {"Create", func() error {
				_, err := api.Create(ctx, speakers.CreateOpts{}, speakers.WithCreateField("local_as", 1))
				return err
			}()},
			"create nil option": {"Create", func() error { _, err := api.Create(ctx, speakers.CreateOpts{}, nil); return err }()},
			"update extension": {"Update", func() error {
				_, err := api.Update(ctx, "bs-1", speakers.UpdateOpts{}, speakers.WithUpdateField("advertise_tenant_networks", true))
				return err
			}()},
			"add peer extension": {"AddBGPPeer", func() error {
				_, err := api.AddBGPPeer(ctx, "bs-1", speakers.AddBGPPeerOpts{}, speakers.WithAddBGPPeerField("bgp_peer_id", "x"))
				return err
			}()},
			"remove peer extension": {"RemoveBGPPeer", api.RemoveBGPPeer(ctx, "bs-1", speakers.RemoveBGPPeerOpts{}, speakers.WithRemoveBGPPeerField("bgp_peer_id", "x"))},
			"add network nil option": {"AddGatewayNetwork", func() error {
				_, err := api.AddGatewayNetwork(ctx, "bs-1", speakers.AddGatewayNetworkOpts{}, nil)
				return err
			}()},
			"remove network extension": {"RemoveGatewayNetwork", api.RemoveGatewayNetwork(ctx, "bs-1", speakers.RemoveGatewayNetworkOpts{}, speakers.WithRemoveGatewayNetworkField("network_id", "x"))},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeBGPSpeakerOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
