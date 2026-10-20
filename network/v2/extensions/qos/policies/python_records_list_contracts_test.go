package policies_test

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

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonQoSPolicyListRecordsLazyQueryClassificationAndMRODefaults(t *testing.T) {
	var calls []nativePolicyCall
	body := `{"policies":[{"tenant_id":"owner","rules":{"type":"bandwidth_limit","max_kbps":9007199254740993},"created_at":"not-a-native-time","revision_number":1.75,"x_new":{"nested":null},"self":"literal"}]}`
	api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
		want := url.Values{"id": {"literal/id"}, "fields": {"is_shared", "tags", "x_new"}, "shared": {"False"}, "tags-any": {"a", "b"}, "project_id": {"owner"}, "is_default": {"False"}, "sort_key": {"name"}, "sort_dir": {"desc"}}
		if req.URL.EscapedPath() != "/neutron/v2.0/qos/policies" || !reflect.DeepEqual(req.URL.Query(), want) || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header, want)
		}
		response := nativePolicyWire(203, body)
		response.Header.Set("If-Match", "HTTP-header-only")
		response.Header.Set("X-Request-Id", "page-receipt")
		return response
	})
	api.RawClient().MoreHeaders = map[string]string{"Accept": "text/xml"}
	sequence := api.ListRecords(context.Background(), policies.WithRecordListQuery("id", "literal/id"), policies.WithRecordListQuery("fields", []string{"is_shared", "tags", "x_new"}), policies.WithRecordListQuery("shared", true), policies.WithRecordListQuery("is_shared", false), policies.WithRecordListQuery("any_tags", []any{"a", nil, "b"}), policies.WithRecordListQuery("project_id", "owner"), policies.WithRecordListQuery("is_default", false), policies.WithRecordListQuery("sort_key", "name"), policies.WithRecordListQuery("sort_dir", "desc"), policies.WithRecordListQuery("ignored_function", func() {}), policies.WithRecordListOpts(policies.RecordListOpts{}))
	// A bulk replacement clears earlier list kwargs without eagerly doing HTTP.
	if len(calls) != 0 {
		t.Fatal("list requested before consumption", calls)
	}
	sequence = api.ListRecords(context.Background(), policies.WithRecordListQuery("id", "literal/id"), policies.WithRecordListQuery("fields", []string{"is_shared", "tags", "x_new"}), policies.WithRecordListQuery("shared", true), policies.WithRecordListQuery("is_shared", false), policies.WithRecordListQuery("any_tags", []any{"a", nil, "b"}), policies.WithRecordListQuery("project_id", "owner"), policies.WithRecordListQuery("is_default", false), policies.WithRecordListQuery("sort_key", "name"), policies.WithRecordListQuery("sort_dir", "desc"), policies.WithRecordListQuery("ignored_function", func() {}), policies.WithRecordListQuery("status", "ignored"), policies.WithRecordListQuery("revision_number", 4), policies.WithRecordListQuery("created_at", "ignored"), policies.WithRecordListQuery("if_match", "ordinary-unknown"))
	var observed []*policies.PolicyRecord
	for value, err := range sequence {
		if err != nil {
			t.Fatal(err)
		}
		observed = append(observed, value)
	}
	if len(calls) != 1 || len(observed) != 1 {
		t.Fatal(calls, observed)
	}
	got := observed[0]
	pythonFindField(t, got, "id", `null`)
	pythonFindField(t, got, "name", `null`)
	pythonFindField(t, got, "project_id", `"owner"`)
	pythonFindField(t, got, "is_shared", `null`)
	pythonFindField(t, got, "is_default", `null`)
	pythonFindField(t, got, "tags", `[]`)
	pythonFindField(t, got, "rules", `[{"type":"bandwidth_limit","max_kbps":9007199254740993}]`)
	pythonFindField(t, got, "created_at", `"not-a-native-time"`)
	pythonFindField(t, got, "revision_number", `1.75`)
	pythonFindField(t, got, "location", `null`)
	pythonFindWireField(t, got, "self", `"literal"`)
	if got.Resource.Body["self"] != nil || got.Resource.Body["if_match"] != nil || got.Wire.Body["id"] != nil || got.StatusCode != 203 || string(got.Envelope) != body || got.Header.Get("X-Request-Id") != "page-receipt" {
		t.Fatal(got)
	}
	got.Header.Set("X-Request-Id", "changed")
	got.Resource.Header.Set("If-Match", "changed")
	if got.Wire.Header.Get("X-Request-Id") != "page-receipt" || got.Wire.Header.Get("If-Match") != "HTTP-header-only" || api.RawClient().MoreHeaders["Accept"] != "text/xml" {
		t.Fatal(got, api.RawClient())
	}
}

