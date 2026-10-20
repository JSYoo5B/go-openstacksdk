package cloudread_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageCaptureOwnsHeaderPolicyAndKeepsOriginalSourceSticky(t *testing.T) {
	for _, kind := range []string{"", "image"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client(kind, "/glance/v2/")
			client.MoreHeaders = map[string]string{"X-Source": "original", "Accept": "application/source"}
			captured, err := cloudread.Capture(context.Background(), client, "image")
			if err != nil {
				t.Fatal(err)
			}
			if captured.Client.Type != "image" || captured.Client.ProviderClient != client.ProviderClient {
				t.Fatal(captured)
			}
			if err := captured.WithPolicy(context.Background(), nil, map[string]string{"Accept": "application/json", "X-Owned": "private"}); err != nil {
				t.Fatal(err)
			}
			captured.Client.MoreHeaders["X-Source"] = "private"
			if client.MoreHeaders["X-Source"] != "original" || client.MoreHeaders["Accept"] != "application/source" || client.MoreHeaders["X-Owned"] != "" {
				t.Fatal(client.MoreHeaders)
			}
			original := client.ResourceBase
			client.ResourceBase = cloud.Server.URL + "/foreign/"
			if !errors.Is(captured.Guard(context.Background()), resource.ErrInvalidOption) {
				t.Fatal("source change was accepted")
			}
			client.ResourceBase = original
			if !errors.Is(captured.Guard(context.Background()), resource.ErrInvalidOption) {
				t.Fatal("restored source erased observed fault")
			}
		})
	}
}

func TestImageCaptureRejectsForeignProtocolAndRoleBeforeHTTP(t *testing.T) {
	for _, headers := range []map[string]string{{"Authorization": "foreign"}, {"X-OpenStack-Nova-API-Version": "2.1"}, {"X-OpenStack-Volume-API-Version": "3.1"}, {"OpenStack-API-Version": "compute 2.1"}} {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/glance/v2/")
		client.MoreHeaders = headers
		if _, err := cloudread.Capture(context.Background(), client, "image"); err == nil {
			t.Fatal(headers)
		}
	}
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/nova/v2.1/")
	if _, err := cloudread.Capture(context.Background(), client, "image"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
}
