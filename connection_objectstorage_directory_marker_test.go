package gophercloudsdk_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionSwiftDirectoryMarkerOwnsMediaAndFreshCapture(t *testing.T) {
	ctx := context.Background()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("first")
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.ObjectStorage, "v1", "https://cloud.test/swift/v1/AUTH_account/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	same, err := conn.ObjectStorageV1(ctx)
	if err != nil || same != service || service.Objects.RawClient() != service.RawClient() || service.RawClient().ProviderClient != provider {
		t.Fatal("shared client", err)
	}
	source := service.RawClient()
	source.ResourceBase = "https://cloud.test/reverse/a%20b/v1/AUTH_live/"
	source.MoreHeaders = map[string]string{"Content-Type": "source/type", "X-Source": "first"}
	container, name := "백업 %2F?#", "folder/마커 %2F?#"
	target := source.ResourceBase + url.PathEscape(container) + "/" + url.PathEscape(name)
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body []byte
		var readErr error
		if r.Body != nil {
			body, readErr = io.ReadAll(r.Body)
		}
		wantSource, wantToken := "first", "first"
		if calls == 2 {
			wantSource, wantToken = "later", "second"
		}
		if r.Method != "PUT" || r.URL.String() != target || readErr != nil || len(body) != 0 || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "application/directory" || r.Header.Get("X-Call") != "factory" || r.Header.Get("X-Object-Meta-Owner") != "sdk" || r.Header.Get("X-Source") != wantSource || r.Header.Get("X-Auth-Token") != wantToken {
			t.Errorf("request%d %s %s body%q headers%v", calls, r.Method, r.URL, body, r.Header)
		}
		status := 201
		if calls == 2 {
			status = 202
		}
		return &http.Response{Request: r, StatusCode: status, Header: http.Header{"X-Proof": {"owned"}}, Body: io.NopCloser(strings.NewReader("marker acknowledgement"))}, nil
	})
	headers := map[string]string{"Content-Type": "caller/type", "X-Call": "factory"}
	metadata := map[string]string{"Owner": "sdk"}
	factory := objects.WithDirectoryMarkerOpts(objects.DirectoryMarkerOpts{Headers: headers, Metadata: metadata})
	headers["X-Call"], metadata["Owner"] = "mutated", "outside"
	callbacks := 0
	first, err := service.Objects.CreateDirectoryMarkerObject(ctx, container, name, factory, func(*objects.DirectoryMarkerOpts) error { callbacks++; return nil })
	if err != nil || first == nil || first.Source != "bytes" || first.Mode != "ordinary" || first.Size != 0 || first.MD5 != "" || first.SHA256 != "" || first.Skipped || first.Ordinary == nil || first.Ordinary.Acknowledgement == nil || first.Ordinary.Acknowledgement.StatusCode != 201 || len(first.Ordinary.Attempts) != 1 || callbacks != 1 || calls != 1 {
		t.Fatal(first, err, calls, callbacks)
	}
	source.MoreHeaders["X-Source"] = "later"
	source.MoreHeaders["Content-Type"] = "later/type"
	provider.SetToken("second")
	second, err := service.Objects.CreateDirectoryMarkerObject(ctx, container, name, factory)
	if err != nil || second == nil || second.Ordinary == nil || second.Ordinary.Acknowledgement == nil || second.Ordinary.Acknowledgement.StatusCode != 202 || calls != 2 || first.Capabilities != nil || first.Discovery != nil || first.Manifest != nil || len(first.Segments) != 0 || len(first.Cleanup) != 0 {
		t.Fatal(first, second, err, calls)
	}
	first.Ordinary.Acknowledgement.Body[0] = '!'
	second.Ordinary.Acknowledgement.Header.Set("X-Proof", "changed")
	if string(first.Ordinary.Attempts[0].Response.Body) != "marker acknowledgement" || second.Ordinary.Attempts[0].Response.Header.Get("X-Proof") != "owned" || source.MoreHeaders["Content-Type"] != "later/type" || source.MoreHeaders["X-Call"] != "" {
		t.Fatal("phase proof or source aliased")
	}
}
