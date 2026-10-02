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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network/v2/extensions/trunks"
	"gophercloudsdk/network/v2/ports"
	"gophercloudsdk/resource"
)

// Independent pinned Trunk table: Resource + TagMixin, not NetworkResource.
// In particular sub_ports is a query, while id and tenant_id are local Body.
func trunkListQueries() map[string]string {
	return map[string]string{"name": "name", "description": "description", "fields": "fields", "port_id": "port_id", "status": "status", "sub_ports": "sub_ports", "project_id": "project_id", "is_admin_state_up": "admin_state_up", "limit": "limit", "marker": "marker", "tags": "tags", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any"}
}

type trunkListFixture struct {
	name string
	open func(*testing.T, *gophercloud.ServiceClient) *trunks.API
}

func trunkListFixtures() []trunkListFixture {
	return []trunkListFixture{
		{"leaf", func(_ *testing.T, c *gophercloud.ServiceClient) *trunks.API { return trunks.New(c) }},
		{"connection", func(t *testing.T, c *gophercloud.ServiceClient) *trunks.API {
			s := networkSubnetBodyConnection(t, c)
			a := s.API.Trunks
			if a.RawClient() != s.RawClient() || a.RawClient().ProviderClient != c.ProviderClient {
				t.Fatal("Trunk facade did not share the cached client")
			}
			return a
		}},
	}
}

const trunkListPath = networkExtensionPrefix + "trunks"

func trunkListRow(t *testing.T, id, name string, fields map[string]json.RawMessage) string {
	t.Helper()
	body := map[string]json.RawMessage{"id": json.RawMessage(fmt.Sprintf("%q", id)), "name": json.RawMessage(fmt.Sprintf("%q", name)), "project_id": json.RawMessage(`"native-project"`), "tenant_id": json.RawMessage(`"native-tenant"`), "status": json.RawMessage(`"ACTIVE"`), "port_id": json.RawMessage(`"parent"`), "admin_state_up": json.RawMessage(`true`), "revision_number": json.RawMessage(`9`)}
	for k, v := range fields {
		if v == nil {
			delete(body, k)
		} else {
			body[k] = v
		}
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func trunkListPage(rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,"links":{"next":%q}`, next)
	}
	return `{"trunks":[` + rows + `]` + links + `}`
}

func trunkListWant(t *testing.T, values []*trunks.Trunk, err error, ids ...string) {
	t.Helper()
	actual := make([]string, 0, len(values))
	for _, value := range values {
		actual = append(actual, value.ID)
	}
	if err != nil || !reflect.DeepEqual(actual, append([]string{}, ids...)) {
		t.Fatal("actual native collection", actual, err, "want", ids)
	}
}

func TestTrunkListFiltersEntireQueryAndAliasDescriptor(t *testing.T) {
	canonical := trunkListQueries()
	accepted := map[string]string{}
	for k, wire := range canonical {
		accepted[k], accepted[wire] = wire, wire
	}
	if len(canonical) != 14 || len(accepted) != 18 {
		t.Fatal(canonical, accepted)
	}
	for _, f := range trunkListFixtures() {
		for k, wire := range accepted {
			t.Run(f.name+"/"+k, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{wire: {"server-value"}}) {
						t.Error(k, r.URL)
					}
					testcloud.JSON(w, 200, trunkListPage(trunkListRow(t, "result", "Different", nil), ""))
				})
				v, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilter(k, "server-value"))
				trunkListWant(t, v, err, "result")
				if calls.Load() != 1 {
					t.Fatal(calls.Load())
				}
			})
		}
		for k, wire := range canonical {
			if k == wire {
				continue
			}
			for _, tc := range []struct {
				name  string
				value any
				want  []string
			}{{"nil", nil, nil}, {"false", false, []string{"false"}}, {"empty", "", []string{""}}, {"empty-array", []any{}, nil}, {"repeated", []any{"first", nil, "last"}, []string{"first", "last"}}} {
				t.Run(f.name+"/precedence/"+k+"/"+tc.name, func(t *testing.T) {
					cloud := testcloud.New(t)
					cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
						want := url.Values{}
						if tc.want != nil {
							want[wire] = tc.want
						}
						if !reflect.DeepEqual(r.URL.Query(), want) {
							t.Error(r.URL.Query(), want)
						}
						testcloud.JSON(w, 200, trunkListPage(trunkListRow(t, "result", "Different", nil), ""))
					})
					v, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{k: tc.value, wire: json.RawMessage(`{`)}))
					trunkListWant(t, v, err, "result")
				})
			}
			for _, reverse := range []bool{false, true} {
				t.Run(f.name+"/last-wins/"+k+fmt.Sprint(reverse), func(t *testing.T) {
					cloud := testcloud.New(t)
					options := []resource.ListOption{resource.WithFilter(wire, "wire-first"), resource.WithFilter(k, "canonical-last")}
					want := "canonical-last"
					if reverse {
						options[0], options[1], want = options[1], options[0], "wire-first"
					}
					cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Query().Get(wire) != want {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 200, trunkListPage(trunkListRow(t, "result", "Different", nil), ""))
					})
					v, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), options...)
					trunkListWant(t, v, err, "result")
				})
			}
		}
		t.Run(f.name+"/scalar-encoding", func(t *testing.T) {
			cloud := testcloud.New(t)
			want := url.Values{"fields": {"id", "true", "false", "9007199254740993", ""}, "admin_state_up": {"false"}, "name": {""}, "limit": {"2e1"}, "sub_ports": {"child-one", "child-two"}}
			cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, trunkListPage(trunkListRow(t, "result", "Different", nil), ""))
			})
			v, err := f.open(t, networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(map[string]any{"fields": []any{"id", nil, true, false, json.Number("9007199254740993"), ""}, "is_admin_state_up": false, "name": "", "limit": json.Number("2e1"), "sub_ports": []string{"child-one", "child-two"}, "description": nil, "marker": []any{}}))
			trunkListWant(t, v, err, "result")
		})
	}
}

func TestTrunkListFiltersRawDescriptorsAndNativeProjection(t *testing.T) {
	for _, f := range trunkListFixtures() {
		t.Run(f.name+"/all-two-and-query-namespaces", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			want := url.Values{}
			cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("local Body leaked", r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, trunkListPage(trunkListRow(t, "target", "Different", map[string]json.RawMessage{"sub_ports": json.RawMessage(`[{"port_id":"child","segmentation_id":101,"segmentation_type":"vlan","unknown":9007199254740993},null]`), "created_at": json.RawMessage(`"2025-01-02T03:04:05+00:00"`), "updated_at": json.RawMessage(`"2025-01-03T03:04:05Z"`)}), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			for _, option := range []resource.ListOption{resource.WithFilter("id", "target"), resource.WithFilter("tenant_id", "native-tenant"), resource.WithFilters(map[string]any{"id": "target", "tenant_id": "native-tenant"})} {
				v, err := a.Resources.All(context.Background(), option)
				trunkListWant(t, v, err, "target")
				if v[0].Name != "Different" || v[0].ProjectID != "native-project" || len(v[0].Subports) != 2 || v[0].Subports[0].SegmentationID != 101 || v[0].Subports[1] != (trunks.Subport{}) || v[0].CreatedAt.IsZero() || !v[0].AdminStateUp || v[0].RevisionNumber != 9 {
					t.Fatal("native projection changed", v[0])
				}
			}
			want = url.Values{"id": {"server-id"}, "tenant_id": {"wire-tenant"}, "project_id": {"wire-project"}, "name": {"server-name"}, "status": {"BUILD"}, "sub_ports": {"server-child"}, "admin_state_up": {"false"}}
			v, err := a.Resources.All(context.Background(), resource.WithFilters(map[string]any{"id": "target", "tenant_id": "native-tenant", "project_id": "wire-project", "name": "server-name", "status": "BUILD", "sub_ports": "server-child", "is_admin_state_up": false}), resource.WithQuery("id", "server-id"), resource.WithQuery("tenant_id", "wire-tenant"))
			trunkListWant(t, v, err, "target")
			if calls.Load() != 4 || v[0].Status != "ACTIVE" || v[0].TenantID != "native-tenant" {
				t.Fatal(calls.Load(), v)
			}
			want = url.Values{}
			for _, value := range []any{nil, false, 1, []any{}, map[string]any{}, "Different"} {
				v, err = a.Resources.All(context.Background(), resource.WithFilter("id", value))
				trunkListWant(t, v, err)
			}
		})
		t.Run(f.name+"/missing-null-empty-no-alternate", func(t *testing.T) {
			cloud := testcloud.New(t)
			rows := trunkListRow(t, "discard", "missing", map[string]json.RawMessage{"id": nil, "tenant_id": nil}) + "," + trunkListRow(t, "discard", "null", map[string]json.RawMessage{"id": json.RawMessage(`null`), "tenant_id": json.RawMessage(`null`)}) + "," + trunkListRow(t, "", "empty", map[string]json.RawMessage{"tenant_id": json.RawMessage(`""`)}) + "," + trunkListRow(t, "value", "normal", nil)
			cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, trunkListPage(rows, "")) })
			a := f.open(t, networkExtensionClient(cloud))
			for _, tc := range []struct {
				key   string
				value any
				names []string
			}{{"id", nil, []string{"missing", "null"}}, {"tenant_id", nil, []string{"missing", "null"}}, {"id", "", []string{"empty"}}, {"tenant_id", "", []string{"empty"}}, {"id", "missing", nil}} {
				v, err := a.Resources.All(context.Background(), resource.WithFilter(tc.key, tc.value))
				var names []string
				for _, row := range v {
					names = append(names, row.Name)
				}
				if err != nil || !reflect.DeepEqual(names, tc.names) {
					t.Fatal(tc.key, tc.value, names, err)
				}
			}
		})
	}
}

func TestTrunkListFiltersSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	for _, f := range trunkListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"vendor": {"kept"}, "admin_state_up": {"false"}, "fields": {"id", "tenant_id"}}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, trunkListPage(trunkListRow(t, "target", "Different", nil), ""))
			})
			qfields := []any{"id", "tenant_id"}
			rawTenant := json.RawMessage(`"native-tenant"`)
			bulk := map[string]any{"id": "target", "tenant_id": rawTenant, "is_admin_state_up": false, "fields": qfields}
			option := resource.WithFilters(bulk)
			copy(rawTenant, []byte(`"mutated-value"`))
			qfields[0], bulk["is_admin_state_up"] = "changed", true
			delete(bulk, "id")
			a := f.open(t, networkExtensionClient(cloud))
			opts := []resource.ListOption{option, resource.WithQuery("vendor", "kept")}
			v, err := a.Resources.All(context.Background(), opts...)
			trunkListWant(t, v, err, "target")
			v[0].ID = "caller-mutated"
			var wg sync.WaitGroup
			for range 6 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					rows, err := a.Resources.All(context.Background(), opts...)
					if err != nil || len(rows) != 1 || rows[0].ID != "target" {
						t.Error(rows, err)
					}
				}()
			}
			wg.Wait()
			for _, prefix := range [][]resource.ListOption{{resource.WithFilter("tenant_id", math.NaN()), resource.WithFilter("tenant_id", "native-tenant")}, {resource.WithFilter("headers", nil), resource.WithFilters(nil)}, {resource.WithFilter("tenant_id", make(chan int)), resource.WithFilters(map[string]any{})}} {
				options := append(prefix, resource.WithFilter("is_admin_state_up", false), resource.WithFilter("fields", []string{"id", "tenant_id"}), resource.WithQuery("vendor", "kept"), resource.WithBodyFilter("id", "target"))
				rows, err := a.Resources.All(context.Background(), options...)
				trunkListWant(t, rows, err, "target")
			}
			if calls.Load() != 10 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestTrunkListFiltersLazyPreflightAndNamespaceCollisions(t *testing.T) {
	for _, f := range trunkListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, trunkListPage(trunkListRow(t, "result", "Different", nil), ""))
			})
			a := f.open(t, networkExtensionClient(cloud))
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if v, err := a.Resources.All(ctx, resource.WithFilter("id", nil)); v != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
				t.Fatal("canceled list started HTTP", v, err, calls.Load())
			}
			for _, pair := range [][]resource.ListOption{{resource.WithFilter("is_admin_state_up", false), resource.WithQuery("admin_state_up", "false")}, {resource.WithFilter("fields", nil), resource.WithQuery("fields", "")}, {resource.WithFilter("name", nil), resource.WithName("Target")}, {resource.WithFilter("status", "ACTIVE"), resource.WithStatus("ACTIVE")}, {resource.WithFilter("limit", 1), resource.WithPageSize(1)}, {resource.WithFilter("id", nil), resource.WithBodyFilter("id", nil)}, {resource.WithFilter("tenant_id", nil), resource.WithBodyFilter("tenant_id", nil)}} {
				for _, reverse := range []bool{false, true} {
					options := append([]resource.ListOption(nil), pair...)
					if reverse {
						options[0], options[1] = options[1], options[0]
					}
					v, err := a.Resources.All(context.Background(), options...)
					if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
						t.Fatal(v, err, calls.Load())
					}
				}
			}
			invalid := []resource.ListOption{resource.WithFilter("fields", map[string]any{"id": true}), resource.WithFilter("sub_ports", [][]string{{"child"}}), resource.WithFilter("tenant_id", math.NaN()), resource.WithBodyFilter("name", "Different"), resource.WithBodyFilter("status", "ACTIVE"), resource.WithBodyFilter("project_id", "project"), resource.WithBodyFilter("sub_ports", []any{}), resource.WithBodyFilter("revision_number", 9), resource.WithBodyFilter("created_at", nil), resource.WithBodyFilter("id", json.RawMessage(`{`)), resource.WithMaxItems(-1)}
			for _, k := range []string{"max_items", "paginated", "base_path", "allow_unknown_params", "session", "headers", "microversion", "resource_type", "jmespath_filters"} {
				invalid = append(invalid, resource.WithFilter(k, nil))
			}
			for _, option := range invalid {
				seq := a.Resources.List(context.Background(), option)
				if calls.Load() != 0 {
					t.Fatal("eager HTTP")
				}
				seen := 0
				for v, err := range seq {
					seen++
					if v != nil || (!errors.Is(err, resource.ErrInvalidOption) && !errors.Is(err, resource.ErrUnsupported)) {
						t.Fatal(v, err)
					}
				}
				if seen != 1 || calls.Load() != 0 {
					t.Fatal(seen, calls.Load())
				}
			}
			v, err := a.Resources.All(context.Background(), resource.WithFilters(map[string]any{"revision_number": json.RawMessage(`{`), "created_at": make(chan int), "updated_at": math.NaN(), "sort_key": map[string]any{"id": true}, "sort_dir": make(chan int), "is_shared": make(chan int)}))
			trunkListWant(t, v, err, "result")
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
			_, err = ports.New(networkExtensionClient(cloud)).Resources.All(context.Background(), resource.WithFilters(nil))
			if !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
		})
	}
}

func TestTrunkListFiltersControlsAndNativeWholePageFailures(t *testing.T) {
	for _, f := range trunkListFixtures() {
		for _, mode := range []string{"cap-filtered", "cap-null", "single-page", "break", "empty-object", "query-null", "null-row", "later-null", "bad-id", "bad-tenant", "bad-bool", "bad-revision", "bad-subports", "bad-subport-port", "bad-subport-segmentation", "no-zone-time", "bad-date"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud, foreign := testcloud.New(t), testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); w.WriteHeader(500) })
				rows := trunkListRow(t, "first", "Target", nil)
				options := []resource.ListOption{resource.WithFilter("tenant_id", "native-tenant")}
				nativeFailure := false
				switch mode {
				case "cap-filtered":
					options = append(options, resource.WithMaxItems(1), resource.WithName("NotTarget"))
					rows += "," + trunkListRow(t, "second", "NotTarget", nil)
				case "cap-null":
					options = append(options, resource.WithMaxItems(1))
					rows += ",null"
				case "single-page":
					options = append(options, resource.WithPaginated(false))
				case "break":
					rows += ",null"
				case "empty-object":
					rows, options = `{}`, []resource.ListOption{resource.WithFilter("id", nil), resource.WithPaginated(false)}
				case "query-null":
					rows, options = `null`, []resource.ListOption{resource.WithFilters(nil), resource.WithPaginated(false)}
				case "null-row":
					rows = `null`
					options = append(options, resource.WithName("NotTarget"))
				case "later-null":
					rows += ",null"
				default:
					key, raw := "id", `true`
					switch mode {
					case "bad-tenant":
						key, raw = "tenant_id", `{}`
					case "bad-bool":
						key, raw = "admin_state_up", `"true"`
					case "bad-revision":
						key, raw = "revision_number", `"9"`
					case "bad-subports":
						key, raw = "sub_ports", `{}`
					case "bad-subport-port":
						key, raw = "sub_ports", `[{"port_id":false}]`
					case "bad-subport-segmentation":
						key, raw = "sub_ports", `[{"segmentation_id":"101"}]`
					case "no-zone-time":
						key, raw = "created_at", `"2025-01-02T03:04:05"`
					case "bad-date":
						key, raw = "updated_at", `"not-time"`
					}
					rows += "," + trunkListRow(t, "invalid", "Other", map[string]json.RawMessage{key: json.RawMessage(raw)})
					options = append(options, resource.WithMaxItems(1))
					nativeFailure = true
				}
				cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Has("limit") {
						t.Error("raw cap became wire limit", r.URL)
					}
					testcloud.JSON(w, 200, trunkListPage(rows, foreign.Server.URL+"/foreign"))
				})
				a := f.open(t, networkExtensionClient(cloud))
				if mode == "break" {
					seen := 0
					for v, err := range a.Resources.List(context.Background(), options...) {
						if err != nil || v == nil || v.ID != "first" {
							t.Fatal(v, err)
						}
						seen++
						break
					}
					if seen != 1 || calls.Load() != 1 || followed.Load() != 0 {
						t.Fatal(seen, calls.Load(), followed.Load())
					}
					return
				}
				v, err := a.Resources.All(context.Background(), options...)
				switch mode {
				case "cap-filtered":
					trunkListWant(t, v, err)
				case "cap-null", "single-page":
					trunkListWant(t, v, err, "first")
				case "empty-object", "query-null":
					trunkListWant(t, v, err, "")
				default:
					if v != nil || err == nil {
						t.Fatal("All hid a terminal error or returned partial rows", v, err)
					}
					if nativeFailure {
						var typed *json.UnmarshalTypeError
						var date *time.ParseError
						if !errors.As(err, &typed) && !errors.As(err, &date) {
							t.Fatal("native cause lost", err)
						}
					} else if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				}
				if calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal(calls.Load(), followed.Load())
				}
			})
		}
	}
}

func TestTrunkListFiltersPagingLiveSourceAndNativeStatus(t *testing.T) {
	for _, f := range trunkListFixtures() {
		for _, mode := range []string{"complete", "first-filtered", "duplicates", "late-http", "late-decode", "cycle", "foreign", "cancel", "wrong-links", "malformed-link", "204", "json-204", "201"} {
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
					response, err := base.RoundTrip(r)
					if r.URL.Query().Get("marker") == "" {
						c.ProviderClient.SetToken("later-token")
						if mode == "cancel" {
							cancel()
						}
					}
					return response, err
				})
				cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("X-Configured") != "kept" || r.Header.Get("OpenStack-API-Version") != "network 2.0" || !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "tenant_id"}) || r.URL.Query().Get("admin_state_up") != "false" || r.URL.Query().Get("status") != "vendor-status" {
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
							testcloud.JSON(w, 200, trunkListPage(trunkListRow(t, "invalid", "Different", map[string]json.RawMessage{"sub_ports": json.RawMessage(`[{"segmentation_id":"101"}]`)}), ""))
							return
						}
						id := "second"
						if mode == "duplicates" {
							id = "first"
						}
						testcloud.JSON(w, 200, trunkListPage(trunkListRow(t, id, "Different", nil), ""))
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
					next := cloud.Server.URL + trunkListPath + "?marker=next&admin_state_up=false&fields=id&fields=tenant_id&status=vendor-status"
					if mode == "cycle" {
						next = cloud.Server.URL + r.URL.String()
					}
					if mode == "foreign" {
						next = foreign.Server.URL + "/foreign"
					}
					fields := map[string]json.RawMessage{}
					if mode == "first-filtered" {
						fields["tenant_id"] = json.RawMessage(`"other-tenant"`)
					}
					row := trunkListRow(t, "first", "Different", fields)
					if mode == "wrong-links" {
						testcloud.JSON(w, 300, `{"trunks":[`+row+`],"trunks_links":[{"rel":"next","href":`+fmt.Sprintf("%q", next)+`}],"next":`+fmt.Sprintf("%q", next)+`}`)
						return
					}
					if mode == "malformed-link" {
						testcloud.JSON(w, 300, `{"trunks":[`+row+`],"links":{"next":false}}`)
						return
					}
					testcloud.JSON(w, 300, trunkListPage(row, next))
				})
				a := f.open(t, c)
				v, err := a.Resources.All(ctx, resource.WithFilter("tenant_id", "native-tenant"), resource.WithFilter("is_admin_state_up", false), resource.WithFilter("fields", []string{"id", "tenant_id"}), resource.WithQuery("status", "vendor-status"))
				switch mode {
				case "complete":
					trunkListWant(t, v, err, "first", "second")
				case "first-filtered":
					trunkListWant(t, v, err, "second")
				case "duplicates":
					trunkListWant(t, v, err, "first", "first")
				case "wrong-links":
					trunkListWant(t, v, err, "first")
				case "204":
					trunkListWant(t, v, err)
				default:
					if v != nil || err == nil {
						t.Fatal("All hid terminal page error", v, err)
					}
					if mode == "cycle" && !errors.Is(err, resource.ErrPaginationCycle) || mode == "cancel" && !errors.Is(err, context.Canceled) || mode == "json-204" && !errors.Is(err, io.EOF) || mode == "201" && !gophercloud.ResponseCodeIs(err, 201) {
						t.Fatal(mode, err)
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

func TestTrunkListFiltersNativeAndIdentitySurfaceIsolation(t *testing.T) {
	for _, f := range trunkListFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets, actions atomic.Int32
			expected := url.Values{"id": {"wire-id"}, "tenant_id": {"wire-tenant"}, "project_id": {"wire-project"}, "status": {"wire-status"}, "sub_ports": {"wire-subports"}, "revision_number": {"vendor-revision"}, "vendor": {"kept"}}
			cloud.Mux.HandleFunc("GET "+trunkListPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), expected) {
					t.Error(r.URL.Query(), expected)
				}
				rows := trunkListRow(t, "target", "Target", map[string]json.RawMessage{"status": json.RawMessage(`"aCtIvE"`)}) + "," + trunkListRow(t, "other", "Other", map[string]json.RawMessage{"tenant_id": json.RawMessage(`"other-tenant"`), "status": json.RawMessage(`"BUILD"`)})
				testcloud.JSON(w, 200, trunkListPage(rows, ""))
			})
			cloud.Mux.HandleFunc("GET "+trunkListPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				want := url.Values{}
				if gets.Load() == 2 {
					want = url.Values{"tenant_id": {"wire-tenant"}, "status": {"wire-status"}}
				}
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				testcloud.JSON(w, 200, `{"trunk":`+trunkListRow(t, "lookup", "Different", nil)+`}`)
			})
			for _, operation := range []string{"get_subports", "add_subports", "remove_subports"} {
				method := "PUT"
				if operation == "get_subports" {
					method = "GET"
				}
				cloud.Mux.HandleFunc(method+" "+trunkListPath+"/lookup/"+operation, func(w http.ResponseWriter, r *http.Request) {
					actions.Add(1)
					if r.URL.RawQuery != "" {
						t.Error(r.URL)
					}
					if operation == "get_subports" {
						testcloud.JSON(w, 200, `{"sub_ports":[{"port_id":"child","segmentation_id":101,"segmentation_type":"vlan"}]}`)
						return
					}
					body, err := io.ReadAll(r.Body)
					want := `{"sub_ports":[{"segmentation_id":101,"segmentation_type":"vlan","port_id":"child"}]}`
					if operation == "remove_subports" {
						want = `{"sub_ports":[{"port_id":"child"}]}`
					}
					var actualJSON, wantJSON any
					if err != nil || json.Unmarshal(body, &actualJSON) != nil || json.Unmarshal([]byte(want), &wantJSON) != nil || !reflect.DeepEqual(actualJSON, wantJSON) {
						t.Error(string(body), err)
					}
					// Native update-subports extraction reads a root Trunk, not {trunk:...}.
					testcloud.JSON(w, 200, trunkListRow(t, "lookup", "Different", nil))
				})
			}
			a := f.open(t, networkExtensionClient(cloud))
			var raw []resource.ListOption
			for k, values := range expected {
				raw = append(raw, resource.WithQuery(k, values[0]))
			}
			v, err := a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilter("id", "target"), resource.WithFilter("tenant_id", "native-tenant"))...)
			trunkListWant(t, v, err, "target")
			v, err = a.Resources.All(context.Background(), append(append([]resource.ListOption(nil), raw...), resource.WithFilters(nil))...)
			trunkListWant(t, v, err, "target", "other")
			up := false
			expected = url.Values{"name": {"typed-native"}, "admin_state_up": {"false"}, "tenant_id": {"wire-tenant"}, "revision_number": {"9"}}
			var ids []string
			for row, err := range a.List(context.Background(), trunks.WithListOptions(trunks.ListOpts{Name: "typed-native", AdminStateUp: &up, TenantID: "wire-tenant", RevisionNumber: "9"})) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, row.ID)
			}
			if !reflect.DeepEqual(ids, []string{"target", "other"}) {
				t.Fatal(ids)
			}
			found, err := a.Get(context.Background(), "lookup")
			if err != nil || found == nil || found.ID != "lookup" {
				t.Fatal(found, err)
			}
			found, err = a.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindQuery("tenant_id", "wire-tenant"), resource.WithIdentityFindQuery("status", "wire-status"))
			if err != nil || found == nil || found.ID != "lookup" || gets.Load() != 2 {
				t.Fatal(found, err, gets.Load())
			}
			subports, err := a.GetSubports(context.Background(), "lookup")
			if err != nil || len(subports) != 1 || subports[0].SegmentationID != 101 {
				t.Fatal(subports, err)
			}
			found, err = a.AddSubports(context.Background(), "lookup", trunks.AddSubportsOpts{Subports: subports})
			if err != nil || found == nil || found.ID != "lookup" {
				t.Fatal(found, err)
			}
			found, err = a.RemoveSubports(context.Background(), "lookup", trunks.RemoveSubportsOpts{Subports: []trunks.RemoveSubport{{PortID: "child"}}})
			if err != nil || found == nil || found.ID != "lookup" || actions.Load() != 3 {
				t.Fatal(found, err, actions.Load())
			}
			expected = url.Values{"name": {"Target"}, "status": {"ACTIVE"}}
			v, err = a.Resources.All(context.Background(), resource.WithBodyFilter("tenant_id", "native-tenant"), resource.WithName("Target"), resource.WithStatus("earlier-status"), resource.WithQuery("status", "ACTIVE"))
			trunkListWant(t, v, err, "target")
			if lists.Load() != 4 {
				t.Fatal(lists.Load())
			}
		})
	}
}
