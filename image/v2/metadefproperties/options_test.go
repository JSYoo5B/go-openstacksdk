package metadefproperties

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func propertyOptionPointer[T any](value T) *T { return &value }

type propertyOptionMarshaler struct {
	cause error
	calls *int
}

func (value propertyOptionMarshaler) MarshalJSON() ([]byte, error) {
	if value.calls != nil {
		*value.calls++
	}
	if value.cause != nil {
		return nil, value.cause
	}
	return []byte(`{"n":1}`), nil
}

func TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins(t *testing.T) {
	input := CreateOpts{Headers: map[string]string{"X-Original": "before"}, Type: propertyOptionPointer("number"), Title: propertyOptionPointer("before"), Description: propertyOptionPointer("before"), Attributes: map[string]json.RawMessage{"minimum": json.RawMessage(`1.25`), "default": json.RawMessage(`null`)}}
	option := WithCreateOpts(input)
	*input.Type = "after"
	*input.Title = "after"
	*input.Description = "after"
	input.Attributes["minimum"][0] = '2'
	input.Headers["X-Original"] = "after"
	config, err := prepareCreate([]CreateOption{WithCreateHeader("X-Removed", "yes"), option, WithCreateTitle(""), WithCreateHeader("x-original", "last")})
	if err != nil || *config.Type != "number" || *config.Title != "" || *config.Description != "before" || string(config.Attributes["minimum"]) != "1.25" || string(config.Attributes["default"]) != "null" || config.Headers["X-Original"] != "last" || config.Headers["X-Removed"] != "" {
		t.Fatalf("fullsnapshot %+v %v", config, err)
	}
	*config.Type = "local"
	config.Attributes["minimum"][0] = '9'
	config.Headers["X-Original"] = "local"
	again, err := prepareCreate([]CreateOption{option})
	if err != nil || *again.Type != "number" || string(again.Attributes["minimum"]) != "1.25" || again.Headers["X-Original"] != "before" {
		t.Fatalf("reusable %+v %v", again, err)
	}
	updateInput := UpdateOpts{Name: propertyOptionPointer("renamed"), Type: propertyOptionPointer("string"), Title: propertyOptionPointer(""), Attributes: map[string]json.RawMessage{"readonly": json.RawMessage(`false`)}}
	update := WithUpdateOpts(updateInput)
	*updateInput.Name = "after"
	updateInput.Attributes["readonly"][0] = 't'
	updated, err := prepareUpdate([]UpdateOption{WithUpdateHeader("X-Removed", "yes"), update})
	if err != nil || *updated.Name != "renamed" || string(updated.Attributes["readonly"]) != "false" || updated.Headers["X-Removed"] != "" {
		t.Fatalf("update %+v %v", updated, err)
	}
	resourceType := "OS::Type"
	get := WithGetOpts(GetOpts{ResourceType: &resourceType})
	resourceType = "changed"
	fetched, err := prepareGet([]GetOption{get, WithGetResourceType("")})
	if err != nil || fetched.ResourceType == nil || *fetched.ResourceType != "" {
		t.Fatalf("get %+v %v", fetched, err)
	}
	ignore := false
	deletion := WithDeleteOpts(DeleteOpts{IgnoreMissing: &ignore})
	ignore = true
	deleted, err := prepareDelete([]DeleteOption{WithDeleteIgnoreMissing(true), deletion})
	if err != nil || *deleted.IgnoreMissing {
		t.Fatalf("delete %+v %v", deleted, err)
	}
	bulk, err := prepareDeleteAll([]DeleteAllOption{WithDeleteAllHeader("X-Removed", "yes"), WithDeleteAllOpts(DeleteAllOpts{}), WithDeleteAllHeader("X-Bulk", "last")})
	if err != nil || bulk.Headers["X-Removed"] != "" || bulk.Headers["X-Bulk"] != "last" {
		t.Fatalf("bulk %+v %v", bulk, err)
	}
	listed, err := prepareList([]ListOption{WithListMaxItems(9), WithListOpts(ListOpts{MaxItems: 1}), WithListMaxItems(2)})
	if err != nil || listed.MaxItems != 2 {
		t.Fatalf("list %+v %v", listed, err)
	}
	nested := map[string]any{"value": []any{json.Number("9007199254740993")}}
	attrs := map[string]any{"default": nested, "readonly": false}
	plural := WithCreateAttributes(attrs)
	nested["value"].([]any)[0] = "changed"
	attrs["readonly"] = true
	literal := json.RawMessage(`{"n":1}`)
	singular := WithCreateAttribute("custom", literal)
	literal[5] = '2'
	captured, err := prepareCreate([]CreateOption{WithCreateType("unknown"), WithCreateTitle(""), WithCreateAttribute("removed", 1), plural, singular, WithCreateAttribute("readonly", false)})
	if err != nil || string(captured.Attributes["default"]) != `{"value":[9007199254740993]}` || string(captured.Attributes["custom"]) != `{"n":1}` || string(captured.Attributes["readonly"]) != "false" {
		t.Fatalf("factory snapshots %+v %v", captured, err)
	}
	if _, exists := captured.Attributes["removed"]; exists {
		t.Fatal("plural does not replace")
	}
	var zero GetOpts
	if err := WithGetHeader("X-Direct", "yes")(&zero); err != nil || zero.Headers["X-Direct"] != "yes" {
		t.Fatalf("zerohelper %+v %v", zero, err)
	}
}

