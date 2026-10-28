package consistencygroups_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type cgCall struct {
	method, path, escaped, query, body string
	headers                            http.Header
}

func cgWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual"}})
}

const cgBase = "/cinder/v3/project/consistencygroups"

func cgAPI(t *testing.T, version string, reply func(*http.Request) *http.Response) (*consistencygroups.API, *testcloud.Recorder) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := testcloud.NewRecorder(func(r *http.Request) (*http.Response, error) { return reply(r), nil })
	cloud.SetTransport(calls)
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = version
	return consistencygroups.New(client), calls
}

// cgCalls projects recorded calls onto the local call fields.
func cgCalls(rec *testcloud.Recorder) []cgCall {
	calls := []cgCall{}
	for _, call := range rec.Calls() {
		calls = append(calls, cgCall{call.Method, call.Path, call.EscapedPath, call.RawQuery, call.Body, call.Header})
	}
	return calls
}
func cgField(t *testing.T, v *consistencygroups.ConsistencyGroupRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatal(key, want, v)
	}
}
func cgJSON(t *testing.T, got, want string) {
	t.Helper()
	var a, b any
	if json.Unmarshal([]byte(got), &a) != nil || json.Unmarshal([]byte(want), &b) != nil || !reflect.DeepEqual(a, b) {
		t.Fatal(got, want)
	}
}
func collectCG(seq func(func(*consistencygroups.ConsistencyGroupRecord, error) bool)) ([]*consistencygroups.ConsistencyGroupRecord, error) {
	var values []*consistencygroups.ConsistencyGroupRecord
	for v, err := range seq {
		if err != nil {
			return values, err
		}
		values = append(values, v)
	}
	return values, nil
}

func TestPythonCinderConsistencyGroupGetLiteralProjectionDefaultVersionAndNoCache(t *testing.T) {
	body := `{"consistencygroup":{"id":null,"name":false,"volume_types":"type-a,type-b","status":9007199254740993,"cls":false,"unknown":9007199254740995}}`
	for _, version := range []string{"", "3.90"} {
		api, calls := cgAPI(t, version, func(*http.Request) *http.Response { return cgWire(200, body) })
		for range 2 {
			value, err := api.GetRecord(context.Background(), "literal/id %2F")
			if err != nil || value == nil || value.StatusCode != 200 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "actual" || len(value.Resource.Body) != 10 || value.Resource.Body["unknown"] != nil || string(value.Wire.Body["unknown"]) != "9007199254740995" || string(value.Wire.Body["cls"]) != "false" {
				t.Fatal(value, err, cgCalls(calls))
			}
			for key, want := range map[string]string{"id": "null", "name": "false", "volume_types": `["type-a,type-b"]`, "status": "9007199254740993", "availability_zone": "null", "location": "null"} {
				cgField(t, value, key, want)
			}
			value.Resource.Body["volume_types"][2] = 'X'
			value.Header.Set("X-Request-Id", "changed")
			if string(value.Wire.Body["volume_types"]) != `"type-a,type-b"` || value.Resource.Header.Get("X-Request-Id") != "actual" || value.Wire.Header.Get("X-Request-Id") != "actual" {
				t.Fatal("aliased record", value)
			}
		}
		if calls.Len() != 2 || api.RawClient().Microversion != version {
			t.Fatal(cgCalls(calls), api.RawClient())
		}
		for _, call := range cgCalls(calls) {
			expected := ""
			if version != "" {
				expected = "volume " + version
			}
			if call.method != http.MethodGet || call.escaped != cgBase+"/literal%2Fid%20%252F" || call.query != "" || call.body != "" || call.headers.Get("OpenStack-API-Version") != expected {
				t.Fatal(call)
			}
		}
	}
}

func TestPythonCinderConsistencyGroupAcceptedDecoderKeepsActualEnvelopeAndSeeds(t *testing.T) {
	for _, example := range []struct {
		body  string
		valid bool
	}{{`{"consistencygroup":{}}`, true}, {`{"name":0,"unknown":false}`, true}, {"", true}, {"{bad", true}, {`null`, false}, {`[]`, false}, {`{"consistencygroup":null}`, false}, {`{"consistencygroup":[]}`, false}, {string([]byte{0xff, '{'}), false}} {
		api, calls := cgAPI(t, "", func(*http.Request) *http.Response { return cgWire(202, example.body) })
		value, err := api.GetRecord(context.Background(), "seed")
		if value == nil || (err == nil) != example.valid || calls.Len() != 1 || value.StatusCode != 202 || string(value.Envelope) != example.body || value.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(example, value, err, cgCalls(calls))
		}
		if !example.valid {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != example.body || proof.Header.Get("X-Request-Id") != "actual" {
				t.Fatal(err, proof)
			}
		} else {
			cgField(t, value, "id", `"seed"`)
		}
	}
}

