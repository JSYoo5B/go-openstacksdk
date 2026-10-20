package trunks_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonTrunkList(t *testing.T, api *trunks.API, options ...trunks.RecordListOption) ([]*trunks.TrunkListRecord, error) {
	t.Helper()
	var values []*trunks.TrunkListRecord
	for value, err := range api.ListRecords(context.Background(), options...) {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

func TestPythonTrunkListRecordDescriptorsSubportsTagsAndIndependentProof(t *testing.T) {
	for _, item := range []struct{ name, row, admin, subports, tags, project string }{
		{"default", `{}`, "null", "null", "[]", "null"},
		{"coercion", `{"id":42,"name":null,"tenant_id":"tenant","is_admin_state_up":false,"admin_state_up":"False","sub_ports":false,"tags":null,"revision_number":"untyped","created_at":"not-a-time","unknown":9007199254740993,"if_match":"body","self":"reserved"}`, "true", "[false]", "null", `"tenant"`},
		{"explicit null alias", `{"project_id":null,"tenant_id":"fallback","admin_state_up":null,"sub_ports":null,"tags":"single"}`, "null", "null", `["single"]`, "null"},
		{"binary64 and untyped subports", `{"admin_state_up":1e-400,"sub_ports":[null,false,{"port_id":7,"segmentation_type":null,"segmentation_id":"١٠٠","unknown":false}],"tags":false}`, "false", `[null,false,{"port_id":7,"segmentation_type":null,"segmentation_id":"١٠٠","unknown":false}]`, "[false]", "null"},
	} {
		t.Run(item.name, func(t *testing.T) {
			var calls []nativeTrunkCall
			body := `{"trunks":[` + item.row + `]}`
			api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.Method != "GET" || req.URL.Path != "/neutron/v2.0/trunks" || req.URL.RawQuery != "" || req.Header.Get("Accept") != "application/json" {
					t.Fatal(req)
				}
				r := nativeTrunkWire(203, body)
				r.Header.Set("If-Match", "actual")
				return r
			})
			rows, err := api.AllRecords(context.Background())
			if err != nil || len(rows) != 1 || len(calls) != 1 {
				t.Fatal(rows, err, calls)
			}
			v := rows[0]
			for key, want := range map[string]string{"is_admin_state_up": item.admin, "sub_ports": item.subports, "tags": item.tags, "project_id": item.project, "location": "null"} {
				pythonFindField(t, v, key, want)
			}
			if v.Wire == nil || v.StatusCode != 203 || v.Resource.StatusCode != 203 || v.Wire.StatusCode != 203 || string(v.Envelope) != body || v.Header.Get("If-Match") != "actual" || v.Resource.Header.Get("If-Match") != "actual" || v.Wire.Header.Get("If-Match") != "actual" || v.Resource.Body["self"] != nil {
				t.Fatal(v)
			}
			if item.name == "coercion" {
				pythonFindField(t, v, "revision_number", `"untyped"`)
				pythonFindField(t, v, "created_at", `"not-a-time"`)
				pythonFindField(t, v, "if_match", `"body"`)
				pythonFindWireField(t, v, "admin_state_up", `"False"`)
				v.Resource.Body["unknown"][0] = 'X'
				v.Header.Set("If-Match", "mutated")
				v.Envelope[0] = 'X'
				if string(v.Wire.Body["unknown"]) != "9007199254740993" || v.Wire.Header.Get("If-Match") != "actual" || v.Resource.Header.Get("If-Match") != "actual" {
					t.Fatal("receipt alias", v)
				}
			}
		})
	}
}

