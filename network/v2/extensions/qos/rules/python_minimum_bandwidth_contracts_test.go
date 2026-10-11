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
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonMinimumBandwidthField(t *testing.T, value *rules.MinimumBandwidthRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s: want %s, record=%#v", key, want, value)
	}
}
func pythonMinimumBandwidthAPI(t *testing.T, calls *[]nativeRuleCall, reply func(*http.Request) *http.Response) *rules.API {
	t.Helper()
	api, _ := nativeRuleAPI(t, calls, reply)
	return api
}
func pythonMinimumBandwidthRead(ctx context.Context, api *rules.API, op string) (*rules.MinimumBandwidthRecord, error) {
	switch op {
	case "get":
		return api.GetMinimumBandwidthRecord(ctx, "parent", "target")
	case "find":
		return api.FindMinimumBandwidthRecord(ctx, "parent", "target")
	default:
		values, err := api.AllMinimumBandwidthRecords(ctx, "parent")
		if len(values) > 0 {
			return values[0], err
		}
		return nil, err
	}
}

func TestPythonMinimumBandwidthMemberLiteralRoutesSourceQueryAndURIView(t *testing.T) {
	for _, op := range []string{"get", "find"} {
		for _, includedParent := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/parent-in-body=%t", op, includedParent), func(t *testing.T) {
				var calls []nativeRuleCall
				body := `{"minimum_bandwidth_rule":{"min_kbps":"not-an-int","x_false":false,"direction":0,"x_new":9007199254740993,"self":"raw"}}`
				if includedParent {
					body = `{"minimum_bandwidth_rule":{"qos_policy_id":"foreign","min_kbps":"not-an-int","x_false":false,"direction":0,"x_new":9007199254740993,"self":"raw"}}`
				}
				parent, identity := "parent/id %2F", "rule/id %2F"
				api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
					if req.URL.EscapedPath() != "/neutron/v2.0/qos/policies/parent%2Fid%20%252F/minimum_bandwidth_rules/rule%2Fid%20%252F" {
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
				var value *rules.MinimumBandwidthRecord
				var err error
				if op == "get" {
					value, err = api.GetMinimumBandwidthRecord(context.Background(), parent, identity)
				} else {
					value, err = api.FindMinimumBandwidthRecord(context.Background(), parent, identity)
				}
				if err != nil || value == nil || value.QoSPolicyID != parent || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "receipt" || len(calls) != 1 || calls[0].body != "" {
					t.Fatal(value, err, calls)
				}
				pythonMinimumBandwidthField(t, value, "id", `"rule/id %2F"`)
				pythonMinimumBandwidthField(t, value, "name", `null`)
				pythonMinimumBandwidthField(t, value, "min_kbps", `"not-an-int"`)
				pythonMinimumBandwidthField(t, value, "x_false", `false`)
				pythonMinimumBandwidthField(t, value, "direction", `0`)
				pythonMinimumBandwidthField(t, value, "x_new", `9007199254740993`)
				pythonMinimumBandwidthField(t, value, "location", `null`)
				if value.Resource.Body["tags"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["self"] != nil {
					t.Fatal(value.Resource.Body)
				}
				if includedParent {
					pythonMinimumBandwidthField(t, value, "qos_policy_id", `"parent/id %2F"`)
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

func TestPythonMinimumBandwidthGetNoFallbackAndFindMissingCompleteSearch(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		var calls []nativeRuleCall
		api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/target") {
				return nativeRuleWire(code, `{"error":"member hidden"}`)
			}
			if len(req.URL.Query()) != 0 {
				t.Fatal("no parent query or name hint on LIST", req.URL)
			}
			return nativeRuleWire(200, `{"minimum_bandwidth_rules":[]}`)
		})
		value, err := api.GetMinimumBandwidthRecord(context.Background(), "parent", "target")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != code || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		calls = nil
		value, err = api.FindMinimumBandwidthRecord(context.Background(), "parent", "target")
		if value != nil || err != nil || len(calls) != 2 {
			t.Fatal(value, err, calls)
		}
		calls = nil
		value, err = api.FindMinimumBandwidthRecord(context.Background(), "parent", "target", rules.WithMinimumBandwidthFindIgnoreMissing(false))
		if value != nil || !errors.Is(err, resource.ErrNotFound) || len(calls) != 2 {
			t.Fatal(value, err, calls)
		}
	}
}

func TestPythonMinimumBandwidthFindBodySeedsAndListLocalFilters(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			want := url.Values{"qos_policy_id": {"parent"}, "name": {"target"}, "min_kbps": {"True"}, "direction": {"egress"}, "x_seed": {"keep"}, "fields": {"id", "direction"}}
			if !reflect.DeepEqual(req.URL.Query(), want) {
				t.Fatal(req.URL.Query(), want)
			}
			return nativeRuleWire(404, "")
		}
		if len(req.URL.Query()) != 0 {
			t.Fatal("Body fields are local; fields unknown is discarded", req.URL)
		}
		return nativeRuleWire(200, `{"minimum_bandwidth_rules":[{"id":"no","name":"target","min_kbps":0,"direction":"egress"},{"id":"yes","name":"target","min_kbps":1,"direction":"egress","qos_policy_id":"foreign"}]}`)
	})
	value, err := api.FindMinimumBandwidthRecord(context.Background(), "parent", "target", rules.WithMinimumBandwidthFindQuery("name", "target"), rules.WithMinimumBandwidthFindQuery("min_kbps", true), rules.WithMinimumBandwidthFindQuery("direction", "egress"), rules.WithMinimumBandwidthFindQuery("x_seed", "keep"), rules.WithMinimumBandwidthFindQuery("fields", []string{"id", "direction"}))
	if err != nil || value == nil || len(calls) != 2 || value.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(value, err, calls)
	}
	pythonMinimumBandwidthField(t, value, "id", `"yes"`)
	if string(value.Wire.Body["qos_policy_id"]) != `"foreign"` || value.QoSPolicyID != "parent" {
		t.Fatal(value)
	}
	calls = nil
	seeded := pythonMinimumBandwidthAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(204, "") })
	value, err = seeded.FindMinimumBandwidthRecord(context.Background(), "parent", "target", rules.WithMinimumBandwidthFindQuery("min_kbps", "untyped"), rules.WithMinimumBandwidthFindQuery("x_seed", false))
	if err != nil || value.Wire != nil || value.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(value, err)
	}
	pythonMinimumBandwidthField(t, value, "min_kbps", `"untyped"`)
	pythonMinimumBandwidthField(t, value, "x_seed", `false`)
}

