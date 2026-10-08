package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

const userProjectRecordsPath = "/reverse/identity/v3/users/user-id/projects"

func collectUserProjectRecords(t *testing.T, sequence iter.Seq2[*users.UserProjectRecord, error]) ([]*users.UserProjectRecord, error) {
	t.Helper()
	values := make([]*users.UserProjectRecord, 0)
	for value, err := range sequence {
		if err != nil {
			return values, err
		}
		if value == nil || value.Resource == nil || value.Wire == nil {
			t.Fatal("successful row lacks owned resource or wire", value)
		}
		values = append(values, value)
	}
	return values, nil
}

func userProjectRecordIDs(t *testing.T, values []*users.UserProjectRecord) []string {
	t.Helper()
	ids := make([]string, 0, len(values))
	for _, value := range values {
		var id string
		if err := json.Unmarshal(value.Wire.Body["id"], &id); err != nil {
			t.Fatal("actual record ID", err, value.Wire.Body)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestIdentityUserProjectRecordsSemanticQueriesAliasesAndOwnedOptions(t *testing.T) {
	// Independent source declaration: eleven canonical names, fifteen spellings.
	queries := map[string]string{"limit": "limit", "marker": "marker", "domain_id": "domain_id", "is_domain": "is_domain", "name": "name", "parent_id": "parent_id", "is_enabled": "enabled", "enabled": "enabled", "tags": "tags", "any_tags": "tags-any", "tags-any": "tags-any", "not_tags": "not-tags", "not-tags": "not-tags", "not_any_tags": "not-tags-any", "not-tags-any": "not-tags-any"}
	for key, wire := range queries {
		t.Run(key, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			var input any = "server-value"
			want := "server-value"
			if wire == "limit" {
				input, want = 2, "2"
			}
			if wire == "enabled" || wire == "is_domain" {
				input, want = false, "false"
			}
			cloud.Mux.HandleFunc(userProjectRecordsPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				if !reflect.DeepEqual(r.URL.Query(), url.Values{wire: {want}}) || r.ContentLength != 0 {
					t.Error(r.URL, r.ContentLength)
				}
				// The server deliberately does not enforce the supplied query.
				testcloud.JSON(w, 200, `{"projects":[{"id":"returned","name":"different","domain_id":"different","parent_id":"different","enabled":true,"is_domain":true,"tags":["different"]}]}`)
			})
			api := users.New(cloud.Client("identity", "/reverse/identity/v3"))
			values, err := collectUserProjectRecords(t, api.ListProjectRecords(context.Background(), "user-id", users.WithProjectListFilter(key, input), users.WithProjectListPaginated(false)))
			if err != nil || calls.Load() != 1 || !reflect.DeepEqual(userProjectRecordIDs(t, values), []string{"returned"}) {
				t.Fatal(values, err, calls.Load())
			}
		})
	}
	for _, pair := range [][2]string{{"is_enabled", "enabled"}, {"any_tags", "tags-any"}, {"not_tags", "not-tags"}, {"not_any_tags", "not-tags-any"}} {
		for _, selected := range []struct {
			name  string
			value any
			want  url.Values
		}{{"null", nil, url.Values{}}, {"false", false, url.Values{pair[1]: {"false"}}}, {"empty", "", url.Values{pair[1]: {""}}}} {
			t.Run(pair[0]+" bulk "+selected.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(userProjectRecordsPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), selected.want) {
						t.Error(r.URL.Query(), selected.want)
					}
					testcloud.JSON(w, 200, `{"projects":[{"id":"bulk"}]}`)
				})
				api := users.New(cloud.Client("identity", "/reverse/identity/v3"))
				values, err := collectUserProjectRecords(t, api.ListProjectRecords(context.Background(), "user-id", users.WithProjectListFilters(map[string]any{pair[0]: selected.value, pair[1]: func() {}, "vendor:discard": func() {}})))
				if err != nil || calls.Load() != 1 || len(values) != 1 {
					t.Fatal(values, err, calls.Load())
				}
			})
		}
	}
	t.Run("snapshot lazy callbacks and individual last wins", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, callbacks atomic.Int32
		filters := map[string]any{"name": "original", "any_tags": []string{"one", "two"}}
		frozen := users.WithProjectListFilters(filters)
		filters["name"], filters["any_tags"].([]string)[0] = "changed", "changed"
		cloud.Mux.HandleFunc(userProjectRecordsPath, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if !reflect.DeepEqual(r.URL.Query(), url.Values{"name": {"original"}, "tags-any": {"one", "two"}, "enabled": {"true"}}) {
				t.Error(r.URL)
			}
			th.TestHeader(t, r, "X-Trace", "callback")
			th.TestHeader(t, r, "OpenStack-API-Version", "identity 3.13")
			testcloud.JSON(w, 200, `{"projects":[{"id":"owned"}]}`)
		})
		api := users.New(cloud.Client("identity", "/reverse/identity/v3"))
		seq := api.ListProjectRecords(context.Background(), "user-id", frozen, users.WithProjectListFilter("is_enabled", false), users.WithProjectListFilter("enabled", true), users.WithProjectListMicroversion("3.13"), func(config *request.Config[users.ListProjectRecordsOpts]) error {
			callbacks.Add(1)
			config.Headers["X-Trace"] = "callback"
			return nil
		})
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("iterator was not lazy")
		}
		for repeat := 0; repeat < 2; repeat++ {
			values, err := collectUserProjectRecords(t, seq)
			if err != nil || len(values) != 1 {
				t.Fatal(values, err)
			}
		}
		if calls.Load() != 2 || callbacks.Load() != 2 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
		if api.RawClient().Microversion != "" || len(api.RawClient().MoreHeaders) != 0 {
			t.Fatal("operation policy mutated the selected client", api.RawClient())
		}
	})
}

