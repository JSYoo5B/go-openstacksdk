package openstack_test

import (
	"bytes"
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

func TestConnectionOwnedImageMemberFamilySharesSourceLocationAndActualReceipts(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	cloud := "member-mutation-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"project"`)}}
	location, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("token-0")
	calls := 0
	methods := []string{http.MethodPost, http.MethodGet, http.MethodPut, http.MethodDelete}
	paths := []string{"images/parent/members", "images/parent/members/returned-add", "images/parent/members/returned-get", "images/parent/members/returned-get"}
	bodies := []string{`{"member":"returned-add","created_at":false}`, `{"member":"returned-get","status":[null,false],"image_id":"foreign","location":{"foreign":true}}`, "opaque update", "opaque\xff"}
	codes := []int{201, 203, 202, 299}
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		if index >= len(methods) || req.Method != methods[index] || req.URL.String() != endpoint+paths[index] || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "shared" || req.Header.Get("X-Auth-Token") != fmt.Sprintf("token-%d", index) || req.Header.Get("OpenStack-API-Version") != "image 2.10" || req.Header.Get("X-Call") != "member-family" {
			t.Fatal(index, req.Method, req.URL, req.Header)
		}
		if index == 0 || index == 2 {
			var actual map[string]json.RawMessage
			th.AssertNoErr(t, json.NewDecoder(req.Body).Decode(&actual))
			want := map[string]json.RawMessage{"member": json.RawMessage(`"submitted"`), "status": json.RawMessage(`"future"`)}
			if index == 2 {
				want = map[string]json.RawMessage{"member": json.RawMessage(`"returned-get"`), "status": json.RawMessage(`null`)}
			}
			th.CheckDeepEquals(t, want, actual)
		} else if req.Body != nil {
			t.Fatal("member GET/DELETE has a body", req.Body)
		}
		calls++
		provider.SetToken(fmt.Sprintf("token-%d", calls))
		return &http.Response{Request: req, StatusCode: codes[index], Header: http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}}, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	upper, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source := versioned.RawClient()
	source.MoreHeaders, source.Microversion = map[string]string{"X-Source": "shared"}, "2.10"
	if upper.RawClient() != source || upper.API.RawClient() != source || upper.API.Members.RawClient() != source || source.ProviderClient != provider || calls != 0 {
		t.Fatal("member family does not share connection binding")
	}
	parent := resource.ID("parent")
	added, err := conn.AddImageMemberRecord(context.Background(), parent, image.WithImageMemberRecordMemberID("submitted"), image.WithImageMemberRecordStatus("future"), image.WithImageMemberRecordWriteHeader("X-Call", "member-family"))
	th.AssertNoErr(t, err)
	if added == nil || string(added.Resource.Body["id"]) != `"returned-add"` || string(added.Resource.Body["status"]) != `"future"` {
		t.Fatal(added)
	}
	fetched, err := conn.GetImageMemberRecord(context.Background(), parent, image.ImageMemberRecordRequest{Record: added}, image.WithImageMemberHeader("X-Call", "member-family"))
	th.AssertNoErr(t, err)
	if fetched == nil || string(fetched.Resource.Body["id"]) != `"returned-get"` || string(fetched.Resource.Body["created_at"]) != `null` || string(fetched.Wire.Body["image_id"]) != `"foreign"` {
		t.Fatal(fetched)
	}
	updated, err := conn.UpdateImageMemberRecord(context.Background(), parent, image.ImageMemberRecordRequest{Record: fetched}, image.WithImageMemberRecordStatus(nil), image.WithImageMemberRecordWriteHeader("X-Call", "member-family"))
	th.AssertNoErr(t, err)
	if updated == nil || updated.Wire != nil || string(updated.Resource.Body["id"]) != `"returned-get"` || string(updated.Resource.Body["status"]) != `null` {
		t.Fatal(updated)
	}
	ack, err := conn.RemoveImageMemberRecord(context.Background(), parent, image.ImageMemberRecordRequest{Record: updated}, image.WithRemoveImageMemberHeader("X-Call", "member-family"))
	th.AssertNoErr(t, err)
	if ack == nil || ack.ImageID != "parent" || ack.MemberID != "returned-get" || ack.StatusCode != 299 || !bytes.Equal(ack.Body, []byte(bodies[3])) || ack.Header.Get("X-Proof") != "response-3" || calls != 4 {
		t.Fatal(ack, calls)
	}
	for index, record := range []*image.ImageMemberRecord{added, fetched, updated} {
		if record.ImageID == nil || *record.ImageID != "parent" || record.StatusCode != codes[index] || record.Header.Get("X-Proof") != fmt.Sprintf("response-%d", index) || string(record.Envelope) != bodies[index] || len(record.Resource.Body) != 9 || string(record.Resource.Body["image_id"]) != `"parent"` || string(record.Resource.Body["location"]) != string(location) {
			t.Fatal(index, record)
		}
	}
	updated.Resource.Body["member_id"][1] = 'X'
	updated.Header.Set("X-Proof", "caller changed")
	updated.Envelope[0] = '!'
	ack.Header.Set("X-Proof", "caller changed")
	ack.Body[0] = '!'
	if string(fetched.Resource.Body["member_id"]) != `"returned-get"` || fetched.Header.Get("X-Proof") != "response-1" || added.Header.Get("X-Proof") != "response-0" || string(added.Envelope) != bodies[0] || source.MoreHeaders["X-Source"] != "shared" {
		t.Fatal("member results alias a prior operation or source")
	}
}
