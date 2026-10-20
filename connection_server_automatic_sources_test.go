package openstack_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionStandaloneAutomaticIPRetainsLazyComputeSourceGuard(t *testing.T) {
	for _, field := range []string{"API", "Servers", "Flavors"} {
		t.Run(field, func(t *testing.T) {
			cloud, conn, posts, raw := connectionReadyFixture(t, "progress")
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { return cloud.Server.URL + "/compute/", nil }
			server := connectionAddressServer()
			server.Status = "ACTIVE"
			callbacks := 0
			result, err := conn.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: server},
				compute.WithAutomaticIPTimeout(time.Second), compute.WithAutomaticIPPollInterval(time.Millisecond),
				compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false)),
				compute.WithAutomaticIPProgress(func(*compute.Server) error {
					callbacks++
					cached, err := conn.Compute(context.Background())
					if err != nil {
						return err
					}
					switch field {
					case "API":
						cached.API = nil
					case "Servers":
						cached.Servers = nil
					case "Flavors":
						cached.Flavors = nil
					}
					return nil
				}))
			if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.Observed || result.Assignment == nil || !result.Assignment.Allocated || result.Server.Name != "matching raw" || posts.Load() != 1 || raw.Load() != 1 || callbacks != 1 {
				t.Fatal(result, err, posts.Load(), raw.Load(), callbacks)
			}
		})
	}
}
