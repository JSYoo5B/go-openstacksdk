package agents_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonParityCoreAgentTransport func(*http.Request) (*http.Response, error)

func (transport pythonParityCoreAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonParityCoreAgentCall struct{ method, path, query, body string }

func pythonParityCoreAgentAPI(t *testing.T, calls *[]pythonParityCoreAgentCall, reply func(*http.Request) (int, string)) (*agents.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonParityCoreAgentTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonParityCoreAgentCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return agents.New(client), cloud
}

const pythonParityCoreAgentBase = "/neutron/v2.0/agents"

// Python agents(**query) transposes is_alive/is_admin_state_up to alive and
// admin_state_up and follows agents_links; Go sends the same wire keys through
// ListOpts and WithListQuery and follows the same link.
func TestPythonParityCoreAgentListQueryAndLinks(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreAgentCall
	var cloud *testcloud.Cloud
	api, cloud := pythonParityCoreAgentAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"agents":[{"id":"ag-1","agent_type":"DHCP agent","host":"net-1","alive":true,"admin_state_up":false,"resources_synced":true,"configurations":{"networks":2}}],"agents_links":[{"rel":"next","href":"` + cloud.Server.URL + `/neutron/v2.0/agents?marker=ag-1"}]}`
		}
		return 200, `{"agents":[]}`
	})
	alive := true
	var rows []*agents.Agent
	for value, err := range api.List(ctx,
		agents.WithListOptions(agents.ListOpts{AgentType: "DHCP agent", Host: "net-1", Alive: &alive}),
		agents.WithListQuery("admin_state_up", "False")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if len(rows) != 1 || rows[0].ID != "ag-1" || rows[0].AdminStateUp || !rows[0].ResourcesSynced || rows[0].Configurations["networks"] != float64(2) {
		t.Fatalf("%+v", rows)
	}
	first, _ := url.ParseQuery(calls[0].query)
	want := url.Values{"agent_type": {"DHCP agent"}, "host": {"net-1"}, "alive": {"true"}, "admin_state_up": {"False"}}
	if len(calls) != 2 || calls[0].path != pythonParityCoreAgentBase || !reflect.DeepEqual(first, want) || calls[1].query != "marker=ag-1" {
		t.Fatalf("%+v %v", calls, first)
	}

	// Without arguments both sides send a bare GET agents.
	calls = nil
	all, err := api.All(ctx)
	if err != nil || len(calls) != 2 || calls[0].query != "" || len(all) != 1 {
		t.Fatalf("%+v %v", calls, err)
	}
}

// Python update_agent commits only the given attributes in the agent envelope
// with PUT; delete_agent ignores 404 by default.
func TestPythonParityCoreAgentUpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreAgentCall
	api, _ := pythonParityCoreAgentAPI(t, &calls, func(req *http.Request) (int, string) {
		return 200, `{"agent":{"id":"ag-1","description":"rack 2","admin_state_up":false}}`
	})
	description, disabled := "rack 2", false
	updated, err := api.Update(ctx, "ag-1", agents.UpdateOpts{Description: &description, AdminStateUp: &disabled})
	if err != nil || updated.Description != "rack 2" || updated.AdminStateUp {
		t.Fatal(updated, err)
	}
	if !reflect.DeepEqual(calls, []pythonParityCoreAgentCall{{http.MethodPut, pythonParityCoreAgentBase + "/ag-1", "", `{"agent":{"admin_state_up":false,"description":"rack 2"}}`}}) {
		t.Fatalf("%+v", calls)
	}

	for _, tc := range []struct {
		name    string
		status  int
		options []resource.LookupOption
		missing bool
	}{
		{"deleted", 204, nil, false},
		{"missing ignored by default", 404, nil, false},
		{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonParityCoreAgentCall
			api, _ := pythonParityCoreAgentAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			err := api.Remove(ctx, resource.ID("ag-1"), tc.options...)
			if tc.missing != errors.Is(err, resource.ErrNotFound) || !tc.missing && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonParityCoreAgentCall{{http.MethodDelete, pythonParityCoreAgentBase + "/ag-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// Python add_dhcp_agent_to_network posts {"network_id": id}; remove sends
// DELETE on the member route (Python also attaches a JSON body that Go omits).
// dhcp_agent_hosting_networks and get_agent without arguments send a bare GET.
func TestPythonParityCoreAgentDHCPNetworkRoutes(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreAgentCall
	api, _ := pythonParityCoreAgentAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodPost:
			return 201, `null`
		case req.Method == http.MethodDelete:
			return 204, ""
		case strings.HasSuffix(req.URL.Path, "/dhcp-networks"):
			return 200, `{"networks":[{"id":"net-1","name":"private"}]}`
		}
		return 200, `{"agent":{"id":"ag-1","agent_type":"DHCP agent"}}`
	})
	if err := api.ScheduleDHCPNetwork(ctx, "ag-1", agents.ScheduleDHCPNetworkOpts{NetworkID: "net-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveDHCPNetwork(ctx, "ag-1", "net-1"); err != nil {
		t.Fatal(err)
	}
	networks, err := api.ListDHCPNetworks(ctx, "ag-1")
	if err != nil || len(networks) != 1 || networks[0].Name != "private" {
		t.Fatal(networks, err)
	}
	got, err := api.Get(ctx, "ag-1")
	if err != nil || got.AgentType != "DHCP agent" {
		t.Fatal(got, err)
	}
	want := []pythonParityCoreAgentCall{
		{http.MethodPost, pythonParityCoreAgentBase + "/ag-1/dhcp-networks", "", `{"network_id":"net-1"}`},
		{http.MethodDelete, pythonParityCoreAgentBase + "/ag-1/dhcp-networks/net-1", "", ""},
		{http.MethodGet, pythonParityCoreAgentBase + "/ag-1/dhcp-networks", "", ""},
		{http.MethodGet, pythonParityCoreAgentBase + "/ag-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
