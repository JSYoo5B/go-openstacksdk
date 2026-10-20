package routers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonRouterReadField(t *testing.T, value *routers.RouterReadRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s: want %s, record=%#v", key, want, value)
	}
}
func pythonRouterRead(ctx context.Context, api *routers.API, op string) (*routers.RouterReadRecord, error) {
	if op == "find" {
		return api.FindRecord(ctx, "target")
	}
	values, err := api.AllRecords(ctx)
	if len(values) > 0 {
		return values[0], err
	}
	return nil, err
}

func TestPythonRouterReadFindLiteralRawQuerySeedsAndHTTPHeader(t *testing.T) {
	var calls []nativeRouterCall
	body := `{"router":{"name":"edge","revision":0,"x_number":9007199254740993,"admin_state_up":false,"location":{"forged":true},"self":"raw"}}`
	api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.EscapedPath() != "/neutron/v2.0/routers/router%2Fid%20%252F" {
			t.Fatal(req.URL)
		}
		want := url.Values{"x_seed": {"keep"}, "is_admin_state_up": {"False"}, "evpn_vni": {"0"}, "fields": {"is_ha", "id"}, "IF_MATCH": {"extension"}, "If-MaTcH": {"seed"}}
		if !reflect.DeepEqual(req.URL.Query(), want) {
			t.Fatal(req.URL.Query(), want)
		}
		r := nativeRouterWire(203, body)
		r.Header["If-Match"] = []string{"first", "second"}
		r.Header.Set("X-Request-Id", "member")
		return r
	})
	value, err := api.FindRecord(context.Background(), "router/id %2F", routers.WithRecordFindQuery("x_seed", "keep"), routers.WithRecordFindQuery("is_admin_state_up", false), routers.WithRecordFindQuery("evpn_vni", 0), routers.WithRecordFindQuery("description", nil), routers.WithRecordFindQuery("fields", []string{"is_ha", "id"}), routers.WithRecordFindQuery("IF_MATCH", "extension"), routers.WithRecordFindQuery("If-MaTcH", "seed"))
	if err != nil || value == nil || len(calls) != 1 || calls[0].body != "" || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "member" {
		t.Fatal(value, err, calls)
	}
	for key, want := range map[string]string{"id": `"router/id %2F"`, "x_seed": `"keep"`, "is_admin_state_up": `false`, "evpn_vni": `0`, "description": `null`, "IF_MATCH": `"extension"`, "if_match": `["first, second"]`, "revision_number": `0`, "tags": `[]`, "location": `null`} {
		pythonRouterReadField(t, value, key, want)
	}
	if value.Resource.Body["self"] != nil || value.Resource.Body["admin_state_up"] != nil || value.Resource.Body["revision"] != nil || value.Resource.Body["If-MaTcH"] != nil || value.Wire.Body["x_seed"] != nil || string(value.Wire.Body["location"]) != `{"forged":true}` {
		t.Fatal(value)
	}
	value.Resource.Body["x_number"][0] = '1'
	value.Resource.Header.Set("X-Request-Id", "changed")
	value.Header.Set("X-Request-Id", "changed")
	if string(value.Wire.Body["x_number"]) != `9007199254740993` || value.Wire.Header.Get("X-Request-Id") != "member" || string(value.Envelope) != body {
		t.Fatal("aliased evidence", value)
	}
}

func TestPythonRouterReadSparseMROAliasesAndDescriptorCasts(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       map[string]string
	}{
		{"sparse", `{"router":{}}`, map[string]string{"revision_number": `null`, "is_ha": `null`, "availability_zones": `null`, "external_gateway_info": `null`, "created_at": `null`, "updated_at": `null`, "tags": `[]`, "if_match": `null`, "name": `null`, "project_id": `null`, "tenant_id": `null`}},
		{"casts and Source revision shadow", `{"router":{"revision_number":"8","revision":"7","admin_state_up":"False","distributed":0,"ha":[],"enable_ndp_proxy":1e-400,"evpn_vni":" +0042 ","availability_zone_hints":"zone-a","availability_zones":false,"routes":{"destination":"10.0.0.0/24"},"external_gateway_info":false,"created_at":{"raw":true},"updated_at":0,"tenant_id":"tenant","tags":null}}`, map[string]string{"revision_number": `7`, "is_admin_state_up": `true`, "is_distributed": `false`, "is_ha": `false`, "enable_ndp_proxy": `false`, "evpn_vni": `0`, "availability_zone_hints": `["zone-a"]`, "availability_zones": `[false]`, "routes": `[{"destination":"10.0.0.0/24"}]`, "external_gateway_info": `{}`, "created_at": `{"raw":true}`, "updated_at": `0`, "project_id": `"tenant"`, "tenant_id": `"tenant"`, "tags": `null`}},
		{"ordered canonical wins last and explicit project null", `{"router":{"revision":7,"revision_number":8,"tenant_id":"tenant","project_id":null,"admin_state_up":true,"is_admin_state_up":false,"tags":"tag"}}`, map[string]string{"revision_number": `8`, "is_admin_state_up": `false`, "project_id": `null`, "tenant_id": `"tenant"`, "tags": `["tag"]`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativeRouterCall
			api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(200, test.body) })
			value, err := api.FindRecord(context.Background(), "target")
			if err != nil || len(calls) != 1 {
				t.Fatal(value, err, calls)
			}
			for key, want := range test.want {
				pythonRouterReadField(t, value, key, want)
			}
		})
	}
}