func TestPythonQoSPolicyListRecordsRulesAndTenantLocalFilterSemantics(t *testing.T) {
	cases := []struct {
		name, row, key, filter string
		matched, failed        bool
	}{
		{"full ordered rules", `"rules":[{"max_kbps":9007199254740993,"nested":{"bool":true}}]`, "rules", `[{"max_kbps":9007199254740993,"nested":{"bool":1}}]`, true, false},
		{"large exact rule mismatch", `"rules":[{"max_kbps":9007199254740993}]`, "rules", `[{"max_kbps":9007199254740992}]`, false, false},
		{"list descriptor wraps dictionary", `"rules":{"x":null}`, "rules", `[{"x":null}]`, true, false},
		{"rules missing is null", `"x":1`, "rules", `null`, true, false},
		{"explicit rules null", `"rules":null`, "rules", `null`, true, false},
		{"empty rules dictionary filter false", `"rules":[]`, "rules", `{}`, false, false},
		{"nonempty rules empty dictionary truthy", `"rules":[1]`, "rules", `{}`, true, false},
		{"nonempty dictionary requires mapping", `"rules":[1]`, "rules", `{"x":1}`, false, true},
		{"nested tenant dict", `"tenant_id":{"nested":{"flag":true},"extra":0}`, "tenant_id", `{"nested":{"flag":1}}`, true, false},
		{"missing tenant null", `"project_id":"owner"`, "tenant_id", `null`, true, false},
		{"float underflow empty dict false", `"tenant_id":1e-4000`, "tenant_id", `{}`, false, false},
		{"float underflow equality stays exact", `"tenant_id":1e-4000`, "tenant_id", `0`, false, false},
	}
	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePolicyCall
			body := `{"policies":[{"id":"match","name":"gold",` + testcase.row + `}]}`
			api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.RawQuery != "" {
					t.Fatal("Body filter leaked to server", req.URL)
				}
				r := nativePolicyWire(200, body)
				r.Header.Set("X-Request-Id", "local-filter")
				return r
			})
			values, err := api.AllRecords(context.Background(), policies.WithRecordListQuery(testcase.key, json.RawMessage(testcase.filter)))
			if len(calls) != 1 {
				t.Fatal(calls)
			}
			if testcase.failed {
				var receipt *resource.ResponseError
				if !errors.As(err, &receipt) || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "local-filter" || values != nil {
					t.Fatal(values, err, receipt)
				}
				return
			}
			if err != nil || (len(values) == 1) != testcase.matched {
				t.Fatal(values, err)
			}
		})
	}
}