func TestPythonCinderConsistencyGroupCreateRawPartialNullAndSnapshotOwnership(t *testing.T) {
	fields := map[string]json.RawMessage{"volume_types": json.RawMessage(`"type-a,type-b"`), "name": json.RawMessage(`null`), "description": json.RawMessage(`false`), "unknown": json.RawMessage(`invalid-ignored`)}
	headers := map[string]string{"X-Option": "owned"}
	version := "3.90"
	option := consistencygroups.WithRecordCreateOpts(consistencygroups.RecordCreateOpts{Fields: fields, Headers: headers, Microversion: &version})
	fields["volume_types"][1] = 'X'
	headers["X-Option"] = "changed"
	version = "3.1"
	var held *consistencygroups.RecordCreateOpts
	api, calls := cgAPI(t, "", func(*http.Request) *http.Response {
		if held != nil {
			held.Fields["volume_types"][1] = 'Y'
			held.Headers["X-Option"] = "retained"
			*held.Microversion = "3.1"
		}
		return cgWire(202, `{"consistencygroup":{"id":"new","name":"wire"}}`)
	})
	for range 2 {
		value, err := api.CreateRecord(context.Background(), option, func(c *consistencygroups.RecordCreateOpts) error { held = c; return nil })
		if err != nil || value == nil || value.StatusCode != 202 || api.RawClient().Microversion != "" {
			t.Fatal(value, err, cgCalls(calls))
		}
		cgField(t, value, "volume_types", `["type-a,type-b"]`)
		cgField(t, value, "description", `false`)
		cgField(t, value, "name", `"wire"`)
	}
	for _, call := range cgCalls(calls) {
		if call.method != http.MethodPost || call.path != cgBase || call.headers.Get("X-Option") != "owned" || call.headers.Get("OpenStack-API-Version") != "volume 3.90" {
			t.Fatal(call)
		}
		cgJSON(t, call.body, `{"consistencygroup":{"volume_types":"type-a,type-b","name":null,"description":false}}`)
	}
}

func TestPythonCinderConsistencyGroupCreateFromSourceNullDefaultsAndLiteralReferences(t *testing.T) {
	for _, variant := range []string{"default", "empty_id", "group", "snapshot", "both"} {
		var options []consistencygroups.RecordSourceOption
		want := `{"consistencygroup-from-src":{"cgsnapshot_id":null,"source_cgid":null,"name":null,"description":null}}`
		switch variant {
		case "empty_id":
			options = append(options, consistencygroups.WithRecordSourceConsistencyGroupID(""))
		case "group":
			options = append(options, consistencygroups.WithRecordSourceConsistencyGroupID("literal/id"), consistencygroups.WithRecordSourceName(""))
			want = `{"consistencygroup-from-src":{"cgsnapshot_id":null,"source_cgid":"literal/id","name":"","description":null}}`
		case "snapshot":
			options = append(options, consistencygroups.WithRecordSourceConsistencyGroupSnapshotID("snapshot/%"), consistencygroups.WithRecordSourceDescription(""))
			want = `{"consistencygroup-from-src":{"cgsnapshot_id":"snapshot/%","source_cgid":null,"name":null,"description":""}}`
		case "both":
			options = append(options, consistencygroups.WithRecordSourceConsistencyGroupSnapshotID("snapshot"), consistencygroups.WithRecordSourceConsistencyGroupID("group"))
			want = `{"consistencygroup-from-src":{"cgsnapshot_id":"snapshot","source_cgid":"group","name":null,"description":null}}`
		}
		accepted := variant == "group" || variant == "snapshot"
		api, calls := cgAPI(t, "", func(*http.Request) *http.Response {
			if accepted {
				return cgWire(202, `{"consistencygroup":{"id":"new"}}`)
			}
			return cgWire(400, `{"stock":"exactly one source is required"}`)
		})
		value, err := api.CreateFromSourceRecord(context.Background(), options...)
		if value == nil || calls.Len() != 1 || cgCalls(calls)[0].method != http.MethodPost || cgCalls(calls)[0].path != cgBase+"/create_from_src" || cgCalls(calls)[0].query != "" || (err == nil) != accepted {
			t.Fatal(variant, value, err, cgCalls(calls))
		}
		cgJSON(t, cgCalls(calls)[0].body, want)
		if accepted {
			cgField(t, value, "name", `null`)
			cgField(t, value, "description", `null`)
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 400 || value.StatusCode != 400 || string(native.Body) != string(value.Envelope) {
				t.Fatal(value, err)
			}
		}
	}
}

