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
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionImageMemberRecordsShareLocationAndFallback(t *testing.T) {
	const base = "https://cloud.test/reverse/glance/v2/"
	cloud := "member-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	location, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("member-0")
	calls := 0
	bodies := []string{
		`{"members":[{"member":"first","image_id":"foreign","location":{"foreign":true}}],"next":"/v2/images/parent/members?marker=second"}`,
		`{"members":{"id":null,"member":"second","name":"display"},"next":null}`,
		`{"message":"forbidden direct member"}`,
		`{"members":[{"member":"passive-id","name":"member target","self":"https://foreign.test/member"}],"next":"/v2/images/parent/members?marker=tail"}`,
		`{"members":[{"member":"other"}],"next":null}`,
	}
	codes := []int{203, 206, 403, 201, 200}
	paths := []string{"images/parent/members", "images/parent/members?marker=second", "images/parent/members/member%20target", "images/parent/members", "images/parent/members?marker=tail"}
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		if calls >= len(bodies) {
			t.Fatalf("unexpected request %s", req.URL)
		}
		index := calls
		if req.Method != http.MethodGet || req.URL.String() != base+paths[index] || req.Body != nil || req.Header.Get("X-Auth-Token") != fmt.Sprintf("member-%d", index) || req.Header.Get("X-Source") != "shared" || req.Header.Get("X-Call") != "records" || req.Header.Get("Accept") != "application/json" {
			t.Fatalf("request%d %s %s headers=%v", index, req.Method, req.URL, req.Header)
		}
		calls++
		provider.SetToken(fmt.Sprintf("member-%d", calls))
		return &http.Response{StatusCode: codes[index], Header: http.Header{"X-Proof": {fmt.Sprintf("page-%d", index)}}, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", base), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	cloud = "caller changed"
	upper, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	client := versioned.RawClient()
	client.MoreHeaders = map[string]string{"X-Source": "shared"}
	if upper.RawClient() != client || upper.API.RawClient() != client || upper.API.Members.RawClient() != client || client.ProviderClient != provider || calls != 0 {
		t.Fatal("member records replaced the shared client or made eager requests")
	}
	parent := resource.ID("parent")
	headers := map[string]string{"X-Call": "records"}
	listHeader := image.WithImageMemberRecordListHeaders(headers)
	findHeader := image.WithFindImageMemberRecordHeaders(headers)
	headers["X-Call"] = "caller changed"
	rows, err := upper.AllImageMemberRecords(context.Background(), parent, listHeader)
	th.AssertNoErr(t, err)
	if len(rows) != 2 || calls != 2 || string(rows[0].Resource.Body["id"]) != `"first"` || string(rows[1].Resource.Body["id"]) != "null" {
		t.Fatalf("rows=%+v calls=%d", rows, calls)
	}
	for index, record := range rows {
		if record.ImageID == nil || *record.ImageID != "parent" || len(record.Resource.Body) != 9 || string(record.Resource.Body["image_id"]) != `"parent"` || string(record.Resource.Body["location"]) != string(location) || record.StatusCode != codes[index] || string(record.Envelope) != bodies[index] {
			t.Fatalf("record%d=%+v", index, record)
		}
	}
	if string(rows[0].Wire.Body["image_id"]) != `"foreign"` || string(rows[0].Wire.Body["location"]) != `{"foreign":true}` {
		t.Fatal("wire was overwritten by fixed scope or location")
	}
	found, err := upper.FindImageMemberRecord(context.Background(), parent, "member target", findHeader)
	th.AssertNoErr(t, err)
	if found == nil || found.StatusCode != 201 || calls != 5 || string(found.Resource.Body["id"]) != `"passive-id"` || string(found.Resource.Body["location"]) != string(location) || string(found.Wire.Body["self"]) != `"https://foreign.test/member"` {
		t.Fatalf("found=%+v calls=%d", found, calls)
	}
	rows[0].Resource.Body["location"][0] = '!'
	rows[0].Header.Set("X-Proof", "caller")
	if string(found.Resource.Body["location"]) != string(location) || rows[1].Header.Get("X-Proof") != "page-1" || found.Header.Get("X-Proof") != "page-3" || client.MoreHeaders["X-Source"] != "shared" {
		t.Fatal("records share caller-mutated evidence or source headers changed")
	}
}
