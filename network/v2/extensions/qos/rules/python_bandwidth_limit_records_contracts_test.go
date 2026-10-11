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

func pythonBandwidthField(t *testing.T, value *rules.BandwidthLimitRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s: want %s, record=%#v", key, want, value)
	}
}
func pythonBandwidthAPI(t *testing.T, calls *[]nativeRuleCall, reply func(*http.Request) *http.Response) *rules.API {
	t.Helper()
	api, _ := nativeRuleAPI(t, calls, reply)
	return api
}
func pythonBandwidthRead(ctx context.Context, api *rules.API, op string) (*rules.BandwidthLimitRecord, error) {
	switch op {
	case "get":
		return api.GetBandwidthLimitRecord(ctx, "parent", "target")
	case "find":
		return api.FindBandwidthLimitRecord(ctx, "parent", "target")
	default:
		values, err := api.AllBandwidthLimitRecords(ctx, "parent")
		if len(values) > 0 {
			return values[0], err
		}
		return nil, err
	}
}

func TestPythonBandwidthLimitMemberLiteralRoutesSourceQueryAndURIView(t *testing.T) {
	for _, op := range []string{"get", "find"} {
		for _, includedParent := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/parent-in-body=%t", op, includedParent), func(t *testing.T) {
				var calls []nativeRuleCall
				body := `{"bandwidth_limit_rule":{"max_kbps":"not-an-int","max_burst_kbps":false,"direction":0,"x_new":9007199254740993,"self":"raw"}}`
				if includedParent {
					body = `{"bandwidth_limit_rule":{"qos_policy_id":"foreign","max_kbps":"not-an-int","max_burst_kbps":false,"direction":0,"x_new":9007199254740993,"self":"raw"}}`
				}
				parent, identity := "parent/id %2F", "rule/id %2F"
				api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
					if req.URL.EscapedPath() != "/neutron/v2.0/qos/policies/parent%2Fid%20%252F/bandwidth_limit_rules/rule%2Fid%20%252F" {
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
				var value *rules.BandwidthLimitRecord
				var err error
				if op == "get" {
					value, err = api.GetBandwidthLimitRecord(context.Background(), parent, identity)
				} else {
					value, err = api.FindBandwidthLimitRecord(context.Background(), parent, identity)
				}
				if err != nil || value == nil || value.QoSPolicyID != parent || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "receipt" || len(calls) != 1 || calls[0].body != "" {
					t.Fatal(value, err, calls)
				}
				pythonBandwidthField(t, value, "id", `"rule/id %2F"`)
				pythonBandwidthField(t, value, "name", `null`)
				pythonBandwidthField(t, value, "max_kbps", `"not-an-int"`)
				pythonBandwidthField(t, value, "max_burst_kbps", `false`)
				pythonBandwidthField(t, value, "direction", `0`)
				pythonBandwidthField(t, value, "x_new", `9007199254740993`)
				pythonBandwidthField(t, value, "location", `null`)
				if value.Resource.Body["tags"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["self"] != nil {
					t.Fatal(value.Resource.Body)
				}
				if includedParent {
					pythonBandwidthField(t, value, "qos_policy_id", `"parent/id %2F"`)
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

func TestPythonBandwidthLimitGetNoFallbackAndFindMissingCompleteSearch(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		var calls []nativeRuleCall
		api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/target") {
				return nativeRuleWire(code, `{"error":"member hidden"}`)
			}
			if len(req.URL.Query()) != 0 {
				t.Fatal("no parent query or name hint on LIST", req.URL)
			}
			return nativeRuleWire(200, `{"bandwidth_limit_rules":[]}`)
		})
		value, err := api.GetBandwidthLimitRecord(context.Background(), "parent", "target")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != code || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		calls = nil
		value, err = api.FindBandwidthLimitRecord(context.Background(), "parent", "target")
		if value != nil || err != nil || len(calls) != 2 {
			t.Fatal(value, err, calls)
		}
		calls = nil
		value, err = api.FindBandwidthLimitRecord(context.Background(), "parent", "target", rules.WithBandwidthLimitFindIgnoreMissing(false))
		if value != nil || !errors.Is(err, resource.ErrNotFound) || len(calls) != 2 {
			t.Fatal(value, err, calls)
		}
	}
}

func TestPythonBandwidthLimitFindBodySeedsAndListLocalFilters(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			want := url.Values{"qos_policy_id": {"parent"}, "name": {"target"}, "max_kbps": {"True"}, "max_burst_kbps": {"nested"}, "direction": {"egress"}, "x_seed": {"keep"}, "fields": {"id", "direction"}}
			if !reflect.DeepEqual(req.URL.Query(), want) {
				t.Fatal(req.URL.Query(), want)
			}
			return nativeRuleWire(404, "")
		}
		if len(req.URL.Query()) != 0 {
			t.Fatal("Body fields are local; fields unknown is discarded", req.URL)
		}
		return nativeRuleWire(200, `{"bandwidth_limit_rules":[{"id":"no","name":"target","max_kbps":0,"direction":"egress","max_burst_kbps":{"nested":null}},{"id":"yes","name":"target","max_kbps":1,"direction":"egress","max_burst_kbps":{"nested":null},"qos_policy_id":"foreign"}]}`)
	})
	value, err := api.FindBandwidthLimitRecord(context.Background(), "parent", "target", rules.WithBandwidthLimitFindQuery("name", "target"), rules.WithBandwidthLimitFindQuery("max_kbps", true), rules.WithBandwidthLimitFindQuery("max_burst_kbps", map[string]any{"nested": nil}), rules.WithBandwidthLimitFindQuery("direction", "egress"), rules.WithBandwidthLimitFindQuery("x_seed", "keep"), rules.WithBandwidthLimitFindQuery("fields", []string{"id", "direction"}))
	if err != nil || value == nil || len(calls) != 2 || value.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(value, err, calls)
	}
	pythonBandwidthField(t, value, "id", `"yes"`)
	if string(value.Wire.Body["qos_policy_id"]) != `"foreign"` || value.QoSPolicyID != "parent" {
		t.Fatal(value)
	}
	calls = nil
	seeded := pythonBandwidthAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(204, "") })
	value, err = seeded.FindBandwidthLimitRecord(context.Background(), "parent", "target", rules.WithBandwidthLimitFindQuery("max_kbps", "untyped"), rules.WithBandwidthLimitFindQuery("x_seed", false))
	if err != nil || value.Wire != nil || value.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(value, err)
	}
	pythonBandwidthField(t, value, "max_kbps", `"untyped"`)
	pythonBandwidthField(t, value, "x_seed", `false`)
}

