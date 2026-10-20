package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func subnetCloudListCall(s *network.Service, ctx context.Context, search bool, opts ...network.SubnetReadOption) (*network.SubnetRecords, error) {
	if search {
		return s.SearchSubnetRecords(ctx, "literal*id", opts...)
	}
	return s.ListSubnetRecords(ctx, opts...)
}
func TestCloudSubnetListsDirectQuerySearchOverrideAndFalseyFilters(t *testing.T) {
	for _, search := range []bool{false, true} {
		for _, raw := range []string{"null", "false", "0", `""`, "[]", "{}"} {
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				want := ""
				if search {
					want = "literal*id"
				}
				if r.URL.Path != "/neutron/v2.0/subnets" || r.URL.Query().Get("name") != want {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"subnets":[{"id":"actual","name":"other"}]}`)
			})
			result, err := subnetCloudListCall(service, context.Background(), search, network.WithSubnetReadFiltersJSON(json.RawMessage(raw)))
			if err != nil || len(result.Subnets) != 1 || len(result.Inventory) != 1 || len(result.Value) != 1 || count.Load() != 1 {
				t.Fatal(search, raw, result, err, count.Load())
			}
		}
	}
	for _, raw := range []string{`{"name":"override","is_dhcp_enabled":false}`, `[["name","override"],["is_dhcp_enabled",false]]`} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("name") != "override" || r.URL.Query().Get("enable_dhcp") != "False" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"subnets":[]}`)
		})
		result, err := service.SearchSubnetRecords(context.Background(), "12345678-1234-1234-1234-123456789abc", network.WithSubnetReadFiltersJSON(json.RawMessage(raw)))
		if err != nil || string(result.JSON) != "[]" || count.Load() != 1 {
			t.Fatal(raw, result, err)
		}
	}
}
func TestCloudSubnetListsOnlyListMissingServiceSkipsInvalidFilters(t *testing.T) {
	var service *network.Service
	result, err := service.ListSubnetRecords(context.Background(), network.WithSubnetNetworkAvailable(false), network.WithSubnetReadFiltersJSON(json.RawMessage(`{"headers":"bad","jmespath_filters":"bad!"}`)))
	if err != nil || result == nil || string(result.JSON) != "[]" || result.LastResponse != nil || len(result.Inventory) != 0 {
		t.Fatal(result, err)
	}
	result, err = service.SearchSubnetRecords(context.Background(), "", network.WithSubnetNetworkAvailable(false))
	if result == nil || !errors.Is(err, resource.ErrInvalidOption) || result.JSON != nil {
		t.Fatal(result, err)
	}
	active, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"subnets":[]}`) })
	result, err = active.SearchSubnetRecords(context.Background(), "", network.WithSubnetNetworkAvailable(false))
	if err != nil || count.Load() != 1 {
		t.Fatal(result, err, count.Load())
	}
	for _, raw := range []string{`"bad"`, `[1]`, `[["name","pair"]]`} {
		before := count.Load()
		_, err := active.ListSubnetRecords(context.Background(), network.WithSubnetReadFiltersJSON(json.RawMessage(raw)))
		if !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
			t.Fatal(raw, err)
		}
	}
}
func TestCloudSubnetListsLiteralPolymorphismNoInventedResourcesOrReceipts(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want string
		fail bool
	}{{"`[]`", "[]", false}, {"`[null,false,0,{}]`", "[null,false,0,{}]", false}, {"`\"가x\"`", `["가","x"]`, false}, {"`{\"z\":1,\"a\":2}`", `["a","z"]`, false}, {"[0]", "", true}, {"[*].id", "", true}, {"length(@)", "", true}} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("unexpected request", r.URL)
			testcloud.JSON(w, 200, `{"subnets":[]}`)
		})
		raw, _ := json.Marshal(map[string]any{"jmespath_filters": tc.expr, "headers": "bad", "base_path": false})
		result, err := service.ListSubnetRecords(context.Background(), network.WithSubnetReadFiltersJSON(raw))
		if (err != nil) != tc.fail || result == nil || result.LastResponse != nil || len(result.Inventory) != 0 || len(result.Subnets) != 0 || count.Load() != 0 {
			t.Fatal(tc, result, err, count.Load())
		}
		if !tc.fail && string(result.JSON) != tc.want {
			t.Fatal(tc, result)
		}
	}
}
func TestCloudSubnetListsNestedGeneratorCarriesLocationAndDelayedGuards(t *testing.T) {
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"subnets":[{"id":"one","project_id":"foreign","availability_zone":false}]}`)
	})
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"current"`)}}
	result, err := service.ListSubnetRecords(context.Background(), network.WithSubnetReadFiltersJSON(json.RawMessage(`{"jmespath_filters":"to_array(@)"}`)), network.WithSubnetReadCurrentLocation(location))
	if err != nil || result.JSON != nil || len(result.Value) != 1 || count.Load() != 0 || len(result.Inventory) != 0 {
		t.Fatal(result, err)
	}
	generator := result.Value[0].(*subnets.RecordListGenerator)
	seen := 0
	for row, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
		seen++
		var loc map[string]json.RawMessage
		_ = json.Unmarshal(row.Resource.Body["location"], &loc)
		if string(loc["zone"]) != "false" {
			t.Fatal(loc)
		}
		var p map[string]json.RawMessage
		_ = json.Unmarshal(loc["project"], &p)
		if string(p["id"]) != `"foreign"` || string(p["name"]) != "null" {
			t.Fatal(p)
		}
	}
	if seen != 1 || count.Load() != 1 {
		t.Fatal(seen, count.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	result, err = service.ListSubnetRecords(ctx, network.WithSubnetReadFiltersJSON(json.RawMessage(`{"jmespath_filters":"to_array(@)","base_path":""}`)))
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	before := count.Load()
	for row, err := range result.Value[0].(*subnets.RecordListGenerator).Records {
		if row != nil || !errors.Is(err, context.Canceled) {
			t.Fatal(row, err)
		}
	}
	if count.Load() != before {
		t.Fatal(count.Load())
	}
	result, err = service.ListSubnetRecords(context.Background(), network.WithSubnetReadFiltersJSON(json.RawMessage(`{"jmespath_filters":"to_array(@)"}`)), network.WithSubnetReadCurrentLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage("[")}}))
	if err != nil {
		t.Fatal(err)
	}
	before = count.Load()
	seenError := false
	for row, err := range result.Value[0].(*subnets.RecordListGenerator).Records {
		var proof *resource.ResponseError
		if row != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "qos-mutation-actual" || string(proof.Body) != `{"subnets":[{"id":"one","project_id":"foreign","availability_zone":false}]}` {
			t.Fatal(row, err, proof)
		}
		seenError = true
	}
	if !seenError || count.Load() != before+1 {
		t.Fatal(seenError, count.Load())
	}

}
func TestCloudSubnetListsSDKLocationRawZoneProjectsAndOwnedEvidence(t *testing.T) {
	for _, zone := range []string{"null", "false", "0", "[]", `{"x":1}`} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, 203, `{"subnets":[{"id":"one","project_id":false,"availability_zone":`+zone+`,"location":{"spoof":true}}]}`)
		})
		name := "configured"
		location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &name}}
		result, err := service.ListSubnetRecords(context.Background(), network.WithSubnetReadCurrentLocation(location))
		if err != nil || len(result.Subnets) != 1 || count.Load() != 1 {
			t.Fatal(result, err)
		}
		var loc map[string]json.RawMessage
		_ = json.Unmarshal(result.Subnets[0].Resource.Body["location"], &loc)
		if string(loc["zone"]) != zone {
			t.Fatal(loc)
		}
		var project map[string]json.RawMessage
		_ = json.Unmarshal(loc["project"], &project)
		if string(project["id"]) != `"current"` || string(project["name"]) != `"configured"` || string(result.Subnets[0].Wire.Body["location"]) != `{"spoof":true}` {
			t.Fatal(result)
		}
		result.Subnets[0].Resource.Body["id"][1] = 'X'
		result.Subnets[0].Envelope[0] = 'X'
		result.Subnets[0].Header.Set("X-Request-Id", "consumer")
		if string(result.Inventory[0].Resource.Body["id"]) != `"one"` || string(result.LastResponse.Resource.Body["id"]) != `"one"` || result.Inventory[0].Header.Get("X-Request-Id") != "qos-mutation-actual" {
			t.Fatal(result)
		}
	}
}
func TestCloudSubnetListsAllPagesCommitOnlyCompleteRowsAndActualFailure(t *testing.T) {
	for _, search := range []bool{false, true} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("marker") == "late" {
				testcloud.JSON(w, 403, `{"error":"late page"}`)
				return
			}
			testcloud.JSON(w, 203, `{"subnets":[{"id":"first"}],"next":"?marker=late"}`)
		})
		result, err := subnetCloudListCall(service, context.Background(), search)
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 403 || result == nil || len(result.Inventory) != 1 || len(result.Subnets) != 0 || result.Value != nil || result.JSON != nil || result.LastResponse.Resource != nil || result.LastResponse.StatusCode != 403 || string(result.LastResponse.Envelope) != `{"error":"late page"}` || result.LastResponse.Header.Get("X-Request-Id") != "qos-mutation-actual" || count.Load() != 2 {
			t.Fatal(result, err, native, count.Load())
		}
	}
}
func TestCloudSubnetListsSnapshotsHeaderNilResetAndSourceMutation(t *testing.T) {
	var observed []string
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		observed = append(observed, r.Header.Get("X-Trace"))
		if r.URL.Query().Get("name") != "owned" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"subnets":[]}`)
	})
	raw := json.RawMessage(`{"name":"owned","headers":{"X-Trace":"filter"}}`)
	opt := network.WithSubnetReadFiltersJSON(raw)
	raw[2] = 'X'
	result, err := service.ListSubnetRecords(context.Background(), opt, network.WithSubnetReadHeaders(nil))
	if err != nil || count.Load() != 1 || !reflect.DeepEqual(observed, []string{""}) {
		t.Fatal(result, err, observed)
	}
	before := count.Load()
	result, err = service.ListSubnetRecords(context.Background(), func(*network.SubnetReadOpts) error { service.API.RawClient().Microversion = "changed"; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
		t.Fatal(result, err, count.Load())
	}
}
func TestCloudSubnetListsPreflightCancellationNilOptionsAndTerminalDescriptor(t *testing.T) {
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 203, `{"subnets":[{"id":"bad","ip_version":"²"}]}`)
	})
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop before list")
	cancel(cause)
	result, err := service.ListSubnetRecords(ctx)
	if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || count.Load() != 0 {
		t.Fatal(result, err)
	}
	result, err = service.ListSubnetRecords(context.Background(), nil)
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 0 {
		t.Fatal(result, err)
	}
	result, err = service.ListSubnetRecords(context.Background())
	var proof *resource.ResponseError
	if result == nil || !errors.As(err, &proof) || proof.StatusCode != 203 || result.LastResponse == nil || result.LastResponse.Resource != nil || result.LastResponse.StatusCode != 203 || len(result.Inventory) != 0 || result.Value != nil || count.Load() != 1 {
		t.Fatal(result, err, proof)
	}
}