func TestPythonRouterReadFindHeaderConstructorSpellingAndNullSeeds(t *testing.T) {
	for _, test := range []struct {
		key   string
		input any
		want  string
	}{
		{"if_match", nil, `null`}, {"if_match", "seed", `["seed"]`}, {"IF-MATCH", []string{"a", "b"}, `["a","b"]`}, {"IF_MATCH", "extension", `null`},
	} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(204, "") })
		value, err := api.FindRecord(context.Background(), "target", routers.WithRecordFindQuery(test.key, test.input), routers.WithRecordFindQuery("x_null", nil), routers.WithRecordFindQuery("is_ha", false))
		if err != nil || value == nil || value.Wire != nil || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		pythonRouterReadField(t, value, "if_match", test.want)
		pythonRouterReadField(t, value, "x_null", `null`)
		pythonRouterReadField(t, value, "is_ha", `false`)
		if test.key == "IF_MATCH" {
			pythonRouterReadField(t, value, "IF_MATCH", `"extension"`)
		}
		if len(calls) > 0 && strings.Contains(calls[0].query, "x_null") {
			t.Fatal(calls)
		}
	}
}

func TestPythonRouterReadListExactAliasesRawFieldsTagsAndLocalFilters(t *testing.T) {
	var calls []nativeRouterCall
	body := `{"routers":[{"id":"wrong","revision":8,"tenant_id":"tenant","routes":[],"external_gateway_info":{"network_id":"ext"},"created_at":"literal","enable_ndp_proxy":1},{"id":"right","revision":7,"tenant_id":"tenant","routes":[],"external_gateway_info":{"network_id":"ext","extra":false},"created_at":"literal","enable_ndp_proxy":1,"IF-MATCH":"row","x_extension":9007199254740993}]}`
	api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
		want := url.Values{"name": {"edge"}, "admin_state_up": {"False"}, "distributed": {"False"}, "ha": {"True"}, "fields": {"is_admin_state_up", "id"}, "tags": {"one,two"}, "tags-any": {"any"}, "not-tags": {"not"}, "not-tags-any": {"none"}, "sort_key": {"name"}}
		if !reflect.DeepEqual(req.URL.Query(), want) || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL.Query(), want, req.Header)
		}
		r := nativeRouterWire(200, body)
		r.Header.Set("If-Match", "HTTP-page")
		r.Header.Set("X-Request-Id", "whole-page")
		return r
	})
	seq := api.ListRecords(context.Background(), routers.WithRecordListQuery("name", "edge"), routers.WithRecordListQuery("admin_state_up", true), routers.WithRecordListQuery("is_admin_state_up", false), routers.WithRecordListQuery("is_distributed", false), routers.WithRecordListQuery("is_ha", true), routers.WithRecordListQuery("fields", []string{"is_admin_state_up", "id"}), routers.WithRecordListQuery("tags", "one,two"), routers.WithRecordListQuery("any_tags", "any"), routers.WithRecordListQuery("not_tags", "not"), routers.WithRecordListQuery("not_any_tags", "none"), routers.WithRecordListQuery("sort_key", "name"), routers.WithRecordListQuery("revision_number", 7), routers.WithRecordListQuery("tenant_id", "tenant"), routers.WithRecordListQuery("routes", []any{}), routers.WithRecordListQuery("external_gateway_info", map[string]any{"network_id": "ext"}), routers.WithRecordListQuery("created_at", "literal"), routers.WithRecordListQuery("enable_ndp_proxy", true), routers.WithRecordListQuery("revision", func() {}), routers.WithRecordListQuery("if_match", func() {}), routers.WithRecordListQuery("unknown", func() {}))
	if len(calls) != 0 {
		t.Fatal("eager list", calls)
	}
	var values []*routers.RouterReadRecord
	for value, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if len(values) != 1 || len(calls) != 1 || string(values[0].Envelope) != body || values[0].Header.Get("X-Request-Id") != "whole-page" {
		t.Fatal(values, calls)
	}
	pythonRouterReadField(t, values[0], "id", `"right"`)
	pythonRouterReadField(t, values[0], "project_id", `"tenant"`)
	pythonRouterReadField(t, values[0], "if_match", `["row"]`)
	pythonRouterReadField(t, values[0], "x_extension", `9007199254740993`)
	if values[0].Resource.Body["IF-MATCH"] != nil || string(values[0].Wire.Body["IF-MATCH"]) != `"row"` {
		t.Fatal(values[0])
	}
}

