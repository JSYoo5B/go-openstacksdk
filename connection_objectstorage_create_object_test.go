package gophercloudsdk_test

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/objectstorage/v1/objects"
)

type connectionCreateReader struct {
	*strings.Reader
	closed, sought int
}

func (r *connectionCreateReader) Close() error {
	r.closed++
	return fmt.Errorf("caller Close forbidden")
}
func (r *connectionCreateReader) Seek(offset int64, whence int) (int64, error) {
	r.sought++
	return 0, fmt.Errorf("caller Seek forbidden")
}

func TestConnectionObjectCreateSharesClientAndOwnsSources(t *testing.T) {
	ctx := context.Background()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("swift-0")
	const endpoint = "https://cloud.test/reverse/swift/v1/AUTH_account%25/"
	const resourceBase = "https://cloud.test/other%25/v1/AUTH_account%25/"
	const container = "백업 %2F?#"
	const object = "folder/내용 %2F?#"
	const target = "https://cloud.test/other%25/v1/AUTH_account%25/%EB%B0%B1%EC%97%85%20%252F%3F%23/folder%2F%EB%82%B4%EC%9A%A9%20%252F%3F%23"
	const data = "binary\x00data"
	const stream = "stream\x00bytes"
	md5Digest := fmt.Sprintf("%x", md5.Sum([]byte(stream)))
	shaDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(stream)))
	var source *gophercloud.ServiceClient
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		method, route := http.MethodHead, target
		if n == 0 {
			method = http.MethodPut
		}
		if n == 2 {
			method = http.MethodGet
			route = "https://cloud.test/reverse/swift/info"
		}
		sourceIndex := n
		if n == 3 {
			sourceIndex = 2
		}
		if r.Method != method || r.URL.String() != route || r.Header.Get("X-Auth-Token") != fmt.Sprintf("swift-%d", n+1) || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", sourceIndex) {
			t.Fatal(n, r.Method, r.URL, r.Header)
		}
		status, body := http.StatusOK, "head-proof"
		headers := http.Header{"X-Proof": {fmt.Sprint(n)}}
		switch n {
		case 0:
			sent, err := io.ReadAll(r.Body)
			if err != nil || string(sent) != data || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("X-Call") != "owned" || len(r.Header.Values("Etag")) != 0 {
				t.Fatal(string(sent), r.Header, err)
			}
			status, body = http.StatusCreated, "bytes-ack"
		case 1:
			if r.Body != nil || r.Header.Get("X-Call") != "stale-owned" {
				t.Fatal(r.Header, r.Body)
			}
			headers.Set("X-Object-Meta-X-Sdk-Md5", md5Digest)
		case 2:
			if r.Body != nil || len(r.Header.Values("Content-Type")) != 0 || r.Header.Get("X-Call") != "" {
				t.Fatal("upload header leaked", r.Header)
			}
			body = `{"swift":{"max_file_size":1024},"slo":{"min_segment_size":0}}`
		case 3:
			if r.Body != nil || len(r.Header.Values("Content-Type")) != 0 || r.Header.Get("X-Call") != "" {
				t.Fatal("upload header leaked", r.Header)
			}
			headers.Set("X-Object-Meta-X-Sdk-Md5", md5Digest)
			headers.Set("X-Object-Meta-X-Sdk-Sha256", shaDigest)
		default:
			t.Fatal("unexpected write after stale match", n)
		}
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		provider.SetToken(fmt.Sprintf("swift-%d", calls+1))
		return &http.Response{Request: r, StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body))}, nil
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
	source.ResourceBase = resourceBase
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	input := []byte(data)
	headers := map[string]string{"X-Call": "owned", "Content-Type": "application/octet-stream"}
	option := objects.WithCreateObjectHeaders(headers)
	headers["X-Call"] = "caller mutation"
	callbacks := 0
	created, err := service.Objects.CreateObject(ctx, container, object, objects.CreateObjectInput{Data: input}, option, func(o *objects.CreateObjectOpts) error {
		callbacks++
		input[0] = '!'
		provider.SetToken("swift-1")
		return nil
	})
	if err != nil || created == nil || created.Source != "bytes" || created.Mode != "ordinary" || created.Size != int64(len(data)) || created.Capabilities != nil || created.Discovery != nil || created.Ordinary == nil || created.Ordinary.Acknowledgement == nil || created.Ordinary.Acknowledgement.StatusCode != 201 || string(created.Ordinary.Acknowledgement.Body) != "bytes-ack" || created.Skipped || callbacks != 1 || calls != 1 {
		t.Fatal(created, err, callbacks, calls)
	}
	stale, err := service.Objects.IsObjectStale(ctx, container, object, "/explicit/path/which/does/not/exist", objects.WithIsObjectStaleMD5(md5Digest), objects.WithIsObjectStaleHeader("X-Call", "stale-owned"))
	if err != nil || stale == nil || stale.Stale == nil || *stale.Stale || stale.Discovery == nil || stale.Discovery.StatusCode != 200 || stale.MD5 != md5Digest || stale.RemoteMD5 == nil || *stale.RemoteMD5 != md5Digest || calls != 2 {
		t.Fatal(stale, err, calls)
	}
	reader := &connectionCreateReader{Reader: strings.NewReader(stream)}
	skipped, err := service.Objects.CreateObject(ctx, container, object, objects.CreateObjectInput{Reader: reader}, objects.WithCreateObjectHeader("Content-Type", "text/plain"), objects.WithCreateObjectHeader("X-Call", "upload-only"), objects.WithCreateObjectMetadataValue("changed", "must-not-trigger-write"))
	if err != nil || skipped == nil || skipped.Source != "reader" || !skipped.Skipped || skipped.Mode != "" || skipped.Capabilities == nil || skipped.Capabilities.Response == nil || skipped.Capabilities.Response.StatusCode != 200 || skipped.Discovery == nil || skipped.Discovery.StatusCode != 200 || skipped.Ordinary != nil || skipped.Manifest != nil || skipped.MD5 != md5Digest || skipped.SHA256 != shaDigest || calls != 4 || reader.Len() != 0 || reader.closed != 0 || reader.sought != 0 {
		t.Fatal(skipped, err, calls, reader)
	}
	created.Ordinary.Acknowledgement.Body[0] = '!'
	stale.Discovery.Header.Set("X-Proof", "changed")
	if string(created.Ordinary.Attempts[0].Response.Body) != "bytes-ack" || skipped.Discovery.Header.Get("X-Proof") != "3" || skipped.Capabilities.Response.Header.Get("X-Proof") != "2" || source.ResourceBase != resourceBase || provider.Token() != "swift-5" || headers["X-Call"] != "caller mutation" {
		t.Fatal("proof/source ownership")
	}
}
