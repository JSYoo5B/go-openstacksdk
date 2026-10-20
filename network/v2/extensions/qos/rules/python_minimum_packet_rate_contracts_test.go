package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonMinimumPacketRateField(t *testing.T, value *rules.MinimumPacketRateRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s: want %s, record=%#v", key, want, value)
	}
}
func pythonMinimumPacketRateAPI(t *testing.T, calls *[]nativeRuleCall, reply func(*http.Request) *http.Response) *rules.API {
	t.Helper()
	api, _ := nativeRuleAPI(t, calls, reply)
	return api
}
func pythonMinimumPacketRateRead(ctx context.Context, api *rules.API, op string) (*rules.MinimumPacketRateRecord, error) {
	switch op {
	case "get":
		return api.GetMinimumPacketRateRecord(ctx, "parent", "target")
	case "find":
		return api.FindMinimumPacketRateRecord(ctx, "parent", "target")
	default:
		values, err := api.AllMinimumPacketRateRecords(ctx, "parent")
		if len(values) > 0 {
			return values[0], err
		}
		return nil, err
	}
}

func TestPythonMinimumPacketRateMemberLiteralRoutesSourceQueryAndURIView(t *testing.T) {
	for _, op := range []string{"get", "find"} {
		for _, includedParent := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/parent-in-body=%t", op, includedParent), func(t *testing.T) {
				var calls []nativeRuleCall
				body := `{"minimum_packet_rate_rule":{"min_kpps":"not-an-int","x_unknown":false,"direction":0,"x_new":9007199254740993,"self":"raw"}}`
				if includedParent {
					body = `{"minimum_packet_rate_rule":{"qos_policy_id":"foreign","min_kpps":"not-an-int","x_unknown":false,"direction":0,"x_new":9007199254740993,"self":"raw"}}`
				}
				parent, identity := "parent/id %2F", "rule/id %2F"
				api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
					if req.URL.EscapedPath() != "/neutron/v2.0/qos/policies/parent%2Fid%20%252F/minimum_packet_rate_rules/rule%2Fid%20%252F" {
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
				var value *rules.MinimumPacketRateRecord
				var err error
				if op == "get" {
					value, err = api.GetMinimumPacketRateRecord(context.Background(), parent, identity)
				} else {
					value, err = api.FindMinimumPacketRateRecord(context.Background(), parent, identity)
				}
				if err != nil || value == nil || value.QoSPolicyID != parent || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "receipt" || len(calls) != 1 || calls[0].body != "" {
					t.Fatal(value, err, calls)
				}
				pythonMinimumPacketRateField(t, value, "id", `"rule/id %2F"`)
				pythonMinimumPacketRateField(t, value, "name", `null`)
				pythonMinimumPacketRateField(t, value, "min_kpps", `"not-an-int"`)
				pythonMinimumPacketRateField(t, value, "x_unknown", `false`)
				pythonMinimumPacketRateField(t, value, "direction", `0`)
				pythonMinimumPacketRateField(t, value, "x_new", `9007199254740993`)
				pythonMinimumPacketRateField(t, value, "location", `null`)
				if value.Resource.Body["tags"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["self"] != nil {
					t.Fatal(value.Resource.Body)
				}
				if includedParent {
					pythonMinimumPacketRateField(t, value, "qos_policy_id", `"parent/id %2F"`)
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

func TestPythonMinimumPacketRateGetNoFallbackAndFindMissingCompleteSearch(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		var calls []nativeRuleCall
		api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/target") {
				return nativeRuleWire(code, `{"error":"member hidden"}`)
			}
			if len(req.URL.Query()) != 0 {
				t.Fatal("no parent query or name hint on LIST", req.URL)
			}
			return nativeRuleWire(200, `{"minimum_packet_rate_rules":[]}`)
		})
		value, err := api.GetMinimumPacketRateRecord(context.Background(), "parent", "target")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != code || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		calls = nil
		value, err = api.FindMinimumPacketRateRecord(context.Background(), "parent", "target")
		if value != nil || err != nil || len(calls) != 2 {
			t.Fatal(value, err, calls)
		}
		calls = nil
		value, err = api.FindMinimumPacketRateRecord(context.Background(), "parent", "target", rules.WithMinimumPacketRateFindIgnoreMissing(false))
		if value != nil || !errors.Is(err, resource.ErrNotFound) || len(calls) != 2 {
			t.Fatal(value, err, calls)
		}
	}
}

func TestPythonMinimumPacketRateFindBodySeedsAndListLocalFilters(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			want := url.Values{"qos_policy_id": {"parent"}, "name": {"target"}, "min_kpps": {"True"}, "x_unknown": {"nested"}, "direction": {"egress"}, "x_seed": {"keep"}, "fields": {"id", "direction"}}
			if !reflect.DeepEqual(req.URL.Query(), want) {
				t.Fatal(req.URL.Query(), want)
			}
			return nativeRuleWire(404, "")
		}
		if len(req.URL.Query()) != 0 {
			t.Fatal("Body fields are local; fields and x_unknown are discarded", req.URL)
		}
		return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"no","name":"target","min_kpps":0,"direction":"egress","x_unknown":{"nested":null}},{"id":"yes","name":"target","min_kpps":1,"direction":"egress","x_unknown":{"nested":null},"qos_policy_id":"foreign"}]}`)
	})
	value, err := api.FindMinimumPacketRateRecord(context.Background(), "parent", "target", rules.WithMinimumPacketRateFindQuery("name", "target"), rules.WithMinimumPacketRateFindQuery("min_kpps", true), rules.WithMinimumPacketRateFindQuery("x_unknown", map[string]any{"nested": nil}), rules.WithMinimumPacketRateFindQuery("direction", "egress"), rules.WithMinimumPacketRateFindQuery("x_seed", "keep"), rules.WithMinimumPacketRateFindQuery("fields", []string{"id", "direction"}))
	if err != nil || value == nil || len(calls) != 2 || value.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(value, err, calls)
	}
	pythonMinimumPacketRateField(t, value, "id", `"yes"`)
	if string(value.Wire.Body["qos_policy_id"]) != `"foreign"` || value.QoSPolicyID != "parent" {
		t.Fatal(value)
	}
	calls = nil
	seeded := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(204, "") })
	value, err = seeded.FindMinimumPacketRateRecord(context.Background(), "parent", "target", rules.WithMinimumPacketRateFindQuery("min_kpps", "untyped"), rules.WithMinimumPacketRateFindQuery("x_seed", false))
	if err != nil || value.Wire != nil || value.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(value, err)
	}
	pythonMinimumPacketRateField(t, value, "min_kpps", `"untyped"`)
	pythonMinimumPacketRateField(t, value, "x_seed", `false`)
}

func TestPythonMinimumPacketRateFindLaterDuplicateAndListFailureRemainTerminal(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		var calls []nativeRuleCall
		api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			switch len(calls) {
			case 1:
				return nativeRuleWire(404, "")
			case 2:
				return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"first","name":"target"}],"minimum_packet_rate_rules_links":[{"rel":"next","href":"?marker=first"}]}`)
			default:
				if req.URL.Query().Get("marker") != "first" {
					t.Fatal(req.URL)
				}
				if duplicate {
					return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"later","name":"target"}]}`)
				}
				return nativeRuleWire(403, `{"error":"rule access denied despite shared policy"}`)
			}
		})
		value, err := api.FindMinimumPacketRateRecord(context.Background(), "parent", "target")
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

func TestPythonMinimumPacketRateListLazyQueryLocalFiltersAndUnknowns(t *testing.T) {
	var calls []nativeRuleCall
	body := `{"minimum_packet_rate_rules":[{"id":"wrong","name":"demo","min_kpps":0,"direction":"egress"},{"id":"right","name":"demo","min_kpps":1,"direction":"egress","x_unknown":"raw","qos_policy_id":"foreign","tags":false,"x_ext":9007199254740993,"self":"wire"}]}`
	api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(req.URL.Query()) != 0 || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header)
		}
		response := nativeRuleWire(200, body)
		response.Header.Set("If-Match", "HTTP-only")
		return response
	})
	seq := api.ListMinimumPacketRateRecords(context.Background(), "parent", rules.WithMinimumPacketRateListQuery("name", "demo"), rules.WithMinimumPacketRateListQuery("min_kpps", true), rules.WithMinimumPacketRateListQuery("direction", "egress"), rules.WithMinimumPacketRateListQuery("fields", func() {}), rules.WithMinimumPacketRateListQuery("tags", func() {}), rules.WithMinimumPacketRateListOpts(rules.MinimumPacketRateListOpts{Query: map[string]json.RawMessage{"name": json.RawMessage(`"demo"`), "min_kpps": json.RawMessage(`true`), "direction": json.RawMessage(`"egress"`), "ignored": json.RawMessage(`[`)}}))
	if len(calls) != 0 {
		t.Fatal("eager list", calls)
	}
	var values []*rules.MinimumPacketRateRecord
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
	pythonMinimumPacketRateField(t, value, "id", `"right"`)
	pythonMinimumPacketRateField(t, value, "x_unknown", `"raw"`)
	pythonMinimumPacketRateField(t, value, "tags", `false`)
	if value.Resource.Body["qos_policy_id"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["self"] != nil || value.QoSPolicyID != "parent" || string(value.Wire.Body["qos_policy_id"]) != `"foreign"` || string(value.Envelope) != body {
		t.Fatal(value)
	}
}

func TestPythonMinimumPacketRateListRawCapSinglePageAndEarlyStop(t *testing.T) {
	for _, mode := range []string{"cap", "single", "break"} {
		var calls []nativeRuleCall
		api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			if mode == "cap" && req.URL.Query().Get("limit") != "1" {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"filtered","direction":"ingress"},{"id":"kept","direction":"egress"}],"minimum_packet_rate_rules_links":[{"rel":"next","href":"?marker=kept"}]}`)
		})
		opts := []rules.MinimumPacketRateListOption{}
		if mode == "cap" {
			opts = append(opts, rules.WithMinimumPacketRateListLimit(0), rules.WithMinimumPacketRateListMaxItems(1), rules.WithMinimumPacketRateListQuery("direction", "egress"))
		}
		if mode == "single" {
			opts = append(opts, rules.WithMinimumPacketRateListPaginated(false))
		}
		count := 0
		for value, err := range api.ListMinimumPacketRateRecords(context.Background(), "parent", opts...) {
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

func TestPythonMinimumPacketRateListContinuationCannotChangeParentOrFilters(t *testing.T) {
	for _, href := range []string{"?marker=one", "/v2.0/qos/policies/parent/minimum_packet_rate_rules?marker=one", "/v2.0/qos/policies/foreign/minimum_packet_rate_rules?marker=one", "?marker=one&limit=9", "https://foreign.invalid/qos/policies/parent/minimum_packet_rate_rules"} {
		var calls []nativeRuleCall
		api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeRuleWire(200, fmt.Sprintf(`{"minimum_packet_rate_rules":[{"id":"one"}],"next":%q}`, href))
			}
			if req.URL.Path != "/neutron/v2.0/qos/policies/parent/minimum_packet_rate_rules" || req.URL.Query().Get("marker") != "one" {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, `{"minimum_packet_rate_rules":[]}`)
		})
		values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent", rules.WithMinimumPacketRateListLimit(1))
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