func TestPythonRouterReadFindCleanFallbackNamePresenceAndMissing(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, nameNull := range []bool{false, true} {
			var calls []nativeRouterCall
			api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeRouterWire(code, `{"error":"clean fallback"}`)
				}
				want := url.Values{"name": {"target"}}
				if nameNull {
					want = url.Values{}
				}
				if !reflect.DeepEqual(req.URL.Query(), want) {
					t.Fatal(req.URL.Query(), want)
				}
				return nativeRouterWire(200, `{"routers":[]}`)
			})
			options := []routers.RecordFindOption{}
			if nameNull {
				options = append(options, routers.WithRecordFindQuery("name", nil))
			}
			value, err := api.FindRecord(context.Background(), "target", options...)
			if value != nil || err != nil || len(calls) != 2 {
				t.Fatal(value, err, calls)
			}
			calls = nil
			options = append(options, routers.WithRecordFindIgnoreMissing(false))
			value, err = api.FindRecord(context.Background(), "target", options...)
			if value != nil || !errors.Is(err, resource.ErrNotFound) || len(calls) != 2 {
				t.Fatal(value, err, calls)
			}
		}
	}
}

func TestPythonRouterReadFindWholeSearchRejectsLateDuplicateOrListError(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
			switch len(calls) {
			case 1:
				return nativeRouterWire(404, "")
			case 2:
				return nativeRouterWire(200, `{"routers":[{"id":"first","name":"target"}],"routers_links":[{"rel":"next","href":"?marker=first"}]}`)
			default:
				if req.URL.Query().Get("marker") != "first" || req.URL.Query().Get("name") != "target" {
					t.Fatal(req.URL)
				}
				if duplicate {
					return nativeRouterWire(200, `{"routers":[{"id":"second","name":"target"}]}`)
				}
				r := nativeRouterWire(403, `{"error":"late forbidden"}`)
				r.Header.Set("X-Request-Id", "late")
				return r
			}
		})
		value, err := api.FindRecord(context.Background(), "target")
		if value != nil || err == nil || len(calls) != 3 {
			t.Fatal(value, err, calls)
		}
		if duplicate {
			if !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Request-Id") != "late" {
				t.Fatal(err, native)
			}
		}
	}
}

