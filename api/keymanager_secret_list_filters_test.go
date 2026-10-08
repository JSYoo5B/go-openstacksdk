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
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Independent source tables: openstacksdk ef55d7d Secret query/Body declarations
// (secret.py:35–89) and QueryParameters defaults (resource.py:295–299).
func secretSemanticQueries() map[string]string {
	return map[string]string{"acl_only": "acl_only", "algorithm": "alg", "bits": "bits", "created": "created", "expiration": "expiration", "limit": "limit", "marker": "marker", "mode": "mode", "name": "name", "secret_type": "secret_type", "sort": "sort", "updated": "updated"}
}

type secretListFixture struct {
	name string
	open func(*testing.T, *testcloud.Cloud) *secrets.API
}

func secretListFixtures() []secretListFixture {
	return []secretListFixture{
		{"leaf", func(t *testing.T, cloud *testcloud.Cloud) *secrets.API { return secrets.New(secretFetchClient(cloud)) }},
		{"cached connection", func(t *testing.T, cloud *testcloud.Cloud) *secrets.API {
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.KeyManager, cloud.Server.URL+"/catalog/v1/"))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.KeyManagerV1(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			again, err := conn.KeyManager(context.Background())
			if err != nil || again != service || service.Secrets.RawClient() != service.RawClient() || service.RawClient().ProviderClient != cloud.Provider {
				t.Fatal(service, again, err)
			}
			service.RawClient().ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
			return service.Secrets
		}},
	}
}

func secretListRow(t *testing.T, name string, fields map[string]any) string {
	t.Helper()
	row := map[string]any{"name": name, "secret_ref": "https://foreign.invalid/secrets/" + name, "status": "ACTIVE"}
	for key, value := range fields {
		row[key] = value
	}
	encoded, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func secretListWant(t *testing.T, values []*secrets.Secret, err error, names ...string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]string, len(values))
	for i, value := range values {
		if value == nil {
			t.Fatal("nil native model", values)
		}
		actual[i] = value.Name
	}
	if !reflect.DeepEqual(actual, append([]string{}, names...)) {
		t.Fatal(actual, names)
	}
}

func secretListBodyTarget() (map[string]any, map[string]any) {
	fields := map[string]any{
		"id": "literal-id", "bit_length": json.Number("9007199254740993"),
		"content_types": map[string]any{"default": "text/plain", "vendor": "preserved"},
		"created":       "2026-10-02T03:04:05", "updated": "2026-10-02T03:04:06", "expiration": "2027-10-02T03:04:05",
		"secret_ref": "https://foreign.invalid/secrets/raw%2Fid?query=ignored#fragment", "status": "ACTIVE",
		"payload":              map[string]any{"nested": map[string]any{"n": json.Number("9007199254740993"), "extra": true}},
		"payload_content_type": "text/plain", "payload_content_encoding": "base64",
	}
	filters := map[string]any{
		"id": "literal-id", "bit_length": json.Number("9007199254740993"),
		"content_types": map[string]any{"default": "text/plain"},
		"created_at":    fields["created"], "updated_at": fields["updated"], "expires_at": fields["expiration"],
		"secret_ref": fields["secret_ref"], "secret_id": "raw%2Fid", "status": "ACTIVE",
		"payload":              map[string]any{"nested": map[string]any{"n": json.Number("9007199254740993")}},
		"payload_content_type": "text/plain", "payload_content_encoding": "base64",
	}
	return fields, filters
}

func TestKeyManagerSecretListFiltersEntireQueryDescriptor(t *testing.T) {
	canonical, accepted := secretSemanticQueries(), make(map[string]string)
	for name, wire := range canonical {
		accepted[name], accepted[wire] = wire, wire
	}
	if len(canonical) != 12 || len(accepted) != 13 {
		t.Fatal(len(canonical), len(accepted))
	}
	for _, f := range secretListFixtures() {
		for field, wire := range accepted {
			t.Run(f.name+"/"+field, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{wire: {"wire-value"}}) {
						t.Error(field, r.URL)
					}
					testcloud.JSON(w, 200, secretFindPage(secretListRow(t, "server-returned", nil), ""))
				})
				values, err := f.open(t, cloud).Resources.All(context.Background(), resource.WithFilter(field, "wire-value"))
				secretListWant(t, values, err, "server-returned")
				if calls.Load() != 1 {
					t.Fatal(calls.Load())
				}
			})
		}
	}
}

