package rbacpolicies_test

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

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func rbacRecordField(t *testing.T, value *rbacpolicies.RBACPolicyRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("field %s: want %s, got %#v", key, want, value)
	}
}
func rbacRecordAPI(t *testing.T, calls *[]nativeRBACCall, reply func(*http.Request) *http.Response) *rbacpolicies.API {
	t.Helper()
	api, _ := nativeRBACAPI(t, calls, reply)
	return api
}

func TestPythonRBACPolicyGetRecordFieldsAliasesDefaultsAndOwnedEvidence(t *testing.T) {
	for _, fields := range [][]string{nil, {}, {"target_project_id", "project_id", "tenant_id", "unknown", "target_project_id"}} {
		t.Run(fmt.Sprint(fields), func(t *testing.T) {
			var calls []nativeRBACCall
			body := `{"rbac_policy":{"target_tenant":false,"tenant_id":"owner","action":0,"object_type":{},"x_new":9007199254740993,"self":"drop","cls":null,"tags":null,"revision_number":1.75}}`
			api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
				want := url.Values{}
				if len(fields) != 0 {
					want["fields"] = []string{"target_tenant", "project_id", "tenant_id", "unknown", "target_tenant"}
				}
				if req.URL.EscapedPath() != "/neutron/v2.0/rbac-policies/literal%2Fid%20%252F" || !reflect.DeepEqual(req.URL.Query(), want) {
					t.Fatal(req.URL, want)
				}
				r := nativeRBACWire(203, body)
				r.Header.Set("X-Request-Id", "member-receipt")
				r.Header.Set("If-Match", "header-only")
				return r
			})
			option := rbacpolicies.WithRecordGetFields(fields...)
			if len(fields) > 0 {
				fields[0] = "mutated"
			}
			value, err := api.GetRecord(context.Background(), "literal/id %2F", option)
			if err != nil || len(calls) != 1 || calls[0].body != "" || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "member-receipt" {
				t.Fatal(value, err, calls)
			}
			rbacRecordField(t, value, "id", `"literal/id %2F"`)
			rbacRecordField(t, value, "name", `null`)
			rbacRecordField(t, value, "target_project_id", `false`)
			rbacRecordField(t, value, "project_id", `"owner"`)
			rbacRecordField(t, value, "action", `0`)
			rbacRecordField(t, value, "object_type", `{}`)
			rbacRecordField(t, value, "object_id", `null`)
			rbacRecordField(t, value, "location", `null`)
			rbacRecordField(t, value, "x_new", `9007199254740993`)
			rbacRecordField(t, value, "cls", `null`)
			rbacRecordField(t, value, "tags", `null`)
			rbacRecordField(t, value, "revision_number", `1.75`)
			if value.Resource.Body["self"] != nil || value.Resource.Body["target_tenant"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["fields"] != nil || value.Wire.Body["id"] != nil {
				t.Fatal(value)
			}
			value.Resource.Body["x_new"][0] = 'X'
			value.Header.Set("X-Request-Id", "changed")
			value.Resource.Header.Set("If-Match", "changed")
			if string(value.Wire.Body["x_new"]) != `9007199254740993` || value.Wire.Header.Get("X-Request-Id") != "member-receipt" || value.Wire.Header.Get("If-Match") != "header-only" {
				t.Fatal(value)
			}
		})
	}
}

