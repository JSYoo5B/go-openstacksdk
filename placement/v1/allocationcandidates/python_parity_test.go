package allocationcandidates_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates"
)

type pythonCandidateTransport func(*http.Request) (*http.Response, error)

func (transport pythonCandidateTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonCandidateCall struct{ method, path, query, version string }

// AllocationCandidate declares _max_microversion 1.34, so the helper pins that version.
func pythonCandidateAPI(t *testing.T, calls *[]pythonCandidateCall, body string) *allocationcandidates.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonCandidateTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pythonCandidateCall{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("OpenStack-API-Version")})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = "1.34"
	return allocationcandidates.New(client)
}

const pythonCandidateBody = `{"allocation_requests":[` +
	`{"allocations":{"rp-1":{"resources":{"VCPU":1}},"rp-3":{"resources":{"DISK_GB":10}}},"mappings":{"":["rp-1"],"1":["rp-3"]}},` +
	`{"allocations":{"rp-2":{"resources":{"VCPU":1}},"rp-3":{"resources":{"DISK_GB":10}}},"mappings":{"":["rp-2"],"1":["rp-3"]}}],` +
	`"provider_summaries":{"rp-1":{"resources":{"VCPU":{"capacity":8,"used":2}},"traits":["CUSTOM_A"],"parent_provider_uuid":null,"root_provider_uuid":"rp-1"},` +
	`"rp-2":{"resources":{"VCPU":{"capacity":4,"used":0}},"traits":[],"parent_provider_uuid":null,"root_provider_uuid":"rp-2"},` +
	`"rp-3":{"resources":{"DISK_GB":{"capacity":100,"used":0}},"traits":[],"parent_provider_uuid":null,"root_provider_uuid":"rp-3"}}}`

func TestPythonAllocationCandidatesQueryAndCandidates(t *testing.T) {
	var calls []pythonCandidateCall
	api := pythonCandidateAPI(t, &calls, pythonCandidateBody)
	// allocation_candidates(resources="VCPU:1", required=["CUSTOM_A", "!CUSTOM_B"], member_of=["agg-1", "in:agg-2,agg-3"],
	//     limit=2, group_policy="isolate", resources1="DISK_GB:10", required1="CUSTOM_SSD", member_of1="agg-4", in_tree1="rp-3")
	var pages []*allocationcandidates.AllocationCandidates
	for page, err := range api.List(context.Background(), allocationcandidates.WithListOptions(allocationcandidates.ListOpts{
		Resources: "VCPU:1", Required: []string{"CUSTOM_A", "!CUSTOM_B"}, MemberOf: []string{"agg-1", "in:agg-2,agg-3"}, Limit: 2, GroupPolicy: "isolate",
		ResourceGroups: map[string]allocationcandidates.ResourceGroup{"1": {Resources: "DISK_GB:10", Required: []string{"CUSTOM_SSD"}, MemberOf: "agg-4", InTree: "rp-3"}},
	})) {
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
	}
	if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != "/placement/allocation_candidates" || calls[0].version != "placement 1.34" {
		t.Fatalf("%+v", calls)
	}
	query, err := url.ParseQuery(calls[0].query)
	wantQuery := url.Values{
		"resources": {"VCPU:1"}, "required": {"CUSTOM_A", "!CUSTOM_B"}, "member_of": {"agg-1", "in:agg-2,agg-3"}, "limit": {"2"}, "group_policy": {"isolate"},
		"resources1": {"DISK_GB:10"}, "required1": {"CUSTOM_SSD"}, "member_of1": {"agg-4"}, "in_tree1": {"rp-3"},
	}
	if err != nil || !reflect.DeepEqual(query, wantQuery) {
		t.Fatal(query, err)
	}
	// Go yields the whole response once; Python yields one candidate per allocation request
	// with provider_summaries reduced to that candidate's providers.
	if len(pages) != 1 || len(pages[0].AllocationRequests) != 2 {
		t.Fatal(pages)
	}
	page := pages[0]
	for i, wantProviders := range [][]string{{"rp-1", "rp-3"}, {"rp-2", "rp-3"}} {
		request := page.AllocationRequests[i]
		var providers []string
		for _, provider := range wantProviders {
			if _, ok := request.Allocations[provider]; !ok {
				t.Fatal(i, provider, request.Allocations)
			}
			if _, ok := page.ProviderSummaries[provider]; !ok {
				t.Fatal(provider)
			}
			providers = append(providers, provider)
		}
		if len(request.Allocations) != len(providers) || request.Mappings == nil || (*request.Mappings)["1"][0] != "rp-3" {
			t.Fatal(i, request)
		}
	}
	summary := page.ProviderSummaries["rp-1"]
	if summary.Resources["VCPU"].Capacity != 8 || summary.Resources["VCPU"].Used != 2 || !reflect.DeepEqual(*summary.Traits, []string{"CUSTOM_A"}) || *summary.RootProviderUUID != "rp-1" {
		t.Fatal(summary)
	}
}

// Python forwards member_of1=["agg-4", "agg-5"] as a repeated key.
// Go ResourceGroup.MemberOf holds one value and WithListQuery replaces rather than appends.
func TestPythonAllocationCandidatesSuffixedMemberOfIsSingle(t *testing.T) {
	var calls []pythonCandidateCall
	api := pythonCandidateAPI(t, &calls, `{"allocation_requests":[],"provider_summaries":{}}`)
	for _, err := range api.List(context.Background(), allocationcandidates.WithListOptions(allocationcandidates.ListOpts{
		ResourceGroups: map[string]allocationcandidates.ResourceGroup{"1": {Resources: "VCPU:1", MemberOf: "agg-4"}},
	}), allocationcandidates.WithListQuery("member_of1", "agg-5")) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(calls) != 1 || calls[0].query != "member_of1=agg-5&resources1=VCPU%3A1" {
		t.Fatalf("%+v", calls)
	}
}
