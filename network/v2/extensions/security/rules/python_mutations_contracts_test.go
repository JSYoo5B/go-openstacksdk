package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonSecurityRuleAPI(t *testing.T, calls *[]nativeRuleCall, reply func(*http.Request) *http.Response) *rules.API {
	t.Helper()
	cloud := testcloud.New(t)
	nativeRuleRecorder(cloud, calls, reply)
	return rules.New(nativeRuleClient(cloud))
}

// SecurityGroupRule._prepare_request removes a falsy remote address group only
// if the singular envelope exists. Resource.bulk_create requests no envelope
// for each row, so its final plural body preserves those raw empty values.
func TestPythonSecurityRuleCreatePresenceAliasesAndRemoteAddressGroup(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []rules.AttributeOption
		want    map[string]any
	}{
		{"omitted", nil, map[string]any{}},
		{"native required and zero defaults bypassed", []rules.AttributeOption{rules.WithAttribute("direction", ""), rules.WithAttribute("ether_type", ""), rules.WithAttribute("security_group_id", ""), rules.WithAttribute("description", ""), rules.WithAttribute("port_range_min", 0), rules.WithAttribute("port_range_max", 0), rules.WithAttribute("protocol", nil), rules.WithAttribute("remote_group_id", nil)}, map[string]any{"direction": "", "ethertype": "", "security_group_id": "", "description": "", "port_range_min": float64(0), "port_range_max": float64(0), "protocol": nil, "remote_group_id": nil}},
		{"unknown and query-only attrs stay Body", []rules.AttributeOption{rules.WithAttribute("ether_type", "IPv6"), rules.WithAttribute("x_extension", false), rules.WithAttribute("fields", []string{"id"})}, map[string]any{"ethertype": "IPv6", "x_extension": false, "fields": []any{"id"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativeRuleCall
			api := pythonSecurityRuleAPI(t, &calls, func(*http.Request) *http.Response {
				return nativeRuleWire(201, `{"security_group_rule":`+nativeRuleRow+`}`)
			})
			value, err := api.CreateWithAttributes(context.Background(), test.options...)
			if err != nil || value == nil || value.ID != "id-1" {
				t.Fatal(value, err)
			}
			if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/neutron/v2.0/security-group-rules" || calls[0].query != "" {
				t.Fatal(calls)
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, map[string]any{"security_group_rule": test.want}) {
				t.Fatal(body, test.want)
			}
		})
	}
	for _, test := range []struct {
		name  string
		value any
		want  string
	}{
		{"null", nil, ""}, {"false", false, ""}, {"zero", 0, ""}, {"negative zero exponent", json.Number("-0.00e+1000"), ""},
		{"empty string", "", ""}, {"empty list", []string{}, ""}, {"empty object", map[string]any{}, ""},
		{"address ID", "ag", `"ag"`}, {"zero string remains truthy", "0", `"0"`}, {"false string remains truthy", "False", `"False"`},
		{"nonempty list", []any{nil}, `[null]`}, {"nonempty object", map[string]any{"x": nil}, `{"x":null}`},
		{"precise nonzero", json.Number("0.00000000000000000001"), "0.00000000000000000001"}, {"small exponent nonzero", json.Number("1e-999"), "1e-999"},
	} {
		t.Run("remote address group/"+test.name, func(t *testing.T) {
			var calls []nativeRuleCall
			api := pythonSecurityRuleAPI(t, &calls, func(*http.Request) *http.Response {
				return nativeRuleWire(201, `{"security_group_rule":`+nativeRuleRow+`}`)
			})
			if _, err := api.CreateWithAttributes(context.Background(), rules.WithAttribute("remote_address_group_id", test.value)); err != nil {
				t.Fatal(err)
			}
			var body map[string]map[string]json.RawMessage
			if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
				t.Fatal(err)
			}
			raw, present := body["security_group_rule"]["remote_address_group_id"]
			if (test.want != "") != present || (present && string(raw) != test.want) {
				t.Fatal(body, test.want)
			}
		})
	}
}

