package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func endpointCloudFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*identity.EndpointReadCloud, *testcloud.Cloud, *atomic.Int32) {
	t.Helper()
	fixture := testcloud.New(t)
	calls := &atomic.Int32{}
	fixture.Mux.HandleFunc("/identity/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
		}
		w.Header().Set("X-Request-Id", "cloud-endpoint")
		reply(w, r)
	})
	client := fixture.Client("identity", "/catalog/unused/")
	client.ResourceBase = fixture.Server.URL + "/identity/v3/"
	return identity.NewEndpointReadCloud(endpoints.New(client)), fixture, calls
}
func endpointCloudField(t *testing.T, row *endpoints.EndpointRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatal(key, want, row)
	}
}
func endpointCloudExpression(expression string) json.RawMessage {
	raw, _ := json.Marshal(expression)
	return raw
}

func TestPythonEndpointCloudSearchGlobLocalFiltersAndActualRecords(t *testing.T) {
	body := `{"endpoints":[{"id":"a","name":"ops-alpha","interface":"public","enabled":1,"region_id":"RegionOne","service_id":"svc","links":{"flag":1,"n":9007199254740992},"url":"literal","unknown":9007199254740993},{"id":"b","name":"other","interface":"internal","enabled":false},{"id":"a","name":"ops-beta","interface":"public","enabled":"false","region_id":"RegionOne","service_id":"svc","links":{"flag":true,"n":9007199254740992},"url":"literal"}]}`
	cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("r.URL.RawQuery = %v, want %v", r.URL.RawQuery, "")
			return
		}
		if got := r.Header.Values("Accept"); len(got) == 0 || slices.ContainsFunc(got, func(value string) bool { return value != "application/json" }) {
			t.Errorf("header %s = %q, want %q", "Accept", got, "application/json")
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil || len(raw) != 0 || r.ContentLength != 0 {
			t.Error(string(raw), err, r.ContentLength)
		}
		testcloud.JSON(w, 203, body)
	})
	result, err := cloud.SearchEndpoints(context.Background(), "ops-*", identity.WithEndpointReadFilters(json.RawMessage(`{"interface":"public","is_enabled":true,"links":{"flag":true,"n":9007199254740993.0},"region_id":"RegionOne","service_id":"svc","url":"literal"}`)))
	if err != nil || result == nil || len(result.Endpoints) != 2 || len(result.Inventory) != 3 || result.Expression || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	for _, row := range result.Endpoints {
		endpointCloudField(t, row, "id", `"a"`)
		endpointCloudField(t, row, "is_enabled", "true")
		if row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "cloud-endpoint" || string(row.Envelope) != body {
			t.Fatal(row)
		}
	}
	if result.Endpoints[0].Resource.Body["unknown"] != nil || string(result.Endpoints[0].Wire.Body["unknown"]) != "9007199254740993" {
		t.Fatal(result)
	}
	result.Endpoints[0].Resource.Body["id"][1] = 'X'
	result.Endpoints[0].Header.Set("X-Request-Id", "consumer")
	result.Endpoints[0].Envelope[0] = 'X'
	endpointCloudField(t, result.Inventory[0], "id", `"a"`)
	if result.Inventory[0].Header.Get("X-Request-Id") != "cloud-endpoint" || string(result.Inventory[0].Envelope) != body {
		t.Fatal(result.Inventory[0])
	}
}

