package metadefobjects

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func objectOptionPointer[T any](value T) *T { return &value }

func TestMetadefObjectsOptionsSnapshotsReplacementAndLastWins(t *testing.T) {
	description := "before"
	properties := map[string]json.RawMessage{"key": json.RawMessage(`{"type":"before"}`)}
	required := []string{"key", "comma,entry", ""}
	headers := map[string]string{"X-Original": "before"}
	option := WithCreateOpts(CreateOpts{Headers: headers, Description: &description, Properties: properties, Required: required})
	description = "after"
	properties["key"][9] = 'X'
	required[0] = "after"
	headers["X-Original"] = "after"
	config, err := prepareCreate([]CreateOption{WithCreateHeader("X-Removed", "yes"), WithCreateDescription("removed"), option, WithCreateHeader("x-original", "last")})
	if err != nil || *config.Description != "before" || string(config.Properties["key"]) != `{"type":"before"}` || config.Required[0] != "key" || config.Headers["X-Original"] != "last" || config.Headers["X-Removed"] != "" {
		t.Fatalf("full snapshot %+v %v", config, err)
	}
	*config.Description = "local"
	config.Properties["key"][9] = 'L'
	config.Required[0] = "local"
	config.Headers["X-Original"] = "local"
	again, err := prepareCreate([]CreateOption{option})
	if err != nil || *again.Description != "before" || string(again.Properties["key"]) != `{"type":"before"}` || again.Required[0] != "key" || again.Headers["X-Original"] != "before" {
		t.Fatalf("reusable snapshot %+v %v", again, err)
	}
	updateInput := UpdateOpts{Name: objectOptionPointer("new::name"), Description: objectOptionPointer("before"), Properties: map[string]json.RawMessage{"x": json.RawMessage(`{}`)}, Required: []string{}, Headers: map[string]string{"X-Update": "before"}}
	update := WithUpdateOpts(updateInput)
	*updateInput.Name = "after"
	updateInput.Properties["x"] = json.RawMessage(`null`)
	updateInput.Headers["X-Update"] = "after"
	updated, err := prepareUpdate([]UpdateOption{WithUpdateHeader("X-Removed", "yes"), update, WithUpdateDescription("")})
	if err != nil || *updated.Name != "new::name" || *updated.Description != "" || string(updated.Properties["x"]) != `{}` || updated.Required == nil || updated.Headers["X-Removed"] != "" {
		t.Fatalf("update replacement %+v %v", updated, err)
	}
	ignore := false
	deletion := WithDeleteOpts(DeleteOpts{IgnoreMissing: &ignore, Headers: map[string]string{"X-Delete": "before"}})
	ignore = true
	deleted, err := prepareDelete([]DeleteOption{WithDeleteIgnoreMissing(true), deletion})
	if err != nil || *deleted.IgnoreMissing {
		t.Fatalf("delete snapshot %+v %v", deleted, err)
	}
	bulk, err := prepareDeleteAll([]DeleteAllOption{WithDeleteAllHeader("X-Removed", "yes"), WithDeleteAllOpts(DeleteAllOpts{}), WithDeleteAllHeader("X-Bulk", "last")})
	if err != nil || bulk.Headers["X-Removed"] != "" || bulk.Headers["X-Bulk"] != "last" {
		t.Fatalf("bulk replacement %+v %v", bulk, err)
	}
	inputHeaders := map[string]string{"x-snapshot": "before"}
	get := WithGetHeaders(inputHeaders)
	inputHeaders["x-snapshot"] = "after"
	fetched, err := prepareGet([]GetOption{get, WithGetHeader("X-Snapshot", "last")})
	if err != nil || fetched.Headers["X-Snapshot"] != "last" {
		t.Fatalf("header lastwins %+v %v", fetched, err)
	}
	listed, err := prepareList([]ListOption{WithListMaxItems(9), WithListOpts(ListOpts{MaxItems: 2}), WithListMaxItems(3)})
	if err != nil || listed.MaxItems != 3 {
		t.Fatalf("cap lastwins %+v %v", listed, err)
	}
	propertyInput := map[string]json.RawMessage{"x": json.RawMessage(`{"arbitrary":true}`)}
	propertyOption := WithCreateProperties(propertyInput)
	propertyInput["x"][2] = 'Z'
	requiredInput := []string{"literal,entry"}
	requiredOption := WithCreateRequired(requiredInput)
	requiredInput[0] = "mutated"
	captured, err := prepareCreate([]CreateOption{propertyOption, requiredOption})
	if err != nil || string(captured.Properties["x"]) != `{"arbitrary":true}` || captured.Required[0] != "literal,entry" {
		t.Fatalf("field helper snapshots %+v %v", captured, err)
	}
	var zero GetOpts
	if err := WithGetHeader("X-Direct", "yes")(&zero); err != nil || zero.Headers["X-Direct"] != "yes" {
		t.Fatalf("direct zero helper %+v %v", zero, err)
	}
}

