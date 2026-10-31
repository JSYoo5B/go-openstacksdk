package connectiontest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionSecurityGroupRecordsNeutronDefaultsAndLocationBeforeFilter(t *testing.T) {
	cloud := testcloud.New(t)
	locates := 0
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		locates++
		if o.Type != "network" {
			t.Fatal(o)
		}
		return cloud.Server.URL + "/", nil
	}
	configured := "configured"
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(resource.CloudLocation{Cloud: &configured}))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	cloud.Mux.HandleFunc("GET /v2.0/security-groups", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"security_groups":[{"id":"sg","name":"web","location":{"cloud":"forged"}}]}`)
	})
	result, err := conn.SearchSecurityGroupRecords(context.Background(), "web", network.WithSecurityGroupReadFiltersJSON(json.RawMessage(`{"location":{"cloud":"configured"}}`)))
	if err != nil || result == nil || result.Backend != "neutron" || len(result.Groups) != 1 {
		t.Fatal(result, err)
	}
	if string(result.Groups[0].Wire.Body["location"]) != `{"cloud":"forged"}` {
		t.Fatal(result.Groups[0])
	}
	result, err = conn.GetSecurityGroupRecord(context.Background(), "web")
	if err != nil || result == nil || result.Group == nil || locates != 1 || calls != 2 {
		t.Fatal(result, err, locates, calls)
	}
}

func TestConnectionSecurityGroupRecordsCleanCatalogAbsenceSelectsNova(t *testing.T) {
	cloud := testcloud.New(t)
	computeLocates := 0
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		if o.Type == "network" {
			return "", gophercloud.ErrEndpointNotFound{}
		}
		if o.Type == "compute" {
			computeLocates++
			return cloud.Server.URL + "/v2.1/p/", nil
		}
		t.Fatal(o)
		return "", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	cloud.Mux.HandleFunc("GET /v2.1/p/os-security-groups", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"security_groups":[{"id":42,"name":"web","description":"d","tenant_id":"p","rules":[],"extra":"retained"}]}`)
	})
	result, err := conn.SearchSecurityGroupRecords(context.Background(), "web")
	if err != nil || result == nil || result.Backend != "nova" || len(result.Groups) != 1 || result.NovaResponse == nil || result.NovaResponse.StatusCode != 200 || computeLocates != 1 || calls != 1 {
		t.Fatal(result, err, computeLocates, calls)
	}
	if string(result.Groups[0].Wire.Body["extra"]) != `"retained"` || string(result.Groups[0].Resource.Body["id"]) != `42` {
		t.Fatal(result.Groups[0])
	}
}

func TestConnectionSecurityGroupRecordsDoNotFallbackOnJoinedFailureOrCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	cause := errors.New("catalog transport failed")
	computeLocates := 0
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		if o.Type == "compute" {
			computeLocates++
		}
		return "", errors.Join(gophercloud.ErrEndpointNotFound{}, cause)
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.SearchSecurityGroupRecords(context.Background(), ""); !errors.Is(err, cause) || computeLocates != 0 {
		t.Fatal(err, computeLocates)
	}
	if _, err := conn.GetSecurityGroupRecord(nil, "sg"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cancel(cause)
	if _, err := conn.SecurityGroupCloud(ctx); !errors.Is(err, cause) || computeLocates != 0 {
		t.Fatal(err, computeLocates)
	}
}

func TestConnectionSecurityGroupExistingReferenceBypassesDiscoveryAndOptions(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { calls++; return "", gophercloud.ErrEndpointNotFound{} }
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	reference := map[string]json.RawMessage{"id": json.RawMessage(`null`), "name": json.RawMessage(`"existing"`)}
	optionCalls := 0
	result, err := conn.GetSecurityGroupRecord(context.Background(), reference, func(*network.SecurityGroupReadOpts) error { optionCalls++; return errors.New("should bypass") })
	if err != nil || result == nil || result.Backend != "existing" || calls != 0 || optionCalls != 0 {
		t.Fatal(result, err, calls, optionCalls)
	}
	reference["name"][1] = 'X'
	var selected map[string]json.RawMessage
	if err := json.Unmarshal(result.Value, &selected); err != nil || string(selected["name"]) != `"existing"` {
		t.Fatal(selected, err)
	}
}
