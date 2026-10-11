package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonPacketRateField(t *testing.T, value *rules.PacketRateLimitRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s: want %s, record=%#v", key, want, value)
	}
}
func pythonPacketRateAPI(t *testing.T, calls *[]nativeRuleCall, reply func(*http.Request) *http.Response) *rules.API {
	t.Helper()
	api, _ := nativeRuleAPI(t, calls, reply)
	return api
}
func pythonPacketRateRead(ctx context.Context, api *rules.API, op string) (*rules.PacketRateLimitRecord, error) {
	switch op {
	case "get":
		return api.GetPacketRateLimitRecord(ctx, "parent", "target")
	case "find":
		return api.FindPacketRateLimitRecord(ctx, "parent", "target")
	default:
		values, err := api.AllPacketRateLimitRecords(ctx, "parent")
		if len(values) > 0 {
			return values[0], err
		}
		return nil, err
	}
}

func TestPythonPacketRateLimitMemberLiteralRoutesSourceQueryAndURIView(t *testing.T) {
	for _, op := range []string{"get", "find"} {
		for _, includedParent := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/parent-in-body=%t", op, includedParent), func(t *testing.T) {
				var calls []nativeRuleCall
				body := `{"packet_rate_limit_rule":{"max_kpps":"not-an-int","max_burst_kpps":false,"direction":0,"x_new":9007199254740993,"self":"raw"}}`
				if includedParent {
					body = `{"packet_rate_limit_rule":{"qos_policy_id":"foreign","max_kpps":"not-an-int","max_burst_kpps":false,"direction":0,"x_new":9007199254740993,"self":"raw"}}`
				}
				parent, identity := "parent/id %2F", "rule/id %2F"
				api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
					if req.URL.EscapedPath() != "/neutron/v2.0/qos/policies/parent%2Fid%20%252F/packet_rate_limit_rules/rule%2Fid%20%252F" {
						t.Fatal(req.URL)
					}
					want := url.Values{}
					if op == "find" {
						want.Set("qos_policy_id", parent)
					}
					if !reflect.DeepEqual(req.URL.Query(), want) {
						t.Fatal(req.URL.Query(), want)
					}
					response := nativeRuleWire(203, body)
					response.Header.Set("If-Match", "actual-header")
					response.Header.Set("X-Request-Id", "receipt")
					return response
				})
				var value *rules.PacketRateLimitRecord
				var err error
				if op == "get" {
					value, err = api.GetPacketRateLimitRecord(context.Background(), parent, identity)
				} else {
					value, err = api.FindPacketRateLimitRecord(context.Background(), parent, identity)
				}
				if err != nil || value == nil || value.QoSPolicyID != parent || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "receipt" || len(calls) != 1 || calls[0].body != "" {
					t.Fatal(value, err, calls)
				}
				pythonPacketRateField(t, value, "id", `"rule/id %2F"`)
				pythonPacketRateField(t, value, "name", `null`)
				pythonPacketRateField(t, value, "max_kpps", `"not-an-int"`)
				pythonPacketRateField(t, value, "max_burst_kpps", `false`)
				pythonPacketRateField(t, value, "direction", `0`)
				pythonPacketRateField(t, value, "x_new", `9007199254740993`)
				pythonPacketRateField(t, value, "location", `null`)
				if value.Resource.Body["tags"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["self"] != nil {
					t.Fatal(value.Resource.Body)
				}
				if includedParent {
					pythonPacketRateField(t, value, "qos_policy_id", `"parent/id %2F"`)
					if string(value.Wire.Body["qos_policy_id"]) != `"foreign"` {
						t.Fatal(value.Wire)
					}
				} else if value.Resource.Body["qos_policy_id"] != nil {
					t.Fatal("invented Body URI field", value.Resource.Body)
				}
				value.Resource.Body["x_new"][0] = '1'
				value.Resource.Header.Set("X-Request-Id", "changed")
				value.Header.Set("X-Request-Id", "changed")
				if string(value.Wire.Body["x_new"]) != `9007199254740993` || value.Wire.Header.Get("X-Request-Id") != "receipt" || string(value.Envelope) != body {
					t.Fatal("aliased evidence", value)
				}
			})
		}
	}
}

func TestPythonPacketRateLimitGetNoFallbackAndFindMissingCompleteSearch(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		var calls []nativeRuleCall
		api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/target") {
				return nativeRuleWire(code, `{"error":"member hidden"}`)
			}
			if len(req.URL.Query()) != 0 {
				t.Fatal("no parent query or name hint on LIST", req.URL)
			}
			return nativeRuleWire(200, `{"packet_rate_limit_rules":[]}`)
		})
		value, err := api.GetPacketRateLimitRecord(context.Background(), "parent", "target")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != code || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		calls = nil
		value, err = api.FindPacketRateLimitRecord(context.Background(), "parent", "target")
		if value != nil || err != nil || len(calls) != 2 {
			t.Fatal(value, err, calls)
		}
		calls = nil
		value, err = api.FindPacketRateLimitRecord(context.Background(), "parent", "target", rules.WithPacketRateLimitFindIgnoreMissing(false))
		if value != nil || !errors.Is(err, resource.ErrNotFound) || len(calls) != 2 {
			t.Fatal(value, err, calls)
		}
	}
}