func TestPythonRouterReadListRawCapSinglePageAndEarlyStop(t *testing.T) {
	for _, mode := range []string{"cap", "single", "break"} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
			if mode == "cap" && req.URL.Query().Get("limit") != "1" {
				t.Fatal(req.URL)
			}
			return nativeRouterWire(200, `{"routers":[{"id":"filtered","revision":0},{"id":"kept","revision":1}],"next":"?marker=kept"}`)
		})
		options := []routers.RecordListOption{}
		if mode == "cap" {
			options = append(options, routers.WithRecordListLimit(0), routers.WithRecordListMaxItems(1), routers.WithRecordListQuery("revision_number", 1))
		}
		if mode == "single" {
			options = append(options, routers.WithRecordListPaginated(false))
		}
		count := 0
		for value, err := range api.ListRecords(context.Background(), options...) {
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

func TestPythonRouterReadListContinuationScopeAndRawMarkerOwnership(t *testing.T) {
	for _, href := range []string{"?marker=one", "/v2.0/routers?marker=one", "?marker=one&name=changed", "/v2.0/ports?marker=one", "https://foreign.invalid/v2.0/routers"} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeRouterWire(200, fmt.Sprintf(`{"routers":[{"id":"one"}],"next":%q}`, href))
			}
			if req.URL.Path != "/neutron/v2.0/routers" || req.URL.Query().Get("marker") != "one" || req.URL.Query().Get("name") != "edge" {
				t.Fatal(req.URL)
			}
			return nativeRouterWire(200, `{"routers":[]}`)
		})
		values, err := api.AllRecords(context.Background(), routers.WithRecordListQuery("name", "edge"))
		safe := href == "?marker=one" || href == "/v2.0/routers?marker=one"
		if safe {
			if err != nil || len(values) != 1 || len(calls) != 2 {
				t.Fatal(href, values, err, calls)
			}
		} else {
			var proof *resource.ResponseError
			if values != nil || !errors.As(err, &proof) || len(calls) != 1 {
				t.Fatal(href, values, err, calls)
			}
		}
	}
	var calls []nativeRouterCall
	api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRouterWire(200, `{"routers":[{"id":"physical"}]}`)
		}
		if req.URL.Query().Get("marker") != "physical" || req.URL.Query().Get("limit") != "3" {
			t.Fatal(req.URL)
		}
		return nativeRouterWire(200, `{"routers":[]}`)
	})
	count := 0
	for value, err := range api.ListRecords(context.Background(), routers.WithRecordListLimit(3)) {
		if err != nil {
			t.Fatal(err)
		}
		value.Resource.Body["id"] = json.RawMessage(`"changed"`)
		value.Wire.Body["id"] = json.RawMessage(`"changed"`)
		count++
	}
	if count != 1 || len(calls) != 2 {
		t.Fatal(count, calls)
	}
}

func TestPythonRouterReadListFalseyNextAndWholePageProof(t *testing.T) {
	for _, token := range []string{"1e-400", "-1e-400", "0e100", "1", "true"} {
		var calls []nativeRouterCall
		body := `{"routers":[{"id":"one"}],"next":` + token + `}`
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeRouterWire(200, body)
			r.Header.Set("X-Request-Id", "next")
			return r
		})
		values, err := api.AllRecords(context.Background())
		if token == "1" || token == "true" {
			var proof *resource.ResponseError
			if values != nil || !errors.As(err, &proof) || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "next" {
				t.Fatal(token, values, err, proof)
			}
		} else if err != nil || len(values) != 1 || string(values[0].Envelope) != body {
			t.Fatal(token, values, err)
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
}

func TestPythonRouterReadListConstructorDecodeAndLocalFilterFailures(t *testing.T) {
	bodies := []string{`{"routers":null}`, `{"routers":[false]}`, `{"routers":[{"evpn_vni":"²"}]}`, `{"routers":[{"revision":1e400}]}`, "{\"routers\":[{\"name\":\"" + string([]byte{255}) + "\"}]}"}
	for _, key := range []string{"cls", "connection", "microversion", "_synchronized"} {
		for _, token := range []string{"null", "false"} {
			bodies = append(bodies, `{"routers":[{"`+key+`":`+token+`}]}`)
		}
	}
	for _, body := range bodies {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeRouterWire(200, body)
			r.Header.Set("X-Request-Id", "bad-page")
			return r
		})
		values, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if values != nil || !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "bad-page" || len(calls) != 1 {
			t.Fatal(body, values, err, proof, calls)
		}
	}
	for _, test := range []struct {
		actual string
		filter any
		count  int
		fails  bool
	}{
		{`null`, map[string]any{}, 0, false}, {`{}`, map[string]any{}, 0, false}, {`{"extra":false}`, map[string]any{}, 1, false}, {`{"extra":false}`, map[string]any{"missing": nil}, 1, false},
	} {
		var calls []nativeRouterCall
		body := `{"routers":[{"id":"one","external_gateway_info":` + test.actual + `}]}`
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(200, body) })
		values, err := api.AllRecords(context.Background(), routers.WithRecordListQuery("external_gateway_info", test.filter))
		if err != nil || len(values) != test.count {
			t.Fatal(test, values, err)
		}
	}
	// An untyped Body can be a truthy scalar; nested-filter mapping.get then fails.
	var calls []nativeRouterCall
	body := `{"routers":[{"id":"one","created_at":"scalar"}]}`
	api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(200, body) })
	values, err := api.AllRecords(context.Background(), routers.WithRecordListQuery("created_at", map[string]any{"key": nil}))
	var proof *resource.ResponseError
	if values != nil || !errors.As(err, &proof) || string(proof.Body) != body {
		t.Fatal(values, err, proof)
	}
}

