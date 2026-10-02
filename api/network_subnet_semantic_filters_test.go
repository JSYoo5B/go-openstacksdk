package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network/v2/subnets"
	"gophercloudsdk/resource"
)

// Independent expected table from openstacksdk ef55d7d Subnet._query_mapping,
// inherited tags and QueryParameters defaults. The HTTP fixtures do not load
// a runtime manifest or derive expectations from the generated descriptor.
func subnetSemanticQueries() map[string]string {
	return map[string]string{
		"any_tags": "tags-any", "cidr": "cidr", "description": "description",
		"dns_publish_fixed_ip": "dns_publish_fixed_ip", "fields": "fields", "gateway_ip": "gateway_ip",
		"id": "id", "ip_version": "ip_version", "ipv6_address_mode": "ipv6_address_mode", "ipv6_ra_mode": "ipv6_ra_mode",
		"is_dhcp_enabled": "enable_dhcp", "limit": "limit", "marker": "marker", "name": "name", "network_id": "network_id",
		"not_any_tags": "not-tags-any", "not_tags": "not-tags", "project_id": "project_id", "segment_id": "segment_id",
		"sort_dir": "sort_dir", "sort_key": "sort_key", "subnet_pool_id": "subnetpool_id", "tags": "tags",
		"use_default_subnet_pool": "use_default_subnetpool",
	}
}

func TestNetworkSubnetSemanticFiltersEntireQueryDescriptor(t *testing.T) {
	canonical := subnetSemanticQueries()
	accepted := make(map[string]string)
	for name, wire := range canonical {
		accepted[name], accepted[wire] = wire, wire
	}
	if len(canonical) != 24 || len(accepted) != 30 {
		t.Fatal("independent source table changed", len(canonical), len(accepted))
	}
	for _, f := range subnetRawBodyFixtures() {
		for field, wire := range accepted {
			t.Run(f.name+"/"+field, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{wire: {"wire-value"}}) {
						t.Error("semantic query mapping changed", field, r.URL)
					}
					// Query name/id/tags must remain server-only rather than
					// becoming local predicates on this unrelated return value.
					testcloud.JSON(w, 200, subnetRawBodyPage(subnetRawBodyRow(t, "server-returned", "Different", nil), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter(field, "wire-value"))
				subnetRawBodyWant(t, values, err, "server-returned")
				if calls.Load() != 1 {
					t.Fatal("query classification caused extra HTTP", calls.Load())
				}
			})
		}
	}
}

