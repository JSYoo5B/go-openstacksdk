package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	qospolicies "github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/subnets"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Pinned Subnet Body fields are independent of its query mapping. The raw
// ordinary-collection lane pairs original JSON with native typed extraction;
// filters retain presence, original timestamp text and exact decimal values.
// PrefixLength is absent from the native Subnet model and untyped in Python:
// it retains every JSON type without coercion. RevisionNumber's integer
// descriptor still follows native typed extraction. Typed List/Get/FindIdentity
// and other collections' typed Body projections retain their own contracts.
type subnetRawBodyFixture struct {
	name string
	open func(*testing.T, *gophercloud.ServiceClient) *subnets.API
}

func subnetRawBodyFixtures() []subnetRawBodyFixture {
	return []subnetRawBodyFixture{
		{"leaf", func(_ *testing.T, c *gophercloud.ServiceClient) *subnets.API { return subnets.New(c) }},
		{"connection", func(t *testing.T, c *gophercloud.ServiceClient) *subnets.API {
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
			if err != nil || again != s || s.API.Subnets.RawClient() != raw || raw.ProviderClient != c.ProviderClient {
				t.Fatal("raw Subnet collection did not share the cached source client", err)
			}
			return s.API.Subnets
		}},
	}
}

const subnetRawBodyPath = networkExtensionPrefix + "subnets"

func subnetRawBodyValues() map[string]json.RawMessage {
	return map[string]json.RawMessage{
		"allocation_pools": json.RawMessage(`[{"start":"10.0.0.2","end":"10.0.0.20","vendor":{"quota":9007199254740993}},null]`),
		"created_at":       json.RawMessage(`"2025-01-02T03:04:05+00:00"`),
		"dns_nameservers":  json.RawMessage(`["192.0.2.1",null]`),
		"host_routes":      json.RawMessage(`[null,{"destination":"192.0.2.0/24","nexthop":"10.0.0.1","vendor":9007199254740993}]`),
		"prefixlen":        json.RawMessage(`24`),
		"revision_number":  json.RawMessage(`9007199254740993`),
		"service_types":    json.RawMessage(`[null,"network:router_interface"]`),
		"tenant_id":        json.RawMessage(`"tenant-original"`),
		"updated_at":       json.RawMessage(`"2025-01-03T03:04:05+00:00"`),
	}
}