func TestPythonCinderConsistencyGroupUpdateDirtyNoopBlankVersionAndResponseMerge(t *testing.T) {
	api, calls := cgAPI(t, "", func(*http.Request) *http.Response { t.Fatal("no-op HTTP"); return nil })
	for _, options := range [][]consistencygroups.RecordUpdateOption{nil, {consistencygroups.WithRecordUpdateField("unknown", make(chan int))}, {consistencygroups.WithRecordUpdateField("id", "ignored")}} {
		value, err := api.UpdateRecord(context.Background(), "literal", options...)
		if err != nil || value == nil || value.StatusCode != 0 || value.Envelope != nil || value.Header != nil || value.Wire != nil || calls.Len() != 0 {
			t.Fatal(value, err, cgCalls(calls))
		}
		cgField(t, value, "id", `"literal"`)
	}
	for _, version := range []string{"", "3.5", "3.6", "3.90"} {
		accepted := version == "3.6" || version == "3.90"
		api, calls := cgAPI(t, version, func(*http.Request) *http.Response {
			if accepted {
				return cgWire(202, "")
			}
			return cgWire(400, `{"stock":"blank update requires3.6"}`)
		})
		value, err := api.UpdateRecord(context.Background(), "id", consistencygroups.WithRecordUpdateField("name", ""), consistencygroups.WithRecordUpdateField("id", "other"))
		if value == nil || calls.Len() != 1 || cgCalls(calls)[0].method != http.MethodPut || cgCalls(calls)[0].path != cgBase+"/id" || (err == nil) != accepted {
			t.Fatal(version, value, err, cgCalls(calls))
		}
		cgJSON(t, cgCalls(calls)[0].body, `{"consistencygroup":{"name":""}}`)
		if accepted {
			cgField(t, value, "id", `"id"`)
			cgField(t, value, "name", `""`)
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 400 || string(value.Envelope) != string(native.Body) {
				t.Fatal(value, err)
			}
		}
	}
}

func TestPythonCinderConsistencyGroupDeleteCorrectedBodyReplayAndStrictMissing(t *testing.T) {
	for _, choice := range []string{"default", "force", "source", "missing"} {
		api, calls := cgAPI(t, "", func(*http.Request) *http.Response {
			if choice == "source" {
				return cgWire(400, `{"stock":"consistencygroup key required"}`)
			}
			if choice == "missing" {
				return cgWire(404, `{"missing":"actual"}`)
			}
			return cgWire(202, `invalid ignored ack`)
		})
		var options []consistencygroups.RecordDeleteOption
		want := `{"consistencygroup":{"force":false}}`
		if choice == "force" {
			options = append(options, consistencygroups.WithRecordDeleteForce(true))
			want = `{"consistencygroup":{"force":true}}`
		}
		if choice == "source" {
			options = append(options, consistencygroups.WithRecordDeletePinnedSourceWire(true))
			want = `{"consistency_group":{"force":false}}`
		}
		err := api.DeleteRecord(context.Background(), "literal/id %2F", options...)
		if calls.Len() != 1 || cgCalls(calls)[0].method != http.MethodPost || cgCalls(calls)[0].escaped != cgBase+"/literal%2Fid%20%252F/delete" || cgCalls(calls)[0].headers.Get("OpenStack-API-Version") != "" || (err == nil) != (choice == "default" || choice == "force") {
			t.Fatal(choice, err, cgCalls(calls))
		}
		cgJSON(t, cgCalls(calls)[0].body, want)
		if choice == "source" || choice == "missing" {
			var native gophercloud.ErrUnexpectedResponseCode
			code := 400
			if choice == "missing" {
				code = 404
			}
			if !errors.As(err, &native) || native.Actual != code || native.ResponseHeader.Get("X-Request-Id") != "actual" {
				t.Fatal(err)
			}
		}
	}
}

func TestPythonCinderConsistencyGroupListDeclaredQueryLocalFiltersAndRawCap(t *testing.T) {
	body := `{"consistencygroups":[{"id":"first","name":"other"},{"id":"second","name":"target","volume_types":"type"}],"next":"?marker=next"}`
	api, calls := cgAPI(t, "", func(*http.Request) *http.Response { return cgWire(200, body) })
	values, err := collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListQuery("name", "target"), consistencygroups.WithRecordListQuery("sort", []string{"name:asc", "id:desc"}), consistencygroups.WithRecordListQuery("all_tenants", make(chan int)), consistencygroups.WithRecordListQuery("unknown", make(chan int)), consistencygroups.WithRecordListMaxItems(2)))
	if err != nil || len(values) != 1 || calls.Len() != 1 || cgCalls(calls)[0].path != cgBase+"/detail" || cgCalls(calls)[0].query != "limit=2&sort=name%3Aasc&sort=id%3Adesc" {
		t.Fatal(values, err, cgCalls(calls))
	}
	cgField(t, values[0], "id", `"second"`)
	cgField(t, values[0], "volume_types", `["type"]`)
	api, calls = cgAPI(t, "", func(*http.Request) *http.Response { return cgWire(200, body) })
	values, err = collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListQuery("name", "target"), consistencygroups.WithRecordListMaxItems(1)))
	if err != nil || len(values) != 0 || calls.Len() != 1 || cgCalls(calls)[0].query != "limit=1" {
		t.Fatal(values, err, cgCalls(calls))
	}
}

