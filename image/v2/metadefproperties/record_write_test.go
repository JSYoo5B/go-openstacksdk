package metadefproperties

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

var propertyRecordWrites = []struct {
	name   string
	update bool
}{{"create", false}, {"update", true}}

type propertyRecordWriteOptions struct {
	create []RecordCreateOption
	update []RecordUpdateOption
}

func propertyRecordWriteCall(scope *NamespaceScope, ctx context.Context, update bool, input RecordRequest, options propertyRecordWriteOptions) (*Record, error) {
	if update {
		return scope.UpdateRecord(ctx, input, options.update...)
	}
	return scope.CreateRecord(ctx, options.create...)
}
func propertyRecordWriteAttributes(fields map[string]any) propertyRecordWriteOptions {
	return propertyRecordWriteOptions{create: []RecordCreateOption{WithRecordCreateAttributes(fields)}, update: []RecordUpdateOption{WithRecordUpdateAttributes(fields)}}
}
func propertyRecordWriteDefaults(update bool) map[string]string {
	result := propertyRecordDefaults(propertyRecordChild)
	if !update {
		result["id"] = "null"
	}
	return result
}

// Only a request assertion: all response delivery/faults use propertyCore.
func propertyRecordAssertWriteBody(t *testing.T, req *http.Request, update bool, identity string, want map[string]string) {
	t.Helper()
	method := http.MethodPost
	path := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(propertyRecordParent) + "/properties"
	if update {
		method = http.MethodPut
		path += "/" + url.PathEscape(identity)
	}
	if req.Method != method || req.URL.EscapedPath() != path || req.URL.RawQuery != "" || req.Body == nil {
		t.Fatal("fixed JSON write", req.Method, req.URL, req.Body)
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		t.Fatal("flat dictionary body", string(data), err)
	}
	actual := make(map[string]string, len(fields))
	for key, raw := range fields {
		actual[key] = string(raw)
	}
	if !reflect.DeepEqual(actual, want) {
		t.Fatal("raw dirty wire body", actual, want)
	}
}

func TestMetadefPropertyRecordWritesKeepAllRawDirtyFieldsSeparateFromView(t *testing.T) {
	fields := []struct{ canonical, wire, raw, projected string }{
		{"id", "id", `{"passive":true}`, `{"passive":true}`},
		{"name", "name", `{"passive":"bad/name"}`, `{"passive":"bad/name"}`},
		{"type", "type", `["arbitrary",false]`, `["arbitrary",false]`},
		{"title", "title", `900719925474099312345`, `900719925474099312345`},
		{"description", "description", `null`, `null`},
		{"operators", "operators", `"scalar operator"`, `["scalar operator"]`},
		{"default", "default", `{"large":900719925474099312345}`, `{"large":900719925474099312345}`},
		{"is_readonly", "readonly", `"false"`, `true`},
		{"minimum", "minimum", `"-2"`, `0`},
		{"maximum", "maximum", `2.9`, `2`},
		{"enum", "enum", `true`, `[true]`},
		{"pattern", "pattern", `false`, `false`},
		{"min_length", "minLength", `"04"`, `4`},
		{"max_length", "maxLength", `null`, `null`},
		{"items", "items", `[1,2]`, `{}`},
		{"require_unique_items", "uniqueItems", `0.0`, `false`},
		{"min_items", "minItems", `0`, `0`},
		{"max_items", "maxItems", `900719925474099312345`, `900719925474099312345`},
		{"allow_additional_items", "additionalItems", `""`, `false`},
	}
	for _, operation := range propertyRecordWrites {
		for _, field := range fields {
			if operation.update && field.canonical == "id" {
				continue
			}
			t.Run(operation.name+"/"+field.canonical, func(t *testing.T) {
				calls := 0
				client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					propertyRecordAssertWriteBody(t, req, operation.update, propertyRecordChild, map[string]string{field.wire: field.raw})
					return propertyCoreJSON(req, 203, `{}`), nil
				})
				record, err := propertyRecordWriteCall(propertyRecordScope(t, client, Dependencies{}), context.Background(), operation.update, RecordRequest{ID: propertyRecordChild}, propertyRecordWriteAttributes(map[string]any{field.canonical: json.RawMessage(field.raw)}))
				want := propertyRecordWriteDefaults(operation.update)
				want[field.canonical] = field.projected
				if !operation.update && field.canonical == "name" {
					want["id"] = field.projected
				}
				if err != nil || record == nil || calls != 1 || record.Key != nil || record.Namespace != propertyRecordParent || record.StatusCode != 203 || !reflect.DeepEqual(propertyRecordValues(record), want) {
					t.Fatal(record, err, calls, propertyRecordValues(record), want)
				}
				if record.Wire == nil || len(record.Wire.Body) != 0 || string(record.Envelope) != `{}` {
					t.Fatal("server Wire was seeded", record)
				}
			})
		}
	}
}

