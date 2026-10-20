package cloudread

import (
	"context"
	"errors"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPlacementSourceOwnsVersionHeadersAndStickyRoute(t *testing.T) {
	for _, kind := range []string{"", "placement"} {
		t.Run("kind="+kind, func(t *testing.T) {
			client := cloudReadPolicyClient("https://placement.test/reverse/")
			client.Type = kind
			client.Microversion = "1.34"
			client.MoreHeaders["OpenStack-API-Version"] = "placement 1.34"
			source, err := Capture(context.Background(), client, "placement")
			if err != nil || source.Client.Type != "placement" || client.Type != kind {
				t.Fatal(source, err, client.Type)
			}
			if err := source.WithPolicy(context.Background(), nil, map[string]string{"OpenStack-API-Version": "placement 1.34"}); err != nil {
				t.Fatal(err)
			}
			client.MoreHeaders["X-Captured"] = "later"
			if source.Client.MoreHeaders["X-Captured"] != "original" {
				t.Fatal(source.Client.MoreHeaders)
			}
			original := client.ResourceBase
			client.ResourceBase = "https://placement.test/other/"
			if err := source.Guard(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			client.ResourceBase = original
			if err := source.Guard(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("route change forgotten", err)
			}
		})
	}
}

func TestPlacementSourceRejectsConflictingServiceAndHeaders(t *testing.T) {
	for _, kind := range []string{"compute", "network", "volumev3", "identity", "key-manager"} {
		client := cloudReadPolicyClient("https://placement.test/")
		client.Type = kind
		if source, err := Capture(context.Background(), client, "placement"); source != nil || !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(kind, source, err)
		}
	}
	for _, headers := range []map[string]string{
		{"OpenStack-API-Version": "compute 2.96"},
		{"OpenStack-API-Version": "placement 1.33"},
		{"X-OpenStack-Nova-API-Version": "1.34"},
		{"X-OpenStack-Volume-API-Version": "1.34"},
		{"X-Auth-Token": "caller-token"},
	} {
		client := cloudReadPolicyClient("https://placement.test/")
		client.Type = "placement"
		client.Microversion = "1.34"
		client.MoreHeaders = headers
		if source, err := Capture(context.Background(), client, "placement"); source != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, source, err)
		}
	}
}
