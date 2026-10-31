package rules_test

import (
	"bytes"
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
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const findBase = "/neutron/v2.0/security-group-rules"

func findAPI(t *testing.T, calls *[]nativeRuleCall, reply func(*http.Request) *http.Response) (*rules.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	nativeRuleRecorder(cloud, calls, reply)
	return rules.New(nativeRuleClient(cloud)), cloud
}
func findValue(t *testing.T, record *rules.SecurityGroupRuleRecord, key, want string) {
	t.Helper()
	if record == nil || record.Resource == nil || string(record.Resource.Body[key]) != want {
		t.Fatalf("%s: %#v want %s", key, record, want)
	}
}
func TestPythonSecurityRuleFindRecordGetFirstRawKwargsSeedAndWire(t *testing.T) {
	var calls []nativeRuleCall
	var seen http.Header
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		seen = req.Header.Clone()
		response := nativeRuleWire(203, `{"security_group_rule":{"id":null,"name":false,"project_id":null,"tenant_id":"other","revision_number":"١٢","created_at":"not-native-time","x_new":9007199254740993,"port_range_min":false,"port_range_max":"٣٢","ether_type":"canonical","ethertype":"last","protocol":0,"remote_address_group_id":null,"self":"reserved","if_match":"body-collision"}}`)
		response.Header.Set("If-Match", "actual-header")
		response.Header.Set("X-Request-Id", "wire")
		return response
	})
	api.RawClient().Microversion = "3.4"
	record, err := api.FindRecord(context.Background(), "folder/name%", rules.WithRecordFindQuery("name", "seed-name"), rules.WithRecordFindQuery("x_cached", map[string]any{"false": false, "zero": 0}), rules.WithRecordFindQuery("if_match", []string{"cached"}), rules.WithRecordFindHeaders(map[string]string{"X-Trace": "owned"}))
	if err != nil || len(calls) != 1 || calls[0].method != "GET" || calls[0].body != "" || calls[0].path != findBase+"/folder/name%" || seen.Get("OpenStack-API-Version") != "network 3.4" || seen.Get("If-Match") != "" || seen.Get("X-Trace") != "owned" {
		t.Fatal(record, err, calls, seen)
	}
	query, _ := url.ParseQuery(calls[0].query)
	if query.Get("name") != "seed-name" || query.Get("if_match") != "cached" || !reflect.DeepEqual(query["x_cached"], []string{"false", "zero"}) {
		t.Fatal(query)
	}
	for key, want := range map[string]string{"id": "null", "name": "false", "project_id": "null", "tenant_id": `"other"`, "revision_number": "12", "created_at": `"not-native-time"`, "x_cached": `{"false":false,"zero":0}`, "x_new": "9007199254740993", "tags": "[]", "if_match": `["actual-header"]`, "location": "null", "port_range_min": "false", "port_range_max": "32", "ether_type": `"last"`, "protocol": "0", "remote_address_group_id": "null"} {
		findValue(t, record, key, want)
	}
	if record.Resource.Body["self"] != nil || record.Wire == nil || record.Wire.Body["x_cached"] != nil || string(record.Wire.Body["revision_number"]) != `"١٢"` || record.StatusCode != 203 || record.Header.Get("X-Request-Id") != "wire" {
		t.Fatal(record)
	}
	record.Resource.Body["x_new"][0] = 'X'
	record.Header.Set("X-Request-Id", "changed")
	if string(record.Wire.Body["x_new"]) != "9007199254740993" || record.Wire.Header.Get("X-Request-Id") != "wire" || record.Resource.Header.Get("X-Request-Id") != "wire" {
		t.Fatal(record)
	}
}
func TestPythonSecurityRuleFindRecordEmptyMalformedAndWrongShapes(t *testing.T) {
	for _, item := range []struct {
		body  string
		valid bool
	}{{"", true}, {"{bad", true}, {`{}`, true}, {`{"security_group_rule":{}}`, true}, {`null`, false}, {`[]`, false}, {`false`, false}, {`{"security_group_rule":null}`, false}, {`{"security_group_rule":[]}`, false}, {"{\"security_group_rule\":{\"name\":\"\xff\"}}", false}} {
		t.Run(fmt.Sprintf("%q", item.body), func(t *testing.T) {
			var calls []nativeRuleCall
			api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(202, item.body) })
			record, err := api.FindRecord(context.Background(), "seed", rules.WithRecordFindQuery("x_cached", false), rules.WithRecordFindQuery("if_match", []any{}))
			if (err == nil) != item.valid || record == nil || record.StatusCode != 202 || string(record.Envelope) != item.body || len(calls) != 1 {
				t.Fatal(item, record, err, calls)
			}
			if item.valid {
				findValue(t, record, "id", `"seed"`)
				findValue(t, record, "x_cached", "false")
				findValue(t, record, "if_match", `[]`)
			} else {
				var receipt *resource.ResponseError
				if !errors.As(err, &receipt) || receipt.StatusCode != 202 || string(receipt.Body) != item.body {
					t.Fatal(err, receipt)
				}
			}
		})
	}
}
func TestPythonSecurityRuleFindRecordFallbackCodesMissingAndCleanRetry(t *testing.T) {
	for _, code := range []int{400, 403, 404, 401, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeRuleCall
			api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path == findBase {
					return nativeRuleWire(200, `{"security_group_rules":[{"id":"wanted","name":"different"}]}`)
				}
				return nativeRuleWire(code, `{"missing":"actual"}`)
			})
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				return original
			}
			record, err := api.FindRecord(context.Background(), "wanted")
			if code == 401 || code == 409 {
				if record != nil || !gophercloud.ResponseCodeIs(err, code) || len(calls) != 1 {
					t.Fatal(record, err, calls)
				}
				return
			}
			if err != nil || record == nil || len(calls) != 2 {
				t.Fatal(record, err, calls)
			}
			query, _ := url.ParseQuery(calls[1].query)
			if query.Has("name") {
				t.Fatal(query)
			}
		})
	}
	var calls []nativeRuleCall
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == findBase {
			return nativeRuleWire(200, `{"security_group_rules":[]}`)
		}
		return nativeRuleWire(404, `{"missing":true}`)
	})
	record, err := api.FindRecord(context.Background(), "missing")
	if record != nil || err != nil || len(calls) != 2 {
		t.Fatal(record, err, calls)
	}
	record, err = api.FindRecord(context.Background(), "missing", rules.WithRecordFindIgnoreMissing(false))
	if record != nil || !errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, 404) || len(calls) != 4 {
		t.Fatal(record, err, calls)
	}
}
func TestPythonSecurityRuleFindRecordQueryAliasesScopeAndLocalFilters(t *testing.T) {
	var calls []nativeRuleCall
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != findBase {
			return nativeRuleWire(404, `{}`)
		}
		return nativeRuleWire(200, `{"security_group_rules":[{"id":"wanted","name":"wanted","created_at":{"enabled":1,"extra":false},"x_row":0}]}`)
	})
	record, err := api.FindRecord(context.Background(), "wanted", rules.WithRecordFindQuery("nativeRuleWireALIAS", "discarded"), rules.WithRecordFindQuery("nativeRuleClientALIAS", nil), rules.WithRecordFindQuery("any_tags", []any{"one", nil, "two"}), rules.WithRecordFindQuery("fields", []any{"name", nil, "id"}), rules.WithRecordFindQuery("created_at", map[string]any{"enabled": true}), rules.WithRecordFindQuery("x_unknown", map[string]any{"key": false}), rules.WithRecordFindQuery("all_projects", true))
	if err != nil || record == nil || len(calls) != 2 {
		t.Fatal(record, err, calls)
	}
	member, _ := url.ParseQuery(calls[0].query)
	list, _ := url.ParseQuery(calls[1].query)
	if member.Get("nativeRuleWireALIAS") != "discarded" || member.Has("nativeRuleClientALIAS") || member.Get("created_at") != "enabled" || member.Get("x_unknown") != "key" || member.Has("all_projects") || list.Has("nativeRuleWireALIAS") || list.Has("nativeRuleClientALIAS") || list.Has("created_at") || list.Has("x_unknown") || list.Has("all_projects") || !reflect.DeepEqual(list["tags-any"], []string{"one", "two"}) || !reflect.DeepEqual(list["fields"], []string{"name", "id"}) {
		t.Fatal(member, list)
	}
	var requests []nativeRuleCall
	api, _ = findAPI(t, &requests, func(req *http.Request) *http.Response {
		if req.URL.Path != findBase {
			return nativeRuleWire(404, `{}`)
		}
		return nativeRuleWire(200, `{"security_group_rules":[{"id":null,"name":"wanted"}]}`)
	})
	record, err = api.FindRecord(context.Background(), "wanted", rules.WithRecordFindQuery("name", nil))
	if err != nil || record != nil || len(requests) != 2 {
		t.Fatal(record, err, requests)
	}
	// Explicit name=None is server omission for a group and a local None filter
	// for a rule; neither can accidentally become an automatic name hint.
	query, _ := url.ParseQuery(requests[1].query)
	if query.Has("name") {
		t.Fatal(query)
	}
}
func TestPythonSecurityRuleFindRecordLateDuplicateAndPageFailure(t *testing.T) {
	for _, lane := range []string{"unique", "duplicate", "error"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeRuleCall
			api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path != findBase {
					return nativeRuleWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") == "next" {
					if lane == "error" {
						return nativeRuleWire(500, `{"late":"actual"}`)
					}
					if lane == "duplicate" {
						return nativeRuleWire(200, `{"security_group_rules":[{"id":"another","name":"wanted"}]}`)
					}
					return nativeRuleWire(200, `{"security_group_rules":[{"id":"other","name":"other"}]}`)
				}
				return nativeRuleWire(200, `{"security_group_rules":[{"id":"wanted","name":"different"}],"security_group_rules_links":[{"rel":"next","href":"?marker=next"}]}`)
			})
			record, err := api.FindRecord(context.Background(), "wanted")
			if len(calls) != 3 {
				t.Fatal(calls)
			}
			if lane == "unique" {
				if err != nil || record == nil {
					t.Fatal(record, err)
				}
				findValue(t, record, "id", `"wanted"`)
			} else {
				if record != nil || err == nil || lane == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) || lane == "error" && !gophercloud.ResponseCodeIs(err, 500) {
					t.Fatal(record, err)
				}
			}
		})
	}
}
func TestPythonSecurityRuleFindRecordMarkerCapAndSinglePage(t *testing.T) {
	var calls []nativeRuleCall
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != findBase {
			return nativeRuleWire(404, `{}`)
		}
		if req.URL.Query().Get("marker") == "wanted" {
			return nativeRuleWire(200, `{"security_group_rules":[]}`)
		}
		return nativeRuleWire(200, `{"security_group_rules":{"id":"wanted","name":"wanted"}}`)
	})
	record, err := api.FindRecord(context.Background(), "wanted", rules.WithRecordFindLimit(5))
	if err != nil || record == nil || len(calls) != 3 || !strings.Contains(calls[2].query, "marker=wanted") || calls[0].query != "" {
		t.Fatal(record, err, calls)
	}
	calls = nil
	record, err = api.FindRecord(context.Background(), "wanted", rules.WithRecordFindLimit(5), rules.WithRecordFindPaginated(false))
	if err != nil || record == nil || len(calls) != 2 {
		t.Fatal(record, err, calls)
	}
	calls = nil
	api, _ = findAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != findBase {
			return nativeRuleWire(404, `{}`)
		}
		return nativeRuleWire(200, `{"security_group_rules":[{"id":"skip","name":"wanted","created_at":false},{"id":"wanted","name":"wanted","created_at":1},{"id":"bad","name":"wanted","connection":null}],"next":"https://bad.example/security-group-rules"}`)
	})
	record, err = api.FindRecord(context.Background(), "wanted", rules.WithRecordFindQuery("created_at", true), rules.WithRecordFindMaxItems(2), rules.WithRecordFindLimit(0))
	if err != nil || record == nil || len(calls) != 2 {
		t.Fatal(record, err, calls)
	}
	findValue(t, record, "id", `"wanted"`)
	q, _ := url.ParseQuery(calls[1].query)
	if q.Get("limit") != "2" {
		t.Fatal(q)
	}
}
func TestPythonSecurityRuleFindRecordPaginationRepresentationsAndGuards(t *testing.T) {
	for _, lane := range []string{"plural", "next", "http", "dictionary", "versioned", "empty", "filter", "path", "cycle"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeRuleCall
			body := `{"security_group_rules":[{"id":"first","name":"other"}]}`
			api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path != findBase {
					return nativeRuleWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") != "" {
					if lane == "cycle" {
						return nativeRuleWire(200, `{"security_group_rules":[{"id":"again"}],"next":"?marker=next"}`)
					}
					return nativeRuleWire(200, `{"security_group_rules":[{"id":"wanted","name":"wanted"}]}`)
				}
				response := nativeRuleWire(200, body)
				switch lane {
				case "plural":
					response = nativeRuleWire(200, `{"security_group_rules":[{"id":"first"}],"security_group_rules_links":[{"rel":"next","href":"?marker=next"}]}`)
				case "next", "cycle":
					response = nativeRuleWire(200, `{"security_group_rules":[{"id":"first"}],"next":"?marker=next"}`)
				case "http":
					response.Header.Set("Link", "<?marker=next>; rel=\"next\"")
				case "dictionary":
					response = nativeRuleWire(200, `{"security_group_rules":[{"id":"first"}],"links":{"next":"?marker=next"}}`)
				case "versioned":
					response = nativeRuleWire(200, `{"security_group_rules":[{"id":"first"}],"next":"/v2.0/security-group-rules?marker=next"}`)
				case "empty":
					response = nativeRuleWire(200, `{"security_group_rules":[],"next":"https://bad.example/security-group-rules"}`)
				case "filter":
					response = nativeRuleWire(200, `{"security_group_rules":[{"id":"first"}],"next":"?marker=next&project_id=changed"}`)
				case "path":
					response = nativeRuleWire(200, `{"security_group_rules":[{"id":"first"}],"next":"/other/security-group-rules?marker=next"}`)
				}
				return response
			})
			record, err := api.FindRecord(context.Background(), "wanted", rules.WithRecordFindQuery("project_id", "fixed"))
			if lane == "filter" || lane == "path" || lane == "cycle" {
				var receipt *resource.ResponseError
				if record != nil || err == nil || !errors.As(err, &receipt) || receipt.StatusCode != 200 {
					t.Fatal(record, err, receipt, calls)
				}
			} else if lane == "empty" {
				if record != nil || err != nil || len(calls) != 2 {
					t.Fatal(record, err, calls)
				}
			} else if record == nil || err != nil || len(calls) != 3 {
				t.Fatal(record, err, calls)
			}
		})
	}
}
func TestPythonSecurityRuleFindRecordListHeadersAndConstructorControls(t *testing.T) {
	var calls []nativeRuleCall
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != findBase {
			return nativeRuleWire(404, `{}`)
		}
		response := nativeRuleWire(200, `{"security_group_rules":[{"id":null,"name":"wanted","if_match":"row-header","self":"reserved","x_row":9007199254740993,"port_range_min":false,"port_range_max":"٣٢","ether_type":"canonical","ethertype":"last","protocol":0,"remote_address_group_id":null}]}`)
		response.Header.Set("If-Match", "actual-page-header")
		return response
	})
	record, err := api.FindRecord(context.Background(), "wanted")
	if err != nil || record == nil || record.StatusCode != 200 || record.Header.Get("If-Match") != "actual-page-header" {
		t.Fatal(record, err)
	}
	findValue(t, record, "if_match", `["row-header"]`)
	findValue(t, record, "id", "null")
	if record.Resource.Body["self"] != nil || record.Wire.Body["self"] == nil {
		t.Fatal(record)
	}
	for _, row := range []string{`null`, `{"id":"wanted","connection":null}`, `{"id":"wanted","microversion":null}`, `{"id":"wanted","_synchronized":null}`, `{"id":"wanted","revision_number":"²"}`} {
		t.Run(row, func(t *testing.T) {
			var requests []nativeRuleCall
			api, _ := findAPI(t, &requests, func(req *http.Request) *http.Response {
				if req.URL.Path != findBase {
					return nativeRuleWire(404, `{}`)
				}
				return nativeRuleWire(200, `{"security_group_rules":[`+row+`]}`)
			})
			value, err := api.FindRecord(context.Background(), "wanted")
			var receipt *resource.ResponseError
			if value != nil || err == nil || len(requests) != 2 || !errors.As(err, &receipt) || !bytes.Contains(receipt.Body, []byte(row)) {
				t.Fatal(value, err, requests, receipt)
			}
		})
	}
}
func TestPythonSecurityRuleFindRecordDictionaryTruthinessAndEquality(t *testing.T) {
	for _, item := range []struct {
		actual, filter string
		match, valid   bool
	}{{`"truthy"`, `{}`, true, true}, {`"truthy"`, `{"k":true}`, false, false}, {`""`, `{}`, false, true}, {`1e-4000`, `{}`, false, true}, {`1e4000`, `{}`, true, true}, {`null`, `null`, true, true}, {`1`, `true`, true, true}, {`{"nested":{"enabled":1,"extra":false}}`, `{"nested":{"enabled":true}}`, true, true}, {`{"nested":{}}`, `{"nested":{}}`, false, true}} {
		t.Run(item.actual+item.filter, func(t *testing.T) {
			var calls []nativeRuleCall
			api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path != findBase {
					return nativeRuleWire(404, `{}`)
				}
				return nativeRuleWire(200, `{"security_group_rules":[{"id":"wanted","name":"wanted","created_at":`+item.actual+`}]}`)
			})
			record, err := api.FindRecord(context.Background(), "wanted", rules.WithRecordFindQuery("created_at", json.RawMessage(item.filter)))
			if (record != nil) != item.match || (err == nil) != item.valid {
				t.Fatal(item, record, err)
			}
			if !item.valid {
				var receipt *resource.ResponseError
				if !errors.As(err, &receipt) || receipt.StatusCode != 200 {
					t.Fatal(err, receipt)
				}
			}
		})
	}
}
func TestPythonSecurityRuleFindRecordInvalidControlsAndDescriptorFailBeforeHTTP(t *testing.T) {
	var calls []nativeRuleCall
	api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(200, `{}`) })
	for _, options := range [][]rules.RecordFindOption{{nil}, {rules.WithRecordFindLimit(-1)}, {rules.WithRecordFindMaxItems(-1)}, {rules.WithRecordFindQuery("id", "other")}, {rules.WithRecordFindQuery("connection", nil)}, {rules.WithRecordFindQuery("base_path", "/other")}, {rules.WithRecordFindQuery("headers", map[string]any{})}, {rules.WithRecordFindQuery("microversion", "3.4")}, {rules.WithRecordFindQuery("revision_number", json.RawMessage(`"²"`))}, {rules.WithRecordFindQuery("x_unknown", json.RawMessage(`invalid`))}, {rules.WithRecordFindHeaders(map[string]string{"X-Auth-Token": "bad"})}, {rules.WithRecordFindHeaders(map[string]string{"OpenStack-API-Version": "network 3.4"})}} {
		_, err := api.FindRecord(context.Background(), "wanted", options...)
		if err == nil || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	}
	for _, identity := range []string{"", "..", "a\n", "\xff"} {
		_, err := api.FindRecord(context.Background(), identity)
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	}
	api.RawClient().Type = "compute"
	_, err := api.FindRecord(context.Background(), "wanted")
	if !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}
