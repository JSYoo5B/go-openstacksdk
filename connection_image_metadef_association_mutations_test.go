package openstack_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	types "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageMetadefAssociationMutationsShareLocationAndIdentity(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const createdBody = `{"name":"passive/name","created_at":[1],"location":{"cloud":"foreign"},"namespace_name":"foreign","vendor":1e400}`
	cloud := "configured-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	expected, err := facts.ForResource(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("initial")
	calls, retries := 0, 0
	provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	provider.HTTPClient.Transport = testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.RawQuery != "" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("X-Option") != "owned" {
			t.Fatal("unexpected association mutation", req.Method, req.URL, req.Header)
		}
		collection := endpoint + "metadefs/namespaces/OS::Nova/resource_types"
		status, body := http.StatusAccepted, "\xffopaque acknowledgement"
		switch req.Method {
		case http.MethodPost:
			payload, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var decoded map[string]json.RawMessage
			if err := json.Unmarshal(payload, &decoded); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
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
		return testcloud.Response(status, body, http.Header{"X-Proof": {"actual"}}), nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cloud = "caller mutation"
	upper, err := conn.Image(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	versioned, err := conn.ImageV2(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if upper.API.MetadefResourceTypes.RawClient() != versioned.RawClient() || versioned.RawClient().ProviderClient != provider || calls != 0 {
		t.Fatal("binding lost shared provider or made HTTP")
	}
	for index, api := range []*types.API{upper.API.MetadefResourceTypes, versioned.MetadefResourceTypes} {
		t.Run([]string{"Image", "ImageV2"}[index], func(t *testing.T) {
			scope, err := api.InNamespace(context.Background(), "OS::Nova")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			created, err := scope.CreateRecord(context.Background(), types.WithRecordCreateAttributes(map[string]any{
				"id": "input-id", "name": "OS::Nova::Server", "created_at": false,
				"updated_at": json.Number("9007199254740993"), "prefix": "hw_", "properties_target": nil,
			}), types.WithRecordCreateHeader("X-Option", "owned"), func(*types.RecordCreateOpts) error {
				provider.SetToken("live")
				return nil
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
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
	if calls != 4 {
		t.Fatalf("calls = %v, want %v", calls, 4)
	}
	if retries != 0 {
		t.Fatalf("retries = %v, want %v", retries, 0)
	}
}
