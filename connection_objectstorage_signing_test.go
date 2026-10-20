package openstack_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	swift "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionSwiftSigningSharesClientAndCapturesEachWorkflow(t *testing.T) {
	ctx := context.Background()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("token-0")
	const endpoint = "https://cloud.test/v1/AUTH_account%25/"
	const container = "백업 %2F?#"
	const base = "https://cloud.test/data%25/v1/AUTH_account%25/"
	const path = "/v1/AUTH_account%/bucket/dir/% ?#한"
	stamp := time.Unix(1700000000, 0)
	var source *gophercloud.ServiceClient
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		target := endpoint
		if n == 0 {
			target = base + url.PathEscape(container)
		}
		captured := "initial"
		if n == 2 {
			captured = "after-form"
		}
		if r.Method != http.MethodHead || r.URL.String() != target || r.Header.Get("X-Source") != captured || r.Header.Get("X-Auth-Token") != fmt.Sprintf("token-%d", n+1) {
			t.Fatal(n, r.Method, r.URL, r.Header)
		}
		if n < 2 {
			if r.Header.Get("X-Call") != "form" || r.Header.Get("X-Newest") != "false" {
				t.Fatal(r.Header)
			}
		} else if r.Header.Get("X-Call") != "url" || len(r.Header.Values("X-Newest")) != 0 {
			t.Fatal(r.Header)
		}
		h := http.Header{"X-Proof": {fmt.Sprint(n)}}
		if n == 0 {
			h.Set("X-Container-Meta-Temp-Url-Key-2", "")
		} else if n == 1 {
			h.Set("X-Account-Meta-Temp-Url-Key", "account-primary")
			h.Set("X-Account-Meta-Temp-Url-Key-2", "account-secondary")
		} else {
			h.Set("X-Account-Meta-Temp-Url-Key", "account-only")
		}
		source.MoreHeaders["X-Source"] = "after-form"
		provider.SetToken(fmt.Sprintf("token-%d", calls+1))
		return &http.Response{Request: r, StatusCode: 204, Header: h, Body: io.NopCloser(strings.NewReader("opaque"))}, nil
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
	if err != nil || versioned != service || service.RawClient().ProviderClient != provider || service.Accounts.RawClient() != service.RawClient() || service.Containers.RawClient() != service.RawClient() {
		t.Fatal("shared facade", err)
	}
	source = service.RawClient()
	source.ResourceBase = "invalid unused base"
	source.MoreHeaders = map[string]string{"X-Source": "initial"}
	key := []byte{'k', 255, 'y'}
	explicit, err := service.GenerateTempURL(ctx, path, 60, "gEt", swift.WithGenerateTempURLKey(key), swift.WithGenerateTempURLTimestamp(stamp))
	if err != nil || explicit == nil || explicit.Discovery != nil || explicit.Expires != 1700000060 || calls != 0 {
		t.Fatal(explicit, err, calls)
	}
	parsed, err := url.Parse(explicit.URL)
	if err != nil || parsed.Path != path || parsed.Query().Get("temp_url_sig") != explicit.Signature || parsed.Query().Get("temp_url_expires") != "1700000060" {
		t.Fatal(explicit, err)
	}
	input := swift.FormSignatureInput{ObjectPrefix: "dir/% ?#한", RedirectURL: "https://example.test/done", MaxFileSize: 123, MaxUploadCount: 3, Timeout: 60}
	localForm, err := service.GenerateFormSignature(ctx, container, input, swift.WithGenerateFormSignatureKey(key), swift.WithGenerateFormSignatureTimestamp(stamp))
	if err != nil || localForm == nil || localForm.Discovery != nil || calls != 0 || localForm.Path != "/v1/AUTH_account%/"+container+"/"+input.ObjectPrefix {
		t.Fatal(localForm, err, calls)
	}
	parsed, err = url.Parse(localForm.URL)
	if err != nil || parsed.Path != localForm.Path || parsed.Scheme != "https" || parsed.Host != "cloud.test" || parsed.RawQuery != "" {
		t.Fatal(localForm, err)
	}
	source.ResourceBase = base
	headers := map[string]string{"X-Call": "form"}
	formOption := swift.WithGenerateFormSignatureHeaders(headers)
	headers["X-Call"] = "caller mutation"
	form, err := service.GenerateFormSignature(ctx, container, input, formOption, swift.WithGenerateFormSignatureNewest(false), swift.WithGenerateFormSignatureTimestamp(stamp), func(o *swift.GenerateFormSignatureOpts) error {
		source.MoreHeaders["X-Source"] = "callback mutation"
		provider.SetToken("token-1")
		return nil
	})
	if err != nil || form == nil || form.Discovery == nil || form.Discovery.Container == nil || form.Discovery.Account == nil || form.Discovery.FromContainer || !form.Discovery.Secondary || string(form.Discovery.Key) != "account-secondary" || calls != 2 {
		t.Fatal(form, err, calls)
	}
	source.ResourceBase = "invalid unused base"
	generated, err := service.GenerateTempURL(ctx, path, 1700000060, "GET", swift.WithGenerateTempURLAbsolute(true), swift.WithGenerateTempURLTimestamp(stamp), swift.WithGenerateTempURLHeader("X-Call", "url"))
	if err != nil || generated == nil || generated.Discovery == nil || generated.Discovery.Container != nil || generated.Discovery.Account == nil || generated.Discovery.Secondary || string(generated.Discovery.Key) != "account-only" || calls != 3 {
		t.Fatal(generated, err, calls)
	}
	form.Discovery.Key[0] = '!'
	form.Discovery.Account.Header.Set("X-Proof", "changed")
	form.Discovery.Account.Metadata.Values["temp-url-key-2"] = "changed"
	if string(generated.Discovery.Key) != "account-only" || generated.Discovery.Account.Header.Get("X-Proof") != "2" || localForm.Discovery != nil || explicit.Discovery != nil || string(key) != string([]byte{'k', 255, 'y'}) {
		t.Fatal("result or caller key ownership")
	}
}
