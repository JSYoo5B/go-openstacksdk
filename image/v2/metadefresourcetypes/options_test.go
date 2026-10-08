package metadefresourcetypes

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

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func rtOptionPointer[T any](value T) *T { return &value }

func TestMetadefResourceTypesOptionsSnapshotsReplacementAndLastWins(t *testing.T) {
	prefix, target := "before", "image"
	headers := map[string]string{"X-Caller": "before"}
	option := WithCreateOpts(CreateOpts{Headers: headers, Prefix: &prefix, PropertiesTarget: &target})
	prefix, target = "changed", "changed"
	headers["X-Caller"] = "changed"
	first, err := prepareCreate([]CreateOption{option})
	if err != nil {
		t.Fatal(err)
	}
	if *first.Prefix != "before" || *first.PropertiesTarget != "image" || first.Headers["X-Caller"] != "before" {
		t.Fatalf("helper capture %+v", first)
	}
	*first.Prefix = "consumer"
	first.Headers["X-Caller"] = "consumer"
	second, err := prepareCreate([]CreateOption{option})
	if err != nil || *second.Prefix != "before" || second.Headers["X-Caller"] != "before" {
		t.Fatal("reused helper changed", err)
	}
	value, err := prepareCreate([]CreateOption{WithCreatePrefix("one"), WithCreatePropertiesTarget("two"), WithCreateHeader("X-Caller", "old"), WithCreateOpts(CreateOpts{}), WithCreateHeader("x-caller", "final"), WithCreatePrefix("")})
	if err != nil || value.Prefix == nil || *value.Prefix != "" || value.PropertiesTarget != nil || !reflect.DeepEqual(value.Headers, map[string]string{"X-Caller": "final"}) {
		t.Fatalf("whole replacement/last wins %v %+v", err, value)
	}
	mergedHeaders := map[string]string{"X-First": "one", "X-Caller": "first"}
	merged := WithCreateHeaders(mergedHeaders)
	mergedHeaders["X-First"] = "mutated"
	value, err = prepareCreate([]CreateOption{merged, WithCreateHeader("x-caller", "last")})
	if err != nil || value.Headers["X-First"] != "one" || value.Headers["X-Caller"] != "last" {
		t.Fatal(err, value)
	}
	ignore := false
	deleteHeaders := map[string]string{"X-Caller": "original"}
	deleteOption := WithDeleteOpts(DeleteOpts{Headers: deleteHeaders, IgnoreMissing: &ignore})
	ignore = true
	deleteHeaders["X-Caller"] = "changed"
	deletion, err := prepareDelete([]DeleteOption{deleteOption})
	if err != nil || *deletion.IgnoreMissing || deletion.Headers["X-Caller"] != "original" {
		t.Fatal(err, deletion)
	}
	deletion, err = prepareDelete([]DeleteOption{deleteOption, WithDeleteOpts(DeleteOpts{}), WithDeleteHeader("X-Final", "yes")})
	if err != nil || deletion.IgnoreMissing != nil || !reflect.DeepEqual(deletion.Headers, map[string]string{"X-Final": "yes"}) {
		t.Fatal(err, deletion)
	}
	listHeaders := map[string]string{"X-Caller": "original"}
	listOption := WithListOpts(ListOpts{Headers: listHeaders, MaxItems: 3})
	listHeaders["X-Caller"] = "changed"
	listing, err := prepareList([]ListOption{listOption, WithListMaxItems(1), WithListHeader("x-caller", "final")})
	if err != nil || listing.MaxItems != 1 || listing.Headers["X-Caller"] != "final" {
		t.Fatal(err, listing)
	}
	listing, err = prepareList([]ListOption{listOption, WithListOpts(ListOpts{})})
	if err != nil || listing.MaxItems != 0 || len(listing.Headers) != 0 {
		t.Fatal(err, listing)
	}
	// The map helpers for every concrete family own their inputs too.
	for _, which := range []string{"delete", "list"} {
		source := map[string]string{"X-Source": "owned"}
		if which == "delete" {
			o := WithDeleteHeaders(source)
			source["X-Source"] = "changed"
			v, e := prepareDelete([]DeleteOption{o})
			if e != nil || v.Headers["X-Source"] != "owned" {
				t.Fatal(e, v)
			}
		} else {
			o := WithListHeaders(source)
			source["X-Source"] = "changed"
			v, e := prepareList([]ListOption{o})
			if e != nil || v.Headers["X-Source"] != "owned" {
				t.Fatal(e, v)
			}
		}
	}
}

