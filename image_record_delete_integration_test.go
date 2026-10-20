package openstack_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionOwnedImageDeletionSeparatesWholeRecordFromStoreAcknowledgement(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id, storeID = "fixed/한글:%?", "store /标签:%?"
	cloud := "delete-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"project"`)}}
	location, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("delete-0")
	calls := 0
	const fetch = `{"id":"fixed/한글:%?","name":"fetched","protected":"false","size":"04","properties":{"exact":900719925474099312345}}`
	bodies := []string{fetch, "whole opaque\xff", `{"id":"passive store response"}`, "literal opaque"}
	codes := []int{201, 203, 299, 204}
	methods := []string{http.MethodGet, http.MethodDelete, http.MethodDelete, http.MethodDelete}
	paths := []string{"images/" + url.PathEscape(id), "images/" + url.PathEscape(id), "stores/" + url.PathEscape(storeID) + "/" + url.PathEscape(id), "images/literal"}
	var source *gophercloud.ServiceClient
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		if index >= len(methods) || req.Method != methods[index] || req.URL.String() != endpoint+paths[index] || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Source") != fmt.Sprintf("source-%d", index) || req.Header.Get("X-Auth-Token") != fmt.Sprintf("delete-%d", index) || req.Header.Get("OpenStack-API-Version") != "image 2.10" || req.Header.Get("X-Call") != "delete-record" {
			t.Fatal(index, req.Method, req.URL, req.Header, req.Body)
		}
		calls++
		provider.SetToken(fmt.Sprintf("delete-%d", calls))
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		header := http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}}
		if index == 0 {
			header.Set("OpenStack-image-import-methods", "old, source")
		}
		if index == 1 {
			header.Set("OpenStack-image-import-methods", "new,, raw")
		}
		return &http.Response{Request: req, StatusCode: codes[index], Header: header, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	upper, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source = versioned.RawClient()
	source.MoreHeaders, source.Microversion = map[string]string{"X-Source": "source-0"}, "2.10"
	if upper.RawClient() != source || upper.API.RawClient() != source || source.ProviderClient != provider || calls != 0 {
		t.Fatal("owned deletion replaced shared source")
	}
	seed, err := upper.GetImageRecord(context.Background(), image.ImageRecordRequest{ID: id}, image.WithImageRecordHeader("X-Call", "delete-record"))
	th.AssertNoErr(t, err)
	seed.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
	whole, err := conn.DeleteImageRecord(context.Background(), image.ImageRecordDeleteRequest{Record: seed}, image.WithImageRecordDeleteHeader("X-Call", "delete-record"))
	th.AssertNoErr(t, err)
	if whole == nil || whole.Record == nil || whole.Acknowledgement == nil || whole.Acknowledgement.ImageID != id || whole.Acknowledgement.StoreID != "" || whole.Acknowledgement.StatusCode != 203 || !bytes.Equal(whole.Acknowledgement.Body, []byte(bodies[1])) || whole.Record.StatusCode != 201 || whole.Record.Header.Get("X-Proof") != "response-0" || string(whole.Record.Envelope) != fetch || string(whole.Record.Resource.Body["id"]) != `"fixed/한글:%?"` || string(whole.Record.Resource.Body["size"]) != `4` || string(whole.Record.Resource.Body["is_protected"]) != `true` || string(whole.Record.Resource.Body["location"]) != string(location) {
		t.Fatal(whole)
	}
	th.CheckDeepEquals(t, []string{"new", "", " raw"}, whole.Record.ImportMethods)
	store := &serviceinfo.StoreRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"store /标签:%?"`)}}}}
	stored, err := conn.DeleteImageRecord(context.Background(), image.ImageRecordDeleteRequest{Record: whole.Record}, image.WithImageRecordDeleteStoreRecord(store), image.WithImageRecordDeleteHeader("X-Call", "delete-record"))
	th.AssertNoErr(t, err)
	if stored == nil || stored.Record != nil || stored.Acknowledgement == nil || stored.Acknowledgement.ImageID != id || stored.Acknowledgement.StoreID != storeID || stored.Acknowledgement.StatusCode != 299 || string(stored.Acknowledgement.Body) != bodies[2] {
		t.Fatal(stored)
	}
	literal, err := upper.DeleteImageRecord(context.Background(), image.ImageRecordDeleteRequest{ID: "literal"}, image.WithImageRecordDeleteHeader("X-Call", "delete-record"))
	th.AssertNoErr(t, err)
	if literal == nil || literal.Record == nil || literal.Acknowledgement == nil || literal.Record.StatusCode != 0 || literal.Record.Wire != nil || literal.Record.Envelope != nil || literal.Acknowledgement.StatusCode != 204 || calls != 4 || string(literal.Record.Resource.Body["location"]) != string(location) {
		t.Fatal(literal, calls)
	}
	whole.Record.Resource.Body["name"][1] = 'X'
	whole.Record.Wire.Body["name"][1] = 'X'
	whole.Record.Header.Set("X-Proof", "caller changed")
	whole.Record.Envelope[0] = '!'
	whole.Acknowledgement.Body[0] = '!'
	whole.Acknowledgement.Header.Set("X-Proof", "caller changed")
	if string(seed.Resource.Body["name"]) != `"fetched"` || string(seed.Wire.Body["name"]) != `"fetched"` || seed.Header.Get("X-Proof") != "response-0" || string(seed.Envelope) != fetch || stored.Acknowledgement.Header.Get("X-Proof") != "response-2" || source.MoreHeaders["X-Source"] != "source-4" {
		t.Fatal("deletion results alias seed, another receipt or shared source")
	}
}