func TestPythonEndpointCloudSearchFalseyFiltersAndPythonIdentifiers(t *testing.T) {
	cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"endpoints":[{"id":null,"name":false},{"id":1,"name":"True"}]}`)
	})
	for _, raw := range []string{"null", "false", "0", "1e-9999", `""`, "[]", "{}"} {
		result, err := cloud.SearchEndpoints(context.Background(), "", identity.WithEndpointReadFilters(json.RawMessage(raw)))
		if err != nil || result == nil || len(result.Endpoints) != 2 || result.Expression {
			t.Fatal(raw, result, err)
		}
	}
	for _, id := range []string{"None", "False", "1", "True"} {
		result, err := cloud.SearchEndpoints(context.Background(), id)
		if err != nil || len(result.Endpoints) != 1 {
			t.Fatal(id, result, err)
		}
	}
	if calls.Load() != 11 {
		t.Fatal(calls.Load())
	}
}

func TestPythonEndpointCloudLocalFilterErrorsFollowEveryPageAndKeepInventory(t *testing.T) {
	for _, raw := range []string{`"["`, `{"unknown":true}`, `{"links":{"x":1}}`, `{invalid`, `true`} {
		cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("marker") == "" {
				testcloud.JSON(w, 203, `{"endpoints":[{"id":"a","links":"truthy"}],"next":"?marker=a"}`)
			} else {
				testcloud.JSON(w, 200, `{"endpoints":[{"id":"b","links":"truthy"}]}`)
			}
		})
		result, err := cloud.SearchEndpoints(context.Background(), "", identity.WithEndpointReadFilters(json.RawMessage(raw)))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 2 || result.Endpoints != nil || result.Value != nil || calls.Load() != 2 || errors.As(err, &proof) {
			t.Fatal(raw, result, err, proof, calls.Load())
		}
		if result.Inventory[0].StatusCode != 203 || result.Inventory[1].StatusCode != 200 {
			t.Fatal(result.Inventory)
		}
	}
	cloud, _, _ := endpointCloudFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"a","name":"other","links":"truthy"}]}`)
	})
	result, err := cloud.SearchEndpoints(context.Background(), "", identity.WithEndpointReadFilters(json.RawMessage(`{"name":"mismatch","links":{"x":1}}`)))
	if err != nil || string(result.Value) != "[]" || result.Endpoints == nil {
		t.Fatal(result, err)
	}
	result, err = cloud.SearchEndpoints(context.Background(), "", identity.WithEndpointReadFilters(json.RawMessage(`{"links":{"x":1},"name":"mismatch"}`)))
	if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 1 {
		t.Fatal(result, err)
	}
	result, err = cloud.SearchEndpoints(context.Background(), "no-match", identity.WithEndpointReadFilters(json.RawMessage(`true`)))
	if err != nil || string(result.Value) != "[]" {
		t.Fatal(result, err)
	}
}

func TestPythonEndpointCloudNestedEmptyFilterUsesActualTruthiness(t *testing.T) {
	cloud, _, _ := endpointCloudFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"empty","links":{}},{"id":"none","links":null},{"id":"truthy","links":{"extra":1}},{"id":"scalar","links":"x"}]}`)
	})
	result, err := cloud.SearchEndpoints(context.Background(), "", identity.WithEndpointReadFilters(json.RawMessage(`{"links":{}}`)))
	if err != nil || len(result.Endpoints) != 2 {
		t.Fatal(result, err)
	}
	endpointCloudField(t, result.Endpoints[0], "id", `"truthy"`)
	endpointCloudField(t, result.Endpoints[1], "id", `"scalar"`)
}

func TestPythonEndpointCloudJMESUsesMaterializedListAndKeepsNoInventedReceipt(t *testing.T) {
	cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("r.URL.RawQuery = %v, want %v", r.URL.RawQuery, "")
			return
		}
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"a","name":"same"},{"id":"b","name":"other"}]}`)
	})
	for _, tc := range []struct{ id, expression, want string }{{"", "[].id", `["a","b"]`}, {"same", "[0].id", `"a"`}, {"", "length(@)", "2"}, {"absent", "length(@)", "0"}, {"", "`null`", "null"}, {"", "`[false,0]`", "[false,0]"}} {
		before := calls.Load()
		result, err := cloud.SearchEndpoints(context.Background(), tc.id, identity.WithEndpointReadFilters(endpointCloudExpression(tc.expression)))
		if err != nil || result == nil || !result.Expression || string(result.Value) != tc.want || result.Endpoints != nil || len(result.Inventory) != 2 || calls.Load() != before+1 {
			t.Fatal(tc, result, err, calls.Load())
		}
	}
}

