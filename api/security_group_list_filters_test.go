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

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	securitygroups "github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/ports"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Independent pinned SecurityGroup query table. Inherited revision_number and
// both tenant/project keys are server-only, rather than local Body selectors.
func securityGroupListQueries() map[string]string {
	return map[string]string{"description": "description", "fields": "fields", "id": "id", "name": "name", "stateful": "stateful", "project_id": "project_id", "tenant_id": "tenant_id", "revision_number": "revision_number", "sort_dir": "sort_dir", "sort_key": "sort_key", "limit": "limit", "marker": "marker", "is_shared": "shared", "tags": "tags", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any"}
}

type securityGroupListFixture struct {
	name string
	open func(*testing.T, *gophercloud.ServiceClient) *securitygroups.API
}

func securityGroupListFixtures() []securityGroupListFixture {
	return []securityGroupListFixture{
		{"leaf", func(_ *testing.T, c *gophercloud.ServiceClient) *securitygroups.API { return securitygroups.New(c) }},
		{"connection", func(t *testing.T, c *gophercloud.ServiceClient) *securitygroups.API {
			s := networkSubnetBodyConnection(t, c)
			a := s.API.SecurityGroups
			if a.RawClient() != s.RawClient() || a.RawClient().ProviderClient != c.ProviderClient {
				t.Fatal("SecurityGroup facade did not share cached client")
			}
			return a
		}},
	}
}

const securityGroupListPath = networkExtensionPrefix + "security-groups"

func securityGroupListRow(t *testing.T, id, name string, fields map[string]json.RawMessage) string {
	t.Helper()
	body := map[string]json.RawMessage{"id": json.RawMessage(fmt.Sprintf("%q", id)), "name": json.RawMessage(fmt.Sprintf("%q", name)), "project_id": json.RawMessage(`"native-project"`), "tenant_id": json.RawMessage(`"native-tenant"`), "stateful": json.RawMessage(`false`), "revision_number": json.RawMessage(`9`)}
	for k, v := range fields {
		if v == nil {
			delete(body, k)
		} else {
			body[k] = v
		}
	}
	b, e := json.Marshal(body)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func securityGroupListPage(rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,"security_groups_links":[{"rel":"next","href":%q}]`, next)
	}
	return `{"security_groups":[` + rows + `]` + links + `}`
}
func securityGroupListWant(t *testing.T, values []*securitygroups.SecGroup, e error, ids ...string) {
	t.Helper()
	actual := make([]string, 0, len(values))
	for _, v := range values {
		actual = append(actual, v.ID)
	}
	if e != nil || !reflect.DeepEqual(actual, append([]string{}, ids...)) {
		t.Fatal("actual native collection", actual, e, "want", ids)
	}
}
func securityGroupListTarget() (map[string]json.RawMessage, map[string]any) {
	rules := json.RawMessage(`[{"id":"rule","direction":"ingress","ethertype":"IPv4","security_group_id":"target","port_range_min":80,"port_range_max":80,"protocol":"tcp","tenant_id":"rule-tenant","project_id":"rule-project","revision_number":7,"created_at":"2025-01-02T03:04:05Z","updated_at":"2025-01-03T03:04:05Z","vendor":{"n":9007199254740993,"nullable":null}},null]`)
	fields := map[string]json.RawMessage{"created_at": json.RawMessage(`"2025-01-02T03:04:05+00:00"`), "updated_at": json.RawMessage(`"2025-01-03T03:04:05+00:00"`), "security_group_rules": rules}
	return fields, map[string]any{"created_at": "2025-01-02T03:04:05+00:00", "updated_at": "2025-01-03T03:04:05+00:00", "security_group_rules": append(json.RawMessage(nil), rules...)}
}

func TestSecurityGroupListFiltersEntireQueryAndAliasDescriptor(t *testing.T) {
	canonical := securityGroupListQueries()
	accepted := map[string]string{}
	for k, w := range canonical {
		accepted[k], accepted[w] = w, w
	}
	if len(canonical) != 17 || len(accepted) != 21 {
		t.Fatal(canonical, accepted)
	}
	for _, f := range securityGroupListFixtures() {
		for k, w := range accepted {
			t.Run(f.name+"/"+k, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(wr http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{w: {"server-value"}}) {
						t.Error(k, r.URL)
					}
					testcloud.JSON(wr, 200, securityGroupListPage(securityGroupListRow(t, "result", "Different", nil), ""))
				})
				a := f.open(t, networkExtensionClient(cloud))
				rows, err := a.Resources.All(context.Background(), resource.WithFilter(k, "server-value"))
				securityGroupListWant(t, rows, err, "result")
				if calls.Load() != 1 {
					t.Fatal(calls.Load())
				}
			})
		}
		for k, w := range canonical {
			if k == w {
				continue
			}
			for _, tc := range []struct {
				name  string
				value any
				want  []string
			}{{"nil", nil, nil}, {"false", false, []string{"false"}}, {"empty", "", []string{""}}, {"empty-array", []any{}, nil}, {"repeats", []any{"first", nil, "last"}, []string{"first", "last"}}} {
				t.Run(f.name+"/precedence/"+k+"/"+tc.name, func(t *testing.T) {
					cloud := testcloud.New(t)
					cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(wr http.ResponseWriter, r *http.Request) {
						want := url.Values{}
						if tc.want != nil {
							want[w] = tc.want
						}
						if !reflect.DeepEqual(r.URL.Query(), want) {
							t.Error(r.URL.Query(), want)
						}
						testcloud.JSON(wr, 200, securityGroupListPage(securityGroupListRow(t, "result", "Different", nil), ""))
					})
					rows, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{k: tc.value, w: json.RawMessage(`{`)}))
					securityGroupListWant(t, rows, err, "result")
				})
			}
			for _, reverse := range []bool{false, true} {
				t.Run(f.name+"/last-wins/"+k+fmt.Sprint(reverse), func(t *testing.T) {
					cloud := testcloud.New(t)
					opts := []resource.ListOption{resource.WithFilter(w, "wire-first"), resource.WithFilter(k, "canonical-last")}
					want := "canonical-last"
					if reverse {
						opts[0], opts[1], want = opts[1], opts[0], "wire-first"
					}
					cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(wr http.ResponseWriter, r *http.Request) {
						if r.URL.Query().Get(w) != want {
							t.Error(r.URL)
						}
						testcloud.JSON(wr, 200, securityGroupListPage(securityGroupListRow(t, "result", "Different", nil), ""))
					})
					rows, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), opts...)
					securityGroupListWant(t, rows, err, "result")
				})
			}
		}
		t.Run(f.name+"/scalar-encoding", func(t *testing.T) {
			cloud := testcloud.New(t)
			want := url.Values{"fields": {"id", "true", "false", "9007199254740993", ""}, "shared": {"false"}, "name": {""}, "limit": {"2e1"}, "stateful": {"true"}}
			cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(w http.ResponseWriter, r *http.Request) {
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, securityGroupListPage(securityGroupListRow(t, "result", "Different", nil), ""))
			})
			rows, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{"fields": []any{"id", nil, true, false, json.Number("9007199254740993"), ""}, "is_shared": false, "name": "", "limit": json.Number("2e1"), "stateful": true, "description": nil, "marker": []any{}}))
			securityGroupListWant(t, rows, err, "result")
		})
	}
}

func TestSecurityGroupListFiltersRawDescriptorsAndNativeProjection(t *testing.T) {
	fields, filters := securityGroupListTarget()
	if len(fields) != 3 || len(filters) != 3 {
		t.Fatal(fields, filters)
	}
	for _, f := range securityGroupListFixtures() {
		t.Run(f.name+"/all-three", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("local fields leaked", r.URL)
				}
				testcloud.JSON(w, 200, securityGroupListPage(securityGroupListRow(t, "target", "Different", fields), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for k, v := range filters {
				rows, e := a.Resources.All(context.Background(), resource.WithFilter(k, v))
				securityGroupListWant(t, rows, e, "target")
			}
			rows, e := a.Resources.All(context.Background(), resource.WithFilters(filters))
			securityGroupListWant(t, rows, e, "target")
			v := rows[0]
			if calls.Load() != 4 || len(v.Rules) != 2 || v.Rules[1].ID != "" || v.Rules[0].PortRangeMin != 80 || v.Rules[0].RevisionNumber != 7 || v.Rules[0].ProjectID != "rule-project" || v.Rules[0].CreatedAt.IsZero() || v.CreatedAt.IsZero() || v.RevisionNumber != 9 || v.Stateful || v.TenantID != "native-tenant" || v.ProjectID != "native-project" {
				t.Fatal("native model changed", v, calls.Load())
			}
			rounded := strings.Replace(string(fields["security_group_rules"]), "9007199254740993", "9007199254740992", 1)
			reordered := `[null,` + strings.TrimSuffix(strings.TrimPrefix(string(fields["security_group_rules"]), "["), ",null]") + `]`
			for _, option := range []resource.ListOption{
				resource.WithFilter("security_group_rules", json.RawMessage(`[{"id":"rule"},null]`)),
				resource.WithFilter("security_group_rules", json.RawMessage(rounded)),
				resource.WithFilter("security_group_rules", json.RawMessage(reordered)),
				resource.WithFilter("security_group_rules", json.RawMessage(`{"id":"rule"}`)),
				resource.WithFilter("created_at", "2025-01-02T03:04:05Z"),
			} {
				rows, e = a.Resources.All(context.Background(), option)
				securityGroupListWant(t, rows, e)
			}
		})
		t.Run(f.name+"/missing-null-empty-and-old-time", func(t *testing.T) {
			cloud := testcloud.New(t)
			rows := securityGroupListRow(t, "missing", "Different", nil) + "," + securityGroupListRow(t, "null", "Different", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`null`), "created_at": json.RawMessage(`null`), "updated_at": json.RawMessage(`null`)}) + "," + securityGroupListRow(t, "empty", "Different", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[]`)}) + "," + securityGroupListRow(t, "value", "Different", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[null]`), "created_at": json.RawMessage(`"2025-01-02T03:04:05"`), "updated_at": json.RawMessage(`"2025-01-03T03:04:05"`)})
			cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, securityGroupListPage(rows, "")) })
			a := f.open(t, networkExtensionClient(cloud))
			for _, tc := range []struct {
				k   string
				v   any
				ids []string
			}{{"security_group_rules", nil, []string{"missing", "null"}}, {"security_group_rules", []any{}, []string{"empty"}}, {"security_group_rules", []any{nil}, []string{"value"}}, {"security_group_rules", []any{map[string]any{}}, nil}, {"created_at", nil, []string{"missing", "null", "empty"}}, {"created_at", "2025-01-02T03:04:05", []string{"value"}}, {"updated_at", "2025-01-03T03:04:05Z", nil}} {
				values, e := a.Resources.All(context.Background(), resource.WithFilter(tc.k, tc.v))
				securityGroupListWant(t, values, e, tc.ids...)
			}
		})
	}
}

func TestSecurityGroupListFiltersSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	for _, f := range securityGroupListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"vendor": {"kept"}, "shared": {"false"}, "fields": {"id", "security_group_rules"}}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, securityGroupListPage(securityGroupListRow(t, "target", "Different", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[{"id":"original","vendor":{"n":9007199254740993}},null]`)}), ""))
			})
			values := []any{map[string]any{"id": "original", "vendor": map[string]any{"n": json.Number("9007199254740993")}}, nil}
			qfields := []any{"id", "security_group_rules"}
			bulk := map[string]any{"security_group_rules": values, "is_shared": false, "fields": qfields}
			option := resource.WithFilters(bulk)
			values[0].(map[string]any)["id"] = "changed"
			qfields[0] = "changed"
			bulk["is_shared"] = true
			delete(bulk, "security_group_rules")
			a := f.open(t, networkExtensionClient(cloud))
			opts := []resource.ListOption{option, resource.WithQuery("vendor", "kept")}
			rows, e := a.Resources.All(context.Background(), opts...)
			securityGroupListWant(t, rows, e, "target")
			rows[0].Rules[0].ID = "caller-mutated"
			var wg sync.WaitGroup
			for range 6 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					v, e := a.Resources.All(context.Background(), opts...)
					if e != nil || len(v) != 1 || v[0].Rules[0].ID != "original" {
						t.Error(v, e)
					}
				}()
			}
			wg.Wait()
			for _, prefix := range [][]resource.ListOption{{resource.WithFilter("security_group_rules", math.NaN()), resource.WithFilter("security_group_rules", json.RawMessage(`[{"id":"original","vendor":{"n":9007199254740993}},null]`))}, {resource.WithFilter("headers", nil), resource.WithFilters(nil)}, {resource.WithFilter("created_at", make(chan int)), resource.WithFilters(map[string]any{})}} {
				options := append(prefix, resource.WithFilter("is_shared", false), resource.WithFilter("fields", []string{"id", "security_group_rules"}), resource.WithQuery("vendor", "kept"), resource.WithBodyFilter("updated_at", nil))
				v, e := a.Resources.All(context.Background(), options...)
				securityGroupListWant(t, v, e, "target")
			}
			if calls.Load() != 10 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestSecurityGroupListFiltersLazyPreflightAndNamespaceCollisions(t *testing.T) {
	for _, f := range securityGroupListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, securityGroupListPage(securityGroupListRow(t, "result", "Different", nil), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for _, pair := range [][]resource.ListOption{{resource.WithFilter("is_shared", false), resource.WithQuery("shared", "false")}, {resource.WithFilter("revision_number", 9), resource.WithQuery("revision_number", "9")}, {resource.WithFilter("fields", nil), resource.WithQuery("fields", "")}, {resource.WithFilter("name", nil), resource.WithName("Target")}, {resource.WithFilter("limit", 1), resource.WithPageSize(1)}, {resource.WithFilter("created_at", nil), resource.WithBodyFilter("created_at", nil)}} {
				for _, reverse := range []bool{false, true} {
					opts := append([]resource.ListOption(nil), pair...)
					if reverse {
						opts[0], opts[1] = opts[1], opts[0]
					}
					v, e := a.Resources.All(context.Background(), opts...)
					if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
						t.Fatal(v, e, calls.Load())
					}
				}
			}
			invalid := []resource.ListOption{resource.WithFilter("fields", map[string]any{"id": true}), resource.WithFilter("fields", [][]string{{"id"}}), resource.WithFilter("security_group_rules", math.NaN()), resource.WithBodyFilter("name", "Different"), resource.WithBodyFilter("status", "ACTIVE"), resource.WithBodyFilter("id", "result"), resource.WithBodyFilter("is_shared", false), resource.WithBodyFilter("revision_number", 9), resource.WithBodyFilter("tenant_id", "tenant"), resource.WithBodyFilter("stateful", false), resource.WithBodyFilter("security_group_rules", json.RawMessage(`{`)), resource.WithMaxItems(-1), resource.WithStatus("ACTIVE")}
			for _, k := range []string{"max_items", "paginated", "base_path", "allow_unknown_params", "session", "headers", "microversion", "resource_type", "jmespath_filters"} {
				invalid = append(invalid, resource.WithFilter(k, nil))
			}
			for _, option := range invalid {
				seq := a.Resources.List(context.Background(), option)
				if calls.Load() != 0 {
					t.Fatal("eager HTTP")
				}
				seen := 0
				for v, e := range seq {
					seen++
					if v != nil || (!errors.Is(e, resource.ErrInvalidOption) && !errors.Is(e, resource.ErrUnsupported)) {
						t.Fatal(v, e)
					}
				}
				if seen != 1 || calls.Load() != 0 {
					t.Fatal(seen, calls.Load())
				}
			}
			v, e := a.Resources.All(context.Background(), resource.WithFilters(map[string]any{"status": json.RawMessage(`{`), "rules": make(chan int), "revision": math.NaN()}))
			securityGroupListWant(t, v, e, "result")
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			v, e = a.Resources.All(ctx, resource.WithFilters(nil))
			if v != nil || !errors.Is(e, context.Canceled) || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			_, e = ports.New(networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(nil))
			if !errors.Is(e, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatal(e, calls.Load())
			}
		})
	}
}
func TestSecurityGroupListFiltersControlsAndNativeWholePageFailures(t *testing.T) {
	modes := []string{"cap-body", "cap-name", "single-page", "break", "null-row", "empty-object", "late-null", "cap-null", "query-null", "bad-id", "bad-stateful", "bad-revision", "bad-tenant", "bad-project", "bad-tags", "bad-rules", "bad-rule-id", "bad-rule-port", "bad-rule-revision", "bad-rule-date", "mixed-rule-date", "bad-date", "mixed-date"}
	for _, f := range securityGroupListFixtures() {
		for _, mode := range modes {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud, foreign := testcloud.New(t), testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); w.WriteHeader(500) })
				rows := securityGroupListRow(t, "first", "Different", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[]`)}) + "," + securityGroupListRow(t, "target", "Target", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[null]`)})
				opts := []resource.ListOption{resource.WithFilter("security_group_rules", []any{}), resource.WithPaginated(false)}
				nativeFailure := false
				switch mode {
				case "cap-body":
					opts = []resource.ListOption{resource.WithFilter("security_group_rules", []any{nil}), resource.WithMaxItems(1)}
				case "cap-name":
					opts = append(opts, resource.WithName("Target"), resource.WithMaxItems(1))
				case "single-page":
				case "break":
					opts = []resource.ListOption{resource.WithFilter("security_group_rules", []any{})}
				case "null-row", "query-null":
					rows = `null`
					opts = []resource.ListOption{resource.WithFilter("created_at", nil), resource.WithMaxItems(1)}
					if mode == "query-null" {
						opts = []resource.ListOption{resource.WithFilter("name", "server-only"), resource.WithMaxItems(1)}
					}
				case "empty-object":
					rows = `{}`
					opts = []resource.ListOption{resource.WithFilter("created_at", nil), resource.WithMaxItems(1)}
				case "late-null", "cap-null":
					rows = securityGroupListRow(t, "first", "Different", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[]`)}) + ",null"
					if mode == "cap-null" {
						opts = append(opts, resource.WithMaxItems(1))
					}
				default:
					key, raw := "security_group_rules", `{}`
					switch mode {
					case "bad-id":
						key, raw = "id", `false`
					case "bad-stateful":
						key, raw = "stateful", `"false"`
					case "bad-revision":
						key, raw = "revision_number", `"9"`
					case "bad-tenant":
						key, raw = "tenant_id", `1`
					case "bad-project":
						key, raw = "project_id", `false`
					case "bad-tags":
						key, raw = "tags", `[true]`
					case "bad-rule-id":
						raw = `[{"id":false}]`
					case "bad-rule-port":
						raw = `[{"port_range_min":"80"}]`
					case "bad-rule-revision":
						raw = `[{"revision_number":false}]`
					case "bad-rule-date":
						raw = `[{"created_at":"not-time"}]`
					case "mixed-rule-date":
						raw = `[{"created_at":"2025-01-02T03:04:05","updated_at":"2025-01-03T03:04:05Z"}]`
					case "bad-date":
						key, raw = "created_at", `"not-time"`
					case "mixed-date":
						key, raw = "created_at", `"2025-01-02T03:04:05"`
					}
					fields := map[string]json.RawMessage{key: json.RawMessage(raw)}
					if mode == "mixed-date" {
						fields["updated_at"] = json.RawMessage(`"2025-01-03T03:04:05Z"`)
					}
					rows += "," + securityGroupListRow(t, "invalid", "Other", fields)
					opts = append(opts, resource.WithMaxItems(1))
					nativeFailure = true
				}
				cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Has("limit") {
						t.Error("cap became wire limit", r.URL)
					}
					testcloud.JSON(w, 200, securityGroupListPage(rows, foreign.Server.URL+"/foreign"))
				})
				a := f.open(t, networkExtensionClient(cloud))
				if mode == "break" {
					seen := 0
					for v, e := range a.Resources.List(context.Background(), opts...) {
						if e != nil || v == nil || v.ID != "first" {
							t.Fatal(v, e)
						}
						seen++
						break
					}
					if seen != 1 || calls.Load() != 1 || followed.Load() != 0 {
						t.Fatal(seen, calls.Load(), followed.Load())
					}
					return
				}
				v, e := a.Resources.All(context.Background(), opts...)
				switch {
				case strings.HasPrefix(mode, "cap-"):
					if mode == "cap-null" {
						securityGroupListWant(t, v, e, "first")
					} else {
						securityGroupListWant(t, v, e)
					}
				case mode == "single-page":
					securityGroupListWant(t, v, e, "first")
				case mode == "empty-object", mode == "query-null":
					securityGroupListWant(t, v, e, "")
				default:
					if v != nil || e == nil {
						t.Fatal("All hid error or returned partial rows", v, e)
					}
					if nativeFailure {
						var cause *json.UnmarshalTypeError
						if !errors.As(e, &cause) && !strings.Contains(mode, "date") {
							t.Fatal("lost native typed cause", e)
						}
					} else if !errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal(e)
					}
				}
				if calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal(calls.Load(), followed.Load())
				}
			})
		}
	}
}

func TestSecurityGroupListFiltersPagingLiveSourceAndNativeStatus(t *testing.T) {
	for _, f := range securityGroupListFixtures() {
		for _, mode := range []string{"complete", "first-filtered", "duplicates", "late-http", "late-decode", "cycle", "foreign", "cancel", "wrong-links", "204", "json-204", "201"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud, foreign := testcloud.New(t), testcloud.New(t)
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
				cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("X-Configured") != "kept" || r.Header.Get("OpenStack-API-Version") != "network 2.0" || !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "security_group_rules"}) || r.URL.Query().Get("shared") != "false" || r.URL.Query().Get("status") != "vendor-status" {
						t.Error("source/query changed", r.Header, r.URL)
					}
					if r.URL.Query().Get("marker") != "" {
						if r.Header.Get("X-Auth-Token") != "later-token" {
							t.Error(r.Header)
						}
						if mode == "late-http" {
							w.WriteHeader(404)
							return
						}
						if mode == "late-decode" {
							testcloud.JSON(w, 200, securityGroupListPage(securityGroupListRow(t, "invalid", "Different", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[{"port_range_min":"80"}]`)}), ""))
							return
						}
						id := "second"
						if mode == "duplicates" {
							id = "first"
						}
						testcloud.JSON(w, 200, securityGroupListPage(securityGroupListRow(t, id, "Different", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[]`)}), ""))
						return
					}
					if r.Header.Get("X-Auth-Token") != "first-token" {
						t.Error(r.Header)
					}
					if mode == "204" || mode == "json-204" {
						if mode == "json-204" {
							w.Header().Set("Content-Type", "application/json")
						}
						w.WriteHeader(204)
						return
					}
					if mode == "201" {
						w.WriteHeader(201)
						return
					}
					next := cloud.Server.URL + securityGroupListPath + "?marker=next&shared=false&fields=id&fields=security_group_rules&status=vendor-status"
					if mode == "cycle" {
						next = cloud.Server.URL + r.URL.String()
					}
					if mode == "foreign" {
						next = foreign.Server.URL + "/foreign"
					}
					raw := json.RawMessage(`[]`)
					if mode == "first-filtered" {
						raw = json.RawMessage(`[null]`)
					}
					row := securityGroupListRow(t, "first", "Different", map[string]json.RawMessage{"security_group_rules": raw})
					if mode == "wrong-links" {
						testcloud.JSON(w, 300, `{"security_groups":[`+row+`],"links":{"next":`+fmt.Sprintf("%q", next)+`},"next":`+fmt.Sprintf("%q", next)+`}`)
						return
					}
					testcloud.JSON(w, 300, securityGroupListPage(row, next))
				})
				a := f.open(t, c)
				v, e := a.Resources.All(ctx, resource.WithFilter("security_group_rules", []any{}), resource.WithFilter("is_shared", false), resource.WithFilter("fields", []string{"id", "security_group_rules"}), resource.WithQuery("status", "vendor-status"))
				switch mode {
				case "complete":
					securityGroupListWant(t, v, e, "first", "second")
				case "first-filtered":
					securityGroupListWant(t, v, e, "second")
				case "duplicates":
					securityGroupListWant(t, v, e, "first", "first")
				case "wrong-links":
					securityGroupListWant(t, v, e, "first")
				case "204":
					securityGroupListWant(t, v, e)
				default:
					if v != nil || e == nil {
						t.Fatal("All hid terminal page error", v, e)
					}
					if mode == "cycle" && !errors.Is(e, resource.ErrPaginationCycle) || mode == "cancel" && !errors.Is(e, context.Canceled) || mode == "json-204" && !errors.Is(e, io.EOF) || mode == "201" && !gophercloud.ResponseCodeIs(e, 201) {
						t.Fatal(mode, e)
					}
				}
				wantCalls, wantMiddleware, wantFollow := int32(1), int32(1), int32(0)
				if mode == "complete" || mode == "first-filtered" || mode == "duplicates" || mode == "late-http" || mode == "late-decode" {
					wantCalls, wantMiddleware = 2, 2
				}
				if mode == "foreign" {
					wantMiddleware, wantFollow = 2, 1
				}
				if calls.Load() != wantCalls || middleware.Load() != wantMiddleware || followed.Load() != wantFollow || a.RawClient().ProviderClient != c.ProviderClient {
					t.Fatal(calls.Load(), middleware.Load(), followed.Load())
				}
			})
		}
	}
}

func TestSecurityGroupListFiltersNativeAndIdentitySurfaceIsolation(t *testing.T) {
	for _, f := range securityGroupListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets, creates atomic.Int32
			rawQuery := url.Values{"name": {"server-pattern"}, "status": {"vendor-status"}, "shared": {"false"}, "revision_number": {"raw-revision"}, "tenant_id": {"raw-tenant"}, "project_id": {"raw-project"}, "security_group_rules": {"wire-rules"}, "created_at": {"wire-date"}, "vendor": {"kept"}}
			expected := rawQuery
			cloud.Mux.HandleFunc("GET "+securityGroupListPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), expected) {
					t.Error("native/query lane changed", r.URL.Query(), expected)
				}
				testcloud.JSON(w, 200, securityGroupListPage(securityGroupListRow(t, "target", "Target", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[]`)})+","+securityGroupListRow(t, "other", "Other", map[string]json.RawMessage{"security_group_rules": json.RawMessage(`[null]`)}), ""))
			})
			cloud.Mux.HandleFunc("GET "+securityGroupListPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				want := url.Values{}
				if gets.Load() == 2 {
					want = url.Values{"security_group_rules": {"wire-rules"}, "status": {"vendor-status"}}
				}
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, `{"security_group":`+securityGroupListRow(t, "lookup", "Different", nil)+`}`)
			})
			cloud.Mux.HandleFunc("POST "+securityGroupListPath, func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				b, e := io.ReadAll(r.Body)
				if e != nil || string(b) != `{"security_group":{"name":"created","stateful":false}}` || r.URL.RawQuery != "" {
					t.Error(string(b), e, r.URL)
				}
				testcloud.JSON(w, 201, `{"security_group":`+securityGroupListRow(t, "created", "created", nil)+`}`)
			})
			a := f.open(t, networkExtensionClient(cloud))
			var raw []resource.ListOption
			for k, v := range rawQuery {
				raw = append(raw, resource.WithQuery(k, v[0]))
			}
			rows, e := a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilter("security_group_rules", []any{}), resource.WithFilter("status", json.RawMessage(`{`)))...)
			securityGroupListWant(t, rows, e, "target")
			rows, e = a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilters(nil))...)
			securityGroupListWant(t, rows, e, "target", "other")
			stateful, revision := false, 9
			expected = url.Values{"name": {"typed-native"}, "stateful": {"false"}, "revision_number": {"9"}}
			var ids []string
			for v, e := range a.List(context.Background(), securitygroups.WithListOptions(securitygroups.ListOpts{Name: "typed-native", Stateful: &stateful, RevisionNumber: &revision})) {
				if e != nil {
					t.Fatal(e)
				}
				ids = append(ids, v.ID)
			}
			if !reflect.DeepEqual(ids, []string{"target", "other"}) {
				t.Fatal(ids)
			}
			found, e := a.Get(context.Background(), "lookup")
			if e != nil || found == nil || found.ID != "lookup" {
				t.Fatal(found, e)
			}
			found, e = a.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindQuery("security_group_rules", "wire-rules"), resource.WithIdentityFindQuery("status", "vendor-status"))
			if e != nil || found == nil || found.ID != "lookup" || gets.Load() != 2 || lists.Load() != 3 {
				t.Fatal(found, e, gets.Load(), lists.Load())
			}
			created, e := a.Create(context.Background(), securitygroups.CreateOpts{Name: "created", Stateful: &stateful})
			if e != nil || created == nil || created.ID != "created" || creates.Load() != 1 {
				t.Fatal(created, e, creates.Load())
			}
			expected = rawQuery
			expected["name"] = []string{"Target"}
			var local []resource.ListOption
			for k, v := range expected {
				if k != "name" {
					local = append(local, resource.WithQuery(k, v[0]))
				}
			}
			local = append(local, resource.WithName("Target"), resource.WithBodyFilter("security_group_rules", []any{}))
			rows, e = a.Resources.All(context.Background(), local...)
			securityGroupListWant(t, rows, e, "target")
		})
	}
}
