package cloudread

import (
	"context"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"testing"
)

func TestNetworkSourceCaptureKeepsPrivateTypeAndStickyRoute(t *testing.T) {
	for _, kind := range []string{"", "network"} {
		t.Run("kind="+kind, func(t *testing.T) {
			client := cloudReadPolicyClient("https://network.test/reverse/v2.0/")
			client.Type = kind
			client.Microversion = ""
			source, err := Capture(context.Background(), client, "network")
			if err != nil || source.Client.Type != "network" || client.Type != kind {
				t.Fatal(source, err, client.Type)
			}
			client.MoreHeaders["X-Captured"] = "later"
			if source.Client.MoreHeaders["X-Captured"] != "original" || source.Guard(context.Background()) != nil {
				t.Fatal(source)
			}
			original := client.ResourceBase
			client.ResourceBase = "https://network.test/changed/v2.0/"
			if err := source.Guard(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			client.ResourceBase = original
			if err := source.Guard(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("route change was forgotten", err)
			}
		})
	}
}

func TestNetworkSourceCaptureRejectsCrossServiceAndVersionHeaders(t *testing.T) {
	for _, kind := range []string{"compute", "volumev3", "identity", "key-manager"} {
		client := cloudReadPolicyClient("https://network.test/v2.0/")
		client.Type = kind
		if got, err := Capture(context.Background(), client, "network"); got != nil || !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(kind, got, err)
		}
	}
	for _, headers := range []map[string]string{
		{"OpenStack-API-Version": "compute 2.96"},
		{"X-OpenStack-Nova-API-Version": "2.96"},
		{"X-OpenStack-Volume-API-Version": "3.71"},
		{"X-Auth-Token": "caller-token"},
	} {
		client := cloudReadPolicyClient("https://network.test/v2.0/")
		client.Type = "network"
		client.Microversion = "2.96"
		client.MoreHeaders = headers
		if got, err := Capture(context.Background(), client, "network"); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, got, err)
		}
	}
}