func TestPythonEndpointCloudGetNoneBypassesDeprecatedFalseySearch(t *testing.T) {
	for _, raw := range []string{"omitted", "null", "false", "0", "1e-9999", `""`, "[]", "{}"} {
		t.Run(raw, func(t *testing.T) {
			member, list := 0, 0
			cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/identity/v3/endpoints/one" {
					member++
					testcloud.JSON(w, 203, `{"endpoint":{"id":"one","name":"direct"}}`)
				} else {
					list++
					if r.URL.Path != "/identity/v3/endpoints" {
						t.Errorf("r.URL.Path = %v, want %v", r.URL.Path, "/identity/v3/endpoints")
						return
					}
					if r.URL.RawQuery != "" {
						t.Errorf("r.URL.RawQuery = %v, want %v", r.URL.RawQuery, "")
						return
					}
					testcloud.JSON(w, 200, `{"endpoints":[{"id":"one","name":"listed"}]}`)
				}
			})
			var options []identity.EndpointReadOption
			if raw != "omitted" {
				options = append(options, identity.WithEndpointReadFilters(json.RawMessage(raw)))
			}
			result, err := cloud.GetEndpoint(context.Background(), "one", options...)
			if err != nil || result == nil || result.Endpoint == nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			deprecated := raw != "omitted" && raw != "null"
			if result.DeprecatedFilters != deprecated {
				t.Fatal(result)
			}
			want := "direct"
			if deprecated {
				want = "listed"
				if member != 0 || list != 1 || len(result.Inventory) != 1 {
					t.Fatal(member, list, result)
				}
			} else if member != 1 || list != 0 {
				t.Fatal(member, list)
			}
			endpointCloudField(t, result.Endpoint, "name", `"`+want+`"`)
		})
	}
}

