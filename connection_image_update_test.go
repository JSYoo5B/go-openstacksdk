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
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageUpdateFacadeOwnsPatchAndResponse(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("update-0")
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "이미지% ?#"
	const actual = `{"id":"actual-id","name":"actual-name","min_disk":0,"protected":false,"tags":[],"x-number":9007199254740995}`
	calls := 0
	var source *gophercloud.ServiceClient
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		path, sourceHeader := endpoint+"images/"+url.PathEscape(id), fmt.Sprintf("source-%d", n)
		method := http.MethodPatch
		if n == 3 {
			path = endpoint + "images"
			method = http.MethodGet
		}
		if n == 4 {
			path = endpoint + "images/resolved-id"
			sourceHeader = "source-3"
		}
		if r.Method != method || r.URL.Scheme+"://"+r.URL.Host+r.URL.EscapedPath() != path || r.Header.Get("X-Auth-Token") != fmt.Sprintf("update-%d", n) || r.Header.Get("X-Source") != sourceHeader {
			t.Fatalf("request%d: %s %s %v", n, r.Method, r.URL, r.Header)
		}
		if n == 3 {
			if r.URL.Query().Get("name") != "Ubuntu 24" || len(r.URL.Query()) != 1 || r.Body != nil || r.Header.Get("X-Call") != "named" || r.Header.Get("Content-Type") != "source/type" || r.Header.Get("Accept") != "source/accept" {
				t.Fatal("Name media/source changed", r.URL, r.Header)
			}
		} else {
			if r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || r.Header.Get("Accept") != "application/json" {
				t.Fatal("PATCH route/media lost", r.URL, r.Header)
			}
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			var got []image.ImagePatch
			if err = json.Unmarshal(raw, &got); err != nil || got == nil {
				t.Fatalf("patch=%s err=%v", raw, err)
			}
			var want []image.ImagePatch
			switch n {
			case 0:
				want = []image.ImagePatch{{Op: "add", Path: "/name", Value: json.RawMessage(`""`)}, {Op: "add", Path: "/protected", Value: json.RawMessage(`false`)}, {Op: "add", Path: "/min_disk", Value: json.RawMessage(`0`)}, {Op: "add", Path: "/tags", Value: json.RawMessage(`[]`)}, {Op: "add", Path: "/a~1b~0", Value: json.RawMessage(`{"huge":1e1000}`)}, {Op: "add", Path: "/null", Value: json.RawMessage(`null`)}, {Op: "remove", Path: "/old~0key"}}
				if r.Header.Get("X-Call") != "owned" {
					t.Fatal("update header snapshot lost")
				}
			case 1:
				want = []image.ImagePatch{{Op: "add", Path: "/bool", Value: json.RawMessage(`false`)}, {Op: "add", Path: "/null", Value: json.RawMessage(`null`)}, {Op: "add", Path: "/number", Value: json.RawMessage(`9007199254740995`)}}
				if r.Header.Get("X-Call") != "properties" {
					t.Fatal(r.Header)
				}
			case 2:
				want = []image.ImagePatch{}
			case 4:
				want = []image.ImagePatch{{Op: "add", Path: "/name", Value: json.RawMessage(`"changed"`)}}
				if r.Header.Get("X-Call") != "named" {
					t.Fatal(r.Header)
				}
			case 5, 6:
				want = []image.ImagePatch{}
			default:
				t.Fatalf("unexpected operation%d", n)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("request%d got=%s want=%+v", n, raw, want)
			}
		}
		code, body := 200, actual
		switch n {
		case 3:
			body = `{"images":[{"id":"resolved-id","name":"Ubuntu 24"}]}`
		case 4:
			body = `{"id":"passive-response","name":null,"properties":false}`
		case 5:
			body = `{"size":1.5}`
		case 6:
			code, body = 404, "image hidden or missing"
		}
		provider.SetToken(fmt.Sprintf("update-%d", calls))
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		return &http.Response{Request: r, StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {fmt.Sprint(n)}}, Body: io.NopCloser(strings.NewReader(body))}, nil
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
	native, err := conn.ImageV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	source = upper.RawClient()
	if native.RawClient() != source || upper.API.Images.RawClient() != source || source.ProviderClient != provider {
		t.Fatal("update facade replaced shared client")
	}
	source.MoreHeaders = map[string]string{"X-Source": "source-0", "Content-Type": "source/type", "Accept": "source/accept"}
	headers := map[string]string{"X-Call": "owned"}
	h := image.WithUpdateImageHeaders(headers)
	headers["X-Call"] = "changed"
	updated, err := upper.UpdateImage(ctx, resource.ID(id), h, image.WithUpdateImageName(""), image.WithUpdateImageProtected(false), image.WithUpdateImageMinDisk(0), image.WithUpdateImageTags(), image.WithUpdateImageFields(map[string]json.RawMessage{"null": json.RawMessage(`null`), "a/b~": json.RawMessage(`{"huge":1e1000}`)}), image.WithUpdateImageRemoveField("old~key"))
	if err != nil || updated == nil || *updated.ID != "actual-id" || *updated.MinDisk != 0 || *updated.Protected || updated.Tags == nil || updated.StatusCode != 200 || calls != 1 {
		t.Fatalf("image=%+v err=%v calls=%d", updated, err, calls)
	}
	updated.Properties["x-number"][0] = '1'
	if string(updated.Body["x-number"]) != "9007199254740995" {
		t.Fatal("update projection aliases Body")
	}
	properties := map[string]json.RawMessage{"number": json.RawMessage(`9007199254740995`), "null": json.RawMessage(`null`)}
	policy := image.WithSetImagePropertiesProperties(properties)
	properties["number"][0] = '1'
	set, err := upper.SetImageProperties(ctx, resource.ID(id), policy, image.WithSetImagePropertiesProperty("bool", false), image.WithSetImagePropertiesHeader("X-Call", "properties"))
	if err != nil || set == nil || set.Header.Get("X-Proof") != "1" || calls != 2 {
		t.Fatal(set, err, calls)
	}
	empty, err := upper.UpdateImage(ctx, resource.ID(id))
	if err != nil || empty == nil || empty.Header.Get("X-Proof") != "2" || calls != 3 {
		t.Fatal(empty, err, calls)
	}
	named, err := upper.UpdateImage(ctx, resource.Name("Ubuntu 24"), image.WithUpdateImageName("changed"), image.WithUpdateImageHeader("X-Call", "named"))
	if err != nil || named == nil || *named.ID != "passive-response" || named.Name != nil || string(named.Properties["properties"]) != "false" || calls != 5 {
		t.Fatal(named, err, calls)
	}
	bad, err := upper.SetImageProperties(ctx, resource.ID(id))
	var response *resource.ResponseError
	if bad != nil || !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != `{"size":1.5}` || response.Header.Get("X-Proof") != "5" || calls != 6 {
		t.Fatal(bad, err, calls)
	}
	missing, err := upper.UpdateImage(ctx, resource.ID(id))
	var failure gophercloud.ErrUnexpectedResponseCode
	if missing != nil || !errors.As(err, &failure) || failure.Actual != 404 || string(failure.Body) != "image hidden or missing" || calls != 7 {
		t.Fatal(missing, err, calls)
	}
	if source.MoreHeaders["Content-Type"] != "source/type" || source.MoreHeaders["Accept"] != "source/accept" || source.ProviderClient != provider {
		t.Fatal("SDK modified source media/provider")
	}
}
