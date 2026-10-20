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

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const containerListPath = "/reverse/barbican/v1/containers"

type containerListFixture struct {
	name string
	open func(*testing.T, *testcloud.Cloud) *containers.API
}

func containerListFixtures() []containerListFixture {
	return []containerListFixture{
		{"leaf", func(t *testing.T, c *testcloud.Cloud) *containers.API { return containers.New(secretFetchClient(c)) }},
		{"cached connection", func(t *testing.T, c *testcloud.Cloud) *containers.API {
			conn, err := sdk.FromProvider(c.Provider, sdk.WithEndpoint(sdk.KeyManager, c.Server.URL+"/catalog/v1/"))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.KeyManagerV1(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			again, err := conn.KeyManager(context.Background())
			if err != nil || again != service || service.Containers.RawClient() != service.RawClient() || service.RawClient().ProviderClient != c.Provider {
				t.Fatal(service, again, err)
			}
			service.RawClient().ResourceBase = c.Server.URL + "/reverse/barbican/v1/"
			return service.Containers
		}},
	}
}

func containerListRow(t *testing.T, name string, fields map[string]any) string {
	t.Helper()
	row := map[string]any{"name": name, "container_ref": "https://foreign.invalid/containers/" + name, "status": "ACTIVE", "type": "generic"}
	for k, v := range fields {
		row[k] = v
	}
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func containerListPage(rows, next string) string {
	b, _ := json.Marshal(next)
	return `{"containers":[` + rows + `],"next":` + string(b) + `}`
}

func containerListWant(t *testing.T, values []*containers.Container, err error, names ...string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]string, len(values))
	for i, v := range values {
		if v == nil {
			t.Fatal("nil native model", values)
		}
		actual[i] = v.Name
	}
	if !reflect.DeepEqual(actual, append([]string{}, names...)) {
		t.Fatal(actual, names)
	}
}

// Independent source classification: container.py:17–48 + inherited Resource
// id and QueryParameters limit/marker. Native ListOpts.Name is a distinct API.
func TestKeyManagerContainerListFiltersEntireDescriptor(t *testing.T) {
	fields := map[string]any{
		"id": "literal-id", "name": "target", "container_ref": "https://foreign.invalid/containers/raw%2Fid?ignored=yes#fragment",
		"created": "2026-10-02T03:04:05", "updated": "2026-10-02T03:04:06", "status": "ACTIVE", "type": "generic",
		"secret_refs": []any{map[string]any{"name": "secret", "secret_ref": "https://foreign.invalid/secrets/s", "vendor": map[string]any{"n": json.Number("9007199254740993")}}},
		"consumers":   []any{map[string]any{"name": "consumer", "url": "https://foreign.invalid/consumer/c", "vendor": true}},
	}
	filters := map[string]any{"id": fields["id"], "name": fields["name"], "container_ref": fields["container_ref"], "container_id": "raw%2Fid", "created_at": fields["created"], "updated_at": fields["updated"], "secret_refs": fields["secret_refs"], "consumers": fields["consumers"], "status": fields["status"], "type": fields["type"]}
	if len(filters) != 10 {
		t.Fatal(len(filters))
	}
	for _, f := range containerListFixtures() {
		for field, filter := range filters {
			t.Run(f.name+"/"+field, func(t *testing.T) {
				c := testcloud.New(t)
				var calls atomic.Int32
				decoy := map[string]any{}
				for k, v := range fields {
					decoy[k] = v
				}
				switch field {
				case "container_id":
					decoy["container_ref"] = "https://foreign.invalid/containers/other"
				case "created_at":
					decoy["created"] = "2026-10-02T03:04:04"
				case "updated_at":
					decoy["updated"] = "2026-10-02T03:04:04"
				case "secret_refs", "consumers":
					decoy[field] = []any{}
				default:
					decoy[field] = "other"
				}
				// Keep the target's typed name; use a different display name only when
				// the tested local property is not itself name.
				decoyName := "decoy"
				if field != "name" {
					decoy["name"] = "decoy"
				}
				c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if len(r.URL.Query()) != 0 {
						t.Error("Body selector leaked", field, r.URL)
					}
					testcloud.JSON(w, 200, containerListPage(containerListRow(t, decoyName, decoy)+","+containerListRow(t, "target", fields), ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter(field, filter))
				containerListWant(t, values, err, "target")
				if values[0].Created.Year() != 2026 || len(values[0].SecretRefs) != 1 || values[0].SecretRefs[0].Name != "secret" || calls.Load() != 1 {
					t.Fatal(values, calls.Load())
				}
			})
		}
		for _, key := range []string{"limit", "marker"} {
			t.Run(f.name+"/query "+key, func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), url.Values{key: {"one", "two"}}) {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, containerListPage(containerListRow(t, "unrelated", nil), ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter(key, []any{"one", "two"}))
				containerListWant(t, values, err, "unrelated")
			})
		}
	}
}