func TestPythonSecurityRuleFindRecordOptionsSnapshotCancellationAndMarshal(t *testing.T) {
	var calls []nativeRuleCall
	var headers http.Header
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response { headers = req.Header.Clone(); return nativeRuleWire(204, "") })
	input := map[string]any{"false": false}
	query := map[string]json.RawMessage{"x_cached": json.RawMessage(`{"false":false}`)}
	version := "3.4"
	option := rules.WithRecordFindOpts(rules.RecordFindOpts{Query: query, Headers: map[string]string{"X-Trace": "original"}, Microversion: &version})
	query["x_cached"][9] = 'X'
	version = "changed"
	var retained *rules.RecordFindOpts
	record, err := api.FindRecord(context.Background(), "wanted", option, rules.WithRecordFindQuery("x_input", input), func(v *rules.RecordFindOpts) error { retained = v; input["false"] = true; return nil }, func(*rules.RecordFindOpts) error {
		retained.Headers["X-Auth-Token"] = "later"
		retained.Query["x_cached"] = json.RawMessage(`true`)
		*retained.Microversion = "changed"
		return nil
	})
	if err != nil || record == nil || len(calls) != 1 || headers.Get("X-Trace") != "original" || headers.Get("OpenStack-API-Version") != "network 3.4" {
		t.Fatal(record, err, calls, headers)
	}
	findValue(t, record, "x_cached", `{"false":false}`)
	findValue(t, record, "x_input", `{"false":false}`)
	calls = nil
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callback := errors.New("cancel cause"), errors.New("callback")
	_, err = api.FindRecord(ctx, "wanted", func(*rules.RecordFindOpts) error { cancel(cause); return callback })
	if len(calls) != 0 || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callback) {
		t.Fatal(err, calls)
	}
	original := api.RawClient().ResourceBase
	defer func() { api.RawClient().ResourceBase = original }()
	_, err = api.FindRecord(context.Background(), "wanted", func(*rules.RecordFindOpts) error { api.RawClient().ResourceBase += "changed/"; return callback })
	if len(calls) != 0 || !errors.Is(err, callback) || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err, calls)
	}
	api.RawClient().ResourceBase = original
	_, err = api.FindRecord(context.Background(), "wanted", rules.WithRecordFindQuery("x_marshal", findMarshaler(func() ([]byte, error) { return nil, callback })))
	if len(calls) != 0 || !errors.Is(err, callback) {
		t.Fatal(err, calls)
	}
}
func TestPythonSecurityRuleFindRecordConcurrentOptionAndRecordOwnership(t *testing.T) {
	var calls []nativeRuleCall
	api, cloud := findAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(200, `{}`) })
	var attempts atomic.Int64
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		attempts.Add(1)
		if req.Header.Get("X-Trace") != "owned" {
			t.Error(req.Header)
		}
		return nativeRuleWire(200, `{"security_group_rule":{"id":"wanted","x_row":{"value":0}}}`), nil
	}))
	option := rules.WithRecordFindOpts(rules.RecordFindOpts{Query: map[string]json.RawMessage{"x_cached": json.RawMessage(`false`)}, Headers: map[string]string{"X-Trace": "owned"}})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			record, err := api.FindRecord(context.Background(), "wanted", option)
			if err != nil || record == nil {
				t.Error(record, err)
				return
			}
			record.Resource.Body["x_row"][0] = 'X'
			if string(record.Wire.Body["x_row"]) != `{"value":0}` {
				t.Error(record)
			}
		}()
	}
	wg.Wait()
	if attempts.Load() != 8 {
		t.Fatal(attempts.Load())
	}
}

