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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	qospolicies "gophercloudsdk/network/v2/extensions/qos/policies"
	"gophercloudsdk/network/v2/extensions/security/addressgroups"
	"gophercloudsdk/resource"
)

// Independent tables from pinned AddressGroup._query_mapping and inherited
// QueryParameters/Resource fields. Query-capable attributes stay server-only;
// the deprecated tenant_id and original list values require the raw row lane.
type addressGroupListFixture struct {
	name string
	open func(*testing.T, *gophercloud.ServiceClient) *addressgroups.API
}

func addressGroupListFixtures() []addressGroupListFixture {
	return []addressGroupListFixture{
		{"leaf", func(_ *testing.T, c *gophercloud.ServiceClient) *addressgroups.API { return addressgroups.New(c) }},
		{"connection", func(t *testing.T, c *gophercloud.ServiceClient) *addressgroups.API {
			s := networkBodyFilterConnection(t, c)
			if s.API.SecurityAddressGroups.RawClient() != s.RawClient() {
				t.Fatal("AddressGroup collection did not share the cached service client")
			}
			return s.API.SecurityAddressGroups
		}},
	}
}

const addressGroupListPath = networkExtensionPrefix + "address-groups"

func addressGroupListRow(t *testing.T, id, name string, fields map[string]json.RawMessage) string {
	t.Helper()
	body := map[string]json.RawMessage{"id": json.RawMessage(fmt.Sprintf("%q", id)), "name": json.RawMessage(fmt.Sprintf("%q", name)), "project_id": json.RawMessage(`"project-independent"`), "description": json.RawMessage(`"native-description"`)}
	for key, value := range fields {
		body[key] = value
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func addressGroupListPage(rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,"address_groups_links":[{"rel":"next","href":%q}]`, next)
	}
	return `{"address_groups":[` + rows + `]` + links + `}`
}

func addressGroupListWant(t *testing.T, values []*addressgroups.AddressGroup, err error, ids ...string) {
	t.Helper()
	actual := make([]string, 0, len(values))
	for _, value := range values {
		actual = append(actual, value.ID)
	}
	want := append([]string{}, ids...)
	if err != nil || !reflect.DeepEqual(actual, want) {
		t.Fatal("actual native collection result", actual, err, "want", want)
	}
}

func TestAddressGroupListFiltersEntireQueryDescriptor(t *testing.T) {
	queries := []string{"limit", "marker", "fields", "sort_key", "sort_dir", "name", "description", "project_id"}
	for _, f := range addressGroupListFixtures() {
		for _, key := range queries {
			t.Run(f.name+"/"+key, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{key: {"server-value"}}) {
						t.Error("query classification changed", key, r.URL)
					}
					// Even a queried name is not an implicit local predicate.
					testcloud.JSON(w, 200, addressGroupListPage(addressGroupListRow(t, "server-returned", "Different", nil), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter(key, "server-value"))
				addressGroupListWant(t, values, err, "server-returned")
				if calls.Load() != 1 {
					t.Fatal("query filtering performed additional HTTP", calls.Load())
				}
			})
		}
		t.Run(f.name+"/encoding", func(t *testing.T) {
			cloud := testcloud.New(t)
			want := url.Values{"fields": {"id", "name", "true", "false", "9007199254740993", ""}, "sort_dir": {"false"}, "limit": {"2e1"}, "name": {""}}
			cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, r *http.Request) {
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("query lexical values or omitted-key encoding changed", r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, addressGroupListPage(addressGroupListRow(t, "result", "Different", nil), ""))
			})
			values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{
				"fields": []any{"id", nil, "name", true, false, json.Number("9007199254740993"), ""}, "sort_dir": false, "limit": json.Number("2e1"), "name": "", "description": nil, "marker": []any{},
			}))
			addressGroupListWant(t, values, err, "result")
		})
	}
}

func TestAddressGroupListFiltersRawBodyDescriptorAndPresence(t *testing.T) {
	for _, f := range addressGroupListFixtures() {
		for _, key := range []string{"id", "tenant_id", "addresses"} {
			t.Run(f.name+"/"+key, func(t *testing.T) {
				cloud := testcloud.New(t)
				fields := map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant-original"`), "addresses": json.RawMessage(`["192.0.2.1",null,"198.51.100.0/24"]`)}
				filter := any("target")
				if key != "id" {
					filter = fields[key]
				}
				rows := addressGroupListRow(t, "target", "Different", fields)
				fields[key] = json.RawMessage(`"decoy"`)
				if key == "addresses" {
					fields[key] = json.RawMessage(`[]`)
				}
				rows += "," + addressGroupListRow(t, "decoy", "Different", fields)
				cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), url.Values{"project_id": {"server-project"}}) {
						t.Error("Body field leaked to query or aliased tenant to project", key, r.URL)
					}
					testcloud.JSON(w, 200, addressGroupListPage(rows, ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter(key, filter), resource.WithFilter("project_id", "server-project"))
				addressGroupListWant(t, values, err, "target")
				if values[0].ProjectID != "project-independent" || !reflect.DeepEqual(values[0].Addresses, []string{"192.0.2.1", "", "198.51.100.0/24"}) {
					t.Fatal("native projection was replaced by raw filter values", values[0])
				}
			})
		}
		for _, tc := range []struct {
			name, raw string
			filter    any
			match     bool
		}{
			{"missing", "missing", nil, true}, {"null", "null", nil, true}, {"empty-distinct", "[]", nil, false},
			{"empty", "[]", []string{}, true}, {"ordered", `["192.0.2.1","198.51.100.0/24"]`, []string{"192.0.2.1", "198.51.100.0/24"}, true},
			{"order", `["192.0.2.1","198.51.100.0/24"]`, []string{"198.51.100.0/24", "192.0.2.1"}, false},
			{"shorter", `["192.0.2.1","198.51.100.0/24"]`, []string{"192.0.2.1"}, false},
			{"longer", `["192.0.2.1","198.51.100.0/24"]`, []string{"192.0.2.1", "198.51.100.0/24", "::1"}, false},
			{"no-cidr-normalization", `["192.0.2.1/24"]`, []string{"192.0.2.0/24"}, false},
			{"raw-null-element", `[null,"192.0.2.1"]`, []any{nil, "192.0.2.1"}, true},
			{"null-not-empty-string", `[null,"192.0.2.1"]`, []string{"", "192.0.2.1"}, false},
			{"wrong-filter-type", `["192.0.2.1"]`, "192.0.2.1", false},
		} {
			t.Run(f.name+"/addresses/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				fields := make(map[string]json.RawMessage)
				if tc.raw != "missing" {
					fields["addresses"] = json.RawMessage(tc.raw)
				}
				cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.RawQuery != "" {
						t.Error("explicit Body filter became a query", r.URL)
					}
					testcloud.JSON(w, 200, addressGroupListPage(addressGroupListRow(t, "target", "Different", fields), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithBodyFilter("addresses", tc.filter))
				var want []string
				if tc.match {
					want = []string{"target"}
				}
				addressGroupListWant(t, values, err, want...)
			})
		}
		for _, tc := range []struct {
			name, field, raw string
			filter           any
			match            bool
		}{
			{"id-null", "id", "null", nil, true}, {"id-missing", "id", "missing", nil, true}, {"id-null-not-empty", "id", "null", "", false}, {"id-empty", "id", `""`, "", true},
			{"tenant-no-project-fallback", "tenant_id", "missing", "project-independent", false},
			{"tenant-null", "tenant_id", "null", nil, true}, {"tenant-exact-number", "tenant_id", "9007199254740993", json.Number("9007199254740993"), true},
			{"tenant-no-rounding", "tenant_id", "9007199254740993", json.Number("9007199254740992"), false},
			{"tenant-decimal-equality", "tenant_id", "1.000e0", json.Number("1"), true}, {"tenant-bool-number-distinct", "tenant_id", "true", 1, false},
			{"tenant-object-subset", "tenant_id", `{"vendor":{"n":9007199254740993,"extra":true}}`, json.RawMessage(`{"vendor":{"n":9007199254740993}}`), true},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				row := addressGroupListRow(t, "target", "Different", nil)
				var fields map[string]json.RawMessage
				if err := json.Unmarshal([]byte(row), &fields); err != nil {
					t.Fatal(err)
				}
				if tc.raw == "missing" {
					delete(fields, tc.field)
				} else {
					fields[tc.field] = json.RawMessage(tc.raw)
				}
				raw, err := json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, _ *http.Request) {
					testcloud.JSON(w, 200, addressGroupListPage(string(raw), ""))
				})
				values, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter(tc.field, tc.filter))
				if err != nil || len(values) != map[bool]int{false: 0, true: 1}[tc.match] {
					t.Fatal("raw field presence or precision changed", values, err)
				}
			})
		}
	}
}

func TestAddressGroupListFiltersSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	for _, f := range addressGroupListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			addresses := []any{nil, "192.0.2.1"}
			fields := []any{"id", "name", nil}
			tenant := "tenant-original"
			input := map[string]any{"addresses": addresses, "fields": fields, "tenant_id": &tenant}
			bulk := resource.WithFilters(input)
			input["addresses"], addresses[1], fields[0], tenant = nil, "changed", "changed", "changed"
			var expected atomic.Value
			expected.Store(url.Values{"fields": {"id", "name"}, "vendor": {"kept"}})
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), expected.Load().(url.Values)) {
					t.Error("semantic snapshots or namespace clear changed query", r.URL.Query())
				}
				rows := addressGroupListRow(t, "target", "Different", map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant-original"`), "addresses": json.RawMessage(`[null,"192.0.2.1"]`)})
				rows += "," + addressGroupListRow(t, "other", "Different", map[string]json.RawMessage{"tenant_id": json.RawMessage(`"tenant-original"`), "addresses": json.RawMessage(`[]`)})
				testcloud.JSON(w, 200, addressGroupListPage(rows, ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			opts := []resource.ListOption{resource.WithQuery("vendor", "kept"), bulk}
			values, err := a.Resources.All(context.Background(), opts...)
			addressGroupListWant(t, values, err, "target")
			var wg sync.WaitGroup
			for range 8 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					values, err := a.Resources.All(context.Background(), opts...)
					addressGroupListWant(t, values, err, "target")
				}()
			}
			wg.Wait()
			expected.Store(url.Values{"vendor": {"kept"}})
			for _, clear := range []resource.ListOption{resource.WithFilters(nil), resource.WithFilters(map[string]any{})} {
				values, err = a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), opts...), clear, resource.WithBodyFilter("tenant_id", "tenant-original"))...)
				addressGroupListWant(t, values, err, "target", "other")
			}
			for _, tc := range []struct {
				options []resource.ListOption
				want    []string
			}{
				{[]resource.ListOption{resource.WithFilter("fields", math.NaN()), resource.WithFilter("fields", nil), resource.WithFilter("addresses", make(chan int)), resource.WithFilter("addresses", []any{nil, "192.0.2.1"})}, []string{"target"}},
				{[]resource.ListOption{resource.WithFilter("fields", map[string]any{"bad": true}), resource.WithFilter("addresses", json.RawMessage(`{`)), resource.WithFilter("headers", true), resource.WithFilters(nil)}, []string{"target", "other"}},
				{[]resource.ListOption{resource.WithFilter("addresses", []string{}), resource.WithFilter("addresses", []any{nil, "192.0.2.1"})}, []string{"target"}},
			} {
				values, err = a.Resources.All(context.Background(), append([]resource.ListOption{resource.WithQuery("vendor", "kept")}, tc.options...)...)
				addressGroupListWant(t, values, err, tc.want...)
			}
			if calls.Load() != 14 {
				t.Fatal("option reuse changed request count", calls.Load())
			}
		})
	}
}

