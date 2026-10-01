package senlin_test

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func bodyFilterDescriptor() senlin.BodyFilterSpec {
	return senlin.BodyFilterSpec{Namespace: "test.local_filters", Fields: map[string]string{
		"id": "id", "name": "name", "schema": "schema", "user": "user", "user_id": "user",
	}}
}

func TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps(t *testing.T) {
	descriptor := bodyFilterDescriptor()
	large := json.RawMessage(`9007199254740993`)
	input := map[string]any{"limit": large, "nested": []any{false, "kept"}}
	option := senlin.WithBodyFilter[senlin.ListOpts](descriptor, "schema", input)
	input["nested"].([]any)[1] = "changed"
	large[0] = '1'
	delete(descriptor.Fields, "schema")
	for iteration := range 2 {
		cfg, err := request.Apply(senlin.ListOpts{}, option,
			senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "user_id", "earlier"),
			senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "user", "last"))
		if err != nil {
			t.Fatal(err)
		}
		filters, err := senlin.PrepareBodyFilters(cfg, bodyFilterDescriptor())
		if err != nil || string(filters["schema"]) != `{"limit":9007199254740993,"nested":[false,"kept"]}` || string(filters["user"]) != `"last"` || len(filters) != 2 {
			t.Fatal(iteration, filters, err)
		}
		stored := cfg.Arguments[bodyFilterDescriptor().Namespace].(map[string]json.RawMessage)
		filters["schema"][0] = '['
		if stored["schema"][0] != '{' {
			t.Fatal("prepared raw bytes alias option config")
		}
		stored["user"][1] = 'X'
		if string(filters["user"]) != `"last"` {
			t.Fatal("config raw bytes alias prepared filters")
		}
	}
	previous := map[string]json.RawMessage{"id": json.RawMessage(`"original"`)}
	cfg, err := request.Apply(senlin.ListOpts{}, request.WithArgument[senlin.ListOpts](bodyFilterDescriptor().Namespace, previous), option)
	if err != nil {
		t.Fatal(err)
	}
	stored := cfg.Arguments[bodyFilterDescriptor().Namespace].(map[string]json.RawMessage)
	stored["id"][1] = 'X'
	if len(previous) != 1 || string(previous["id"]) != `"original"` {
		t.Fatal("applying option mutated caller argument map", previous)
	}
}

func TestBodyFilterPrepareRejectsForeignMalformedAndAmbiguousOptions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		option senlin.ListOption
	}{
		{"foreign-namespace", request.WithArgument[senlin.ListOpts]("other.local_filters", map[string]json.RawMessage{"id": json.RawMessage(`"one"`)})},
		{"wrong-type", request.WithArgument[senlin.ListOpts](bodyFilterDescriptor().Namespace, map[string]any{"id": "one"})},
		{"unknown-field", request.WithArgument[senlin.ListOpts](bodyFilterDescriptor().Namespace, map[string]json.RawMessage{"unknown": json.RawMessage(`null`)})},
		{"malformed", request.WithArgument[senlin.ListOpts](bodyFilterDescriptor().Namespace, map[string]json.RawMessage{"id": json.RawMessage(`"one`)})},
		{"absent-JSON", request.WithArgument[senlin.ListOpts](bodyFilterDescriptor().Namespace, map[string]json.RawMessage{"id": nil})},
		{"multiple-JSON-values", request.WithArgument[senlin.ListOpts](bodyFilterDescriptor().Namespace, map[string]json.RawMessage{"id": json.RawMessage(`null false`)})},
		{"ambiguous-alias", request.WithArgument[senlin.ListOpts](bodyFilterDescriptor().Namespace, map[string]json.RawMessage{"user": json.RawMessage(`"one"`), "user_id": json.RawMessage(`"two"`)})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := request.Apply(senlin.ListOpts{}, tc.option)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := senlin.PrepareBodyFilters(cfg, bodyFilterDescriptor()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("invalid custom input accepted", err)
			}
		})
	}
	for _, descriptor := range []senlin.BodyFilterSpec{
		{Fields: map[string]string{"id": "id"}},
		{Namespace: "filters", Fields: map[string]string{"alias": "missing"}},
	} {
		if _, err := request.Apply(senlin.ListOpts{}, senlin.WithBodyFilter[senlin.ListOpts](descriptor, "id", "one")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("invalid descriptor accepted", err)
		}
	}
	for _, key := range []string{"user", "user_id", "schema"} {
		if err := senlin.RejectBodyFilterQuery(url.Values{key: {"value"}}, bodyFilterDescriptor()); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("Body field remained in wire query", key, err)
		}
	}
	if err := senlin.RejectBodyFilterQuery(url.Values{"vendor": {"kept"}, "limit": {"2"}}, bodyFilterDescriptor()); err != nil {
		t.Fatal("unrelated extension or concrete pagination rejected", err)
	}
	for _, option := range []senlin.ListOption{
		senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "unknown", true),
		senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "id", func() {}),
		senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "id", json.RawMessage(`false true`)),
	} {
		if _, err := request.Apply(senlin.ListOpts{}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("unserializable or unknown filter accepted", err)
		}
	}
}