func TestPythonCinderConsistencyGroupListStockSingularLinksOffsetsMarkersAndBreak(t *testing.T) {
	api, calls := cgAPI(t, "3.90", func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "second" {
			return cgWire(200, `{"consistencygroups":[]}`)
		}
		if r.URL.Query().Get("offset") == "2" {
			return cgWire(200, `{"consistencygroups":[{"id":"second","name":"target"}]}`)
		}
		return cgWire(200, `{"consistencygroups":[{"id":"first","name":"target"}],"consistencygroup_links":[{"rel":"next","href":"?offset=2&limit=2"}]}`)
	})
	values, err := collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListDetails(false), consistencygroups.WithRecordListLimit(2), consistencygroups.WithRecordListQuery("sort", "name:asc"), consistencygroups.WithRecordListQuery("name", "target")))
	if err != nil || len(values) != 2 || calls.Len() != 3 || cgCalls(calls)[0].path != cgBase || cgCalls(calls)[1].query != "limit=2&offset=2&sort=name%3Aasc" || cgCalls(calls)[2].query != "limit=2&marker=second&sort=name%3Aasc" {
		t.Fatal(values, err, cgCalls(calls))
	}
	for _, call := range cgCalls(calls) {
		if call.headers.Get("OpenStack-API-Version") != "volume 3.90" || call.headers.Get("Accept") != "application/json" || call.path != cgBase {
			t.Fatal(call)
		}
	}
	api, calls = cgAPI(t, "", func(*http.Request) *http.Response {
		return cgWire(200, `{"consistencygroups":[{"id":"first"}],"next":"?offset=2"}`)
	})
	for value, err := range api.ListRecords(context.Background()) {
		if err != nil || value == nil {
			t.Fatal(value, err)
		}
		break
	}
	if calls.Len() != 1 {
		t.Fatal("consumer stop must not read next", cgCalls(calls))
	}
}

func TestPythonCinderConsistencyGroupListContinuationRejectsChangedScopeAndCursor(t *testing.T) {
	for _, next := range []string{"?sort=id:desc&offset=2", "?offset=0", "?offset=2&marker=x", "https://other.example/consistencygroups", "/cinder/v3/other/consistencygroups/detail?offset=2", "?limit=3&offset=2"} {
		body := `{"consistencygroups":[{"id":"first"}],"next":"` + next + `"}`
		api, calls := cgAPI(t, "", func(*http.Request) *http.Response { return cgWire(200, body) })
		values, err := collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListLimit(2), consistencygroups.WithRecordListQuery("sort", "name:asc")))
		var proof *resource.ResponseError
		if err == nil || len(values) > 1 || calls.Len() != 1 || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(next, values, err, cgCalls(calls), proof)
		}
	}
}

func TestPythonCinderConsistencyGroupListDictionaryFiltersDefaultsAndShallowEquality(t *testing.T) {
	for _, example := range []struct {
		value, filter string
		matches       bool
		fail          bool
	}{{`"scalar"`, `{}`, true, false}, {`1`, `{}`, true, false}, {`[]`, `{}`, false, false}, {`{}`, `{}`, false, false}, {`{"nested":{}}`, `{"nested":{}}`, true, false}, {`"scalar"`, `{"nested":null}`, false, true}, {`{"key":null}`, `{"missing":null}`, true, false}} {
		body := `{"consistencygroups":[{"id":"id","description":` + example.value + `}]}`
		api, calls := cgAPI(t, "", func(*http.Request) *http.Response { return cgWire(202, body) })
		values, err := collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListOpts(consistencygroups.RecordListOpts{Query: map[string]json.RawMessage{"description": json.RawMessage(example.filter)}})))
		if calls.Len() != 1 || (err != nil) != example.fail || (len(values) == 1) != example.matches {
			t.Fatal(example, values, err, cgCalls(calls))
		}
		if example.fail {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body {
				t.Fatal(err, proof)
			}
		}
	}
	api, calls := cgAPI(t, "", func(*http.Request) *http.Response {
		return cgWire(200, `{"consistencygroups":[{"id":"id","add_volumes":"actual","description":"scalar"}]}`)
	})
	values, err := collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListQuery("add_volumes", "different"), consistencygroups.WithRecordListQuery("description", map[string]any{"key": true})))
	if err != nil || len(values) != 0 || calls.Len() != 1 {
		t.Fatal("first false must short circuit", values, err, cgCalls(calls))
	}
	api, calls = cgAPI(t, "", func(*http.Request) *http.Response { return cgWire(200, `{"consistencygroups":[{"id":"id"}]}`) })
	values, err = collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListQuery("description", nil)))
	if err != nil || len(values) != 1 || calls.Len() != 1 {
		t.Fatal("absent known descriptor defaults null", values, err, cgCalls(calls))
	}
}

func TestPythonCinderConsistencyGroupListExactDecimalPolicyIsExplicit(t *testing.T) {
	api, calls := cgAPI(t, "", func(*http.Request) *http.Response {
		return cgWire(200, `{"consistencygroups":[{"id":"id","description":9007199254740992}]}`)
	})
	values, err := collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListOpts(consistencygroups.RecordListOpts{Query: map[string]json.RawMessage{"description": json.RawMessage(`9007199254740993.0`)}})))
	if err != nil || len(values) != 0 || calls.Len() != 1 {
		t.Fatal("Go exact decimals do not round to binary64", values, err, cgCalls(calls))
	}
	api, calls = cgAPI(t, "", func(*http.Request) *http.Response {
		return cgWire(200, `{"consistencygroups":[{"id":"id","description":1e-400}]}`)
	})
	values, err = collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListQuery("description", map[string]any{})))
	if err != nil || len(values) != 1 || calls.Len() != 1 {
		t.Fatal("nonzero exact decimal remains truthy", values, err, cgCalls(calls))
	}
}

