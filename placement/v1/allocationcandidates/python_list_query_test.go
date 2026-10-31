package allocationcandidates_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonCandidateStatusAPI(t *testing.T, calls *[]pythonCandidateCall, status int, body string) *allocationcandidates.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pythonCandidateCall{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("OpenStack-API-Version")})
		return testcloud.Response(status, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = "1.34"
	return allocationcandidates.New(client)
}

// allocation_candidates(resources1="VCPU:1", member_of1=["agg-4", "agg-5"],
// required=["CUSTOM_A", "!CUSTOM_B"], same_subtree=["_A", "_B"])
func TestPythonAllocationCandidatesRepeatedQueryValues(t *testing.T) {
	var calls []pythonCandidateCall
	api := pythonCandidateStatusAPI(t, &calls, http.StatusOK, pythonCandidateBody)
	var pages []*allocationcandidates.AllocationCandidates
	for page, err := range api.List(context.Background(),
		allocationcandidates.WithListOptions(allocationcandidates.ListOpts{
			Required:       []string{"CUSTOM_A", "!CUSTOM_B"},
			ResourceGroups: map[string]allocationcandidates.ResourceGroup{"1": {Resources: "VCPU:1", MemberOf: "agg-old"}},
		}),
		allocationcandidates.WithListQueryValues("member_of1", "agg-4", "agg-5"),
		allocationcandidates.WithListQueryValues("same_subtree", "_A", "_B"),
	) {
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
	}
	if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != "/placement/allocation_candidates" || calls[0].version != "placement 1.34" {
		t.Fatalf("%+v", calls)
	}
	query, err := url.ParseQuery(calls[0].query)
	want := url.Values{"required": {"CUSTOM_A", "!CUSTOM_B"}, "resources1": {"VCPU:1"}, "member_of1": {"agg-4", "agg-5"}, "same_subtree": {"_A", "_B"}}
	if err != nil || !reflect.DeepEqual(query, want) {
		t.Fatal(query, err)
	}
	if len(pages) != 1 || len(pages[0].AllocationRequests) != 2 || len(pages[0].ProviderSummaries) != 3 {
		t.Fatal(pages)
	}
	// A later WithListQuery for the key wins.
	calls = nil
	for range api.List(context.Background(), allocationcandidates.WithListQueryValues("member_of1", "agg-4", "agg-5"), allocationcandidates.WithListQuery("member_of1", "agg-6")) {
	}
	if len(calls) != 1 || calls[0].query != "member_of1=agg-6" {
		t.Fatalf("%+v", calls)
	}
}

func TestPythonAllocationCandidatesRepeatedQueryRejections(t *testing.T) {
	var calls []pythonCandidateCall
	api := pythonCandidateStatusAPI(t, &calls, http.StatusBadRequest, `{"errors":[{"status":400,"title":"Bad Request"}]}`)
	for _, err := range api.List(context.Background(), allocationcandidates.WithListQueryValues(" ", "agg-4")) {
		var operation *resource.OperationError
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "List" {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatalf("%+v", calls)
	}
	seen := false
	for _, err := range api.List(context.Background(), allocationcandidates.WithListQueryValues("member_of1", "agg-4", "agg-5")) {
		seen = true
		if !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) {
			t.Fatal(err)
		}
	}
	if !seen || len(calls) != 1 || calls[0].query != "member_of1=agg-4&member_of1=agg-5" {
		t.Fatalf("%+v", calls)
	}
}