func TestPythonRouterReadListPartialIterationAndAllDropLateError(t *testing.T) {
	for _, all := range []bool{false, true} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeRouterWire(200, `{"routers":[{"id":"one"}],"next":"?marker=one"}`)
			}
			return nativeRouterWire(409, `{"error":"late"}`)
		})
		if all {
			values, err := api.AllRecords(context.Background())
			if values != nil || err == nil || len(calls) != 2 {
				t.Fatal(values, err, calls)
			}
		} else {
			count := 0
			var last error
			for value, err := range api.ListRecords(context.Background()) {
				if err != nil {
					last = err
					break
				}
				if value == nil {
					t.Fatal(value)
				}
				count++
			}
			if count != 1 || last == nil || len(calls) != 2 {
				t.Fatal(count, last, calls)
			}
		}
	}
}

type pythonRouterReadMarshalFailure struct{ cause error }

func (v pythonRouterReadMarshalFailure) MarshalJSON() ([]byte, error) { return nil, v.cause }

func TestPythonRouterReadOwnedOptionsMarshalCausesAndPreflight(t *testing.T) {
	raw := json.RawMessage(`"owned"`)
	query := map[string]json.RawMessage{"x_seed": raw}
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.0"
	option := routers.WithRecordFindOpts(routers.RecordFindOpts{Query: query, Headers: headers, Microversion: &version})
	raw[1] = 'x'
	headers["X-Trace"] = "changed"
	version = "changed"
	var escaped *routers.RecordFindOpts
	var calls []nativeRouterCall
	api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("x_seed") != "owned" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.0" {
			t.Fatal(req.URL, req.Header)
		}
		escaped.Query["x_seed"][1] = 'z'
		escaped.Headers["X-Trace"] = "late"
		return nativeRouterWire(200, `{"router":{}}`)
	})
	value, err := api.FindRecord(context.Background(), "target", option, func(c *routers.RecordFindOpts) error { escaped = c; return nil })
	if err != nil || value == nil {
		t.Fatal(value, err)
	}
	pythonRouterReadField(t, value, "x_seed", `"owned"`)
	before := len(calls)
	for _, identity := range []string{"", " ", ".", "..", "bad\n", string([]byte{255})} {
		_, err := api.FindRecord(context.Background(), identity)
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(identity, err, calls)
		}
	}
	for _, key := range []string{"name_or_id", "ignore_missing", "allow_unknown_params", "jmespath_filters", "cls", "self", "connection", "session", "microversion", "headers", "_synchronized", "base_path", "list_base_path", "resource_type", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs", "id"} {
		_, err := api.FindRecord(context.Background(), "target", routers.WithRecordFindQuery(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(key, err, calls)
		}
		if key != "id" {
			values, err := api.AllRecords(context.Background(), routers.WithRecordListQuery(key, nil))
			if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
				t.Fatal(key, values, err, calls)
			}
		}
	}
	cause := errors.New("custom marshaler failure")
	for _, op := range []string{"find", "list"} {
		var err error
		if op == "find" {
			_, err = api.FindRecord(context.Background(), "target", routers.WithRecordFindQuery("revision_number", pythonRouterReadMarshalFailure{cause}))
		} else {
			_, err = api.AllRecords(context.Background(), routers.WithRecordListQuery("revision_number", pythonRouterReadMarshalFailure{cause}))
		}
		var marshal *json.MarshalerError
		if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &marshal) || len(calls) != before {
			t.Fatal(op, err, calls)
		}
	}
	_, err = api.FindRecord(context.Background(), "target", nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
	_, err = api.AllRecords(context.Background(), nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
	_, err = api.FindRecord(nil, "target")
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
	var missing *routers.API
	_, err = missing.FindRecord(context.Background(), "target")
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	_, err = api.FindRecord(context.Background(), "target", func(*routers.RecordFindOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
}

func TestPythonRouterReadListUnknownsDiscardedAndNilMapComposition(t *testing.T) {
	var calls []nativeRouterCall
	api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("name") != "edge" || req.Header.Get("X-Trace") != "owned" {
			t.Fatal(req.URL, req.Header)
		}
		return nativeRouterWire(200, `{"routers":[{"id":"one"}]}`)
	})
	values, err := api.AllRecords(context.Background(), routers.WithRecordListQuery("x_unknown", pythonRouterReadMarshalFailure{errors.New("must not execute")}), routers.WithRecordListOpts(routers.RecordListOpts{Query: map[string]json.RawMessage{"unknown": json.RawMessage(`[`)}}), func(c *routers.RecordListOpts) error { c.Query = nil; c.Headers = nil; return nil }, func(c *routers.RecordListOpts) error {
		c.Query["name"] = json.RawMessage(`"edge"`)
		c.Headers["X-Trace"] = "owned"
		return nil
	})
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
}

func TestPythonRouterReadMemberAndListStatusReceiptBoundaries(t *testing.T) {
	for _, op := range []string{"find", "list"} {
		var calls []nativeRouterCall
		status := 200
		body := `{"router":{}}`
		if op == "list" {
			body = `{"routers":[]}`
		}
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeRouterWire(status, body)
			r.Header.Set("X-Request-Id", "status-proof")
			return r
		})
		for _, code := range []int{200, 203, 299, 302, 399} {
			status = code
			calls = nil
			_, err := pythonRouterRead(context.Background(), api, op)
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
			_, err := pythonRouterRead(context.Background(), api, op)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodGet || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, code, err, native, calls)
			}
		}
		status = 200
		for _, bad := range []string{`null`, `[]`, `{"router":null}`, `{"router":false}`} {
			if op == "list" {
				bad = `{"routers":null}`
			}
			body = bad
			calls = nil
			_, err := pythonRouterRead(context.Background(), api, op)
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || string(proof.Body) != bad || proof.Header.Get("X-Request-Id") != "status-proof" || len(calls) != 1 {
				t.Fatal(op, bad, err, proof, calls)
			}
		}
	}
	// Parse failures on member fetch preserve Source constructor seed/defaults.
	for _, body := range []string{"", "not-json", "{"} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(200, body) })
		value, err := api.FindRecord(context.Background(), "target", routers.WithRecordFindQuery("revision_number", 0))
		if err != nil || value == nil || value.Wire != nil || string(value.Envelope) != body || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		pythonRouterReadField(t, value, "revision_number", `0`)
		pythonRouterReadField(t, value, "tags", `[]`)
	}
}