func TestAddressGroupListFiltersLazyPreflightAndNamespaceCollisions(t *testing.T) {
	for _, f := range addressGroupListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, addressGroupListPage(addressGroupListRow(t, "result", "Different", nil), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			collisions := [][]resource.ListOption{
				{resource.WithFilter("name", nil), resource.WithName("Target")}, {resource.WithFilter("fields", []any{}), resource.WithQuery("fields", "")},
				{resource.WithFilter("limit", 2), resource.WithPageSize(2)}, {resource.WithFilter("addresses", nil), resource.WithBodyFilter("addresses", nil)},
				{resource.WithFilter("tenant_id", "tenant"), resource.WithBodyFilter("tenant_id", "tenant")},
			}
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
			invalid := []resource.ListOption{resource.WithFilter("fields", map[string]any{"id": true}), resource.WithFilter("fields", [][]string{{"id"}}), resource.WithFilter("addresses", math.NaN()), resource.WithBodyFilter("tenant_id", json.RawMessage(`{`)), resource.WithBodyFilter("name", "Different"), resource.WithBodyFilter("project_id", "project"), resource.WithStatus("ACTIVE"), resource.WithMaxItems(-1)}
			for _, key := range []string{"max_items", "paginated", "base_path", "allow_unknown_params", "session", "headers", "microversion", "resource_type", "jmespath_filters"} {
				invalid = append(invalid, resource.WithFilter(key, nil))
			}
			for _, option := range invalid {
				seq := a.Resources.List(context.Background(), option)
				if calls.Load() != 0 {
					t.Fatal("iterator construction performed HTTP")
				}
				seen := 0
				for value, err := range seq {
					seen++
					if value != nil || (!errors.Is(err, resource.ErrInvalidOption) && !errors.Is(err, resource.ErrUnsupported)) {
						t.Fatal(value, err)
					}
				}
				if seen != 1 || calls.Load() != 0 {
					t.Fatal("invalid options did not yield one terminal preflight error", seen, calls.Load())
				}
			}
			values, err := a.Resources.All(context.Background(), resource.WithFilters(map[string]any{"status": make(chan int), "algorithm": math.NaN(), "unknown": json.RawMessage(`{`)}))
			addressGroupListWant(t, values, err, "result")
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			values, err = a.Resources.All(ctx, resource.WithFilters(nil))
			if values != nil || !errors.Is(err, context.Canceled) || calls.Load() != 1 {
				t.Fatal("cancellation was hidden or performed HTTP", values, err, calls.Load())
			}
			_, err = qospolicies.New(networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(nil))
			if !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatal("unrelated explicit Body binding gained semantics", err, calls.Load())
			}
		})
	}
}

