package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"gophercloudsdk/blockstorage/v3/limits"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const cinderLimitsBody = `{"limits":{"absolute":{"maxTotalVolumes":10,"totalVolumesUsed":1},"rate":[]}}`

func newCinderLimitsAPI(cloud *testcloud.Cloud, version string) *limits.API {
	client := cloud.Client("block-storage", "/cinder/v3/catalog-project")
	client.Microversion = version
	return limits.New(client)
}

func TestCinderLimitsImplicitFetchWorksAtLegacyVersionAndNativeGetRemains(t *testing.T) {
	for _, version := range []string{"", "3.0", "3.38", "latest"} {
		t.Run("version-"+version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Volume-API-Version") != version {
					t.Errorf("bad implicit limits: %s %s %v", r.Method, r.URL, r.Header)
				}
				testcloud.JSON(w, 200, cinderLimitsBody)
			})
			api := newCinderLimitsAPI(cloud, version)
			value, err := api.Fetch(context.Background())
			if err != nil || value.ProjectID != "" || *value.Absolute.MaxTotalVolumes != 10 || value.StatusCode != 200 {
				t.Fatalf("value=%+v err=%v", value, err)
			}
			native, err := api.Get(context.Background())
			if err != nil || native.Absolute.MaxTotalVolumes != 10 || calls.Load() != 2 {
				t.Fatalf("native=%+v calls=%d err=%v", native, calls.Load(), err)
			}
		})
	}
}

func TestCinderLimitsProjectFilteringRequires339BeforeIdentityOrHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unsupported project limits made HTTP: %s", r.URL)
	})
	for _, version := range []string{"", "3.0", "3.38", "latest"} {
		api := newCinderLimitsAPI(cloud, version)
		for _, ref := range []resource.Ref{resource.ID("p"), resource.Name("tenant")} {
			if _, err := api.InProject(context.Background(), ref, limits.WithIdentityClient(cloud.Client("identity", "/identity/v3"))); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatalf("version=%s err=%v", version, err)
			}
		}
		if _, err := api.CurrentProject(context.Background()); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
		if _, err := api.Fetch(context.Background(), limits.WithGetOptions(limits.GetOpts{ProjectID: "p"})); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
		if _, err := api.Fetch(context.Background(), limits.WithGetQuery("project_id", "p")); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
	}
	for _, version := range []string{"2.39", "3.bad", "3.-1", "3.039"} {
		if _, err := newCinderLimitsAPI(cloud, version).Fetch(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("version=%s err=%v", version, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newCinderLimitsAPI(cloud, "bad").InProject(ctx, resource.Name("tenant")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestCinderLimitsExactProjectNamesUseSeparateKeystoneOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, calls atomic.Int32
	cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-OpenStack-Volume-API-Version") != "" {
			t.Errorf("bad Keystone request: %s %v", r.URL, r.Header)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"projects":[{"id":"wrong","name":"tenant-other"}],"links":{"next":%q}}`, cloud.Server.URL+"/identity/v3/projects/next"))
	})
	cloud.Mux.HandleFunc("/identity/v3/projects/next", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		testcloud.JSON(w, 200, `{"projects":[{"id":"project-fixed","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "project_id=project-fixed" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.70" {
			t.Errorf("bad limits target/version: %s %v", r.URL, r.Header)
		}
		testcloud.JSON(w, 200, cinderLimitsBody)
	})
	scope, err := newCinderLimitsAPI(cloud, "3.70").InProject(context.Background(), resource.Name("tenant"), limits.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil || scope.ProjectID() != "project-fixed" || lookups.Load() != 2 || calls.Load() != 0 {
		t.Fatalf("scope=%v lookups=%d calls=%d err=%v", scope, lookups.Load(), calls.Load(), err)
	}
	for range 2 {
		value, err := scope.Get(context.Background())
		if err != nil || value.ProjectID != "project-fixed" {
			t.Fatalf("value=%v err=%v", value, err)
		}
	}
	if lookups.Load() != 2 || calls.Load() != 2 {
		t.Fatal("project re-resolved")
	}
}

func TestCinderLimitsScopeRechecksVersionBeforeSendingProjectFilter(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, cinderLimitsBody)
	})
	client := cloud.Client("block-storage", "/cinder/v3/catalog-project")
	client.Microversion = "3.39"
	scope, err := limits.New(client).InProject(context.Background(), resource.ID("fixed"))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"latest", "3.38", ""} {
		client.Microversion = version
		if value, err := scope.Get(context.Background()); value != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
			t.Fatal(version, value, err, calls.Load())
		}
	}
	client.Microversion = "3.70"
	if value, err := scope.Get(context.Background()); err != nil || value.ProjectID != "fixed" || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
}

