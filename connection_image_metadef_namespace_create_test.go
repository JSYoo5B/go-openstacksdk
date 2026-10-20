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
	ns "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageMetadefNamespaceNestedCreateSharesClientAndOwnsInput(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("nested-0")
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const identity = "OS::Nested::공유"
	const accepted = `{"namespace":"server-other","properties":{"server":{"precision":9007199254740993}},"objects":[],"tags":[{"name":"returned"}],"resource_type_associations":[],"self":"https://foreign.test/child"}`
	expected := []string{
		`{"namespace":"OS::Nested::공유","properties":{"/key%\n":{"type":"integer","title":"","default":9007199254740993}},"objects":[{"name":"../object%","description":"","properties":{},"required":["/key%\n","/key%\n"]}],"tags":[{"name":""},{"name":"/tag%\n"},{"name":"/tag%\n"}],"resource_type_associations":[{"name":"OS::Long/Type%","prefix":"","properties_target":"first\nsecond"}]}`,
		`{"namespace":"OS::Nested::공유","properties":{},"objects":[],"tags":[],"resource_type_associations":[]}`,
		`{"namespace":"OS::Nested::공유","protected":false}`,
		`{"namespace":"OS::Renamed","protected":false}`,
		`{"namespace":"OS::Nested::공유","tags":[{"name":"failed"}]}`,
	}
	calls := 0
	decode := func(raw string) any {
		t.Helper()
		var value any
		d := json.NewDecoder(strings.NewReader(raw))
		d.UseNumber()
		if err := d.Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		if calls >= len(expected) {
			t.Fatalf("unexpected child/cleanup request: %s %s", r.Method, r.URL)
		}
		wantMethod, wantURL := http.MethodPost, endpoint+"metadefs/namespaces"
		if calls == 3 {
			wantMethod, wantURL = http.MethodPut, wantURL+"/"+url.PathEscape(identity)
		}
		if r.Method != wantMethod || r.URL.String() != wantURL || r.Header.Get("X-Auth-Token") != fmt.Sprintf("nested-%d", calls) || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", calls) {
			t.Fatalf("request%d: %s %s headers=%v", calls, r.Method, r.URL, r.Header)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil || !reflect.DeepEqual(decode(string(raw)), decode(expected[calls])) {
			t.Fatalf("body%d=%s want=%s err=%v", calls, raw, expected[calls], err)
		}
		code, body := http.StatusCreated, accepted
		if calls == 3 {
			code = http.StatusOK
		}
		if calls == 4 {
			code, body = http.StatusBadRequest, "partial server write"
		}
		calls++
		provider.SetToken(fmt.Sprintf("nested-%d", calls))
		return &http.Response{StatusCode: code, Header: http.Header{"X-Proof": {fmt.Sprint(calls)}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint))
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
	if upper.RawClient() != client || upper.API.MetadefNamespaces.RawClient() != client || version.MetadefNamespaces.RawClient() != client || version.MetadefResourceTypes.RawClient() != client || version.MetadefProperties.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("nested Create replaced shared image client")
	}
	client.MoreHeaders = map[string]string{"X-Source": "source-0"}
	precision := json.RawMessage(`9007199254740993`)
	properties := map[string]ns.PropertyDefinition{"/key%\n": {Type: "integer", Title: "", Attributes: map[string]json.RawMessage{"default": precision}}}
	propertyOption := ns.WithCreateProperties(properties)
	precision[0] = '1'
	delete(properties, "/key%\n")
	empty, target := "", "first\nsecond"
	callbacks := 0
	value, err := version.MetadefNamespaces.Create(ctx, identity, propertyOption,
		ns.WithCreateObjects([]ns.ObjectDefinition{{Name: "../object%", Description: &empty, Properties: map[string]ns.PropertyDefinition{}, Required: []string{"/key%\n", "/key%\n"}}}),
		ns.WithCreateTags([]ns.TagDefinition{{Name: ""}, {Name: "/tag%\n"}, {Name: "/tag%\n"}}),
		ns.WithCreateResourceTypeAssociations([]ns.ResourceTypeAssociationDefinition{{Name: "OS::Long/Type%", Prefix: &empty, PropertiesTarget: &target}}),
		func(config *ns.CreateOpts) error { callbacks++; return nil })
	if err != nil || value == nil || value.Namespace == nil || *value.Namespace != "server-other" || value.StatusCode != 201 || callbacks != 1 || string(value.Body["properties"]) != `{"server":{"precision":9007199254740993}}` {
		t.Fatalf("unseeded response=%+v callbacks=%d err=%v", value, callbacks, err)
	}
	value.Body["properties"][0] = 'x'
	client.MoreHeaders["X-Source"] = "source-1"
	value, err = upper.API.MetadefNamespaces.Create(ctx, identity, propertyOption,
		ns.WithCreateOpts(ns.CreateOpts{Properties: map[string]ns.PropertyDefinition{}, Objects: []ns.ObjectDefinition{}, Tags: []ns.TagDefinition{}, ResourceTypeAssociations: []ns.ResourceTypeAssociationDefinition{}}))
	if err != nil || value == nil || string(value.Body["properties"]) != `{"server":{"precision":9007199254740993}}` {
		t.Fatalf("independent response=%+v err=%v", value, err)
	}
	client.MoreHeaders["X-Source"] = "source-2"
	value, err = version.MetadefNamespaces.Create(ctx, identity, propertyOption, ns.WithCreateProperties(nil), ns.WithCreateProtected(false))
	if err != nil || value == nil {
		t.Fatal(err)
	}
	client.MoreHeaders["X-Source"] = "source-3"
	value, err = version.MetadefNamespaces.Update(ctx, identity, ns.WithUpdateNamespace("OS::Renamed"), ns.WithUpdateProtected(false))
	if err != nil || value == nil || value.StatusCode != 200 {
		t.Fatal(err)
	}
	client.MoreHeaders["X-Source"] = "source-4"
	value, err = version.MetadefNamespaces.Create(ctx, identity, ns.WithCreateTags([]ns.TagDefinition{{Name: "failed"}}))
	var native gophercloud.ErrUnexpectedResponseCode
	if value != nil || !errors.As(err, &native) || native.Actual != 400 || string(native.Body) != "partial server write" || calls != 5 {
		t.Fatalf("server failure/compensation: value=%+v err=%v calls=%d", value, err, calls)
	}
	value, err = version.MetadefNamespaces.Create(ctx, identity, ns.WithCreateProperties(map[string]ns.PropertyDefinition{string([]byte{0xff}): {Type: "string", Title: ""}}))
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 5 {
		t.Fatalf("invalid original key reached HTTP: value=%+v err=%v calls=%d", value, err, calls)
	}
}
