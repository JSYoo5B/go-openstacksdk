package peers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonBGPPeerCall records the wire request openstacksdk would also send.
type pythonBGPPeerCall struct{ method, path, query, body string }

type pythonBGPPeerTransport func(*http.Request) (*http.Response, error)

func (transport pythonBGPPeerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonBGPPeerWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonBGPPeerAPI(t *testing.T, reply func(*http.Request) *http.Response) (*peers.API, *[]pythonBGPPeerCall, *testcloud.Cloud) {
	t.Helper()
	calls := &[]pythonBGPPeerCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonBGPPeerTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonBGPPeerCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return peers.New(client), calls, cloud
}

const (
	pythonBGPPeerBase = "/neutron/v2.0/bgp-peers"
	pythonBGPPeerRow  = `{"id":"bp-1","name":"peer","auth_type":"none","peer_ip":"192.0.2.10","remote_as":64512,"project_id":"p","tenant_id":"p"}`
)

func TestPythonBGPPeerProxyRequests(t *testing.T) {
	ctx := context.Background()
	deletes := []int{404, 404, 409, 204, 404}
	api, calls, _ := pythonBGPPeerAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			code := deletes[0]
			deletes = deletes[1:]
			return pythonBGPPeerWire(code, `{"NeutronError":{"type":"x"}}`)
		case http.MethodPost:
			return pythonBGPPeerWire(201, `{"bgp_peer":`+pythonBGPPeerRow+`}`)
		}
		return pythonBGPPeerWire(200, `{"bgp_peer":`+pythonBGPPeerRow+`}`)
	})
	// create_bgp_peer(name="peer", peer_ip="192.0.2.10", remote_as=64512, auth_type="none").
	// Go always sends name, auth_type, peer_ip and remote_as, so auth_type must be explicit.
	created, err := api.Create(ctx, peers.CreateOpts{Name: "peer", PeerIP: "192.0.2.10", RemoteAS: 64512, AuthType: "none"})
	if err != nil || created.ID != "bp-1" || created.RemoteAS != 64512 || created.PeerIP != "192.0.2.10" || created.AuthType != "none" || created.ProjectID != "p" {
		t.Fatal(created, err)
	}
	// get_bgp_peer("bp-1")
	if got, err := api.Get(ctx, "bp-1"); err != nil || got.Name != "peer" {
		t.Fatal(got, err)
	}
	// delete_bgp_peer("bp-1") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("bp-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("bp-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("bp-1")); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("bp-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "bp-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonBGPPeerCall{
		{http.MethodPost, pythonBGPPeerBase, "", `{"bgp_peer":{"auth_type":"none","name":"peer","peer_ip":"192.0.2.10","remote_as":64512}}`},
		{http.MethodGet, pythonBGPPeerBase + "/bp-1", "", ""},
	}
	for range 5 {
		want = append(want, pythonBGPPeerCall{http.MethodDelete, pythonBGPPeerBase + "/bp-1", "", ""})
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPPeerUpdateCannotClearName(t *testing.T) {
	ctx := context.Background()
	api, calls, _ := pythonBGPPeerAPI(t, func(*http.Request) *http.Response {
		return pythonBGPPeerWire(200, `{"bgp_peer":`+pythonBGPPeerRow+`}`)
	})
	// update_bgp_peer("bp-1", name="edge", password="pw")
	if got, err := api.Update(ctx, "bp-1", peers.UpdateOpts{Name: "edge", Password: "pw"}); err != nil || got.ID != "bp-1" {
		t.Fatal(got, err)
	}
	// update_bgp_peer("bp-1", name="") sends {"name": ""}; Go omits it and rejects the core key.
	if _, err := api.Update(ctx, "bp-1", peers.UpdateOpts{Name: ""}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "bp-1", peers.UpdateOpts{}, peers.WithUpdateField("name", "")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	want := []pythonBGPPeerCall{
		{http.MethodPut, pythonBGPPeerBase + "/bp-1", "", `{"bgp_peer":{"name":"edge","password":"pw"}}`},
		{http.MethodPut, pythonBGPPeerBase + "/bp-1", "", `{"bgp_peer":{}}`},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPPeerListHasNoQueryOrPaging(t *testing.T) {
	ctx := context.Background()
	var cloud *testcloud.Cloud
	api, calls, cloud := pythonBGPPeerAPI(t, func(*http.Request) *http.Response {
		return pythonBGPPeerWire(200, `{"bgp_peers":[`+pythonBGPPeerRow+`,{"id":"bp-2","name":"other","remote_as":64513}],"bgp_peers_links":[{"rel":"next","href":"`+cloud.Server.URL+pythonBGPPeerBase+`?marker=bp-2"}]}`)
	})
	// bgp_peers(): one GET; Go does not follow bgp_peers_links like Python does.
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"bp-1", "bp-2"}) {
		t.Fatal(ids)
	}
	// bgp_peers(name="other") filters locally in Python too.
	named, err := api.All(ctx, resource.WithName("other"))
	if err != nil || len(named) != 1 || named[0].ID != "bp-2" {
		t.Fatal(named, err)
	}
	// bgp_peers(limit=1) and bgp_peers(marker=...) cannot be sent.
	for _, key := range []string{"limit", "marker"} {
		if _, err := api.All(ctx, resource.WithQuery(key, "1")); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(key, err)
		}
	}
	want := []pythonBGPPeerCall{
		{http.MethodGet, pythonBGPPeerBase, "", ""},
		{http.MethodGet, pythonBGPPeerBase, "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPPeerFindChainWithoutQuery(t *testing.T) {
	ctx := context.Background()
	rows := pythonBGPPeerRow
	api, calls, _ := pythonBGPPeerAPI(t, func(req *http.Request) *http.Response {
		switch req.URL.Path {
		case pythonBGPPeerBase + "/bp-1":
			return pythonBGPPeerWire(200, `{"bgp_peer":`+pythonBGPPeerRow+`}`)
		case pythonBGPPeerBase:
			return pythonBGPPeerWire(200, `{"bgp_peers":[`+rows+`]}`)
		}
		return pythonBGPPeerWire(404, `{"NeutronError":{"type":"BgpPeerNotFound"}}`)
	})
	// find_bgp_peer("peer"): GET by ID, then the unfiltered list matched locally,
	// because BgpPeer has no name query parameter.
	if got, err := api.Find(ctx, resource.ID("peer"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("peer")); err != nil || got.ID != "bp-1" {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.ID("bp-1"), resource.WithIgnoreMissing()); err != nil || got.Name != "peer" {
		t.Fatal(got, err)
	}
	if _, err := api.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	rows = pythonBGPPeerRow + "," + strings.Replace(pythonBGPPeerRow, "bp-1", "bp-9", 1)
	if _, err := api.Find(ctx, resource.Name("peer")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	want := []pythonBGPPeerCall{
		{http.MethodGet, pythonBGPPeerBase + "/peer", "", ""},
		{http.MethodGet, pythonBGPPeerBase, "", ""},
		{http.MethodGet, pythonBGPPeerBase + "/bp-1", "", ""},
		{http.MethodGet, pythonBGPPeerBase, "", ""},
		{http.MethodGet, pythonBGPPeerBase, "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
