package routers_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonRouterL3Transport func(*http.Request) (*http.Response, error)

func (transport pythonRouterL3Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonRouterL3Call struct{ method, path, query, body string }

func pythonRouterL3API(t *testing.T, calls *[]pythonRouterL3Call, code int, body string) *routers.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonRouterL3Transport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonRouterL3Call{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return routers.New(client)
}

// routers_hosting_l3_agents lists RouterL3Agent (Agent fields) at /routers/{router_id}/l3-agents.
func TestPythonRouterHostingL3Agents(t *testing.T) {
	ctx := context.Background()
	t.Run("routers_hosting_l3_agents", func(t *testing.T) {
		var calls []pythonRouterL3Call
		api := pythonRouterL3API(t, &calls, 200, `{"agents":[{"id":"a-1","agent_type":"L3 agent","binary":"neutron-l3-agent","host":"net-1","alive":true,"admin_state_up":true,"availability_zone":"nova","ha_state":"active","configurations":{"agent_mode":"legacy"}},{"id":"a-2","host":"net-2","ha_state":"standby"}]}`)
		var hosts, states []string
		for agent, err := range api.ListL3Agents(ctx, "r-1") {
			if err != nil {
				t.Fatal(err)
			}
			hosts = append(hosts, agent.Host)
			states = append(states, agent.HAState)
			if agent.ID == "a-1" && (!agent.Alive || !agent.AdminStateUp || agent.AgentType != "L3 agent" || agent.Binary != "neutron-l3-agent" || agent.AvailabilityZone != "nova" || agent.Configurations["agent_mode"] != "legacy") {
				t.Fatal(agent)
			}
		}
		if !reflect.DeepEqual(hosts, []string{"net-1", "net-2"}) || !reflect.DeepEqual(states, []string{"active", "standby"}) {
			t.Fatal(hosts, states)
		}
		if !reflect.DeepEqual(calls, []pythonRouterL3Call{{http.MethodGet, "/neutron/v2.0/routers/r-1/l3-agents", "", ""}}) {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("error", func(t *testing.T) {
		var calls []pythonRouterL3Call
		api := pythonRouterL3API(t, &calls, 404, `{}`)
		failed := false
		for _, err := range api.ListL3Agents(ctx, "r-1") {
			if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
				t.Fatal(err)
			}
			failed = true
		}
		if !failed || len(calls) != 1 {
			t.Fatalf("%+v", calls)
		}
	})
}