func TestCinderLimitsRecordedAuthenticationFreezesProjectAndRejectsManualTokens(t *testing.T) {
	for _, kind := range []string{"v2", "v3", "manual"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var auth gophercloud.AuthResult
			if kind == "v2" {
				var result tokens2.CreateResult
				result.Body = map[string]any{"access": map[string]any{"token": map[string]any{"id": "token", "expires": "2030-01-01T00:00:00Z", "tenant": map[string]any{"id": "project-fixed"}}}}
				auth = result
			}
			if kind == "v3" {
				var result tokens3.CreateResult
				result.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "project-fixed"}}}
				result.Header = http.Header{"X-Subject-Token": []string{"token"}}
				auth = result
			}
			if auth != nil {
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			api := newCinderLimitsAPI(cloud, "3.39")
			scope, err := api.CurrentProject(context.Background())
			if kind == "manual" {
				if scope != nil || !errors.Is(err, resource.ErrUnsupported) {
					t.Fatalf("scope=%v err=%v", scope, err)
				}
				return
			}
			if err != nil || scope.ProjectID() != "project-fixed" {
				t.Fatalf("scope=%v err=%v", scope, err)
			}
			cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("project_id") != "project-fixed" {
					t.Errorf("scope followed changed auth: %s", r.URL)
				}
				testcloud.JSON(w, 200, cinderLimitsBody)
			})
			var changed tokens3.CreateResult
			changed.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "new-project"}}}
			changed.Header = http.Header{"X-Subject-Token": []string{"new-token"}}
			if err := cloud.Provider.SetTokenAndAuthResult(changed); err != nil {
				t.Fatal(err)
			}
			if _, err := scope.Get(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCinderLimitsQueryInputsCannotWidenFixedProject(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("project_id") != "project+&fixed" || r.URL.Query().Get("vendor") != "literal" || len(r.URL.Query()["project_id"]) != 1 {
			t.Errorf("bad filtered query: %s", r.URL)
		}
		testcloud.JSON(w, 200, cinderLimitsBody)
	})
	api := newCinderLimitsAPI(cloud, "3.39")
	value, err := api.Fetch(context.Background(), limits.WithGetOptions(limits.GetOpts{ProjectID: "project+&fixed"}), limits.WithGetQuery("vendor", "literal"))
	if err != nil || value.ProjectID != "project+&fixed" {
		t.Fatalf("value=%v err=%v", value, err)
	}
	scope, err := api.InProject(context.Background(), resource.ID("project+&fixed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Get(context.Background(), limits.WithGetQuery("vendor", "literal")); err != nil {
		t.Fatal(err)
	}
	for _, option := range []limits.GetOption{limits.WithGetOptions(limits.GetOpts{ProjectID: "other"}), limits.WithGetQuery("project_id", "other"), limits.WithGetQuery("project_id", "project+&fixed"), limits.WithGetQuery("tenant_id", "other"), nil, request.WithField[limits.GetOpts]("force", true), request.WithArgument[limits.GetOpts]("other", true)} {
		if _, err := scope.Get(context.Background(), option); err == nil {
			t.Fatal("scope replacement accepted")
		}
	}
	for _, option := range []limits.GetOption{limits.WithGetQuery("project_id", ""), limits.WithGetOptions(limits.GetOpts{ProjectID: "bad/id"}), func(c *request.Config[limits.GetOpts]) error {
		c.Query["project_id"] = []string{"one", "two"}
		return nil
	}, func(c *request.Config[limits.GetOpts]) error { c.Query["project_id"] = nil; return nil }} {
		if _, err := api.Fetch(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.Fetch(context.Background(), limits.WithGetOptions(limits.GetOpts{ProjectID: "one"}), limits.WithGetQuery("project_id", "one")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("invalid queries made HTTP: %d", calls.Load())
	}
}

func TestCinderLimitsVersionHeadersCannotOverrideSelectedMicroversion(t *testing.T) {
	for _, tc := range []struct {
		name, version, kind string
		headers             map[string]string
		valid               bool
	}{{"matching legacy", "3.39", "block-storage", map[string]string{"x-openstack-volume-api-version": "3.39"}, true}, {"matching generic", "3.70", "volumev3", map[string]string{"OpenStack-API-Version": "volume 3.70"}, true}, {"blank type explicit", "3.39", "", map[string]string{"X-OpenStack-Volume-API-Version": "3.39"}, true}, {"lower legacy", "3.39", "block-storage", map[string]string{"X-OpenStack-Volume-API-Version": "3.38"}, false}, {"lower generic", "3.39", "block-storage", map[string]string{"openstack-api-version": "volume 3.38"}, false}, {"wrong generic service", "3.39", "block-storage", map[string]string{"OpenStack-API-Version": "block-storage 3.39"}, false}, {"suppressed legacy", "3.39", "block-storage", map[string]string{"X-OpenStack-Volume-API-Version": ""}, false}, {"implicit upgraded header", "", "block-storage", map[string]string{"X-OpenStack-Volume-API-Version": "3.39"}, false}, {"blank type no header", "3.39", "", nil, false}, {"wrong client service", "3.39", "compute", nil, false}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("X-OpenStack-Volume-API-Version") != tc.version && r.Header.Get("OpenStack-API-Version") != "volume "+tc.version {
					t.Errorf("actual version header missing: %v", r.Header)
				}
				testcloud.JSON(w, 200, cinderLimitsBody)
			})
			client := cloud.Client(tc.kind, "/cinder/v3/catalog-project")
			client.Microversion = tc.version
			client.MoreHeaders = tc.headers
			api := limits.New(client)
			scope, err := api.InProject(context.Background(), resource.ID("fixed"))
			if !tc.valid {
				if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatalf("calls=%d err=%v", calls.Load(), err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := scope.Get(context.Background()); err != nil || calls.Load() != 1 {
				t.Fatalf("calls=%d err=%v", calls.Load(), err)
			}
		})
	}
}

