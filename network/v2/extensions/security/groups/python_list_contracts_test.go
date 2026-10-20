package groups_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listRecordWire(code int, body string) *http.Response {
	r := nativeSecGroupWire(code, body)
	r.Header.Set("X-Request-Id", "security-list-actual")
	return r
}
func listRecordProof(t *testing.T, err error, code int, body string) {
	t.Helper()
	var p *resource.ResponseError
	if !errors.As(err, &p) || p.StatusCode != code || string(p.Body) != body || p.Header.Get("X-Request-Id") != "security-list-actual" {
		t.Fatal(err, p)
	}
}
func TestPythonSecurityGroupListRecordsRawDescriptorsDefaultsAndOwnedEvidence(t *testing.T) {
	var calls []nativeSecGroupCall
	body := `{"security_groups":[{"id":null,"name":false,"created_at":9007199254740993,"updated_at":"literal","stateful":0,"tenant_id":"tenant","project_id":null,"revision_number":"١٢","shared":"false","tags":false,"security_group_rules":{"id":"nested","port_range_min":null,"x_new":0},"IF-MATCH":"row","IF_MATCH":"unknown","self":"wire-only","location":{"project":{"id":"forged"}},"x_new":{"false":false,"zero":0}}]}`
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != findBase || req.URL.RawQuery != "" || req.Method != "GET" || req.Body != nil || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header)
		}
		r := listRecordWire(203, body)
		r.Header.Set("If-Match", "page-header")
		return r
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	v := rows[0]
	for key, want := range map[string]string{"id": "null", "name": "false", "created_at": "9007199254740993", "updated_at": `"literal"`, "stateful": "0", "project_id": "null", "tenant_id": `"tenant"`, "revision_number": "12", "is_shared": "true", "tags": `[false]`, "security_group_rules": `[{"id":"nested","port_range_min":null,"x_new":0}]`, "if_match": `["row"]`, "IF_MATCH": `"unknown"`, "description": "null", "location": "null"} {
		findValue(t, v, key, want)
	}
	if v.StatusCode != 203 || string(v.Envelope) != body || v.Header.Get("If-Match") != "page-header" || v.Resource.Body["self"] != nil || v.Resource.Body["shared"] != nil || string(v.Wire.Body["shared"]) != `"false"` || string(v.Wire.Body["IF-MATCH"]) != `"row"` {
		t.Fatal(v)
	}
	v.Resource.Body["x_new"][0] = 'X'
	v.Header.Set("X-Request-Id", "consumer")
	v.Envelope[0] = 'X'
	if string(v.Wire.Body["x_new"]) != `{"false":false,"zero":0}` || v.Resource.Header.Get("X-Request-Id") != "security-list-actual" || v.Wire.Header.Get("X-Request-Id") != "security-list-actual" {
		t.Fatal(v)
	}
	calls = nil
	api, _ = findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(200, `{"security_groups":[{}]}`) })
	rows, err = api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 || len(rows[0].Resource.Body) != 14 {
		t.Fatal(rows, err)
	}
	for key, want := range map[string]string{"id": "null", "project_id": "null", "tenant_id": "null", "revision_number": "null", "tags": "[]", "security_group_rules": "null", "if_match": "null", "is_shared": "null"} {
		findValue(t, rows[0], key, want)
	}
}
func TestPythonSecurityGroupListRecordsAliasAndIntegerPresence(t *testing.T) {
	for _, item := range []struct{ row, project, shared, revision string }{{`{"tenant_id":"fallback","shared":null,"revision_number":true}`, `"fallback"`, `null`, `true`}, {`{"project_id":null,"tenant_id":"fallback","is_shared":false,"shared":"false","revision_number":"٢"}`, `null`, `true`, `2`}, {`{"tenant_id":"fallback","project_id":0,"shared":1e-400,"revision_number":900719925474099312345}`, `0`, `false`, `900719925474099312345`}} {
		var calls []nativeSecGroupCall
		api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(200, `{"security_groups":[`+item.row+`]}`) })
		rows, err := api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
		findValue(t, rows[0], "project_id", item.project)
		findValue(t, rows[0], "is_shared", item.shared)
		findValue(t, rows[0], "revision_number", item.revision)
	}
	var calls []nativeSecGroupCall
	body := `{"security_groups":[{"revision_number":"²"}]}`
	api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(203, body) })
	rows, err := api.AllRecords(context.Background())
	if rows != nil || err == nil || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	listRecordProof(t, err, 203, body)
}
func TestPythonSecurityGroupListRecordsQueryAliasesAndNullCanonicalPriority(t *testing.T) {
	var calls []nativeSecGroupCall
	var query map[string][]string
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		query = req.URL.Query()
		return listRecordWire(200, `{"security_groups":[]}`)
	})
	options := []groups.RecordListOption{groups.WithRecordListQuery("description", "desc"), groups.WithRecordListQuery("fields", []string{"id", "shared"}), groups.WithRecordListQuery("id", []string{"one", "two"}), groups.WithRecordListQuery("name", "name"), groups.WithRecordListQuery("stateful", false), groups.WithRecordListQuery("project_id", "project"), groups.WithRecordListQuery("tenant_id", "tenant"), groups.WithRecordListQuery("revision_number", 0), groups.WithRecordListQuery("sort_key", []string{"id", "name"}), groups.WithRecordListQuery("sort_dir", []string{"asc", "desc"}), groups.WithRecordListQuery("tags", "tags"), groups.WithRecordListQuery("any_tags", "canonical-any"), groups.WithRecordListQuery("tags-any", "wire-any"), groups.WithRecordListQuery("not_tags", "canonical-not"), groups.WithRecordListQuery("not-tags", "wire-not"), groups.WithRecordListQuery("not_any_tags", "canonical-not-any"), groups.WithRecordListQuery("not-tags-any", "wire-not-any"), groups.WithRecordListQuery("is_shared", nil), groups.WithRecordListQuery("shared", true), groups.WithRecordListLimit(0), groups.WithRecordListMarker("initial"), groups.WithRecordListQuery("unknown", make(chan int)), groups.WithRecordListQuery("if_match", make(chan int))}
	rows, err := api.AllRecords(context.Background(), options...)
	if err != nil || rows == nil || len(rows) != 0 || len(calls) != 1 || len(query) != 16 || query["shared"] != nil || query["unknown"] != nil || query["if_match"] != nil || !reflect.DeepEqual(query["id"], []string{"one", "two"}) || !reflect.DeepEqual(query["fields"], []string{"id", "shared"}) || !reflect.DeepEqual(query["stateful"], []string{"False"}) || !reflect.DeepEqual(query["tags-any"], []string{"canonical-any"}) || !reflect.DeepEqual(query["not-tags"], []string{"canonical-not"}) || !reflect.DeepEqual(query["not-tags-any"], []string{"canonical-not-any"}) || !reflect.DeepEqual(query["project_id"], []string{"project"}) || !reflect.DeepEqual(query["tenant_id"], []string{"tenant"}) {
		t.Fatal(rows, err, calls, query)
	}
}
func TestPythonSecurityGroupListRecordsLocalFiltersAndOrderedShortCircuit(t *testing.T) {
	for _, lane := range []string{"subset", "false-first", "mapping-error", "empty-dict", "underflow"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeSecGroupCall
			body := `{"security_groups":[{"created_at":"truthy","updated_at":1e-400,"security_group_rules":{"enabled":1,"extra":false}}]}`
			api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(200, body) })
			var options []groups.RecordListOption
			switch lane {
			case "subset":
				options = []groups.RecordListOption{groups.WithRecordListQuery("security_group_rules", []map[string]any{{"enabled": 1, "extra": false}})}
			case "false-first":
				options = []groups.RecordListOption{groups.WithRecordListQuery("updated_at", true), groups.WithRecordListQuery("created_at", map[string]any{"x": true})}
			case "mapping-error":
				options = []groups.RecordListOption{groups.WithRecordListQuery("created_at", map[string]any{"x": true}), groups.WithRecordListQuery("updated_at", true)}
			case "empty-dict":
				options = []groups.RecordListOption{groups.WithRecordListQuery("created_at", map[string]any{})}
			case "underflow":
				options = []groups.RecordListOption{groups.WithRecordListQuery("updated_at", map[string]any{})}
			}
			rows, err := api.AllRecords(context.Background(), options...)
			if lane == "mapping-error" {
				if rows != nil || err == nil {
					t.Fatal(rows, err)
				}
				listRecordProof(t, err, 200, body)
			} else {
				want := 0
				if lane == "subset" || lane == "empty-dict" {
					want = 1
				}
				if err != nil || rows == nil || len(rows) != want {
					t.Fatal(rows, err)
				}
			}
			if len(calls) != 1 || calls[0].query != "" {
				t.Fatal(calls)
			}
		})
	}
}
func TestPythonSecurityGroupListRecordsPagingRepresentationsAndFixedRoute(t *testing.T) {
	for _, lane := range []string{"plural", "next", "http", "dictionary", "versioned", "falsey", "empty", "foreign", "collection", "filter", "cycle", "late"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeSecGroupCall
			body := ""
			api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("marker") != "" {
					if lane == "late" {
						return listRecordWire(403, `{"late":"denied"}`)
					}
					if lane == "cycle" {
						body = `{"security_groups":[{"id":"again"}],"next":"?marker=next"}`
						return listRecordWire(200, body)
					}
					return listRecordWire(200, `{"security_groups":[{"id":"second"}]}`)
				}
				body = `{"security_groups":[{"id":"first"}]}`
				r := listRecordWire(200, body)
				switch lane {
				case "plural":
					body = `{"security_groups":[{"id":"first"}],"security_groups_links":[{"rel":"next","href":"?marker=next"}]}`
				case "next", "cycle", "late":
					body = `{"security_groups":[{"id":"first"}],"next":"?marker=next"}`
				case "http":
					r.Header.Set("Link", "<?marker=next>; rel=\"next\"")
					return r
				case "dictionary":
					body = `{"security_groups":[{"id":"first"}],"links":{"next":"?marker=next"}}`
				case "versioned":
					body = `{"security_groups":[{"id":"first"}],"next":"/v2.0/security-groups?marker=next"}`
				case "falsey":
					body = `{"security_groups":[{"id":"first"}],"next":1e-400}`
				case "empty":
					body = `{"security_groups":[],"next":"https://foreign.invalid"}`
				case "foreign":
					body = `{"security_groups":[{"id":"first"}],"next":"https://foreign.invalid"}`
				case "collection":
					body = `{"security_groups":[{"id":"first"}],"next":"security-group-rules?marker=next"}`
				case "filter":
					body = `{"security_groups":[{"id":"first"}],"next":"?marker=next&name=changed"}`
				}
				return listRecordWire(200, body)
			})
			rows, err := api.AllRecords(context.Background())
			if lane == "foreign" || lane == "collection" || lane == "filter" || lane == "cycle" {
				if rows != nil || err == nil {
					t.Fatal(rows, err)
				}
				listRecordProof(t, err, 200, body)
			} else if lane == "late" {
				if rows != nil || !gophercloud.ResponseCodeIs(err, 403) || len(calls) != 2 {
					t.Fatal(rows, err, calls)
				}
			} else {
				want := 2
				if lane == "falsey" {
					want = 1
				}
				if lane == "empty" {
					want = 0
				}
				if err != nil || rows == nil || len(rows) != want {
					t.Fatal(rows, err, calls)
				}
			}
		})
	}
}
func TestPythonSecurityGroupListRecordsRawCapsMarkerAndUnconsumedRows(t *testing.T) {
	var calls []nativeSecGroupCall
	api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "first" {
			return listRecordWire(200, `{"security_groups":[]}`)
		}
		return listRecordWire(200, `{"security_groups":[{"id":"first"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), groups.WithRecordListLimit(5))
	if err != nil || len(rows) != 1 || len(calls) != 2 || !strings.Contains(calls[1].query, "marker=first") {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	rows, err = api.AllRecords(context.Background(), groups.WithRecordListLimit(5), groups.WithRecordListPaginated(false))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	body := `{"security_groups":[{"id":"skip","created_at":"other"},{"id":"keep","created_at":"wanted"},{"cls":null}],"next":"https://foreign.invalid"}`
	api, _ = findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(200, body) })
	options := []groups.RecordListOption{groups.WithRecordListLimit(0), groups.WithRecordListMaxItems(2), groups.WithRecordListQuery("created_at", "wanted")}
	rows, err = api.AllRecords(context.Background(), options...)
	if err != nil || len(rows) != 1 || len(calls) != 1 || calls[0].query != "limit=2" {
		t.Fatal(rows, err, calls)
	}
	findValue(t, rows[0], "id", `"keep"`)
	calls = nil
	rows, err = api.AllRecords(context.Background(), groups.WithRecordListMaxItems(1), groups.WithRecordListQuery("created_at", "wanted"))
	if err != nil || rows == nil || len(rows) != 0 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	for _, err := range api.ListRecords(context.Background(), groups.WithRecordListQuery("created_at", "wanted")) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
}
func TestPythonSecurityGroupListRecordsEnvelopeAndConstructorErrorsKeepPage(t *testing.T) {
	for _, item := range []struct {
		body  string
		valid bool
		count int
	}{{`{"security_groups":[]}`, true, 0}, {`{"security_groups":{}}`, true, 1}, {`{"security_groups":null}`, false, 0}, {`{"security_groups":false}`, false, 0}, {`{"security_groups":[null]}`, false, 0}, {`{}`, false, 0}, {`[]`, false, 0}, {"", false, 0}, {"{bad", false, 0}, {`{"security_groups":[{"cls":null}]}`, false, 0}, {`{"security_groups":[{"connection":null}]}`, false, 0}, {`{"security_groups":[{"microversion":null}]}`, false, 0}, {`{"security_groups":[{"_synchronized":null}]}`, false, 0}, {"{\"security_groups\":[{\"name\":\"\xff\"}]}", false, 0}} {
		t.Run(fmt.Sprintf("%q", item.body), func(t *testing.T) {
			var calls []nativeSecGroupCall
			api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(203, item.body) })
			rows, err := api.AllRecords(context.Background())
			if (err == nil) != item.valid || len(rows) != item.count || len(calls) != 1 {
				t.Fatal(item, rows, err, calls)
			}
			if item.valid {
				if rows == nil {
					t.Fatal(rows)
				}
			} else {
				if rows != nil {
					t.Fatal(rows)
				}
				listRecordProof(t, err, 203, item.body)
			}
		})
	}
}
func TestPythonSecurityGroupListRecordsPreflightAndIgnoredValues(t *testing.T) {
	var calls []nativeSecGroupCall
	api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("preflight reached HTTP"); return nil })
	opts := [][]groups.RecordListOption{{nil}, {groups.WithRecordListLimit(-1)}, {groups.WithRecordListMaxItems(-1)}, {groups.WithRecordListMarker("")}, {groups.WithRecordListQuery("limit", []int{1, 2})}, {groups.WithRecordListQuery("limit", 1.5)}, {groups.WithRecordListQuery("created_at", json.RawMessage(`bad`))}, {groups.WithRecordListHeaders(map[string]string{"X-Auth-Token": "bad"})}, {groups.WithRecordListHeaders(map[string]string{"OpenStack-API-Version": "network 2.0"})}, {groups.WithRecordListQuery("jmespath_filters", "")}, {groups.WithRecordListOpts(groups.RecordListOpts{Query: map[string]json.RawMessage{"jmespath_filters": json.RawMessage(`null`)}})}}
	for _, key := range []string{"cls", "session", "connection", "_synchronized", "base_path", "list_base_path", "self", "resource_type", "location", "__conflicting_attrs", "headers", "microversion"} {
		opts = append(opts, []groups.RecordListOption{groups.WithRecordListQuery(key, nil)})
	}
	for _, option := range opts {
		rows, err := api.AllRecords(context.Background(), option...)
		if rows != nil || err == nil || len(calls) != 0 {
			t.Fatal(rows, err, calls)
		}
	}
	var nilAPI *groups.API
	_, err := nilAPI.AllRecords(context.Background())
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	_, err = api.AllRecords(nil)
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	api.RawClient().Type = "image"
	_, err = api.AllRecords(context.Background())
	if !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}
func TestPythonSecurityGroupListRecordsLazySnapshotsAndConcurrentReuse(t *testing.T) {
	var calls []nativeSecGroupCall
	var header http.Header
	api, cloud := findAPI(t, &calls, func(req *http.Request) *http.Response {
		header = req.Header.Clone()
		if req.URL.Query().Get("name") != "seed" {
			t.Fatal(req.URL)
		}
		return listRecordWire(200, `{"security_groups":[{"id":"one","x_new":{"x":false}}]}`)
	})
	raw := json.RawMessage(`"seed"`)
	version := "2.4"
	option := groups.WithRecordListOpts(groups.RecordListOpts{Query: map[string]json.RawMessage{"name": raw}, Headers: map[string]string{"X-Trace": "owned"}, Microversion: &version})
	raw[1] = 'X'
	version = "changed"
	var retained *groups.RecordListOpts
	applied := 0
	options := []groups.RecordListOption{option, func(v *groups.RecordListOpts) error { applied++; retained = v; return nil }, func(*groups.RecordListOpts) error {
		retained.Query["name"] = json.RawMessage(`false`)
		retained.Headers["X-Auth-Token"] = "bad"
		*retained.Microversion = "changed"
		return nil
	}}
	iterator := api.ListRecords(context.Background(), options...)
	options[0] = nil
	if len(calls) != 0 || applied != 0 {
		t.Fatal(calls, applied)
	}
	for _, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(calls) != 2 || applied != 2 || header.Get("X-Trace") != "owned" || header.Get("OpenStack-API-Version") != "network 2.4" {
		t.Fatal(calls, applied, header)
	}
	var attempts atomic.Int32
	cloud.Provider.HTTPClient.Transport = nativeSecGroupTransport(func(*http.Request) (*http.Response, error) {
		attempts.Add(1)
		return listRecordWire(200, `{"security_groups":[{"id":"one","x_new":{"x":false}}]}`), nil
	})
	shared := groups.WithRecordListHeaders(map[string]string{"X-Trace": "owned"})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := api.AllRecords(context.Background(), shared)
			if err != nil || len(rows) != 1 {
				t.Error(rows, err)
				return
			}
			rows[0].Resource.Body["x_new"][0] = 'X'
			if string(rows[0].Wire.Body["x_new"]) != `{"x":false}` {
				t.Error(rows)
			}
		}()
	}
	wg.Wait()
	if attempts.Load() != 8 {
		t.Fatal(attempts.Load())
	}
}
func TestPythonSecurityGroupListRecordsSelectedVersionAndListHeaders(t *testing.T) {
	for _, selected := range []string{"", "2.4", "latest"} {
		var calls []nativeSecGroupCall
		var headers http.Header
		api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
			headers = req.Header.Clone()
			return listRecordWire(200, `{"security_groups":[{}]}`)
		})
		api.RawClient().Microversion = selected
		api.RawClient().MoreHeaders = map[string]string{"Accept": "configured", "X-Captured": "stable"}
		rows, err := api.AllRecords(context.Background())
		wanted := ""
		if selected != "" {
			wanted = "network " + selected
		}
		if err != nil || len(rows) != 1 || len(calls) != 1 || headers.Get("OpenStack-API-Version") != wanted || headers.Get("Accept") != "application/json" || headers.Get("X-Captured") != "stable" || api.RawClient().Microversion != selected {
			t.Fatal(rows, err, calls, headers)
		}
		calls = nil
		rows, err = api.AllRecords(context.Background(), groups.WithRecordListMicroversion(""), groups.WithRecordListHeaders(map[string]string{"aCcEpT": "owned", "If-Match": "selected"}))
		if err != nil || len(rows) != 1 || len(calls) != 1 || headers.Get("OpenStack-API-Version") != "" || headers.Get("Accept") != "owned" || headers.Get("If-Match") != "selected" || api.RawClient().Microversion != selected {
			t.Fatal(rows, err, calls, headers)
		}
	}
}
func TestPythonSecurityGroupListRecordsNativeStatusPolicy(t *testing.T) {
	for _, code := range []int{199, 200, 203, 204, 299, 300, 399, 400, 403, 404, 409, 503} {
		var calls []nativeSecGroupCall
		body := `{"security_groups":[{}]}`
		api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(code, body) })
		rows, err := api.AllRecords(context.Background())
		if code >= 200 && code < 400 {
			if err != nil || len(rows) != 1 || rows[0].StatusCode != code {
				t.Fatal(code, rows, err)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if rows != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "security-list-actual" {
				t.Fatal(rows, err, native)
			}
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
}
func TestPythonSecurityGroupListRecordsNativeRetryAndOriginalAcceptance(t *testing.T) {
	for _, lane := range []string{"clean", "reauth", "body", "version", "omit", "auth", "codes", "custom"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeSecGroupCall
			var tokens []string
			api, cloud := findAPI(t, &calls, func(req *http.Request) *http.Response {
				tokens = append(tokens, req.Header.Get("X-Auth-Token"))
				if len(calls) == 1 {
					code := 503
					if lane == "reauth" {
						code = 401
					}
					return listRecordWire(code, `{"original":"reject"}`)
				}
				if lane == "codes" {
					return listRecordWire(400, `{"outside":"acceptance"}`)
				}
				return listRecordWire(200, `{"security_groups":[{}]}`)
			})
			api.RawClient().Microversion = "2.4"
			cause := errors.New("retry callback")
			if lane == "reauth" {
				cloud.Provider.ReauthFunc = func(context.Context) error { cloud.Provider.SetToken("renewed"); return nil }
			} else {
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					switch lane {
					case "body":
						opts.JSONBody = map[string]any{"new": true}
					case "version":
						opts.MoreHeaders["OpenStack-API-Version"] = "network changed"
					case "omit":
						opts.OmitHeaders = []string{"OpenStack-API-Version"}
					case "auth":
						opts.MoreHeaders["Authorization"] = "foreign"
					case "codes":
						opts.OkCodes = append(opts.OkCodes, 400)
					case "custom":
						return cause
					}
					return nil
				}
			}
			rows, err := api.AllRecords(context.Background())
			if lane == "clean" || lane == "reauth" {
				if err != nil || len(rows) != 1 || len(calls) != 2 || calls[0] != calls[1] {
					t.Fatal(rows, err, calls)
				}
				if lane == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
					t.Fatal(tokens)
				}
			} else {
				wanted, count := 503, 1
				if lane == "codes" {
					wanted, count = 400, 2
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if rows != nil || !errors.As(err, &native) || native.Actual != wanted || len(calls) != count || lane == "custom" && !errors.Is(err, cause) || lane != "custom" && lane != "codes" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(rows, err, native, calls)
				}
			}
		})
	}
}
func TestPythonSecurityGroupListRecordsPhysicalErrorsDoNotInvokeRetry(t *testing.T) {
	for _, code := range []int{200, 400, 403, 404, 409, 503} {
		for _, lane := range []string{"read", "close", "read+close", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", code, lane), func(t *testing.T) {
				var calls []nativeSecGroupCall
				body := `{"security_groups":[{"id":"one"}],"next":1e-400}`
				readCause, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				physical := &findFaultBody{data: []byte(body)}
				switch lane {
				case "read":
					physical.readErr = readCause
				case "close":
					physical.closeErr = closeCause
				case "read+close":
					physical.readErr = readCause
					physical.closeErr = closeCause
				}
				api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { r := listRecordWire(code, body); r.Body = physical; return r })
				original := api.RawClient().ResourceBase
				if lane == "cancel" {
					physical.onRead = func() { cancel(cancelCause) }
				}
				if lane == "source" {
					physical.onRead = func() { api.RawClient().ResourceBase += "changed/" }
					physical.onClose = func() { api.RawClient().ResourceBase = original }
				}
				retryCalls := 0
				api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retryCalls++
					return nil
				}
				rows, err := api.AllRecords(ctx)
				api.RawClient().ResourceBase = original
				if rows != nil || err == nil || len(calls) != 1 || physical.closed != 1 || retryCalls != 0 {
					t.Fatal(rows, err, calls, physical.closed, retryCalls)
				}
				if code < 400 {
					listRecordProof(t, err, code, body)
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "security-list-actual" {
						t.Fatal(err, native)
					}
				}
				if (lane == "read" || lane == "read+close") && !errors.Is(err, readCause) || (lane == "close" || lane == "read+close") && !errors.Is(err, closeCause) || lane == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) || lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestPythonSecurityGroupListRecordsPublicFixtureAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var count atomic.Int32
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, req *http.Request) {
		count.Add(1)
		if req.Method != "GET" || req.URL.Path != findBase || req.URL.RawQuery != "name=literal+name" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.URL, req.Header)
		}
		testcloud.JSON(w, 200, `{"security_groups":[{"id":"one"}]}`)
	})
	api := groups.New(nativeSecGroupClient(cloud))
	rows, err := api.AllRecords(context.Background(), groups.WithRecordListQuery("name", "literal name"))
	if err != nil || len(rows) != 1 || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callback := errors.New("cancel"), errors.New("callback")
	rows, err = api.AllRecords(ctx, func(*groups.RecordListOpts) error { cancel(cause); return callback })
	if rows != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callback) || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
	original := api.RawClient().ResourceBase
	rows, err = api.AllRecords(context.Background(), func(*groups.RecordListOpts) error { api.RawClient().ResourceBase += "changed/"; return callback })
	api.RawClient().ResourceBase = original
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callback) || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
}

func TestPythonSecurityGroupListRecordsConditionalRequestOwnership(t *testing.T) {
	for _, key := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
		for _, lane := range []string{"absent", "absent-custom", "captured", "explicit", "changed", "omitted"} {
			t.Run(key+"/"+lane, func(t *testing.T) {
				var calls []nativeSecGroupCall
				var observed []http.Header
				body := `{"original":"conditional"}`
				value := `"selected-etag"`
				if strings.HasSuffix(key, "-Since") {
					value = "Wed, 21 Oct 2015 07:28:00 GMT"
				}
				api, _ := findAPI(t, &calls, func(req *http.Request) *http.Response {
					observed = append(observed, req.Header.Clone())
					if len(calls) == 1 {
						return listRecordWire(503, body)
					}
					return listRecordWire(200, `{"security_groups":[{}]}`)
				})
				var options []groups.RecordListOption
				if lane == "captured" {
					api.RawClient().MoreHeaders = map[string]string{strings.ToLower(key): value}
				}
				if lane == "explicit" || lane == "changed" || lane == "omitted" {
					options = append(options, groups.WithRecordListHeaders(map[string]string{strings.ToLower(key): value}))
				}
				cause := errors.New("conditional callback")
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					switch lane {
					case "absent", "absent-custom":
						opts.MoreHeaders[strings.ToLower(key)] = "injected"
					case "changed":
						opts.MoreHeaders[key] = "changed"
					case "omitted":
						opts.OmitHeaders = []string{strings.ToLower(key)}
					}
					if lane == "absent-custom" {
						return cause
					}
					return nil
				}
				rows, err := api.AllRecords(context.Background(), options...)
				if lane == "captured" || lane == "explicit" {
					if err != nil || len(rows) != 1 || len(calls) != 2 || len(observed) != 2 || observed[0].Get(key) != value || observed[1].Get(key) != value {
						t.Fatal(rows, err, calls, observed)
					}
					findValue(t, rows[0], "if_match", "null")
					return
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, cause) != (lane == "absent-custom") || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "security-list-actual" || len(calls) != 1 {
					t.Fatal(rows, err, native, calls)
				}
				want := ""
				if lane == "changed" || lane == "omitted" {
					want = value
				}
				if len(observed) != 1 || observed[0].Get(key) != want {
					t.Fatal(observed, want)
				}
			})
		}
	}
}