func TestPythonMinimumPacketRateListMarkerFallbackAndFalseyNumericNext(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"one"}]}`)
		}
		if req.URL.Query().Get("marker") != "one" || req.URL.Query().Get("limit") != "3" {
			t.Fatal(req.URL)
		}
		return nativeRuleWire(200, `{"minimum_packet_rate_rules":[]}`)
	})
	values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent", rules.WithMinimumPacketRateListLimit(3))
	if err != nil || len(values) != 1 || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
	for _, token := range []string{"1e-4000", "0e100", "1"} {
		calls = nil
		api = pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"one"}],"next":`+token+`}`)
		})
		values, err = api.AllMinimumPacketRateRecords(context.Background(), "parent")
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

func TestPythonMinimumPacketRateAllStatusAndMalformedReceiptBoundaries(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		var calls []nativeRuleCall
		status := 200
		body := `{"minimum_packet_rate_rule":{}}`
		if op == "list" {
			body = `{"minimum_packet_rate_rules":[]}`
		}
		api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeRuleWire(status, body)
			response.Header.Set("X-Request-Id", "status-proof")
			return response
		})
		for _, code := range []int{200, 203, 299, 302, 399} {
			status = code
			calls = nil
			_, err := pythonMinimumPacketRateRead(context.Background(), api, op)
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
			_, err := pythonMinimumPacketRateRead(context.Background(), api, op)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, code, err, native, calls)
			}
		}
		status = 200
		for _, bad := range []string{`null`, `[]`, `{"minimum_packet_rate_rule":null}`, `{"minimum_packet_rate_rule":false}`} {
			if op == "list" {
				bad = `{"minimum_packet_rate_rules":null}`
			}
			body = bad
			calls = nil
			_, err := pythonMinimumPacketRateRead(context.Background(), api, op)
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || string(proof.Body) != bad || proof.Header.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, bad, err, proof, calls)
			}
		}
	}
}

