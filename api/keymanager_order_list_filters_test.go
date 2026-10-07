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

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/servers"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/orders"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func orderListRow(t *testing.T, tag string, fields map[string]any) string {
	t.Helper()
	row := map[string]any{"order_ref": "https://foreign.invalid/orders/" + tag, "secret_ref": "https://foreign.invalid/secrets/secret-" + tag, "sub_status": tag, "status": "ACTIVE", "type": "key"}
	for k, v := range fields {
		row[k] = v
	}
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func orderListPage(rows, next string) string {
	b, _ := json.Marshal(next)
	return `{"orders":[` + rows + `],"next":` + string(b) + `}`
}
func orderListWant(t *testing.T, values []*orders.Order, err error, tags ...string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]string, len(values))
	for i, v := range values {
		if v == nil {
			t.Fatal("nil native model", values)
		}
		actual[i] = v.SubStatus
	}
	if !reflect.DeepEqual(actual, append([]string{}, tags...)) {
		t.Fatal(actual, tags)
	}
}

// Independent classification: order.py:17–58 + inherited id/name and
// QueryParameters limit/marker. Meta.Name is not a top-level resource name.
func TestKeyManagerOrderListFiltersEntireDescriptor(t *testing.T) {
	fields := map[string]any{"id": "literal-id", "name": "top-name", "created": "2026-10-02T03:04:05", "updated": "2026-10-02T03:04:06", "creator_id": "creator", "order_ref": "https://foreign.invalid/orders/order%2Fid?ignore=yes#fragment", "secret_ref": "https://foreign.invalid/secrets/secret%2Fid?ignore=yes#fragment", "status": "ACTIVE", "sub_status": "target", "sub_status_message": "running", "type": "key", "meta": map[string]any{"name": "nested-name", "algorithm": "aes", "bit_length": 256, "expiration": "2027-10-02T03:04:05", "mode": "cbc", "payload_content_type": "application/octet-stream", "vendor": map[string]any{"n": json.Number("9007199254740993"), "extra": true}}}
	filters := map[string]any{"id": fields["id"], "name": fields["name"], "created_at": fields["created"], "updated_at": fields["updated"], "creator_id": fields["creator_id"], "order_ref": fields["order_ref"], "order_id": "order%2Fid", "secret_ref": fields["secret_ref"], "secret_id": "secret%2Fid", "status": fields["status"], "sub_status": fields["sub_status"], "sub_status_message": fields["sub_status_message"], "type": fields["type"], "meta": map[string]any{"vendor": map[string]any{"n": json.Number("9007199254740993")}}}
	if len(filters) != 14 {
		t.Fatal(len(filters))
	}
	for _, f := range orderListFixtures() {
		for key, filter := range filters {
			t.Run(f.name+"/"+key, func(t *testing.T) {
				c := testcloud.New(t)
				decoy := map[string]any{}
				for k, v := range fields {
					decoy[k] = v
				}
				decoy["sub_status"] = "decoy"
				switch key {
				case "order_id":
					decoy["order_ref"] = "https://foreign.invalid/orders/other"
				case "secret_id":
					decoy["secret_ref"] = "https://foreign.invalid/secrets/other"
				case "created_at":
					decoy["created"] = "2026-10-02T03:04:04"
				case "updated_at":
					decoy["updated"] = "2026-10-02T03:04:04"
				case "meta":
					decoy["meta"] = map[string]any{"name": "other"}
				default:
					decoy[key] = "other"
				}
				var calls atomic.Int32
				c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if len(r.URL.Query()) != 0 {
						t.Error("Body leaked", key, r.URL)
					}
					testcloud.JSON(w, 200, orderListPage(orderListRow(t, "decoy", decoy)+","+orderListRow(t, "target", fields), ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter(key, filter))
				orderListWant(t, values, err, "target")
				if values[0].Meta.Name != "nested-name" || values[0].Meta.BitLength != 256 || values[0].Meta.Expiration.Year() != 2027 || calls.Load() != 1 {
					t.Fatal(values, calls.Load())
				}
			})
		}
		for _, key := range []string{"limit", "marker"} {
			t.Run(f.name+"/query "+key, func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), url.Values{key: {"one", "two"}}) {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, orderListPage(orderListRow(t, "unrelated", nil), ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter(key, []any{"one", "two"}))
				orderListWant(t, values, err, "unrelated")
			})
		}
	}
}