func TestKeyManagerContainerListFiltersPassiveIDsAndRawJSON(t *testing.T) {
	for _, f := range containerListFixtures() {
		for _, tc := range []struct {
			name, field    string
			fields         map[string]any
			filter         any
			match, invalid bool
		}{
			{"full ref ID", "id", map[string]any{"container_ref": "https://foreign.invalid/containers/ref"}, "https://foreign.invalid/containers/ref", true, false},
			{"no derived ID", "id", map[string]any{"container_ref": "https://foreign.invalid/containers/ref"}, "ref", false, false},
			{"literal null wins", "id", map[string]any{"id": nil}, nil, true, false},
			{"null blocks alternate", "id", map[string]any{"id": nil, "container_ref": "https://foreign.invalid/containers/ref"}, "https://foreign.invalid/containers/ref", false, false},
			{"large raw ID", "id", map[string]any{"id": json.Number("9007199254740993")}, json.Number("9007199254740993"), true, false},
			{"bool differs number", "id", map[string]any{"id": true}, 1, false, false},
			{"ID subset", "id", map[string]any{"id": map[string]any{"n": 1, "extra": true}}, map[string]any{"n": json.Number("1.0")}, true, false},
			{"empty actual object", "id", map[string]any{"id": map[string]any{}}, map[string]any{}, false, false},
			{"empty filter nonempty object", "id", map[string]any{"id": map[string]any{"n": 1}}, map[string]any{}, true, false},
			{"percent preserved", "container_id", map[string]any{"container_ref": "https://foreign.invalid/containers/raw%2Fid?ignore=yes#fragment"}, "raw%2Fid", true, false},
			{"literal Unicode space", "container_id", map[string]any{"container_ref": "https://foreign.invalid/containers/한 글"}, "한 글", true, false},
			{"trailing empty", "container_id", map[string]any{"container_ref": "https://foreign.invalid/containers/"}, "", true, false},
			{"null formatter bypass", "container_id", map[string]any{"container_ref": nil}, nil, true, false},
			{"selected invalid", "container_id", map[string]any{"container_ref": "relative/ref"}, "ref", false, true},
			{"Go parser boundary", "container_id", map[string]any{"container_ref": "https://host:notport/ref"}, "ref", false, true},
			{"unselected invalid", "status", map[string]any{"container_ref": "relative/ref"}, "ACTIVE", true, false},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, containerListPage(containerListRow(t, "target", tc.fields), ""))
				})
				opts := []resource.ListOption{resource.WithFilter(tc.field, tc.filter)}
				if tc.invalid {
					opts = append(opts, resource.WithName("other"))
				}
				values, err := f.open(t, c).Resources.All(context.Background(), opts...)
				if tc.invalid {
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(values, err)
					}
				} else if tc.match {
					containerListWant(t, values, err, "target")
				} else {
					containerListWant(t, values, err)
				}
			})
		}
		for _, key := range []string{"secret_refs", "consumers"} {
			for _, tc := range []struct {
				name           string
				actual, filter any
				present, match bool
			}{
				{"missing null", nil, nil, false, true}, {"null", nil, nil, true, true}, {"empty distinct", []any{}, nil, true, false}, {"empty exact", []any{}, []any{}, true, true},
				{"null element", []any{nil}, []any{nil}, true, true},
				{"ordered exact", []any{map[string]any{"name": "first", "vendor": json.Number("9007199254740993")}, map[string]any{"name": "second"}}, []any{map[string]any{"name": "first", "vendor": json.Number("9007199254740993")}, map[string]any{"name": "second"}}, true, true},
				{"order differs", []any{map[string]any{"name": "first"}, map[string]any{"name": "second"}}, []any{map[string]any{"name": "second"}, map[string]any{"name": "first"}}, true, false},
				{"nested object exact", []any{map[string]any{"name": "first", "vendor": true}}, []any{map[string]any{"name": "first"}}, true, false},
				{"no float rounding", []any{map[string]any{"vendor": json.Number("9007199254740993")}}, []any{map[string]any{"vendor": json.Number("9007199254740992")}}, true, false},
				{"object filter array mismatch", []any{map[string]any{"name": "first"}}, map[string]any{}, true, false},
			} {
				t.Run(f.name+"/"+key+"/"+tc.name, func(t *testing.T) {
					c := testcloud.New(t)
					fields := map[string]any{}
					if tc.present {
						fields[key] = tc.actual
					}
					c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
						testcloud.JSON(w, 200, containerListPage(containerListRow(t, "target", fields), ""))
					})
					values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter(key, tc.filter))
					if tc.match {
						containerListWant(t, values, err, "target")
					} else {
						containerListWant(t, values, err)
					}
				})
			}
		}
	}
}