func TestPythonEndpointCloudGetCleanFallbackExactIDNameDuplicatesAndMissing(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, mode := range []string{"id", "name", "duplicate", "missing", "glob"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/identity/v3/endpoints" {
						testcloud.JSON(w, status, `{"error":"member"}`)
						return
					}
					if r.URL.Query().Get("marker") == "" {
						if r.URL.RawQuery != "" {
							t.Errorf("r.URL.RawQuery = %v, want %v", r.URL.RawQuery, "")
							return
						}
						testcloud.JSON(w, 203, `{"endpoints":[{"id":"first","name":"other"}],"next":"?marker=first"}`)
						return
					}
					body := `{"endpoints":[]}`
					switch mode {
					case "id":
						body = `{"endpoints":[{"id":"needle","name":"other"}]}`
					case "name":
						body = `{"endpoints":[{"id":"actual","name":"needle"}]}`
					case "duplicate":
						body = `{"endpoints":[{"id":"needle"},{"id":"second","name":"needle"}]}`
					case "glob":
						body = `{"endpoints":[{"id":"actual","name":"needle-name"}]}`
					}
					testcloud.JSON(w, 200, body)
				})
				id := "needle"
				if mode == "glob" {
					id = "needle*"
				}
				result, err := cloud.GetEndpoint(context.Background(), id)
				if calls.Load() != 3 || result == nil {
					t.Fatal(result, err, calls.Load())
				}
				if mode == "duplicate" {
					if !errors.Is(err, resource.ErrAmbiguous) || result.Endpoint != nil {
						t.Fatal(result, err)
					}
					return
				}
				if err != nil {
					t.Fatal(result, err)
				}
				if mode == "missing" || mode == "glob" {
					if result.Endpoint != nil || result.Value != nil {
						t.Fatal(result)
					}
					return
				}
				if result.Endpoint == nil || result.Endpoint.StatusCode != 200 || result.Endpoint.Header.Get("X-Request-Id") != "cloud-endpoint" {
					t.Fatal(result)
				}
			})
		}
	}
	cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"a","name":"needle-one"},{"id":"b","name":"needle-two"}]}`)
	})
	result, err := cloud.GetEndpoint(context.Background(), "needle*", identity.WithEndpointReadFilters(json.RawMessage(`false`)))
	if result == nil || !result.DeprecatedFilters || !errors.Is(err, resource.ErrAmbiguous) || calls.Load() != 1 || len(result.Inventory) != 2 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestPythonEndpointCloudGetSeedAliasesWireAndExplicitLocation(t *testing.T) {
	cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); got != "/identity/v3/endpoints/literal%2Fname%20%252F" {
			t.Errorf("r.URL.EscapedPath() = %v, want %v", got, "/identity/v3/endpoints/literal%2Fname%20%252F")
			return
		}
		testcloud.JSON(w, 203, `{"endpoint":{"enabled":true,"is_enabled":false,"links":"literal","location":{"spoof":true},"unknown":9007199254740993}}`)
	})
	location := resource.CloudLocation{}
	cloudName := "devstack"
	location.Cloud = &cloudName
	location.Project.ID = json.RawMessage(`"current"`)
	option := identity.WithEndpointReadLocation(location)
	cloudName = "changed"
	location.Project.ID[1] = 'X'
	result, err := cloud.GetEndpoint(context.Background(), "literal/name %2F", option)
	if err != nil || result == nil || result.Endpoint == nil || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	endpointCloudField(t, result.Endpoint, "id", `"literal/name %2F"`)
	endpointCloudField(t, result.Endpoint, "is_enabled", "false")
	endpointCloudField(t, result.Endpoint, "links", `"literal"`)
	if string(result.Endpoint.Resource.Body["location"]) != `{"cloud":"devstack","region_name":null,"zone":null,"project":{"id":"current","name":null,"domain_id":null,"domain_name":null}}` || string(result.Endpoint.Wire.Body["unknown"]) != "9007199254740993" {
		t.Fatal(result.Endpoint)
	}
	if result.Endpoint.Wire.Body["id"] != nil || result.Endpoint.Resource.Body["unknown"] != nil {
		t.Fatal(result.Endpoint)
	}
	plain, _, _ := endpointCloudFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 203, `{"endpoint":{"id":null}}`) })
	result, err = plain.GetEndpoint(context.Background(), "requested")
	if err != nil || result.Endpoint == nil {
		t.Fatal(result, err)
	}
	endpointCloudField(t, result.Endpoint, "id", "null")
	endpointCloudField(t, result.Endpoint, "location", "null")
}

func TestPythonEndpointCloudGetFilteredArbitraryTruthLenAndIndex(t *testing.T) {
	cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, `{"endpoints":[{"id":"a"}]}`) })
	for _, tc := range []struct {
		expression, want       string
		absent, duplicate, bad bool
	}{
		{expression: "`null`", absent: true}, {expression: "`false`", absent: true}, {expression: "`0`", absent: true}, {expression: "`1e-9999`", absent: true}, {expression: "`[]`", absent: true}, {expression: "`{}`", absent: true},
		{expression: "`[false]`", want: "false"}, {expression: "`[0]`", want: "0"}, {expression: "`[\"\"]`", want: `""`}, {expression: "`[null]`", absent: true}, {expression: "`\"중\"`", want: `"중"`},
		{expression: "`[1,2]`", duplicate: true}, {expression: "`\"중a\"`", duplicate: true}, {expression: "`{\"a\":1,\"b\":2}`", duplicate: true}, {expression: "`true`", bad: true}, {expression: "`1`", bad: true}, {expression: "`{\"0\":1}`", bad: true},
	} {
		before := calls.Load()
		result, err := cloud.GetEndpoint(context.Background(), "", identity.WithEndpointReadFilters(endpointCloudExpression(tc.expression)))
		if result == nil || !result.DeprecatedFilters || result.Endpoint != nil || len(result.Inventory) != 1 || calls.Load() != before+1 {
			t.Fatal(tc, result, err, calls.Load())
		}
		if tc.duplicate {
			if !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(tc, err)
			}
			continue
		}
		if tc.bad {
			if !errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(tc, err)
			}
			continue
		}
		if err != nil || (result.Value == nil) != tc.absent || (!tc.absent && string(result.Value) != tc.want) {
			t.Fatal(tc, result, err)
		}
	}
}