func TestKeyManagerOrderListFiltersPassiveIdentityAndRawName(t *testing.T) {
	for _, f := range orderListFixtures() {
		for _, tc := range []struct {
			name, key      string
			fields         map[string]any
			filter         any
			match, invalid bool
		}{
			{"full order ref", "id", map[string]any{"order_ref": "https://foreign.invalid/orders/order", "secret_ref": "https://foreign.invalid/secrets/secret"}, "https://foreign.invalid/orders/order", true, false},
			{"id is not secret ref", "id", nil, "https://foreign.invalid/secrets/secret-target", false, false},
			{"id not derived suffix", "id", nil, "target", false, false},
			{"literal null wins", "id", map[string]any{"id": nil}, nil, true, false}, {"null blocks ref", "id", map[string]any{"id": nil}, "https://foreign.invalid/orders/target", false, false},
			{"literal empty", "id", map[string]any{"id": ""}, "", true, false}, {"large raw id", "id", map[string]any{"id": json.Number("9007199254740993")}, json.Number("9007199254740993"), true, false},
			{"order path", "order_id", map[string]any{"order_ref": "https://foreign.invalid/orders/raw%2Forder?ignore=yes#fragment"}, "raw%2Forder", true, false},
			{"secret path independent", "secret_id", map[string]any{"secret_ref": "https://foreign.invalid/secrets/raw%2Fsecret"}, "raw%2Fsecret", true, false},
			{"literal Unicode space", "order_id", map[string]any{"order_ref": "https://foreign.invalid/orders/한 글"}, "한 글", true, false},
			{"order trailing slash", "order_id", map[string]any{"order_ref": "https://foreign.invalid/orders/"}, "", true, false},
			{"secret trailing slash", "secret_id", map[string]any{"secret_ref": "https://foreign.invalid/secrets/"}, "", true, false},
			{"order null formatter", "order_id", map[string]any{"order_ref": nil}, nil, true, false}, {"secret null formatter", "secret_id", map[string]any{"secret_ref": nil}, nil, true, false},
			{"selected bad order", "order_id", map[string]any{"order_ref": "relative/order"}, "order", false, true}, {"selected bad secret", "secret_id", map[string]any{"secret_ref": "https://host:notport/secret"}, "secret", false, true},
			{"unselected bad refs", "status", map[string]any{"order_ref": "relative/order", "secret_ref": "relative/secret"}, "ACTIVE", true, false},
			{"raw name exact", "name", map[string]any{"name": "raw-name", "meta": map[string]any{"name": "nested-name"}}, "raw-name", true, false},
			{"nested name not promoted", "name", map[string]any{"meta": map[string]any{"name": "nested-name"}}, "nested-name", false, false},
			{"missing raw name null", "name", map[string]any{"meta": map[string]any{"name": "nested-name"}}, nil, true, false},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, orderListPage(orderListRow(t, "target", tc.fields), ""))
				})
				opts := []resource.ListOption{resource.WithFilter(tc.key, tc.filter)}
				if tc.invalid {
					opts = append(opts, resource.WithStatus("BUILD"))
				}
				values, err := f.open(t, c).Resources.All(context.Background(), opts...)
				if tc.invalid {
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(values, err)
					}
				} else if tc.match {
					orderListWant(t, values, err, "target")
				} else {
					orderListWant(t, values, err)
				}
			})
		}
	}
}

func TestKeyManagerOrderListFiltersRawMetaAndExactJSON(t *testing.T) {
	for _, f := range orderListFixtures() {
		for _, tc := range []struct {
			name           string
			actual, filter any
			present, match bool
		}{
			{"missing null", nil, nil, false, true}, {"null", nil, nil, true, true}, {"empty actual", map[string]any{}, map[string]any{}, true, false},
			{"empty subset nonempty", map[string]any{"vendor": 1}, map[string]any{}, true, true},
			{"recursive subset", map[string]any{"vendor": map[string]any{"n": json.Number("9007199254740993"), "extra": true}, "other": 2}, map[string]any{"vendor": map[string]any{"n": json.Number("9007199254740993")}}, true, true},
			{"no float rounding", map[string]any{"vendor": json.Number("9007199254740993")}, map[string]any{"vendor": json.Number("9007199254740992")}, true, false},
			{"decimal exact", map[string]any{"vendor": json.Number("1.000")}, map[string]any{"vendor": json.Number("1e0")}, true, true},
			{"array exact", map[string]any{"vendor": []any{nil, map[string]any{"n": 1, "extra": true}}}, map[string]any{"vendor": []any{nil, map[string]any{"n": json.Number("1e0"), "extra": true}}}, true, true},
			{"array nested object not subset", map[string]any{"vendor": []any{map[string]any{"n": 1, "extra": true}}}, map[string]any{"vendor": []any{map[string]any{"n": 1}}}, true, false},
			{"array order", map[string]any{"vendor": []any{"one", "two"}}, map[string]any{"vendor": []any{"two", "one"}}, true, false},
			{"array length", map[string]any{"vendor": []any{1, 2}}, map[string]any{"vendor": []any{1}}, true, false},
			{"bool number strict", map[string]any{"vendor": true}, map[string]any{"vendor": 1}, true, false},
			{"nested object versus array", map[string]any{"vendor": []any{1}}, map[string]any{"vendor": map[string]any{}}, true, false},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				c := testcloud.New(t)
				fields := map[string]any{}
				if tc.present {
					fields["meta"] = tc.actual
				}
				c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, orderListPage(orderListRow(t, "target", fields), ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter("meta", tc.filter))
				if tc.match {
					orderListWant(t, values, err, "target")
				} else {
					orderListWant(t, values, err)
				}
			})
		}
	}
}