func TestMetadefPropertiesOptionsCallbackAndFactoryOwnership(t *testing.T) {
	var retained *CreateOpts
	calls := 0
	first := CreateOption(func(o *CreateOpts) error {
		calls++
		if o.Headers == nil {
			t.Fatal("Headers nil")
		}
		o.Type = propertyOptionPointer("string")
		o.Title = propertyOptionPointer("before")
		o.Description = propertyOptionPointer("before")
		o.Attributes = map[string]json.RawMessage{"default": json.RawMessage(`{"n":1}`)}
		o.Headers["X-Handle"] = "before"
		retained = o
		return nil
	})
	second := CreateOption(func(o *CreateOpts) error {
		calls++
		*retained.Type = "old"
		*retained.Title = "old"
		retained.Attributes["default"][5] = '2'
		retained.Headers["X-Handle"] = "old"
		if *o.Type != "string" || *o.Title != "before" || string(o.Attributes["default"]) != `{"n":1}` || o.Headers["X-Handle"] != "before" {
			t.Fatal("old callback handle aliases new")
		}
		return nil
	})
	config, err := prepareCreate([]CreateOption{first, second})
	if err != nil || calls != 2 {
		t.Fatal(err)
	}
	*retained.Title = "closed"
	retained.Attributes["default"][5] = '3'
	retained.Headers["X-Handle"] = "closed"
	if *config.Title != "before" || string(config.Attributes["default"]) != `{"n":1}` || config.Headers["X-Handle"] != "before" {
		t.Fatal("closed handle affects prepared")
	}
	encodedCalls := 0
	factory := WithCreateAttributes(map[string]any{"default": propertyOptionMarshaler{calls: &encodedCalls}})
	if encodedCalls != 1 {
		t.Fatalf("factory marshals%d", encodedCalls)
	}
	for i := 0; i < 2; i++ {
		if _, err := prepareCreate([]CreateOption{WithCreateType("string"), WithCreateTitle(""), factory}); err != nil {
			t.Fatal(err)
		}
	}
	if encodedCalls != 1 {
		t.Fatal("factory remarshal on reuse")
	}
	encodedCalls = 0
	bad := WithCreateAttributes(map[string]any{string([]byte{0xff}): propertyOptionMarshaler{calls: &encodedCalls}})
	if _, err := prepareCreate([]CreateOption{WithCreateType("string"), WithCreateTitle(""), bad}); !errors.Is(err, resource.ErrInvalidOption) || encodedCalls != 0 {
		t.Fatalf("badkey beforeMarshal calls%d %v", encodedCalls, err)
	}
	requests := 0
	scope := propertyCoreScope(t, propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Header.Get("X-Chosen") != "yes" {
			t.Fatal(req.Header)
		}
		return propertyCoreJSON(req, 200, `{}`), nil
	}), "parent")
	var options []GetOption
	options = []GetOption{func(*GetOpts) error {
		options[1] = func(*GetOpts) error { return errors.New("caller slice changed") }
		return nil
	}, WithGetHeader("X-Chosen", "yes")}
	if _, err := scope.Get(context.Background(), "child", options...); err != nil || requests != 1 {
		t.Fatalf("methodslice %v %d", err, requests)
	}
	requests, calls = 0, 0
	scope = propertyCoreScope(t, propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		requests++
		return propertyCoreJSON(req, 200, `{"properties":{"one":{},"two":{}}}`), nil
	}), "parent")
	listOptions := []ListOption{func(*ListOpts) error { calls++; return nil }, WithListMaxItems(1)}
	seq := scope.List(context.Background(), listOptions...)
	listOptions[1] = func(*ListOpts) error { return errors.New("lazy slice changed") }
	if calls != 0 || requests != 0 {
		t.Fatal("iterator eager")
	}
	for i := 0; i < 2; i++ {
		count := 0
		for value, err := range seq {
			if value == nil || err != nil {
				t.Fatal(err)
			}
			count++
		}
		if count != 1 {
			t.Fatal(count)
		}
	}
	if calls != 2 || requests != 2 {
		t.Fatalf("lazyreuse callbacks%d requests%d", calls, requests)
	}
}