func TestKeyManagerSecretListFiltersEntireRawBodyDescriptor(t *testing.T) {
	for _, f := range secretListFixtures() {
		fields, filters := secretListBodyTarget()
		if len(filters) != 12 {
			t.Fatal(len(filters))
		}
		for field, filter := range filters {
			t.Run(f.name+"/"+field, func(t *testing.T) {
				cloud := testcloud.New(t)
				decoy := make(map[string]any)
				for key, value := range fields {
					decoy[key] = value
				}
				wire := field
				switch field {
				case "created_at":
					wire = "created"
				case "updated_at":
					wire = "updated"
				case "expires_at":
					wire = "expiration"
				case "secret_id":
					wire = "secret_ref"
				}
				switch field {
				case "bit_length":
					decoy[wire] = json.Number("9007199254740992")
				case "content_types":
					decoy[wire] = map[string]string{"default": "wrong"}
				case "payload":
					decoy[wire] = map[string]any{"nested": map[string]any{"n": json.Number("9007199254740992")}}
				case "created_at", "updated_at", "expires_at":
					decoy[wire] = "2025-10-02T03:04:05"
				case "secret_id", "secret_ref":
					decoy[wire] = "https://foreign.invalid/secrets/decoy"
				default:
					decoy[wire] = "wrong"
				}
				var calls atomic.Int32
				cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.RawQuery != "" {
						t.Error("local Body field leaked", field, r.URL)
					}
					testcloud.JSON(w, 200, secretFindPage(secretListRow(t, "decoy", decoy)+","+secretListRow(t, "target", fields), ""))
				})
				values, err := f.open(t, cloud).Resources.All(context.Background(), resource.WithFilter(field, filter))
				secretListWant(t, values, err, "target")
				if values[0].BitLength != 9007199254740993 || values[0].Created.Year() != 2026 || calls.Load() != 1 {
					t.Fatal(values, calls.Load())
				}
			})
		}
	}
}

func TestKeyManagerSecretListFiltersPassiveIDsAndFormatter(t *testing.T) {
	for _, f := range secretListFixtures() {
		for _, tc := range []struct {
			name, field    string
			fields         map[string]any
			value          any
			match, invalid bool
		}{
			{"id full reference", "id", map[string]any{"secret_ref": "https://foreign.invalid/secrets/ref"}, "https://foreign.invalid/secrets/ref", true, false},
			{"id does not derive suffix", "id", map[string]any{"secret_ref": "https://foreign.invalid/secrets/ref"}, "ref", false, false},
			{"literal null wins", "id", map[string]any{"id": nil, "secret_ref": "https://foreign.invalid/secrets/ref"}, nil, true, false},
			{"null blocks alternate", "id", map[string]any{"id": nil, "secret_ref": "https://foreign.invalid/secrets/ref"}, "https://foreign.invalid/secrets/ref", false, false},
			{"literal empty wins", "id", map[string]any{"id": ""}, "", true, false},
			{"literal large integer", "id", map[string]any{"id": json.Number("9007199254740993")}, json.Number("9007199254740993"), true, false},
			{"literal object subset", "id", map[string]any{"id": map[string]any{"a": 1, "b": 2}}, map[string]any{"a": 1}, true, false},
			{"JSON bool differs number", "id", map[string]any{"id": true}, 1, false, false},
			{"formatted raw percent", "secret_id", map[string]any{"secret_ref": "https://foreign.invalid/secrets/raw%2Fid?ignored=yes#frag"}, "raw%2Fid", true, false},
			{"formatted literal unicode space", "secret_id", map[string]any{"secret_ref": "https://foreign.invalid/secrets/한 글"}, "한 글", true, false},
			{"formatted trailing slash", "secret_id", map[string]any{"secret_ref": "https://foreign.invalid/secrets/"}, "", true, false},
			{"formatted null", "secret_id", map[string]any{"secret_ref": nil}, nil, true, false},
			{"selected relative ref", "secret_id", map[string]any{"secret_ref": "relative/ref"}, "ref", false, true},
			{"selected missing authority", "secret_id", map[string]any{"secret_ref": "https:///ref"}, "ref", false, true},
			{"selected missing path", "secret_id", map[string]any{"secret_ref": "https://foreign.invalid"}, "", false, true},
			{"Go parser authority boundary", "secret_id", map[string]any{"secret_ref": "https://foreign.invalid:notport/ref"}, "ref", false, true},
			{"unselected malformed ref", "status", map[string]any{"secret_ref": "relative/ref"}, "ACTIVE", true, false},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 200, secretFindPage(secretListRow(t, "target", tc.fields), ""))
				})
				options := []resource.ListOption{resource.WithFilter(tc.field, tc.value)}
				if tc.invalid {
					options = append(options, resource.WithName("does not match"))
				}
				values, err := f.open(t, cloud).Resources.All(context.Background(), options...)
				if tc.invalid {
					if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(values, err)
					}
				} else if tc.match {
					secretListWant(t, values, err, "target")
				} else {
					secretListWant(t, values, err)
				}
				if calls.Load() != 1 {
					t.Fatal(calls.Load())
				}
			})
		}
	}
}