func TestIdentityUserProjectRecordsLocalBodiesNormalizedViewAndIndependentWire(t *testing.T) {
	for _, check := range []struct {
		name, field string
		filter      any
		want        []string
	}{
		{"id", "id", "hit", []string{"hit"}},
		{"description", "description", "target", []string{"hit"}},
		{"options subset", "options", map[string]any{"nested": map[string]any{"missing": nil}}, []string{"hit"}},
		{"options normalized before matching", "options", false, []string{}},
		{"links subset", "links", map[string]any{"self": "passive"}, []string{"hit"}},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(userProjectRecordsPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"name": {"server-only"}}) {
					t.Error("local field became query", r.URL)
				}
				testcloud.JSON(w, 200, `{"projects":[{"id":"hit","name":"different","description":"target","options":{"nested":{"anchor":true}},"links":{"self":"passive"}},{"id":"miss","name":"server-only","description":"other","options":{"nested":{}},"links":{"self":"different"}},{"id":"normalized","options":false}]}`)
			})
			api := users.New(cloud.Client("identity", "/reverse/identity/v3"))
			values, err := collectUserProjectRecords(t, api.ListProjectRecords(context.Background(), "user-id", users.WithProjectListFilter("name", "server-only"), users.WithProjectListFilter(check.field, check.filter), users.WithProjectListFilters(map[string]any{"name": "server-only", check.field: check.filter, "user:unknown": func() {}, "location": func() {}, "created_at": func() {}})))
			if err != nil || calls.Load() != 1 || !reflect.DeepEqual(userProjectRecordIDs(t, values), check.want) {
				t.Fatal(values, err, calls.Load())
			}
		})
	}
	t.Run("cached connection raw types and independent receipt", func(t *testing.T) {
		cloud := testcloud.New(t)
		conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/catalog/identity/v3/"))
		if err != nil {
			t.Fatal(err)
		}
		service, err := conn.Identity(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		cached, err := conn.IdentityV3(context.Background())
		if err != nil || cached != service || service.Users.RawClient() != service.RawClient() || service.RawClient().ProviderClient != cloud.Provider {
			t.Fatal(service, cached, err)
		}
		service.RawClient().ResourceBase = cloud.Server.URL + "/reverse/identity/v3/"
		cloud.Provider.SetToken("record-live")
		var calls atomic.Int32
		cloud.Mux.HandleFunc(userProjectRecordsPath, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestHeader(t, r, "X-Auth-Token", "record-live")
			th.TestHeader(t, r, "X-Trace", "owned")
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			w.Header().Set("X-Record", "actual")
			testcloud.JSON(w, 200, `{"projects":[{"id":"false","user_id":"response-user","options":false,"vendor":9007199254740993123},{"id":"array","options":[]},{"id":"null","options":null},{"id":"missing"},{"id":"object","options":{"immutable":false}}]}`)
		})
		values, err := collectUserProjectRecords(t, service.Users.ListProjectRecords(context.Background(), "user-id", users.WithProjectListHeader("X-Trace", "owned")))
		if err != nil || calls.Load() != 1 || len(values) != 5 {
			t.Fatal(values, err, calls.Load())
		}
		for index, value := range values {
			if string(value.Resource.Body["user_id"]) != `"user-id"` || value.Resource.StatusCode != 200 || value.Wire.StatusCode != 200 || value.Resource.Header.Get("X-Record") != "actual" || value.Wire.Header.Get("X-Record") != "actual" {
				t.Fatal(index, value)
			}
		}
		if string(values[0].Wire.Body["user_id"]) != `"response-user"` || string(values[0].Wire.Body["options"]) != "false" || string(values[0].Resource.Body["options"]) != "{}" || string(values[0].Resource.Body["vendor"]) != "9007199254740993123" || string(values[1].Wire.Body["options"]) != "[]" || string(values[1].Resource.Body["options"]) != "{}" || string(values[2].Resource.Body["options"]) != "null" {
			t.Fatal(values)
		}
		if _, exists := values[3].Resource.Body["options"]; exists {
			t.Fatal("missing options synthesized")
		}
		if _, exists := values[1].Wire.Body["user_id"]; exists {
			t.Fatal("URI leaked into actual wire")
		}
		values[0].Resource.Body["id"][1] = 'X'
		values[0].Resource.Header.Set("X-Record", "view changed")
		values[0].Wire.Body["options"][0] = 't'
		if string(values[0].Wire.Body["id"]) != `"false"` || values[0].Wire.Header.Get("X-Record") != "actual" || string(values[0].Resource.Body["options"]) != "{}" || string(values[1].Resource.Body["options"]) != "{}" {
			t.Fatal("view, wire or rows alias")
		}
	})
}

