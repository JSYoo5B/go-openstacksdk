package openstack_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionOwnedImageUpdateSharesBindingSourceAndCleanLifecycle(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "fixed/한글"
	cloud := "owned-image-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"project"`)}}
	location, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("first")
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.String() != endpoint+"images/"+url.PathEscape(id) || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "shared" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
			t.Fatal(req.Method, req.URL, req.Header)
		}
		body := `{"id":"fixed/한글","name":"before","protected":"false"}`
		if calls == 1 {
			if req.Method != http.MethodGet || req.Body != nil || req.Header.Get("X-Auth-Token") != "first" {
				t.Fatal(req.Method, req.Body, req.Header)
			}
			provider.SetToken("live")
		} else if calls == 2 {
			if req.Method != http.MethodPatch || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("X-Call") != "owned-update" || req.Header.Get("Accept") != "" || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" {
				t.Fatal(req.Method, req.Header)
			}
			raw, err := io.ReadAll(req.Body)
			th.AssertNoErr(t, err)
			var actual []map[string]json.RawMessage
			th.AssertNoErr(t, json.Unmarshal(raw, &actual))
			th.CheckDeepEquals(t, []map[string]json.RawMessage{{"op": json.RawMessage(`"replace"`), "path": json.RawMessage(`"/name"`), "value": json.RawMessage(`"after"`)}}, actual)
			body = `{"name":"server name","vendor":900719925474099312345}`
		} else {
			t.Fatal("clean record replayed PATCH", calls, req.URL)
		}
		return &http.Response{Request: req, StatusCode: 203, Header: http.Header{"X-Proof": {"actual"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	upper, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source := versioned.RawClient()
	source.MoreHeaders, source.Microversion = map[string]string{"X-Source": "shared"}, "2.10"
	if upper.RawClient() != source || upper.API.RawClient() != source || source.ProviderClient != provider || calls != 0 {
		t.Fatal("connection image services do not share binding")
	}
	seed, err := upper.GetImageRecord(context.Background(), image.ImageRecordRequest{ID: id})
	th.AssertNoErr(t, err)
	updated, err := conn.UpdateImageRecord(context.Background(), image.ImageRecordUpdateRequest{Record: seed}, image.WithImageRecordAttribute("name", "after"), image.WithImageRecordHeader("X-Call", "owned-update"))
	th.AssertNoErr(t, err)
	if updated == nil || calls != 2 || string(updated.Resource.Body["name"]) != `"server name"` || string(updated.Resource.Body["is_protected"]) != "true" || string(updated.Resource.Body["location"]) != string(location) || updated.StatusCode != 203 || updated.Header.Get("X-Proof") != "actual" || string(updated.Wire.Body["vendor"]) != "900719925474099312345" {
		t.Fatal(updated, calls)
	}
	clean, err := upper.UpdateImageRecord(context.Background(), image.ImageRecordUpdateRequest{Record: updated})
	th.AssertNoErr(t, err)
	if clean == nil || clean == updated || clean.Resource == updated.Resource || clean.Wire == updated.Wire || calls != 2 || string(seed.Resource.Body["name"]) != `"before"` || source.MoreHeaders["X-Source"] != "shared" {
		t.Fatal(clean, updated, seed, calls)
	}
	clean.Resource.Body["name"][1] = 'X'
	clean.Wire.Body["vendor"][0] = '0'
	clean.Header.Set("X-Proof", "changed")
	if string(updated.Resource.Body["name"]) != `"server name"` || string(updated.Wire.Body["vendor"]) != "900719925474099312345" || updated.Header.Get("X-Proof") != "actual" {
		t.Fatal("connection forwarding aliases returned records")
	}
}
