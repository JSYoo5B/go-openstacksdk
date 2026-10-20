package openstack_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("read-0")
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "이미지% ?#"
	const actual = `{"id":"actual-id","name":"actual-name","size":9007199254740993,"protected":false,"os_hidden":null,"tags":[],"created_at":"literal","x-number":9007199254740995,"properties":[],"metadata":true}`
	calls := 0
	var source *gophercloud.ServiceClient
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		path, header := endpoint+"images/"+url.PathEscape(id), fmt.Sprintf("source-%d", n)
		if n == 1 || n == 2 || n == 3 {
			path = endpoint + "images"
		} else if n == 4 {
			path, header = endpoint+"images/resolved-id", "source-3"
		}
		if r.Method != http.MethodGet || r.URL.Scheme+"://"+r.URL.Host+r.URL.EscapedPath() != path || r.Header.Get("X-Auth-Token") != fmt.Sprintf("read-%d", n) || r.Header.Get("X-Source") != header || r.Body != nil {
			t.Fatalf("request%d: %s %s %v", n, r.Method, r.URL, r.Header)
		}
		q := r.URL.Query()
		if n == 1 || n == 2 {
			if len(q["tag"]) != 2 || q["tag"][0] != "first" || q["tag"][1] != "second" || q.Get("limit") != "0" || len(q["name"]) != 1 || q["name"][0] != "" || q.Get("protected") != "false" || q.Get("os_hidden") != "false" || q.Get("size_min") != "0" || r.Header.Get("X-Call") != "list" || len(q) != 6+n-1 {
				t.Fatalf("owned list query/headers: %s %v", r.URL, r.Header)
			}
			if n == 2 && q.Get("marker") != "second-page" {
				t.Fatal("canonical marker lost")
			}
		} else if n == 3 {
			if q.Get("name") != "Ubuntu 24" || len(q) != 1 {
				t.Fatal(r.URL)
			}
		} else if r.URL.RawQuery != "" {
			t.Fatal("GetImage acquired query", r.URL)
		}
		code, body := http.StatusOK, actual
		switch n {
		case 0:
			if r.Header.Get("X-Call") != "owned" {
				t.Fatal("GET option snapshot lost")
			}
		case 1:
			body = `{"images":[` + actual + `],"next":"/v2/images?limit=0&name=&protected=false&os_hidden=false&size_min=0&tag=first&marker=second-page"}`
		case 2:
			body = `{"images":[],"next":null}`
		case 3:
			body = `{"images":[{"id":"resolved-id","name":"Ubuntu 24"}]}`
		case 4:
			if r.Header.Get("X-Call") != "named" {
				t.Fatal("Name resolver lost fixed option headers")
			}
			body = `{"id":"fresh-renamed","name":null,"size":null,"x-number":null}`
		case 5:
			code, body = http.StatusNotFound, "image hidden or missing"
		case 6:
			body = `{"tags":[null]}`
		default:
			t.Fatalf("unexpected operation: %s", r.URL)
		}
		provider.SetToken(fmt.Sprintf("read-%d", calls))
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
	version, err := conn.ImageV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	source = upper.RawClient()
	if version.RawClient() != source || upper.API.Images.RawClient() != source || source.ProviderClient != provider {
		t.Fatal("read facade replaced shared image client")
	}
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	headers := map[string]string{"X-Call": "owned"}
	option := image.WithGetImageHeaders(headers)
	headers["X-Call"] = "changed"
	fetched, err := upper.GetImage(ctx, resource.ID(id), option)
	if err != nil || fetched == nil || fetched.ID == nil || *fetched.ID != "actual-id" || fetched.Size == nil || *fetched.Size != 9007199254740993 || fetched.Protected == nil || *fetched.Protected || fetched.Hidden != nil || fetched.Tags == nil || fetched.StatusCode != 200 || calls != 1 {
		t.Fatalf("owned actual image=%+v err=%v calls=%d", fetched, err, calls)
	}
	fetched.Properties["x-number"][0] = '1'
	if string(fetched.Body["x-number"]) != "9007199254740995" || string(fetched.Properties["properties"]) != "[]" || string(fetched.Properties["metadata"]) != "true" {
		t.Fatal("raw projection aliases or forces nested property shape")
	}
	rows, err := upper.AllImages(ctx, image.WithListImagesLimit(0), image.WithListImagesName(""), image.WithListImagesProtected(false), image.WithListImagesHidden(false), image.WithListImagesSizeMin(0), image.WithListImagesTags("first", "second"), image.WithListImagesHeader("X-Call", "list"))
	if err != nil || len(rows) != 1 || *rows[0].ID != "actual-id" || rows[0].Header.Get("X-Proof") != "1" || calls != 3 {
		t.Fatalf("restored repeat query=%+v err=%v calls=%d", rows, err, calls)
	}
	named, err := upper.GetImage(ctx, resource.Name("Ubuntu 24"), image.WithGetImageHeader("X-Call", "named"))
	if err != nil || named == nil || *named.ID != "fresh-renamed" || named.Name != nil || named.Size != nil || string(named.Properties["x-number"]) != "null" || calls != 5 {
		t.Fatalf("fresh Name image=%+v err=%v calls=%d", named, err, calls)
	}
	missing, err := upper.GetImage(ctx, resource.ID(id))
	var native gophercloud.ErrUnexpectedResponseCode
	if missing != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "image hidden or missing" || calls != 6 {
		t.Fatalf("strict404=%+v %v calls=%d", missing, err, calls)
	}
	bad, err := upper.GetImage(ctx, resource.ID(id))
	var response *resource.ResponseError
	if bad != nil || !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != `{"tags":[null]}` || response.Header.Get("X-Proof") != "6" || calls != 7 {
		t.Fatalf("owned200 model error=%+v %v calls=%d", bad, err, calls)
	}
}