func TestNetworkSubnetSemanticFiltersEncodingAndCanonicalPrecedence(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		for _, tc := range []struct {
			name   string
			fields map[string]any
			want   url.Values
		}{
			{"scalars-and-repeats", map[string]any{"fields": []any{"prefix_length", nil, "revision_number", true, false, json.Number("9007199254740993"), ""}, "is_dhcp_enabled": false, "ip_version": json.Number("4.0"), "limit": json.Number("2e1"), "name": "", "gateway_ip": nil, "tags": []any{}}, url.Values{"fields": {"prefix_length", "revision_number", "true", "false", "9007199254740993", ""}, "enable_dhcp": {"false"}, "ip_version": {"4.0"}, "limit": {"2e1"}, "name": {""}}},
			{"all-null-array", map[string]any{"fields": []any{nil, nil}}, url.Values{}},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), tc.want) {
						t.Error("JSON query encoding changed", r.URL.Query(), tc.want)
					}
					testcloud.JSON(w, 200, subnetRawBodyPage(subnetRawBodyRow(t, "result", "Different", nil), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(tc.fields))
				subnetRawBodyWant(t, values, err, "result")
			})
		}
		for canonical, wire := range subnetSemanticQueries() {
			if canonical == wire {
				continue
			}
			for _, tc := range []struct {
				name  string
				value any
				want  []string
			}{
				{"nil", nil, nil}, {"false", false, []string{"false"}}, {"empty-string", "", []string{""}},
				{"empty-array", []any{}, nil}, {"repeats", []any{"first", nil, "last"}, []string{"first", "last"}},
			} {
				t.Run(f.name+"/"+canonical+"/"+tc.name, func(t *testing.T) {
					cloud := testcloud.New(t)
					cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
						want := make(url.Values)
						if tc.want != nil {
							want[wire] = tc.want
						}
						if !reflect.DeepEqual(r.URL.Query(), want) {
							t.Error("bulk canonical value did not win over its wire alias", r.URL.Query(), want)
						}
						testcloud.JSON(w, 200, subnetRawBodyPage(subnetRawBodyRow(t, "result", "Different", nil), ""))
					})
					// The unused wire value is not serialized as a query.
					values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{canonical: tc.value, wire: json.RawMessage(`{`)}))
					subnetRawBodyWant(t, values, err, "result")
				})
			}
		}
		for _, reverse := range []bool{false, true} {
			t.Run(f.name+"/individual-last-wins/"+map[bool]string{false: "canonical-last", true: "wire-last"}[reverse], func(t *testing.T) {
				cloud := testcloud.New(t)
				want := "true"
				opts := []resource.ListOption{resource.WithFilter("enable_dhcp", false), resource.WithFilter("is_dhcp_enabled", true)}
				if reverse {
					want, opts[0], opts[1] = "false", opts[1], opts[0]
				}
				cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), url.Values{"enable_dhcp": {want}}) {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, subnetRawBodyPage(subnetRawBodyRow(t, "result", "Different", nil), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), opts...)
				subnetRawBodyWant(t, values, err, "result")
			})
		}
	}
}

func TestNetworkSubnetSemanticFiltersNineLocalBodyClassifications(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		for key, raw := range subnetRawBodyValues() {
			field := key
			if field == "prefixlen" {
				field = "prefix_length"
			}
			t.Run(f.name+"/"+field, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				fields := subnetRawBodyValues()
				rows := subnetRawBodyRow(t, "target", "Different", fields)
				switch key {
				case "allocation_pools", "dns_nameservers", "host_routes", "service_types":
					fields[key] = json.RawMessage(`[]`)
				case "prefixlen":
					fields[key] = json.RawMessage(`"24"`)
				case "revision_number":
					fields[key] = json.RawMessage(`0`)
				case "tenant_id":
					fields[key] = json.RawMessage(`"project-independent"`)
				case "created_at":
					fields[key] = json.RawMessage(`"2025-01-02T03:04:05Z"`)
				case "updated_at":
					fields[key] = json.RawMessage(`"2025-01-03T03:04:05Z"`)
				}
				rows += "," + subnetRawBodyRow(t, "decoy", "Different", fields)
				cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{"name": {"server-pattern"}}) {
						t.Error("Body classification became a native query field", field, r.URL)
					}
					testcloud.JSON(w, 200, subnetRawBodyPage(rows, ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter(field, raw), resource.WithFilter("name", "server-pattern"))
				subnetRawBodyWant(t, values, err, "target")
				if calls.Load() != 1 {
					t.Fatal("Body classification refetched raw values", calls.Load())
				}
			})
		}
	}
}

func TestNetworkSubnetSemanticFiltersSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			fields := []any{"id", "name", nil}
			dns := []any{"192.0.2.1", nil}
			enabled := false
			prefix := json.RawMessage(`24`)
			input := map[string]any{"fields": fields, "is_dhcp_enabled": &enabled, "dns_nameservers": dns, "prefix_length": prefix}
			bulk := resource.WithFilters(input)
			input["fields"], fields[0], dns[0], enabled, prefix[0] = nil, "changed", "changed", true, '0'
			base := url.Values{"vendor": {"kept"}}
			full := url.Values{"vendor": {"kept"}, "fields": {"id", "name"}, "enable_dhcp": {"false"}}
			var expected atomic.Value
			expected.Store(full)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), expected.Load().(url.Values)) {
					t.Error("namespace snapshot or clear changed wire fields", r.URL.Query(), expected.Load())
				}
				values := subnetRawBodyValues()
				rows := subnetRawBodyRow(t, "target", "Different", values)
				values["dns_nameservers"] = json.RawMessage(`[]`)
				rows += "," + subnetRawBodyRow(t, "dns-other", "Different", values)
				values["tenant_id"], values["dns_nameservers"] = json.RawMessage(`"other-tenant"`), subnetRawBodyValues()["dns_nameservers"]
				rows += "," + subnetRawBodyRow(t, "tenant-other", "Different", values)
				testcloud.JSON(w, 200, subnetRawBodyPage(rows, ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			baseOpts := []resource.ListOption{resource.WithQuery("vendor", "kept"), resource.WithBodyFilter("tenant_id", "tenant-original")}
			all := func(extra ...resource.ListOption) ([]*subnets.Subnet, error) {
				return a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), baseOpts...), extra...)...)
			}
			values, err := all(bulk)
			subnetRawBodyWant(t, values, err, "target")
			values, err = all(bulk, resource.WithFilter("dns_nameservers", []any{}))
			subnetRawBodyWant(t, values, err, "dns-other")
			expected.Store(base)
			for _, replacement := range []resource.ListOption{resource.WithFilters(nil), resource.WithFilters(map[string]any{}), resource.WithFilters(map[string]any{"prefix_length": 24}), resource.WithFilters(map[string]any{"unknown": make(chan int)})} {
				values, err := all(bulk, replacement)
				subnetRawBodyWant(t, values, err, "target", "dns-other")
			}
			// Semantic options validate the final selected values, so a
			// superseded invalid capture or cleared control cannot poison
			// subsequent iteration. Other namespaces remain intact.
			for _, tc := range []struct {
				options []resource.ListOption
				ids     []string
			}{
				{[]resource.ListOption{resource.WithFilter("fields", make(chan int)), resource.WithFilter("fields", nil)}, []string{"target", "dns-other"}},
				{[]resource.ListOption{resource.WithFilter("dns_nameservers", json.RawMessage(`{`)), resource.WithFilter("dns_nameservers", []any{})}, []string{"dns-other"}},
				{[]resource.ListOption{resource.WithFilters(map[string]any{"fields": map[string]any{"bad": true}}), resource.WithFilters(nil)}, []string{"target", "dns-other"}},
				{[]resource.ListOption{resource.WithFilter("microversion", nil), resource.WithFilters(map[string]any{})}, []string{"target", "dns-other"}},
				{[]resource.ListOption{resource.WithFilters(map[string]any{"fields": json.RawMessage(`{`), "dns_nameservers": make(chan int)}), resource.WithFilters(map[string]any{"prefix_length": 24})}, []string{"target", "dns-other"}},
			} {
				values, err := all(tc.options...)
				subnetRawBodyWant(t, values, err, tc.ids...)
			}
			expected.Store(full)
			opts := append(append([]resource.ListOption(nil), baseOpts...), bulk)
			seq := a.Resources.List(context.Background(), opts...)
			opts[len(opts)-1] = resource.WithFilters(nil)
			for range 2 {
				var ids []string
				for value, err := range seq {
					if err != nil {
						t.Fatal(err)
					}
					ids = append(ids, value.ID)
				}
				if !reflect.DeepEqual(ids, []string{"target"}) {
					t.Fatal("caller option slice changed retained semantics", ids)
				}
			}
			var group sync.WaitGroup
			for range 6 {
				group.Add(1)
				go func() {
					defer group.Done()
					values, err := all(bulk)
					if err != nil || len(values) != 1 || values[0].ID != "target" {
						t.Error(values, err)
					}
				}()
			}
			group.Wait()
			if calls.Load() != 19 {
				t.Fatal("semantic filters made extra HTTP or lost independent reuse", calls.Load())
			}
		})
	}
}

