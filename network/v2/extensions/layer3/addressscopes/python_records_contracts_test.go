package addressscopes_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonScopeField(t *testing.T, value *addressscopes.AddressScopeRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s want %s record=%#v", key, want, value)
	}
}
func pythonScopeList(t *testing.T, api *addressscopes.API, opts ...addressscopes.RecordListOption) ([]*addressscopes.AddressScopeRecord, error) {
	t.Helper()
	var values []*addressscopes.AddressScopeRecord
	for value, err := range api.ListRecords(context.Background(), opts...) {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

func TestPythonAddressScopeGetFieldsDescriptorDefaultsAndReceipts(t *testing.T) {
	for _, testcase := range []struct {
		name, body string
		checks     map[string]string
	}{
		{"omitted", `{"address_scope":{}}`, map[string]string{"id": `"target"`, "name": "null", "project_id": "null", "tenant_id": "null", "ip_version": "null", "is_shared": "null", "location": "null"}},
		{"explicit null", `{"address_scope":{"id":null,"project_id":null,"tenant_id":"fallback","ip_version":null,"shared":null}}`, map[string]string{"id": "null", "project_id": "null", "tenant_id": `"fallback"`, "ip_version": "null", "is_shared": "null"}},
		{"coercion unknown and ordered aliases", `{"address_scope":{"tenant_id":"tenant","ip_version":"٦","shared":false,"is_shared":"False","x_big":9007199254740993,"if_match":"literal","self":"drop"}}`, map[string]string{"project_id": `"tenant"`, "tenant_id": `"tenant"`, "ip_version": "6", "is_shared": "true", "x_big": "9007199254740993", "if_match": `"literal"`}},
		{"reverse aliases and truncation", `{"address_scope":{"is_shared":true,"shared":0,"ip_version":4.9}}`, map[string]string{"is_shared": "false", "ip_version": "4"}},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
				if !reflect.DeepEqual(req.URL.Query()["fields"], []string{"shared", "project_id", "x_unknown", "shared"}) {
					t.Fatal(req.URL)
				}
				response := nativeScopeWire(203, testcase.body)
				response.Header.Set("If-Match", "actual")
				response.Header.Set("X-Request-Id", "get-proof")
				return response
			})
			value, err := api.GetRecord(context.Background(), "target", addressscopes.WithRecordReadFields("is_shared", "project_id", "x_unknown", "shared"))
			if err != nil || value == nil || len(calls) != 1 || calls[0].path != "/neutron/v2.0/address-scopes/target" || value.StatusCode != 203 || string(value.Envelope) != testcase.body || value.Header.Get("X-Request-Id") != "get-proof" {
				t.Fatal(value, err, calls)
			}
			for key, want := range testcase.checks {
				pythonScopeField(t, value, key, want)
			}
			for _, key := range []string{"tags", "revision_number"} {
				if _, exists := value.Resource.Body[key]; exists {
					t.Fatal("invented NetworkResource/TagMixin default", key)
				}
			}
			if _, exists := value.Resource.Body["self"]; exists {
				t.Fatal("Source removes self")
			}
			if _, exists := value.Resource.Body["if_match"]; exists && testcase.name != "coercion unknown and ordered aliases" {
				t.Fatal("invented Header descriptor")
			}
			if value.Wire == nil || value.Wire.Header.Get("If-Match") != "actual" {
				t.Fatal("lost actual HTTP header", value)
			}
			value.Resource.Body["id"] = json.RawMessage(`"mutated"`)
			value.Resource.Header.Set("X-Request-Id", "mutated")
			if value.Header.Get("X-Request-Id") != "get-proof" || value.Wire.Header.Get("X-Request-Id") != "get-proof" || string(value.Envelope) != testcase.body {
				t.Fatal("shared proof")
			}
		})
	}
	for _, body := range []string{"", `{`} {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(204, body) })
		seed := map[string]json.RawMessage{"name": json.RawMessage(`"retained"`), "shared": json.RawMessage("false")}
		value, err := api.GetRecord(context.Background(), "target", addressscopes.WithRecordReadSeed(seed), addressscopes.WithRecordReadFields())
		if err != nil || value.Wire != nil || string(value.Envelope) != body || calls[0].query != "" {
			t.Fatal(value, err, calls)
		}
		pythonScopeField(t, value, "name", `"retained"`)
		pythonScopeField(t, value, "is_shared", "false")
	}
}

