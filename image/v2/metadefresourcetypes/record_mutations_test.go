package metadefresourcetypes

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
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const rtMutationChild = " child::名字 "

var rtMutationOperations = []struct {
	name     string
	deletion bool
}{{"create", false}, {"delete", true}}

type rtMutationOptions struct {
	create   []RecordCreateOption
	deletion []DeleteOption
}

func rtMutationCall(scope *NamespaceScope, ctx context.Context, deletion bool, input RecordRequest, options rtMutationOptions) (*Record, *Acknowledgement, error) {
	if deletion {
		ack, err := scope.DeleteRecord(ctx, input, options.deletion...)
		return nil, ack, err
	}
	record, err := scope.CreateRecord(ctx, options.create...)
	return record, nil, err
}
func rtMutationDefaults() map[string]string {
	parent, _ := json.Marshal(rtRecordParent)
	return map[string]string{"id": "null", "name": "null", "created_at": "null", "updated_at": "null", "prefix": "null", "properties_target": "null", "namespace_name": string(parent), "location": "null"}
}
func rtMutationValues(record *Record) map[string]string {
	fields := make(map[string]string)
	if record != nil && record.Resource != nil {
		for key, raw := range record.Resource.Body {
			fields[key] = string(raw)
		}
	}
	return fields
}
func rtMutationSeed(t *testing.T, body string) *resource.RawResource {
	t.Helper()
	var seed resource.RawResource
	if err := json.Unmarshal([]byte(body), &seed); err != nil {
		t.Fatal(err)
	}
	return &seed
}

// This asserts requests only; response delivery/faults remain in rtCore helpers.
func rtMutationAssertRequest(t *testing.T, req *http.Request, deletion bool, identity string, want map[string]string) {
	t.Helper()
	method := http.MethodPost
	path := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(rtRecordParent) + "/resource_types"
	if deletion {
		method = http.MethodDelete
		path += "/" + url.PathEscape(identity)
	}
	if req.Method != method || req.URL.EscapedPath() != path || req.URL.RawQuery != "" {
		t.Fatal("fixed association route", req.Method, req.URL)
	}
	if deletion {
		if req.Body != nil {
			t.Fatal("delete invented a request body")
		}
		return
	}
	if req.Body == nil {
		t.Fatal("POST lost empty dictionary body")
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		t.Fatal("flat association body", string(data), err)
	}
	actual := make(map[string]string, len(fields))
	for key, raw := range fields {
		actual[key] = string(raw)
	}
	if !reflect.DeepEqual(actual, want) {
		t.Fatal("raw supplied association fields", actual, want)
	}
}

func TestMetadefResourceTypeRecordCreateSixRawFieldsAndEmptyBody(t *testing.T) {
	for _, check := range []struct{ name, field, raw string }{
		{"untyped id", "id", `{"identity":900719925474099312345}`},
		{"untyped name", "name", `false`}, {"untyped created_at", "created_at", `[1e400,null]`},
		{"untyped updated_at", "updated_at", `900719925474099312345`}, {"untyped prefix", "prefix", `{"nested":"raw"}`},
		{"untyped properties_target", "properties_target", `["target",false]`},
		{"null id", "id", `null`}, {"null name", "name", `null`}, {"null created_at", "created_at", `null`},
		{"null updated_at", "updated_at", `null`}, {"null prefix", "prefix", `null`}, {"null properties_target", "properties_target", `null`},
		{"empty passive name", "name", `""`}, {"unsafe passive name", "name", `"bad/name"`},
		{"overlong passive prefix", "prefix", `"` + strings.Repeat("界", 81) + `"`},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				rtMutationAssertRequest(t, req, false, "", map[string]string{check.field: check.raw})
				return rtCoreJSON(req, 201, `{}`), nil
			})
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			record, err := scope.CreateRecord(context.Background(), WithRecordCreateAttribute(check.field, json.RawMessage(check.raw)))
			want := rtMutationDefaults()
			want[check.field] = check.raw
			if check.field == "name" {
				want["id"] = check.raw
			}
			if err != nil || record == nil || calls != 1 || record.Namespace == nil || *record.Namespace != rtRecordParent || !reflect.DeepEqual(rtMutationValues(record), want) || record.Wire == nil || len(record.Wire.Body) != 0 {
				t.Fatal(record, err, calls, rtMutationValues(record), want)
			}
		})
	}
	for _, mode := range []string{"no attributes", "all unknown failures"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				rtMutationAssertRequest(t, req, false, "", map[string]string{})
				return rtCoreJSON(req, 204, ""), nil
			})
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			var options []RecordCreateOption
			if mode == "all unknown failures" {
				options = []RecordCreateOption{WithRecordCreateAttributes(map[string]any{"future": make(chan int), "value": make(chan int), "location": make(chan int), "schema": make(chan int)})}
			}
			record, err := scope.CreateRecord(context.Background(), options...)
			if err != nil || record == nil || calls != 1 || record.Wire != nil || record.StatusCode != 204 || len(record.Envelope) != 0 || !reflect.DeepEqual(rtMutationValues(record), rtMutationDefaults()) {
				t.Fatal("empty create did not post {}", record, err, calls)
			}
		})
	}
}

