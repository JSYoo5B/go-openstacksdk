package connectiontest_test

import (
	"context"
	"encoding/json"
	"errors"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"testing"
)

func TestConnectionCycle8MissingNeutronOnlyListsSkipOptions(t *testing.T) {
	cloud := testcloud.New(t)
	calls, opts := 0, 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { calls++; return "", gophercloud.ErrEndpointNotFound{} }
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	n, err := conn.ListNetworkRecords(context.Background(), func(*network.NetworkCloudListOpts) error { opts++; return errors.New("must skip") })
	if err != nil || n == nil || string(n.JSON) != "[]" || opts != 0 {
		t.Fatal(n, err, opts)
	}
	s, err := conn.ListSubnetRecords(context.Background(), func(*network.SubnetReadOpts) error { opts++; return errors.New("must skip") })
	if err != nil || s == nil || string(s.JSON) != "[]" || opts != 0 {
		t.Fatal(s, err, opts)
	}
	if _, err := conn.SearchNetworkRecords(context.Background(), "x"); err == nil {
		t.Fatal("search swallowed absence")
	}
	if _, err := conn.SearchSubnetRecords(context.Background(), "x"); err == nil || calls != 4 {
		t.Fatal(err, calls)
	}
}
func TestConnectionCycle8NetworkReadsCacheServiceAndForwardNameQuery(t *testing.T) {
	cloud := testcloud.New(t)
	locates, gets := 0, 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { locates++; return cloud.Server.URL + "/network/", nil }
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		gets++
		if r.URL.Query().Get("name") != "net" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"n","name":"net","project_id":"foreign","availability_zone":false}]}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
		gets++
		if r.URL.Query().Get("name") != "sub" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"subnets":[{"id":"s","name":"sub","project_id":"foreign","availability_zone":0}]}`)
	})
	n, err := conn.SearchNetworkRecords(context.Background(), "net")
	if err != nil || n == nil || len(n.Networks) != 1 || string(n.Networks[0].Resource.Body["location"]) == "null" {
		t.Fatal(n, err)
	}
	s, err := conn.SearchSubnetRecords(context.Background(), "sub")
	if err != nil || s == nil || len(s.Subnets) != 1 || string(s.Subnets[0].Resource.Body["location"]) == "null" {
		t.Fatal(s, err)
	}
	if locates != 1 || gets != 2 {
		t.Fatal(locates, gets)
	}
}
func TestConnectionCycle8ConsoleUsesCachedComputeAndSuppliedDictionary(t *testing.T) {
	cloud := testcloud.New(t)
	locates, actions := 0, 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return cloud.Server.URL + "/compute/v2.1/", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("POST /compute/v2.1/servers/s/action", func(w http.ResponseWriter, r *http.Request) { actions++; testcloud.JSON(w, 200, `{"output":false}`) })
	for range 2 {
		v, err := conn.GetServerConsole(context.Background(), "ignored", compute.WithServerConsoleServerJSON(json.RawMessage(`{"id":"s"}`)))
		if err != nil || v == nil || string(v.Value) != "false" || !v.HasOutput {
			t.Fatal(v, err)
		}
	}
	if locates != 1 || actions != 2 {
		t.Fatal(locates, actions)
	}
}
func TestConnectionCycle8CancelledReadsNeverDiscover(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { calls++; return "", gophercloud.ErrEndpointNotFound{} }
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("custom read cancel")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	_, err = conn.ListNetworkRecords(ctx)
	if !errors.Is(err, cause) {
		t.Fatal(err)
	}
	_, err = conn.ListSubnetRecords(ctx)
	if !errors.Is(err, cause) {
		t.Fatal(err)
	}
	_, err = conn.SearchServerGroups(ctx, "")
	if !errors.Is(err, cause) {
		t.Fatal(err)
	}
	_, err = conn.GetServerGroup(ctx, "x")
	if !errors.Is(err, cause) {
		t.Fatal(err)
	}
	_, err = conn.GetServerConsole(ctx, "x")
	if !errors.Is(err, cause) || calls != 0 {
		t.Fatal(err, calls)
	}
}