func TestPythonMinimumBandwidthFindLaterDuplicateAndListFailureRemainTerminal(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		var calls []nativeRuleCall
		api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			switch len(calls) {
			case 1:
				return nativeRuleWire(404, "")
			case 2:
				return nativeRuleWire(200, `{"minimum_bandwidth_rules":[{"id":"first","name":"target"}],"minimum_bandwidth_rules_links":[{"rel":"next","href":"?marker=first"}]}`)
			default:
				if req.URL.Query().Get("marker") != "first" {
					t.Fatal(req.URL)
				}
				if duplicate {
					return nativeRuleWire(200, `{"minimum_bandwidth_rules":[{"id":"later","name":"target"}]}`)
				}
				return nativeRuleWire(403, `{"error":"rule access denied despite shared policy"}`)
			}
		})
		value, err := api.FindMinimumBandwidthRecord(context.Background(), "parent", "target")
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

func TestPythonMinimumBandwidthListLazyQueryLocalFiltersAndUnknowns(t *testing.T) {
	var calls []nativeRuleCall
	body := `{"minimum_bandwidth_rules":[{"id":"wrong","name":"demo","min_kbps":0,"direction":"egress"},{"id":"right","name":"demo","min_kbps":1,"direction":"egress","x_untyped":"raw","qos_policy_id":"foreign","tags":false,"x_ext":9007199254740993,"self":"wire"}]}`
	api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(req.URL.Query()) != 0 || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header)
		}
		response := nativeRuleWire(200, body)
		response.Header.Set("If-Match", "HTTP-only")
		return response
	})
	seq := api.ListMinimumBandwidthRecords(context.Background(), "parent", rules.WithMinimumBandwidthListQuery("name", "demo"), rules.WithMinimumBandwidthListQuery("min_kbps", true), rules.WithMinimumBandwidthListQuery("direction", "egress"), rules.WithMinimumBandwidthListQuery("fields", func() {}), rules.WithMinimumBandwidthListQuery("tags", func() {}), rules.WithMinimumBandwidthListOpts(rules.MinimumBandwidthListOpts{Query: map[string]json.RawMessage{"name": json.RawMessage(`"demo"`), "min_kbps": json.RawMessage(`true`), "direction": json.RawMessage(`"egress"`), "ignored": json.RawMessage(`[`)}}))
	if len(calls) != 0 {
		t.Fatal("eager list", calls)
	}
	var values []*rules.MinimumBandwidthRecord
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
	pythonMinimumBandwidthField(t, value, "id", `"right"`)
	pythonMinimumBandwidthField(t, value, "x_untyped", `"raw"`)
	pythonMinimumBandwidthField(t, value, "tags", `false`)
	if value.Resource.Body["qos_policy_id"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["self"] != nil || value.QoSPolicyID != "parent" || string(value.Wire.Body["qos_policy_id"]) != `"foreign"` || string(value.Envelope) != body {
		t.Fatal(value)
	}
}