func TestMetadefPropertiesOptionsValidationAndDefaults(t *testing.T) {
	if _, err := prepareCreate(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("missing required create %v", err)
	}
	if _, err := prepareUpdate(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("missing required update %v", err)
	}
	created, err := prepareCreate([]CreateOption{WithCreateType("arbitrary-type"), WithCreateTitle(""), WithCreateDescription(strings.Repeat("界", 1000) + "\n"), WithCreateAttributes(map[string]any{"default": nil, "minimum": json.Number("1.25"), "maximum": json.Number("9007199254740993"), "": true, "unknown-extension": []any{}})})
	if err != nil || created.Headers == nil || *created.Title != "" || string(created.Attributes["default"]) != "null" {
		t.Fatalf("literaldefinition %+v %v", created, err)
	}
	fetched, err := prepareGet(nil)
	if err != nil || fetched.ResourceType != nil || fetched.Headers == nil {
		t.Fatalf("getdefault %+v %v", fetched, err)
	}
	deleted, err := prepareDelete(nil)
	if err != nil || deleted.IgnoreMissing != nil {
		t.Fatalf("deletedefault %+v %v", deleted, err)
	}
	bulk, err := prepareDeleteAll(nil)
	if err != nil || bulk.Headers == nil {
		t.Fatalf("bulkdefault %+v %v", bulk, err)
	}
	listed, err := prepareList(nil)
	if err != nil || listed.MaxItems != 0 {
		t.Fatalf("listdefault %+v %v", listed, err)
	}
	for _, key := range []string{"name", "type", "title", "description", "self", "schema", "created_at", "updated_at", "namespace_name"} {
		if _, err := prepareCreate([]CreateOption{WithCreateType("string"), WithCreateTitle(""), WithCreateAttribute(key, nil)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("reserved%s %v", key, err)
		}
	}
	if _, err := prepareCreate([]CreateOption{WithCreateType("string"), WithCreateTitle(""), WithCreateAttribute("Name", 1)}); err != nil {
		t.Fatal("casefolded reserved attributes", err)
	}
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`{`), json.RawMessage("\"\xff\"")} {
		if _, err := prepareCreate([]CreateOption{WithCreateOpts(CreateOpts{Type: propertyOptionPointer("string"), Title: propertyOptionPointer(""), Attributes: map[string]json.RawMessage{"default": raw}})}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("badraw%q %v", raw, err)
		}
	}
	for _, key := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "Accept", "Content-Type", "OpenStack-API-Version", "X-OpenStack-Glance-Api-Version", "X-OpenStack-Image-Size"} {
		if _, err := prepareGet([]GetOption{WithGetHeader(key, "value")}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("header%s %v", key, err)
		}
	}
	for _, headers := range []map[string]string{{"X-A": "one", "x-a": "two"}, {"Bad Key": "v"}, {"X-A": "a\rb"}, {"X-A": string([]byte{0xff})}} {
		if _, err := prepareDeleteAll([]DeleteAllOption{WithDeleteAllHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("badheaders %#v %v", headers, err)
		}
	}
	checks := []func() error{func() error { _, e := prepareCreate([]CreateOption{nil}); return e }, func() error { _, e := prepareUpdate([]UpdateOption{nil}); return e }, func() error { _, e := prepareGet([]GetOption{nil}); return e }, func() error { _, e := prepareDelete([]DeleteOption{nil}); return e }, func() error { _, e := prepareDeleteAll([]DeleteAllOption{nil}); return e }, func() error { _, e := prepareList([]ListOption{nil}); return e }}
	for i, check := range checks {
		if err := check(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nilfamily%d %v", i, err)
		}
	}
	cause := errors.New("callbackcause")
	if _, err := prepareUpdate([]UpdateOption{func(*UpdateOpts) error { return cause }}); err != cause {
		t.Fatal(err)
	}
	for _, option := range []CreateOption{WithCreateAttributes(map[string]any{"default": make(chan int)}), WithCreateAttribute("default", make(chan int))} {
		_, err := prepareCreate([]CreateOption{WithCreateType("string"), WithCreateTitle(""), option})
		var encoding *json.UnsupportedTypeError
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &encoding) {
			t.Fatalf("factoryerrorcause %v", err)
		}
	}
	empty, err := prepareCreate([]CreateOption{WithCreateType("string"), WithCreateTitle(""), WithCreateAttributes(map[string]any{})})
	if err != nil || empty.Attributes == nil {
		t.Fatalf("emptyattrs %+v %v", empty, err)
	}
	nilAttrs, err := prepareCreate([]CreateOption{WithCreateType("string"), WithCreateTitle(""), WithCreateAttributes(nil)})
	if err != nil || nilAttrs.Attributes != nil {
		t.Fatalf("nilattrs %+v %v", nilAttrs, err)
	}
	if !reflect.DeepEqual(propertyBody("child", nilAttrs.Type, nilAttrs.Title, nil, nil), map[string]any{"name": "child", "type": "string", "title": ""}) {
		t.Fatal("defaultflatbody")
	}
}