func TestPythonAddressScopeListQueryAliasesLocalFiltersAndRawCap(t *testing.T) {
	var calls []nativeScopeCall
	api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
		query := req.URL.Query()
		if query.Get("shared") != "False" || query.Get("limit") != "2" || !reflect.DeepEqual(query["fields"], []string{"is_shared", "id"}) || query.Get("name") != "scope" {
			t.Fatal(query)
		}
		for _, key := range []string{"tenant_id", "x_unknown", "all_projects", "paginated", "max_items"} {
			if query.Has(key) {
				t.Fatal("unexpected list query", query)
			}
		}
		response := nativeScopeWire(200, `{"address_scopes":[{"id":"skip","name":"scope","tenant_id":"other","shared":true},{"id":"keep","name":"scope","tenant_id":"tenant","shared":true},{"id":"unobserved","tenant_id":"tenant"}],"next":"?marker=later"}`)
		response.Header.Set("If-Match", "page-only")
		return response
	})
	values, err := pythonScopeList(t, api, addressscopes.WithRecordListQuery("shared", true), addressscopes.WithRecordListQuery("is_shared", false), addressscopes.WithRecordListQuery("tenant_id", "tenant"), addressscopes.WithRecordListQuery("name", "scope"), addressscopes.WithRecordListQuery("fields", []string{"is_shared", "id"}), addressscopes.WithRecordListQuery("x_unknown", map[string]any{"anything": false}), addressscopes.WithRecordListQuery("all_projects", true), addressscopes.WithRecordListQuery("max_items", 2))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	pythonScopeField(t, values[0], "id", `"keep"`)
	pythonScopeField(t, values[0], "project_id", `"tenant"`)
	pythonScopeField(t, values[0], "is_shared", "true")
	if _, exists := values[0].Resource.Body["if_match"]; exists {
		t.Fatal("page Header became Body")
	}
	if values[0].Header.Get("If-Match") != "page-only" {
		t.Fatal(values[0].Header)
	}
	var singletonCalls []nativeScopeCall
	singleton, _ := nativeScopeAPI(t, &singletonCalls, func(req *http.Request) *http.Response {
		if req.URL.Query().Has("name") || req.URL.Query().Has("id") {
			t.Fatal(req.URL)
		}
		return nativeScopeWire(200, `{"address_scopes":{"id":"keep","name":"scope","is_shared":false,"shared":1,"x_unknown":7}}`)
	})
	rows, err := pythonScopeList(t, singleton, addressscopes.WithRecordListQuery("name", nil), addressscopes.WithRecordListQuery("id", "keep"), addressscopes.WithRecordListQuery("paginated", false))
	if err != nil || len(rows) != 1 || len(singletonCalls) != 1 {
		t.Fatal(rows, err, singletonCalls)
	}
	pythonScopeField(t, rows[0], "is_shared", "true")
	pythonScopeField(t, rows[0], "x_unknown", "7")
}

func TestPythonAddressScopeListPagingTruthinessAndPassiveMarker(t *testing.T) {
	for _, testcase := range []struct {
		name      string
		paginated any
		pages     int
	}{
		{"null", nil, 1}, {"zero", 0, 1}, {"empty array", []any{}, 1}, {"false text", "False", 2}, {"one", 1, 2}, {"fraction underflow", json.RawMessage("1e-400"), 1}, {"fraction overflow", json.RawMessage("1e400"), 2},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 2 {
					if req.URL.Query().Get("marker") != "123" || req.URL.Query().Get("limit") != "4" || req.URL.Query().Get("shared") != "True" {
						t.Fatal(req.URL)
					}
					return nativeScopeWire(200, `{"address_scopes":[]}`)
				}
				return nativeScopeWire(200, `{"address_scopes":[{"id":123,"name":"raw-marker"}]}`)
			})
			rows, err := pythonScopeList(t, api, addressscopes.WithRecordListQuery("paginated", testcase.paginated), addressscopes.WithRecordListQuery("limit", 4), addressscopes.WithRecordListQuery("is_shared", true))
			if err != nil || len(rows) != 1 || len(calls) != testcase.pages {
				t.Fatal(rows, err, calls)
			}
		})
	}
	var calls []nativeScopeCall
	api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 2 {
			if req.URL.Query().Get("marker") != "wire" {
				t.Fatal("mutable view marker", req.URL)
			}
			return nativeScopeWire(200, `{"address_scopes":[]}`)
		}
		return nativeScopeWire(200, `{"address_scopes":[{"id":"wire"}]}`)
	})
	for value, err := range api.ListRecords(context.Background(), addressscopes.WithRecordListQuery("limit", 1)) {
		if err != nil {
			t.Fatal(err)
		}
		value.Resource.Body["id"] = json.RawMessage(`"changed"`)
	}
	if len(calls) != 2 {
		t.Fatal(calls)
	}
	for _, next := range []string{`"/v2.0/address-scopes?marker=next"`, `[{"rel":"next","href":"?marker=next"}]`, `{"next":"?marker=next"}`} {
		var linkCalls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &linkCalls, func(req *http.Request) *http.Response {
			if len(linkCalls) == 2 {
				return nativeScopeWire(200, `{"address_scopes":[]}`)
			}
			body := `{"address_scopes":[{"id":"first"}],"next":` + next + `}`
			if strings.HasPrefix(next, "[") || strings.HasPrefix(next, "{") {
				body = `{"address_scopes":[{"id":"first"}],"address_scopes_links":` + next + `}`
			}
			return nativeScopeWire(200, body)
		})
		_, err := pythonScopeList(t, api)
		if err != nil || len(linkCalls) != 2 {
			t.Fatal(next, err, linkCalls)
		}
	}
}

