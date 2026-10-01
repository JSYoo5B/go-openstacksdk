package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/clustering/v1/clusterpolicies"
	"gophercloudsdk/clustering/v1/events"
	"gophercloudsdk/clustering/v1/policytypes"
	"gophercloudsdk/clustering/v1/profiletypes"
	"gophercloudsdk/clustering/v1/services"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type clusteringBodyValue struct {
	key   string
	value any
}

type clusteringBodyAlias struct {
	alias, wire string
	value       any
}

type clusteringBodyRequest struct {
	filters []clusteringBodyValue
	query   []clusteringBodyValue
	maximum int
	first   bool
	foreign bool
	capture func(map[string]any)
}

func clusteringBodyOptions[T any](input clusteringBodyRequest,
	filter func(string, any) request.Option[T], maximum func(int) request.Option[T],
	paginated func(bool) request.Option[T], query func(string, string) request.Option[T]) []request.Option[T] {
	options := []request.Option[T]{maximum(input.maximum)}
	if input.first {
		options = append(options, paginated(false))
	}
	for _, value := range input.filters {
		options = append(options, filter(value.key, value.value))
	}
	for _, value := range input.query {
		options = append(options, query(value.key, fmt.Sprint(value.value)))
	}
	if input.foreign {
		options = append(options, func(config *request.Config[T]) error {
			config.Arguments["other.local_filters"] = map[string]json.RawMessage{"name": json.RawMessage(`"ignored"`)}
			return nil
		})
	}
	if input.capture != nil {
		options = append(options, func(config *request.Config[T]) error { input.capture(config.Arguments); return nil })
	}
	return options
}

type clusteringBodyFixture struct {
	path, plural, primary, alias, object, nullable, excluded string
	marker, binding                                          bool
	list                                                     func(*testing.T, context.Context, *gophercloud.ServiceClient, clusteringBodyRequest) iter.Seq2[*resource.Metadata, error]
}

func clusteringBodyFixtures() []clusteringBodyFixture {
	return []clusteringBodyFixture{
		{"actions", "actions", "user", "user_id", "inputs", "owner", "data", true, false,
			func(_ *testing.T, ctx context.Context, client *gophercloud.ServiceClient, input clusteringBodyRequest) iter.Seq2[*resource.Metadata, error] {
				options := clusteringBodyOptions(input, actions.WithListFilter, actions.WithListMaxItems, actions.WithListPaginated, actions.WithListQuery)
				return typedListMetadata(actions.New(client).List(ctx, options...), func(value *actions.Action) *resource.Metadata { return &value.Metadata })
			}},
		{"events", "events", "user", "user_id", "meta_data", "timestamp", "topic", true, false,
			func(_ *testing.T, ctx context.Context, client *gophercloud.ServiceClient, input clusteringBodyRequest) iter.Seq2[*resource.Metadata, error] {
				options := clusteringBodyOptions(input, events.WithListFilter, events.WithListMaxItems, events.WithListPaginated, events.WithListQuery)
				return typedListMetadata(events.New(client).List(ctx, options...), func(value *events.Event) *resource.Metadata { return &value.Metadata })
			}},
		{"services", "services", "host", "host", "", "disabled_reason", "topic", false, false,
			func(_ *testing.T, ctx context.Context, client *gophercloud.ServiceClient, input clusteringBodyRequest) iter.Seq2[*resource.Metadata, error] {
				options := clusteringBodyOptions(input, services.WithListFilter, services.WithListMaxItems, services.WithListPaginated, services.WithListQuery)
				return typedListMetadata(services.New(client).List(ctx, options...), func(value *services.Service) *resource.Metadata { return &value.Metadata })
			}},
		{"profile-types", "profile_types", "name", "name", "schema", "support_status", "version", false, false,
			func(_ *testing.T, ctx context.Context, client *gophercloud.ServiceClient, input clusteringBodyRequest) iter.Seq2[*resource.Metadata, error] {
				options := clusteringBodyOptions(input, profiletypes.WithListFilter, profiletypes.WithListMaxItems, profiletypes.WithListPaginated, profiletypes.WithListQuery)
				return typedListMetadata(profiletypes.New(client).List(ctx, options...), func(value *profiletypes.ProfileType) *resource.Metadata { return &value.Metadata })
			}},
		{"policy-types", "policy_types", "name", "name", "schema", "support_status", "version", false, false,
			func(_ *testing.T, ctx context.Context, client *gophercloud.ServiceClient, input clusteringBodyRequest) iter.Seq2[*resource.Metadata, error] {
				options := clusteringBodyOptions(input, policytypes.WithListFilter, policytypes.WithListMaxItems, policytypes.WithListPaginated, policytypes.WithListQuery)
				return typedListMetadata(policytypes.New(client).List(ctx, options...), func(value *policytypes.PolicyType) *resource.Metadata { return &value.Metadata })
			}},
		{"clusters/canonical-parent/policies", "cluster_policies", "policy_name", "policy_name", "data", "name", "cluster_id", false, true,
			func(t *testing.T, ctx context.Context, client *gophercloud.ServiceClient, input clusteringBodyRequest) iter.Seq2[*resource.Metadata, error] {
				scope, err := clusterpolicies.New(client).InCluster(ctx, resource.ID("input-parent"))
				if err != nil {
					t.Fatal(err)
				}
				if scope.ClusterID() != "canonical-parent" {
					t.Fatal(scope.ClusterID())
				}
				options := clusteringBodyOptions(input, clusterpolicies.WithListFilter, clusterpolicies.WithListMaxItems, clusterpolicies.WithListPaginated, clusterpolicies.WithListQuery)
				return typedListMetadata(scope.List(ctx, options...), func(value *clusterpolicies.ClusterPolicy) *resource.Metadata { return &value.Metadata })
			}},
	}
}