func TestPythonMinimumBandwidthListRawCapSinglePageAndEarlyStop(t *testing.T) {
	for _, mode := range []string{"cap", "single", "break"} {
		var calls []nativeRuleCall
		api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			if mode == "cap" && req.URL.Query().Get("limit") != "1" {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, `{"minimum_bandwidth_rules":[{"id":"filtered","direction":"ingress"},{"id":"kept","direction":"egress"}],"minimum_bandwidth_rules_links":[{"rel":"next","href":"?marker=kept"}]}`)
		})
		opts := []rules.MinimumBandwidthListOption{}
		if mode == "cap" {
			opts = append(opts, rules.WithMinimumBandwidthListLimit(0), rules.WithMinimumBandwidthListMaxItems(1), rules.WithMinimumBandwidthListQuery("direction", "egress"))
		}
		if mode == "single" {
			opts = append(opts, rules.WithMinimumBandwidthListPaginated(false))
		}
		count := 0
		for value, err := range api.ListMinimumBandwidthRecords(context.Background(), "parent", opts...) {
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

func TestPythonMinimumBandwidthListContinuationCannotChangeParentOrFilters(t *testing.T) {
	for _, href := range []string{"?marker=one", "/v2.0/qos/policies/parent/minimum_bandwidth_rules?marker=one", "/v2.0/qos/policies/foreign/minimum_bandwidth_rules?marker=one", "?marker=one&limit=9", "https://foreign.invalid/qos/policies/parent/minimum_bandwidth_rules"} {
		var calls []nativeRuleCall
		api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeRuleWire(200, fmt.Sprintf(`{"minimum_bandwidth_rules":[{"id":"one"}],"next":%q}`, href))
			}
			if req.URL.Path != "/neutron/v2.0/qos/policies/parent/minimum_bandwidth_rules" || req.URL.Query().Get("marker") != "one" {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, `{"minimum_bandwidth_rules":[]}`)
		})
		values, err := api.AllMinimumBandwidthRecords(context.Background(), "parent", rules.WithMinimumBandwidthListLimit(1))
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

func TestPythonMinimumBandwidthListMarkerFallbackAndFalseyNumericNext(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRuleWire(200, `{"minimum_bandwidth_rules":[{"id":"one"}]}`)
		}
		if req.URL.Query().Get("marker") != "one" || req.URL.Query().Get("limit") != "3" {
			t.Fatal(req.URL)
		}
		return nativeRuleWire(200, `{"minimum_bandwidth_rules":[]}`)
	})
	values, err := api.AllMinimumBandwidthRecords(context.Background(), "parent", rules.WithMinimumBandwidthListLimit(3))
	if err != nil || len(values) != 1 || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
	for _, token := range []string{"1e-4000", "0e100", "1"} {
		calls = nil
		api = pythonMinimumBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeRuleWire(200, `{"minimum_bandwidth_rules":[{"id":"one"}],"next":`+token+`}`)
		})
		values, err = api.AllMinimumBandwidthRecords(context.Background(), "parent")
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