func TestMetadefObjectsOptionsCallbackDeepOwnership(t *testing.T) {
	var retained *CreateOpts
	callbacks := 0
	first := CreateOption(func(o *CreateOpts) error {
		callbacks++
		if o.Headers == nil {
			t.Fatal("nil initial Headers")
		}
		o.Description = objectOptionPointer("before")
		o.Properties = map[string]json.RawMessage{"x": json.RawMessage(`{"n":1}`)}
		o.Required = []string{"before"}
		o.Headers["X-Handle"] = "before"
		retained = o
		return nil
	})
	second := CreateOption(func(o *CreateOpts) error {
		callbacks++
		*retained.Description = "old"
		retained.Properties["x"][5] = '2'
		retained.Required[0] = "old"
		retained.Headers["X-Handle"] = "old"
		if *o.Description != "before" || string(o.Properties["x"]) != `{"n":1}` || o.Required[0] != "before" || o.Headers["X-Handle"] != "before" {
			t.Fatal("earlier callback aliases new config")
		}
		return nil
	})
	config, err := prepareCreate([]CreateOption{first, second})
	if err != nil || callbacks != 2 {
		t.Fatalf("callbacks%d %v", callbacks, err)
	}
	*retained.Description = "closed"
	retained.Properties["x"][5] = '3'
	retained.Required[0] = "closed"
	retained.Headers["X-Handle"] = "closed"
	if *config.Description != "before" || string(config.Properties["x"]) != `{"n":1}` || config.Required[0] != "before" || config.Headers["X-Handle"] != "before" {
		t.Fatal("closed handle changed prepared value")
	}
	requests := 0
	scope := objectCoreScope(t, objectCoreClient(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Header.Get("X-Chosen") != "yes" {
			t.Fatalf("chosen header %#v", req.Header)
		}
		return objectCoreJSON(req, 200, `{}`), nil
	}), "parent")
	var options []GetOption
	options = []GetOption{func(*GetOpts) error {
		options[1] = func(*GetOpts) error { return errors.New("changed caller slice") }
		return nil
	}, WithGetHeader("X-Chosen", "yes")}
	if _, err := scope.Get(context.Background(), "child", options...); err != nil || requests != 1 {
		t.Fatalf("owned method slice %v requests%d", err, requests)
	}
	callbacks, requests = 0, 0
	scope = objectCoreScope(t, objectCoreClient(func(req *http.Request) (*http.Response, error) {
		requests++
		return objectCoreJSON(req, 200, `{"objects":[{},{}]}`), nil
	}), "parent")
	listOptions := []ListOption{func(*ListOpts) error { callbacks++; return nil }, WithListMaxItems(1)}
	seq := scope.List(context.Background(), listOptions...)
	listOptions[1] = func(*ListOpts) error { return errors.New("mutated lazy slice") }
	if callbacks != 0 || requests != 0 {
		t.Fatal("iterator eager")
	}
	for iteration := 0; iteration < 2; iteration++ {
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
	if callbacks != 2 || requests != 2 {
		t.Fatalf("lazy reuse callbacks%d requests%d", callbacks, requests)
	}
}

