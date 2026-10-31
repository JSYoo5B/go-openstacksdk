package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Reuse testcloud's public Gophercloud testhelper and the existing native
// transport adapter. No new HTTP server or second fake OpenStack stack.
func cloudRuleTypes(t *testing.T, calls *[]string, reply func(*http.Request, int) *http.Response) *network.Service {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, req.Method+" "+strings.TrimPrefix(req.URL.EscapedPath(), "/neutron/v2.0")+"?"+req.URL.RawQuery)
		response := reply(req, len(*calls))
		response.Request = req
		return response, nil
	}))
	client := cloud.Client("network", "/neutron/v2.0/")
	client.ResourceBase = client.Endpoint
	return network.New(client)
}
func cloudRuleTypeWire(status int, body string) *http.Response {
	return testcloud.Response(status, body, http.Header{"X-Request-Id": {"qos-type-proof"}})
}
func cloudRuleTypeReply(req *http.Request, _ int) *http.Response {
	if strings.HasSuffix(req.URL.Path, "/extensions") {
		return cloudRuleTypeWire(200, `{"extensions":[{"alias":"qos"}]}`)
	}
	return cloudRuleTypeWire(200, `{"rule_types":[{"type":"bandwidth_limit","drivers":null}]}`)
}
func cloudRuleTypeFilters(raw string) network.QoSRuleTypeListOption {
	return network.WithQoSRuleTypeListFilters(json.RawMessage(raw))
}

func TestCloudQoSRuleTypesFreshCompleteGateAndMissingShortCircuit(t *testing.T) {
	var calls []string
	service := cloudRuleTypes(t, &calls, cloudRuleTypeReply)
	for range 2 {
		result, err := service.ListQoSRuleTypeValues(context.Background())
		if err != nil || len(result.RuleTypes) != 1 || len(result.Extensions) != 1 {
			t.Fatal(result, err)
		}
	}
	want := []string{"GET /extensions?", "GET /qos/rule-types?", "GET /extensions?", "GET /qos/rule-types?"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls)
	}
	calls = nil
	missing := cloudRuleTypes(t, &calls, func(*http.Request, int) *http.Response { return cloudRuleTypeWire(200, `{"extensions":[]}`) })
	invoked := false
	result, err := missing.ListQoSRuleTypeValues(context.Background(), func(*network.QoSRuleTypeListOpts) error { invoked = true; return nil }, cloudRuleTypeFilters(`not JSON`))
	if result == nil || !errors.Is(err, network.ErrUnavailableExtension) || invoked || !reflect.DeepEqual(calls, []string{"GET /extensions?"}) {
		t.Fatal(result, err, calls, invoked)
	}
}

func TestCloudQoSRuleTypesGateReadsPastFirstMatchAndKeepsActualFailure(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		var calls []string
		service := cloudRuleTypes(t, &calls, func(req *http.Request, n int) *http.Response {
			if n == 1 {
				if malformed {
					return cloudRuleTypeWire(200, `{"extensions":[{"alias":"qos"},{"alias":{}}]}`)
				}
				return cloudRuleTypeWire(200, `{"extensions":[{"alias":"qos"}],"extensions_links":[{"rel":"next","href":"`+req.URL.Scheme+`://`+req.URL.Host+`/neutron/v2.0/extensions?marker=late"}]}`)
			}
			return cloudRuleTypeWire(403, `{"forbidden":"late-extension-page"}`)
		})
		result, err := service.ListQoSRuleTypeValues(context.Background())
		if err == nil || result == nil || len(result.Extensions) == 0 {
			t.Fatal(result, err, calls)
		}
		if malformed {
			var actual *resource.ResponseError
			if !errors.As(err, &actual) || actual.StatusCode != 200 || string(actual.Body) != `{"extensions":[{"alias":"qos"},{"alias":{}}]}` || actual.Header.Get("X-Request-Id") != "qos-type-proof" || len(calls) != 1 {
				t.Fatal(actual, err, calls)
			}
		} else {
			// A clean rejected HTTP response keeps the native VALUE error;
			// accepted-response projection errors above use ResponseError.
			var actual gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &actual) || actual.Actual != 403 || actual.Method != "GET" || !strings.HasSuffix(actual.URL, "/extensions?marker=late") || string(actual.Body) != `{"forbidden":"late-extension-page"}` || actual.ResponseHeader.Get("X-Request-Id") != "qos-type-proof" || len(calls) != 2 {
				t.Fatal(actual, err, calls)
			}
		}
	}
}

