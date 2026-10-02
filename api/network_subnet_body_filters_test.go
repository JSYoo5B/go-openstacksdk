package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/network/v2/extensions/subnetpools"
	"gophercloudsdk/network/v2/extensions/trunks"
	"gophercloudsdk/network/v2/networks"
	"gophercloudsdk/resource"
)

// Pinned Python Network.subnet_ids is a Body alias for subnets; SubnetPool
// prefixes is also a Body field. Trunk.sub_ports is a mapped query instead.
// These ordinary collection tests retain native typed decoding: nil slices
// project to null, ordered strings remain literal, and no CIDR normalization
// or original wire-presence restoration is performed by local filters.
type networkSubnetBodyFixture struct {
	name, singular, plural, field string
	open                          func(*testing.T, *gophercloud.ServiceClient) networkBodyFilterAccess
}

func networkSubnetBodyConnection(t *testing.T, c *gophercloud.ServiceClient) *network.Service {
	t.Helper()
	conn, err := sdk.FromProvider(c.ProviderClient, sdk.WithEndpoint(sdk.Network, c.Endpoint))
	if err != nil {
		t.Fatal(err)
	}
	s, err := conn.Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw := s.RawClient()
	raw.ResourceBase, raw.Microversion, raw.MoreHeaders = c.ResourceBase, c.Microversion, c.MoreHeaders
	again, err := conn.Network(context.Background())
	if err != nil || again != s || raw.ProviderClient != c.ProviderClient || s.API.Networks.RawClient() != raw || s.API.SubnetPools.RawClient() != raw {
		t.Fatal("subnet Body-filter APIs did not share the cached network client", err)
	}
	return s
}

func networkSubnetBodyFixtures() []networkSubnetBodyFixture {
	var fixtures []networkSubnetBodyFixture
	for _, mode := range []string{"leaf", "connection", "manual"} {
		fixtures = append(fixtures, networkSubnetBodyFixture{
			name: "network-" + mode, singular: "network", plural: "networks", field: "subnets",
			open: func(t *testing.T, c *gophercloud.ServiceClient) networkBodyFilterAccess {
				a := networks.New(c)
				collection, raw := a.Resources, a.RawClient()
				if mode != "leaf" {
					s := networkSubnetBodyConnection(t, c)
					a, raw = s.API.Networks, s.RawClient()
					collection = a.Resources
					if mode == "manual" {
						collection = s.Networks
					}
				}
				view := func(v *networks.Network) *networkExtensionView {
					return &networkExtensionView{id: v.ID, name: v.Name, project: v.ProjectID, nested: !v.CreatedAt.IsZero() && !v.UpdatedAt.IsZero() && v.RevisionNumber == 9}
				}
				return networkBodyFilterAccessFor(collection.FindIdentity, collection, func(ctx context.Context) iter.Seq2[*networks.Network, error] { return a.List(ctx) }, raw, view)
			},
		})
	}
	for _, connected := range []bool{false, true} {
		name := "subnet-pool-leaf"
		if connected {
			name = "subnet-pool-connection"
		}
		fixtures = append(fixtures, networkSubnetBodyFixture{
			name: name, singular: "subnetpool", plural: "subnetpools", field: "prefixes",
			open: func(t *testing.T, c *gophercloud.ServiceClient) networkBodyFilterAccess {
				a := subnetpools.New(c)
				if connected {
					a = networkSubnetBodyConnection(t, c).API.SubnetPools
				}
				view := func(v *subnetpools.SubnetPool) *networkExtensionView {
					return &networkExtensionView{id: v.ID, name: v.Name, project: v.ProjectID, nested: v.DefaultPrefixLen == 24 && v.MinPrefixLen == 16 && v.MaxPrefixLen == 28 && !v.CreatedAt.IsZero() && !v.UpdatedAt.IsZero() && v.RevisionNumber == 9}
				}
				return networkBodyFilterAccessFor(a.FindIdentity, a.Resources, func(ctx context.Context) iter.Seq2[*subnetpools.SubnetPool, error] { return a.List(ctx) }, a.RawClient(), view)
			},
		})
	}
	return fixtures
}

