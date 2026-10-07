package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	qospolicies "github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/ports"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Pinned Resource.list separates QoSPolicy.rules and AddressGroup.addresses
// from wire query fields. These SDK-owned options compare original row fields
// after native whole-page decoding; returned QoS map numbers remain float64.
// Raw WithQuery remains an independent wire
// extension; typed API.List and FindIdentity do not acquire ambient filters.
type networkBodyFilterAccess struct {
	networkExtensionAccess
	list   func(context.Context, ...resource.ListOption) iter.Seq2[*networkExtensionView, error]
	native func(context.Context) iter.Seq2[*networkExtensionView, error]
}

type networkBodyFilterFixture struct {
	meta  networkExtensionFixture
	field string
	open  func(*testing.T, *gophercloud.ServiceClient) networkBodyFilterAccess
}

func networkBodyFilterSequence[T any](seq iter.Seq2[*T, error], view func(*T) *networkExtensionView) iter.Seq2[*networkExtensionView, error] {
	return func(yield func(*networkExtensionView, error) bool) {
		for value, err := range seq {
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(view(value), nil) {
				return
			}
		}
	}
}

func networkBodyFilterAccessFor[T any](find func(context.Context, string, ...resource.IdentityFindOption) (*T, error), collection *resource.Collection[T], native func(context.Context) iter.Seq2[*T, error], raw *gophercloud.ServiceClient, view func(*T) *networkExtensionView) networkBodyFilterAccess {
	access := networkExtensionAccessFor(find, collection, raw, view)
	access.all = func(ctx context.Context, opts ...resource.ListOption) ([]*networkExtensionView, error) {
		values, err := collection.All(ctx, opts...)
		if values == nil {
			return nil, err
		}
		result := make([]*networkExtensionView, 0, len(values))
		for _, value := range values {
			result = append(result, view(value))
		}
		return result, err
	}
	return networkBodyFilterAccess{
		networkExtensionAccess: access,
		list: func(ctx context.Context, opts ...resource.ListOption) iter.Seq2[*networkExtensionView, error] {
			return networkBodyFilterSequence(collection.List(ctx, opts...), view)
		},
		native: func(ctx context.Context) iter.Seq2[*networkExtensionView, error] {
			return networkBodyFilterSequence(native(ctx), view)
		},
	}
}

func networkBodyFilterConnection(t *testing.T, c *gophercloud.ServiceClient) *network.Service {
	t.Helper()
	conn, err := sdk.FromProvider(c.ProviderClient, sdk.WithEndpoint(sdk.Network, c.Endpoint))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw := service.RawClient()
	raw.ResourceBase, raw.Microversion, raw.MoreHeaders = c.ResourceBase, c.Microversion, c.MoreHeaders
	again, err := conn.Network(context.Background())
	if err != nil || again != service || raw.ProviderClient != c.ProviderClient || service.API.QoSPolicies.RawClient() != raw || service.API.SecurityAddressGroups.RawClient() != raw {
		t.Fatal("body-filter APIs did not share the cached network client", err)
	}
	return service
}

func networkBodyFilterFixtures() []networkBodyFilterFixture {
	var fixtures []networkBodyFilterFixture
	for _, connected := range []bool{false, true} {
		suffix := "leaf"
		if connected {
			suffix = "connection"
		}
		fixtures = append(fixtures, networkBodyFilterFixture{
			meta: networkExtensionFixture{name: "qos-" + suffix, singular: "policy", plural: "policies"}, field: "rules",
			open: func(t *testing.T, c *gophercloud.ServiceClient) networkBodyFilterAccess {
				a := qospolicies.New(c)
				if connected {
					a = networkBodyFilterConnection(t, c).API.QoSPolicies
				}
				view := func(v *qospolicies.Policy) *networkExtensionView {
					return &networkExtensionView{id: v.ID, name: v.Name, project: v.ProjectID}
				}
				return networkBodyFilterAccessFor(a.FindIdentity, a.Resources, func(ctx context.Context) iter.Seq2[*qospolicies.Policy, error] { return a.List(ctx) }, a.RawClient(), view)
			},
		}, networkBodyFilterFixture{
			meta: networkExtensionFixture{name: "addresses-" + suffix, singular: "address_group", plural: "address_groups"}, field: "addresses",
			open: func(t *testing.T, c *gophercloud.ServiceClient) networkBodyFilterAccess {
				a := addressgroups.New(c)
				if connected {
					a = networkBodyFilterConnection(t, c).API.SecurityAddressGroups
				}
				view := func(v *addressgroups.AddressGroup) *networkExtensionView {
					return &networkExtensionView{id: v.ID, name: v.Name, project: v.ProjectID}
				}
				return networkBodyFilterAccessFor(a.FindIdentity, a.Resources, func(ctx context.Context) iter.Seq2[*addressgroups.AddressGroup, error] { return a.List(ctx) }, a.RawClient(), view)
			},
		})
	}
	return fixtures
}