func TestKeyManagerOrderListFiltersOptionsQueriesAndPreflight(t *testing.T) {
	for _, f := range orderListFixtures() {
		for _, tc := range []struct {
			name  string
			opts  []resource.ListOption
			query url.Values
			tags  []string
		}{
			{"query scalar repeats", []resource.ListOption{resource.WithFilters(map[string]any{"limit": []any{"one", nil, false, json.Number("1e0")}, "marker": ""})}, url.Values{"limit": {"one", "false", "1e0"}, "marker": {""}}, []string{"target", "other"}},
			{"raw name independent", []resource.ListOption{resource.WithFilter("name", "raw-name"), resource.WithQuery("name", "server-only")}, url.Values{"name": {"server-only"}}, []string{"target"}},
			{"raw status ignored", []resource.ListOption{resource.WithFilter("creator_id", "creator"), resource.WithQuery("status", "vendor")}, url.Values{}, []string{"target", "other"}},
			{"semantic status plus common status", []resource.ListOption{resource.WithFilter("name", "raw-name"), resource.WithStatus("active")}, url.Values{}, []string{"target"}},
			{"unknown bad JSON discard", []resource.ListOption{resource.WithFilters(map[string]any{"offset": math.NaN(), "algorithm": make(chan int)})}, url.Values{}, []string{"target", "other"}},
			{"last selected valid", []resource.ListOption{resource.WithFilter("meta", make(chan int)), resource.WithFilter("meta", map[string]any{"name": "nested"})}, url.Values{}, []string{"target", "other"}},
			{"clear preserves namespaces", []resource.ListOption{resource.WithFilter("meta", make(chan int)), resource.WithFilters(nil), resource.WithQuery("offset", "2"), resource.WithBodyFilter("creator_id", "creator")}, url.Values{"offset": {"2"}}, []string{"target", "other"}},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), tc.query) {
						t.Error(r.URL.Query(), tc.query)
					}
					testcloud.JSON(w, 200, orderListPage(orderListRow(t, "target", map[string]any{"name": "raw-name", "creator_id": "creator", "meta": map[string]any{"name": "nested"}})+","+orderListRow(t, "other", map[string]any{"name": "other-name", "creator_id": "creator", "meta": map[string]any{"name": "nested"}, "status": "BUILD"}), ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), tc.opts...)
				orderListWant(t, values, err, tc.tags...)
			})
		}
		t.Run(f.name+"/snapshot concurrent reuse", func(t *testing.T) {
			c := testcloud.New(t)
			vendor := map[string]any{"n": json.Number("9007199254740993")}
			meta := map[string]any{"vendor": vendor}
			marker := []any{"one", "two"}
			input := map[string]any{"meta": meta, "marker": marker}
			option := resource.WithFilters(input)
			vendor["n"] = 0
			meta["vendor"] = nil
			marker[0] = "caller"
			input["meta"] = nil
			var calls atomic.Int32
			c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["marker"], []string{"one", "two"}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, orderListPage(orderListRow(t, "target", map[string]any{"meta": json.RawMessage(`{"vendor":{"n":9007199254740993}}`)}), ""))
			})
			api := f.open(t, c)
			var wg sync.WaitGroup
			for range 8 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					values, err := api.Resources.All(context.Background(), option)
					if err != nil || len(values) != 1 || values[0].SubStatus != "target" {
						t.Error(values, err)
					}
				}()
			}
			wg.Wait()
			if calls.Load() != 8 {
				t.Fatal(calls.Load())
			}
		})
		t.Run(f.name+"/lazy rejected namespaces", func(t *testing.T) {
			c := testcloud.New(t)
			var calls atomic.Int32
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, orderListPage("", ""))
			})
			api := f.open(t, c)
			invalid := [][]resource.ListOption{{resource.WithFilter("limit", 1), resource.WithPageSize(1)}, {resource.WithFilter("marker", nil), resource.WithQuery("marker", "1")}, {resource.WithFilter("meta", map[string]any{}), resource.WithBodyFilter("meta", map[string]any{})}, {resource.WithFilter("limit", map[string]any{})}, {resource.WithFilter("meta", make(chan int))}, {resource.WithBodyFilter("created", "not attribute")}, {resource.WithBodyFilter("meta", make(chan int)), resource.WithBodyFilters(nil)}, {resource.WithMaxItems(-1)}, {resource.WithName("nested")}}
			for _, key := range []string{"max_items", "paginated", "base_path", "allow_unknown_params", "headers", "microversion", "jmespath_filters", "resource_type", "session"} {
				invalid = append(invalid, []resource.ListOption{resource.WithFilter(key, "value")})
			}
			for _, opts := range invalid {
				seq := api.Resources.List(context.Background(), opts...)
				if calls.Load() != 0 {
					t.Fatal("eagerHTTP")
				}
				n := 0
				for value, err := range seq {
					n++
					if value != nil || !errors.Is(err, resource.ErrInvalidOption) && !errors.Is(err, resource.ErrUnsupported) {
						t.Fatal(value, err)
					}
				}
				if n != 1 || calls.Load() != 0 {
					t.Fatal(n, calls.Load())
				}
			}
			if _, err := servers.New(api.RawClient()).Resources.All(context.Background(), resource.WithFilters(nil)); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(err)
			}
		})
	}
}