type pythonMinimumPacketRateFaultBody struct {
	data        string
	read, close error
	used        bool
	closes      int
	onClose     func()
}

func (b *pythonMinimumPacketRateFaultBody) Read(p []byte) (int, error) {
	if b.used {
		return 0, io.EOF
	}
	b.used = true
	return copy(p, b.data), b.read
}
func (b *pythonMinimumPacketRateFaultBody) Close() error {
	b.closes++
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

func TestPythonMinimumPacketRatePhysicalReadCloseCancelAndSourceProof(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, status := range []int{200, 404, 500} {
			for _, fault := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", op, status, fault), func(t *testing.T) {
					read, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel in Close")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					var calls []nativeRuleCall
					var client *gophercloud.ServiceClient
					body := `{"minimum_packet_rate_rule":{"id":"target"}}`
					if op == "list" {
						body = `{"minimum_packet_rate_rules":[{"id":"target"}]}`
					}
					physical := &pythonMinimumPacketRateFaultBody{data: body}
					switch fault {
					case "read-close":
						physical.read, physical.close = read, closeCause
					case "cancel":
						physical.onClose = func() { cancel(cancelCause) }
					case "source":
						physical.onClose = func() { client.ResourceBase += "changed/" }
					}
					api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
						r := nativeRuleWire(status, body)
						r.Body = physical
						r.Header.Set("X-Request-Id", "physical-proof")
						return r
					})
					client = api.RawClient()
					_, err := pythonMinimumPacketRateRead(ctx, api, op)
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

func TestPythonMinimumPacketRateOptionSnapshotsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`"owned"`)
	query := map[string]json.RawMessage{"x_seed": raw}
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.0"
	option := rules.WithMinimumPacketRateFindOpts(rules.MinimumPacketRateFindOpts{Query: query, Headers: headers, Microversion: &version})
	raw[1] = 'x'
	headers["X-Trace"] = "changed"
	version = "changed"
	var escaped *rules.MinimumPacketRateFindOpts
	var calls []nativeRuleCall
	api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("x_seed") != "owned" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.0" {
			t.Fatal(req.URL, req.Header)
		}
		escaped.Query["x_seed"][1] = 'z'
		escaped.Headers["X-Trace"] = "late"
		return nativeRuleWire(200, `{"minimum_packet_rate_rule":{}}`)
	})
	value, err := api.FindMinimumPacketRateRecord(context.Background(), "parent", "target", option, func(c *rules.MinimumPacketRateFindOpts) error { escaped = c; return nil })
	if err != nil || value == nil {
		t.Fatal(value, err)
	}
	pythonMinimumPacketRateField(t, value, "x_seed", `"owned"`)
	for _, parent := range []string{"", " ", ".", "..", "bad\n", string([]byte{255})} {
		before := len(calls)
		_, err := api.GetMinimumPacketRateRecord(context.Background(), parent, "target")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(parent, err, calls)
		}
	}
	for _, key := range []string{"qos_policy_id", "qos_policy", "qos_rule_id", "qos_rule", "ignore_missing", "name_or_id", "cls", "base_path", "connection", "location", "__conflicting_attrs", "headers", "microversion", "id"} {
		before := len(calls)
		_, err := api.FindMinimumPacketRateRecord(context.Background(), "parent", "target", rules.WithMinimumPacketRateFindQuery(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(key, err, calls)
		}
		if key != "id" {
			values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent", rules.WithMinimumPacketRateListQuery(key, nil))
			if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
				t.Fatal(key, values, err, calls)
			}
		}
	}
	before := len(calls)
	_, err = api.GetMinimumPacketRateRecord(context.Background(), "parent", "target", nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
	_, err = api.GetMinimumPacketRateRecord(context.Background(), "parent", "target", func(*rules.MinimumPacketRateGetOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
}

func TestPythonMinimumPacketRateVersionPhasesCleanRetryAndOwnedHeaderPolicy(t *testing.T) {
	var calls []nativeRuleCall
	var versions []string
	api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if len(req.Header.Values("OpenStack-API-Version")) != 1 {
			t.Fatal(req.Header)
		}
		if len(calls) == 1 {
			return nativeRuleWire(403, `{"error":"fallback"}`)
		}
		return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"target"}]}`)
	})
	source := api.RawClient()
	source.Microversion = "2.0"
	source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
	value, err := api.FindMinimumPacketRateRecord(context.Background(), "parent", "target", rules.WithMinimumPacketRateFindMicroversion("2.1"))
	if err != nil || value == nil || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.0"}) || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" || source.Microversion != "2.0" {
		t.Fatal(value, err, versions, source)
	}
	for _, op := range []string{"get", "list"} {
		calls = nil
		versions = nil
		api = pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			versions = append(versions, req.Header.Get("OpenStack-API-Version"))
			if len(calls) == 1 {
				return nativeRuleWire(503, `{"error":"retry"}`)
			}
			body := `{"minimum_packet_rate_rule":{"id":"target"}}`
			if op == "list" {
				body = `{"minimum_packet_rate_rules":[{"id":"target"}]}`
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
		value, err = pythonMinimumPacketRateRead(context.Background(), api, op)
		if err != nil || value == nil || len(calls) != 2 || retries != 1 || !reflect.DeepEqual(versions, []string{"network 2.0", "network 2.0"}) {
			t.Fatal(op, value, err, calls, versions)
		}
	}
	for _, mutation := range []string{"header", "version", "omit", "status"} {
		calls = nil
		api = pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
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
		_, err = api.AllMinimumPacketRateRecords(context.Background(), "parent")
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

func TestPythonMinimumPacketRateListPartialIterationAndAllDropLateError(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"first"}],"next":"?marker=first"}`)
		}
		return nativeRuleWire(500, `{"error":"late"}`)
	})
	count, failures := 0, 0
	for value, err := range api.ListMinimumPacketRateRecords(context.Background(), "parent") {
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
	values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent")
	if values != nil || err == nil || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
}

