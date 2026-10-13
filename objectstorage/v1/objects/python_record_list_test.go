package objects_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonObjectListResult struct {
	records []*objects.ObjectRecord
	err     error
}

func pythonObjectListCollect(api *objects.API, container string, options ...objects.RecordListOption) pythonObjectListResult {
	var result pythonObjectListResult
	for record, err := range api.ListRecords(context.Background(), container, options...) {
		if err != nil {
			result.err = err
			result.records = append(result.records, record)
			return result
		}
		result.records = append(result.records, record)
	}
	return result
}

func pythonObjectListNames(t *testing.T, records []*objects.ObjectRecord) []string {
	t.Helper()
	names := make([]string, 0, len(records))
	for _, record := range records {
		names = append(names, string(record.Resource.Body["name"]))
	}
	return names
}

func pythonObjectListQueries(calls []pythonObjectCall) []string {
	queries := make([]string, 0, len(calls))
	for _, call := range calls {
		queries = append(queries, call.query)
	}
	return queries
}

func pythonObjectListEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestPythonObjectListRecordsDefaultRequestAndProjection(t *testing.T) {
	page := `[{"name":"a.txt","hash":"h1","bytes":5,"content_type":"text/plain","last_modified":"2024-01-01T00:00:00.000000","symlink_path":"/v1/AUTH_p/o/t","container":"other"},` +
		`{"subdir":"dir/"},` +
		`{"name":"m","bytes":"9","X-Static-Large-Object":"yes","if_match":"e1","range":"bytes=0-1","Content-Type":"application/x","metadata":{"Owner":"team"}}]`
	api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
		w.Header().Set("X-Container-Object-Count", "3")
		testcloud.JSON(w, 200, page)
	})
	result := pythonObjectListCollect(api, "box")
	if result.err != nil || len(result.records) != 3 {
		t.Fatal(result.err, len(result.records))
	}
	got := calls()
	if len(got) != 1 {
		t.Fatal(got)
	}
	call := got[0]
	if call.method != "GET" || call.path != "/swift/v1/AUTH_p/box" || call.query != "format=json" || call.body != "" || call.header.Get("Accept") != "application/json" || call.header.Get("X-Auth-Token") != "test-token" || call.header.Get("OpenStack-API-Version") != "" {
		t.Fatalf("%+v", call)
	}
	first := result.records[0]
	want := map[string]string{
		"name": `"a.txt"`, "id": `"a.txt"`, "_hash": `"h1"`, "_bytes": `5`, "_content_type": `"text/plain"`,
		"_last_modified": `"2024-01-01T00:00:00.000000"`, "content_length": `5`, "etag": `"h1"`, "content_type": `"text/plain"`,
		"last_modified_at": `"2024-01-01T00:00:00.000000"`, "updated_at": `"2024-01-01T00:00:00.000000"`,
		"location": `null`, "is_newest": `null`, "symlink_target": `null`, "object_manifest": `null`,
	}
	for key, value := range want {
		if string(first.Resource.Body[key]) != value {
			t.Fatalf("%s=%s want %s", key, first.Resource.Body[key], value)
		}
	}
	if _, exists := first.Resource.Body["symlink_path"]; exists {
		t.Fatal("unknown row member must not enter the Source view")
	}
	if first.Container != "box" || string(first.Wire.Body["symlink_path"]) != `"/v1/AUTH_p/o/t"` || string(first.Wire.Body["container"]) != `"other"` {
		t.Fatalf("%+v %v", first, first.Wire.Body)
	}
	if first.StatusCode != 200 || string(first.Envelope) != page || first.Header.Get("X-Container-Object-Count") != "3" || first.Resource.StatusCode != 200 {
		t.Fatalf("%+v", first)
	}
	subdir := result.records[1]
	if string(subdir.Resource.Body["name"]) != "null" || string(subdir.Resource.Body["id"]) != "null" || string(subdir.Wire.Body["subdir"]) != `"dir/"` || subdir.Container != "box" {
		t.Fatalf("%v %v", subdir.Resource.Body, subdir.Wire.Body)
	}
	typed := result.records[2]
	wantTyped := map[string]string{
		"_bytes": `9`, "content_length": `9`, "is_static_large_object": `true`, "if_match": `["e1"]`,
		"range": `{}`, "content_type": `"application/x"`, "_content_type": `null`, "etag": `null`,
	}
	for key, value := range wantTyped {
		if string(typed.Resource.Body[key]) != value {
			t.Fatalf("%s=%s want %s", key, typed.Resource.Body[key], value)
		}
	}
	if string(typed.Metadata["X-Object-Meta-Owner"]) != `"team"` || len(typed.Metadata) != 1 {
		t.Fatal(typed.Metadata)
	}
}