func TestPythonSecurityRuleOptionSnapshotsHeadersAndPreflight(t *testing.T) {
	var calls []nativeRuleCall
	var headers []string
	api := pythonSecurityRuleAPI(t, &calls, func(req *http.Request) *http.Response {
		headers = append(headers, req.Header.Get("If-Match"))
		if req.Header.Get("X-Trace") != "trace" {
			t.Fatal(req.Header)
		}
		return nativeRuleWire(201, `{"security_group_rule":`+nativeRuleRow+`}`)
	})
	configured := map[string]string{"if-match": "old", "If-Match": "other", "X-Trace": "trace"}
	api.RawClient().MoreHeaders = configured
	values, conditions := []string{"original"}, []string{"first", "second"}
	bodyOption, headerOption := rules.WithAttribute("x_extension", values), rules.WithAttribute("if_match", conditions)
	values[0], conditions[0] = "changed", "changed"
	for i := 0; i < 2; i++ {
		if _, err := api.CreateWithAttributes(context.Background(), bodyOption, headerOption, rules.WithAttribute("description", "first"), rules.WithAttribute("description", "last")); err != nil {
			t.Fatal(err)
		}
	}
	for _, call := range calls {
		var body map[string]map[string]any
		if err := json.Unmarshal([]byte(call.body), &body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body["security_group_rule"], map[string]any{"x_extension": []any{"original"}, "description": "last"}) {
			t.Fatal(body)
		}
	}
	if !reflect.DeepEqual(headers, []string{"first, second", "first, second"}) || !reflect.DeepEqual(configured, map[string]string{"if-match": "old", "If-Match": "other", "X-Trace": "trace"}) {
		t.Fatal(headers, configured)
	}
	calls = nil
	for _, options := range [][]rules.AttributeOption{
		{nil}, {rules.WithAttribute(" ", 1)}, {rules.WithAttribute("x_extension", make(chan int))},
		{rules.WithAttribute("microversion", "2.0")}, {rules.WithAttribute("base_path", "/other")},
		{rules.WithAttribute("if_match", []int{1})}, {rules.WithAttribute("if_match", []any{"a", nil})}, {rules.WithIfMatch("bad\r\nheader")},
		{request.WithHeader[struct{}]("Authorization", "replaced")}, {request.WithField[struct{}]("If-Match", "body")},
		{request.WithQuery[struct{}]("fields", "id")}, {request.WithArgument[struct{}]("if_revision", int64(0))},
	} {
		_, err := api.CreateWithAttributes(context.Background(), options...)
		nativeRuleOperation(t, err, "CreateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	for _, test := range []struct {
		options []rules.AttributeOption
		header  string
	}{
		{[]rules.AttributeOption{rules.WithIfMatch()}, ""},
		{[]rules.AttributeOption{rules.WithAttribute("if_match", nil)}, "None"},
		{[]rules.AttributeOption{rules.WithIfMatch("first"), rules.WithAttribute("If-Match", "last")}, "last"},
	} {
		if _, err := api.CreateWithAttributes(context.Background(), test.options...); err != nil {
			t.Fatal(err)
		}
		if headers[len(headers)-1] != test.header {
			t.Fatal(headers, test.header)
		}
	}
}

func TestPythonSecurityRuleBulkBodyOrderLastHeaderAndSnapshots(t *testing.T) {
	var calls []nativeRuleCall
	var headers []string
	api := pythonSecurityRuleAPI(t, &calls, func(req *http.Request) *http.Response {
		headers = append(headers, req.Header.Get("If-Match"))
		return nativeRuleWire(201, `{"security_group_rules":[`+nativeRuleRow+`,{"id":"id-2"}]}`)
	})
	configured := map[string]string{"if-match": "configured", "X-Trace": "trace"}
	api.RawClient().MoreHeaders = configured
	data := [][]rules.AttributeOption{
		{rules.WithAttribute("ether_type", "IPv6"), rules.WithAttribute("remote_address_group_id", ""), rules.WithIfMatch("ignored-first")},
		{rules.WithAttribute("port_range_min", 0), rules.WithAttribute("remote_address_group_id", nil), rules.WithIfMatch("last")},
	}
	options := []rules.AttributeOption{rules.WithAttribute("x_extension", []string{"original"})}
	data[0] = append(data[0], func(*request.Config[struct{}]) error {
		data[1][0] = rules.WithAttribute("mutated", true)
		options[0] = rules.WithAttribute("mutated", true)
		return nil
	})
	data = append(data, options)
	values, err := api.CreateBulkWithAttributes(context.Background(), data...)
	if err != nil || len(values) != 2 || values[0].ID != "id-1" || values[1].ID != "id-2" {
		t.Fatal(values, err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/neutron/v2.0/security-group-rules" || calls[0].query != "" {
		t.Fatal(calls)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"security_group_rules": []any{
		map[string]any{"ethertype": "IPv6", "remote_address_group_id": ""},
		map[string]any{"port_range_min": float64(0), "remote_address_group_id": nil},
		map[string]any{"x_extension": []any{"original"}},
	}}
	if !reflect.DeepEqual(body, want) || !reflect.DeepEqual(headers, []string{"configured"}) || !reflect.DeepEqual(configured, map[string]string{"if-match": "configured", "X-Trace": "trace"}) {
		t.Fatal(body, headers, configured)
	}
	if _, err := api.CreateBulkWithAttributes(context.Background(), []rules.AttributeOption{rules.WithIfMatch("first")}, []rules.AttributeOption{rules.WithIfMatch("last")}); err != nil {
		t.Fatal(err)
	}
	if headers[len(headers)-1] != "last" {
		t.Fatal(headers)
	}
	if _, err := api.CreateBulkWithAttributes(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if calls[len(calls)-1].body != `{"security_group_rules":[{}]}` {
		t.Fatal(calls)
	}
}

func TestPythonSecurityRuleBulkRejectsInvalidRowsBeforeHTTP(t *testing.T) {
	var calls []nativeRuleCall
	api := pythonSecurityRuleAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(201, `{"security_group_rules":[]}`) })
	for _, rows := range [][][]rules.AttributeOption{
		nil, {}, {{nil}}, {nil, {rules.WithAttribute("x", make(chan int))}},
		{{rules.WithIfMatch("bad\r\nheader")}, {rules.WithIfMatch("last")}},
		{{rules.WithAttribute("if_match", []any{nil})}, {}},
		{{request.WithQuery[struct{}]("fields", "id")}}, {{rules.WithAttribute("microversion", "2.0")}},
	} {
		_, err := api.CreateBulkWithAttributes(context.Background(), rows...)
		nativeRuleOperation(t, err, "CreateBulkWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestPythonSecurityRuleMutationStatusesDecodeAndContextCauses(t *testing.T) {
	for _, bulk := range []bool{false, true} {
		operation := "CreateWithAttributes"
		if bulk {
			operation = "CreateBulkWithAttributes"
		}
		for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 400, 404, 409, 500} {
			t.Run(fmt.Sprintf("%s/%d", operation, code), func(t *testing.T) {
				var calls []nativeRuleCall
				body := `{"security_group_rule":` + nativeRuleRow + `}`
				if bulk {
					body = `{"security_group_rules":[` + nativeRuleRow + `]}`
				}
				api := pythonSecurityRuleAPI(t, &calls, func(*http.Request) *http.Response {
					reply := nativeRuleWire(code, body)
					reply.Header.Set("X-Request-Id", "rule-mutation")
					return reply
				})
				var single *rules.SecGroupRule
				var many []rules.SecGroupRule
				var err error
				if bulk {
					many, err = api.CreateBulkWithAttributes(context.Background(), nil)
				} else {
					single, err = api.CreateWithAttributes(context.Background())
				}
				switch {
				case code == 204:
					if err != nil || single != nil || many != nil {
						t.Fatal(single, many, err)
					}
				case code < 400:
					if err != nil || (bulk && (len(many) != 1 || many[0].ID != "id-1")) || (!bulk && (single == nil || single.ID != "id-1")) {
						t.Fatal(single, many, err)
					}
				default:
					nativeRuleOperation(t, err, operation)
					var status gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &status) || status.Actual != code || string(status.Body) != body || status.ResponseHeader.Get("X-Request-Id") != "rule-mutation" {
						t.Fatal(err, status)
					}
				}
				if len(calls) != 1 {
					t.Fatal(calls)
				}
			})
		}
		for _, lane := range []string{"pre-canceled", "empty option callback", "response boundary", "decode type", "bodyless JSON"} {
			t.Run(operation+"/"+lane, func(t *testing.T) {
				cause := errors.New("caller stopped rule mutation")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var calls []nativeRuleCall
				api := pythonSecurityRuleAPI(t, &calls, func(*http.Request) *http.Response {
					if lane == "response boundary" {
						cancel(cause)
					}
					if lane == "decode type" {
						if bulk {
							return nativeRuleWire(200, `{"security_group_rules":["invalid"]}`)
						}
						return nativeRuleWire(200, `{"security_group_rule":[]}`)
					}
					if lane == "bodyless JSON" {
						return nativeRuleWire(200, "")
					}
					if bulk {
						return nativeRuleWire(200, `{"security_group_rules":[`+nativeRuleRow+`]}`)
					}
					return nativeRuleWire(200, `{"security_group_rule":`+nativeRuleRow+`}`)
				})
				var options []rules.AttributeOption
				switch lane {
				case "pre-canceled":
					cancel(cause)
				case "empty option callback":
					options = []rules.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
				default:
					options = []rules.AttributeOption{rules.WithAttribute("description", "")}
				}
				var err error
				if bulk {
					_, err = api.CreateBulkWithAttributes(ctx, options)
				} else {
					_, err = api.CreateWithAttributes(ctx, options...)
				}
				nativeRuleOperation(t, err, operation)
				wantCalls := 1
				switch lane {
				case "decode type":
					var decode *json.UnmarshalTypeError
					if !errors.As(err, &decode) {
						t.Fatal(err)
					}
				case "bodyless JSON":
					if !errors.Is(err, io.EOF) {
						t.Fatal(err)
					}
				default:
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal(err)
					}
					if lane != "response boundary" {
						wantCalls = 0
					}
				}
				if len(calls) != wantCalls {
					t.Fatal(calls)
				}
			})
		}
	}
}