func TestPythonQoSPolicyListRecordsRawCapsFirstPageAndEarlyBreak(t *testing.T) {
	for _, testcase := range []struct {
		name                string
		options             []policies.RecordListOption
		wantRows, wantCalls int
		wantLimit           string
	}{
		{"raw cap counts filtered row", []policies.RecordListOption{policies.WithRecordListMaxItems(1), policies.WithRecordListQuery("tenant_id", true)}, 0, 1, "1"},
		{"false first page retains both rows", []policies.RecordListOption{policies.WithRecordListPaginated(false)}, 2, 1, ""},
		{"float underflow pagination false", []policies.RecordListOption{policies.WithRecordListQuery("paginated", json.RawMessage(`1e-4000`))}, 2, 1, ""},
		{"explicit zero limit stays zero", []policies.RecordListOption{policies.WithRecordListLimit(0), policies.WithRecordListPaginated(false)}, 2, 1, "0"},
		{"no cap follows empty second page", nil, 2, 2, ""},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePolicyCall
			api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("limit") != testcase.wantLimit {
					t.Fatal(req.URL, testcase.wantLimit)
				}
				if req.URL.Query().Get("marker") == "next" {
					return nativePolicyWire(200, `{"policies":[]}`)
				}
				return nativePolicyWire(200, `{"policies":[{"id":"filtered","tenant_id":2},{"id":"match","tenant_id":1}],"next":"?marker=next"}`)
			})
			values, err := api.AllRecords(context.Background(), testcase.options...)
			if err != nil || len(values) != testcase.wantRows || len(calls) != testcase.wantCalls {
				t.Fatal(values, err, calls)
			}
		})
	}
	var calls []nativePolicyCall
	api := pythonPolicyFindAPI(t, &calls, func(*http.Request) *http.Response {
		return nativePolicyWire(200, `{"policies":[{"id":"one"},{"id":"two"}],"next":"?marker=next"}`)
	})
	for value, err := range api.ListRecords(context.Background()) {
		if value == nil || err != nil {
			t.Fatal(value, err)
		}
		break
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
}

func TestPythonQoSPolicyListRecordsContinuationFormatsMarkerFallbackAndScope(t *testing.T) {
	for _, testcase := range []struct {
		name, extra, header string
		limit               bool
	}{
		{"plural links", `,"policies_links":[{"rel":"next","href":"?marker=next"}]`, "", false},
		{"top next", `,"next":"/v2.0/qos/policies?marker=next"`, "", false},
		{"dictionary next", `,"links":{"next":"?marker=next"}`, "", false},
		{"RFC Link", "", `<?marker=next>; rel="next"`, false},
		{"marker fallback after short page", "", "", true},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePolicyCall
			api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("shared") != "False" || req.Header.Get("Accept") != "application/json" {
					t.Fatal(req.URL, req.Header)
				}
				if len(calls) == 2 {
					if req.URL.Query().Get("marker") != "next" {
						t.Fatal(req.URL)
					}
					return nativePolicyWire(200, `{"policies":[]}`)
				}
				r := nativePolicyWire(200, `{"policies":[{"id":"next","shared":false}]`+testcase.extra+`}`)
				if testcase.header != "" {
					r.Header.Set("Link", testcase.header)
				}
				return r
			})
			options := []policies.RecordListOption{policies.WithRecordListQuery("is_shared", false)}
			if testcase.limit {
				options = append(options, policies.WithRecordListLimit(3))
			}
			values, err := api.AllRecords(context.Background(), options...)
			if err != nil || len(values) != 1 || len(calls) != 2 {
				t.Fatal(values, err, calls)
			}
		})
	}
	for _, next := range []string{"?shared=True&marker=next", "?shared=False", "https://other.invalid/qos/policies?marker=next", "/neutron/v2.0/ports?marker=next"} {
		t.Run(next, func(t *testing.T) {
			var calls []nativePolicyCall
			raw, _ := json.Marshal(next)
			body := `{"policies":[{"id":"first"}],"next":` + string(raw) + `}`
			api := pythonPolicyFindAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(200, body) })
			values, err := api.AllRecords(context.Background(), policies.WithRecordListQuery("is_shared", false))
			var receipt *resource.ResponseError
			if values != nil || err == nil || !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != body || len(calls) != 1 {
				t.Fatal(values, err, receipt, calls)
			}
		})
	}
}

