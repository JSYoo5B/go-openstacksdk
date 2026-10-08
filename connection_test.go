package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func connection(t *testing.T, cloud *testcloud.Cloud) *sdk.Connection {
	t.Helper()
	c, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"),
		sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"),
		sdk.WithEndpoint(sdk.Image, cloud.Server.URL+"/image/v2"),
		sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/volume/v3/project"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLazyConcurrentServiceCache(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	var lookups atomic.Int32
	provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if opts.Region != "region" || opts.Availability != gophercloud.AvailabilityInternal {
			t.Errorf("endpoint options: %+v", opts)
		}
		return "https://compute.example/v2.1/project/", nil
	}
	c, err := sdk.FromProvider(provider, sdk.WithRegion("region"), sdk.WithInterface(gophercloud.AvailabilityInternal), sdk.WithMicroversion(sdk.Compute, "2.52"))
	if err != nil {
		t.Fatal(err)
	}
	if lookups.Load() != 0 {
		t.Fatal("eager endpoint lookup")
	}
	results := make([]*compute.Service, 20)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			results[i], err = c.Compute(context.Background())
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	for _, result := range results {
		if result != results[0] {
			t.Fatal("different cached service instances")
		}
	}
	if lookups.Load() != 1 {
		t.Fatalf("lookups=%d", lookups.Load())
	}
	if results[0].RawClient().ProviderClient != provider || results[0].RawClient().Microversion != "2.52" {
		t.Fatal("provider or microversion lost")
	}
}

func TestInvalidConnectionOptions(t *testing.T) {
	for _, option := range []sdk.ConnectionOption{
		sdk.WithEndpoint(sdk.Compute, "/relative"), sdk.WithEndpoint(sdk.Service("unknown"), "https://example.com"),
		sdk.WithInterface("invalid"), sdk.WithMicroversion(sdk.Compute, "3.1"),
		sdk.WithHTTPClient(http.Client{}), nil,
	} {
		if _, err := sdk.FromProvider(&gophercloud.ProviderClient{}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("expected invalid option, got %v", err)
		}
	}
	if _, err := sdk.FromProvider(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, _ := sdk.FromProvider(&gophercloud.ProviderClient{}, sdk.WithEndpoint(sdk.Compute, "https://example.com"))
	if _, err := c.Compute(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestConnectAuthenticatesWithExplicitOptions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Subject-Token", "authenticated-token")
		testcloud.JSON(w, http.StatusCreated, `{"token":{"expires_at":"2099-01-01T00:00:00Z","catalog":[]}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/id", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "authenticated-token" {
			t.Error("authentication token missing")
		}
		testcloud.JSON(w, 200, `{"server":{"id":"id","name":"example"}}`)
	})
	c, err := sdk.Connect(context.Background(), sdk.WithAuth(gophercloud.AuthOptions{IdentityEndpoint: cloud.Server.URL + "/v3", Username: "test", Password: "test", DomainName: "Default"}), sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Servers.Get(context.Background(), "id"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("authentication calls=%d", calls.Load())
	}
}

func TestConnectCloudAndEnvironmentSources(t *testing.T) {
	for _, source := range []string{"cloud", "environment"} {
		t.Run(source, func(t *testing.T) {
			cloud := testcloud.New(t)
			for _, key := range []string{"OS_CLOUD", "OS_CLIENT_CONFIG_FILE", "OS_REGION_NAME", "OS_INTERFACE", "OS_USERID", "OS_TENANT_ID", "OS_TENANT_NAME", "OS_PROJECT_ID", "OS_PROJECT_NAME", "OS_DOMAIN_ID", "OS_APPLICATION_CREDENTIAL_ID", "OS_APPLICATION_CREDENTIAL_NAME", "OS_APPLICATION_CREDENTIAL_SECRET", "OS_SYSTEM_SCOPE", "OS_PASSCODE"} {
				t.Setenv(key, "")
			}
			t.Setenv("OS_AUTH_URL", cloud.Server.URL+"/v3")
			t.Setenv("OS_USERNAME", "environment-user")
			t.Setenv("OS_PASSWORD", "environment-password")
			t.Setenv("OS_DOMAIN_NAME", "Default")
			wantUser := "environment-user"
			var opts []sdk.ConnectionOption
			if source == "cloud" {
				wantUser = "cloud-user"
				t.Setenv("OS_CLOUD", "wrong-cloud")
				path := filepath.Join(t.TempDir(), "clouds.yaml")
				body := fmt.Sprintf("clouds:\n  dev:\n    region_name: region-cloud\n    auth:\n      auth_url: %s/v3\n      username: cloud-user\n      password: cloud-password\n      user_domain_name: Default\n", cloud.Server.URL)
				if err := os.WriteFile(path, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
				opts = append(opts, sdk.WithCloud("dev"), sdk.WithCloudFiles(path), sdk.WithRegion("region-override"))
			} else {
				t.Setenv("OS_REGION_NAME", "region-override")
			}
			cloud.Mux.HandleFunc("POST /v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Auth struct {
						Identity struct {
							Password struct {
								User struct {
									Name string `json:"name"`
								} `json:"user"`
							} `json:"password"`
						} `json:"identity"`
					} `json:"auth"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Auth.Identity.Password.User.Name != wantUser {
					t.Errorf("auth user=%q want=%q", body.Auth.Identity.Password.User.Name, wantUser)
				}
				w.Header().Set("X-Subject-Token", "authenticated-token")
				testcloud.JSON(w, 201, fmt.Sprintf(`{"token":{"expires_at":"2099-01-01T00:00:00Z","catalog":[{"type":"compute","name":"nova","endpoints":[{"id":"cloud-region","interface":"public","region":"region-cloud","url":%q},{"id":"override-region","interface":"public","region":"region-override","url":%q}]}]}}`, cloud.Server.URL+"/other/v2.1/project/", cloud.Server.URL+"/compute/v2.1/project/"))
			})
			conn, err := sdk.Connect(context.Background(), opts...)
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if service.RawClient().Endpoint != cloud.Server.URL+"/compute/v2.1/project/" {
				t.Fatalf("endpoint=%q", service.RawClient().Endpoint)
			}
		})
	}
}
