package openstack_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const body = `{"name":"passive/name","minItems":"04","location":{"cloud":"foreign"},"namespace_name":"foreign"}`
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
		if req.URL.RawQuery != "" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("X-Option") != "owned" {
			t.Fatal("unexpected property write request", req.Method, req.URL, req.Header)
		}
		payload, err := io.ReadAll(req.Body)
		th.AssertNoErr(t, err)
		var decoded map[string]json.RawMessage
		th.AssertNoErr(t, json.Unmarshal(payload, &decoded))
		want := map[string]json.RawMessage{"title": json.RawMessage(`null`)}
		path := endpoint + "metadefs/namespaces/OS::Nova/properties/seed-id"
		if req.Method == http.MethodPost {
			path = endpoint + "metadefs/namespaces/OS::Nova/properties"
			want = map[string]json.RawMessage{"name": json.RawMessage(`"new-property"`), "minLength": json.RawMessage(`"03"`)}
		} else if req.Method != http.MethodPut {
			t.Fatal(req.Method)
		}
		if req.URL.String() != path || !reflect.DeepEqual(decoded, want) {
			t.Fatal("owned raw payload or fixed route changed", req.URL, string(payload))
		}
		return &http.Response{StatusCode: 202, Header: http.Header{"X-Proof": {"actual"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	cloud = "caller mutation"
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	if service.API.MetadefProperties.RawClient() != versioned.RawClient() || calls != 0 {
		t.Fatal("shared client lost")
	}
	for index, api := range []*metadefproperties.API{service.API.MetadefProperties, versioned.MetadefProperties} {
		t.Run([]string{"Image", "ImageV2"}[index], func(t *testing.T) {
			scope, err := api.InNamespace(context.Background(), "OS::Nova")
			th.AssertNoErr(t, err)
			provider.SetToken("live")
			created, err := scope.CreateRecord(context.Background(), metadefproperties.WithRecordCreateAttributes(map[string]any{"name": "new-property", "min_length": "03"}), metadefproperties.WithRecordCreateHeader("X-Option", "owned"))
			th.AssertNoErr(t, err)
			seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"seed-id"`), "title": json.RawMessage(`"discarded old title"`), "minLength": json.RawMessage(`"not an integer"`)}, Header: http.Header{"X-Old": {"seed"}}, StatusCode: 201}}
			updated, err := scope.UpdateRecord(context.Background(), metadefproperties.RecordRequest{Resource: seed}, metadefproperties.WithRecordUpdateAttribute("title", nil), metadefproperties.WithRecordUpdateHeader("X-Option", "owned"))
			th.AssertNoErr(t, err)
			for _, record := range []*metadefproperties.Record{created, updated} {
				if record.Namespace != "OS::Nova" || record.Key != nil || string(record.Resource.Body["location"]) != string(expected) || string(record.Resource.Body["namespace_name"]) != `"OS::Nova"` || string(record.Resource.Body["name"]) != `"passive/name"` || string(record.Resource.Body["min_items"]) != "4" || record.StatusCode != 202 || record.Header.Get("X-Proof") != "actual" || string(record.Envelope) != body || record.Wire == nil || string(record.Wire.Body["location"]) != `{"cloud":"foreign"}` {
					t.Fatal("Connection view/receipt lost", record)
				}
			}
			if string(created.Resource.Body["min_length"]) != "3" || string(created.Resource.Body["id"]) != `"passive/name"` || string(updated.Resource.Body["id"]) != `"seed-id"` || string(updated.Resource.Body["title"]) != "null" || string(seed.Body["title"]) != `"discarded old title"` {
				t.Fatal(created, updated, seed)
			}
			before := calls
			noop, err := scope.UpdateRecord(context.Background(), metadefproperties.RecordRequest{Resource: seed}, metadefproperties.WithRecordUpdateHeader("X-Option", "owned"))
			th.AssertNoErr(t, err)
			if calls != before || noop.StatusCode != 0 || noop.Wire != nil || noop.Envelope != nil || noop.Header != nil || noop.Resource.StatusCode != 0 || noop.Resource.Header != nil || string(noop.Resource.Body["location"]) != string(expected) || string(noop.Resource.Body["id"]) != `"seed-id"` || string(noop.Resource.Body["name"]) != "null" || string(noop.Resource.Body["title"]) != "null" {
				t.Fatal("no-op fabricated response or reused definition", noop, calls)
			}
		})
	}
	th.AssertEquals(t, 4, calls)
}