func TestPythonRBACPolicyGetRecordEnvelopeAndMissingAreTerminal(t *testing.T) {
	for _, body := range []string{"", `not JSON`, `{"rbac_policy":{}}`, `{"id":"flat","target_tenant":null}`} {
		var calls []nativeRBACCall
		api := rbacRecordAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(200, body) })
		value, err := api.GetRecord(context.Background(), "seed")
		if err != nil || value == nil || len(calls) != 1 || string(value.Envelope) != body {
			t.Fatal(value, err, calls)
		}
		want := `"seed"`
		if strings.Contains(body, "flat") {
			want = `"flat"`
		}
		rbacRecordField(t, value, "id", want)
		rbacRecordField(t, value, "target_project_id", `null`)
	}
	for _, body := range []string{`null`, `[]`, `{"rbac_policy":null}`, `{"rbac_policy":false}`} {
		var calls []nativeRBACCall
		api := rbacRecordAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeRBACWire(200, body)
			r.Header.Set("X-Request-Id", "decode-receipt")
			return r
		})
		_, err := api.GetRecord(context.Background(), "seed")
		var receipt *resource.ResponseError
		if !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "decode-receipt" || len(calls) != 1 {
			t.Fatal(err, receipt, calls)
		}
	}
	var calls []nativeRBACCall
	api := rbacRecordAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(404, `{"error":"missing"}`) })
	value, err := api.GetRecord(context.Background(), "missing")
	var native gophercloud.ErrUnexpectedResponseCode
	if value != nil || !errors.As(err, &native) || native.Actual != 404 || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
}

func TestPythonRBACPolicyFindRecordFallbackQuerySeedsAndNoNameHint(t *testing.T) {
	var calls []nativeRBACCall
	api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			if req.URL.Query().Get("x_seed") != "nested" || req.URL.Query().Get("target_project_id") != "canonical" || req.URL.Query().Get("target_tenant") != "wire" || req.Header.Get("OpenStack-API-Version") != "network 2.2" || req.URL.Query().Has("all_projects") {
				t.Fatal(req.URL, req.Header)
			}
			return nativeRBACWire(403, `{"error":"hidden"}`)
		}
		want := url.Values{"fields": {"target_project_id", "id"}, "target_tenant": {"canonical"}, "action": {"False"}, "project_id": {"owner"}}
		if !reflect.DeepEqual(req.URL.Query(), want) || req.Header.Get("OpenStack-API-Version") != "network 2.1" || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header, want)
		}
		return nativeRBACWire(200, `{"rbac_policies":[{"id":"filtered","name":"target","tenant_id":2},{"id":"match","name":"target","tenant_id":1,"target_tenant":"*","x_new":null}]}`)
	})
	api.RawClient().Microversion = "2.1"
	value, err := api.FindRecord(context.Background(), "target", rbacpolicies.WithRecordFindMicroversion("2.2"), rbacpolicies.WithRecordFindQuery("x_seed", map[string]any{"nested": nil}), rbacpolicies.WithRecordFindQuery("target_tenant", "wire"), rbacpolicies.WithRecordFindQuery("target_project_id", "canonical"), rbacpolicies.WithRecordFindQuery("fields", []string{"target_project_id", "id"}), rbacpolicies.WithRecordFindQuery("tenant_id", true), rbacpolicies.WithRecordFindQuery("action", false), rbacpolicies.WithRecordFindQuery("project_id", "owner"), rbacpolicies.WithRecordFindQuery("all_projects", true))
	if err != nil || value == nil || len(calls) != 2 {
		t.Fatal(value, err, calls)
	}
	rbacRecordField(t, value, "id", `"match"`)
	rbacRecordField(t, value, "project_id", `1`)
	rbacRecordField(t, value, "target_project_id", `"*"`)
	rbacRecordField(t, value, "x_new", `null`)
	if value.Resource.Body["x_seed"] != nil || api.RawClient().Microversion != "2.1" {
		t.Fatal(value, api.RawClient())
	}
	var direct []nativeRBACCall
	api = rbacRecordAPI(t, &direct, func(*http.Request) *http.Response { return nativeRBACWire(200, `{"rbac_policy":{}}`) })
	value, err = api.FindRecord(context.Background(), "literal", rbacpolicies.WithRecordFindQuery("target_tenant", "first"), rbacpolicies.WithRecordFindQuery("target_project_id", nil), rbacpolicies.WithRecordFindQuery("x_seed", false), rbacpolicies.WithRecordFindQuery("name", 0))
	if err != nil || len(direct) != 1 {
		t.Fatal(value, err, direct)
	}
	rbacRecordField(t, value, "target_project_id", `null`)
	rbacRecordField(t, value, "x_seed", `false`)
	rbacRecordField(t, value, "name", `0`)
}

