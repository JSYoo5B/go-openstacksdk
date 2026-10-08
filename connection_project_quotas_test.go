package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	infraquotas "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	tokens "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

type quotaConnectionCase struct {
	name      string
	service   sdk.Service
	endpoint  string
	quotaPath string
	body      string
	bind      func(*sdk.Connection, context.Context, resource.Ref) (string, func(context.Context) error, error)
	current   func(*sdk.Connection, context.Context) (string, error)
}

func quotaConnectionCases() []quotaConnectionCase {
	return []quotaConnectionCase{
		{
			name: "Magnum", service: sdk.ContainerInfra, endpoint: "/magnum/v1", quotaPath: "/magnum/v1/quotas/resolved/Cluster", body: `{"id":1,"project_id":"wire-project","resource":"Cluster","hard_limit":8}`,
			bind: func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (string, func(context.Context) error, error) {
				parent, err := c.ContainerInfraProjectQuotas(ctx, ref)
				if err != nil {
					return "", nil, err
				}
				scope, err := parent.ForResource(infraquotas.Cluster)
				if err != nil {
					return "", nil, err
				}
				return parent.ProjectID(), func(ctx context.Context) error {
					value, err := scope.Get(ctx)
					if err == nil && (value.RequestProjectID != "resolved" || value.RequestResource != infraquotas.Cluster || value.ProjectID != "wire-project" || value.HardLimit != 8) {
						return errors.New("fixed quota pair changed")
					}
					return err
				}, nil
			},
			current: func(c *sdk.Connection, ctx context.Context) (string, error) {
				scope, err := c.CurrentContainerInfraProjectQuotas(ctx)
				if err != nil {
					return "", err
				}
				return scope.ProjectID(), nil
			},
		},
		{
			name: "Manila", service: sdk.SharedFileSystem, endpoint: "/manila/v2/admin", quotaPath: "/manila/v2/admin/quota-sets/resolved", body: `{"quota_set":{"id":"wire-project","shares":8}}`,
			bind: func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (string, func(context.Context) error, error) {
				scope, err := c.SharedFileSystemProjectQuotas(ctx, ref)
				if err != nil {
					return "", nil, err
				}
				return scope.ProjectID(), func(ctx context.Context) error {
					value, err := scope.Get(ctx)
					if err == nil && (value.ProjectID != "resolved" || value.ID != "wire-project" || value.Shares == nil || *value.Shares != 8) {
						return errors.New("fixed target changed")
					}
					return err
				}, nil
			},
			current: func(c *sdk.Connection, ctx context.Context) (string, error) {
				scope, err := c.CurrentSharedFileSystemProjectQuotas(ctx)
				if err != nil {
					return "", err
				}
				return scope.ProjectID(), nil
			},
		},
		{
			name: "Designate", service: sdk.DNS, endpoint: "/designate/v2", quotaPath: "/designate/v2/quotas/resolved", body: `{"zones":8,"wire_extension":"preserved"}`,
			bind: func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (string, func(context.Context) error, error) {
				scope, err := c.DNSProjectQuotas(ctx, ref)
				if err != nil {
					return "", nil, err
				}
				return scope.ProjectID(), func(ctx context.Context) error {
					value, err := scope.Get(ctx)
					if err == nil && (value.ProjectID != "resolved" || value.Zones != 8 || string(value.Body["wire_extension"]) != `"preserved"`) {
						return errors.New("fixed target changed")
					}
					return err
				}, nil
			},
			current: func(c *sdk.Connection, ctx context.Context) (string, error) {
				scope, err := c.CurrentDNSProjectQuotas(ctx)
				if err != nil {
					return "", err
				}
				return scope.ProjectID(), nil
			},
		},
		{
			name: "Octavia", service: sdk.LoadBalancer, endpoint: "/octavia", quotaPath: "/octavia/v2.0/lbaas/quotas/resolved", body: `{"quota":{"project_id":"wire-project","loadbalancer":8}}`,
			bind: func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (string, func(context.Context) error, error) {
				scope, err := c.LoadBalancerProjectQuotas(ctx, ref)
				if err != nil {
					return "", nil, err
				}
				return scope.ProjectID(), func(ctx context.Context) error {
					value, err := scope.Get(ctx)
					if err == nil && (value.ProjectID != "resolved" || string(value.Body["project_id"]) != `"wire-project"` || value.Loadbalancer != 8) {
						return errors.New("fixed target changed")
					}
					return err
				}, nil
			},
			current: func(c *sdk.Connection, ctx context.Context) (string, error) {
				scope, err := c.CurrentLoadBalancerProjectQuotas(ctx)
				if err != nil {
					return "", err
				}
				return scope.ProjectID(), nil
			},
		},
		{
			name: "Cinder", service: sdk.BlockStorage, endpoint: "/cinder/v3/admin", quotaPath: "/cinder/v3/admin/os-quota-sets/resolved", body: `{"quota_set":{"id":"wire-project","volumes":8}}`,
			bind: func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (string, func(context.Context) error, error) {
				scope, err := c.BlockStorageProjectQuotas(ctx, ref)
				if err != nil {
					return "", nil, err
				}
				return scope.ProjectID(), func(ctx context.Context) error {
					value, err := scope.Get(ctx)
					if err == nil && (value.ProjectID != "resolved" || value.ID != "wire-project") {
						return errors.New("fixed target changed")
					}
					return err
				}, nil
			},
			current: func(c *sdk.Connection, ctx context.Context) (string, error) {
				scope, err := c.CurrentBlockStorageProjectQuotas(ctx)
				if err != nil {
					return "", err
				}
				return scope.ProjectID(), nil
			},
		},
		{
			name: "Neutron", service: sdk.Network, endpoint: "/neutron/v2.0", quotaPath: "/neutron/v2.0/quotas/resolved", body: `{"quota":{"network":8,"id":"wire-project"}}`,
			bind: func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (string, func(context.Context) error, error) {
				scope, err := c.NetworkProjectQuotas(ctx, ref)
				if err != nil {
					return "", nil, err
				}
				return scope.ProjectID(), func(ctx context.Context) error {
					value, err := scope.Get(ctx)
					if err == nil && (value.ProjectID != "resolved" || string(value.Body["id"]) != `"wire-project"`) {
						return errors.New("fixed target changed")
					}
					return err
				}, nil
			},
			current: func(c *sdk.Connection, ctx context.Context) (string, error) {
				scope, err := c.CurrentNetworkProjectQuotas(ctx)
				if err != nil {
					return "", err
				}
				return scope.ProjectID(), nil
			},
		},
	}
}