type filterRow struct {
	resource.Metadata
	ID   string `json:"id"`
	Name string `json:"name"`
}

func filterCollectionSpec(client *gophercloud.ServiceClient) rest.CollectionSpec[filterRow] {
	return rest.CollectionSpec[filterRow]{Client: client, Path: "items", Kind: "filter-test", PluralKey: "items",
		Metadata: func(value *filterRow) *resource.Metadata { return &value.Metadata },
		Validate: func(ctx context.Context) error { return senlin.Validate(ctx, client) },
		Paging:   rest.PagePolicy[filterRow]{MaxItemsLimitHint: true, StopOnEmptyPage: true}}
}

func collectBodyFiltered(t *testing.T, stream iter.Seq2[*filterRow, error]) ([]string, error) {
	t.Helper()
	ids := make([]string, 0)
	var failure error
	for value, err := range stream {
		if err != nil {
			if value != nil || failure != nil {
				t.Fatal("expected one nil-row terminal error", value, err, failure)
			}
			failure = err
			continue
		}
		if value == nil || failure != nil {
			t.Fatal(value, failure)
		}
		ids = append(ids, value.ID)
	}
	return ids, failure
}

func TestBodyFilteredListPreflightIsLazyAndRejectsWireOrNamespaceBypass(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/items", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"items":[]}`)
	})
	for _, option := range []senlin.ListOption{
		request.WithQuery[senlin.ListOpts]("schema", "wire"),
		request.WithQuery[senlin.ListOpts]("user_id", "wire"),
		request.WithArgument[senlin.ListOpts]("other.local_filters", map[string]json.RawMessage{}),
		request.WithArgument[senlin.ListOpts](bodyFilterDescriptor().Namespace, map[string]json.RawMessage{"id": nil}),
		request.WithField[senlin.ListOpts]("id", "unsupported"),
		request.WithHeader[senlin.ListOpts]("X-Example", "unsupported"),
		senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "unknown", "value"),
		senlin.WithMaxItems(-1),
		nil,
	} {
		stream := senlin.ListWithBodyFilters(context.Background(), filterCollectionSpec(cloud.Client("clustering", "/senlin")), bodyFilterDescriptor(), option)
		if calls.Load() != 0 {
			t.Fatal("eager construction", calls.Load())
		}
		rows, err := collectBodyFiltered(t, stream)
		var operation *resource.OperationError
		if len(rows) != 0 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "List" || calls.Load() != 0 {
			t.Fatal(rows, err, calls.Load())
		}
	}
	cfg, err := request.Apply(senlin.ListOpts{}, senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "id", "value"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := senlin.Query(cfg); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("plain Query unexpectedly accepts Body filter Arguments", err)
	}
	if _, err := senlin.Query(cfg, bodyFilterDescriptor().Namespace); err != nil {
		t.Fatal("owned Query argument rejected", err)
	}
}

