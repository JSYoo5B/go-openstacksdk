package gophercloudsdk_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionSwiftObjectWaitSharesClientAndFreshCapture(t *testing.T) {
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
	versioned, err := conn.ObjectStorageV1(ctx)
	if err != nil || versioned != service || service.Objects.RawClient() != service.RawClient() || service.RawClient().ProviderClient != provider {
		t.Fatal("shared client", err)
	}
	source := service.RawClient()
	source.ResourceBase = "https://cloud.test/reverse/a%20b/v1/AUTH_live/"
	source.MoreHeaders = map[string]string{"X-Source": "captured"}
	container, object := "백업 %2F?#", "folder/내용 %2F?#"
	target := source.ResourceBase + url.PathEscape(container) + "/" + url.PathEscape(object)
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		wantSource, wantToken := "captured", "first"
		if calls == 2 {
			wantToken = "second"
		}
		if calls >= 3 {
			wantSource, wantToken = "later", "second"
		}
		if calls == 4 {
			wantToken = "third"
		}
		if r.Method != "HEAD" || r.URL.String() != target || r.Body != nil || r.Header.Get("X-Call") != "factory" || r.Header.Get("X-Source") != wantSource || r.Header.Get("X-Auth-Token") != wantToken {
			t.Errorf("poll%d %s %s body=%v headers=%v", calls, r.Method, r.URL, r.Body, r.Header)
		}
		status, body := 200, "exists"
		header := http.Header{"X-Proof": {"owned"}, "X-Object-Meta-State": {"pending"}}
		if calls == 2 {
			status, body = 404, "missing"
		}
		if calls == 4 {
			status, body = 204, "state proof"
			header.Set("X-Object-Meta-State", "aCtIvE")
		}
		if calls > 4 {
			t.Errorf("unexpected extra poll%d", calls)
		}
		return &http.Response{Request: r, StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	interval, timeout := time.Millisecond, time.Second
	headers := map[string]string{"X-Call": "factory"}
	factory := objects.WithObjectWaitOpts(objects.ObjectWaitOpts{Headers: headers, Interval: &interval, Timeout: &timeout})
	headers["X-Call"], interval, timeout = "caller mutation", -time.Second, -time.Second
	callbacks := 0
	deleted, err := service.Objects.WaitForDelete(ctx, container, object, factory, objects.WithObjectWaitProgressCallback(func(progress int) error {
		callbacks++
		if progress != 0 {
			t.Errorf("invented progress%d", progress)
		}
		source.MoreHeaders["X-Source"] = "later"
		provider.SetToken("second")
		return nil
	}))
	if err != nil || deleted == nil || !deleted.Complete || !deleted.Deleted || deleted.Status != nil || deleted.Polls != 2 || deleted.Last == nil || deleted.Last.Acknowledgement == nil || deleted.Last.Acknowledgement.StatusCode != 404 || callbacks != 1 || calls != 2 {
		t.Fatal(deleted, err, calls, callbacks)
	}
	if value, err := service.Objects.WaitForStatus(ctx, container, object, "ACTIVE"); value != nil || !errors.Is(err, resource.ErrUnsupported) || calls != 2 {
		t.Fatal("Swift has no default status", value, err, calls)
	}
	ready, err := service.Objects.WaitForStatus(ctx, container, object, "ACTIVE", factory, objects.WithObjectWaitStatusHeader("X-Object-Meta-State"), objects.WithObjectWaitProgressCallback(func(progress int) error {
		callbacks++
		if progress != 0 {
			t.Errorf("invented progress%d", progress)
		}
		provider.SetToken("third")
		return nil
	}))
	if err != nil || ready == nil || !ready.Complete || ready.Deleted || ready.Status == nil || *ready.Status != "aCtIvE" || ready.Polls != 2 || ready.Last == nil || ready.Last.Acknowledgement == nil || ready.Last.Acknowledgement.StatusCode != 204 || callbacks != 2 || calls != 4 {
		t.Fatal(ready, err, calls, callbacks)
	}
	deleted.Last.Acknowledgement.Body[0] = '!'
	ready.Last.Acknowledgement.Header.Set("X-Proof", "changed")
	if string(deleted.Last.Attempts[0].Response.Body) != "missing" || ready.Last.Attempts[0].Response.Header.Get("X-Proof") != "owned" || source.MoreHeaders["X-Source"] != "later" || provider.Token() != "third" {
		t.Fatal("phase proof or original source aliased")
	}
}