func TestNetworkSubnetSemanticFiltersUnknownReservedAndNamespaceCollisions(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"project_id": {"project"}}) {
					t.Error("unknown semantic value leaked or activated a local predicate", r.URL)
				}
				testcloud.JSON(w, 200, subnetRawBodyPage(subnetRawBodyRow(t, "server-returned", "Different", nil), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			values, err := a.Resources.All(context.Background(), resource.WithFilters(map[string]any{"project_id": "project", "status": make(chan int), "prefixlen": json.RawMessage(`{`), "unknown": math.NaN(), "CIDR": func() {}}))
			subnetRawBodyWant(t, values, err, "server-returned")
			values, err = a.Resources.All(context.Background(), resource.WithFilter("project_id", "project"), resource.WithFilter("status", make(chan int)), resource.WithFilter("prefixlen", json.RawMessage(`{`)))
			subnetRawBodyWant(t, values, err, "server-returned")
			before := calls.Load()
			collisions := [][]resource.ListOption{
				{resource.WithFilter("is_dhcp_enabled", false), resource.WithQuery("enable_dhcp", "false")},
				{resource.WithFilter("limit", 10), resource.WithPageSize(10)},
				{resource.WithFilter("name", nil), resource.WithName("Target")},
				{resource.WithFilter("name", "Target"), resource.WithName("Target")},
				{resource.WithFilter("fields", []any{}), resource.WithQuery("fields", "")},
				{resource.WithFilter("prefix_length", 24), resource.WithBodyFilter("prefixlen", 24)},
				{resource.WithFilter("dns_nameservers", nil), resource.WithBodyFilter("dns_nameservers", nil)},
			}
			for _, options := range collisions {
				for _, reverse := range []bool{false, true} {
					opts := append([]resource.ListOption(nil), options...)
					if reverse {
						opts[0], opts[1] = opts[1], opts[0]
					}
					values, err := a.Resources.All(context.Background(), opts...)
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
						t.Fatal("namespace collision was order-dependent or performed HTTP", values, err, calls.Load())
					}
				}
			}
			for _, key := range []string{"session", "paginated", "base_path", "allow_unknown_params", "microversion", "headers", "max_items", "resource_type", "jmespath_filters"} {
				values, err := a.Resources.All(context.Background(), resource.WithFilter(key, nil))
				if values != nil || (!errors.Is(err, resource.ErrInvalidOption) && !errors.Is(err, resource.ErrUnsupported)) || calls.Load() != before {
					t.Fatal("source control entered semantic namespace", key, values, err, calls.Load())
				}
			}
			for _, value := range []any{map[string]any{"field": "id"}, []any{[]any{"id"}}, make(chan int), json.RawMessage(`{`)} {
				seq := a.Resources.List(context.Background(), resource.WithFilter("fields", value))
				if calls.Load() != before {
					t.Fatal("iterator construction performed HTTP")
				}
				var seen int
				for value, err := range seq {
					if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(value, err)
					}
					seen++
				}
				if seen != 1 {
					t.Fatal("invalid known query did not yield one terminal error", seen)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if values, err := a.Resources.All(ctx, resource.WithFilters(nil)); values != nil || !errors.Is(err, context.Canceled) || calls.Load() != before {
				t.Fatal("canceled semantic iteration performed HTTP", values, err, calls.Load())
			}
		})
	}
}