func subnetRawBodyRow(t *testing.T, id, name string, fields map[string]json.RawMessage) string {
	t.Helper()
	body := map[string]json.RawMessage{"id": json.RawMessage(fmt.Sprintf("%q", id)), "name": json.RawMessage(fmt.Sprintf("%q", name)), "project_id": json.RawMessage(`"project-independent"`)}
	for key, raw := range fields {
		body[key] = raw
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func subnetRawBodyPage(rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,"subnets_links":[{"rel":"next","href":%q}]`, next)
	}
	return `{"subnets":[` + rows + `]` + links + `}`
}

func subnetRawBodyWant(t *testing.T, values []*subnets.Subnet, err error, ids ...string) {
	t.Helper()
	got := make([]string, 0, len(values))
	for _, value := range values {
		got = append(got, value.ID)
	}
	if err != nil || strings.Join(got, ",") != strings.Join(ids, ",") {
		t.Fatal(got, err, "want", ids)
	}
}

func TestNetworkSubnetRawBodyFiltersNineFieldsAndOriginalValues(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		for _, field := range []string{"allocation_pools", "created_at", "dns_nameservers", "host_routes", "prefixlen", "revision_number", "service_types", "tenant_id", "updated_at"} {
			t.Run(f.name+"/"+field, func(t *testing.T) {
				cloud := testcloud.New(t)
				value := subnetRawBodyValues()[field]
				rows := subnetRawBodyRow(t, "missing", "same", nil) + "," + subnetRawBodyRow(t, "null", "same", map[string]json.RawMessage{field: json.RawMessage(`null`)}) + "," + subnetRawBodyRow(t, "target", "same", map[string]json.RawMessage{field: value})
				array := strings.HasPrefix(string(value), "[")
				if array {
					rows += "," + subnetRawBodyRow(t, "empty", "same", map[string]json.RawMessage{field: json.RawMessage(`[]`)})
				} else if field == "revision_number" || field == "prefixlen" {
					rows += "," + subnetRawBodyRow(t, "zero", "same", map[string]json.RawMessage{field: json.RawMessage(`0`)})
				} else if field == "tenant_id" {
					rows += "," + subnetRawBodyRow(t, "project-only-decoy", "same", map[string]json.RawMessage{"project_id": value})
				} else {
					other := json.RawMessage(`"2025-01-02T03:04:05Z"`)
					old := json.RawMessage(`"2025-01-02T03:04:05"`)
					if field == "updated_at" {
						other = json.RawMessage(`"2025-01-03T03:04:05Z"`)
						old = json.RawMessage(`"2025-01-03T03:04:05"`)
					}
					rows += "," + subnetRawBodyRow(t, "same-instant-different-text", "same", map[string]json.RawMessage{field: other})
					rows += "," + subnetRawBodyRow(t, "old-format", "same", map[string]json.RawMessage{field: old})
				}
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.RawQuery != "" {
						t.Error("raw Body field leaked to query", r.URL)
					}
					testcloud.JSON(w, 200, subnetRawBodyPage(rows, ""))
				})
				a := f.open(t, networkExtensionClient(cloud))
				values, err := a.Resources.All(context.Background(), resource.WithBodyFilter(field, value))
				subnetRawBodyWant(t, values, err, "target")
				values, err = a.Resources.All(context.Background(), resource.WithBodyFilter(field, nil))
				wantNull := []string{"missing", "null"}
				if field == "tenant_id" {
					wantNull = append(wantNull, "project-only-decoy")
				}
				subnetRawBodyWant(t, values, err, wantNull...)
				if array {
					values, err = a.Resources.All(context.Background(), resource.WithBodyFilter(field, []any{}))
					subnetRawBodyWant(t, values, err, "empty")
					var entries []json.RawMessage
					if err := json.Unmarshal(value, &entries); err != nil {
						t.Fatal(err)
					}
					entries[0], entries[1] = entries[1], entries[0]
					values, err = a.Resources.All(context.Background(), resource.WithBodyFilter(field, entries))
					subnetRawBodyWant(t, values, err)
					values, err = a.Resources.All(context.Background(), resource.WithBodyFilter(field, entries[:1]))
					subnetRawBodyWant(t, values, err)
				} else if field == "created_at" || field == "updated_at" {
					old := json.RawMessage(`"2025-01-02T03:04:05"`)
					if field == "updated_at" {
						old = json.RawMessage(`"2025-01-03T03:04:05"`)
					}
					values, err = a.Resources.All(context.Background(), resource.WithBodyFilter(field, old))
					subnetRawBodyWant(t, values, err, "old-format")
				}
				values, err = a.Resources.All(context.Background(), resource.WithBodyFilter(field, false))
				subnetRawBodyWant(t, values, err)
				if calls.Load() < 3 {
					t.Fatal("fresh raw filters did not issue independent requests", calls.Load())
				}
			})
		}
	}
}

func TestNetworkSubnetRawBodyFiltersExactNumbersAndUntypedPrefix(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var rows []string
			for _, item := range []struct{ id, raw string }{
				{"number", `24`}, {"string", `"24"`}, {"decimal", `24.0`}, {"exponent", `2.4e1`},
				{"negative-number", `-24`}, {"negative-string", `"-24"`}, {"zero", `0`},
				{"big-number", `9007199254740993`}, {"big-string", `"9007199254740993"`}, {"null", `null`},
				{"fraction", `24.5`}, {"bool", `false`}, {"vendor-string", `"not-an-integer"`},
				{"object", `{"vendor":{"quota":9007199254740993},"extra":true}`}, {"empty-object", `{}`},
				{"array", `[9007199254740993,null,{"flag":true}]`},
			} {
				rows = append(rows, subnetRawBodyRow(t, item.id, "same", map[string]json.RawMessage{"prefixlen": json.RawMessage(item.raw)}))
			}
			rows = append(rows, subnetRawBodyRow(t, "missing", "same", nil))
			cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, subnetRawBodyPage(strings.Join(rows, ","), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for _, tc := range []struct {
				filter any
				ids    []string
			}{
				{24, []string{"number", "decimal", "exponent"}},
				{json.RawMessage(`2.4e1`), []string{"number", "decimal", "exponent"}},
				{"24", []string{"string"}}, {-24, []string{"negative-number"}}, {"-24", []string{"negative-string"}},
				{0, []string{"zero"}}, {nil, []string{"null", "missing"}},
				{json.RawMessage(`9007199254740993`), []string{"big-number"}}, {"9007199254740993", []string{"big-string"}},
				{json.RawMessage(`9007199254740992`), nil},
				{json.RawMessage(`24.5`), []string{"fraction"}}, {false, []string{"bool"}}, {"not-an-integer", []string{"vendor-string"}},
				{json.RawMessage(`{"vendor":{"quota":9007199254740993}}`), []string{"object"}}, {map[string]any{}, []string{"object"}},
				{json.RawMessage(`[9007199254740993,null,{"flag":true}]`), []string{"array"}},
				{json.RawMessage(`[9007199254740993,null,{}]`), nil},
			} {
				values, err := a.Resources.All(context.Background(), resource.WithBodyFilter("prefix_length", tc.filter))
				subnetRawBodyWant(t, values, err, tc.ids...)
			}
		})
		for _, field := range []string{"allocation_pools", "host_routes", "revision_number"} {
			t.Run(f.name+"/precision/"+field, func(t *testing.T) {
				cloud := testcloud.New(t)
				original := subnetRawBodyValues()[field]
				rows := subnetRawBodyRow(t, "exact", "same", map[string]json.RawMessage{field: original}) + "," + subnetRawBodyRow(t, "rounded", "same", map[string]json.RawMessage{field: json.RawMessage(strings.ReplaceAll(string(original), "9007199254740993", "9007199254740992"))})
				cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, subnetRawBodyPage(rows, "")) })
				a := f.open(t, networkExtensionClient(cloud))
				values, err := a.Resources.All(context.Background(), resource.WithBodyFilter(field, original))
				subnetRawBodyWant(t, values, err, "exact")
				values, err = a.Resources.All(context.Background(), resource.WithBodyFilter(field, json.RawMessage(strings.ReplaceAll(string(original), "9007199254740993", "9007199254740992"))))
				subnetRawBodyWant(t, values, err, "rounded")
				if field == "allocation_pools" || field == "host_routes" {
					// Native structs discard vendor keys; arrays still compare
					// complete original objects rather than typed projections.
					var entries []json.RawMessage
					if err := json.Unmarshal(original, &entries); err != nil {
						t.Fatal(err)
					}
					index := 0
					if field == "host_routes" {
						index = 1
					}
					var object map[string]json.RawMessage
					if err := json.Unmarshal(entries[index], &object); err != nil {
						t.Fatal(err)
					}
					delete(object, "vendor")
					entries[index], _ = json.Marshal(object)
					values, err = a.Resources.All(context.Background(), resource.WithBodyFilter(field, entries))
					subnetRawBodyWant(t, values, err)
				}
			})
		}
	}
}

func TestNetworkSubnetRawBodyFiltersSnapshotsAliasesAndConcurrentReuse(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			source := []any{"192.0.2.1", nil}
			pool := []any{map[string]any{"start": "10.0.0.2", "end": "10.0.0.20", "vendor": map[string]any{"quota": json.Number("9007199254740993")}}, nil}
			filters := map[string]any{"allocation_pools": pool, "prefix_length": 24}
			bulk, single := resource.WithBodyFilters(filters), resource.WithBodyFilter("dns_nameservers", source)
			raw := json.RawMessage(`"2025-01-02T03:04:05+00:00"`)
			timestamp := resource.WithBodyFilter("created_at", raw)
			filters["prefix_length"], source[0], raw[1] = 0, "changed", 'X'
			pool[0].(map[string]any)["vendor"].(map[string]any)["quota"] = 0
			target := subnetRawBodyValues()
			empty := map[string]json.RawMessage{"allocation_pools": json.RawMessage(`[]`), "prefixlen": json.RawMessage(`0`), "dns_nameservers": json.RawMessage(`[]`), "revision_number": json.RawMessage(`0`)}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("snapshot became query", r.URL)
				}
				testcloud.JSON(w, 200, subnetRawBodyPage(subnetRawBodyRow(t, "target", "same", target)+","+subnetRawBodyRow(t, "empty", "same", empty), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for _, opts := range [][]resource.ListOption{
				{bulk}, {single}, {timestamp}, {single, timestamp},
				{resource.WithBodyFilter("prefixlen", 0), resource.WithBodyFilter("prefix_length", 24)},
				{resource.WithBodyFilter("prefix_length", 0), resource.WithBodyFilter("prefixlen", 24)},
			} {
				values, err := a.Resources.All(context.Background(), opts...)
				subnetRawBodyWant(t, values, err, "target")
			}
			values, err := a.Resources.All(context.Background(), single, resource.WithBodyFilter("revision_number", 0))
			subnetRawBodyWant(t, values, err)
			values, err = a.Resources.All(context.Background(), single, resource.WithBodyFilters(map[string]any{"allocation_pools": []any{}}))
			subnetRawBodyWant(t, values, err, "empty")
			values, err = a.Resources.All(context.Background(), resource.WithBodyFilter("prefixlen", 24), resource.WithBodyFilter("prefix_length", 0))
			subnetRawBodyWant(t, values, err, "empty")
			before := calls.Load()
			if values, err := a.Resources.All(context.Background(), resource.WithBodyFilters(map[string]any{"prefixlen": 24, "prefix_length": 24})); values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
				t.Fatal("bulk alias conflict reached HTTP", values, err, calls.Load())
			}
			opts := []resource.ListOption{bulk}
			seq := a.Resources.List(context.Background(), opts...)
			opts[0] = resource.WithBodyFilter("prefixlen", 0)
			for range 2 {
				var ids []string
				for value, err := range seq {
					if err != nil {
						t.Fatal(err)
					}
					ids = append(ids, value.ID)
				}
				if !reflect.DeepEqual(ids, []string{"target"}) {
					t.Fatal("retained iterator changed", ids)
				}
			}
			for _, clear := range []resource.ListOption{resource.WithBodyFilters(nil), resource.WithBodyFilters(map[string]any{})} {
				values, err := a.Resources.All(context.Background(), bulk, clear)
				subnetRawBodyWant(t, values, err, "target", "empty")
			}
			var group sync.WaitGroup
			for range 6 {
				group.Add(1)
				go func() {
					defer group.Done()
					values, err := a.Resources.All(context.Background(), bulk, single)
					if err != nil || len(values) != 1 || values[0].ID != "target" {
						t.Error(values, err)
					}
				}()
			}
			group.Wait()
			if calls.Load() != 19 {
				t.Fatal("immutable options did not produce independent iterations", calls.Load())
			}
		})
	}
}

func TestNetworkSubnetRawBodyFiltersWireAndNativeIsolation(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var phase atomic.Value
			phase.Store("local")
			var lists, gets atomic.Int32
			cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				q := r.URL.Query()
				if q.Get("prefixlen") != "wire-prefix" || q.Get("prefix_length") != "wire-alias" || q.Get("revision_number") != "wire-revision" || q.Get("tenant_id") != "wire-tenant" || q.Get("status") != "vendor-status" || q.Get("vendor") != "a&b" {
					t.Error("raw query changed", r.URL)
				}
				if phase.Load() == "local" && q.Get("name") != "Target" || phase.Load() == "native" && q.Has("name") {
					t.Error("local name or typed native query changed", r.URL)
				}
				fields := subnetRawBodyValues()
				rows := subnetRawBodyRow(t, "target", "Target", fields)
				fields["tenant_id"] = json.RawMessage(`"other"`)
				rows += "," + subnetRawBodyRow(t, "other", "Target", fields)
				testcloud.JSON(w, 200, subnetRawBodyPage(rows, ""))
			})
			cloud.Mux.HandleFunc("GET "+subnetRawBodyPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.URL.Query().Get("prefixlen") != "wire-only" || len(r.URL.Query()) != 1 {
					t.Error("identity query changed", r.URL)
				}
				// Prefixlen is not modeled natively; ordinary local filters
				// must not alter native GET/FindIdentity's model or query policy.
				testcloud.JSON(w, 200, `{"subnet":`+subnetRawBodyRow(t, "lookup", "Different", map[string]json.RawMessage{"prefixlen": json.RawMessage(`"vendor-native-value"`)})+`}`)
			})
			a := f.open(t, networkExtensionClient(cloud))
			bulk := make(map[string]any)
			for key, raw := range subnetRawBodyValues() {
				bulk[key] = raw
			}
			opts := []resource.ListOption{resource.WithBodyFilters(bulk), resource.WithName("Target")}
			for _, item := range [][2]string{{"prefixlen", "wire-prefix"}, {"prefix_length", "wire-alias"}, {"revision_number", "wire-revision"}, {"tenant_id", "wire-tenant"}, {"status", "vendor-status"}, {"vendor", "a&b"}} {
				opts = append(opts, resource.WithQuery(item[0], item[1]))
			}
			values, err := a.Resources.All(context.Background(), opts...)
			subnetRawBodyWant(t, values, err, "target")
			phase.Store("native")
			var nativeOpts []subnets.ListOption
			for _, item := range [][2]string{{"prefixlen", "wire-prefix"}, {"prefix_length", "wire-alias"}, {"revision_number", "wire-revision"}, {"tenant_id", "wire-tenant"}, {"status", "vendor-status"}, {"vendor", "a&b"}} {
				nativeOpts = append(nativeOpts, subnets.WithListQuery(item[0], item[1]))
			}
			var ids []string
			for value, err := range a.List(context.Background(), nativeOpts...) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, value.ID)
			}
			if !reflect.DeepEqual(ids, []string{"target", "other"}) {
				t.Fatal("typed List acquired local filters", ids)
			}
			value, err := a.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindQuery("prefixlen", "wire-only"))
			if err != nil || value == nil || value.ID != "lookup" || lists.Load() != 2 || gets.Load() != 1 {
				t.Fatal("native identity path changed", value, err, lists.Load(), gets.Load())
			}
		})
	}
	t.Run("qos-raw-comparison-keeps-native-return-projection", func(t *testing.T) {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("GET "+networkExtensionPrefix+"qos/policies", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"policies":[{"id":"rounded","name":"same","rules":[{"quota":9007199254740993}]}]}`)
		})
		a := qospolicies.New(networkExtensionClient(cloud))
		for _, tc := range []struct {
			raw   string
			count int
		}{{`[{"quota":9007199254740993}]`, 1}, {`[{"quota":9007199254740992}]`, 0}} {
			values, err := a.Resources.All(context.Background(), resource.WithBodyFilter("rules", json.RawMessage(tc.raw)))
			if err != nil || len(values) != tc.count {
				t.Fatal("raw QoS comparison used rounded native numbers", values, err)
			}
			if len(values) != 0 && values[0].Rules[0]["quota"] != float64(9007199254740992) {
				t.Fatal("raw comparison replaced native returned float64 decoding", values[0].Rules)
			}
		}
	})
}