func TestPythonTrunkListRecordDeclaredQueryAliasesAndUnknownDiscard(t *testing.T) {
	for _, item := range []struct {
		key, wire string
		value     any
		want      []string
	}{
		{"limit", "limit", 0, []string{"0"}}, {"marker", "marker", "cursor", []string{"cursor"}},
		{"name", "name", "regex.*", []string{"regex.*"}}, {"description", "description", "", []string{""}},
		{"fields", "fields", []any{"is_admin_state_up", nil, "port_id"}, []string{"is_admin_state_up", "port_id"}},
		{"port_id", "port_id", "parent", []string{"parent"}}, {"status", "status", "ACTIVE", []string{"ACTIVE"}},
		{"sub_ports", "sub_ports", []any{false, 0}, []string{"False", "0"}}, {"project_id", "project_id", "project", []string{"project"}},
		{"is_admin_state_up", "admin_state_up", false, []string{"False"}}, {"tags", "tags", []string{"a", "b"}, []string{"a", "b"}},
		{"any_tags", "tags-any", "a", []string{"a"}}, {"not_tags", "not-tags", "a", []string{"a"}}, {"not_any_tags", "not-tags-any", "a", []string{"a"}},
	} {
		t.Run(item.key, func(t *testing.T) {
			var calls []nativeTrunkCall
			api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
				if !reflect.DeepEqual(req.URL.Query()[item.wire], item.want) || len(req.URL.Query()) != 1 {
					t.Fatal(req.URL, item.want)
				}
				return nativeTrunkWire(200, `{"trunks":[]}`)
			})
			rows, err := api.AllRecords(context.Background(), trunks.WithRecordListQuery(item.key, item.value))
			if err != nil || len(rows) != 0 || len(calls) != 1 {
				t.Fatal(rows, err, calls)
			}
		})
	}
	var calls []nativeTrunkCall
	api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.RawQuery != "" {
			t.Fatal(req.URL)
		}
		return nativeTrunkWire(200, `{"trunks":[{"id":"one","status":"unexpected"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), trunks.WithRecordListQuery("x_unknown", make(chan int)), trunks.WithRecordListQuery("sort_key", func() {}), trunks.WithRecordListQuery("all_projects", make(chan int)), trunks.WithRecordListOpts(trunks.RecordListOpts{Query: map[string]json.RawMessage{"x_bad": json.RawMessage("bad"), "name": json.RawMessage("null"), "is_admin_state_up": json.RawMessage("null"), "admin_state_up": json.RawMessage("true"), "tags-any": json.RawMessage(`"wire"`), "any_tags": json.RawMessage("null")}}))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonTrunkListRecordLocalFiltersAliasesAndOrderedFailure(t *testing.T) {
	for _, item := range []struct {
		actual, filter string
		count          int
		valid          bool
	}{
		{`null`, `null`, 1, true}, {`false`, `0`, 1, true}, {`1e-400`, `{}`, 0, true},
		{`"truthy"`, `{}`, 1, true}, {`"truthy"`, `{"k":true}`, 0, false},
		{`{"nested":{"enabled":1,"extra":false}}`, `{"nested":{"enabled":true}}`, 1, true},
		{`{"nested":{}}`, `{"nested":{}}`, 0, true},
	} {
		t.Run(item.actual+item.filter, func(t *testing.T) {
			var calls []nativeTrunkCall
			body := `{"trunks":[{"id":"one","tenant_id":` + item.actual + `,"project_id":"other"}]}`
			api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.RawQuery != "" {
					t.Fatal(req.URL)
				}
				return nativeTrunkWire(203, body)
			})
			rows, err := api.AllRecords(context.Background(), trunks.WithRecordListQuery("tenant_id", json.RawMessage(item.filter)))
			if (err == nil) != item.valid || len(rows) != item.count || len(calls) != 1 {
				t.Fatal(rows, err, calls)
			}
			if !item.valid {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body {
					t.Fatal(err, proof)
				}
			}
		})
	}
	var calls []nativeTrunkCall
	api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeTrunkWire(200, `{"trunks":[{"id":"one","tenant_id":"scalar"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), trunks.WithRecordListQuery("id", "skip"), trunks.WithRecordListQuery("tenant_id", map[string]any{"k": true}))
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	rows, err = api.AllRecords(context.Background(), trunks.WithRecordListQuery("tenant_id", map[string]any{"k": true}), trunks.WithRecordListQuery("id", "skip"))
	var proof *resource.ResponseError
	if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 200 {
		t.Fatal(rows, err, proof)
	}
}

func TestPythonTrunkListRecordPagingRepresentationsAndSourceNextTruthiness(t *testing.T) {
	for _, lane := range []string{"plural", "next", "http", "dictionary", "versioned", "underflow", "negative underflow", "truthy nonstring", "empty"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeTrunkCall
			firstBody := `{"trunks":[{"id":"one"}]}`
			api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 2 {
					if req.URL.Query().Get("marker") != "next" {
						t.Fatal(req.URL)
					}
					return nativeTrunkWire(200, `{"trunks":[]}`)
				}
				switch lane {
				case "plural":
					firstBody = `{"trunks":[{"id":"one"}],"trunks_links":[{"rel":"next","href":"?marker=next"}]}`
				case "next":
					firstBody = `{"trunks":[{"id":"one"}],"next":"?marker=next"}`
				case "dictionary":
					firstBody = `{"trunks":[{"id":"one"}],"links":{"next":"?marker=next"}}`
				case "versioned":
					firstBody = `{"trunks":[{"id":"one"}],"next":"/v2.0/trunks?marker=next"}`
				case "underflow":
					firstBody = `{"trunks":[{"id":"one"}],"next":1e-400}`
				case "negative underflow":
					firstBody = `{"trunks":[{"id":"one"}],"next":-1e-400}`
				case "truthy nonstring":
					firstBody = `{"trunks":[{"id":"one"}],"next":1e-300}`
				case "empty":
					firstBody = `{"trunks":[],"next":"https://foreign.example/trunks"}`
				}
				r := nativeTrunkWire(203, firstBody)
				r.Header.Set("X-Request-Id", "next-proof")
				if lane == "http" {
					r.Header.Set("Link", `<?marker=next>; rel="next"`)
				}
				return r
			})
			rows, err := api.AllRecords(context.Background())
			if lane == "truthy nonstring" {
				var proof *resource.ResponseError
				if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != firstBody || proof.Header.Get("X-Request-Id") != "next-proof" || len(calls) != 1 {
					t.Fatal(rows, err, proof, calls)
				}
				return
			}
			wantCalls := 2
			wantRows := 1
			if lane == "underflow" || lane == "negative underflow" || lane == "empty" {
				wantCalls = 1
			}
			if lane == "empty" {
				wantRows = 0
			}
			if err != nil || len(rows) != wantRows || len(calls) != wantCalls {
				t.Fatal(rows, err, calls)
			}
			if len(rows) > 0 && string(rows[0].Envelope) != firstBody {
				t.Fatal(rows)
			}
		})
	}
}

