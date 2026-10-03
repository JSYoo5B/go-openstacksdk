package gophercloudsdk_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/image/v2/imageimport"
	"gophercloudsdk/image/v2/serviceinfo"
)

type serviceInfoTransport func(*http.Request) (*http.Response, error)

func (f serviceInfoTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConnectionImageServiceInfoSharesClientPrefixAndLiveAuthentication(t *testing.T) {
	var calls int
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("initial")
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodGet || r.Header.Get("X-Auth-Token") != "updated" || r.Header.Get("X-Source") != "captured" || r.URL.Host != "cloud.test" || r.Body != nil {
			t.Fatalf("request=%s %s headers=%v body=%v", r.Method, r.URL, r.Header, r.Body)
		}
		var body string
		switch r.URL.Path {
		case "/reverse/image/v2/info/stores":
			body = `{"stores":[{"id":"backend","default":"false","properties":{"vendor":9007199254740993}}]}`
		case "/reverse/image/v2/info/stores/detail":
			body = `{"stores":[{"id":"backend","type":"file","read-only":"true","weight":1}]}`
		case "/reverse/image/v2/info/import":
			body = `{"import-methods":{"description":"available","type":"array","value":["glance-direct"]},"vendor":9007199254740993}`
		default:
			t.Fatalf("unexpected discovery request: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {"actual"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/reverse/image/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	version, err := conn.ImageV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := conn.Image(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client := version.RawClient()
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	if upper.RawClient() != client || version.ServiceInfo.RawClient() != client || upper.API.ServiceInfo.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("discovery replaced the cached authenticated client")
	}
	again, err := conn.ImageV2(ctx)
	if err != nil || again != version {
		t.Fatalf("version cache=%p/%p err=%v", version, again, err)
	}
	provider.SetToken("updated")
	stores, err := version.ServiceInfo.AllStores(ctx)
	if err != nil || len(stores) != 1 || stores[0].ID != "backend" || stores[0].IsDefault == nil || *stores[0].IsDefault || stores[0].Header.Get("X-Proof") != "actual" || stores[0].StatusCode != 200 || string(stores[0].Properties["vendor"]) != "9007199254740993" {
		t.Fatalf("stores=%+v err=%v", stores, err)
	}
	details, err := upper.API.ServiceInfo.AllStores(ctx, serviceinfo.WithListStoresDetails(true))
	if err != nil || len(details) != 1 || details[0].ReadOnly == nil || !*details[0].ReadOnly || details[0].Weight == nil || *details[0].Weight != 1 {
		t.Fatalf("details=%+v err=%v", details, err)
	}
	info, err := version.ServiceInfo.GetImportInfo(ctx)
	if err != nil || info == nil || info.ImportMethods == nil || !reflect.DeepEqual(info.ImportMethods.Value, []string{"glance-direct"}) || string(info.Body["vendor"]) != "9007199254740993" || info.Header.Get("X-Proof") != "actual" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	// Compile and execute the original generated ABI at the same native route.
	var nativeGet func(context.Context) (*imageimport.ImportInfo, error) = version.ImageImport.Get
	native, err := nativeGet(ctx)
	if err != nil || native == nil || !reflect.DeepEqual(native.ImportMethods.Value, []string{"glance-direct"}) || calls != 4 {
		t.Fatalf("native=%+v err=%v requests=%d", native, err, calls)
	}
}
