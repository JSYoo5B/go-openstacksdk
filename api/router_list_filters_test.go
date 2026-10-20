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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Independent Router query declarations from pinned router.py and TagMixin.
// In particular, id/name/status and ha remain server-only semantic conditions.
func routerListQueries() map[string]string {
	return map[string]string{"description": "description", "fields": "fields", "flavor_id": "flavor_id", "id": "id", "name": "name", "status": "status", "project_id": "project_id", "sort_key": "sort_key", "sort_dir": "sort_dir", "limit": "limit", "marker": "marker", "is_admin_state_up": "admin_state_up", "is_distributed": "distributed", "is_ha": "ha", "tags": "tags", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any"}
}

type routerListAccess struct {
	api        *routers.API
	collection *resource.Collection[routers.Router]
	raw        *gophercloud.ServiceClient
}
type routerListFixture struct {
	name string
	open func(*testing.T, *gophercloud.ServiceClient) routerListAccess
}

func routerListFixtures() []routerListFixture {
	var out []routerListFixture
	for _, connected := range []bool{false, true} {
		name := "leaf"
		if connected {
			name = "connection"
		}
		out = append(out, routerListFixture{name, func(t *testing.T, c *gophercloud.ServiceClient) routerListAccess {
			a := routers.New(c)
			if connected {
				s := networkSubnetBodyConnection(t, c)
				a = s.API.Routers
				if a.RawClient() != s.RawClient() || a.RawClient().ProviderClient != c.ProviderClient {
					t.Fatal("Router API did not share cached client")
				}
			}
			return routerListAccess{a, a.Resources, a.RawClient()}
		}})
	}
	return out
}

const routerListPath = networkExtensionPrefix + "routers"

func routerListRow(t *testing.T, id, name string, fields map[string]json.RawMessage) string {
	t.Helper()
	body := map[string]json.RawMessage{"id": json.RawMessage(fmt.Sprintf("%q", id)), "name": json.RawMessage(fmt.Sprintf("%q", name)), "status": json.RawMessage(`"ACTIVE"`), "project_id": json.RawMessage(`"native-project"`), "admin_state_up": json.RawMessage(`false`), "distributed": json.RawMessage(`false`)}
	for k, v := range fields {
		if v == nil {
			delete(body, k)
		} else {
			body[k] = v
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
func routerListPage(rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,"routers_links":[{"rel":"next","href":%q}]`, next)
	}
	return `{"routers":[` + rows + `]` + links + `}`
}
func routerListWant(t *testing.T, values []*routers.Router, err error, ids ...string) {
	t.Helper()
	actual := make([]string, 0, len(values))
	for _, v := range values {
		actual = append(actual, v.ID)
	}
	if err != nil || !reflect.DeepEqual(actual, append([]string{}, ids...)) {
		t.Fatal("actual native collection", actual, err, "want", ids)
	}
}
func routerListTarget() (map[string]json.RawMessage, map[string]any) {
	fields := map[string]json.RawMessage{
		"availability_zone_hints": json.RawMessage(`["az-one",null]`),
		"availability_zones":      json.RawMessage(`["az-two",{"n":9007199254740993}]`),
		"created_at":              json.RawMessage(`"2025-01-02T03:04:05+00:00"`),
		"updated_at":              json.RawMessage(`"2025-01-03T03:04:05+00:00"`),
		"enable_ndp_proxy":        json.RawMessage(`"false"`),
		"evpn_vni":                json.RawMessage(`" +001500 "`),
		"revision":                json.RawMessage(`9007199254740993`),
		"revision_number":         json.RawMessage(`9`),
		"tenant_id":               json.RawMessage(`"local-tenant"`),
		"external_gateway_info":   json.RawMessage(`{"network_id":"external","enable_snat":null,"external_fixed_ips":[{"ip_address":"192.0.2.1","subnet_id":"external-subnet","n":9007199254740993},null],"vendor":{"n":9007199254740993,"nullable":null}}`),
		"routes":                  json.RawMessage(`[{"nexthop":"192.0.2.1","destination":"198.51.100.0/24","n":9007199254740993},null]`),
	}
	filters := map[string]any{
		"availability_zone_hints": json.RawMessage(`["az-one",null]`),
		"availability_zones":      json.RawMessage(`["az-two",{"n":9007199254740993}]`),
		"created_at":              "2025-01-02T03:04:05+00:00",
		"updated_at":              "2025-01-03T03:04:05+00:00",
		"enable_ndp_proxy":        true,
		"evpn_vni":                1500,
		"revision_number":         json.Number("9007199254740993"),
		"tenant_id":               "local-tenant",
		"external_gateway_info":   json.RawMessage(`{"vendor":{"n":9007199254740993}}`),
		"routes":                  json.RawMessage(`[{"nexthop":"192.0.2.1","destination":"198.51.100.0/24","n":9007199254740993},null]`),
	}
	return fields, filters
}

func TestRouterListFiltersEntireQueryAndAliasDescriptor(t *testing.T) {
	canonical := routerListQueries()
	accepted := map[string]string{}
	for k, w := range canonical {
		accepted[k], accepted[w] = w, w
	}
	if len(canonical) != 18 || len(accepted) != 24 {
		t.Fatal(canonical, accepted)
	}
	for _, f := range routerListFixtures() {
		for k, w := range accepted {
			t.Run(f.name+"/"+k, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+routerListPath, func(wr http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{w: {"server-value"}}) {
						t.Error(k, r.URL)
					}
					testcloud.JSON(wr, 200, routerListPage(routerListRow(t, "result", "Different", nil), ""))
				})
				a := f.open(t, networkExtensionClient(cloud))
				rows, err := a.collection.All(context.Background(), resource.WithFilter(k, "server-value"))
				routerListWant(t, rows, err, "result")
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
					cloud.Mux.HandleFunc("GET "+routerListPath, func(wr http.ResponseWriter, r *http.Request) {
						want := url.Values{}
						if tc.want != nil {
							want[w] = tc.want
						}
						if !reflect.DeepEqual(r.URL.Query(), want) {
							t.Error(r.URL.Query(), want)
						}
						testcloud.JSON(wr, 200, routerListPage(routerListRow(t, "result", "Different", nil), ""))
					})
					rows, err := f.open(t, networkExtensionClient(cloud)).collection.All(context.Background(), resource.WithFilters(map[string]any{k: tc.value, w: json.RawMessage(`{`)}))
					routerListWant(t, rows, err, "result")
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
					cloud.Mux.HandleFunc("GET "+routerListPath, func(wr http.ResponseWriter, r *http.Request) {
						if r.URL.Query().Get(w) != want {
							t.Error(r.URL)
						}
						testcloud.JSON(wr, 200, routerListPage(routerListRow(t, "result", "Different", nil), ""))
					})
					rows, err := f.open(t, networkExtensionClient(cloud)).collection.All(context.Background(), opts...)
					routerListWant(t, rows, err, "result")
				})
			}
		}
		t.Run(f.name+"/scalar-encoding", func(t *testing.T) {
			cloud := testcloud.New(t)
			want := url.Values{"fields": {"id", "true", "false", "9007199254740993", ""}, "ha": {"false"}, "name": {""}, "limit": {"2e1"}, "distributed": {"true"}}
			cloud.Mux.HandleFunc("GET "+routerListPath, func(w http.ResponseWriter, r *http.Request) {
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, routerListPage(routerListRow(t, "result", "Different", nil), ""))
			})
			rows, err := f.open(t, networkExtensionClient(cloud)).collection.All(context.Background(), resource.WithFilters(map[string]any{"fields": []any{"id", nil, true, false, json.Number("9007199254740993"), ""}, "is_ha": false, "name": "", "limit": json.Number("2e1"), "is_distributed": true, "description": nil, "marker": []any{}}))
			routerListWant(t, rows, err, "result")
		})
	}
}

func TestRouterListFiltersRawDescriptorsAndNativeProjection(t *testing.T) {
	properties := map[string]string{"availability_zone_hints": "availability_zone_hints", "availability_zones": "availability_zones", "created_at": "created_at", "enable_ndp_proxy": "enable_ndp_proxy", "evpn_vni": "evpn_vni", "external_gateway_info": "external_gateway_info", "revision_number": "revision", "routes": "routes", "tenant_id": "tenant_id", "updated_at": "updated_at"}
	fields, filters := routerListTarget()
	if len(properties) != 10 || len(filters) != 10 {
		t.Fatal(properties, filters)
	}
	for _, f := range routerListFixtures() {
		t.Run(f.name+"/all-ten", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+routerListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("local fields leaked", r.URL)
				}
				testcloud.JSON(w, 200, routerListPage(routerListRow(t, "target", "Different", fields), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for k, v := range filters {
				rows, e := a.collection.All(context.Background(), resource.WithFilter(k, v))
				routerListWant(t, rows, e, "target")
			}
			rows, e := a.collection.All(context.Background(), resource.WithFilters(filters))
			routerListWant(t, rows, e, "target")
			v := rows[0]
			if calls.Load() != 11 || v.RevisionNumber != 9 || v.TenantID != "local-tenant" || v.ProjectID != "native-project" || !reflect.DeepEqual(v.AvailabilityZoneHints, []string{"az-one", ""}) || len(v.Routes) != 2 || v.Routes[1] != (routers.Route{}) || v.GatewayInfo.EnableSNAT != nil || len(v.GatewayInfo.ExternalFixedIPs) != 2 || v.GatewayInfo.ExternalFixedIPs[1] != (routers.ExternalFixedIP{}) || v.CreatedAt.IsZero() {
				t.Fatal("raw matching changed native model", v, calls.Load())
			}
			for _, option := range []resource.ListOption{
				resource.WithFilter("routes", json.RawMessage(`[{"n":9007199254740993},null]`)),
				resource.WithFilter("routes", json.RawMessage(`[{"nexthop":"192.0.2.1","destination":"198.51.100.0/24","n":9007199254740992},null]`)),
				resource.WithFilter("external_gateway_info", json.RawMessage(`{"vendor":{"n":9007199254740992}}`)),
				resource.WithFilter("external_gateway_info", json.RawMessage(`{"enable_snat":false}`)),
				resource.WithFilter("created_at", "2025-01-02T03:04:05Z"),
				resource.WithFilter("evpn_vni", "1500"),
				resource.WithFilter("revision_number", 9),
				resource.WithFilter("tenant_id", "native-project"),
				resource.WithFilter("availability_zone_hints", []string{"az-one", ""}),
			} {
				rows, e = a.collection.All(context.Background(), option)
				routerListWant(t, rows, e)
			}
			rows, e = a.collection.All(context.Background(), resource.WithBodyFilter("revision", json.Number("9007199254740993")))
			routerListWant(t, rows, e, "target")
			rows, e = a.collection.All(context.Background(), resource.WithBodyFilter("revision_number", json.Number("9007199254740993")))
			routerListWant(t, rows, e, "target")
		})
		t.Run(f.name+"/boolean-truthiness", func(t *testing.T) {
			cloud := testcloud.New(t)
			var rows []string
			for _, tc := range []struct{ id, raw string }{{"missing", "missing"}, {"null", "null"}, {"false", "false"}, {"zero", "-0.000e999999999999999999999"}, {"empty-string", `""`}, {"empty-array", "[]"}, {"empty-object", "{}"}, {"true", "true"}, {"tiny", "1e-999999999999999999999"}, {"negative", "-9007199254740993"}, {"false-string", `"false"`}, {"array", "[null]"}, {"object", `{"n":null}`}} {
				fields := map[string]json.RawMessage{}
				if tc.raw != "missing" {
					fields["enable_ndp_proxy"] = json.RawMessage(tc.raw)
				}
				rows = append(rows, routerListRow(t, tc.id, "Different", fields))
			}
			cloud.Mux.HandleFunc("GET "+routerListPath, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, routerListPage(strings.Join(rows, ","), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for _, tc := range []struct {
				v   any
				ids []string
			}{{nil, []string{"missing", "null"}}, {false, []string{"false", "zero", "empty-string", "empty-array", "empty-object"}}, {true, []string{"true", "tiny", "negative", "false-string", "array", "object"}}, {1, nil}, {"false", nil}} {
				values, e := a.collection.All(context.Background(), resource.WithFilter("enable_ndp_proxy", tc.v))
				routerListWant(t, values, e, tc.ids...)
			}
		})
		t.Run(f.name+"/null-object-time-and-exact-integers", func(t *testing.T) {
			cloud := testcloud.New(t)
			rows := routerListRow(t, "missing", "Different", nil) + "," + routerListRow(t, "null", "Different", map[string]json.RawMessage{"revision": json.RawMessage(`null`), "evpn_vni": json.RawMessage(`null`), "external_gateway_info": json.RawMessage(`null`), "routes": json.RawMessage(`null`), "tenant_id": json.RawMessage(`null`)}) + "," + routerListRow(t, "empty", "Different", map[string]json.RawMessage{"revision": json.RawMessage(`0`), "evpn_vni": json.RawMessage(`0`), "external_gateway_info": json.RawMessage(`{}`), "routes": json.RawMessage(`[]`), "tenant_id": json.RawMessage(`""`)}) + "," + routerListRow(t, "value", "Different", map[string]json.RawMessage{"revision": json.RawMessage(`" -00042 "`), "evpn_vni": json.RawMessage(`1.5e3`), "external_gateway_info": json.RawMessage(`{"vendor":{"n":1}}`), "routes": json.RawMessage(`[null]`), "tenant_id": json.RawMessage(`"local"`), "created_at": json.RawMessage(`"2025-01-02T03:04:05"`), "updated_at": json.RawMessage(`"2025-01-03T03:04:05"`)})
			cloud.Mux.HandleFunc("GET "+routerListPath, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, routerListPage(rows, "")) })
			a := f.open(t, networkExtensionClient(cloud))
			for _, tc := range []struct {
				k   string
				v   any
				ids []string
			}{{"revision_number", nil, []string{"missing", "null"}}, {"revision_number", -42, []string{"value"}}, {"evpn_vni", 1500, []string{"value"}}, {"routes", nil, []string{"missing", "null"}}, {"routes", []any{}, []string{"empty"}}, {"routes", []any{nil}, []string{"value"}}, {"external_gateway_info", map[string]any{}, []string{"value"}}, {"tenant_id", nil, []string{"missing", "null"}}, {"tenant_id", "", []string{"empty"}}, {"created_at", "2025-01-02T03:04:05", []string{"value"}}, {"created_at", "2025-01-02T03:04:05Z", nil}} {
				values, e := a.collection.All(context.Background(), resource.WithFilter(tc.k, tc.v))
				routerListWant(t, values, e, tc.ids...)
			}
		})
	}
}

func TestRouterListFiltersSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	for _, f := range routerListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+routerListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"vendor": {"kept"}, "ha": {"false"}, "fields": {"id", "name"}}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, routerListPage(routerListRow(t, "target", "Different", map[string]json.RawMessage{"routes": json.RawMessage(`[{"nexthop":"original","destination":"198.51.100.0/24"},null]`), "revision": json.RawMessage(`42`)}), ""))
			})
			values := []any{map[string]any{"nexthop": "original", "destination": "198.51.100.0/24"}, nil}
			queryFields := []any{"id", "name"}
			bulk := map[string]any{"routes": values, "is_ha": false, "fields": queryFields}
			option := resource.WithFilters(bulk)
			values[0].(map[string]any)["nexthop"] = "changed"
			queryFields[0] = "changed"
			bulk["is_ha"] = true
			delete(bulk, "routes")
			a := f.open(t, networkExtensionClient(cloud))
			options := []resource.ListOption{option, resource.WithQuery("vendor", "kept")}
			rows, e := a.collection.All(context.Background(), options...)
			routerListWant(t, rows, e, "target")
			rows[0].Routes[0].NextHop = "caller-mutated"
			var wg sync.WaitGroup
			for range 6 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					v, e := a.collection.All(context.Background(), options...)
					if e != nil || len(v) != 1 || v[0].Routes[0].NextHop != "original" {
						t.Error(v, e)
					}
				}()
			}
			wg.Wait()
			for _, prefix := range [][]resource.ListOption{{resource.WithFilter("routes", math.NaN()), resource.WithFilter("routes", []any{map[string]any{"nexthop": "original", "destination": "198.51.100.0/24"}, nil})}, {resource.WithFilter("headers", nil), resource.WithFilters(nil)}, {resource.WithFilter("revision_number", make(chan int)), resource.WithFilters(map[string]any{})}} {
				opts := append(prefix, resource.WithFilter("is_ha", false), resource.WithFilter("fields", []string{"id", "name"}), resource.WithQuery("vendor", "kept"), resource.WithBodyFilter("revision", 42))
				v, e := a.collection.All(context.Background(), opts...)
				routerListWant(t, v, e, "target")
			}
			for _, tc := range []struct {
				opts []resource.ListOption
				ids  []string
			}{{[]resource.ListOption{resource.WithBodyFilter("revision", 0), resource.WithBodyFilter("revision_number", 42)}, []string{"target"}}, {[]resource.ListOption{resource.WithBodyFilter("revision_number", 42), resource.WithBodyFilter("revision", 0)}, nil}} {
				opts := append(tc.opts, resource.WithFilter("is_ha", false), resource.WithFilter("fields", []string{"id", "name"}), resource.WithQuery("vendor", "kept"))
				v, e := a.collection.All(context.Background(), opts...)
				routerListWant(t, v, e, tc.ids...)
			}
			if calls.Load() != 12 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestRouterListFiltersLazyPreflightAndNamespaceCollisions(t *testing.T) {
	for _, f := range routerListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+routerListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, routerListPage(routerListRow(t, "result", "Different", nil), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for _, pair := range [][]resource.ListOption{{resource.WithFilter("is_ha", false), resource.WithQuery("ha", "false")}, {resource.WithFilter("fields", nil), resource.WithQuery("fields", "")}, {resource.WithFilter("name", nil), resource.WithName("Target")}, {resource.WithFilter("status", "ACTIVE"), resource.WithStatus("ACTIVE")}, {resource.WithFilter("limit", 1), resource.WithPageSize(1)}, {resource.WithFilter("revision_number", nil), resource.WithBodyFilter("revision", nil)}, {resource.WithFilter("external_gateway_info", map[string]any{}), resource.WithBodyFilter("external_gateway_info", map[string]any{})}} {
				for _, reverse := range []bool{false, true} {
					opts := append([]resource.ListOption(nil), pair...)
					if reverse {
						opts[0], opts[1] = opts[1], opts[0]
					}
					v, e := a.collection.All(context.Background(), opts...)
					if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
						t.Fatal(v, e, calls.Load())
					}
				}
			}
			invalid := []resource.ListOption{resource.WithFilter("fields", map[string]any{"id": true}), resource.WithFilter("fields", [][]string{{"id"}}), resource.WithFilter("enable_ndp_proxy", math.NaN()), resource.WithBodyFilter("name", "Different"), resource.WithBodyFilter("status", "ACTIVE"), resource.WithBodyFilter("id", "result"), resource.WithBodyFilter("ha", false), resource.WithBodyFilter("project_id", "project"), resource.WithBodyFilters(map[string]any{"revision_number": nil, "revision": nil}), resource.WithBodyFilter("evpn_vni", json.RawMessage(`{`)), resource.WithMaxItems(-1)}
			for _, k := range []string{"max_items", "paginated", "base_path", "allow_unknown_params", "session", "headers", "microversion", "resource_type", "jmespath_filters"} {
				invalid = append(invalid, resource.WithFilter(k, nil))
			}
			for _, option := range invalid {
				seq := a.collection.List(context.Background(), option)
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
			v, e := a.collection.All(context.Background(), resource.WithFilters(map[string]any{"revision": make(chan int), "HA": json.RawMessage(`{`), "unknown": math.NaN()}))
			routerListWant(t, v, e, "result")
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			v, e = a.collection.All(ctx, resource.WithFilters(nil))
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
func TestRouterListFiltersControlsAndNativeWholePageFailures(t *testing.T) {
	modes := []string{"cap-body", "cap-name", "cap-status", "single-page", "break", "null-row", "empty-object", "late-null", "cap-null", "query-null", "raw-fraction", "raw-fraction-cap", "raw-fraction-unselected", "raw-bool", "raw-object", "bad-id", "bad-bool", "bad-routes", "bad-gateway", "bad-snat", "bad-fixed-ips", "bad-hints", "bad-tags", "bad-revision", "bad-tenant", "bad-date", "mixed-date"}
	for _, f := range routerListFixtures() {
		for _, mode := range modes {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud, foreign := testcloud.New(t), testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); w.WriteHeader(500) })
				rows := routerListRow(t, "first", "Different", map[string]json.RawMessage{"routes": json.RawMessage(`[]`), "status": json.RawMessage(`"DOWN"`)}) + "," + routerListRow(t, "target", "Target", map[string]json.RawMessage{"routes": json.RawMessage(`[null]`)})
				opts := []resource.ListOption{resource.WithFilter("routes", []any{}), resource.WithPaginated(false)}
				nativeFailure := false
				switch mode {
				case "cap-body":
					opts = []resource.ListOption{resource.WithFilter("routes", []any{nil}), resource.WithMaxItems(1)}
				case "cap-name":
					opts = append(opts, resource.WithName("Target"), resource.WithMaxItems(1))
				case "cap-status":
					opts = append(opts, resource.WithStatus("ACTIVE"), resource.WithMaxItems(1))
				case "single-page":
				case "break":
					opts = []resource.ListOption{resource.WithFilter("routes", []any{})}
				case "null-row", "query-null":
					rows = "null"
					opts = []resource.ListOption{resource.WithFilter("enable_ndp_proxy", nil), resource.WithMaxItems(1)}
					if mode == "query-null" {
						opts = []resource.ListOption{resource.WithFilter("name", "server-only"), resource.WithMaxItems(1)}
					}
				case "empty-object":
					rows = "{}"
					opts = []resource.ListOption{resource.WithFilter("enable_ndp_proxy", nil), resource.WithMaxItems(1)}
				case "late-null", "cap-null":
					rows = routerListRow(t, "first", "Different", map[string]json.RawMessage{"routes": json.RawMessage(`[]`)}) + ",null"
					if mode == "cap-null" {
						opts = append(opts, resource.WithMaxItems(1))
					}
				case "raw-fraction", "raw-fraction-cap", "raw-fraction-unselected", "raw-bool", "raw-object":
					raw := `42.5`
					if mode == "raw-bool" {
						raw = `true`
					}
					if mode == "raw-object" {
						raw = `{}`
					}
					rows = routerListRow(t, "first", "Target", map[string]json.RawMessage{"revision": json.RawMessage(`42`), "routes": json.RawMessage(`[]`)}) + "," + routerListRow(t, "fraction", "Other", map[string]json.RawMessage{"revision": json.RawMessage(raw), "routes": json.RawMessage(`[]`)})
					opts = []resource.ListOption{resource.WithFilter("revision_number", 42), resource.WithName("Target"), resource.WithPaginated(false)}
					if mode == "raw-fraction-cap" {
						opts = append(opts, resource.WithMaxItems(1))
					}
					if mode == "raw-fraction-unselected" {
						opts = []resource.ListOption{resource.WithFilter("routes", []any{}), resource.WithPaginated(false)}
					}
				default:
					key, raw := "routes", `[{"nexthop":1}]`
					switch mode {
					case "bad-id":
						key, raw = "id", `false`
					case "bad-bool":
						key, raw = "admin_state_up", `"false"`
					case "bad-gateway":
						key, raw = "external_gateway_info", `{"network_id":1}`
					case "bad-snat":
						key, raw = "external_gateway_info", `{"enable_snat":"false"}`
					case "bad-fixed-ips":
						key, raw = "external_gateway_info", `{"external_fixed_ips":[{"ip_address":true}]}`
					case "bad-hints":
						key, raw = "availability_zone_hints", `[{}]`
					case "bad-tags":
						key, raw = "tags", `[true]`
					case "bad-revision":
						key, raw = "revision_number", `"9"`
					case "bad-tenant":
						key, raw = "tenant_id", `1`
					case "bad-date":
						key, raw = "created_at", `"not-time"`
					case "mixed-date":
						key, raw = "created_at", `"2025-01-02T03:04:05"`
					}
					fields := map[string]json.RawMessage{key: json.RawMessage(raw)}
					if mode == "mixed-date" {
						fields["updated_at"] = json.RawMessage(`"2025-01-03T03:04:05Z"`)
					}
					rows += "," + routerListRow(t, "invalid", "Other", fields)
					opts = append(opts, resource.WithMaxItems(1))
					nativeFailure = true
				}
				cloud.Mux.HandleFunc("GET "+routerListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Has("limit") {
						t.Error("cap became wire limit", r.URL)
					}
					testcloud.JSON(w, 200, routerListPage(rows, foreign.Server.URL+"/foreign"))
				})
				a := f.open(t, networkExtensionClient(cloud))
				if mode == "break" {
					seen := 0
					for v, e := range a.collection.List(context.Background(), opts...) {
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
				v, e := a.collection.All(context.Background(), opts...)
				switch {
				case strings.HasPrefix(mode, "cap-"):
					if mode == "cap-null" {
						routerListWant(t, v, e, "first")
					} else {
						routerListWant(t, v, e)
					}
				case mode == "single-page", mode == "raw-fraction-cap":
					routerListWant(t, v, e, "first")
				case mode == "raw-fraction-unselected":
					routerListWant(t, v, e, "first", "fraction")
				case mode == "empty-object", mode == "query-null":
					routerListWant(t, v, e, "")
				default:
					if v != nil || e == nil {
						t.Fatal("All hid error or returned partial rows", v, e)
					}
					if nativeFailure {
						var syntax *json.UnmarshalTypeError
						if !errors.As(e, &syntax) && mode != "bad-date" && mode != "mixed-date" {
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

func TestRouterListFiltersPagingLiveSourceAndNativeStatus(t *testing.T) {
	for _, f := range routerListFixtures() {
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
				cloud.Mux.HandleFunc("GET "+routerListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("X-Configured") != "kept" || r.Header.Get("OpenStack-API-Version") != "network 2.0" || !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) || r.URL.Query().Get("ha") != "false" || r.URL.Query().Get("status") != "vendor-status" {
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
							testcloud.JSON(w, 200, routerListPage(routerListRow(t, "invalid", "Different", map[string]json.RawMessage{"external_gateway_info": json.RawMessage(`{"enable_snat":"false"}`)}), ""))
							return
						}
						id := "second"
						if mode == "duplicates" {
							id = "first"
						}
						testcloud.JSON(w, 200, routerListPage(routerListRow(t, id, "Different", map[string]json.RawMessage{"enable_ndp_proxy": json.RawMessage(`true`)}), ""))
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
					next := cloud.Server.URL + routerListPath + "?marker=next&ha=false&fields=id&fields=name&status=vendor-status"
					if mode == "cycle" {
						next = cloud.Server.URL + r.URL.String()
					}
					if mode == "foreign" {
						next = foreign.Server.URL + "/foreign"
					}
					raw := json.RawMessage(`"false"`)
					if mode == "first-filtered" {
						raw = json.RawMessage(`0`)
					}
					row := routerListRow(t, "first", "Different", map[string]json.RawMessage{"enable_ndp_proxy": raw})
					if mode == "wrong-links" {
						testcloud.JSON(w, 300, `{"routers":[`+row+`],"links":{"next":`+fmt.Sprintf("%q", next)+`},"next":`+fmt.Sprintf("%q", next)+`}`)
						return
					}
					testcloud.JSON(w, 300, routerListPage(row, next))
				})
				a := f.open(t, c)
				v, e := a.collection.All(ctx, resource.WithFilter("enable_ndp_proxy", true), resource.WithFilter("is_ha", false), resource.WithFilter("fields", []string{"id", "name"}), resource.WithQuery("status", "vendor-status"))
				switch mode {
				case "complete":
					routerListWant(t, v, e, "first", "second")
				case "first-filtered":
					routerListWant(t, v, e, "second")
				case "duplicates":
					routerListWant(t, v, e, "first", "first")
				case "wrong-links":
					routerListWant(t, v, e, "first")
				case "204":
					routerListWant(t, v, e)
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
				if calls.Load() != wantCalls || middleware.Load() != wantMiddleware || followed.Load() != wantFollow || a.raw.ProviderClient != c.ProviderClient {
					t.Fatal(calls.Load(), middleware.Load(), followed.Load())
				}
			})
		}
	}
}

func TestRouterListFiltersNativeAndIdentitySurfaceIsolation(t *testing.T) {
	for _, f := range routerListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets, actions atomic.Int32
			want := url.Values{"name": {"server-pattern"}, "status": {"vendor-status"}, "revision": {"raw-revision"}, "revision_number": {"raw-native-revision"}, "evpn_vni": {"raw-vni"}, "enable_ndp_proxy": {"raw-flag"}, "tenant_id": {"raw-tenant"}, "vendor": {"kept"}}
			cloud.Mux.HandleFunc("GET "+routerListPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("query classified unexpectedly", r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, routerListPage(routerListRow(t, "target", "Target", map[string]json.RawMessage{"routes": json.RawMessage(`[]`), "revision": json.RawMessage(`false`), "status": json.RawMessage(`"active"`)})+","+routerListRow(t, "other", "Other", map[string]json.RawMessage{"routes": json.RawMessage(`[null]`), "status": json.RawMessage(`"DOWN"`)}), ""))
			})
			cloud.Mux.HandleFunc("GET "+routerListPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				expected := url.Values{}
				if gets.Load() == 2 {
					expected = url.Values{"revision": {"raw-revision"}, "status": {"vendor-status"}}
				}
				if !reflect.DeepEqual(r.URL.Query(), expected) {
					t.Error(r.URL.Query(), expected)
				}
				testcloud.JSON(w, 200, `{"router":`+routerListRow(t, "lookup", "Different", map[string]json.RawMessage{"revision": json.RawMessage(`false`), "enable_ndp_proxy": json.RawMessage(`"false"`)})+`}`)
			})
			cloud.Mux.HandleFunc("PUT "+routerListPath+"/lookup/add_router_interface", func(w http.ResponseWriter, r *http.Request) {
				actions.Add(1)
				body, e := io.ReadAll(r.Body)
				if e != nil || string(body) != `{"subnet_id":"subnet"}` || r.URL.RawQuery != "" {
					t.Error(string(body), e, r.URL)
				}
				testcloud.JSON(w, 200, `{"id":"lookup","tenant_id":"tenant","port_id":"port","subnet_id":"subnet"}`)
			})
			a := f.open(t, networkExtensionClient(cloud))
			var raw []resource.ListOption
			var native []routers.ListOption
			for k, v := range want {
				raw = append(raw, resource.WithQuery(k, v[0]))
				native = append(native, routers.WithListQuery(k, v[0]))
			}
			rows, e := a.collection.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilter("routes", []any{}))...)
			routerListWant(t, rows, e, "target")
			rows, e = a.collection.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilters(nil))...)
			routerListWant(t, rows, e, "target", "other")
			var ids []string
			for v, e := range a.api.List(context.Background(), native...) {
				if e != nil {
					t.Fatal(e)
				}
				ids = append(ids, v.ID)
			}
			if !reflect.DeepEqual(ids, []string{"target", "other"}) {
				t.Fatal(ids)
			}
			found, e := a.api.Get(context.Background(), "lookup")
			if e != nil || found == nil || found.ID != "lookup" {
				t.Fatal(found, e)
			}
			found, e = a.api.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindQuery("revision", "raw-revision"), resource.WithIdentityFindQuery("status", "vendor-status"))
			if e != nil || found == nil || found.ID != "lookup" || gets.Load() != 2 || lists.Load() != 3 {
				t.Fatal(found, e, gets.Load(), lists.Load())
			}
			info, e := a.api.AddInterface(context.Background(), "lookup", routers.AddInterfaceOpts{SubnetID: "subnet"})
			if e != nil || info == nil || info.PortID != "port" || actions.Load() != 1 {
				t.Fatal(info, e, actions.Load())
			}
			want["name"], want["status"] = []string{"Target"}, []string{"ACTIVE"}
			var local []resource.ListOption
			for k, v := range want {
				if k != "name" && k != "status" {
					local = append(local, resource.WithQuery(k, v[0]))
				}
			}
			local = append(local, resource.WithName("Target"), resource.WithStatus("earlier-status"), resource.WithQuery("status", "ACTIVE"), resource.WithBodyFilter("routes", []any{}))
			rows, e = a.collection.All(context.Background(), local...)
			routerListWant(t, rows, e, "target")
		})
	}
}