func TestMetadefPropertyRecordWriteEmptyConstructionAndIdentityOnlyUpdate(t *testing.T) {
	for _, check := range []struct {
		name   string
		update bool
		input  RecordRequest
		attrs  map[string]any
		body   map[string]string
		want   map[string]string
		local  bool
	}{
		{"empty create", false, RecordRequest{}, nil, map[string]string{}, nil, false},
		{"unknown create still posts", false, RecordRequest{}, map[string]any{"future": make(chan int), "value": make(chan int), "location": make(chan int)}, map[string]string{}, nil, false},
		{"empty create name passive", false, RecordRequest{}, map[string]any{"name": ""}, map[string]string{"name": `""`}, map[string]string{"name": `""`, "id": `""`}, false},
		{"unsafe create name passive", false, RecordRequest{}, map[string]any{"name": "bad/name"}, map[string]string{"name": `"bad/name"`}, map[string]string{"name": `"bad/name"`, "id": `"bad/name"`}, false},
		{"long create name passive", false, RecordRequest{}, map[string]any{"name": strings.Repeat("x", 81)}, map[string]string{"name": `"` + strings.Repeat("x", 81) + `"`}, map[string]string{"name": `"` + strings.Repeat("x", 81) + `"`, "id": `"` + strings.Repeat("x", 81) + `"`}, false},
		{"no update attributes", true, RecordRequest{ID: propertyRecordChild}, nil, nil, nil, true},
		{"ignored update fields no PUT", true, RecordRequest{ID: propertyRecordChild}, map[string]any{"schema": make(chan int), "created_at": make(chan int), "location": make(chan int)}, nil, nil, true},
		{"name-only input no PUT", true, RecordRequest{Resource: propertyRecordSeed(t, `{"name":" child::名字 ","title":"old title","minLength":"²"}`)}, nil, nil, nil, true},
		{"malformed nonidentity input ignored", true, RecordRequest{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`" child::名字 "`), "minimum": json.RawMessage(`bad JSON`), "name": json.RawMessage(`null`)}}}}, map[string]any{"title": nil}, map[string]string{"title": "null"}, nil, false},
		{"name equal to identity still dirty", true, RecordRequest{ID: propertyRecordChild}, map[string]any{"name": propertyRecordChild}, map[string]string{"name": `" child::名字 "`}, map[string]string{"name": `" child::名字 "`}, false},
		{"null name still dirty", true, RecordRequest{ID: propertyRecordChild}, map[string]any{"name": nil}, map[string]string{"name": "null"}, nil, false},
		{"explicit defaults still dirty", true, RecordRequest{ID: propertyRecordChild}, map[string]any{"min_length": 0, "min_items": 0, "require_unique_items": false}, map[string]string{"minLength": "0", "minItems": "0", "uniqueItems": "false"}, nil, false},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls, locations := 0, 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				propertyRecordAssertWriteBody(t, req, check.update, propertyRecordChild, check.body)
				return propertyCoreJSON(req, 204, ""), nil
			})
			cloud := "local snapshot"
			location := resource.CloudLocation{Cloud: &cloud}
			captured, err := location.ForResource(nil, location.Zone)
			if err != nil {
				t.Fatal(err)
			}
			options := propertyRecordWriteAttributes(check.attrs)
			options.create = append(options.create, WithRecordCreateHeader("X-Only", "header"))
			options.update = append(options.update, WithRecordUpdateHeader("X-Only", "header"))
			record, err := propertyRecordWriteCall(propertyRecordScope(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return location, nil }}), context.Background(), check.update, check.input, options)
			want := propertyRecordWriteDefaults(check.update)
			want["location"] = string(captured)
			for key, value := range check.want {
				want[key] = value
			}
			if err != nil || record == nil || locations != 1 || record.Key != nil || record.Namespace != propertyRecordParent || !reflect.DeepEqual(propertyRecordValues(record), want) {
				t.Fatal(record, err, calls, locations, propertyRecordValues(record), want)
			}
			if check.local {
				if calls != 0 || record.Wire != nil || record.Envelope != nil || record.Header != nil || record.StatusCode != 0 || record.Resource.Header != nil || record.Resource.StatusCode != 0 {
					t.Fatal("local commit invented HTTP evidence", record, calls)
				}
			} else if calls != 1 || record.StatusCode != 204 || record.Wire != nil || len(record.Envelope) != 0 || record.Header.Get("X-Proof") != "original" {
				t.Fatal("empty success lost actual evidence", record, calls)
			}
		})
	}
	t.Run("resource identity captured before callbacks", func(t *testing.T) {
		calls := 0
		seed := propertyRecordSeed(t, `{"id":"fixed-id","name":"old-name","title":"old-title","minimum":"²"}`)
		client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			propertyRecordAssertWriteBody(t, req, true, "fixed-id", map[string]string{"name": `"renamed/name"`})
			return propertyCoreJSON(req, 200, `{}`), nil
		})
		record, err := propertyRecordScope(t, client, Dependencies{}).UpdateRecord(context.Background(), RecordRequest{Resource: seed}, func(*RecordUpdateOpts) error { seed.Body["id"] = json.RawMessage(`"caller changed"`); return nil }, WithRecordUpdateAttribute("name", "renamed/name"))
		if err != nil || record == nil || calls != 1 || string(record.Resource.Body["id"]) != `"fixed-id"` || string(record.Resource.Body["name"]) != `"renamed/name"` || string(record.Resource.Body["title"]) != "null" || string(record.Resource.Body["minimum"]) != "null" {
			t.Fatal("old resource definition entered new commit", record, err, calls)
		}
	})
}