func TestKeyManagerSecretListFiltersExactJSONAndPresence(t *testing.T) {
	for _, f := range secretListFixtures() {
		for _, tc := range []struct {
			name           string
			actual, filter any
			present, match bool
		}{
			{"missing null", nil, nil, false, true}, {"present null", nil, nil, true, true},
			{"empty array distinct", []any{}, nil, true, false}, {"empty array exact", []any{}, []any{}, true, true},
			{"array order", []any{"a", nil, "b"}, []any{"a", nil, "b"}, true, true}, {"array wrong order", []any{"a", "b"}, []any{"b", "a"}, true, false},
			{"array exact nested objects", []any{map[string]any{"a": 1, "extra": 2}}, []any{map[string]any{"a": 1}}, true, false},
			{"recursive object subset", map[string]any{"nested": map[string]any{"a": 1, "extra": 2}, "other": 3}, map[string]any{"nested": map[string]any{"a": json.Number("1e0")}}, true, true},
			{"empty actual object", map[string]any{}, map[string]any{}, true, false}, {"empty filter nonempty object", map[string]any{"a": 1}, map[string]any{}, true, true},
			{"object filter against array", []any{1}, map[string]any{}, true, false},
			{"exact large number", json.Number("9007199254740993"), json.Number("9007199254740993"), true, true},
			{"no float rounding", json.Number("9007199254740993"), json.Number("9007199254740992"), true, false},
			{"decimal equality", json.Number("1.000"), json.Number("1e0"), true, true},
			{"giant exponent equality", json.Number("1e1000000000"), json.Number("10e999999999"), true, true},
			{"bool number strict", true, 1, true, false}, {"string number strict", "1", 1, true, false},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				fields := make(map[string]any)
				if tc.present {
					fields["payload"] = tc.actual
				}
				cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, secretFindPage(secretListRow(t, "target", fields), ""))
				})
				values, err := f.open(t, cloud).Resources.All(context.Background(), resource.WithFilter("payload", tc.filter))
				if tc.match {
					secretListWant(t, values, err, "target")
				} else {
					secretListWant(t, values, err)
				}
			})
		}
	}
}