func TestPythonAddressScopeFindGETSeedAndCleanFallback(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					if req.URL.Query().Get("shared") != "True" || req.URL.Query().Get("is_shared") != "False" || req.URL.Query().Get("tenant_id") != "tenant" || req.URL.Query().Get("x_unknown") != "7" || req.URL.Query().Has("all_projects") || req.Header.Get("Accept") != "text/xml" {
						t.Fatal(req.URL, req.Header)
					}
					return nativeScopeWire(code, `{"error":"missing"}`)
				}
				if req.URL.Query().Get("name") != "target" || req.URL.Query().Get("shared") != "False" || req.URL.Query().Has("tenant_id") || req.URL.Query().Has("x_unknown") || req.Header.Get("Accept") != "application/json" {
					t.Fatal(req.URL, req.Header)
				}
				return nativeScopeWire(200, `{"address_scopes":[{"id":"target","name":"other","tenant_id":"tenant"}]}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"Accept": "text/xml"}
			value, err := api.FindRecord(context.Background(), "target", addressscopes.WithRecordFindQuery("shared", true), addressscopes.WithRecordFindQuery("is_shared", false), addressscopes.WithRecordFindQuery("tenant_id", "tenant"), addressscopes.WithRecordFindQuery("x_unknown", 7), addressscopes.WithRecordFindQuery("all_projects", true), addressscopes.WithRecordFindQuery("paginated", false))
			if err != nil || value == nil || len(calls) != 2 {
				t.Fatal(value, err, calls)
			}
			pythonScopeField(t, value, "name", `"other"`)
			if _, exists := value.Resource.Body["x_unknown"]; exists {
				t.Fatal("GET seed leaked into fallback row")
			}
			if api.RawClient().MoreHeaders["Accept"] != "text/xml" {
				t.Fatal("mutated source")
			}
		})
	}
	var calls []nativeScopeCall
	api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
		return nativeScopeWire(200, `{"address_scope":{"ip_version":6}}`)
	})
	value, err := api.FindRecord(context.Background(), "target", addressscopes.WithRecordFindQuery("is_shared", 0), addressscopes.WithRecordFindQuery("tenant_id", "tenant"), addressscopes.WithRecordFindQuery("x_unknown", nil))
	if err != nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
	pythonScopeField(t, value, "id", `"target"`)
	pythonScopeField(t, value, "is_shared", "false")
	pythonScopeField(t, value, "project_id", `"tenant"`)
	pythonScopeField(t, value, "x_unknown", "null")
}

func TestPythonAddressScopeFindMissingPassiveIdentityDuplicateAndLateFailure(t *testing.T) {
	for _, ignore := range []bool{true, false} {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeScopeWire(404, "")
			}
			if req.URL.Query().Has("name") {
				t.Fatal("explicit null name lost", req.URL)
			}
			return nativeScopeWire(200, `{"address_scopes":[]}`)
		})
		value, err := api.FindRecord(context.Background(), "target", addressscopes.WithRecordFindIgnoreMissing(ignore), addressscopes.WithRecordFindQuery("name", nil))
		if value != nil || len(calls) != 2 || (ignore && err != nil) || (!ignore && !errors.Is(err, resource.ErrNotFound)) {
			t.Fatal(value, err, calls)
		}
	}
	for _, mode := range []string{"duplicate", "late malformed", "late rejected", "passive null ID"} {
		t.Run(mode, func(t *testing.T) {
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeScopeWire(404, "")
				}
				if mode == "passive null ID" {
					return nativeScopeWire(200, `{"address_scopes":[{"id":null,"name":"other"}]}`)
				}
				if len(calls) == 2 {
					return nativeScopeWire(200, `{"address_scopes":[{"id":"one","name":"target"}],"next":"?marker=next"}`)
				}
				switch mode {
				case "duplicate":
					return nativeScopeWire(200, `{"address_scopes":[{"id":"target","name":"other"}]}`)
				case "late rejected":
					return nativeScopeWire(503, `{"error":"late"}`)
				default:
					return nativeScopeWire(200, `{`)
				}
			})
			opts := []addressscopes.RecordFindOption{addressscopes.WithRecordFindQuery("max_items", 10)}
			if mode == "passive null ID" {
				opts = append(opts, addressscopes.WithRecordFindQuery("paginated", false))
			}
			value, err := api.FindRecord(context.Background(), "target", opts...)
			if value != nil {
				t.Fatal("earlier candidate escaped", value, err)
			}
			if mode == "passive null ID" {
				if err != nil || len(calls) != 2 {
					t.Fatal(err, calls)
				}
				return
			}
			if err == nil || len(calls) != 3 {
				t.Fatal(err, calls)
			}
			if mode == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			if mode == "late rejected" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"late"}` {
					t.Fatal(err, native)
				}
			}
		})
	}
	for _, code := range []int{401, 429, 500, 503} {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(code, `{"error":"terminal"}`) })
		value, err := api.FindRecord(context.Background(), "target")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != code || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
	}
}

