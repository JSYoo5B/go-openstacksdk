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
	"gophercloudsdk/network/v2/extensions/trunks"
	"gophercloudsdk/network/v2/networks"
	"gophercloudsdk/resource"
)

// Independent pinned Network query declarations, including inherited tag aliases.
// name/status/id and every flag in this map remain server-only conditions.
func networkListQueries() map[string]string {
	return map[string]string{"description": "description", "fields": "fields", "name": "name", "status": "status", "project_id": "project_id", "sort_key": "sort_key", "sort_dir": "sort_dir", "id": "id", "limit": "limit", "marker": "marker", "ipv4_address_scope_id": "ipv4_address_scope", "ipv6_address_scope_id": "ipv6_address_scope", "is_admin_state_up": "admin_state_up", "is_port_security_enabled": "port_security_enabled", "is_router_external": "router:external", "is_shared": "shared", "provider_network_type": "provider:network_type", "provider_physical_network": "provider:physical_network", "provider_segmentation_id": "provider:segmentation_id", "tags": "tags", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any"}
}

type networkListAccess struct {
	api        *networks.API
	collection *resource.Collection[networks.Network]
	raw        *gophercloud.ServiceClient
}
type networkListFixture struct {
	name string
	open func(*testing.T, *gophercloud.ServiceClient) networkListAccess
}

func networkListFixtures() []networkListFixture {
	out := []networkListFixture{}
	for _, mode := range []string{"leaf", "connection", "manual"} {
		out = append(out, networkListFixture{mode, func(t *testing.T, c *gophercloud.ServiceClient) networkListAccess {
			a := networks.New(c)
			collection, raw := a.Resources, c
			if mode != "leaf" {
				s := networkSubnetBodyConnection(t, c)
				a, raw = s.API.Networks, s.RawClient()
				collection = a.Resources
				if mode == "manual" {
					collection = s.Networks
				}
			}
			return networkListAccess{a, collection, raw}
		}})
	}
	return out
}

const networkListPath = networkExtensionPrefix + "networks"

func networkListRow(t *testing.T, id, name string, fields map[string]json.RawMessage) string {
	t.Helper()
	body := map[string]json.RawMessage{"id": json.RawMessage(fmt.Sprintf("%q", id)), "name": json.RawMessage(fmt.Sprintf("%q", name)), "status": json.RawMessage(`"ACTIVE"`), "project_id": json.RawMessage(`"native-project"`), "admin_state_up": json.RawMessage(`false`), "shared": json.RawMessage(`false`)}
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
func networkListPage(rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,"networks_links":[{"rel":"next","href":%q}]`, next)
	}
	return `{"networks":[` + rows + `]` + links + `}`
}
func networkListWant(t *testing.T, values []*networks.Network, err error, ids ...string) {
	t.Helper()
	actual := make([]string, 0, len(values))
	for _, v := range values {
		actual = append(actual, v.ID)
	}
	if err != nil || !reflect.DeepEqual(actual, append([]string{}, ids...)) {
		t.Fatal("actual native collection", actual, err, "want", ids)
	}
}
func networkListTarget() (map[string]json.RawMessage, map[string]any) {
	fields := map[string]json.RawMessage{"availability_zone_hints": json.RawMessage(`["az-one",null]`), "availability_zones": json.RawMessage(`["az-two",{"n":9007199254740993}]`), "created_at": json.RawMessage(`"2025-01-02T03:04:05+00:00"`), "updated_at": json.RawMessage(`"2025-01-03T03:04:05+00:00"`), "dns_domain": json.RawMessage(`"example."`), "is_default": json.RawMessage(`"false"`), "vlan_qinq": json.RawMessage(`-0.000e999999999999999999999`), "vlan_transparent": json.RawMessage(`[]`), "pvlan": json.RawMessage(`{"nullable":null}`), "mtu": json.RawMessage(`" +001500 "`), "qos_policy_id": json.RawMessage(`{"n":9007199254740993,"other":null}`), "revision_number": json.RawMessage(`9`), "segments": json.RawMessage(`[{"id":"segment","n":9007199254740993,"nullable":null}]`), "subnets": json.RawMessage(`["subnet-one",null]`)}
	filters := map[string]any{"availability_zone_hints": json.RawMessage(`["az-one",null]`), "availability_zones": json.RawMessage(`["az-two",{"n":9007199254740993}]`), "created_at": "2025-01-02T03:04:05+00:00", "updated_at": "2025-01-03T03:04:05+00:00", "dns_domain": "example.", "is_default": true, "is_vlan_qinq": false, "is_vlan_transparent": false, "pvlan": true, "mtu": 1500, "qos_policy_id": json.RawMessage(`{"n":9007199254740993}`), "revision_number": 9, "segments": json.RawMessage(`[{"id":"segment","n":9007199254740993,"nullable":null}]`), "subnet_ids": json.RawMessage(`["subnet-one",null]`)}
	return fields, filters
}

func TestNetworkListFiltersEntireQueryAndAliasDescriptor(t *testing.T) {
	canonical := networkListQueries()
	accepted := map[string]string{}
	for k, w := range canonical {
		accepted[k], accepted[w] = w, w
	}
	if len(canonical) != 23 || len(accepted) != 35 {
		t.Fatal(canonical, accepted)
	}
	for _, f := range networkListFixtures() {
		for k, w := range accepted {
			t.Run(f.name+"/"+k, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+networkListPath, func(wr http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{w: {"server-value"}}) {
						t.Error(k, r.URL)
					}
					testcloud.JSON(wr, 200, networkListPage(networkListRow(t, "result", "Different", nil), ""))
				})
				a := f.open(t, networkExtensionClient(cloud))
				rows, err := a.collection.All(context.Background(), resource.WithFilter(k, "server-value"))
				networkListWant(t, rows, err, "result")
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
					cloud.Mux.HandleFunc("GET "+networkListPath, func(wr http.ResponseWriter, r *http.Request) {
						want := url.Values{}
						if tc.want != nil {
							want[w] = tc.want
						}
						if !reflect.DeepEqual(r.URL.Query(), want) {
							t.Error(r.URL.Query(), want)
						}
						testcloud.JSON(wr, 200, networkListPage(networkListRow(t, "result", "Different", nil), ""))
					})
					rows, err := f.open(t, networkExtensionClient(cloud)).collection.All(context.Background(), resource.WithFilters(map[string]any{k: tc.value, w: json.RawMessage(`{`)}))
					networkListWant(t, rows, err, "result")
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
					cloud.Mux.HandleFunc("GET "+networkListPath, func(wr http.ResponseWriter, r *http.Request) {
						if r.URL.Query().Get(w) != want {
							t.Error(r.URL)
						}
						testcloud.JSON(wr, 200, networkListPage(networkListRow(t, "result", "Different", nil), ""))
					})
					rows, err := f.open(t, networkExtensionClient(cloud)).collection.All(context.Background(), opts...)
					networkListWant(t, rows, err, "result")
				})
			}
		}
		t.Run(f.name+"/scalar-encoding", func(t *testing.T) {
			cloud := testcloud.New(t)
			want := url.Values{"fields": {"id", "true", "false", "9007199254740993", ""}, "shared": {"false"}, "name": {""}, "limit": {"2e1"}, "router:external": {"true"}}
			cloud.Mux.HandleFunc("GET "+networkListPath, func(w http.ResponseWriter, r *http.Request) {
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, networkListPage(networkListRow(t, "result", "Different", nil), ""))
			})
			rows, err := f.open(t, networkExtensionClient(cloud)).collection.All(context.Background(), resource.WithFilters(map[string]any{"fields": []any{"id", nil, true, false, json.Number("9007199254740993"), ""}, "is_shared": false, "name": "", "limit": json.Number("2e1"), "is_router_external": true, "description": nil, "marker": []any{}}))
			networkListWant(t, rows, err, "result")
		})
	}
}

func TestNetworkListFiltersRawDescriptorsAndBooleanTruthiness(t *testing.T) {
	properties := map[string]string{"availability_zone_hints": "availability_zone_hints", "availability_zones": "availability_zones", "created_at": "created_at", "dns_domain": "dns_domain", "is_default": "is_default", "is_vlan_qinq": "vlan_qinq", "is_vlan_transparent": "vlan_transparent", "mtu": "mtu", "pvlan": "pvlan", "qos_policy_id": "qos_policy_id", "revision_number": "revision_number", "segments": "segments", "subnet_ids": "subnets", "updated_at": "updated_at"}
	fields, filters := networkListTarget()
	if len(properties) != 14 || len(filters) != 14 {
		t.Fatal(properties, filters)
	}
	for _, f := range networkListFixtures() {
		t.Run(f.name+"/all-fourteen", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+networkListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("local fields leaked", r.URL)
				}
				testcloud.JSON(w, 200, networkListPage(networkListRow(t, "target", "Different", fields), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for k, v := range filters {
				rows, err := a.collection.All(context.Background(), resource.WithFilter(k, v))
				networkListWant(t, rows, err, "target")
			}
			rows, err := a.collection.All(context.Background(), resource.WithFilters(filters))
			networkListWant(t, rows, err, "target")
			if calls.Load() != 15 || !reflect.DeepEqual(rows[0].Subnets, []string{"subnet-one", ""}) || !reflect.DeepEqual(rows[0].AvailabilityZoneHints, []string{"az-one", ""}) || rows[0].RevisionNumber != 9 || rows[0].CreatedAt.IsZero() {
				t.Fatal("raw comparison changed native model", rows[0], calls.Load())
			}
			for _, option := range []resource.ListOption{resource.WithFilter("segments", json.RawMessage(`[{"n":9007199254740993}]`)), resource.WithFilter("availability_zones", json.RawMessage(`["az-two",{"n":9007199254740992}]`)), resource.WithFilter("qos_policy_id", json.RawMessage(`{"n":9007199254740992}`)), resource.WithFilter("created_at", "2025-01-02T03:04:05Z"), resource.WithFilter("mtu", "1500"), resource.WithFilter("subnet_ids", []string{"subnet-one", ""})} {
				rows, err = a.collection.All(context.Background(), option)
				networkListWant(t, rows, err)
			}
		})
		t.Run(f.name+"/bool-null-zero-and-truthiness", func(t *testing.T) {
			cloud := testcloud.New(t)
			var all []string
			cases := []struct{ id, raw string }{{"missing", "missing"}, {"null", "null"}, {"false", "false"}, {"zero", "-0.000e999999999999999999999"}, {"empty-string", `""`}, {"empty-array", "[]"}, {"empty-object", "{}"}, {"true", "true"}, {"tiny", "1e-999999999999999999999"}, {"negative", "-9007199254740993"}, {"false-string", `"false"`}, {"space", `" "`}, {"array", "[null]"}, {"object", `{"nullable":null}`}}
			for _, tc := range cases {
				fields := map[string]json.RawMessage{}
				for _, wire := range []string{"is_default", "pvlan", "vlan_qinq", "vlan_transparent"} {
					if tc.raw != "missing" {
						fields[wire] = json.RawMessage(tc.raw)
					}
				}
				all = append(all, networkListRow(t, tc.id, "Different", fields))
			}
			cloud.Mux.HandleFunc("GET "+networkListPath, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, networkListPage(strings.Join(all, ","), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for _, key := range []string{"is_default", "pvlan", "is_vlan_qinq", "is_vlan_transparent"} {
				for _, tc := range []struct {
					value any
					ids   []string
				}{{nil, []string{"missing", "null"}}, {false, []string{"false", "zero", "empty-string", "empty-array", "empty-object"}}, {true, []string{"true", "tiny", "negative", "false-string", "space", "array", "object"}}, {1, nil}, {"false", nil}} {
					rows, err := a.collection.All(context.Background(), resource.WithFilter(key, tc.value))
					networkListWant(t, rows, err, tc.ids...)
				}
			}
		})
		t.Run(f.name+"/raw-null-empty-objects-and-time", func(t *testing.T) {
			cloud := testcloud.New(t)
			rows := networkListRow(t, "missing", "Different", nil) + "," + networkListRow(t, "null", "Different", map[string]json.RawMessage{"mtu": json.RawMessage(`null`), "qos_policy_id": json.RawMessage(`null`), "subnets": json.RawMessage(`null`)}) + "," + networkListRow(t, "empty", "Different", map[string]json.RawMessage{"mtu": json.RawMessage(`0`), "qos_policy_id": json.RawMessage(`{}`), "subnets": json.RawMessage(`[]`)}) + "," + networkListRow(t, "value", "Different", map[string]json.RawMessage{"mtu": json.RawMessage(`1.5e3`), "qos_policy_id": json.RawMessage(`{"n":1}`), "subnets": json.RawMessage(`[null]`), "created_at": json.RawMessage(`"2025-01-02T03:04:05"`), "updated_at": json.RawMessage(`"2025-01-03T03:04:05"`)})
			cloud.Mux.HandleFunc("GET "+networkListPath, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, networkListPage(rows, "")) })
			a := f.open(t, networkExtensionClient(cloud))
			for _, tc := range []struct {
				key string
				v   any
				ids []string
			}{{"mtu", nil, []string{"missing", "null"}}, {"mtu", 1500, []string{"value"}}, {"subnet_ids", nil, []string{"missing", "null"}}, {"subnet_ids", []any{}, []string{"empty"}}, {"subnet_ids", []any{nil}, []string{"value"}}, {"qos_policy_id", map[string]any{}, []string{"value"}}, {"created_at", "2025-01-02T03:04:05", []string{"value"}}, {"created_at", "2025-01-02T03:04:05Z", nil}} {
				v, e := a.collection.All(context.Background(), resource.WithFilter(tc.key, tc.v))
				networkListWant(t, v, e, tc.ids...)
			}
		})
	}
}

func TestNetworkListFiltersSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	for _, f := range networkListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+networkListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("vendor") != "kept" || r.URL.Query().Get("shared") != "false" || !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, networkListPage(networkListRow(t, "target", "Different", map[string]json.RawMessage{"subnets": json.RawMessage(`["original",null]`), "mtu": json.RawMessage(`1500`)}), ""))
			})
			values := []any{"original", nil}
			queryFields := []any{"id", "name"}
			bulk := map[string]any{"subnet_ids": values, "is_shared": false, "fields": queryFields}
			option := resource.WithFilters(bulk)
			values[0] = "changed"
			queryFields[0] = "changed"
			bulk["is_shared"] = true
			delete(bulk, "subnet_ids")
			a := f.open(t, networkExtensionClient(cloud))
			options := []resource.ListOption{option, resource.WithQuery("vendor", "kept")}
			rows, err := a.collection.All(context.Background(), options...)
			networkListWant(t, rows, err, "target")
			rows[0].Subnets[0] = "caller-mutated"
			var wg sync.WaitGroup
			for range 6 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					v, e := a.collection.All(context.Background(), options...)
					if e != nil || len(v) != 1 || !reflect.DeepEqual(v[0].Subnets, []string{"original", ""}) {
						t.Error(v, e)
					}
				}()
			}
			wg.Wait()
			for _, prefix := range [][]resource.ListOption{{resource.WithFilter("subnet_ids", math.NaN()), resource.WithFilter("subnet_ids", []any{"original", nil})}, {resource.WithFilter("headers", nil), resource.WithFilters(nil)}, {resource.WithFilter("mtu", make(chan int)), resource.WithFilters(map[string]any{})}} {
				opts := append(prefix, resource.WithFilter("is_shared", false), resource.WithFilter("fields", []string{"id", "name"}), resource.WithQuery("vendor", "kept"), resource.WithBodyFilter("mtu", 1500))
				v, e := a.collection.All(context.Background(), opts...)
				networkListWant(t, v, e, "target")
			}
			for _, tc := range []struct {
				opts []resource.ListOption
				want []string
			}{{[]resource.ListOption{resource.WithBodyFilter("subnets", []any{"wrong"}), resource.WithBodyFilter("subnet_ids", []any{"original", nil})}, []string{"target"}}, {[]resource.ListOption{resource.WithBodyFilter("subnet_ids", []any{"original", nil}), resource.WithBodyFilter("subnets", []any{"wrong"})}, nil}} {
				opts := append(tc.opts, resource.WithFilter("is_shared", false), resource.WithFilter("fields", []string{"id", "name"}), resource.WithQuery("vendor", "kept"))
				v, e := a.collection.All(context.Background(), opts...)
				networkListWant(t, v, e, tc.want...)
			}
			if calls.Load() != 12 {
				t.Fatal("snapshot/clear did not make independent requests", calls.Load())
			}
		})
	}
}

func TestNetworkListFiltersLazyPreflightAndNamespaceCollisions(t *testing.T) {
	for _, f := range networkListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+networkListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, networkListPage(networkListRow(t, "result", "Different", nil), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for _, pair := range [][]resource.ListOption{{resource.WithFilter("is_shared", false), resource.WithQuery("shared", "false")}, {resource.WithFilter("fields", nil), resource.WithQuery("fields", "")}, {resource.WithFilter("name", nil), resource.WithName("Target")}, {resource.WithFilter("status", "ACTIVE"), resource.WithStatus("ACTIVE")}, {resource.WithFilter("limit", 1), resource.WithPageSize(1)}, {resource.WithFilter("subnet_ids", nil), resource.WithBodyFilter("subnets", nil)}, {resource.WithFilter("is_vlan_qinq", false), resource.WithBodyFilter("vlan_qinq", false)}, {resource.WithFilter("mtu", 1500), resource.WithBodyFilter("mtu", 1500)}} {
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
			invalid := []resource.ListOption{resource.WithFilter("fields", map[string]any{"id": true}), resource.WithFilter("fields", [][]string{{"id"}}), resource.WithFilter("is_default", math.NaN()), resource.WithBodyFilter("name", "Different"), resource.WithBodyFilter("status", "ACTIVE"), resource.WithBodyFilter("id", "result"), resource.WithBodyFilter("tenant_id", "tenant"), resource.WithBodyFilters(map[string]any{"subnet_ids": nil, "subnets": nil}), resource.WithBodyFilters(map[string]any{"is_vlan_qinq": false, "vlan_qinq": false}), resource.WithBodyFilter("mtu", json.RawMessage(`{`)), resource.WithMaxItems(-1)}
			for _, k := range []string{"max_items", "paginated", "base_path", "allow_unknown_params", "session", "headers", "microversion", "resource_type", "jmespath_filters"} {
				invalid = append(invalid, resource.WithFilter(k, nil))
			}
			for _, opt := range invalid {
				seq := a.collection.List(context.Background(), opt)
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
			v, e := a.collection.All(context.Background(), resource.WithFilters(map[string]any{"subnets": make(chan int), "vlan_qinq": json.RawMessage(`{`), "vlan_transparent": math.NaN(), "tenant_id": func() {}, "unknown": make(chan int)}))
			networkListWant(t, v, e, "result")
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			v, e = a.collection.All(ctx, resource.WithFilters(nil))
			if v != nil || !errors.Is(e, context.Canceled) || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			_, e = trunks.New(networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(nil))
			if !errors.Is(e, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatal(e, calls.Load())
			}
		})
	}
}

func TestNetworkListFiltersControlsAndNativeWholePageFailures(t *testing.T) {
	modes := []string{"cap-body", "cap-name", "cap-status", "single-page", "break", "null-row", "empty-object", "late-null", "cap-null", "query-null", "mtu-fraction", "mtu-fraction-cap", "mtu-fraction-unselected", "bad-id", "bad-bool", "bad-subnets", "bad-hints", "bad-tags", "bad-revision", "bad-date", "mixed-date"}
	for _, f := range networkListFixtures() {
		for _, mode := range modes {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud, foreign := testcloud.New(t), testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); w.WriteHeader(500) })
				rows := networkListRow(t, "first", "Different", map[string]json.RawMessage{"subnets": json.RawMessage(`[]`), "status": json.RawMessage(`"DOWN"`)}) + "," + networkListRow(t, "target", "Target", map[string]json.RawMessage{"subnets": json.RawMessage(`["one"]`)})
				opts := []resource.ListOption{resource.WithFilter("subnet_ids", []any{}), resource.WithPaginated(false)}
				nativeFailure := false
				switch mode {
				case "cap-body":
					opts = []resource.ListOption{resource.WithFilter("subnet_ids", []string{"one"}), resource.WithMaxItems(1)}
				case "cap-name":
					opts = append(opts, resource.WithName("Target"), resource.WithMaxItems(1))
				case "cap-status":
					opts = append(opts, resource.WithStatus("ACTIVE"), resource.WithMaxItems(1))
				case "single-page":
				case "break":
					opts = []resource.ListOption{resource.WithFilter("subnet_ids", []any{})}
				case "null-row", "query-null":
					rows = "null"
					opts = []resource.ListOption{resource.WithFilter("is_default", nil), resource.WithMaxItems(1)}
					if mode == "query-null" {
						opts = []resource.ListOption{resource.WithFilter("name", "server-only"), resource.WithMaxItems(1)}
					}
				case "empty-object":
					rows = "{}"
					opts = []resource.ListOption{resource.WithFilter("is_default", nil), resource.WithMaxItems(1)}
				case "late-null", "cap-null":
					rows = networkListRow(t, "first", "Different", map[string]json.RawMessage{"subnets": json.RawMessage(`[]`)}) + ",null"
					if mode == "cap-null" {
						opts = append(opts, resource.WithMaxItems(1))
					}
				case "mtu-fraction", "mtu-fraction-cap", "mtu-fraction-unselected":
					rows = networkListRow(t, "first", "Target", map[string]json.RawMessage{"mtu": json.RawMessage(`1500`), "subnets": json.RawMessage(`[]`)}) + "," + networkListRow(t, "fraction", "Other", map[string]json.RawMessage{"mtu": json.RawMessage(`1500.5`), "subnets": json.RawMessage(`[]`)})
					opts = []resource.ListOption{resource.WithFilter("mtu", 1500), resource.WithName("Target"), resource.WithPaginated(false)}
					if mode == "mtu-fraction-cap" {
						opts = append(opts, resource.WithMaxItems(1))
					}
					if mode == "mtu-fraction-unselected" {
						opts = []resource.ListOption{resource.WithFilter("subnet_ids", []any{}), resource.WithPaginated(false)}
					}
				default:
					key, raw := "subnets", `[1]`
					switch mode {
					case "bad-id":
						key, raw = "id", "false"
					case "bad-bool":
						key, raw = "shared", `"false"`
					case "bad-hints":
						key, raw = "availability_zone_hints", `[{}]`
					case "bad-tags":
						key, raw = "tags", `[true]`
					case "bad-revision":
						key, raw = "revision_number", `"9"`
					case "bad-date":
						key, raw = "created_at", `"not-time"`
					case "mixed-date":
						key, raw = "created_at", `"2025-01-02T03:04:05"`
					}
					fields := map[string]json.RawMessage{key: json.RawMessage(raw)}
					if mode == "mixed-date" {
						fields["updated_at"] = json.RawMessage(`"2025-01-03T03:04:05Z"`)
					}
					rows += "," + networkListRow(t, "invalid", "Other", fields)
					opts = append(opts, resource.WithMaxItems(1))
					nativeFailure = true
				}
				cloud.Mux.HandleFunc("GET "+networkListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Has("limit") {
						t.Error("cap became wire limit", r.URL)
					}
					testcloud.JSON(w, 200, networkListPage(rows, foreign.Server.URL+"/foreign"))
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
						networkListWant(t, v, e, "first")
					} else {
						networkListWant(t, v, e)
					}
				case mode == "single-page" || mode == "mtu-fraction-cap":
					networkListWant(t, v, e, "first")
				case mode == "query-null" || mode == "empty-object":
					networkListWant(t, v, e, "")
				case mode == "mtu-fraction-unselected":
					networkListWant(t, v, e, "first", "fraction")
				case mode == "null-row" || mode == "late-null" || mode == "mtu-fraction":
					if v != nil || !errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal("selected raw failure was hidden", v, e)
					}
					var operation *resource.OperationError
					if !errors.As(e, &operation) || operation.Operation != "list" {
						t.Fatal(e)
					}
					wantKind := "networks"
					if f.name == "manual" {
						wantKind = "network"
					}
					if operation.Resource != wantKind {
						t.Fatal("collection Kind changed", operation.Resource, wantKind)
					}
				case nativeFailure:
					if v != nil || e == nil || errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal("native wholepage error changed", v, e)
					}
					if mode == "bad-date" || mode == "mixed-date" {
						var cause *time.ParseError
						if !errors.As(e, &cause) {
							t.Fatal(e)
						}
					} else {
						var cause *json.UnmarshalTypeError
						if !errors.As(e, &cause) {
							t.Fatal(e)
						}
					}
				default:
					t.Fatal("unhandled mode", mode)
				}
				if calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal("cap/current-page error fetched continuation", calls.Load(), followed.Load())
				}
			})
		}
	}
}

func TestNetworkListFiltersPagingLiveSourceAndNativeStatus(t *testing.T) {
	for _, f := range networkListFixtures() {
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
				cloud.Mux.HandleFunc("GET "+networkListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("X-Configured") != "kept" || r.Header.Get("OpenStack-API-Version") != "network 2.0" || !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) || r.URL.Query().Get("shared") != "false" || r.URL.Query().Get("status") != "vendor-status" {
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
							testcloud.JSON(w, 200, networkListPage(networkListRow(t, "invalid", "Different", map[string]json.RawMessage{"shared": json.RawMessage(`"false"`)}), ""))
							return
						}
						id := "second"
						if mode == "duplicates" {
							id = "first"
						}
						testcloud.JSON(w, 200, networkListPage(networkListRow(t, id, "Different", map[string]json.RawMessage{"is_default": json.RawMessage(`true`)}), ""))
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
					next := cloud.Server.URL + networkListPath + "?marker=next&shared=false&fields=id&fields=name&status=vendor-status"
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
					row := networkListRow(t, "first", "Different", map[string]json.RawMessage{"is_default": raw})
					if mode == "wrong-links" {
						testcloud.JSON(w, 300, `{"networks":[`+row+`],"links":{"next":`+fmt.Sprintf("%q", next)+`},"next":`+fmt.Sprintf("%q", next)+`}`)
						return
					}
					testcloud.JSON(w, 300, networkListPage(row, next))
				})
				a := f.open(t, c)
				v, e := a.collection.All(ctx, resource.WithFilter("is_default", true), resource.WithFilter("is_shared", false), resource.WithFilter("fields", []string{"id", "name"}), resource.WithQuery("status", "vendor-status"))
				switch mode {
				case "complete":
					networkListWant(t, v, e, "first", "second")
				case "first-filtered":
					networkListWant(t, v, e, "second")
				case "duplicates":
					networkListWant(t, v, e, "first", "first")
				case "wrong-links":
					networkListWant(t, v, e, "first")
				case "204":
					networkListWant(t, v, e)
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

func TestNetworkListFiltersNativeAndIdentitySurfaceIsolation(t *testing.T) {
	for _, f := range networkListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets atomic.Int32
			want := url.Values{"name": {"server-pattern"}, "status": {"vendor-status"}, "subnets": {"raw-subnets"}, "subnet_ids": {"raw-subnet-ids"}, "mtu": {"raw-mtu"}, "is_default": {"raw-flag"}, "tenant_id": {"raw-tenant"}, "vendor": {"kept"}}
			cloud.Mux.HandleFunc("GET "+networkListPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("query was classified unexpectedly", r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, networkListPage(networkListRow(t, "target", "Target", map[string]json.RawMessage{"subnets": json.RawMessage(`[]`), "status": json.RawMessage(`"active"`)})+","+networkListRow(t, "other", "Other", map[string]json.RawMessage{"subnets": json.RawMessage(`["one"]`), "status": json.RawMessage(`"DOWN"`)}), ""))
			})
			cloud.Mux.HandleFunc("GET "+networkListPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"mtu": {"raw-mtu"}, "subnet_ids": {"raw-subnet-ids"}, "status": {"vendor-status"}}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"network":`+networkListRow(t, "lookup", "Different", map[string]json.RawMessage{"mtu": json.RawMessage(`false`), "is_default": json.RawMessage(`"false"`)})+`}`)
			})
			a := f.open(t, networkExtensionClient(cloud))
			var raw []resource.ListOption
			var native []networks.ListOption
			for k, v := range want {
				raw = append(raw, resource.WithQuery(k, v[0]))
				native = append(native, networks.WithListQuery(k, v[0]))
			}
			v, e := a.collection.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilter("subnet_ids", []any{}))...)
			networkListWant(t, v, e, "target")
			v, e = a.collection.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilters(nil))...)
			networkListWant(t, v, e, "target", "other")
			var nativeIDs []string
			for v, e := range a.api.List(context.Background(), native...) {
				if e != nil {
					t.Fatal(e)
				}
				nativeIDs = append(nativeIDs, v.ID)
			}
			if !reflect.DeepEqual(nativeIDs, []string{"target", "other"}) {
				t.Fatal(nativeIDs)
			}
			found, e := a.collection.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindQuery("mtu", "raw-mtu"), resource.WithIdentityFindQuery("subnet_ids", "raw-subnet-ids"), resource.WithIdentityFindQuery("status", "vendor-status"))
			if e != nil || found == nil || found.ID != "lookup" || gets.Load() != 1 || lists.Load() != 3 {
				t.Fatal(found, e, gets.Load(), lists.Load())
			}
			want["name"], want["status"] = []string{"Target"}, []string{"ACTIVE"}
			var local []resource.ListOption
			for k, v := range want {
				if k != "name" && k != "status" {
					local = append(local, resource.WithQuery(k, v[0]))
				}
			}
			local = append(local, resource.WithName("Target"), resource.WithStatus("ACTIVE"), resource.WithBodyFilter("subnet_ids", []any{}))
			v, e = a.collection.All(context.Background(), local...)
			networkListWant(t, v, e, "target")
		})
	}
}