func TestPythonMinimumBandwidthAllStatusAndMalformedReceiptBoundaries(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		var calls []nativeRuleCall
		status := 200
		body := `{"minimum_bandwidth_rule":{}}`
		if op == "list" {
			body = `{"minimum_bandwidth_rules":[]}`
		}
		api := pythonMinimumBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeRuleWire(status, body)
			response.Header.Set("X-Request-Id", "status-proof")
			return response
		})
		for _, code := range []int{200, 203, 299, 302, 399} {
			status = code
			calls = nil
			_, err := pythonMinimumBandwidthRead(context.Background(), api, op)
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
			_, err := pythonMinimumBandwidthRead(context.Background(), api, op)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, code, err, native, calls)
			}
		}
		status = 200
		for _, bad := range []string{`null`, `[]`, `{"minimum_bandwidth_rule":null}`, `{"minimum_bandwidth_rule":false}`} {
			if op == "list" {
				bad = `{"minimum_bandwidth_rules":null}`
			}
			body = bad
			calls = nil
			_, err := pythonMinimumBandwidthRead(context.Background(), api, op)
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || string(proof.Body) != bad || proof.Header.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, bad, err, proof, calls)
			}
		}
	}
}

type pythonMinimumBandwidthFaultBody struct {
	data        string
	read, close error
	used        bool
	closes      int
	onClose     func()
}