func TestKeyManagerSecretListFiltersQueriesSnapshotsAndNamespacePolicies(t *testing.T) {
	for _, f := range secretListFixtures() {
		for _, tc := range []struct {
			name    string
			options []resource.ListOption
			query   url.Values
		}{
			{"scalars repeated", []resource.ListOption{resource.WithFilters(map[string]any{"mode": []any{"first", nil, "second", false, json.Number("1e0")}, "acl_only": false, "bits": json.Number("2.0"), "name": "", "algorithm": nil})}, url.Values{"mode": {"first", "second", "false", "1e0"}, "acl_only": {"false"}, "bits": {"2.0"}, "name": {""}}},
			{"canonical nil wins", []resource.ListOption{resource.WithFilters(map[string]any{"algorithm": nil, "alg": "ignored"})}, url.Values{}},
			{"canonical false wins", []resource.ListOption{resource.WithFilters(map[string]any{"algorithm": false, "alg": "ignored"})}, url.Values{"alg": {"false"}}},
			{"canonical empty wins", []resource.ListOption{resource.WithFilters(map[string]any{"algorithm": "", "alg": "ignored"})}, url.Values{"alg": {""}}},
			{"canonical empty array wins", []resource.ListOption{resource.WithFilters(map[string]any{"algorithm": []any{}, "alg": "ignored"})}, url.Values{}},
			{"individual wire last", []resource.ListOption{resource.WithFilter("algorithm", "first"), resource.WithFilter("alg", "second")}, url.Values{"alg": {"second"}}},
			{"individual canonical last", []resource.ListOption{resource.WithFilter("alg", "first"), resource.WithFilter("algorithm", "second")}, url.Values{"alg": {"second"}}},
			{"unknown discard bad JSON", []resource.ListOption{resource.WithFilters(map[string]any{"offset": math.NaN(), "vendor": make(chan int)})}, url.Values{}},
			{"final known override removes error", []resource.ListOption{resource.WithFilter("mode", make(chan int)), resource.WithFilter("mode", "valid")}, url.Values{"mode": {"valid"}}},
			{"clear semantic keeps other namespaces", []resource.ListOption{resource.WithFilter("payload", make(chan int)), resource.WithFilters(nil), resource.WithQuery("mode", "raw"), resource.WithBodyFilter("status", "ACTIVE")}, url.Values{"mode": {"raw"}}},
			{"body and raw same text independent", []resource.ListOption{resource.WithFilter("created_at", "2026-10-02T03:04:05"), resource.WithQuery("created_at", "wire-extension"), resource.WithQuery("status", "wire-status")}, url.Values{"created_at": {"wire-extension"}}},
		} {
			t.Run(f.name+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), tc.query) {
						t.Error(r.URL.Query(), tc.query)
					}
					testcloud.JSON(w, 200, secretFindPage(secretListRow(t, "unrelated-name", map[string]any{"created": "2026-10-02T03:04:05"}), ""))
				})
				values, err := f.open(t, cloud).Resources.All(context.Background(), tc.options...)
				secretListWant(t, values, err, "unrelated-name")
			})
		}
		t.Run(f.name+"/construction snapshots and concurrent reuse", func(t *testing.T) {
			cloud := testcloud.New(t)
			nested := []any{"first", map[string]any{"n": json.Number("9007199254740993")}}
			query := []any{"one", "two"}
			input := map[string]any{"payload": nested, "mode": query}
			option := resource.WithFilters(input)
			nested[0] = "caller"
			nested[1].(map[string]any)["n"] = 0
			query[0] = "caller"
			input["mode"] = "caller"
			var calls atomic.Int32
			cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["mode"], []string{"one", "two"}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, secretFindPage(secretListRow(t, "target", map[string]any{"payload": json.RawMessage(`["first",{"n":9007199254740993}]`)}), ""))
			})
			api := f.open(t, cloud)
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
	}
}