func TestMetadefResourceTypeRecordCreateActualStatusesSeedOverlayAndOwnedChannels(t *testing.T) {
	body := `{"id":null,"name":["passive"],"created_at":{"untyped":true},"updated_at":false,"namespace_name":"wire parent","location":{"cloud":"wire"},"unknown":{"number":900719925474099312345},"links":"https://foreign.test/"}`
	for _, code := range []int{200, 201, 202, 203, 204, 299, 300, 304, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				rtMutationAssertRequest(t, req, false, "", map[string]string{"name": `"seed name"`, "prefix": `{"seed":true}`, "properties_target": "null"})
				return rtCoreJSON(req, code, body), nil
			})
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			record, err := scope.CreateRecord(context.Background(), WithRecordCreateAttributes(map[string]any{"name": "seed name", "prefix": json.RawMessage(`{"seed":true}`), "properties_target": nil}))
			want := rtMutationDefaults()
			want["name"] = `["passive"]`
			want["created_at"] = `{"untyped":true}`
			want["updated_at"] = "false"
			want["prefix"] = `{"seed":true}`
			if err != nil || record == nil || record.Wire == nil || record.Namespace == nil || *record.Namespace != rtRecordParent || calls != 1 || !reflect.DeepEqual(rtMutationValues(record), want) {
				t.Fatal(record, err, calls, rtMutationValues(record), want)
			}
			if record.StatusCode != code || record.Resource.StatusCode != code || record.Wire.StatusCode != code || record.Header.Get("X-Proof") != "original" || record.Resource.Header.Get("X-Proof") != "original" || record.Wire.Header.Get("X-Proof") != "original" || string(record.Envelope) != body || string(record.Wire.Body["unknown"]) != `{"number":900719925474099312345}` || record.Resource.CreatedAt != nil || record.Resource.UpdatedAt != nil || record.Resource.Links != nil {
				t.Fatal("actual channels or untyped dates lost", record)
			}
			for _, key := range []string{"unknown", "links", "schema"} {
				if _, present := record.Resource.Body[key]; present {
					t.Fatal("Wire field escaped declared class", key)
				}
			}
			record.Resource.Body["name"][0] = 'X'
			record.Resource.Header.Set("X-Proof", "view changed")
			record.Header.Set("X-Proof", "record changed")
			record.Envelope[0] = 'X'
			*record.Namespace = "caller changed"
			if string(record.Wire.Body["name"]) != `["passive"]` || string(record.Resource.Body["namespace_name"]) != `"OS::空 白"` || record.Wire.Header.Get("X-Proof") != "original" || string(record.Wire.Body["location"]) != `{"cloud":"wire"}` {
				t.Fatal("created result channels alias")
			}
		})
	}
	for _, check := range []struct{ name, body, id string }{
		{"response omits id name alias", `{"name":"server name"}`, `"server name"`},
		{"seed explicit id survives response name", `{"name":"server name"}`, `false`},
		{"last duplicate id wins", `{"id":"first","id":null,"name":"server name"}`, `null`},
	} {
		t.Run(check.name, func(t *testing.T) {
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) { return rtCoreJSON(req, 200, check.body), nil })
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			options := []RecordCreateOption{WithRecordCreateAttribute("name", "seed name")}
			if strings.HasPrefix(check.name, "seed") {
				options = append(options, WithRecordCreateAttribute("id", false))
			}
			record, err := scope.CreateRecord(context.Background(), options...)
			if err != nil || record == nil || string(record.Resource.Body["id"]) != check.id || string(record.Resource.Body["name"]) != `"server name"` {
				t.Fatal("id presence overlay", record, err)
			}
		})
	}
}