func TestPythonCinderConsistencyGroupListConstructorAndShapesKeepActualPage(t *testing.T) {
	for _, body := range []string{`{"consistencygroups":null}`, `{"consistencygroups":[null]}`, `{"consistencygroups":[{"id":"id","cls":null}]}`, `{"consistencygroups":[{"id":"id","cls":false}]}`, `{"consistencygroups":[{"id":"id","connection":false}]}`, `{"consistencygroups":[{"id":"id","microversion":null}]}`, `{"consistencygroups":[{"id":"id","_synchronized":false}]}`, ""} {
		api, calls := cgAPI(t, "", func(*http.Request) *http.Response { return cgWire(202, body) })
		values, err := collectCG(api.ListRecords(context.Background()))
		var proof *resource.ResponseError
		if len(values) != 0 || calls.Len() != 1 || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(body, values, err, proof, cgCalls(calls))
		}
	}
	api, calls := cgAPI(t, "", func(*http.Request) *http.Response {
		return cgWire(200, `{"consistencygroups":{"id":"singleton","volume_types":false,"unknown":0}}`)
	})
	values, err := collectCG(api.ListRecords(context.Background()))
	if err != nil || len(values) != 1 || calls.Len() != 1 || string(values[0].Wire.Body["unknown"]) != "0" {
		t.Fatal(values, err, cgCalls(calls))
	}
	cgField(t, values[0], "volume_types", `[false]`)
}

func TestPythonCinderConsistencyGroupOptionsMapsLazyProfilesAndSourceOwnership(t *testing.T) {
	query := map[string]json.RawMessage{"name": json.RawMessage(`"target"`)}
	headers := map[string]string{"Accept": "custom", "X-Policy": "owned"}
	option := consistencygroups.WithRecordListOpts(consistencygroups.RecordListOpts{Query: query, Headers: headers})
	query["name"][1] = 'X'
	headers["X-Policy"] = "changed"
	api, calls := cgAPI(t, "", func(*http.Request) *http.Response {
		return cgWire(200, `{"consistencygroups":[{"id":"id","name":"target"}]}`)
	})
	original := map[string]string{"Accept": "source", "X-Source": "source"}
	expected := map[string]string{"Accept": "source", "X-Source": "source"}
	api.RawClient().MoreHeaders = original
	options := []consistencygroups.RecordListOption{option}
	sequence := api.ListRecords(context.Background(), options...)
	options[0] = nil
	if calls.Len() != 0 {
		t.Fatal("list is lazy", cgCalls(calls))
	}
	values, err := collectCG(sequence)
	if err != nil || len(values) != 1 || calls.Len() != 1 || cgCalls(calls)[0].headers.Get("Accept") != "custom" || cgCalls(calls)[0].headers.Get("X-Policy") != "owned" || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) {
		t.Fatal(values, err, cgCalls(calls))
	}
	values, err = collectCG(api.ListRecords(context.Background(), func(c *consistencygroups.RecordListOpts) error {
		if c.Query == nil || c.Headers == nil {
			t.Fatal(c)
		}
		c.Headers["X-First"] = "owned"
		return nil
	}))
	if err != nil || len(values) != 1 || calls.Len() != 2 || cgCalls(calls)[1].headers.Get("Accept") != "application/json" || cgCalls(calls)[1].headers.Get("X-First") != "owned" || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) {
		t.Fatal(values, err, cgCalls(calls))
	}
}

func TestPythonCinderConsistencyGroupPreflightControlsAndJoinedOptionCauses(t *testing.T) {
	for _, invoke := range []func(*consistencygroups.API) error{
		func(a *consistencygroups.API) error { _, e := a.GetRecord(nil, "id"); return e },
		func(a *consistencygroups.API) error {
			_, e := (*consistencygroups.API)(nil).GetRecord(context.Background(), "id")
			return e
		},
		func(a *consistencygroups.API) error { _, e := a.GetRecord(context.Background(), " "); return e },
		func(a *consistencygroups.API) error { _, e := a.GetRecord(context.Background(), "id", nil); return e },
		func(a *consistencygroups.API) error {
			_, e := a.CreateRecord(context.Background(), consistencygroups.WithRecordCreateField("cls", nil))
			return e
		},
		func(a *consistencygroups.API) error {
			_, e := a.UpdateRecord(context.Background(), "id", consistencygroups.WithRecordUpdateField("base_path", false))
			return e
		},
		func(a *consistencygroups.API) error {
			_, e := a.CreateFromSourceRecord(context.Background(), consistencygroups.WithRecordSourceName(string([]byte{0xff})))
			return e
		},
		func(a *consistencygroups.API) error { e := a.DeleteRecord(context.Background(), "id", nil); return e },
		func(a *consistencygroups.API) error {
			_, e := collectCG(a.ListRecords(context.Background(), consistencygroups.WithRecordListQuery("cls", false)))
			return e
		},
		func(a *consistencygroups.API) error {
			_, e := collectCG(a.ListRecords(context.Background(), consistencygroups.WithRecordListQuery("limit", -1)))
			return e
		},
		func(a *consistencygroups.API) error {
			_, e := a.GetRecord(context.Background(), "id", consistencygroups.WithRecordGetHeaders(map[string]string{"OpenStack-API-Version": "volume 3.90"}))
			return e
		},
	} {
		api, calls := cgAPI(t, "", func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return nil })
		err := invoke(api)
		if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
			t.Fatal(err, cgCalls(calls))
		}
	}
	optionCause, cancelCause := errors.New("option cause"), errors.New("custom cancel")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	api, calls := cgAPI(t, "", func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return nil })
	_, err := api.CreateRecord(ctx, func(c *consistencygroups.RecordCreateOpts) error {
		if c.Fields == nil || c.Headers == nil {
			t.Fatal(c)
		}
		api.RawClient().ResourceBase += "changed/"
		cancel(cancelCause)
		return optionCause
	})
	for _, cause := range []error{optionCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
		if !errors.Is(err, cause) {
			t.Fatal(cause, err)
		}
	}
	if calls.Len() != 0 {
		t.Fatal(cgCalls(calls))
	}
}