func TestKeyManagerSecretListFiltersLazyPreflightAndCollisions(t *testing.T) {
	for _, f := range secretListFixtures() {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			testcloud.JSON(w, 200, secretFindPage("", ""))
		})
		api := f.open(t, cloud)
		for _, options := range [][]resource.ListOption{
			{resource.WithFilters(map[string]any{"algorithm": "same"}), resource.WithQuery("alg", "same")},
			{resource.WithQuery("alg", "same"), resource.WithFilters(map[string]any{"algorithm": nil})},
			{resource.WithFilter("limit", 1), resource.WithPageSize(1)},
			{resource.WithFilter("name", "same"), resource.WithName("same")},
			{resource.WithBodyFilter("status", "ACTIVE"), resource.WithFilter("status", "ACTIVE")},
			{resource.WithFilter("mode", map[string]any{"nested": 1})},
			{resource.WithFilter("mode", []any{[]any{1}})},
			{resource.WithFilter("payload", make(chan int))},
			{resource.WithBodyFilter("payload", make(chan int)), resource.WithBodyFilters(nil)},
			{resource.WithBodyFilter("created", "not an attribute")},
			{resource.WithMaxItems(-1), resource.WithFilter("status", "ACTIVE")},
		} {
			seq := api.Resources.List(context.Background(), options...)
			if calls.Load() != 0 {
				t.Fatal("eager request", calls.Load())
			}
			observations := 0
			for value, err := range seq {
				observations++
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(value, err)
				}
			}
			if observations != 1 || calls.Load() != 0 {
				t.Fatal(observations, calls.Load())
			}
		}
		for _, control := range []string{"max_items", "paginated", "base_path", "resource_type", "allow_unknown_params", "headers", "microversion", "jmespath_filters", "session"} {
			values, err := api.Resources.All(context.Background(), resource.WithFilter(control, "value"))
			if values != nil || err == nil || calls.Load() != 0 {
				t.Fatal(control, values, err, calls.Load())
			}
		}
		if _, err := servers.New(api.RawClient()).Resources.All(context.Background(), resource.WithFilters(nil)); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
	}
}

