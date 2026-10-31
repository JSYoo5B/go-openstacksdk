package projects_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func projectRecordAPI(t *testing.T, reply func(*http.Request) *http.Response) (*projects.API, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) { return reply(req), nil }))
	client := fixture.Client("identity", "/catalog/")
	client.ResourceBase = fixture.Server.URL + "/keystone/v3/"
	return projects.New(client), fixture
}
func projectRecordWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"project-proof"}})
}
func projectRecordField(t *testing.T, r *projects.ProjectRecord, key, want string) {
	t.Helper()
	if r == nil || r.Resource == nil || string(r.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, r)
	}
}
func TestPythonProjectRecordsDeclaredViewKeepsPhysicalEvidenceAndPresence(t *testing.T) {
	calls := 0
	body := `{"projects":[{"id":"a","name":"alpha","enabled":false,"is_enabled":true,"is_domain":"","options":5,"tags":"one","links":{"self":"/v3/projects/a"},"location":{"forged":1},"x_extension":{"n":9007199254740993}},{"id":"b","enabled":null,"tags":null}]}`
	api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != "GET" || r.URL.Path != pythonProjectBase || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.Method, r.URL, r.Header, r.Body)
		}
		return projectRecordWire(203, body)
	})
	records, err := api.AllRecords(context.Background())
	if err != nil || len(records) != 2 || calls != 1 {
		t.Fatal(records, err, calls)
	}
	for k, w := range map[string]string{"is_enabled": "true", "is_domain": "false", "options": "{}", "tags": `["one"]`, "links": `{"self":"/v3/projects/a"}`, "location": "null", "x_extension": `{"n":9007199254740993}`} {
		projectRecordField(t, records[0], k, w)
	}
	projectRecordField(t, records[1], "is_enabled", "null")
	projectRecordField(t, records[1], "options", "null")
	projectRecordField(t, records[1], "tags", "null")
	if string(records[0].Wire.Body["enabled"]) != "false" || string(records[0].Envelope) != body || records[0].Header.Get("X-Request-Id") != "project-proof" || records[0].StatusCode != 203 {
		t.Fatal(records[0])
	}
	records[0].Resource.Body["x_extension"][0] = '['
	records[0].Header.Set("X-Request-Id", "caller")
	if string(records[0].Wire.Body["x_extension"]) != `{"n":9007199254740993}` || records[0].Wire.Header.Get("X-Request-Id") != "project-proof" || records[1].Header.Get("X-Request-Id") != "project-proof" {
		t.Fatal(records)
	}
}
func TestPythonProjectRecordsQueryAliasesAndOrderedLoadedLocalFilters(t *testing.T) {
	var query url.Values
	api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
		query = r.URL.Query()
		return projectRecordWire(200, `{"projects":[{"id":"a","description":0.10000000000000001,"options":{"enabled":1.0}},{"id":"b","description":0.2,"options":{"enabled":false}}]}`)
	})
	records, err := api.AllRecords(context.Background(), projects.WithRecordListQuery("is_enabled", false), projects.WithRecordListQuery("enabled", true), projects.WithRecordListQuery("tags-any", []string{"a", "b"}), projects.WithRecordListQuery("description", json.RawMessage(`0.1`)), projects.WithRecordListQuery("options", map[string]any{"enabled": true}), projects.WithRecordListQuery("x_unknown", func() {}))
	if err != nil || len(records) != 1 {
		t.Fatal(records, err)
	}
	if !reflect.DeepEqual(query, url.Values{"enabled": {"False"}, "tags-any": {"a", "b"}}) {
		t.Fatal(query)
	}
	for _, raw := range []string{`"truthy"`, `[1]`, `1`, `true`, `{"any":1}`} {
		api, _ := projectRecordAPI(t, func(*http.Request) *http.Response {
			return projectRecordWire(200, `{"projects":[{"id":"a","links":`+raw+`}]}`)
		})
		rows, err := api.AllRecords(context.Background(), projects.WithRecordListQuery("links", map[string]any{}))
		if err != nil || len(rows) != 1 {
			t.Fatal(raw, rows, err)
		}
	}
}
func TestPythonProjectRecordsLimitMarkerWithoutLinkRawCapAndBreak(t *testing.T) {
	calls := 0
	var markers []string
	api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		markers = append(markers, r.URL.Query().Get("marker"))
		if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("domain_id") != "d" {
			t.Fatal(r.URL)
		}
		switch calls {
		case 1:
			return projectRecordWire(200, `{"projects":[{"id":"a","description":"skip"},{"id":"b","description":"keep"}]}`)
		case 2:
			return projectRecordWire(200, `{"projects":[{"id":"c","description":"keep"}]}`)
		default:
			return projectRecordWire(200, `{"projects":[]}`)
		}
	})
	rows, err := api.AllRecords(context.Background(), projects.WithRecordListLimit(2), projects.WithRecordListQuery("domain_id", "d"), projects.WithRecordListQuery("description", "keep"))
	if err != nil || len(rows) != 2 || !reflect.DeepEqual(markers, []string{"", "b", "c"}) {
		t.Fatal(rows, err, markers)
	}
	calls = 0
	markers = nil
	rows, err = api.AllRecords(context.Background(), projects.WithRecordListLimit(2), projects.WithRecordListQuery("domain_id", "d"), projects.WithRecordListQuery("description", "keep"), projects.WithRecordListMaxItems(2))
	if err != nil || len(rows) != 1 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	calls = 0
	for row, err := range api.ListRecords(context.Background(), projects.WithRecordListLimit(2), projects.WithRecordListQuery("domain_id", "d")) {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
		break
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}
func TestPythonProjectRecordsKeystoneDictionaryNextAndRouteGuards(t *testing.T) {
	for _, mode := range []string{"dictionary", "foreign", "filter change", "cycle"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if calls > 1 && mode != "cycle" {
					return projectRecordWire(200, `{"projects":[]}`)
				}
				next := "?marker=a&domain_id=d"
				if mode == "foreign" {
					next = "https://foreign.invalid/v3/projects?marker=a&domain_id=d"
				}
				if mode == "filter change" {
					next = "?marker=a&domain_id=changed"
				}
				if mode == "dictionary" {
					return projectRecordWire(200, `{"projects":[{"id":"a"}],"links":{"next":"`+next+`"}}`)
				}
				return projectRecordWire(200, `{"projects":[{"id":"a"}],"next":"`+next+`"}`)
			})
			rows, err := api.AllRecords(context.Background(), projects.WithRecordListQuery("domain_id", "d"))
			if mode == "dictionary" {
				if err != nil || len(rows) != 1 || calls != 1 {
					t.Fatal(rows, err, calls)
				}
			} else {
				var proof *resource.ResponseError
				if err == nil || rows != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "project-proof" {
					t.Fatal(rows, err, proof)
				}
			}
		})
	}
}
func TestPythonProjectFindRecordsGETFirstSeedAliasAndCleanFallback(t *testing.T) {
	for _, status := range []int{200, 400, 403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var urls []string
			api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
				urls = append(urls, r.URL.RequestURI())
				if len(urls) == 1 {
					if status == 200 {
						return projectRecordWire(status, `{"project":{"name":"different","enabled":false}}`)
					}
					return projectRecordWire(status, `{"error":"fallback"}`)
				}
				return projectRecordWire(200, `{"projects":[{"id":"a","name":"demo","description":"local","enabled":true}]}`)
			})
			row, err := api.FindRecord(context.Background(), "demo", projects.WithRecordFindQuery("domain_id", "d"), projects.WithRecordFindQuery("is_enabled", true), projects.WithRecordFindQuery("enabled", false), projects.WithRecordFindQuery("description", "local"), projects.WithRecordFindQuery("x_seed", map[string]any{"n": 1}))
			if err != nil || row == nil {
				t.Fatal(row, err, urls)
			}
			if status == 200 {
				projectRecordField(t, row, "id", `"demo"`)
				projectRecordField(t, row, "is_enabled", "false")
				projectRecordField(t, row, "x_seed", `{"n":1}`)
				if len(urls) != 1 {
					t.Fatal(urls)
				}
			} else {
				projectRecordField(t, row, "id", `"a"`)
				if len(urls) != 2 {
					t.Fatal(urls)
				}
				u, _ := url.Parse(urls[1])
				want := url.Values{"domain_id": {"d"}, "enabled": {"True"}, "name": {"demo"}}
				if !reflect.DeepEqual(u.Query(), want) {
					t.Fatal(u.Query(), want)
				}
			}
		})
	}
}
func TestPythonProjectFindRecordsDefaultMissingAmbiguousAndLateHTTPProof(t *testing.T) {
	for _, mode := range []string{"default missing", "strict missing", "duplicate", "late failure"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := projectRecordAPI(t, func(*http.Request) *http.Response {
				calls++
				if calls == 1 {
					return projectRecordWire(404, `{"error":"not id"}`)
				}
				if mode == "default missing" || mode == "strict missing" {
					return projectRecordWire(200, `{"projects":[]}`)
				}
				if calls == 2 {
					return projectRecordWire(200, `{"projects":[{"id":"a","name":"demo"}],"next":"?marker=a&name=demo"}`)
				}
				if mode == "duplicate" {
					return projectRecordWire(200, `{"projects":[{"id":"b","name":"demo"}]}`)
				}
				return projectRecordWire(503, `{"error":"late actual"}`)
			})
			options := []projects.RecordFindOption{}
			if mode == "strict missing" {
				options = append(options, projects.WithRecordFindIgnoreMissing(false))
			}
			row, err := api.FindRecord(context.Background(), "demo", options...)
			if mode == "default missing" {
				if row != nil || err != nil {
					t.Fatal(row, err)
				}
				return
			}
			want := resource.ErrNotFound
			if mode == "duplicate" {
				want = resource.ErrAmbiguous
			}
			if mode == "late failure" {
				var proof gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.As(err, &proof) || proof.Actual != 503 || string(proof.Body) != `{"error":"late actual"}` || proof.ResponseHeader.Get("X-Request-Id") != "project-proof" || proof.Method != http.MethodGet || !strings.Contains(proof.URL, "/keystone/v3/projects?marker=a") {
					t.Fatal(row, err, proof)
				}
				return
			}
			if !errors.Is(err, want) {
				t.Fatal(row, err, want)
			}
		})
	}
}
func TestPythonProjectRecordsProxyOpaqueJMESIsUnreadAndRetainsGenerator(t *testing.T) {
	calls := 0
	api, _ := projectRecordAPI(t, func(*http.Request) *http.Response {
		calls++
		return projectRecordWire(200, `{"projects":[{"id":"a"}]}`)
	})
	for expression, want := range map[string]string{"[0].id": "null", "`[]`": "[]", "`[1,false,null]`": "[1,false,null]"} {
		result, err := api.ProxyListRecords(context.Background(), projects.WithRecordListJMESPath(expression))
		if err != nil || string(result.JSON) != want || calls != 0 {
			t.Fatal(expression, result, err, calls)
		}
	}
	result, err := api.ProxyListRecords(context.Background(), projects.WithRecordListJMESPath("@"))
	g, ok := result.Value.(*projects.RecordListGenerator)
	if err != nil || !ok || result.JSON != nil || calls != 0 {
		t.Fatal(result, err, calls)
	}
	for row, err := range g.Records {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	result, err = api.ProxyListRecords(context.Background(), projects.WithRecordListJMESPath("to_array(@)"))
	if err != nil || result.JSON != nil {
		t.Fatal(result, err)
	}
	if _, err = api.ProxyListRecords(context.Background(), projects.WithRecordListJMESPath("length(@)")); !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
		t.Fatal(err, calls)
	}
}
func TestPythonProjectRecordsOptionsSnapshotAndSourceCancellationPreflight(t *testing.T) {
	calls := 0
	var query url.Values
	api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		query = r.URL.Query()
		if r.Header.Get("X-Custom") != "owned" {
			t.Fatal(r.Header)
		}
		return projectRecordWire(200, `{"projects":[]}`)
	})
	headers := map[string]string{"X-Custom": "owned"}
	queryMap := map[string]json.RawMessage{"domain_id": json.RawMessage(`"owned"`)}
	option := projects.WithRecordListOpts(projects.RecordListOpts{Query: queryMap, Headers: headers})
	headers["X-Custom"] = "changed"
	queryMap["domain_id"][1] = 'X'
	_, err := api.AllRecords(context.Background(), option)
	if err != nil || calls != 1 || query.Get("domain_id") != "owned" {
		t.Fatal(err, calls, query)
	}
	for _, mode := range []string{"nil context", "nil option", "source changed", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			ctx := context.Background()
			options := []projects.RecordListOption{}
			cause := errors.New("caller canceled project")
			if mode == "nil context" {
				ctx = nil
			}
			if mode == "nil option" {
				options = append(options, nil)
			}
			if mode == "source changed" {
				old := api.RawClient().Endpoint
				options = append(options, func(*projects.RecordListOpts) error {
					api.RawClient().Endpoint = "https://changed.invalid/"
					return nil
				})
				defer func() { api.RawClient().Endpoint = old }()
			}
			if mode == "cancel" {
				next, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = next
			}
			rows, err := api.AllRecords(ctx, options...)
			var proof *resource.ResponseError
			if rows != nil || err == nil || errors.As(err, &proof) || calls != before {
				t.Fatal(rows, err, proof, calls)
			}
			if mode == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal(err)
			}
		})
	}
}