func TestPythonCinderConsistencyGroupEveryOptionFamilyStartsWithOwnedHeaders(t *testing.T) {
	api, calls := cgAPI(t, "", func(r *http.Request) *http.Response {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/detail") {
			return cgWire(200, `{"consistencygroups":[]}`)
		}
		return cgWire(202, `{"consistencygroup":{"id":"id"}}`)
	})
	source := map[string]string{"X-Source": "original"}
	expected := map[string]string{"X-Source": "original"}
	api.RawClient().MoreHeaders = source
	require := func(headers map[string]string) {
		if headers == nil || len(headers) != 0 {
			t.Fatal("option map must be empty owned map", headers)
		}
		headers["X-First"] = "owned"
	}
	if _, err := api.GetRecord(context.Background(), "id", consistencygroups.WithRecordGetHeaders(nil), func(c *consistencygroups.RecordGetOpts) error { require(c.Headers); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := api.CreateRecord(context.Background(), consistencygroups.WithRecordCreateOpts(consistencygroups.RecordCreateOpts{Headers: nil}), func(c *consistencygroups.RecordCreateOpts) error { require(c.Headers); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := api.UpdateRecord(context.Background(), "id", func(c *consistencygroups.RecordUpdateOpts) error {
		require(c.Headers)
		c.Fields["name"] = json.RawMessage(`"owned"`)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.CreateFromSourceRecord(context.Background(), func(c *consistencygroups.RecordSourceOpts) error { require(c.Headers); return nil }); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteRecord(context.Background(), "id", func(c *consistencygroups.RecordDeleteOpts) error { require(c.Headers); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := collectCG(api.ListRecords(context.Background(), func(c *consistencygroups.RecordListOpts) error { require(c.Headers); return nil })); err != nil {
		t.Fatal(err)
	}
	if _, err := api.FindRecord(context.Background(), "id", func(c *consistencygroups.RecordFindOpts) error { require(c.Headers); return nil }); err != nil {
		t.Fatal(err)
	}
	if calls.Len() != 7 || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) {
		t.Fatal(cgCalls(calls), api.RawClient().MoreHeaders)
	}
	for _, call := range cgCalls(calls) {
		if call.headers.Get("X-First") != "owned" || call.headers.Get("X-Source") != "original" {
			t.Fatal(call)
		}
	}
}

func TestPythonCinderConsistencyGroupPhysicalReceiptsStopHooksAndRetainAllCauses(t *testing.T) {
	for _, operation := range []string{"get", "create", "source", "update", "delete", "list"} {
		for _, code := range []int{202, 400, 401, 403, 404, 503} {
			readCause, closeCause, cancelCause := errors.New("actual read"), errors.New("actual close"), errors.New("custom cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			faulty := &testcloud.Body{Data: `{"partial":`, ReadErr: readCause, CloseErr: closeCause}
			api, calls := cgAPI(t, "", func(*http.Request) *http.Response {
				response := cgWire(code, "")
				response.Body = faulty
				return response
			})
			original := api.RawClient().ResourceBase
			faulty.OnRead = func() { api.RawClient().ResourceBase = original + "changed/" }
			faulty.OnClose = func() { api.RawClient().ResourceBase = original; cancel(cancelCause) }
			retry, reauth := 0, 0
			api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retry++
				return nil
			}
			api.RawClient().ReauthFunc = func(context.Context) error { reauth++; return nil }
			var value *consistencygroups.ConsistencyGroupRecord
			var err error
			switch operation {
			case "get":
				value, err = api.GetRecord(ctx, "id")
			case "create":
				value, err = api.CreateRecord(ctx)
			case "source":
				value, err = api.CreateFromSourceRecord(ctx)
			case "update":
				value, err = api.UpdateRecord(ctx, "id", consistencygroups.WithRecordUpdateField("name", "owned"))
			case "delete":
				err = api.DeleteRecord(ctx, "id")
			case "list":
				_, err = collectCG(api.ListRecords(ctx))
			}
			if calls.Len() != 1 || faulty.Closes() != 1 || retry != 0 || reauth != 0 {
				t.Fatal(operation, code, value, err, cgCalls(calls), faulty.Closes(), retry, reauth)
			}
			for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
				if !errors.Is(err, cause) {
					t.Fatal(operation, code, cause, err)
				}
			}
			if operation != "delete" && operation != "list" && (value == nil || value.StatusCode != code || string(value.Envelope) != faulty.Data || value.Header.Get("X-Request-Id") != "actual" || value.Resource != nil) {
				t.Fatal(operation, code, value, err)
			}
			if code < 400 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != faulty.Data || proof.Header.Get("X-Request-Id") != "actual" {
					t.Fatal(operation, code, err, proof)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				original := err
				if code == 401 {
					var authentication *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &authentication) || !errors.Is(authentication.ErrReauth, readCause) || !errors.Is(authentication.ErrReauth, closeCause) {
						t.Fatal(operation, code, err, authentication)
					}
					original = authentication.ErrOriginal
				}
				if !errors.As(original, &native) || native.Actual != code || string(native.Body) != faulty.Data || native.ResponseHeader.Get("X-Request-Id") != "actual" {
					t.Fatal(operation, code, err, native)
				}
			}
			cancel(nil)
		}
	}
}

func TestPythonCinderConsistencyGroupNativeRetryOwnsPayloadVersionAndFraming(t *testing.T) {
	for _, selected := range []string{"", "3.90"} {
		for _, mode := range []string{"version", "volume_version", "nova", "authorization", "body", "content_type", "omit_content_type", "omit_version"} {
			if selected == "" && mode == "omit_version" {
				continue
			}
			api, calls := cgAPI(t, selected, func(*http.Request) *http.Response { return cgWire(503, `{"retry":"actual"}`) })
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				switch mode {
				case "version":
					opts.MoreHeaders["OpenStack-API-Version"] = "volume 3.99"
				case "volume_version":
					opts.MoreHeaders["X-OpenStack-Volume-API-Version"] = "3.99"
				case "nova":
					opts.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.99"
				case "authorization":
					opts.MoreHeaders["Authorization"] = "forged"
				case "body":
					opts.JSONBody = map[string]any{"consistencygroup": map[string]any{"name": "changed"}}
				case "content_type":
					opts.MoreHeaders["Content-Type"] = "text/plain"
				case "omit_content_type":
					opts.OmitHeaders = append(opts.OmitHeaders, "Content-Type")
				case "omit_version":
					opts.OmitHeaders = append(opts.OmitHeaders, "OpenStack-API-Version")
				}
				return nil
			}
			value, err := api.CreateRecord(context.Background(), consistencygroups.WithRecordCreateField("name", "owned"))
			var native gophercloud.ErrUnexpectedResponseCode
			if value == nil || value.StatusCode != 503 || string(value.Envelope) != `{"retry":"actual"}` || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Request-Id") != "actual" || calls.Len() != 1 {
				t.Fatal(selected, mode, value, err, cgCalls(calls))
			}
			cgJSON(t, cgCalls(calls)[0].body, `{"consistencygroup":{"name":"owned"}}`)
		}
	}
}

func TestPythonCinderConsistencyGroupClean401AndOrdinary503RetainOwnedPOST(t *testing.T) {
	for _, code := range []int{401, 503} {
		attempt := 0
		api, calls := cgAPI(t, "3.90", func(*http.Request) *http.Response {
			attempt++
			if attempt == 1 {
				return cgWire(code, `{"expired":"clean"}`)
			}
			return cgWire(202, `{"consistencygroup":{"id":"new"}}`)
		})
		callbacks := 0
		if code == 401 {
			api.RawClient().ReauthFunc = func(context.Context) error { callbacks++; api.RawClient().SetToken("live"); return nil }
		} else {
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks++
				opts.MoreHeaders["X-Retry"] = "ordinary"
				api.RawClient().SetToken("live")
				return nil
			}
		}
		value, err := api.CreateRecord(context.Background(), consistencygroups.WithRecordCreateField("volume_types", "type-a,type-b"))
		if err != nil || value == nil || calls.Len() != 2 || callbacks != 1 || cgCalls(calls)[1].headers.Get("X-Auth-Token") != "live" || code == 503 && cgCalls(calls)[1].headers.Get("X-Retry") != "ordinary" {
			t.Fatal(code, value, err, cgCalls(calls), callbacks)
		}
		for _, call := range cgCalls(calls) {
			if call.method != http.MethodPost || call.headers.Get("OpenStack-API-Version") != "volume 3.90" {
				t.Fatal(call)
			}
			cgJSON(t, call.body, `{"consistencygroup":{"volume_types":"type-a,type-b"}}`)
		}
	}
}

func TestPythonCinderConsistencyGroupSourceAdmissionAndSelectedAliasesRemainUncapped(t *testing.T) {
	for _, code := range []int{100, 202, 299, 399} {
		api, calls := cgAPI(t, "3.90", func(*http.Request) *http.Response { return cgWire(code, "opaque-ack") })
		source := map[string]string{"OpenStack-API-Version": "volume 3.90", "X-OpenStack-Volume-API-Version": "3.90"}
		expected := map[string]string{"OpenStack-API-Version": "volume 3.90", "X-OpenStack-Volume-API-Version": "3.90"}
		api.RawClient().MoreHeaders = source
		value, err := api.GetRecord(context.Background(), "id", consistencygroups.WithRecordGetMicroversion("3.99"))
		if err != nil || value == nil || value.StatusCode != code || calls.Len() != 1 || cgCalls(calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.99" || api.RawClient().Microversion != "3.90" || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) {
			t.Fatal(code, value, err, cgCalls(calls))
		}
		if err := api.DeleteRecord(context.Background(), "id"); err != nil || calls.Len() != 2 || cgCalls(calls)[1].headers.Get("OpenStack-API-Version") != "volume 3.90" {
			t.Fatal(code, err, cgCalls(calls))
		}
	}
}

func TestPythonCinderConsistencyGroupListNextUsesPythonBinary64TruthinessAndActualPage(t *testing.T) {
	for _, test := range []struct {
		next   string
		truthy bool
	}{
		{`1e-400`, false}, {`-1e-400`, false}, {`-0.0`, false}, {`0`, false}, {`null`, false},
		{`1e-300`, true}, {`1e400`, true}, {`-1e400`, true},
	} {
		t.Run(test.next, func(t *testing.T) {
			body := `{"consistencygroups":[{"id":"first","name":"data"}],"next":` + test.next + `}`
			api, calls := cgAPI(t, "", func(r *http.Request) *http.Response {
				if r.URL.Path != cgBase+"/detail" || r.URL.RawQuery != "" {
					t.Fatal("unexpected continuation", r.URL)
				}
				return cgWire(203, body)
			})
			values, err := collectCG(api.ListRecords(context.Background()))
			if len(values) != 1 || calls.Len() != 1 || (err != nil) != test.truthy {
				t.Fatal(test, values, err, cgCalls(calls))
			}
			cgField(t, values[0], "id", `"first"`)
			if values[0].StatusCode != 203 || string(values[0].Envelope) != body || values[0].Header.Get("X-Request-Id") != "actual" {
				t.Fatal(values[0])
			}
			if test.truthy {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual" {
					t.Fatal(test, err, proof)
				}
			}
		})
	}
}

