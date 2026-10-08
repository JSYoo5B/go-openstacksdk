package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCollectionValidateItemPreservesSingularHTTPResponse(t *testing.T) {
	const body = `{"item":{"id":"wrong-parent","name":"binding","owner":"other","vendor":9007199254740993}}`
	var calls, validations atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/v1/items/binding" {
			t.Errorf("request=%s %s", r.Method, r.URL)
		}
		writeList(w, body)
	})
	cause := errors.New("response is outside the selected parent")
	spec.ValidateItem = func(value *listItem) error {
		validations.Add(1)
		if value.ID != "wrong-parent" || value.StatusCode != 200 || value.Header.Get("X-Page") != "kept" || string(value.Body["vendor"]) != "9007199254740993" {
			t.Fatalf("validation did not receive decoded evidence: %+v", value)
		}
		// Mutating model evidence must not corrupt the retained wire response.
		value.Header.Set("X-Page", "changed")
		value.Body["vendor"][0] = '0'
		return cause
	}
	value, err := Collection(spec).Get(context.Background(), "binding")
	var response *resource.ResponseError
	if value != nil || !errors.Is(err, cause) || !errors.As(err, &response) || response.StatusCode != 200 || response.Header.Get("X-Page") != "kept" || string(response.Body) != body || calls.Load() != 1 || validations.Load() != 1 {
		t.Fatalf("value=%+v error=%v response=%+v calls/validation=%d/%d", value, err, response, calls.Load(), validations.Load())
	}
}

func TestListValidateItemRejectsRowWithWholePageEvidence(t *testing.T) {
	const body = `{"items":[{"id":"first","owner":"selected"},{"id":"second","owner":"other"}],"next":"?marker=unvisited"}`
	var calls, validations atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		writeList(w, body)
	})
	cause := errors.New("wrong parent")
	spec.ValidateItem = func(value *listItem) error {
		validations.Add(1)
		if string(value.Body["owner"]) != `"selected"` {
			return fmt.Errorf("binding %s: %w", value.ID, cause)
		}
		return nil
	}
	values, err := collectList(context.Background(), spec, nil)
	var response *resource.ResponseError
	if len(values) != 1 || values[0].ID != "first" || !errors.Is(err, cause) || !errors.As(err, &response) || response.StatusCode != 200 || response.Header.Get("X-Page") != "kept" || string(response.Body) != body || calls.Load() != 1 || validations.Load() != 2 {
		t.Fatalf("values=%v error=%v response=%+v calls/validation=%d/%d", values, err, response, calls.Load(), validations.Load())
	}
}

func TestListValidateItemKeepsConsumerBreakLazy(t *testing.T) {
	var calls, validations atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		writeList(w, `{"items":[{"id":"first"},{"id":"second"}],"next":"?marker=unvisited"}`)
	})
	spec.ValidateItem = func(value *listItem) error {
		validations.Add(1)
		if value.ID != "first" {
			return errors.New("unconsumed row was validated")
		}
		return nil
	}
	stream := List(context.Background(), spec, nil)
	if calls.Load() != 0 || validations.Load() != 0 {
		t.Fatal("creating stream was eager")
	}
	for value, err := range stream {
		if err != nil || value.ID != "first" {
			t.Fatalf("value=%v error=%v", value, err)
		}
		break
	}
	if calls.Load() != 1 || validations.Load() != 1 {
		t.Fatalf("calls/validation=%d/%d", calls.Load(), validations.Load())
	}
}

func TestListInitialQueryPolicyDoesNotRejectServerContinuation(t *testing.T) {
	var calls, initialChecks, pageChecks atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("owner") != "fixed" {
			t.Errorf("query validation changed initial filters: %s", r.URL)
		}
		if r.URL.Query().Get("marker") == "" {
			writeList(w, `{"items":[{"id":"first"}],"next":"?marker=server-token"}`)
		} else if r.URL.Query().Get("marker") == "server-token" {
			writeList(w, `{"items":[{"id":"second"}]}`)
		} else {
			t.Errorf("unexpected continuation: %s", r.URL)
		}
	})
	cause := errors.New("caller marker unsupported")
	spec.ValidateInitialQuery = func(_ context.Context, query url.Values) error {
		initialChecks.Add(1)
		if query.Has("marker") {
			return cause
		}
		query.Set("owner", "changed callback copy")
		return nil
	}
	spec.ValidateQuery = func(context.Context, url.Values) error { pageChecks.Add(1); return nil }
	stream := List(context.Background(), spec, url.Values{"owner": {"fixed"}, "marker": {"caller-token"}})
	if calls.Load() != 0 || initialChecks.Load() != 0 {
		t.Fatal("creating list eagerly validated or fetched")
	}
	for value, err := range stream {
		if value != nil || !errors.Is(err, cause) {
			t.Fatalf("value=%v error=%v", value, err)
		}
	}
	if calls.Load() != 0 || initialChecks.Load() != 1 || pageChecks.Load() != 0 {
		t.Fatalf("invalid query counts HTTP/initial/page=%d/%d/%d", calls.Load(), initialChecks.Load(), pageChecks.Load())
	}
	values, err := collectList(context.Background(), spec, url.Values{"owner": {"fixed"}})
	if err != nil || len(values) != 2 || values[1].ID != "second" || calls.Load() != 2 || initialChecks.Load() != 2 || pageChecks.Load() != 2 {
		t.Fatalf("values=%v error=%v HTTP/initial/page=%d/%d/%d", values, err, calls.Load(), initialChecks.Load(), pageChecks.Load())
	}
}