func networkBodyFilterArrays(f networkBodyFilterFixture) (target, reverse, extra string) {
	if f.field == "rules" {
		first := `{"type":"bandwidth_limit","max_kbps":1200,"nested":{"enabled":true}}`
		second := `{"type":"dscp_marking","dscp_mark":16}`
		return "[" + first + "," + second + "]", "[" + second + "," + first + "]", "[" + strings.Replace(first, `"max_kbps":1200`, `"max_kbps":1200,"extra":true`, 1) + "," + second + "]"
	}
	return `["192.0.2.1","198.51.100.0/24"]`, `["198.51.100.0/24","192.0.2.1"]`, `["192.0.2.1","198.51.100.0/24","2001:db8::/64"]`
}

func networkBodyFilterRow(f networkBodyFilterFixture, id, name, raw string) string {
	body := fmt.Sprintf(`"id":%q,"name":%q,"project_id":"project"`, id, name)
	if raw != "missing" {
		body += fmt.Sprintf(`,%q:%s`, f.field, raw)
	}
	return "{" + body + "}"
}

func networkBodyFilterIDs(values []*networkExtensionView) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.id)
	}
	return ids
}

func networkBodyFilterWantIDs(t *testing.T, values []*networkExtensionView, err error, want ...string) {
	t.Helper()
	if err != nil || strings.Join(networkBodyFilterIDs(values), ",") != strings.Join(want, ",") {
		t.Fatal(networkBodyFilterIDs(values), err, "want", want)
	}
}

func TestNetworkBodyFiltersNativeValueSemantics(t *testing.T) {
	for _, f := range networkBodyFilterFixtures() {
		t.Run(f.meta.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			target, reverse, extra := networkBodyFilterArrays(f)
			rows := []string{networkBodyFilterRow(f, "missing", "same", "missing"), networkBodyFilterRow(f, "null", "same", "null"), networkBodyFilterRow(f, "empty", "same", "[]"), networkBodyFilterRow(f, "ordered", "same", target), networkBodyFilterRow(f, "reversed", "same", reverse), networkBodyFilterRow(f, "extra", "same", extra)}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+networkQoSAddressesPath(f.meta), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("local filter became a query", r.URL)
				}
				testcloud.JSON(w, 200, networkQoSAddressesPage(f.meta, strings.Join(rows, ","), ""))
			})
			access := f.open(t, networkQoSAddressesClient(cloud))
			cases := []struct {
				name  string
				value any
				ids   []string
			}{
				{"nil", nil, []string{"missing", "null"}},
				{"json-null", json.RawMessage(`null`), []string{"missing", "null"}},
				{"empty-array", []string{}, []string{"empty"}},
				{"exact-order", json.RawMessage(target), []string{"ordered"}},
				{"reverse-order", json.RawMessage(reverse), []string{"reversed"}},
				{"wrong-scalar", "same", nil},
				{"wrong-bool", false, nil},
				{"object-subset-against-array", map[string]any{}, nil},
			}
			if f.field == "rules" {
				cases = append(cases, struct {
					name  string
					value any
					ids   []string
				}{"inner-object-is-exact", json.RawMessage(`[{"type":"bandwidth_limit"},{"type":"dscp_marking"}]`), nil})
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					values, err := access.all(context.Background(), resource.WithBodyFilter(f.field, tc.value))
					networkBodyFilterWantIDs(t, values, err, tc.ids...)
				})
			}
			if calls.Load() != int32(len(cases)) {
				t.Fatal("fresh local-filter iterations did not fetch independently", calls.Load())
			}
		})
	}
}