func TestNetworkSubnetSemanticFiltersRawAndNativeSurfaceIsolation(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets atomic.Int32
			cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				want := url.Values{"prefix_length": {"wire-alias"}, "prefixlen": {"wire-prefix"}, "revision_number": {"wire-revision"}, "tenant_id": {"wire-tenant"}, "status": {"vendor-status"}}
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("raw wire options changed under semantics", r.URL.Query(), want)
				}
				fields := subnetRawBodyValues()
				rows := subnetRawBodyRow(t, "target", "Different", fields)
				fields["prefixlen"] = json.RawMessage(`"24"`)
				rows += "," + subnetRawBodyRow(t, "string-prefix", "Different", fields)
				testcloud.JSON(w, 200, subnetRawBodyPage(rows, ""))
			})
			cloud.Mux.HandleFunc("GET "+subnetRawBodyPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"status": {"vendor-status"}, "prefix_length": {"wire-alias"}}) {
					t.Error("FindIdentity acquired semantic classification", r.URL)
				}
				testcloud.JSON(w, 200, `{"subnet":`+subnetRawBodyRow(t, "lookup", "Different", map[string]json.RawMessage{"prefixlen": json.RawMessage(`false`)})+`}`)
			})
			a := f.open(t, networkExtensionClient(cloud))
			var raw []resource.ListOption
			var native []subnets.ListOption
			for _, item := range [][2]string{{"prefix_length", "wire-alias"}, {"prefixlen", "wire-prefix"}, {"revision_number", "wire-revision"}, {"tenant_id", "wire-tenant"}, {"status", "vendor-status"}} {
				raw = append(raw, resource.WithQuery(item[0], item[1]))
				native = append(native, subnets.WithListQuery(item[0], item[1]))
			}
			values, err := a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilter("prefix_length", 24))...)
			subnetRawBodyWant(t, values, err, "target")
			values, err = a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilters(nil))...)
			subnetRawBodyWant(t, values, err, "target", "string-prefix")
			var ids []string
			for value, err := range a.List(context.Background(), native...) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, value.ID)
			}
			if !reflect.DeepEqual(ids, []string{"target", "string-prefix"}) {
				t.Fatal("native List acquired semantic predicate", ids)
			}
			value, err := a.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindQuery("status", "vendor-status"), resource.WithIdentityFindQuery("prefix_length", "wire-alias"))
			if err != nil || value == nil || value.ID != "lookup" || lists.Load() != 3 || gets.Load() != 1 {
				t.Fatal("semantic options affected native identity path", value, err, lists.Load(), gets.Load())
			}
		})
	}
	// QoS keeps its explicit Body-only policy; AddressGroup has its own descriptor.
	boundaries := networkBodyFilterFixtures()
	for _, f := range boundaries {
		if f.meta.name != "qos-leaf" {
			continue
		}
		t.Run("explicit-body-only/"+f.meta.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			target, _, _ := networkBodyFilterArrays(f)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+networkQoSAddressesPath(f.meta), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, networkQoSAddressesPage(f.meta, networkBodyFilterRow(f, "target", "Different", target), ""))
			})
			access := f.open(t, networkExtensionClient(cloud))
			for _, option := range []resource.ListOption{resource.WithFilter("name", "Different"), resource.WithFilters(nil)} {
				values, err := access.all(context.Background(), option)
				if values != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
					t.Fatal("unaudited semantic binding became available", values, err, calls.Load())
				}
			}
			values, err := access.all(context.Background(), resource.WithBodyFilter(f.field, json.RawMessage(target)))
			networkBodyFilterWantIDs(t, values, err, "target")
		})
	}
	for _, f := range networkSubnetBodyFixtures() {
		if f.name != "network-leaf" && f.name != "subnet-pool-leaf" {
			continue
		}
		t.Run("explicit-body-only/"+f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			target, _, _ := networkSubnetBodyArrays(f)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+networkSubnetBodyPath(f), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, networkSubnetBodyPage(f, networkSubnetBodyRow(f, "target", "Different", target), ""))
			})
			access := f.open(t, networkExtensionClient(cloud))
			for _, option := range []resource.ListOption{resource.WithFilter("name", "Different"), resource.WithFilters(nil)} {
				values, err := access.all(context.Background(), option)
				if values != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
					t.Fatal("unaudited semantic binding became available", values, err, calls.Load())
				}
			}
			values, err := access.all(context.Background(), resource.WithBodyFilter(f.field, json.RawMessage(target)))
			networkBodyFilterWantIDs(t, values, err, "target")
		})
	}
}