func TestMetadefObjectsOptionsValidationAndDefaults(t *testing.T) {
	created, err := prepareCreate(nil)
	if err != nil || created.Headers == nil || created.Description != nil || created.Properties != nil || created.Required != nil {
		t.Fatalf("create default %+v %v", created, err)
	}
	updated, err := prepareUpdate(nil)
	if err != nil || updated.Name != nil || updated.Description != nil || updated.Properties != nil || updated.Required != nil {
		t.Fatalf("update default %+v %v", updated, err)
	}
	fetched, err := prepareGet(nil)
	if err != nil || fetched.Headers == nil {
		t.Fatalf("get %+v %v", fetched, err)
	}
	deleted, err := prepareDelete(nil)
	if err != nil || deleted.IgnoreMissing != nil {
		t.Fatalf("delete %+v %v", deleted, err)
	}
	bulk, err := prepareDeleteAll(nil)
	if err != nil || bulk.Headers == nil {
		t.Fatalf("bulk %+v %v", bulk, err)
	}
	listed, err := prepareList(nil)
	if err != nil || listed.MaxItems != 0 {
		t.Fatalf("list %+v %v", listed, err)
	}
	if err := literal(strings.Repeat("界", 80)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"OS::Compute::Libvirt", " name ", "e\u0301", "é"} {
		if err := literal(name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := prepareCreate([]CreateOption{WithCreateDescription(strings.Repeat("界", 1000) + "\n"), WithCreateProperties(map[string]json.RawMessage{"": json.RawMessage(`{}`), "comma,newline\n": json.RawMessage(`{"nonstandard":true}`)}), WithCreateRequired([]string{"", "x,y", "line\nnext", "x,y"})}); err != nil {
		t.Fatalf("literal payload overrestricted %v", err)
	}
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`[]`), json.RawMessage(`false`), json.RawMessage(`"string"`), json.RawMessage(`1`), json.RawMessage(`{`), json.RawMessage("{\"x\":\"\xff\"}")} {
		if _, err := prepareCreate([]CreateOption{WithCreateProperties(map[string]json.RawMessage{"x": raw})}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("bad property %q %v", raw, err)
		}
	}
	for _, key := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "Accept", "Content-Type", "OpenStack-API-Version", "X-OpenStack-Glance-Api-Version", "X-OpenStack-Image-Size"} {
		if _, err := prepareGet([]GetOption{WithGetHeader(key, "value")}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("protected header %s %v", key, err)
		}
	}
	for _, headers := range []map[string]string{{"X-A": "one", "x-a": "two"}, {"Bad Key": "x"}, {"X-A": "a\rb"}, {"X-A": string([]byte{0xff})}} {
		if _, err := prepareDeleteAll([]DeleteAllOption{WithDeleteAllHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("bad headers %#v %v", headers, err)
		}
	}
	nilChecks := []func() error{func() error { _, e := prepareCreate([]CreateOption{nil}); return e }, func() error { _, e := prepareUpdate([]UpdateOption{nil}); return e }, func() error { _, e := prepareGet([]GetOption{nil}); return e }, func() error { _, e := prepareDelete([]DeleteOption{nil}); return e }, func() error { _, e := prepareDeleteAll([]DeleteAllOption{nil}); return e }, func() error { _, e := prepareList([]ListOption{nil}); return e }}
	for i, check := range nilChecks {
		if err := check(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil family%d %v", i, err)
		}
	}
	cause := errors.New("callback-cause")
	if _, err := prepareCreate([]CreateOption{func(*CreateOpts) error { return cause }}); err != cause {
		t.Fatalf("lost cause identity %v", err)
	}
	explicit, err := prepareCreate([]CreateOption{WithCreateDescription(""), WithCreateProperties(map[string]json.RawMessage{}), WithCreateRequired([]string{})})
	if err != nil || !reflect.DeepEqual(objectBody("child", explicit.Description, explicit.Properties, explicit.Required), map[string]any{"name": "child", "description": "", "properties": map[string]json.RawMessage{}, "required": []string{}}) {
		t.Fatalf("explicit nil/empty %+v %v", explicit, err)
	}
	if _, err := validateHeaders(map[string]string{"Accept": "custom", "OpenStack-API-Version": "image 2.9"}, true, "2.9"); err != nil {
		t.Fatal(err)
	}
}

func TestMetadefObjectsOptionsParallelReusableHelpers(t *testing.T) {
	var calls atomic.Int32
	scope := objectCoreScope(t, objectCoreClient(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		var body map[string]json.RawMessage
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			return nil, err
		}
		if req.Header.Get("X-Reuse") != "owned" || string(body["description"]) != `"before"` || string(body["properties"]) != `{"x":{"n":1}}` || string(body["required"]) != `["x"]` {
			return nil, errors.New("helper snapshot changed")
		}
		return objectCoreJSON(req, 201, `{"name":"server","properties":{"x":1},"required":[],"unknown":9007199254740993}`), nil
	}), "parent")
	input := CreateOpts{Headers: map[string]string{"X-Reuse": "owned"}, Description: objectOptionPointer("before"), Properties: map[string]json.RawMessage{"x": json.RawMessage(`{"n":1}`)}, Required: []string{"x"}}
	option := WithCreateOpts(input)
	input.Headers["X-Reuse"] = "mutated"
	*input.Description = "mutated"
	input.Properties["x"][5] = '2'
	input.Required[0] = "mutated"
	var wg sync.WaitGroup
	failures := make(chan error, 24)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := scope.Create(context.Background(), "child", option)
			if err != nil {
				failures <- err
				return
			}
			value.Properties["x"][0] = '8'
			value.Body["unknown"][0] = '8'
			value.Header.Set("X-Proof", "local")
			*value.Name = "local"
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if calls.Load() != 24 {
		t.Fatalf("requests%d", calls.Load())
	}
}