func TestPythonMinimumPacketRateSparseDefaultsAndForgedLocation(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, location := range []string{`{"project":"forged"}`, `false`, `"forged"`, `null`} {
			var calls []nativeRuleCall
			row := `{"id":"target","location":` + location + `,"x_future":9007199254740993}`
			body := `{"minimum_packet_rate_rule":` + row + `}`
			if op == "list" {
				body = `{"minimum_packet_rate_rules":[` + row + `]}`
			}
			api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(207, body) })
			value, err := pythonMinimumPacketRateRead(context.Background(), api, op)
			if err != nil || value == nil || len(calls) != 1 || value.StatusCode != 207 || string(value.Envelope) != body {
				t.Fatal(op, value, err, calls)
			}
			for _, key := range []string{"name", "min_kpps", "direction", "location"} {
				pythonMinimumPacketRateField(t, value, key, `null`)
			}
			pythonMinimumPacketRateField(t, value, "id", `"target"`)
			pythonMinimumPacketRateField(t, value, "x_future", `9007199254740993`)
			if value.Resource.Body["tags"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["qos_policy_id"] != nil || string(value.Wire.Body["location"]) != location {
				t.Fatal(value.Resource.Body, value.Wire.Body)
			}
		}
	}
	// Scalar/default preservation is defined by untyped Source Body descriptors.
	for _, raw := range []string{`null`, `false`, `0`, `"not-an-integer"`, `{"nested":[1,null]}`} {
		var calls []nativeRuleCall
		body := `{"minimum_packet_rate_rule":{"id":null,"min_kpps":` + raw + `,"direction":` + raw + `}}`
		api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(200, body) })
		value, err := api.GetMinimumPacketRateRecord(context.Background(), "parent", "target")
		if err != nil || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		pythonMinimumPacketRateField(t, value, "id", `null`)
		pythonMinimumPacketRateField(t, value, "min_kpps", raw)
		pythonMinimumPacketRateField(t, value, "direction", raw)
	}
}