func TestAddressGroupListFiltersControlsAndNativeWholePageFailures(t *testing.T) {
	for _, f := range addressGroupListFixtures() {
		for _, mode := range []string{"raw-cap-before-name", "single-page", "break", "null", "late-null", "cap-null", "empty-object", "query-null", "bad-addresses-object", "bad-address-element", "bad-id", "bad-name", "bad-description", "bad-project"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls, followed atomic.Int32
				foreign := testcloud.New(t)
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); w.WriteHeader(500) })
				rows := addressGroupListRow(t, "first", "Different", map[string]json.RawMessage{"addresses": json.RawMessage(`[]`)})
				rows += "," + addressGroupListRow(t, "target", "Target", map[string]json.RawMessage{"addresses": json.RawMessage(`["192.0.2.1"]`)})
				opts := []resource.ListOption{resource.WithFilter("addresses", []string{})}
				switch mode {
				case "raw-cap-before-name":
					opts = append(opts, resource.WithName("Target"), resource.WithMaxItems(1))
				case "single-page":
					opts = append(opts, resource.WithPaginated(false))
				case "null", "query-null":
					rows = "null"
				case "late-null", "cap-null":
					rows = addressGroupListRow(t, "first", "Different", map[string]json.RawMessage{"addresses": json.RawMessage(`[]`)}) + ",null"
				case "empty-object":
					rows = "{}"
					opts = []resource.ListOption{resource.WithFilter("addresses", nil), resource.WithPaginated(false)}
				default:
					if strings.HasPrefix(mode, "bad-") {
						field, raw := "addresses", "{}"
						switch mode {
						case "bad-address-element":
							raw = `[false]`
						case "bad-id":
							field, raw = "id", "false"
						case "bad-name":
							field, raw = "name", "[]"
						case "bad-description":
							field, raw = "description", "5"
						case "bad-project":
							field, raw = "project_id", "{}"
						}
						rows += "," + addressGroupListRow(t, "invalid", "Different", map[string]json.RawMessage{field: json.RawMessage(raw)})
						opts = append(opts, resource.WithMaxItems(1))
					}
				}
				if mode == "cap-null" {
					opts = append(opts, resource.WithMaxItems(1))
				}
				if mode == "query-null" {
					opts = []resource.ListOption{resource.WithFilter("name", "server-only"), resource.WithPaginated(false)}
				}
				cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, _ *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 200, addressGroupListPage(rows, foreign.Server.URL+"/foreign"))
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
						t.Fatal("consumer break fetched continuation", seen, calls.Load(), followed.Load())
					}
					return
				}
				values, err := a.Resources.All(context.Background(), opts...)
				switch {
				case mode == "raw-cap-before-name":
					addressGroupListWant(t, values, err)
				case mode == "single-page" || mode == "cap-null":
					addressGroupListWant(t, values, err, "first")
				case mode == "empty-object" || mode == "query-null":
					addressGroupListWant(t, values, err, "")
				case mode == "null" || mode == "late-null":
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal("consumed null row was accepted or partial All returned", values, err)
					}
				default:
					var cause *json.UnmarshalTypeError
					if values != nil || !errors.As(err, &cause) {
						t.Fatal("native whole-page decoder or error cause changed", values, err)
					}
				}
				if calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal("cap/terminal current-page error fetched continuation", calls.Load(), followed.Load())
				}
			})
		}
	}
}

