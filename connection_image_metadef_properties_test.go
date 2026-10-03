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
	prop "gophercloudsdk/image/v2/metadefproperties"
)

func TestConnectionImageMetadefPropertiesShareClientAndFixedScope(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("property-0")
	const parent, child = "OS::공유 Namespace", "OS::CPU Property"
	const prefix = "https://cloud.test/reverse/glance/v2/"
	collection := prefix + "metadefs/namespaces/" + url.PathEscape(parent) + "/properties"
	const row = `{"name":"passive-name","type":"integer","title":"","description":"","default":9007199254740993,"self":"https://foreign.test/child","schema":"foreign","created_at":"unparsed.12345678901","updated_at":null,"links":42}`
	methods := []string{"POST", "GET", "PUT", "GET", "GET", "DELETE", "DELETE", "DELETE", "DELETE"}
	codes := []int{201, 200, 200, 200, 200, 204, 204, 404, 404}
	bodies := []string{row, row, row, `{"properties":{"wire-key":` + row + `,"unused":null},"next":false}`, `{"properties":{"wire-key":` + row + `,"second":{"type":"string"}},"next":"https://foreign.test/unused"}`, "opaque child", "opaque collection", "hidden or missing", "strict missing"}
	calls, retries := 0, 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		if calls >= len(methods) {
			t.Fatalf("unexpected request %s", r.URL)
		}
		index := calls
		want := collection
		if index == 1 || index == 2 || index == 5 {
			want += "/" + url.PathEscape(child)
		}
		if index == 1 {
			want += "?resource_type=" + url.QueryEscape("OS::Nova::Server")
		}
		if index == 7 {
			want += "/missing"
		}
		if r.Method != methods[index] || r.URL.String() != want || r.Header.Get("X-Auth-Token") != fmt.Sprintf("property-%d", index) || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", index) {
			t.Fatalf("request%d %s %s headers=%v want=%s", index, r.Method, r.URL, r.Header, want)
		}
		if index == 0 || index == 2 {
			raw, err := io.ReadAll(r.Body)
			var body map[string]json.RawMessage
			if err != nil || json.Unmarshal(raw, &body) != nil || string(body["type"]) != `"integer"` || string(body["title"]) != `""` {
				t.Fatalf("invalid required body %s: %v", raw, err)
			}
			if index == 0 && (len(body) != 4 || string(body["name"]) != `"OS::CPU Property"` || string(body["default"]) != "9007199254740993") {
				t.Fatalf("create flat schema %s", raw)
			}
			if index == 2 && (len(body) != 3 || string(body["name"]) != `"renamed"`) {
				t.Fatalf("replacement %s", raw)
			}
		} else if r.Body != nil {
			t.Fatalf("bodyless request%d acquired body", index)
		}
		calls++
		provider.SetToken(fmt.Sprintf("property-%d", calls))
		return &http.Response{StatusCode: codes[index], Header: http.Header{"X-Proof": {fmt.Sprintf("property-%d", index)}}, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
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
	client := version.RawClient()
	client.MoreHeaders = map[string]string{"X-Source": "source-0"}
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return errors.New("unexpected retry")
	}
	if upper.API.MetadefProperties.RawClient() != client || version.MetadefProperties.RawClient() != client || version.MetadefObjects.RawClient() != client || version.MetadefNamespaces.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("aggregate replaced shared client")
	}
	scope, err := upper.API.MetadefProperties.InNamespace(ctx, parent)
	if err != nil || scope.RawClient() != client || scope.NamespaceName() != parent || calls != 0 {
		t.Fatalf("scope=%+v err=%v calls=%d", scope, err, calls)
	}
	other, err := version.MetadefProperties.InNamespace(ctx, parent)
	if err != nil || other.RawClient() != client || calls != 0 {
		t.Fatalf("other=%+v err=%v", other, err)
	}
	advance := func() { client.MoreHeaders = map[string]string{"X-Source": fmt.Sprintf("source-%d", calls)} }
	check := func(value *prop.Property, err error, code int) {
		t.Helper()
		if err != nil || value == nil || value.StatusCode != code || value.Name == nil || *value.Name != "passive-name" || value.Type == nil || *value.Type != "integer" || value.Title == nil || *value.Title != "" || value.CreatedAt == nil || *value.CreatedAt != "unparsed.12345678901" || value.UpdatedAt != nil || value.Links != nil || string(value.Body["default"]) != "9007199254740993" {
			t.Fatalf("property=%+v err=%v", value, err)
		}
	}
	created, err := scope.Create(ctx, child, prop.WithCreateType("integer"), prop.WithCreateTitle(""), prop.WithCreateAttribute("default", json.Number("9007199254740993")))
	check(created, err, 201)
	advance()
	fetched, err := other.Get(ctx, child, prop.WithGetResourceType("OS::Nova::Server"))
	check(fetched, err, 200)
	if created.Key != nil || fetched.Key != nil {
		t.Fatal("CRUD synthesized dictionary key")
	}
	created.Header.Set("X-Proof", "caller")
	created.Body["default"][0] = '0'
	if fetched.Header.Get("X-Proof") != "property-1" || string(fetched.Body["default"]) != "9007199254740993" {
		t.Fatal("response ownership aliases")
	}
	advance()
	updated, err := scope.Update(ctx, child, prop.WithUpdateName("renamed"), prop.WithUpdateType("integer"), prop.WithUpdateTitle(""))
	check(updated, err, 200)
	advance()
	sequence := scope.List(ctx, prop.WithListMaxItems(1))
	if calls != 3 {
		t.Fatal("eager iterator")
	}
	for value, err := range sequence {
		check(value, err, 200)
		if value.Key == nil || *value.Key != "wire-key" || *value.Name != "passive-name" {
			t.Fatal("key/name provenance")
		}
	}
	advance()
	all, err := other.All(ctx)
	if err != nil || len(all) != 2 || all[1].Key == nil || *all[1].Key != "second" || all[1].Name != nil || all[1].Body["name"] != nil || calls != 5 {
		t.Fatalf("all=%+v err=%v calls=%d", all, err, calls)
	}
	advance()
	deleted, err := scope.Delete(ctx, child, prop.WithDeleteIgnoreMissing(false))
	if err != nil || deleted == nil || deleted.Name == nil || *deleted.Name != child || deleted.Namespace != parent || deleted.StatusCode != 204 || string(deleted.Body) != "opaque child" {
		t.Fatalf("delete=%+v err=%v", deleted, err)
	}
	advance()
	bulk, err := other.DeleteAll(ctx)
	if err != nil || bulk == nil || bulk.Name != nil || bulk.Namespace != parent || string(bulk.Body) != "opaque collection" {
		t.Fatalf("bulk=%+v err=%v", bulk, err)
	}
	advance()
	missing, err := scope.Delete(ctx, "missing")
	if err != nil || missing != nil || calls != 8 || retries != 0 {
		t.Fatalf("missing=%+v err=%v retries=%d", missing, err, retries)
	}
	advance()
	client.RetryFunc = nil
	strict, err := scope.DeleteAll(ctx)
	var status gophercloud.ErrUnexpectedResponseCode
	if strict != nil || !errors.As(err, &status) || status.Actual != 404 || calls != 9 {
		t.Fatalf("strict=%+v err=%v calls=%d", strict, err, calls)
	}
	callbacks := 0
	client.Endpoint = "https://changed.test/v2/"
	value, err := scope.Get(ctx, child, func(*prop.GetOpts) error { callbacks++; return nil })
	if value != nil || err == nil || callbacks != 0 || calls != 9 {
		t.Fatalf("retarget value=%+v err=%v callbacks=%d calls=%d", value, err, callbacks, calls)
	}
}