func TestPythonAddressScopeRecordOptionSnapshotsConstructorPhaseAndPreflight(t *testing.T) {
	var calls []nativeScopeCall
	api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("fields") != "shared" || req.Header.Get("X-Custom") != "before" {
			t.Fatal(req.URL, req.Header)
		}
		return nativeScopeWire(200, `{"address_scope":{}}`)
	})
	fields := []string{"is_shared"}
	seed := map[string]json.RawMessage{"name": json.RawMessage(`"before"`)}
	headers := map[string]string{"X-Custom": "before"}
	option := addressscopes.WithRecordReadOpts(addressscopes.RecordReadOpts{Fields: fields, Seed: seed, Headers: headers})
	fields[0] = "bad"
	seed["name"] = json.RawMessage(`"bad"`)
	headers["X-Custom"] = "bad"
	var retained *addressscopes.RecordReadOpts
	value, err := api.GetRecord(context.Background(), "target", option, func(c *addressscopes.RecordReadOpts) error { retained = c; return nil }, func(*addressscopes.RecordReadOpts) error {
		retained.Fields[0] = "bad"
		retained.Seed["name"][1] = 'x'
		retained.Headers["X-Custom"] = "bad"
		return nil
	})
	if err != nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
	pythonScopeField(t, value, "name", `"before"`)
	cases := []struct {
		name string
		call func() error
	}{
		{"nil get", func() error { _, err := api.GetRecord(context.Background(), "target", nil); return err }},
		{"nil find", func() error { _, err := api.FindRecord(context.Background(), "target", nil); return err }},
		{"find ID collision", func() error {
			_, err := api.FindRecord(context.Background(), "target", addressscopes.WithRecordFindQuery("id", "other"))
			return err
		}},
		{"descriptor seed before HTTP", func() error {
			_, err := api.FindRecord(context.Background(), "target", addressscopes.WithRecordFindQuery("ip_version", "²"))
			return err
		}},
		{"reserved list control", func() error {
			_, err := pythonScopeList(t, api, addressscopes.WithRecordListQuery("base_path", "foreign"))
			return err
		}},
		{"invalid max items", func() error {
			_, err := pythonScopeList(t, api, addressscopes.WithRecordListQuery("max_items", -1))
			return err
		}},
		{"auth header", func() error {
			_, err := api.GetRecord(context.Background(), "target", addressscopes.WithRecordReadHeaders(map[string]string{"X-Auth-Token": "foreign"}))
			return err
		}},
		{"invalid fields UTF8", func() error {
			_, err := api.GetRecord(context.Background(), "target", addressscopes.WithRecordReadFields(string([]byte{255})))
			return err
		}},
	}
	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			err := testcase.call()
			if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
				t.Fatal(err, calls)
			}
		})
	}
	t.Run("captured source before option", func(t *testing.T) {
		var selectedCalls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &selectedCalls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
		endpoint := api.RawClient().Endpoint
		_, err := api.GetRecord(context.Background(), "target", func(*addressscopes.RecordReadOpts) error {
			api.RawClient().Endpoint = endpoint + "changed/"
			return nil
		})
		if !errors.Is(err, resource.ErrInvalidOption) || len(selectedCalls) != 0 {
			t.Fatal(err, selectedCalls)
		}
	})
	t.Run("already canceled", func(t *testing.T) {
		cause := errors.New("cancel cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(cause)
		_, err := api.FindRecord(ctx, "target")
		if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonAddressScopeRecordListLazyReuseAndFiltersShortCircuit(t *testing.T) {
	var calls []nativeScopeCall
	api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeScopeWire(200, `{"address_scopes":[{"id":"keep","tenant_id":"tenant","project_id":"project"}]}`)
	})
	count := 0
	seq := api.ListRecords(context.Background(), func(c *addressscopes.RecordListOpts) error {
		count++
		return addressscopes.WithRecordListQuery("paginated", false)(c)
	})
	if count != 0 || len(calls) != 0 {
		t.Fatal("eager list")
	}
	for range 2 {
		for _, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if count != 2 || len(calls) != 2 {
		t.Fatal(count, calls)
	}
	for _, firstMatches := range []bool{false, true} {
		t.Run(map[bool]string{false: "short circuit", true: "dictionary error"}[firstMatches], func(t *testing.T) {
			var filterCalls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &filterCalls, func(*http.Request) *http.Response {
				response := nativeScopeWire(200, `{"address_scopes":[{"id":"keep","tenant_id":"scalar"}]}`)
				response.Header.Set("X-Request-Id", "filter-proof")
				return response
			})
			wanted := "skip"
			if firstMatches {
				wanted = "keep"
			}
			values, err := pythonScopeList(t, api, addressscopes.WithRecordListQuery("id", wanted), addressscopes.WithRecordListQuery("tenant_id", map[string]any{"part": "v"}), addressscopes.WithRecordListQuery("paginated", false))
			if len(values) != 0 || len(filterCalls) != 1 {
				t.Fatal(values, err, filterCalls)
			}
			if !firstMatches {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var receipt *resource.ResponseError
			if !errors.As(err, &receipt) || receipt.StatusCode != 200 || receipt.Header.Get("X-Request-Id") != "filter-proof" || !strings.Contains(string(receipt.Body), `"tenant_id":"scalar"`) {
				t.Fatal(err, receipt)
			}
		})
	}
}

func TestPythonAddressScopeRecordConstructorCollisionsAndMalformedProof(t *testing.T) {
	for _, key := range []string{"connection", "microversion", "_synchronized"} {
		t.Run(key, func(t *testing.T) {
			var calls []nativeScopeCall
			body := `{"address_scopes":[{"id":"one","` + key + `":null}]}`
			api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
				response := nativeScopeWire(200, body)
				response.Header.Set("X-Request-Id", "constructor")
				return response
			})
			_, err := pythonScopeList(t, api)
			var receipt *resource.ResponseError
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &receipt) || string(receipt.Body) != body || receipt.StatusCode != 200 || receipt.Header.Get("X-Request-Id") != "constructor" || len(calls) != 1 {
				t.Fatal(err, receipt, calls)
			}
			calls = nil
			api, _ = nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(200, `{"address_scope":{"`+key+`":null}}`) })
			value, err := api.GetRecord(context.Background(), "target")
			if err != nil || value == nil || string(value.Resource.Body[key]) != "null" || len(calls) != 1 {
				t.Fatal(value, err, calls)
			}
		})
	}
	for _, body := range []string{`{`, `{"address_scopes":null}`, `{"address_scopes":[{"id":"one","ip_version":"²"}]}`} {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(203, body) })
		_, err := pythonScopeList(t, api)
		var receipt *resource.ResponseError
		if !errors.As(err, &receipt) || string(receipt.Body) != body || receipt.StatusCode != 203 || len(calls) != 1 {
			t.Fatal(body, err, receipt, calls)
		}
	}
	for _, next := range []string{`https://foreign.example/address-scopes?marker=x`, `?marker=again`} {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeScopeWire(200, `{"address_scopes":[{"id":"one"}],"next":"`+next+`"}`)
		})
		_, err := pythonScopeList(t, api)
		if err == nil || len(calls) > 2 {
			t.Fatal(next, err, calls)
		}
	}
}