func TestIdentityUserProjectRecordsContinuationFormsRawCapsAndSinglePage(t *testing.T) {
	for _, mode := range []string{"dictionary", "links", "projects_links", "next", "HTTP Link", "server limit", "short limit", "single page", "empty stops"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(userProjectRecordsPath, func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				want := url.Values{"domain_id": {"domain"}}
				if mode == "short limit" {
					want.Set("limit", "5")
				}
				if call > 1 {
					want.Set("marker", "wire-last")
					if mode == "server limit" {
						want.Set("limit", "25")
					}
					if call == 3 {
						want.Set("marker", "second")
					}
					th.TestHeader(t, r, "X-Auth-Token", "next-token")
				}
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				if call > 3 {
					t.Error("unbounded continuation")
					testcloud.JSON(w, 200, `{"projects":[]}`)
					return
				}
				if call == 3 {
					testcloud.JSON(w, 200, `{"projects":[],"next":"`+cloud.Server.URL+userProjectRecordsPath+`?marker=must-not-follow"}`)
					return
				}
				if call == 2 {
					testcloud.JSON(w, 200, `{"projects":[{"id":"second","description":"target"}]}`)
					return
				}
				cloud.Provider.SetToken("next-token")
				next := cloud.Server.URL + userProjectRecordsPath + "?marker=wire-last"
				if mode == "server limit" {
					next += "&limit=25"
				}
				continuation := ""
				switch mode {
				case "dictionary", "single page":
					continuation = `,"links":{"next":"` + next + `"}`
				case "links", "server limit", "empty stops":
					continuation = `,"links":[{"rel":"next","href":"` + next + `"}]`
				case "projects_links":
					continuation = `,"projects_links":[{"rel":"next","href":"` + next + `"}]`
				case "next":
					continuation = `,"next":"` + next + `"`
				case "HTTP Link":
					w.Header().Set("Link", "<"+next+">; rel=\"next\"")
				}
				rows := `{"id":"first","description":"target"},{"id":"wire-last","description":"drop"}`
				if mode == "empty stops" {
					rows = ""
				}
				testcloud.JSON(w, 200, `{"projects":[`+rows+`]`+continuation+`}`)
			})
			api := users.New(cloud.Client("identity", "/reverse/identity/v3"))
			options := []users.ListProjectRecordsOption{users.WithProjectListFilter("domain_id", "domain"), users.WithProjectListFilter("description", "target")}
			if mode == "short limit" {
				options = append(options, users.WithProjectListOptions(users.ListProjectRecordsOpts{Limit: 5}))
			}
			if mode == "single page" {
				options = append(options, users.WithProjectListPaginated(false))
			}
			values, err := collectUserProjectRecords(t, api.ListProjectRecords(context.Background(), "user-id", options...))
			wantIDs, wantCalls := []string{"first", "second"}, int32(2)
			if mode == "short limit" {
				wantCalls = 3
			}
			if mode == "single page" {
				wantIDs, wantCalls = []string{"first"}, 1
			}
			if mode == "empty stops" {
				wantIDs, wantCalls = []string{}, 1
			}
			if err != nil || calls.Load() != wantCalls || !reflect.DeepEqual(userProjectRecordIDs(t, values), wantIDs) {
				t.Fatal(values, err, calls.Load(), wantCalls)
			}
		})
	}
	for _, cap := range []int{1, 2, 0} {
		t.Run("raw cap "+strconv.Itoa(cap), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := `{"projects":[{"id":"miss","description":"drop"},{"id":"hit","description":"target"},false]}`
			cloud.Mux.HandleFunc(userProjectRecordsPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				want := url.Values{}
				if cap > 0 {
					want.Set("limit", strconv.Itoa(cap))
				}
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL.Query(), want)
				}
				w.Header().Set("X-Cap", "actual")
				testcloud.JSON(w, 200, body)
			})
			api := users.New(cloud.Client("identity", "/reverse/identity/v3"))
			values, err := collectUserProjectRecords(t, api.ListProjectRecords(context.Background(), "user-id", users.WithProjectListMaxItems(cap), users.WithProjectListFilter("description", "target")))
			want := []string{}
			if cap != 1 {
				want = []string{"hit"}
			}
			if calls.Load() != 1 || !reflect.DeepEqual(userProjectRecordIDs(t, values), want) {
				t.Fatal(values, err, calls.Load())
			}
			if cap == 0 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Cap") != "actual" {
					t.Fatal("missing whole-page record decode evidence", err, proof)
				}
			} else if err != nil {
				t.Fatal("decoded beyond raw cap", err)
			}
		})
	}
}