type cgReservedMarshal func() ([]byte, error)

func (f cgReservedMarshal) MarshalJSON() ([]byte, error) { return f() }

func TestPythonCinderConsistencyGroupListReservedControlsFailBeforeMarshalAndHTTP(t *testing.T) {
	for _, key := range []string{"self", "cls", "session", "headers", "paginated", "max_items", "allow_unknown_params", "details", "jmespath_filters", "__conflicting_attrs", "base_path", "list_base_path", "microversion", "connection", "_synchronized", "resource_type"} {
		for _, input := range []any{nil, false, 0, cgReservedMarshal(func() ([]byte, error) { t.Fatal("reserved control invoked Marshaler", key); return nil, nil })} {
			api, calls := cgAPI(t, "", func(*http.Request) *http.Response { t.Fatal("reserved control reached HTTP", key); return nil })
			option := consistencygroups.WithRecordListQuery(key, input)
			if calls.Len() != 0 {
				t.Fatal("option factory executed HTTP", key)
			}
			values, err := collectCG(api.ListRecords(context.Background(), option))
			if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
				t.Fatal(key, input, values, err, cgCalls(calls))
			}
		}
		for _, raw := range []json.RawMessage{json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`0`), json.RawMessage(`not JSON`)} {
			for _, callback := range []bool{false, true} {
				api, calls := cgAPI(t, "", func(*http.Request) *http.Response { t.Fatal("raw reserved control reached HTTP", key); return nil })
				var option consistencygroups.RecordListOption
				if callback {
					option = func(c *consistencygroups.RecordListOpts) error { c.Query[key] = raw; return nil }
				} else {
					option = consistencygroups.WithRecordListOpts(consistencygroups.RecordListOpts{Query: map[string]json.RawMessage{key: raw}})
				}
				values, err := collectCG(api.ListRecords(context.Background(), option))
				if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
					t.Fatal(key, string(raw), callback, values, err, cgCalls(calls))
				}
			}
		}
	}
	api, calls := cgAPI(t, "", func(*http.Request) *http.Response { return cgWire(200, `{"consistencygroups":[]}`) })
	values, err := collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListQuery("SESSION", cgReservedMarshal(func() ([]byte, error) { t.Fatal("ordinary unknown parameter invoked Marshaler"); return nil, nil }))))
	if err != nil || len(values) != 0 || calls.Len() != 1 || cgCalls(calls)[0].query != "" {
		t.Fatal(values, err, cgCalls(calls))
	}
	api, calls = cgAPI(t, "", func(*http.Request) *http.Response {
		return cgWire(200, `{"consistencygroups":[{"id":"id"}],"next":"?marker=second"}`)
	})
	values, err = collectCG(api.ListRecords(context.Background(), consistencygroups.WithRecordListDetails(false), consistencygroups.WithRecordListPaginated(false), consistencygroups.WithRecordListMaxItems(1)))
	if err != nil || len(values) != 1 || calls.Len() != 1 || cgCalls(calls)[0].path != cgBase || cgCalls(calls)[0].query != "limit=1" {
		t.Fatal("typed controls remain owned functional options", values, err, cgCalls(calls))
	}
}