func TestPythonPacketRateLimitFindBodySeedsAndListLocalFilters(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			want := url.Values{"qos_policy_id": {"parent"}, "name": {"target"}, "max_kpps": {"True"}, "max_burst_kpps": {"nested"}, "direction": {"egress"}, "x_seed": {"keep"}, "fields": {"id", "direction"}}
			if !reflect.DeepEqual(req.URL.Query(), want) {
				t.Fatal(req.URL.Query(), want)
			}
			return nativeRuleWire(404, "")
		}
		if len(req.URL.Query()) != 0 {
			t.Fatal("Body fields are local; fields unknown is discarded", req.URL)
		}
		return nativeRuleWire(200, `{"packet_rate_limit_rules":[{"id":"no","name":"target","max_kpps":0,"direction":"egress","max_burst_kpps":{"nested":null}},{"id":"yes","name":"target","max_kpps":1,"direction":"egress","max_burst_kpps":{"nested":null},"qos_policy_id":"foreign"}]}`)
	})
	value, err := api.FindPacketRateLimitRecord(context.Background(), "parent", "target", rules.WithPacketRateLimitFindQuery("name", "target"), rules.WithPacketRateLimitFindQuery("max_kpps", true), rules.WithPacketRateLimitFindQuery("max_burst_kpps", map[string]any{"nested": nil}), rules.WithPacketRateLimitFindQuery("direction", "egress"), rules.WithPacketRateLimitFindQuery("x_seed", "keep"), rules.WithPacketRateLimitFindQuery("fields", []string{"id", "direction"}))
	if err != nil || value == nil || len(calls) != 2 || value.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(value, err, calls)
	}
	pythonPacketRateField(t, value, "id", `"yes"`)
	if string(value.Wire.Body["qos_policy_id"]) != `"foreign"` || value.QoSPolicyID != "parent" {
		t.Fatal(value)
	}
	calls = nil
	seeded := pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(204, "") })
	value, err = seeded.FindPacketRateLimitRecord(context.Background(), "parent", "target", rules.WithPacketRateLimitFindQuery("max_kpps", "untyped"), rules.WithPacketRateLimitFindQuery("x_seed", false))
	if err != nil || value.Wire != nil || value.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(value, err)
	}
	pythonPacketRateField(t, value, "max_kpps", `"untyped"`)
	pythonPacketRateField(t, value, "x_seed", `false`)
}

func TestPythonPacketRateLimitFindLaterDuplicateAndListFailureRemainTerminal(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		var calls []nativeRuleCall
		api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			switch len(calls) {
			case 1:
				return nativeRuleWire(404, "")
			case 2:
				return nativeRuleWire(200, `{"packet_rate_limit_rules":[{"id":"first","name":"target"}],"packet_rate_limit_rules_links":[{"rel":"next","href":"?marker=first"}]}`)
			default:
				if req.URL.Query().Get("marker") != "first" {
					t.Fatal(req.URL)
				}
				if duplicate {
					return nativeRuleWire(200, `{"packet_rate_limit_rules":[{"id":"later","name":"target"}]}`)
				}
				return nativeRuleWire(403, `{"error":"rule access denied despite shared policy"}`)
			}
		})
		value, err := api.FindPacketRateLimitRecord(context.Background(), "parent", "target")
		if value != nil || err == nil || len(calls) != 3 {
			t.Fatal(value, err, calls)
		}
		if duplicate {
			if !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 403 || !strings.Contains(string(native.Body), "shared policy") {
				t.Fatal(err, native)
			}
		}
	}
}

func TestPythonPacketRateLimitListLazyQueryLocalFiltersAndUnknowns(t *testing.T) {
	var calls []nativeRuleCall
	body := `{"packet_rate_limit_rules":[{"id":"wrong","name":"demo","max_kpps":0,"direction":"egress"},{"id":"right","name":"demo","max_kpps":1,"direction":"egress","max_burst_kpps":"raw","qos_policy_id":"foreign","tags":false,"x_ext":9007199254740993,"self":"wire"}]}`
	api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(req.URL.Query()) != 0 || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header)
		}
		response := nativeRuleWire(200, body)
		response.Header.Set("If-Match", "HTTP-only")
		return response
	})
	seq := api.ListPacketRateLimitRecords(context.Background(), "parent", rules.WithPacketRateLimitListQuery("name", "demo"), rules.WithPacketRateLimitListQuery("max_kpps", true), rules.WithPacketRateLimitListQuery("direction", "egress"), rules.WithPacketRateLimitListQuery("fields", func() {}), rules.WithPacketRateLimitListQuery("tags", func() {}), rules.WithPacketRateLimitListOpts(rules.PacketRateLimitListOpts{Query: map[string]json.RawMessage{"name": json.RawMessage(`"demo"`), "max_kpps": json.RawMessage(`true`), "direction": json.RawMessage(`"egress"`), "ignored": json.RawMessage(`[`)}}))
	if len(calls) != 0 {
		t.Fatal("eager list", calls)
	}
	var values []*rules.PacketRateLimitRecord
	for value, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, calls)
	}
	value := values[0]
	pythonPacketRateField(t, value, "id", `"right"`)
	pythonPacketRateField(t, value, "max_burst_kpps", `"raw"`)
	pythonPacketRateField(t, value, "tags", `false`)
	if value.Resource.Body["qos_policy_id"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["self"] != nil || value.QoSPolicyID != "parent" || string(value.Wire.Body["qos_policy_id"]) != `"foreign"` || string(value.Envelope) != body {
		t.Fatal(value)
	}
}

