package gophercloudsdk_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	swift "gophercloudsdk/objectstorage/v1"
	"gophercloudsdk/objectstorage/v1/accounts"
	"gophercloudsdk/objectstorage/v1/containers"
)

func TestConnectionTempURLKeySharesClientAndSnapshotsEachWorkflow(t *testing.T) {
	ctx := context.Background()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("token-0")
	const endpoint = "https://cloud.test/proxy/swift/v1/AUTH_account%25/"
	const name = "백업 %2F?#"
	const target = "https://cloud.test/data%25/v1/AUTH_account%25/%EB%B0%B1%EC%97%85%20%252F%3F%23"
	var source *gophercloud.ServiceClient
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		method, url := http.MethodHead, endpoint
		if n < 2 {
			method = http.MethodPost
		}
		if n == 1 || n == 2 || n == 3 {
			url = target
		}
		expectedSource := n
		if n == 4 {
			expectedSource = 3
		}
		if r.Method != method || r.URL.String() != url || r.Header.Get("X-Source") != fmt.Sprintf("source-%d", expectedSource) || r.Header.Get("X-Auth-Token") != fmt.Sprintf("token-%d", n+1) {
			t.Fatal(n, r.Method, r.URL, r.Header)
		}
		if r.Body != nil {
			b, err := io.ReadAll(r.Body)
			if err != nil || len(b) != 0 {
				t.Fatal(string(b), err)
			}
		}
		if n == 0 && (r.Header.Get("X-Account-Meta-Temp-Url-Key") != "primary literal" || len(r.Header.Values("X-Account-Meta-Temp-Url-Key-2")) != 0 || r.Header.Get("X-Call") != "owned") {
			t.Fatal(r.Header)
		}
		if n == 1 && (len(r.Header.Values("X-Container-Meta-Temp-Url-Key-2")) != 1 || r.Header.Get("X-Container-Meta-Temp-Url-Key-2") != "" || len(r.Header.Values("X-Container-Meta-Temp-Url-Key")) != 0) {
			t.Fatal(r.Header)
		}
		if n >= 2 && (len(r.Header.Values("X-Account-Meta-Temp-Url-Key")) != 0 || len(r.Header.Values("X-Container-Meta-Temp-Url-Key-2")) != 0) {
			t.Fatal("setter metadata leaked", r.Header)
		}
		if n == 3 || n == 4 {
			if r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Call") != "fallback" {
				t.Fatal(r.Header)
			}
		} else if r.Header.Get("X-Newest") != "" {
			t.Fatal(r.Header)
		}
		header := http.Header{"X-Proof": {fmt.Sprint(n)}}
		switch n {
		case 2:
			header.Set("X-Container-Meta-Temp-Url-Key", "container-primary")
			header.Set("X-Container-Meta-Temp-Url-Key-2", "보조 key")
		case 3:
			header.Set("X-Container-Meta-Temp-Url-Key-2", "")
		case 4:
			header.Set("X-Account-Meta-Temp-Url-Key", "account-primary")
			header.Set("X-Account-Meta-Temp-Url-Key-2", "account-secondary")
		case 5:
			header.Set("X-Account-Meta-Temp-Url-Key", "account-only")
		}
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		provider.SetToken(fmt.Sprintf("token-%d", calls+1))
		return &http.Response{Request: r, StatusCode: 204, Header: header, Body: io.NopCloser(strings.NewReader("opaque"))}, nil
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
	if err != nil || versioned != service || service.Accounts.RawClient() != service.RawClient() || service.Containers.RawClient() != service.RawClient() || service.RawClient().ProviderClient != provider {
		t.Fatal("shared facade", err)
	}
	source = service.RawClient()
	source.ResourceBase = "https://cloud.test/data%25/v1/AUTH_account%25/"
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	headers := map[string]string{"X-Call": "owned"}
	option := accounts.WithSetTempURLKeyOpts(accounts.SetTempURLKeyOpts{Headers: headers})
	headers["X-Call"] = "changed after factory"
	account, err := service.Accounts.SetTempURLKey(ctx, "primary literal", option, func(o *accounts.SetTempURLKeyOpts) error {
		source.MoreHeaders["X-Source"] = "callback"
		provider.SetToken("token-1")
		return nil
	})
	if err != nil || account == nil || account.StatusCode != 204 || calls != 1 {
		t.Fatal(account, err, calls)
	}
	container, err := service.Containers.SetTempURLKey(ctx, name, "", containers.WithSetTempURLKeySecondary(true))
	if err != nil || container == nil || calls != 2 {
		t.Fatal(container, err, calls)
	}
	selected, err := service.GetTempURLKey(ctx, swift.WithGetTempURLKeyContainer(name))
	if err != nil || selected == nil || string(selected.Key) != "보조 key" || !selected.Secondary || !selected.FromContainer || selected.Container == nil || selected.Account != nil || calls != 3 {
		t.Fatal(selected, err, calls)
	}
	fallback, err := service.GetTempURLKey(ctx, swift.WithGetTempURLKeyOpts(swift.GetTempURLKeyOpts{Container: name}), swift.WithGetTempURLKeyHeader("X-Call", "fallback"), swift.WithGetTempURLKeyNewest(false))
	if err != nil || fallback == nil || string(fallback.Key) != "account-secondary" || !fallback.Secondary || fallback.FromContainer || fallback.Container == nil || fallback.Account == nil || calls != 5 {
		t.Fatal(fallback, err, calls)
	}
	// Account-only discovery uses Endpoint even if the unused ResourceBase is invalid.
	source.ResourceBase = "invalid unused base"
	accountOnly, err := service.GetTempURLKey(ctx)
	if err != nil || accountOnly == nil || string(accountOnly.Key) != "account-only" || accountOnly.Secondary || accountOnly.FromContainer || accountOnly.Container != nil || accountOnly.Account == nil || calls != 6 {
		t.Fatal(accountOnly, err, calls)
	}
	selected.Key[0] = '!'
	selected.Container.Header.Set("X-Proof", "changed")
	selected.Container.Metadata.Values["temp-url-key-2"] = "changed"
	if string(fallback.Key) != "account-secondary" || fallback.Container.Header.Get("X-Proof") != "3" || fallback.Container.Metadata.Values["temp-url-key-2"] != "" || string(accountOnly.Key) != "account-only" || account.Header.Get("X-Proof") != "0" || container.Header.Get("X-Proof") != "1" {
		t.Fatal("workflow evidence aliases")
	}
}
