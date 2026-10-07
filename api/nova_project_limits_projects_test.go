package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/limits"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

func TestNovaLimitsProjectIDsAreLazyAndExactNamesUseSeparateKeystoneOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var identity, compute atomic.Int32
	cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		identity.Add(1)
		if r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-OpenStack-Nova-API-Version") != "" {
			t.Error(r.URL, r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"projects": []map[string]string{{"id": "wrong", "name": "tenant-extra"}}, "links": map[string]string{"next": cloud.Server.URL + "/identity/v3/projects/page2"}})
	})
	cloud.Mux.HandleFunc("/identity/v3/projects/page2", func(w http.ResponseWriter, r *http.Request) {
		identity.Add(1)
		testcloud.JSON(w, 200, `{"projects":[{"id":"project-fixed","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("/nova/limits", func(w http.ResponseWriter, r *http.Request) {
		compute.Add(1)
		if !reflect.DeepEqual(r.URL.Query()["tenant_id"], []string{"project-fixed"}) {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, novaLimitsBody)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("wrong resolver service or limits URL=%s", r.URL)
	})
	client := cloud.Client("compute", "/nova")
	client.Microversion = "2.57"
	api := limits.New(client)
	explicit, err := api.InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil || explicit.ProjectID() != "project-fixed" || identity.Load() != 0 || compute.Load() != 0 {
		t.Fatal(explicit, err)
	}
	resolved, err := api.InProject(context.Background(), resource.Name("tenant"), limits.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil || resolved.ProjectID() != "project-fixed" || identity.Load() != 2 || compute.Load() != 0 {
		t.Fatal(resolved, err, identity.Load(), compute.Load())
	}
	for _, scope := range []*limits.ProjectLimitsScope{explicit, resolved, resolved} {
		if value, err := scope.Get(context.Background()); err != nil || value.ProjectID != "project-fixed" {
			t.Fatal(value, err)
		}
	}
	if identity.Load() != 2 || compute.Load() != 3 {
		t.Fatal("Get repeated project-name resolution")
	}
}

func TestNovaLimitsProjectLookupFailuresNeverCallCompute(t *testing.T) {
	for _, tc := range []struct {
		body string
		code int
		want error
	}{
		{`{"projects":[]}`, 200, resource.ErrNotFound},
		{`{"projects":[{"id":"one","name":"tenant","domain_id":"one"},{"id":"two","name":"tenant","domain_id":"two"}]}`, 200, resource.ErrAmbiguous},
		{`{"projects":[{"id":"bad/path","name":"tenant"}]}`, 200, resource.ErrInvalidOption},
		{`{"error":"project denied"}`, 403, nil},
	} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.Header().Set("X-Openstack-Request-Id", "keystone-denied")
			testcloud.JSON(w, tc.code, tc.body)
		})
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("lookup failure made Compute request=%s", r.URL)
		})
		scope, err := limits.New(cloud.Client("compute", "/nova")).InProject(context.Background(), resource.Name("tenant"), limits.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
		if scope != nil || err == nil || tc.want != nil && !errors.Is(err, tc.want) || calls.Load() != 1 {
			t.Fatal(scope, err)
		}
		if tc.code == 403 {
			var response gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &response) || response.Actual != 403 || string(response.Body) != tc.body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "keystone-denied" {
				t.Fatal(err)
			}
		}
	}
}

func TestNovaLimitsNilCanceledAndMalformedBindingsNeverCallHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid binding reached HTTP=%s", r.URL) })
	client := cloud.Client("compute", "/nova")
	api := limits.New(client)
	for _, ref := range []resource.Ref{resource.ID(""), resource.ID("bad/path"), resource.ID("bad?other"), resource.ID("bad%2Fother")} {
		if scope, err := api.InProject(context.Background(), ref); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(scope, err)
		}
	}
	for _, option := range []limits.ProjectOption{nil, limits.WithIdentityClient(nil), limits.WithIdentityClient(client)} {
		if scope, err := api.InProject(context.Background(), resource.ID("project"), option); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(scope, err)
		}
	}
	if scope, err := api.InProject(context.Background(), resource.Name("f43d6b43-4d4d-4c44-a444-4e444e444e44")); scope != nil || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal("UUID-shaped name was guessed as an ID", scope, err)
	}
	for _, invalid := range []*limits.API{nil, limits.New(nil), limits.New(&gophercloud.ServiceClient{})} {
		if scope, err := invalid.InProject(context.Background(), resource.ID("project")); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(scope, err)
		}
		if value, err := invalid.Fetch(context.Background()); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
		if value, err := invalid.CurrentProject(context.Background()); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	for _, scope := range []*limits.ProjectLimitsScope{nil, {}} {
		if value, err := scope.Get(context.Background()); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if scope, err := api.InProject(ctx, resource.Name("tenant"), limits.WithIdentityClient(cloud.Client("identity", "/identity/v3"))); scope != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(scope, err)
	}
	if value, err := api.Fetch(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(value, err)
	}
	if value, err := api.CurrentProject(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(value, err)
	}
}

func TestNovaLimitsCurrentProjectFreezesRecordedKeystoneV2AndV3Scope(t *testing.T) {
	for _, version := range []string{"v2", "v3"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/nova/limits", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["tenant_id"], []string{"project-fixed"}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, novaLimitsBody)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("current scope guessed path=%s", r.URL) })
			if version == "v3" {
				setQuotaProjectAuth(t, cloud.Provider, "project-fixed")
			} else {
				var auth tokens2.CreateResult
				auth.Body = map[string]any{"access": map[string]any{"token": map[string]any{"id": "v2-token", "expires": "2026-10-01T00:00:00.000Z", "tenant": map[string]any{"id": "project-fixed"}}}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := limits.New(cloud.Client("compute", "/nova")).CurrentProject(context.Background())
			if err != nil || scope == nil || scope.ProjectID() != "project-fixed" || calls.Load() != 0 {
				t.Fatal(scope, err)
			}
			setQuotaProjectAuth(t, cloud.Provider, "changed-project")
			if value, err := scope.Get(context.Background()); err != nil || value.ProjectID != "project-fixed" || calls.Load() != 1 {
				t.Fatal(value, err)
			}
		})
	}
}

func TestNovaLimitsCurrentProjectRejectsMissingUnscopedAndMalformedAuth(t *testing.T) {
	for _, tc := range []struct {
		name string
		body any
		want error
	}{
		{"manual", nil, resource.ErrUnsupported},
		{"system", map[string]any{"token": map[string]any{"system": map[string]any{"all": true}}}, resource.ErrUnsupported},
		{"domain", map[string]any{"token": map[string]any{"domain": map[string]any{"id": "domain"}}}, resource.ErrUnsupported},
		{"empty project", map[string]any{"token": map[string]any{"project": map[string]any{"id": ""}}}, resource.ErrUnsupported},
		{"bad project", map[string]any{"token": map[string]any{"project": map[string]any{"id": "bad/path"}}}, resource.ErrInvalidOption},
		{"malformed", map[string]any{"token": map[string]any{"project": "bad"}}, nil},
	} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid recorded auth reached HTTP=%s", r.URL) })
		if tc.body != nil {
			var auth tokens3.CreateResult
			auth.Body, auth.Header = tc.body, http.Header{"X-Subject-Token": []string{"recorded-token"}}
			if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
				t.Fatal(err)
			}
		}
		scope, err := limits.New(cloud.Client("compute", "/nova/guessed-project")).CurrentProject(context.Background())
		if scope != nil || err == nil || tc.want != nil && !errors.Is(err, tc.want) {
			t.Fatal(tc.name, scope, err)
		}
		if tc.name == "malformed" {
			var cause *json.UnmarshalTypeError
			if !errors.As(err, &cause) {
				t.Fatal(err)
			}
		}
	}
}