func TestKeyManagerSecretListFiltersControlsAndNativeFailures(t *testing.T) {
	for _, f := range secretListFixtures() {
		for _, mode := range []string{"raw cap before filter", "first page", "break", "local name status", "raw status ignored", "full native decode", "null row", "late null row", "cap skips null row", "cap skips selected formatter"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud, foreign := testcloud.New(t), testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
				first := secretListRow(t, "first", map[string]any{"payload": nil})
				rows := first + "," + secretListRow(t, "second", map[string]any{"payload": nil})
				switch mode {
				case "local name status":
					rows = first + "," + secretListRow(t, "first", map[string]any{"payload": nil, "status": "BUILD"}) + "," + secretListRow(t, "other", map[string]any{"payload": nil})
				case "full native decode":
					rows = first + `,{"name":"bad","content_types":{"default":false}}`
				case "null row":
					rows = `null`
				case "late null row", "cap skips null row":
					rows = first + `,null`
				case "cap skips selected formatter":
					rows = secretListRow(t, "first", map[string]any{"secret_ref": "https://foreign.invalid/secrets/valid"}) + "," + secretListRow(t, "second", map[string]any{"secret_ref": "relative/ref"})
				}
				cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Has("status") {
						t.Error("existing status query drop changed", r.URL)
					}
					if mode == "local name status" && r.URL.Query().Get("name") != "first" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, secretFindPage(rows, foreign.Server.URL+"/never"))
				})
				api := f.open(t, cloud)
				options := []resource.ListOption{resource.WithFilter("payload", nil)}
				switch mode {
				case "local name status":
					options = append(options, resource.WithName("first"), resource.WithStatus("active"), resource.WithPaginated(false))
				case "raw status ignored":
					options = append(options, resource.WithQuery("status", "arbitrary-wire-extension"), resource.WithPaginated(false))
				case "raw cap before filter":
					options = append(options, resource.WithMaxItems(1), resource.WithName("second"))
				case "first page":
					options = append(options, resource.WithPaginated(false))
				case "full native decode", "cap skips null row":
					options = append(options, resource.WithMaxItems(1))
				case "cap skips selected formatter":
					options = []resource.ListOption{resource.WithFilter("secret_id", "valid"), resource.WithMaxItems(1)}
				}
				if mode == "break" {
					count := 0
					for value, err := range api.Resources.List(context.Background(), options...) {
						if err != nil || value.Name != "first" {
							t.Fatal(value, err)
						}
						count++
						break
					}
					if count != 1 {
						t.Fatal(count)
					}
				} else {
					values, err := api.Resources.All(context.Background(), options...)
					switch mode {
					case "full native decode":
						var native *json.UnmarshalTypeError
						if values != nil || !errors.As(err, &native) {
							t.Fatal(values, err)
						}
					case "null row", "late null row":
						if values != nil || !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal(values, err)
						}
					case "raw cap before filter":
						secretListWant(t, values, err)
					case "first page", "raw status ignored":
						secretListWant(t, values, err, "first", "second")
					default:
						secretListWant(t, values, err, "first")
					}
				}
				if calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal(calls.Load(), followed.Load())
				}
			})
		}
		for _, body := range []string{`{"name":"bad","bit_length":"1"}`, `{"name":"bad","bit_length":1.0}`, `{"name":"bad","created":"2026-10-02T03:04:05Z"}`, `{"name":"bad","secret_ref":false}`, `{"name":"bad","content_types":[]}`} {
			t.Run(f.name+"/native malformed "+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, secretFindPage(secretListRow(t, "first", nil)+","+body, ""))
				})
				values, err := f.open(t, cloud).Resources.All(context.Background(), resource.WithFilter("payload", nil), resource.WithMaxItems(1))
				if values != nil || err == nil {
					t.Fatal(values, err)
				}
			})
		}
		for _, code := range []int{200, 204, 300, 201} {
			t.Run(fmt.Sprintf("%s/native page status %d", f.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if code == 204 {
						w.WriteHeader(code)
						return
					}
					testcloud.JSON(w, code, secretFindPage(secretListRow(t, "native", nil), ""))
				})
				values, err := f.open(t, cloud).Resources.All(context.Background(), resource.WithFilter("payload", nil))
				switch code {
				case 200, 300:
					secretListWant(t, values, err, "native")
				case 204:
					secretListWant(t, values, err)
				default:
					if values != nil || !gophercloud.ResponseCodeIs(err, code) {
						t.Fatal(values, err)
					}
				}
				if calls.Load() != 1 {
					t.Fatal(calls.Load())
				}
			})
		}
		t.Run(f.name+"/native empty JSON 204 parser", func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
				// Native PageResultFrom parses advertised JSON before IsEmpty.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(204)
			})
			values, err := f.open(t, cloud).Resources.All(context.Background(), resource.WithFilter("payload", nil))
			if values != nil || !errors.Is(err, io.EOF) {
				t.Fatal(values, err)
			}
		})
	}
}