func TestPythonPacketRateLimitListRawCapSinglePageAndEarlyStop(t *testing.T) {
	for _, mode := range []string{"cap", "single", "break"} {
		var calls []nativeRuleCall
		api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			if mode == "cap" && req.URL.Query().Get("limit") != "1" {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, `{"packet_rate_limit_rules":[{"id":"filtered","direction":"ingress"},{"id":"kept","direction":"egress"}],"packet_rate_limit_rules_links":[{"rel":"next","href":"?marker=kept"}]}`)
		})
		opts := []rules.PacketRateLimitListOption{}
		if mode == "cap" {
			opts = append(opts, rules.WithPacketRateLimitListLimit(0), rules.WithPacketRateLimitListMaxItems(1), rules.WithPacketRateLimitListQuery("direction", "egress"))
		}
		if mode == "single" {
			opts = append(opts, rules.WithPacketRateLimitListPaginated(false))
		}
		count := 0
		for value, err := range api.ListPacketRateLimitRecords(context.Background(), "parent", opts...) {
			if err != nil || value == nil {
				t.Fatal(value, err)
			}
			count++
			if mode == "break" {
				break
			}
		}
		want := 2
		if mode == "cap" {
			want = 0
		}
		if mode == "break" {
			want = 1
		}
		if count != want || len(calls) != 1 {
			t.Fatal(mode, count, calls)
		}
	}
}

func TestPythonPacketRateLimitListContinuationCannotChangeParentOrFilters(t *testing.T) {
	for _, href := range []string{"?marker=one", "/v2.0/qos/policies/parent/packet_rate_limit_rules?marker=one", "/v2.0/qos/policies/foreign/packet_rate_limit_rules?marker=one", "?marker=one&limit=9", "https://foreign.invalid/qos/policies/parent/packet_rate_limit_rules"} {
		var calls []nativeRuleCall
		api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeRuleWire(200, fmt.Sprintf(`{"packet_rate_limit_rules":[{"id":"one"}],"next":%q}`, href))
			}
			if req.URL.Path != "/neutron/v2.0/qos/policies/parent/packet_rate_limit_rules" || req.URL.Query().Get("marker") != "one" {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, `{"packet_rate_limit_rules":[]}`)
		})
		values, err := api.AllPacketRateLimitRecords(context.Background(), "parent", rules.WithPacketRateLimitListLimit(1))
		safe := href == "?marker=one" || strings.HasPrefix(href, "/v2.0/qos/policies/parent/")
		if safe {
			if err != nil || len(values) != 1 || len(calls) != 2 {
				t.Fatal(href, values, err, calls)
			}
		} else {
			var proof *resource.ResponseError
			if values != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || len(calls) != 1 {
				t.Fatal(href, values, err, calls)
			}
		}
	}
}

func TestPythonPacketRateLimitListMarkerFallbackAndFalseyNumericNext(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRuleWire(200, `{"packet_rate_limit_rules":[{"id":"one"}]}`)
		}
		if req.URL.Query().Get("marker") != "one" || req.URL.Query().Get("limit") != "3" {
			t.Fatal(req.URL)
		}
		return nativeRuleWire(200, `{"packet_rate_limit_rules":[]}`)
	})
	values, err := api.AllPacketRateLimitRecords(context.Background(), "parent", rules.WithPacketRateLimitListLimit(3))
	if err != nil || len(values) != 1 || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
	for _, token := range []string{"1e-4000", "0e100", "1"} {
		calls = nil
		api = pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeRuleWire(200, `{"packet_rate_limit_rules":[{"id":"one"}],"next":`+token+`}`)
		})
		values, err = api.AllPacketRateLimitRecords(context.Background(), "parent")
		if token == "1" {
			var proof *resource.ResponseError
			if values != nil || !errors.As(err, &proof) {
				t.Fatal(values, err)
			}
		} else if err != nil || len(values) != 1 {
			t.Fatal(token, values, err)
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
}

