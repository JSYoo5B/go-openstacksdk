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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/ports"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Independent pinned SubnetPool QueryParameters, not native ListOpts q tags.
// id is local; name/project/tags/default/shared/address_scope are server queries.
func subnetPoolListQueries() map[string]string {
	return map[string]string{"address_scope_id": "address_scope_id", "any_tags": "tags-any", "description": "description", "fields": "fields", "ip_version": "ip_version", "is_default": "is_default", "is_shared": "shared", "limit": "limit", "marker": "marker", "name": "name", "not_any_tags": "not-tags-any", "not_tags": "not-tags", "project_id": "project_id", "sort_dir": "sort_dir", "sort_key": "sort_key", "tags": "tags"}
}

type subnetPoolListFixture struct {
	name string
	open func(*testing.T, *gophercloud.ServiceClient) *subnetpools.API
}

func subnetPoolListFixtures() []subnetPoolListFixture {
	return []subnetPoolListFixture{
		{"leaf", func(_ *testing.T, c *gophercloud.ServiceClient) *subnetpools.API { return subnetpools.New(c) }},
		{"connection", func(t *testing.T, c *gophercloud.ServiceClient) *subnetpools.API {
			s := networkSubnetBodyConnection(t, c)
			if s.API.SubnetPools.RawClient() != s.RawClient() {
				t.Fatal("SubnetPool did not share the cached network client")
			}
			return s.API.SubnetPools
		}},
	}
}

const subnetPoolListPath = networkExtensionPrefix + "subnetpools"

