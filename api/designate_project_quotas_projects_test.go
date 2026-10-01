package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"gophercloudsdk/dns/v2/quotas"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestDesignateQuotaProjectReferencesResolveOnceWithSeparateKeystone(t *testing.T) {
	cloud := testcloud.New(t)
	var identity, dns atomic.Int32
	cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		identity.Add(1)
		if r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-Auth-Sudo-Project-ID") != "" || r.Header.Get("OpenStack-API-Version") != "" {
			t.Error(r.URL, r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"projects": []map[string]string{{"id": "wrong", "name": "tenant-extra"}}, "links": map[string]string{"next": cloud.Server.URL + "/identity/v3/projects/page2"}})
	})
	cloud.Mux.HandleFunc("/identity/v3/projects/page2", func(w http.ResponseWriter, r *http.Request) {
		identity.Add(1)
		testcloud.JSON(w, 200, `{"projects":[{"id":"project-fixed","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("/dns/v2/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		dns.Add(1)
		if r.Header.Get("X-Auth-Sudo-Project-ID") != "project-fixed" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, designateQuotaBody)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("wrong service route=%s", r.URL)
		http.Error(w, "wrong route", 404)
	})
	client := cloud.Client("dns", "/dns/v2")
	client.Microversion = "2.0"
	api := quotas.New(client)
	explicit, err := api.InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil || explicit.ProjectID() != "project-fixed" || dns.Load() != 0 || identity.Load() != 0 {
		t.Fatal(explicit, err)
	}
	resolved, err := api.InProject(context.Background(), resource.Name("tenant"), quotas.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil || resolved.ProjectID() != "project-fixed" || dns.Load() != 0 || identity.Load() != 2 {
		t.Fatal(resolved, err, identity.Load())
	}
	for _, scope := range []*quotas.ProjectQuotaScope{explicit, resolved, resolved} {
		if value, err := scope.Get(context.Background()); err != nil || value.ProjectID != "project-fixed" {
			t.Fatal(value, err)
		}
	}
	if dns.Load() != 3 || identity.Load() != 2 {
		t.Fatal("quota operation repeated project resolution")
	}
}

func TestDesignateQuotaProjectResolutionFailuresDoNotCallDNS(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       error
	}{
		{"missing", `{"projects":[]}`, 200, resource.ErrNotFound},
		{"duplicate domains", `{"projects":[{"id":"one","name":"tenant","domain_id":"one"},{"id":"two","name":"tenant","domain_id":"two"}]}`, 200, resource.ErrAmbiguous},
		{"bad ID", `{"projects":[{"id":"bad/path","name":"tenant"}]}`, 200, resource.ErrInvalidOption},
		{"forbidden", `{"error":"project lookup denied"}`, 403, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lookups atomic.Int32
			cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
				lookups.Add(1)
				w.Header().Set("X-Openstack-Request-Id", "project-denied")
				testcloud.JSON(w, tc.status, tc.body)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("lookup failure called DNS: %s", r.URL) })
			scope, err := quotas.New(cloud.Client("dns", "/dns/v2")).InProject(context.Background(), resource.Name("tenant"), quotas.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
			if scope != nil || err == nil || tc.want != nil && !errors.Is(err, tc.want) || lookups.Load() != 1 {
				t.Fatal(scope, err, lookups.Load())
			}
			if tc.status == 403 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 || string(response.Body) != tc.body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "project-denied" {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestDesignateQuotaProjectScopeInputsNeverGuessNamesOrClients(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid binding called HTTP: %s", r.URL) })
	client := cloud.Client("dns", "/dns/v2")
	api := quotas.New(client)
	for _, ref := range []resource.Ref{resource.ID(""), resource.ID("bad/path"), resource.ID("project?other"), resource.ID("project%2Fother")} {
		if scope, err := api.InProject(context.Background(), ref); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(scope, err)
		}
	}
	for _, option := range []quotas.ProjectOption{nil, quotas.WithIdentityClient(nil), quotas.WithIdentityClient(client)} {
		if scope, err := api.InProject(context.Background(), resource.ID("project"), option); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(scope, err)
		}
	}
	if scope, err := api.InProject(context.Background(), resource.Name("f43ace6b-4a48-4daa-a259-13f977e23861")); scope != nil || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal("UUID-shaped name was treated as an ID", scope, err)
	}
	for _, invalid := range []*quotas.API{nil, quotas.New(nil), quotas.New(&gophercloud.ServiceClient{})} {
		if scope, err := invalid.InProject(context.Background(), resource.ID("project")); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(scope, err)
		}
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if scope, err := api.InProject(canceled, resource.Name("tenant"), quotas.WithIdentityClient(cloud.Client("identity", "/identity/v3"))); scope != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(scope, err)
	}
}

func TestDesignateQuotaCurrentProjectFreezesRecordedV2V3Identity(t *testing.T) {
	for _, version := range []string{"v2", "v3"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/dns/v2/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("X-Auth-Sudo-Project-ID") != "project-fixed" || r.Header.Get("X-Auth-All-Projects") != "false" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, designateQuotaBody)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("current scope guessed a route=%s", r.URL) })
			if version == "v3" {
				setQuotaProjectAuth(t, cloud.Provider, "project-fixed")
			} else {
				var auth tokens2.CreateResult
				auth.Body = map[string]any{"access": map[string]any{"token": map[string]any{"id": "v2-token", "expires": "2026-10-01T00:00:00.000Z", "tenant": map[string]any{"id": "project-fixed"}}}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := quotas.New(cloud.Client("dns", "/dns/v2")).CurrentProject(context.Background(), quotas.WithAllProjects(false))
			if err != nil || scope == nil || scope.ProjectID() != "project-fixed" || calls.Load() != 0 {
				t.Fatal(scope, err)
			}
			setQuotaProjectAuth(t, cloud.Provider, "changed-project")
			if value, err := scope.Get(context.Background()); err != nil || value.ProjectID != "project-fixed" || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
}

func TestDesignateQuotaCurrentProjectRejectsMissingUnscopedAndInvalidAuth(t *testing.T) {
	for _, tc := range []struct {
		name string
		body any
		want error
	}{
		{"manual", nil, resource.ErrUnsupported},
		{"system", map[string]any{"token": map[string]any{"system": map[string]any{"all": true}}}, resource.ErrUnsupported},
		{"domain", map[string]any{"token": map[string]any{"domain": map[string]any{"id": "domain"}}}, resource.ErrUnsupported},
		{"empty project", map[string]any{"token": map[string]any{"project": map[string]any{"id": ""}}}, resource.ErrUnsupported},
		{"invalid project", map[string]any{"token": map[string]any{"project": map[string]any{"id": "bad/path"}}}, resource.ErrInvalidOption},
		{"malformed project", map[string]any{"token": map[string]any{"project": "bad"}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid auth triggered HTTP=%s", r.URL) })
			if tc.body != nil {
				var auth tokens3.CreateResult
				auth.Body, auth.Header = tc.body, http.Header{"X-Subject-Token": []string{"auth-token"}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := quotas.New(cloud.Client("dns", "/dns/v2/guessed-project")).CurrentProject(context.Background())
			if scope != nil || err == nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatal(scope, err)
			}
			if tc.name == "malformed project" {
				var decode *json.UnmarshalTypeError
				if !errors.As(err, &decode) {
					t.Fatal(err)
				}
			}
		})
	}
}