func TestPythonObjectListRecordsQueryMappingHeadersAndLocalFilters(t *testing.T) {
	rows := `[{"name":"a","bytes":5,"content_type":"text/plain"},{"name":"b","bytes":5,"content_type":"image/png"},{"name":"c","bytes":7,"content_type":"text/plain"}]`
	for _, tc := range []struct {
		label   string
		options []objects.RecordListOption
		query   string
		names   []string
	}{
		{"server mapping and ignored unknown keys", []objects.RecordListOption{
			objects.WithRecordListPrefix("logs/"),
			objects.WithRecordListQuery("multipart-manifest", "put"),
			objects.WithRecordListQuery("multipart_manifest", "get"),
			objects.WithRecordListQuery("symlink", "get"),
			objects.WithRecordListQuery("temp_url_sig", "s"),
			objects.WithRecordListQuery("temp_url_expires", 10),
			objects.WithRecordListQuery("filename", "f"),
			objects.WithRecordListQuery("delimiter", "/"),
			objects.WithRecordListQuery("end_marker", "z"),
			objects.WithRecordListQuery("allow_unknown_params", true),
		}, "filename=f&format=json&multipart-manifest=get&prefix=logs%2F&symlink=get&temp_url_expires=10&temp_url_sig=s", []string{`"a"`, `"b"`, `"c"`}},
		{"bytes filter is local", []objects.RecordListOption{objects.WithRecordListQuery("_bytes", 5)}, "format=json", []string{`"a"`, `"b"`}},
		{"name and content type filters", []objects.RecordListOption{objects.WithRecordListQuery("_content_type", "text/plain"), objects.WithRecordListQuery("name", "c")}, "format=json", []string{`"c"`}},
		{"id filter uses name", []objects.RecordListOption{objects.WithRecordListQuery("id", "b")}, "format=json", []string{`"b"`}},
		{"header attribute names are not filters", []objects.RecordListOption{objects.WithRecordListQuery("content_type", "nope"), objects.WithRecordListQuery("hash", "x")}, "format=json", []string{`"a"`, `"b"`, `"c"`}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
				testcloud.JSON(w, 200, rows)
			})
			result := pythonObjectListCollect(api, "box", tc.options...)
			if result.err != nil {
				t.Fatal(result.err)
			}
			if names := pythonObjectListNames(t, result.records); !pythonObjectListEqual(names, tc.names) {
				t.Fatal(names)
			}
			if got := calls(); len(got) != 1 || got[0].query != tc.query || got[0].path != "/swift/v1/AUTH_p/box" {
				t.Fatal(got)
			}
		})
	}
	t.Run("method headers and microversion", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
			testcloud.JSON(w, 200, `[]`)
		})
		result := pythonObjectListCollect(api, "box", objects.WithRecordListHeaders(map[string]string{"X-Newest": "true", "Accept": "application/json; q=1"}), objects.WithRecordListMicroversion("1.0"))
		if result.err != nil || len(result.records) != 0 {
			t.Fatal(result)
		}
		got := calls()
		if len(got) != 1 || got[0].header.Get("X-Newest") != "true" || got[0].header.Get("Accept") != "application/json; q=1" || got[0].header.Get("OpenStack-API-Version") != "object-store 1.0" {
			t.Fatal(got)
		}
	})
	t.Run("Source keyword collisions fail before HTTP", func(t *testing.T) {
		for _, key := range []string{"container", "paginated", "format", "headers", "microversion", "max_items", "base_path", "jmespath_filters"} {
			api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
				testcloud.JSON(w, 200, `[]`)
			})
			result := pythonObjectListCollect(api, "box", objects.WithRecordListQuery(key, "x"))
			var operation *resource.OperationError
			if !errors.As(result.err, &operation) || operation.Operation != "ListRecords" || operation.Resource != "objects" || !errors.Is(result.err, resource.ErrInvalidOption) || len(calls()) != 0 {
				t.Fatal(key, result.err, calls())
			}
			raw := pythonObjectListCollect(api, "box", objects.WithRecordListOpts(objects.RecordListOpts{Query: map[string]json.RawMessage{key: json.RawMessage(`"x"`)}}))
			if !errors.Is(raw.err, resource.ErrInvalidOption) || len(calls()) != 0 {
				t.Fatal(key, raw.err)
			}
		}
	})
}

