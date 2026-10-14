package agents_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const pythonDHCPNetworksPath = pythonParityCoreAgentBase + "/ag-1/dhcp-networks"

var pythonDHCPNetworkRows = map[string]string{
	"":      `{"id":"net-1","name":"a","mtu":1500,"subnets":["s1"],"shared":true,"self":"x","tags":["t"]}`,
	"net-1": `{"id":"net-2","name":"b","mtu":1450,"subnets":["s1"],"shared":true,"dns_domain":null}`,
	"net-2": `{"id":"net-3","name":"c","mtu":1450,"subnets":[],"shared":true}`,
}

func pythonDHCPNetworkQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	values, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatal(err)
	}
	return values
}

func pythonDHCPNetworkIDs(t *testing.T, seq func(func(*agents.DHCPNetworkRecord, error) bool)) []string {
	t.Helper()
	ids := []string{}
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, row.Network.ID)
	}
	return ids
}

// dhcp_agent_hosting_networks(agent, is_shared=True, fields=["id", "name"],
// limit=1): the first page advertises a /v2.0 relative next link, the second
// has none, so Python continues with marker=<last id> until an empty page.
func TestPythonDHCPNetworkRecordsQueryAndPaging(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreAgentCall
	api, _ := pythonParityCoreAgentAPI(t, &calls, func(req *http.Request) (int, string) {
		marker := req.URL.Query().Get("marker")
		switch marker {
		case "":
			return 200, `{"networks":[` + pythonDHCPNetworkRows[""] + `],"networks_links":[{"rel":"next","href":"/v2.0/agents/ag-1/dhcp-networks?fields=id&fields=name&limit=1&marker=net-1&shared=True"}]}`
		case "net-1":
			return 200, `{"networks":[` + pythonDHCPNetworkRows["net-1"] + `]}`
		}
		return 200, `{"networks":[]}`
	})
	var rows []*agents.DHCPNetworkRecord
	for row, err := range api.ListDHCPNetworkRecords(ctx, "ag-1",
		agents.WithDHCPNetworkRecordsQuery("is_shared", true),
		agents.WithDHCPNetworkRecordsQuery("fields", []string{"id", "name"}),
		agents.WithDHCPNetworkRecordsQuery("limit", 1),
		agents.WithDHCPNetworkRecordsQuery("unknown_key", "dropped"),
		agents.WithDHCPNetworkRecordsHeader("X-Trace", "t"),
	) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 2 || rows[0].Network.ID != "net-1" || rows[0].Network.Name != "a" || !rows[0].Network.Shared || rows[0].AgentID != "ag-1" || rows[0].StatusCode != 200 {
		t.Fatal(rows)
	}
	if _, exists := rows[0].Fields["self"]; exists || string(rows[0].Fields["mtu"]) != "1500" || string(rows[1].Fields["dns_domain"]) != "null" {
		t.Fatal(rows[0].Fields, rows[1].Fields)
	}
	want := []url.Values{
		{"shared": {"True"}, "fields": {"id", "name"}, "limit": {"1"}},
		{"shared": {"True"}, "fields": {"id", "name"}, "limit": {"1"}, "marker": {"net-1"}},
		{"shared": {"True"}, "fields": {"id", "name"}, "limit": {"1"}, "marker": {"net-2"}},
	}
	if len(calls) != len(want) {
		t.Fatalf("%+v", calls)
	}
	for i, call := range calls {
		if call.method != http.MethodGet || call.path != pythonDHCPNetworksPath || !reflect.DeepEqual(pythonDHCPNetworkQuery(t, call.query), want[i]) {
			t.Fatalf("call %d: %+v", i, call)
		}
	}
}