func TestPythonRBACPolicyFindRecordCleanFallbackMissingAndCompleteUniqueness(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, strict := range []bool{false, true} {
			var calls []nativeRBACCall
			api := rbacRecordAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeRBACWire(status, "")
				}
				return nativeRBACWire(200, `{"rbac_policies":[]}`)
			})
			value, err := api.FindRecord(context.Background(), "none", rbacpolicies.WithRecordFindIgnoreMissing(!strict))
			if value != nil || len(calls) != 2 || strict != errors.Is(err, resource.ErrNotFound) {
				t.Fatal(status, strict, value, err, calls)
			}
		}
	}
	for _, second := range []string{`{"id":"b","name":"target"}`, `{"id":"target","name":"other"}`} {
		var calls []nativeRBACCall
		api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeRBACWire(404, "")
			}
			if req.URL.Query().Get("marker") == "next" {
				return nativeRBACWire(200, `{"rbac_policies":[`+second+`]}`)
			}
			return nativeRBACWire(200, `{"rbac_policies":[{"id":"a","name":"target"}],"next":"?marker=next"}`)
		})
		value, err := api.FindRecord(context.Background(), "target")
		if value != nil || !errors.Is(err, resource.ErrAmbiguous) || len(calls) != 3 {
			t.Fatal(value, err, calls)
		}
	}
}

func TestPythonRBACPolicyListRecordQueriesDefaultsAndUnknownDiscard(t *testing.T) {
	var calls []nativeRBACCall
	body := `{"rbac_policies":[{"tenant_id":false,"target_project_id":"first","target_tenant":0,"action":{},"created_at":"untyped","revision_number":1.75,"x_new":9007199254740993,"self":"wire-only"}]}`
	api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
		want := url.Values{"action": {"False"}, "fields": {"target_project_id", "x_new"}, "object_id": {"n1", "n2"}, "object_type": {"network"}, "project_id": {""}, "target_tenant": {"0"}}
		if !reflect.DeepEqual(req.URL.Query(), want) || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header, want)
		}
		r := nativeRBACWire(200, body)
		r.Header.Set("If-Match", "header-only")
		return r
	})
	api.RawClient().MoreHeaders = map[string]string{"Accept": "text/xml"}
	seq := api.ListRecords(context.Background(), rbacpolicies.WithRecordListQuery("action", false), rbacpolicies.WithRecordListQuery("fields", []string{"target_project_id", "x_new"}), rbacpolicies.WithRecordListQuery("object_id", []any{"n1", nil, "n2"}), rbacpolicies.WithRecordListQuery("object_type", "network"), rbacpolicies.WithRecordListQuery("project_id", ""), rbacpolicies.WithRecordListQuery("target_project_id", 0), rbacpolicies.WithRecordListQuery("target_tenant", true), rbacpolicies.WithRecordListQuery("unknown_function", func() {}), rbacpolicies.WithRecordListQuery("tags", []string{"ignored"}), rbacpolicies.WithRecordListQuery("sort_key", "ignored"), rbacpolicies.WithRecordListQuery("created_at", 1))
	if len(calls) != 0 {
		t.Fatal("list eager", calls)
	}
	var values []*rbacpolicies.RBACPolicyRecord
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
	rbacRecordField(t, value, "id", `null`)
	rbacRecordField(t, value, "name", `null`)
	rbacRecordField(t, value, "project_id", `false`)
	rbacRecordField(t, value, "target_project_id", `0`)
	rbacRecordField(t, value, "object_id", `null`)
	rbacRecordField(t, value, "location", `null`)
	rbacRecordField(t, value, "created_at", `"untyped"`)
	rbacRecordField(t, value, "revision_number", `1.75`)
	if value.Resource.Body["tags"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["self"] != nil || string(value.Wire.Body["self"]) != `"wire-only"` || api.RawClient().MoreHeaders["Accept"] != "text/xml" {
		t.Fatal(value, api.RawClient())
	}
}

