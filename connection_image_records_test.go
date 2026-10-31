package openstack_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageRecordsShareSourceLocationAndHiddenDiscovery(t *testing.T) {
	const base = "https://cloud.test/reverse/glance/v2/"
	cloud := "image-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	location, err := facts.ForResource(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("image-0")
	bodies := []string{
		`{"name":"got","owner":"foreign-owner","location":{"foreign":true}}`,
		`{"images":[{"id":"first","size":"9007199254740993"}],"next":"/v2/images?marker=tail"}`,
		`{"images":{"id":"last","os_hidden":true},"next":null}`,
		`{"message":"direct forbidden"}`,
		`{"images":[]}`,
		`{"images":[{"id":"passive-id","name":"hidden","location":{"foreign":true},"extension":false}],"next":null}`,
	}
	codes := []int{203, 206, 201, 403, 200, 206}
	paths := []string{"images/get", "images", "images?marker=tail", "images/hidden", "images?name=hidden", "images?os_hidden=True"}
	calls := 0
	provider.HTTPClient.Transport = testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if calls >= len(bodies) {
			t.Fatal("unexpected request", req.URL)
		}
		index := calls
		if req.Method != http.MethodGet || req.URL.String() != base+paths[index] || req.Body != nil || req.Header.Get("X-Auth-Token") != fmt.Sprintf("image-%d", index) || req.Header.Get("X-Source") != "shared" || req.Header.Get("X-Call") != "records" {
			t.Fatal(index, req.URL, req.Header)
		}
		calls++
		provider.SetToken(fmt.Sprintf("image-%d", calls))
		return testcloud.Response(codes[index], bodies[index], http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}, "Openstack-Image-Import-Methods": {"web-download, glance-direct"}}), nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", base), sdk.WithCloudLocation(facts))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cloud = "caller changed"
	upper, err := conn.Image(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	versioned, err := conn.ImageV2(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	client := versioned.RawClient()
	client.MoreHeaders = map[string]string{"X-Source": "shared"}
	if upper.RawClient() != client || upper.API.RawClient() != client || client.ProviderClient != provider || calls != 0 {
		t.Fatal("shared source or eager call")
	}
	got, err := upper.GetImageRecord(context.Background(), image.ImageRecordRequest{ID: "get"}, image.WithImageRecordHeader("X-Call", "records"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || calls != 1 || len(got.Resource.Body) != 65 || string(got.Resource.Body["id"]) != `"get"` || string(got.Resource.Body["owner_id"]) != `"foreign-owner"` || string(got.Resource.Body["location"]) != string(location) || len(got.ImportMethods) != 2 || got.ImportMethods[1] != " glance-direct" {
		t.Fatal(got, calls)
	}
	rows, err := upper.AllImageRecords(context.Background(), image.WithImageRecordListHeader("X-Call", "records"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 || calls != 3 || string(rows[0].Resource.Body["size"]) != "9007199254740993" || string(rows[1].Resource.Body["is_hidden"]) != "true" {
		t.Fatal(rows, calls)
	}
	found, err := upper.FindImageRecord(context.Background(), "hidden", image.WithFindImageRecordHeader("X-Call", "records"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil || calls != 6 || found.StatusCode != 206 || string(found.Resource.Body["id"]) != `"passive-id"` || string(found.Resource.Body["location"]) != string(location) || string(found.Wire.Body["location"]) != `{"foreign":true}` {
		t.Fatal(found, calls)
	}
	for _, record := range []*image.ImageRecord{got, rows[0], rows[1], found} {
		if string(record.Resource.Body["location"]) != string(location) || len(record.Resource.Body) != 65 {
			t.Fatal(record)
		}
	}
	got.Resource.Body["location"][0] = '!'
	got.Header.Set("X-Proof", "caller")
	if string(found.Resource.Body["location"]) != string(location) || rows[0].Header.Get("X-Proof") != "response-1" || found.Header.Get("X-Proof") != "response-5" || client.MoreHeaders["X-Source"] != "shared" {
		t.Fatal("caller aliases or source header mutation")
	}
}
