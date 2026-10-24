package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TestEnvironmentCloudSuppliesNetworkRoleConfiguration covers Python's
// environment-variable cloud for role settings: OS_* variables become cloud
// settings, so OS_INTERNAL_NETWORK/OS_EXTERNAL_NETWORK legacy roles, the
// OS_USE_* flags and an OS_PROFILE built-in vendor profile shape discovery.
// String has_<service> values and shutdown profiles are configuration errors,
// and a typed WithNetworkRoles policy still replaces the environment roles.
func TestEnvironmentCloudSuppliesNetworkRoleConfiguration(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		env         map[string]string
		typed       bool
		destination string
		calls       int32
		invalid     bool
	}{
		{name: "legacy internal network", env: map[string]string{"OS_INTERNAL_NETWORK": "private"}, destination: "private", calls: 1},
		{name: "legacy external network", env: map[string]string{"OS_EXTERNAL_NETWORK": "public"}, calls: 2},
		{name: "use flags disable", env: map[string]string{"OS_USE_EXTERNAL_NETWORK": "false", "OS_USE_INTERNAL_NETWORK": "FALSE"}, calls: 0},
		{name: "builtin profile disables network", env: map[string]string{"OS_PROFILE": "rackspace", "OS_INTERNAL_NETWORK": "private"}, calls: 0},
		{name: "typed policy wins", env: map[string]string{"OS_USE_EXTERNAL_NETWORK": "false", "OS_USE_INTERNAL_NETWORK": "false"}, typed: true, destination: "private", calls: 1},
		{name: "string has flag", env: map[string]string{"OS_HAS_NETWORK": "false"}, invalid: true},
		{name: "shutdown profile", env: map[string]string{"OS_PROFILE": "betacloud"}, invalid: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := connectionRoleFixture(t, cloud)
			cloud.Mux.HandleFunc("GET /network/v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"subnets":[]}`)
			})
			cloud.Mux.HandleFunc("POST /v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Subject-Token", "authenticated-token")
				testcloud.JSON(w, 201, `{"token":{"expires_at":"2099-01-01T00:00:00Z","catalog":[]}}`)
			})
			for _, key := range []string{"OS_CLOUD", "OS_CLIENT_CONFIG_FILE", "OS_REGION_NAME", "OS_INTERFACE", "OS_USERID", "OS_PROJECT_ID", "OS_PROJECT_NAME", "OS_TENANT_ID", "OS_TENANT_NAME", "OS_DOMAIN_ID", "OS_APPLICATION_CREDENTIAL_ID", "OS_APPLICATION_CREDENTIAL_NAME", "OS_APPLICATION_CREDENTIAL_SECRET", "OS_SYSTEM_SCOPE", "OS_PASSCODE", "OS_PROFILE", "OS_HAS_NETWORK", "OS_INTERNAL_NETWORK", "OS_EXTERNAL_NETWORK", "OS_USE_EXTERNAL_NETWORK", "OS_USE_INTERNAL_NETWORK", "OS_NETWORKS", "OS_SECGROUP_SOURCE"} {
				// Register restoration, then remove: Python treats an empty
				// variable as a present setting.
				t.Setenv(key, "")
				os.Unsetenv(key)
			}
			t.Setenv("OS_AUTH_URL", cloud.Server.URL+"/v3")
			t.Setenv("OS_USERNAME", "env-user")
			t.Setenv("OS_PASSWORD", "env-password")
			t.Setenv("OS_DOMAIN_NAME", "Default")
			for key, value := range scenario.env {
				t.Setenv(key, value)
			}
			opts := []sdk.ConnectionOption{sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network")}
			if scenario.typed {
				opts = append(opts, sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true})))
			}
			conn, err := sdk.Connect(context.Background(), opts...)
			if scenario.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) || conn != nil {
					t.Fatal(conn, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			destination, err := conn.GetNATDestination(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if calls.Load() != scenario.calls {
				t.Fatal(calls.Load())
			}
			if scenario.destination == "" {
				if destination != nil {
					t.Fatal(destination)
				}
				return
			}
			if destination == nil || destination.ID != scenario.destination {
				t.Fatal(destination)
			}
		})
	}
}