type findFaultBody struct {
	data              []byte
	readErr, closeErr error
	onRead, onClose   func()
	closed            int
}

func (b *findFaultBody) Read(p []byte) (int, error) {
	if len(b.data) > 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		if b.onRead != nil {
			b.onRead()
			b.onRead = nil
		}
		return n, nil
	}
	if b.readErr != nil {
		return 0, b.readErr
	}
	return 0, io.EOF
}
func (b *findFaultBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type findMarshaler func() ([]byte, error)

func (f findMarshaler) MarshalJSON() ([]byte, error) { return f() }
func TestPythonSecurityRuleFindRecordPhysicalAcceptedRejectedAndListErrors(t *testing.T) {
	for _, phase := range []string{"member", "list"} {
		for _, status := range []int{200, 400, 403, 404, 409, 503} {
			for _, lane := range []string{"read", "close", "read+close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", phase, status, lane), func(t *testing.T) {
					var calls []nativeRuleCall
					api, cloud := findAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(200, `{}`) })
					readCause, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("custom cancel")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					client := api.RawClient()
					original := client.ResourceBase
					defer func() { client.ResourceBase = original }()
					physical := &findFaultBody{data: []byte(`{"actual":"physical"}`)}
					switch lane {
					case "read":
						physical.readErr = readCause
					case "close":
						physical.closeErr = closeCause
					case "read+close":
						physical.readErr = readCause
						physical.closeErr = closeCause
					case "cancel":
						physical.onRead = func() { cancel(cancelCause) }
					case "source":
						physical.onRead = func() { client.ResourceBase += "changed/" }
						physical.onClose = func() { client.ResourceBase = original }
					}
					attempts := 0
					cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
						attempts++
						if phase == "list" && req.URL.Path != findBase {
							return nativeRuleWire(404, `{}`), nil
						}
						return testcloud.StreamResponse(status, physical, http.Header{"X-Request-Id": {"physical"}}), nil
					}))
					record, err := api.FindRecord(ctx, "wanted")
					want := 1
					if phase == "list" {
						want = 2
					}
					if err == nil || attempts != want || physical.closed != 1 {
						t.Fatal(record, err, attempts, physical.closed)
					}
					if status < 400 {
						var receipt *resource.ResponseError
						if !errors.As(err, &receipt) || receipt.StatusCode != status || string(receipt.Body) != `{"actual":"physical"}` || receipt.Header.Get("X-Request-Id") != "physical" || phase == "member" && (record == nil || record.Resource != nil) || phase == "list" && record != nil {
							t.Fatal(record, err, receipt)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if record != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != `{"actual":"physical"}` || native.ResponseHeader.Get("X-Request-Id") != "physical" {
							t.Fatal(record, err, native)
						}
					}
					if (lane == "read" || lane == "read+close") && !errors.Is(err, readCause) || (lane == "close" || lane == "read+close") && !errors.Is(err, closeCause) || lane == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) || lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				})
			}
		}
	}
}
func TestPythonSecurityRuleFindRecordRetryKeepsOriginalRejectionAndRequest(t *testing.T) {
	for _, lane := range []string{"body", "version", "custom", "codes", "retry", "reauth"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeRuleCall
			var tokens []string
			api, cloud := findAPI(t, &calls, func(req *http.Request) *http.Response {
				tokens = append(tokens, req.Header.Get("X-Auth-Token"))
				if len(calls) == 1 {
					if lane == "reauth" {
						return nativeRuleWire(401, `{"expired":true}`)
					}
					return nativeRuleWire(503, `{"retry":"actual"}`)
				}
				if lane == "codes" {
					return nativeRuleWire(400, `{"outside":"original policy"}`)
				}
				return nativeRuleWire(200, `{"security_group_rule":{"id":"wanted"}}`)
			})
			api.RawClient().Microversion = "3.4"
			callback := errors.New("retry callback")
			if lane == "reauth" {
				cloud.Provider.ReauthFunc = func(context.Context) error { cloud.Provider.SetToken("renewed"); return nil }
			} else {
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					switch lane {
					case "body":
						opts.JSONBody = map[string]any{"changed": true}
					case "version":
						opts.MoreHeaders["OpenStack-API-Version"] = "network 3.5"
					case "custom":
						return callback
					case "codes":
						opts.OkCodes = append(opts.OkCodes, 400)
					}
					return nil
				}
			}
			record, err := api.FindRecord(context.Background(), "wanted", rules.WithRecordFindQuery("x_query", false))
			if lane == "retry" || lane == "reauth" {
				if err != nil || record == nil || len(calls) != 2 || calls[0] != calls[1] {
					t.Fatal(record, err, calls)
				}
				if lane == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
					t.Fatal(tokens)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				wantCode, wantCalls := 503, 1
				if lane == "codes" {
					wantCode, wantCalls = 400, 2
				}
				if record != nil || !errors.As(err, &native) || native.Actual != wantCode || len(calls) != wantCalls {
					t.Fatal(record, err, native, calls)
				}
				if lane == "custom" {
					if !errors.Is(err, callback) {
						t.Fatal(err)
					}
				} else if lane != "codes" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestPythonSecurityRuleFindRecordNullableIdentityAndLiteralEscape(t *testing.T) {
	for _, row := range []string{`{"id":null,"name":"wanted"}`, `{"id":42,"name":"wanted"}`, `{"name":"wanted"}`} {
		var calls []nativeRuleCall
		api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.Path != findBase {
				return nativeRuleWire(404, `{}`)
			}
			return nativeRuleWire(200, `{"security_group_rules":[`+row+`]}`)
		})
		record, err := api.FindRecord(context.Background(), "wanted")
		if err != nil || record == nil {
			t.Fatal(row, record, err, calls)
		}
	}
	var calls []nativeRuleCall
	var escaped string
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		escaped = req.URL.EscapedPath()
		return nativeRuleWire(299, `{"security_group_rule":{"id":null}}`)
	})
	record, err := api.FindRecord(context.Background(), "folder/name%")
	if err != nil || record == nil || record.StatusCode != 299 || escaped != findBase+"/folder%2Fname%25" || len(calls) != 1 {
		t.Fatal(record, err, escaped, calls)
	}
}
func TestPythonSecurityRuleFindRecordProtocolAbsenceAndPerCallVersion(t *testing.T) {
	var calls []nativeRuleCall
	var seen []string
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		seen = append(seen, req.Header.Get("OpenStack-API-Version"))
		if req.URL.Path == findBase {
			return nativeRuleWire(200, `{"security_group_rules":[{"id":"wanted"}]}`)
		}
		return nativeRuleWire(404, `{}`)
	})
	api.RawClient().Microversion = "3.4"
	record, err := api.FindRecord(context.Background(), "wanted", rules.WithRecordFindMicroversion(""))
	if err != nil || record == nil || len(calls) != 2 || !reflect.DeepEqual(seen, []string{"", "network 3.4"}) || api.RawClient().Microversion != "3.4" {
		t.Fatal(record, err, calls, seen)
	}
	calls = nil
	seen = nil
	record, err = api.FindRecord(context.Background(), "wanted", rules.WithRecordFindMicroversion("latest"))
	if err != nil || record == nil || len(calls) != 2 || !reflect.DeepEqual(seen, []string{"network latest", "network 3.4"}) || api.RawClient().Microversion != "3.4" {
		t.Fatal(record, err, calls, seen)
	}
	api.RawClient().Microversion = ""
	api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.MoreHeaders["OpenStack-API-Version"] = "network changed"
		return nil
	}
	calls = nil
	_, err = api.FindRecord(context.Background(), "wanted")
	if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 404) || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}