func TestMetadefResourceTypesOptionsCallbacksAndRequestOwnership(t *testing.T) {
	calls, callbacks := 0, 0
	var retained *CreateOpts
	var retainedDelete *DeleteOpts
	var retainedList *ListOpts
	client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		switch req.Method {
		case "POST":
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body, map[string]string{"name": "child", "prefix": "owned", "properties_target": "image"}) || req.Header.Get("X-Caller") != "owned" {
				t.Fatalf("retained callback changed request %+v %v", body, req.Header)
			}
			return rtCoreJSON(req, 201, `{"name":"server"}`), nil
		case "DELETE":
			if req.Header.Get("X-Caller") != "owned" {
				t.Fatal("delete pointer retained")
			}
			return rtCoreJSON(req, 204, ""), nil
		case "GET":
			if req.Header.Get("X-Caller") != "owned" {
				t.Fatal("list config retained")
			}
			return rtCoreJSON(req, 200, `{"resource_types":[]}`), nil
		default:
			t.Fatal(req.Method)
			return nil, nil
		}
	})
	scope := rtCoreScope(t, client, "parent")
	first := CreateOption(func(c *CreateOpts) error {
		callbacks++
		if c.Headers == nil {
			t.Fatal("default maps not initialized")
		}
		c.Headers["X-Caller"] = "owned"
		c.Prefix = rtOptionPointer("owned")
		c.PropertiesTarget = rtOptionPointer("image")
		retained = c
		return nil
	})
	second := CreateOption(func(c *CreateOpts) error {
		callbacks++
		retained.Headers["X-Caller"] = "changed"
		*retained.Prefix = "changed"
		retained.PropertiesTarget = nil
		if *c.Prefix != "owned" || *c.PropertiesTarget != "image" {
			t.Fatal("callbacks shared pointers")
		}
		return nil
	})
	options := []CreateOption{first, second}
	if _, err := scope.Create(context.Background(), "child", options...); err != nil {
		t.Fatal(err)
	}
	// A copied option slice permits reusing a lazy sequence after caller mutation.
	listingOptions := []ListOption{func(c *ListOpts) error { callbacks++; c.Headers["X-Caller"] = "owned"; retainedList = c; return nil }, func(c *ListOpts) error {
		callbacks++
		retainedList.Headers["X-Caller"] = "changed"
		retainedList.MaxItems = 99
		if c.MaxItems != 0 {
			t.Fatal("retained list config")
		}
		return nil
	}}
	seq := New(client).List(context.Background(), listingOptions...)
	listingOptions[0] = nil
	for _, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := scope.Delete(context.Background(), "child", func(c *DeleteOpts) error {
		callbacks++
		c.Headers["X-Caller"] = "owned"
		c.IgnoreMissing = rtOptionPointer(false)
		retainedDelete = c
		return nil
	}, func(c *DeleteOpts) error {
		callbacks++
		retainedDelete.Headers["X-Caller"] = "changed"
		*retainedDelete.IgnoreMissing = true
		if *c.IgnoreMissing {
			t.Fatal("retained delete config")
		}
		return nil
	})
	if err != nil || calls != 3 || callbacks != 6 {
		t.Fatal(err, calls, callbacks)
	}
	for _, pair := range []struct {
		source string
		target string
	}{{"X-Caller", "x-caller"}, {"x-caller", "X-Caller"}} {
		v, err := prepareCreate([]CreateOption{WithCreateHeader(pair.source, "one"), WithCreateHeader(pair.target, "two")})
		if err != nil || len(v.Headers) != 1 || v.Headers["X-Caller"] != "two" {
			t.Fatal(err, v)
		}
	}
}