func clusteringBodyCloud(t *testing.T, fixture clusteringBodyFixture) (*testcloud.Cloud, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", resourceListPrefix)
	client.Microversion = "1.7"
	if fixture.binding {
		var parents atomic.Int32
		cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/clusters/input-parent", func(w http.ResponseWriter, _ *http.Request) {
			parents.Add(1)
			testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-parent"}}`)
		})
		t.Cleanup(func() {
			if parents.Load() != 1 {
				t.Error("fixed scope parent must be prepared once", parents.Load())
			}
		})
	}
	return cloud, client
}

func clusteringBodyRow(fixture clusteringBodyFixture, id string, fields map[string]any) map[string]any {
	row := map[string]any{"id": id, "name": id}
	if fixture.binding {
		row["policy_id"], row["cluster_id"] = "policy-"+id, "canonical-parent"
	}
	for key, value := range fields {
		row[key] = value
	}
	return row
}

func clusteringBodyJSON(t *testing.T, fixture clusteringBodyFixture, rows ...any) string {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{fixture.plural: rows})
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func clusteringBodyCollect(t *testing.T, values iter.Seq2[*resource.Metadata, error]) ([]*resource.Metadata, error) {
	t.Helper()
	rows := make([]*resource.Metadata, 0)
	var failure error
	for value, err := range values {
		if err != nil {
			if failure != nil || value != nil {
				t.Fatal("error must be terminal and have no row", failure, value, err)
			}
			failure = err
		} else {
			if failure != nil || value == nil {
				t.Fatal("invalid successful row or row after error", value, failure)
			}
			rows = append(rows, value)
		}
	}
	return rows, failure
}

func TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse(t *testing.T) {
	for _, fixture := range clusteringBodyFixtures() {
		t.Run(fixture.plural, func(t *testing.T) {
			cloud, client := clusteringBodyCloud(t, fixture)
			var calls atomic.Int32
			var captured map[string]any
			cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+fixture.path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if len(r.URL.Query()) != 1 || r.URL.Query().Get("vendor") != "kept" {
					t.Error("local Body filters leaked into wire query", r.URL)
				}
				for _, argument := range captured {
					if values, ok := argument.(map[string]json.RawMessage); ok {
						values[fixture.primary] = json.RawMessage(`"changed-in-http"`)
					}
				}
				w.Header().Set("X-Request-ID", "body-filter-snapshot")
				testcloud.JSON(w, 200, clusteringBodyJSON(t, fixture, clusteringBodyRow(fixture, "row", map[string]any{fixture.primary: "kept"})))
			})
			wanted := "kept"
			iterator := fixture.list(t, context.Background(), client, clusteringBodyRequest{
				filters: []clusteringBodyValue{{fixture.alias, "rejected"}, {fixture.primary, &wanted}},
				query:   []clusteringBodyValue{{"vendor", "kept"}}, first: true,
				capture: func(arguments map[string]any) { captured = arguments },
			})
			wanted = "changed-by-caller"
			if calls.Load() != 0 {
				t.Fatal("child list must be lazy")
			}
			for attempt := range 2 {
				rows, err := clusteringBodyCollect(t, iterator)
				if err != nil || len(rows) != 1 || string(rows[0].Body[fixture.primary]) != `"kept"` || rows[0].Header.Get("X-Request-ID") != "body-filter-snapshot" || calls.Load() != int32(attempt+1) {
					t.Fatal("filter snapshot/last-wins/reuse changed", rows, err, calls.Load())
				}
			}
		})
	}
}

func TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal(t *testing.T) {
	for _, fixture := range clusteringBodyFixtures() {
		cases := []clusteringBodyAlias{}
		switch fixture.plural {
		case "actions":
			cases = append(cases,
				clusteringBodyAlias{"target_id", "target", "target"},
				clusteringBodyAlias{"owner_id", "owner", "owner"},
				clusteringBodyAlias{"user_id", "user", "user"},
				clusteringBodyAlias{"project_id", "project", "project"},
				clusteringBodyAlias{"domain_id", "domain", "domain"},
				clusteringBodyAlias{"start_at", "start_time", json.Number("9007199254740993.125")},
				clusteringBodyAlias{"end_at", "end_time", json.Number("0")},
				clusteringBodyAlias{"status", "status", "READY"})
		case "events":
			cases = append(cases,
				clusteringBodyAlias{"generated_at", "timestamp", "2026-10-01T00:00:00"},
				clusteringBodyAlias{"obj_id", "oid", "object"},
				clusteringBodyAlias{"obj_name", "oname", "object-name"},
				clusteringBodyAlias{"obj_type", "otype", "NODE"},
				clusteringBodyAlias{"user_id", "user", "user"},
				clusteringBodyAlias{"project_id", "project", "project"},
				clusteringBodyAlias{"level", "level", json.Number("20")})
		case "services":
			cases = append(cases, clusteringBodyAlias{"name", "name", "service-name"}, clusteringBodyAlias{"status", "status", "UP"})
		case "profile_types", "policy_types":
			cases = append(cases, clusteringBodyAlias{"support_status", "support_status", map[string]any{"status": "SUPPORTED"}})
		case "cluster_policies":
			cases = append(cases, clusteringBodyAlias{"is_enabled", "enabled", false}, clusteringBodyAlias{"policy_type", "policy_type", "senlin.policy.scaling-1.0"})
		}
		for _, tc := range cases {
			t.Run(fixture.plural+"/"+tc.alias, func(t *testing.T) {
				cloud, client := clusteringBodyCloud(t, fixture)
				cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					if len(r.URL.Query()) != 0 {
						t.Error("explicit local query-capable filter was serialized", r.URL)
					}
					testcloud.JSON(w, 200, clusteringBodyJSON(t, fixture, clusteringBodyRow(fixture, "row", map[string]any{tc.wire: tc.value})))
				})
				rows, err := clusteringBodyCollect(t, fixture.list(t, context.Background(), client, clusteringBodyRequest{filters: []clusteringBodyValue{{tc.alias, tc.value}}, first: true}))
				if err != nil || len(rows) != 1 {
					t.Fatal("source alias did not match its raw wire field", rows, err)
				}
			})
		}
	}
}

func TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching(t *testing.T) {
	for _, fixture := range clusteringBodyFixtures() {
		for _, mode := range []string{"filtered-cap", "matching-cap", "single-page"} {
			t.Run(fixture.plural+"/"+mode, func(t *testing.T) {
				cloud, client := clusteringBodyCloud(t, fixture)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Has(fixture.primary) || r.URL.Query().Has(fixture.alias) || fixture.binding && r.URL.Query().Has("limit") {
						t.Error("filter or binding cap leaked", r.URL)
					}
					w.Header().Set("Link", `<https://foreign.example/unused>; rel="next"`)
					rows := []any{clusteringBodyRow(fixture, "first", map[string]any{fixture.primary: "miss"}), clusteringBodyRow(fixture, "second", map[string]any{fixture.primary: "kept"})}
					if mode != "single-page" {
						rows = append(rows, false)
					}
					testcloud.JSON(w, 200, clusteringBodyJSON(t, fixture, rows...))
				})
				input := clusteringBodyRequest{filters: []clusteringBodyValue{{fixture.alias, "kept"}}, maximum: 2}
				want := 1
				if mode == "filtered-cap" {
					input.maximum, want = 1, 0
				} else if mode == "single-page" {
					input.maximum, input.first = 0, true
				}
				rows, err := clusteringBodyCollect(t, fixture.list(t, context.Background(), client, input))
				if err != nil || len(rows) != want || calls.Load() != 1 {
					t.Fatal("filtered row did not consume raw cap, or unused row/link was processed", rows, err, calls.Load())
				}
			})
		}
	}
}

func TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent(t *testing.T) {
	for _, fixture := range clusteringBodyFixtures() {
		t.Run(fixture.plural, func(t *testing.T) {
			cloud, client := clusteringBodyCloud(t, fixture)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+fixture.path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				query := r.URL.Query()
				if query.Get("vendor") != "kept" || query.Has(fixture.primary) || query.Has(fixture.alias) {
					t.Error("query continuity/filter changed", r.URL)
				}
				if query.Get("marker") == "" {
					if !fixture.marker {
						w.Header().Set("Link", fmt.Sprintf(`<%s/%s?marker=wire-first>; rel="next"`, resourceListPrefix, fixture.path))
					}
					testcloud.JSON(w, 200, clusteringBodyJSON(t, fixture, clusteringBodyRow(fixture, "wire-first", map[string]any{fixture.primary: "miss"})))
					return
				}
				if query.Get("marker") != "wire-first" {
					t.Error("marker came from matching row rather than raw response", r.URL)
				}
				testcloud.JSON(w, 200, clusteringBodyJSON(t, fixture, clusteringBodyRow(fixture, "last", map[string]any{fixture.primary: "kept"})))
			})
			rows, err := clusteringBodyCollect(t, fixture.list(t, context.Background(), client, clusteringBodyRequest{
				filters: []clusteringBodyValue{{fixture.primary, "kept"}}, query: []clusteringBodyValue{{"vendor", "kept"}}, maximum: 2,
			}))
			if err != nil || len(rows) != 1 || calls.Load() != 2 {
				t.Fatal("filtered first page stopped or fixed parent/raw cap changed", rows, err, calls.Load())
			}
		})
	}
}

func TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP(t *testing.T) {
	for _, fixture := range clusteringBodyFixtures() {
		for _, mode := range []string{"known-body-query", "alias-query", "unknown-filter", "source-excluded-field", "bad-json", "foreign-namespace"} {
			t.Run(fixture.plural+"/"+mode, func(t *testing.T) {
				cloud, client := clusteringBodyCloud(t, fixture)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+fixture.path, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
				input := clusteringBodyRequest{}
				switch mode {
				case "known-body-query":
					input.query = []clusteringBodyValue{{fixture.primary, "kept"}}
				case "alias-query":
					input.query = []clusteringBodyValue{{fixture.alias, "kept"}}
				case "unknown-filter":
					input.filters = []clusteringBodyValue{{"future_not_a_body_attribute", "kept"}}
				case "source-excluded-field":
					input.filters = []clusteringBodyValue{{fixture.excluded, nil}}
				case "bad-json":
					input.filters = []clusteringBodyValue{{fixture.primary, json.RawMessage(`{`)}}
				case "foreign-namespace":
					input.foreign = true
				}
				rows, err := clusteringBodyCollect(t, fixture.list(t, context.Background(), client, input))
				if len(rows) != 0 || calls.Load() != 0 || !(errors.Is(err, resource.ErrInvalidOption) || mode == "foreign-namespace" && errors.Is(err, resource.ErrUnsupported)) {
					t.Fatal("invalid filter or cross-namespace argument reached child HTTP", rows, err, calls.Load())
				}
			})
		}
	}
}

func TestClusteringBodyFiltersNestedSubsetsExactNumbersArraysAndBooleans(t *testing.T) {
	for _, fixture := range clusteringBodyFixtures() {
		if fixture.object == "" {
			continue
		}
		for _, mode := range []string{"nested-subset", "ordered-array", "array-object-exact", "bool-is-not-number", "empty-actual-object"} {
			t.Run(fixture.plural+"/"+mode, func(t *testing.T) {
				cloud, client := clusteringBodyCloud(t, fixture)
				actual := any(map[string]any{"nested": map[string]any{"big": json.Number("9007199254740993.125"), "extra": true}})
				wanted := any(map[string]any{"nested": map[string]any{"big": json.Number("9007199254740993125e-3")}})
				want := 1
				switch mode {
				case "ordered-array":
					actual, wanted, want = map[string]any{"items": []int{1, 2}}, map[string]any{"items": []int{2, 1}}, 0
				case "array-object-exact":
					actual, wanted, want = map[string]any{"items": []any{map[string]any{"n": 1, "extra": true}}}, map[string]any{"items": []any{map[string]any{"n": 1}}}, 0
				case "bool-is-not-number":
					actual, wanted, want = map[string]any{"n": 1}, map[string]any{"n": true}, 0
				case "empty-actual-object":
					actual, wanted, want = map[string]any{}, map[string]any{}, 0
				}
				cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Has(fixture.object) {
						t.Error("nested filter went to server", r.URL)
					}
					testcloud.JSON(w, 200, clusteringBodyJSON(t, fixture, clusteringBodyRow(fixture, "row", map[string]any{fixture.object: actual})))
				})
				optionValue, err := json.Marshal(wanted)
				if err != nil {
					t.Fatal(err)
				}
				iterator := fixture.list(t, context.Background(), client, clusteringBodyRequest{filters: []clusteringBodyValue{{fixture.object, json.RawMessage(optionValue)}}, first: true})
				for i := range optionValue {
					optionValue[i] = '!'
				}
				rows, err := clusteringBodyCollect(t, iterator)
				if err != nil || len(rows) != want {
					t.Fatal("raw filter JSON semantics or input ownership changed", rows, err)
				}
			})
		}
	}
}

func TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues(t *testing.T) {
	for _, fixture := range clusteringBodyFixtures() {
		t.Run(fixture.plural, func(t *testing.T) {
			cloud, client := clusteringBodyCloud(t, fixture)
			cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+fixture.path, func(w http.ResponseWriter, _ *http.Request) {
				missing := clusteringBodyRow(fixture, "missing", nil)
				delete(missing, fixture.nullable)
				testcloud.JSON(w, 200, clusteringBodyJSON(t, fixture, missing,
					clusteringBodyRow(fixture, "null", map[string]any{fixture.nullable: nil}),
					clusteringBodyRow(fixture, "value", map[string]any{fixture.nullable: "present"})))
			})
			rows, err := clusteringBodyCollect(t, fixture.list(t, context.Background(), client, clusteringBodyRequest{filters: []clusteringBodyValue{{fixture.nullable, nil}}, first: true}))
			if err != nil || len(rows) != 2 {
				t.Fatal(rows, err)
			}
			if _, exists := rows[0].Body[fixture.nullable]; exists || string(rows[1].Body[fixture.nullable]) != "null" {
				t.Fatal("matching null erased raw presence evidence", rows)
			}
		})
	}
}

