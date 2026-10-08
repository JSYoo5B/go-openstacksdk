package openstack_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionImageRecordWaitsShareOwnedSourceAndLatestObservation(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	cloud := "wait-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	location, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("wait-0")
	calls := 0
	var source *gophercloud.ServiceClient
	bodies := []string{
		`{"id":"fixed","status":"queued","owner":"original","properties":{"seed":true}}`,
		`{"id":"response-other","status":"saving","vendor":"first","size":"100"}`,
		`{"status":"ACTIVE","owner":"updated"}`,
		`{"id":"drift-delete","status":"pending","owner":"delete-owner"}`,
		`{"message":"missing"}`,
	}
	codes := []int{200, 203, 206, 201, 404}
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		if index >= len(bodies) {
			t.Fatal("unexpected request", req.URL)
		}
		ordinary, call := "source-0", "seed"
		if index == 1 || index == 2 {
			ordinary, call = "source-1", "status"
		}
		if index == 3 || index == 4 {
			ordinary, call = "source-3", "delete"
		}
		if req.Method != http.MethodGet || req.URL.String() != endpoint+"images/fixed" || req.Body != nil || req.Header.Get("X-Auth-Token") != fmt.Sprintf("wait-%d", index) || req.Header.Get("X-Source") != ordinary || req.Header.Get("X-Call") != call {
			t.Fatal(index, req.Method, req.URL, req.Header)
		}
		calls++
		provider.SetToken(fmt.Sprintf("wait-%d", calls))
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		header := http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}}
		if index == 0 || index == 1 {
			header.Set("OpenStack-image-import-methods", "web-download, glance-direct")
		}
		return &http.Response{Request: req, StatusCode: codes[index], Header: header, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	native, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source = native.RawClient()
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	if source != service.RawClient() || source.ProviderClient != provider || calls != 0 {
		t.Fatal("shared source changed")
	}
	seed, err := service.GetImageRecord(context.Background(), image.ImageRecordRequest{ID: "fixed"}, image.WithImageRecordHeader("X-Call", "seed"))
	th.AssertNoErr(t, err)
	originalBody := string(seed.Envelope)
	callbacks, options := 0, 0
	status, err := service.WaitForImageRecordStatus(context.Background(), seed, "active",
		image.WithImageRecordWaitPollInterval(time.Millisecond), image.WithImageRecordWaitTimeout(time.Second),
		image.WithImageRecordWaitHeader("X-Call", "status"),
		func(config *image.ImageRecordWaitOpts) error {
			options++
			config.Callback = func(progress int) {
				callbacks++
				if progress != 0 {
					t.Fatal(progress)
				}
			}
			return nil
		})
	th.AssertNoErr(t, err)
	if calls != 3 || options != 1 || callbacks != 1 || status == nil || status == seed || status.StatusCode != 206 || string(status.Resource.Body["id"]) != `"response-other"` || string(status.Resource.Body["size"]) != "100" || string(status.Resource.Body["owner_id"]) != `"updated"` || len(status.ImportMethods) != 0 {
		t.Fatal(status, calls, options, callbacks)
	}
	if string(status.Resource.Body["location"]) != string(location) || status.Header.Get("X-Proof") != "response-2" || string(seed.Envelope) != originalBody || string(seed.Resource.Body["status"]) != `"queued"` || seed.Header.Get("X-Proof") != "response-0" || len(seed.ImportMethods) != 2 {
		t.Fatal("source facts or seed ownership lost", seed, status)
	}
	deleted, err := service.WaitForImageRecordDelete(context.Background(), seed,
		image.WithImageRecordWaitPollInterval(time.Millisecond), image.WithImageRecordWaitTimeout(time.Second),
		image.WithImageRecordWaitHeader("X-Call", "delete"),
		func(config *image.ImageRecordWaitOpts) error {
			options++
			config.Callback = func(progress int) {
				callbacks++
				if progress != 0 {
					t.Fatal(progress)
				}
			}
			return nil
		})
	th.AssertNoErr(t, err)
	if calls != 5 || options != 2 || callbacks != 2 || deleted == nil || deleted.StatusCode != 201 || deleted.Header.Get("X-Proof") != "response-3" || string(deleted.Resource.Body["id"]) != `"drift-delete"` || string(deleted.Resource.Body["owner_id"]) != `"delete-owner"` || string(deleted.Resource.Body["location"]) != string(location) {
		t.Fatal(deleted, calls, options, callbacks)
	}
	if string(seed.Envelope) != originalBody || string(seed.Resource.Body["status"]) != `"queued"` {
		t.Fatal("caller seed changed", seed)
	}
	deleted.Resource.Body["location"][0] = '!'
	deleted.Header.Set("X-Proof", "caller")
	if string(status.Resource.Body["location"]) != string(location) || seed.Header.Get("X-Proof") != "response-0" || status.Header.Get("X-Proof") != "response-2" || source.MoreHeaders["X-Source"] != "source-5" {
		t.Fatal("observations or original source aliased")
	}
}