type pythonScopeFaultBody struct {
	data            string
	read, close     error
	used            bool
	onRead, onClose func()
}

func (b *pythonScopeFaultBody) Read(p []byte) (int, error) {
	if b.used {
		return 0, io.EOF
	}
	b.used = true
	if b.onRead != nil {
		b.onRead()
	}
	return copy(p, b.data), b.read
}
func (b *pythonScopeFaultBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

func TestPythonAddressScopeRecordPhysicalFaultsRetainAllCausesAndNoReplay(t *testing.T) {
	for _, op := range []string{"GetRecord", "FindRecord", "ListRecords"} {
		for _, code := range []int{200, 404, 429, 500, 503} {
			t.Run(op+"/"+http.StatusText(code), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("read cause"), errors.New("close cause"), errors.New("cancel cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var calls []nativeScopeCall
				var client *gophercloud.ServiceClient
				endpoint := ""
				body := `{"error":"physical"}`
				if code == 200 {
					body = `{"address_scope":{"id":"one"}}`
					if op == "ListRecords" {
						body = `{"address_scopes":[{"id":"one"}]}`
					}
				}
				api, cloud := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativeScopeWire(code, body)
					response.Header.Set("X-Request-Id", "physical")
					response.Body = &pythonScopeFaultBody{data: body, read: readCause, close: closeCause, onRead: func() { client.Endpoint = endpoint + "changed/" }, onClose: func() { client.Endpoint = endpoint; cancel(cancelCause) }}
					return response
				})
				client = api.RawClient()
				endpoint = client.Endpoint
				cloud.Provider.MaxBackoffRetries = 2
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
				var value *addressscopes.AddressScopeRecord
				var err error
				switch op {
				case "GetRecord":
					value, err = api.GetRecord(ctx, "one")
				case "FindRecord":
					value, err = api.FindRecord(ctx, "one")
				case "ListRecords":
					for row, cause := range api.ListRecords(ctx) {
						value, err = row, cause
					}
				}
				if value != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
					t.Fatal(value, err, calls)
				}
				nativeScopeOperation(t, err, op)
				if code >= 400 {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "physical" {
						t.Fatal(err, native)
					}
				} else {
					var receipt *resource.ResponseError
					if !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "physical" {
						t.Fatal(err, receipt)
					}
				}
			})
		}
	}
}

