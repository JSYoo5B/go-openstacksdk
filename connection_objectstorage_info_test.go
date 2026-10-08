package openstack_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	swift "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionSwiftInfoSharesClientAndCapturesEachWorkflow(t *testing.T) {
	ctx := context.Background()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("initial")
	const endpoint = "https://cloud.test/proxy%25/v1/AUTH_account%25/"
	const target = "https://cloud.test/proxy%25/info"
	var source *gophercloud.ServiceClient
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		expectedSource := "initial"
		if calls > 1 {
			expectedSource = "later"
		}
		if r.Method != http.MethodGet || r.URL.String() != target || r.Body != nil || r.Header.Get("X-Source") != expectedSource || r.Header.Get("X-Auth-Token") != fmt.Sprintf("token-%d", calls) || r.Header.Get("X-Call") != "info" {
			t.Fatal(calls, r.Method, r.URL, r.Header)
		}
		source.MoreHeaders["X-Source"] = "later"
		body := `{"swift":{"max_file_size":100},"slo":{"min_segment_size":10},"plugin":{"precise":9007199254740993}}`
		return &http.Response{Request: r, StatusCode: 200, Header: http.Header{"X-Proof": {fmt.Sprint(calls)}}, Body: io.NopCloser(strings.NewReader(body))}, nil
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
	if err != nil || versioned != service || service.RawClient().ProviderClient != provider || service.Objects.RawClient() != service.RawClient() {
		t.Fatal("shared facade", err)
	}
	source = service.RawClient()
	source.ResourceBase = "invalid unused base"
	source.MoreHeaders = map[string]string{"X-Source": "initial"}
	headers := map[string]string{"X-Call": "info"}
	infoOption := swift.WithGetInfoHeaders(headers)
	segmentOption := swift.WithObjectSegmentSizeHeaders(headers)
	headers["X-Call"] = "mutated caller"
	info, err := service.GetInfo(ctx, infoOption, func(o *swift.GetInfoOpts) error {
		source.MoreHeaders["X-Source"] = "callback change"
		provider.SetToken("token-1")
		return nil
	})
	if err != nil || info == nil || info.StatusCode != 200 || info.Header.Get("X-Proof") != "1" || string(info.Swift["max_file_size"]) != "100" || string(info.Body["plugin"]) != `{"precise":9007199254740993}` {
		t.Fatal(info, err)
	}
	provider.SetToken("token-2")
	result, err := service.GetObjectSegmentSize(ctx, segmentOption, swift.WithObjectSegmentSize(0))
	if err != nil || result == nil || result.RequestedSize != 0 || result.Size != 10 || result.MaxFileSize != 100 || result.MinSegmentSize != 10 || result.UsedFallback || result.Info == nil || result.Header.Get("X-Proof") != "2" {
		t.Fatal(result, err)
	}
	provider.SetToken("token-3")
	reset, err := service.GetObjectSegmentSize(ctx, segmentOption, swift.WithObjectSegmentSize(0), swift.WithoutObjectSegmentSize())
	if err != nil || reset == nil || reset.RequestedSize != 1073741824 || reset.Size != 100 || calls != 3 {
		t.Fatal(reset, err, calls)
	}
	info.Swift["max_file_size"][0] = '9'
	result.Header.Set("X-Proof", "changed")
	if string(reset.Info.Swift["max_file_size"]) != "100" || reset.Header.Get("X-Proof") != "3" || reset.Info.Header.Get("X-Proof") != "3" || provider.Token() != "token-3" || source.ResourceBase != "invalid unused base" || headers["X-Call"] != "mutated caller" {
		t.Fatal("response/source ownership")
	}
}