func TestPythonSecurityRuleFindRecordSourceArgumentCollisions(t *testing.T) {
	var calls []nativeRuleCall
	api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("preflight reached HTTP"); return nil })
	for _, key := range []string{"ignore_missing", "name_or_id", "session", "cls"} {
		record, err := api.FindRecord(context.Background(), "wanted", rules.WithRecordFindQuery(key, false))
		if record != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, record, err, calls)
		}
	}
	var listed []nativeRuleCall
	body := `{"security_group_rules":[{"id":"wanted","cls":null}]}`
	api, _ = findAPI(t, &listed, func(req *http.Request) *http.Response {
		if req.URL.Path != findBase {
			return nativeRuleWire(404, `{}`)
		}
		response := nativeRuleWire(203, body)
		response.Header.Set("X-Request-Id", "constructor-cls")
		return response
	})
	record, err := api.FindRecord(context.Background(), "wanted")
	var proof *resource.ResponseError
	if record != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || len(listed) != 2 || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "constructor-cls" {
		t.Fatal(record, err, proof, listed)
	}
}

func TestPythonSecurityRuleFindRecordHeaderAliasDomains(t *testing.T) {
	for _, lane := range []string{"seed", "row", "actual-header"} {
		for _, key := range []string{"if_match", "if-match", "IF-MATCH", "If-MaTcH", "IF_MATCH"} {
			t.Run(lane+"/"+key, func(t *testing.T) {
				var calls []nativeRuleCall
				alias, _ := json.Marshal(key)
				api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
					if lane == "row" && req.URL.Path != findBase {
						return nativeRuleWire(404, `{}`)
					}
					body := `{"security_group_rule":{}}`
					if lane == "row" {
						body = `{"security_group_rules":[{"id":"wanted",` + string(alias) + `:"value"}]}`
					}
					response := nativeRuleWire(200, body)
					if lane == "actual-header" {
						response.Header[key] = []string{"first", "second"}
					} else if lane == "row" {
						response.Header.Set("If-Match", "page-ignored")
					}
					return response
				})
				options := []rules.RecordFindOption{}
				if lane == "seed" {
					options = append(options, rules.WithRecordFindQuery(key, "value"))
				}
				record, err := api.FindRecord(context.Background(), "wanted", options...)
				if err != nil || record == nil {
					t.Fatal(record, err, calls)
				}
				wanted := `["value"]`
				if lane == "actual-header" {
					wanted = `["first, second"]`
				}
				if key == "IF_MATCH" {
					wanted = `null`
				}
				findValue(t, record, "if_match", wanted)
				if key == "IF_MATCH" && lane != "actual-header" {
					findValue(t, record, "IF_MATCH", `"value"`)
				}
				if key == "IF_MATCH" && lane == "actual-header" && record.Resource.Body[key] != nil {
					t.Fatal("unknown response Header became Body", record)
				}
				if lane == "seed" {
					query, _ := url.ParseQuery(calls[0].query)
					if query.Get(key) != "value" {
						t.Fatal(calls)
					}
				}
			})
		}
	}
	for _, order := range []string{`"if_match":"first","IF-MATCH":"last","IF_MATCH":"extension"`, `"IF-MATCH":"first","if_match":"last","IF_MATCH":"extension"`} {
		var calls []nativeRuleCall
		api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.Path != findBase {
				return nativeRuleWire(404, `{}`)
			}
			return nativeRuleWire(200, `{"security_group_rules":[{"id":"wanted",`+order+`}]}`)
		})
		record, err := api.FindRecord(context.Background(), "wanted")
		if err != nil || record == nil {
			t.Fatal(record, err, calls)
		}
		findValue(t, record, "if_match", `["last"]`)
		findValue(t, record, "IF_MATCH", `"extension"`)
	}
}