func networkSubnetBodyPath(f networkSubnetBodyFixture) string {
	return networkExtensionPrefix + f.plural
}

func networkSubnetBodyRow(f networkSubnetBodyFixture, id, name, raw string) string {
	body := fmt.Sprintf(`"id":%q,"name":%q,"project_id":"project","revision_number":9,"created_at":"2025-01-02T03:04:05Z","updated_at":"2025-01-03T03:04:05Z"`, id, name)
	if f.field == "prefixes" {
		body += `,"default_prefixlen":"24","min_prefixlen":16,"max_prefixlen":"28","ip_version":4`
	} else {
		body += `,"status":"ACTIVE"`
	}
	if raw != "missing" {
		body += fmt.Sprintf(`,%q:%s`, f.field, raw)
	}
	return "{" + body + "}"
}

func networkSubnetBodyPage(f networkSubnetBodyFixture, rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,%q:[{"rel":"next","href":%q}]`, f.plural+"_links", next)
	}
	return fmt.Sprintf(`{%q:[%s]%s}`, f.plural, rows, links)
}

func networkSubnetBodyArrays(f networkSubnetBodyFixture) (target, reverse, equivalent string) {
	if f.field == "prefixes" {
		return `["10.0.0.1/24","2001:0db8:0:0::/64"]`, `["2001:0db8:0:0::/64","10.0.0.1/24"]`, `["10.0.0.0/24","2001:db8::/64"]`
	}
	return `["subnet-alpha","subnet-beta"]`, `["subnet-beta","subnet-alpha"]`, `["SUBNET-alpha","subnet-beta"]`
}

func TestNetworkSubnetBodyFiltersNativeValueSemantics(t *testing.T) {
	for _, f := range networkSubnetBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			target, reverse, equivalent := networkSubnetBodyArrays(f)
			var targetStrings []string
			if err := json.Unmarshal([]byte(target), &targetStrings); err != nil {
				t.Fatal(err)
			}
			rows := networkSubnetBodyRow(f, "missing", "same", "missing") + "," + networkSubnetBodyRow(f, "null", "same", "null") + "," + networkSubnetBodyRow(f, "empty", "same", "[]") + "," + networkSubnetBodyRow(f, "ordered", "same", target) + "," + networkSubnetBodyRow(f, "reversed", "same", reverse) + "," + networkSubnetBodyRow(f, "literal", "same", equivalent)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+networkSubnetBodyPath(f), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("local array became a wire query", r.URL)
				}
				testcloud.JSON(w, 200, networkSubnetBodyPage(f, rows, ""))
			})
			access := f.open(t, networkExtensionClient(cloud))
			cases := []struct {
				name  string
				value any
				ids   []string
			}{
				{"nil", nil, []string{"missing", "null"}},
				{"raw-null", json.RawMessage(`null`), []string{"missing", "null"}},
				{"empty-array", []string{}, []string{"empty"}},
				{"ordered", json.RawMessage(target), []string{"ordered"}},
				{"reversed", json.RawMessage(reverse), []string{"reversed"}},
				{"literal-not-normalized", json.RawMessage(equivalent), []string{"literal"}},
				{"shorter-array", targetStrings[:1], nil},
				{"longer-array", append(append([]string(nil), targetStrings...), "extra"), nil},
				{"wrong-scalar", "subnet-alpha", nil},
				{"wrong-element-type", []any{false, "subnet-beta"}, nil},
				{"object-subset-against-array", map[string]any{}, nil},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					values, err := access.all(context.Background(), resource.WithBodyFilter(f.field, tc.value))
					networkBodyFilterWantIDs(t, values, err, tc.ids...)
				})
			}
			if calls.Load() != int32(len(cases)) {
				t.Fatal("fresh array filters did not fetch independently", calls.Load())
			}
		})
	}
}

func TestNetworkSubnetBodyFiltersAliasesSnapshotsAndConcurrentReuse(t *testing.T) {
	for _, f := range networkSubnetBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			target, _, _ := networkSubnetBodyArrays(f)
			var source []string
			if err := json.Unmarshal([]byte(target), &source); err != nil {
				t.Fatal(err)
			}
			key := f.field
			if key == "subnets" {
				key = "subnet_ids"
			}
			filters := map[string]any{key: source}
			raw := json.RawMessage(target)
			bulk, single, rawOption := resource.WithBodyFilters(filters), resource.WithBodyFilter(key, source), resource.WithBodyFilter(f.field, raw)
			filters[key], source[0], raw[1] = nil, "changed", ' '
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+networkSubnetBodyPath(f), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("snapshotted local filter leaked into query", r.URL)
				}
				testcloud.JSON(w, 200, networkSubnetBodyPage(f, networkSubnetBodyRow(f, "target", "same", target)+","+networkSubnetBodyRow(f, "empty", "same", "[]"), ""))
			})
			access := f.open(t, networkExtensionClient(cloud))
			for _, opts := range [][]resource.ListOption{
				{bulk}, {single}, {rawOption},
				{resource.WithBodyFilter(f.field, nil), single},
				{resource.WithBodyFilter(key, nil), rawOption},
				{resource.WithBodyFilters(map[string]any{f.field: nil}), bulk},
			} {
				values, err := access.all(context.Background(), opts...)
				networkBodyFilterWantIDs(t, values, err, "target")
			}
			values, err := access.all(context.Background(), single, resource.WithBodyFilter(f.field, nil))
			networkBodyFilterWantIDs(t, values, err)
			if f.field == "subnets" {
				before := calls.Load()
				for _, value := range []any{nil, json.RawMessage(target)} {
					values, err = access.all(context.Background(), resource.WithBodyFilters(map[string]any{"subnets": value, "subnet_ids": value}))
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
						t.Fatal("bulk canonical/alias collision was not rejected before HTTP", values, err, calls.Load())
					}
				}
			}
			opts := []resource.ListOption{single}
			seq := access.list(context.Background(), opts...)
			opts[0] = resource.WithBodyFilter(f.field, nil)
			for range 2 {
				var ids []string
				for value, err := range seq {
					if err != nil {
						t.Fatal(err)
					}
					ids = append(ids, value.id)
				}
				if strings.Join(ids, ",") != "target" {
					t.Fatal("caller option slice changed retained iterator", ids)
				}
			}
			for _, clear := range []resource.ListOption{resource.WithBodyFilters(nil), resource.WithBodyFilters(map[string]any{})} {
				values, err = access.all(context.Background(), single, clear)
				networkBodyFilterWantIDs(t, values, err, "target", "empty")
			}
			var group sync.WaitGroup
			for range 6 {
				group.Add(1)
				go func() {
					defer group.Done()
					values, err := access.all(context.Background(), bulk)
					if err != nil || strings.Join(networkBodyFilterIDs(values), ",") != "target" {
						t.Error(networkBodyFilterIDs(values), err)
					}
				}()
			}
			group.Wait()
			if calls.Load() != 17 {
				t.Fatal("reused Body options did not make fresh requests", calls.Load())
			}
		})
	}
}

func TestNetworkSubnetBodyFiltersLocalWireAndNativeSurfaces(t *testing.T) {
	for _, f := range networkSubnetBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			target, reverse, _ := networkSubnetBodyArrays(f)
			rows := networkSubnetBodyRow(f, "right", "Target", target) + "," + networkSubnetBodyRow(f, "wrong-name", "Other", target) + "," + networkSubnetBodyRow(f, "wrong-body", "Target", reverse)
			if f.field == "subnets" {
				rows += "," + strings.Replace(networkSubnetBodyRow(f, "wrong-status", "Target", target), `"status":"ACTIVE"`, `"status":"DOWN"`, 1)
			}
			var phase atomic.Value
			phase.Store("local")
			var lists, gets atomic.Int32
			path := networkSubnetBodyPath(f)
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if phase.Load() == "local" {
					status := "vendor-status"
					if f.field == "subnets" {
						status = "ACTIVE"
					}
					q := r.URL.Query()
					if q.Get("subnet_ids") != "wire-alias" || q.Get("subnets") != "wire-subnets" || q.Get("prefixes") != "wire-prefixes" || q.Get("vendor") != "a&b" || q.Get("status") != status || q.Get("name") != "Target" || len(q) != 6 {
						t.Error("local Body filters changed independent wire fields", r.URL)
					}
				} else if r.URL.RawQuery != "" {
					t.Error("typed native List acquired ambient filters", r.URL)
				}
				testcloud.JSON(w, 200, networkSubnetBodyPage(f, rows, ""))
			})
			cloud.Mux.HandleFunc("GET "+path+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.URL.Query().Get(f.field) != "wire-only" || len(r.URL.Query()) != 1 {
					t.Error("FindIdentity raw query changed", r.URL)
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:%s}`, f.singular, networkSubnetBodyRow(f, "lookup", "Different", reverse)))
			})
			access := f.open(t, networkExtensionClient(cloud))
			opts := []resource.ListOption{resource.WithBodyFilter(f.field, json.RawMessage(target)), resource.WithName("Target"), resource.WithQuery("subnet_ids", "wire-alias"), resource.WithQuery("subnets", "wire-subnets"), resource.WithQuery("prefixes", "wire-prefixes"), resource.WithQuery("vendor", "a&b"), resource.WithQuery("status", "vendor-status")}
			if f.field == "subnets" {
				opts = append(opts, resource.WithStatus("ACTIVE"))
			}
			values, err := access.all(context.Background(), opts...)
			networkBodyFilterWantIDs(t, values, err, "right")
			phase.Store("native")
			var ids []string
			for value, err := range access.native(context.Background()) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, value.id)
			}
			want := "right,wrong-name,wrong-body"
			if f.field == "subnets" {
				want += ",wrong-status"
			}
			if strings.Join(ids, ",") != want {
				t.Fatal("typed native List changed", ids)
			}
			value, err := access.find(context.Background(), "lookup", resource.WithIdentityFindQuery(f.field, "wire-only"))
			if err != nil || value == nil || value.id != "lookup" || lists.Load() != 2 || gets.Load() != 1 {
				t.Fatal("ordinary filtering changed independent identity lookup", value, err, lists.Load(), gets.Load())
			}
		})
	}
}