func TestPythonBandwidthLimitFindLaterDuplicateAndListFailureRemainTerminal(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		var calls []nativeRuleCall
		api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			switch len(calls) {
			case 1:
				return nativeRuleWire(404, "")
			case 2:
				return nativeRuleWire(200, `{"bandwidth_limit_rules":[{"id":"first","name":"target"}],"bandwidth_limit_rules_links":[{"rel":"next","href":"?marker=first"}]}`)
			default:
				if req.URL.Query().Get("marker") != "first" {
					t.Fatal(req.URL)
				}
				if duplicate {
					return nativeRuleWire(200, `{"bandwidth_limit_rules":[{"id":"later","name":"target"}]}`)
				}
				return nativeRuleWire(403, `{"error":"rule access denied despite shared policy"}`)
			}
		})
		value, err := api.FindBandwidthLimitRecord(context.Background(), "parent", "target")
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

func TestPythonBandwidthLimitListLazyQueryLocalFiltersAndUnknowns(t *testing.T) {
	var calls []nativeRuleCall
	body := `{"bandwidth_limit_rules":[{"id":"wrong","name":"demo","max_kbps":0,"direction":"egress"},{"id":"right","name":"demo","max_kbps":1,"direction":"egress","max_burst_kbps":"raw","qos_policy_id":"foreign","tags":false,"x_ext":9007199254740993,"self":"wire"}]}`
	api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(req.URL.Query()) != 0 || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header)
		}
		response := nativeRuleWire(200, body)
		response.Header.Set("If-Match", "HTTP-only")
		return response
	})
	seq := api.ListBandwidthLimitRecords(context.Background(), "parent", rules.WithBandwidthLimitListQuery("name", "demo"), rules.WithBandwidthLimitListQuery("max_kbps", true), rules.WithBandwidthLimitListQuery("direction", "egress"), rules.WithBandwidthLimitListQuery("fields", func() {}), rules.WithBandwidthLimitListQuery("tags", func() {}), rules.WithBandwidthLimitListOpts(rules.BandwidthLimitListOpts{Query: map[string]json.RawMessage{"name": json.RawMessage(`"demo"`), "max_kbps": json.RawMessage(`true`), "direction": json.RawMessage(`"egress"`), "ignored": json.RawMessage(`[`)}}))
	if len(calls) != 0 {
		t.Fatal("eager list", calls)
	}
	var values []*rules.BandwidthLimitRecord
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
	pythonBandwidthField(t, value, "id", `"right"`)
	pythonBandwidthField(t, value, "max_burst_kbps", `"raw"`)
	pythonBandwidthField(t, value, "tags", `false`)
	if value.Resource.Body["qos_policy_id"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["self"] != nil || value.QoSPolicyID != "parent" || string(value.Wire.Body["qos_policy_id"]) != `"foreign"` || string(value.Envelope) != body {
		t.Fatal(value)
	}
}