func TestKeyManagerContainerListFiltersLocalNameAndQueryIsolation(t *testing.T) {
	for _, f := range containerListFixtures() {
		for _, tc := range []struct {
			name  string
			opts  []resource.ListOption
			query url.Values
			want  []string
		}{
			{"semantic name local", []resource.ListOption{resource.WithFilter("name", "target")}, url.Values{}, []string{"target", "target"}},
			{"explicit name server hint", []resource.ListOption{resource.WithName("target"), resource.WithFilter("type", "generic")}, url.Values{"name": {"target"}}, []string{"target", "target"}},
			{"raw name independent", []resource.ListOption{resource.WithQuery("name", "server-wildcard"), resource.WithFilter("name", "target")}, url.Values{"name": {"server-wildcard"}}, []string{"target", "target"}},
			{"raw status ignored", []resource.ListOption{resource.WithQuery("status", "vendor"), resource.WithFilter("type", "generic")}, url.Values{}, []string{"target", "other", "target"}},
			{"local status AND", []resource.ListOption{resource.WithStatus("active"), resource.WithName("target"), resource.WithFilter("type", "generic")}, url.Values{"name": {"target"}}, []string{"target"}},
			{"Body status exact", []resource.ListOption{resource.WithFilter("status", "active")}, url.Values{}, nil},
			{"unknown discard invalid", []resource.ListOption{resource.WithFilters(map[string]any{"algorithm": math.NaN(), "offset": make(chan int)})}, url.Values{}, []string{"target", "other", "target"}},
			{"clear leaves raw and Body", []resource.ListOption{resource.WithFilter("secret_refs", make(chan int)), resource.WithFilters(nil), resource.WithQuery("vendor", "raw"), resource.WithBodyFilter("type", "generic")}, url.Values{"vendor": {"raw"}}, []string{"target", "other", "target"}},
			{"last valid wins", []resource.ListOption{resource.WithFilter("name", make(chan int)), resource.WithFilter("name", "target")}, url.Values{}, []string{"target", "target"}},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), tc.query) {
						t.Error(r.URL.Query(), tc.query)
					}
					testcloud.JSON(w, 200, containerListPage(containerListRow(t, "target", nil)+","+containerListRow(t, "other", nil)+","+containerListRow(t, "target", map[string]any{"status": "BUILD"}), ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), tc.opts...)
				containerListWant(t, values, err, tc.want...)
			})
		}
	}
}