func subnetPoolListRow(t *testing.T, id, name string, fields map[string]json.RawMessage) string {
	t.Helper()
	body := map[string]json.RawMessage{"id": json.RawMessage(fmt.Sprintf("%q", id)), "name": json.RawMessage(fmt.Sprintf("%q", name)), "project_id": json.RawMessage(`"project-independent"`), "description": json.RawMessage(`"native-description"`), "default_prefixlen": json.RawMessage(`"24"`), "min_prefixlen": json.RawMessage(`16`), "max_prefixlen": json.RawMessage(`"28"`), "ip_version": json.RawMessage(`4`), "is_default": json.RawMessage(`false`), "shared": json.RawMessage(`false`)}
	for key, value := range fields {
		if value == nil {
			delete(body, key)
		} else {
			body[key] = value
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
func subnetPoolListPage(rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,"subnetpools_links":[{"rel":"next","href":%q}]`, next)
	}
	return `{"subnetpools":[` + rows + `]` + links + `}`
}
func subnetPoolListWant(t *testing.T, values []*subnetpools.SubnetPool, err error, ids ...string) {
	t.Helper()
	actual := make([]string, 0, len(values))
	for _, v := range values {
		actual = append(actual, v.ID)
	}
	if err != nil || !reflect.DeepEqual(actual, append([]string{}, ids...)) {
		t.Fatal("actual native collection result", actual, err, "want", ids)
	}
}

func TestSubnetPoolListFiltersEntireQueryAndAliasDescriptor(t *testing.T) {
	canonical := subnetPoolListQueries()
	accepted := make(map[string]string)
	for key, wire := range canonical {
		accepted[key], accepted[wire] = wire, wire
	}
	if len(canonical) != 16 || len(accepted) != 20 {
		t.Fatal("independent descriptor changed", canonical, accepted)
	}
	for _, f := range subnetPoolListFixtures() {
		for key, wire := range accepted {
			t.Run(f.name+"/"+key, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{wire: {"server-value"}}) {
						t.Error("semantic query alias changed", key, r.URL)
					}
					// Returned name/tags/default/shared intentionally differ:
					// all of these queried fields must remain server-only.
					testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "server-returned", "Different", map[string]json.RawMessage{"tags": json.RawMessage(`["different-tag"]`)}), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter(key, "server-value"))
				subnetPoolListWant(t, values, err, "server-returned")
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
					cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
						want := make(url.Values)
						if tc.want != nil {
							want[wire] = tc.want
						}
						if !reflect.DeepEqual(r.URL.Query(), want) {
							t.Error("canonical bulk presence did not override wire alias", r.URL.Query(), want)
						}
						testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "result", "Different", nil), ""))
					})
					values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{key: tc.value, wire: json.RawMessage(`{`)}))
					subnetPoolListWant(t, values, err, "result")
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
					cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
						if !reflect.DeepEqual(r.URL.Query(), url.Values{wire: {want}}) {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "result", "Different", nil), ""))
					})
					values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), options...)
					subnetPoolListWant(t, values, err, "result")
				})
			}
		}
		t.Run(f.name+"/scalar-encoding", func(t *testing.T) {
			cloud := testcloud.New(t)
			want := url.Values{"fields": {"id", "name", "true", "false", "9007199254740993", ""}, "shared": {"false"}, "is_default": {"true"}, "limit": {"2e1"}, "name": {""}}
			cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "result", "Different", nil), ""))
			})
			values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{"fields": []any{"id", nil, "name", true, false, json.Number("9007199254740993"), ""}, "is_shared": false, "is_default": true, "limit": json.Number("2e1"), "name": "", "description": nil, "marker": []any{}}))
			subnetPoolListWant(t, values, err, "result")
		})
	}
}

func TestSubnetPoolListFiltersRawPresenceIntegersAndTimestamps(t *testing.T) {
	properties := map[string]string{"id": "id", "created_at": "created_at", "default_prefix_length": "default_prefixlen", "default_quota": "default_quota", "maximum_prefix_length": "max_prefixlen", "minimum_prefix_length": "min_prefixlen", "prefixes": "prefixes", "revision_number": "revision_number", "tenant_id": "tenant_id", "updated_at": "updated_at"}
	target := map[string]json.RawMessage{"id": json.RawMessage(`"target"`), "created_at": json.RawMessage(`"2025-01-02T03:04:05+00:00"`), "updated_at": json.RawMessage(`"2025-01-03T03:04:05+00:00"`), "default_prefixlen": json.RawMessage(`"24"`), "min_prefixlen": json.RawMessage(`1.6e1`), "max_prefixlen": json.RawMessage(`28.0`), "default_quota": json.RawMessage(`256`), "revision_number": json.RawMessage(`9`), "tenant_id": json.RawMessage(`"tenant-original"`), "prefixes": json.RawMessage(`["192.0.2.1/24",null,"2001:0db8::/64"]`)}
	filters := map[string]any{"id": "target", "created_at": "2025-01-02T03:04:05+00:00", "updated_at": "2025-01-03T03:04:05+00:00", "default_prefix_length": 24, "minimum_prefix_length": 16, "maximum_prefix_length": 28, "default_quota": 256, "revision_number": 9, "tenant_id": "tenant-original", "prefixes": json.RawMessage(`["192.0.2.1/24",null,"2001:0db8::/64"]`)}
	if len(properties) != 10 || len(filters) != 10 {
		t.Fatal("independent Body descriptor changed")
	}
	for _, f := range subnetPoolListFixtures() {
		t.Run(f.name+"/all-ten", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("Body filter leaked to query", r.URL)
				}
				testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "target", "Different", target), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for key, value := range filters {
				values, err := a.Resources.All(context.Background(), resource.WithFilter(key, value))
				subnetPoolListWant(t, values, err, "target")
			}
			values, err := a.Resources.All(context.Background(), resource.WithFilters(filters))
			subnetPoolListWant(t, values, err, "target")
			if values[0].ProjectID != "project-independent" || values[0].TenantID != "tenant-original" || !reflect.DeepEqual(values[0].Prefixes, []string{"192.0.2.1/24", "", "2001:0db8::/64"}) || values[0].DefaultPrefixLen != 24 || values[0].MinPrefixLen != 16 || values[0].MaxPrefixLen != 28 {
				t.Fatal("raw matching changed native projection", values[0])
			}
			if calls.Load() != 11 {
				t.Fatal(calls.Load())
			}
		})
		for _, tc := range []struct {
			name, field, raw string
			filter           any
			match            bool
		}{
			{"prefix-missing", "prefixes", "missing", nil, true}, {"prefix-null", "prefixes", "null", nil, true}, {"prefix-empty", "prefixes", "[]", nil, false}, {"prefix-empty-array", "prefixes", "[]", []string{}, true},
			{"prefix-null-element", "prefixes", `["192.0.2.1/24",null]`, json.RawMessage(`["192.0.2.1/24",null]`), true}, {"prefix-null-not-empty-string", "prefixes", `["192.0.2.1/24",null]`, []string{"192.0.2.1/24", ""}, false},
			{"prefix-order", "prefixes", `["one","two"]`, []string{"two", "one"}, false}, {"prefix-length", "prefixes", `["one","two"]`, []string{"one"}, false}, {"prefix-no-cidr-normalization", "prefixes", `["192.0.2.1/24"]`, []string{"192.0.2.0/24"}, false}, {"prefix-scalar-filter", "prefixes", `["one"]`, "one", false},
			{"tenant-missing", "tenant_id", "missing", nil, true}, {"tenant-null", "tenant_id", "null", nil, true}, {"tenant-empty-distinct", "tenant_id", `""`, nil, false}, {"tenant-empty", "tenant_id", `""`, "", true}, {"tenant-no-project-fallback", "tenant_id", "missing", "project-independent", false},
			{"id-null", "id", "null", nil, true}, {"id-missing", "id", "missing", nil, true}, {"id-empty-distinct", "id", `""`, nil, false},
			{"quota-null", "default_quota", "null", nil, true}, {"quota-missing", "default_quota", "missing", nil, true}, {"quota-zero-distinct", "default_quota", "0", nil, false}, {"revision-null", "revision_number", "null", nil, true}, {"revision-missing", "revision_number", "missing", nil, true},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				fields := map[string]json.RawMessage{tc.field: json.RawMessage(tc.raw)}
				if tc.raw == "missing" {
					fields[tc.field] = nil
				}
				cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.RawQuery != "" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "target", "Different", fields), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithBodyFilter(tc.field, tc.filter))
				var want []string
				if tc.match {
					want = []string{"target"}
					if tc.field == "id" {
						want = []string{""}
					}
				}
				subnetPoolListWant(t, values, err, want...)
			})
		}
		for property, field := range map[string]string{"default_prefix_length": "default_prefixlen", "minimum_prefix_length": "min_prefixlen", "maximum_prefix_length": "max_prefixlen", "default_quota": "default_quota", "revision_number": "revision_number"} {
			for _, tc := range []struct {
				name, raw  string
				filter     any
				match, bad bool
			}{
				{"integer", "24", 24, true, false}, {"negative", "-24", -24, true, false}, {"string-filter", "24", "24", false, false}, {"bool-filter", "1", true, false, false},
				{"integral-decimal", "24.0", 24, true, false}, {"integral-exponent", "2.4e1", 24, true, false}, {"numeric-string", `"24"`, 24, true, false}, {"signed-string", `"-24"`, -24, true, false},
				{"fraction", "24.5", 24, false, true}, {"large-exact", "9007199254740993", json.Number("9007199254740993"), true, false}, {"large-rounded-mismatch", "9007199254740993", json.Number("9007199254740992"), false, false},
			} {
				prefix := strings.HasSuffix(field, "prefixlen")
				if !prefix && (tc.name != "integer" && tc.name != "negative" && tc.name != "string-filter" && tc.name != "bool-filter") {
					continue
				}
				t.Run(f.name+"/"+property+"/"+tc.name, func(t *testing.T) {
					cloud := testcloud.New(t)
					cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, _ *http.Request) {
						testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "target", "Different", map[string]json.RawMessage{field: json.RawMessage(tc.raw)}), ""))
					})
					a := f.open(t, networkExtensionClient(cloud))
					values, err := a.Resources.All(context.Background(), resource.WithFilter(property, tc.filter))
					if tc.bad {
						if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal("selected nonintegral value was accepted", values, err)
						}
						return
					}
					var want []string
					if tc.match {
						want = []string{"target"}
					}
					subnetPoolListWant(t, values, err, want...)
					// Large numeric response matching is raw; the architecture-dependent
					// native int/float64 projection is deliberately not asserted as precise.
				})
			}
		}
		for _, tc := range []struct {
			name, created, updated, filter string
			match                          bool
		}{
			{"old-no-z", "2025-01-02T03:04:05", "2025-01-03T03:04:05", "2025-01-02T03:04:05", true},
			{"new-offset", "2025-01-02T03:04:05+00:00", "2025-01-03T03:04:05Z", "2025-01-02T03:04:05+00:00", true},
			{"same-instant-original-text", "2025-01-02T03:04:05+00:00", "2025-01-03T03:04:05Z", "2025-01-02T03:04:05Z", false},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, _ *http.Request) {
					testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "target", "Different", map[string]json.RawMessage{"created_at": json.RawMessage(fmt.Sprintf("%q", tc.created)), "updated_at": json.RawMessage(fmt.Sprintf("%q", tc.updated))}), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter("created_at", tc.filter))
				var want []string
				if tc.match {
					want = []string{"target"}
				}
				subnetPoolListWant(t, values, err, want...)
			})
		}
	}
}
func TestSubnetPoolListFiltersSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	for _, f := range subnetPoolListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			prefixes := []any{"192.0.2.1/24", nil}
			fields := []any{"id", "name", nil}
			shared := false
			tenant := "tenant-original"
			number := json.RawMessage(`24.0`)
			input := map[string]any{"prefixes": prefixes, "fields": fields, "is_shared": &shared, "tenant_id": &tenant, "default_prefix_length": number}
			bulk := resource.WithFilters(input)
			input["prefixes"], prefixes[0], prefixes[1], fields[0], shared, tenant = nil, "changed", true, "changed", true, "changed"
			copy(number, []byte(`25.0`))
			var expected atomic.Value
			expected.Store(url.Values{"fields": {"id", "name"}, "shared": {"false"}, "vendor": {"kept"}})
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), expected.Load().(url.Values)) {
					t.Error("option snapshot changed", r.URL.Query())
				}
				rows := subnetPoolListRow(t, "target", "Different", map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant-original"`), "prefixes": json.RawMessage(`["192.0.2.1/24",null]`)}) + "," + subnetPoolListRow(t, "other", "Different", map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant-original"`), "prefixes": json.RawMessage(`[]`)})
				testcloud.JSON(w, 200, subnetPoolListPage(rows, ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			opts := []resource.ListOption{resource.WithQuery("vendor", "kept"), bulk}
			seq := a.Resources.List(context.Background(), opts...)
			replacement := append([]resource.ListOption(nil), opts...)
			replacement[1] = resource.WithFilters(nil)
			var initial []*subnetpools.SubnetPool
			for v, err := range seq {
				if err != nil {
					t.Fatal(err)
				}
				initial = append(initial, v)
			}
			subnetPoolListWant(t, initial, nil, "target")
			var wg sync.WaitGroup
			for range 8 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					values, err := a.Resources.All(context.Background(), opts...)
					subnetPoolListWant(t, values, err, "target")
				}()
			}
			wg.Wait()
			expected.Store(url.Values{"vendor": {"kept"}})
			for _, clear := range []resource.ListOption{resource.WithFilters(nil), resource.WithFilters(map[string]any{})} {
				values, err := a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), opts...), clear, resource.WithBodyFilter("tenant_id", "tenant-original"))...)
				subnetPoolListWant(t, values, err, "target", "other")
			}
			for _, tc := range []struct {
				options []resource.ListOption
				want    []string
			}{
				{[]resource.ListOption{resource.WithFilter("fields", math.NaN()), resource.WithFilter("fields", nil), resource.WithFilter("prefixes", make(chan int)), resource.WithFilter("prefixes", json.RawMessage(`["192.0.2.1/24",null]`))}, []string{"target"}},
				{[]resource.ListOption{resource.WithFilter("fields", map[string]any{"bad": true}), resource.WithFilter("default_prefix_length", json.RawMessage(`{`)), resource.WithFilter("headers", true), resource.WithFilters(nil)}, []string{"target", "other"}},
				{[]resource.ListOption{resource.WithFilter("prefixes", []any{}), resource.WithFilter("prefixes", json.RawMessage(`["192.0.2.1/24",null]`))}, []string{"target"}},
			} {
				values, err := a.Resources.All(context.Background(), append([]resource.ListOption{resource.WithQuery("vendor", "kept")}, tc.options...)...)
				subnetPoolListWant(t, values, err, tc.want...)
			}
			for _, property := range []string{"default_prefix_length", "minimum_prefix_length", "maximum_prefix_length"} {
				wire := map[string]string{"default_prefix_length": "default_prefixlen", "minimum_prefix_length": "min_prefixlen", "maximum_prefix_length": "max_prefixlen"}[property]
				want := map[string]int{"default_prefixlen": 24, "min_prefixlen": 16, "max_prefixlen": 28}[wire]
				values, err := a.Resources.All(context.Background(), resource.WithQuery("vendor", "kept"), resource.WithBodyFilter(property, 0), resource.WithBodyFilter(wire, want))
				subnetPoolListWant(t, values, err, "target", "other")
			}
			if calls.Load() != 17 {
				t.Fatal("fresh independent option reuse requests", calls.Load())
			}
		})
	}
}

