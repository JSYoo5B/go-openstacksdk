package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	ns "gophercloudsdk/image/v2/metadefnamespaces"
)

func TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("namespace-0")
	const base = "https://cloud.test/reverse/glance/v2/metadefs/namespaces"
	const identity = "OS::SDK::공유 Demo"
	const row = `{"namespace":"passive-other","display_name":"","protected":false,"visibility":"future","self":"https://foreign.test/other","schema":"foreign","created_at":"unparsed.12345678901","updated_at":null,"properties":42,"objects":null,"tags":[null],"resource_type_associations":false,"links":42,"precision":9007199254740993}`
	methods := []string{"POST", "GET", "PUT", "GET", "GET", "GET", "DELETE", "DELETE"}
	codes := []int{201, 200, 200, 200, 200, 200, 204, 404}
	listQuery := url.Values{"limit": {"1"}, "visibility": {"public"}, "resource_types": {"OS::Glance::Image,OS::Nova::Flavor"}, "sort_key": {"namespace"}, "sort_dir": {"asc"}}
	nextQuery := url.Values{}
	for key, values := range listQuery {
		nextQuery[key] = append([]string(nil), values...)
	}
	nextQuery.Set("marker", "OS::Page::Two")
	bodies := []string{row, row, row, `{"namespaces":[` + row + `,null],"next":"https://foreign.test/unused"}`, `{"namespaces":[` + row + `],"next":"/v2/metadefs/namespaces?` + nextQuery.Encode() + `"}`, `{"namespaces":[{"namespace":"","protected":null,"created_at":null}],"next":null}`, "opaque acknowledgement", "hidden or missing"}
	calls, retries := 0, 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		if calls >= len(methods) {
			t.Fatalf("unexpected request %s", r.URL)
		}
		index := calls
		wantURL := base
		if index == 1 || index == 2 || index == 6 {
			wantURL += "/" + url.PathEscape(identity)
		}
		if index == 1 {
			wantURL += "?resource_type=" + url.QueryEscape("OS::Glance::Image")
		}
		if index == 3 || index == 4 {
			wantURL += "?" + listQuery.Encode()
		}
		if index == 5 {
			wantURL += "?" + nextQuery.Encode()
		}
		if index == 7 {
			wantURL += "/missing"
		}
		if r.Method != methods[index] || r.URL.String() != wantURL || r.Header.Get("X-Auth-Token") != fmt.Sprintf("namespace-%d", index) || r.Header.Get("X-Source") != "shared" || r.Header.Get("X-Call") != "namespaces" {
			t.Fatalf("request%d %s %s headers=%v want=%s", index, r.Method, r.URL, r.Header, wantURL)
		}
		if index == 0 || index == 2 {
			raw, err := io.ReadAll(r.Body)
			var body map[string]any
			if err != nil || json.Unmarshal(raw, &body) != nil {
				t.Fatalf("invalid body %s: %v", raw, err)
			}
			want := map[string]any{"namespace": identity, "protected": false, "description": "first\nsecond"}
			if index == 2 {
				want = map[string]any{"namespace": "OS::Renamed", "protected": false}
			}
			if !reflect.DeepEqual(body, want) {
				t.Fatalf("body%d %s want=%v", index, raw, want)
			}
		} else if r.Body != nil {
			t.Fatalf("bodyless operation%d acquired body", index)
		}
		calls++
		provider.SetToken(fmt.Sprintf("namespace-%d", calls))
		return &http.Response{StatusCode: codes[index], Header: http.Header{"X-Proof": {fmt.Sprintf("namespace-%d", index)}, "Link": {`<https://foreign.test/next>; rel="next"`}}, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/reverse/glance/v2/"))
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
	client.MoreHeaders = map[string]string{"X-Source": "shared"}
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return fmt.Errorf("unexpected404 retry")
	}
	if upper.RawClient() != client || upper.API.RawClient() != client || upper.API.MetadefNamespaces.RawClient() != client || version.MetadefNamespaces.RawClient() != client || version.Images.RawClient() != client || version.Members.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("namespace aggregate replaced shared/native client")
	}
	again, err := conn.ImageV2(ctx)
	if err != nil || again != version {
		t.Fatalf("cache=%p/%p err=%v", again, version, err)
	}
	check := func(value *ns.Namespace, err error, code int) {
		t.Helper()
		if err != nil || value == nil || value.StatusCode != code || value.Namespace == nil || *value.Namespace != "passive-other" || value.DisplayName == nil || *value.DisplayName != "" || value.IsProtected == nil || *value.IsProtected || value.Visibility == nil || *value.Visibility != "future" || value.CreatedAt == nil || *value.CreatedAt != "unparsed.12345678901" || value.UpdatedAt != nil || value.Links != nil || string(value.Body["properties"]) != "42" || string(value.Body["precision"]) != "9007199254740993" {
			t.Fatalf("namespace=%+v err=%v", value, err)
		}
	}
	created, err := upper.API.MetadefNamespaces.Create(ctx, identity, ns.WithCreateHeaders(map[string]string{"X-Call": "namespaces"}), ns.WithCreateProtected(false), ns.WithCreateDescription("first\nsecond"))
	check(created, err, 201)
	fetched, err := version.MetadefNamespaces.Get(ctx, identity, ns.WithGetHeader("X-Call", "namespaces"), ns.WithGetResourceType("OS::Glance::Image"))
	check(fetched, err, 200)
	created.Header.Set("X-Proof", "caller")
	created.Body["namespace"][1] = '!'
	if fetched.Header.Get("X-Proof") != "namespace-1" || *created.Namespace != "passive-other" || string(fetched.Body["namespace"]) != `"passive-other"` {
		t.Fatal("separate responses alias raw/typed evidence")
	}
	updated, err := version.MetadefNamespaces.Update(ctx, identity, ns.WithUpdateHeader("X-Call", "namespaces"), ns.WithUpdateNamespace("OS::Renamed"), ns.WithUpdateProtected(false))
	check(updated, err, 200)
	listOpts := []ns.ListOption{ns.WithListHeader("X-Call", "namespaces"), ns.WithListLimit(1), ns.WithListVisibility("public"), ns.WithListResourceTypes("OS::Glance::Image,OS::Nova::Flavor"), ns.WithListSortKey("namespace"), ns.WithListSortDir("asc")}
	capOpts := append(append([]ns.ListOption(nil), listOpts...), ns.WithListMaxItems(1))
	sequence := version.MetadefNamespaces.List(ctx, capOpts...)
	if calls != 3 {
		t.Fatal("eager iterator")
	}
	rows := 0
	for value, err := range sequence {
		check(value, err, 200)
		rows++
	}
	if calls != 4 || rows != 1 {
		t.Fatalf("cap rows=%d calls=%d", rows, calls)
	}
	all, err := upper.API.MetadefNamespaces.All(ctx, listOpts...)
	if err != nil || len(all) != 2 || calls != 6 || all[1].Namespace == nil || *all[1].Namespace != "" || all[1].IsProtected != nil || all[1].CreatedAt != nil {
		t.Fatalf("all=%+v err=%v calls=%d", all, err, calls)
	}
	all[0].Header.Set("X-Proof", "caller row")
	if all[1].Header.Get("X-Proof") != "namespace-5" {
		t.Fatal("page headers alias")
	}
	deleted, err := version.MetadefNamespaces.Delete(ctx, identity, ns.WithDeleteHeader("X-Call", "namespaces"), ns.WithDeleteIgnoreMissing(false))
	if err != nil || deleted == nil || deleted.Namespace != identity || deleted.StatusCode != 204 || string(deleted.Body) != "opaque acknowledgement" {
		t.Fatalf("delete=%+v err=%v", deleted, err)
	}
	absent, err := upper.API.MetadefNamespaces.Delete(ctx, "missing", ns.WithDeleteOpts(ns.DeleteOpts{Headers: map[string]string{"X-Call": "namespaces"}}))
	if err != nil || absent != nil || calls != 8 || retries != 0 {
		t.Fatalf("defaultdelete=%+v err=%v calls=%d retries=%d", absent, err, calls, retries)
	}
}
