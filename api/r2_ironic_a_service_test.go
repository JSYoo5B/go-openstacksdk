package api_test

import (
	"testing"

	baremetal "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

func TestR2IronicAServiceExposesTheSDKOwnedBindingsOnItsClient(t *testing.T) {
	cloud := testcloud.New(t)
	client := ironicClient(cloud, "baremetal", "1.32")
	service := baremetal.New(client)
	for name, raw := range map[string]any{
		"chassis":          service.Chassis().RawClient(),
		"portgroups":       service.PortGroups().RawClient(),
		"volumeconnectors": service.VolumeConnectors().RawClient(),
		"volumetargets":    service.VolumeTargets().RawClient(),
	} {
		if raw != client {
			t.Fatalf("%s must share the service client", name)
		}
	}
}