func TestPythonPacketRateLimitAllStatusAndMalformedReceiptBoundaries(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		var calls []nativeRuleCall
		status := 200
		body := `{"packet_rate_limit_rule":{}}`
		if op == "list" {
			body = `{"packet_rate_limit_rules":[]}`
		}
		api := pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeRuleWire(status, body)
			response.Header.Set("X-Request-Id", "status-proof")
			return response
		})
		for _, code := range []int{200, 203, 299, 302, 399} {
			status = code
			calls = nil
			_, err := pythonPacketRateRead(context.Background(), api, op)
			if err != nil || len(calls) != 1 {
				t.Fatal(op, code, err, calls)
			}
		}
		for code := 400; code <= 599; code++ {
			status = code
			calls = nil
			if op == "find" && (code == 400 || code == 403 || code == 404) {
				continue
			}
			_, err := pythonPacketRateRead(context.Background(), api, op)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, code, err, native, calls)
			}
		}
		status = 200
		for _, bad := range []string{`null`, `[]`, `{"packet_rate_limit_rule":null}`, `{"packet_rate_limit_rule":false}`} {
			if op == "list" {
				bad = `{"packet_rate_limit_rules":null}`
			}
			body = bad
			calls = nil
			_, err := pythonPacketRateRead(context.Background(), api, op)
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || string(proof.Body) != bad || proof.Header.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, bad, err, proof, calls)
			}
		}
	}

	// Successful member JSON parsing failures keep the constructor view;
	// this is distinct from a valid JSON representation of the wrong shape.
	for _, op := range []string{"get", "find"} {
		for _, raw := range []string{"", "{", "not-json"} {
			var calls []nativeRuleCall
			status := 203
			if raw == "" {
				status = 204
			}
			api := pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
				wire := nativeRuleWire(status, raw)
				wire.Header.Set("X-Request-Id", "parse-default-proof")
				return wire
			})
			var value *rules.PacketRateLimitRecord
			var err error
			if op == "get" {
				value, err = api.GetPacketRateLimitRecord(context.Background(), "parent", "target")
			} else {
				value, err = api.FindPacketRateLimitRecord(context.Background(), "parent", "target", rules.WithPacketRateLimitFindQuery("max_kpps", false), rules.WithPacketRateLimitFindQuery("x_seed", "seed"))
			}
			if err != nil || value == nil || value.Resource == nil || value.Wire != nil || value.StatusCode != status || string(value.Envelope) != raw || value.Header.Get("X-Request-Id") != "parse-default-proof" || value.Resource.StatusCode != status || len(calls) != 1 {
				t.Fatal(op, raw, value, err, calls)
			}
			pythonPacketRateField(t, value, "id", `"target"`)
			expected := `null`
			if op == "find" {
				expected = `false`
				pythonPacketRateField(t, value, "x_seed", `"seed"`)
			}
			pythonPacketRateField(t, value, "max_kpps", expected)
		}
	}
}

func TestPythonPacketRateLimitPhysicalReadCloseCancelAndSourceProof(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, status := range []int{200, 404, 500} {
			for _, fault := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", op, status, fault), func(t *testing.T) {
					read, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel in Close")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					var calls []nativeRuleCall
					var client *gophercloud.ServiceClient
					body := `{"packet_rate_limit_rule":{"id":"target"}}`
					if op == "list" {
						body = `{"packet_rate_limit_rules":[{"id":"target"}]}`
					}
					physical := &pythonBandwidthFaultBody{data: body}
					switch fault {
					case "read-close":
						physical.read, physical.close = read, closeCause
					case "cancel":
						physical.onClose = func() { cancel(cancelCause) }
					case "source":
						physical.onClose = func() { client.ResourceBase += "changed/" }
					}
					api := pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
						r := nativeRuleWire(status, body)
						r.Body = physical
						r.Header.Set("X-Request-Id", "physical-proof")
						return r
					})
					client = api.RawClient()
					_, err := pythonPacketRateRead(ctx, api, op)
					if err == nil || len(calls) != 1 || physical.closes != 1 {
						t.Fatal(err, calls, physical.closes)
					}
					switch fault {
					case "read-close":
						if !errors.Is(err, read) || !errors.Is(err, closeCause) {
							t.Fatal(err)
						}
					case "cancel":
						if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
							t.Fatal(err)
						}
					case "source":
						if !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal(err)
						}
					}
					if status < 400 {
						var proof *resource.ResponseError
						if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "physical-proof" {
							t.Fatal(err, proof)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "physical-proof" {
							t.Fatal(err, native)
						}
					}
				})
			}
		}
	}
}

