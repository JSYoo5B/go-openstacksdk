package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	qospolicies "gophercloudsdk/network/v2/extensions/qos/policies"
	securitygroups "gophercloudsdk/network/v2/extensions/security/groups"
	"gophercloudsdk/resource"
)

// The expected map is independent of the emitted runtime descriptor: pinned
// QoSPolicy inherits Resource+TagMixin, not NetworkResource. Its id and tags
// are server queries; only rules and the deprecated tenant_id are local.
func qosPolicyListQueries() map[string]string {
	return map[string]string{"any_tags": "tags-any", "description": "description", "fields": "fields", "id": "id", "is_default": "is_default", "is_shared": "shared", "limit": "limit", "marker": "marker", "name": "name", "not_any_tags": "not-tags-any", "not_tags": "not-tags", "project_id": "project_id", "sort_dir": "sort_dir", "sort_key": "sort_key", "tags": "tags"}
}

type qosPolicyListFixture struct {
	name string
	open func(*testing.T, *gophercloud.ServiceClient) *qospolicies.API
}

func qosPolicyListFixtures() []qosPolicyListFixture {
	return []qosPolicyListFixture{
		{"leaf", func(_ *testing.T, c *gophercloud.ServiceClient) *qospolicies.API { return qospolicies.New(c) }},
		{"connection", func(t *testing.T, c *gophercloud.ServiceClient) *qospolicies.API {
			s := networkBodyFilterConnection(t, c)
			if s.API.QoSPolicies.RawClient() != s.RawClient() {
				t.Fatal("QoS policy did not share the cached network client")
			}
			return s.API.QoSPolicies
		}},
	}
}

const qosPolicyListPath = networkExtensionPrefix + "qos/policies"