func TestPythonBandwidthLimitListRawCapSinglePageAndEarlyStop(t *testing.T) {
	for _, mode := range []string{"cap", "single", "break"} {
		var calls []nativeRuleCall
		api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			if mode == "cap" && req.URL.Query().Get("limit") != "1" {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, `{"bandwidth_limit_rules":[{"id":"filtered","direction":"ingress"},{"id":"kept","direction":"egress"}],"bandwidth_limit_rules_links":[{"rel":"next","href":"?marker=kept"}]}`)
		})
		opts := []rules.BandwidthLimitListOption{}
		if mode == "cap" {
			opts = append(opts, rules.WithBandwidthLimitListLimit(0), rules.WithBandwidthLimitListMaxItems(1), rules.WithBandwidthLimitListQuery("direction", "egress"))
		}
		if mode == "single" {
			opts = append(opts, rules.WithBandwidthLimitListPaginated(false))
		}
		count := 0
		for value, err := range api.ListBandwidthLimitRecords(context.Background(), "parent", opts...) {
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

func TestPythonBandwidthLimitListContinuationCannotChangeParentOrFilters(t *testing.T) {
	for _, href := range []string{"?marker=one", "/v2.0/qos/policies/parent/bandwidth_limit_rules?marker=one", "/v2.0/qos/policies/foreign/bandwidth_limit_rules?marker=one", "?marker=one&limit=9", "https://foreign.invalid/qos/policies/parent/bandwidth_limit_rules"} {
		var calls []nativeRuleCall
		api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeRuleWire(200, fmt.Sprintf(`{"bandwidth_limit_rules":[{"id":"one"}],"next":%q}`, href))
			}
			if req.URL.Path != "/neutron/v2.0/qos/policies/parent/bandwidth_limit_rules" || req.URL.Query().Get("marker") != "one" {
				t.Fatal(req.URL)
			}
			return nativeRuleWire(200, `{"bandwidth_limit_rules":[]}`)
		})
		values, err := api.AllBandwidthLimitRecords(context.Background(), "parent", rules.WithBandwidthLimitListLimit(1))
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

