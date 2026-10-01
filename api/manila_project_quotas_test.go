package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
	"gophercloudsdk/sharedfilesystems/v2/quotasets"
)

func newManilaQuotaScope(t *testing.T, cloud *testcloud.Cloud, version string) *quotasets.ProjectQuotaScope {
	t.Helper()
	client := cloud.Client("shared-file-system", "/manila/v2/catalog-project")
	client.Microversion = version
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("quota-project"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestManilaProjectQuotaExactNamesUseSeparateKeystoneAndFreezeID(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, calls atomic.Int32
	cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-OpenStack-Manila-API-Version") != "" {
			t.Errorf("invalid Keystone request: %s %v", r.URL, r.Header)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"projects":[{"id":"wrong","name":"tenant-other"}],"links":{"next":%q}}`, cloud.Server.URL+"/identity/v3/projects/next"))
	})
	cloud.Mux.HandleFunc("/identity/v3/projects/next", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		testcloud.JSON(w, 200, `{"projects":[{"id":"quota-project","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-sets/quota-project", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Manila-API-Version") != "2.80" {
			t.Errorf("invalid quota request: %s %v", r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-project","shares":0}}`)
	})
	client := cloud.Client("shared-file-system", "/manila/v2/catalog-project")
	client.Microversion = "2.80"
	scope, err := quotasets.New(client).InProject(context.Background(), resource.Name("tenant"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil || scope.ProjectID() != "quota-project" || lookups.Load() != 2 || calls.Load() != 0 {
		t.Fatalf("scope=%v lookups=%d calls=%d err=%v", scope, lookups.Load(), calls.Load(), err)
	}
	for range 2 {
		value, err := scope.Get(context.Background())
		if err != nil || value.ProjectID != "quota-project" || value.ID != "wire-project" {
			t.Fatalf("value=%+v err=%v", value, err)
		}
	}
	if lookups.Load() != 2 || calls.Load() != 2 {
		t.Fatal("quota calls repeated project resolution")
	}
}

func TestManilaProjectQuotaRecordedV2V3AuthenticationAndNoEndpointGuess(t *testing.T) {
	for _, version := range []string{"v2", "v3", "manual"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var auth gophercloud.AuthResult
			if version == "v2" {
				auth = tokens2.CreateResult{Result: gophercloud.Result{Body: map[string]any{"access": map[string]any{"token": map[string]any{"id": "token", "expires": "2030-01-01T00:00:00Z", "tenant": map[string]any{"id": "quota-project"}}}}}}
			}
			if version == "v3" {
				var result tokens3.CreateResult
				result.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "quota-project"}}}
				result.Header = http.Header{"X-Subject-Token": []string{"token"}}
				auth = result
			}
			if auth != nil {
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := quotasets.New(cloud.Client("shared-file-system", "/manila/v2/quota-project")).CurrentProject(context.Background())
			if version == "manual" {
				if scope != nil || !errors.Is(err, resource.ErrUnsupported) {
					t.Fatalf("scope=%v err=%v", scope, err)
				}
				return
			}
			if err != nil || scope.ProjectID() != "quota-project" {
				t.Fatalf("scope=%v err=%v", scope, err)
			}
		})
	}
}

func TestManilaProjectQuotaLegacyRoutesAndDetailVersionPreflight(t *testing.T) {
	for _, tc := range []struct {
		version, root string
		detail        bool
	}{{"", "os-quota-sets", false}, {"2.6", "os-quota-sets", false}, {"2.7", "quota-sets", false}, {"2.25", "quota-sets", true}, {"latest", "quota-sets", true}} {
		t.Run("version-"+tc.version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			path := "/manila/v2/catalog-project/" + tc.root + "/quota-project"
			cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"quota_set":{"shares":7}}`)
			})
			cloud.Mux.HandleFunc(path+"/defaults", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"quota_set":{"shares":50}}`)
			})
			cloud.Mux.HandleFunc(path+"/detail", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"quota_set":{"shares":{"limit":7,"in_use":1,"reserved":2}}}`)
			})
			scope := newManilaQuotaScope(t, cloud, tc.version)
			if value, err := scope.Get(context.Background()); err != nil || *value.Shares != 7 {
				t.Fatalf("value=%v err=%v", value, err)
			}
			if value, err := scope.Defaults(context.Background()); err != nil || *value.Shares != 50 {
				t.Fatalf("defaults=%v err=%v", value, err)
			}
			value, err := scope.Detail(context.Background())
			if tc.detail {
				if err != nil || *value.Shares.InUse != 1 || *value.Shares.Reserved != 2 || calls.Load() != 3 {
					t.Fatalf("detail=%v calls=%d err=%v", value, calls.Load(), err)
				}
			} else if value != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 2 {
				t.Fatalf("detail=%v calls=%d err=%v", value, calls.Load(), err)
			}
		})
	}
}

func TestManilaProjectQuotaResponsePreservesNullOmissionExtensionsAndExactInts(t *testing.T) {
	cloud := testcloud.New(t)
	path := "/manila/v2/catalog-project/quota-sets/quota-project"
	cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Quota-Request", "limits")
		testcloud.JSON(w, 200, `{"quota_set":{"id":"wire","shares":9007199254740993,"snapshots":null,"gigabytes":-1,"vendor":{"counter":9223372036854775807}}}`)
	})
	cloud.Mux.HandleFunc(path+"/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Quota-Request", "detail")
		testcloud.JSON(w, 200, `{"quota_set":{"shares":{"limit":9007199254740993,"in_use":null,"reserved":0,"vendor":9007199254740995},"snapshots":null}}`)
	})
	scope := newManilaQuotaScope(t, cloud, "2.80")
	value, err := scope.Get(context.Background())
	if err != nil || value.ProjectID != "quota-project" || value.ID != "wire" || *value.Shares != 9007199254740993 || value.Snapshots != nil || value.Backups != nil || string(value.Body["snapshots"]) != "null" || value.Body["backups"] != nil || !strings.Contains(string(value.Body["vendor"]), "9223372036854775807") || value.StatusCode != 200 || value.Header.Get("X-Quota-Request") != "limits" {
		t.Fatalf("quota=%+v err=%v", value, err)
	}
	detail, err := scope.Detail(context.Background())
	if err != nil || *detail.Shares.Limit != 9007199254740993 || detail.Shares.InUse != nil || *detail.Shares.Reserved != 0 || detail.Snapshots != nil || !strings.Contains(string(detail.Body["shares"]), "9007199254740995") || detail.StatusCode != 200 || detail.Header.Get("X-Quota-Request") != "detail" {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
}

func TestManilaProjectQuotaUpdateSnapshotsPointersAndNestedExtensionsExactly(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-sets/quota-project", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		quota := body["quota_set"]
		if r.Method != "PUT" || r.URL.RawQuery != "" || string(quota["shares"]) != "9007199254740993" || string(quota["snapshots"]) != "0" || string(quota["gigabytes"]) != "-1" || string(quota["force"]) != "false" || quota["backups"] != nil || string(quota["vendor"]) != `{"counter":9007199254740995}` || string(quota["other"]) != `{"mode":"old"}` {
			t.Errorf("unexpected body=%v method=%s url=%s", quota, r.Method, r.URL)
		}
		testcloud.JSON(w, 200, `{"quota_set":{"shares":9007199254740993}}`)
	})
	shares, snapshots, gigabytes := int64(9007199254740993), int64(0), int64(-1)
	extra := map[string]any{"vendor": map[string]any{"counter": int64(9007199254740995)}}
	option := quotasets.WithQuotaOptions(quotasets.UpdateOpts{Shares: &shares, Snapshots: &snapshots, Gigabytes: &gigabytes, Extra: extra})
	other := map[string]any{"mode": "old"}
	field := quotasets.WithUpdateField("other", other)
	shares, snapshots, gigabytes = 99, 99, 99
	extra["vendor"].(map[string]any)["counter"] = 99
	other["mode"] = "new"
	scope := newManilaQuotaScope(t, cloud, "2.80")
	for range 2 {
		value, err := scope.Update(context.Background(), quotasets.UpdateOpts{}, option, quotasets.WithUpdateForce(false), field)
		if err != nil || *value.Shares != 9007199254740993 {
			t.Fatalf("value=%+v err=%v", value, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("unexpected update call count")
	}
}

func TestManilaProjectQuotaUpdateInvalidInputsMakeNoRequests(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid input sent %s", r.URL) })
	scope := newManilaQuotaScope(t, cloud, "2.80")
	bad := int64(-2)
	if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{Shares: &bad}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, key := range []string{"shares", "force", "id", "project_id", "user_id", "share_type", "usage", "reservation", "quota_set", " "} {
		for _, option := range []quotasets.UpdateOption{quotasets.WithUpdateField(key, 1), quotasets.WithQuotaOptions(quotasets.UpdateOpts{Extra: map[string]any{key: 1}})} {
			if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{}, option); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("key=%q err=%v", key, err)
			}
		}
	}
	for _, option := range []quotasets.UpdateOption{nil, request.WithQuery[quotasets.UpdateOpts]("user_id", "user"), request.WithArgument[quotasets.UpdateOpts]("other", 1), quotasets.WithUpdateField("bad", make(chan int))} {
		if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{}, option); err == nil {
			t.Fatal("invalid option accepted")
		}
	}
}

func TestManilaProjectQuotaStrictStatusResetAndHTTPDecodeCauses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		reset  bool
		ignore bool
		want   error
	}{
		{"get203", 203, `{"quota_set":{}}`, false, false, nil}, {"get202", 202, `{"quota_set":{}}`, false, false, nil}, {"get404", 404, `{}`, false, false, resource.ErrNotFound}, {"getMalformed", 200, `{"quota_set":{"shares":"bad"}}`, false, false, nil}, {"getOverflow", 200, `{"quota_set":{"shares":9223372036854775808}}`, false, false, nil}, {"getNull", 200, `{"quota_set":null}`, false, false, nil}, {"reset204", 204, ``, true, false, nil}, {"reset200", 200, ``, true, false, nil}, {"reset404", 404, `{}`, true, false, resource.ErrNotFound}, {"resetIgnore404", 404, `{}`, true, true, nil}, {"resetIgnore403", 403, `{}`, true, true, nil}, {"reset202", 202, ``, true, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-sets/quota-project", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Reset", "accepted")
				testcloud.JSON(w, tc.status, tc.body)
			})
			scope := newManilaQuotaScope(t, cloud, "2.80")
			var err error
			if tc.reset {
				result, resetErr := scope.Reset(context.Background(), quotasets.WithResetIgnoreMissing(tc.ignore))
				err = resetErr
				if tc.name == "reset202" && (err != nil || result.ProjectID != "quota-project" || result.StatusCode != 202 || result.Header.Get("X-Reset") != "accepted") {
					t.Fatalf("result=%v err=%v", result, err)
				}
				if tc.name == "resetIgnore404" && (err != nil || result != nil) {
					t.Fatalf("result=%v err=%v", result, err)
				}
			} else {
				_, err = scope.Get(context.Background())
			}
			if tc.name != "reset202" && tc.name != "resetIgnore404" && err == nil {
				t.Fatal("invalid response accepted")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
			if tc.status >= 400 && tc.name != "resetIgnore404" {
				var httpCause gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &httpCause) || httpCause.Actual != tc.status {
					t.Fatalf("HTTP cause lost: %v", err)
				}
			}
			if tc.name == "getMalformed" || tc.name == "getOverflow" {
				var cause *json.UnmarshalTypeError
				if !errors.As(err, &cause) {
					t.Fatalf("decode cause lost: %v", err)
				}
			}
			if calls.Load() != 1 {
				t.Fatal("operation issued follow-up request")
			}
		})
	}
}

func TestManilaProjectQuotaPreflightAndCancellationPreserveContext(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-sets/quota-project", func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	scope := newManilaQuotaScope(t, cloud, "2.80")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := scope.Get(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("context cause lost: %v", err)
	}
	for _, version := range []string{"3.1", "2.bad", "2.-1", "2.025"} {
		client := cloud.Client("shared-file-system", "/manila")
		client.Microversion = version
		if _, err := quotasets.New(client).InProject(context.Background(), resource.ID("p")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("version=%s err=%v", version, err)
		}
	}
	if _, err := quotasets.New(nil).InProject(context.Background(), resource.ID("p")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := quotasets.New(cloud.Client("shared-file-system", "/manila")).InProject(context.Background(), resource.Name("tenant")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	var zero quotasets.ProjectQuotaScope
	if _, err := zero.Reset(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if _, err := zero.Reset(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