func TestNetworkSubnetSemanticFiltersControlsAndTerminalPages(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		for _, mode := range []string{"cap", "single-page", "break", "two-pages", "late-http", "cycle", "whole-page", "cancel"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
					page := calls.Add(1)
					q := r.URL.Query()
					if q.Get("name") != "server-pattern" || !reflect.DeepEqual(q["fields"], []string{"id", "name"}) || q.Has("tenant_id") || q.Has("dns_nameservers") || q.Has("limit") || q.Has("max_items") || q.Has("paginated") {
						t.Error("semantic Body/control leaked to URL", r.URL)
					}
					q.Set("marker", "second")
					next := cloud.Server.URL + subnetRawBodyPath + "?" + q.Encode()
					fields := map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant-original"`), "dns_nameservers": json.RawMessage(`["192.0.2.1",null]`)}
					candidate := subnetRawBodyRow(t, "candidate", "Different", fields)
					rows := candidate
					if mode == "cap" || mode == "single-page" || mode == "break" || mode == "two-pages" && page == 1 {
						rows = subnetRawBodyRow(t, "wrong-tenant", "Different", map[string]json.RawMessage{"project_id": fields["tenant_id"], "dns_nameservers": fields["dns_nameservers"]}) + "," + subnetRawBodyRow(t, "wrong-dns", "Different", map[string]json.RawMessage{"tenant_id": fields["tenant_id"], "dns_nameservers": json.RawMessage(`[]`)})
						if mode != "two-pages" {
							rows += "," + candidate
							next = "https://foreign.invalid/not-read"
						}
					} else if mode == "whole-page" {
						rows += "," + subnetRawBodyRow(t, "bad", "Other", map[string]json.RawMessage{"dns_nameservers": json.RawMessage(`[false]`)})
					} else if page > 1 {
						switch mode {
						case "late-http":
							w.WriteHeader(503)
							return
						case "two-pages":
							next = ""
						case "cycle":
							rows = subnetRawBodyRow(t, "nonmatch", "Different", nil)
						case "cancel":
							cancel()
							next = ""
						}
					}
					testcloud.JSON(w, 200, subnetRawBodyPage(rows, next))
				})
				a := f.open(t, networkExtensionClient(cloud))
				opts := []resource.ListOption{resource.WithFilters(map[string]any{"tenant_id": "tenant-original", "dns_nameservers": []any{"192.0.2.1", nil}, "name": "server-pattern", "fields": []string{"id", "name"}})}
				switch mode {
				case "cap":
					opts = append(opts, resource.WithMaxItems(2))
				case "whole-page":
					opts = append(opts, resource.WithMaxItems(1))
				case "single-page":
					opts = append(opts, resource.WithPaginated(false))
				case "break":
					var seen int
					for value, err := range a.Resources.List(ctx, opts...) {
						if err != nil || value == nil || value.ID != "candidate" {
							t.Fatal(value, err)
						}
						seen++
						break
					}
					if seen != 1 || calls.Load() != 1 {
						t.Fatal(seen, calls.Load())
					}
					return
				}
				values, err := a.Resources.All(ctx, opts...)
				switch mode {
				case "cap":
					subnetRawBodyWant(t, values, err)
				case "single-page", "two-pages":
					subnetRawBodyWant(t, values, err, "candidate")
				default:
					if values != nil || err == nil || mode == "late-http" && !gophercloud.ResponseCodeIs(err, 503) || mode == "cycle" && !errors.Is(err, resource.ErrPaginationCycle) || mode == "cancel" && !errors.Is(err, context.Canceled) {
						t.Fatal("semantic filters hid terminal page failure", values, err)
					}
					if mode == "whole-page" {
						var cause *json.UnmarshalTypeError
						if !errors.As(err, &cause) {
							t.Fatal("native whole-page cause was lost", err)
						}
					}
				}
				wantCalls := int32(2)
				if mode == "cap" || mode == "single-page" || mode == "whole-page" {
					wantCalls = 1
				}
				if calls.Load() != wantCalls {
					t.Fatal("extra HTTP or hidden advertised continuation", calls.Load(), wantCalls)
				}
			})
		}
	}
}