func TestPythonMinimumPacketRateNoneQuerySeedsAndLocalDescriptorFilters(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		want := url.Values{"qos_policy_id": {"parent"}, "name": {"False"}, "fields": {"id", "direction"}}
		if !reflect.DeepEqual(req.URL.Query(), want) {
			t.Fatal(req.URL.Query(), want)
		}
		return nativeRuleWire(204, "")
	})
	value, err := api.FindMinimumPacketRateRecord(context.Background(), "parent", "target", rules.WithMinimumPacketRateFindQuery("limit", nil), rules.WithMinimumPacketRateFindQuery("min_kpps", nil), rules.WithMinimumPacketRateFindQuery("x_future", nil), rules.WithMinimumPacketRateFindQuery("name", false), rules.WithMinimumPacketRateFindQuery("fields", []string{"id", "direction"}))
	if err != nil || value == nil || value.Wire != nil || len(calls) != 1 || value.StatusCode != 204 || len(value.Envelope) != 0 {
		t.Fatal(value, err, calls)
	}
	for _, key := range []string{"min_kpps", "x_future", "limit"} {
		pythonMinimumPacketRateField(t, value, key, `null`)
	}
	pythonMinimumPacketRateField(t, value, "name", `false`)
	pythonMinimumPacketRateField(t, value, "fields", `["id","direction"]`)
	calls = nil
	api = pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(req.URL.Query()) != 0 {
			t.Fatal(req.URL)
		}
		return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"missing"},{"id":"zero","min_kpps":0,"direction":false},{"id":"null","min_kpps":null},{"id":"one","min_kpps":1}]}`)
	})
	values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent", rules.WithMinimumPacketRateListQuery("min_kpps", nil), rules.WithMinimumPacketRateListQuery("fields", func() {}))
	if err != nil || len(values) != 2 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	calls = nil
	values, err = api.AllMinimumPacketRateRecords(context.Background(), "parent", rules.WithMinimumPacketRateListQuery("min_kpps", false), rules.WithMinimumPacketRateListQuery("direction", false))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	pythonMinimumPacketRateField(t, values[0], "id", `"zero"`)
}

func TestPythonMinimumPacketRateListConstructorCollisionsAndDictionaryFilters(t *testing.T) {
	for _, row := range []string{`{"cls":null}`, `{"cls":false}`, `{"connection":null}`, `{"microversion":false}`, `{"_synchronized":null}`, `null`, `false`, `[]`} {
		var calls []nativeRuleCall
		body := `{"minimum_packet_rate_rules":[` + row + `]}`
		api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeRuleWire(299, body)
			r.Header.Set("X-Request-Id", "constructor-proof")
			return r
		})
		values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent")
		var proof *resource.ResponseError
		if values != nil || !errors.As(err, &proof) || proof.StatusCode != 299 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "constructor-proof" || len(calls) != 1 {
			t.Fatal(row, values, err, proof, calls)
		}
	}
	for _, body := range []string{`{}`, `{"minimum_packet_rate_rules":null}`, `{"minimum_packet_rate_rules":[]`, "{\"minimum_packet_rate_rules\":[{\"id\":\"bad\",\"direction\":\"\xff\"}]}"} {
		var calls []nativeRuleCall
		api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(203, body) })
		values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent")
		var proof *resource.ResponseError
		if values != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || len(calls) != 1 {
			t.Fatal(values, err, proof, calls)
		}
	}
	for _, actual := range []string{`"truthy"`, `{}`, `false`, `1e-400`, `{"nested":{"on":1}}`} {
		var calls []nativeRuleCall
		body := `{"minimum_packet_rate_rules":[{"id":"target","min_kpps":` + actual + `}]}`
		api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(200, body) })
		values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent", rules.WithMinimumPacketRateListQuery("min_kpps", map[string]any{}))
		want := 0
		if actual == `"truthy"` || strings.HasPrefix(actual, `{"nested"`) {
			want = 1
		}
		if err != nil || len(values) != want || len(calls) != 1 {
			t.Fatal(actual, values, err, calls)
		}
	}
	for _, actual := range []string{`{"nested":{"on":1,"extra":2}}`, `"truthy"`} {
		var calls []nativeRuleCall
		body := `{"minimum_packet_rate_rules":[{"id":"target","min_kpps":` + actual + `}]}`
		api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(200, body) })
		values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent", rules.WithMinimumPacketRateListQuery("min_kpps", map[string]any{"nested": map[string]any{"on": true}}))
		if strings.HasPrefix(actual, `{"nested"`) {
			if err != nil || len(values) != 1 {
				t.Fatal(values, err)
			}
		} else {
			var proof *resource.ResponseError
			if values != nil || !errors.As(err, &proof) || string(proof.Body) != body || len(calls) != 1 {
				t.Fatal(values, err, proof, calls)
			}
		}
	}
}

func TestPythonMinimumPacketRatePhysicalFaultsNeverInvokeRetryOrReauth(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, status := range []int{200, 299, 401, 403, 404, 503, 599} {
			for _, fault := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", op, status, fault), func(t *testing.T) {
					var calls []nativeRuleCall
					readCause, closeCause, cancelCause := errors.New("read fault"), errors.New("close fault"), errors.New("custom cancel")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					body := `{"minimum_packet_rate_rule":{"id":"target"}}`
					if op == "list" {
						body = `{"minimum_packet_rate_rules":[{"id":"target"}]}`
					}
					physical := &pythonMinimumPacketRateFaultBody{data: body}
					api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
						r := nativeRuleWire(status, body)
						r.Body = physical
						r.Header.Set("X-Request-Id", "physical")
						return r
					})
					originalURL := api.RawClient().ServiceURL("qos/policies/parent/minimum_packet_rate_rules")
					if op != "list" {
						originalURL = api.RawClient().ServiceURL("qos/policies/parent/minimum_packet_rate_rules", "target")
						if op == "find" {
							originalURL += "?qos_policy_id=parent"
						}
					}
					switch fault {
					case "read-close":
						physical.read, physical.close = readCause, closeCause
					case "cancel":
						physical.onClose = func() { cancel(cancelCause) }
					case "source":
						physical.onClose = func() { api.RawClient().ResourceBase += "changed/" }
					}
					retries, reauths := 0, 0
					api.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						retries++
						return nil
					}
					api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
					_, err := pythonMinimumPacketRateRead(ctx, api, op)
					if err == nil || len(calls) != 1 || physical.closes != 1 || retries != 0 || reauths != 0 {
						t.Fatal(err, calls, physical.closes, retries, reauths)
					}
					switch fault {
					case "read-close":
						if !errors.Is(err, readCause) || !errors.Is(err, closeCause) {
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
						if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "physical" {
							t.Fatal(err, proof)
						}
					} else {
						receiptErr := err
						if status == 401 {
							// Native Request returns a pointer with two public, non-unwrapped errors.
							var reauthentication *gophercloud.ErrUnableToReauthenticate
							if !errors.As(err, &reauthentication) || reauthentication.ErrOriginal == nil || reauthentication.ErrReauth == nil {
								t.Fatal(err, reauthentication)
							}
							switch fault {
							case "read-close":
								if !errors.Is(reauthentication.ErrReauth, readCause) || !errors.Is(reauthentication.ErrReauth, closeCause) {
									t.Fatal(err, reauthentication)
								}
							case "cancel":
								if !errors.Is(reauthentication.ErrReauth, context.Canceled) || !errors.Is(reauthentication.ErrReauth, cancelCause) {
									t.Fatal(err, reauthentication)
								}
							case "source":
								if !errors.Is(reauthentication.ErrReauth, resource.ErrInvalidOption) {
									t.Fatal(err, reauthentication)
								}
							}
							receiptErr = reauthentication.ErrOriginal
						}
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(receiptErr, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != originalURL || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "physical" {
							t.Fatal(err, native, originalURL)
						}
					}
				})
			}
		}
	}
}

func TestPythonMinimumPacketRateRetryVersionAbsenceAndLiveTokenOwnership(t *testing.T) {
	for _, selected := range []string{"", "2.0"} {
		for _, mutation := range []string{"version", "foreign-version", "authorization", "body"} {
			var calls []nativeRuleCall
			api := pythonMinimumPacketRateAPI(t, &calls, func(*http.Request) *http.Response {
				r := nativeRuleWire(503, `{"error":"original"}`)
				r.Header.Set("X-Request-Id", "owned-proof")
				return r
			})
			api.RawClient().Microversion = selected
			api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				if o.MoreHeaders == nil {
					o.MoreHeaders = map[string]string{}
				}
				switch mutation {
				case "version":
					o.MoreHeaders["Openstack-Api-Version"] = "network 2.9"
				case "foreign-version":
					o.MoreHeaders["X-OpenStack-Volume-API-Version"] = "3.99"
				case "authorization":
					o.MoreHeaders["Authorization"] = "forged"
				case "body":
					o.JSONBody = map[string]any{"body": "forged"}
				}
				return nil
			}
			_, err := api.GetMinimumPacketRateRecord(context.Background(), "parent", "target")
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"original"}` || native.ResponseHeader.Get("X-Request-Id") != "owned-proof" || len(calls) != 1 || calls[0].body != "" {
				t.Fatal(selected, mutation, err, native, calls)
			}
		}
	}
	for _, mode := range []string{"retry", "reauth"} {
		var calls []nativeRuleCall
		var provider *gophercloud.ProviderClient
		api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				code := 503
				if mode == "reauth" {
					code = 401
				}
				return nativeRuleWire(code, `{"error":"first"}`)
			}
			if req.Header.Get("X-Auth-Token") != "refreshed" || req.Header.Get("OpenStack-API-Version") != "" || calls[1].method != http.MethodGet || calls[1].body != "" {
				t.Fatal(req.Header, calls)
			}
			if mode == "retry" && req.Header.Get("X-Retry") != "owned" {
				t.Fatal(req.Header)
			}
			return nativeRuleWire(200, `{"minimum_packet_rate_rule":{"id":"target"}}`)
		})
		provider = api.RawClient().ProviderClient
		provider.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, original error, count uint) error {
			if count > 1 {
				return original
			}
			if o.MoreHeaders == nil {
				o.MoreHeaders = map[string]string{}
			}
			o.MoreHeaders["X-Retry"] = "owned"
			provider.SetToken("refreshed")
			return nil
		}
		provider.ReauthFunc = func(context.Context) error { provider.SetToken("refreshed"); return nil }
		value, err := api.GetMinimumPacketRateRecord(context.Background(), "parent", "target")
		if err != nil || value == nil || len(calls) != 2 {
			t.Fatal(mode, value, err, calls)
		}
	}
}

