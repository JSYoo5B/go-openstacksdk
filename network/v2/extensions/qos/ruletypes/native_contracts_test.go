package ruletypes_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRuleTypeTransport func(*http.Request) (*http.Response, error)

func (transport nativeRuleTypeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeRuleTypeCall struct{ method, path, query string }

func nativeRuleTypeAPI(t *testing.T, calls *[]nativeRuleTypeCall, code int, body string) *ruletypes.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeRuleTypeTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeRuleTypeCall{req.Method, req.URL.Path, req.URL.RawQuery})
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return ruletypes.New(client)
}

const nativeRuleTypeRow = `{"type":"bandwidth_limit","drivers":[{"name":"ovs","supported_parameters":[{"parameter_name":"max_kbps","parameter_type":"range","parameter_values":{"start":0,"end":2147483647}},{"parameter_name":"direction","parameter_type":"choices","parameter_values":["ingress","egress"]}]}]}`

func TestNativeQoSRuleTypeGetAndList(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRuleTypeCall
	api := nativeRuleTypeAPI(t, &calls, 200, `{"rule_type":`+nativeRuleTypeRow+`}`)
	got, err := api.GetRuleType(ctx, "bandwidth_limit")
	if err != nil || got.Type != "bandwidth_limit" || got.Drivers[0].Name != "ovs" {
		t.Fatal(got, err)
	}
	// parameter_values keeps its JSON shape as an untyped value.
	params := got.Drivers[0].SupportedParameters
	if !reflect.DeepEqual(params[0].ParameterValues, map[string]any{"start": float64(0), "end": float64(2147483647)}) || !reflect.DeepEqual(params[1].ParameterValues, []any{"ingress", "egress"}) {
		t.Fatal(params)
	}
	// The listing is a single page; links are not followed.
	listAPI := nativeRuleTypeAPI(t, &calls, 200, `{"rule_types":[{"type":"bandwidth_limit"},{"type":"dscp_marking","drivers":null}],"rule_types_links":[{"rel":"next","href":"/never"}],"links":{"next":"/never"}}`)
	var types []string
	for value, err := range listAPI.ListRuleTypes(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		types = append(types, value.Type)
	}
	want := []nativeRuleTypeCall{
		{http.MethodGet, "/neutron/v2.0/qos/rule-types/bandwidth_limit", ""},
		{http.MethodGet, "/neutron/v2.0/qos/rule-types", ""},
	}
	if !reflect.DeepEqual(types, []string{"bandwidth_limit", "dscp_marking"}) || !reflect.DeepEqual(calls, want) {
		t.Fatal(types, calls)
	}
}

func TestNativeQoSRuleTypeStatusesAndDecode(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 202, 204, 404} {
		var calls []nativeRuleTypeCall
		_, err := nativeRuleTypeAPI(t, &calls, code, `{"rule_type":{}}`).GetRuleType(ctx, "x")
		var wrapped *resource.OperationError
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &wrapped) || wrapped.Operation != "GetRuleType" || wrapped.Resource != "ruletypes" || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
			t.Fatal(code, err)
		}
	}
	for _, body := range []string{`{"rule_type":[]}`, `{`} {
		var calls []nativeRuleTypeCall
		_, err := nativeRuleTypeAPI(t, &calls, 200, body).GetRuleType(ctx, "x")
		var wrapped *resource.OperationError
		if !errors.As(err, &wrapped) || wrapped.Operation != "GetRuleType" {
			t.Fatal(body, err)
		}
	}
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"rule_types":[]}`}, {200, `{}`}, {204, ""}} {
		var calls []nativeRuleTypeCall
		var errs []error
		count := 0
		for value, err := range nativeRuleTypeAPI(t, &calls, tc.code, tc.body).ListRuleTypes(ctx) {
			if err != nil {
				errs = append(errs, err)
			} else if value != nil {
				count++
			}
		}
		var native gophercloud.ErrUnexpectedResponseCode
		switch {
		case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
		case tc.code == 200 && len(errs) == 0 && count == 0:
		case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
		default:
			t.Fatal(tc.code, tc.body, errs, count)
		}
	}
}