func TestNetworkSubnetRawBodyFiltersControlsAndContinuation(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		for _, mode := range []string{"raw-cap", "prefix-after-cap", "prefix-all-pages", "single-page", "break", "two-pages", "late-http", "cycle", "foreign", "cancel", "wrong-link-shape"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
					page := calls.Add(1)
					q := r.URL.Query()
					if q.Get("name") != "Target" || q.Get("vendor") != "kept" || q.Has("dns_nameservers") || q.Has("tenant_id") || q.Has("limit") || q.Has("max_items") || q.Has("paginated") {
						t.Error("raw control/filter wire leak", r.URL)
					}
					q.Set("marker", "second")
					next := cloud.Server.URL + subnetRawBodyPath + "?" + q.Encode()
					fields := map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant-original"`), "dns_nameservers": json.RawMessage(`["192.0.2.1",null]`)}
					if strings.HasPrefix(mode, "prefix-") {
						fields["prefixlen"] = json.RawMessage(`24`)
					}
					candidate := subnetRawBodyRow(t, "candidate", "Target", fields)
					rows := candidate
					if strings.HasPrefix(mode, "prefix-") {
						// Native Subnet ignores prefixlen. Its untyped raw bool
						// is a normal mismatch against the numeric filter.
						fields["prefixlen"] = json.RawMessage(`true`)
						if page == 1 {
							rows += "," + subnetRawBodyRow(t, "bool-prefix", "Other", fields)
						} else {
							rows, next = subnetRawBodyRow(t, "bool-prefix-later", "Target", fields), ""
						}
						if mode == "prefix-after-cap" {
							next = "https://foreign.invalid/not-read"
						}
					} else if mode == "raw-cap" || mode == "single-page" || mode == "break" || mode == "two-pages" && page == 1 {
						rows = subnetRawBodyRow(t, "project-decoy", "Target", map[string]json.RawMessage{"project_id": fields["tenant_id"], "dns_nameservers": fields["dns_nameservers"]}) + "," + subnetRawBodyRow(t, "wrong-name", "Other", fields)
						if mode != "two-pages" {
							rows += "," + candidate
							next = "https://foreign.invalid/not-read"
						}
					} else if mode == "wrong-link-shape" {
						testcloud.JSON(w, 200, `{"subnets":[`+rows+`],"links":{"next":"https://foreign.invalid/ignored"}}`)
						return
					} else if mode == "foreign" {
						next = "https://foreign.invalid/rejected"
					} else if page > 1 {
						switch mode {
						case "late-http":
							w.WriteHeader(503)
							return
						case "cancel":
							cancel()
							next = ""
						case "two-pages":
							next = ""
						case "cycle":
							rows = subnetRawBodyRow(t, "nonmatch", "Other", nil)
						}
					}
					testcloud.JSON(w, 200, subnetRawBodyPage(rows, next))
				})
				a := f.open(t, networkExtensionClient(cloud))
				opts := []resource.ListOption{resource.WithBodyFilter("tenant_id", "tenant-original"), resource.WithBodyFilter("dns_nameservers", []any{"192.0.2.1", nil}), resource.WithName("Target"), resource.WithQuery("vendor", "kept")}
				if strings.HasPrefix(mode, "prefix-") {
					opts = append(opts, resource.WithBodyFilter("prefixlen", 24))
				}
				switch mode {
				case "raw-cap":
					opts = append(opts, resource.WithMaxItems(2))
				case "prefix-after-cap":
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
				case "raw-cap":
					subnetRawBodyWant(t, values, err)
				case "prefix-after-cap", "prefix-all-pages", "single-page", "wrong-link-shape", "two-pages":
					subnetRawBodyWant(t, values, err, "candidate")
				default:
					if values != nil || err == nil || mode == "late-http" && !gophercloud.ResponseCodeIs(err, 503) || mode == "cycle" && !errors.Is(err, resource.ErrPaginationCycle) || mode == "cancel" && !errors.Is(err, context.Canceled) {
						t.Fatal("local filters hid a terminal continuation failure", values, err)
					}
				}
				wantCalls := int32(2)
				if mode == "raw-cap" || mode == "prefix-after-cap" || mode == "single-page" || mode == "wrong-link-shape" || mode == "foreign" {
					wantCalls = 1
				}
				if calls.Load() != wantCalls {
					t.Fatal("unexpected continuation/follow-up", calls.Load(), wantCalls)
				}
			})
		}
	}
}