func TestPythonPacketRateLimitOptionSnapshotsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`"owned"`)
	query := map[string]json.RawMessage{"x_seed": raw}
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.0"
	option := rules.WithPacketRateLimitFindOpts(rules.PacketRateLimitFindOpts{Query: query, Headers: headers, Microversion: &version})
	raw[1] = 'x'
	headers["X-Trace"] = "changed"
	version = "changed"
	var escaped *rules.PacketRateLimitFindOpts
	var calls []nativeRuleCall
	api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("x_seed") != "owned" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.0" {
			t.Fatal(req.URL, req.Header)
		}
		escaped.Query["x_seed"][1] = 'z'
		escaped.Headers["X-Trace"] = "late"
		return nativeRuleWire(200, `{"packet_rate_limit_rule":{}}`)
	})
	value, err := api.FindPacketRateLimitRecord(context.Background(), "parent", "target", option, func(c *rules.PacketRateLimitFindOpts) error { escaped = c; return nil })
	if err != nil || value == nil {
		t.Fatal(value, err)
	}
	pythonPacketRateField(t, value, "x_seed", `"owned"`)
	for _, parent := range []string{"", " ", ".", "..", "bad\n", string([]byte{255})} {
		before := len(calls)
		_, err := api.GetPacketRateLimitRecord(context.Background(), parent, "target")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(parent, err, calls)
		}
	}
	for _, key := range []string{"qos_policy_id", "qos_policy", "qos_rule_id", "qos_rule", "ignore_missing", "name_or_id", "cls", "base_path", "connection", "location", "__conflicting_attrs", "headers", "microversion", "id"} {
		before := len(calls)
		_, err := api.FindPacketRateLimitRecord(context.Background(), "parent", "target", rules.WithPacketRateLimitFindQuery(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(key, err, calls)
		}
		if key != "id" {
			values, err := api.AllPacketRateLimitRecords(context.Background(), "parent", rules.WithPacketRateLimitListQuery(key, nil))
			if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
				t.Fatal(key, values, err, calls)
			}
		}
	}
	before := len(calls)
	_, err = api.GetPacketRateLimitRecord(context.Background(), "parent", "target", nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
	_, err = api.GetPacketRateLimitRecord(context.Background(), "parent", "target", func(*rules.PacketRateLimitGetOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
}

func TestPythonPacketRateLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy(t *testing.T) {
	var calls []nativeRuleCall
	var versions []string
	api := pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if len(req.Header.Values("OpenStack-API-Version")) != 1 {
			t.Fatal(req.Header)
		}
		if len(calls) == 1 {
			return nativeRuleWire(403, `{"error":"fallback"}`)
		}
		return nativeRuleWire(200, `{"packet_rate_limit_rules":[{"id":"target"}]}`)
	})
	source := api.RawClient()
	source.Microversion = "2.0"
	source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
	value, err := api.FindPacketRateLimitRecord(context.Background(), "parent", "target", rules.WithPacketRateLimitFindMicroversion("2.1"))
	if err != nil || value == nil || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.0"}) || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" || source.Microversion != "2.0" {
		t.Fatal(value, err, versions, source)
	}
	for _, op := range []string{"get", "list"} {
		calls = nil
		versions = nil
		api = pythonPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			versions = append(versions, req.Header.Get("OpenStack-API-Version"))
			if len(calls) == 1 {
				return nativeRuleWire(503, `{"error":"retry"}`)
			}
			body := `{"packet_rate_limit_rule":{"id":"target"}}`
			if op == "list" {
				body = `{"packet_rate_limit_rules":[{"id":"target"}]}`
			}
			return nativeRuleWire(200, body)
		})
		source = api.RawClient()
		source.Microversion = "2.0"
		source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
		retries := 0
		source.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
			retries++
			if retries == 1 {
				return nil
			}
			return original
		}
		value, err = pythonPacketRateRead(context.Background(), api, op)
		if err != nil || value == nil || len(calls) != 2 || retries != 1 || !reflect.DeepEqual(versions, []string{"network 2.0", "network 2.0"}) {
			t.Fatal(op, value, err, calls, versions)
		}
	}
	for _, mutation := range []string{"header", "version", "omit", "status"} {
		calls = nil
		api = pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeRuleWire(503, `{"error":"original"}`)
			r.Header.Set("X-Request-Id", "retry-proof")
			return r
		})
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			if opts.MoreHeaders == nil {
				opts.MoreHeaders = map[string]string{}
			}
			switch mutation {
			case "header":
				opts.MoreHeaders["Authorization"] = "bad"
			case "version":
				opts.MoreHeaders["OpenStack-API-Version"] = "network 2.9"
			case "omit":
				opts.OmitHeaders = append(opts.OmitHeaders, "accept")
			case "status":
				opts.OkCodes = append(opts.OkCodes, 503)
			}
			return nil
		}
		_, err = api.AllPacketRateLimitRecords(context.Background(), "parent")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Request-Id") != "retry-proof" {
			t.Fatal(mutation, err, native)
		}
		if mutation != "status" && (!errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1) {
			t.Fatal(mutation, err, calls)
		}
		if mutation == "status" && len(calls) != 2 {
			t.Fatal(mutation, err, calls)
		}
	}
}

func TestPythonPacketRateLimitListPartialIterationAndAllDropLateError(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRuleWire(200, `{"packet_rate_limit_rules":[{"id":"first"}],"next":"?marker=first"}`)
		}
		return nativeRuleWire(500, `{"error":"late"}`)
	})
	count, failures := 0, 0
	for value, err := range api.ListPacketRateLimitRecords(context.Background(), "parent") {
		if err != nil {
			failures++
			continue
		}
		if value == nil {
			t.Fatal(value)
		}
		count++
	}
	if count != 1 || failures != 1 || len(calls) != 2 {
		t.Fatal(count, failures, calls)
	}
	calls = nil
	values, err := api.AllPacketRateLimitRecords(context.Background(), "parent")
	if values != nil || err == nil || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
}

