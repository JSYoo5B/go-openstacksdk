package openstack_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageSchemasShareNativeClientAndKeepDiscoveryPassive(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("schema-0")
	const raw = `{"name":"passive server name","properties":{"precise":{"default":9007199254740993}},"definitions":{"nullable":null},"required":[],"additionalProperties":{"limit":1.00000000000000001},"links":42,"created_at":{"opaque":true},"updated_at":false,"id":"unused","$ref":"https://foreign.test/ref"}`
	calls := 0
	paths := []string{
		"schemas/images",
		"schemas/image",
		"schemas/members",
		"schemas/member",
		"schemas/tasks",
		"schemas/task",
		"schemas/metadefs/namespace",
		"schemas/metadefs/namespaces",
		"schemas/metadefs/resource_type",
		"schemas/metadefs/resource_types",
		"schemas/metadefs/object",
		"schemas/metadefs/objects",
		"schemas/metadefs/property",
		"schemas/metadefs/properties",
		"schemas/metadefs/tag",
		"schemas/metadefs/tags",
	}
	var wireHeader http.Header
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		if calls >= len(paths) {
			t.Fatalf("unexpected schema continuation: %s", r.URL)
		}
		if r.Method != http.MethodGet || r.URL.String() != "https://cloud.test/reverse/glance/v2/"+paths[calls] || r.Body != nil || r.Header.Get("X-Source") != "shared" || r.Header.Get("X-Call") != "override" || r.Header.Get("X-Opts") != "replacement" || r.Header.Get("X-Auth-Token") != fmt.Sprintf("schema-%d", calls) {
			t.Fatalf("request %d: %s %s body=%v headers=%v", calls, r.Method, r.URL, r.Body, r.Header)
		}
		wireHeader = http.Header{"X-Proof": {fmt.Sprintf("schema-%d", calls)}, "Link": {`<https://foreign.test/next>; rel="next"`}}
		calls++
		return &http.Response{StatusCode: 200, Header: wireHeader, Body: io.NopCloser(strings.NewReader(raw))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/reverse/glance/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service, err := conn.Image(ctx)
	if err != nil {
		t.Fatal(err)
	}
	versioned, err := conn.ImageV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client := versioned.RawClient()
	client.MoreHeaders = map[string]string{"X-Source": "shared"}
	if service.RawClient() != client || service.API.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("schema discovery replaced shared native client")
	}
	methods := []func(context.Context, ...image.GetSchemaOption) (*image.Schema, error){
		service.GetImagesSchema,
		service.GetImageSchema,
		service.GetMembersSchema,
		service.GetMemberSchema,
		service.GetTasksSchema,
		service.GetTaskSchema,
		service.GetMetadefNamespaceSchema,
		service.GetMetadefNamespacesSchema,
		service.GetMetadefResourceTypeSchema,
		service.GetMetadefResourceTypesSchema,
		service.GetMetadefObjectSchema,
		service.GetMetadefObjectsSchema,
		service.GetMetadefPropertySchema,
		service.GetMetadefPropertiesSchema,
		service.GetMetadefTagSchema,
		service.GetMetadefTagsSchema,
	}
	for index, get := range methods {
		provider.SetToken(fmt.Sprintf("schema-%d", index))
		result, err := get(ctx, image.WithGetSchemaOpts(image.GetSchemaOpts{Headers: map[string]string{"X-Opts": "replacement"}}), image.WithGetSchemaHeader("X-Call", "initial"), image.WithGetSchemaHeaders(map[string]string{"x-call": "override"}))
		if err != nil || result == nil || result.Name == nil || *result.Name != "passive server name" || result.StatusCode != 200 || result.Header.Get("X-Proof") != fmt.Sprintf("schema-%d", index) || result.Required == nil || len(result.Required) != 0 || result.CreatedAt != nil || result.UpdatedAt != nil || result.Links != nil {
			t.Fatalf("schema %d: result=%+v err=%v", index, result, err)
		}
		if string(result.Properties["precise"]) != `{"default":9007199254740993}` || string(result.Definitions["nullable"]) != "null" || string(result.AdditionalProperties) != `{"limit":1.00000000000000001}` || string(result.Body["links"]) != "42" || string(result.Body["$ref"]) != `"https://foreign.test/ref"` {
			t.Fatalf("schema %d lost raw data: %+v", index, result)
		}
		result.Properties["precise"][0] = '!'
		result.AdditionalProperties[0] = '!'
		result.Definitions["nullable"][0] = '!'
		if string(result.Body["properties"]) != `{"precise":{"default":9007199254740993}}` || string(result.Body["definitions"]) != `{"nullable":null}` || string(result.Body["additionalProperties"]) != `{"limit":1.00000000000000001}` {
			t.Fatal("schema fields alias original Body")
		}
		result.Header.Set("X-Proof", "caller change")
		if wireHeader.Get("X-Proof") != fmt.Sprintf("schema-%d", index) {
			t.Fatal("schema result header aliases transport")
		}
	}
	if calls != 16 {
		t.Fatalf("calls=%d", calls)
	}
}