func TestNetworkSubnetRawBodyFiltersNativeWholePageFailures(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		for _, tc := range []struct{ field, raw string }{
			{"allocation_pools", `[false]`}, {"dns_nameservers", `[false]`}, {"host_routes", `[{"destination":true}]`},
			{"service_types", `[{}]`}, {"created_at", `"invalid-time"`}, {"updated_at", `"invalid-time"`},
			{"revision_number", `false`}, {"revision_number", `"9"`},
		} {
			t.Run(f.name+"/native/"+tc.field+"/"+tc.raw, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				page := subnetRawBodyPage(subnetRawBodyRow(t, "candidate", "Target", subnetRawBodyValues())+","+subnetRawBodyRow(t, "bad", "Other", map[string]json.RawMessage{tc.field: json.RawMessage(tc.raw)}), "")
				cloud.Mux.HandleFunc("GET "+subnetRawBodyPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 200, page)
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithMaxItems(1), resource.WithName("Target"), resource.WithBodyFilter("tenant_id", "tenant-original"))
				if values != nil || err == nil || calls.Load() != 1 {
					t.Fatal("native whole-page validation was hidden by cap/name/raw filters", values, err, calls.Load())
				}
				if tc.field == "created_at" || tc.field == "updated_at" {
					var cause *time.ParseError
					if !errors.As(err, &cause) {
						t.Fatal("native timestamp cause was lost", err)
					}
				} else {
					var cause *json.UnmarshalTypeError
					if !errors.As(err, &cause) {
						t.Fatal("native typed decode cause was lost", err)
					}
				}
			})
		}
	}
}

