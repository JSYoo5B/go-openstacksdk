package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	obj "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageMetadefObjectsShareClientAndFixedScope(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("object-0")
	const parent = "OS::공유 Namespace"
	const child = "OS::CPU Object"
	const prefix = "https://cloud.test/reverse/glance/v2/"
	collection := prefix + "metadefs/namespaces/" + url.PathEscape(parent) + "/objects"
	const row = `{"name":"passive-child","description":"","properties":{"future":42,"nested":{"precision":9007199254740993}},"required":["literal,comma",""],"self":"https://foreign.test/child","schema":"foreign","created_at":"unparsed.12345678901","updated_at":null,"links":42,"precision":9007199254740993}`
	methods := []string{"POST", "GET", "PUT", "GET", "GET", "DELETE", "DELETE", "DELETE", "DELETE"}
	codes := []int{201, 200, 200, 200, 200, 204, 204, 404, 404}
	bodies := []string{row, row, row, `{"objects":[` + row + `,null],"next":[false]}`, `{"objects":[` + row + `,{"name":"","properties":{},"required":[]}],"next":{"url":"https://foreign.test/unused"}}`, "opaque child acknowledgement", "opaque collection acknowledgement", "hidden or missing", "strict collection missing"}
	calls, retries := 0, 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		if calls >= len(methods) {
			t.Fatalf("unexpected request %s", r.URL)
		}
		index := calls
		wantURL := collection
		if index == 1 || index == 2 || index == 5 {
			wantURL += "/" + url.PathEscape(child)
		}
		if index == 7 {
			wantURL += "/missing"
		}
		if r.Method != methods[index] || r.URL.String() != wantURL || r.Header.Get("X-Auth-Token") != fmt.Sprintf("object-%d", index) || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", index) || r.Header.Get("X-Call") != "objects" {
			t.Fatalf("request%d %s %s headers=%v want=%s", index, r.Method, r.URL, r.Header, wantURL)
		}
		if index == 0 || index == 2 {
			raw, err := io.ReadAll(r.Body)
			var body map[string]json.RawMessage
			if err != nil || json.Unmarshal(raw, &body) != nil {
				t.Fatalf("invalid body %s: %v", raw, err)
			}
			if index == 0 {
				var required []string
				if len(body) != 4 || string(body["name"]) != `"OS::CPU Object"` || string(body["description"]) != `"first\nsecond"` || !strings.Contains(string(body["properties"]), "9007199254740993") || json.Unmarshal(body["required"], &required) != nil || !reflect.DeepEqual(required, []string{"CPU", "literal,comma"}) {
					t.Fatalf("create body %s", raw)
				}
			} else if len(body) != 2 || string(body["name"]) != `"renamed"` || string(body["properties"]) != "{}" {
				t.Fatalf("replacement body %s", raw)
			}
		} else if r.Body != nil {
			t.Fatalf("bodyless operation%d acquired body", index)
		}
		calls++
		provider.SetToken(fmt.Sprintf("object-%d", calls))
		return &http.Response{StatusCode: codes[index], Header: http.Header{"X-Proof": {fmt.Sprintf("object-%d", index)}, "Link": {`<https://foreign.test/next>; rel="next"`}}, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
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
		return fmt.Errorf("unexpected404 retry")
	}
	if upper.RawClient() != client || upper.API.RawClient() != client || upper.API.MetadefObjects.RawClient() != client || version.MetadefObjects.RawClient() != client || version.MetadefNamespaces.RawClient() != client || version.Images.RawClient() != client || version.Members.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("object aggregate replaced shared/native client")
	}
	again, err := conn.ImageV2(ctx)
	if err != nil || again != version {
		t.Fatalf("cache=%p/%p err=%v", again, version, err)
	}
	scope, err := upper.API.MetadefObjects.InNamespace(ctx, parent)
	if err != nil || scope.RawClient() != client || scope.NamespaceName() != parent || calls != 0 {
		t.Fatalf("scope=%+v err=%v calls=%d", scope, err, calls)
	}
	other, err := version.MetadefObjects.InNamespace(ctx, parent)
	if err != nil || other.RawClient() != client || calls != 0 {
		t.Fatalf("other scope=%+v err=%v", other, err)
	}
	advanceHeader := func() { client.MoreHeaders = map[string]string{"X-Source": fmt.Sprintf("source-%d", calls)} }
	check := func(value *obj.Object, err error, code int) {
		t.Helper()
		if err != nil || value == nil || value.StatusCode != code || value.Name == nil || *value.Name != "passive-child" || value.Description == nil || *value.Description != "" || string(value.Properties["future"]) != "42" || string(value.Properties["nested"]) != `{"precision":9007199254740993}` || !reflect.DeepEqual(value.Required, []string{"literal,comma", ""}) || value.CreatedAt == nil || *value.CreatedAt != "unparsed.12345678901" || value.UpdatedAt != nil || value.Links != nil || string(value.Body["precision"]) != "9007199254740993" {
			t.Fatalf("object=%+v err=%v", value, err)
		}
	}
	created, err := scope.Create(ctx, child, obj.WithCreateHeader("X-Call", "objects"), obj.WithCreateDescription("first\nsecond"), obj.WithCreateProperties(map[string]json.RawMessage{"CPU": json.RawMessage(`{"type":"integer","precision":9007199254740993}`)}), obj.WithCreateRequired([]string{"CPU", "literal,comma"}))
	check(created, err, 201)
	advanceHeader()
	fetched, err := other.Get(ctx, child, obj.WithGetHeader("X-Call", "objects"))
	check(fetched, err, 200)
	created.Header.Set("X-Proof", "caller")
	created.Properties["future"][0] = '0'
	created.Required[0] = "caller"
	if fetched.Header.Get("X-Proof") != "object-1" || string(fetched.Properties["future"]) != "42" || fetched.Required[0] != "literal,comma" || !strings.Contains(string(created.Body["properties"]), `"future":42`) || string(created.Body["required"]) != `["literal,comma",""]` {
		t.Fatal("field/raw/separate response ownership aliases")
	}
	advanceHeader()
	updated, err := scope.Update(ctx, child, obj.WithUpdateHeader("X-Call", "objects"), obj.WithUpdateName("renamed"), obj.WithUpdateProperties(map[string]json.RawMessage{}))
	check(updated, err, 200)
	advanceHeader()
	sequence := scope.List(ctx, obj.WithListHeader("X-Call", "objects"), obj.WithListMaxItems(1))
	if calls != 3 {
		t.Fatal("eager iterator")
	}
	rows := 0
	for value, err := range sequence {
		check(value, err, 200)
		rows++
	}
	if rows != 1 || calls != 4 {
		t.Fatalf("cap rows=%d calls=%d", rows, calls)
	}
	advanceHeader()
	all, err := other.All(ctx, obj.WithListHeader("X-Call", "objects"))
	if err != nil || len(all) != 2 || calls != 5 || all[1].Name == nil || *all[1].Name != "" || all[1].Properties == nil || len(all[1].Properties) != 0 || all[1].Required == nil || len(all[1].Required) != 0 {
		t.Fatalf("all=%+v err=%v calls=%d", all, err, calls)
	}
	all[0].Header.Set("X-Proof", "caller row")
	if all[1].Header.Get("X-Proof") != "object-4" {
		t.Fatal("finite page row headers alias")
	}
	advanceHeader()
	deleted, err := scope.Delete(ctx, child, obj.WithDeleteHeader("X-Call", "objects"), obj.WithDeleteIgnoreMissing(false))
	if err != nil || deleted == nil || deleted.Namespace != parent || deleted.Name == nil || *deleted.Name != child || deleted.StatusCode != 204 || string(deleted.Body) != "opaque child acknowledgement" {
		t.Fatalf("delete=%+v err=%v", deleted, err)
	}
	advanceHeader()
	deletedAll, err := other.DeleteAll(ctx, obj.WithDeleteAllHeader("X-Call", "objects"))
	if err != nil || deletedAll == nil || deletedAll.Namespace != parent || deletedAll.Name != nil || deletedAll.StatusCode != 204 || string(deletedAll.Body) != "opaque collection acknowledgement" {
		t.Fatalf("deleteall=%+v err=%v", deletedAll, err)
	}
	advanceHeader()
	absent, err := scope.Delete(ctx, "missing", obj.WithDeleteOpts(obj.DeleteOpts{Headers: map[string]string{"X-Call": "objects"}}))
	if err != nil || absent != nil || calls != 8 || retries != 0 {
		t.Fatalf("defaultdelete=%+v err=%v calls=%d retries=%d", absent, err, calls, retries)
	}
	advanceHeader()
	client.RetryFunc = nil
	strict, err := scope.DeleteAll(ctx, obj.WithDeleteAllHeader("X-Call", "objects"))
	var status gophercloud.ErrUnexpectedResponseCode
	if strict != nil || err == nil || !errors.As(err, &status) || status.Actual != 404 || calls != 9 {
		t.Fatalf("strictdeleteall=%+v err=%v calls=%d", strict, err, calls)
	}
	callbacks := 0
	client.Endpoint = "https://changed.test/v2/"
	value, err := scope.Get(ctx, child, func(*obj.GetOpts) error { callbacks++; return nil })
	if value != nil || err == nil || callbacks != 0 || calls != 9 {
		t.Fatalf("scope retarget value=%+v err=%v callbacks=%d calls=%d", value, err, callbacks, calls)
	}
	client.Endpoint = prefix
}