func TestPythonQoSPolicyListRecordsSnapshotsCallbacksAndPreflight(t *testing.T) {
	query := map[string]json.RawMessage{"fields": json.RawMessage(`["id","tags"]`), "x_unknown": json.RawMessage(`{malformed`)}
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.1"
	option := policies.WithRecordListOpts(policies.RecordListOpts{Query: query, Headers: headers, Microversion: &version})
	query["fields"][2] = 'x'
	headers["X-Trace"] = "changed"
	version = "changed"
	var escaped *policies.RecordListOpts
	var calls []nativePolicyCall
	api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
		if !reflect.DeepEqual(req.URL.Query(), url.Values{"fields": {"id", "tags"}}) || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.1" {
			t.Fatal(req.URL, req.Header)
		}
		escaped.Query["fields"][2] = 'z'
		escaped.Headers["X-Trace"] = "changed callback"
		return nativePolicyWire(200, `{"policies":[{"id":"one"}]}`)
	})
	values, err := api.AllRecords(context.Background(), option, func(c *policies.RecordListOpts) error { escaped = c; return nil })
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	for _, options := range [][]policies.RecordListOption{
		{nil}, {policies.WithRecordListMaxItems(-1)},
		{policies.WithRecordListOpts(policies.RecordListOpts{Query: map[string]json.RawMessage{"name": json.RawMessage(`[`)}})},
		{policies.WithRecordListQuery("base_path", "/escape")},
		{policies.WithRecordListQuery("location", nil)},
		{policies.WithRecordListMicroversion("bad\r\nversion")},
		{policies.WithRecordListHeaders(map[string]string{"Cookie": "secret"})},
	} {
		var none []nativePolicyCall
		invalid := pythonPolicyFindAPI(t, &none, func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return nil })
		values, err := invalid.AllRecords(context.Background(), options...)
		if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(none) != 0 {
			t.Fatal(values, err, none)
		}
	}
	for _, key := range []string{"base_path", "microversion", "connection", "resource_type", "list_base_path", "_synchronized", "__conflicting_attrs", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "session", "headers"} {
		var none []nativePolicyCall
		invalid := pythonPolicyFindAPI(t, &none, func(*http.Request) *http.Response { t.Fatal("control HTTP"); return nil })
		values, err := invalid.AllRecords(context.Background(), policies.WithRecordListQuery(key, nil))
		if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(none) != 0 {
			t.Fatal(key, values, err, none)
		}
	}
	var none []nativePolicyCall
	invalid := pythonPolicyFindAPI(t, &none, func(*http.Request) *http.Response { t.Fatal("source HTTP"); return nil })
	values, err = invalid.AllRecords(context.Background(), func(*policies.RecordListOpts) error { invalid.RawClient().ResourceBase += "changed/"; return nil })
	if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(none) != 0 {
		t.Fatal(values, err, none)
	}
	var nilAPI *policies.API
	if _, err := nilAPI.AllRecords(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.AllRecords(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestPythonQoSPolicyListRecordsAcceptedAndRejectedStatusEvidence(t *testing.T) {
	for status := 200; status < 600; status++ {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []nativePolicyCall
			body := `{"policies":[{"id":"one"}]}`
			api := pythonPolicyFindAPI(t, &calls, func(*http.Request) *http.Response {
				r := nativePolicyWire(status, body)
				r.Header.Set("X-Request-Id", "status-receipt")
				return r
			})
			values, err := api.AllRecords(context.Background())
			if len(calls) != 1 {
				t.Fatal(calls)
			}
			if status < 400 {
				if err != nil || len(values) != 1 || values[0].StatusCode != status || string(values[0].Envelope) != body {
					t.Fatal(values, err)
				}
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if values != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "status-receipt" || native.URL != api.RawClient().ResourceBase+"qos/policies" {
				t.Fatal(values, err, native)
			}
		})
	}
}

func TestPythonQoSPolicyListRecordsPhysicalFaultCancellationAndStickySource(t *testing.T) {
	for _, status := range []int{200, 203, 400, 403, 404, 500} {
		for _, fault := range []string{"read-close", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", status, fault), func(t *testing.T) {
				read, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("caller canceled in Close")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var calls []nativePolicyCall
				var source *gophercloud.ServiceClient
				body := `{"policies":[{"id":"one"}]}`
				api := pythonPolicyFindAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativePolicyWire(status, body)
					response.Header.Set("X-Request-Id", "physical-receipt")
					b := &pythonFindFaultBody{data: body}
					switch fault {
					case "read-close":
						b.read = read
						b.close = closeCause
					case "cancel":
						b.onClose = func() { cancel(cancelCause) }
					case "source":
						b.onClose = func() { source.ResourceBase += "changed/" }
					}
					response.Body = b
					return response
				})
				source = api.RawClient()
				values, err := api.AllRecords(ctx)
				if values != nil || err == nil || len(calls) != 1 {
					t.Fatal(values, err, calls)
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
					var receipt *resource.ResponseError
					if !errors.As(err, &receipt) || receipt.StatusCode != status || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "physical-receipt" {
						t.Fatal(err, receipt)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "physical-receipt" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

func TestPythonQoSPolicyListRecordsPartialIteratorAndAllDropOnLateFailure(t *testing.T) {
	for _, all := range []bool{false, true} {
		t.Run(fmt.Sprint(all), func(t *testing.T) {
			var calls []nativePolicyCall
			api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativePolicyWire(200, `{"policies":[{"id":"one","name":"first"}],"next":"?marker=next"}`)
				}
				response := nativePolicyWire(503, `{"error":"late"}`)
				response.Header.Set("X-Request-Id", "late-receipt")
				return response
			})
			var values []*policies.PolicyRecord
			var err error
			if all {
				values, err = api.AllRecords(context.Background())
			} else {
				for value, cause := range api.ListRecords(context.Background()) {
					if cause != nil {
						err = cause
						break
					}
					values = append(values, value)
				}
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Request-Id") != "late-receipt" || len(calls) != 2 {
				t.Fatal(values, err, native, calls)
			}
			if all && values != nil || !all && len(values) != 1 {
				t.Fatal(values, all)
			}
		})
	}
	for _, body := range []string{`{}`, `not JSON`, ``, `{"policies":null}`, `{"policies":[null]}`, `{"policies":[4]}`, `{"policies":[{"connection":null}]}`, `{"policies":[{"microversion":null}]}`, `{"policies":[{"_synchronized":null}]}`} {
		var calls []nativePolicyCall
		api := pythonPolicyFindAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(204, body) })
		values, err := api.AllRecords(context.Background())
		var receipt *resource.ResponseError
		if values != nil || !errors.As(err, &receipt) || receipt.StatusCode != 204 || string(receipt.Body) != body || len(calls) != 1 {
			t.Fatal(body, values, err, receipt, calls)
		}
	}
}