func TestPythonTrunkListRecordRawCapMarkerAndConsumerStop(t *testing.T) {
	var calls []nativeTrunkCall
	api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("limit") != "2" {
			t.Fatal(req.URL)
		}
		return nativeTrunkWire(200, `{"trunks":[{"id":"skip","tenant_id":"other"},{"id":"keep","tenant_id":"tenant"},{"connection":null}],"next":"https://foreign.example/trunks"}`)
	})
	rows, err := api.AllRecords(context.Background(), trunks.WithRecordListQuery("tenant_id", "tenant"), trunks.WithRecordListLimit(0), trunks.WithRecordListMaxItems(2))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	pythonFindField(t, rows[0], "id", `"keep"`)
	calls = nil
	api, _ = nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 2 {
			if req.URL.Query().Get("marker") != "42" || req.URL.Query().Get("limit") != "4" {
				t.Fatal(req.URL)
			}
			return nativeTrunkWire(200, `{"trunks":[]}`)
		}
		return nativeTrunkWire(200, `{"trunks":[{"id":42,"name":"passive"}]}`)
	})
	for v, e := range api.ListRecords(context.Background(), trunks.WithRecordListLimit(4)) {
		if e != nil {
			t.Fatal(e)
		}
		v.Resource.Body["id"] = json.RawMessage(`"changed"`)
	}
	if len(calls) != 2 {
		t.Fatal(calls)
	}
	calls = nil
	api, _ = nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeTrunkWire(200, `{"trunks":[{"id":"one"},{"connection":null}],"next":"https://foreign.example/trunks"}`)
	})
	for _, e := range api.ListRecords(context.Background()) {
		if e != nil {
			t.Fatal(e)
		}
		break
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
	rows, err = api.AllRecords(context.Background(), trunks.WithRecordListPaginated(false), trunks.WithRecordListMaxItems(1))
	if err != nil || len(rows) != 1 || len(calls) != 2 {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonTrunkListRecordMalformedConstructorStatusAndPermissionProof(t *testing.T) {
	for _, body := range []string{`{bad`, `null`, `{}`, `{"trunks":null}`, `{"trunks":[null]}`, `{"trunks":[false]}`, `{"trunks":[{"connection":null}]}`, `{"trunks":[{"microversion":null}]}`, `{"trunks":[{"_synchronized":null}]}`, `{"trunks":[{"cls":null}]}`} {
		var calls []nativeTrunkCall
		api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeTrunkWire(203, body)
			r.Header.Set("X-Request-Id", "constructor")
			return r
		})
		rows, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "constructor" || len(calls) != 1 {
			t.Fatal(body, rows, err, proof, calls)
		}
	}
	for _, status := range []int{199, 200, 204, 299, 300, 399, 400, 403, 404, 503} {
		var calls []nativeTrunkCall
		api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response { return nativeTrunkWire(status, `{"trunks":[{}]}`) })
		api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, cause error, _ uint) error {
			return cause
		}
		rows, err := api.AllRecords(context.Background())
		valid := status >= 200 && status < 400
		if (err == nil) != valid || len(calls) != 1 || valid && len(rows) != 1 || !valid && rows != nil {
			t.Fatal(status, rows, err, calls)
		}
		if !valid {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != status {
				t.Fatal(err, native)
			}
		}
	}
	var calls []nativeTrunkCall
	api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response { return nativeTrunkWire(204, "") })
	rows, err := api.AllRecords(context.Background())
	var proof *resource.ResponseError
	if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 204 || len(calls) != 1 {
		t.Fatal(rows, err, proof, calls)
	}
}

