package openstack_test

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

	sdk "github.com/JSYoo5B/go-openstacksdk"
	tag "github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageMetadefTagsShareClientAndPaging(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("tag-0")
	const parent, child = "OS::공유 Namespace", "OS::CPU Tag"
	const prefix = "https://cloud.test/reverse/glance/v2/"
	collection := prefix + "metadefs/namespaces/" + url.PathEscape(parent) + "/tags"
	const row = `{"name":"passive-name","created_at":"unparsed.12345678901","updated_at":null,"links":42,"precision":9007199254740993}`
	methods := []string{"POST", "GET", "PUT", "POST", "GET", "GET", "GET", "GET", "DELETE", "DELETE", "DELETE", "DELETE", "POST", "POST"}
	codes := []int{201, 200, 200, 201, 200, 200, 200, 200, 204, 204, 404, 404, 201, 201}
	bodies := []string{row, row, row, `{"tags":[{"name":"one"},{"name":"two"}]}`, `{"tags":[` + row + `,null],"next":"https://foreign.test/unused"}`, `{"tags":[{"name":"raw-page1"}]}`, `{"tags":[{"name":"raw-page2"}]}`, `{"tags":[]}`, "opaque child", "opaque collection", "strict missing", "strict bulk missing", `{"tags":[]}`, `{"tags":[{"name":"appended"}]}`}
	var client *gophercloud.ServiceClient
	calls, retries := 0, 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		index := calls
		if index >= len(methods) {
			t.Fatalf("unexpected request %s", r.URL)
		}
		want := collection
		if index == 0 || index == 1 || index == 2 || index == 8 {
			want += "/" + url.PathEscape(child)
		}
		if index == 10 {
			want += "/missing"
		}
		if index == 4 {
			want += "?limit=2"
		}
		if index >= 5 && index <= 7 {
			q := url.Values{"limit": {"1"}, "sort_dir": {"asc"}, "sort_key": {"name"}}
			if index == 6 {
				q.Set("marker", "raw-page1")
			}
			if index == 7 {
				q.Set("marker", "raw-page2")
			}
			want += "?" + q.Encode()
		}
		if r.Method != methods[index] || r.URL.String() != want || r.Header.Get("X-Auth-Token") != fmt.Sprintf("tag-%d", index) || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", index) {
			t.Fatalf("request%d %s %s headers=%v want=%s", index, r.Method, r.URL, r.Header, want)
		}
		if index == 2 || index == 3 || index == 12 || index == 13 {
			raw, err := io.ReadAll(r.Body)
			var body map[string]json.RawMessage
			if err != nil || json.Unmarshal(raw, &body) != nil {
				t.Fatalf("body %s %v", raw, err)
			}
			expected := []string{`{"name":"renamed"}`, `{"tags":[{"name":"one"},{"name":"two"}]}`, `{"tags":[]}`, `{"tags":[{"name":"appended"}]}`}
			slot := map[int]int{2: 0, 3: 1, 12: 2, 13: 3}[index]
			if string(raw) != expected[slot] {
				t.Fatalf("body%d %s", index, raw)
			}
			if index != 2 {
				appendHeader := "False"
				if index == 13 {
					appendHeader = "True"
				}
				if r.Header.Get("X-OpenStack-Append") != appendHeader {
					t.Fatalf("append%d %v", index, r.Header)
				}
			}
		} else if r.Body != nil {
			t.Fatalf("bodyless%d acquired body", index)
		}
		if index != 3 && index != 12 && index != 13 && r.Header.Get("X-OpenStack-Append") != "" {
			t.Fatal("append header leaked", index)
		}
		calls++
		provider.SetToken(fmt.Sprintf("tag-%d", calls))
		client.MoreHeaders = map[string]string{"X-Source": fmt.Sprintf("source-%d", calls)}
		return &http.Response{StatusCode: codes[index], Header: http.Header{"X-Proof": {fmt.Sprintf("tag-%d", index)}}, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
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
	if upper.API.MetadefTags.RawClient() != client || version.MetadefTags.RawClient() != client || version.MetadefProperties.RawClient() != client || version.MetadefObjects.RawClient() != client || version.MetadefNamespaces.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("aggregate replaced shared client")
	}
	scope, err := upper.API.MetadefTags.InNamespace(ctx, parent)
	if err != nil || scope.RawClient() != client || scope.NamespaceName() != parent || calls != 0 {
		t.Fatalf("scope=%+v err=%v calls=%d", scope, err, calls)
	}
	other, err := version.MetadefTags.InNamespace(ctx, parent)
	if err != nil || other.RawClient() != client || calls != 0 {
		t.Fatalf("other=%+v err=%v", other, err)
	}
	check := func(value *tag.Tag, err error, code int) {
		t.Helper()
		if err != nil || value == nil || value.StatusCode != code || value.Name == nil || *value.Name != "passive-name" || value.CreatedAt == nil || *value.CreatedAt != "unparsed.12345678901" || value.UpdatedAt != nil || value.Links != nil || string(value.Body["precision"]) != "9007199254740993" {
			t.Fatalf("tag=%+v err=%v", value, err)
		}
	}
	created, err := scope.Create(ctx, child)
	check(created, err, 201)
	fetched, err := other.Get(ctx, child)
	check(fetched, err, 200)
	created.Header.Set("X-Proof", "caller")
	created.Body["precision"][0] = '0'
	if fetched.Header.Get("X-Proof") != "tag-1" || string(fetched.Body["precision"]) != "9007199254740993" {
		t.Fatal("response ownership aliases")
	}
	updated, err := scope.Update(ctx, child, tag.WithUpdateName("renamed"))
	check(updated, err, 200)
	names := []string{"one", "two"}
	set, err := scope.Set(ctx, names, func(*tag.SetOpts) error { names[0] = "mutated"; return nil })
	if err != nil || set == nil || set.StatusCode != 201 || len(set.Tags) != 2 || *set.Tags[0].Name != "one" || set.Header.Get("X-Proof") != "tag-3" {
		t.Fatalf("set=%+v err=%v", set, err)
	}
	*set.Tags[0].Name = "caller"
	set.Tags[0].Header.Set("X-Proof", "caller")
	if !strings.Contains(string(set.Body["tags"]), `"name":"one"`) || set.Tags[1].Header.Get("X-Proof") != "tag-3" {
		t.Fatal("set root/rows alias")
	}
	rows := 0
	for value, err := range scope.List(ctx, tag.WithListLimit(2), tag.WithListMaxItems(1)) {
		check(value, err, 200)
		rows++
	}
	if rows != 1 || calls != 5 {
		t.Fatal("local cap", rows, calls)
	}
	all, err := other.All(ctx, tag.WithListLimit(1), tag.WithListSortKey("name"), tag.WithListSortDir("asc"))
	if err != nil || len(all) != 2 || *all[0].Name != "raw-page1" || *all[1].Name != "raw-page2" || all[0].Header.Get("X-Proof") != "tag-5" || all[1].Header.Get("X-Proof") != "tag-6" || calls != 8 {
		t.Fatalf("all=%+v err=%v calls=%d", all, err, calls)
	}
	deleted, err := scope.Delete(ctx, child)
	if err != nil || deleted == nil || deleted.Name == nil || *deleted.Name != child || deleted.Namespace != parent || deleted.StatusCode != 204 || string(deleted.Body) != "opaque child" {
		t.Fatalf("delete=%+v err=%v", deleted, err)
	}
	bulk, err := other.DeleteAll(ctx)
	if err != nil || bulk == nil || bulk.Name != nil || bulk.Namespace != parent || string(bulk.Body) != "opaque collection" {
		t.Fatalf("bulk=%+v err=%v", bulk, err)
	}
	missing, err := scope.Delete(ctx, "missing")
	var status gophercloud.ErrUnexpectedResponseCode
	if missing != nil || !errors.As(err, &status) || status.Actual != 404 || retries != 1 {
		t.Fatalf("missing=%+v err=%v retries=%d", missing, err, retries)
	}
	missing, err = scope.DeleteAll(ctx)
	if missing != nil || !errors.As(err, &status) || status.Actual != 404 || retries != 2 {
		t.Fatalf("bulk missing=%+v err=%v retries=%d", missing, err, retries)
	}
	empty, err := scope.Set(ctx, nil)
	if err != nil || empty == nil || empty.Tags == nil || len(empty.Tags) != 0 || empty.StatusCode != 201 {
		t.Fatalf("empty=%+v err=%v", empty, err)
	}
	appended, err := scope.Set(ctx, []string{"appended"}, tag.WithSetAppend(true))
	if err != nil || appended == nil || len(appended.Tags) != 1 || *appended.Tags[0].Name != "appended" || calls != 14 {
		t.Fatalf("append=%+v err=%v calls=%d", appended, err, calls)
	}
	callbacks := 0
	client.Endpoint = "https://changed.test/v2/"
	value, err := scope.Get(ctx, child, func(*tag.GetOpts) error { callbacks++; return nil })
	if value != nil || err == nil || callbacks != 0 || calls != 14 {
		t.Fatalf("retarget value=%+v err=%v callbacks=%d calls=%d", value, err, callbacks, calls)
	}
}