func TestPythonQoSPolicyRecordsCanonicalVersionAliasSurvivesCleanRetriesAndPhases(t *testing.T) {
	for _, list := range []bool{false, true} {
		t.Run(fmt.Sprintf("clean503/list=%t", list), func(t *testing.T) {
			var calls []nativePolicyCall
			var versions []string
			api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
				versions = append(versions, req.Header.Get("OpenStack-API-Version"))
				if len(req.Header.Values("OpenStack-API-Version")) != 1 {
					t.Fatal(req.Header)
				}
				if len(calls) == 1 {
					return nativePolicyWire(503, `{"error":"retry"}`)
				}
				body := `{"policy":{"id":"one"}}`
				if list {
					body = `{"policies":[{"id":"one"}]}`
				}
				return nativePolicyWire(200, body)
			})
			source := api.RawClient()
			source.Microversion = "2.0"
			source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
			before := map[string]string{"Openstack-Api-Version": "network 2.0"}
			retries := 0
			source.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries++
				if retries == 1 {
					return nil
				}
				return original
			}
			if list {
				values, err := api.AllRecords(context.Background())
				if err != nil || len(values) != 1 {
					t.Fatal(values, err)
				}
			} else {
				value, err := api.FindRecord(context.Background(), "one")
				if err != nil || value == nil {
					t.Fatal(value, err)
				}
			}
			if len(calls) != 2 || retries != 1 || !reflect.DeepEqual(versions, []string{"network 2.0", "network 2.0"}) || !reflect.DeepEqual(source.MoreHeaders, before) || source.Microversion != "2.0" {
				t.Fatal(calls, retries, versions, source)
			}
		})
	}
	var calls []nativePolicyCall
	var versions []string
	api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if len(calls) == 1 {
			return nativePolicyWire(403, `{"error":"fallback"}`)
		}
		return nativePolicyWire(200, `{"policies":[{"id":"match","name":"target"}]}`)
	})
	source := api.RawClient()
	source.Microversion = "2.0"
	source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
	value, err := api.FindRecord(context.Background(), "target", policies.WithRecordFindMicroversion("2.1"))
	if err != nil || value == nil || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.0"}) || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" {
		t.Fatal(value, err, versions, source)
	}
}