func TestNetworkBodyFiltersLocalWireAndNativeSurfaces(t *testing.T) {
	for _, f := range networkBodyFilterFixtures() {
		t.Run(f.meta.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			target, reverse, _ := networkBodyFilterArrays(f)
			rows := networkBodyFilterRow(f, "right", "Target", target) + "," + networkBodyFilterRow(f, "wrong-name", "Other", target) + "," + networkBodyFilterRow(f, "wrong-body", "Target", reverse)
			var phase atomic.Value
			phase.Store("local")
			var lists, gets atomic.Int32
			path := networkQoSAddressesPath(f.meta)
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if phase.Load() == "local" {
					if r.URL.Query().Get(f.field) != "wire-value" || r.URL.Query().Get("name") != "Target" || r.URL.Query().Get("status") != "vendor-status" {
						t.Error("wire query and local name policy changed", r.URL)
					}
				} else if r.URL.RawQuery != "" {
					t.Error("native List gained an ambient local filter or query", r.URL)
				}
				testcloud.JSON(w, 200, networkQoSAddressesPage(f.meta, rows, ""))
			})
			cloud.Mux.HandleFunc("GET "+path+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.URL.Query().Get(f.field) != "wire-value" || len(r.URL.Query()) != 1 {
					t.Error("FindIdentity lost its independent raw query", r.URL)
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:%s}`, f.meta.singular, networkBodyFilterRow(f, "lookup", "Different", reverse)))
			})
			access := f.open(t, networkQoSAddressesClient(cloud))
			values, err := access.all(context.Background(), resource.WithBodyFilter(f.field, json.RawMessage(target)), resource.WithQuery(f.field, "wire-value"), resource.WithName("Target"), resource.WithQuery("status", "vendor-status"))
			networkBodyFilterWantIDs(t, values, err, "right")
			phase.Store("native")
			var nativeIDs []string
			for value, err := range access.native(context.Background()) {
				if err != nil {
					t.Fatal(err)
				}
				nativeIDs = append(nativeIDs, value.id)
			}
			if strings.Join(nativeIDs, ",") != "right,wrong-name,wrong-body" {
				t.Fatal("native List acquired local filtering", nativeIDs)
			}
			value, err := access.find(context.Background(), "lookup", resource.WithIdentityFindQuery(f.field, "wire-value"))
			if err != nil || value == nil || value.id != "lookup" || lists.Load() != 2 || gets.Load() != 1 {
				t.Fatal("identity lookup acquired ambient local filtering", value, err, lists.Load(), gets.Load())
			}
		})
	}
}

func TestNetworkBodyFiltersSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	for _, f := range networkBodyFilterFixtures() {
		t.Run(f.meta.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var source any
			var target string
			if f.field == "rules" {
				source = []map[string]any{{"kind": "one", "nested": map[string]any{"enabled": true}}}
				target = `[{"kind":"one","nested":{"enabled":true}}]`
			} else {
				source = []string{"192.0.2.1", "198.51.100.0/24"}
				target = `["192.0.2.1","198.51.100.0/24"]`
			}
			filters := map[string]any{f.field: source}
			bulk, single := resource.WithBodyFilters(filters), resource.WithBodyFilter(f.field, source)
			filters[f.field] = nil
			if f.field == "rules" {
				values := source.([]map[string]any)
				values[0]["kind"] = "changed"
				values[0]["nested"].(map[string]any)["enabled"] = false
			} else {
				source.([]string)[0] = "changed"
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+networkQoSAddressesPath(f.meta), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("body snapshot leaked into wire query", r.URL)
				}
				testcloud.JSON(w, 200, networkQoSAddressesPage(f.meta, networkBodyFilterRow(f, "target", "same", target)+","+networkBodyFilterRow(f, "empty", "same", "[]"), ""))
			})
			access := f.open(t, networkQoSAddressesClient(cloud))
			for _, opts := range [][]resource.ListOption{
				{bulk}, {single},
				{resource.WithBodyFilter(f.field, nil), bulk},
				{resource.WithBodyFilters(map[string]any{f.field: nil}), single},
				{resource.WithBodyFilter(f.field, nil), single},
			} {
				values, err := access.all(context.Background(), opts...)
				networkBodyFilterWantIDs(t, values, err, "target")
			}
			values, err := access.all(context.Background(), single, resource.WithBodyFilter(f.field, nil))
			networkBodyFilterWantIDs(t, values, err)
			options := []resource.ListOption{single}
			seq := access.list(context.Background(), options...)
			options[0] = resource.WithBodyFilter(f.field, nil)
			for range 2 {
				var ids []string
				for value, err := range seq {
					if err != nil {
						t.Fatal(err)
					}
					ids = append(ids, value.id)
				}
				if strings.Join(ids, ",") != "target" {
					t.Fatal("caller option slice changed a retained iterator", ids)
				}
			}
			for _, clear := range []resource.ListOption{resource.WithBodyFilters(nil), resource.WithBodyFilters(map[string]any{})} {
				values, err := access.all(context.Background(), resource.WithBodyFilter(f.field, nil), clear)
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
			if calls.Load() != 16 {
				t.Fatal("reused body options did not make independent requests", calls.Load())
			}
		})
	}
}

func TestNetworkBodyFiltersLazyPreflightAndUnsupportedBindings(t *testing.T) {
	for _, f := range networkBodyFilterFixtures() {
		t.Run(f.meta.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			access := f.open(t, networkQoSAddressesClient(cloud))
			cases := []struct {
				name   string
				option resource.ListOption
			}{
				{"unknown", resource.WithBodyFilter("unknown", nil)},
				{"case-alias", resource.WithBodyFilter(strings.ToUpper(f.field), nil)},
				{"empty-key", resource.WithBodyFilter("", nil)},
				{"malformed-json", resource.WithBodyFilter(f.field, json.RawMessage(`{`))},
				{"non-json", resource.WithBodyFilter(f.field, make(chan int))},
				{"nan", resource.WithBodyFilter(f.field, math.NaN())},
				{"bulk-unknown", resource.WithBodyFilters(map[string]any{f.field: nil, "unknown": []string{}})},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					seq := access.list(context.Background(), tc.option)
					if calls.Load() != 0 {
						t.Fatal("constructing iterator performed HTTP")
					}
					var errorsSeen int
					for value, err := range seq {
						if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal(value, err)
						}
						switch tc.name {
						case "malformed-json":
							var cause *json.SyntaxError
							if !errors.As(err, &cause) {
								t.Fatal("JSON syntax cause lost", err)
							}
						case "non-json":
							var cause *json.UnsupportedTypeError
							if !errors.As(err, &cause) {
								t.Fatal("unsupported JSON type cause lost", err)
							}
						case "nan":
							var cause *json.UnsupportedValueError
							if !errors.As(err, &cause) {
								t.Fatal("unsupported JSON value cause lost", err)
							}
						}
						errorsSeen++
					}
					if errorsSeen != 1 {
						t.Fatal("invalid option did not yield one terminal error", errorsSeen)
					}
				})
			}
			if _, err := access.all(context.Background(), resource.WithBodyFilter(f.field, nil), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal("binding gained a typed status filter", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := access.all(ctx, resource.WithBodyFilters(nil)); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			unsupported := ports.New(networkQoSAddressesClient(cloud)).Resources
			for _, option := range []resource.ListOption{resource.WithBodyFilters(nil), resource.WithBodyFilters(map[string]any{}), resource.WithBodyFilter(f.field, nil)} {
				if _, err := unsupported.All(context.Background(), option); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal("explicit empty body option enabled an unaudited binding", err)
				}
			}
			if calls.Load() != 0 {
				t.Fatal("invalid/unsupported/canceled local filter performed HTTP", calls.Load())
			}
		})
	}
}

func TestNetworkBodyFiltersRawControlsNameAndConsumerBreak(t *testing.T) {
	for _, f := range networkBodyFilterFixtures() {
		for _, mode := range []string{"raw-cap", "single-page", "break", "all"} {
			t.Run(f.meta.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				target, reverse, _ := networkBodyFilterArrays(f)
				path := networkQoSAddressesPath(f.meta)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Get("name") != "Target" || r.URL.Query().Has(f.field) || r.URL.Query().Has("limit") || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") {
						t.Error("local filter/control wire policy changed", r.URL)
					}
					if calls.Add(1) > 1 {
						w.WriteHeader(503)
						return
					}
					rows := networkBodyFilterRow(f, "wrong-body", "Target", reverse) + "," + networkBodyFilterRow(f, "wrong-name", "Other", target) + "," + networkBodyFilterRow(f, "right", "Target", target)
					next := r.URL.Query()
					next.Set("marker", "second")
					testcloud.JSON(w, 200, networkQoSAddressesPage(f.meta, rows, cloud.Server.URL+path+"?"+next.Encode()))
				})
				access := f.open(t, networkQoSAddressesClient(cloud))
				opts := []resource.ListOption{resource.WithBodyFilter(f.field, json.RawMessage(target)), resource.WithName("Target")}
				if mode == "raw-cap" {
					opts = append(opts, resource.WithMaxItems(2))
				}
				if mode == "single-page" {
					opts = append(opts, resource.WithPaginated(false))
				}
				if mode == "break" {
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
				switch mode {
				case "all":
					if values != nil || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 2 {
						t.Fatal(values, err, calls.Load())
					}
				case "raw-cap":
					networkBodyFilterWantIDs(t, values, err)
					if calls.Load() != 1 {
						t.Fatal(calls.Load())
					}
				case "single-page":
					networkBodyFilterWantIDs(t, values, err, "right")
					if calls.Load() != 1 {
						t.Fatal(calls.Load())
					}
				}
			})
		}
	}
}

func TestNetworkBodyFiltersNativePageErrorsAndLateCancellation(t *testing.T) {
	for _, f := range networkBodyFilterFixtures() {
		for _, mode := range []string{"whole-page", "late-http", "late-decode", "cycle", "cancel"} {
			t.Run(f.meta.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				target, _, _ := networkBodyFilterArrays(f)
				path := networkQoSAddressesPath(f.meta)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					page := calls.Add(1)
					next := r.URL.Query()
					next.Set("marker", "second")
					link := cloud.Server.URL + path + "?" + next.Encode()
					rows := networkBodyFilterRow(f, "candidate", "Target", target)
					if mode == "whole-page" {
						// A row with a different name and body cannot hide native
						// decode failure, even beyond the requested raw row cap.
						rows += "," + networkQoSAddressesInvalidRow(f.meta)
					} else if page > 1 {
						switch mode {
						case "late-http":
							w.WriteHeader(503)
							return
						case "late-decode":
							rows = networkQoSAddressesInvalidRow(f.meta)
							link = ""
						case "cycle":
							rows = networkBodyFilterRow(f, "different", "Other", "[]")
						case "cancel":
							cancel()
							rows = networkBodyFilterRow(f, "different", "Other", "[]")
							link = ""
						}
					}
					testcloud.JSON(w, 200, networkQoSAddressesPage(f.meta, rows, link))
				})
				access := f.open(t, networkQoSAddressesClient(cloud))
				opts := []resource.ListOption{resource.WithBodyFilter(f.field, json.RawMessage(target)), resource.WithName("Target")}
				if mode == "whole-page" {
					opts = append(opts, resource.WithMaxItems(1))
				}
				values, err := access.all(ctx, opts...)
				if values != nil || err == nil || mode == "late-http" && !gophercloud.ResponseCodeIs(err, 503) || mode == "cancel" && !errors.Is(err, context.Canceled) || mode == "whole-page" && calls.Load() != 1 || mode != "whole-page" && calls.Load() != 2 {
					t.Fatal("body filtering or cap hid a native/late failure", values, err, calls.Load())
				}
			})
		}
	}
}

func TestNetworkBodyFiltersQoSNativeNumberProjectionAndStrictTypes(t *testing.T) {
	for _, f := range networkBodyFilterFixtures() {
		if f.field != "rules" {
			continue
		}
		t.Run(f.meta.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			rows := networkBodyFilterRow(f, "rounded", "same", `[{"quota":9007199254740993}]`) + "," + networkBodyFilterRow(f, "bool", "same", `[{"flag":false}]`) + "," + networkBodyFilterRow(f, "number", "same", `[{"flag":0}]`) + "," + networkBodyFilterRow(f, "decimal", "same", `[{"quota":1200}]`)
			cloud.Mux.HandleFunc("GET "+networkQoSAddressesPath(f.meta), func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, networkQoSAddressesPage(f.meta, rows, ""))
			})
			access := f.open(t, networkQoSAddressesClient(cloud))
			for _, tc := range []struct{ filter, id string }{
				{`[{"quota":9007199254740993}]`, "rounded"},
				{`[{"quota":9007199254740992}]`, ""},
				{`[{"flag":false}]`, "bool"},
				{`[{"flag":0}]`, "number"},
				{`[{"quota":1.2e3}]`, "decimal"},
			} {
				values, err := access.all(context.Background(), resource.WithBodyFilter("rules", json.RawMessage(tc.filter)))
				if tc.id == "" {
					networkBodyFilterWantIDs(t, values, err)
				} else {
					networkBodyFilterWantIDs(t, values, err, tc.id)
				}
			}
		})
	}
}