func TestSubnetPoolListFiltersLazyPreflightAndNamespaceCollisions(t *testing.T) {
	for _, f := range subnetPoolListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("unknown fields leaked", r.URL)
				}
				testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "result", "Different", nil), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for _, pair := range [][]resource.ListOption{
				{resource.WithFilter("is_shared", false), resource.WithQuery("shared", "false")}, {resource.WithFilter("any_tags", nil), resource.WithQuery("tags-any", "")}, {resource.WithFilter("fields", []any{}), resource.WithQuery("fields", "")}, {resource.WithFilter("name", nil), resource.WithName("Target")}, {resource.WithFilter("limit", 2), resource.WithPageSize(2)},
				{resource.WithFilter("id", nil), resource.WithBodyFilter("id", nil)}, {resource.WithFilter("tenant_id", "tenant"), resource.WithBodyFilter("tenant_id", "tenant")}, {resource.WithFilter("default_prefix_length", 24), resource.WithBodyFilter("default_prefixlen", 24)},
			} {
				for _, reverse := range []bool{false, true} {
					opts := append([]resource.ListOption(nil), pair...)
					if reverse {
						opts[0], opts[1] = opts[1], opts[0]
					}
					values, err := a.Resources.All(context.Background(), opts...)
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
						t.Fatal("collision did HTTP", values, err, calls.Load())
					}
				}
			}
			invalid := []resource.ListOption{resource.WithFilter("fields", map[string]any{"id": true}), resource.WithFilter("fields", [][]string{{"id"}}), resource.WithFilter("prefixes", math.NaN()), resource.WithBodyFilter("tenant_id", json.RawMessage(`{`)), resource.WithBodyFilter("name", "Different"), resource.WithBodyFilter("is_shared", false), resource.WithBodyFilter("project_id", "project"), resource.WithBodyFilter("tags", []string{}), resource.WithStatus("ACTIVE"), resource.WithMaxItems(-1), resource.WithBodyFilters(map[string]any{"default_prefix_length": 24, "default_prefixlen": 24})}
			for _, key := range []string{"max_items", "paginated", "base_path", "allow_unknown_params", "session", "headers", "microversion", "resource_type", "jmespath_filters"} {
				invalid = append(invalid, resource.WithFilter(key, nil))
			}
			for _, option := range invalid {
				seq := a.Resources.List(context.Background(), option)
				if calls.Load() != 0 {
					t.Fatal("constructing iterator did HTTP")
				}
				seen := 0
				for value, err := range seq {
					seen++
					if value != nil || (!errors.Is(err, resource.ErrInvalidOption) && !errors.Is(err, resource.ErrUnsupported)) {
						t.Fatal(value, err)
					}
				}
				if seen != 1 || calls.Load() != 0 {
					t.Fatal("preflight did not yield one error", seen, calls.Load())
				}
			}
			values, err := a.Resources.All(context.Background(), resource.WithFilters(map[string]any{"default_prefixlen": make(chan int), "min_prefixlen": json.RawMessage(`{`), "max_prefixlen": math.NaN(), "status": func() {}, "unknown": make(chan int)}))
			subnetPoolListWant(t, values, err, "result")
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			values, err = a.Resources.All(ctx, resource.WithFilters(nil))
			if values != nil || !errors.Is(err, context.Canceled) || calls.Load() != 1 {
				t.Fatal(values, err, calls.Load())
			}
			_, err = ports.New(networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(nil))
			if !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatal("unaudited binding gained semantic filter", err, calls.Load())
			}
		})
	}
}
func TestSubnetPoolListFiltersControlsAndNativeWholePageFailures(t *testing.T) {
	modes := []string{"raw-cap-before-body", "raw-cap-before-name", "single-page", "break", "null-row", "empty-row", "late-null", "cap-null", "query-null", "selected-fraction", "fraction-after-cap", "fraction-unselected", "bad-prefix-array", "bad-prefix-element", "bad-tenant", "bad-bool", "bad-quota", "bad-revision", "bad-tags", "bad-date", "mixed-date", "bad-id"}
	for _, field := range []string{"default_prefixlen", "min_prefixlen", "max_prefixlen"} {
		for _, kind := range []string{"missing", "null", "bool", "object", "string"} {
			modes = append(modes, field+"/"+kind)
		}
	}
	for _, f := range subnetPoolListFixtures() {
		for _, mode := range modes {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				foreign := testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); w.WriteHeader(500) })
				rows := subnetPoolListRow(t, "first", "Different", map[string]json.RawMessage{"prefixes": json.RawMessage(`[]`)}) + "," + subnetPoolListRow(t, "target", "Target", map[string]json.RawMessage{"prefixes": json.RawMessage(`["one"]`)})
				opts := []resource.ListOption{resource.WithFilter("prefixes", []any{})}
				nativeFailure := false
				switch mode {
				case "raw-cap-before-body":
					opts = []resource.ListOption{resource.WithFilter("prefixes", []string{"one"}), resource.WithMaxItems(1)}
				case "raw-cap-before-name":
					opts = append(opts, resource.WithName("Target"), resource.WithMaxItems(1))
				case "single-page":
					opts = append(opts, resource.WithPaginated(false))
				case "break":
				case "null-row", "query-null":
					rows = "null"
					nativeFailure = true
				case "empty-row":
					rows = "{}"
					nativeFailure = true
				case "late-null", "cap-null":
					rows = subnetPoolListRow(t, "first", "Different", map[string]json.RawMessage{"prefixes": json.RawMessage(`[]`)}) + ",null"
					nativeFailure = true
					if mode == "cap-null" {
						opts = append(opts, resource.WithMaxItems(1))
					}
				case "selected-fraction", "fraction-after-cap", "fraction-unselected":
					rows = subnetPoolListRow(t, "first", "Target", map[string]json.RawMessage{"prefixes": json.RawMessage(`[]`)}) + "," + subnetPoolListRow(t, "fraction", "Other", map[string]json.RawMessage{"prefixes": json.RawMessage(`[]`), "default_prefixlen": json.RawMessage(`24.5`)})
					opts = []resource.ListOption{resource.WithFilter("default_prefix_length", 24), resource.WithName("Target")}
					if mode == "fraction-after-cap" {
						opts = append(opts, resource.WithMaxItems(1))
					}
					if mode == "fraction-unselected" {
						opts = []resource.ListOption{resource.WithFilter("prefixes", []any{}), resource.WithPaginated(false)}
					}
				default:
					fields := map[string]json.RawMessage{}
					if strings.Contains(mode, "/") {
						parts := strings.Split(mode, "/")
						raw := map[string]string{"null": "null", "bool": "false", "object": "{}", "string": `"not-int"`}[parts[1]]
						if parts[1] == "missing" {
							fields[parts[0]] = nil
						} else {
							fields[parts[0]] = json.RawMessage(raw)
						}
					} else {
						field, raw := "prefixes", "{}"
						switch mode {
						case "bad-prefix-element":
							raw = `[true]`
						case "bad-tenant":
							field, raw = "tenant_id", "1"
						case "bad-bool":
							field, raw = "shared", `"false"`
						case "bad-quota":
							field, raw = "default_quota", `"24"`
						case "bad-revision":
							field, raw = "revision_number", `"9"`
						case "bad-tags":
							field, raw = "tags", `[1]`
						case "bad-date":
							field, raw = "created_at", `"not-time"`
						case "mixed-date":
							field, raw = "created_at", `"2025-01-02T03:04:05"`
							fields["updated_at"] = json.RawMessage(`"2025-01-03T03:04:05Z"`)
						case "bad-id":
							field, raw = "id", "false"
						}
						fields[field] = json.RawMessage(raw)
					}
					rows += "," + subnetPoolListRow(t, "invalid", "Other", fields)
					opts = append(opts, resource.WithMaxItems(1))
					nativeFailure = true
				}
				if mode == "query-null" {
					opts = []resource.ListOption{resource.WithFilter("name", "server-only"), resource.WithMaxItems(1)}
				}
				cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Has("limit") {
						t.Error("raw cap invented wire limit", r.URL)
					}
					testcloud.JSON(w, 200, subnetPoolListPage(rows, foreign.Server.URL+"/foreign"))
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
					subnetPoolListWant(t, values, err)
				case mode == "single-page" || mode == "fraction-after-cap":
					subnetPoolListWant(t, values, err, "first")
				case mode == "fraction-unselected":
					subnetPoolListWant(t, values, err, "first", "fraction")
					if values[1].DefaultPrefixLen != 24 {
						t.Fatal("unselected native fractional truncation changed", values[1])
					}
				case mode == "selected-fraction":
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal("selected fraction hidden by later name mismatch", values, err)
					}
				case nativeFailure:
					if values != nil || err == nil || errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal("native whole-page error lost or relabeled", values, err)
					}
					switch {
					case strings.HasSuffix(mode, "/string"):
						var cause *strconv.NumError
						if !errors.As(err, &cause) {
							t.Fatal("native Atoi cause lost", err)
						}
					case mode == "bad-date" || mode == "mixed-date":
						var cause *time.ParseError
						if !errors.As(err, &cause) {
							t.Fatal("native timestamp cause lost", err)
						}
					case strings.Contains(mode, "/") || strings.Contains(mode, "null") || mode == "empty-row":
						if !strings.Contains(err.Error(), "unexpected type") {
							t.Fatal("required native prefix cause lost", err)
						}
					default:
						var cause *json.UnmarshalTypeError
						if !errors.As(err, &cause) {
							t.Fatal("native typed decode cause lost", err)
						}
					}
				default:
					t.Fatal("unhandled mode", mode)
				}
				if calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal("current page failure/cap fetched continuation", calls.Load(), followed.Load())
				}
			})
		}
	}
}