func TestKeyManagerContainerListFiltersSnapshotsAndLazyPreflight(t *testing.T) {
	for _, f := range containerListFixtures() {
		t.Run(f.name+"/concurrent owned snapshots", func(t *testing.T) {
			c := testcloud.New(t)
			nested := map[string]any{"name": "secret", "vendor": json.Number("9007199254740993")}
			refs := []any{nested}
			markers := []any{"one", "two"}
			input := map[string]any{"secret_refs": refs, "marker": markers}
			option := resource.WithFilters(input)
			nested["vendor"] = 0
			refs[0] = nil
			markers[0] = "caller"
			input["marker"] = "caller"
			var calls atomic.Int32
			c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["marker"], []string{"one", "two"}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, containerListPage(containerListRow(t, "target", map[string]any{"secret_refs": json.RawMessage(`[{"name":"secret","vendor":9007199254740993}]`)}), ""))
			})
			api := f.open(t, c)
			var wg sync.WaitGroup
			for range 8 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					values, err := api.Resources.All(context.Background(), option)
					if err != nil || len(values) != 1 || values[0].Name != "target" {
						t.Error(values, err)
					}
				}()
			}
			wg.Wait()
			if calls.Load() != 8 {
				t.Fatal(calls.Load())
			}
		})
		t.Run(f.name+"/lazy invalid options", func(t *testing.T) {
			c := testcloud.New(t)
			var calls atomic.Int32
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, containerListPage("", ""))
			})
			api := f.open(t, c)
			invalid := [][]resource.ListOption{
				{resource.WithFilter("limit", 1), resource.WithQuery("limit", "1")}, {resource.WithFilter("limit", 1), resource.WithPageSize(1)},
				{resource.WithFilter("marker", nil), resource.WithQuery("marker", "1")}, {resource.WithFilter("name", "target"), resource.WithBodyFilter("name", "target")},
				{resource.WithFilter("secret_refs", make(chan int))}, {resource.WithFilter("limit", map[string]any{"bad": 1})},
				{resource.WithBodyFilter("created", "not attribute")}, {resource.WithBodyFilter("consumers", make(chan int)), resource.WithBodyFilters(nil)},
				{resource.WithMaxItems(-1), resource.WithFilter("name", "target")},
			}
			for _, key := range []string{"max_items", "paginated", "base_path", "allow_unknown_params", "headers", "microversion", "jmespath_filters", "resource_type", "session"} {
				invalid = append(invalid, []resource.ListOption{resource.WithFilter(key, "value")})
			}
			for _, opts := range invalid {
				seq := api.Resources.List(context.Background(), opts...)
				if calls.Load() != 0 {
					t.Fatal("eagerHTTP")
				}
				observations := 0
				for value, err := range seq {
					observations++
					if value != nil || !errors.Is(err, resource.ErrInvalidOption) && !errors.Is(err, resource.ErrUnsupported) {
						t.Fatal(value, err)
					}
				}
				if observations != 1 || calls.Load() != 0 {
					t.Fatal(observations, calls.Load())
				}
			}
			if _, err := servers.New(api.RawClient()).Resources.All(context.Background(), resource.WithFilters(nil)); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(err)
			}
		})
	}
}