func TestKeyManagerOrderListFiltersControlsAndNativeFailures(t *testing.T) {
	for _, f := range orderListFixtures() {
		for _, mode := range []string{"cap before local status", "first page", "break", "null row", "query only null", "late null", "cap null", "empty object", "cap formatter"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				c, foreign := testcloud.New(t), testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
				first := orderListRow(t, "first", nil)
				rows := first + "," + orderListRow(t, "second", map[string]any{"status": "BUILD"})
				switch mode {
				case "null row", "query only null":
					rows = `null`
				case "late null", "cap null":
					rows = first + `,null`
				case "empty object":
					rows = `{}`
				case "cap formatter":
					rows = first + "," + orderListRow(t, "second", map[string]any{"order_ref": "relative/order"})
				}
				c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Has("status") {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, orderListPage(rows, foreign.Server.URL+"/never"))
				})
				opts := []resource.ListOption{resource.WithFilter("meta", nil)}
				switch mode {
				case "cap before local status":
					opts = append(opts, resource.WithMaxItems(1), resource.WithStatus("BUILD"))
				case "first page", "empty object":
					opts = append(opts, resource.WithPaginated(false))
				case "query only null":
					opts = []resource.ListOption{resource.WithFilter("limit", 1), resource.WithPaginated(false)}
				case "cap null":
					opts = append(opts, resource.WithMaxItems(1))
				case "cap formatter":
					opts = []resource.ListOption{resource.WithFilter("order_id", "first"), resource.WithMaxItems(1)}
				}
				api := f.open(t, c)
				if mode == "break" {
					n := 0
					for value, err := range api.Resources.List(context.Background(), opts...) {
						if err != nil || value.SubStatus != "first" {
							t.Fatal(value, err)
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
					case "cap before local status":
						orderListWant(t, values, err)
					case "first page":
						orderListWant(t, values, err, "first", "second")
					case "query only null", "empty object":
						orderListWant(t, values, err, "")
					default:
						orderListWant(t, values, err, "first")
					}
				}
				if calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal(calls.Load(), followed.Load())
				}
			})
		}
		for _, bad := range []string{`{"meta":false}`, `{"meta":{"bit_length":"256"}}`, `{"meta":{"bit_length":256.0}}`, `{"meta":{"algorithm":false}}`, `{"meta":{"expiration":"2026-10-02T03:04:05Z"}}`, `{"created":"2026-10-02T03:04:05Z"}`, `{"order_ref":false}`, `{"secret_ref":false}`, `{"creator_id":false}`, `{"error_status_code":404}`, `{"status":{}}`} {
			t.Run(f.name+"/native "+bad, func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, orderListPage(orderListRow(t, "first", nil)+","+bad, ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter("meta", nil), resource.WithMaxItems(1))
				if values != nil || err == nil {
					t.Fatal(values, err)
				}
			})
		}
	}
}