func TestPythonPacketRateLimitComputedLocationAndMissingDefaults(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, forged := range []string{"", `null`, `false`, `"forged"`, `{"cloud":"attacker","project":{"id":"foreign"}}`} {
			t.Run(op+"/location="+forged, func(t *testing.T) {
				var calls []nativeRuleCall
				row := `{"id":"target","max_kpps":1e400,"max_burst_kpps":false,"direction":0,"x_extension":9007199254740993}`
				if forged == "" {
					row = `{}`
				} else {
					row = row[:len(row)-1] + `,"location":` + forged + `}`
				}
				body := `{"packet_rate_limit_rule":` + row + `}`
				if op == "list" {
					body = `{"packet_rate_limit_rules":[` + row + `]}`
				}
				api := pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(203, body) })
				value, err := pythonPacketRateRead(context.Background(), api, op)
				if err != nil || value == nil || len(calls) != 1 || value.StatusCode != 203 || string(value.Envelope) != body {
					t.Fatal(value, err, calls)
				}
				pythonPacketRateField(t, value, "location", `null`)
				if forged == "" {
					for _, key := range []string{"name", "max_kpps", "max_burst_kpps", "direction"} {
						pythonPacketRateField(t, value, key, `null`)
					}
					if op == "list" {
						pythonPacketRateField(t, value, "id", `null`)
					} else {
						pythonPacketRateField(t, value, "id", `"target"`)
					}
					if value.Wire.Body["location"] != nil {
						t.Fatal(value.Wire)
					}
				} else {
					pythonPacketRateField(t, value, "max_kpps", `1e400`)
					pythonPacketRateField(t, value, "max_burst_kpps", `false`)
					pythonPacketRateField(t, value, "direction", `0`)
					if string(value.Wire.Body["location"]) != forged {
						t.Fatal(value.Wire)
					}
					value.Resource.Body["location"] = json.RawMessage(`{"mutated":true}`)
					value.Envelope[0] = '['
					if string(value.Wire.Body["location"]) != forged || string(value.Wire.Body["max_kpps"]) != `1e400` {
						t.Fatal(value.Wire)
					}
				}
				if value.QoSPolicyID != "parent" || value.Resource.Body["tags"] != nil || value.Resource.Body["revision_number"] != nil || value.Resource.Body["if_match"] != nil {
					t.Fatal(value)
				}
			})
		}
	}
}

type pythonPacketRateMixedBody struct {
	*pythonBandwidthFaultBody
	onRead func()
}

func (b *pythonPacketRateMixedBody) Read(p []byte) (int, error) {
	if b.onRead != nil {
		invoke := b.onRead
		b.onRead = nil
		invoke()
	}
	return b.pythonBandwidthFaultBody.Read(p)
}
func TestPythonPacketRateLimitMixedPhysicalFaultsStopCallbacksAndNative401Proof(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, status := range []int{200, 400, 401, 403, 404, 429, 500, 503} {
			t.Run(fmt.Sprintf("%s/%d", op, status), func(t *testing.T) {
				var calls []nativeRuleCall
				readCause, closeCause, cancelCause := errors.New("packet read"), errors.New("packet close"), errors.New("packet cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				body := `{"packet_rate_limit_rule":{"id":"target","max_kpps":false},"actual":"whole physical body"}`
				if op == "list" {
					body = `{"packet_rate_limit_rules":[{"id":"target","max_kpps":false}],"actual":"whole physical body"}`
				}
				var physical *pythonPacketRateMixedBody
				api, cloud := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response {
					wire := nativeRuleWire(status, "")
					wire.Body = physical
					wire.Header.Set("X-Request-Id", "mixed-physical")
					return wire
				})
				source := api.RawClient()
				original := source.ResourceBase
				defer func() { source.ResourceBase = original }()
				target := source.ServiceURL("qos", "policies", "parent", "packet_rate_limit_rules")
				if op != "list" {
					target = source.ServiceURL("qos", "policies", "parent", "packet_rate_limit_rules", "target")
				}
				if op == "find" {
					target += "?qos_policy_id=parent"
				}
				physical = &pythonPacketRateMixedBody{pythonBandwidthFaultBody: &pythonBandwidthFaultBody{data: body, read: readCause, close: closeCause, onClose: func() { source.ResourceBase = original }}, onRead: func() { source.ResourceBase += "changed/"; cancel(cancelCause) }}
				retries, reauth := 0, 0
				source.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				cloud.Provider.ReauthFunc = func(context.Context) error { reauth++; return nil }
				value, err := pythonPacketRateRead(ctx, api, op)
				if value != nil && value.Resource != nil {
					t.Fatal("physical fault produced Resource", value, err)
				}
				if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 || physical.closes != 1 || retries != 0 || reauth != 0 {
					t.Fatal(value, err, calls, physical.closes, retries, reauth)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "mixed-physical" {
						t.Fatal(err, proof)
					}
				} else {
					originalErr := err
					if status == http.StatusUnauthorized {
						var nativeAuth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &nativeAuth) || !errors.Is(nativeAuth.ErrReauth, readCause) || !errors.Is(nativeAuth.ErrReauth, closeCause) || !errors.Is(nativeAuth.ErrReauth, cancelCause) || !errors.Is(nativeAuth.ErrReauth, context.Canceled) || !errors.Is(nativeAuth.ErrReauth, resource.ErrInvalidOption) {
							t.Fatal(err, nativeAuth)
						}
						originalErr = nativeAuth.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(originalErr, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != target || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "mixed-physical" {
						t.Fatal(err, native, target)
					}
				}
			})
		}
	}
}