func TestKeyManagerContainerListFiltersControlsAndWholePageFailures(t *testing.T) {
	for _, f := range containerListFixtures() {
		for _, mode := range []string{"cap before filter", "first page", "break", "null row", "query only null", "late null", "cap null", "cap formatter", "empty object"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				c, foreign := testcloud.New(t), testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
				first := containerListRow(t, "first", nil)
				rows := first + "," + containerListRow(t, "second", nil)
				switch mode {
				case "null row", "query only null":
					rows = `null`
				case "late null", "cap null":
					rows = first + `,null`
				case "cap formatter":
					rows = first + "," + containerListRow(t, "second", map[string]any{"container_ref": "relative/ref"})
				case "empty object":
					rows = `{}`
				}
				c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 200, containerListPage(rows, foreign.Server.URL+"/never"))
				})
				api := f.open(t, c)
				opts := []resource.ListOption{resource.WithFilter("secret_refs", nil)}
				switch mode {
				case "query only null":
					opts = []resource.ListOption{resource.WithFilter("limit", 1), resource.WithPaginated(false)}
				case "cap before filter":
					opts = append(opts, resource.WithMaxItems(1), resource.WithName("second"))
				case "first page", "empty object":
					opts = append(opts, resource.WithPaginated(false))
				case "cap null":
					opts = append(opts, resource.WithMaxItems(1))
				case "cap formatter":
					opts = []resource.ListOption{resource.WithFilter("container_id", "first"), resource.WithMaxItems(1)}
				}
				if mode == "break" {
					n := 0
					for v, err := range api.Resources.List(context.Background(), opts...) {
						if err != nil || v.Name != "first" {
							t.Fatal(v, err)
						}
						n++
						break
					}
					if n != 1 {
						t.Fatal(n)
					}
				} else {
					values, err := api.Resources.All(context.Background(), opts...)
					switch mode {
					case "null row", "late null":
						if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal(values, err)
						}
					case "cap before filter":
						containerListWant(t, values, err)
					case "first page":
						containerListWant(t, values, err, "first", "second")
					case "empty object", "query only null":
						containerListWant(t, values, err, "")
					default:
						containerListWant(t, values, err, "first")
					}
				}
				if calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal(calls.Load(), followed.Load())
				}
			})
		}
		for _, bad := range []string{`{"name":false}`, `{"status":{}}`, `{"type":false}`, `{"container_ref":false}`, `{"secret_refs":{}}`, `{"secret_refs":[{"name":false}]}`, `{"consumers":"scalar"}`, `{"consumers":[{"url":false}]}`, `{"created":"2026-10-02T03:04:05Z"}`, `{"creator_id":false}`} {
			t.Run(f.name+"/native known field "+bad, func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, containerListPage(containerListRow(t, "first", nil)+","+bad, ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter("secret_refs", nil), resource.WithMaxItems(1))
				if values != nil || err == nil {
					t.Fatal(values, err)
				}
			})
		}
	}
}

func TestKeyManagerContainerListFiltersPagingAndLiveSource(t *testing.T) {
	for _, f := range containerListFixtures() {
		for _, mode := range []string{"complete", "duplicate rows", "late404", "late decode", "cycle", "foreign", "cancel"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				c, foreign := testcloud.New(t), testcloud.New(t)
				api := f.open(t, c)
				api.RawClient().MoreHeaders = map[string]string{"X-Configured": "preserved"}
				api.RawClient().Microversion = "1.0"
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var calls, followed, middleware atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
				transport := c.Provider.HTTPClient.Transport
				if transport == nil {
					transport = http.DefaultTransport
				}
				c.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					middleware.Add(1)
					resp, err := transport.RoundTrip(r)
					if !r.URL.Query().Has("offset") {
						c.Provider.SetToken("later-token")
						if mode == "cancel" {
							cancel()
						}
					}
					return resp, err
				})
				c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query()["marker"], []string{"one", "two"}) || r.Header.Get("X-Configured") != "preserved" || r.Header.Get("OpenStack-API-Version") != "key-manager 1.0" {
						t.Error(r.URL, r.Header)
					}
					if !r.URL.Query().Has("offset") {
						next := c.Server.URL + containerListPath + "?offset=2&marker=one&marker=two"
						if mode == "foreign" {
							next = foreign.Server.URL + containerListPath + "?offset=2&marker=one&marker=two"
						}
						testcloud.JSON(w, 200, containerListPage(containerListRow(t, "first", nil), next))
						return
					}
					if r.Header.Get("X-Auth-Token") != "later-token" {
						t.Error(r.Header)
					}
					if mode == "late404" {
						testcloud.JSON(w, 404, `{"error":"late"}`)
						return
					}
					rows, next := containerListRow(t, "second", nil), ""
					if mode == "duplicate rows" {
						rows = containerListRow(t, "first", nil)
					}
					if mode == "late decode" {
						rows = `{"secret_refs":true}`
					}
					if mode == "cycle" {
						next = c.Server.URL + containerListPath + "?offset=2&marker=one&marker=two"
					}
					testcloud.JSON(w, 200, containerListPage(rows, next))
				})
				values, err := api.Resources.All(ctx, resource.WithFilter("marker", []any{"one", "two"}), resource.WithFilter("type", "generic"))
				switch mode {
				case "complete":
					containerListWant(t, values, err, "first", "second")
				case "duplicate rows":
					containerListWant(t, values, err, "first", "first")
				default:
					if values != nil || err == nil {
						t.Fatal(values, err)
					}
				}
				if mode == "cycle" && !errors.Is(err, resource.ErrPaginationCycle) {
					t.Fatal(err)
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				wantCalls, wantMiddleware, wantFollowed := int32(2), int32(2), int32(0)
				if mode == "cancel" {
					wantCalls, wantMiddleware = 1, 1
				}
				if mode == "foreign" {
					wantCalls, wantFollowed = 1, 1
				}
				if calls.Load() != wantCalls || middleware.Load() != wantMiddleware || followed.Load() != wantFollowed || api.RawClient().ProviderClient != c.Provider {
					t.Fatal(calls.Load(), middleware.Load(), followed.Load())
				}
			})
		}
	}
}