func TestClusteringBodyFiltersRawCatalogIDDoesNotFallbackAndBindingValidationPrecedesFilter(t *testing.T) {
	for _, fixture := range clusteringBodyFixtures() {
		if fixture.plural != "profile_types" && fixture.plural != "policy_types" && !fixture.binding {
			continue
		}
		t.Run(fixture.plural, func(t *testing.T) {
			cloud, client := clusteringBodyCloud(t, fixture)
			body := ""
			cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+fixture.path, func(w http.ResponseWriter, _ *http.Request) {
				row := clusteringBodyRow(fixture, "name-as-id", nil)
				delete(row, "id")
				body = clusteringBodyJSON(t, fixture, row)
				w.Header().Set("X-Request-ID", "raw-identity")
				testcloud.JSON(w, 200, body)
			})
			rows, err := clusteringBodyCollect(t, fixture.list(t, context.Background(), client, clusteringBodyRequest{filters: []clusteringBodyValue{{"id", "name-as-id"}}, first: true}))
			if fixture.binding {
				var evidence *resource.ResponseError
				if len(rows) != 0 || !errors.As(err, &evidence) || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "raw-identity" {
					t.Fatal("invalid binding identity was hidden by nonmatching filter", rows, err, evidence)
				}
			} else if err != nil || len(rows) != 0 {
				t.Fatal("raw Body id incorrectly fell back to catalog name", rows, err)
			}
		})
	}
}

func TestClusteringBodyFiltersServicesVersionGateOnFilteredContinuation(t *testing.T) {
	fixture := clusteringBodyFixtures()[2]
	cloud, client := clusteringBodyCloud(t, fixture)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/services", func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		client.Microversion = "1.6"
		w.Header().Set("Link", fmt.Sprintf(`<%s/services?marker=next>; rel="next"`, resourceListPrefix))
		testcloud.JSON(w, 200, clusteringBodyJSON(t, fixture, clusteringBodyRow(fixture, "filtered", map[string]any{"host": "miss"})))
	})
	rows, err := clusteringBodyCollect(t, fixture.list(t, context.Background(), client, clusteringBodyRequest{filters: []clusteringBodyValue{{"host", "kept"}}}))
	if len(rows) != 0 || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
		t.Fatal("filtered rows bypassed next-page version gate", rows, err, calls.Load())
	}
}

func TestClusteringBodyFiltersEventLevelUsesRawJSONRatherThanTypedString(t *testing.T) {
	fixture := clusteringBodyFixtures()[1]
	for _, wanted := range []any{json.Number("20"), "20"} {
		t.Run(fmt.Sprintf("%T", wanted), func(t *testing.T) {
			cloud, client := clusteringBodyCloud(t, fixture)
			cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/events", func(w http.ResponseWriter, _ *http.Request) {
				testcloud.JSON(w, 200, clusteringBodyJSON(t, fixture, clusteringBodyRow(fixture, "event", map[string]any{"level": json.Number("20")})))
			})
			rows, err := clusteringBodyCollect(t, fixture.list(t, context.Background(), client, clusteringBodyRequest{filters: []clusteringBodyValue{{"level", wanted}}, first: true}))
			count := 0
			if _, numeric := wanted.(json.Number); numeric {
				count = 1
			}
			if err != nil || len(rows) != count {
				t.Fatal("typed Event.Level string coerced the raw numeric filter", rows, err)
			}
		})
	}
}