func TestPythonObjectListRecordsCountAndMarkerContinuation(t *testing.T) {
	pages := map[string]string{
		"format=json":                   `[{"name":"a"},{"name":"b"}]`,
		"format=json&marker=b":          `[{"name":"c"}]`,
		"format=json&limit=2":           `[{"name":"a"},{"name":"b"}]`,
		"format=json&limit=2&marker=b":  `[{"name":"c"}]`,
		"format=json&limit=2&marker=c":  `[]`,
		"format=json&prefix=p":          `[{"name":"a"},{"name":"b"}]`,
		"format=json&marker=b&prefix=p": `[{"name":"c"}]`,
		"format=json&limit=0":           `[{"name":"a"},{"name":"b"}]`,
	}
	for _, tc := range []struct {
		label   string
		count   string
		options []objects.RecordListOption
		queries []string
	}{
		{"count header follows last name", "3", nil, []string{"format=json", "format=json&marker=b"}},
		{"count equal to consumed rows stops", "2", nil, []string{"format=json"}},
		{"truthy limit fallback until empty page", "", []objects.RecordListOption{objects.WithRecordListLimit(2)}, []string{"format=json&limit=2", "format=json&limit=2&marker=b", "format=json&limit=2&marker=c"}},
		{"prefix is kept on continuation", "3", []objects.RecordListOption{objects.WithRecordListPrefix("p")}, []string{"format=json&prefix=p", "format=json&marker=b&prefix=p"}},
		{"falsey limit is sent once and not restored", "3", []objects.RecordListOption{objects.WithRecordListLimit(0)}, []string{"format=json&limit=0", "format=json&marker=b"}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
				if tc.count != "" {
					w.Header().Set("X-Container-Object-Count", tc.count)
				}
				body, ok := pages[call.query]
				if !ok {
					body = `[]`
				}
				testcloud.JSON(w, 200, body)
			})
			result := pythonObjectListCollect(api, "box", tc.options...)
			if result.err != nil {
				t.Fatal(result.err)
			}
			if queries := pythonObjectListQueries(calls()); !pythonObjectListEqual(queries, tc.queries) {
				t.Fatal(queries)
			}
		})
	}
	t.Run("local filters do not change the consumed count", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
			w.Header().Set("X-Container-Object-Count", "3")
			testcloud.JSON(w, 200, pages[call.query])
		})
		result := pythonObjectListCollect(api, "box", objects.WithRecordListQuery("name", "c"))
		if result.err != nil || !pythonObjectListEqual(pythonObjectListNames(t, result.records), []string{`"c"`}) {
			t.Fatal(result)
		}
		if queries := pythonObjectListQueries(calls()); !pythonObjectListEqual(queries, []string{"format=json", "format=json&marker=b"}) {
			t.Fatal(queries)
		}
	})
	t.Run("RFC Link next precedes count and stays on the container", func(t *testing.T) {
		for _, tc := range []struct {
			link    string
			queries []string
			ok      bool
		}{
			{`</swift/v1/AUTH_p/box?format=json&marker=zz>; rel="next"`, []string{"format=json", "format=json&marker=zz"}, true},
			{`</swift/v1/AUTH_p/other?format=json&marker=zz>; rel="next"`, []string{"format=json"}, false},
		} {
			api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
				w.Header().Set("X-Container-Object-Count", "3")
				if call.query == "format=json" {
					w.Header().Set("Link", tc.link)
					testcloud.JSON(w, 200, `[{"name":"a"},{"name":"b"}]`)
					return
				}
				testcloud.JSON(w, 200, `[{"name":"zz1"}]`)
			})
			result := pythonObjectListCollect(api, "box")
			if (result.err == nil) != tc.ok {
				t.Fatal(tc.link, result.err)
			}
			if queries := pythonObjectListQueries(calls()); !pythonObjectListEqual(queries, tc.queries) {
				t.Fatal(queries)
			}
		}
	})
	t.Run("repeated marker is a pagination cycle", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
			w.Header().Set("X-Container-Object-Count", "10")
			testcloud.JSON(w, 200, `[{"name":"a"},{"name":"b"}]`)
		})
		result := pythonObjectListCollect(api, "box")
		var operation *resource.OperationError
		if !errors.As(result.err, &operation) || operation.Operation != "ListRecords" || !errors.Is(result.err, resource.ErrPaginationCycle) {
			t.Fatal(result.err)
		}
		if queries := pythonObjectListQueries(calls()); !pythonObjectListEqual(queries, []string{"format=json", "format=json&marker=b"}) {
			t.Fatal(queries)
		}
	})
}