func TestPythonDHCPNetworkRecordsClientFiltersAndControls(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreAgentCall
	api, _ := pythonParityCoreAgentAPI(t, &calls, func(req *http.Request) (int, string) {
		rows := `[` + pythonDHCPNetworkRows[""] + `,` + pythonDHCPNetworkRows["net-1"] + `,` + pythonDHCPNetworkRows["net-2"] + `]`
		if req.URL.Query().Get("marker") != "" {
			rows = `[]`
		}
		return 200, `{"networks":` + rows + `}`
	})
	// dhcp_agent_hosting_networks(agent, mtu=1450, subnet_ids=["s1"]) filters
	// Body attributes locally; the server sees no query.
	if ids := pythonDHCPNetworkIDs(t, api.ListDHCPNetworkRecords(ctx, "ag-1",
		agents.WithDHCPNetworkRecordsQuery("mtu", 1450), agents.WithDHCPNetworkRecordsQuery("subnet_ids", []string{"s1"}))); !reflect.DeepEqual(ids, []string{"net-2"}) {
		t.Fatal(ids)
	}
	// dns_domain=None matches a missing or null attribute.
	if ids := pythonDHCPNetworkIDs(t, api.ListDHCPNetworkRecords(ctx, "ag-1", agents.WithDHCPNetworkRecordsQuery("dns_domain", nil))); !reflect.DeepEqual(ids, []string{"net-1", "net-2", "net-3"}) {
		t.Fatal(ids)
	}
	// max_items=2 is sent as limit=2, counts the filtered-out net-1 and stops
	// after two rows without another page.
	if ids := pythonDHCPNetworkIDs(t, api.ListDHCPNetworkRecords(ctx, "ag-1",
		agents.WithDHCPNetworkRecordsMaxItems(2), agents.WithDHCPNetworkRecordsQuery("mtu", 1450))); !reflect.DeepEqual(ids, []string{"net-2"}) {
		t.Fatal(ids)
	}
	// paginated=False with limit=3 reads only the first page.
	if ids := pythonDHCPNetworkIDs(t, api.ListDHCPNetworkRecords(ctx, "ag-1",
		agents.WithDHCPNetworkRecordsPaginated(false), agents.WithDHCPNetworkRecordsQuery("limit", 3))); len(ids) != 3 {
		t.Fatal(ids)
	}
	want := []string{"", "", "limit=2", "limit=3"}
	if len(calls) != len(want) {
		t.Fatalf("%+v", calls)
	}
	for i, call := range calls {
		if call.path != pythonDHCPNetworksPath || call.query != want[i] {
			t.Fatalf("call %d: %+v", i, call)
		}
	}
}

func TestPythonDHCPNetworkRecordsRejectsInputsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreAgentCall
	api, _ := pythonParityCoreAgentAPI(t, &calls, func(*http.Request) (int, string) {
		return 404, `{"NeutronError":{"type":"AgentNotFound","message":"missing"}}`
	})
	cases := []struct {
		agent   string
		options []agents.DHCPNetworkRecordsOption
	}{
		{" ", nil},
		{"ag-1", []agents.DHCPNetworkRecordsOption{nil}},
		{"ag-1", []agents.DHCPNetworkRecordsOption{agents.WithDHCPNetworkRecordsQuery("max_items", 1)}},
		{"ag-1", []agents.DHCPNetworkRecordsOption{agents.WithDHCPNetworkRecordsQuery(" ", 1)}},
		{"ag-1", []agents.DHCPNetworkRecordsOption{agents.WithDHCPNetworkRecordsMaxItems(-1)}},
		{"ag-1", []agents.DHCPNetworkRecordsOption{agents.WithDHCPNetworkRecordsQuery("bad", func() {})}},
	}
	for i, tc := range cases {
		for _, err := range api.ListDHCPNetworkRecords(ctx, tc.agent, tc.options...) {
			var operation *resource.OperationError
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "ListDHCPNetworkRecords" {
				t.Fatal(i, err)
			}
		}
	}
	if len(calls) != 0 {
		t.Fatalf("%+v", calls)
	}
	seen := false
	for _, err := range api.ListDHCPNetworkRecords(ctx, "ag-1") {
		seen = true
		var operation *resource.OperationError
		if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) || !errors.As(err, &operation) || operation.Resource != "agents" {
			t.Fatal(err)
		}
	}
	var nilAPI *agents.API
	for _, err := range nilAPI.ListDHCPNetworkRecords(ctx, "ag-1") {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if !seen || len(calls) != 1 {
		t.Fatalf("%+v", calls)
	}
}
