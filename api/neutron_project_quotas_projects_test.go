package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

func TestNeutronProjectQuotaResolvesNameOnceUsingSeparateIdentityClient(t *testing.T) {
	cloud := testcloud.New(t)
	var identityCalls, networkCalls atomic.Int32
	cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		identityCalls.Add(1)
		if r.Method != http.MethodGet || r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-Network-Only") != "" {
			t.Errorf("identity URL/header=%s/%v", r.URL, r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"projects": []map[string]string{{"id": "wrong", "name": "tenant-extra"}}, "links": map[string]string{"next": cloud.Server.URL + "/identity/v3/projects/page2"}})
	})
	cloud.Mux.HandleFunc("/identity/v3/projects/page2", func(w http.ResponseWriter, r *http.Request) {
		identityCalls.Add(1)
		testcloud.JSON(w, 200, `{"projects":[{"id":"project-fixed","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		networkCalls.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("X-Network-Only") != "configured" {
			t.Errorf("network URL/header=%s/%v", r.URL, r.Header)
		}
		testcloud.JSON(w, 200, neutronQuotaLimits)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("lookup/quota used the wrong service: %s", r.URL.Path)
		testcloud.JSON(w, 404, "{}")
	})
	client := cloud.Client("network", "/neutron/v2.0")
	client.MoreHeaders = map[string]string{"X-Network-Only": "configured"}
	scope, err := quotas.New(client).InProject(context.Background(), resource.Name("tenant"), quotas.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil || scope == nil || scope.ProjectID() != "project-fixed" || identityCalls.Load() != 2 || networkCalls.Load() != 0 {
		t.Fatalf("scope=%v calls=%d/%d error=%v", scope, identityCalls.Load(), networkCalls.Load(), err)
	}
	for range 2 {
		if value, err := scope.Get(context.Background()); err != nil || value.ProjectID != "project-fixed" {
			t.Fatalf("value=%+v error=%v", value, err)
		}
	}
	if identityCalls.Load() != 2 || networkCalls.Load() != 2 {
		t.Fatal("quota operation resolved the project name again")
	}
}

func TestNeutronProjectQuotaLookupFailuresPreventNetworkRequests(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       error
	}{
		{"missing", `{"projects":[]}`, 200, resource.ErrNotFound},
		{"ambiguous", `{"projects":[{"id":"one","name":"tenant"},{"id":"two","name":"tenant"}]}`, 200, resource.ErrAmbiguous},
		{"bad returned ID", `{"projects":[{"id":"bad/id","name":"tenant"}]}`, 200, resource.ErrInvalidOption},
		{"forbidden", `{"error":"identity forbidden"}`, 403, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var identityCalls atomic.Int32
			cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
				identityCalls.Add(1)
				w.Header().Set("X-Openstack-Request-Id", "identity-denied")
				testcloud.JSON(w, tc.status, tc.body)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("failed lookup requested Neutron: %s", r.URL.Path)
			})
			scope, err := quotas.New(cloud.Client("network", "/neutron/v2.0")).InProject(context.Background(), resource.Name("tenant"), quotas.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
			if scope != nil || err == nil || identityCalls.Load() != 1 || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("scope=%v calls=%d error=%v", scope, identityCalls.Load(), err)
			}
			if tc.status == 403 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 || response.ResponseHeader.Get("X-Openstack-Request-Id") != "identity-denied" || string(response.Body) != tc.body {
					t.Fatalf("response=%+v error=%v", response, err)
				}
			}
		})
	}
}

func setNeutronQuotaProjectAuth(t *testing.T, provider *gophercloud.ProviderClient, id string) {
	t.Helper()
	var auth tokens3.CreateResult
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": id}}}
	auth.Header = http.Header{"X-Subject-Token": []string{"current-project-token"}}
	if err := provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
}

func TestNeutronProjectQuotaCurrentProjectUsesRecordedV3AndV2Auth(t *testing.T) {
	for _, version := range []string{"v3", "v2"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/neutron/v2.0/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, neutronQuotaLimits)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("current scope guessed/refreshed a project: %s", r.URL.Path)
			})
			if version == "v3" {
				setNeutronQuotaProjectAuth(t, cloud.Provider, "project-fixed")
			} else {
				var auth tokens2.CreateResult
				auth.Body = map[string]any{"access": map[string]any{"token": map[string]any{"id": "v2-token", "expires": "2026-10-01T00:00:00.000Z", "tenant": map[string]any{"id": "project-fixed"}}}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := quotas.New(cloud.Client("network", "/neutron/v2.0")).CurrentProject(context.Background())
			if err != nil || scope == nil || scope.ProjectID() != "project-fixed" || calls.Load() != 0 {
				t.Fatalf("scope=%v calls=%d error=%v", scope, calls.Load(), err)
			}
			setNeutronQuotaProjectAuth(t, cloud.Provider, "changed-project")
			if value, err := scope.Get(context.Background()); err != nil || value == nil || value.ProjectID != "project-fixed" || calls.Load() != 1 {
				t.Fatalf("value=%+v calls=%d error=%v", value, calls.Load(), err)
			}
		})
	}
}

type neutronQuotaUnsupportedAuth struct{}

func (*neutronQuotaUnsupportedAuth) ExtractTokenID() (string, error) { return "custom-token", nil }

func TestNeutronProjectQuotaCurrentProjectRejectsMissingOrMalformedAuth(t *testing.T) {
	for _, tc := range []struct {
		name string
		auth gophercloud.AuthResult
		body any
		id   string
		want error
	}{
		{name: "manual", want: resource.ErrUnsupported},
		{name: "unsupported", auth: &neutronQuotaUnsupportedAuth{}, want: resource.ErrUnsupported},
		{name: "typed nil", auth: (*neutronQuotaUnsupportedAuth)(nil), want: resource.ErrUnsupported},
		{name: "system", body: map[string]any{"token": map[string]any{"system": map[string]any{"all": true}}}, want: resource.ErrUnsupported},
		{name: "domain", body: map[string]any{"token": map[string]any{"domain": map[string]any{"id": "domain"}}}, want: resource.ErrUnsupported},
		{name: "empty", body: map[string]any{"token": map[string]any{"project": map[string]any{"id": ""}}}, want: resource.ErrUnsupported},
		{name: "bad ID", id: "project/path", want: resource.ErrInvalidOption},
		{name: "malformed", body: map[string]any{"token": map[string]any{"project": "bad"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("invalid auth made HTTP request: %s", r.URL.Path)
			})
			if tc.auth != nil {
				if err := cloud.Provider.SetTokenAndAuthResult(tc.auth); err != nil {
					t.Fatal(err)
				}
			} else if tc.id != "" {
				setNeutronQuotaProjectAuth(t, cloud.Provider, tc.id)
			} else if tc.body != nil {
				var auth tokens3.CreateResult
				auth.Body, auth.Header = tc.body, http.Header{"X-Subject-Token": []string{"private-token"}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := quotas.New(cloud.Client("network", "/neutron/v2.0/guessed-project")).CurrentProject(context.Background())
			if scope != nil || err == nil || tc.want != nil && !errors.Is(err, tc.want) || strings.Contains(err.Error(), "private-token") || strings.Contains(err.Error(), "custom-token") {
				t.Fatalf("scope=%v error=%v", scope, err)
			}
			if tc.name == "malformed" {
				var cause *json.UnmarshalTypeError
				if !errors.As(err, &cause) {
					t.Fatalf("authentication decoder cause lost: %v", err)
				}
			}
		})
	}
}

func TestNeutronProjectQuotaScopeConstructionPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid scope made HTTP request: %s", r.URL.Path)
	})
	api := quotas.New(cloud.Client("network", "/neutron/v2.0"))
	for _, ref := range []resource.Ref{resource.ID(""), resource.ID("bad/path"), resource.ID("../project"), resource.Name("tenant")} {
		if scope, err := api.InProject(context.Background(), ref); scope != nil || err == nil {
			t.Fatalf("ref=%v scope=%v error=%v", ref, scope, err)
		}
	}
	for _, option := range []quotas.ProjectOption{nil, quotas.WithIdentityClient(nil), quotas.WithIdentityClient(api.RawClient()), quotas.WithIdentityClient(&gophercloud.ServiceClient{Type: "identity"})} {
		if scope, err := api.InProject(context.Background(), resource.Name("tenant"), option); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("scope=%v error=%v", scope, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if scope, err := api.InProject(ctx, resource.ID("project")); scope != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("scope=%v error=%v", scope, err)
	}
	if scope, err := api.CurrentProject(ctx); scope != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("current=%v error=%v", scope, err)
	}
	for _, invalid := range []*quotas.API{nil, quotas.New(nil), quotas.New(&gophercloud.ServiceClient{})} {
		if scope, err := invalid.InProject(context.Background(), resource.ID("project")); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid client scope=%v error=%v", scope, err)
		}
	}
}