func TestPythonObjectListRecordsMaxItemsSinglePageAndPageBound(t *testing.T) {
	rows := `[{"name":"a"},{"name":"b"},{"name":"c"}]`
	reply := func(call pythonObjectCall, w http.ResponseWriter) {
		w.Header().Set("X-Container-Object-Count", "10")
		if call.query == "format=json&marker=c" {
			testcloud.JSON(w, 200, `[{"name":"d"}]`)
			return
		}
		testcloud.JSON(w, 200, rows)
	}
	t.Run("max items fills a falsey limit and stops at the raw cap", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, reply)
		result := pythonObjectListCollect(api, "box", objects.WithRecordListMaxItems(2))
		if result.err != nil || !pythonObjectListEqual(pythonObjectListNames(t, result.records), []string{`"a"`, `"b"`}) {
			t.Fatal(result)
		}
		if queries := pythonObjectListQueries(calls()); !pythonObjectListEqual(queries, []string{"format=json&limit=2"}) {
			t.Fatal(queries)
		}
	})
	t.Run("explicit limit wins over max items", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, reply)
		result := pythonObjectListCollect(api, "box", objects.WithRecordListMaxItems(2), objects.WithRecordListLimit(3))
		if result.err != nil || len(result.records) != 2 {
			t.Fatal(result)
		}
		if queries := pythonObjectListQueries(calls()); !pythonObjectListEqual(queries, []string{"format=json&limit=3"}) {
			t.Fatal(queries)
		}
	})
	t.Run("single page skips continuation", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, reply)
		result := pythonObjectListCollect(api, "box", objects.WithRecordListSinglePage(true))
		if result.err != nil || len(result.records) != 3 || len(calls()) != 1 {
			t.Fatal(result, calls())
		}
	})
	t.Run("page bound fails with the last page receipt", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, reply)
		result := pythonObjectListCollect(api, "box", objects.WithRecordListMaxPages(1))
		last := result.records[len(result.records)-1]
		var response *resource.ResponseError
		if !errors.Is(result.err, resource.ErrInvalidOption) || !errors.As(result.err, &response) || response.StatusCode != 200 || last.StatusCode != 200 || last.Container != "box" || len(calls()) != 1 {
			t.Fatal(result.err, calls())
		}
	})
	t.Run("negative controls fail before HTTP", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, reply)
		for _, option := range []objects.RecordListOption{objects.WithRecordListMaxItems(-1), objects.WithRecordListMaxPages(-1), objects.WithRecordListLimit(-1), nil} {
			result := pythonObjectListCollect(api, "box", option)
			if !errors.Is(result.err, resource.ErrInvalidOption) || result.records[0] != nil {
				t.Fatal(result)
			}
		}
		if len(calls()) != 0 {
			t.Fatal(calls())
		}
	})
}

func TestPythonObjectListRecordsStatusesAndOperationContext(t *testing.T) {
	for _, tc := range []struct {
		label  string
		status int
		body   string
		ok     bool
	}{
		{"accepted json page", 200, `[{"name":"a"}]`, true},
		{"accepted singleton object", 203, `{"name":"a"}`, true},
		{"empty 204 is not JSON", 204, ``, false},
		{"scalar row fails", 200, `["a"]`, false},
		{"constructor collision fails", 200, `[{"name":"a","microversion":"1"}]`, false},
		{"missing container is not suppressed", 404, `<html>missing</html>`, false},
		{"forbidden", 403, `denied`, false},
		{"server error", 500, `boom`, false},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
				if tc.status == 204 {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, tc.status, tc.body)
			})
			result := pythonObjectListCollect(api, "box")
			if len(calls()) != 1 {
				t.Fatal(calls())
			}
			if tc.ok {
				if result.err != nil || len(result.records) != 1 || string(result.records[0].Resource.Body["name"]) != `"a"` || result.records[0].StatusCode != tc.status {
					t.Fatal(result)
				}
				return
			}
			var operation *resource.OperationError
			if !errors.As(result.err, &operation) || operation.Operation != "ListRecords" || operation.Resource != "objects" {
				t.Fatal(result.err)
			}
			receipt := result.records[len(result.records)-1]
			if receipt == nil || receipt.StatusCode != tc.status || receipt.Container != "box" || string(receipt.Envelope) != tc.body || receipt.Resource != nil {
				t.Fatalf("%+v", receipt)
			}
			if tc.status >= 400 && !gophercloud.ResponseCodeIs(result.err, tc.status) {
				t.Fatal(result.err)
			}
		})
	}
	t.Run("invalid container fails before HTTP", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) { testcloud.JSON(w, 200, `[]`) })
		for _, name := range []string{"", "a/b"} {
			result := pythonObjectListCollect(api, name)
			var operation *resource.OperationError
			if !errors.As(result.err, &operation) || operation.Operation != "ListRecords" || !errors.Is(result.err, resource.ErrInvalidOption) {
				t.Fatal(name, result.err)
			}
		}
		if len(calls()) != 0 {
			t.Fatal(calls())
		}
	})
	t.Run("escaped container path", func(t *testing.T) {
		api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) { testcloud.JSON(w, 200, `[]`) })
		if result := pythonObjectListCollect(api, "my box?"); result.err != nil {
			t.Fatal(result.err)
		}
		if got := calls(); len(got) != 1 || got[0].path != "/swift/v1/AUTH_p/my%20box%3F" || got[0].query != "format=json" {
			t.Fatal(got)
		}
	})
}