func TestNetworkSubnetRawBodyFiltersLazyPreflightAndCapabilities(t *testing.T) {
	for _, f := range subnetRawBodyFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			a := f.open(t, networkExtensionClient(cloud))
			for _, option := range []resource.ListOption{
				resource.WithBodyFilter("project_id", "project"), resource.WithBodyFilter("PREFIXLEN", 24),
				resource.WithBodyFilter("unknown", nil), resource.WithBodyFilter("allocation_pools", json.RawMessage(`{`)),
				resource.WithBodyFilters(map[string]any{"tenant_id": nil, "unknown": nil}),
			} {
				seq := a.Resources.List(context.Background(), option)
				if calls.Load() != 0 {
					t.Fatal("List construction performed HTTP")
				}
				var errorsSeen int
				for value, err := range seq {
					if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(value, err)
					}
					errorsSeen++
				}
				if errorsSeen != 1 {
					t.Fatal(errorsSeen)
				}
			}
			if values, err := a.Resources.All(context.Background(), resource.WithBodyFilter("tenant_id", nil), resource.WithStatus("ACTIVE")); values != nil || !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal("Subnet acquired typed status capability", values, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if values, err := a.Resources.All(ctx, resource.WithBodyFilters(nil)); values != nil || !errors.Is(err, context.Canceled) {
				t.Fatal(values, err)
			}
			if calls.Load() != 0 {
				t.Fatal("invalid or canceled raw filtering performed HTTP", calls.Load())
			}
		})
	}
}