func TestMetadefPropertyRecordWritesActualStatusesOverlayAndOwnedChannels(t *testing.T) {
	responseBody := `{"id":{"passive":true},"name":null,"minimum":"04","readonly":"false","namespace_name":"wire parent","location":{"cloud":"wire"},"created_at":"wire date","self":"https://foreign.test/","future":{"number":900719925474099312345}}`
	for _, operation := range propertyRecordWrites {
		for _, code := range []int{200, 201, 202, 203, 204, 299, 300, 304, 399} {
			t.Run(fmt.Sprintf("%s/%d", operation.name, code), func(t *testing.T) {
				calls := 0
				client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					propertyRecordAssertWriteBody(t, req, operation.update, propertyRecordChild, map[string]string{"title": `"seed title"`, "name": `"seed name"`, "minLength": "6"})
					return propertyCoreJSON(req, code, responseBody), nil
				})
				record, err := propertyRecordWriteCall(propertyRecordScope(t, client, Dependencies{}), context.Background(), operation.update, RecordRequest{ID: propertyRecordChild}, propertyRecordWriteAttributes(map[string]any{"title": "seed title", "name": "seed name", "min_length": 6}))
				if err != nil || record == nil || record.Wire == nil || calls != 1 || record.Namespace != propertyRecordParent || record.Key != nil || len(record.Resource.Body) != 21 || string(record.Resource.Body["id"]) != `{"passive":true}` || string(record.Resource.Body["name"]) != "null" || string(record.Resource.Body["title"]) != `"seed title"` || string(record.Resource.Body["minimum"]) != "4" || string(record.Resource.Body["min_length"]) != "6" || string(record.Resource.Body["is_readonly"]) != "true" || string(record.Resource.Body["location"]) != "null" {
					t.Fatal(record, err, calls)
				}
				if record.StatusCode != code || record.Resource.StatusCode != code || record.Wire.StatusCode != code || record.Header.Get("X-Proof") != "original" || record.Resource.Header.Get("X-Proof") != "original" || record.Wire.Header.Get("X-Proof") != "original" || string(record.Envelope) != responseBody || string(record.Wire.Body["future"]) != `{"number":900719925474099312345}` || string(record.Wire.Body["readonly"]) != `"false"` {
					t.Fatal("actual channels narrowed", record)
				}
				for _, key := range []string{"self", "created_at", "future", "schema"} {
					if _, exists := record.Resource.Body[key]; exists {
						t.Fatal("unknown response field escaped projection", key)
					}
				}
				record.Resource.Body["id"][0] = 'X'
				record.Resource.Header.Set("X-Proof", "view changed")
				record.Envelope[0] = 'X'
				record.Header.Set("X-Proof", "record changed")
				if string(record.Wire.Body["id"]) != `{"passive":true}` || record.Wire.Header.Get("X-Proof") != "original" || string(record.Wire.Body["location"]) != `{"cloud":"wire"}` {
					t.Fatal("owned result channels alias")
				}
			})
		}
	}
	for _, operation := range propertyRecordWrites {
		t.Run(operation.name+" response duplicate aliases", func(t *testing.T) {
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				return propertyCoreJSON(req, 200, `{"readonly":false,"is_readonly":true,"readonly":null,"minLength":4,"min_length":"02"}`), nil
			})
			record, err := propertyRecordWriteCall(propertyRecordScope(t, client, Dependencies{}), context.Background(), operation.update, RecordRequest{ID: propertyRecordChild}, propertyRecordWriteAttributes(map[string]any{"title": "seed"}))
			if err != nil || record == nil || string(record.Resource.Body["is_readonly"]) != "true" || string(record.Resource.Body["min_length"]) != "2" || string(record.Wire.Body["readonly"]) != "null" {
				t.Fatal("response ordered alias rules changed", record, err)
			}
		})
	}
}