func TestPythonBandwidthLimitListMarkerFallbackAndFalseyNumericNext(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRuleWire(200, `{"bandwidth_limit_rules":[{"id":"one"}]}`)
		}
		if req.URL.Query().Get("marker") != "one" || req.URL.Query().Get("limit") != "3" {
			t.Fatal(req.URL)
		}
		return nativeRuleWire(200, `{"bandwidth_limit_rules":[]}`)
	})
	values, err := api.AllBandwidthLimitRecords(context.Background(), "parent", rules.WithBandwidthLimitListLimit(3))
	if err != nil || len(values) != 1 || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
	for _, token := range []string{"1e-4000", "0e100", "1"} {
		calls = nil
		api = pythonBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeRuleWire(200, `{"bandwidth_limit_rules":[{"id":"one"}],"next":`+token+`}`)
		})
		values, err = api.AllBandwidthLimitRecords(context.Background(), "parent")
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

func TestPythonBandwidthLimitAllStatusAndMalformedReceiptBoundaries(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		var calls []nativeRuleCall
		status := 200
		body := `{"bandwidth_limit_rule":{}}`
		if op == "list" {
			body = `{"bandwidth_limit_rules":[]}`
		}
		api := pythonBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeRuleWire(status, body)
			response.Header.Set("X-Request-Id", "status-proof")
			return response
		})
		for _, code := range []int{200, 203, 299, 302, 399} {
			status = code
			calls = nil
			_, err := pythonBandwidthRead(context.Background(), api, op)
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
			_, err := pythonBandwidthRead(context.Background(), api, op)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, code, err, native, calls)
			}
		}
		status = 200
		for _, bad := range []string{`null`, `[]`, `{"bandwidth_limit_rule":null}`, `{"bandwidth_limit_rule":false}`} {
			if op == "list" {
				bad = `{"bandwidth_limit_rules":null}`
			}
			body = bad
			calls = nil
			_, err := pythonBandwidthRead(context.Background(), api, op)
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || string(proof.Body) != bad || proof.Header.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, bad, err, proof, calls)
			}
		}
	}
}

type pythonBandwidthFaultBody struct {
	data        string
	read, close error
	used        bool
	closes      int
	onClose     func()
}