func (b *pythonMinimumBandwidthFaultBody) Read(p []byte) (int, error) {
	if b.used {
		return 0, io.EOF
	}
	b.used = true
	return copy(p, b.data), b.read
}
func (b *pythonMinimumBandwidthFaultBody) Close() error {
	b.closes++
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

func TestPythonMinimumBandwidthPhysicalReadCloseCancelAndSourceProof(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, status := range []int{200, 404, 500} {
			for _, fault := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", op, status, fault), func(t *testing.T) {
					read, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel in Close")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					var calls []nativeRuleCall
					var client *gophercloud.ServiceClient
					body := `{"minimum_bandwidth_rule":{"id":"target"}}`
					if op == "list" {
						body = `{"minimum_bandwidth_rules":[{"id":"target"}]}`
					}
					physical := &pythonMinimumBandwidthFaultBody{data: body}
					switch fault {
					case "read-close":
						physical.read, physical.close = read, closeCause
					case "cancel":
						physical.onClose = func() { cancel(cancelCause) }
					case "source":
						physical.onClose = func() { client.ResourceBase += "changed/" }
					}
					api := pythonMinimumBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
						r := nativeRuleWire(status, body)
						r.Body = physical
						r.Header.Set("X-Request-Id", "physical-proof")
						return r
					})
					client = api.RawClient()
					_, err := pythonMinimumBandwidthRead(ctx, api, op)
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

func TestPythonMinimumBandwidthOptionSnapshotsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`"owned"`)
	query := map[string]json.RawMessage{"x_seed": raw}
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.0"
	option := rules.WithMinimumBandwidthFindOpts(rules.MinimumBandwidthFindOpts{Query: query, Headers: headers, Microversion: &version})
	raw[1] = 'x'
	headers["X-Trace"] = "changed"
	version = "changed"
	var escaped *rules.MinimumBandwidthFindOpts
	var calls []nativeRuleCall
	api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("x_seed") != "owned" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.0" {
			t.Fatal(req.URL, req.Header)
		}
		escaped.Query["x_seed"][1] = 'z'
		escaped.Headers["X-Trace"] = "late"
		return nativeRuleWire(200, `{"minimum_bandwidth_rule":{}}`)
	})
	value, err := api.FindMinimumBandwidthRecord(context.Background(), "parent", "target", option, func(c *rules.MinimumBandwidthFindOpts) error { escaped = c; return nil })
	if err != nil || value == nil {
		t.Fatal(value, err)
	}
	pythonMinimumBandwidthField(t, value, "x_seed", `"owned"`)
	for _, parent := range []string{"", " ", ".", "..", "bad\n", string([]byte{255})} {
		before := len(calls)
		_, err := api.GetMinimumBandwidthRecord(context.Background(), parent, "target")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(parent, err, calls)
		}
	}
	for _, key := range []string{"qos_policy_id", "qos_policy", "qos_rule_id", "qos_rule", "ignore_missing", "name_or_id", "allow_unknown_params", "jmespath_filters", "cls", "base_path", "connection", "location", "__conflicting_attrs", "headers", "microversion", "id"} {
		before := len(calls)
		_, err := api.FindMinimumBandwidthRecord(context.Background(), "parent", "target", rules.WithMinimumBandwidthFindQuery(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(key, err, calls)
		}
		if key != "id" {
			values, err := api.AllMinimumBandwidthRecords(context.Background(), "parent", rules.WithMinimumBandwidthListQuery(key, nil))
			if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
				t.Fatal(key, values, err, calls)
			}
		}
	}
	before := len(calls)
	_, err = api.GetMinimumBandwidthRecord(context.Background(), "parent", "target", nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}

	var composed []nativeRuleCall
	safe := pythonMinimumBandwidthAPI(t, &composed, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Trace") != "composed" {
			t.Fatal(req.Header)
		}
		body := `{"minimum_bandwidth_rule":{}}`
		if !strings.HasSuffix(req.URL.Path, "/target") {
			body = `{"minimum_bandwidth_rules":[]}`
		}
		return nativeRuleWire(200, body)
	})
	_, composeErr := safe.GetMinimumBandwidthRecord(context.Background(), "parent", "target", func(c *rules.MinimumBandwidthGetOpts) error { c.Headers = nil; return nil }, func(c *rules.MinimumBandwidthGetOpts) error { c.Headers["X-Trace"] = "composed"; return nil })
	if composeErr != nil {
		t.Fatal(composeErr)
	}
	_, composeErr = safe.FindMinimumBandwidthRecord(context.Background(), "parent", "target", func(c *rules.MinimumBandwidthFindOpts) error { c.Query = nil; c.Headers = nil; return nil }, func(c *rules.MinimumBandwidthFindOpts) error {
		c.Query["x_seed"] = json.RawMessage(`true`)
		c.Headers["X-Trace"] = "composed"
		return nil
	})
	if composeErr != nil {
		t.Fatal(composeErr)
	}
	_, composeErr = safe.AllMinimumBandwidthRecords(context.Background(), "parent", func(c *rules.MinimumBandwidthListOpts) error { c.Query = nil; c.Headers = nil; return nil }, func(c *rules.MinimumBandwidthListOpts) error { c.Headers["X-Trace"] = "composed"; return nil })
	if composeErr != nil || len(composed) != 3 {
		t.Fatal(composeErr, composed)
	}
	_, err = api.GetMinimumBandwidthRecord(context.Background(), "parent", "target", func(*rules.MinimumBandwidthGetOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
}

func TestPythonMinimumBandwidthVersionPhasesCleanRetryAndOwnedHeaderPolicy(t *testing.T) {
	var calls []nativeRuleCall
	var versions []string
	api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if len(req.Header.Values("OpenStack-API-Version")) != 1 {
			t.Fatal(req.Header)
		}
		if len(calls) == 1 {
			return nativeRuleWire(403, `{"error":"fallback"}`)
		}
		return nativeRuleWire(200, `{"minimum_bandwidth_rules":[{"id":"target"}]}`)
	})
	source := api.RawClient()
	source.Microversion = "2.0"
	source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
	value, err := api.FindMinimumBandwidthRecord(context.Background(), "parent", "target", rules.WithMinimumBandwidthFindMicroversion("2.1"))
	if err != nil || value == nil || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.0"}) || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" || source.Microversion != "2.0" {
		t.Fatal(value, err, versions, source)
	}
	for _, op := range []string{"get", "list"} {
		calls = nil
		versions = nil
		api = pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			versions = append(versions, req.Header.Get("OpenStack-API-Version"))
			if len(calls) == 1 {
				return nativeRuleWire(503, `{"error":"retry"}`)
			}
			body := `{"minimum_bandwidth_rule":{"id":"target"}}`
			if op == "list" {
				body = `{"minimum_bandwidth_rules":[{"id":"target"}]}`
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
		value, err = pythonMinimumBandwidthRead(context.Background(), api, op)
		if err != nil || value == nil || len(calls) != 2 || retries != 1 || !reflect.DeepEqual(versions, []string{"network 2.0", "network 2.0"}) {
			t.Fatal(op, value, err, calls, versions)
		}
	}
	for _, mutation := range []string{"header", "version", "omit", "status"} {
		calls = nil
		api = pythonMinimumBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
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
		_, err = api.AllMinimumBandwidthRecords(context.Background(), "parent")
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

func TestPythonMinimumBandwidthListPartialIterationAndAllDropLateError(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonMinimumBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRuleWire(200, `{"minimum_bandwidth_rules":[{"id":"first"}],"next":"?marker=first"}`)
		}
		return nativeRuleWire(500, `{"error":"late"}`)
	})
	count, failures := 0, 0
	for value, err := range api.ListMinimumBandwidthRecords(context.Background(), "parent") {
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
	values, err := api.AllMinimumBandwidthRecords(context.Background(), "parent")
	if values != nil || err == nil || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
}