func TestPythonMinimumPacketRateExplicitListVersionAppliesToEveryPage(t *testing.T) {
	for _, override := range []string{"2.9", ""} {
		var calls []nativeRuleCall
		api := pythonMinimumPacketRateAPI(t, &calls, func(req *http.Request) *http.Response {
			want := ""
			if override != "" {
				want = "network " + override
			}
			if req.Header.Get("OpenStack-API-Version") != want || len(req.Header.Values("OpenStack-API-Version")) > 1 || req.Header.Get("Accept") != "application/json" {
				t.Fatal(req.Header)
			}
			if len(calls) == 1 {
				return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"first"}],"next":"?marker=first"}`)
			}
			if req.URL.Query().Get("marker") != "first" {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, `{"minimum_packet_rate_rules":[]}`)
		})
		source := api.RawClient()
		source.Microversion = "2.0"
		source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0", "Accept": "source/default"}
		values, err := api.AllMinimumPacketRateRecords(context.Background(), "parent", rules.WithMinimumPacketRateListMicroversion(override))
		if err != nil || len(values) != 1 || len(calls) != 2 || source.Microversion != "2.0" || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" || source.MoreHeaders["Accept"] != "source/default" {
			t.Fatal(values, err, calls, source)
		}
	}
}

func TestPythonMinimumPacketRateReusableOptionsConcurrentOwnedViews(t *testing.T) {
	raw := json.RawMessage(`"owned"`)
	header := map[string]string{"X-Trace": "owned"}
	query := map[string]json.RawMessage{"name": raw}
	version := "2.0"
	option := rules.WithMinimumPacketRateListOpts(rules.MinimumPacketRateListOpts{Query: query, Headers: header, Microversion: &version})
	raw[1] = 'x'
	query["name"] = json.RawMessage(`"changed"`)
	header["X-Trace"] = "changed"
	version = "changed"
	const workers = 8
	apis := make([]*rules.API, workers)
	calls := make([][]nativeRuleCall, workers)
	for i := range workers {
		apis[i] = pythonMinimumPacketRateAPI(t, &calls[i], func(req *http.Request) *http.Response {
			if len(req.URL.Query()) != 0 || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.0" {
				t.Fatal(req.URL, req.Header)
			}
			return nativeRuleWire(200, `{"minimum_packet_rate_rules":[{"id":"target","name":"owned","min_kpps":0}]}`)
		})
	}
	type answer struct {
		index int
		value *rules.MinimumPacketRateRecord
		err   error
	}
	results := make(chan answer, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			values, err := apis[index].AllMinimumPacketRateRecords(context.Background(), "parent", option)
			var value *rules.MinimumPacketRateRecord
			if len(values) == 1 {
				value = values[0]
			}
			results <- answer{index, value, err}
		}(i)
	}
	wg.Wait()
	close(results)
	values := make([]*rules.MinimumPacketRateRecord, workers)
	for result := range results {
		if result.err != nil || result.value == nil || len(calls[result.index]) != 1 {
			t.Fatal(result, calls[result.index])
		}
		values[result.index] = result.value
	}
	values[0].Resource.Body["name"][1] = 'x'
	values[0].Wire.Body["name"][1] = 'z'
	values[0].Envelope[0] = 'x'
	for _, value := range values[1:] {
		pythonMinimumPacketRateField(t, value, "name", `"owned"`)
		if string(value.Wire.Body["name"]) != `"owned"` || value.Envelope[0] != '{' {
			t.Fatal("shared snapshot", value)
		}
	}
}