func TestPythonTrunkListRecordLateErrorsDiscardAllAndDuplicatesStayRows(t *testing.T) {
	for _, fail := range []bool{false, true} {
		var calls []nativeTrunkCall
		api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 2 {
				if fail {
					return nativeTrunkWire(403, `{"forbidden":"later"}`)
				}
				return nativeTrunkWire(200, `{"trunks":[{"id":"same","name":"same"}]}`)
			}
			return nativeTrunkWire(200, `{"trunks":[{"id":"same","name":"same"}],"next":"?marker=next"}`)
		})
		rows, err := api.AllRecords(context.Background())
		if fail {
			if rows != nil || !gophercloud.ResponseCodeIs(err, 403) {
				t.Fatal(rows, err)
			}
		} else if err != nil || len(rows) != 2 || rows[0] == rows[1] {
			t.Fatal(rows, err)
		}
		if len(calls) != 2 {
			t.Fatal(calls)
		}
		var wrapped *resource.OperationError
		if fail && (!errors.As(err, &wrapped) || wrapped.Operation != "AllRecords" || wrapped.Resource != "trunks") {
			t.Fatal(err, wrapped)
		}
	}
}

func TestPythonTrunkListRecordOptionOwnershipLazyReuseAndPreflight(t *testing.T) {
	var calls []nativeTrunkCall
	var retained *trunks.RecordListOpts
	api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
		retained.Query["name"] = json.RawMessage(`"retained mutation"`)
		retained.Headers["X-Trace"] = "mutated"
		*retained.Microversion = "changed"
		if req.URL.Query().Get("name") != "original" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.4" {
			t.Fatal(req)
		}
		return nativeTrunkWire(200, `{"trunks":[{}]}`)
	})
	query := map[string]json.RawMessage{"name": json.RawMessage(`"original"`)}
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.4"
	owned := trunks.WithRecordListOpts(trunks.RecordListOpts{Query: query, Headers: headers, Microversion: &version})
	query["name"][1] = 'X'
	headers["X-Trace"] = "changed"
	version = "changed"
	callbacks := 0
	options := []trunks.RecordListOption{owned, func(c *trunks.RecordListOpts) error { callbacks++; retained = c; return nil }}
	seq := api.ListRecords(context.Background(), options...)
	options[0] = nil
	if len(calls) != 0 || callbacks != 0 {
		t.Fatal(calls, callbacks)
	}
	for i := 0; i < 2; i++ {
		count := 0
		for _, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			count++
		}
		if count != 1 {
			t.Fatal(count)
		}
	}
	if len(calls) != 2 || callbacks != 2 {
		t.Fatal(calls, callbacks)
	}
	calls = nil
	api, _ = nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return nil })
	for _, option := range []trunks.RecordListOption{nil, trunks.WithRecordListLimit(-1), trunks.WithRecordListMaxItems(-1), trunks.WithRecordListQuery("name", json.RawMessage("bad")), trunks.WithRecordListQuery("cls", nil), trunks.WithRecordListQuery("allow_unknown_params", true), trunks.WithRecordListQuery("jmespath_filters", "[*]"), trunks.WithRecordListQuery("location", nil), trunks.WithRecordListQuery("base_path", nil), trunks.WithRecordListQuery("headers", nil), trunks.WithRecordListQuery("session", nil), trunks.WithRecordListQuery("microversion", nil), trunks.WithRecordListQuery("bad\n", nil), trunks.WithRecordListHeaders(map[string]string{"X-Auth-Token": "fake"}), trunks.WithRecordListHeaders(map[string]string{"OpenStack-API-Version": "network 2.0"})} {
		rows, err := api.AllRecords(context.Background(), option)
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(rows, err, calls)
		}
	}
	for _, ctx := range []context.Context{nil, func() context.Context { c, cancel := context.WithCancel(context.Background()); cancel(); return c }()} {
		rows, err := api.AllRecords(ctx)
		if rows != nil || err == nil || len(calls) != 0 {
			t.Fatal(rows, err, calls)
		}
	}
	var nilAPI *trunks.API
	rows, err := nilAPI.AllRecords(context.Background())
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(rows, err)
	}
	cause := errors.New("option cause")
	source := api.RawClient()
	original := source.ResourceBase
	rows, err = api.AllRecords(context.Background(), func(*trunks.RecordListOpts) error { source.ResourceBase += "changed/"; return cause })
	source.ResourceBase = original
	if rows != nil || !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(rows, err, calls)
	}
	originalAPI := *api
	replacement := *source
	rows, err = api.AllRecords(context.Background(), func(*trunks.RecordListOpts) error { *api = *trunks.New(&replacement); return nil })
	*api = originalAPI
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(rows, err, calls)
	}
	rows, err = api.AllRecords(context.Background(), func(c *trunks.RecordListOpts) error { c.Query = nil; c.Headers = nil; return nil }, trunks.WithRecordListQuery("name", "valid"), func(c *trunks.RecordListOpts) error { c.Headers["X-Trace"] = "valid"; return cause })
	if rows != nil || !errors.Is(err, cause) || len(calls) != 0 {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonTrunkListRecordExplicitAndAbsentVersionsAcrossPages(t *testing.T) {
	for _, item := range []struct {
		selected, override string
		explicit           bool
	}{{"", "", false}, {"2.0", "", false}, {"2.0", "2.4", true}, {"2.0", "", true}, {"", "latest", true}} {
		var calls []nativeTrunkCall
		api, _ := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
			version := item.selected
			if item.explicit {
				version = item.override
			}
			want := ""
			if version != "" {
				want = "network " + version
			}
			if req.Header.Get("OpenStack-API-Version") != want || want == "" && len(req.Header.Values("OpenStack-API-Version")) != 0 || req.Header.Get("Accept") != "explicit" {
				t.Fatal(req.Header, item)
			}
			if len(calls) == 2 {
				return nativeTrunkWire(200, `{"trunks":[]}`)
			}
			return nativeTrunkWire(200, `{"trunks":[{"id":"one"}],"next":"?marker=next"}`)
		})
		source := api.RawClient()
		source.Microversion = item.selected
		source.MoreHeaders = map[string]string{"Accept": "configured"}
		if item.selected != "" {
			source.MoreHeaders["openstack-api-version"] = "network " + item.selected
		}
		before := make(map[string]string)
		for k, v := range source.MoreHeaders {
			before[k] = v
		}
		options := []trunks.RecordListOption{trunks.WithRecordListHeaders(map[string]string{"aCcEpT": "explicit"})}
		if item.explicit {
			options = append(options, trunks.WithRecordListMicroversion(item.override))
		}
		rows, err := api.AllRecords(context.Background(), options...)
		if err != nil || len(rows) != 1 || len(calls) != 2 || source.Microversion != item.selected || !reflect.DeepEqual(source.MoreHeaders, before) {
			t.Fatal(rows, err, calls, source)
		}
	}
}

