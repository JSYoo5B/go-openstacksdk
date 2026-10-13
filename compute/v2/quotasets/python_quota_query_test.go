package quotasets_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestNovaQuotaRecordProjectUserUsageAndArbitraryFetchQueries(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || (r.Body != nil && r.Body != http.NoBody) || r.URL.Query().Get("a") != "False" || strings.Join(r.URL.Query()["fields"], ",") != "cores,ram" || r.URL.Query().Has("usage") {
			t.Error(r.Method, r.URL, r.Body)
		}
		if r.URL.Query().Get("user_id") != "" && r.URL.Query().Get("user_id") != "user" {
			t.Error(r.URL)
		}
		body := `{"quota_set":{"id":"foreign","cores":"12","force":1e-400,"injected_files":5}}`
		if strings.HasSuffix(r.URL.Path, "/detail") {
			body = `{"quota_set":{"cores":{"limit":20,"in_use":3,"reserved":null},"custom":{"limit":9007199254740993,"in_use":2}}}`
		}
		testcloud.JSON(w, 203, body)
	})
	client := cloud.Client("compute", "/compute/v2.1/")
	client.Microversion = "2.96"
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	query := []quotasets.QuotaReadOption{quotasets.WithQuotaReadQuery("fields", []string{"cores", "ram"}), quotasets.WithQuotaReadQuery("a", false)}
	got, err := scope.GetRecord(context.Background(), query...)
	if err != nil || got.StatusCode != 203 || string(got.Resource.Body["cores"]) != "12" || string(got.Resource.Body["force"]) != "false" || string(got.Resource.Body["id"]) != `"foreign"` || got.ProjectID != "project" || got.Microversion != "2.96" {
		t.Fatal(got, err)
	}
	user, err := scope.InUser(context.Background(), resource.ID("user"))
	if err != nil {
		t.Fatal(err)
	}
	query = append(query, quotasets.WithQuotaUsage(true))
	detail, err := user.GetRecord(context.Background(), query...)
	if err != nil || detail.UserID != "user" || string(detail.Resource.Body["cores"]) != "20" || string(detail.Resource.Body["usage"]) != `{"cores":3,"custom":2}` || string(detail.Resource.Body["reservation"]) != `{"cores":null}` || string(detail.Resource.Body["custom"]) != "9007199254740993" {
		t.Fatal(detail, err)
	}
	if calls != 2 || client.Microversion != "2.96" {
		t.Fatal(calls, client)
	}
}
func TestNovaQuotaRecordDefaultDiscoverySelectsCeilingAndRetainsReceipts(t *testing.T) {
	for _, tc := range []struct{ maximum, want string }{{"2.110", "2.56"}, {"2.40", "2.40"}} {
		t.Run(tc.maximum, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path == "/compute/v2.1/" {
					if r.Header.Get("OpenStack-API-Version") != "" {
						t.Error(r.Header)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":%q,"links":[{"rel":"self","href":%q}]}}`, tc.maximum, cloud.Server.URL+"/compute/v2.1/"))
					return
				}
				if r.Header.Get("OpenStack-API-Version") != "compute "+tc.want || r.Header.Get("X-OpenStack-Nova-API-Version") != tc.want {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, `{"quota_set":{"cores":1}}`)
			})
			client := cloud.Client("compute", "/compute/v2.1/")
			scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := scope.GetRecord(context.Background())
			if err != nil || result.Microversion != tc.want || len(result.Discovery.Responses) != 1 || !result.Discovery.Found || calls != 2 || client.Microversion != "" {
				t.Fatal(result, err, calls, client)
			}
		})
	}
}
func TestNovaQuotaRecordExplicitVersionSkipsDiscoveryAndAllowsNoVersion(t *testing.T) {
	for _, selected := range []string{"2.100", ""} {
		t.Run(selected, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/compute/v2.1/os-quota-sets/project" || r.Header.Get("X-OpenStack-Nova-API-Version") != selected {
					t.Error(r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"quota_set":{"cores":2}}`)
			})
			scope, err := quotasets.New(cloud.Client("compute", "/compute/v2.1/")).InProject(context.Background(), resource.ID("project"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := scope.GetRecord(context.Background(), quotasets.WithQuotaReadMicroversion(selected))
			if err != nil || result.Microversion != selected || calls != 1 || len(result.Discovery.Responses) != 0 {
				t.Fatal(result, err, calls)
			}
		})
	}
}
func TestNovaQuotaRecordBasePathCollisionAndRequiresIDFailBeforeDiscovery(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/compute/", func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 200, `{}`) })
	scope, err := quotasets.New(cloud.Client("compute", "/compute/v2.1/")).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range []quotasets.QuotaReadOption{quotasets.WithQuotaReadQuery("base_path", nil), quotasets.WithQuotaReadQuery("base_path", "/other"), quotasets.WithQuotaReadRequiresID(true), nil} {
		_, err := scope.GetRecord(context.Background(), option)
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls != 0 {
		t.Fatal(calls)
	}
}
func TestNovaQuotaRecordOptionInputSnapshotAndNoCache(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("user_id") != "project-query" || r.URL.Query().Get("tag") != "original" || r.URL.Query().Has("skip_cache") {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"quota_set":{"cores":1}}`)
	})
	client := cloud.Client("compute", "/compute/v2.1/")
	client.Microversion = "2.56"
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(`"original"`)
	query := map[string]json.RawMessage{"tag": raw, "user_id": json.RawMessage(`"project-query"`)}
	option := quotasets.WithQuotaReadOpts(quotasets.QuotaReadOpts{Query: query})
	raw[1] = 'X'
	query["late"] = json.RawMessage("true")
	for range 2 {
		if _, err := scope.GetRecord(context.Background(), option, quotasets.WithQuotaReadSkipCache(false)); err != nil {
			t.Fatal(err)
		}
	}
	user, err := scope.InUser(context.Background(), resource.ID("fixed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := user.GetRecord(context.Background(), quotasets.WithQuotaReadQuery("user_id", nil)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}
func TestNovaQuotaRecord404AndZeroScopesPreserveOperation(t *testing.T) {
	scope, _ := pythonRejudgeQuotaScope(t, `{"error":"unused"}`)
	// A nil receiver fails through the same public operation instead of panicking.
	var empty *quotasets.ProjectQuotaScope
	_, err := empty.GetRecord(context.Background())
	var operation *resource.OperationError
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "GetRecord" {
		t.Fatal(err)
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "missing-actual")
		testcloud.JSON(w, 404, `{"error":"missing"}`)
	})
	client := cloud.Client("compute", "/compute/v2.1/")
	client.Microversion = "2.56"
	scope, err = quotasets.New(client).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := scope.GetRecord(context.Background())
	if result != nil || !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &operation) || operation.Operation != "GetRecord" {
		t.Fatal(result, err)
	}
}

type novaQuotaBasePathMarshaler struct{ calls *int }

func (v novaQuotaBasePathMarshaler) MarshalJSON() ([]byte, error) {
	*v.calls++
	return []byte(`"/other"`), nil
}
func TestNovaQuotaBasePathPresenceRejectsWithoutInvokingCallerMarshaler(t *testing.T) {
	cloud := testcloud.New(t)
	httpCalls, marshaled := 0, 0
	cloud.Mux.HandleFunc("/compute/", func(w http.ResponseWriter, r *http.Request) { httpCalls++; testcloud.JSON(w, 200, `{}`) })
	scope, err := quotasets.New(cloud.Client("compute", "/compute/v2.1/")).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = scope.GetRecord(context.Background(), quotasets.WithQuotaReadQuery("base_path", novaQuotaBasePathMarshaler{&marshaled}))
	if !errors.Is(err, resource.ErrInvalidOption) || httpCalls != 0 || marshaled != 0 {
		t.Fatal(err, httpCalls, marshaled)
	}
}