func TestPythonQoSPolicyListRecordsOperationHeadersAndRetryOwnership(t *testing.T) {
	for _, version := range []string{"", "2.0"} {
		for _, mutation := range []string{"version", "omit-version", "foreign-version", "authorization", "service-token", "cookie", "host", "protocol", "accept", "omit-accept", "case-alias"} {
			t.Run(version+mutation, func(t *testing.T) {
				var calls []nativePolicyCall
				api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
					r := nativePolicyWire(503, `{"error":"original retry"}`)
					r.Header.Set("X-Request-Id", "header-receipt")
					return r
				})
				api.RawClient().Microversion = version
				api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
					if options.MoreHeaders == nil {
						options.MoreHeaders = map[string]string{}
					}
					switch mutation {
					case "version":
						options.MoreHeaders["OpenStack-API-Version"] = "network 2.9"
					case "omit-version":
						options.OmitHeaders = append(options.OmitHeaders, "Openstack-Api-Version")
						if version == "" {
							options.MoreHeaders["OpenStack-API-Version"] = "network 2.9"
						}
					case "foreign-version":
						options.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.9"
					case "authorization":
						options.MoreHeaders["Authorization"] = "bad"
					case "service-token":
						options.MoreHeaders["X-Service-Token"] = "bad"
					case "cookie":
						options.MoreHeaders["Cookie"] = "bad"
					case "host":
						options.MoreHeaders["Host"] = "bad"
					case "protocol":
						options.MoreHeaders["Connection"] = "close"
					case "accept":
						options.MoreHeaders["Accept"] = "text/xml"
					case "omit-accept":
						options.OmitHeaders = append(options.OmitHeaders, "accept")
					case "case-alias":
						options.MoreHeaders["aCcEpT"] = "application/json"
					}
					return nil
				}
				values, err := api.AllRecords(context.Background())
				var native gophercloud.ErrUnexpectedResponseCode
				if values != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Request-Id") != "header-receipt" || len(calls) != 1 {
					t.Fatal(values, err, native, calls)
				}
			})
		}
	}
	for _, explicit := range []bool{false, true} {
		var calls []nativePolicyCall
		api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
			want := "application/json"
			if explicit {
				want = "text/plain"
			}
			if req.Header.Get("Accept") != want {
				t.Fatal(req.Header, want)
			}
			return nativePolicyWire(200, `{"policies":[]}`)
		})
		api.RawClient().MoreHeaders = map[string]string{"Accept": "text/xml"}
		options := []policies.RecordListOption{}
		if explicit {
			options = append(options, policies.WithRecordListHeaders(map[string]string{"Accept": "text/plain"}))
		}
		values, err := api.AllRecords(context.Background(), options...)
		if err != nil || len(values) != 0 || len(calls) != 1 || api.RawClient().MoreHeaders["Accept"] != "text/xml" {
			t.Fatal(values, err, calls)
		}
	}
}