func TestPythonRouterReadFindVersionOverrideIsMemberOnly(t *testing.T) {
	for _, selected := range []string{"", "2.0"} {
		var calls []nativeRouterCall
		var versions []string
		api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
			versions = append(versions, req.Header.Get("OpenStack-API-Version"))
			switch len(calls) {
			case 1:
				return nativeRouterWire(403, `{"error":"fallback"}`)
			case 2:
				return nativeRouterWire(200, `{"routers":[{"id":"one","name":"target"}],"next":"?marker=one"}`)
			default:
				return nativeRouterWire(200, `{"routers":[]}`)
			}
		})
		source := api.RawClient()
		source.Microversion = selected
		if selected != "" {
			source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network " + selected}
		}
		value, err := api.FindRecord(context.Background(), "target", routers.WithRecordFindMicroversion("2.1"))
		fallback := ""
		if selected != "" {
			fallback = "network " + selected
		}
		if err != nil || value == nil || len(calls) != 3 || !reflect.DeepEqual(versions, []string{"network 2.1", fallback, fallback}) || source.Microversion != selected {
			t.Fatal(value, err, calls, versions, source)
		}
	}
}

func TestPythonRouterReadPhysicalFaultsNeverInvokeRetryOrReauth(t *testing.T) {
	for _, op := range []string{"find", "list"} {
		for _, status := range []int{200, 299, 401, 403, 404, 503, 599} {
			for _, fault := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", op, status, fault), func(t *testing.T) {
					var calls []nativeRouterCall
					readCause, closeCause, cancelCause := errors.New("read fault"), errors.New("close fault"), errors.New("custom cancel")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					body := `{"router":{"id":"target"}}`
					if op == "list" {
						body = `{"routers":[{"id":"target"}]}`
					}
					physical := &pythonRouterActionObservedBody{body: []byte(body)}
					api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
						r := nativeRouterWire(status, body)
						r.Body = physical
						r.Header.Set("X-Request-Id", "physical")
						return r
					})
					originalURL := api.RawClient().ServiceURL("routers")
					if op != "list" {
						originalURL = api.RawClient().ServiceURL("routers", "target")
					}
					switch fault {
					case "read-close":
						physical.readErr, physical.closeErr = readCause, closeCause
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
					_, err := pythonRouterRead(ctx, api, op)
					if err == nil || len(calls) != 1 || physical.closed != 1 || retries != 0 || reauths != 0 {
						t.Fatal(err, calls, physical.closed, retries, reauths)
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

func TestPythonRouterReadRetryVersionAbsenceAndLiveTokenOwnership(t *testing.T) {
	for _, selected := range []string{"", "2.0"} {
		for _, mutation := range []string{"version", "foreign-version", "authorization", "body"} {
			var calls []nativeRouterCall
			api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
				r := nativeRouterWire(503, `{"error":"original"}`)
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
			_, err := api.FindRecord(context.Background(), "target")
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"original"}` || native.ResponseHeader.Get("X-Request-Id") != "owned-proof" || len(calls) != 1 || calls[0].body != "" {
				t.Fatal(selected, mutation, err, native, calls)
			}
		}
	}
	for _, mode := range []string{"retry", "reauth"} {
		var calls []nativeRouterCall
		var provider *gophercloud.ProviderClient
		api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				code := 503
				if mode == "reauth" {
					code = 401
				}
				return nativeRouterWire(code, `{"error":"first"}`)
			}
			if req.Header.Get("X-Auth-Token") != "refreshed" || req.Header.Get("OpenStack-API-Version") != "" || calls[1].method != http.MethodGet || calls[1].body != "" {
				t.Fatal(req.Header, calls)
			}
			if mode == "retry" && req.Header.Get("X-Retry") != "owned" {
				t.Fatal(req.Header)
			}
			return nativeRouterWire(200, `{"router":{"id":"target"}}`)
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
		value, err := api.FindRecord(context.Background(), "target")
		if err != nil || value == nil || len(calls) != 2 {
			t.Fatal(mode, value, err, calls)
		}
	}
}

func TestPythonRouterReadExplicitListVersionAppliesToEveryPage(t *testing.T) {
	for _, override := range []string{"2.9", ""} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
			want := ""
			if override != "" {
				want = "network " + override
			}
			if req.Header.Get("OpenStack-API-Version") != want || len(req.Header.Values("OpenStack-API-Version")) > 1 || req.Header.Get("Accept") != "application/json" {
				t.Fatal(req.Header)
			}
			if len(calls) == 1 {
				return nativeRouterWire(200, `{"routers":[{"id":"first"}],"next":"?marker=first"}`)
			}
			if req.URL.Query().Get("marker") != "first" {
				t.Fatal(req.URL)
			}
			return nativeRouterWire(200, `{"routers":[]}`)
		})
		source := api.RawClient()
		source.Microversion = "2.0"
		source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0", "Accept": "source/default"}
		values, err := api.AllRecords(context.Background(), routers.WithRecordListMicroversion(override))
		if err != nil || len(values) != 1 || len(calls) != 2 || source.Microversion != "2.0" || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" || source.MoreHeaders["Accept"] != "source/default" {
			t.Fatal(values, err, calls, source)
		}
	}
}

func TestPythonRouterReadReusableOptionsConcurrentOwnedViews(t *testing.T) {
	raw := json.RawMessage(`"owned"`)
	header := map[string]string{"X-Trace": "owned"}
	query := map[string]json.RawMessage{"name": raw}
	version := "2.0"
	option := routers.WithRecordListOpts(routers.RecordListOpts{Query: query, Headers: header, Microversion: &version})
	raw[1] = 'x'
	query["name"] = json.RawMessage(`"changed"`)
	header["X-Trace"] = "changed"
	version = "changed"
	const workers = 8
	apis := make([]*routers.API, workers)
	calls := make([][]nativeRouterCall, workers)
	for i := range workers {
		apis[i] = pythonCoreRouterAPI(t, &calls[i], func(req *http.Request) *http.Response {
			if req.URL.Query().Get("name") != "owned" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.0" {
				t.Fatal(req.URL, req.Header)
			}
			return nativeRouterWire(200, `{"routers":[{"id":"target","name":"owned","min_kpps":0}]}`)
		})
	}
	type answer struct {
		index int
		value *routers.RouterReadRecord
		err   error
	}
	results := make(chan answer, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			values, err := apis[index].AllRecords(context.Background(), option)
			var value *routers.RouterReadRecord
			if len(values) == 1 {
				value = values[0]
			}
			results <- answer{index, value, err}
		}(i)
	}
	wg.Wait()
	close(results)
	values := make([]*routers.RouterReadRecord, workers)
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
		pythonRouterReadField(t, value, "name", `"owned"`)
		if string(value.Wire.Body["name"]) != `"owned"` || value.Envelope[0] != '{' {
			t.Fatal("shared snapshot", value)
		}
	}
}

func TestPythonRouterReadSourceMutationDuringReadRemainsTerminalAfterRestore(t *testing.T) {
	for _, op := range []string{"find", "list"} {
		var calls []nativeRouterCall
		body := `{"router":{"id":"target"}}`
		if op == "list" {
			body = `{"routers":[{"id":"target"}]}`
		}
		physical := &pythonRouterActionObservedBody{body: []byte(body)}
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeRouterWire(404, body)
			r.Body = physical
			r.Header.Set("X-Request-Id", "transient")
			return r
		})
		original := api.RawClient().ResourceBase
		physical.onRead = func() { api.RawClient().ResourceBase = original + "changed/" }
		physical.onClose = func() { api.RawClient().ResourceBase = original }
		retries, reauths := 0, 0
		api.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
		_, err := pythonRouterRead(context.Background(), api, op)
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "transient" || len(calls) != 1 || physical.closed != 1 || retries != 0 || reauths != 0 || api.RawClient().ResourceBase != original {
			t.Fatal(op, err, native, calls, physical.closed, retries, reauths)
		}
	}
}

func TestPythonRouterReadRetryCannotOmitOrInjectOwnedProtocolHeaders(t *testing.T) {
	for _, op := range []string{"find", "list"} {
		for _, selected := range []string{"", "2.0"} {
			for _, mutation := range []string{"inject-condition", "omit-condition", "omit-version", "omit-accept"} {
				if selected == "" && mutation == "omit-version" {
					continue
				}
				if op == "find" && mutation == "omit-accept" {
					continue
				}
				var calls []nativeRouterCall
				api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
					r := nativeRouterWire(503, `{"error":"owned header receipt"}`)
					r.Header.Set("X-Request-Id", "guard")
					return r
				})
				api.RawClient().Microversion = selected
				if mutation == "omit-condition" {
					api.RawClient().MoreHeaders = map[string]string{"If-Match": "selected"}
				}
				api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
					if o.MoreHeaders == nil {
						o.MoreHeaders = map[string]string{}
					}
					switch mutation {
					case "inject-condition":
						o.MoreHeaders["If-Match"] = "new"
					case "omit-condition":
						o.OmitHeaders = append(o.OmitHeaders, "if-match")
					case "omit-version":
						o.OmitHeaders = append(o.OmitHeaders, "Openstack-Api-Version")
					case "omit-accept":
						o.OmitHeaders = append(o.OmitHeaders, "accept")
					}
					return nil
				}
				_, err := pythonRouterRead(context.Background(), api, op)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"owned header receipt"}` || native.ResponseHeader.Get("X-Request-Id") != "guard" || len(calls) != 1 || calls[0].body != "" {
					t.Fatal(op, selected, mutation, err, native, calls)
				}
			}
		}
	}
}

func TestPythonRouterReadIntegerUnicodeFloatAndExactLocalEquality(t *testing.T) {
	var calls []nativeRouterCall
	body := `{"routers":[{"id":"decimal","evpn_vni":"４٢","revision":true,"x_number":9007199254740993},{"id":"float","evpn_vni":42.9,"revision":1},{"id":"signed","evpn_vni":"-42","revision":"NaN"}]}`
	api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(200, body) })
	values, err := api.AllRecords(context.Background(), routers.WithRecordListQuery("evpn_vni", 42), routers.WithRecordListQuery("revision_number", true))
	if err != nil || len(values) != 2 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	pythonRouterReadField(t, values[0], "evpn_vni", `42`)
	pythonRouterReadField(t, values[0], "revision_number", `true`)
	pythonRouterReadField(t, values[1], "evpn_vni", `42`)
	// Source local equality normally rounds float filters, while Go deliberately
	// retains exact-decimal comparison. Integer descriptors still cast binary64.
	calls = nil
	body = `{"routers":[{"id":"exact","created_at":9007199254740993}]}`
	values, err = api.AllRecords(context.Background(), routers.WithRecordListQuery("created_at", json.RawMessage(`9007199254740992.0`)))
	if err != nil || len(values) != 0 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
}