func TestMetadefResourceTypeRecordCreateToleratedDecodeAndStrictWholeBody(t *testing.T) {
	for _, check := range []struct {
		name, body string
		tolerated  bool
	}{
		{"empty", "", true}, {"invalid JSON", `not JSON`, true}, {"unfinished object", `{"name":`, true},
		{"null", `null`, false}, {"array", `[]`, false}, {"string", `"scalar"`, false}, {"number", `2`, false}, {"boolean", `false`, false},
		{"invalid UTF8 unknown field", "{\"future\":\"\xff\"}", false},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return rtCoreJSON(req, 202, check.body), nil })
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			record, err := scope.CreateRecord(context.Background(), WithRecordCreateAttribute("prefix", false), WithRecordCreateAttribute("created_at", json.RawMessage(`1e400`)))
			if calls != 1 {
				t.Fatal(calls)
			}
			if check.tolerated {
				if err != nil || record == nil || record.Wire != nil || record.StatusCode != 202 || string(record.Envelope) != check.body || string(record.Resource.Body["prefix"]) != "false" || string(record.Resource.Body["created_at"]) != "1e400" || record.Header.Get("X-Proof") != "original" {
					t.Fatal("tolerated decode lost seeded untyped view", record, err)
				}
			} else {
				if record != nil || err == nil {
					t.Fatal(record, err)
				}
				rtCoreProof(t, err, 202, check.body)
			}
		})
	}
}

