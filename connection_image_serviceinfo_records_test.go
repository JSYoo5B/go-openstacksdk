package openstack_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionImageServiceInfoRecordsShareLocationAndNativeClient(t *testing.T) {
	for _, versioned := range []bool{false, true} {
		label := "Image"
		if versioned {
			label = "ImageV2"
		}
		t.Run(label, func(t *testing.T) {
			const base = "https://cloud.test/reverse/glance/v2/"
			cloud := "owned-cloud"
			facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
			location, err := facts.ForResource(nil, nil)
			th.AssertNoErr(t, err)
			provider := &gophercloud.ProviderClient{}
			provider.UseTokenLock()
			provider.SetToken("info-0")
			bodies := []string{
				`{"import-methods":{"value":[null,4],"extension":9007199254740993},"location":{"foreign":true},"vendor":1e400}`,
				`{"stores":[{"id":"store-a","default":"false","properties":false,"location":{"foreign":true},"type":"file"}],"next":"/v2/info/stores?marker=store-b"}`,
				`{"stores":{"name":"named-only","default":0,"properties":{"large":9007199254740993},"location":{"foreign":true}},"next":false}`,
			}
			codes := []int{201, 203, 206}
			calls := 0
			provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
				if calls >= len(bodies) {
					t.Fatal("unexpected discovery request", req.URL)
				}
				index := calls
				urls := []string{base + "info/import", base + "info/stores?limit=1", base + "info/stores?limit=1&marker=store-b"}
				if req.Method != http.MethodGet || req.URL.String() != urls[index] || req.Body != nil || req.Header.Get("X-Auth-Token") != fmt.Sprintf("info-%d", index) || req.Header.Get("X-Source") != "shared" || req.Header.Get("X-Call") != "captured" || req.Header.Get("Accept") != "application/json" {
					t.Fatalf("request%d %s %s headers=%v", index, req.Method, req.URL, req.Header)
				}
				calls++
				provider.SetToken(fmt.Sprintf("info-%d", calls))
				return &http.Response{StatusCode: codes[index], Header: http.Header{"X-Proof": {fmt.Sprintf("page-%d", index)}}, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
			})
			conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", base), sdk.WithCloudLocation(facts))
			th.AssertNoErr(t, err)
			cloud = "caller mutation"
			upper, err := conn.Image(context.Background())
			th.AssertNoErr(t, err)
			version, err := conn.ImageV2(context.Background())
			th.AssertNoErr(t, err)
			api := upper.API.ServiceInfo
			if versioned {
				api = version.ServiceInfo
			}
			native := version.RawClient()
			native.MoreHeaders = map[string]string{"X-Source": "shared"}
			if api.RawClient() != native || upper.API.ServiceInfo.RawClient() != native || native.ProviderClient != provider || calls != 0 {
				t.Fatal("owned discovery replaced native client or made an extra request")
			}
			headers := map[string]string{"X-Call": "captured"}
			importHeader := serviceinfo.WithImportRecordHeaders(headers)
			storeHeader := serviceinfo.WithStoreRecordListHeaders(headers)
			headers["X-Call"] = "caller mutation"
			info, err := api.GetImportInfoRecord(context.Background(), importHeader)
			th.AssertNoErr(t, err)
			if info == nil || info.Resource == nil || info.Wire == nil || len(info.Resource.Body) != 4 || info.StatusCode != 201 || string(info.Resource.Body["location"]) != string(location) || string(info.Resource.Body["id"]) != "null" || string(info.Resource.Body["import_methods"]) != `{"value":[null,4],"extension":9007199254740993}` || string(info.Wire.Body["location"]) != `{"foreign":true}` || string(info.Wire.Body["vendor"]) != "1e400" || string(info.Envelope) != bodies[0] {
				t.Fatal("import declared view/provenance", info)
			}
			rows, err := api.AllStoreRecords(context.Background(), storeHeader, serviceinfo.WithStoreRecordListLimit(1), serviceinfo.WithStoreRecordListMaxItems(2))
			th.AssertNoErr(t, err)
			if len(rows) != 2 || calls != 3 {
				t.Fatal(rows, calls)
			}
			for i, row := range rows {
				if row.Resource == nil || row.Wire == nil || len(row.Resource.Body) != 6 || row.StatusCode != codes[i+1] || row.Header.Get("X-Proof") != fmt.Sprintf("page-%d", i+1) || string(row.Resource.Body["location"]) != string(location) || string(row.Envelope) != bodies[i+1] {
					t.Fatal("store view/receipt", i, row)
				}
			}
			if string(rows[0].Resource.Body["is_default"]) != "true" || string(rows[0].Resource.Body["properties"]) != "{}" || string(rows[1].Resource.Body["id"]) != "null" || string(rows[1].Resource.Body["is_default"]) != "false" {
				t.Fatal("Source descriptor or inherited ID changed", rows)
			}
			if _, exists := rows[0].Resource.Body["type"]; exists {
				t.Fatal("detail field became declared Body")
			}
			info.Resource.Body["import_methods"][0] = '!'
			info.Header.Set("X-Proof", "changed")
			rows[0].Envelope[0] = '!'
			rows[0].Resource.Header.Set("X-Proof", "changed")
			if string(info.Wire.Body["import-methods"]) != `{"value":[null,4],"extension":9007199254740993}` || info.Wire.Header.Get("X-Proof") != "page-0" || rows[0].Wire.Header.Get("X-Proof") != "page-1" || string(rows[1].Envelope) != bodies[2] {
				t.Fatal("proof channels alias")
			}
		})
	}
}