type pythonPacketRateMarshalValue struct {
	cause error
	calls *int
}

func (v pythonPacketRateMarshalValue) MarshalJSON() ([]byte, error) {
	(*v.calls)++
	return nil, v.cause
}
func TestPythonPacketRateLimitOwnedNilMapsMarshalCausesAndCanceledPreflight(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(200, `{"packet_rate_limit_rules":[{}]}`) })
	cause := errors.New("caller JSON cause")
	conversions := 0
	value := pythonPacketRateMarshalValue{cause: cause, calls: &conversions}
	for _, op := range []string{"find", "list"} {
		var err error
		if op == "find" {
			_, err = api.FindPacketRateLimitRecord(context.Background(), "parent", "target", rules.WithPacketRateLimitFindQuery("max_kpps", value))
		} else {
			_, err = api.AllPacketRateLimitRecords(context.Background(), "parent", rules.WithPacketRateLimitListQuery("max_kpps", value))
		}
		var marshal *json.MarshalerError
		if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &marshal) || len(calls) != 0 {
			t.Fatal(op, err, marshal, calls)
		}
	}
	if conversions != 2 {
		t.Fatal(conversions)
	}
	for _, op := range []string{"find", "list"} {
		var err error
		if op == "find" {
			_, err = api.FindPacketRateLimitRecord(context.Background(), "parent", "target", rules.WithPacketRateLimitFindQuery("max_kpps", json.RawMessage(`bad`)))
		} else {
			_, err = api.AllPacketRateLimitRecords(context.Background(), "parent", rules.WithPacketRateLimitListQuery("max_kpps", json.RawMessage(`bad`)))
		}
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(op, err, calls)
		}
	}
	_, err := api.FindPacketRateLimitRecord(context.Background(), "parent", "target", func(c *rules.PacketRateLimitFindOpts) error { c.Query = nil; c.Headers = nil; return nil }, func(c *rules.PacketRateLimitFindOpts) error {
		c.Query["max_kpps"] = json.RawMessage(`false`)
		c.Headers["X-Trace"] = "owned"
		return cause
	})
	if !errors.Is(err, cause) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	_, err = api.AllPacketRateLimitRecords(context.Background(), "parent", func(c *rules.PacketRateLimitListOpts) error { c.Query = nil; c.Headers = nil; return nil }, func(c *rules.PacketRateLimitListOpts) error {
		c.Query["max_kpps"] = json.RawMessage(`false`)
		c.Headers["X-Trace"] = "owned"
		return cause
	})
	if !errors.Is(err, cause) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	_, err = api.GetPacketRateLimitRecord(context.Background(), "parent", "target", func(c *rules.PacketRateLimitGetOpts) error { c.Headers = nil; return nil }, func(c *rules.PacketRateLimitGetOpts) error { c.Headers["X-Trace"] = "owned"; return cause })
	if !errors.Is(err, cause) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	callbacks := 0
	_, err = api.FindPacketRateLimitRecord(ctx, "parent", "target", func(*rules.PacketRateLimitFindOpts) error { callbacks++; return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || callbacks != 0 || len(calls) != 0 {
		t.Fatal(err, callbacks, calls)
	}
	var nilAPI *rules.API
	for _, op := range []string{"get", "find", "list"} {
		_, err = pythonPacketRateRead(context.Background(), nilAPI, op)
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(op, err, calls)
		}
	}
	for _, key := range []string{"jmespath_filters", "allow_unknown_params"} {
		_, err = api.FindPacketRateLimitRecord(context.Background(), "parent", "target", rules.WithPacketRateLimitFindQuery(key, true))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
		_, err = api.AllPacketRateLimitRecords(context.Background(), "parent", rules.WithPacketRateLimitListQuery(key, true))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
	rows, err := api.AllPacketRateLimitRecords(context.Background(), "parent", rules.WithPacketRateLimitListQuery("unknown", value), rules.WithPacketRateLimitListQuery("fields", func() {}))
	if err != nil || len(rows) != 1 || len(calls) != 1 || conversions != 2 || calls[0].query != "" {
		t.Fatal(rows, err, conversions, calls)
	}
}

func TestPythonPacketRateLimitClean401LiveAuthExplicitHeadersAndVersionAbsence(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, selected := range []string{"", "2.0"} {
			for _, clear := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/selected=%s/clear=%t", op, selected, clear), func(t *testing.T) {
					var calls []nativeRuleCall
					var tokens []string
					expected := selected
					if clear {
						expected = ""
					}
					api, cloud := nativeRuleAPI(t, &calls, func(req *http.Request) *http.Response {
						tokens = append(tokens, req.Header.Get("X-Auth-Token"))
						if req.Header.Get("Accept") != "application/packet+json" {
							t.Fatal(req.Header)
						}
						for _, key := range []string{"Authorization", "X-Service-Token", "Cookie", "Host", "Content-Length", "Transfer-Encoding", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version"} {
							if _, exists := req.Header[http.CanonicalHeaderKey(key)]; exists {
								t.Fatal(key, req.Header)
							}
						}
						got := req.Header.Get("OpenStack-API-Version")
						if expected == "" {
							if _, exists := req.Header["Openstack-Api-Version"]; exists {
								t.Fatal(req.Header)
							}
						} else if got != "network "+expected || len(req.Header.Values("OpenStack-API-Version")) != 1 {
							t.Fatal(req.Header)
						}
						if len(calls) == 1 {
							return nativeRuleWire(401, `{"auth":"renew"}`)
						}
						body := `{"packet_rate_limit_rule":{"id":"target"}}`
						if op == "list" {
							body = `{"packet_rate_limit_rules":[{"id":"target"}]}`
						}
						return nativeRuleWire(200, body)
					})
					source := api.RawClient()
					source.Microversion = selected
					source.MoreHeaders = map[string]string{"Accept": "configured", "X-Trace": "selected"}
					if selected != "" {
						source.MoreHeaders["openstack-api-version"] = "network " + selected
					}
					originalHeaders := make(map[string]string)
					for k, v := range source.MoreHeaders {
						originalHeaders[k] = v
					}
					reauth := 0
					cloud.Provider.ReauthFunc = func(context.Context) error { reauth++; cloud.Provider.SetToken("renewed"); return nil }
					var record *rules.PacketRateLimitRecord
					var err error
					headers := map[string]string{"aCcEpT": "application/packet+json"}
					switch op {
					case "get":
						options := []rules.PacketRateLimitGetOption{rules.WithPacketRateLimitGetHeaders(headers)}
						if clear {
							options = append(options, rules.WithPacketRateLimitGetMicroversion(""))
						}
						record, err = api.GetPacketRateLimitRecord(context.Background(), "parent", "target", options...)
					case "find":
						options := []rules.PacketRateLimitFindOption{rules.WithPacketRateLimitFindHeaders(headers)}
						if clear {
							options = append(options, rules.WithPacketRateLimitFindMicroversion(""))
						}
						record, err = api.FindPacketRateLimitRecord(context.Background(), "parent", "target", options...)
					case "list":
						options := []rules.PacketRateLimitListOption{rules.WithPacketRateLimitListHeaders(headers)}
						if clear {
							options = append(options, rules.WithPacketRateLimitListMicroversion(""))
						}
						rows, listErr := api.AllPacketRateLimitRecords(context.Background(), "parent", options...)
						err = listErr
						if len(rows) == 1 {
							record = rows[0]
						}
					}
					headers["aCcEpT"] = "late"
					if err != nil || record == nil || reauth != 1 || len(calls) != 2 || calls[0] != calls[1] || !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) || source.Microversion != selected || !reflect.DeepEqual(source.MoreHeaders, originalHeaders) {
						t.Fatal(record, err, calls, tokens, reauth, source)
					}
				})
			}
		}
	}
}