func TestPythonMinimumBandwidthComputedLocationCannotBeForgedInResponse(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, selected := range []string{"", "2.2"} {
			var calls []nativeRuleCall
			row := `{"id":"target","location":{"cloud":"forged","project":{"id":"foreign"}}}`
			if op != "list" {
				row = `{"id":"target","location":{"cloud":"forged","project":{"id":"foreign"}},"microversion":"foreign"}`
			}
			body := `{"minimum_bandwidth_rule":` + row + `}`
			if op == "list" {
				body = `{"minimum_bandwidth_rules":[` + row + `]}`
			}
			api := pythonMinimumBandwidthAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(200, body) })
			api.RawClient().Microversion = selected
			value, err := pythonMinimumBandwidthRead(context.Background(), api, op)
			if err != nil || value == nil || len(calls) != 1 || string(value.Envelope) != body {
				t.Fatal(op, value, err, calls)
			}
			pythonMinimumBandwidthField(t, value, "location", `null`)
			if string(value.Wire.Body["location"]) != `{"cloud":"forged","project":{"id":"foreign"}}` {
				t.Fatal(value.Wire)
			}
			if op != "list" {
				want := `null`
				if selected != "" {
					want = `"2.2"`
				}
				pythonMinimumBandwidthField(t, value, "microversion", want)
				if string(value.Wire.Body["microversion"]) != `"foreign"` {
					t.Fatal(value.Wire)
				}
			}
		}
	}
	var calls []nativeRuleCall
	api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("OpenStack-API-Version") != "network 2.4" {
			t.Fatal(req.Header)
		}
		return nativeRuleWire(200, `{"minimum_bandwidth_rule":{"microversion":"foreign","location":false}}`)
	})
	api.RawClient().Microversion = "2.2"
	value, err := api.GetMinimumBandwidthRecord(context.Background(), "parent", "target", rules.WithMinimumBandwidthGetMicroversion("2.4"))
	if err != nil || value == nil || api.RawClient().Microversion != "2.2" {
		t.Fatal(value, err)
	}
	pythonMinimumBandwidthField(t, value, "microversion", `"2.4"`)
	pythonMinimumBandwidthField(t, value, "location", `null`)
	if string(value.Wire.Body["location"]) != `false` {
		t.Fatal(value.Wire)
	}
}

func TestPythonMinimumBandwidthListConstructorAndUntypedDictionaryFilters(t *testing.T) {
	for _, key := range []string{"cls", "connection", "microversion", "_synchronized"} {
		var calls []nativeRuleCall
		body := fmt.Sprintf(`{"minimum_bandwidth_rules":[{"id":"one",%q:null}]}`, key)
		api := pythonMinimumBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeRuleWire(203, body)
			r.Header.Set("X-Request-Id", "constructor")
			return r
		})
		values, err := api.AllMinimumBandwidthRecords(context.Background(), "parent")
		var proof *resource.ResponseError
		if values != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "constructor" || len(calls) != 1 {
			t.Fatal(key, values, err, proof, calls)
		}
	}
	for _, tc := range []struct {
		actual, filter string
		count          int
		fault          bool
	}{
		{`{"nested":{"v":1,"extra":false}}`, `{"nested":{"v":true}}`, 1, false},
		{`1e-4000`, `{}`, 0, false},
		{`1`, `{}`, 1, false},
		{`1`, `{"nested":null}`, 0, true},
		{`9007199254740993.0`, `9007199254740992`, 0, false},
	} {
		var calls []nativeRuleCall
		body := `{"minimum_bandwidth_rules":[{"id":"one","min_kbps":` + tc.actual + `}]}`
		api := pythonMinimumBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(req.URL.Query()) != 0 {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, body)
		})
		values, err := api.AllMinimumBandwidthRecords(context.Background(), "parent", rules.WithMinimumBandwidthListQuery("min_kbps", json.RawMessage(tc.filter)))
		if tc.fault {
			var proof *resource.ResponseError
			if values != nil || !errors.As(err, &proof) || string(proof.Body) != body {
				t.Fatal(tc, values, err, proof)
			}
		} else if err != nil || len(values) != tc.count {
			t.Fatal(tc, values, err)
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
}

type pythonMinimumBandwidthMarshalValue struct {
	cause error
	calls *int
}

func (v pythonMinimumBandwidthMarshalValue) MarshalJSON() ([]byte, error) {
	(*v.calls)++
	return nil, v.cause
}

