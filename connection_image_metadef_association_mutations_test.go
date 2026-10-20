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
	types "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionImageMetadefAssociationMutationsShareLocationAndIdentity(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const createdBody = `{"name":"passive/name","created_at":[1],"location":{"cloud":"foreign"},"namespace_name":"foreign","vendor":1e400}`
	cloud := "configured-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	expected, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("initial")
	calls, retries := 0, 0
	provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.RawQuery != "" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("X-Option") != "owned" {
			t.Fatal("unexpected association mutation", req.Method, req.URL, req.Header)
		}
		collection := endpoint + "metadefs/namespaces/OS::Nova/resource_types"
		status, body := http.StatusAccepted, "\xffopaque acknowledgement"
		switch req.Method {
		case http.MethodPost:
			payload, err := io.ReadAll(req.Body)
			th.AssertNoErr(t, err)
			var decoded map[string]json.RawMessage
			th.AssertNoErr(t, json.Unmarshal(payload, &decoded))
			want := map[string]json.RawMessage{
				"id": json.RawMessage(`"input-id"`), "name": json.RawMessage(`"OS::Nova::Server"`),
				"created_at": json.RawMessage(`false`), "updated_at": json.RawMessage(`9007199254740993`),
				"prefix": json.RawMessage(`"hw_"`), "properties_target": json.RawMessage(`null`),
			}
			if req.URL.String() != collection || !reflect.DeepEqual(decoded, want) {
				t.Fatal("raw attributes or fixed parent changed", req.URL, string(payload))
			}
			status, body = http.StatusNonAuthoritativeInfo, createdBody
		case http.MethodDelete:
			if req.URL.String() != collection+"/input-id" || req.Body != nil {
				t.Fatal("Resource identity or body changed", req.URL)
			}
			if calls == 4 {
				status = http.StatusNotFound
			}
		default:
			t.Fatal(req.Method)
		}
		return &http.Response{StatusCode: status, Header: http.Header{"X-Proof": {"actual"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	cloud = "caller mutation"
	upper, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	if upper.API.MetadefResourceTypes.RawClient() != versioned.RawClient() || versioned.RawClient().ProviderClient != provider || calls != 0 {
		t.Fatal("binding lost shared provider or made HTTP")
	}
	for index, api := range []*types.API{upper.API.MetadefResourceTypes, versioned.MetadefResourceTypes} {
		t.Run([]string{"Image", "ImageV2"}[index], func(t *testing.T) {
			scope, err := api.InNamespace(context.Background(), "OS::Nova")
			th.AssertNoErr(t, err)
			created, err := scope.CreateRecord(context.Background(), types.WithRecordCreateAttributes(map[string]any{
				"id": "input-id", "name": "OS::Nova::Server", "created_at": false,
				"updated_at": json.Number("9007199254740993"), "prefix": "hw_", "properties_target": nil,
			}), types.WithRecordCreateHeader("X-Option", "owned"), func(*types.RecordCreateOpts) error {
				provider.SetToken("live")
				return nil
			})
			th.AssertNoErr(t, err)
			if created.Namespace == nil || *created.Namespace != "OS::Nova" || len(created.Resource.Body) != 8 ||
				string(created.Resource.Body["location"]) != string(expected) || string(created.Resource.Body["namespace_name"]) != `"OS::Nova"` ||
				string(created.Resource.Body["id"]) != `"input-id"` || string(created.Resource.Body["name"]) != `"passive/name"` ||
				string(created.Resource.Body["created_at"]) != `[1]` || string(created.Resource.Body["updated_at"]) != `9007199254740993` ||
				string(created.Resource.Body["prefix"]) != `"hw_"` || string(created.Resource.Body["properties_target"]) != `null` ||
				created.StatusCode != 203 || created.Header.Get("X-Proof") != "actual" || string(created.Envelope) != createdBody ||
				created.Wire == nil || string(created.Wire.Body["location"]) != `{"cloud":"foreign"}` || string(created.Wire.Body["vendor"]) != `1e400` {
				t.Fatal("Connection projection/raw receipt lost", created)
			}
			seed := created.Resource.Clone()
			seed.Body["namespace_name"] = json.RawMessage(`"different parent"`)
			ack, err := scope.DeleteRecord(context.Background(), types.RecordRequest{Resource: seed}, types.WithDeleteHeader("X-Option", "owned"), func(*types.DeleteOpts) error {
				seed.Body["id"] = json.RawMessage(`"caller changed"`)
				return nil
			})
			th.AssertNoErr(t, err)
			wantStatus := 202
			if index == 1 {
				wantStatus = 404
			}
			if ack == nil || ack.Namespace != "OS::Nova" || ack.Name == nil || *ack.Name != "input-id" ||
				ack.StatusCode != wantStatus || ack.Header.Get("X-Proof") != "actual" || string(ack.Body) != "\xffopaque acknowledgement" ||
				string(seed.Body["namespace_name"]) != `"different parent"` || string(created.Resource.Body["id"]) != `"input-id"` {
				t.Fatal("immutable identity/opaque receipt lost", ack, seed)
			}
		})
	}
	th.AssertEquals(t, 4, calls)
	th.AssertEquals(t, 0, retries)
}