type pythonTrunkListFaultBody struct {
	*pythonFindFaultBody
	onRead func()
	closes int
}

func (b *pythonTrunkListFaultBody) Read(p []byte) (int, error) {
	if b.onRead != nil {
		f := b.onRead
		b.onRead = nil
		f()
	}
	return b.pythonFindFaultBody.Read(p)
}
func (b *pythonTrunkListFaultBody) Close() error { b.closes++; return b.pythonFindFaultBody.Close() }

func TestPythonTrunkListRecordPhysicalMixedFaultsNoReplayAndActualReceipt(t *testing.T) {
	for _, status := range []int{200, 400, 401, 403, 404, 409, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []nativeTrunkCall
			readCause, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var physical *pythonTrunkListFaultBody
			api, cloud := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response {
				r := nativeTrunkWire(status, "")
				r.Body = physical
				r.Header.Set("X-Request-Id", "physical")
				return r
			})
			source := api.RawClient()
			original := source.ResourceBase
			originalURL := source.ServiceURL("trunks")
			defer func() { source.ResourceBase = original }()
			body := `{"trunks":[{"id":"one"}],"actual":"physical"}`
			physical = &pythonTrunkListFaultBody{pythonFindFaultBody: &pythonFindFaultBody{data: body, read: readCause, close: closeCause, onClose: func() { source.ResourceBase = original }}, onRead: func() { source.ResourceBase += "changed/"; cancel(cancelCause) }}
			retries, reauth := 0, 0
			source.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			cloud.Provider.ReauthFunc = func(context.Context) error { reauth++; return nil }
			rows, err := api.AllRecords(ctx)
			if rows != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 || physical.closes != 1 || retries != 0 || reauth != 0 {
				t.Fatal(rows, err, calls, physical.closes, retries, reauth)
			}
			if status < 400 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "physical" {
					t.Fatal(err, proof)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				original := err
				if status == http.StatusUnauthorized {
					// Gophercloud intentionally does not Unwrap two independent
					// reauthentication failures. Its public fields own both proofs.
					var reauthentication *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &reauthentication) || !errors.Is(reauthentication.ErrReauth, readCause) || !errors.Is(reauthentication.ErrReauth, closeCause) || !errors.Is(reauthentication.ErrReauth, cancelCause) || !errors.Is(reauthentication.ErrReauth, context.Canceled) || !errors.Is(reauthentication.ErrReauth, resource.ErrInvalidOption) {
						t.Fatal(err, reauthentication)
					}
					original = reauthentication.ErrOriginal
				}
				if !errors.As(original, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "physical" || native.Method != http.MethodGet || native.URL != originalURL {
					t.Fatal(err, native)
				}
			}
		})
	}
}