func TestCinderLimitsResponseKeepsExactOptionalAbsoluteAndRateValues(t *testing.T) {
	cloud := testcloud.New(t)
	body := `{"limits":{"absolute":{"maxTotalVolumes":9007199254740993,"maxTotalSnapshots":null,"maxTotalVolumeGigabytes":-1,"totalVolumesUsed":0,"vendor":9223372036854775807},"rate":[{"uri":"/volumes","regex":".*","vendor":{"counter":9007199254740995},"limit":[{"value":9223372036854775807,"remaining":null,"next-available":1700000000,"verb":"PUT","unit":"MINUTE","vendor":9007199254740993},{"next-available":"2026-10-01T00:00:00Z","remaining":0},{"next-available":null},{}]}],"vendor_root":9007199254740997}}`
	cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Limits", "kept")
		testcloud.JSON(w, 200, body)
	})
	value, err := newCinderLimitsAPI(cloud, "3.70").Fetch(context.Background())
	if err != nil || *value.Absolute.MaxTotalVolumes != 9007199254740993 || value.Absolute.MaxTotalSnapshots != nil || value.Absolute.MaxTotalBackups != nil || *value.Absolute.MaxTotalVolumeGigabytes != -1 || *value.Absolute.TotalVolumesUsed != 0 || string(value.AbsoluteBody["maxTotalSnapshots"]) != "null" || value.AbsoluteBody["maxTotalBackups"] != nil || string(value.AbsoluteBody["vendor"]) != "9223372036854775807" || value.StatusCode != 200 || value.Header.Get("X-Limits") != "kept" || string(value.Body["vendor_root"]) != "9007199254740997" {
		t.Fatalf("limits=%+v err=%v", value, err)
	}
	rules := value.Rate[0].Limits
	if *rules[0].Value != 9223372036854775807 || rules[0].Remaining != nil || string(rules[0].NextAvailable) != "1700000000" || rules[0].Verb != "PUT" || rules[0].Unit != "MINUTE" || string(rules[0].Body["vendor"]) != "9007199254740993" || string(rules[1].NextAvailable) != `"2026-10-01T00:00:00Z"` || *rules[1].Remaining != 0 || string(rules[2].NextAvailable) != "null" || rules[3].NextAvailable != nil || value.Rate[0].URI != "/volumes" || !strings.Contains(string(value.Rate[0].Body["vendor"]), "9007199254740995") {
		t.Fatalf("rate=%+v", value.Rate)
	}
}

func TestCinderLimitsNullAndOmittedSectionsRemainDistinct(t *testing.T) {
	for _, body := range []string{`{"limits":{"absolute":null,"rate":null}}`, `{"limits":{}}`} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
		value, err := newCinderLimitsAPI(cloud, "3.0").Fetch(context.Background())
		if err != nil || value.Absolute != nil || value.Rate != nil || value.AbsoluteBody != nil {
			t.Fatalf("limits=%v err=%v", value, err)
		}
		if strings.Contains(body, "null") {
			if string(value.Body["absolute"]) != "null" || string(value.Body["rate"]) != "null" {
				t.Fatal("null sections lost")
			}
		} else if value.Body["absolute"] != nil || value.Body["rate"] != nil {
			t.Fatal("omitted sections synthesized")
		}
	}
}

