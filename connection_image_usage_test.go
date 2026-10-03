package gophercloudsdk_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"

	sdk "gophercloudsdk"
	"gophercloudsdk/image/v2/serviceinfo"
)

func TestConnectionImageUsageSharesServiceInfoAndCurrentProject(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("first-project-token")
	var calls int
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		wantToken, wantHeader := "first-project-token", "versioned"
		if calls == 2 {
			wantToken, wantHeader = "second-project-token", "upper"
		}
		if calls > 2 || r.Method != http.MethodGet || r.URL.String() != "https://cloud.test/reverse/image/v2/info/usage" || r.Body != nil || r.Header.Get("X-Auth-Token") != wantToken || r.Header.Get("X-Source") != "shared" || r.Header.Get("X-Call") != wantHeader {
			t.Fatalf("unexpected request %d: %s %s headers=%v body=%v", calls, r.Method, r.URL, r.Header, r.Body)
		}
		body := `{"usage":{"image_size_total":{"limit":9007199254740993,"usage":0,"vendor":{"precise":9007199254740995}}},"vendor":9007199254740997}`
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Proof": {"actual"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/reverse/image/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	versioned, err := conn.ImageV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := conn.Image(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client := versioned.RawClient()
	client.MoreHeaders = map[string]string{"X-Source": "shared"}
	if upper.RawClient() != client || versioned.ServiceInfo.RawClient() != client || upper.API.ServiceInfo.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("usage discovery replaced the cached authenticated client")
	}
	check := func(info *serviceinfo.UsageInfo, err error) {
		t.Helper()
		if err != nil || info == nil || info.StatusCode != 200 || info.Header.Get("X-Proof") != "actual" || string(info.Body["vendor"]) != "9007199254740997" {
			t.Fatalf("usage=%+v err=%v", info, err)
		}
		metric := info.Usage["image_size_total"]
		if metric == nil || metric.Limit == nil || *metric.Limit != 9007199254740993 || metric.Usage == nil || *metric.Usage != 0 || string(metric.Body["vendor"]) != `{"precise":9007199254740995}` {
			t.Fatalf("metric=%+v", metric)
		}
	}
	first, err := versioned.ServiceInfo.GetUsageInfo(ctx, serviceinfo.WithGetUsageInfoHeader("X-Call", "versioned"))
	check(first, err)
	*first.Usage["image_size_total"].Limit = 1
	first.Body["vendor"][0] = '0'
	provider.SetToken("second-project-token")
	second, err := upper.API.ServiceInfo.GetUsageInfo(ctx, serviceinfo.WithGetUsageInfoHeader("X-Call", "upper"))
	check(second, err)
	if calls != 2 || client.ProviderClient != provider {
		t.Fatalf("requests=%d provider=%p", calls, client.ProviderClient)
	}
}
