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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			provider := &gophercloud.ProviderClient{}
			provider.UseTokenLock()
			provider.SetToken("before-options")
			calls, callbacks := 0, 0
			var actualHeader http.Header
			const raw = `{"id":false,"name":[1,null],"additionalProperties":{"n":900719925474099312345},"properties":{"nullable":null},"required":[false,{},1],"location":{"server":"foreign"},"project_id":"foreign-project","availability_zone":"foreign-zone","$ref":"https://foreign.test/ref"}`
			provider.HTTPClient.Transport = testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.String() != "https://cloud.test/reverse/glance/v2/schemas/"+tc.path || req.Body != nil || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "shared" || req.Header.Get("X-Call") != "owned" || req.Header.Get("X-Auth-Token") != "after-options" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Fatal(req.Method, req.URL, req.Header, req.Body)
				}
				actualHeader = http.Header{"X-Connection-Proof": {fmt.Sprint(calls)}, "Content-Type": {"text/plain"}, "Link": {`<https://foreign.test/next>; rel="next"`}}
				return testcloud.Response(201, raw, actualHeader), nil
			})
			conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/reverse/glance/v2/"), sdk.WithCloudLocation(facts))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cloud = "caller changed cloud"
			facts.Zone[0] = '!'
			facts.Project.ID[0] = '8'
			ctx := context.Background()
			service, err := conn.Image(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			versioned, err := conn.ImageV2(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
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
			if string(got.Resource.Body["location"]) != string(wantLocation) {
				t.Fatalf("string(got.Resource.Body[\"location\"]) = %v, want %v", string(got.Resource.Body["location"]), string(wantLocation))
			}
			if string(got.Wire.Body["location"]) != `{"server":"foreign"}` {
				t.Fatalf("string(got.Wire.Body[\"location\"]) = %v, want %v", string(got.Wire.Body["location"]), `{"server":"foreign"}`)
			}
			if string(got.Resource.Body["id"]) != "false" {
				t.Fatalf("string(got.Resource.Body[\"id\"]) = %v, want %v", string(got.Resource.Body["id"]), "false")
			}
			if string(got.Resource.Body["name"]) != `[1,null]` {
				t.Fatalf("string(got.Resource.Body[\"name\"]) = %v, want %v", string(got.Resource.Body["name"]), `[1,null]`)
			}
			if string(got.Resource.Body["properties"]) != `{"nullable":null}` {
				t.Fatalf("string(got.Resource.Body[\"properties\"]) = %v, want %v", string(got.Resource.Body["properties"]), `{"nullable":null}`)
			}
			if tc.meta {
				if len(got.Resource.Body) != 7 {
					t.Fatalf("len(got.Resource.Body) = %v, want %v", len(got.Resource.Body), 7)
				}
				if string(got.Resource.Body["additional_properties"]) != "true" {
					t.Fatalf("string(got.Resource.Body[\"additional_properties\"]) = %v, want %v", string(got.Resource.Body["additional_properties"]), "true")
				}
				if string(got.Resource.Body["required"]) != `[false,{},1]` {
					t.Fatalf("string(got.Resource.Body[\"required\"]) = %v, want %v", string(got.Resource.Body["required"]), `[false,{},1]`)
				}
			} else {
				if len(got.Resource.Body) != 5 {
					t.Fatalf("len(got.Resource.Body) = %v, want %v", len(got.Resource.Body), 5)
				}
				if string(got.Resource.Body["additional_properties"]) != `{"n":900719925474099312345}` {
					t.Fatalf("string(got.Resource.Body[\"additional_properties\"]) = %v, want %v", string(got.Resource.Body["additional_properties"]), `{"n":900719925474099312345}`)
				}
			}
			if client.MoreHeaders["X-Call"] != "" {
				t.Fatalf("client.MoreHeaders[\"X-Call\"] = %v, want %v", client.MoreHeaders["X-Call"], "")
			}
			if client.Microversion != "2.10" {
				t.Fatalf("client.Microversion = %v, want %v", client.Microversion, "2.10")
			}
			got.Resource.Body["location"][0] = '!'
			got.Resource.Header.Set("X-Connection-Proof", "view changed")
			got.Wire.Header.Set("X-Connection-Proof", "wire changed")
			got.Header.Set("X-Connection-Proof", "record changed")
			got.Envelope[0] = '!'
			current, err := conn.CurrentLocation()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			currentJSON, err := json.Marshal(current)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(currentJSON) != string(wantLocation) || string(got.Wire.Body["location"]) != `{"server":"foreign"}` || actualHeader.Get("X-Connection-Proof") != "1" {
				t.Fatal("schema view leaked into Connection or actual receipt", string(currentJSON), got)
			}
		})
	}
}
