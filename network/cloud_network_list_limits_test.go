package network_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network"
)

// TestCloudNetworkSubnetListsSourceLimitsAndCaps covers cloud
// list/search_networks and list/search_subnets filters forwarded to
// Resource.list: an integer max_items equal to one whole page performs the
// next marker GET, a falsey raw limit (false) is sent without the marker GET,
// a negative raw limit is sent and drives the marker GET, raw true/negative/
// fractional caps compare before each raw row, and base_path=""
// returns an empty result without HTTP even with an invalid microversion.
func TestCloudNetworkSubnetListsSourceLimitsAndCaps(t *testing.T) {
	type lister func(*network.Service, json.RawMessage) (int, error)
	listNetworks := func(search bool) lister {
		return func(s *network.Service, filters json.RawMessage) (int, error) {
			var result *network.NetworkCloudRecords
			var err error
			if search {
				result, err = s.SearchNetworkRecords(context.Background(), "", network.WithNetworkCloudFilters(filters))
			} else {
				result, err = s.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(filters))
			}
			if result == nil {
				return 0, err
			}
			return len(result.Networks), err
		}
	}
	listSubnets := func(search bool) lister {
		return func(s *network.Service, filters json.RawMessage) (int, error) {
			var result *network.SubnetRecords
			var err error
			if search {
				result, err = s.SearchSubnetRecords(context.Background(), "", network.WithSubnetReadFiltersJSON(filters))
			} else {
				result, err = s.ListSubnetRecords(context.Background(), network.WithSubnetReadFiltersJSON(filters))
			}
			if result == nil {
				return 0, err
			}
			return len(result.Subnets), err
		}
	}
	for _, family := range []struct {
		name, plural string
		list         lister
	}{
		{"list_networks", "networks", listNetworks(false)},
		{"search_networks", "networks", listNetworks(true)},
		{"list_subnets", "subnets", listSubnets(false)},
		{"search_subnets", "subnets", listSubnets(true)},
	} {
		page := `{"` + family.plural + `":[{"id":"one"},{"id":"two"}]}`
		empty := `{"` + family.plural + `":[]}`
		path := "/v2.0/" + family.plural
		for _, tc := range []struct {
			name, filters string
			replies       map[string]receiptReply
			rows          int
		}{
			{"cap equals page", `{"max_items":2}`, map[string]receiptReply{
				"GET " + path + "?limit=2":            {status: http.StatusOK, body: page},
				"GET " + path + "?limit=2&marker=two": {status: http.StatusOK, body: `{"` + family.plural + `":[{"id":"three"}]}`},
			}, 2},
			{"falsey limit", `{"limit":false}`, map[string]receiptReply{
				"GET " + path + "?limit=False": {status: http.StatusOK, body: page},
			}, 2},
			{"negative limit", `{"limit":-1}`, map[string]receiptReply{
				"GET " + path + "?limit=-1":            {status: http.StatusOK, body: page},
				"GET " + path + "?limit=-1&marker=two": {status: http.StatusOK, body: empty},
			}, 2},
			{"true cap", `{"max_items":true}`, map[string]receiptReply{
				"GET " + path + "?limit=True": {status: http.StatusOK, body: page},
			}, 1},
			{"negative cap", `{"max_items":-1}`, map[string]receiptReply{
				"GET " + path + "?limit=-1": {status: http.StatusOK, body: page},
			}, 0},
			{"fractional cap", `{"max_items":1.5}`, map[string]receiptReply{
				"GET " + path + "?limit=1.5":            {status: http.StatusOK, body: page},
				"GET " + path + "?limit=1.5&marker=two": {status: http.StatusOK, body: empty},
			}, 2},
			{"empty base path", `{"base_path":"","microversion":"broken"}`, map[string]receiptReply{}, 0},
		} {
			t.Run(family.name+"/"+tc.name, func(t *testing.T) {
				client, calls := sourceStateClient(t, tc.replies)
				rows, err := family.list(network.New(client), json.RawMessage(tc.filters))
				if err != nil || rows != tc.rows || len(calls()) != len(tc.replies) {
					t.Fatal(rows, err, calls())
				}
			})
		}
	}
}