func TestPythonProjectRecordsPhysicalReadCloseNative401AndCanceledGuardNeverFallback(t *testing.T) {
	for _, status := range []int{200, 400, 403, 404, 401, 429, 498, 503} {
		for _, mode := range []string{"read", "close", "both", "restore cancel"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				calls, retries, reauth, backoff := 0, 0, 0, 0
				readCause := errors.New("project physical read")
				closeCause := errors.New("project physical close")
				cancelCause := errors.New("project caller cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				body := &testcloud.Body{Data: `{"project":{"id":"a"}}`}
				if mode == "read" || mode == "both" {
					body.ReadErr = readCause
				}
				if mode == "close" || mode == "both" {
					body.CloseErr = closeCause
				}
				var api *projects.API
				api, fixture := projectRecordAPI(t, func(r *http.Request) *http.Response {
					calls++
					response := projectRecordWire(status, body.Data)
					response.Body = body
					return response
				})
				if mode == "restore cancel" {
					old := api.RawClient().Endpoint
					body.OnRead = func() { api.RawClient().Endpoint = "https://changed.invalid/"; cancel(cancelCause) }
					body.CloseErr = closeCause
					defer func() { api.RawClient().Endpoint = old }()
				}
				fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; return nil }
				fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoff++
					return nil
				}
				row, err := api.FindRecord(ctx, "a")
				if err == nil || row != nil || calls != 1 || body.Closes() != 1 || retries != 0 || reauth != 0 || backoff != 0 {
					t.Fatal(row, err, calls, body.Closes(), retries, reauth, backoff)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body.Data || proof.Header.Get("X-Request-Id") != "project-proof" {
						t.Fatal(err, proof)
					}
				} else {
					original := err
					if status == 401 {
						var native *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &native) || native.ErrOriginal == nil || native.ErrReauth == nil {
							t.Fatal(err, native)
						}
						original = native.ErrOriginal
					}
					var proof gophercloud.ErrUnexpectedResponseCode
					if !errors.As(original, &proof) || proof.Actual != status || string(proof.Body) != body.Data || proof.ResponseHeader.Get("X-Request-Id") != "project-proof" || proof.Method != http.MethodGet || !strings.Contains(proof.URL, "/keystone/v3/projects/a") {
						t.Fatal(err, proof)
					}
				}
				if body.ReadErr != nil && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if body.CloseErr != nil && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if mode == "restore cancel" && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption)) {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestPythonProjectRecordsCleanReauthenticationAndVersionHeaderAreIdentityOwned(t *testing.T) {
	calls, reauth := 0, 0
	api, fixture := projectRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Header.Get("OpenStack-API-Version") != "identity 3.1" {
			t.Fatal(r.Header)
		}
		if calls == 1 {
			return projectRecordWire(401, `{"error":"renew"}`)
		}
		if r.Header.Get("X-Auth-Token") != "renewed" {
			t.Fatal(r.Header)
		}
		return projectRecordWire(200, `{"projects":[]}`)
	})
	fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; fixture.Provider.SetToken("renewed"); return nil }
	_, err := api.AllRecords(context.Background(), projects.WithRecordListMicroversion("3.1"))
	if err != nil || calls != 2 || reauth != 1 {
		t.Fatal(err, calls, reauth)
	}
}