func TestMetadefPropertyRecordWritesSuccessDecodeAndAcceptedFaultReceipts(t *testing.T) {
	for _, operation := range propertyRecordWrites {
		for _, check := range []struct {
			name, body string
			tolerated  bool
		}{
			{"empty", "", true}, {"invalid JSON", `not JSON`, true}, {"unfinished object", `{"minimum":`, true},
			{"null", `null`, false}, {"array", `[]`, false}, {"string", `"scalar"`, false}, {"number", `7`, false}, {"boolean", `false`, false},
			{"bad descriptor", `{"minimum":"²"}`, false}, {"overflow descriptor", `{"maximum":1e400}`, false}, {"whole body invalid UTF8", "{\"future\":\"\xff\"}", false},
		} {
			t.Run(operation.name+"/"+check.name, func(t *testing.T) {
				calls := 0
				client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return propertyCoreJSON(req, 202, check.body), nil
				})
				record, err := propertyRecordWriteCall(propertyRecordScope(t, client, Dependencies{}), context.Background(), operation.update, RecordRequest{ID: propertyRecordChild}, propertyRecordWriteAttributes(map[string]any{"title": "seed", "min_length": "04"}))
				if calls != 1 {
					t.Fatal(calls)
				}
				if check.tolerated {
					if err != nil || record == nil || record.Wire != nil || record.StatusCode != 202 || string(record.Envelope) != check.body || string(record.Resource.Body["title"]) != `"seed"` || string(record.Resource.Body["min_length"]) != "4" {
						t.Fatal("tolerated decode discarded seed or receipt", record, err)
					}
				} else {
					if record != nil || err == nil {
						t.Fatal(record, err)
					}
					propertyCoreProof(t, err, 202, check.body)
				}
			})
		}
	}
	for _, operation := range propertyRecordWrites {
		for _, mode := range []string{"read close cancel", "scope changed on Close", "observed source restored"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				calls, retries := 0, 0
				readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				bodyText := `{"title":"server"}`
				var scope *NamespaceScope
				var client *gophercloud.ServiceClient
				var body *propertyCoreBody
				client = propertyCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					body = &propertyCoreBody{reader: strings.NewReader(bodyText)}
					if mode == "read close cancel" {
						body.reader = &propertyCoreReader{data: bodyText, err: readErr, after: func() { cancel(cause) }}
						body.closeErr = closeErr
					}
					if mode == "scope changed on Close" {
						return propertyCoreHTTP(req, 203, &propertyRecordCloseBody{propertyCoreBody: body, after: func() { scope.namespace = "changed" }}), nil
					}
					if mode == "observed source restored" {
						body.reader = &propertyCoreReader{data: bodyText, err: io.EOF, after: func() {
							client.Microversion = "2.3"
							if err := rest.CheckOperationGuard(req.Context()); !errors.Is(err, resource.ErrInvalidOption) {
								t.Error("source change unobserved", err)
							}
						}}
						return propertyCoreHTTP(req, 203, &propertyRecordCloseBody{propertyCoreBody: body, after: func() { client.Microversion = "" }}), nil
					}
					return propertyCoreHTTP(req, 203, body), nil
				})
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				scope = propertyRecordScope(t, client, Dependencies{})
				record, err := propertyRecordWriteCall(scope, ctx, operation.update, RecordRequest{ID: propertyRecordChild}, propertyRecordWriteAttributes(map[string]any{"title": "seed"}))
				if record != nil || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
					t.Fatal(record, err, calls, retries, body.closes)
				}
				proof := propertyCoreProof(t, err, 203, bodyText)
				if mode == "read close cancel" {
					for _, expected := range []error{readErr, closeErr, cause, context.Canceled} {
						if !errors.Is(err, expected) {
							t.Fatal("accepted cause lost", expected, err)
						}
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("sticky guard cause lost", err)
				}
				if proof.Header.Get("X-Proof") != "original" {
					t.Fatal("fault proof changed")
				}
			})
		}
	}
}