func TestKeyManagerContainerListFiltersNativeSurfaceAndStatusPolicy(t *testing.T) {
	for _, f := range containerListFixtures() {
		for _, code := range []int{200, 204, 300, 201} {
			t.Run(fmt.Sprintf("%s/native page status %d", f.name, code), func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
					if code == 204 {
						w.WriteHeader(204)
						return
					}
					testcloud.JSON(w, code, containerListPage(containerListRow(t, "native", nil), ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter("secret_refs", nil))
				switch code {
				case 200, 300:
					containerListWant(t, values, err, "native")
				case 204:
					containerListWant(t, values, err)
				default:
					if values != nil || !gophercloud.ResponseCodeIs(err, code) {
						t.Fatal(values, err)
					}
				}
			})
		}
		t.Run(f.name+"/JSON204 EOF", func(t *testing.T) {
			c := testcloud.New(t)
			c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(204)
			})
			values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter("secret_refs", nil))
			if values != nil || !errors.Is(err, io.EOF) {
				t.Fatal(values, err)
			}
		})
		t.Run(f.name+"/typed List Get and Ref unchanged", func(t *testing.T) {
			c := testcloud.New(t)
			var lists, gets atomic.Int32
			c.Mux.HandleFunc(containerListPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "server-hint" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, containerListPage(containerListRow(t, "one", nil)+","+containerListRow(t, "two", nil), ""))
			})
			c.Mux.HandleFunc(containerListPath+"/container-alpha", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, containerListRow(t, "direct", nil))
			})
			api := f.open(t, c)
			n := 0
			for value, err := range api.List(context.Background(), containers.WithListOptions(containers.ListOpts{Name: "server-hint"})) {
				if value == nil || err != nil {
					t.Fatal(value, err)
				}
				n++
			}
			if n != 2 || lists.Load() != 1 {
				t.Fatal(n, lists.Load())
			}
			value, err := api.Get(context.Background(), "container-alpha")
			if err != nil || value.Name != "direct" {
				t.Fatal(value, err)
			}
			value, err = api.Resources.Find(context.Background(), resource.ID("container-alpha"))
			if err != nil || value.Name != "direct" || gets.Load() != 2 {
				t.Fatal(value, err, gets.Load())
			}
			if _, err := api.Resources.FindIdentity(context.Background(), "container-alpha"); !errors.Is(err, resource.ErrUnsupported) || gets.Load() != 2 || lists.Load() != 1 {
				t.Fatal(err, gets.Load(), lists.Load())
			}
		})
	}
}