func TestNetworkSubnetBodyFiltersRawControlsAndConsumerBreak(t *testing.T) {
	for _, f := range networkSubnetBodyFixtures() {
		for _, mode := range []string{"raw-cap", "status-cap", "single-page", "break", "all"} {
			if mode == "status-cap" && f.field != "subnets" {
				continue
			}
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				target, reverse, _ := networkSubnetBodyArrays(f)
				path := networkSubnetBodyPath(f)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					q := r.URL.Query()
					if q.Get("name") != "Target" || q.Has(f.field) || q.Has("subnet_ids") || q.Has("limit") || q.Has("max_items") || q.Has("paginated") || q.Get("vendor") != "kept" || f.field == "subnets" && q.Get("status") != "ACTIVE" {
						t.Error("local filters or controls changed wire policy", r.URL)
					}
					if calls.Add(1) > 1 {
						w.WriteHeader(503)
						return
					}
					rows := networkSubnetBodyRow(f, "wrong-body", "Target", reverse) + "," + networkSubnetBodyRow(f, "wrong-name", "Other", target)
					if f.field == "subnets" {
						rows += "," + strings.Replace(networkSubnetBodyRow(f, "wrong-status", "Target", target), `"status":"ACTIVE"`, `"status":"DOWN"`, 1)
					}
					rows += "," + networkSubnetBodyRow(f, "right", "Target", target)
					q.Set("marker", "second")
					next := cloud.Server.URL + path + "?" + q.Encode()
					if mode != "all" {
						next = "https://foreign.invalid/should-not-be-read"
					}
					testcloud.JSON(w, 200, networkSubnetBodyPage(f, rows, next))
				})
				access := f.open(t, networkExtensionClient(cloud))
				opts := []resource.ListOption{resource.WithBodyFilter(f.field, json.RawMessage(target)), resource.WithName("Target"), resource.WithQuery("vendor", "kept")}
				if f.field == "subnets" {
					opts = append(opts, resource.WithStatus("ACTIVE"))
				}
				switch mode {
				case "raw-cap":
					opts = append(opts, resource.WithMaxItems(2))
				case "status-cap":
					opts = append(opts, resource.WithMaxItems(3))
				case "single-page":
					opts = append(opts, resource.WithPaginated(false))
				case "break":
					var seen int
					for value, err := range access.list(context.Background(), opts...) {
						if err != nil || value == nil || value.id != "right" {
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
				values, err := access.all(context.Background(), opts...)
				if mode == "all" {
					if values != nil || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 2 {
						t.Fatal("All hid partial rows or a late server failure", values, err, calls.Load())
					}
				} else {
					if mode == "single-page" {
						networkBodyFilterWantIDs(t, values, err, "right")
					} else {
						networkBodyFilterWantIDs(t, values, err)
					}
					if calls.Load() != 1 {
						t.Fatal("local stopping parsed or followed a foreign continuation", calls.Load())
					}
				}
			})
		}
	}
}