func TestPythonSecurityRuleFindRecordMicroversionPhasesAndAcceptDefault(t *testing.T) {
	for _, selected := range []string{"", "3.4"} {
		for _, explicit := range []string{"", "latest", "3.5"} {
			for _, callerAccept := range []bool{false, true} {
				t.Run(selected+"/"+explicit+"/"+fmt.Sprint(callerAccept), func(t *testing.T) {
					var calls []nativeRuleCall
					var versions, accepts []string
					api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
						versions = append(versions, req.Header.Get("OpenStack-API-Version"))
						accepts = append(accepts, req.Header.Get("Accept"))
						if req.URL.Path != findBase {
							return nativeRuleWire(404, `{}`)
						}
						return nativeRuleWire(200, `{"security_group_rules":[{"id":"wanted"}]}`)
					})
					source := api.RawClient()
					source.Microversion = selected
					source.MoreHeaders = map[string]string{"Accept": "text/xml"}
					if selected != "" {
						source.MoreHeaders["oPeNsTaCk-ApI-vErSiOn"] = "network " + selected
					}
					options := []rules.RecordFindOption{rules.WithRecordFindMicroversion(explicit)}
					wantAccept := []string{"text/xml", "application/json"}
					if callerAccept {
						options = append(options, rules.WithRecordFindHeaders(map[string]string{"aCcEpT": "text/plain"}))
						wantAccept = []string{"text/plain", "text/plain"}
					}
					record, err := api.FindRecord(context.Background(), "wanted", options...)
					memberVersion, listVersion := "", ""
					if explicit != "" {
						memberVersion = "network " + explicit
					}
					if selected != "" {
						listVersion = "network " + selected
					}
					if err != nil || record == nil || len(calls) != 2 || !reflect.DeepEqual(versions, []string{memberVersion, listVersion}) || !reflect.DeepEqual(accepts, wantAccept) || source.Microversion != selected || source.MoreHeaders["Accept"] != "text/xml" {
						t.Fatal(record, err, calls, versions, accepts, source)
					}
					if selected != "" && source.MoreHeaders["oPeNsTaCk-ApI-vErSiOn"] != "network "+selected {
						t.Fatal(source.MoreHeaders)
					}
				})
			}
		}
	}
}