func TestAddressGroupListFiltersPagingLiveSourceAndNativeStatus(t *testing.T) {
	for _, f := range addressGroupListFixtures() {
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
				cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("X-Configured") != "kept" || r.Header.Get("OpenStack-API-Version") != "network 2.0" || r.URL.Query().Get("project_id") != "project-query" || r.URL.Query().Get("status") != "vendor-status" {
						t.Error("source or query changed", r.Header, r.URL)
					}
					if r.URL.Query().Get("marker") != "" {
						if r.Header.Get("X-Auth-Token") != "later-token" {
							t.Error("provider token did not remain live", r.Header)
						}
						switch mode {
						case "late-http":
							w.WriteHeader(404)
							return
						case "late-decode":
							testcloud.JSON(w, 200, addressGroupListPage(addressGroupListRow(t, "invalid", "Different", map[string]json.RawMessage{"addresses": json.RawMessage(`{}`)}), ""))
							return
						}
						id := "second"
						if mode == "duplicates" {
							id = "first"
						}
						testcloud.JSON(w, 200, addressGroupListPage(addressGroupListRow(t, id, "Different", map[string]json.RawMessage{"addresses": json.RawMessage(`[]`)}), ""))
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
					next := cloud.Server.URL + addressGroupListPath + "?marker=next&project_id=project-query&status=vendor-status"
					if mode == "cycle" {
						next = cloud.Server.URL + r.URL.String()
					}
					if mode == "foreign" {
						next = foreign.Server.URL + "/foreign"
					}
					row := addressGroupListRow(t, "first", "Different", map[string]json.RawMessage{"addresses": json.RawMessage(`[]`)})
					if mode == "wrong-link-shape" {
						testcloud.JSON(w, 300, `{"address_groups":[`+row+`],"links":{"next":`+fmt.Sprintf("%q", next)+`}}`)
						return
					}
					testcloud.JSON(w, 300, addressGroupListPage(row, next))
				})
				a := f.open(t, c)
				values, err := a.Resources.All(ctx, resource.WithFilter("addresses", []string{}), resource.WithFilter("project_id", "project-query"), resource.WithQuery("status", "vendor-status"))
				switch mode {
				case "complete":
					addressGroupListWant(t, values, err, "first", "second")
				case "duplicates":
					addressGroupListWant(t, values, err, "first", "first")
				case "wrong-link-shape":
					addressGroupListWant(t, values, err, "first")
				case "204":
					addressGroupListWant(t, values, err)
				default:
					if values != nil || err == nil {
						t.Fatal("terminal paging observation returned partial All", values, err)
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
					t.Fatal("native continuation or shared provider changed", calls.Load(), middleware.Load(), followed.Load())
				}
			})
		}
	}
}