func TestPythonQoSPolicyListRecordsLiveReauthenticationAndSourceContext(t *testing.T) {
	var calls []nativePolicyCall
	var versions, tokens []string
	api := pythonPolicyFindAPI(t, &calls, func(req *http.Request) *http.Response {
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		tokens = append(tokens, req.Header.Get("X-Auth-Token"))
		if len(calls) == 1 {
			return nativePolicyWire(401, `{"error":"expired"}`)
		}
		return nativePolicyWire(200, `{"policies":[{"id":"one","shared":1e-4000,"is_default":false}]}`)
	})
	source := api.RawClient()
	source.Microversion = "2.0"
	source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
	reauths := 0
	source.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; source.ProviderClient.SetToken("fresh"); return nil }
	values, err := api.AllRecords(context.Background(), policies.WithRecordListMicroversion("2.1"))
	if err != nil || len(values) != 1 || len(calls) != 2 || reauths != 1 || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.1"}) || !reflect.DeepEqual(tokens, []string{"test-token", "fresh"}) || source.Microversion != "2.0" || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" {
		t.Fatal(values, err, calls, reauths, versions, tokens, source)
	}
	pythonFindField(t, values[0], "is_shared", `false`)
	pythonFindWireField(t, values[0], "shared", `1e-4000`)
	cause := errors.New("cancel list before callbacks")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	ran := false
	if values, err = api.AllRecords(ctx, func(*policies.RecordListOpts) error { ran = true; return nil }); values != nil || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || ran {
		t.Fatal(values, err, ran)
	}
	for _, body := range []string{`{"policies":{"id":"singleton","shared":true}}`, `{"policies":[{"id":"ordered","is_shared":false,"shared":true,"rules":0,"tags":false}]}`} {
		var singleton []nativePolicyCall
		other := pythonPolicyFindAPI(t, &singleton, func(*http.Request) *http.Response { return nativePolicyWire(200, body) })
		values, err = other.AllRecords(context.Background())
		if err != nil || len(values) != 1 || len(singleton) != 1 {
			t.Fatal(values, err, singleton)
		}
		pythonFindField(t, values[0], "is_shared", `true`)
		if strings.Contains(body, `"rules":0`) {
			pythonFindField(t, values[0], "rules", `[0]`)
			pythonFindField(t, values[0], "tags", `[false]`)
		}
	}
}

func TestPythonQoSPolicyRecordsFalseyNumericNextUsesSourceTruthiness(t *testing.T) {
	for _, next := range []string{`1e-4000`, `-1e-4000`, `-0.0`, `0`, `null`, `false`, `""`, `5e-324`, `1e4000`, `1`, `true`} {
		for _, find := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/find=%t", next, find), func(t *testing.T) {
				var calls []nativePolicyCall
				body := `{"policies":[{"id":"match","name":"target"}],"next":` + next + `}`
				api := pythonPolicyFindAPI(t, &calls, func(*http.Request) *http.Response {
					if find && len(calls) == 1 {
						return nativePolicyWire(404, `{"error":"fallback"}`)
					}
					return nativePolicyWire(200, body)
				})
				wantError := next == `5e-324` || next == `1e4000` || next == `1` || next == `true`
				var err error
				var value *policies.PolicyRecord
				var values []*policies.PolicyRecord
				if find {
					value, err = api.FindRecord(context.Background(), "target")
				} else {
					values, err = api.AllRecords(context.Background())
				}
				expected := 1
				if find {
					expected = 2
				}
				if len(calls) != expected {
					t.Fatal(calls)
				}
				if wantError {
					var receipt *resource.ResponseError
					if !errors.As(err, &receipt) || string(receipt.Body) != body || receipt.StatusCode != 200 || value != nil || values != nil {
						t.Fatal(value, values, err, receipt)
					}
					return
				}
				if err != nil || find && value == nil || !find && len(values) != 1 {
					t.Fatal(value, values, err)
				}
				if find {
					if string(value.Envelope) != body {
						t.Fatal(value)
					}
				} else if string(values[0].Envelope) != body {
					t.Fatal(values)
				}
			})
		}
	}
}