func TestPythonObjectCloudListObjectsPrefixAndCountPaging(t *testing.T) {
	for _, tc := range []struct {
		label   string
		prefix  string
		count   string
		queries []string
	}{
		{"no prefix single page", "", "2", []string{"format=json"}},
		{"prefix follows count marker", "al", "5", []string{"format=json&prefix=al", "format=json&marker=beta&prefix=al"}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
				w.Header().Set("X-Container-Object-Count", tc.count)
				if call.query == "format=json&marker=beta&prefix=al" {
					testcloud.JSON(w, 200, `[]`)
					return
				}
				testcloud.JSON(w, 200, `[{"name":"alpha","bytes":1,"hash":"x"},{"name":"beta","bytes":0,"hash":"y"}]`)
			})
			var options []objects.RecordListOption
			if tc.prefix != "" {
				options = append(options, objects.WithRecordListPrefix(tc.prefix))
			}
			result := pythonObjectListCollect(api, "box", options...)
			if result.err != nil || !pythonObjectListEqual(pythonObjectListNames(t, result.records), []string{`"alpha"`, `"beta"`}) {
				t.Fatal(result)
			}
			got := calls()
			if !pythonObjectListEqual(pythonObjectListQueries(got), tc.queries) {
				t.Fatal(got)
			}
			for _, call := range got {
				if call.method != "GET" || call.path != "/swift/v1/AUTH_p/box" || call.header.Get("Accept") != "application/json" {
					t.Fatal(call)
				}
			}
		})
	}
}

type pythonObjectListMarshaler struct {
	calls *int
	cause error
}

func (v pythonObjectListMarshaler) MarshalJSON() ([]byte, error) { *v.calls++; return nil, v.cause }

func TestPythonObjectListRecordsReservedControlsRejectBeforeMarshalAndUnknownDTOValuesAreDiscarded(t *testing.T) {
	api, calls := pythonObjectAPI(t, func(_ pythonObjectCall, w http.ResponseWriter) { testcloud.JSON(w, 200, `[]`) })
	marshals := 0
	cause := errors.New("must not marshal unsupported controls")
	for _, key := range []string{"container", "paginated", "format", "headers", "microversion", "max_items", "base_path", "jmespath_filters", "session", "cls", "self", "resource_type", "__conflicting_attrs"} {
		result := pythonObjectListCollect(api, "box", objects.WithRecordListQuery(key, pythonObjectListMarshaler{&marshals, cause}))
		if !errors.Is(result.err, resource.ErrInvalidOption) || errors.Is(result.err, cause) || marshals != 0 || len(calls()) != 0 {
			t.Fatal(key, result.err, marshals, calls())
		}
		result = pythonObjectListCollect(api, "box", objects.WithRecordListOpts(objects.RecordListOpts{Query: map[string]json.RawMessage{key: json.RawMessage(`bad`)}}))
		if !errors.Is(result.err, resource.ErrInvalidOption) || len(calls()) != 0 {
			t.Fatal(key, result.err, calls())
		}
	}
	result := pythonObjectListCollect(api, "box", objects.WithRecordListQuery("delimiter", pythonObjectListMarshaler{&marshals, cause}), objects.WithRecordListOpts(objects.RecordListOpts{Query: map[string]json.RawMessage{"unknown": json.RawMessage(`bad`)}}))
	if result.err != nil || marshals != 0 || len(calls()) != 1 || calls()[0].query != "format=json" {
		t.Fatal(result.err, marshals, calls())
	}
	result = pythonObjectListCollect(api, "box", objects.WithRecordListQuery("name", pythonObjectListMarshaler{&marshals, cause}))
	if !errors.Is(result.err, cause) || marshals != 1 || len(calls()) != 1 {
		t.Fatal(result.err, marshals, calls())
	}
}