func TestCloudQoSRuleTypesFalseyFiltersAndTruthyShapeAfterGate(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `0.0`, `1e-4000`, `""`, `[]`, `{}`} {
		t.Run(raw, func(t *testing.T) {
			var calls []string
			service := cloudRuleTypes(t, &calls, cloudRuleTypeReply)
			result, err := service.ListQoSRuleTypeValues(context.Background(), cloudRuleTypeFilters(raw))
			if err != nil || result.JSON == nil || len(result.RuleTypes) != 1 || calls[1] != "GET /qos/rule-types?" {
				t.Fatal(result, err, calls)
			}
		})
	}
	for _, raw := range []string{`true`, `1`, `"type"`, `[1]`, `1e4000`, `{`, string([]byte{'"', 0xff, '"'})} {
		var calls []string
		service := cloudRuleTypes(t, &calls, cloudRuleTypeReply)
		result, err := service.ListQoSRuleTypeValues(context.Background(), cloudRuleTypeFilters(raw))
		if err == nil || result == nil || len(calls) != 1 || len(result.Extensions) != 1 || result.JSON != nil {
			t.Fatal(raw, result, err, calls)
		}
	}
}

func TestCloudQoSRuleTypesOriginalKwargsOwnedHeadersVersionPathAndConflictMerge(t *testing.T) {
	var calls []string
	service := cloudRuleTypes(t, &calls, func(req *http.Request, n int) *http.Response {
		if n == 1 {
			return cloudRuleTypeReply(req, n)
		}
		if req.URL.Path != "/neutron/v2.0/qos/type-catalog" || req.URL.Query().Get("type") != "dscp_marking" || req.URL.Query().Get("all_rules") != "False" || len(req.URL.Query()["fields"]) != 2 || req.URL.Query().Has("unknown") || req.Header.Get("X-List-Trace") != "source" || req.Header.Get("OpenStack-API-Version") != "network 2.3" {
			t.Fatal(req.URL, req.Header)
		}
		return cloudRuleTypeWire(203, `{"rule_types":[{"type":"dscp_marking"}]}`)
	})
	raw := json.RawMessage(`{"type":"old","fields":["type","drivers"],"all_rules":false,"base_path":"qos/type-catalog","microversion":"2.3","headers":{"X-List-Trace":"source"},"unknown":{"x":1},"__conflicting_attrs":{"type":"dscp_marking","jmespath_filters":"[0]"}}`)
	option := network.WithQoSRuleTypeListFilters(raw)
	raw[0] = 'x'
	result, err := service.ListQoSRuleTypeValues(context.Background(), option)
	if err != nil || len(result.RuleTypes) != 1 || result.RuleTypes[0].StatusCode != 203 || len(calls) != 2 {
		t.Fatal(result, err, calls)
	}
	for _, raw := range []string{`{"self":null}`, `{"cls":false}`, `{"__conflicting_attrs":{"paginated":false}}`, `{"__conflicting_attrs":{"base_path":"qos/other"}}`} {
		calls = nil
		service := cloudRuleTypes(t, &calls, cloudRuleTypeReply)
		_, err := service.ListQoSRuleTypeValues(context.Background(), cloudRuleTypeFilters(raw))
		if err == nil || len(calls) != 1 {
			t.Fatal(raw, err, calls)
		}
	}
}