func TestKeyManagerSecretListFiltersPagingSourceAndNativeIsolation(t *testing.T) {
	for _, f := range secretListFixtures() {
		for _, mode := range []string{"complete", "duplicate rows", "late404", "late decode", "cycle", "foreign", "cancel"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud, foreign := testcloud.New(t), testcloud.New(t)
				api := f.open(t, cloud)
				api.RawClient().MoreHeaders = map[string]string{"X-Configured": "preserved"}
				api.RawClient().Microversion = "1.0"
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var calls, followed, middleware atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
				transport := cloud.Provider.HTTPClient.Transport
				if transport == nil {
					transport = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					middleware.Add(1)
					response, err := transport.RoundTrip(r)
					if !r.URL.Query().Has("offset") {
						cloud.Provider.SetToken("later-token")
						if mode == "cancel" {
							cancel()
						}
					}
					return response, err
				})
				cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query()["mode"], []string{"one", "two"}) || r.Header.Get("X-Configured") != "preserved" || r.Header.Get("OpenStack-API-Version") != "key-manager 1.0" {
						t.Error(r.URL, r.Header)
					}
					if !r.URL.Query().Has("offset") {
						next := cloud.Server.URL + secretFindCollectionPath + "?offset=10&mode=one&mode=two"
						if mode == "foreign" {
							next = foreign.Server.URL + secretFindCollectionPath + "?offset=10&mode=one&mode=two"
						}
						testcloud.JSON(w, 200, secretFindPage(secretListRow(t, "first", map[string]any{"payload": map[string]any{"n": 1, "vendor": true}}), next))
						return
					}
					if r.Header.Get("X-Auth-Token") != "later-token" {
						t.Error(r.Header)
					}
					if mode == "late404" {
						testcloud.JSON(w, 404, `{"error":"late"}`)
						return
					}
					rows, next := secretListRow(t, "second", map[string]any{"payload": map[string]any{"n": 1, "vendor": false}}), ""
					if mode == "duplicate rows" {
						rows = secretListRow(t, "first", map[string]any{"payload": map[string]any{"n": 1, "vendor": true}})
					}
					if mode == "late decode" {
						rows = `{"name":"bad","bit_length":false}`
					}
					if mode == "cycle" {
						next = cloud.Server.URL + secretFindCollectionPath + "?offset=10&mode=one&mode=two"
					}
					testcloud.JSON(w, 200, secretFindPage(rows, next))
				})
				values, err := api.Resources.All(ctx, resource.WithFilter("mode", []any{"one", "two"}), resource.WithFilter("payload", map[string]any{"n": 1}))
				if mode == "complete" {
					secretListWant(t, values, err, "first", "second")
				} else if mode == "duplicate rows" {
					secretListWant(t, values, err, "first", "first")
				} else if values != nil || err == nil {
					t.Fatal(values, err)
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if mode == "cycle" && !errors.Is(err, resource.ErrPaginationCycle) {
					t.Fatal(err)
				}
				wantCalls, wantMiddleware, wantFollowed := int32(2), int32(2), int32(0)
				if mode == "cancel" {
					wantCalls, wantMiddleware = 1, 1
				}
				if mode == "foreign" {
					// This ordinary native pager retains its advertised-URL
					// behavior; Fetch/FindIdentity use separate guarded pagers.
					wantCalls, wantFollowed = 1, 1
				}
				if calls.Load() != wantCalls || middleware.Load() != wantMiddleware || followed.Load() != wantFollowed || api.RawClient().ProviderClient != cloud.Provider {
					t.Fatal(calls.Load(), middleware.Load(), followed.Load())
				}
			})
		}
		t.Run(f.name+"/native List Fetch and FindIdentity unchanged", func(t *testing.T) {
			cloud := testcloud.New(t)
			var listCalls, payloads atomic.Int32
			cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
				listCalls.Add(1)
				if r.URL.Query().Get("payload") != "wire-only" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, secretFindPage(secretListRow(t, "one", map[string]any{"payload": nil})+","+secretListRow(t, "two", map[string]any{"payload": "other"}), ""))
			})
			cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"name":"fetched","content_types":{"default":"text/plain"}}`)
			})
			cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) { payloads.Add(1); _, _ = w.Write([]byte("original")) })
			api := f.open(t, cloud)
			count := 0
			for value, err := range api.List(context.Background(), secrets.WithListQuery("payload", "wire-only")) {
				if err != nil || value == nil {
					t.Fatal(value, err)
				}
				count++
			}
			if count != 2 || listCalls.Load() != 1 {
				t.Fatal(count, listCalls.Load())
			}
			value, err := api.Fetch(context.Background(), resource.ID("secret-alpha"))
			if err != nil || value.Payload == nil || value.Payload.Text == nil || *value.Payload.Text != "original" {
				t.Fatal(value, err)
			}
			value, err = api.FindIdentity(context.Background(), "secret-alpha")
			if err != nil || value.Payload == nil || payloads.Load() != 2 {
				t.Fatal(value, err, payloads.Load())
			}
			if _, err = api.FindIdentity(context.Background(), "unsafe name", resource.WithIdentityFindQuery("payload", "value")); !errors.Is(err, resource.ErrUnsupported) || listCalls.Load() != 1 {
				t.Fatal(err, listCalls.Load())
			}
		})
	}
}
