package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionListPortValuesAbsentServiceSkipsFilterExpansion(t *testing.T) {
	fixture := testcloud.New(t)
	lookups, callbacks := 0, 0
	fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		lookups++
		return "", gophercloud.ErrEndpointNotFound{}
	}
	conn, err := sdk.FromProvider(fixture.Provider)
	if err != nil {
		t.Fatal(err)
	}
	result, err := conn.ListPortValues(context.Background(), ports.WithSourceCloudListFilters(json.RawMessage("true")), func(*ports.SourceCloudListOpts) error { callbacks++; return errors.New("must not execute") })
	if err != nil || result == nil || string(result.JSON) != "[]" || result.Values == nil || result.Ports == nil || result.Inventory == nil || callbacks != 0 || lookups != 1 {
		t.Fatal(result, err, lookups, callbacks)
	}
}

func TestConnectionListPortValuesCachedClientLocationAndTypedListCompatibility(t *testing.T) {
	fixture := testcloud.New(t)
	lookups, requests := 0, 0
	fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		lookups++
		return fixture.Server.URL + "/network/", nil
	}
	name := "scope-project"
	loc := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}
	conn, err := sdk.FromProvider(fixture.Provider, sdk.WithCloudLocation(loc))
	if err != nil {
		t.Fatal(err)
	}
	fixture.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		requests++
		testcloud.JSON(w, 203, `{"ports":[{"id":"p","project_id":"other"}]}`)
	})
	result, err := conn.ListPortValues(context.Background())
	if err != nil || result == nil || len(result.Ports) != 1 || requests != 1 || lookups != 1 {
		t.Fatal(result, err, requests, lookups)
	}
	var actual resource.CloudLocation
	if err := json.Unmarshal(result.Ports[0].Resource.Body["location"], &actual); err != nil {
		t.Fatal(err)
	}
	if string(actual.Project.ID) != `"other"` || actual.Project.Name != nil || result.Ports[0].Wire == nil || result.Ports[0].StatusCode != 203 {
		t.Fatal(actual, result)
	}
	literal := json.RawMessage("{\"jmespath_filters\":\"`[{\\\"id\\\":\\\"literal\\\"}]`\"}")
	values, err := conn.ListPortValues(context.Background(), ports.WithSourceCloudListFilters(literal))
	if err != nil || values == nil || string(values.JSON) != `[{"id":"literal"}]` || values.Ports != nil || len(values.Inventory) != 0 || requests != 1 || lookups != 1 {
		t.Fatal(values, err, requests, lookups)
	}
	typed, err := conn.ListPorts(context.Background())
	if err != nil || len(typed) != 1 || requests != 2 {
		t.Fatal(typed, err, requests)
	}
}
