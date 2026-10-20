package openstack_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionContainerMetadataSharesClientAndOwnsEvidence(t *testing.T) {
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
		method := http.MethodHead
		if n == 1 || n == 2 || n == 5 {
			method = http.MethodPost
		}
		if r.Method != method || r.URL.String() != target || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", n) || r.Header.Get("X-Auth-Token") != fmt.Sprintf("swift-%d", n+1) {
			t.Fatal(n, r.Method, r.URL, r.Header)
		}
		for key := range r.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-account-meta-") {
				t.Fatal("account metadata header leaked to container", key)
			}
		}
		if r.Body != nil {
			b, err := io.ReadAll(r.Body)
			if err != nil || len(b) != 0 {
				t.Fatal(string(b), err)
			}
		}
		switch n {
		case 0:
			if r.Header.Get("X-Newest") != "false" {
				t.Fatal(r.Header)
			}
		case 1:
			if r.Header.Get("X-Container-Meta-Owner") != "설명%20" || r.Header.Get("X-Container-Meta-Empty") != "" || len(r.Header.Values("X-Container-Meta-Empty")) != 1 || r.Header.Get("X-Call") != "owned" || len(r.Header.Values("X-Container-Sync-Key")) != 1 || r.Header.Get("X-Container-Sync-Key") != "" || r.Header.Get("X-Newest") != "" {
				t.Fatal(r.Header)
			}
		case 2:
			if len(r.Header.Values("X-Container-Meta-Owner")) != 1 || r.Header.Get("X-Container-Meta-Owner") != "" || r.Header.Get("X-Remove-Container-Meta-Owner") != "" {
				t.Fatal(r.Header)
			}
		case 5:
			if r.Header.Get("X-Container-Meta-Native") != "value" || r.Header.Get("X-Remove-Container-Meta-Old") != "remove" {
				t.Fatal(r.Header)
			}
		}
		header := http.Header{"X-Proof": {fmt.Sprint(n)}, "X-Container-Object-Count": {"0"}}
		if n == 0 {
			header["X-Container-Bytes-Used"] = []string{"9007199254740995"}
			header["X-Timestamp"] = []string{""}
			header["Last-Modified"] = []string{"literal PUT date"}
			header["X-Container-Read"] = []string{"one", "two"}
			header["X-Container-Meta-Owner"] = []string{"observed"}
			header["X-Container-Meta-Temp-Url-Key"] = []string{"passive-key"}
		}
		if n == 3 {
			header["X-Container-Bytes-Used"] = []string{"1.5"}
		}
		code := http.StatusNoContent
		if n == 4 {
			code = http.StatusOK
		}
		if n == 5 {
			code = http.StatusCreated
		}
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		provider.SetToken(fmt.Sprintf("swift-%d", calls+1))
		return &http.Response{Request: r, StatusCode: code, Header: header, Body: io.NopCloser(strings.NewReader("opaque"))}, nil
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
	get, err := service.Containers.GetMetadata(ctx, name, containers.WithGetMetadataNewest(false), func(o *containers.GetMetadataOpts) error {
		source.MoreHeaders["X-Source"] = "callback"
		provider.SetToken("swift-1")
		return nil
	})
	if err != nil || get == nil || get.Metadata == nil || get.Metadata.BytesUsed == nil || get.Metadata.ObjectCount == nil || *get.Metadata.BytesUsed != 9007199254740995 || *get.Metadata.ObjectCount != 0 || get.Metadata.LastModified == nil || *get.Metadata.LastModified != "literal PUT date" || get.Metadata.Timestamp == nil || *get.Metadata.Timestamp != "" || get.Metadata.Values["owner"] != "observed" || get.Metadata.Values["temp-url-key"] != "passive-key" || get.StatusCode != 204 || string(get.Body) != "opaque" || calls != 1 {
		t.Fatal(get, err, calls)
	}
	get.Metadata.Values["owner"] = "changed"
	if get.Header.Get("X-Container-Meta-Owner") != "observed" || len(get.Header.Values("X-Container-Read")) != 2 {
		t.Fatal("metadata aliases header")
	}
	values := map[string]string{"Owner": "설명%20", "Empty": ""}
	headers := map[string]string{"X-Call": "owned", "X-Container-Sync-Key": ""}
	option := containers.WithMetadataHeaders(headers)
	headers["X-Call"] = "changed"
	set, err := service.Containers.SetMetadata(ctx, name, values, option, func(o *containers.MetadataOpts) error {
		values["Owner"] = "late"
		source.MoreHeaders["X-Source"] = "callback"
		return nil
	})
	if err != nil || set == nil || set.StatusCode != 204 || string(set.Body) != "opaque" || calls != 2 {
		t.Fatal(set, err, calls)
	}
	deleted, err := service.Containers.DeleteMetadata(ctx, name, []string{"Owner"})
	if err != nil || deleted == nil || deleted.StatusCode != 204 || calls != 3 {
		t.Fatal(deleted, err, calls)
	}
	bad, err := service.Containers.GetMetadata(ctx, name)
	var proof *resource.ResponseError
	if bad == nil || bad.Metadata != nil || bad.StatusCode != 204 || string(bad.Body) != "opaque" || !errors.As(err, &proof) || proof.Header.Get("X-Container-Bytes-Used") != "1.5" || proof.StatusCode != 204 || calls != 4 {
		t.Fatal(bad, err, calls)
	}
	bad.Header.Set("X-Proof", "changed")
	bad.Body[0] = '!'
	if proof.Header.Get("X-Proof") != "3" || string(proof.Body) != "opaque" || get.Header.Get("X-Proof") != "0" || set.Header.Get("X-Proof") != "1" || deleted.Header.Get("X-Proof") != "2" {
		t.Fatal("response evidence aliases")
	}
	if _, err := service.Containers.Get(ctx, name); err != nil {
		t.Fatal("native Get compatibility", err)
	}
	if _, err := service.Containers.Update(ctx, name, containers.UpdateOpts{Metadata: map[string]string{"native": "value"}, RemoveMetadata: []string{"Old"}}); err != nil || calls != 6 {
		t.Fatal("native Update201 compatibility", err, calls)
	}
}