func TestNetworkSubnetBodyFiltersNativePageErrorsAndContinuation(t *testing.T) {
	for _, f := range networkSubnetBodyFixtures() {
		modes := []string{"whole-array", "whole-timestamp", "late-http", "late-decode", "cycle", "cancel", "foreign", "wrong-link-shape", "native-200", "native-204", "native-300", "old-time", "new-time"}
		if f.field == "prefixes" {
			for _, key := range []string{"default_prefixlen", "min_prefixlen", "max_prefixlen"} {
				for _, bad := range []string{"missing", "null", "bool"} {
					modes = append(modes, key+"/"+bad)
				}
			}
		}
		for _, mode := range modes {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				target, _, _ := networkSubnetBodyArrays(f)
				path := networkSubnetBodyPath(f)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					page := calls.Add(1)
					q := r.URL.Query()
					q.Set("marker", "second")
					next := cloud.Server.URL + path + "?" + q.Encode()
					rows := networkSubnetBodyRow(f, "candidate", "Target", target)
					status := 200
					switch {
					case strings.HasPrefix(mode, "whole-") || strings.Contains(mode, "/"):
						bad := networkSubnetBodyRow(f, "bad", "Other", `[false]`)
						if mode == "whole-timestamp" {
							bad = strings.Replace(networkSubnetBodyRow(f, "bad", "Other", "[]"), "2025-01-02T03:04:05Z", "invalid-time", 1)
						} else if parts := strings.Split(mode, "/"); len(parts) == 2 {
							bad = networkPoolsTrunksChangeRow(t, networkSubnetBodyRow(f, "bad", "Other", "[]"), parts[0], parts[1])
						}
						rows += "," + bad
					case mode == "wrong-link-shape":
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[%s],"links":{"next":"https://foreign.invalid/ignored"}}`, f.plural, rows))
						return
					case mode == "foreign":
						next = "https://foreign.invalid/rejected"
					case strings.HasPrefix(mode, "native-"):
						next = ""
						if mode == "native-204" {
							w.WriteHeader(204)
							return
						}
						if mode == "native-300" {
							status = 300
						}
					case mode == "old-time" || mode == "new-time":
						next = ""
						if mode == "old-time" {
							rows = strings.ReplaceAll(rows, "03:04:05Z", "03:04:05")
						} else if f.field == "prefixes" {
							rows = strings.ReplaceAll(strings.ReplaceAll(rows, `"default_prefixlen":"24"`, `"default_prefixlen":24`), `"max_prefixlen":"28"`, `"max_prefixlen":28`)
						}
					case page > 1:
						switch mode {
						case "late-http":
							w.WriteHeader(503)
							return
						case "late-decode":
							rows, next = networkSubnetBodyRow(f, "bad", "Other", `[false]`), ""
						case "cycle":
							rows = networkSubnetBodyRow(f, "other", "Other", "[]")
						case "cancel":
							cancel()
							rows, next = networkSubnetBodyRow(f, "other", "Other", "[]"), ""
						}
					}
					testcloud.JSON(w, status, networkSubnetBodyPage(f, rows, next))
				})
				access := f.open(t, networkExtensionClient(cloud))
				opts := []resource.ListOption{resource.WithBodyFilter(f.field, json.RawMessage(target)), resource.WithName("Target")}
				whole := strings.HasPrefix(mode, "whole-") || strings.Contains(mode, "/")
				if whole {
					opts = append(opts, resource.WithMaxItems(1))
				}
				values, err := access.all(ctx, opts...)
				success := mode == "wrong-link-shape" || strings.HasPrefix(mode, "native-") || mode == "old-time" || mode == "new-time"
				if success {
					if mode == "native-204" {
						networkBodyFilterWantIDs(t, values, err)
						if values == nil {
							t.Fatal("successful empty All became nil")
						}
					} else {
						networkBodyFilterWantIDs(t, values, err, "candidate")
						if !values[0].nested {
							t.Fatal("native timestamps or prefix-length extraction changed", values[0])
						}
					}
					if calls.Load() != 1 {
						t.Fatal("native final page or wrong-shaped link followed", calls.Load())
					}
					return
				}
				wantCalls := int32(2)
				if whole || mode == "foreign" {
					wantCalls = 1
				}
				if values != nil || err == nil || calls.Load() != wantCalls || mode == "late-http" && !gophercloud.ResponseCodeIs(err, 503) || mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal("Body filter or raw cap hid native/continuation failure", values, err, calls.Load())
				}
			})
		}
	}
}

func TestNetworkSubnetBodyFiltersLazyPreflightAndExclusions(t *testing.T) {
	for _, f := range networkSubnetBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			access := f.open(t, networkExtensionClient(cloud))
			for _, option := range []resource.ListOption{
				resource.WithBodyFilter("unknown", nil),
				resource.WithBodyFilter(strings.ToUpper(f.field), nil),
				resource.WithBodyFilter(f.field, json.RawMessage(`{`)),
				resource.WithBodyFilter(f.field, make(chan int)),
				resource.WithBodyFilters(map[string]any{f.field: nil, "unknown": []string{}}),
			} {
				seq := access.list(context.Background(), option)
				if calls.Load() != 0 {
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
					t.Fatal("invalid filter did not yield one terminal error", seen)
				}
			}
			if f.field == "prefixes" {
				for _, key := range []string{"subnet_ids", "subnets"} {
					values, err := access.all(context.Background(), resource.WithBodyFilter(key, nil))
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal("Network-only alias enabled on SubnetPool", values, err)
					}
				}
				if _, err := access.all(context.Background(), resource.WithBodyFilter(f.field, nil), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal("SubnetPool gained typed status", err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if values, err := access.all(ctx, resource.WithBodyFilters(nil)); values != nil || !errors.Is(err, context.Canceled) {
				t.Fatal(values, err)
			}
			if calls.Load() != 0 {
				t.Fatal("preflight failure performed HTTP", calls.Load())
			}
		})
	}
	// The pinned Trunk query maps sub_ports: it remains a raw wire filter and
	// does not gain a local Body capability simply because the model has it.
	t.Run("trunk-query-not-body", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		f := networkExtensionFixture{singular: "trunk", plural: "trunks"}
		cloud.Mux.HandleFunc("GET "+networkExtensionPrefix+"trunks", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Query().Get("sub_ports") != "wire-only" || len(r.URL.Query()) != 1 {
				t.Error("Trunk sub_ports wire query changed", r.URL)
			}
			testcloud.JSON(w, 200, networkPoolsTrunksPage(f, networkPoolsTrunksRow(f, "trunk", "Target"), ""))
		})
		a := trunks.New(networkExtensionClient(cloud))
		for _, option := range []resource.ListOption{resource.WithBodyFilter("sub_ports", []any{}), resource.WithBodyFilters(map[string]any{"sub_ports": []any{}})} {
			values, err := a.Resources.All(context.Background(), option)
			if values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal("Trunk local filter was unexpectedly enabled", values, err, calls.Load())
			}
		}
		values, err := a.Resources.All(context.Background(), resource.WithBodyFilters(nil), resource.WithQuery("sub_ports", "wire-only"))
		if err != nil || len(values) != 1 || values[0].ID != "trunk" || calls.Load() != 1 {
			t.Fatal(values, err, calls.Load())
		}
	})
}