func TestSubnetPoolListFiltersPagingLiveSourceAndNativeStatus(t *testing.T) {
	for _, f := range subnetPoolListFixtures() {
		for _, mode := range []string{"complete", "all-first-filtered", "duplicates", "late-http", "late-decode", "cycle", "foreign", "cancel", "wrong-link-shape", "204", "json-204", "201"} {
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
				cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
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
							testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, "invalid", "Different", map[string]json.RawMessage{"prefixes": json.RawMessage(`{}`)}), ""))
							return
						}
						id := "second"
						if mode == "duplicates" {
							id = "first"
						}
						testcloud.JSON(w, 200, subnetPoolListPage(subnetPoolListRow(t, id, "Different", map[string]json.RawMessage{"prefixes": json.RawMessage(`[]`)}), ""))
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
					next := cloud.Server.URL + subnetPoolListPath + "?marker=next&shared=false&fields=id&fields=name&status=vendor-status"
					if mode == "cycle" {
						next = cloud.Server.URL + r.URL.String()
					}
					if mode == "foreign" {
						next = foreign.Server.URL + "/foreign"
					}
					row := subnetPoolListRow(t, "first", "Different", map[string]json.RawMessage{"prefixes": json.RawMessage(`[]`)})
					if mode == "all-first-filtered" {
						row = subnetPoolListRow(t, "dropped", "Different", map[string]json.RawMessage{"prefixes": json.RawMessage(`["non-match"]`)})
					}
					if mode == "wrong-link-shape" {
						testcloud.JSON(w, 300, `{"subnetpools":[`+row+`],"links":{"next":`+fmt.Sprintf("%q", next)+`},"next":`+fmt.Sprintf("%q", next)+`}`)
						return
					}
					testcloud.JSON(w, 300, subnetPoolListPage(row, next))
				})
				a := f.open(t, c)
				values, err := a.Resources.All(ctx, resource.WithFilter("prefixes", []any{}), resource.WithFilter("is_shared", false), resource.WithFilter("fields", []string{"id", "name"}), resource.WithQuery("status", "vendor-status"))
				switch mode {
				case "complete":
					subnetPoolListWant(t, values, err, "first", "second")
				case "all-first-filtered":
					subnetPoolListWant(t, values, err, "second")
				case "duplicates":
					subnetPoolListWant(t, values, err, "first", "first")
				case "wrong-link-shape":
					subnetPoolListWant(t, values, err, "first")
				case "204":
					subnetPoolListWant(t, values, err)
				default:
					if values != nil || err == nil {
						t.Fatal("terminal page observation returned partial All", values, err)
					}
					if mode == "cycle" && !errors.Is(err, resource.ErrPaginationCycle) || mode == "cancel" && !errors.Is(err, context.Canceled) || mode == "json-204" && !errors.Is(err, io.EOF) || mode == "201" && !gophercloud.ResponseCodeIs(err, 201) {
						t.Fatal(mode, err)
					}
				}
				wantCalls, wantMiddleware, wantFollow := int32(1), int32(1), int32(0)
				if mode == "complete" || mode == "all-first-filtered" || mode == "duplicates" || mode == "late-http" || mode == "late-decode" {
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

func TestSubnetPoolListFiltersNativeAndIdentitySurfaceIsolation(t *testing.T) {
	for _, f := range subnetPoolListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets, changes atomic.Int32
			want := url.Values{"id": {"wire-id"}, "tenant_id": {"wire-tenant"}, "prefixes": {"wire-prefixes"}, "default_prefixlen": {"wire-size"}, "default_prefix_length": {"raw-attribute"}, "status": {"vendor-status"}, "name": {"server-pattern"}, "vendor": {"kept"}}
			cloud.Mux.HandleFunc("GET "+subnetPoolListPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("raw/native query acquired semantic classification", r.URL.Query(), want)
				}
				rows := subnetPoolListRow(t, "target", "Target", map[string]json.RawMessage{"prefixes": json.RawMessage(`[]`)}) + "," + subnetPoolListRow(t, "other", "Other", map[string]json.RawMessage{"prefixes": json.RawMessage(`["one"]`)})
				testcloud.JSON(w, 200, subnetPoolListPage(rows, ""))
			})
			cloud.Mux.HandleFunc("GET "+subnetPoolListPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"id": {"wire-id"}, "prefixes": {"wire-prefixes"}, "default_prefix_length": {"raw-attribute"}, "status": {"vendor-status"}}) {
					t.Error("FindIdentity acquired semantic classification", r.URL)
				}
				testcloud.JSON(w, 200, `{"subnetpool":`+subnetPoolListRow(t, "lookup", "Different", nil)+`}`)
			})
			for _, action := range []string{"add_prefixes", "remove_prefixes"} {
				cloud.Mux.HandleFunc("PUT "+subnetPoolListPath+"/lookup/"+action, func(w http.ResponseWriter, r *http.Request) {
					changes.Add(1)
					if r.URL.RawQuery != "" {
						t.Error("list filters reached prefix operation", r.URL)
					}
					raw, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					var fields map[string]json.RawMessage
					if err = json.Unmarshal(raw, &fields); err != nil || len(fields) != 1 || string(fields["prefixes"]) != `["192.0.2.1/24"]` {
						t.Error("native prefix operation changed", string(raw), err)
					}
					testcloud.JSON(w, 200, `{"prefixes":["192.0.2.1/24"]}`)
				})
			}
			a := f.open(t, networkExtensionClient(cloud))
			var raw []resource.ListOption
			var native []subnetpools.ListOption
			for key, values := range want {
				raw = append(raw, resource.WithQuery(key, values[0]))
				native = append(native, subnetpools.WithListQuery(key, values[0]))
			}
			values, err := a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilter("prefixes", []any{}), resource.WithFilter("id", "target"))...)
			subnetPoolListWant(t, values, err, "target")
			values, err = a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilters(nil))...)
			subnetPoolListWant(t, values, err, "target", "other")
			var nativeIDs []string
			for value, err := range a.List(context.Background(), native...) {
				if err != nil {
					t.Fatal(err)
				}
				nativeIDs = append(nativeIDs, value.ID)
			}
			if !reflect.DeepEqual(nativeIDs, []string{"target", "other"}) {
				t.Fatal("native typed List gained local predicates", nativeIDs)
			}
			value, err := a.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindQuery("id", "wire-id"), resource.WithIdentityFindQuery("prefixes", "wire-prefixes"), resource.WithIdentityFindQuery("default_prefix_length", "raw-attribute"), resource.WithIdentityFindQuery("status", "vendor-status"))
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
			named = append(named, resource.WithName("Target"), resource.WithBodyFilter("prefixes", []any{}))
			values, err = a.Resources.All(context.Background(), named...)
			subnetPoolListWant(t, values, err, "target")
			before := lists.Load()
			values, err = a.Resources.All(context.Background(), resource.WithStatus("ACTIVE"), resource.WithBodyFilter("prefixes", []any{}))
			if values != nil || !errors.Is(err, resource.ErrUnsupported) || lists.Load() != before {
				t.Fatal("native SubnetPool fabricated status", values, err, lists.Load())
			}
			prefixes, err := a.AddPrefixes(context.Background(), "lookup", subnetpools.PrefixesOpsOpts{Prefixes: []string{"192.0.2.1/24"}})
			if err != nil || !reflect.DeepEqual(prefixes, []string{"192.0.2.1/24"}) {
				t.Fatal("AddPrefixes native result changed", prefixes, err)
			}
			prefixes, err = a.RemovePrefixes(context.Background(), "lookup", subnetpools.PrefixesOpsOpts{Prefixes: []string{"192.0.2.1/24"}})
			if err != nil || !reflect.DeepEqual(prefixes, []string{"192.0.2.1/24"}) || changes.Load() != 2 {
				t.Fatal("RemovePrefixes native result changed", prefixes, err, changes.Load())
			}
		})
	}
}