func TestPythonMinimumBandwidthKnownMarshalCausesPreflightAndUnknownDiscard(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonMinimumBandwidthAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(200, `{"minimum_bandwidth_rules":[]}`) })
	cause := errors.New("caller marshaler cause")
	conversions := 0
	value := pythonMinimumBandwidthMarshalValue{cause: cause, calls: &conversions}
	find := rules.WithMinimumBandwidthFindQuery("min_kbps", value)
	_, err := api.FindMinimumBandwidthRecord(context.Background(), "parent", "target", find)
	var marshal *json.MarshalerError
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cause) || !errors.As(err, &marshal) || conversions != 1 || len(calls) != 0 {
		t.Fatal(err, marshal, conversions, calls)
	}
	list := rules.WithMinimumBandwidthListQuery("min_kbps", value)
	rows, err := api.AllMinimumBandwidthRecords(context.Background(), "parent", list)
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cause) || !errors.As(err, &marshal) || conversions != 2 || len(calls) != 0 {
		t.Fatal(rows, err, conversions, calls)
	}
	rows, err = api.AllMinimumBandwidthRecords(context.Background(), "parent", rules.WithMinimumBandwidthListQuery("max_kbps", value), rules.WithMinimumBandwidthListQuery("tags", func() {}))
	if err != nil || len(rows) != 0 || conversions != 2 || len(calls) != 1 {
		t.Fatal(rows, err, conversions, calls)
	}
}

type pythonMinimumBandwidthReadHook struct {
	*pythonMinimumBandwidthFaultBody
	after func()
}

func (b *pythonMinimumBandwidthReadHook) Read(p []byte) (int, error) {
	n, err := b.pythonMinimumBandwidthFaultBody.Read(p)
	if b.after != nil {
		b.after()
	}
	return n, err
}

func TestPythonMinimumBandwidthMixedFaultsStopRetryAndReauthCallbacks(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, status := range []int{200, 401, 404, 503} {
			t.Run(fmt.Sprintf("%s/%d", op, status), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				read, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
				body := `{"minimum_bandwidth_rule":{"id":"target"}}`
				if op == "list" {
					body = `{"minimum_bandwidth_rules":[{"id":"target"}]}`
				}
				var calls []nativeRuleCall
				var physical *pythonMinimumBandwidthReadHook
				api, cloud := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response {
					r := nativeRuleWire(status, body)
					r.Body = physical
					r.Header.Set("X-Request-Id", "mixed-proof")
					return r
				})
				client := api.RawClient()
				original := client.ResourceBase
				originalURL := client.ServiceURL("qos/policies/parent/minimum_bandwidth_rules")
				if op != "list" {
					originalURL = client.ServiceURL("qos/policies/parent/minimum_bandwidth_rules", "target")
					if op == "find" {
						originalURL += "?qos_policy_id=parent"
					}
				}
				physical = &pythonMinimumBandwidthReadHook{pythonMinimumBandwidthFaultBody: &pythonMinimumBandwidthFaultBody{data: body, read: read, close: closeCause, onClose: func() { client.ResourceBase = original; cancel(cancelCause) }}, after: func() { client.ResourceBase = original + "changed/" }}
				retry, reauth := 0, 0
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retry++
					return nil
				}
				cloud.Provider.ReauthFunc = func(context.Context) error { reauth++; return nil }
				_, err := pythonMinimumBandwidthRead(ctx, api, op)
				if !errors.Is(err, read) || !errors.Is(err, closeCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || retry != 0 || reauth != 0 || len(calls) != 1 || physical.closes != 1 {
					t.Fatal(err, retry, reauth, calls, physical.closes)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "mixed-proof" {
						t.Fatal(err, proof)
					}
				} else {
					originalErr := err
					if status == 401 {
						var reauthentication *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &reauthentication) || !errors.Is(reauthentication.ErrReauth, read) || !errors.Is(reauthentication.ErrReauth, closeCause) || !errors.Is(reauthentication.ErrReauth, cancelCause) || !errors.Is(reauthentication.ErrReauth, resource.ErrInvalidOption) {
							t.Fatal(err, reauthentication)
						}
						originalErr = reauthentication.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(originalErr, &native) || native.Actual != status || native.Method != "GET" || native.URL != originalURL || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "mixed-proof" {
						t.Fatal(err, native, originalURL)
					}
				}
			})
		}
	}
}
