package allocationcandidates_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeACTransport func(*http.Request) (*http.Response, error)

func (transport nativeACTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeACWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeACCall struct{ method, path, query, body, version string }

func nativeACAPI(t *testing.T, microversion string, calls *[]nativeACCall, reply func(*http.Request) *http.Response) *allocationcandidates.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeACTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeACCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = microversion
	return allocationcandidates.New(client)
}

func nativeACOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "allocationcandidates" {
		t.Fatal("generated allocationcandidates context", err, wrapped)
	}
}

func nativeACCollect(api *allocationcandidates.API, options ...allocationcandidates.ListOption) ([]*allocationcandidates.AllocationCandidates110, []error) {
	var values []*allocationcandidates.AllocationCandidates110
	var errs []error
	for value, err := range api.List(context.Background(), options...) {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		values = append(values, value)
	}
	return values, errs
}

const nativeAC110Body = `{"allocation_requests":[{"allocations":[{"resource_provider":{"uuid":"rp-1"},"resources":{"VCPU":1,"MEMORY_MB":512}}]}],"provider_summaries":{"rp-1":{"resources":{"VCPU":{"capacity":8,"used":2}}}}}`

func TestNativeAllocationCandidatesQueryAndDecode(t *testing.T) {
	var calls []nativeACCall
	api := nativeACAPI(t, "1.10", &calls, func(*http.Request) *http.Response { return nativeACWire(200, nativeAC110Body) })
	values, errs := nativeACCollect(api, allocationcandidates.WithListOptions(allocationcandidates.ListOpts{
		Resources:    "VCPU:1,MEMORY_MB:512",
		Required:     []string{"CUSTOM_A", "in:HW_CPU_X86_AVX,!CUSTOM_B"},
		MemberOf:     []string{"in:agg-1,agg-2", "!agg-3"},
		InTree:       "root-1",
		GroupPolicy:  "isolate",
		Limit:        5,
		RootRequired: "CUSTOM_ROOT",
		SameSubtree:  []string{"_NIC", "_PORT"},
		ResourceGroups: map[string]allocationcandidates.ResourceGroup{
			"_NIC": {Resources: "NET_BW_EGR_KILOBIT_PER_SEC:10", Required: []string{"CUSTOM_PHYSNET", ""}, MemberOf: "agg-4", InTree: "nic-root"},
		},
	}), allocationcandidates.WithListQuery("extra", "1"))
	// The 1.10-1.11 shape is the only one the facade decodes; one page yields one value.
	if len(errs) != 0 || len(values) != 1 || values[0].AllocationRequests[0].Allocations[0].ResourceProvider.UUID != "rp-1" ||
		values[0].AllocationRequests[0].Allocations[0].Resources["MEMORY_MB"] != 512 || values[0].ProviderSummaries["rp-1"].Resources["VCPU"].Capacity != 8 {
		t.Fatal(values, errs)
	}
	// Lists are repeated keys, group keys get the suffix, empty group required entries are skipped, and the whole query is percent-encoded.
	query := "extra=1&group_policy=isolate&in_tree=root-1&in_tree_NIC=nic-root&limit=5" +
		"&member_of=in%3Aagg-1%2Cagg-2&member_of=%21agg-3&member_of_NIC=agg-4" +
		"&required=CUSTOM_A&required=in%3AHW_CPU_X86_AVX%2C%21CUSTOM_B&required_NIC=CUSTOM_PHYSNET" +
		"&resources=VCPU%3A1%2CMEMORY_MB%3A512&resources_NIC=NET_BW_EGR_KILOBIT_PER_SEC%3A10" +
		"&root_required=CUSTOM_ROOT&same_subtree=_NIC&same_subtree=_PORT"
	want := []nativeACCall{{http.MethodGet, "/placement/allocation_candidates", query, "", "placement 1.10"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	t.Run("1.12 dictionary allocations fail the generated decode", func(t *testing.T) {
		var calls []nativeACCall
		api := nativeACAPI(t, "1.39", &calls, func(*http.Request) *http.Response {
			return nativeACWire(200, `{"allocation_requests":[{"allocations":{"rp-1":{"resources":{"VCPU":1}}},"mappings":{"":["rp-1"]}}],"provider_summaries":{"rp-1":{"resources":{"VCPU":{"capacity":8,"used":0}},"traits":[],"parent_provider_uuid":null,"root_provider_uuid":"rp-1"}}}`)
		})
		// The generated List uses ExtractAllocationCandidates110 instead of the 1.12+ ExtractAllocationCandidates.
		values, errs := nativeACCollect(api, allocationcandidates.WithListOptions(allocationcandidates.ListOpts{Resources: "VCPU:1"}))
		if len(values) != 0 || len(errs) != 1 || len(calls) != 1 || calls[0].version != "placement 1.39" {
			t.Fatal(values, errs, calls)
		}
		var typeErr *json.UnmarshalTypeError
		if !errors.As(errs[0], &typeErr) || typeErr.Value != "object" {
			t.Fatal(errs[0])
		}
	})
}

func TestNativeAllocationCandidatesStatusesAndPreflight(t *testing.T) {
	t.Run("status, empty candidates and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {201, `{}`}, {200, `{"allocation_requests":[],"provider_summaries":{}}`}, {200, `{}`}, {204, ""}} {
			var calls []nativeACCall
			api := nativeACAPI(t, "1.10", &calls, func(*http.Request) *http.Response { return nativeACWire(tc.code, tc.body) })
			values, errs := nativeACCollect(api)
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			// The pager error is not wrapped with operation context.
			case (tc.code == 404 || tc.code == 201) && len(errs) == 1 && errors.As(errs[0], &native) && native.Actual == tc.code && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			// No candidates means an empty page, so the stream yields nothing rather than an empty value.
			case tc.code == 200 && len(errs) == 0 && len(values) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, tc.body, values, errs)
			}
			if len(calls) != 1 || calls[0].query != "" {
				t.Fatal(calls)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeACCall
		api := nativeACAPI(t, "1.10", &calls, func(*http.Request) *http.Response { return nativeACWire(200, nativeAC110Body) })
		for name, options := range map[string][]allocationcandidates.ListOption{
			"empty query key": {allocationcandidates.WithListQuery("", "x")},
			"nil option":      {nil},
		} {
			values, errs := nativeACCollect(api, options...)
			if len(values) != 0 || len(errs) != 1 {
				t.Fatal(name, values, errs)
			}
			nativeACOperation(t, errs[0], "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
