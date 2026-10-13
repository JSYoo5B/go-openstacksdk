package quotasets_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestCinderQuotaRecordUsageDefaultAndNestedNormalization(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		want := "False"
		if calls == 2 {
			want = "True"
		}
		if r.Method != "GET" || r.URL.Path != "/cinder/v3/os-quota-sets/project" || r.URL.Query().Get("usage") != want || r.Header.Get("OpenStack-API-Version") != "" || (r.Body != nil && r.Body != http.NoBody) {
			t.Error(r.Method, r.URL, r.Header, r.Body)
		}
		testcloud.JSON(w, 200, `{"quota_set":{"id":"response-project","volumes":{"limit":"12","in_use":2,"reserved":1,"allocated":9},"gigabytes":0,"volumes_lvm":{"limit":9007199254740993,"reserved":null}}}`)
	})
	client := cloud.Client("volumev3", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/"
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	for _, usage := range []bool{false, true} {
		result, err := scope.GetRecord(context.Background(), quotasets.WithQuotaUsage(usage))
		if err != nil || result.ProjectID != "project" || string(result.Resource.Body["id"]) != `"response-project"` || string(result.Resource.Body["volumes"]) != "12" || string(result.Resource.Body["usage"]) != `{"volumes":2}` || string(result.Resource.Body["reservation"]) != `{"volumes":1,"volumes_lvm":null}` || string(result.Resource.Body["volumes_lvm"]) != "9007199254740993" || len(result.Discovery.Responses) != 0 {
			t.Fatal(result, err)
		}
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}
func TestCinderQuotaRecordFetchParametersAndBasePathControls(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		q := r.URL.Query()
		if r.URL.Path != "/cinder/v3/os-quota-sets/project/defaults" || q.Get("usage") != "False" || strings.Join(q["fields"], ",") != "volumes,gigabytes" || q.Get("custom") != "False" || q.Has("none") || q.Has("skip_cache") || q.Has("resource_response_key") || q.Has("base_path") || q.Has("microversion") || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.100" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 203, `{"quota_set":{"volumes":7}}`)
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := scope.GetRecord(context.Background(), quotasets.WithQuotaReadQuery("fields", []string{"volumes", "gigabytes"}), quotasets.WithQuotaReadQuery("custom", false), quotasets.WithQuotaReadQuery("none", nil), quotasets.WithQuotaReadQuery("base_path", "/os-quota-sets/%(project_id)s/defaults"), quotasets.WithQuotaReadQuery("microversion", "3.100"), quotasets.WithQuotaReadQuery("skip_cache", true), quotasets.WithQuotaReadQuery("resource_response_key", []int{1}))
	if err != nil || result.StatusCode != 203 || result.Microversion != "3.100" || string(result.Resource.Body["volumes"]) != "7" || calls != 1 || client.Microversion != "" {
		t.Fatal(result, err, calls, client)
	}
}
func TestCinderQuotaRecordBoundParamsAndOptionOwnership(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("tag") != "original" || r.URL.Query().Has("late") {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"quota_set":{}}`)
	})
	scope, err := quotasets.New(cloud.Client("volumev3", "/cinder/v3/")).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(`"original"`)
	input := quotasets.QuotaReadOpts{Query: map[string]json.RawMessage{"tag": raw}}
	option := quotasets.WithQuotaReadOpts(input)
	raw[1] = 'X'
	input.Query["late"] = json.RawMessage("true")
	for range 2 {
		if _, err := scope.GetRecord(context.Background(), option); err != nil {
			t.Fatal(err)
		}
	}
	for _, bad := range []quotasets.QuotaReadOption{nil, quotasets.WithQuotaReadRequiresID(true), quotasets.WithQuotaReadQuery("session", nil), quotasets.WithQuotaReadQuery("base_path", 1), quotasets.WithQuotaReadBasePath("https://foreign.invalid/quotas")} {
		if _, err := scope.GetRecord(context.Background(), bad); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}
func TestCinderQuotaRecordAcceptsEmptyAndRetainsDecodeProof(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		fail bool
	}{{"", false}, {"not-json", false}, {`{}`, false}, {`{"quota_set":null}`, true}, {`{"quota_set":{"volumes":"²"}}`, true}} {
		t.Run(tc.raw, func(t *testing.T) {
			code := 204
			if tc.raw != "" {
				code = 203
			}
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-Id", "actual-quota")
				testcloud.JSON(w, code, tc.raw)
			})
			scope, err := quotasets.New(cloud.Client("volumev3", "/cinder/v3/")).InProject(context.Background(), resource.ID("project"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := scope.GetRecord(context.Background())
			if result == nil || result.StatusCode != code || string(result.Envelope) != tc.raw {
				t.Fatal(result, err)
			}
			if tc.fail {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != tc.raw || proof.Header.Get("X-Request-Id") != "actual-quota" || result.Resource != nil {
					t.Fatal(result, err, proof)
				}
			} else if err != nil || result.Resource == nil {
				t.Fatal(result, err)
			}
		})
	}
}
func TestCinderQuotaRecord404AndCancellationKeepOriginalFailure(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	raw := `{"error":"missing"}`
	cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Request-Id", "missing-actual")
		testcloud.JSON(w, 404, raw)
	})
	scope, err := quotasets.New(cloud.Client("volumev3", "/cinder/v3/")).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := scope.GetRecord(context.Background(), quotasets.WithQuotaReadErrorMessage("quota lookup"))
	var native gophercloud.ErrUnexpectedResponseCode
	var operation *resource.OperationError
	if result != nil || !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "missing-actual" || !errors.As(err, &operation) || operation.Operation != "GetRecord" {
		t.Fatal(result, err, native)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop quota")
	cancel(cause)
	_, err = scope.GetRecord(ctx)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 1 {
		t.Fatal(err, calls)
	}
}
func TestCinderQuotaRecordLocationAndLiteralProjectURI(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/cinder/v3/os-quota-sets/project-literal-1" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"quota_set":{"project_id":"spoof","volumes":null}}`)
	})
	scope, err := quotasets.New(cloud.Client("volumev3", "/cinder/v3/")).InProject(context.Background(), resource.ID("project-literal-1"))
	if err != nil {
		t.Fatal(err)
	}
	name := "configured"
	result, err := scope.GetRecord(context.Background(), quotasets.WithQuotaReadLocation(resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"current"`)}}))
	if err != nil || string(result.Resource.Body["project_id"]) != `"project-literal-1"` || string(result.Wire.Body["project_id"]) != `"spoof"` || !strings.Contains(string(result.Resource.Body["location"]), `"id":"project-literal-1"`) {
		t.Fatal(result, err)
	}
}