func TestConnectionServiceQuotaNamesUseSeparateIdentityClient(t *testing.T) {
	for _, test := range quotaConnectionCases() {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lookups, gets atomic.Int32
			cloud.Mux.HandleFunc("GET /keystone/v3/projects", func(w http.ResponseWriter, r *http.Request) {
				lookups.Add(1)
				if r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-OpenStack-Volume-API-Version") != "" || r.Header.Get("OpenStack-API-Version") != "" || r.Header.Get("X-Auth-Sudo-Project-ID") != "" {
					t.Errorf("identity headers/query: %s %v", r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"projects":[{"id":"resolved","name":"tenant"}]}`)
			})
			cloud.Mux.HandleFunc("GET "+test.quotaPath, func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if test.service == sdk.BlockStorage && r.Header.Get("X-OpenStack-Volume-API-Version") != "3.70" {
					t.Error(r.Header)
				}
				if test.service == sdk.SharedFileSystem && r.Header.Get("X-OpenStack-Manila-API-Version") != "2.39" {
					t.Error(r.Header)
				}
				if test.service == sdk.DNS && r.Header.Get("X-Auth-Sudo-Project-ID") != "resolved" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, test.body)
			})
			options := []sdk.ConnectionOption{sdk.WithEndpoint(test.service, cloud.Server.URL+test.endpoint), sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/keystone/v3")}
			if test.service == sdk.BlockStorage {
				options = append(options, sdk.WithMicroversion(sdk.BlockStorage, "3.70"))
			}
			if test.service == sdk.SharedFileSystem {
				options = append(options, sdk.WithMicroversion(sdk.SharedFileSystem, "2.39"))
			}
			c, err := sdk.FromProvider(cloud.Provider, options...)
			if err != nil {
				t.Fatal(err)
			}
			id, get, err := test.bind(c, context.Background(), resource.Name("tenant"))
			if err != nil || id != "resolved" {
				t.Fatalf("id=%s error=%v", id, err)
			}
			for range 2 {
				if err := get(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if lookups.Load() != 1 || gets.Load() != 2 {
				t.Fatalf("lookup/get counts=%d/%d", lookups.Load(), gets.Load())
			}
		})
	}
}

func TestConnectionServiceQuotasIDAuthAndCancellation(t *testing.T) {
	for _, test := range quotaConnectionCases() {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
				t.Errorf("unexpected catalog lookup: %+v", options)
				return "", errors.New("unexpected catalog lookup")
			}
			c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(test.service, cloud.Server.URL+test.endpoint))
			if err != nil {
				t.Fatal(err)
			}
			if id, _, err := test.bind(c, context.Background(), resource.ID("explicit")); err != nil || id != "explicit" {
				t.Fatalf("explicit ID=%s error=%v", id, err)
			}
			if _, err := test.current(c, context.Background()); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatalf("manual token=%v", err)
			}
			auth := tokens.CreateResult{}
			auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "current"}}}
			auth.Header = http.Header{"X-Subject-Token": []string{"auth-token"}}
			if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
				t.Fatal(err)
			}
			if id, err := test.current(c, context.Background()); err != nil || id != "current" {
				t.Fatalf("auth scope=%s error=%v", id, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, _, err := test.bind(c, ctx, resource.Name("tenant")); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if _, err := test.current(c, ctx); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if _, _, err := test.bind(c, context.Background(), resource.ID("../escape")); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}
