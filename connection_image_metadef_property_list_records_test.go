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

func TestConnectionImageMetadefPropertyListsShareOwnedLocationAndFiniteRoutes(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const body = `{"properties":{"dictionary/name":{"minLength":"03","title":{"raw":true},"location":{"cloud":"foreign"},"namespace_name":"foreign"},"drop":{"minLength":0}},"next":"https://foreign.test/","properties_links":[{"rel":"next","href":"https://foreign.test/"}]}`
	cloud := "configured-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	expected, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("initial")
	calls, callbacks := 0, 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls > 2 || req.Method != http.MethodGet || req.URL.String() != endpoint+"metadefs/namespaces/OS::Nova/properties" || req.Body != nil || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("X-Option") != "owned" {
			t.Fatal("unexpected property list request", req.Method, req.URL, req.Header)
		}
		return &http.Response{StatusCode: 203, Header: http.Header{"X-Proof": {"actual"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	cloud = "caller mutation"
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	if service.API.MetadefProperties.RawClient() != versioned.RawClient() || versioned.RawClient().ProviderClient != provider || calls != 0 {
		t.Fatal("binding lost the original client or made HTTP")
	}
	options := []metadefproperties.RecordListOption{
		metadefproperties.WithRecordListFilter("min_length", 3),
		metadefproperties.WithRecordListMaxItems(2),
		func(opts *metadefproperties.RecordListOpts) error {
			callbacks++
			provider.SetToken("live")
			opts.Headers = map[string]string{"X-Option": "owned"}
			return nil
		},
	}
	for index, api := range []*metadefproperties.API{service.API.MetadefProperties, versioned.MetadefProperties} {
		scope, err := api.InNamespace(context.Background(), "OS::Nova")
		th.AssertNoErr(t, err)
		var rows []*metadefproperties.Record
		if index == 0 {
			sequence := scope.ListRecords(context.Background(), options...)
			if calls != 0 || callbacks != 0 {
				t.Fatal("list creation was eager")
			}
			for record, listErr := range sequence {
				th.AssertNoErr(t, listErr)
				rows = append(rows, record)
			}
		} else {
			rows, err = scope.AllRecords(context.Background(), options...)
			th.AssertNoErr(t, err)
		}
		if len(rows) != 1 {
			t.Fatal("projected local filter lost", rows)
		}
		record := rows[0]
		if record.Key == nil || *record.Key != "dictionary/name" || record.Namespace != "OS::Nova" || string(record.Resource.Body["name"]) != `"dictionary/name"` || string(record.Resource.Body["id"]) != `"dictionary/name"` || string(record.Resource.Body["location"]) != string(expected) || string(record.Resource.Body["namespace_name"]) != `"OS::Nova"` || string(record.Resource.Body["min_length"]) != "3" {
			t.Fatal("key seed or Connection facts lost", record)
		}
		if record.Wire == nil || string(record.Wire.Body["location"]) != `{"cloud":"foreign"}` || record.Wire.Body["name"] != nil || record.StatusCode != 203 || record.Header.Get("X-Proof") != "actual" || string(record.Envelope) != body {
			t.Fatal("actual dictionary row/page receipt lost", record)
		}
	}
	th.AssertEquals(t, 2, calls)
	th.AssertEquals(t, 2, callbacks)
}