func TestKeyManagerOrderListFiltersPagingAndLiveSource(t *testing.T) {
	for _, f := range orderListFixtures() {
		for _, mode := range []string{"complete", "duplicate rows", "late404", "late decode", "cycle", "foreign", "cancel", "wrong links"} {
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
				c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query()["marker"], []string{"one", "two"}) || r.Header.Get("X-Configured") != "preserved" || r.Header.Get("OpenStack-API-Version") != "key-manager 1.0" {
						t.Error(r.URL, r.Header)
					}
					if !r.URL.Query().Has("offset") {
						next := c.Server.URL + orderListPath + "?offset=2&marker=one&marker=two"
						if mode == "foreign" {
							next = foreign.Server.URL + orderListPath + "?offset=2&marker=one&marker=two"
						}
						if mode == "wrong links" {
							testcloud.JSON(w, 200, `{"orders":[`+orderListRow(t, "first", nil)+`],"links":{"next":`+fmt.Sprintf("%q", next)+`}}`)
							return
						}
						testcloud.JSON(w, 200, orderListPage(orderListRow(t, "first", nil), next))
						return
					}
					if r.Header.Get("X-Auth-Token") != "later-token" {
						t.Error(r.Header)
					}
					if mode == "late404" {
						testcloud.JSON(w, 404, `{"error":"late"}`)
						return
					}
					rows, next := orderListRow(t, "second", nil), ""
					if mode == "duplicate rows" {
						rows = orderListRow(t, "first", nil)
					}
					if mode == "late decode" {
						rows = `{"meta":{"bit_length":false}}`
					}
					if mode == "cycle" {
						next = c.Server.URL + orderListPath + "?offset=2&marker=one&marker=two"
					}
					testcloud.JSON(w, 200, orderListPage(rows, next))
				})
				values, err := api.Resources.All(ctx, resource.WithFilter("marker", []any{"one", "two"}), resource.WithFilter("type", "key"))
				switch mode {
				case "complete":
					orderListWant(t, values, err, "first", "second")
				case "duplicate rows":
					orderListWant(t, values, err, "first", "first")
				case "wrong links":
					orderListWant(t, values, err, "first")
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
				if mode == "cancel" || mode == "wrong links" {
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

func TestKeyManagerOrderListFiltersNativeStatusAndSurfaceIsolation(t *testing.T) {
	for _, f := range orderListFixtures() {
		for _, code := range []int{200, 204, 300, 201} {
			t.Run(fmt.Sprintf("%s/page status %d", f.name, code), func(t *testing.T) {
				c := testcloud.New(t)
				c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
					if code == 204 {
						w.WriteHeader(code)
						return
					}
					testcloud.JSON(w, code, orderListPage(orderListRow(t, "native", nil), ""))
				})
				values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter("meta", nil))
				switch code {
				case 200, 300:
					orderListWant(t, values, err, "native")
				case 204:
					orderListWant(t, values, err)
				default:
					if values != nil || !gophercloud.ResponseCodeIs(err, code) {
						t.Fatal(values, err)
					}
				}
			})
		}
		t.Run(f.name+"/empty JSON 204 EOF", func(t *testing.T) {
			c := testcloud.New(t)
			c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(204)
			})
			values, err := f.open(t, c).Resources.All(context.Background(), resource.WithFilter("meta", nil))
			if values != nil || !errors.Is(err, io.EOF) {
				t.Fatal(values, err)
			}
		})
		t.Run(f.name+"/native typed List unchanged", func(t *testing.T) {
			c := testcloud.New(t)
			var calls atomic.Int32
			c.Mux.HandleFunc(orderListPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("name") != "wire-extension" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, orderListPage(orderListRow(t, "one", map[string]any{"name": "unknown-top", "meta": map[string]any{"name": "nested"}})+","+orderListRow(t, "two", nil), ""))
			})
			api := f.open(t, c)
			n := 0
			for value, err := range api.List(context.Background(), orders.WithListQuery("name", "wire-extension")) {
				if value == nil || err != nil {
					t.Fatal(value, err)
				}
				n++
			}
			if n != 2 || calls.Load() != 1 {
				t.Fatal(n, calls.Load())
			}
			if _, err := api.Resources.FindIdentity(context.Background(), "order-alpha"); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
		})
	}
}
