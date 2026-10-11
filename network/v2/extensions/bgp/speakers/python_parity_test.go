package speakers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonBGPSpeakerCall records the wire request openstacksdk would also send.
type pythonBGPSpeakerCall struct{ method, path, query, body string }

type pythonBGPSpeakerTransport func(*http.Request) (*http.Response, error)

func (transport pythonBGPSpeakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonBGPSpeakerWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonBGPSpeakerAPI(t *testing.T, reply func(*http.Request) *http.Response) (*speakers.API, *agents.API, *[]pythonBGPSpeakerCall, *testcloud.Cloud) {
	t.Helper()
	calls := &[]pythonBGPSpeakerCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonBGPSpeakerTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonBGPSpeakerCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return speakers.New(client), agents.New(client), calls, cloud
}

const (
	pythonBGPSpeakerBase  = "/neutron/v2.0/bgp-speakers"
	pythonBGPSpeakerAgent = "/neutron/v2.0/agents"
	pythonBGPSpeakerRow   = `{"id":"bs-1","name":"spk","ip_version":4,"local_as":65000,"advertise_floating_ip_host_routes":true,"advertise_tenant_networks":true,"networks":["n-1"],"peers":["bp-1"],"project_id":"p","tenant_id":"p"}`
	pythonBGPSpeakerDRRow = `{"id":"ag-1","agent_type":"BGP dynamic routing agent","binary":"neutron-bgp-dragent","host":"net1","alive":true,"admin_state_up":true,"topic":"bgp_dragent"}`
)

func TestPythonBGPSpeakerProxyRequests(t *testing.T) {
	ctx := context.Background()
	deletes := []int{404, 404, 409, 204, 404}
	api, _, calls, _ := pythonBGPSpeakerAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			code := deletes[0]
			deletes = deletes[1:]
			return pythonBGPSpeakerWire(code, `{"NeutronError":{"type":"x"}}`)
		case http.MethodPost:
			return pythonBGPSpeakerWire(201, `{"bgp_speaker":`+pythonBGPSpeakerRow+`}`)
		}
		return pythonBGPSpeakerWire(200, `{"bgp_speaker":`+pythonBGPSpeakerRow+`}`)
	})
	// create_bgp_speaker(name="spk", ip_version=4, local_as=65000). Go always sends both
	// advertise flags, so the server defaults (true) must be passed explicitly.
	created, err := api.Create(ctx, speakers.CreateOpts{Name: "spk", IPVersion: 4, LocalAS: "65000", AdvertiseFloatingIPHostRoutes: true, AdvertiseTenantNetworks: true})
	if err != nil || created.ID != "bs-1" || created.LocalAS != 65000 || created.IPVersion != 4 || !created.AdvertiseTenantNetworks || !reflect.DeepEqual(created.Peers, []string{"bp-1"}) || !reflect.DeepEqual(created.Networks, []string{"n-1"}) {
		t.Fatal(created, err)
	}
	// get_bgp_speaker("bs-1")
	if got, err := api.Get(ctx, "bs-1"); err != nil || got.Name != "spk" {
		t.Fatal(got, err)
	}
	// delete_bgp_speaker("bs-1") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("bs-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("bs-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("bs-1")); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("bs-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "bs-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonBGPSpeakerCall{
		{http.MethodPost, pythonBGPSpeakerBase, "", `{"bgp_speaker":{"advertise_floating_ip_host_routes":true,"advertise_tenant_networks":true,"ip_version":4,"local_as":"65000","name":"spk"}}`},
		{http.MethodGet, pythonBGPSpeakerBase + "/bs-1", "", ""},
	}
	for range 5 {
		want = append(want, pythonBGPSpeakerCall{http.MethodDelete, pythonBGPSpeakerBase + "/bs-1", "", ""})
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPSpeakerUpdateAlwaysSendsAdvertiseFlags(t *testing.T) {
	ctx := context.Background()
	api, _, calls, _ := pythonBGPSpeakerAPI(t, func(*http.Request) *http.Response {
		return pythonBGPSpeakerWire(200, `{"bgp_speaker":`+pythonBGPSpeakerRow+`}`)
	})
	// update_bgp_speaker("bs-1", name="edge") sends only name; Go also resets both flags.
	if _, err := api.Update(ctx, "bs-1", speakers.UpdateOpts{Name: "edge"}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "bs-1", speakers.UpdateOpts{Name: "edge"}, speakers.WithUpdateField("advertise_tenant_networks", true)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	want := []pythonBGPSpeakerCall{
		{http.MethodPut, pythonBGPSpeakerBase + "/bs-1", "", `{"bgp_speaker":{"advertise_floating_ip_host_routes":false,"advertise_tenant_networks":false,"name":"edge"}}`},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPSpeakerListAndFindHaveNoQuery(t *testing.T) {
	ctx := context.Background()
	var cloud *testcloud.Cloud
	api, _, calls, cloud := pythonBGPSpeakerAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path == pythonBGPSpeakerBase {
			return pythonBGPSpeakerWire(200, `{"bgp_speakers":[`+pythonBGPSpeakerRow+`],"bgp_speakers_links":[{"rel":"next","href":"`+cloud.Server.URL+pythonBGPSpeakerBase+`?marker=bs-1"}]}`)
		}
		return pythonBGPSpeakerWire(404, `{"NeutronError":{"type":"BgpSpeakerNotFound"}}`)
	})
	// bgp_speakers(): one GET; Go does not follow bgp_speakers_links like Python does.
	all, err := api.All(ctx)
	if err != nil || len(all) != 1 || all[0].ID != "bs-1" {
		t.Fatal(all, err)
	}
	for _, key := range []string{"limit", "marker"} {
		if _, err := api.All(ctx, resource.WithQuery(key, "1")); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(key, err)
		}
	}
	// find_bgp_speaker("spk"): GET by ID, then the unfiltered list matched locally.
	if got, err := api.Find(ctx, resource.ID("spk"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("spk")); err != nil || got.ID != "bs-1" {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("none"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	want := []pythonBGPSpeakerCall{
		{http.MethodGet, pythonBGPSpeakerBase, "", ""},
		{http.MethodGet, pythonBGPSpeakerBase + "/spk", "", ""},
		{http.MethodGet, pythonBGPSpeakerBase, "", ""},
		{http.MethodGet, pythonBGPSpeakerBase, "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPSpeakerPeerActions(t *testing.T) {
	ctx := context.Background()
	removeStatus := http.StatusOK
	api, _, calls, _ := pythonBGPSpeakerAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/remove_bgp_peer") {
			return pythonBGPSpeakerWire(removeStatus, `{"NeutronError":{"type":"BgpSpeakerPeerNotAssociated"}}`)
		}
		return pythonBGPSpeakerWire(200, `{"bgp_peer_id":"bp-1"}`)
	})
	// add_bgp_peer_to_speaker("bs-1", "bp-1") returns the response dict.
	added, err := api.AddBGPPeer(ctx, "bs-1", speakers.AddBGPPeerOpts{BGPPeerID: "bp-1"})
	if err != nil || added.BGPPeerID != "bp-1" {
		t.Fatal(added, err)
	}
	// remove_bgp_peer_from_speaker("bs-1", "bp-1") returns None and raises on HTTP errors.
	if err := api.RemoveBGPPeer(ctx, "bs-1", speakers.RemoveBGPPeerOpts{BGPPeerID: "bp-1"}); err != nil {
		t.Fatal(err)
	}
	removeStatus = http.StatusBadRequest
	if err := api.RemoveBGPPeer(ctx, "bs-1", speakers.RemoveBGPPeerOpts{BGPPeerID: "bp-1"}); !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) {
		t.Fatal(err)
	}
	want := []pythonBGPSpeakerCall{
		{http.MethodPut, pythonBGPSpeakerBase + "/bs-1/add_bgp_peer", "", `{"bgp_peer_id":"bp-1"}`},
		{http.MethodPut, pythonBGPSpeakerBase + "/bs-1/remove_bgp_peer", "", `{"bgp_peer_id":"bp-1"}`},
		{http.MethodPut, pythonBGPSpeakerBase + "/bs-1/remove_bgp_peer", "", `{"bgp_peer_id":"bp-1"}`},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPSpeakerDynamicRoutingAgentCalls(t *testing.T) {
	ctx := context.Background()
	scheduleStatus, removeStatus := http.StatusCreated, http.StatusNoContent
	speakerAPI, agentAPI, calls, _ := pythonBGPSpeakerAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodPost:
			return pythonBGPSpeakerWire(scheduleStatus, `{"bgp_speaker_id":"bs-1"}`)
		case req.Method == http.MethodDelete:
			return pythonBGPSpeakerWire(removeStatus, ``)
		case req.URL.Path == pythonBGPSpeakerBase+"/bs-1/bgp-dragents":
			return pythonBGPSpeakerWire(200, `{"agents":[`+pythonBGPSpeakerDRRow+`]}`)
		case req.URL.Path == pythonBGPSpeakerAgent:
			return pythonBGPSpeakerWire(200, `{"agents":[{"id":"ag-0","agent_type":"L3 agent"},`+pythonBGPSpeakerDRRow+`]}`)
		case req.URL.Path == pythonBGPSpeakerAgent+"/ag-1/bgp-drinstances":
			return pythonBGPSpeakerWire(200, `{"bgp_speakers":[{"id":"bs-1","name":"spk"}]}`)
		}
		return pythonBGPSpeakerWire(200, `{"bgp_speakers":[{"id":"bs-0","name":"other"},`+pythonBGPSpeakerRow+`]}`)
	})
	// get_bgp_dragents_hosting_speaker("bs-1"): Python reads the hosting IDs, lists agents
	// and keeps the matching rows; Go decodes the hosting rows and can repeat the agent list.
	var hosting []string
	for agent, err := range agentAPI.ListDRAgentHostingBGPSpeakers(ctx, "bs-1") {
		if err != nil || agent.Host != "net1" || !agent.Alive || agent.AgentType != "BGP dynamic routing agent" {
			t.Fatal(agent, err)
		}
		hosting = append(hosting, agent.ID)
	}
	var matched []string
	for agent, err := range agentAPI.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(hosting, agent.ID) {
			matched = append(matched, agent.ID+"/"+agent.Binary)
		}
	}
	if !reflect.DeepEqual(matched, []string{"ag-1/neutron-bgp-dragent"}) {
		t.Fatal(matched)
	}
	// get_bgp_speakers_hosted_by_dragent("ag-1"): same two-step shape for speakers.
	var hosted []string
	for speaker, err := range agentAPI.ListBGPSpeakers(ctx, "ag-1") {
		if err != nil {
			t.Fatal(err)
		}
		hosted = append(hosted, speaker.ID)
	}
	var speakersMatched []string
	for speaker, err := range speakerAPI.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(hosted, speaker.ID) {
			speakersMatched = append(speakersMatched, speaker.ID+"/"+speaker.Name)
		}
	}
	if !reflect.DeepEqual(speakersMatched, []string{"bs-1/spk"}) {
		t.Fatal(speakersMatched)
	}
	// add_bgp_speaker_to_dragent("ag-1", "bs-1") and remove_bgp_speaker_from_dragent("ag-1", "bs-1").
	if err := agentAPI.ScheduleBGPSpeaker(ctx, "ag-1", agents.ScheduleBGPSpeakerOpts{SpeakerID: "bs-1"}); err != nil {
		t.Fatal(err)
	}
	if err := agentAPI.RemoveBGPSpeaker(ctx, "ag-1", "bs-1"); err != nil {
		t.Fatal(err)
	}
	// Python never checks these two responses; Go reports the HTTP failure.
	scheduleStatus, removeStatus = http.StatusConflict, http.StatusNotFound
	if err := agentAPI.ScheduleBGPSpeaker(ctx, "ag-1", agents.ScheduleBGPSpeakerOpts{SpeakerID: "bs-1"}); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	if err := agentAPI.RemoveBGPSpeaker(ctx, "ag-1", "bs-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	drinstances := pythonBGPSpeakerAgent + "/ag-1/bgp-drinstances"
	want := []pythonBGPSpeakerCall{
		{http.MethodGet, pythonBGPSpeakerBase + "/bs-1/bgp-dragents", "", ""},
		{http.MethodGet, pythonBGPSpeakerAgent, "", ""},
		{http.MethodGet, drinstances, "", ""},
		{http.MethodGet, pythonBGPSpeakerBase, "", ""},
		{http.MethodPost, drinstances, "", `{"bgp_speaker_id":"bs-1"}`},
		{http.MethodDelete, drinstances + "/bs-1", "", ""},
		{http.MethodPost, drinstances, "", `{"bgp_speaker_id":"bs-1"}`},
		{http.MethodDelete, drinstances + "/bs-1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
