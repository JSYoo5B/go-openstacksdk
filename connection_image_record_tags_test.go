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
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionImageRecordTagsShareOwnedSourceAndMutationReceipts(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const identity, tag = "fixed/한글", "duplicate/标签"
	cloud := "tag-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	location, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("tag-0")
	calls := 0
	var source *gophercloud.ServiceClient
	seedBody := `{"id":"fixed/한글","status":"active","owner":"owner","tags":["duplicate/标签","keep"],"properties":{"large":900719925474099312345}}`
	bodies := []string{seedBody, "opaque\xff", `{"id":"response-decoy","tags":null}`, "literal-ack"}
	codes := []int{203, 299, 206, 201}
	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPut}
	paths := []string{"images/" + url.PathEscape(identity), "images/" + url.PathEscape(identity) + "/tags/" + url.PathEscape(tag), "images/" + url.PathEscape(identity) + "/tags/" + url.PathEscape(tag), "images/literal/tags/new"}
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		if index >= len(bodies) {
			t.Fatal("unexpected HTTP request", req.URL)
		}
		if req.Method != methods[index] || req.URL.String() != endpoint+paths[index] || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Auth-Token") != fmt.Sprintf("tag-%d", index) || req.Header.Get("X-Source") != fmt.Sprintf("source-%d", index) || req.Header.Get("X-Call") != "records" {
			t.Fatal(index, req.Method, req.URL, req.Header, req.Body)
		}
		calls++
		provider.SetToken(fmt.Sprintf("tag-%d", calls))
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		header := http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}}
		if index == 0 {
			header.Set("OpenStack-image-import-methods", "web-download, glance-direct")
		}
		return &http.Response{Request: req, StatusCode: codes[index], Header: header, Body: io.NopCloser(strings.NewReader(bodies[index]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source = versioned.RawClient()
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	if service.RawClient() != source || source.ProviderClient != provider || calls != 0 {
		t.Fatal("Connection service did not share source")
	}
	seed, err := service.GetImageRecord(context.Background(), image.ImageRecordRequest{ID: identity}, image.WithImageRecordHeader("X-Call", "records"))
	th.AssertNoErr(t, err)
	options := 0
	mutationOption := func(config *image.ImageMutationOpts) error {
		options++
		config.Headers = map[string]string{"X-Call": "records"}
		return nil
	}
	added, err := service.AddImageRecordTag(context.Background(), image.ImageRecordTagRequest{Record: seed}, tag, mutationOption)
	th.AssertNoErr(t, err)
	if added == nil || added.Record == nil || added.Acknowledgement == nil || added.Acknowledgement.StatusCode != 299 || !bytes.Equal(added.Acknowledgement.Body, []byte(bodies[1])) || added.Acknowledgement.ImageID != identity || added.Acknowledgement.Tag != tag || added.Acknowledgement.Header.Get("X-Proof") != "response-1" || string(added.Record.Resource.Body["tags"]) != `["duplicate/标签","keep","duplicate/标签"]` {
		t.Fatal(added)
	}
	removed, err := service.RemoveImageRecordTag(context.Background(), image.ImageRecordTagRequest{Record: added.Record}, tag, mutationOption)
	th.AssertNoErr(t, err)
	if removed == nil || removed.Record == nil || removed.Acknowledgement == nil || removed.Acknowledgement.StatusCode != 206 || string(removed.Acknowledgement.Body) != bodies[2] || string(removed.Record.Resource.Body["tags"]) != `["keep","duplicate/标签"]` || string(removed.Record.Resource.Body["id"]) != `"fixed/한글"` {
		t.Fatal(removed)
	}
	for _, record := range []*image.ImageRecord{seed, added.Record, removed.Record} {
		if record.StatusCode != 203 || record.Header.Get("X-Proof") != "response-0" || string(record.Envelope) != seedBody || string(record.Wire.Body["tags"]) != `["duplicate/标签","keep"]` || string(record.Resource.Body["location"]) != string(location) || len(record.ImportMethods) != 2 || len(record.Resource.Body) != 65 {
			t.Fatal("mutation overwrote fetched receipt or declared state", record)
		}
	}
	literal, err := service.AddImageRecordTag(context.Background(), image.ImageRecordTagRequest{ID: "literal"}, "new", mutationOption)
	th.AssertNoErr(t, err)
	if literal == nil || literal.Record == nil || literal.Acknowledgement == nil || literal.Acknowledgement.StatusCode != 201 || string(literal.Record.Resource.Body["tags"]) != `["new"]` || string(literal.Record.Resource.Body["location"]) != string(location) || literal.Record.StatusCode != 0 || literal.Record.Wire != nil || literal.Record.Envelope != nil || literal.Record.Header != nil || len(literal.Record.ImportMethods) != 0 || calls != 4 || options != 3 {
		t.Fatal("literal constructor fetched or invented image evidence", literal, calls, options)
	}
	added.Record.Resource.Body["tags"][0] = '!'
	added.Record.Envelope[0] = '!'
	added.Record.Wire.Body["tags"][0] = '!'
	added.Record.Header.Set("X-Proof", "caller")
	added.Record.ImportMethods[0] = "caller"
	removed.Acknowledgement.Body[0] = '!'
	removed.Acknowledgement.Header.Set("X-Proof", "caller")
	if string(seed.Resource.Body["tags"]) != `["duplicate/标签","keep"]` || string(seed.Envelope) != seedBody || seed.Header.Get("X-Proof") != "response-0" || seed.ImportMethods[0] != "web-download" || string(removed.Record.Envelope) != seedBody || string(removed.Record.Wire.Body["tags"]) != `["duplicate/标签","keep"]` || removed.Record.Header.Get("X-Proof") != "response-0" || literal.Acknowledgement.Header.Get("X-Proof") != "response-3" || string(literal.Acknowledgement.Body) != bodies[3] || source.MoreHeaders["X-Source"] != "source-4" {
		t.Fatal("mutation results alias the seed, each other or source")
	}
}