func TestPythonRBACPolicyRecordsOrderedResponseAliasesAndNullPrecedence(t *testing.T) {
	for _, tc := range []struct{ row, want, project string }{
		{`"target_project_id":"first","target_tenant":"last","tenant_id":"owner"`, `"last"`, `"owner"`},
		{`"target_tenant":"first","target_project_id":"last","tenant_id":"owner","project_id":null`, `"last"`, `null`},
		{`"target_tenant":"first","target_project_id":null`, `null`, `null`},
	} {
		for _, list := range []bool{false, true} {
			var calls []nativeRBACCall
			body := `{"rbac_policy":{` + tc.row + `}}`
			if list {
				body = `{"rbac_policies":[{` + tc.row + `}]}`
			}
			api := rbacRecordAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(200, body) })
			var value *rbacpolicies.RBACPolicyRecord
			var err error
			if list {
				var rows []*rbacpolicies.RBACPolicyRecord
				rows, err = api.AllRecords(context.Background())
				if len(rows) > 0 {
					value = rows[0]
				}
			} else {
				value, err = api.GetRecord(context.Background(), "seed")
			}
			if err != nil || value == nil || len(calls) != 1 {
				t.Fatal(value, err, calls)
			}
			rbacRecordField(t, value, "target_project_id", tc.want)
			rbacRecordField(t, value, "project_id", tc.project)
		}
	}
}

func TestPythonRBACPolicyListRecordLocalFiltersExactEqualityAndFloatTruthiness(t *testing.T) {
	for _, tc := range []struct {
		raw, filter string
		match, fail bool
	}{
		{`true`, `1`, true, false}, {`null`, `null`, true, false}, {`9007199254740992`, `9007199254740993.0`, false, false},
		{`1e-4000`, `{}`, false, false}, {`1e-4000`, `0`, false, false}, {`"truthy"`, `{}`, true, false}, {`"truthy"`, `{"x":1}`, false, true},
		{`{"nested":{"flag":true},"extra":0}`, `{"nested":{"flag":1}}`, true, false},
	} {
		var calls []nativeRBACCall
		body := `{"rbac_policies":[{"id":"one","tenant_id":` + tc.raw + `}]}`
		api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.RawQuery != "" {
				t.Fatal(req.URL)
			}
			return nativeRBACWire(200, body)
		})
		rows, err := api.AllRecords(context.Background(), rbacpolicies.WithRecordListQuery("tenant_id", json.RawMessage(tc.filter)))
		if len(calls) != 1 {
			t.Fatal(calls)
		}
		if tc.fail {
			var receipt *resource.ResponseError
			if rows != nil || !errors.As(err, &receipt) || string(receipt.Body) != body {
				t.Fatal(rows, err, receipt)
			}
			continue
		}
		if err != nil || (len(rows) == 1) != tc.match {
			t.Fatal(tc, rows, err)
		}
	}
}

func TestPythonRBACPolicyListRecordPagingFormatsAndOriginalWireMarker(t *testing.T) {
	for _, tc := range []struct {
		extra, link string
		limit       bool
	}{
		{`,"rbac_policies_links":[{"rel":"next","href":"?marker=next"}]`, "", false},
		{`,"next":"/v2.0/rbac-policies?marker=next"`, "", false},
		{`,"links":{"next":"?marker=next"}`, "", false},
		{"", `<?marker=next>; rel="next"`, false}, {"", "", true},
	} {
		var calls []nativeRBACCall
		api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.Query().Get("action") != "access_as_shared" {
				t.Fatal(req.URL)
			}
			if len(calls) == 2 {
				want := "next"
				if tc.limit {
					want = "9007199254740993"
				}
				if req.URL.Query().Get("marker") != want {
					t.Fatal(req.URL, want)
				}
				return nativeRBACWire(200, `{"rbac_policies":[]}`)
			}
			r := nativeRBACWire(200, `{"rbac_policies":[{"id":9007199254740993}]`+tc.extra+`}`)
			if tc.link != "" {
				r.Header.Set("Link", tc.link)
			}
			return r
		})
		opts := []rbacpolicies.RecordListOption{rbacpolicies.WithRecordListQuery("action", "access_as_shared")}
		if tc.limit {
			opts = append(opts, rbacpolicies.WithRecordListLimit(3))
		}
		rows, err := api.AllRecords(context.Background(), opts...)
		if err != nil || len(rows) != 1 || len(calls) != 2 {
			t.Fatal(rows, err, calls)
		}
		rbacRecordField(t, rows[0], "id", `9007199254740993`)
	}
}