func TestMetadefResourceTypeRecordCreateFactoriesSelectionAndLocationSnapshot(t *testing.T) {
	t.Run("factories and retained callback storage", func(t *testing.T) {
		calls, locations := 0, 0
		cloud := "captured"
		location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"original project"`)}}
		client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			rtMutationAssertRequest(t, req, false, "", map[string]string{"prefix": `{"nested":"owned"}`, "properties_target": "false"})
			if req.Header.Get("X-Full") != "owned" || req.Header.Get("X-Bulk") != "factory" || req.Header.Get("X-Final") != "last" || req.Header.Get("X-Replaced") != "" || req.Header.Get("X-Source") != "before" {
				t.Fatal("option header ownership", req.Header)
			}
			return rtCoreJSON(req, 200, `{"location":null,"namespace_name":"wire parent"}`), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "before"}
		_, scope := rtRecordBindings(t, client, true, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return location, nil }})
		headers := map[string]string{"X-Full": "owned"}
		bulk := map[string]string{"X-Bulk": "factory"}
		nested := map[string]any{"nested": "owned"}
		attrs := map[string]any{"prefix": nested}
		slice := []resource.ListOption{resource.WithFilters(attrs)}
		full := WithRecordCreateOpts(RecordCreateOpts{Headers: headers, Attributes: slice})
		bulkOption := WithRecordCreateHeaders(bulk)
		var retained *RecordCreateOpts
		options := []RecordCreateOption{WithRecordCreateHeader("X-Replaced", "old"), full, bulkOption,
			func(config *RecordCreateOpts) error {
				retained = config
				cloud = "option changed"
				location.Project.ID[1] = 'X'
				client.MoreHeaders["X-Source"] = "option changed"
				return nil
			},
			func(*RecordCreateOpts) error {
				retained.Headers["X-Full"] = "retained changed"
				retained.Attributes[0] = resource.WithFilter("prefix", "retained changed")
				return nil
			}, WithRecordCreateHeader("x-final", "last"), WithRecordCreateAttribute("properties_target", false)}
		headers["X-Full"] = "caller changed"
		bulk["X-Bulk"] = "caller changed"
		nested["nested"] = "caller changed"
		attrs["prefix"] = "caller changed"
		slice[0] = resource.WithFilter("prefix", "caller changed")
		record, err := scope.CreateRecord(context.Background(), options...)
		if err != nil || record == nil || calls != 1 || locations != 1 || string(record.Resource.Body["prefix"]) != `{"nested":"owned"}` {
			t.Fatal(record, err, calls, locations)
		}
		var actual resource.CloudLocation
		if err := json.Unmarshal(record.Resource.Body["location"], &actual); err != nil || actual.Cloud == nil || *actual.Cloud != "captured" || string(actual.Project.ID) != `"original project"` || string(record.Wire.Body["location"]) != "null" {
			t.Fatal("location capture did not precede options", actual, err)
		}
	})
	for _, check := range []struct {
		name    string
		options []RecordCreateOption
		body    map[string]string
	}{
		{"bulk replaces failed attribute", []RecordCreateOption{WithRecordCreateAttribute("name", make(chan int)), WithRecordCreateAttributes(map[string]any{"prefix": nil})}, map[string]string{"prefix": "null"}},
		{"singular replaces failed attribute", []RecordCreateOption{WithRecordCreateAttribute("prefix", make(chan int)), WithRecordCreateAttribute("prefix", false)}, map[string]string{"prefix": "false"}},
		{"empty bulk clears body", []RecordCreateOption{WithRecordCreateAttribute("name", make(chan int)), WithRecordCreateAttributes(nil)}, map[string]string{}},
		{"ordinary unknown encoding ignored", []RecordCreateOption{WithRecordCreateAttributes(map[string]any{"name": nil, "future": make(chan int), "createdAt": make(chan int)})}, map[string]string{"name": "null"}},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				rtMutationAssertRequest(t, req, false, "", check.body)
				return rtCoreJSON(req, 200, `{}`), nil
			})
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			for iteration := 0; iteration < 2; iteration++ {
				record, err := scope.CreateRecord(context.Background(), check.options...)
				if err != nil || record == nil || calls != iteration+1 {
					t.Fatal("factory could not be reused", record, err, calls)
				}
			}
		})
	}
}

func TestMetadefResourceTypeRecordDeleteOpaqueStatusesAndFixedIdentitySnapshot(t *testing.T) {
	for _, code := range []int{200, 201, 202, 203, 204, 299, 300, 304, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, locations := 0, 0
			body := "opaque response\xff\x00"
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				rtMutationAssertRequest(t, req, true, rtMutationChild, nil)
				if req.Header.Get("X-Option") != "owned" {
					t.Fatal(req.Header)
				}
				return rtCoreJSON(req, code, body), nil
			})
			_, scope := rtRecordBindings(t, client, true, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				return resource.CloudLocation{}, errors.New("not needed for opaque deletion")
			}})
			ack, err := scope.DeleteRecord(context.Background(), RecordRequest{ID: rtMutationChild}, WithDeleteHeader("X-Option", "owned"))
			if err != nil || ack == nil || ack.Namespace != rtRecordParent || ack.Name == nil || *ack.Name != rtMutationChild || ack.StatusCode != code || string(ack.Body) != body || ack.Header.Get("X-Proof") != "original" || calls != 1 || locations != 0 {
				t.Fatal("opaque deletion lost actual evidence or invoked location", ack, err, calls, locations)
			}
		})
	}
	for _, check := range []struct{ name, seed, identity string }{
		{"explicit id", "{\"id\":\"selected-id\",\"name\":\"alias\"}", "selected-id"},
		{"name fallback", `{"name":"selected-name"}`, "selected-name"},
		{"ignore unrelated fields", `{"id":"selected-id","name":false,"prefix":{"raw":true},"created_at":[1e400],"namespace_name":"foreign"}`, "selected-id"},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			seed := rtMutationSeed(t, check.seed)
			seed.Body["unused"] = json.RawMessage(`not JSON`)
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				rtMutationAssertRequest(t, req, true, check.identity, nil)
				return rtCoreJSON(req, 202, `{"name":"response name","namespace_name":"response parent"}`), nil
			})
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			ack, err := scope.DeleteRecord(context.Background(), RecordRequest{Resource: seed}, func(*DeleteOpts) error {
				seed.Body["id"] = json.RawMessage(`"caller changed"`)
				seed.Body["name"] = json.RawMessage(`"caller changed"`)
				return nil
			})
			if err != nil || ack == nil || ack.Name == nil || *ack.Name != check.identity || ack.Namespace != rtRecordParent || calls != 1 {
				t.Fatal("input mutation or response retargeted deletion", ack, err, calls)
			}
		})
	}
}

func TestMetadefResourceTypeRecordDeletePhysicalMissingAndOptionOwnership(t *testing.T) {
	for _, mode := range []string{"default missing", "explicit handled missing", "missing Close error", "strict physical missing", "transport named404"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			closeErr := errors.New("close missing")
			stop := errors.New("native retry stop")
			bodyText := "missing opaque\xff"
			body := &rtCoreBody{reader: strings.NewReader(bodyText)}
			if mode == "missing Close error" {
				body.closeErr = closeErr
			}
			transport404 := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("transport claim")}
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if mode == "transport named404" {
					return nil, transport404
				}
				return rtCoreHTTP(req, 404, body), nil
			})
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return stop
			}
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			var options []DeleteOption
			if mode == "strict physical missing" {
				options = []DeleteOption{WithDeleteIgnoreMissing(false)}
			}
			if mode == "explicit handled missing" {
				options = []DeleteOption{WithDeleteIgnoreMissing(true)}
			}
			ack, err := scope.DeleteRecord(context.Background(), RecordRequest{ID: rtMutationChild}, options...)
			if calls != 1 {
				t.Fatal(calls)
			}
			switch mode {
			case "default missing", "explicit handled missing", "missing Close error":
				if ack == nil || ack.StatusCode != 404 || ack.Name == nil || *ack.Name != rtMutationChild || ack.Namespace != rtRecordParent || string(ack.Body) != bodyText || body.closes != 1 || retries != 0 {
					t.Fatal("handled physical absence was discarded or retried", ack, err, body.closes, retries)
				}
				if mode == "missing Close error" {
					if !errors.Is(err, closeErr) {
						t.Fatal(err)
					}
					rtCoreProof(t, err, 404, bodyText)
				} else if err != nil {
					t.Fatal(err)
				}
			case "transport named404":
				if ack != nil || !errors.Is(err, transport404) || !errors.Is(err, stop) || retries != 1 || body.closes != 0 {
					t.Fatal("transport claim became handled physical404", ack, err, retries, body.closes)
				}
			default:
				var native gophercloud.ErrUnexpectedResponseCode
				if ack != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != bodyText || native.ResponseHeader.Get("X-Proof") != "original" || !errors.Is(err, stop) || retries != 1 || body.closes != 1 {
					t.Fatal("strict physical missing lost native evidence", ack, err, native, retries, body.closes)
				}
			}
		})
	}
	t.Run("retained full options cannot change strict missing", func(t *testing.T) {
		calls := 0
		client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			rtMutationAssertRequest(t, req, true, rtMutationChild, nil)
			if req.Header.Get("X-Full") != "owned" || req.Header.Get("X-Bulk") != "factory" || req.Header.Get("X-Replaced") != "" || req.Header.Get("X-Final") != "last" || req.Header.Get("X-Source") != "before" {
				t.Fatal(req.Header)
			}
			return rtCoreJSON(req, 404, "missing"), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "before"}
		_, scope := rtRecordBindings(t, client, true, Dependencies{})
		ignore := false
		headers := map[string]string{"X-Full": "owned"}
		bulk := map[string]string{"X-Bulk": "factory"}
		full := WithDeleteOpts(DeleteOpts{Headers: headers, IgnoreMissing: &ignore})
		bulkOption := WithDeleteHeaders(bulk)
		var retained *DeleteOpts
		options := []DeleteOption{WithDeleteHeader("X-Replaced", "old"), full, bulkOption, func(config *DeleteOpts) error {
			retained = config
			client.MoreHeaders["X-Source"] = "option changed"
			return nil
		}, func(*DeleteOpts) error {
			retained.Headers["X-Full"] = "retained changed"
			*retained.IgnoreMissing = true
			return nil
		}, WithDeleteHeader("x-final", "last")}
		ignore = true
		headers["X-Full"] = "caller changed"
		bulk["X-Bulk"] = "caller changed"
		ack, err := scope.DeleteRecord(context.Background(), RecordRequest{ID: rtMutationChild}, options...)
		if ack != nil || !gophercloud.ResponseCodeIs(err, 404) || calls != 1 {
			t.Fatal("retained or caller pointers changed missing policy", ack, err, calls)
		}
	})
}

func TestMetadefResourceTypeRecordMutationCompletePreflight(t *testing.T) {
	for _, check := range []struct {
		name    string
		options []RecordCreateOption
	}{
		{"nil option", []RecordCreateOption{nil}},
		{"known encoding failure", []RecordCreateOption{WithRecordCreateAttribute("prefix", make(chan int))}},
		{"nil semantic option", []RecordCreateOption{WithRecordCreateOpts(RecordCreateOpts{Attributes: []resource.ListOption{nil}})}},
		{"nonsemantic carrier", []RecordCreateOption{WithRecordCreateOpts(RecordCreateOpts{Attributes: []resource.ListOption{resource.WithQuery("name", "foreign")}})}},
		{"protected token", []RecordCreateOption{WithRecordCreateHeader("X-Auth-Token", "foreign")}},
		{"Accept owned", []RecordCreateOption{WithRecordCreateHeader("Accept", "text/plain")}},
		{"header aliases", []RecordCreateOption{WithRecordCreateHeaders(map[string]string{"X-Alias": "one", "x-alias": "two"})}},
		{"header newline", []RecordCreateOption{WithRecordCreateHeader("X-Option", "bad\nvalue")}},
	} {
		t.Run("create/"+check.name, func(t *testing.T) {
			calls := 0
			client := rtCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			record, err := scope.CreateRecord(context.Background(), check.options...)
			if record != nil || err == nil || calls != 0 {
				t.Fatal(record, err, calls)
			}
		})
	}
	for _, key := range []string{"resource_type", "namespace_name", "namespace", "base_path", "requires_id", "session", "microversion", "headers", "connection", "_synchronized", "__conflicting_attrs", "resource_request_key", "resource_response_key", "resource_type_class", "prepend_key", "has_body", "retry_on_conflict"} {
		t.Run("create reserved "+key, func(t *testing.T) {
			calls := 0
			client := rtCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			record, err := scope.CreateRecord(context.Background(), WithRecordCreateAttribute(key, make(chan int)))
			if record != nil || err == nil || calls != 0 {
				t.Fatal("reserved binding entered POST", key, record, err, calls)
			}
		})
	}
	for _, check := range []struct {
		name  string
		input RecordRequest
	}{
		{"missing", RecordRequest{}}, {"both forms", RecordRequest{ID: rtMutationChild, Resource: &resource.RawResource{}}}, {"empty Resource", RecordRequest{Resource: &resource.RawResource{}}},
		{"null explicit id authoritative", RecordRequest{Resource: rtMutationSeed(t, `{"id":null,"name":"safe"}`)}},
		{"empty explicit id authoritative", RecordRequest{Resource: rtMutationSeed(t, `{"id":"","name":"safe"}`)}},
		{"nonstring id", RecordRequest{Resource: rtMutationSeed(t, `{"id":false,"name":"safe"}`)}},
		{"unsafe id", RecordRequest{ID: "bad/name"}}, {"overlong id", RecordRequest{ID: strings.Repeat("界", 81)}},
		{"invalid UTF8 raw id", RecordRequest{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": {'"', 0xff, '"'}, "name": json.RawMessage(`"safe"`)}}}}},
	} {
		t.Run("delete/"+check.name, func(t *testing.T) {
			calls, callbacks := 0, 0
			client := rtCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			ack, err := scope.DeleteRecord(context.Background(), check.input, func(*DeleteOpts) error { callbacks++; return nil })
			if ack != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
				t.Fatal("identity preflight invoked callbacks or HTTP", ack, err, calls, callbacks)
			}
		})
	}
}

func TestMetadefResourceTypeRecordMutationStickySourceAndOuterPreflight(t *testing.T) {
	for _, operation := range rtMutationOperations {
		for _, mode := range []string{"nil context", "canceled context", "source lifetime", "outer guard", "callback error", "callback source change", "callback namespace change"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				calls, locations, first, second := 0, 0, 0, 0
				cause := errors.New("guard cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var use context.Context = ctx
				client := rtCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
				_, scope := rtRecordBindings(t, client, true, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{}, nil }})
				switch mode {
				case "nil context":
					use = nil
				case "canceled context":
					cancel(cause)
				case "source lifetime":
					client.Endpoint = "https://changed.test/"
				case "outer guard":
					use = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
				}
				mutate := func() error {
					first++
					if mode == "callback error" {
						return cause
					}
					if mode == "callback source change" {
						client.ResourceBase += "changed/"
					}
					if mode == "callback namespace change" {
						scope.namespace = "changed"
					}
					return nil
				}
				restore := func() error {
					second++
					client.ResourceBase = "https://example.test/reverse/glance/v2/"
					scope.namespace = rtRecordParent
					return nil
				}
				options := rtMutationOptions{create: []RecordCreateOption{func(*RecordCreateOpts) error { return mutate() }, func(*RecordCreateOpts) error { return restore() }}, deletion: []DeleteOption{func(*DeleteOpts) error { return mutate() }, func(*DeleteOpts) error { return restore() }}}
				record, ack, err := rtMutationCall(scope, use, operation.deletion, RecordRequest{ID: rtMutationChild}, options)
				if record != nil || ack != nil || err == nil || calls != 0 || second != 0 {
					t.Fatal("failure restored before HTTP", record, ack, err, calls, locations, first, second)
				}
				if strings.HasPrefix(mode, "callback") {
					if first != 1 {
						t.Fatal(first)
					}
					wantLocations := 1
					if operation.deletion {
						wantLocations = 0
					}
					if locations != wantLocations {
						t.Fatal("wrong location orchestration", locations)
					}
				} else if first != 0 || locations != 0 {
					t.Fatal("source preflight invoked location/options", first, locations)
				}
				if mode == "canceled context" || mode == "outer guard" || mode == "callback error" {
					if !errors.Is(err, cause) {
						t.Fatal("preflight cause lost", err)
					}
				}
			})
		}
	}
	for _, mode := range []string{"location error", "invalid location", "location changed source"} {
		t.Run("create/"+mode, func(t *testing.T) {
			calls, callbacks, locations := 0, 0, 0
			cause := errors.New("location cause")
			client := rtCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			_, scope := rtRecordBindings(t, client, true, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				if mode == "location error" {
					return resource.CloudLocation{}, cause
				}
				if mode == "invalid location" {
					return resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`{`)}}, nil
				}
				client.Endpoint = "https://changed.test/"
				return resource.CloudLocation{}, nil
			}})
			record, err := scope.CreateRecord(context.Background(), func(*RecordCreateOpts) error { callbacks++; return nil })
			if record != nil || err == nil || calls != 0 || callbacks != 0 || locations != 1 {
				t.Fatal(record, err, calls, callbacks, locations)
			}
			if mode == "location error" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}

func TestMetadefResourceTypeRecordMutationAcceptedFaultReceiptsAndFrozenTargets(t *testing.T) {
	for _, operation := range rtMutationOperations {
		for _, mode := range []string{"read close cancel", "scope Close mutation", "observed source restored", "observed outer restored"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				calls, retries := 0, 0
				readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("cancel")
				outerCause := errors.New("outer changed")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				outerFailed := false
				if mode == "observed outer restored" {
					ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
						if outerFailed {
							return outerCause
						}
						return nil
					})
				}
				bodyText := `{"name":"response"}`
				if operation.deletion {
					bodyText = "opaque ack\xff"
				}
				var scope *NamespaceScope
				var client *gophercloud.ServiceClient
				var body *rtCoreBody
				client = rtCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					body = &rtCoreBody{reader: strings.NewReader(bodyText)}
					if mode == "read close cancel" {
						body.reader = &rtCoreReader{data: bodyText, err: readErr, after: func() { cancel(cause) }}
						body.closeErr = closeErr
					}
					if mode == "scope Close mutation" {
						return rtCoreHTTP(req, 203, &rtRecordCloseBody{rtCoreBody: body, after: func() { scope.namespace = "changed parent" }}), nil
					}
					if strings.HasPrefix(mode, "observed") {
						body.reader = &rtCoreReader{data: bodyText, err: io.EOF, after: func() {
							if mode == "observed outer restored" {
								outerFailed = true
							} else {
								client.Microversion = "2.3"
							}
							if err := rest.CheckOperationGuard(req.Context()); err == nil {
								t.Error("change was not observed")
							}
						}}
						return rtCoreHTTP(req, 203, &rtRecordCloseBody{rtCoreBody: body, after: func() { outerFailed = false; client.Microversion = "" }}), nil
					}
					return rtCoreHTTP(req, 203, body), nil
				})
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				_, scope = rtRecordBindings(t, client, true, Dependencies{})
				record, ack, err := rtMutationCall(scope, ctx, operation.deletion, RecordRequest{ID: rtMutationChild}, rtMutationOptions{create: []RecordCreateOption{WithRecordCreateAttribute("prefix", "seed")}})
				if record != nil || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
					t.Fatal(record, ack, err, calls, retries, body.closes)
				}
				proof := rtCoreProof(t, err, 203, bodyText)
				if operation.deletion {
					if ack == nil || ack.Namespace != rtRecordParent || ack.Name == nil || *ack.Name != rtMutationChild || ack.StatusCode != 203 || string(ack.Body) != bodyText || ack.Header.Get("X-Proof") != "original" {
						t.Fatal("opaque fault lost frozen ACK", ack, err)
					}
					ack.Body[0] = 'X'
					ack.Header.Set("X-Proof", "caller changed")
					*ack.Name = "caller changed"
					if string(proof.Body) != bodyText || proof.Header.Get("X-Proof") != "original" {
						t.Fatal("ACK aliases error proof")
					}
				} else if ack != nil {
					t.Fatal("create fabricated ACK")
				}
				if mode == "read close cancel" {
					for _, expected := range []error{readErr, closeErr, cause, context.Canceled} {
						if !errors.Is(err, expected) {
							t.Fatal("accepted cause lost", expected, err)
						}
					}
				} else if mode == "observed outer restored" {
					if !errors.Is(err, outerCause) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestMetadefResourceTypeRecordMutationNativeRetryOwnershipAndLegacyPolicies(t *testing.T) {
	for _, operation := range rtMutationOperations {
		for _, mode := range []string{"equivalent retry body", "changed retry body", "expanded native status"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				calls, retries := 0, 0
				var physical []*rtCoreBody
				client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					rtMutationAssertRequest(t, req, operation.deletion, rtMutationChild, map[string]string{"prefix": `"owned"`})
					code, text := 503, "physical failure"
					if calls > 1 {
						code, text = 203, `{"name":"response"}`
						if mode == "expanded native status" {
							code, text = 400, "outside SDK policy"
						}
					}
					body := &rtCoreBody{reader: strings.NewReader(text)}
					physical = append(physical, body)
					return rtCoreHTTP(req, code, body), nil
				})
				callbackCause := errors.New("callback rejected")
				client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
					retries++
					if retries > 1 {
						return errors.New("unexpected extra retry")
					}
					switch mode {
					case "equivalent retry body":
						if !operation.deletion {
							options.JSONBody = json.RawMessage(bytes.Clone(options.JSONBody.(json.RawMessage)))
						}
					case "changed retry body":
						if operation.deletion {
							options.JSONBody = map[string]any{"foreign": true}
						} else {
							raw := options.JSONBody.(json.RawMessage)
							raw[11] = 'X'
						}
						return callbackCause
					case "expanded native status":
						options.OkCodes = append(options.OkCodes, 400)
					}
					return nil
				}
				_, scope := rtRecordBindings(t, client, true, Dependencies{})
				record, ack, err := rtMutationCall(scope, context.Background(), operation.deletion, RecordRequest{ID: rtMutationChild}, rtMutationOptions{create: []RecordCreateOption{WithRecordCreateAttribute("prefix", "owned")}})
				if mode == "equivalent retry body" {
					if err != nil || calls != 2 || retries != 1 {
						t.Fatal(record, ack, err, calls, retries)
					}
					if operation.deletion {
						if ack == nil || ack.StatusCode != 203 {
							t.Fatal(ack)
						}
					} else if record == nil || record.StatusCode != 203 {
						t.Fatal(record)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if record != nil || ack != nil || !errors.As(err, &native) || native.ResponseHeader.Get("X-Proof") != "original" || retries != 1 {
						t.Fatal("native failure proof lost", record, ack, err, native, calls, retries)
					}
					if mode == "changed retry body" {
						if calls != 1 || native.Actual != 503 || string(native.Body) != "physical failure" || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) {
							t.Fatal("owned request policy lost", err, native, calls)
						}
					} else if calls != 2 || native.Actual != 400 || string(native.Body) != "outside SDK policy" {
						t.Fatal("native status expansion became SDK success", err, native, calls)
					}
					var proof *resource.ResponseError
					if errors.As(err, &proof) {
						t.Fatal("native failure acquired successful response receipt", proof)
					}
				}
				for _, body := range physical {
					if body.closes != 1 {
						t.Fatal("physical close count", body.closes)
					}
				}
			})
		}
	}
	for _, mode := range []string{"legacy create strict201", "legacy create typed response", "legacy create literal name", "legacy delete strict204", "legacy clean missing"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				switch mode {
				case "legacy create strict201":
					return rtCoreJSON(req, 200, `{}`), nil
				case "legacy create typed response":
					return rtCoreJSON(req, 201, `{"name":false}`), nil
				case "legacy delete strict204":
					return rtCoreJSON(req, 202, "opaque"), nil
				default:
					return rtCoreJSON(req, 404, "missing"), nil
				}
			})
			_, scope := rtRecordBindings(t, client, true, Dependencies{})
			if strings.HasPrefix(mode, "legacy create") {
				name := rtMutationChild
				if mode == "legacy create literal name" {
					name = "bad/name"
				}
				value, err := scope.Create(context.Background(), name)
				if value != nil || err == nil {
					t.Fatal("bounded create policy changed", value, err)
				}
				if mode == "legacy create literal name" {
					if calls != 0 || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err, calls)
					}
				} else if mode == "legacy create typed response" {
					if calls != 1 {
						t.Fatal(calls)
					}
					rtCoreProof(t, err, 201, `{"name":false}`)
				} else if calls != 1 || !gophercloud.ResponseCodeIs(err, 200) {
					t.Fatal(err, calls)
				}
			} else {
				ack, err := scope.Delete(context.Background(), rtMutationChild)
				if mode == "legacy clean missing" {
					if ack != nil || err != nil || calls != 1 {
						t.Fatal("bounded missing acquired ACK", ack, err, calls)
					}
				} else if ack != nil || !gophercloud.ResponseCodeIs(err, 202) || calls != 1 {
					t.Fatal("bounded delete expanded status policy", ack, err, calls)
				}
			}
		})
	}
}
