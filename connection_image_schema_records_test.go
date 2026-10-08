package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/image"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionImageSchemaRecordsShareClientAndRetainBareLocation(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind image.SchemaKind
		path string
		meta bool
	}{
		{"ordinary schema", image.SchemaImage, "image", false},
		{"metadef schema", image.SchemaMetadefProperty, "metadefs/property", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud, region, projectName := "configured-cloud", "configured-region", "configured-project"
			facts := resource.CloudLocation{Cloud: &cloud, RegionName: &region, Zone: json.RawMessage(`["bare-zone",900719925474099312345]`), Project: resource.CloudProject{ID: json.RawMessage("900719925474099312345"), Name: &projectName}}
			wantLocation, err := json.Marshal(facts)
			th.AssertNoErr(t, err)
			provider := &gophercloud.ProviderClient{}
			provider.UseTokenLock()
			provider.SetToken("before-options")
			calls, callbacks := 0, 0
			var actualHeader http.Header
			const raw = `{"id":false,"name":[1,null],"additionalProperties":{"n":900719925474099312345},"properties":{"nullable":null},"required":[false,{},1],"location":{"server":"foreign"},"project_id":"foreign-project","availability_zone":"foreign-zone","$ref":"https://foreign.test/ref"}`
			provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.String() != "https://cloud.test/reverse/glance/v2/schemas/"+tc.path || req.Body != nil || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "shared" || req.Header.Get("X-Call") != "owned" || req.Header.Get("X-Auth-Token") != "after-options" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Fatal(req.Method, req.URL, req.Header, req.Body)
				}
				actualHeader = http.Header{"X-Connection-Proof": {fmt.Sprint(calls)}, "Content-Type": {"text/plain"}, "Link": {`<https://foreign.test/next>; rel="next"`}}
				return &http.Response{StatusCode: 201, Header: actualHeader, Body: io.NopCloser(strings.NewReader(raw))}, nil
			})
			conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/reverse/glance/v2/"), sdk.WithCloudLocation(facts))
			th.AssertNoErr(t, err)
			cloud = "caller changed cloud"
			facts.Zone[0] = '!'
			facts.Project.ID[0] = '8'
			ctx := context.Background()
			service, err := conn.Image(ctx)
			th.AssertNoErr(t, err)
			versioned, err := conn.ImageV2(ctx)
			th.AssertNoErr(t, err)
			client := versioned.RawClient()
			client.Microversion = "2.10"
			client.MoreHeaders = map[string]string{"X-Source": "shared"}
			cached, err := conn.Image(ctx)
			if err != nil || cached != service || service.RawClient() != client || service.API.RawClient() != client || client.ProviderClient != provider {
				t.Fatal("schema record binding changed Connection client", service, cached, err)
			}
			got, err := service.GetSchemaRecord(ctx, tc.kind, func(config *image.GetSchemaOpts) error {
				callbacks++
				provider.SetToken("after-options")
				return image.WithGetSchemaHeader("X-Call", "owned")(config)
			})
			if err != nil || got == nil || got.Kind != tc.kind || got.Resource == nil || got.Wire == nil || got.StatusCode != 201 || got.Header.Get("X-Connection-Proof") != "1" || string(got.Envelope) != raw || callbacks != 1 || calls != 1 {
				t.Fatal(got, err, calls, callbacks)
			}
			th.AssertEquals(t, string(wantLocation), string(got.Resource.Body["location"]))
			th.AssertEquals(t, `{"server":"foreign"}`, string(got.Wire.Body["location"]))
			th.AssertEquals(t, "false", string(got.Resource.Body["id"]))
			th.AssertEquals(t, `[1,null]`, string(got.Resource.Body["name"]))
			th.AssertEquals(t, `{"nullable":null}`, string(got.Resource.Body["properties"]))
			if tc.meta {
				th.AssertEquals(t, 7, len(got.Resource.Body))
				th.AssertEquals(t, "true", string(got.Resource.Body["additional_properties"]))
				th.AssertEquals(t, `[false,{},1]`, string(got.Resource.Body["required"]))
			} else {
				th.AssertEquals(t, 5, len(got.Resource.Body))
				th.AssertEquals(t, `{"n":900719925474099312345}`, string(got.Resource.Body["additional_properties"]))
			}
			th.AssertEquals(t, "", client.MoreHeaders["X-Call"])
			th.AssertEquals(t, "2.10", client.Microversion)
			got.Resource.Body["location"][0] = '!'
			got.Resource.Header.Set("X-Connection-Proof", "view changed")
			got.Wire.Header.Set("X-Connection-Proof", "wire changed")
			got.Header.Set("X-Connection-Proof", "record changed")
			got.Envelope[0] = '!'
			current, err := conn.CurrentLocation()
			th.AssertNoErr(t, err)
			currentJSON, err := json.Marshal(current)
			th.AssertNoErr(t, err)
			if string(currentJSON) != string(wantLocation) || string(got.Wire.Body["location"]) != `{"server":"foreign"}` || actualHeader.Get("X-Connection-Proof") != "1" {
				t.Fatal("schema view leaked into Connection or actual receipt", string(currentJSON), got)
			}
		})
	}
}