func (b *pythonBandwidthFaultBody) Read(p []byte) (int, error) {
	if b.used {
		return 0, io.EOF
	}
	b.used = true
	return copy(p, b.data), b.read
}
func (b *pythonBandwidthFaultBody) Close() error {
	b.closes++
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

func TestPythonBandwidthLimitPhysicalReadCloseCancelAndSourceProof(t *testing.T) {
	for _, op := range []string{"get", "find", "list"} {
		for _, status := range []int{200, 404, 500} {
			for _, fault := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", op, status, fault), func(t *testing.T) {
					read, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel in Close")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					var calls []nativeRuleCall
					var client *gophercloud.ServiceClient
					body := `{"bandwidth_limit_rule":{"id":"target"}}`
					if op == "list" {
						body = `{"bandwidth_limit_rules":[{"id":"target"}]}`
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
					api := pythonBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
						r := nativeRuleWire(status, body)
						r.Body = physical
						r.Header.Set("X-Request-Id", "physical-proof")
						return r
					})
					client = api.RawClient()
					_, err := pythonBandwidthRead(ctx, api, op)
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

func TestPythonBandwidthLimitOptionSnapshotsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`"owned"`)
	query := map[string]json.RawMessage{"x_seed": raw}
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.0"
	option := rules.WithBandwidthLimitFindOpts(rules.BandwidthLimitFindOpts{Query: query, Headers: headers, Microversion: &version})
	raw[1] = 'x'
	headers["X-Trace"] = "changed"
	version = "changed"
	var escaped *rules.BandwidthLimitFindOpts
	var calls []nativeRuleCall
	api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("x_seed") != "owned" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.0" {
			t.Fatal(req.URL, req.Header)
		}
		escaped.Query["x_seed"][1] = 'z'
		escaped.Headers["X-Trace"] = "late"
		return nativeRuleWire(200, `{"bandwidth_limit_rule":{}}`)
	})
	value, err := api.FindBandwidthLimitRecord(context.Background(), "parent", "target", option, func(c *rules.BandwidthLimitFindOpts) error { escaped = c; return nil })
	if err != nil || value == nil {
		t.Fatal(value, err)
	}
	pythonBandwidthField(t, value, "x_seed", `"owned"`)
	for _, parent := range []string{"", " ", ".", "..", "bad\n", string([]byte{255})} {
		before := len(calls)
		_, err := api.GetBandwidthLimitRecord(context.Background(), parent, "target")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(parent, err, calls)
		}
	}
	for _, key := range []string{"qos_policy_id", "qos_policy", "qos_rule_id", "qos_rule", "ignore_missing", "name_or_id", "cls", "base_path", "connection", "location", "__conflicting_attrs", "headers", "microversion", "id"} {
		before := len(calls)
		_, err := api.FindBandwidthLimitRecord(context.Background(), "parent", "target", rules.WithBandwidthLimitFindQuery(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(key, err, calls)
		}
		if key != "id" {
			values, err := api.AllBandwidthLimitRecords(context.Background(), "parent", rules.WithBandwidthLimitListQuery(key, nil))
			if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
				t.Fatal(key, values, err, calls)
			}
		}
	}
	before := len(calls)
	_, err = api.GetBandwidthLimitRecord(context.Background(), "parent", "target", nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
	_, err = api.GetBandwidthLimitRecord(context.Background(), "parent", "target", func(*rules.BandwidthLimitGetOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
}

func TestPythonBandwidthLimitVersionPhasesCleanRetryAndOwnedHeaderPolicy(t *testing.T) {
	var calls []nativeRuleCall
	var versions []string
	api := pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if len(req.Header.Values("OpenStack-API-Version")) != 1 {
			t.Fatal(req.Header)
		}
		if len(calls) == 1 {
			return nativeRuleWire(403, `{"error":"fallback"}`)
		}
		return nativeRuleWire(200, `{"bandwidth_limit_rules":[{"id":"target"}]}`)
	})
	source := api.RawClient()
	source.Microversion = "2.0"
	source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
	value, err := api.FindBandwidthLimitRecord(context.Background(), "parent", "target", rules.WithBandwidthLimitFindMicroversion("2.1"))
	if err != nil || value == nil || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.0"}) || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" || source.Microversion != "2.0" {
		t.Fatal(value, err, versions, source)
	}
	for _, op := range []string{"get", "list"} {
		calls = nil
		versions = nil
		api = pythonBandwidthAPI(t, &calls, func(req *http.Request) *http.Response {
			versions = append(versions, req.Header.Get("OpenStack-API-Version"))
			if len(calls) == 1 {
				return nativeRuleWire(503, `{"error":"retry"}`)
			}
			body := `{"bandwidth_limit_rule":{"id":"target"}}`
			if op == "list" {
				body = `{"bandwidth_limit_rules":[{"id":"target"}]}`
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
		value, err = pythonBandwidthRead(context.Background(), api, op)
		if err != nil || value == nil || len(calls) != 2 || retries != 1 || !reflect.DeepEqual(versions, []string{"network 2.0", "network 2.0"}) {
			t.Fatal(op, value, err, calls, versions)
		}
	}
	for _, mutation := range []string{"header", "version", "omit", "status"} {
		calls = nil
		api = pythonBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
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
		_, err = api.AllBandwidthLimitRecords(context.Background(), "parent")
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

func TestPythonBandwidthLimitListPartialIterationAndAllDropLateError(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonBandwidthAPI(t, &calls, func(*http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRuleWire(200, `{"bandwidth_limit_rules":[{"id":"first"}],"next":"?marker=first"}`)
		}
		return nativeRuleWire(500, `{"error":"late"}`)
	})
	count, failures := 0, 0
	for value, err := range api.ListBandwidthLimitRecords(context.Background(), "parent") {
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
	values, err := api.AllBandwidthLimitRecords(context.Background(), "parent")
	if values != nil || err == nil || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
}