func TestPythonObjectListRecordsOwnedOptionsNilHeaderResetAndImmutableProjection(t *testing.T) {
	raw := json.RawMessage(`"owned"`)
	query := map[string]json.RawMessage{"prefix": raw}
	headers := map[string]string{"X-Trace": "owned"}
	version := "1.2"
	option := objects.WithRecordListOpts(objects.RecordListOpts{Query: query, Headers: headers, Microversion: &version})
	raw[1] = 'x'
	query["prefix"] = json.RawMessage(`"changed"`)
	headers["X-Trace"] = "changed"
	version = "changed"
	page := `[{"name":"a","metadata":{"Owner":{"nested":[1,null]},"Numeric":5},"location":{"remote":true},"if_match":"a","IF_MATCH":"ignored","IF-MATCH":["last"]}]`
	api, calls := pythonObjectAPI(t, func(_ pythonObjectCall, w http.ResponseWriter) { testcloud.JSON(w, 200, page) })
	result := pythonObjectListCollect(api, "box", option)
	if result.err != nil || len(result.records) != 1 || calls()[0].query != "format=json&prefix=owned" || calls()[0].header.Get("X-Trace") != "owned" || calls()[0].header.Get("OpenStack-API-Version") != "object-store 1.2" {
		t.Fatal(result.err, calls())
	}
	record := result.records[0]
	if string(record.Metadata["X-Object-Meta-Numeric"]) != `5` || string(record.Metadata["X-Object-Meta-Owner"]) != `{"nested":[1,null]}` || string(record.Resource.Body["location"]) != `null` || string(record.Wire.Body["location"]) != `{"remote":true}` || string(record.Resource.Body["if_match"]) != `["last"]` {
		t.Fatal(record.Resource.Body, record.Wire.Body, record.Metadata)
	}
	record.Resource.Body["if_match"][2] = 'X'
	if string(record.Wire.Body["IF-MATCH"]) != `["last"]` {
		t.Fatal("view aliases wire", record.Wire.Body)
	}
	record.Header.Set("Content-Type", "changed")
	if record.Resource.Header.Get("Content-Type") != "application/json" || record.Wire.Header.Get("Content-Type") != "application/json" {
		t.Fatal("receipt header aliases resource")
	}
	record.Envelope[0] = 'x'
	if string(record.Metadata["X-Object-Meta-Owner"]) != `{"nested":[1,null]}` {
		t.Fatal("metadata aliases envelope")
	}
	result = pythonObjectListCollect(api, "box", func(cfg *objects.RecordListOpts) error { *cfg = objects.RecordListOpts{}; return nil }, objects.WithRecordListHeaders(map[string]string{"X-Trace": "after nil"}))
	if result.err != nil || calls()[1].header.Get("X-Trace") != "after nil" {
		t.Fatal(result.err, calls())
	}
}

func TestPythonObjectListRecordsLocalFiltersUseLoadedFloatsAndDeclaredHeaderTruthiness(t *testing.T) {
	page := `[{"name":"zero","hash":1e-400,"bytes":true,"x-newest":1e-400,"x-static-large-object":-1e-400,"x-detect-content-type":1e400},{"name":"rounded","hash":9007199254740993.0},{"name":"precise","hash":9007199254740993}]`
	for _, tc := range []struct {
		value any
		names []string
	}{
		{false, []string{`"zero"`}}, {json.RawMessage(`9007199254740992`), []string{`"rounded"`}}, {json.RawMessage(`9007199254740993`), []string{`"precise"`}},
	} {
		api, _ := pythonObjectAPI(t, func(_ pythonObjectCall, w http.ResponseWriter) { testcloud.JSON(w, 200, page) })
		result := pythonObjectListCollect(api, "box", objects.WithRecordListQuery("_hash", tc.value))
		if result.err != nil || !pythonObjectListEqual(pythonObjectListNames(t, result.records), tc.names) {
			t.Fatal(result.err, result.records, tc.names)
		}
		if tc.value == false {
			view := result.records[0].Resource.Body
			if string(view["is_newest"]) != "false" || string(view["is_static_large_object"]) != "false" || string(view["is_content_type_detected"]) != "true" || string(view["_bytes"]) != "true" || string(view["content_length"]) != "true" {
				t.Fatal(view)
			}
		}
	}
}