func TestCloudQoSRuleTypesGeneratorJMESPathAndPolymorphicListConversion(t *testing.T) {
	cases := []struct {
		expr, want string
		count      int
		bad        bool
	}{
		{"`[1,null,true]`", `[1,null,true]`, 1, false},
		{"`{\"z\":1,\"a\":2}`", `["a","z"]`, 1, false},
		{"'한😀'", `["한","😀"]`, 1, false},
		{"''", `[]`, 1, false},
		{"[0]", "", 1, true}, {"[*].type", "", 1, true}, {"type(@)", "", 1, true}, {"length(@)", "", 1, true}, {"`false`", "", 1, true},
		{"to_array(@)", "", 1, false}, {"@", "", 2, false},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			var calls []string
			service := cloudRuleTypes(t, &calls, cloudRuleTypeReply)
			raw, _ := json.Marshal(map[string]any{"jmespath_filters": tc.expr})
			result, err := service.ListQoSRuleTypeValues(context.Background(), network.WithQoSRuleTypeListFilters(raw))
			if (err != nil) != tc.bad || len(calls) != tc.count {
				t.Fatal(result, err, calls)
			}
			if tc.bad {
				return
			}
			if tc.want != "" && string(result.JSON) != tc.want {
				t.Fatal(string(result.JSON), tc.want)
			}
			if tc.count == 1 && len(result.RuleTypes) != 0 {
				t.Fatal(result)
			}
			if tc.expr == "to_array(@)" {
				generator, ok := result.Values[0].(*ruletypes.ProxyListGenerator)
				if !ok || result.JSON != nil || len(result.Inventory) != 0 {
					t.Fatal(result)
				}
				for record, err := range generator.Records {
					if err != nil || record.Wire == nil {
						t.Fatal(record, err)
					}
				}
				if len(calls) != 2 {
					t.Fatal(calls)
				}
			}
		})
	}
	// Resource.list validates header/base_path values only when the underlying
	// generator is consumed. A constant expression must skip those values.
	var calls []string
	service := cloudRuleTypes(t, &calls, cloudRuleTypeReply)
	result, err := service.ListQoSRuleTypeValues(context.Background(), cloudRuleTypeFilters(`{"headers":42,"base_path":false,"microversion":[],"jmespath_filters":"`+"`[1]`"+`"}`))
	if err != nil || string(result.JSON) != `[1]` || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
}

func TestCloudQoSRuleTypesEmptyDefaultsAliasesAndWireOwnership(t *testing.T) {
	for _, body := range []string{`{"rule_types":[]}`, `{"rule_types":[{"type":"bandwidth_limit","drivers":{"unknown":9007199254740993},"extra":{"x":null}},{"id":null,"type":"dscp_marking"}]}`} {
		var calls []string
		service := cloudRuleTypes(t, &calls, func(req *http.Request, n int) *http.Response {
			if n == 1 {
				return cloudRuleTypeReply(req, n)
			}
			return cloudRuleTypeWire(202, body)
		})
		result, err := service.ListQoSRuleTypeValues(context.Background())
		if err != nil || result.Values == nil || result.RuleTypes == nil || result.JSON == nil {
			t.Fatal(result, err)
		}
		if len(result.RuleTypes) == 0 {
			if string(result.JSON) != `[]` {
				t.Fatal(result)
			}
			continue
		}
		first, second := result.RuleTypes[0], result.RuleTypes[1]
		if string(first.Resource.Body["id"]) != `"bandwidth_limit"` || first.Wire.Body["id"] != nil || string(first.Resource.Body["drivers"]) != `[{"unknown":9007199254740993}]` || string(second.Resource.Body["id"]) != "null" || string(second.Resource.Body["drivers"]) != "null" || first.StatusCode != 202 || string(first.Envelope) != body {
			t.Fatal(result)
		}
		first.Resource.Body["extra"][0] = 'x'
		first.Envelope[0] = 'x'
		first.Header.Set("X-Request-Id", "changed")
		if string(first.Wire.Body["extra"]) != `{"x":null}` || first.Wire.Header.Get("X-Request-Id") != "qos-type-proof" {
			t.Fatal(first)
		}
	}
}

func TestCloudQoSRuleTypesLatePageFailureRetainsInventoryWithoutSuccessValues(t *testing.T) {
	var calls []string
	service := cloudRuleTypes(t, &calls, func(req *http.Request, n int) *http.Response {
		if n == 1 {
			return cloudRuleTypeReply(req, n)
		}
		if n == 2 {
			return cloudRuleTypeWire(200, `{"rule_types":[{"type":"one"}],"rule_types_links":[{"rel":"next","href":"`+req.URL.Scheme+`://`+req.URL.Host+`/neutron/v2.0/qos/rule-types?marker=one"}]}`)
		}
		return cloudRuleTypeWire(403, `{"late":"forbidden"}`)
	})
	result, err := service.ListQoSRuleTypeValues(context.Background())
	var actual gophercloud.ErrUnexpectedResponseCode
	if err == nil || len(result.Inventory) != 1 || result.Values != nil || result.RuleTypes != nil || result.JSON != nil || !errors.As(err, &actual) || actual.Actual != 403 || actual.Method != "GET" || !strings.HasSuffix(actual.URL, "/qos/rule-types?marker=one") || string(actual.Body) != `{"late":"forbidden"}` || actual.ResponseHeader.Get("X-Request-Id") != "qos-type-proof" || len(calls) != 3 {
		t.Fatal(result, err, calls)
	}
}

