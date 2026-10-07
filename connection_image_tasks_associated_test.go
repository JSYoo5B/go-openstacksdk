package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/image"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestConnectionImageAssociatedTasksSharesClientAndResolvesName(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("associated-0")
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "이미지% ?#"
	calls := 0
	var source *gophercloud.ServiceClient
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		path := endpoint + "images/" + url.PathEscape(id) + "/tasks"
		header := fmt.Sprintf("source-%d", n)
		if n == 2 {
			path = endpoint + "images"
		} else if n == 3 {
			path = endpoint + "images/resolved-id/tasks"
			header = "source-2"
		}
		if r.Method != http.MethodGet || r.URL.Scheme+"://"+r.URL.Host+r.URL.EscapedPath() != path || r.Header.Get("X-Auth-Token") != fmt.Sprintf("associated-%d", n) || r.Header.Get("X-Source") != header || r.Body != nil {
			t.Fatalf("request%d: %s %s headers=%v", n, r.Method, r.URL, r.Header)
		}
		if n == 2 {
			if r.URL.Query().Get("name") != "Ubuntu 24" || len(r.URL.Query()) != 1 {
				t.Fatalf("name resolver query=%s", r.URL)
			}
		} else if r.URL.RawQuery != "" {
			t.Fatalf("finite task request has query: %s", r.URL)
		}
		code, body := http.StatusOK, ""
		switch n {
		case 0:
			if r.Header.Get("X-Call") != "snapshot" {
				t.Fatal("option factory lost header snapshot")
			}
			body = `{"tasks":[{"id":"actual-task","image_id":"actual-image","deleted":false,"deleted_at":"","input":{"n":9007199254740993},"request_id":"canonical","request-id":"passive"},{"deleted":17}],"next":{"invalid":"passive"}}`
		case 1:
			body = `{"tasks":[],"next":"https://foreign.test/unfollowed"}`
		case 2:
			body = `{"images":[{"id":"resolved-id","name":"Ubuntu 24"}]}`
		case 3:
			if r.Header.Get("X-Call") != "named" {
				t.Fatal("name resolution lost fixed call header")
			}
			body = `{"tasks":[{"id":"named-task","deleted":true,"deleted_at":null}]}`
		case 4:
			code, body = http.StatusNotFound, "image hidden or missing"
		case 5:
			body = `{"tasks":[{"deleted":"wrong"}]}`
		default:
			t.Fatalf("unexpected operation: %s", r.URL)
		}
		provider.SetToken(fmt.Sprintf("associated-%d", calls))
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
	if version.RawClient() != source || upper.API.Images.RawClient() != source || upper.API.Tasks.RawClient() != source || source.ProviderClient != provider {
		t.Fatal("associated Tasks replaced the shared image client")
	}
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	headers := map[string]string{"X-Call": "snapshot"}
	option := image.WithListImageTasksHeaders(headers)
	headers["X-Call"] = "changed"
	rows, err := upper.AllImageTasks(ctx, resource.ID(id), option, image.WithListImageTasksMaxItems(1))
	if err != nil || len(rows) != 1 || rows[0].ID == nil || *rows[0].ID != "actual-task" || rows[0].ImageID == nil || *rows[0].ImageID != "actual-image" || rows[0].Deleted == nil || *rows[0].Deleted || rows[0].DeletedAt == nil || *rows[0].DeletedAt != "" || rows[0].StatusCode != 200 || rows[0].Header.Get("X-Proof") != "0" || calls != 1 {
		t.Fatalf("actual owned row=%+v err=%v calls=%d", rows, err, calls)
	}
	rows[0].Input["n"][0] = '1'
	if string(rows[0].Body["input"]) != `{"n":9007199254740993}` || rows[0].RequestID == nil || *rows[0].RequestID != "canonical" || string(rows[0].Body["request-id"]) != `"passive"` {
		t.Fatal("typed field aliases raw metadata or uses decoy alias")
	}
	empty, err := upper.AllImageTasks(ctx, resource.ID(id))
	if err != nil || empty == nil || len(empty) != 0 || calls != 2 {
		t.Fatalf("empty finite list=%v err=%v calls=%d", empty, err, calls)
	}
	named, err := upper.AllImageTasks(ctx, resource.Name("Ubuntu 24"), image.WithListImageTasksHeader("X-Call", "named"))
	if err != nil || len(named) != 1 || named[0].ID == nil || *named[0].ID != "named-task" || named[0].ImageID != nil || named[0].Deleted == nil || !*named[0].Deleted || named[0].DeletedAt != nil || calls != 4 {
		t.Fatalf("name resolution=%+v err=%v calls=%d", named, err, calls)
	}
	missing, err := upper.AllImageTasks(ctx, resource.ID(id))
	var native gophercloud.ErrUnexpectedResponseCode
	if missing != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "image hidden or missing" || calls != 5 {
		t.Fatalf("strict404=%+v %v calls=%d", missing, err, calls)
	}
	bad, err := upper.AllImageTasks(ctx, resource.ID(id))
	var response *resource.ResponseError
	if bad != nil || !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != `{"tasks":[{"deleted":"wrong"}]}` || response.Header.Get("X-Proof") != "5" || calls != 6 {
		t.Fatalf("owned200 decode error=%+v %v calls=%d", bad, err, calls)
	}
	invalid, err := upper.AllImageTasks(ctx, resource.ID("../image"), func(*image.ListImageTasksOpts) error {
		t.Fatal("invalid parent reached option callback")
		return nil
	})
	if invalid != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 6 {
		t.Fatalf("invalid ref=%+v %v calls=%d", invalid, err, calls)
	}
}
