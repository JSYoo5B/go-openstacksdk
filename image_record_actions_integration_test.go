package openstack_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func TestConnectionOwnedImageActionsPreserveRecordAndSeparateReceipts(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "image /한글:%?"
	const fetch = `{"id":"image /한글:%?","name":"before","status":"queued","properties":{"exact":900719925474099312345}}`
	cloud := "action-cloud"
	facts := resource.CloudLocation{Cloud: &cloud}
	location, err := facts.ForResource(nil, nil)
	th.AssertNoErr(t, err)
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("action-0")
	calls := 0
	var source *gophercloud.ServiceClient
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		paths := []string{"images/" + url.PathEscape(id), "images/" + url.PathEscape(id) + "/actions/deactivate", "images/" + url.PathEscape(id) + "/actions/reactivate"}
		if index >= len(paths) || req.URL.String() != endpoint+paths[index] || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Source") != fmt.Sprintf("source-%d", index) || req.Header.Get("X-Auth-Token") != fmt.Sprintf("action-%d", index) || req.Header.Get("OpenStack-API-Version") != "image 2.10" || req.Header.Get("X-Call") != "record-action" {
			t.Fatal(index, req.Method, req.URL, req.Header, req.Body)
		}
		if index == 0 && req.Method != http.MethodGet || index > 0 && req.Method != http.MethodPost {
			t.Fatal(req.Method)
		}
		calls++
		provider.SetToken(fmt.Sprintf("action-%d", calls))
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		header := http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}}
		header.Set("OpenStack-Image-Import-Methods", "ignored action header")
		body, status := "opaque\xff", 299
		if index == 0 {
			body, status = fetch, 201
			header.Set("OpenStack-Image-Import-Methods", "old,source")
		}
		return &http.Response{Request: req, StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	th.AssertNoErr(t, err)
	upper, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source = versioned.RawClient()
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	source.Microversion = "2.10"
	if upper.RawClient() != source || upper.API.RawClient() != source || calls != 0 {
		t.Fatal("actions replaced shared source")
	}
	seed, err := upper.GetImageRecord(context.Background(), image.ImageRecordRequest{ID: id}, image.WithImageRecordHeader("X-Call", "record-action"))
	th.AssertNoErr(t, err)
	seed.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
	deactivated, err := conn.DeactivateImageRecord(context.Background(), image.ImageRecordActionRequest{Record: seed}, image.WithImageRecordActionHeader("X-Call", "record-action"))
	th.AssertNoErr(t, err)
	if deactivated == nil || deactivated.Record == nil || deactivated.Acknowledgement == nil {
		t.Fatal(deactivated)
	}
	reactivated, err := conn.ReactivateImageRecord(context.Background(), image.ImageRecordActionRequest{Record: deactivated.Record}, image.WithImageRecordActionHeader("X-Call", "record-action"))
	th.AssertNoErr(t, err)
	for index, result := range []*image.ImageRecordActionResult{deactivated, reactivated} {
		if result == nil || result.Record == nil || result.Acknowledgement == nil {
			t.Fatal(result)
		}
		record, ack := result.Record, result.Acknowledgement
		if ack.ImageID != id || ack.StatusCode != 299 || !bytes.Equal(ack.Body, []byte("opaque\xff")) || ack.Header.Get("X-Proof") != fmt.Sprintf("response-%d", index+1) || record.StatusCode != 201 || record.Header.Get("X-Proof") != "response-0" || string(record.Envelope) != fetch || string(record.Resource.Body["status"]) != `"queued"` || string(record.Resource.Body["location"]) != string(location) || len(record.ImportMethods) != 0 {
			t.Fatal(record, ack)
		}
	}
	if deactivated.Acknowledgement.Action != "deactivate" || reactivated.Acknowledgement.Action != "reactivate" || calls != 3 {
		t.Fatal(deactivated, reactivated, calls)
	}
	deactivated.Record.Resource.Body["name"][1] = 'X'
	deactivated.Record.Header.Set("X-Proof", "changed")
	deactivated.Record.Envelope[0] = '!'
	deactivated.Acknowledgement.Body[0] = '!'
	if string(seed.Resource.Body["name"]) != `"before"` || seed.Header.Get("X-Proof") != "response-0" || string(seed.Envelope) != fetch || string(reactivated.Record.Resource.Body["name"]) != `"before"` || string(reactivated.Record.Envelope) != fetch || string(reactivated.Acknowledgement.Body) != "opaque\xff" {
		t.Fatal("action result aliases input or subsequent result")
	}
}

func TestConnectionOwnedImageActionsRejectInvalidInvocation(t *testing.T) {
	var conn *sdk.Connection
	for _, action := range []string{"deactivate", "reactivate"} {
		t.Run(action, func(t *testing.T) {
			var err error
			var result *image.ImageRecordActionResult
			if action == "deactivate" {
				result, err = conn.DeactivateImageRecord(context.Background(), image.ImageRecordActionRequest{ID: "fixed"})
			} else {
				result, err = conn.ReactivateImageRecord(context.Background(), image.ImageRecordActionRequest{ID: "fixed"})
			}
			var operation *resource.OperationError
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Resource != "image" || operation.Operation != strings.ToUpper(action[:1])+action[1:]+"ImageRecord" {
				t.Fatal(result, err, operation)
			}
		})
	}
}