func qosPolicyListRow(t *testing.T, id, name string, fields map[string]json.RawMessage) string {
	t.Helper()
	body := map[string]json.RawMessage{"id": json.RawMessage(fmt.Sprintf("%q", id)), "name": json.RawMessage(fmt.Sprintf("%q", name)), "project_id": json.RawMessage(`"project-independent"`), "description": json.RawMessage(`"native-description"`), "is_default": json.RawMessage(`false`), "shared": json.RawMessage(`false`)}
	for key, value := range fields {
		body[key] = value
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func qosPolicyListPage(rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,"policies_links":[{"rel":"next","href":%q}]`, next)
	}
	return `{"policies":[` + rows + `]` + links + `}`
}

func qosPolicyListWant(t *testing.T, values []*qospolicies.Policy, err error, ids ...string) {
	t.Helper()
	actual := make([]string, 0, len(values))
	for _, value := range values {
		actual = append(actual, value.ID)
	}
	if err != nil || !reflect.DeepEqual(actual, append([]string{}, ids...)) {
		t.Fatal("actual native collection result", actual, err, "want", ids)
	}
}

func TestQoSPolicyListFiltersEntireQueryAndAliasDescriptor(t *testing.T) {
	canonical := qosPolicyListQueries()
	accepted := make(map[string]string)
	for key, wire := range canonical {
		accepted[key], accepted[wire] = wire, wire
	}
	if len(canonical) != 15 || len(accepted) != 19 {
		t.Fatal("independent descriptor changed", canonical, accepted)
	}
	for _, f := range qosPolicyListFixtures() {
		for key, wire := range accepted {
			t.Run(f.name+"/"+key, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{wire: {"server-value"}}) {
						t.Error("semantic query alias changed", key, r.URL)
					}
					// Returned id/name/tags/default/shared intentionally differ:
					// all of these queried fields must remain server-only.
					testcloud.JSON(w, 200, qosPolicyListPage(qosPolicyListRow(t, "server-returned", "Different", map[string]json.RawMessage{"tags": json.RawMessage(`["different-tag"]`)}), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter(key, "server-value"))
				qosPolicyListWant(t, values, err, "server-returned")
				if calls.Load() != 1 {
					t.Fatal("query classification performed additional HTTP", calls.Load())
				}
			})
		}
		for key, wire := range canonical {
			if key == wire {
				continue
			}
			for _, tc := range []struct {
				name  string
				value any
				want  []string
			}{
				{"nil", nil, nil}, {"false", false, []string{"false"}}, {"empty-string", "", []string{""}}, {"empty-array", []any{}, nil}, {"repeats", []any{"first", nil, "last"}, []string{"first", "last"}},
			} {
				t.Run(f.name+"/precedence/"+key+"/"+tc.name, func(t *testing.T) {
					cloud := testcloud.New(t)
					cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
						want := make(url.Values)
						if tc.want != nil {
							want[wire] = tc.want
						}
						if !reflect.DeepEqual(r.URL.Query(), want) {
							t.Error("canonical bulk presence did not override wire alias", r.URL.Query(), want)
						}
						testcloud.JSON(w, 200, qosPolicyListPage(qosPolicyListRow(t, "result", "Different", nil), ""))
					})
					values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{key: tc.value, wire: json.RawMessage(`{`)}))
					qosPolicyListWant(t, values, err, "result")
				})
			}
			for _, reverse := range []bool{false, true} {
				t.Run(f.name+"/last-wins/"+key+fmt.Sprint(reverse), func(t *testing.T) {
					cloud := testcloud.New(t)
					options := []resource.ListOption{resource.WithFilter(wire, "first"), resource.WithFilter(key, "last")}
					want := "last"
					if reverse {
						options[0], options[1], want = options[1], options[0], "first"
					}
					cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
						if !reflect.DeepEqual(r.URL.Query(), url.Values{wire: {want}}) {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 200, qosPolicyListPage(qosPolicyListRow(t, "result", "Different", nil), ""))
					})
					values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), options...)
					qosPolicyListWant(t, values, err, "result")
				})
			}
		}
		t.Run(f.name+"/scalar-encoding", func(t *testing.T) {
			cloud := testcloud.New(t)
			want := url.Values{"fields": {"id", "name", "true", "false", "9007199254740993", ""}, "shared": {"false"}, "is_default": {"true"}, "limit": {"2e1"}, "name": {""}}
			cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, qosPolicyListPage(qosPolicyListRow(t, "result", "Different", nil), ""))
			})
			values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{"fields": []any{"id", nil, "name", true, false, json.Number("9007199254740993"), ""}, "is_shared": false, "is_default": true, "limit": json.Number("2e1"), "name": "", "description": nil, "marker": []any{}}))
			qosPolicyListWant(t, values, err, "result")
		})
	}
}

func TestQoSPolicyListFiltersRawRulesAndTenantPresence(t *testing.T) {
	for _, f := range qosPolicyListFixtures() {
		for _, tc := range []struct {
			name, raw string
			filter    any
			match     bool
		}{
			{"missing", "missing", nil, true}, {"null", "null", nil, true}, {"empty-distinct", "[]", nil, false}, {"empty", "[]", []any{}, true},
			{"ordered", `[{"type":"bandwidth_limit","vendor":{"n":9007199254740993,"enabled":true}},null]`, json.RawMessage(`[{"type":"bandwidth_limit","vendor":{"n":9007199254740993,"enabled":true}},null]`), true},
			{"no-rounding", `[{"quota":9007199254740993}]`, json.RawMessage(`[{"quota":9007199254740992}]`), false},
			{"exact-large-number", `[{"quota":9007199254740993}]`, json.RawMessage(`[{"quota":9007199254740993}]`), true},
			{"decimal-equality", `[{"quota":1.2e3}]`, json.RawMessage(`[{"quota":1200.00}]`), true},
			{"inner-objects-not-subsets", `[{"quota":1200,"extra":true}]`, json.RawMessage(`[{"quota":1200}]`), false},
			{"order", `[{"type":"one"},{"type":"two"}]`, json.RawMessage(`[{"type":"two"},{"type":"one"}]`), false},
			{"shorter", `[{"type":"one"},{"type":"two"}]`, json.RawMessage(`[{"type":"one"}]`), false},
			{"longer", `[{"type":"one"}]`, json.RawMessage(`[{"type":"one"},null]`), false},
			{"null-map", `[null,{"key":null}]`, json.RawMessage(`[null,{"key":null}]`), true},
			{"null-map-not-empty", `[null]`, json.RawMessage(`[{}]`), false}, {"bool-number-distinct", `[{"flag":false}]`, json.RawMessage(`[{"flag":0}]`), false},
			{"wrong-scalar-filter", `[{"type":"one"}]`, "one", false}, {"object-against-array", `[{"quota":1}]`, map[string]any{}, false},
		} {
			t.Run(f.name+"/rules/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				fields := map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant"`)}
				if tc.raw != "missing" {
					fields["rules"] = json.RawMessage(tc.raw)
				}
				cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.RawQuery != "" {
						t.Error("local rules became wire query", r.URL)
					}
					testcloud.JSON(w, 200, qosPolicyListPage(qosPolicyListRow(t, "target", "Different", fields), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithBodyFilter("rules", tc.filter))
				var want []string
				if tc.match {
					want = []string{"target"}
				}
				qosPolicyListWant(t, values, err, want...)
				if tc.name == "exact-large-number" && values[0].Rules[0]["quota"] != float64(9007199254740992) {
					t.Fatal("raw matching replaced the returned native float64 projection", values[0].Rules)
				}
				if tc.name == "null-map" && (values[0].Rules[0] != nil || values[0].Rules[1]["key"] != nil) {
					t.Fatal("native null maps changed", values[0].Rules)
				}
			})
		}
		for _, tc := range []struct {
			name, raw string
			filter    any
			match     bool
		}{
			{"missing", "missing", nil, true}, {"null", "null", nil, true}, {"missing-not-empty", "missing", "", false}, {"null-not-empty", "null", "", false}, {"empty", `""`, "", true}, {"no-project-fallback", "missing", "project-independent", false}, {"literal", `"tenant-original"`, "tenant-original", true},
		} {
			t.Run(f.name+"/tenant/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				fields := map[string]json.RawMessage{"rules": json.RawMessage(`[]`)}
				if tc.raw != "missing" {
					fields["tenant_id"] = json.RawMessage(tc.raw)
				}
				cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), url.Values{"project_id": {"server-project"}}) {
						t.Error("tenant filter became a project query", r.URL)
					}
					testcloud.JSON(w, 200, qosPolicyListPage(qosPolicyListRow(t, "target", "Different", fields), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter("tenant_id", tc.filter), resource.WithFilter("project_id", "server-project"), resource.WithFilter("rules", []any{}))
				var want []string
				if tc.match {
					want = []string{"target"}
				}
				qosPolicyListWant(t, values, err, want...)
				if tc.match && values[0].ProjectID != "project-independent" {
					t.Fatal("native project was replaced by the local tenant", values[0])
				}
			})
		}
	}
}

func TestQoSPolicyListFiltersSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	for _, f := range qosPolicyListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			vendor := map[string]any{"n": json.Number("9007199254740993")}
			rules := []any{map[string]any{"vendor": vendor}, nil}
			fields := []any{"id", "name", nil}
			shared := false
			tenant := "tenant-original"
			input := map[string]any{"rules": rules, "fields": fields, "is_shared": &shared, "tenant_id": &tenant}
			bulk := resource.WithFilters(input)
			input["rules"], vendor["n"], rules[1], fields[0], shared, tenant = nil, 0, true, "changed", true, "changed"
			var expected atomic.Value
			expected.Store(url.Values{"fields": {"id", "name"}, "shared": {"false"}, "vendor": {"kept"}})
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), expected.Load().(url.Values)) {
					t.Error("query option snapshot changed", r.URL.Query())
				}
				rows := qosPolicyListRow(t, "target", "Different", map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant-original"`), "rules": json.RawMessage(`[{"vendor":{"n":9007199254740993}},null]`)}) + "," + qosPolicyListRow(t, "other", "Different", map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant-original"`), "rules": json.RawMessage(`[]`)})
				testcloud.JSON(w, 200, qosPolicyListPage(rows, ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			opts := []resource.ListOption{resource.WithQuery("vendor", "kept"), bulk}
			values, err := a.Resources.All(context.Background(), opts...)
			qosPolicyListWant(t, values, err, "target")
			var wg sync.WaitGroup
			for range 8 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					values, err := a.Resources.All(context.Background(), opts...)
					qosPolicyListWant(t, values, err, "target")
				}()
			}
			wg.Wait()
			expected.Store(url.Values{"vendor": {"kept"}})
			for _, clear := range []resource.ListOption{resource.WithFilters(nil), resource.WithFilters(map[string]any{})} {
				values, err = a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), opts...), clear, resource.WithBodyFilter("tenant_id", "tenant-original"))...)
				qosPolicyListWant(t, values, err, "target", "other")
			}
			for _, tc := range []struct {
				options []resource.ListOption
				want    []string
			}{
				{[]resource.ListOption{resource.WithFilter("fields", math.NaN()), resource.WithFilter("fields", nil), resource.WithFilter("rules", make(chan int)), resource.WithFilter("rules", json.RawMessage(`[{"vendor":{"n":9007199254740993}},null]`))}, []string{"target"}},
				{[]resource.ListOption{resource.WithFilter("fields", map[string]any{"bad": true}), resource.WithFilter("rules", json.RawMessage(`{`)), resource.WithFilter("headers", true), resource.WithFilters(nil)}, []string{"target", "other"}},
				{[]resource.ListOption{resource.WithFilter("rules", []any{}), resource.WithFilter("rules", json.RawMessage(`[{"vendor":{"n":9007199254740993}},null]`))}, []string{"target"}},
			} {
				values, err = a.Resources.All(context.Background(), append([]resource.ListOption{resource.WithQuery("vendor", "kept")}, tc.options...)...)
				qosPolicyListWant(t, values, err, tc.want...)
			}
			if calls.Load() != 14 {
				t.Fatal("option reuse changed requests", calls.Load())
			}
		})
	}
}

func TestQoSPolicyListFiltersLazyPreflightAndNamespaceCollisions(t *testing.T) {
	for _, f := range qosPolicyListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("unknown semantic fields leaked", r.URL)
				}
				testcloud.JSON(w, 200, qosPolicyListPage(qosPolicyListRow(t, "result", "Different", nil), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			collisions := [][]resource.ListOption{{resource.WithFilter("is_shared", false), resource.WithQuery("shared", "false")}, {resource.WithFilter("any_tags", nil), resource.WithQuery("tags-any", "")}, {resource.WithFilter("fields", []any{}), resource.WithQuery("fields", "")}, {resource.WithFilter("name", nil), resource.WithName("Target")}, {resource.WithFilter("limit", 2), resource.WithPageSize(2)}, {resource.WithFilter("rules", nil), resource.WithBodyFilter("rules", nil)}, {resource.WithFilter("tenant_id", "tenant"), resource.WithBodyFilter("tenant_id", "tenant")}}
			for _, options := range collisions {
				for _, reverse := range []bool{false, true} {
					opts := append([]resource.ListOption(nil), options...)
					if reverse {
						opts[0], opts[1] = opts[1], opts[0]
					}
					values, err := a.Resources.All(context.Background(), opts...)
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
						t.Fatal("namespace collision did HTTP", values, err, calls.Load())
					}
				}
			}
			invalid := []resource.ListOption{resource.WithFilter("fields", map[string]any{"id": true}), resource.WithFilter("fields", [][]string{{"id"}}), resource.WithFilter("rules", math.NaN()), resource.WithBodyFilter("tenant_id", json.RawMessage(`{`)), resource.WithBodyFilter("id", "id"), resource.WithBodyFilter("name", "Different"), resource.WithBodyFilter("is_shared", false), resource.WithBodyFilter("project_id", "project"), resource.WithStatus("ACTIVE"), resource.WithMaxItems(-1)}
			for _, key := range []string{"max_items", "paginated", "base_path", "allow_unknown_params", "session", "headers", "microversion", "resource_type", "jmespath_filters"} {
				invalid = append(invalid, resource.WithFilter(key, nil))
			}
			for _, option := range invalid {
				seq := a.Resources.List(context.Background(), option)
				if calls.Load() != 0 {
					t.Fatal("constructing iterator performed HTTP")
				}
				seen := 0
				for value, err := range seq {
					seen++
					if value != nil || (!errors.Is(err, resource.ErrInvalidOption) && !errors.Is(err, resource.ErrUnsupported)) {
						t.Fatal(value, err)
					}
				}
				if seen != 1 || calls.Load() != 0 {
					t.Fatal("preflight did not yield one terminal error", seen, calls.Load())
				}
			}
			values, err := a.Resources.All(context.Background(), resource.WithFilters(map[string]any{"created_at": make(chan int), "updated_at": json.RawMessage(`{`), "revision_number": math.NaN(), "status": func() {}, "unknown": make(chan int)}))
			qosPolicyListWant(t, values, err, "result")
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			values, err = a.Resources.All(ctx, resource.WithFilters(nil))
			if values != nil || !errors.Is(err, context.Canceled) || calls.Load() != 1 {
				t.Fatal(values, err, calls.Load())
			}
			_, err = securitygroups.New(networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(nil))
			if !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatal("unaudited binding gained a descriptor", err, calls.Load())
			}
		})
	}
}

func TestQoSPolicyListFiltersControlsAndNativeWholePageFailures(t *testing.T) {
	for _, f := range qosPolicyListFixtures() {
		for _, mode := range []string{"raw-cap-before-body", "raw-cap-before-name", "single-page", "break", "null", "late-null", "cap-null", "empty-object", "query-null", "bad-rules-object", "bad-rule-element", "bad-tenant", "bad-bool", "bad-revision", "bad-tags", "bad-date", "bad-id"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				foreign := testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); w.WriteHeader(500) })
				rows := qosPolicyListRow(t, "first", "Different", map[string]json.RawMessage{"rules": json.RawMessage(`[]`)}) + "," + qosPolicyListRow(t, "target", "Target", map[string]json.RawMessage{"rules": json.RawMessage(`[{"type":"one"}]`)})
				opts := []resource.ListOption{resource.WithFilter("rules", []any{})}
				switch mode {
				case "raw-cap-before-body":
					opts = []resource.ListOption{resource.WithFilter("rules", json.RawMessage(`[{"type":"one"}]`)), resource.WithMaxItems(1)}
				case "raw-cap-before-name":
					opts = append(opts, resource.WithName("Target"), resource.WithMaxItems(1))
				case "single-page":
					opts = append(opts, resource.WithPaginated(false))
				case "null", "query-null":
					rows = "null"
				case "late-null", "cap-null":
					rows = qosPolicyListRow(t, "first", "Different", map[string]json.RawMessage{"rules": json.RawMessage(`[]`)}) + ",null"
				case "empty-object":
					rows = "{}"
					opts = []resource.ListOption{resource.WithFilter("rules", nil), resource.WithPaginated(false)}
				default:
					if strings.HasPrefix(mode, "bad-") {
						field, raw := "rules", "{}"
						switch mode {
						case "bad-rule-element":
							raw = `[true]`
						case "bad-tenant":
							field, raw = "tenant_id", "9007199254740993"
						case "bad-bool":
							field, raw = "shared", `"false"`
						case "bad-revision":
							field, raw = "revision_number", `"1"`
						case "bad-tags":
							field, raw = "tags", `[1]`
						case "bad-date":
							field, raw = "created_at", `"2025-01-02T03:04:05"`
						case "bad-id":
							field, raw = "id", "false"
						}
						rows += "," + qosPolicyListRow(t, "invalid", "Different", map[string]json.RawMessage{field: json.RawMessage(raw)})
						opts = append(opts, resource.WithMaxItems(1))
					}
				}
				if mode == "cap-null" {
					opts = append(opts, resource.WithMaxItems(1))
				}
				if mode == "query-null" {
					opts = []resource.ListOption{resource.WithFilter("name", "server-only"), resource.WithPaginated(false)}
				}
				cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, _ *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 200, qosPolicyListPage(rows, foreign.Server.URL+"/foreign"))
				})
				a := f.open(t, networkExtensionClient(cloud))
				if mode == "break" {
					seen := 0
					for value, err := range a.Resources.List(context.Background(), opts...) {
						if err != nil || value == nil || value.ID != "first" {
							t.Fatal(value, err)
						}
						seen++
						break
					}
					if seen != 1 || calls.Load() != 1 || followed.Load() != 0 {
						t.Fatal(seen, calls.Load(), followed.Load())
					}
					return
				}
				values, err := a.Resources.All(context.Background(), opts...)
				switch {
				case strings.HasPrefix(mode, "raw-cap-before"):
					qosPolicyListWant(t, values, err)
				case mode == "single-page" || mode == "cap-null":
					qosPolicyListWant(t, values, err, "first")
				case mode == "empty-object" || mode == "query-null":
					qosPolicyListWant(t, values, err, "")
				case mode == "null" || mode == "late-null":
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal("consumed null or partial All accepted", values, err)
					}
				default:
					if values != nil || err == nil {
						t.Fatal("native whole-page failure skipped", values, err)
					}
					if mode == "bad-date" {
						var cause *time.ParseError
						if !errors.As(err, &cause) {
							t.Fatal("native timestamp cause lost", err)
						}
					} else {
						var cause *json.UnmarshalTypeError
						if !errors.As(err, &cause) {
							t.Fatal("native cause lost", err)
						}
					}
				}
				if calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal("current-page cap/error fetched continuation", calls.Load(), followed.Load())
				}
			})
		}
	}
}

func TestQoSPolicyListFiltersPagingLiveSourceAndNativeStatus(t *testing.T) {
	for _, f := range qosPolicyListFixtures() {
		for _, mode := range []string{"complete", "duplicates", "late-http", "late-decode", "cycle", "foreign", "cancel", "wrong-link-shape", "204", "json-204", "201"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				foreign := testcloud.New(t)
				var calls, followed, middleware atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); w.WriteHeader(500) })
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				c := networkExtensionClient(cloud)
				c.MoreHeaders, c.Microversion = map[string]string{"X-Configured": "kept"}, "2.0"
				c.ProviderClient.SetToken("first-token")
				base := c.ProviderClient.HTTPClient.Transport
				c.ProviderClient.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					middleware.Add(1)
					resp, err := base.RoundTrip(r)
					if r.URL.Query().Get("marker") == "" {
						c.ProviderClient.SetToken("later-token")
						if mode == "cancel" {
							cancel()
						}
					}
					return resp, err
				})
				cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("X-Configured") != "kept" || r.Header.Get("OpenStack-API-Version") != "network 2.0" || r.URL.Query().Get("shared") != "false" || r.URL.Query().Get("status") != "vendor-status" || !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) {
						t.Error("source/query changed", r.Header, r.URL)
					}
					if r.URL.Query().Get("marker") != "" {
						if r.Header.Get("X-Auth-Token") != "later-token" {
							t.Error("provider token not live", r.Header)
						}
						switch mode {
						case "late-http":
							w.WriteHeader(404)
							return
						case "late-decode":
							testcloud.JSON(w, 200, qosPolicyListPage(qosPolicyListRow(t, "invalid", "Different", map[string]json.RawMessage{"rules": json.RawMessage(`{}`)}), ""))
							return
						}
						id := "second"
						if mode == "duplicates" {
							id = "first"
						}
						testcloud.JSON(w, 200, qosPolicyListPage(qosPolicyListRow(t, id, "Different", map[string]json.RawMessage{"rules": json.RawMessage(`[]`)}), ""))
						return
					}
					if r.Header.Get("X-Auth-Token") != "first-token" {
						t.Error(r.Header)
					}
					if mode == "204" {
						w.WriteHeader(204)
						return
					}
					if mode == "json-204" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(204)
						return
					}
					if mode == "201" {
						w.WriteHeader(201)
						return
					}
					next := cloud.Server.URL + qosPolicyListPath + "?marker=next&shared=false&fields=id&fields=name&status=vendor-status"
					if mode == "cycle" {
						next = cloud.Server.URL + r.URL.String()
					}
					if mode == "foreign" {
						next = foreign.Server.URL + "/foreign"
					}
					row := qosPolicyListRow(t, "first", "Different", map[string]json.RawMessage{"rules": json.RawMessage(`[]`)})
					if mode == "wrong-link-shape" {
						testcloud.JSON(w, 300, `{"policies":[`+row+`],"links":{"next":`+fmt.Sprintf("%q", next)+`},"next":`+fmt.Sprintf("%q", next)+`}`)
						return
					}
					testcloud.JSON(w, 300, qosPolicyListPage(row, next))
				})
				a := f.open(t, c)
				values, err := a.Resources.All(ctx, resource.WithFilter("rules", []any{}), resource.WithFilter("is_shared", false), resource.WithFilter("fields", []string{"id", "name"}), resource.WithQuery("status", "vendor-status"))
				switch mode {
				case "complete":
					qosPolicyListWant(t, values, err, "first", "second")
				case "duplicates":
					qosPolicyListWant(t, values, err, "first", "first")
				case "wrong-link-shape":
					qosPolicyListWant(t, values, err, "first")
				case "204":
					qosPolicyListWant(t, values, err)
				default:
					if values != nil || err == nil {
						t.Fatal("terminal page observation returned partial All", values, err)
					}
					if mode == "cycle" && !errors.Is(err, resource.ErrPaginationCycle) || mode == "cancel" && !errors.Is(err, context.Canceled) || mode == "json-204" && !errors.Is(err, io.EOF) || mode == "201" && !gophercloud.ResponseCodeIs(err, 201) {
						t.Fatal(mode, err)
					}
				}
				wantCalls, wantMiddleware, wantFollow := int32(1), int32(1), int32(0)
				if mode == "complete" || mode == "duplicates" || mode == "late-http" || mode == "late-decode" {
					wantCalls, wantMiddleware = 2, 2
				}
				if mode == "foreign" {
					wantMiddleware, wantFollow = 2, 1
				}
				if calls.Load() != wantCalls || middleware.Load() != wantMiddleware || followed.Load() != wantFollow || a.RawClient().ProviderClient != c.ProviderClient {
					t.Fatal("native continuation/provider changed", calls.Load(), middleware.Load(), followed.Load())
				}
			})
		}
	}
}

func TestQoSPolicyListFiltersNativeAndIdentitySurfaceIsolation(t *testing.T) {
	for _, f := range qosPolicyListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets atomic.Int32
			want := url.Values{"tenant_id": {"wire-tenant"}, "rules": {"wire-rules"}, "is_shared": {"raw-client-alias"}, "shared": {"false"}, "status": {"vendor-status"}, "name": {"server-pattern"}, "vendor": {"kept"}}
			cloud.Mux.HandleFunc("GET "+qosPolicyListPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("raw/native query acquired semantic classification", r.URL.Query(), want)
				}
				rows := qosPolicyListRow(t, "target", "Target", map[string]json.RawMessage{"rules": json.RawMessage(`[]`)}) + "," + qosPolicyListRow(t, "other", "Other", map[string]json.RawMessage{"rules": json.RawMessage(`[{"type":"one"}]`)})
				testcloud.JSON(w, 200, qosPolicyListPage(rows, ""))
			})
			cloud.Mux.HandleFunc("GET "+qosPolicyListPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"tenant_id": {"wire-tenant"}, "rules": {"wire-rules"}, "is_shared": {"raw-client-alias"}, "status": {"vendor-status"}}) {
					t.Error("FindIdentity acquired semantic classification", r.URL)
				}
				testcloud.JSON(w, 200, `{"policy":`+qosPolicyListRow(t, "lookup", "Different", map[string]json.RawMessage{"rules": json.RawMessage(`[]`)})+`}`)
			})
			a := f.open(t, networkExtensionClient(cloud))
			var raw []resource.ListOption
			var native []qospolicies.ListOption
			for key, values := range want {
				raw = append(raw, resource.WithQuery(key, values[0]))
				native = append(native, qospolicies.WithListQuery(key, values[0]))
			}
			values, err := a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilter("rules", []any{}))...)
			qosPolicyListWant(t, values, err, "target")
			values, err = a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilters(nil))...)
			qosPolicyListWant(t, values, err, "target", "other")
			var nativeIDs []string
			for value, err := range a.List(context.Background(), native...) {
				if err != nil {
					t.Fatal(err)
				}
				nativeIDs = append(nativeIDs, value.ID)
			}
			if !reflect.DeepEqual(nativeIDs, []string{"target", "other"}) {
				t.Fatal("typed List gained local predicates", nativeIDs)
			}
			value, err := a.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindQuery("tenant_id", "wire-tenant"), resource.WithIdentityFindQuery("rules", "wire-rules"), resource.WithIdentityFindQuery("is_shared", "raw-client-alias"), resource.WithIdentityFindQuery("status", "vendor-status"))
			if err != nil || value == nil || value.ID != "lookup" || gets.Load() != 1 || lists.Load() != 3 {
				t.Fatal("FindIdentity changed", value, err, gets.Load(), lists.Load())
			}
			want["name"] = []string{"Target"}
			var named []resource.ListOption
			for key, values := range want {
				if key != "name" {
					named = append(named, resource.WithQuery(key, values[0]))
				}
			}
			named = append(named, resource.WithName("Target"), resource.WithBodyFilter("rules", []any{}))
			values, err = a.Resources.All(context.Background(), named...)
			qosPolicyListWant(t, values, err, "target")
			before := lists.Load()
			values, err = a.Resources.All(context.Background(), resource.WithStatus("ACTIVE"), resource.WithBodyFilter("rules", []any{}))
			if values != nil || !errors.Is(err, resource.ErrUnsupported) || lists.Load() != before {
				t.Fatal("native policy has no Status", values, err, lists.Load())
			}
		})
	}
}