func TestIdentityUserProjectRecordsPreflightOwnedFaultsAndPartialErrors(t *testing.T) {
	for _, scenario := range []string{"nil option", "invalid user dot", "user_id filter", "user_id query", "query collision", "selected query object", "source callback", "cancel before"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("invalid policy reached HTTP", r.URL)
				w.WriteHeader(500)
			})
			client := cloud.Client("identity", "/reverse/identity/v3")
			api := users.New(client)
			options := []users.ListProjectRecordsOption{}
			ctx := context.Background()
			userID := "user-id"
			cause := errors.New("membership cancel cause")
			switch scenario {
			case "nil option":
				options = append(options, nil)
			case "invalid user dot":
				userID = "."
			case "user_id filter":
				options = append(options, users.WithProjectListFilter("user_id", "other"))
			case "user_id query":
				options = append(options, users.WithProjectListQuery("user_id", "other"))
			case "query collision":
				options = append(options, users.WithProjectListFilter("is_enabled", false), users.WithProjectListQuery("enabled", "true"))
			case "selected query object":
				options = append(options, users.WithProjectListFilter("name", map[string]any{"invalid": true}))
			case "source callback":
				options = append(options, func(*request.Config[users.ListProjectRecordsOpts]) error {
					callbacks.Add(1)
					client.ResourceBase = cloud.Server.URL + "/replaced/"
					return nil
				}, func(*request.Config[users.ListProjectRecordsOpts]) error { callbacks.Add(1); return nil })
			case "cancel before":
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(cause)
			}
			seq := api.ListProjectRecords(ctx, userID, options...)
			if calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal("not lazy")
			}
			failures := 0
			for value, err := range seq {
				failures++
				if value != nil || err == nil {
					t.Fatal(value, err)
				}
				if scenario == "cancel before" {
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal(err)
					}
				} else if scenario == "user_id filter" {
					if !errors.Is(err, resource.ErrUnsupported) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			wantCallbacks := int32(0)
			if scenario == "source callback" {
				wantCallbacks = 1
			}
			if failures != 1 || calls.Load() != 0 || callbacks.Load() != wantCallbacks {
				t.Fatal(failures, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, scenario := range []string{"read", "close", "late403", "late malformed", "cancel after row"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cause := errors.New("record " + scenario)
			var track *payloadContractTracking
			if scenario == "read" {
				track = payloadContractTrack(cloud, cause, nil)
			}
			if scenario == "close" {
				track = payloadContractTrack(cloud, nil, cause)
			}
			first := `{"projects":[{"id":"first"}],"links":{"next":"` + cloud.Server.URL + userProjectRecordsPath + `?marker=next"}}`
			cloud.Mux.HandleFunc(userProjectRecordsPath, func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				w.Header().Set("X-Record-Page", strconv.Itoa(int(call)))
				if call == 1 {
					testcloud.JSON(w, 200, first)
					return
				}
				if r.URL.Query().Get("marker") != "next" || call != 2 {
					t.Error(r.URL, call)
				}
				if scenario == "late403" {
					testcloud.JSON(w, 403, `{"error":{"message":"later denied"}}`)
				} else {
					testcloud.JSON(w, 200, `{"projects":[false]}`)
				}
			})
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api := users.New(cloud.Client("identity", "/reverse/identity/v3"))
			rows, failures := 0, 0
			for value, err := range api.ListProjectRecords(ctx, "user-id") {
				if err == nil {
					rows++
					if value == nil || string(value.Wire.Body["id"]) != `"first"` {
						t.Fatal(value)
					}
					if scenario == "cancel after row" {
						cancel(cause)
					}
					continue
				}
				failures++
				if value != nil {
					t.Fatal("typed row on terminal failure", value, err)
				}
				if scenario == "late403" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"error":{"message":"later denied"}}` || native.ResponseHeader.Get("X-Record-Page") != "2" {
						t.Fatal(err, native)
					}
				} else if scenario == "cancel after row" {
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal(err)
					}
				} else {
					var proof *resource.ResponseError
					wantBody, wantPage := first, "1"
					if scenario == "late malformed" {
						wantBody, wantPage = `{"projects":[false]}`, "2"
					}
					if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != wantBody || proof.Header.Get("X-Record-Page") != wantPage {
						t.Fatal(err, proof)
					}
					if scenario != "late malformed" && !errors.Is(err, cause) {
						t.Fatal("lost original body failure", err)
					}
				}
			}
			wantRows, wantCalls := 1, int32(2)
			if scenario == "read" || scenario == "close" {
				wantRows, wantCalls = 0, 1
			}
			if scenario == "cancel after row" {
				wantCalls = 1
			}
			if rows != wantRows || failures != 1 || calls.Load() != wantCalls {
				t.Fatal(rows, failures, calls.Load(), wantRows, wantCalls)
			}
			if track != nil && (track.calls.Load() != 1 || track.last(t).closes.Load() != 1) {
				t.Fatal("body replay or close ownership", track.calls.Load(), track.last(t).closes.Load())
			}
		})
	}
}
