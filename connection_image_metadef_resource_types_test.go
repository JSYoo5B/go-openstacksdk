package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	rt "gophercloudsdk/image/v2/metadefresourcetypes"
)

func TestConnectionImageMetadefResourceTypesShareClientAndScopes(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("types-0")
	const parent, child = "OS::공유 Namespace", "OS::Nova::Server"
	const prefix = "https://cloud.test/reverse/glance/v2/"
	global := prefix + "metadefs/resource_types"
	scoped := prefix + "metadefs/namespaces/" + url.PathEscape(parent) + "/resource_types"
	const row = `{"name":"passive-name","created_at":"unparsed.12345678901","updated_at":null,"protected":false,"precision":9007199254740993}`
	methods := []string{"GET", "GET", "POST", "GET", "GET", "DELETE", "DELETE", "DELETE", "POST"}
	codes := []int{200, 200, 201, 200, 200, 204, 404, 404, 201}
	bodies := []string{`{"resource_types":[` + row + `,null],"next":"https://foreign.test/unused"}`, `{"resource_types":[` + row + `]}`, `{"name":"passive-name","prefix":"","properties_target":"image\n","created_at":"literal-date"}`, `{"resource_type_associations":[{"name":"one"},null],"next":"https://foreign.test/unused"}`, `{"resource_type_associations":[{"name":"repeat"},{"name":"repeat"}]}`, "opaque removed association", "hidden or missing association", "strict missing", `{}`}
	var client *gophercloud.ServiceClient
	calls, retries := 0, 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		index := calls
		if index >= len(methods) {
			t.Fatalf("unexpected request %s", r.URL)
		}
		want := scoped
		if index < 2 {
			want = global
		}
		if index >= 5 && index <= 7 {
			want += "/" + url.PathEscape(child)
		}
		if r.Method != methods[index] || r.URL.String() != want || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != fmt.Sprintf("types-%d", index) || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", index) {
			t.Fatalf("request%d %s %s headers=%v want=%s", index, r.Method, r.URL, r.Header, want)
		}
		if index == 2 || index == 8 {
			raw, err := io.ReadAll(r.Body)
			var body map[string]json.RawMessage
			if err != nil || json.Unmarshal(raw, &body) != nil || string(body["name"]) != `"OS::Nova::Server"` {
				t.Fatalf("body%d %s %v", index, raw, err)
			}
			if index == 2 && (len(body) != 3 || string(body["prefix"]) != `""` || string(body["properties_target"]) != `"image\n"`) {
				t.Fatalf("explicit empty/literal body %s", raw)
			}
			if index == 8 && len(body) != 1 {
				t.Fatalf("nil optional omission %s", raw)
			}
		} else if r.Body != nil {
			t.Fatalf("bodyless%d acquired body", index)
		}
		calls++
		provider.SetToken(fmt.Sprintf("types-%d", calls))
		client.MoreHeaders = map[string]string{"X-Source": fmt.Sprintf("source-%d", calls)}
		return &http.Response{StatusCode: codes[index], Header: http.Header{"X-Proof": {fmt.Sprintf("types-%d", index)}}, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", prefix))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	upper, err := conn.Image(ctx)
	if err != nil {
		t.Fatal(err)
	}
	version, err := conn.ImageV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client = version.RawClient()
	client.MoreHeaders = map[string]string{"X-Source": "source-0"}
	client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, cause error, _ uint) error {
		retries++
		return cause
	}
	if upper.API.MetadefResourceTypes.RawClient() != client || version.MetadefResourceTypes.RawClient() != client || version.MetadefTags.RawClient() != client || version.MetadefProperties.RawClient() != client || version.MetadefObjects.RawClient() != client || version.MetadefNamespaces.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("aggregate replaced shared client")
	}
	scope, err := upper.API.MetadefResourceTypes.InNamespace(ctx, parent)
	if err != nil || scope.RawClient() != client || scope.NamespaceName() != parent || calls != 0 {
		t.Fatalf("scope=%+v err=%v calls=%d", scope, err, calls)
	}
	other, err := version.MetadefResourceTypes.InNamespace(ctx, parent)
	if err != nil || other.RawClient() != client || calls != 0 {
		t.Fatalf("other=%+v err=%v", other, err)
	}
	count := 0
	for value, err := range upper.API.MetadefResourceTypes.List(ctx, rt.WithListMaxItems(1)) {
		if err != nil || value == nil || value.Name == nil || *value.Name != "passive-name" || value.CreatedAt == nil || *value.CreatedAt != "unparsed.12345678901" || value.UpdatedAt != nil || string(value.Body["protected"]) != "false" || string(value.Body["precision"]) != "9007199254740993" || value.StatusCode != 200 {
			t.Fatalf("global=%+v err=%v", value, err)
		}
		count++
	}
	if count != 1 || calls != 1 {
		t.Fatal("global localcap", count, calls)
	}
	all, err := version.MetadefResourceTypes.All(ctx)
	if err != nil || len(all) != 1 || all[0].Header.Get("X-Proof") != "types-1" || calls != 2 {
		t.Fatalf("global all=%+v err=%v", all, err)
	}
	created, err := scope.Create(ctx, child, rt.WithCreatePrefix(""), rt.WithCreatePropertiesTarget("image\n"))
	if err != nil || created == nil || created.Name == nil || *created.Name != "passive-name" || created.Prefix == nil || *created.Prefix != "" || created.PropertiesTarget == nil || *created.PropertiesTarget != "image\n" || created.StatusCode != 201 {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	count = 0
	for value, err := range other.List(ctx, rt.WithListMaxItems(1)) {
		if err != nil || value == nil || value.Name == nil || *value.Name != "one" {
			t.Fatalf("association=%+v err=%v", value, err)
		}
		count++
	}
	if count != 1 || calls != 4 {
		t.Fatal("association localcap", count, calls)
	}
	associated, err := scope.All(ctx)
	if err != nil || len(associated) != 2 || *associated[0].Name != "repeat" || *associated[1].Name != "repeat" || calls != 5 {
		t.Fatalf("associations=%+v err=%v", associated, err)
	}
	*associated[0].Name = "caller"
	associated[0].Body["name"][0] = 'x'
	associated[0].Header.Set("X-Proof", "caller")
	if *associated[1].Name != "repeat" || string(associated[1].Body["name"]) != `"repeat"` || associated[1].Header.Get("X-Proof") != "types-4" {
		t.Fatal("association rows alias")
	}
	ack, err := scope.Delete(ctx, child)
	if err != nil || ack == nil || ack.Namespace != parent || ack.Name == nil || *ack.Name != child || ack.StatusCode != 204 || string(ack.Body) != "opaque removed association" {
		t.Fatalf("deleted=%+v err=%v", ack, err)
	}
	ack, err = other.Delete(ctx, child)
	if ack != nil || err != nil || retries != 0 {
		t.Fatalf("ignored=%+v err=%v retries=%d", ack, err, retries)
	}
	ack, err = scope.Delete(ctx, child, rt.WithDeleteIgnoreMissing(false))
	var status gophercloud.ErrUnexpectedResponseCode
	if ack != nil || !errors.As(err, &status) || status.Actual != 404 || retries != 1 {
		t.Fatalf("strict=%+v err=%v retries=%d", ack, err, retries)
	}
	created, err = scope.Create(ctx, child)
	if err != nil || created == nil || created.Name != nil || created.Prefix != nil || created.PropertiesTarget != nil || calls != 9 {
		t.Fatalf("unseeded=%+v err=%v", created, err)
	}
	callbacks := 0
	client.Endpoint = "https://changed.test/v2/"
	created, err = scope.Create(ctx, child, func(*rt.CreateOpts) error { callbacks++; return nil })
	if created != nil || err == nil || callbacks != 0 || calls != 9 {
		t.Fatalf("retarget=%+v err=%v callbacks=%d calls=%d", created, err, callbacks, calls)
	}
}
