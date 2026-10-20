package agents_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeAgentTransport func(*http.Request) (*http.Response, error)

func (transport nativeAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeAgentWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeAgentCall struct{ method, path, query, body string }

func nativeAgentAPI(t *testing.T, calls *[]nativeAgentCall, reply func(*http.Request) *http.Response) (*agents.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeAgentTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeAgentCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return agents.New(client), cloud
}

func nativeAgentOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "agents" {
		t.Fatal("generated agents context", err, wrapped)
	}
}

// Agent timestamps use Neutron's space-separated format without a zone.
const nativeAgentRow = `{"id":"ag-1","agent_type":"L3 agent","alive":true,"admin_state_up":true,"binary":"neutron-l3-agent","host":"net-1","configurations":{"agent_mode":"legacy"},"created_at":"2026-10-10 01:02:03","started_at":"","heartbeat_timestamp":"2026-10-11 04:05:06"}`

func TestNativeAgentRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeAgentCall
	var cloud *testcloud.Cloud
	api, cloud := nativeAgentAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeAgentWire(204, "")
		case req.Method == http.MethodPost:
			return nativeAgentWire(201, "")
		case req.URL.Path == "/neutron/v2.0/agents" || req.URL.Path == "/neutron/v2.0/bgp-speakers/sp-1/bgp-dragents":
			// AgentPage follows agents_links rel=next and ignores a links.next string.
			return nativeAgentWire(200, `{"agents":[`+nativeAgentRow+`],"agents_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/agents"}],"links":{"next":"`+cloud.Server.URL+`/never"}}`)
		case req.URL.Path == "/other/agents":
			return nativeAgentWire(200, `{"agents":[{"id":"ag-2","agent_type":"BGP dynamic routing agent"}]}`)
		case strings.HasSuffix(req.URL.Path, "/dhcp-networks"):
			return nativeAgentWire(200, `{"networks":[{"id":"net-1","name":"private"}]}`)
		case strings.HasSuffix(req.URL.Path, "/l3-routers"):
			return nativeAgentWire(200, `{"routers":[{"id":"r-1","name":"edge"}]}`)
		case strings.HasSuffix(req.URL.Path, "/bgp-drinstances"):
			// The BGP speaker list is a single page, so links.next is not followed.
			return nativeAgentWire(200, `{"bgp_speakers":[{"id":"sp-1","name":"speaker"}],"links":{"next":"`+cloud.Server.URL+`/never"}}`)
		}
		return nativeAgentWire(200, `{"agent":`+nativeAgentRow+`}`)
	})
	got, err := api.Get(ctx, "ag-1")
	if err != nil || !(got.ID == "ag-1" && got.Alive && got.Configurations["agent_mode"] == "legacy" &&
		got.CreatedAt.Equal(time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC)) && got.StartedAt.IsZero() &&
		got.HeartbeatTimestamp.Equal(time.Date(2026, 10, 11, 4, 5, 6, 0, time.UTC))) {
		t.Fatal(got, err)
	}
	empty, disabled := "", false
	if _, err := api.Update(ctx, "ag-1", agents.UpdateOpts{Description: &empty, AdminStateUp: &disabled}, agents.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var rows []*agents.Agent
	for value, err := range api.List(ctx, agents.WithListOptions(agents.ListOpts{AgentType: "L3 agent", Alive: &disabled, Limit: 1}), agents.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "ag-1" && rows[1].ID == "ag-2") {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "ag-1"); err != nil {
		t.Fatal(err)
	}
	networks, err := api.ListDHCPNetworks(ctx, "ag-1")
	if err != nil || len(networks) != 1 || networks[0].ID != "net-1" {
		t.Fatal(networks, err)
	}
	if err := api.ScheduleDHCPNetwork(ctx, "ag-1", agents.ScheduleDHCPNetworkOpts{NetworkID: "net-1"}, agents.WithScheduleDHCPNetworkField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveDHCPNetwork(ctx, "ag-1", "net-1"); err != nil {
		t.Fatal(err)
	}
	routers, err := api.ListL3Routers(ctx, "ag-1")
	if err != nil || len(routers) != 1 || routers[0].Name != "edge" {
		t.Fatal(routers, err)
	}
	if err := api.ScheduleL3Router(ctx, "ag-1", agents.ScheduleL3RouterOpts{RouterID: "r-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveL3Router(ctx, "ag-1", "r-1"); err != nil {
		t.Fatal(err)
	}
	var speakers []string
	for value, err := range api.ListBGPSpeakers(ctx, "ag-1") {
		if err != nil {
			t.Fatal(err)
		}
		speakers = append(speakers, value.ID+"/"+value.Name)
	}
	if !reflect.DeepEqual(speakers, []string{"sp-1/speaker"}) {
		t.Fatal(speakers)
	}
	if err := api.ScheduleBGPSpeaker(ctx, "ag-1", agents.ScheduleBGPSpeakerOpts{SpeakerID: "sp-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveBGPSpeaker(ctx, "ag-1", "sp-1"); err != nil {
		t.Fatal(err)
	}
	var hosts []string
	for value, err := range api.ListDRAgentHostingBGPSpeakers(ctx, "sp-1") {
		if err != nil {
			t.Fatal(err)
		}
		hosts = append(hosts, value.ID)
	}
	if !reflect.DeepEqual(hosts, []string{"ag-1", "ag-2"}) {
		t.Fatal(hosts)
	}
	query, _ := url.ParseQuery(calls[2].query)
	want := []nativeAgentCall{
		{http.MethodGet, "/neutron/v2.0/agents/ag-1", "", ""},
		{http.MethodPut, "/neutron/v2.0/agents/ag-1", "", `{"agent":{"admin_state_up":false,"description":"","x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/agents", calls[2].query, ""},
		{http.MethodGet, "/other/agents", "", ""},
		{http.MethodDelete, "/neutron/v2.0/agents/ag-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/agents/ag-1/dhcp-networks", "", ""},
		{http.MethodPost, "/neutron/v2.0/agents/ag-1/dhcp-networks", "", `{"network_id":"net-1","x_extension":1}`},
		{http.MethodDelete, "/neutron/v2.0/agents/ag-1/dhcp-networks/net-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/agents/ag-1/l3-routers", "", ""},
		{http.MethodPost, "/neutron/v2.0/agents/ag-1/l3-routers", "", `{"router_id":"r-1"}`},
		{http.MethodDelete, "/neutron/v2.0/agents/ag-1/l3-routers/r-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/agents/ag-1/bgp-drinstances", "", ""},
		{http.MethodPost, "/neutron/v2.0/agents/ag-1/bgp-drinstances", "", `{"bgp_speaker_id":"sp-1"}`},
		{http.MethodDelete, "/neutron/v2.0/agents/ag-1/bgp-drinstances/sp-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/bgp-speakers/sp-1/bgp-dragents", "", ""},
		{http.MethodGet, "/other/agents", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"agent_type": {"L3 agent"}, "alive": {"false"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeAgentStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*agents.API) error
	}{
		{"Get", []int{200}, func(api *agents.API) error { _, err := api.Get(ctx, "ag-1"); return err }},
		{"Update", []int{200}, func(api *agents.API) error { _, err := api.Update(ctx, "ag-1", agents.UpdateOpts{}); return err }},
		{"Delete", []int{202, 204}, func(api *agents.API) error { return api.Delete(ctx, "ag-1") }},
		{"ListDHCPNetworks", []int{200}, func(api *agents.API) error { _, err := api.ListDHCPNetworks(ctx, "ag-1"); return err }},
		{"ScheduleDHCPNetwork", []int{201}, func(api *agents.API) error {
			return api.ScheduleDHCPNetwork(ctx, "ag-1", agents.ScheduleDHCPNetworkOpts{NetworkID: "net-1"})
		}},
		{"RemoveDHCPNetwork", []int{202, 204}, func(api *agents.API) error { return api.RemoveDHCPNetwork(ctx, "ag-1", "net-1") }},
		{"ListL3Routers", []int{200}, func(api *agents.API) error { _, err := api.ListL3Routers(ctx, "ag-1"); return err }},
		{"ScheduleL3Router", []int{201}, func(api *agents.API) error {
			return api.ScheduleL3Router(ctx, "ag-1", agents.ScheduleL3RouterOpts{RouterID: "r-1"})
		}},
		{"RemoveL3Router", []int{202, 204}, func(api *agents.API) error { return api.RemoveL3Router(ctx, "ag-1", "r-1") }},
		{"ScheduleBGPSpeaker", []int{201}, func(api *agents.API) error {
			return api.ScheduleBGPSpeaker(ctx, "ag-1", agents.ScheduleBGPSpeakerOpts{SpeakerID: "sp-1"})
		}},
		{"RemoveBGPSpeaker", []int{202, 204}, func(api *agents.API) error { return api.RemoveBGPSpeaker(ctx, "ag-1", "sp-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeAgentCall
				api, _ := nativeAgentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAgentWire(code, `{"agent":{}}`) })
				err := call.call(api)
				nativeAgentOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope and timestamp decode", func(t *testing.T) {
		for body, want := range map[string]string{
			// The plain envelope pointer stays nil, so these return no agent and no error.
			`{}`:                   "nil",
			`{"agent":null}`:       "nil",
			`{"other":{"id":"x"}}`: "nil",
			`{"agent":{"id":"x"}}`: "x",
			`{"agent":[]}`:         "error",
			`{"agent":{"created_at":"2026-10-10T01:02:03Z"}}`: "error",
		} {
			var calls []nativeAgentCall
			api, _ := nativeAgentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAgentWire(200, body) })
			got, err := api.Get(ctx, "ag-1")
			switch want {
			case "error":
				nativeAgentOperation(t, err, "Get")
			case "nil":
				if err != nil || got != nil {
					t.Fatal(body, got, err)
				}
			default:
				if err != nil || got == nil || got.ID != want {
					t.Fatal(body, got, err)
				}
			}
		}
		var calls []nativeAgentCall
		api, _ := nativeAgentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAgentWire(200, `{}`) })
		networks, err := api.ListDHCPNetworks(ctx, "ag-1")
		routers, err2 := api.ListL3Routers(ctx, "ag-1")
		if err != nil || err2 != nil || networks != nil || routers != nil {
			t.Fatal(networks, err, routers, err2)
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		lists := map[string]func(*agents.API) []error{
			"List": func(api *agents.API) (errs []error) {
				for _, err := range api.List(ctx) {
					errs = append(errs, err)
				}
				return errs
			},
			"ListDRAgentHostingBGPSpeakers": func(api *agents.API) (errs []error) {
				for _, err := range api.ListDRAgentHostingBGPSpeakers(ctx, "sp-1") {
					errs = append(errs, err)
				}
				return errs
			},
			"ListBGPSpeakers": func(api *agents.API) (errs []error) {
				for _, err := range api.ListBGPSpeakers(ctx, "ag-1") {
					errs = append(errs, err)
				}
				return errs
			},
		}
		for name, list := range lists {
			for _, tc := range []struct {
				code int
				body string
			}{{404, `{}`}, {200, `{"agents":[],"bgp_speakers":[]}`}, {204, ""}} {
				var calls []nativeAgentCall
				api, _ := nativeAgentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAgentWire(tc.code, tc.body) })
				errs := list(api)
				var native gophercloud.ErrUnexpectedResponseCode
				var wrapped *resource.OperationError
				switch {
				case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && !errors.As(errs[0], &wrapped) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
				case tc.code == 200 && len(errs) == 0:
				case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
				default:
					t.Fatal(name, tc.code, errs)
				}
				if len(calls) != 1 {
					t.Fatal(name, calls)
				}
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeAgentCall
		api, _ := nativeAgentAPI(t, &calls, func(*http.Request) *http.Response { return nativeAgentWire(201, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"dhcp network required": {"ScheduleDHCPNetwork", api.ScheduleDHCPNetwork(ctx, "ag-1", agents.ScheduleDHCPNetworkOpts{})},
			"l3 router required":    {"ScheduleL3Router", api.ScheduleL3Router(ctx, "ag-1", agents.ScheduleL3RouterOpts{})},
			"bgp speaker required":  {"ScheduleBGPSpeaker", api.ScheduleBGPSpeaker(ctx, "ag-1", agents.ScheduleBGPSpeakerOpts{})},
			"dhcp network extension": {"ScheduleDHCPNetwork", api.ScheduleDHCPNetwork(ctx, "ag-1", agents.ScheduleDHCPNetworkOpts{NetworkID: "n"},
				agents.WithScheduleDHCPNetworkField("network_id", "x"))},
			"l3 router extension": {"ScheduleL3Router", api.ScheduleL3Router(ctx, "ag-1", agents.ScheduleL3RouterOpts{RouterID: "r"},
				agents.WithScheduleL3RouterField("router_id", "x"))},
			"bgp speaker extension": {"ScheduleBGPSpeaker", api.ScheduleBGPSpeaker(ctx, "ag-1", agents.ScheduleBGPSpeakerOpts{SpeakerID: "s"},
				agents.WithScheduleBGPSpeakerField("bgp_speaker_id", "x"))},
			"update extension": {"Update", func() error {
				_, err := api.Update(ctx, "ag-1", agents.UpdateOpts{}, agents.WithUpdateField("description", "x"))
				return err
			}()},
			"update nil option": {"Update", func() error { _, err := api.Update(ctx, "ag-1", agents.UpdateOpts{}, nil); return err }()},
			"dhcp nil option":   {"ScheduleDHCPNetwork", api.ScheduleDHCPNetwork(ctx, "ag-1", agents.ScheduleDHCPNetworkOpts{NetworkID: "n"}, nil)},
			"l3 nil option":     {"ScheduleL3Router", api.ScheduleL3Router(ctx, "ag-1", agents.ScheduleL3RouterOpts{RouterID: "r"}, nil)},
			"bgp nil option":    {"ScheduleBGPSpeaker", api.ScheduleBGPSpeaker(ctx, "ag-1", agents.ScheduleBGPSpeakerOpts{SpeakerID: "s"}, nil)},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeAgentOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeAgentOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