func TestPythonAddressScopeRecordCleanRetryVersionHeaderPolicyAndLiveToken(t *testing.T) {
	for _, op := range []string{"GetRecord", "FindRecord", "ListRecords"} {
		for _, version := range []string{"", "2.1"} {
			t.Run(op+"/version="+version, func(t *testing.T) {
				var calls []nativeScopeCall
				api, cloud := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativeScopeWire(404, `{"error":"original"}`)
					response.Header.Set("X-Request-Id", "retry-original")
					return response
				})
				api.RawClient().Microversion = version
				cloud.Provider.MaxBackoffRetries = 2
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					if opts.MoreHeaders == nil {
						opts.MoreHeaders = make(map[string]string)
					}
					opts.MoreHeaders["OpenStack-API-Version"] = "network 2.9"
					return nil
				}
				var err error
				switch op {
				case "GetRecord":
					_, err = api.GetRecord(context.Background(), "target")
				case "FindRecord":
					_, err = api.FindRecord(context.Background(), "target")
				case "ListRecords":
					_, err = pythonScopeList(t, api)
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"error":"original"}` || len(calls) != 1 {
					t.Fatal(err, native, calls)
				}
			})
		}
	}
	var calls []nativeScopeCall
	var client *gophercloud.ServiceClient
	api, cloud := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Auth-Token") != "current" || req.Header.Get("OpenStack-API-Version") != "network 2.4" || req.Header.Get("Accept") != "text/plain" {
			t.Fatal(req.Header)
		}
		return nativeScopeWire(200, `{"address_scopes":[]}`)
	})
	client = api.RawClient()
	cloud.Provider.SetToken("old")
	seq := api.ListRecords(context.Background(), addressscopes.WithRecordListMicroversion("2.4"), addressscopes.WithRecordListHeaders(map[string]string{"Accept": "text/plain"}))
	cloud.Provider.SetToken("current")
	for _, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(calls) != 1 || client.Microversion != "" || len(client.MoreHeaders) != 0 {
		t.Fatal(calls, client)
	}
	t.Run("clean retry member success", func(t *testing.T) {
		var retryCalls []nativeScopeCall
		api, cloud := nativeScopeAPI(t, &retryCalls, func(*http.Request) *http.Response {
			if len(retryCalls) == 1 {
				return nativeScopeWire(503, `{"error":"retry"}`)
			}
			return nativeScopeWire(200, `{"address_scope":{"id":"target"}}`)
		})
		cloud.Provider.MaxBackoffRetries = 2
		cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
		value, err := api.FindRecord(context.Background(), "target")
		if err != nil || value == nil || len(retryCalls) != 2 || retryCalls[0].path != retryCalls[1].path {
			t.Fatal(value, err, retryCalls)
		}
	})
}

func TestPythonAddressScopeRecordFinalStatusBoundariesAndGetMissing(t *testing.T) {
	for _, code := range []int{199, 200, 204, 302, 399} {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeScopeWire(code, `{"address_scope":{"shared":1e-400}}`)
		})
		value, err := api.GetRecord(context.Background(), "target")
		if err != nil || value == nil || value.StatusCode != code || len(calls) != 1 {
			t.Fatal(code, value, err, calls)
		}
		pythonScopeField(t, value, "is_shared", "false")
	}
	for _, op := range []string{"GetRecord", "ListRecords"} {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(404, `{"error":"missing"}`) })
		var err error
		if op == "GetRecord" {
			_, err = api.GetRecord(context.Background(), "target")
		} else {
			_, err = pythonScopeList(t, api)
		}
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 404 || len(calls) != 1 {
			t.Fatal(op, err, calls)
		}
	}
	for _, body := range []string{`{"address_scope":{"shared":1e400}}`, `{"address_scope":{"shared":9007199254740993}}`} {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(200, body) })
		value, err := api.GetRecord(context.Background(), "target")
		if err != nil || value == nil {
			t.Fatal(value, err)
		}
		pythonScopeField(t, value, "is_shared", "true")
		if value.Wire == nil || string(value.Envelope) != body {
			t.Fatal(value)
		}
	}
}

func TestPythonAddressScopeRecordBoundarySourceAndCallbackFailures(t *testing.T) {
	for _, op := range []string{"GetRecord", "FindRecord", "ListRecords"} {
		for _, mode := range []string{"nil context", "wrong service", "bad version", "callback error"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				var calls []nativeScopeCall
				api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("preflight sent HTTP"); return nil })
				ctx := context.Background()
				if mode == "nil context" {
					ctx = nil
				}
				if mode == "wrong service" {
					api.RawClient().Type = "compute"
				}
				callbackCause := errors.New("option failure")
				var err error
				switch op {
				case "GetRecord":
					var opts []addressscopes.RecordReadOption
					if mode == "bad version" {
						opts = append(opts, addressscopes.WithRecordReadMicroversion("bad\nversion"))
					}
					if mode == "callback error" {
						opts = append(opts, func(*addressscopes.RecordReadOpts) error { return callbackCause })
					}
					_, err = api.GetRecord(ctx, "target", opts...)
				case "FindRecord":
					var opts []addressscopes.RecordFindOption
					if mode == "bad version" {
						opts = append(opts, addressscopes.WithRecordFindMicroversion("bad\nversion"))
					}
					if mode == "callback error" {
						opts = append(opts, func(*addressscopes.RecordFindOpts) error { return callbackCause })
					}
					_, err = api.FindRecord(ctx, "target", opts...)
				case "ListRecords":
					var opts []addressscopes.RecordListOption
					if mode == "bad version" {
						opts = append(opts, addressscopes.WithRecordListMicroversion("bad\nversion"))
					}
					if mode == "callback error" {
						opts = append(opts, func(*addressscopes.RecordListOpts) error { return callbackCause })
					}
					for _, cause := range api.ListRecords(ctx, opts...) {
						err = cause
					}
				}
				if err == nil || len(calls) != 0 {
					t.Fatal(err, calls)
				}
				want := resource.ErrInvalidOption
				if mode == "wrong service" {
					want = resource.ErrUnsupported
				}
				if mode == "callback error" {
					want = callbackCause
				}
				if !errors.Is(err, want) {
					t.Fatal(err)
				}
				nativeScopeOperation(t, err, op)
			})
		}
	}
	var calls []nativeScopeCall
	api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.EscapedPath() != "/neutron/v2.0/address-scopes/literal%2Fid%20%252F" {
			t.Fatal(req.URL)
		}
		return nativeScopeWire(200, `{"address_scope":{"id":"http://foreign.example/passive"}}`)
	})
	value, err := api.GetRecord(context.Background(), "literal/id %2F")
	if err != nil || value == nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
	pythonScopeField(t, value, "id", `"http://foreign.example/passive"`)
}

func TestPythonAddressScopeFindGETOnlyVersionAndStandaloneListOverride(t *testing.T) {
	for _, testcase := range []struct {
		name, selected, override string
		explicit                 bool
	}{
		{"omitted with absent selected", "", "", false},
		{"omitted with selected default", "2.0", "", false},
		{"override with absent selected", "", "2.4", true},
		{"override with matching selected header", "2.0", "2.4", true},
		{"explicit empty GET with selected default", "2.0", "", true},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeScopeCall
			var versions []string
			api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
				version := testcase.selected
				if len(calls) == 1 && testcase.explicit {
					version = testcase.override
				}
				want := ""
				if version != "" {
					want = "network " + version
				}
				versions = append(versions, req.Header.Get("OpenStack-API-Version"))
				if req.Header.Get("OpenStack-API-Version") != want || want == "" && len(req.Header.Values("OpenStack-API-Version")) != 0 || req.Header.Get("X-Custom") != "operation" {
					t.Fatal("version/header phase", req.Header, want)
				}
				if len(calls) == 1 {
					if req.Header.Get("Accept") != "text/xml" {
						t.Fatal(req.Header)
					}
					return nativeScopeWire(404, `{"error":"fallback"}`)
				}
				if req.Header.Get("Accept") != "application/json" || req.URL.Query().Get("name") != "target" {
					t.Fatal(req.Header, req.URL)
				}
				if len(calls) == 2 {
					return nativeScopeWire(200, `{"address_scopes":[{"id":"one","name":"target"}],"next":"?marker=next"}`)
				}
				return nativeScopeWire(200, `{"address_scopes":[]}`)
			})
			client := api.RawClient()
			client.Microversion = testcase.selected
			client.MoreHeaders = map[string]string{"Accept": "text/xml", "X-Custom": "selected"}
			if testcase.selected != "" {
				client.MoreHeaders["OpenStack-API-Version"] = "network " + testcase.selected
			}
			before := make(map[string]string)
			for key, value := range client.MoreHeaders {
				before[key] = value
			}
			callbacks := 0
			opts := []addressscopes.RecordFindOption{
				addressscopes.WithRecordFindHeaders(map[string]string{"X-Custom": "operation"}),
				func(*addressscopes.RecordFindOpts) error { callbacks++; return nil },
			}
			if testcase.explicit {
				opts = append(opts, addressscopes.WithRecordFindMicroversion(testcase.override))
			}
			value, err := api.FindRecord(context.Background(), "target", opts...)
			if err != nil || value == nil || len(calls) != 3 || callbacks != 1 || client.Microversion != testcase.selected || !reflect.DeepEqual(client.MoreHeaders, before) {
				t.Fatal(value, err, calls, versions, callbacks, client)
			}
			pythonScopeField(t, value, "id", `"one"`)
		})
	}
	for _, op := range []string{"GetRecord", "ListRecords"} {
		t.Run("explicit standalone override/"+op, func(t *testing.T) {
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.Header.Get("OpenStack-API-Version") != "network 2.4" {
					t.Fatal(req.Header)
				}
				if op == "GetRecord" {
					return nativeScopeWire(200, `{"address_scope":{}}`)
				}
				if len(calls) == 1 {
					return nativeScopeWire(200, `{"address_scopes":[{"id":"one"}],"next":"?marker=next"}`)
				}
				return nativeScopeWire(200, `{"address_scopes":[]}`)
			})
			client := api.RawClient()
			client.Microversion = "2.0"
			client.MoreHeaders = map[string]string{"OpenStack-API-Version": "network 2.0"}
			var err error
			wantCalls := 1
			if op == "GetRecord" {
				_, err = api.GetRecord(context.Background(), "target", addressscopes.WithRecordReadMicroversion("2.4"))
			} else {
				_, err = pythonScopeList(t, api, addressscopes.WithRecordListMicroversion("2.4"))
				wantCalls = 2
			}
			if err != nil || len(calls) != wantCalls || client.Microversion != "2.0" || client.MoreHeaders["OpenStack-API-Version"] != "network 2.0" {
				t.Fatal(err, calls, client)
			}
		})
	}
}