func TestAddressGroupListFiltersNativeAndIdentitySurfaceIsolation(t *testing.T) {
	for _, f := range addressGroupListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets atomic.Int32
			want := url.Values{"id": {"wire-id"}, "tenant_id": {"wire-tenant"}, "addresses": {"wire-addresses"}, "status": {"vendor-status"}, "name": {"server-pattern"}, "vendor": {"kept"}}
			cloud.Mux.HandleFunc("GET "+addressGroupListPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("raw or native query acquired semantic classification", r.URL.Query(), want)
				}
				rows := addressGroupListRow(t, "target", "Target", map[string]json.RawMessage{"addresses": json.RawMessage(`[]`)}) + "," + addressGroupListRow(t, "other", "Other", map[string]json.RawMessage{"addresses": json.RawMessage(`["192.0.2.1"]`)})
				testcloud.JSON(w, 200, addressGroupListPage(rows, ""))
			})
			cloud.Mux.HandleFunc("GET "+addressGroupListPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"tenant_id": {"wire-tenant"}, "addresses": {"wire-addresses"}, "status": {"vendor-status"}}) {
					t.Error("FindIdentity acquired semantic filtering", r.URL)
				}
				testcloud.JSON(w, 200, `{"address_group":`+addressGroupListRow(t, "lookup", "Different", map[string]json.RawMessage{"addresses": json.RawMessage(`[]`)})+`}`)
			})
			a := f.open(t, networkExtensionClient(cloud))
			var raw []resource.ListOption
			var native []addressgroups.ListOption
			for key, values := range want {
				raw = append(raw, resource.WithQuery(key, values[0]))
				native = append(native, addressgroups.WithListQuery(key, values[0]))
			}
			values, err := a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilter("addresses", []string{}))...)
			addressGroupListWant(t, values, err, "target")
			values, err = a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilters(nil))...)
			addressGroupListWant(t, values, err, "target", "other")
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
			value, err := a.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindQuery("tenant_id", "wire-tenant"), resource.WithIdentityFindQuery("addresses", "wire-addresses"), resource.WithIdentityFindQuery("status", "vendor-status"))
			if err != nil || value == nil || value.ID != "lookup" || gets.Load() != 1 || lists.Load() != 3 {
				t.Fatal("FindIdentity changed", value, err, gets.Load(), lists.Load())
			}
			// Explicit WithName keeps its established server hint and exact local
			// predicate; a semantic queried name remains server-only (group1).
			cloud.Mux.HandleFunc("GET "+addressGroupListPath+"/unused", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) })
			want["name"] = []string{"Target"}
			rawName := []resource.ListOption{resource.WithName("Target"), resource.WithQuery("id", "wire-id"), resource.WithQuery("tenant_id", "wire-tenant"), resource.WithQuery("addresses", "wire-addresses"), resource.WithQuery("status", "vendor-status"), resource.WithQuery("vendor", "kept"), resource.WithBodyFilter("addresses", []string{})}
			values, err = a.Resources.All(context.Background(), rawName...)
			addressGroupListWant(t, values, err, "target")
			before := lists.Load()
			values, err = a.Resources.All(context.Background(), resource.WithStatus("ACTIVE"), resource.WithBodyFilter("addresses", []string{}))
			if values != nil || !errors.Is(err, resource.ErrUnsupported) || lists.Load() != before {
				t.Fatal("native model without Status gained status filtering", values, err, lists.Load())
			}
		})
	}
}