func TestPythonObjectListRecordsPhysicalFaultsPreserveReceiptNative401AndStopHooks(t *testing.T) {
	for _, status := range []int{200, 203, 401, 404, 429, 498, 503} {
		for _, mode := range []string{"read", "close", "both", "source and cancellation"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("read physical body"), errors.New("close physical body"), errors.New("caller custom cancellation")
				payload := `[{"name":"actual"}]`
				var client *gophercloud.ServiceClient
				var body *objectMetadataBody
				reads, requests, retries, reauths, backoffs := 0, 0, 0, 0, 0
				saved := objectMetadataBase
				body = &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) {
					reads++
					if reads > 1 {
						return 0, io.EOF
					}
					n := copy(dst, payload)
					if mode == "source and cancellation" {
						client.Endpoint = saved + "changed/"
					}
					if mode == "read" || mode == "both" {
						return n, readCause
					}
					return n, io.EOF
				})}
				if mode == "close" || mode == "both" {
					body.closeErr = closeCause
				}
				if mode == "source and cancellation" {
					body.onClose = func() { client.Endpoint = saved; cancel(cancelCause) }
				}
				client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					requests++
					if r.Method != "GET" || r.URL.String() != objectMetadataBase+"box?format=json" || r.Body != nil && r.Body != http.NoBody {
						t.Errorf("wrong fixed request %s", r.URL)
					}
					return objectMetadataWire(status, body), nil
				})
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				client.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
				client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoffs++
					return nil
				}
				var record *objects.ObjectRecord
				var err error
				for record, err = range objects.New(client).ListRecords(ctx, "box") {
					if err != nil {
						break
					}
				}
				if err == nil || record == nil || record.Resource != nil || record.Wire != nil || string(record.Envelope) != payload || record.StatusCode != status || record.Header.Get("X-Trans-Id") != "actual-object" || requests != 1 || body.closes.Load() != 1 || retries != 0 || reauths != 0 || backoffs != 0 {
					t.Fatal(err, record, requests, body.closes.Load(), retries, reauths, backoffs)
				}
				if (mode == "read" || mode == "both") && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if (mode == "close" || mode == "both") && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if mode == "source and cancellation" && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
					t.Fatal(err)
				}
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != payload {
					t.Fatal(err, proof)
				}
				if status >= 400 {
					receiptErr := err
					if status == 401 {
						var native *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &native) || native.ErrReauth == nil {
							t.Fatal(err, native)
						}
						if (mode == "read" || mode == "both") && !errors.Is(native.ErrReauth, readCause) {
							t.Fatal("native reauth lost read cause", err, native.ErrReauth)
						}
						if (mode == "close" || mode == "both") && !errors.Is(native.ErrReauth, closeCause) {
							t.Fatal("native reauth lost close cause", err, native.ErrReauth)
						}
						if mode == "source and cancellation" && (!errors.Is(native.ErrReauth, resource.ErrInvalidOption) || !errors.Is(native.ErrReauth, cancelCause) || !errors.Is(native.ErrReauth, context.Canceled)) {
							t.Fatal("native reauth lost guard causes", err, native.ErrReauth)
						}
						receiptErr = native.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(receiptErr, &native) || native.Actual != status || native.Method != "GET" || native.URL != objectMetadataBase+"box?format=json" || string(native.Body) != payload || native.ResponseHeader.Get("X-Trans-Id") != "actual-object" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

func TestPythonObjectListRecordsCleanRetriesPreserveRouteHeadersAndLiveToken(t *testing.T) {
	for _, mode := range []string{"reauth", "retry", "backoff"} {
		t.Run(mode, func(t *testing.T) {
			requests, hooks := 0, 0
			var client *gophercloud.ServiceClient
			client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.Method != "GET" || r.URL.RawQuery != "format=json&prefix=owned" || r.Header.Get("Accept") != "custom/json" || r.Header.Get("If-Match") != "owned" || r.Header.Get("OpenStack-API-Version") != "object-store 1.2" {
					t.Errorf("attempt changed %s %v", r.URL, r.Header)
				}
				if requests == 1 {
					code := 503
					if mode == "reauth" {
						code = 401
					}
					if mode == "backoff" {
						code = 429
					}
					return objectMetadataResponse(code, "initial clean rejection"), nil
				}
				if r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Errorf("stale token %v", r.Header)
				}
				return objectMetadataResponse(203, `[{"name":"result"}]`), nil
			})
			renew := func() { hooks++; client.ProviderClient.SetToken("refreshed") }
			switch mode {
			case "reauth":
				client.ProviderClient.ReauthFunc = func(context.Context) error { renew(); return nil }
			case "retry":
				client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, count uint) error {
					if count > 1 {
						return original
					}
					renew()
					return nil
				}
			case "backoff":
				client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { renew(); return nil }
			}
			result := pythonObjectListCollect(objects.New(client), "box", objects.WithRecordListPrefix("owned"), objects.WithRecordListMicroversion("1.2"), objects.WithRecordListHeaders(map[string]string{"Accept": "custom/json", "If-Match": "owned"}))
			if result.err != nil || requests != 2 || hooks != 1 || len(result.records) != 1 || string(result.records[0].Resource.Body["name"]) != `"result"` {
				t.Fatal(result.err, requests, hooks, result.records)
			}
		})
	}
}