func TestPythonRBACPolicyListRecordRawCapsBreakAndConstructorCollisions(t *testing.T) {
	for _, key := range []string{"connection", "microversion", "_synchronized", "cls"} {
		for _, raw := range []string{"null", "false"} {
			var calls []nativeRBACCall
			body := `{"rbac_policies":[{"id":"good"},{"id":"bad","` + key + `":` + raw + `}],"next":"https://unconsumed.invalid"}`
			api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response { return nativeRBACWire(200, body) })
			rows, err := api.AllRecords(context.Background(), rbacpolicies.WithRecordListMaxItems(1))
			if err != nil || len(rows) != 1 || len(calls) != 1 {
				t.Fatal(rows, err, calls)
			}
			calls = nil
			rows, err = api.AllRecords(context.Background())
			var receipt *resource.ResponseError
			if rows != nil || !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != body || len(calls) != 1 {
				t.Fatal(rows, err, receipt, calls)
			}
		}
	}
	for _, single := range []bool{false, true} {
		var calls []nativeRBACCall
		api := rbacRecordAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeRBACWire(200, `{"rbac_policies":[{"id":"one","tenant_id":false},{"id":"two","tenant_id":true}],"next":"https://unconsumed.invalid"}`)
		})
		if single {
			rows, err := api.AllRecords(context.Background(), rbacpolicies.WithRecordListPaginated(false))
			if err != nil || len(rows) != 2 {
				t.Fatal(rows, err)
			}
		} else {
			for row, err := range api.ListRecords(context.Background()) {
				if row == nil || err != nil {
					t.Fatal(row, err)
				}
				break
			}
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
	var calls []nativeRBACCall
	api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("limit") != "1" {
			t.Fatal(req.URL)
		}
		return nativeRBACWire(200, `{"rbac_policies":[{"id":"filtered","tenant_id":false},{"id":"match","tenant_id":true}],"next":"https://unconsumed.invalid"}`)
	})
	rows, err := api.AllRecords(context.Background(), rbacpolicies.WithRecordListMaxItems(1), rbacpolicies.WithRecordListQuery("tenant_id", true))
	if err != nil || len(rows) != 0 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonRBACPolicyListRecordLatePageAndContinuationErrorsKeepActualReceipt(t *testing.T) {
	for _, late := range []bool{false, true} {
		var calls []nativeRBACCall
		first := `{"rbac_policies":[{"id":"one"}],"next":"?marker=next"}`
		if !late {
			first = `{"rbac_policies":[{"id":"one"}],"next":"?action=changed&marker=next"}`
		}
		api := rbacRecordAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeRBACWire(200, first)
			}
			r := nativeRBACWire(503, `{"error":"late"}`)
			r.Header.Set("X-Request-Id", "late-receipt")
			return r
		})
		var observed int
		var err error
		for row, cause := range api.ListRecords(context.Background(), rbacpolicies.WithRecordListQuery("action", "fixed")) {
			if cause != nil {
				err = cause
				break
			}
			if row != nil {
				observed++
			}
		}
		if observed != 1 || err == nil {
			t.Fatal(observed, err, calls)
		}
		if late {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Request-Id") != "late-receipt" || len(calls) != 2 {
				t.Fatal(err, native, calls)
			}
		} else {
			var receipt *resource.ResponseError
			if !errors.As(err, &receipt) || string(receipt.Body) != first || len(calls) != 1 {
				t.Fatal(err, receipt, calls)
			}
		}
	}
}