func TestPythonProjectFindRecordsListBasePathOnlyAfterFallbackAndVersionPhase(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprint(direct), func(t *testing.T) {
			calls := 0
			var paths, versions []string
			api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				paths = append(paths, r.URL.Path)
				versions = append(versions, r.Header.Get("OpenStack-API-Version"))
				if calls == 1 {
					if direct {
						return projectRecordWire(199, `{"project":{"id":"actual","name":"demo"}}`)
					}
					return projectRecordWire(404, `{"error":"not id"}`)
				}
				return projectRecordWire(199, `{"projects":[{"id":"actual","name":"demo"}]}`)
			})
			api.RawClient().Microversion = "3.1"
			path := "/filtered/projects"
			if direct {
				path = "https://unused.invalid/"
			}
			row, err := api.FindRecord(context.Background(), "demo", projects.WithRecordFindListBasePath(path), projects.WithRecordFindMicroversion("3.2"))
			if err != nil || row == nil || row.StatusCode != 199 || api.RawClient().Microversion != "3.1" {
				t.Fatal(row, err, paths, versions)
			}
			if direct {
				if !reflect.DeepEqual(paths, []string{pythonProjectBase + "/demo"}) || !reflect.DeepEqual(versions, []string{"identity 3.2"}) {
					t.Fatal(paths, versions)
				}
			} else {
				if !reflect.DeepEqual(paths, []string{pythonProjectBase + "/demo", "/keystone/v3/filtered/projects"}) || !reflect.DeepEqual(versions, []string{"identity 3.2", "identity 3.1"}) {
					t.Fatal(paths, versions)
				}
			}
		})
	}
}
func TestPythonProjectRecordsInvalidRepresentationRetainsActualHeaderAndPage(t *testing.T) {
	for _, body := range []string{`{"projects":null}`, `{"projects":[null]}`, `{"projects":[{"id":"a","connection":"collision"}]}`, `{"projects":[{"id":"a","links":3}]}`} {
		api, _ := projectRecordAPI(t, func(*http.Request) *http.Response { return projectRecordWire(200, body) })
		options := []projects.RecordListOption{}
		if strings.Contains(body, `"links":3`) {
			options = append(options, projects.WithRecordListQuery("links", map[string]any{"k": "v"}))
		}
		rows, err := api.AllRecords(context.Background(), options...)
		var proof *resource.ResponseError
		if rows != nil || err == nil || !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "project-proof" {
			t.Fatal(body, rows, err, proof)
		}
	}
}
