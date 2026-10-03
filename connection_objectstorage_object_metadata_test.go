package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/objectstorage/v1/objects"
	"gophercloudsdk/resource"
)

func TestConnectionObjectMetadataSharesClientAndOwnsPhases(t *testing.T) {
	ctx := context.Background()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("swift-0")
	const endpoint = "https://cloud.test/reverse/swift/v1/AUTH_account%25/"
	const container = "백업 %2F?#"
	const object = "folder/내용 %2F?#"
	const target = "https://cloud.test/other%25/v1/AUTH_account%25/%EB%B0%B1%EC%97%85%20%252F%3F%23/folder%2F%EB%82%B4%EC%9A%A9%20%252F%3F%23"
	var source *gophercloud.ServiceClient
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		method, route, sourceIndex := http.MethodHead, target, n
		if n == 1 || n == 3 {
			route += "?symlink=get"
		}
		if n == 2 || n == 4 || n == 7 {
			method = http.MethodPost
			if n != 7 {
				sourceIndex--
			}
		}
		if r.Method != method || r.URL.String() != route || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", sourceIndex) || r.Header.Get("X-Auth-Token") != fmt.Sprintf("swift-%d", n+1) {
			t.Fatal(n, r.Method, r.URL, r.Header)
		}
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil || len(body) != 0 {
				t.Fatal(string(body), err)
			}
		}
		switch n {
		case 0:
			if r.Header.Get("X-Newest") != "false" {
				t.Fatal(r.Header)
			}
		case 1:
			if r.Header.Get("X-Call") != "" || r.Header.Get("Content-Disposition") != "" {
				t.Fatal("write options leaked to read", r.Header)
			}
		case 2:
			if r.Header.Get("X-Object-Meta-Owner") != "설명%20" || r.Header.Get("X-Object-Meta-Keep") != "untouched" || len(r.Header.Values("X-Object-Meta-Empty")) != 1 || r.Header.Get("X-Object-Meta-Empty") != "" || r.Header.Get("X-Call") != "owned" || r.Header.Get("Content-Type") != "source/type" || r.Header.Get("Content-Encoding") != "gzip" || r.Header.Get("Content-Disposition") != "attachment" || r.Header.Get("ETag") != "" || r.Header.Get("Last-Modified") != "" {
				t.Fatal("complete owned replacement", r.Header)
			}
		case 4:
			if _, present := r.Header["X-Object-Meta-Owner"]; present || r.Header.Get("X-Object-Meta-Keep") != "untouched" || len(r.Header.Values("X-Object-Meta-Empty")) != 1 || r.Header.Get("X-Remove-Object-Meta-Owner") != "" {
				t.Fatal("selected subtraction", r.Header)
			}
		case 7:
			if r.Header.Get("X-Object-Meta-Native") != "value" || r.Header.Get("X-Remove-Object-Meta-Old") != "remove" {
				t.Fatal(r.Header)
			}
		}
		header := http.Header{"X-Proof": {fmt.Sprint(n)}}
		if n == 0 || n == 1 || n == 3 {
			header["X-Object-Meta-Owner"] = []string{"observed"}
			header["X-Object-Meta-Keep"] = []string{"untouched"}
			header["X-Object-Meta-Empty"] = []string{""}
			header["Content-Length"] = []string{"9007199254740995"}
			header["Content-Type"] = []string{"observed/type"}
			header["Content-Encoding"] = []string{"gzip"}
			header["Content-Disposition"] = []string{"inline"}
			header["ETag"] = []string{"literal etag"}
			header["Last-Modified"] = []string{"literal PUT date"}
			header["X-Timestamp"] = []string{""}
			header["X-Unknown"] = []string{"one", "two"}
		}
		if n == 5 {
			header["Content-Length"] = []string{"1.5"}
		}
		code := http.StatusOK
		if n == 2 || n == 4 {
			code = http.StatusAccepted
		}
		if n == 6 {
			code = http.StatusNoContent
		}
		if n == 7 {
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
	if err != nil || service != versioned || service.Objects.RawClient() != service.RawClient() || service.RawClient().ProviderClient != provider {
		t.Fatal("shared facade", err)
	}
	source = service.RawClient()
	source.ResourceBase = "https://cloud.test/other%25/v1/AUTH_account%25/"
	source.MoreHeaders = map[string]string{"X-Source": "source-0", "Content-Type": "source/type"}
	get, err := service.Objects.GetMetadata(ctx, container, object, objects.WithGetMetadataNewest(false), func(o *objects.GetMetadataOpts) error {
		provider.SetToken("swift-1")
		return nil
	})
	if err != nil || get == nil || get.Metadata == nil || get.Metadata.ContentLength == nil || *get.Metadata.ContentLength != 9007199254740995 || get.Metadata.Timestamp == nil || *get.Metadata.Timestamp != "" || get.Metadata.LastModified == nil || *get.Metadata.LastModified != "literal PUT date" || get.Metadata.Values["owner"] != "observed" || get.StatusCode != 200 || calls != 1 {
		t.Fatal(get, err, calls)
	}
	values := map[string]string{"Owner": "설명%20", "Empty": ""}
	headers := map[string]string{"X-Call": "owned", "Content-Disposition": "attachment"}
	option := objects.WithMetadataHeaders(headers)
	headers["X-Call"] = "late"
	set, err := service.Objects.SetMetadata(ctx, container, object, values, option, func(o *objects.MetadataOpts) error {
		values["Owner"] = "late"
		return nil
	})
	if err != nil || set == nil || set.Before == nil || set.Before.Metadata == nil || set.Before.Metadata.Values["owner"] != "observed" || set.Acknowledgement == nil || set.Acknowledgement.StatusCode != 202 || string(set.Acknowledgement.Body) != "opaque" || calls != 3 {
		t.Fatal(set, err, calls)
	}
	deleted, err := service.Objects.DeleteMetadata(ctx, container, object, []string{"Owner", "Absent"})
	if err != nil || deleted == nil || deleted.Before == nil || deleted.Acknowledgement == nil || deleted.Acknowledgement.StatusCode != 202 || calls != 5 {
		t.Fatal(deleted, err, calls)
	}
	bad, err := service.Objects.GetMetadata(ctx, container, object)
	var proof *resource.ResponseError
	if bad == nil || bad.Metadata != nil || bad.StatusCode != 200 || string(bad.Body) != "opaque" || !errors.As(err, &proof) || proof.Header.Get("Content-Length") != "1.5" || calls != 6 {
		t.Fatal(bad, err, calls)
	}
	bad.Header.Set("X-Proof", "changed")
	bad.Body[0] = '!'
	set.Before.Metadata.Values["owner"] = "changed"
	if proof.Header.Get("X-Proof") != "5" || string(proof.Body) != "opaque" || set.Before.Header.Get("X-Object-Meta-Owner") != "observed" || get.Header.Get("X-Proof") != "0" || set.Before.Header.Get("X-Proof") != "1" || set.Acknowledgement.Header.Get("X-Proof") != "2" || deleted.Before.Header.Get("X-Proof") != "3" || deleted.Acknowledgement.Header.Get("X-Proof") != "4" {
		t.Fatal("phase evidence aliases")
	}
	if _, err := service.Objects.Get(ctx, container, object); err != nil {
		t.Fatal("native Get204", err)
	}
	if _, err := service.Objects.Update(ctx, container, object, objects.UpdateOpts{Metadata: map[string]string{"native": "value"}, RemoveMetadata: []string{"Old"}}); err != nil || calls != 8 {
		t.Fatal("native Update201", err, calls)
	}
}