func TestPythonRBACPolicyRecordOptionsOwnershipAndPreflight(t *testing.T) {
	var calls []nativeRBACCall
	api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("x_seed") != "original" || req.Header.Get("X-Custom") != "original" {
			t.Fatal(req.URL, req.Header)
		}
		return nativeRBACWire(200, `{"rbac_policy":{}}`)
	})
	raw := json.RawMessage(`"original"`)
	headers := map[string]string{"X-Custom": "original"}
	opts := rbacpolicies.WithRecordFindOpts(rbacpolicies.RecordFindOpts{Query: map[string]json.RawMessage{"x_seed": raw}, Headers: headers})
	raw[1] = 'X'
	headers["X-Custom"] = "changed"
	row, err := api.FindRecord(context.Background(), "literal", opts)
	if err != nil || len(calls) != 1 {
		t.Fatal(row, err, calls)
	}
	rbacRecordField(t, row, "x_seed", `"original"`)
	for _, option := range []rbacpolicies.RecordFindOption{nil, rbacpolicies.WithRecordFindQuery("cls", nil), rbacpolicies.WithRecordFindQuery("id", "other"), rbacpolicies.WithRecordFindQuery("location", nil), rbacpolicies.WithRecordFindQuery("base_path", "/ports"), rbacpolicies.WithRecordFindOpts(rbacpolicies.RecordFindOpts{Query: map[string]json.RawMessage{"object_id": json.RawMessage(`{bad`)}}), rbacpolicies.WithRecordFindHeaders(map[string]string{"Authorization": "forged"})} {
		calls = nil
		value, err := api.FindRecord(context.Background(), "literal", option)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(value, err, calls)
		}
	}
	calls = nil
	_, err = api.GetRecord(context.Background(), "literal", nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	calls = nil
	_, err = api.GetRecord(context.Background(), "literal", rbacpolicies.WithRecordGetFields(string([]byte{0xff})))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller canceled")
	cancel(cause)
	_, err = api.GetRecord(ctx, "literal")
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	var missing *rbacpolicies.API
	_, err = missing.FindRecord(context.Background(), "literal")
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	rows, err := api.AllRecords(context.Background(), rbacpolicies.WithRecordListQuery("cls", false))
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(rows, err, calls)
	}
}

// The HTTP fixture/transport is the existing nativeRBACAPI. This wrapper only
// composes standard-library readers with physical Close/cancel/source faults.
type rbacRecordClosingBody struct {
	io.Reader
	close func() error
}

func (b *rbacRecordClosingBody) Close() error { return b.close() }

