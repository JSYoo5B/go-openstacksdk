package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	tokens "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

type limitsConnectionCase struct {
	name, endpoint, path, selector, version string
	service                                 sdk.Service
	bind                                    func(*sdk.Connection, context.Context, resource.Ref) (string, func(context.Context) (string, error), error)
	current                                 func(*sdk.Connection, context.Context) (string, error)
}

func limitsConnectionCases() []limitsConnectionCase {
	return []limitsConnectionCase{
		{
			name: "Nova", service: sdk.Compute, endpoint: "/nova/v2.1", path: "/nova/v2.1/limits", selector: "tenant_id", version: "2.90",
			bind: func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (string, func(context.Context) (string, error), error) {
				scope, err := c.ProjectLimits(ctx, ref)
				if err != nil {
					return "", nil, err
				}
				return scope.ProjectID(), func(ctx context.Context) (string, error) {
					value, err := scope.Get(ctx)
					if err != nil {
						return "", err
					}
					return value.ProjectID, nil
				}, nil
			},
			current: func(c *sdk.Connection, ctx context.Context) (string, error) {
				scope, err := c.CurrentProjectLimits(ctx)
				if err != nil {
					return "", err
				}
				return scope.ProjectID(), nil
			},
		},
		{
			name: "Cinder", service: sdk.BlockStorage, endpoint: "/cinder/v3/admin", path: "/cinder/v3/admin/limits", selector: "project_id", version: "3.70",
			bind: func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (string, func(context.Context) (string, error), error) {
				scope, err := c.BlockStorageProjectLimits(ctx, ref)
				if err != nil {
					return "", nil, err
				}
				return scope.ProjectID(), func(ctx context.Context) (string, error) {
					value, err := scope.Get(ctx)
					if err != nil {
						return "", err
					}
					return value.ProjectID, nil
				}, nil
			},
			current: func(c *sdk.Connection, ctx context.Context) (string, error) {
				scope, err := c.CurrentBlockStorageProjectLimits(ctx)
				if err != nil {
					return "", err
				}
				return scope.ProjectID(), nil
			},
		},
	}
}

func TestConnectionProjectLimitsSeparateIdentityAndFixQueries(t *testing.T) {
	for _, test := range limitsConnectionCases() {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lookups, reads atomic.Int32
			cloud.Mux.HandleFunc("GET /keystone/v3/projects", func(w http.ResponseWriter, r *http.Request) {
				lookups.Add(1)
				if r.URL.Query().Get("name") != "tenant" || r.Header.Get("OpenStack-API-Version") != "" {
					t.Errorf("identity request=%s %v", r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"projects":[{"id":"resolved","name":"tenant"}]}`)
			})
			cloud.Mux.HandleFunc("GET "+test.path, func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if len(r.URL.Query()) != 1 || r.URL.Query().Get(test.selector) != "resolved" || len(r.URL.Query()[test.selector]) != 1 {
					t.Error(r.URL)
				}
				versionHeader := "X-OpenStack-Nova-API-Version"
				if test.service == sdk.BlockStorage {
					versionHeader = "X-OpenStack-Volume-API-Version"
				}
				if r.Header.Get(versionHeader) != test.version || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, `{"limits":{"absolute":{},"rate":[],"project_id":"wire-other"}}`)
			})
			c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(test.service, cloud.Server.URL+test.endpoint), sdk.WithMicroversion(test.service, test.version), sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/keystone/v3"))
			if err != nil {
				t.Fatal(err)
			}
			id, get, err := test.bind(c, context.Background(), resource.Name("tenant"))
			if err != nil || id != "resolved" || reads.Load() != 0 {
				t.Fatalf("id=%s err=%v reads=%d", id, err, reads.Load())
			}
			for range 2 {
				if id, err := get(context.Background()); err != nil || id != "resolved" {
					t.Fatalf("response target=%s err=%v", id, err)
				}
			}
			if lookups.Load() != 1 || reads.Load() != 2 {
				t.Fatalf("lookup/read=%d/%d", lookups.Load(), reads.Load())
			}
		})
	}
}

func TestConnectionProjectLimitsExplicitIDsRecordedAuthAndCancellation(t *testing.T) {
	for _, test := range limitsConnectionCases() {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				t.Errorf("unexpected catalog lookup=%+v", opts)
				return "", errors.New("unexpected catalog lookup")
			}
			c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(test.service, cloud.Server.URL+test.endpoint), sdk.WithMicroversion(test.service, test.version))
			if err != nil {
				t.Fatal(err)
			}
			if id, _, err := test.bind(c, context.Background(), resource.ID("explicit")); err != nil || id != "explicit" {
				t.Fatalf("ID=%s err=%v", id, err)
			}
			if _, err := test.current(c, context.Background()); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(err)
			}
			auth := tokens.CreateResult{}
			auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "authenticated"}}}
			auth.Header = http.Header{"X-Subject-Token": []string{"auth-token"}}
			if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
				t.Fatal(err)
			}
			if id, err := test.current(c, context.Background()); err != nil || id != "authenticated" {
				t.Fatalf("auth ID=%s err=%v", id, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, _, err := test.bind(c, ctx, resource.Name("tenant")); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if _, err := test.current(c, ctx); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}