func TestPythonObjectListRecordsIteratorReuseConsumerBreakAndCancellation(t *testing.T) {
	api, calls := pythonObjectAPI(t, func(_ pythonObjectCall, w http.ResponseWriter) {
		w.Header().Set("X-Container-Object-Count", "50")
		testcloud.JSON(w, 200, `[{"name":"a"},{"name":"b"}]`)
	})
	sequence := api.ListRecords(context.Background(), "box")
	for pass := 0; pass < 2; pass++ {
		for record, err := range sequence {
			if err != nil || string(record.Resource.Body["name"]) != `"a"` {
				t.Fatal(record, err)
			}
			break
		}
	}
	if len(calls()) != 2 {
		t.Fatal(calls())
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller cancellation during yield")
	received := 0
	for record, err := range api.ListRecords(ctx, "box") {
		received++
		if received == 1 {
			if err != nil {
				t.Fatal(err)
			}
			cancel(cause)
			continue
		}
		if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || record == nil || record.Resource != nil || record.StatusCode != 200 {
			t.Fatal(record, err)
		}
	}
	if received != 2 || len(calls()) != 3 {
		t.Fatal(received, calls())
	}
}

func TestPythonObjectListRecordsEmptyDictionaryFilterMatchesEveryTruthyType(t *testing.T) {
	page := `[{"name":"string","hash":"keep"},{"name":"list","hash":[1]},{"name":"number","hash":1},{"name":"bool","hash":true},{"name":"mapping","hash":{"x":1}},{"name":"nested-keep","hash":{"nested":"keep"}},{"name":"nested-empty","hash":{"nested":""}},{"name":"empty-string","hash":""},{"name":"empty-list","hash":[]},{"name":"empty-mapping","hash":{}},{"name":"zero","hash":0},{"name":"false","hash":false},{"name":"null","hash":null}]`
	for _, tc := range []struct {
		filter map[string]any
		names  []string
		failed bool
	}{
		{map[string]any{}, []string{`"string"`, `"list"`, `"number"`, `"bool"`, `"mapping"`, `"nested-keep"`, `"nested-empty"`}, false},
		{map[string]any{"nested": map[string]any{}}, nil, true},
	} {
		api, _ := pythonObjectAPI(t, func(_ pythonObjectCall, w http.ResponseWriter) { testcloud.JSON(w, 200, page) })
		result := pythonObjectListCollect(api, "box", objects.WithRecordListQuery("_hash", tc.filter))
		if tc.failed {
			var proof *resource.ResponseError
			if result.err == nil || !errors.As(result.err, &proof) || proof.StatusCode != 200 || string(proof.Body) != page {
				t.Fatal(result.err, proof)
			}
		} else if result.err != nil || !pythonObjectListEqual(pythonObjectListNames(t, result.records), tc.names) {
			t.Fatal(result.err, result.records)
		}
	}
	nested := `[{"name":"keep","hash":{"nested":"keep"}},{"name":"empty","hash":{"nested":""}}]`
	api, _ := pythonObjectAPI(t, func(_ pythonObjectCall, w http.ResponseWriter) { testcloud.JSON(w, 200, nested) })
	result := pythonObjectListCollect(api, "box", objects.WithRecordListQuery("_hash", map[string]any{"nested": map[string]any{}}))
	if result.err != nil || !pythonObjectListEqual(pythonObjectListNames(t, result.records), []string{`"keep"`}) {
		t.Fatal(result.err, result.records)
	}
}