func TestMetadefResourceTypesOptionsValidationAndDefaults(t *testing.T) {
	for _, field := range []string{"prefix", "target"} {
		for _, text := range []string{"", "line\n\x00", strings.Repeat("界", 80)} {
			var option CreateOption
			if field == "prefix" {
				option = WithCreatePrefix(text)
			} else {
				option = WithCreatePropertiesTarget(text)
			}
			v, err := prepareCreate([]CreateOption{option})
			if err != nil {
				t.Fatal("valid JSON literal constrained", field, err)
			}
			ptr := v.Prefix
			if field == "target" {
				ptr = v.PropertiesTarget
			}
			if ptr == nil || *ptr != text {
				t.Fatal("literal changed")
			}
		}
		for _, text := range []string{strings.Repeat("界", 81), string([]byte{0xff})} {
			var option CreateOption
			if field == "prefix" {
				option = WithCreatePrefix(text)
			} else {
				option = WithCreatePropertiesTarget(text)
			}
			if _, err := prepareCreate([]CreateOption{option}); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		}
	}
	creation, err := prepareCreate(nil)
	if err != nil || creation.Prefix != nil || creation.PropertiesTarget != nil || creation.Headers == nil {
		t.Fatal(err, creation)
	}
	deletion, err := prepareDelete(nil)
	if err != nil || deletion.IgnoreMissing != nil || deletion.Headers == nil {
		t.Fatal(err, deletion)
	}
	listing, err := prepareList(nil)
	if err != nil || listing.MaxItems != 0 || listing.Headers == nil {
		t.Fatal(err, listing)
	}
	for _, headers := range []map[string]string{{"X-Caller": "one", "x-caller": "two"}, {"bad key": "x"}, {"X-Caller": "line\n"}, {"X-Caller": string([]byte{0xff})}, {"Accept": "json"}, {"OpenStack-API-Version": "image 2.2"}, {"Content-Length": "0"}, {"Authorization": "x"}} {
		if _, err := prepareCreate([]CreateOption{WithCreateHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, err)
		}
		if _, err := prepareDelete([]DeleteOption{WithDeleteOpts(DeleteOpts{Headers: headers})}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, err)
		}
		if _, err := prepareList([]ListOption{WithListHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, err)
		}
	}
	if _, err := prepareList([]ListOption{WithListMaxItems(-1)}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, which := range []string{"create", "delete", "list"} {
		cause := errors.New(which)
		var err error
		switch which {
		case "create":
			_, err = prepareCreate([]CreateOption{func(*CreateOpts) error { return cause }})
		case "delete":
			_, err = prepareDelete([]DeleteOption{func(*DeleteOpts) error { return cause }})
		case "list":
			_, err = prepareList([]ListOption{func(*ListOpts) error { return cause }})
		}
		if err != cause {
			t.Fatal("callback cause identity lost", err)
		}
	}
}

func TestMetadefResourceTypesOptionsParallelReusableHelpers(t *testing.T) {
	var calls atomic.Int64
	client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.Header.Get("X-Caller") != "shared" {
			t.Error("header helper snapshot lost")
		}
		switch req.Method {
		case "POST":
			return rtCoreJSON(req, 201, `{"name":"server"}`), nil
		case "DELETE":
			return rtCoreJSON(req, 204, ""), nil
		case "GET":
			return rtCoreJSON(req, 200, `{"resource_type_associations":[{"name":"one"}]}`), nil
		}
		return nil, errors.New("unexpected method")
	})
	scope := rtCoreScope(t, client, "parent")
	headers := map[string]string{"X-Caller": "shared"}
	prefix := "hw_"
	create := WithCreateOpts(CreateOpts{Headers: headers, Prefix: &prefix})
	remove := WithDeleteOpts(DeleteOpts{Headers: headers, IgnoreMissing: rtOptionPointer(false)})
	list := WithListOpts(ListOpts{Headers: headers, MaxItems: 1})
	headers["X-Caller"] = "changed"
	prefix = "changed"
	var wg sync.WaitGroup
	for range 24 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := scope.Create(context.Background(), "child", create)
			if err != nil || v == nil || *v.Name != "server" {
				t.Error(err)
			}
			ack, err := scope.Delete(context.Background(), "child", remove)
			if err != nil || ack == nil {
				t.Error(err)
			}
			rows, err := scope.All(context.Background(), list)
			if err != nil || len(rows) != 1 {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 72 {
		t.Fatal(calls.Load())
	}
}
