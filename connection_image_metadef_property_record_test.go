package openstack_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionImageMetadefPropertyRecordUsesOwnedLocationAndLiveAuth(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	cloud := "configured-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	expected, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("initial")
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls > 2 || req.Method != http.MethodGet || req.URL.String() != endpoint+"metadefs/namespaces/OS::Nova/properties/seed-id" || req.Body != nil || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("X-Option") != "owned" {
			t.Fatal("unexpected property request", req.Method, req.URL, req.Header)
		}
		return &http.Response{StatusCode: 203, Header: http.Header{"X-Proof": {"actual"}}, Body: io.NopCloser(strings.NewReader(`{"id":null,"name":"passive","location":{"cloud":"foreign"},"namespace_name":"foreign"}`))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	cloud = "caller mutation"
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	if service.API.MetadefProperties.RawClient() != versioned.RawClient() || versioned.RawClient().ProviderClient != provider || calls != 0 {
		t.Fatal("binding lost shared client or made HTTP")
	}
	seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"seed-id"`), "title": json.RawMessage(`"seed-title"`)}}}
	for _, api := range []*metadefproperties.API{service.API.MetadefProperties, versioned.MetadefProperties} {
		scope, err := api.InNamespace(context.Background(), "OS::Nova")
		th.AssertNoErr(t, err)
		record, err := scope.GetRecord(context.Background(), metadefproperties.RecordRequest{Resource: seed}, metadefproperties.WithRecordGetAttribute("min_length", "03"), func(opts *metadefproperties.RecordGetOpts) error {
			provider.SetToken("live")
			opts.Headers = map[string]string{"X-Option": "owned"}
			return nil
		})
		th.AssertNoErr(t, err)
		if record.Namespace != "OS::Nova" || string(record.Resource.Body["location"]) != string(expected) || string(record.Resource.Body["namespace_name"]) != `"OS::Nova"` || string(record.Resource.Body["id"]) != "null" || string(record.Resource.Body["title"]) != `"seed-title"` || string(record.Resource.Body["min_length"]) != "3" {
			t.Fatal("Connection facts or property seed projection lost", record)
		}
		if record.Wire == nil || string(record.Wire.Body["location"]) != `{"cloud":"foreign"}` || record.StatusCode != 203 || record.Resource.Header.Get("X-Proof") != "actual" || string(seed.Body["id"]) != `"seed-id"` || len(seed.Body) != 2 {
			t.Fatal("receipt ownership or input immutability lost", record, seed)
		}
	}
	th.AssertEquals(t, 2, calls)
}
