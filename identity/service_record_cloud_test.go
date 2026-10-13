package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func serviceCloudFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*identity.ServiceReadCloud, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("GET /identity/v3/services", reply)
	return identity.NewServiceReadCloud(services.New(fixture.Client("identity", "/identity/v3/"))), fixture
}
func serviceCloudField(t *testing.T, row *services.ServiceRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatal(key, want, row)
	}
}
func TestPythonServiceCloudSearchQuerylessAndLocalGlobSubset(t *testing.T) {
	calls := 0
	cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, readErr := io.ReadAll(r.Body)
		if r.URL.RawQuery != "" || r.Header.Get("Accept") != "application/json" || readErr != nil || len(body) != 0 || r.ContentLength != 0 {
			t.Fatal(r.URL, r.Header, string(body), readErr, r.ContentLength)
		}
		w.Header().Set("X-Request-Id", "cloud-service")
		testcloud.JSON(w, 203, `{"services":[{"id":"a","name":"ops-alpha","description":{"n":0.10000000000000001,"active":true}},{"id":"b","name":"other","description":{"n":0.2}},{"id":"a","name":"ops-beta","description":{"n":0.1,"extra":1}}]}`)
	})
	result, err := cloud.SearchServices(context.Background(), "ops-*", identity.WithServiceReadFilters(json.RawMessage(`{"description":{"n":0.1}}`)))
	if err != nil || result == nil || len(result.Services) != 2 || len(result.Inventory) != 3 || result.Expression || calls != 1 {
		t.Fatal(result, err, calls)
	}
	for _, row := range result.Services {
		serviceCloudField(t, row, "id", `"a"`)
		if row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "cloud-service" {
			t.Fatal(row)
		}
	}
	result.Services[0].Resource.Body["id"][1] = 'X'
	result.Services[0].Header.Set("X-Request-Id", "caller")
	serviceCloudField(t, result.Inventory[0], "id", `"a"`)
	if result.Inventory[0].Header.Get("X-Request-Id") != "cloud-service" {
		t.Fatal(result)
	}
}
func TestPythonServiceCloudSearchStringExpressionsUseMaterializedRows(t *testing.T) {
	calls := 0
	cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.RawQuery != "" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"services":[{"id":"a","name":"same"},{"id":"b","name":"other"}]}`)
	})
	for _, tc := range []struct{ name, filter, want string }{{"", "length(@)", "2"}, {"same", "[0].id", `"a"`}, {"", "[?name==`same`].id", `["a"]`}, {"missing", "length(@)", "0"}, {"", "`null`", "null"}} {
		raw, _ := json.Marshal(tc.filter)
		before := calls
		result, err := cloud.SearchServices(context.Background(), tc.name, identity.WithServiceReadFilters(raw))
		if err != nil || result == nil || !result.Expression || string(result.Value) != tc.want || result.Services != nil || len(result.Inventory) != 2 || calls != before+1 {
			t.Fatal(tc, result, err, calls)
		}
	}
}
func TestPythonServiceCloudFalseyFiltersPreserveDeclaredRows(t *testing.T) {
	calls := 0
	cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"services":[{"id":null,"name":false}]}`)
	})
	for _, raw := range []string{"null", "false", "0", "1e-4000", `""`, "[]", "{}"} {
		result, err := cloud.SearchServices(context.Background(), "", identity.WithServiceReadFilters(json.RawMessage(raw)))
		if err != nil || result == nil || len(result.Services) != 1 || len(result.Inventory) != 1 || result.Expression {
			t.Fatal(raw, result, err)
		}
		serviceCloudField(t, result.Services[0], "id", "null")
		serviceCloudField(t, result.Services[0], "name", "false")
	}
	if calls != 7 {
		t.Fatal(calls)
	}
	result, err := cloud.SearchServices(context.Background(), "None")
	if err != nil || len(result.Services) != 1 {
		t.Fatal(result, err)
	}
}
func TestPythonServiceCloudLocalErrorsFollowEveryPageAndKeepInventory(t *testing.T) {
	for _, raw := range []string{`"["`, `{"unknown":true}`, `{"description":{"x":1}}`, `{invalid`} {
		calls := 0
		cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				testcloud.JSON(w, 203, `{"services":[{"id":"a","description":"truthy"}],"next":"?marker=a"}`)
			} else {
				testcloud.JSON(w, 200, `{"services":[{"id":"b","description":"truthy"}]}`)
			}
		})
		result, err := cloud.SearchServices(context.Background(), "", identity.WithServiceReadFilters(json.RawMessage(raw)))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 2 || result.Services != nil || result.Value != nil || calls != 2 || errors.As(err, &proof) {
			t.Fatal(raw, result, err, proof, calls)
		}
		if result.Inventory[0].StatusCode != 203 || result.Inventory[1].StatusCode != 200 {
			t.Fatal(result)
		}
	}
}
func TestPythonServiceCloudOrderedMismatchHidesUnreachedBadShape(t *testing.T) {
	cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"services":[{"id":"a","name":"other","description":"scalar"}]}`)
	})
	result, err := cloud.SearchServices(context.Background(), "", identity.WithServiceReadFilters(json.RawMessage(`{"name":"mismatch","description":{"x":1}}`)))
	if err != nil || result == nil || string(result.Value) != "[]" || result.Services == nil || len(result.Services) != 0 {
		t.Fatal(result, err)
	}
	result, err = cloud.SearchServices(context.Background(), "", identity.WithServiceReadFilters(json.RawMessage(`{"description":{"x":1},"name":"mismatch"}`)))
	if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 1 {
		t.Fatal(result, err)
	}
}
func TestPythonServiceCloudLateHTTPFailurePrecedesInvalidLocalFilter(t *testing.T) {
	calls := 0
	cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Request-Id", "late-service")
		if calls == 1 {
			testcloud.JSON(w, 203, `{"services":[{"id":"a"}],"next":"?marker=a"}`)
		} else {
			testcloud.JSON(w, 503, `{"error":"actual late"}`)
		}
	})
	result, err := cloud.SearchServices(context.Background(), "", identity.WithServiceReadFilters(json.RawMessage(`"["`)))
	var proof gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &proof) || proof.Actual != 503 || proof.ResponseHeader.Get("X-Request-Id") != "late-service" || string(proof.Body) != `{"error":"actual late"}` || proof.Method != "GET" || !strings.Contains(proof.URL, "/identity/v3/services?marker=a") || result == nil || len(result.Inventory) != 1 || result.Value != nil || calls != 2 {
		t.Fatal(result, err, proof, calls)
	}
}
func TestPythonServiceCloudLocationBeforeFilterAndOwnedArguments(t *testing.T) {
	cloudName, projectName := "configured", "current"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	raw := json.RawMessage(`{"location":{"cloud":"configured","project":{"id":"scope","name":"current"}}}`)
	headers := map[string]string{"X-Owned": "owned"}
	opt := identity.WithServiceReadOpts(identity.ServiceReadOpts{Filters: &raw, Headers: headers, Location: &location})
	cloudName = "changed"
	projectName = "changed"
	raw[0] = '['
	headers["X-Owned"] = "changed"
	cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || r.Header.Get("X-Owned") != "owned" {
			t.Fatal(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"services":[{"id":"a","project_id":"foreign","location":{"cloud":"wire"}}]}`)
	})
	result, err := cloud.SearchServices(context.Background(), "", opt)
	if err != nil || len(result.Services) != 1 || len(result.Inventory) != 1 || string(result.Services[0].Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal(result, err)
	}
	if _, exists := result.Services[0].Resource.Body["project_id"]; exists {
		t.Fatal(result)
	}
}
func TestPythonServiceCloudOptionsGuardCallbacksAndSelectedSourceSnapshot(t *testing.T) {
	calls := 0
	cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
			t.Fatal(r.Header)
		}
		testcloud.JSON(w, 200, `{"services":[]}`)
	})
	client := cloud.RawAPI().RawClient()
	client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
	_, err := cloud.SearchServices(context.Background(), "", identity.ServiceReadOption(func(*identity.ServiceReadOpts) error {
		client.MoreHeaders["X-Snapshot"] = "after"
		client.MoreHeaders["X-Late"] = "after"
		return nil
	}))
	if err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	for _, mode := range []string{"source", "cancel", "nil"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			seen := 0
			endpoint := client.Endpoint
			defer func() { client.Endpoint = endpoint }()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller canceled service search")
			first := identity.ServiceReadOption(func(*identity.ServiceReadOpts) error {
				if mode == "source" {
					client.Endpoint += "changed"
				} else {
					cancel(cause)
				}
				return nil
			})
			if mode == "nil" {
				first = nil
			}
			_, err := cloud.SearchServices(ctx, "", first, identity.ServiceReadOption(func(*identity.ServiceReadOpts) error { seen++; client.Endpoint = endpoint; return nil }))
			var proof *resource.ResponseError
			if err == nil || seen != 0 || calls != before || errors.As(err, &proof) {
				t.Fatal(err, seen, calls, proof)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
	for _, nilCloud := range []*identity.ServiceReadCloud{nil, identity.NewServiceReadCloud(nil)} {
		result, err := nilCloud.SearchServices(context.Background(), "")
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	if !reflect.DeepEqual(cloud.RawAPI().RawClient(), client) {
		t.Fatal(cloud)
	}
}

func TestPythonServiceCloudGetNullFilterMemberVersusAllOtherFalseySearchBranches(t *testing.T) {
	for _, filter := range []string{"omitted", "null", "{}", "[]", "false", "0", "\"\""} {
		t.Run(filter, func(t *testing.T) {
			fixture := testcloud.New(t)
			calls := 0
			fixture.Mux.HandleFunc("GET /identity/v3/services/member", func(w http.ResponseWriter, r *http.Request) {
				calls++
				testcloud.JSON(w, 203, `{"service":{"id":"member","name":"nova","enabled":false}}`)
			})
			fixture.Mux.HandleFunc("GET /identity/v3/services", func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.RawQuery != "" {
					t.Fatal(r.URL)
				}
				testcloud.JSON(w, 200, `{"services":[{"id":"member","name":"nova","enabled":false}]}`)
			})
			cloud := identity.NewServiceReadCloud(services.New(fixture.Client("identity", "/identity/v3/")))
			opts := []identity.ServiceReadOption{}
			if filter != "omitted" {
				opts = append(opts, identity.WithServiceReadFilters(json.RawMessage(filter)))
			}
			result, err := cloud.GetService(context.Background(), "member", opts...)
			deprecated := filter != "omitted" && filter != "null"
			if err != nil || result == nil || result.Service == nil || result.DeprecatedFilters != deprecated || result.Expression || calls != 1 || len(result.Inventory) != 1 {
				t.Fatal(result, err, calls)
			}
			want := 203
			if deprecated {
				want = 200
			}
			if result.Service.StatusCode != want {
				t.Fatal(result)
			}
			result.Service.Resource.Body["id"][1] = 'X'
			result.Service.Header.Set("X-Copy", "changed")
			serviceCloudField(t, result.Inventory[0], "id", `"member"`)
			if result.Inventory[0].Header.Get("X-Copy") != "" {
				t.Fatal(result)
			}
		})
	}
}
func TestPythonServiceCloudGetDefaultFallbackIsLiteralNameNotGlob(t *testing.T) {
	for _, mode := range []string{"exact", "glob", "missing", "duplicate", "late"} {
		t.Run(mode, func(t *testing.T) {
			fixture := testcloud.New(t)
			calls := 0
			identifier := "nova"
			if mode == "glob" {
				identifier = "no*"
			}
			fixture.Mux.HandleFunc("GET /identity/v3/services/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				testcloud.JSON(w, 403, `{"error":"member denied"}`)
			})
			fixture.Mux.HandleFunc("GET /identity/v3/services", func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Query().Get("name") != identifier {
					t.Fatal(r.URL)
				}
				if calls == 2 {
					if mode == "missing" {
						testcloud.JSON(w, 200, `{"services":[]}`)
						return
					}
					testcloud.JSON(w, 203, `{"services":[{"id":"one","name":"nova"}],"next":"?name=`+identifier+`&marker=one"}`)
					return
				}
				if mode == "late" {
					testcloud.JSON(w, 503, `{"error":"late"}`)
					return
				}
				if mode == "duplicate" {
					testcloud.JSON(w, 200, `{"services":[{"id":"one","name":"nova"}]}`)
					return
				}
				testcloud.JSON(w, 200, `{"services":[]}`)
			})
			cloud := identity.NewServiceReadCloud(services.New(fixture.Client("identity", "/identity/v3/")))
			result, err := cloud.GetService(context.Background(), identifier)
			switch mode {
			case "exact":
				if err != nil || result == nil || result.Service == nil || result.Service.StatusCode != 203 || calls != 3 {
					t.Fatal(result, err, calls)
				}
			case "glob", "missing":
				if err != nil || result == nil || result.Service != nil || string(result.Value) != "null" {
					t.Fatal(result, err, calls)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) || result.Service != nil || calls != 3 {
					t.Fatal(result, err, calls)
				}
			case "late":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 503 || calls != 3 {
					t.Fatal(result, err, calls)
				}
			}
		})
	}
}
func TestPythonServiceCloudGetDeprecatedFiltersDuplicatesSearchGlobAndMissing(t *testing.T) {
	for _, mode := range []string{"glob", "missing", "duplicates", "filtered one"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.RawQuery != "" {
					t.Fatal(r.URL)
				}
				testcloud.JSON(w, 200, `{"services":[{"id":"one","name":"nova","is_enabled":true,"description":{"owner":"infra"}},{"id":"one","name":"nova","is_enabled":false,"description":{"owner":"ops"}}]}`)
			})
			name, filter := "no*", json.RawMessage(`{}`)
			if mode == "missing" {
				name = "missing"
			}
			if mode == "filtered one" {
				filter = json.RawMessage(`{"is_enabled":true}`)
			}
			result, err := cloud.GetService(context.Background(), name, identity.WithServiceReadFilters(filter))
			if result == nil || !result.DeprecatedFilters || len(result.Inventory) != 2 || calls != 1 {
				t.Fatal(result, err, calls)
			}
			switch mode {
			case "glob", "duplicates":
				if !errors.Is(err, resource.ErrAmbiguous) || result.Service != nil {
					t.Fatal(result, err)
				}
			case "missing":
				if err != nil || string(result.Value) != "null" || result.Service != nil {
					t.Fatal(result, err)
				}
			case "filtered one":
				if err != nil || result.Service == nil || string(result.Service.Resource.Body["is_enabled"]) != "true" {
					t.Fatal(result, err)
				}
			}
		})
	}
}
func TestPythonServiceCloudGetDeprecatedExpressionTruthLengthAndIntegerIndex(t *testing.T) {
	calls := 0
	cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 203, `{"services":[{"id":"one","name":"nova"}]}`)
	})
	for _, tc := range []struct{ expression, want, kind string }{
		{"`null`", "null", "none"}, {"`false`", "null", "none"}, {"`0`", "null", "none"}, {"`[]`", "null", "none"}, {"`{}`", "null", "none"}, {"`\"\"`", "null", "none"},
		{"`[false]`", "false", "value"}, {"`[42]`", "42", "value"}, {"`\"界\"`", `"界"`, "value"}, {"[0:1].id", `"one"`, "value"}, {"[0:1]", "", "value"},
		{"`[1,2]`", "null", "ambiguous"}, {"`\"ab\"`", "null", "ambiguous"}, {"`{\"a\":1,\"b\":2}`", "null", "ambiguous"},
		{"`{\"a\":1}`", "null", "invalid"}, {"`true`", "null", "invalid"}, {"`1`", "null", "invalid"}, {"`1.5`", "null", "invalid"},
	} {
		t.Run(fmt.Sprint(tc.expression), func(t *testing.T) {
			raw, _ := json.Marshal(tc.expression)
			before := calls
			result, err := cloud.GetService(context.Background(), "", identity.WithServiceReadFilters(raw))
			if result == nil || !result.Expression || !result.DeprecatedFilters || result.Service != nil || len(result.Inventory) != 1 || calls != before+1 {
				t.Fatal(tc, result, err, calls)
			}
			switch tc.kind {
			case "ambiguous":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(result, err)
				}
			case "invalid":
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) {
					t.Fatal(result, err, proof)
				}
			default:
				if err != nil {
					t.Fatal(tc, result, err)
				}
				if tc.want != "" && string(result.Value) != tc.want {
					t.Fatal(tc, result)
				}
				if tc.expression == "[0:1]" && !strings.Contains(string(result.Value), `"id":"one"`) {
					t.Fatal(result)
				}
			}
		})
	}
}
func TestPythonServiceCloudGetOptionCallbackOnceCapturedSourceAndOwnedLocation(t *testing.T) {
	fixture := testcloud.New(t)
	calls, options := 0, 0
	fixture.Mux.HandleFunc("GET /identity/v3/services/one", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Before") != "original" || r.Header.Get("X-After") != "" {
			t.Fatal(r.Header)
		}
		testcloud.JSON(w, 200, `{"service":{"id":"one","name":"nova","project_id":"foreign","location":{"cloud":"wire"}}}`)
	})
	fixture.Mux.HandleFunc("GET /identity/v3/services", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.RawQuery != "" || r.Header.Get("X-Before") != "original" || r.Header.Get("X-After") != "" {
			t.Fatal(r.Header, r.URL)
		}
		testcloud.JSON(w, 200, `{"services":[{"id":"one","name":"nova"}]}`)
	})
	api := services.New(fixture.Client("identity", "/identity/v3/"))
	cloud := identity.NewServiceReadCloud(api)
	name := "configured"
	location := resource.CloudLocation{Cloud: &name, Zone: json.RawMessage(`"az"`), Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}
	loc := identity.WithServiceReadLocation(location)
	name = "changed"
	for _, deprecated := range []bool{false, true} {
		api.RawClient().MoreHeaders = map[string]string{"X-Before": "original"}
		opts := []identity.ServiceReadOption{loc, func(*identity.ServiceReadOpts) error {
			options++
			api.RawClient().MoreHeaders["X-Before"] = "mutated"
			api.RawClient().MoreHeaders["X-After"] = "late"
			return nil
		}}
		if deprecated {
			opts = append(opts, identity.WithServiceReadFilters(json.RawMessage(`{}`)))
		}
		result, err := cloud.GetService(context.Background(), "one", opts...)
		if err != nil || result.Service == nil || !strings.Contains(string(result.Service.Resource.Body["location"]), `"cloud":"configured"`) || !strings.Contains(string(result.Service.Resource.Body["location"]), `"zone":"az"`) {
			t.Fatal(result, err)
		}
	}
	if calls != 2 || options != 2 {
		t.Fatal(calls, options)
	}
}
func TestPythonServiceCloudGetLateHTTPBeforeDeprecatedLocalErrorsAndEmptySelection(t *testing.T) {
	for _, mode := range []string{"late", "empty", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cloud, _ := serviceCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if mode == "empty" {
					testcloud.JSON(w, 200, `{"services":[]}`)
					return
				}
				if calls == 1 {
					testcloud.JSON(w, 203, `{"services":[{"id":"a"}],"next":"?marker=a"}`)
					return
				}
				w.Header().Set("X-Request-Id", "late-get")
				if mode == "late" {
					testcloud.JSON(w, 503, `{"error":"late physical"}`)
				} else {
					testcloud.JSON(w, 200, `{"services":[{"id":"b"}]}`)
				}
			})
			raw := json.RawMessage(`"["`)
			if mode == "empty" {
				raw = json.RawMessage(`true`)
			}
			result, err := cloud.GetService(context.Background(), "", identity.WithServiceReadFilters(raw))
			if result == nil || result.Service != nil || !result.DeprecatedFilters {
				t.Fatal(result, err)
			}
			switch mode {
			case "late":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 503 || proof.ResponseHeader.Get("X-Request-Id") != "late-get" || len(result.Inventory) != 1 || calls != 2 {
					t.Fatal(result, err, calls)
				}
			case "invalid":
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || len(result.Inventory) != 2 || calls != 2 {
					t.Fatal(result, err, calls)
				}
			case "empty":
				if err != nil || string(result.Value) != "null" || calls != 1 {
					t.Fatal(result, err, calls)
				}
			}
		})
	}
}