func TestPythonPacketRateLimitListConstructorCollisionsAndMemberSelectedVersion(t *testing.T) {
	for _, key := range []string{"cls", "connection", "microversion", "_synchronized"} {
		t.Run("list/"+key, func(t *testing.T) {
			var calls []nativeRuleCall
			body := `{"packet_rate_limit_rules":[{"id":"first"},{"id":"second","` + key + `":null}],"extra":"whole-page-proof"}`
			api := pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
				wire := nativeRuleWire(203, body)
				wire.Header.Set("X-Request-Id", "constructor-proof")
				return wire
			})
			rows, err := api.AllPacketRateLimitRecords(context.Background(), "parent")
			var proof *resource.ResponseError
			var operation *resource.OperationError
			if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "constructor-proof" || !errors.As(err, &operation) || operation.Operation != "AllPacketRateLimitRecords" || len(calls) != 1 {
				t.Fatal(rows, err, proof, operation, calls)
			}
			// A collision comes from the accepted remote row, rather than a caller
			// option. Keep the actual projection cause reachable through both
			// operation wrappers and the whole-page physical response receipt.
			if proof.Cause == nil || !errors.Is(err, proof.Cause) || !strings.Contains(proof.Cause.Error(), key) || !strings.Contains(proof.Cause.Error(), "collides with Resource.existing constructor") {
				t.Fatal(err, proof.Cause)
			}
		})
	}
	for _, op := range []string{"get", "find"} {
		for _, selected := range []string{"", "2.0"} {
			t.Run(op+"/version="+selected, func(t *testing.T) {
				var calls []nativeRuleCall
				body := `{"packet_rate_limit_rule":{"id":"target","microversion":{"server":"forged"}}}`
				api := pythonPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(203, body) })
				api.RawClient().Microversion = selected
				value, err := pythonPacketRateRead(context.Background(), api, op)
				if err != nil || value == nil || len(calls) != 1 || string(value.Wire.Body["microversion"]) != `{"server":"forged"}` || string(value.Envelope) != body {
					t.Fatal(value, err, calls)
				}
				expected := `null`
				if selected != "" {
					expected = `"` + selected + `"`
				}
				pythonPacketRateField(t, value, "microversion", expected)
			})
		}
	}
}
