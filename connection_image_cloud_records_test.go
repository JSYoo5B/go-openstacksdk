package openstack_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken(t *testing.T) {
	const base = "https://cloud.test/reverse/glance/v2/"
	cloud := "image-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	location, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("image-0")
	page := `{"images":[{"id":"a","name":"ubuntu","status":"active","location":{"foreign":true}},{"id":"d","name":"ubuntu-old","status":"deleted"}],"next":null}`
	paths := []string{"images?member_status=all", "images", "images", "images/ubuntu", "images/a"}
	bodies := []string{page, page, page, `{"id":"a","name":"ubuntu","status":"active"}`, `{"id":"a","name":"ubuntu","status":"active"}`}
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		if calls >= len(paths) {
			t.Fatal("unexpected request", req.URL)
		}
		index := calls
		if req.Method != http.MethodGet || req.URL.String() != base+paths[index] || req.Body != nil || req.Header.Get("X-Auth-Token") != fmt.Sprintf("image-%d", index) || req.Header.Get("X-Call") != "cloud" {
			t.Fatal(index, req.URL, req.Header)
		}
		calls++
		provider.SetToken(fmt.Sprintf("image-%d", calls))
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}}, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", base), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	cloud = "caller changed"
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	header := image.WithImageRecordQueryHeader("X-Call", "cloud")
	ctx := context.Background()

	all, err := service.AllCloudImageRecords(ctx, header, image.WithImageRecordQueryShowAll(true))
	th.AssertNoErr(t, err)
	searched, err := service.SearchImageRecords(ctx, "ubuntu*", header)
	th.AssertNoErr(t, err)
	filtered, err := service.GetCloudImageRecord(ctx, "ubuntu", header, image.WithImageRecordQueryFilters(json.RawMessage(`{}`)))
	th.AssertNoErr(t, err)
	found, err := service.GetCloudImageRecord(ctx, "ubuntu", header)
	th.AssertNoErr(t, err)
	byID, err := service.GetImageRecordByID(ctx, "a", header)
	th.AssertNoErr(t, err)
	if calls != 5 || len(all.Images) != 2 || len(searched.Images) != 1 || len(searched.Inventory) != 2 || filtered.Image == nil || found.Image == nil || byID == nil {
		t.Fatal(all, searched, filtered, found, byID, calls)
	}
	for _, record := range []*image.ImageRecord{all.Images[0], all.Images[1], searched.Images[0], filtered.Image, found.Image, byID} {
		if string(record.Resource.Body["location"]) != string(location) || len(record.Resource.Body) != 65 {
			t.Fatal(record.Resource.Body["location"], string(location))
		}
	}
	if string(searched.Images[0].Wire.Body["location"]) != `{"foreign":true}` {
		t.Fatal("actual wire location is retained", searched.Images[0].Wire.Body)
	}
}
