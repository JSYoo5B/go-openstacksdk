package agents_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
	"strings"
	"testing"
)

func TestPythonDHCPHostsInheritedQueriesAndDescriptorViews(t *testing.T) {
	var calls []pythonParityCoreAgentCall
	api, _ := pythonParityCoreAgentAPI(t, &calls, func(r *http.Request) (int, string) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/neutron/v2.0/networks/net%2Fid/dhcp-agents" || r.URL.Query().Get("alive") != "False" || r.URL.Query().Get("host") != "node" || r.URL.Query().Has("ignored") {
			t.Error(r.Method, r.URL)
		}
		return 202, `{"agents":[{"id":"a","alive":0,"admin_state_up":"False","resources_synced":[],"configurations":["raw"],"heartbeat_timestamp":"literal","ha_chassis_priority":"٢","unknown":{"large":9007199254740993}}]}`
	})
	rows, err := api.AllDHCPHostRecords(context.Background(), "net/id", agents.WithDHCPHostListQuery("is_alive", false), agents.WithDHCPHostListQuery("host", "node"), agents.WithDHCPHostListQuery("ignored", true))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	row := rows[0]
	for k, w := range map[string]string{"is_alive": "false", "is_admin_state_up": "true", "resources_synced": "false", "configuration": `["raw"]`, "last_heartbeat_at": `"literal"`, "ha_chassis_priority": "2", "name": "null", "location": "null"} {
		if string(row.Resource.Body[k]) != w {
			t.Fatal(k, row.Resource.Body)
		}
	}
	if row.Resource.Body["network_id"] != nil || string(row.Resource.Body["unknown"]) != `{"large":9007199254740993}` || row.StatusCode != 202 || string(row.Wire.Body["alive"]) != "0" {
		t.Fatal(row)
	}
}
func TestPythonDHCPHostsCapsLocalFiltersProxyAndParentRoute(t *testing.T) {
	var calls []pythonParityCoreAgentCall
	api, _ := pythonParityCoreAgentAPI(t, &calls, func(r *http.Request) (int, string) {
		if r.URL.Query().Has("configuration") {
			t.Error(r.URL)
		}
		return 200, `{"agents":[{"id":"one","configurations":{"n":1}},{"id":"two","configurations":{"n":2}}]}`
	})
	proxy, err := api.ProxyDHCPHostRecords(context.Background(), "net", agents.WithDHCPHostListJMESPath("type(@)"))
	if err != nil || len(calls) != 0 || string(proxy.JSON) != "null" {
		t.Fatal(proxy, err, calls)
	}
	rows, err := api.AllDHCPHostRecords(context.Background(), "net", agents.WithDHCPHostListQuery("configuration", map[string]any{"n": 2}), agents.WithDHCPHostListMaxItems(1))
	if err != nil || len(rows) != 0 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	rows, err = api.AllDHCPHostRecords(context.Background(), "net", agents.WithDHCPHostListQuery("configuration", map[string]any{"n": 2}))
	if err != nil || len(rows) != 1 || string(rows[0].Resource.Body["id"]) != `"two"` {
		t.Fatal(rows, err)
	}
	before := len(calls)
	_, err = api.AllDHCPHostRecords(context.Background(), "net", agents.WithDHCPHostListQuery("network_id", "other"))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
	_, err = api.AllDHCPHostRecords(context.Background(), "net", agents.WithDHCPHostListQuery("__dhcp_owned_base_path", "https://outside.invalid/"))
	if err != nil || len(calls) != before+1 || strings.Contains(calls[len(calls)-1].path, "outside") {
		t.Fatal(err, calls)
	}
}
func TestPythonDHCPHostsPagingLateFailureAndOwnedOptions(t *testing.T) {
	var calls []pythonParityCoreAgentCall
	api, cloud := pythonParityCoreAgentAPI(t, &calls, func(r *http.Request) (int, string) {
		if r.URL.Query().Get("marker") == "a" {
			return 503, `{"error":"late"}`
		}
		return 200, `{"agents":[{"id":"a"}],"links":[{"rel":"next","href":"?marker=a"}]}`
	})
	_, err := api.AllDHCPHostRecords(context.Background(), "net")
	if err == nil || len(calls) != 2 {
		t.Fatal(err, calls)
	}
	before := len(calls)
	_, err = api.AllDHCPHostRecords(context.Background(), "net", func(v *agents.DHCPHostListOpts) error {
		v.Query = map[string]json.RawMessage{"host": json.RawMessage(`"owned"`)}
		*api = *agents.New(cloud.Client("network", "/changed/"))
		return nil
	})
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
	for _, id := range []string{"", ".", "..", "bad\n"} {
		_, err = api.AllDHCPHostRecords(context.Background(), id)
		if err == nil {
			t.Fatal(fmt.Sprintf("invalid parent %q accepted", id))
		}
	}
}