func TestPythonTrunkListRecordNativeRetryOwnershipAndCleanReauthentication(t *testing.T) {
	for _, lane := range []string{"retry", "reauth", "body", "version", "omit version", "authorization", "service token", "host", "framing", "foreign version", "custom"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeTrunkCall
			var tokens []string
			api, cloud := nativeTrunkAPI(t, &calls, func(req *http.Request) *http.Response {
				tokens = append(tokens, req.Header.Get("X-Auth-Token"))
				if len(calls) == 1 {
					status := 503
					if lane == "reauth" {
						status = 401
					}
					r := nativeTrunkWire(status, `{"retry":"original"}`)
					r.Header.Set("X-Request-Id", "original")
					return r
				}
				return nativeTrunkWire(200, `{"trunks":[{}]}`)
			})
			client := api.RawClient()
			client.Microversion = "2.0"
			custom := errors.New("callback")
			if lane == "reauth" {
				cloud.Provider.ReauthFunc = func(context.Context) error { cloud.Provider.SetToken("renewed"); return nil }
			} else {
				client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
					switch lane {
					case "body":
						o.JSONBody = map[string]any{"changed": true}
					case "version":
						o.MoreHeaders["OpenStack-API-Version"] = "network 2.5"
					case "omit version":
						o.OmitHeaders = []string{"OpenStack-API-Version"}
					case "authorization":
						o.MoreHeaders["aUtHoRiZaTiOn"] = "fake"
					case "service token":
						o.MoreHeaders["X-Service-Token"] = "fake"
					case "host":
						o.MoreHeaders["Host"] = "fake"
					case "framing":
						o.MoreHeaders["Content-Length"] = "2"
					case "foreign version":
						o.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.5"
					case "custom":
						return custom
					}
					return nil
				}
			}
			rows, err := api.AllRecords(context.Background(), trunks.WithRecordListMicroversion("2.4"), trunks.WithRecordListQuery("name", "owned"))
			if lane == "retry" || lane == "reauth" {
				if err != nil || len(rows) != 1 || len(calls) != 2 || calls[0] != calls[1] {
					t.Fatal(rows, err, calls)
				}
				if lane == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
					t.Fatal(tokens)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if rows != nil || len(calls) != 1 || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"retry":"original"}` || native.ResponseHeader.Get("X-Request-Id") != "original" || lane == "custom" && !errors.Is(err, custom) || lane != "custom" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(rows, err, native, calls)
				}
			}
		})
	}
}

func TestPythonTrunkListRecordConcurrentSnapshotsAndPaginationTruthiness(t *testing.T) {
	var calls []nativeTrunkCall
	api, cloud := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("initial transport"); return nil })
	var observed atomic.Int32
	cloud.Provider.HTTPClient.Transport = nativeTrunkTransport(func(req *http.Request) (*http.Response, error) {
		observed.Add(1)
		if req.URL.Query().Get("name") != "owned" || req.Header.Get("X-Trace") != "owned" {
			t.Error(req)
		}
		return nativeTrunkWire(200, `{"trunks":[{"id":"one"}]}`), nil
	})
	option := trunks.WithRecordListOpts(trunks.RecordListOpts{Query: map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)}, Headers: map[string]string{"X-Trace": "owned"}})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := api.AllRecords(context.Background(), option)
			if err != nil || len(rows) != 1 {
				t.Error(rows, err)
				return
			}
			rows[0].Resource.Body["id"][1] = 'X'
			rows[0].Header.Set("X-Trace", "changed")
			if string(rows[0].Wire.Body["id"]) != `"one"` {
				t.Error(rows)
			}
		}()
	}
	wg.Wait()
	if observed.Load() != 8 {
		t.Fatal(observed.Load())
	}
	for _, item := range []struct {
		raw   string
		pages int
	}{{"null", 1}, {"false", 1}, {"0", 1}, {"[]", 1}, {"1e-400", 1}, {`"False"`, 2}, {"1e400", 2}} {
		var seen []nativeTrunkCall
		api, _ := nativeTrunkAPI(t, &seen, func(*http.Request) *http.Response {
			if len(seen) == 2 {
				return nativeTrunkWire(200, `{"trunks":[]}`)
			}
			return nativeTrunkWire(200, `{"trunks":[{"id":"one"}],"next":"?marker=next"}`)
		})
		rows, err := api.AllRecords(context.Background(), trunks.WithRecordListQuery("paginated", json.RawMessage(item.raw)))
		if err != nil || len(rows) != 1 || len(seen) != item.pages {
			t.Fatal(item, rows, err, seen)
		}
	}
}

func TestPythonTrunkListRecordFixedContinuationGuardsAndWholePageReceipt(t *testing.T) {
	for _, next := range []string{"https://foreign.example/trunks?marker=next", "/other/trunks?marker=next", "?marker=next&project_id=changed", "?marker=same"} {
		var calls []nativeTrunkCall
		body := `{"trunks":[{"id":"one"}],"next":"` + next + `"}`
		api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeTrunkWire(203, body)
			r.Header.Set("X-Request-Id", "continuation")
			return r
		})
		rows, err := api.AllRecords(context.Background(), trunks.WithRecordListQuery("project_id", "fixed"), trunks.WithRecordListMarker("same"))
		var proof *resource.ResponseError
		if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "continuation" || len(calls) != 1 {
			t.Fatal(next, rows, err, proof, calls)
		}
	}
}

// A caller's custom JSON conversion still owns its error; the library adds
// option classification without serializing unknown Source kwargs.
type pythonTrunkListMarshalValue struct {
	cause error
	calls *int
}

func (v pythonTrunkListMarshalValue) MarshalJSON() ([]byte, error) {
	(*v.calls)++
	return nil, v.cause
}

func TestPythonTrunkListRecordKnownMarshalErrorsPreflightAndUnknownDiscard(t *testing.T) {
	var calls []nativeTrunkCall
	api, _ := nativeTrunkAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeTrunkWire(200, `{"trunks":[{"id":"one"}]}`)
	})
	cause := errors.New("caller JSON conversion failed")
	conversions := 0
	value := pythonTrunkListMarshalValue{cause: cause, calls: &conversions}
	known := trunks.WithRecordListQuery("name", value)
	if conversions != 1 || len(calls) != 0 {
		t.Fatal(conversions, calls)
	}
	callbacks := 0
	rows, err := api.AllRecords(context.Background(), known, func(*trunks.RecordListOpts) error { callbacks++; return nil })
	var marshal *json.MarshalerError
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cause) || !errors.As(err, &marshal) || callbacks != 0 || conversions != 1 || len(calls) != 0 {
		t.Fatal(rows, err, marshal, callbacks, conversions, calls)
	}
	rows, err = api.AllRecords(context.Background(), trunks.WithRecordListQuery("undeclared_kwargs", value), trunks.WithRecordListQuery("all_projects", func() {}))
	if err != nil || len(rows) != 1 || len(calls) != 1 || calls[0].query != "" || conversions != 1 {
		t.Fatal(rows, err, calls, conversions)
	}
	calls = nil
	rows, err = api.AllRecords(context.Background(), trunks.WithRecordListQuery("name", func() {}))
	var unsupported *json.UnsupportedTypeError
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &unsupported) || len(calls) != 0 {
		t.Fatal(rows, err, unsupported, calls)
	}
}
