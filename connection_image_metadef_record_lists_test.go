package openstack_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	ns "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionImageMetadefRecordListsShareLocationAndPaging(t *testing.T) {
	for _, versioned := range []bool{false, true} {
		label := "Image"
		if versioned {
			label = "ImageV2"
		}
		t.Run(label, func(t *testing.T) {
			const base = "https://cloud.test/reverse/glance/v2/"
			const parent = "OS::SDK::공유"
			const objectPath = "metadefs/namespaces/OS::SDK::%EA%B3%B5%EC%9C%A0/objects"
			cloud := "owned-cloud"
			facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
			wantLocation, err := facts.ForResource(nil, nil)
			th.AssertNoErr(t, err)
			provider := &gophercloud.ProviderClient{}
			provider.UseTokenLock()
			provider.SetToken("record-0")
			calls, retries := 0, 0
			objectFirst := `{"objects":[{"name":"first","properties":9007199254740993,"required":false,"namespace_name":"foreign","location":{"foreign":true},"self":"https://foreign.test/ignored"}],"next":"/v2/` + objectPath + `?marker=second"}`
			objectSecond := `{"objects":{"id":null,"name":"second","created_at":false,"required":[null],"precision":9007199254740993},"next":null}`
			namespaceFirst := `{"namespaces":{"id":null,"namespace":"OS::API","protected":false,"owner":"foreign-project","resource_type_associations":4,"tags":"t","location":{"foreign":true},"precision":9007199254740993}}`
			namespaceSecond := `{"namespaces":[{"namespace":"OS::Next","protected":0,"resource_type_associations":[{"name":"raw"}],"tags":null}],"next":null}`
			bodies := []string{objectFirst, objectSecond, namespaceFirst, namespaceSecond}
			codes := []int{203, 200, 201, 206}
			provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
				if calls >= len(bodies) {
					t.Fatalf("extra request %s", req.URL)
				}
				index := calls
				expected := base + objectPath
				switch index {
				case 1:
					expected += "?marker=second"
				case 2:
					expected = base + "metadefs/namespaces?" + url.Values{"limit": {"1"}, "visibility": {"public"}}.Encode()
				case 3:
					expected = base + "metadefs/namespaces?" + url.Values{"limit": {"1"}, "marker": {"OS::Next"}, "visibility": {"public"}}.Encode()
				}
				if req.Method != http.MethodGet || req.URL.String() != expected || req.Body != nil || req.Header.Get("X-Auth-Token") != fmt.Sprintf("record-%d", index) || req.Header.Get("X-Source") != "shared" || req.Header.Get("Accept") != "application/json" {
					t.Fatalf("request%d %s %s headers=%v expected=%s", index, req.Method, req.URL, req.Header, expected)
				}
				if index >= 2 && req.Header.Get("X-Call") != "snapshot" {
					t.Fatal("record header snapshot lost")
				}
				header := http.Header{"X-Proof": {fmt.Sprintf("page-%d", index)}}
				if index == 0 {
					header.Set("Link", `</v2/`+objectPath+`?marker=second>; rel="next"`)
				}
				if index == 2 {
					header.Set("Link", `</v2/metadefs/namespaces?marker=OS%3A%3ANext>; rel="next"`)
				}
				calls++
				provider.SetToken(fmt.Sprintf("record-%d", calls))
				return &http.Response{StatusCode: codes[index], Header: header, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
			})
			conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", base), sdk.WithCloudLocation(facts))
			th.AssertNoErr(t, err)
			cloud = "caller mutation"
			upper, err := conn.Image(context.Background())
			th.AssertNoErr(t, err)
			version, err := conn.ImageV2(context.Background())
			th.AssertNoErr(t, err)
			api := upper.API
			if versioned {
				api = version
			}
			native := version.RawClient()
			native.MoreHeaders = map[string]string{"X-Source": "shared"}
			native.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return fmt.Errorf("unexpected native retry")
			}
			if api.MetadefObjects.RawClient() != native || api.MetadefNamespaces.RawClient() != native || native.ProviderClient != provider || upper.RawClient() != native || calls != 0 {
				t.Fatal("owned aggregate replaced shared client or made discovery calls")
			}
			scope, err := api.MetadefObjects.InNamespace(context.Background(), parent)
			th.AssertNoErr(t, err)
			objects, err := scope.AllRecords(context.Background())
			th.AssertNoErr(t, err)
			if len(objects) != 2 || calls != 2 {
				t.Fatalf("object pages=%d calls=%d", len(objects), calls)
			}
			for i, record := range objects {
				if record.Namespace == nil || *record.Namespace != parent || len(record.Resource.Body) != 9 || string(record.Resource.Body["namespace_name"]) != `"OS::SDK::공유"` || string(record.Resource.Body["location"]) != string(wantLocation) || record.StatusCode != codes[i] || record.Header.Get("X-Proof") != fmt.Sprintf("page-%d", i) || string(record.Envelope) != bodies[i] {
					t.Fatalf("object%d record=%+v", i, record)
				}
			}
			if string(objects[0].Resource.Body["id"]) != `"first"` || string(objects[0].Resource.Body["properties"]) != "9007199254740993" || string(objects[0].Wire.Body["namespace_name"]) != `"foreign"` || string(objects[1].Resource.Body["id"]) != "null" {
				t.Fatal("object identity/raw values were replaced")
			}
			headers := map[string]string{"X-Call": "snapshot"}
			ownedHeaders := ns.WithRecordListHeaders(headers)
			headers["X-Call"] = "caller mutation"
			namespaces, err := api.MetadefNamespaces.AllRecords(context.Background(), ownedHeaders, ns.WithRecordListLimit(1), ns.WithRecordListMaxItems(2), ns.WithRecordListFilter("visibility", "public"), ns.WithRecordListFilter("is_protected", false))
			th.AssertNoErr(t, err)
			if len(namespaces) != 2 || calls != 4 || retries != 0 {
				t.Fatalf("namespace pages=%d calls=%d retries=%d", len(namespaces), calls, retries)
			}
			for i, record := range namespaces {
				if len(record.Resource.Body) != 13 || string(record.Resource.Body["location"]) != string(wantLocation) || string(record.Resource.Body["is_protected"]) != "false" || record.StatusCode != codes[i+2] || record.Header.Get("X-Proof") != fmt.Sprintf("page-%d", i+2) || string(record.Envelope) != bodies[i+2] {
					t.Fatalf("namespace%d record=%+v", i, record)
				}
			}
			if string(namespaces[0].Resource.Body["id"]) != "null" || string(namespaces[1].Resource.Body["id"]) != `"OS::Next"` || string(namespaces[0].Resource.Body["resource_type_associations"]) != `[{}]` || string(namespaces[0].Resource.Body["tags"]) != `["t"]` || string(namespaces[1].Resource.Body["tags"]) != "null" || string(namespaces[0].Wire.Body["resource_type_associations"]) != "4" {
				t.Fatal("namespace descriptor/raw identity lost")
			}
			namespaces[0].Header.Set("X-Proof", "mutated")
			namespaces[0].Resource.Body["owner"][1] = 'X'
			namespaces[0].Envelope[0] = 'x'
			if namespaces[0].Wire.Header.Get("X-Proof") != "page-2" || string(namespaces[0].Wire.Body["owner"]) != `"foreign-project"` || namespaces[1].Header.Get("X-Proof") != "page-3" {
				t.Fatal("record proof channels alias")
			}
		})
	}
}