func TestCinderLimitsMalformedSuccessRetainsRawBodyHeaderStatusAndCause(t *testing.T) {
	for _, body := range []string{`{"limits":`, `{"limits":null}`, `{"wrong":{}}`, `{"limits":{"absolute":{"maxTotalVolumes":"bad"}}}`, `{"limits":{"absolute":{"maxTotalVolumes":9223372036854775808}}}`, `{"limits":{"rate":[{"limit":[{"remaining":"bad"}]}]}}`, `{"limits":{"rate":[null]}}`, `{"limits":{"rate":[{"limit":[null]}]}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Limits", "decode-failed")
				testcloud.JSON(w, 200, body)
			})
			value, err := newCinderLimitsAPI(cloud, "3.39").Fetch(context.Background())
			var response *limits.LimitsResponseError
			if value != nil || !errors.As(err, &response) || string(response.Body) != body || response.Header.Get("X-Limits") != "decode-failed" || response.StatusCode != 200 {
				t.Fatalf("value=%v response=%+v err=%v", value, response, err)
			}
			if strings.Contains(body, `"bad"`) || strings.Contains(body, "9223372036854775808") {
				var decode *json.UnmarshalTypeError
				if !errors.As(err, &decode) {
					t.Fatalf("JSON cause lost: %v", err)
				}
			}
		})
	}
}

func TestCinderLimitsStrictHTTPStatusAnd404PreserveNativeCause(t *testing.T) {
	for _, status := range []int{202, 203, 204, 403, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, status, cinderLimitsBody) })
			_, err := newCinderLimitsAPI(cloud, "3.39").Fetch(context.Background())
			var cause gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &cause) || cause.Actual != status {
				t.Fatalf("native cause lost: %v", err)
			}
			if status == 404 && !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
		})
	}
}

func TestCinderLimitsPartialReadRetainsAcceptedHTTPMetadata(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "99")
		w.Header().Set("X-Limits", "partial")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"limits":`))
	})
	_, err := newCinderLimitsAPI(cloud, "3.39").Fetch(context.Background())
	var response *limits.LimitsResponseError
	if !errors.Is(err, io.ErrUnexpectedEOF) || !errors.As(err, &response) || string(response.Body) != `{"limits":` || response.StatusCode != 200 || response.Header.Get("X-Limits") != "partial" {
		t.Fatalf("response=%+v err=%v", response, err)
	}
}

func TestCinderLimitsRedirectsCannotDropProjectAndReauthenticationKeepsQuery(t *testing.T) {
	for _, redirect := range []bool{true, false} {
		t.Run(map[bool]string{true: "redirect", false: "reauth"}[redirect], func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, reauth atomic.Int32
			cloud.Provider.ReauthFunc = func(ctx context.Context) error { reauth.Add(1); cloud.Provider.SetToken("fresh-token"); return nil }
			path := "/cinder/v3/catalog-project/limits"
			cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				if r.URL.RawQuery != "project_id=fixed" {
					t.Errorf("project filter dropped: %s", r.URL)
				}
				if redirect {
					http.Redirect(w, r, path, 307)
					return
				}
				if call == 1 {
					testcloud.JSON(w, 401, `{}`)
					return
				}
				if r.Header.Get("X-Auth-Token") != "fresh-token" {
					t.Error("stale token")
				}
				testcloud.JSON(w, 200, cinderLimitsBody)
			})
			scope, err := newCinderLimitsAPI(cloud, "3.39").InProject(context.Background(), resource.ID("fixed"))
			if err != nil {
				t.Fatal(err)
			}
			value, err := scope.Get(context.Background())
			if redirect {
				if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
					t.Fatalf("calls=%d err=%v", calls.Load(), err)
				}
			} else if err != nil || value.ProjectID != "fixed" || calls.Load() != 2 || reauth.Load() != 1 {
				t.Fatalf("value=%v calls=%d reauth=%d err=%v", value, calls.Load(), reauth.Load(), err)
			}
		})
	}
}

func TestCinderLimitsScopeOnlyFetchesAndPreservesContextCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/cinder/v3/catalog-project/limits", func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	api := newCinderLimitsAPI(cloud, "3.39")
	scope, err := api.InProject(context.Background(), resource.ID("fixed"))
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"List", "Find", "Update", "Delete", "Reset", "Wait"} {
		if _, present := reflect.TypeOf(scope).MethodByName(method); present {
			t.Errorf("limits scope exposes %s", method)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := scope.Get(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	var zero limits.ProjectLimitsScope
	if _, err := zero.Get(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if _, err := zero.Get(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := limits.New(nil).Fetch(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
