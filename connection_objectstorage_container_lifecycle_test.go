package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionContainerLifecycleSharesClientAndPreservesNative(t *testing.T) {
	ctx := context.Background()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("swift-0")
	const endpoint = "https://cloud.test/reverse/swift/v1/AUTH_account%25/"
	const name = "백업 %2F?#"
	const target = "https://cloud.test/other%25/v1/AUTH_account%25/%EB%B0%B1%EC%97%85%20%252F%3F%23"
	var source *gophercloud.ServiceClient
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		method := http.MethodDelete
		if n == 0 || n == 1 {
			method = http.MethodPut
		}
		if r.Method != method || r.URL.String() != target || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", n) || r.Header.Get("X-Auth-Token") != fmt.Sprintf("swift-%d", n+1) {
			t.Fatal(n, r.Method, r.URL, r.Header)
		}
		if r.Body != nil {
			b, err := io.ReadAll(r.Body)
			if err != nil || len(b) != 0 {
				t.Fatal(string(b), err)
			}
		}
		if n == 0 && (r.Header.Get("X-Container-Meta-Owner") != "설명%20" || len(r.Header.Values("X-Container-Meta-Empty")) != 1 || r.Header.Get("X-Call") != "owned" || r.Header.Get("X-Container-Read") != ".r:*" || r.Header.Get("X-Newest") != "") {
			t.Fatal(r.Header)
		}
		if n > 0 && n < 5 && len(r.Header.Values("X-Container-Meta-Owner")) != 0 {
			t.Fatal("metadata leaked into later call", r.Header)
		}
		codes := []int{201, 202, 404, 404, 204}
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		provider.SetToken(fmt.Sprintf("swift-%d", calls+1))
		return &http.Response{Request: r, StatusCode: codes[n], Header: http.Header{"X-Proof": {fmt.Sprint(n)}, "Date": {"opaque invalid date"}}, Body: io.NopCloser(strings.NewReader("opaque"))}, nil
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
	if err != nil || service != versioned || service.Containers.RawClient() != service.RawClient() || service.RawClient().ProviderClient != provider {
		t.Fatal("shared object storage facade", err)
	}
	source = service.RawClient()
	source.ResourceBase = "https://cloud.test/other%25/v1/AUTH_account%25/"
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	values := map[string]string{"Owner": "설명%20", "Empty": ""}
	option := containers.WithCreateContainerMetadata(values)
	values["Owner"] = "changed after factory"
	created, err := service.Containers.CreateContainer(ctx, name, option,
		containers.WithCreateContainerHeader("X-Call", "owned"),
		containers.WithCreateContainerHeader("X-Container-Read", ".r:*"),
		func(o *containers.CreateContainerOpts) error {
			source.MoreHeaders["X-Source"] = "callback"
			provider.SetToken("swift-1")
			return nil
		})
	if err != nil || created == nil || created.StatusCode != 201 || created.IgnoredMissing || string(created.Body) != "opaque" || calls != 1 {
		t.Fatal(created, err, calls)
	}
	existing, err := service.Containers.CreateContainer(ctx, name)
	if err != nil || existing == nil || existing.StatusCode != 202 || calls != 2 {
		t.Fatal(existing, err, calls)
	}
	missing, err := service.Containers.DeleteContainer(ctx, name)
	if err != nil || missing == nil || missing.StatusCode != 404 || !missing.IgnoredMissing || string(missing.Body) != "opaque" || calls != 3 {
		t.Fatal(missing, err, calls)
	}
	strict, err := service.Containers.DeleteContainer(ctx, name, containers.WithDeleteContainerIgnoreMissing(false))
	var native gophercloud.ErrUnexpectedResponseCode
	if strict != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "opaque" || calls != 4 {
		t.Fatal(strict, err, calls)
	}
	deleted, err := service.Containers.DeleteContainer(ctx, name)
	if err != nil || deleted == nil || deleted.StatusCode != 204 || deleted.IgnoredMissing || calls != 5 {
		t.Fatal(deleted, err, calls)
	}
	created.Header.Set("X-Proof", "changed")
	created.Body[0] = '!'
	if existing.Header.Get("X-Proof") != "1" || string(existing.Body) != "opaque" || missing.Header.Get("X-Proof") != "2" || string(missing.Body) != "opaque" || deleted.Header.Get("X-Proof") != "4" {
		t.Fatal("response evidence aliases across calls")
	}
	// Keep the broader native status policy available without parsing dates here.
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		if r.URL.String() != target || n == 5 && r.Method != http.MethodPut || n == 6 && r.Method != http.MethodDelete {
			t.Fatal(n, r.Method, r.URL)
		}
		code := 204
		if n == 6 {
			code = 202
		}
		return &http.Response{Request: r, StatusCode: code, Header: http.Header{"Date": {"Mon, 02 Jan 2006 15:04:05 GMT"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	if _, err := service.Containers.Create(ctx, name, containers.CreateOpts{}); err != nil {
		t.Fatal("native Create204 compatibility", err)
	}
	if _, err := service.Containers.Delete(ctx, name); err != nil || calls != 7 {
		t.Fatal("native Delete202 compatibility", err, calls)
	}
	blocked, err := service.Containers.CreateContainer(ctx, name, func(o *containers.CreateContainerOpts) error {
		source.ResourceBase = "https://cloud.test/changed/"
		return nil
	})
	if blocked != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 7 {
		t.Fatal("source changed before HTTP", blocked, err, calls)
	}
}