func TestPythonEndpointCloudOwnedHeadersOptionsAndConcurrentReuse(t *testing.T) {
	cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Values("X-Selected"); len(got) == 0 || slices.ContainsFunc(got, func(value string) bool { return value != "before" }) {
			t.Errorf("header %s = %q, want %q", "X-Selected", got, "before")
		}
		if got := r.Header.Values("X-Trace"); len(got) == 0 || slices.ContainsFunc(got, func(value string) bool { return value != "owned" }) {
			t.Errorf("header %s = %q, want %q", "X-Trace", got, "owned")
		}
		if got := r.Header.Values("OpenStack-API-Version"); len(got) == 0 || slices.ContainsFunc(got, func(value string) bool { return value != "identity 3.2" }) {
			t.Errorf("header %s = %q, want %q", "OpenStack-API-Version", got, "identity 3.2")
		}
		if got := r.Header.Get("X-OpenStack-Nova-API-Version"); got != "" {
			t.Errorf("r.Header.Get(\"X-OpenStack-Nova-API-Version\") = %v, want %v", got, "")
			return
		}
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"a"}]}`)
	})
	client := cloud.RawAPI().RawClient()
	client.MoreHeaders = map[string]string{"X-Selected": "before"}
	headers := map[string]string{"X-Trace": "owned"}
	filter := json.RawMessage(`{"id":"a"}`)
	option := identity.WithEndpointReadOpts(identity.EndpointReadOpts{Headers: headers, Filters: &filter})
	headers["X-Trace"] = "changed"
	filter[7] = 'X'
	var workers sync.WaitGroup
	for i := 0; i < 3; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := cloud.SearchEndpoints(context.Background(), "", option, identity.WithEndpointReadMicroversion("3.2"))
			if err != nil || len(result.Endpoints) != 1 {
				t.Error(result, err)
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 3 {
		t.Fatal(calls.Load())
	}
	result, err := cloud.SearchEndpoints(context.Background(), "", identity.WithEndpointReadHeaders(nil), func(c *identity.EndpointReadOpts) error {
		c.Headers["X-Trace"] = "owned"
		client.MoreHeaders["X-Selected"] = "after"
		return nil
	}, identity.WithEndpointReadMicroversion("3.2"))
	if err != nil || len(result.Endpoints) != 1 || calls.Load() != 4 || client.Microversion != "" || client.MoreHeaders["X-Selected"] != "after" {
		t.Fatal(result, err, calls.Load())
	}
}

func TestPythonEndpointCloudPreflightSourceCancelAndOuterGuard(t *testing.T) {
	for _, op := range []string{"search", "get"} {
		for _, lane := range []string{"nil", "source", "cancel", "role", "outer", "auth"} {
			t.Run(op+"/"+lane, func(t *testing.T) {
				cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, `{"endpoints":[]}`) })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("stop endpoint read")
				seen := 0
				options := []identity.EndpointReadOption{}
				switch lane {
				case "nil":
					options = append(options, nil)
				case "source":
					options = append(options, func(*identity.EndpointReadOpts) error {
						cloud.RawAPI().RawClient().ResourceBase += "changed/"
						return nil
					})
				case "cancel":
					cancel(cause)
				case "role":
					cloud.RawAPI().RawClient().Type = "network"
				case "outer":
					ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
				case "auth":
					options = append(options, identity.WithEndpointReadHeaders(map[string]string{"X-Auth-Token": "foreign"}))
				}
				if lane != "auth" {
					options = append(options, func(*identity.EndpointReadOpts) error { seen++; return nil })
				}
				var err error
				if op == "search" {
					_, err = cloud.SearchEndpoints(ctx, "", options...)
				} else {
					_, err = cloud.GetEndpoint(ctx, "literal", options...)
				}
				if err == nil || calls.Load() != 0 || seen != 0 {
					t.Fatal(err, calls.Load(), seen)
				}
				if (lane == "cancel" || lane == "outer") && !errors.Is(err, cause) {
					t.Fatal(err)
				}
				if lane == "role" && !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestPythonEndpointCloudLateNativeErrorWinsOverInvalidLocalFilter(t *testing.T) {
	cloud, _, calls := endpointCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 203, `{"endpoints":[{"id":"a"}],"next":"?marker=a"}`)
		} else {
			testcloud.JSON(w, 503, `{"error":"late"}`)
		}
	})
	result, err := cloud.SearchEndpoints(context.Background(), "", identity.WithEndpointReadFilters(json.RawMessage(`"["`)))
	var native gophercloud.ErrUnexpectedResponseCode
	if result == nil || len(result.Inventory) != 1 || result.Value != nil || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"late"}` || native.ResponseHeader.Get("X-Request-Id") != "cloud-endpoint" || calls.Load() != 2 {
		t.Fatal(result, err, native, calls.Load())
	}
}