func TestBodyFilteredListCapUsesValidatedRawRowsAndSnapshotsEachIteration(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	body := `{"items":[{"id":"one","schema":{"n":9007199254740993}},{"id":"two","schema":{"n":9007199254740994}},false],"links":false}`
	cloud.Mux.HandleFunc("GET /senlin/items", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("vendor") != "kept" || r.URL.Query().Has("schema") {
			t.Error("raw filter leaked or controls lost", r.URL)
		}
		testcloud.JSON(w, 200, body)
	})
	input := map[string]any{"n": json.Number("9007199254740994")}
	options := []senlin.ListOption{senlin.WithMaxItems(2), request.WithQuery[senlin.ListOpts]("vendor", "kept"), senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "schema", input)}
	descriptor := bodyFilterDescriptor()
	stream := senlin.ListWithBodyFilters(context.Background(), filterCollectionSpec(cloud.Client("clustering", "/senlin")), descriptor, options...)
	input["n"] = json.Number("9007199254740993")
	options[0], options[2] = nil, nil
	delete(descriptor.Fields, "schema")
	for range 2 {
		rows, err := collectBodyFiltered(t, stream)
		if err != nil || !reflect.DeepEqual(rows, []string{"two"}) {
			t.Fatal("filter/counter/descriptor snapshot changed", rows, err)
		}
	}
	rows, err := collectBodyFiltered(t, senlin.ListWithBodyFilters(context.Background(), filterCollectionSpec(cloud.Client("clustering", "/senlin")), bodyFilterDescriptor(),
		request.WithOptions(senlin.ListOpts{Limit: 2}), request.WithQuery[senlin.ListOpts]("vendor", "kept"), senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "schema", input)))
	var proof *resource.ResponseError
	if !reflect.DeepEqual(rows, []string{"one"}) || !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 200 || calls.Load() != 3 {
		t.Fatal("uncapped trailing decode error lost whole page", rows, err, proof, calls.Load())
	}
}

func TestBodyFilteredListCannotHideIdentityFailureAndPreservesTerminalCancellation(t *testing.T) {
	for _, mode := range []string{"identity", "cancel-at-filtered-cap", "consumer-break"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := `{"items":[{"id":"one","name":"other"},{"id":"two","name":"wanted"}],"links":false}`
			cloud.Mux.HandleFunc("GET /senlin/items", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-ID", "filter-proof")
				testcloud.JSON(w, 200, body)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			spec := filterCollectionSpec(cloud.Client("clustering", "/senlin"))
			identityFailure := errors.New("wrong fixed parent")
			spec.ValidateItem = func(value *filterRow) error {
				if mode == "identity" {
					return identityFailure
				}
				if mode == "cancel-at-filtered-cap" {
					cancel()
				}
				return nil
			}
			filter := senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "name", "wanted")
			if mode == "consumer-break" {
				filter = senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "name", "other")
			}
			stream := senlin.ListWithBodyFilters(ctx, spec, bodyFilterDescriptor(), filter, senlin.WithMaxItems(1))
			if mode == "consumer-break" {
				seen := 0
				stream(func(value *filterRow, err error) bool {
					seen++
					if value == nil || err != nil || value.ID != "one" {
						t.Fatal(value, err)
					}
					cancel()
					return false
				})
				if seen != 1 || calls.Load() != 1 {
					t.Fatal(seen, calls.Load())
				}
				return
			}
			rows, err := collectBodyFiltered(t, stream)
			wantErr := identityFailure
			if mode == "cancel-at-filtered-cap" {
				wantErr = context.Canceled
			}
			var proof *resource.ResponseError
			if len(rows) != 0 || !errors.Is(err, wantErr) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-ID") != "filter-proof" || string(proof.Body) != body || calls.Load() != 1 {
				t.Fatal(rows, err, proof, calls.Load())
			}
		})
	}
}