func TestMetadefPropertyRecordWriteFactoriesSelectionAndCapturedLocation(t *testing.T) {
	for _, operation := range propertyRecordWrites {
		t.Run(operation.name+" owned factories", func(t *testing.T) {
			calls, locations, callbacks := 0, 0, 0
			cloud := "captured"
			location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"original project"`)}}
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				propertyRecordAssertWriteBody(t, req, operation.update, propertyRecordChild, map[string]string{"description": `"last description"`, "default": `{"nested":"owned"}`})
				if req.Header.Get("X-Full") != "owned" || req.Header.Get("X-Bulk") != "factory" || req.Header.Get("X-Final") != "final" || req.Header.Get("X-Replaced") != "" || req.Header.Get("X-Source") != "before" {
					t.Fatal("write option ownership", req.Header)
				}
				return propertyCoreJSON(req, 200, `{"location":{"cloud":"wire"}}`), nil
			})
			client.MoreHeaders = map[string]string{"X-Source": "before"}
			scope := propertyRecordScope(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return location, nil }})
			headers := map[string]string{"X-Full": "owned"}
			bulk := map[string]string{"X-Bulk": "factory"}
			nested := map[string]any{"nested": "owned"}
			attributes := map[string]any{"default": nested, "unknown": make(chan int)}
			slice := []resource.ListOption{resource.WithFilters(attributes)}
			options := propertyRecordWriteOptions{}
			if operation.update {
				var retained *RecordUpdateOpts
				options.update = []RecordUpdateOption{WithRecordUpdateHeader("X-Replaced", "old"), WithRecordUpdateOpts(RecordUpdateOpts{Headers: headers, Attributes: slice}), WithRecordUpdateHeaders(bulk),
					func(config *RecordUpdateOpts) error {
						callbacks++
						retained = config
						cloud = "option changed"
						location.Project.ID[1] = 'X'
						client.MoreHeaders["X-Source"] = "option changed"
						return nil
					},
					func(*RecordUpdateOpts) error {
						retained.Headers["X-Full"] = "retained changed"
						retained.Attributes[0] = resource.WithFilter("default", "retained changed")
						return nil
					}, WithRecordUpdateHeader("x-final", "final"), WithRecordUpdateAttribute("description", "last description")}
			} else {
				var retained *RecordCreateOpts
				options.create = []RecordCreateOption{WithRecordCreateHeader("X-Replaced", "old"), WithRecordCreateOpts(RecordCreateOpts{Headers: headers, Attributes: slice}), WithRecordCreateHeaders(bulk),
					func(config *RecordCreateOpts) error {
						callbacks++
						retained = config
						cloud = "option changed"
						location.Project.ID[1] = 'X'
						client.MoreHeaders["X-Source"] = "option changed"
						return nil
					},
					func(*RecordCreateOpts) error {
						retained.Headers["X-Full"] = "retained changed"
						retained.Attributes[0] = resource.WithFilter("default", "retained changed")
						return nil
					}, WithRecordCreateHeader("x-final", "final"), WithRecordCreateAttribute("description", "last description")}
			}
			headers["X-Full"] = "caller changed"
			bulk["X-Bulk"] = "caller changed"
			nested["nested"] = "caller changed"
			attributes["default"] = "caller changed"
			slice[0] = resource.WithFilter("default", "caller changed")
			record, err := propertyRecordWriteCall(scope, context.Background(), operation.update, RecordRequest{ID: propertyRecordChild}, options)
			if err != nil || record == nil || calls != 1 || locations != 1 || callbacks != 1 || string(record.Resource.Body["default"]) != `{"nested":"owned"}` {
				t.Fatal(record, err, calls, locations, callbacks)
			}
			var actual resource.CloudLocation
			if err := json.Unmarshal(record.Resource.Body["location"], &actual); err != nil || actual.Cloud == nil || *actual.Cloud != "captured" || string(actual.Project.ID) != `"original project"` || string(record.Wire.Body["location"]) != `{"cloud":"wire"}` {
				t.Fatal("write location not captured before options", actual, err)
			}
		})
	}
	for _, operation := range propertyRecordWrites {
		for _, check := range []struct {
			name  string
			steps []resource.ListOption
			body  map[string]string
			view  map[string]string
		}{
			{"canonical beats bad alias", []resource.ListOption{resource.WithFilters(map[string]any{"is_readonly": false, "readonly": make(chan int)})}, map[string]string{"readonly": "false"}, map[string]string{"is_readonly": "false"}},
			{"canonical null beats alias", []resource.ListOption{resource.WithFilters(map[string]any{"min_length": nil, "minLength": "²"})}, map[string]string{"minLength": "null"}, map[string]string{"min_length": "null"}},
			{"wire aliases accepted", []resource.ListOption{resource.WithFilters(map[string]any{"readonly": "false", "maxItems": "03"})}, map[string]string{"readonly": `"false"`, "maxItems": `"03"`}, map[string]string{"is_readonly": "true", "max_items": "3"}},
			{"bulk replaces failed value", []resource.ListOption{resource.WithFilter("title", make(chan int)), resource.WithFilters(map[string]any{"description": "replacement"})}, map[string]string{"description": `"replacement"`}, map[string]string{"title": "null", "description": `"replacement"`}},
			{"singular replaces failed value", []resource.ListOption{resource.WithFilter("title", make(chan int)), resource.WithFilter("title", nil)}, map[string]string{"title": "null"}, map[string]string{"title": "null"}},
			{"empty bulk clears body", []resource.ListOption{resource.WithFilter("title", make(chan int)), resource.WithFilters(nil)}, map[string]string{}, nil},
			{"unknown failure ignored", []resource.ListOption{resource.WithFilters(map[string]any{"title": false, "future": make(chan int)})}, map[string]string{"title": "false"}, map[string]string{"title": "false"}},
		} {
			t.Run(operation.name+"/"+check.name, func(t *testing.T) {
				calls := 0
				client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					propertyRecordAssertWriteBody(t, req, operation.update, propertyRecordChild, check.body)
					return propertyCoreJSON(req, 200, `{}`), nil
				})
				options := propertyRecordWriteOptions{create: []RecordCreateOption{WithRecordCreateOpts(RecordCreateOpts{Attributes: check.steps})}, update: []RecordUpdateOption{WithRecordUpdateOpts(RecordUpdateOpts{Attributes: check.steps})}}
				record, err := propertyRecordWriteCall(propertyRecordScope(t, client, Dependencies{}), context.Background(), operation.update, RecordRequest{ID: propertyRecordChild}, options)
				expectedCalls := 1
				if operation.update && len(check.body) == 0 {
					expectedCalls = 0
				}
				if err != nil || record == nil || calls != expectedCalls {
					t.Fatal(record, err, calls)
				}
				for key, value := range check.view {
					if string(record.Resource.Body[key]) != value {
						t.Fatal("selected view", key, string(record.Resource.Body[key]), value)
					}
				}
			})
		}
	}
}

func TestMetadefPropertyRecordWriteInputAttributeAndHeaderPreflight(t *testing.T) {
	for _, operation := range propertyRecordWrites {
		for _, check := range []struct {
			name    string
			options propertyRecordWriteOptions
		}{
			{"nil operation option", propertyRecordWriteOptions{create: []RecordCreateOption{nil}, update: []RecordUpdateOption{nil}}},
			{"nil semantic option", propertyRecordWriteOptions{create: []RecordCreateOption{WithRecordCreateOpts(RecordCreateOpts{Attributes: []resource.ListOption{nil}})}, update: []RecordUpdateOption{WithRecordUpdateOpts(RecordUpdateOpts{Attributes: []resource.ListOption{nil}})}}},
			{"nonsemantic option", propertyRecordWriteOptions{create: []RecordCreateOption{WithRecordCreateOpts(RecordCreateOpts{Attributes: []resource.ListOption{resource.WithQuery("title", "foreign")}})}, update: []RecordUpdateOption{WithRecordUpdateOpts(RecordUpdateOpts{Attributes: []resource.ListOption{resource.WithQuery("title", "foreign")}})}}},
			{"known encoding failure", propertyRecordWriteAttributes(map[string]any{"title": make(chan int)})},
			{"constructor descriptor failure", propertyRecordWriteAttributes(map[string]any{"minimum": "²"})},
			{"protected token", propertyRecordWriteOptions{create: []RecordCreateOption{WithRecordCreateHeader("X-Auth-Token", "foreign")}, update: []RecordUpdateOption{WithRecordUpdateHeader("X-Auth-Token", "foreign")}}},
			{"duplicate header aliases", propertyRecordWriteOptions{create: []RecordCreateOption{WithRecordCreateHeaders(map[string]string{"X-Alias": "one", "x-alias": "two"})}, update: []RecordUpdateOption{WithRecordUpdateHeaders(map[string]string{"X-Alias": "one", "x-alias": "two"})}}},
			{"invalid header value", propertyRecordWriteOptions{create: []RecordCreateOption{WithRecordCreateHeader("X-Option", "bad\nvalue")}, update: []RecordUpdateOption{WithRecordUpdateHeader("X-Option", "bad\nvalue")}}},
		} {
			t.Run(operation.name+"/"+check.name, func(t *testing.T) {
				calls := 0
				client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
				record, err := propertyRecordWriteCall(propertyRecordScope(t, client, Dependencies{}), context.Background(), operation.update, RecordRequest{ID: propertyRecordChild}, check.options)
				if record != nil || err == nil || calls != 0 {
					t.Fatal("write preflight reached HTTP", record, err, calls)
				}
			})
		}
	}
	for _, operation := range propertyRecordWrites {
		for _, key := range []string{"namespace_name", "namespace", "base_path", "requires_id", "session", "microversion", "headers", "connection", "_synchronized", "__conflicting_attrs", "resource_request_key", "resource_response_key", "resource_type_class", "prepend_key", "has_body", "retry_on_conflict", "resource_type"} {
			t.Run(operation.name+" reserved "+key, func(t *testing.T) {
				calls := 0
				client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
				record, err := propertyRecordWriteCall(propertyRecordScope(t, client, Dependencies{}), context.Background(), operation.update, RecordRequest{ID: propertyRecordChild}, propertyRecordWriteAttributes(map[string]any{key: make(chan int)}))
				if record != nil || err == nil || calls != 0 {
					t.Fatal("reserved binding entered a write", key, record, err, calls)
				}
			})
		}
	}
	for _, check := range []struct {
		name  string
		input RecordRequest
		attrs map[string]any
	}{
		{"missing input", RecordRequest{}, nil},
		{"both inputs", RecordRequest{ID: propertyRecordChild, Resource: &resource.RawResource{}}, nil},
		{"null identity never falls back", RecordRequest{Resource: propertyRecordSeed(t, `{"id":null,"name":"safe"}`)}, nil},
		{"unsafe selected identity", RecordRequest{Resource: propertyRecordSeed(t, `{"id":"bad/name","name":"safe"}`)}, nil},
		{"attributes cannot provide missing identity", RecordRequest{Resource: &resource.RawResource{}}, map[string]any{"name": "safe"}},
		{"id cannot retarget string", RecordRequest{ID: propertyRecordChild}, map[string]any{"id": propertyRecordChild}},
		{"id cannot retarget resource", RecordRequest{Resource: propertyRecordSeed(t, `{"name":" child::名字 "}`)}, map[string]any{"id": nil}},
		{"value binding cannot override input", RecordRequest{ID: propertyRecordChild}, map[string]any{"value": "other"}},
	} {
		t.Run("update/"+check.name, func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			record, err := propertyRecordScope(t, client, Dependencies{}).UpdateRecord(context.Background(), check.input, WithRecordUpdateAttributes(check.attrs))
			if record != nil || err == nil || calls != 0 {
				t.Fatal(record, err, calls)
			}
		})
	}
}

func TestMetadefPropertyRecordWriteGuardsAlsoCoverLocalNoop(t *testing.T) {
	for _, operation := range propertyRecordWrites {
		for _, mode := range []string{"nil context", "canceled context", "source lifetime", "outer preflight", "location failure", "location changed source", "callback failure", "callback source change", "callback namespace change"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				calls, locations, first, second := 0, 0, 0, 0
				cause := errors.New("guard cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var use context.Context = ctx
				client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
				scope := propertyRecordScope(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
					locations++
					if mode == "location failure" {
						return resource.CloudLocation{}, cause
					}
					if mode == "location changed source" {
						client.Endpoint = "https://changed.test/"
					}
					return resource.CloudLocation{}, nil
				}})
				switch mode {
				case "nil context":
					use = nil
				case "canceled context":
					cancel(cause)
				case "source lifetime":
					client.Endpoint = "https://changed.test/"
				case "outer preflight":
					use = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
				}
				mutate := func() error {
					first++
					if mode == "callback failure" {
						return cause
					}
					if mode == "callback source change" {
						client.Endpoint = "https://changed.test/"
					}
					if mode == "callback namespace change" {
						scope.namespace = "changed"
					}
					return nil
				}
				restore := func() error {
					second++
					client.Endpoint = "https://example.test/catalog/"
					scope.namespace = propertyRecordParent
					return nil
				}
				options := propertyRecordWriteOptions{create: []RecordCreateOption{func(*RecordCreateOpts) error { return mutate() }, func(*RecordCreateOpts) error { return restore() }}, update: []RecordUpdateOption{func(*RecordUpdateOpts) error { return mutate() }, func(*RecordUpdateOpts) error { return restore() }}}
				record, err := propertyRecordWriteCall(scope, use, operation.update, RecordRequest{ID: propertyRecordChild}, options)
				if record != nil || err == nil || calls != 0 || second != 0 {
					t.Fatal("guard failure was restored or bypassed by no-op", record, err, calls, locations, first, second)
				}
				if strings.HasPrefix(mode, "callback") {
					if first != 1 || locations != 1 {
						t.Fatal(first, locations)
					}
				} else if first != 0 {
					t.Fatal("preflight invoked options", first)
				}
				if mode == "nil context" || mode == "canceled context" || mode == "source lifetime" || mode == "outer preflight" {
					if locations != 0 {
						t.Fatal("source preflight invoked location", locations)
					}
				}
				if mode == "canceled context" || mode == "outer preflight" || mode == "location failure" || mode == "callback failure" {
					if !errors.Is(err, cause) {
						t.Fatal("preflight cause lost", err)
					}
				}
			})
		}
	}
	t.Run("local update captures location and guards after every callback", func(t *testing.T) {
		calls, locations, callbacks := 0, 0, 0
		cloud := "original"
		location := resource.CloudLocation{Cloud: &cloud}
		client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
		scope := propertyRecordScope(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return location, nil }})
		record, err := scope.UpdateRecord(context.Background(), RecordRequest{ID: propertyRecordChild}, func(*RecordUpdateOpts) error { callbacks++; cloud = "changed"; return nil }, WithRecordUpdateHeader("X-Noop", "validated"))
		var actual resource.CloudLocation
		if err != nil || record == nil || calls != 0 || locations != 1 || callbacks != 1 {
			t.Fatal(record, err, calls, locations, callbacks)
		}
		if err := json.Unmarshal(record.Resource.Body["location"], &actual); err != nil || actual.Cloud == nil || *actual.Cloud != "original" {
			t.Fatal("local result did not own captured location", actual, err)
		}
	})
}

func TestMetadefPropertyRecordWritesNativeBodyOwnershipAndLegacyContracts(t *testing.T) {
	for _, operation := range propertyRecordWrites {
		for _, mode := range []string{"equivalent retry body", "changed retry body", "expanded native status"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				calls, retries := 0, 0
				var physical []*propertyCoreBody
				client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					propertyRecordAssertWriteBody(t, req, operation.update, propertyRecordChild, map[string]string{"title": `"owned"`})
					status, text := 503, "physical failure"
					if calls > 1 {
						status, text = 203, `{"title":"response"}`
						if mode == "expanded native status" {
							status, text = 400, "outside SDK policy"
						}
					}
					body := &propertyCoreBody{reader: strings.NewReader(text)}
					physical = append(physical, body)
					return propertyCoreHTTP(req, status, body), nil
				})
				callbackCause := errors.New("callback rejection")
				client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
					retries++
					if retries > 1 {
						return errors.New("unexpected extra retry")
					}
					switch mode {
					case "equivalent retry body":
						options.JSONBody = json.RawMessage(bytes.Clone(options.JSONBody.(json.RawMessage)))
					case "changed retry body":
						raw := options.JSONBody.(json.RawMessage)
						raw[10] = 'X'
						return callbackCause
					case "expanded native status":
						options.OkCodes = append(options.OkCodes, 400)
					}
					return nil
				}
				record, err := propertyRecordWriteCall(propertyRecordScope(t, client, Dependencies{}), context.Background(), operation.update, RecordRequest{ID: propertyRecordChild}, propertyRecordWriteAttributes(map[string]any{"title": "owned"}))
				if mode == "equivalent retry body" {
					if err != nil || record == nil || calls != 2 || retries != 1 || record.StatusCode != 203 || string(record.Resource.Body["title"]) != `"response"` {
						t.Fatal(record, err, calls, retries)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if record != nil || !errors.As(err, &native) || native.ResponseHeader.Get("X-Proof") != "original" || retries != 1 {
						t.Fatal("native evidence lost", record, err, native, calls, retries)
					}
					if mode == "changed retry body" {
						if calls != 1 || native.Actual != 503 || string(native.Body) != "physical failure" || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) {
							t.Fatal("request JSON ownership lost", err, native, calls)
						}
					} else if calls != 2 || native.Actual != 400 || string(native.Body) != "outside SDK policy" {
						t.Fatal("native status expansion became success", err, native, calls)
					}
					var proof *resource.ResponseError
					if errors.As(err, &proof) {
						t.Fatal("native failure acquired accepted receipt", proof)
					}
				}
				for _, body := range physical {
					if body.closes != 1 {
						t.Fatal("physical body close count", body.closes)
					}
				}
			})
		}
	}
	for _, check := range []struct {
		name     string
		update   bool
		required bool
		code     int
	}{
		{"legacy create requires fields", false, true, 201}, {"legacy update requires fields", true, true, 200},
		{"legacy create strict201", false, false, 200}, {"legacy update strict200", true, false, 201},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return propertyCoreJSON(req, check.code, `{}`), nil
			})
			scope := propertyRecordScope(t, client, Dependencies{})
			var value *Property
			var err error
			if check.update {
				if check.required {
					value, err = scope.Update(context.Background(), propertyRecordChild)
				} else {
					value, err = scope.Update(context.Background(), propertyRecordChild, WithUpdateType("string"), WithUpdateTitle("title"))
				}
			} else {
				if check.required {
					value, err = scope.Create(context.Background(), propertyRecordChild)
				} else {
					value, err = scope.Create(context.Background(), propertyRecordChild, WithCreateType("string"), WithCreateTitle("title"))
				}
			}
			if value != nil || err == nil {
				t.Fatal("legacy write policy changed", value, err)
			}
			if check.required {
				if calls != 0 || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err, calls)
				}
			} else if calls != 1 || !gophercloud.ResponseCodeIs(err, check.code) {
				t.Fatal(err, calls)
			}
		})
	}
}
