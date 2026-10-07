package gophercloudsdk_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionObjectDeleteSharesClientAndOwnsPhases(t *testing.T) {
	ctx := context.Background()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("swift-0")
	const endpoint = "https://cloud.test/reverse/swift/v1/AUTH_account%25/"
	const container = "백업 %2F?#"
	const object = "folder/내용 %2F?#"
	const target = "https://cloud.test/other%25/v1/AUTH_account%25/%EB%B0%B1%EC%97%85%20%252F%3F%23/folder%2F%EB%82%B4%EC%9A%A9%20%252F%3F%23"
	const version = "v%2F?#"
	const bulk = " \r\n" + `{"Response Status":"200 OK","Response Body":"","Number Deleted":2,"Number Not Found":1,"Errors":[],"plugin":9007199254740993}`
	var source *gophercloud.ServiceClient
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		method, route, sourceIndex := http.MethodHead, target, n
		if n == 0 {
			route += "?version-id=" + url.QueryEscape(version)
		}
		if n == 1 {
			method = http.MethodDelete
			route += "?multipart-manifest=delete&version-id=" + url.QueryEscape(version)
			sourceIndex = 0
		}
		if n == 2 {
			method = http.MethodDelete
		}
		if r.Method != method || r.URL.String() != route || r.Body != nil || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", sourceIndex) || r.Header.Get("X-Auth-Token") != fmt.Sprintf("swift-%d", n+1) || r.Header.Get("X-Call") != "owned" {
			t.Fatal(n, r.Method, r.URL, r.Header)
		}
		if n < 2 {
			if r.Header.Get("X-Newest") != "false" {
				t.Fatal("explicit newest absent", r.Header)
			}
		} else if len(r.Header.Values("X-Newest")) != 0 {
			t.Fatal("newest omission lost", r.Header)
		}
		if n == 1 && r.Header.Get("Accept") != "application/json" {
			t.Fatal("SLO report format", r.Header)
		}
		header := http.Header{"X-Proof": {fmt.Sprint(n)}}
		status, body := http.StatusOK, "head-opaque"
		if n == 0 {
			header.Set("X-Static-Large-Object", "TRUE")
		}
		if n == 1 {
			body = bulk
		}
		if n == 2 {
			status = http.StatusAccepted
			body = "ack-opaque"
		}
		if n == 3 {
			status = http.StatusNotFound
			body = "missing-head"
		}
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		provider.SetToken(fmt.Sprintf("swift-%d", calls+1))
		return &http.Response{Request: r, StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.ObjectStorage, "v1", endpoint))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	versioned, err := conn.ObjectStorageV1(ctx)
	if err != nil || service != versioned || service.Objects.RawClient() != service.RawClient() || service.RawClient().ProviderClient != provider {
		t.Fatal("shared facade", err)
	}
	source = service.RawClient()
	source.ResourceBase = "https://cloud.test/other%25/v1/AUTH_account%25/"
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	headers := map[string]string{"X-Call": "owned"}
	option := objects.WithDeleteObjectHeaders(headers)
	headers["X-Call"] = "caller mutation"
	callbacks := 0
	result, err := service.Objects.DeleteObject(ctx, container, object, option, objects.WithDeleteObjectVersionID(version), objects.WithDeleteObjectNewest(false), func(o *objects.DeleteObjectOpts) error {
		callbacks++
		source.MoreHeaders["X-Source"] = "callback change"
		provider.SetToken("swift-1")
		return nil
	})
	if err != nil || result == nil || result.Discovery == nil || result.Deletion == nil || result.Discovery.StatusCode != 200 || result.Deletion.StatusCode != 200 || string(result.Discovery.Body) != "head-opaque" || string(result.Deletion.Body) != bulk || result.StaticLargeObject == nil || !*result.StaticLargeObject || result.IgnoredMissing || result.Bulk == nil || result.Bulk.NumberDeleted != 2 || result.Bulk.NumberNotFound != 1 || string(result.Bulk.Body["plugin"]) != "9007199254740993" || calls != 2 || callbacks != 1 {
		t.Fatal(result, err, calls, callbacks)
	}
	known, err := service.Objects.DeleteObject(ctx, container, object, option, objects.WithDeleteObjectStaticLargeObject(false))
	if err != nil || known == nil || known.Discovery != nil || known.Deletion == nil || known.Deletion.StatusCode != 202 || string(known.Deletion.Body) != "ack-opaque" || known.Bulk != nil || known.StaticLargeObject == nil || *known.StaticLargeObject || calls != 3 {
		t.Fatal(known, err, calls)
	}
	missing, err := service.Objects.DeleteObject(ctx, container, object, option)
	if err != nil || missing == nil || missing.Discovery == nil || missing.Discovery.StatusCode != 404 || string(missing.Discovery.Body) != "missing-head" || missing.Deletion != nil || missing.StaticLargeObject != nil || !missing.IgnoredMissing || calls != 4 {
		t.Fatal(missing, err, calls)
	}
	result.Discovery.Header.Set("X-Proof", "changed")
	result.Deletion.Body[0] = '!'
	result.Bulk.Body["plugin"][0] = '!'
	*result.StaticLargeObject = false
	if string(known.Deletion.Body) != "ack-opaque" || known.Deletion.Header.Get("X-Proof") != "2" || missing.Discovery.Header.Get("X-Proof") != "3" || result.Deletion.Header.Get("X-Proof") != "1" || result.Bulk.Header.Get("X-Proof") != "1" || source.ResourceBase != "https://cloud.test/other%25/v1/AUTH_account%25/" || headers["X-Call"] != "caller mutation" || provider.Token() != "swift-5" {
		t.Fatal("phase or provider ownership")
	}
}