func TestPythonRBACPolicyRecordPhysicalCompoundFaultsAreTerminalAndKeepActualProof(t *testing.T) {
	for _, lane := range []string{"get", "find", "list"} {
		for _, status := range []int{200, 404, 429, 503} {
			t.Run(fmt.Sprintf("%s/%d", lane, status), func(t *testing.T) {
				var calls []nativeRBACCall
				read, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("caller cancel in Close")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				closes := 0
				body := `{"rbac_policy":{"id":"one"}}`
				if lane == "list" {
					body = `{"rbac_policies":[{"id":"one"}]}`
				}
				var source *gophercloud.ServiceClient
				originalBase := ""
				api := rbacRecordAPI(t, &calls, func(*http.Request) *http.Response {
					r := nativeRBACWire(status, body)
					r.Header.Set("X-Request-Id", "compound-receipt")
					r.Body = &rbacRecordClosingBody{Reader: strings.NewReader(body), close: func() error { closes++; source.ResourceBase = originalBase; cancel(cancelCause); return closeCause }}
					// The original source changes after physical bytes are read,
					// then Close restores it; source Guard must remember it.
					r.Body = &rbacRecordReadHook{ReadCloser: r.Body, terminal: read, after: func() { source.ResourceBase = originalBase + "changed/" }}
					return r
				})
				source = api.RawClient()
				originalBase = source.ResourceBase
				source.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
				var err error
				switch lane {
				case "get":
					_, err = api.GetRecord(ctx, "one")
				case "find":
					_, err = api.FindRecord(ctx, "one")
				case "list":
					_, err = api.AllRecords(ctx)
				}
				if !errors.Is(err, read) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 || closes != 1 {
					t.Fatal(err, calls, closes)
				}
				if status < 400 {
					var receipt *resource.ResponseError
					if !errors.As(err, &receipt) || receipt.StatusCode != status || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "compound-receipt" {
						t.Fatal(err, receipt)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || native.Method != "GET" || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "compound-receipt" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

type rbacRecordReadHook struct {
	io.ReadCloser
	terminal error
	after    func()
}

func (b *rbacRecordReadHook) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.after()
	if n > 0 {
		return n, b.terminal
	}
	return n, err
}

func TestPythonRBACPolicyRecordListHeaderAndNativeRetryOwnership(t *testing.T) {
	for _, change := range []string{"Accept", "OpenStack-API-Version", "Authorization", "omit"} {
		var calls []nativeRBACCall
		api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.Header.Get("Accept") != "application/json" || req.Header.Get("OpenStack-API-Version") != "network 2.1" {
				t.Fatal(req.Header)
			}
			return nativeRBACWire(503, `{"error":"retry"}`)
		})
		api.RawClient().Microversion = "2.1"
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
			if change == "omit" {
				options.OmitHeaders = append(options.OmitHeaders, "Accept")
			} else {
				options.MoreHeaders[change] = "changed"
			}
			return nil
		}
		rows, err := api.AllRecords(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || len(calls) != 1 {
			t.Fatal(change, rows, err, native, calls)
		}
	}
	var calls []nativeRBACCall
	api := rbacRecordAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			return nativeRBACWire(503, "")
		}
		if req.Header.Get("X-Retry") != "second" || req.Header.Get("X-Auth-Token") != "fresh-token" || req.Header.Get("Accept") != "custom/list" || req.Header.Get("OpenStack-API-Version") != "network 2.3" {
			t.Fatal(req.Header)
		}
		return nativeRBACWire(200, `{"rbac_policies":[{"id":"one"}]}`)
	})
	source := api.RawClient()
	source.Microversion = "2.1"
	source.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.MoreHeaders["X-Retry"] = "second"
		source.ProviderClient.SetToken("fresh-token")
		return nil
	}
	rows, err := api.AllRecords(context.Background(), rbacpolicies.WithRecordListMicroversion("2.3"), rbacpolicies.WithRecordListHeaders(map[string]string{"Accept": "custom/list"}))
	if err != nil || len(rows) != 1 || len(calls) != 2 || source.Microversion != "2.1" {
		t.Fatal(rows, err, calls, source)
	}
}

func TestPythonRBACPolicyFindRecordNativeClean404RetryStillFallsBackOnce(t *testing.T) {
	var calls []nativeRBACCall
	api := rbacRecordAPI(t, &calls, func(*http.Request) *http.Response {
		if len(calls) == 1 {
			r := nativeRBACWire(404, `{"error":"first"}`)
			r.Header.Set("X-Request-Id", "original404")
			return r
		}
		return nativeRBACWire(200, `{"rbac_policies":[{"id":"match","name":"target"}]}`)
	})
	retries := 0
	api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		retries++
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(original, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Request-Id") != "original404" {
			t.Fatal(original, native)
		}
		return original
	}
	row, err := api.FindRecord(context.Background(), "target")
	if err != nil || row == nil || len(calls) != 2 || retries != 1 || calls[1].path != "/neutron/v2.0/rbac-policies" {
		t.Fatal(row, err, calls, retries)
	}
	rbacRecordField(t, row, "id", `"match"`)
}