func TestMetadefPropertiesOptionsParallelReusableHelpers(t *testing.T) {
	var requests atomic.Int32
	scope := propertyCoreScope(t, propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		var body map[string]json.RawMessage
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			return nil, err
		}
		if req.Header.Get("X-Reuse") != "owned" || string(body["type"]) != `"number"` || string(body["minimum"]) != "1.25" || string(body["maximum"]) != "9007199254740993" {
			return nil, errors.New("helper snapshot changed")
		}
		return propertyCoreJSON(req, 201, `{"name":"server","type":"number","minimum":1.25,"maximum":9007199254740993}`), nil
	}), "parent")
	input := CreateOpts{Headers: map[string]string{"X-Reuse": "owned"}, Type: propertyOptionPointer("number"), Title: propertyOptionPointer(""), Attributes: map[string]json.RawMessage{"minimum": json.RawMessage(`1.25`)}}
	option := WithCreateOpts(input)
	*input.Type = "changed"
	input.Headers["X-Reuse"] = "changed"
	input.Attributes["minimum"][0] = '2'
	maximum := WithCreateAttribute("maximum", json.Number("9007199254740993"))
	var wg sync.WaitGroup
	failures := make(chan error, 24)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := scope.Create(context.Background(), "child", option, maximum)
			if err != nil {
				failures <- err
				return
			}
			value.Body["minimum"][0] = '8'
			value.Header.Set("X-Proof", "local")
			*value.Name = "local"
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if requests.Load() != 24 {
		t.Fatal(requests.Load())
	}
}