type endpointCloudFaultBody struct {
	io.ReadCloser
	cause   error
	closes  *atomic.Int32
	onClose func()
}

func (b *endpointCloudFaultBody) Close() error {
	b.closes.Add(1)
	err := b.ReadCloser.Close()
	if b.onClose != nil {
		b.onClose()
	}
	return errors.Join(err, b.cause)
}

func TestPythonEndpointCloudPhysicalFaultsNeverFallbackRetryOrInventRows(t *testing.T) {
	for _, op := range []string{"search", "get"} {
		for _, status := range []int{203, 401, 404, 429, 503} {
			for _, lane := range []string{"close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", op, status, lane), func(t *testing.T) {
					body := `{"endpoints":[{"id":"one"}]}`
					if op == "get" {
						body = `{"endpoint":{"id":"one"}}`
					}
					cloud, fixture, calls := endpointCloudFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, status, body) })
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					cause := errors.New(lane + " witness")
					closes := &atomic.Int32{}
					base := fixture.Provider.HTTPClient.Transport
					fixture.Provider.HTTPClient.Transport = userCloudHTTP(func(r *http.Request) (*http.Response, error) {
						response, err := base.RoundTrip(r)
						if response != nil {
							b := &endpointCloudFaultBody{ReadCloser: response.Body, closes: closes}
							switch lane {
							case "close":
								b.cause = cause
							case "cancel":
								b.onClose = func() { cancel(cause) }
							case "source":
								b.onClose = func() { cloud.RawAPI().RawClient().ResourceBase += "changed/" }
							}
							response.Body = b
						}
						return response, err
					})
					hooks := 0
					fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						hooks++
						return nil
					}
					fixture.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
					fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
					var err error
					if op == "search" {
						result, failure := cloud.SearchEndpoints(ctx, "")
						err = failure
						if result != nil && (len(result.Inventory) != 0 || result.Value != nil) {
							t.Fatal(result)
						}
					} else {
						result, failure := cloud.GetEndpoint(ctx, "one")
						err = failure
						if result != nil && (result.Endpoint != nil || result.Value != nil) {
							t.Fatal(result)
						}
					}
					if err == nil || calls.Load() != 1 || closes.Load() != 1 || hooks != 0 {
						t.Fatal(err, calls.Load(), closes.Load(), hooks)
					}
					if lane != "source" && !errors.Is(err, cause) {
						t.Fatal(err)
					}
					if lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
					if status < 400 {
						var proof *resource.ResponseError
						if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "cloud-endpoint" {
							t.Fatal(err, proof)
						}
						return
					}
					receipt := err
					if status == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
							t.Fatal(err)
						}
						receipt = auth.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					path := "/identity/v3/endpoints"
					if op == "get" {
						path += "/one"
					}
					if !errors.As(receipt, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != fixture.Server.URL+path || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "cloud-endpoint" {
						t.Fatal(err, native)
					}
				})
			}
		}
	}
}