func TestCloudQoSRuleTypesOptionGuardsStopRestoreAndCancellationCallbacks(t *testing.T) {
	for _, cancelCase := range []bool{false, true} {
		var calls []string
		service := cloudRuleTypes(t, &calls, cloudRuleTypeReply)
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("stop rule types")
		base := service.RawClient().ResourceBase
		later := false
		_, err := service.ListQoSRuleTypeValues(ctx, func(c *network.QoSRuleTypeListOpts) error {
			if cancelCase {
				cancel(cause)
			} else {
				service.RawClient().ResourceBase += "other/"
			}
			return nil
		}, func(*network.QoSRuleTypeListOpts) error {
			later = true
			service.RawClient().ResourceBase = base
			return nil
		})
		cancel(nil)
		if err == nil || later || len(calls) != 1 || (cancelCase && !errors.Is(err, cause)) {
			t.Fatal(err, later, calls)
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("already canceled")
	cancel(cause)
	var calls []string
	service := cloudRuleTypes(t, &calls, cloudRuleTypeReply)
	_, err := service.ListQoSRuleTypeValues(ctx)
	if !errors.Is(err, cause) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

type cloudQoSTypeFaultBody struct {
	body              string
	readErr, closeErr error
	closes            int
	done              bool
}

func (b *cloudQoSTypeFaultBody) Read(p []byte) (int, error) {
	if b.done {
		return 0, io.EOF
	}
	b.done = true
	return copy(p, b.body), b.readErr
}
func (b *cloudQoSTypeFaultBody) Close() error { b.closes++; return b.closeErr }
func TestCloudQoSRuleTypesPhysical401PreservesActualNativeFieldsAndNoHooks(t *testing.T) {
	for _, phase := range []string{"extensions", "qos/rule-types"} {
		var calls []string
		readErr, closeErr := errors.New("type read fault"), errors.New("type close fault")
		body := &cloudQoSTypeFaultBody{body: `{"actual":"partial"}`, readErr: readErr, closeErr: closeErr}
		service := cloudRuleTypes(t, &calls, func(req *http.Request, n int) *http.Response {
			if phase != "extensions" && n == 1 {
				return cloudRuleTypeReply(req, n)
			}
			response := cloudRuleTypeWire(401, "")
			response.Body = body
			return response
		})
		retries, reauths := 0, 0
		service.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		service.RawClient().ReauthFunc = func(context.Context) error { reauths++; return nil }
		_, err := service.ListQoSRuleTypeValues(context.Background())
		var native *gophercloud.ErrUnableToReauthenticate
		if !errors.As(err, &native) {
			t.Fatalf("native 401: %T %v", err, err)
		}
		var actual gophercloud.ErrUnexpectedResponseCode
		if !errors.As(native.ErrOriginal, &actual) || actual.Actual != 401 || actual.Method != "GET" || !strings.HasSuffix(actual.URL, "/"+phase) || string(actual.Body) != body.body || actual.ResponseHeader.Get("X-Request-Id") != "qos-type-proof" || !errors.Is(native.ErrReauth, readErr) || !errors.Is(native.ErrReauth, closeErr) || body.closes != 1 || retries != 0 || reauths != 0 {
			t.Fatal(native, actual, body.closes, retries, reauths)
		}
	}
}

func TestCloudQoSRuleTypesOldStatusAndTypedRawRejectionRemainUnchanged(t *testing.T) {
	var calls []string
	service := cloudRuleTypes(t, &calls, func(req *http.Request, n int) *http.Response {
		if n == 1 {
			return cloudRuleTypeReply(req, n)
		}
		return cloudRuleTypeWire(199, `{"rule_types":[]}`)
	})
	_, err := service.ListQoSRuleTypeValues(context.Background())
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 199 || len(calls) != 2 {
		t.Fatal(err, calls)
	}
	calls = nil
	service = cloudRuleTypes(t, &calls, cloudRuleTypeReply)
	for _, key := range []string{"base_path", "headers", "microversion"} {
		_, err := service.API.QoSRuleTypes.AllRecords(context.Background(), ruletypes.WithRecordListQuery(key, "qos/type-catalog"))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
	for record, err := range service.API.QoSRuleTypes.ListRecords(context.Background(), ruletypes.WithRecordListBasePath("qos/type-catalog")) {
		if err != nil || record == nil {
			t.Fatal(record, err)
		}
	}
	if !reflect.DeepEqual(calls, []string{"GET /qos/type-catalog?"}) {
		t.Fatal(fmt.Sprint(calls))
	}
}