func TestPythonSecurityRuleFindRecordRetryForbiddenHeadersAndOmittedVersion(t *testing.T) {
	for _, list := range []bool{false, true} {
		for _, key := range []string{"aUtHoRiZaTiOn", "x-SeRvIcE-ToKeN", "hOsT", "Cookie", "content-length", "TRANSFER-ENCODING", "Connection", "Trailer", "Te", "Upgrade", "x-OpenStack-Nova-Api-Version", "x-OpenStack-Volume-Api-Version", "omit-selected-version"} {
			for _, custom := range []bool{false, true} {
				t.Run(fmt.Sprintf("%t/%s/%t", list, key, custom), func(t *testing.T) {
					var calls []nativeRuleCall
					callback := errors.New("callback cause")
					body := `{"error":"original rejection"}`
					api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
						if list && req.URL.Path != findBase {
							return nativeRuleWire(404, `{}`)
						}
						response := nativeRuleWire(503, body)
						response.Header.Set("X-Request-Id", "fixed-policy-rejection")
						return response
					})
					api.RawClient().Microversion = "3.4"
					api.RawClient().RetryFunc = func(_ context.Context, _ string, endpoint string, opts *gophercloud.RequestOpts, original error, _ uint) error {
						parsed, err := url.Parse(endpoint)
						if err != nil {
							t.Fatal(err)
						}
						if list && parsed.Path != findBase {
							return original
						}
						if key == "omit-selected-version" {
							opts.OmitHeaders = []string{"oPeNsTaCk-ApI-vErSiOn"}
						} else {
							opts.MoreHeaders[key] = "forbidden"
						}
						if custom {
							return callback
						}
						return nil
					}
					record, err := api.FindRecord(context.Background(), "wanted", rules.WithRecordFindMicroversion("3.5"))
					var native gophercloud.ErrUnexpectedResponseCode
					wantCalls := 1
					if list {
						wantCalls = 2
					}
					if record != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, callback) != custom || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "fixed-policy-rejection" || len(calls) != wantCalls {
						t.Fatal(record, err, native, calls)
					}
				})
			}
		}
	}
}
