package openstack_test

import (
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

func TestConnectionOwnedImagePropertiesSharesSourceResolverAndCommit(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "image /한글:%?"
	const kernelID = "kernel /한글"
	const fetched = `{"id":"image /한글:%?","name":"before","status":"queued","properties":{"old":"keep"}}`
	const saved = `{"id":"image /한글:%?","name":"before","status":"active","kernel_id":"kernel /한글","old":"keep","gpu":"True"}`
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("property-0")
	calls := 0
	var source *gophercloud.ServiceClient
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		expected := []string{endpoint + "images/" + url.PathEscape(id), endpoint + "images", endpoint + "images?marker=second", endpoint + "images/" + url.PathEscape(id)}
		sourceHeader := "source-1"
		if index == 0 {
			sourceHeader = "source-0"
		}
		if index >= len(expected) || req.URL.String() != expected[index] || req.Header.Get("X-Source") != sourceHeader || req.Header.Get("X-Auth-Token") != fmt.Sprintf("property-%d", index) || req.Header.Get("OpenStack-API-Version") != "image 2.10" || req.Header.Get("X-Call") != "property-helper" {
			t.Fatal(index, req.Method, req.URL, req.Header)
		}
		if index < 3 {
			if req.Method != http.MethodGet || req.Body != nil {
				t.Fatal(index, req.Method, req.Body)
			}
		} else {
			if req.Method != http.MethodPatch || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || req.Header.Get("Accept") != "" {
				t.Fatal(req.Method, req.Header)
			}
			var patch []struct {
				Op, Path string
				Value    json.RawMessage
			}
			th.AssertNoErr(t, json.NewDecoder(req.Body).Decode(&patch))
			values := map[string]string{}
			for _, op := range patch {
				if op.Op != "add" {
					t.Fatal(patch)
				}
				values[op.Path] = string(op.Value)
			}
			if len(patch) != 2 || values["/kernel_id"] != `"kernel /한글"` || values["/gpu"] != `"True"` {
				t.Fatal(patch)
			}
		}
		calls++
		provider.SetToken(fmt.Sprintf("property-%d", calls))
		source.MoreHeaders["X-Source"] = fmt.Sprintf("source-%d", calls)
		header := http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}}
		body, status := fetched, 201
		if index == 0 {
			header.Set("OpenStack-Image-Import-Methods", "old,source")
		}
		if index == 1 {
			body, status = `{"images":[{"id":"deleted","name":"kernel-deleted","status":"DELETED"},{"id":"kernel /한글","name":"kernel-first","status":"active"},{"id":"second","name":"kernel-second","status":"active"}],"next":"`+endpoint+`images?marker=second"}`, 299
		}
		if index == 2 {
			body, status = `{"images":[]}`, 200
		}
		if index == 3 {
			body, status = saved, 202
			header.Set("OpenStack-Image-Import-Methods", "response,method")
		}
		return &http.Response{Request: req, StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint))
	th.AssertNoErr(t, err)
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source = versioned.RawClient()
	source.MoreHeaders = map[string]string{"X-Source": "source-0"}
	source.Microversion = "2.10"
	if service.RawClient() != source || service.API.RawClient() != source || calls != 0 {
		t.Fatal("property helper replaced shared source")
	}
	seed, err := service.GetImageRecord(context.Background(), image.ImageRecordRequest{ID: id}, image.WithImageRecordHeader("X-Call", "property-helper"))
	th.AssertNoErr(t, err)
	seed.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
	result, err := conn.UpdateImagePropertiesRecord(context.Background(), image.ImageRecordPropertiesRequest{Record: seed}, image.WithImageRecordProperty("kernel", "kernel*"), image.WithImageRecordProperty("gpu", true), image.WithImageRecordPropertiesHeader("X-Call", "property-helper"))
	th.AssertNoErr(t, err)
	if result == nil || !result.Updated || result.Record == nil || calls != 4 {
		t.Fatal(result, calls)
	}
	record := result.Record
	if string(record.Resource.Body["id"]) != `"image /한글:%?"` || string(record.Resource.Body["kernel_id"]) != `"`+kernelID+`"` || string(record.Resource.Body["properties"]) != `{"gpu":"True","old":"keep"}` || string(record.Resource.Body["status"]) != `"active"` || record.StatusCode != 202 || record.Header.Get("X-Proof") != "response-3" || string(record.Envelope) != saved || len(record.ImportMethods) != 2 {
		t.Fatal(record)
	}
	// Existing nonempty properties make the Source helper return true even
	// with no new options. The shared Update dirty gate sends no extra PATCH.
	again, err := conn.UpdateImagePropertiesRecord(context.Background(), image.ImageRecordPropertiesRequest{Record: record})
	th.AssertNoErr(t, err)
	if again == nil || !again.Updated || again.Record == nil || calls != 4 || again.Record.StatusCode != 202 {
		t.Fatal(again, calls)
	}
	record.Resource.Body["id"][1] = 'X'
	record.Header.Set("X-Proof", "changed")
	record.Envelope[0] = '!'
	if string(seed.Resource.Body["id"]) != `"public decoy"` || seed.StatusCode != 201 || seed.Header.Get("X-Proof") != "response-0" || string(seed.Envelope) != fetched || string(again.Record.Resource.Body["id"]) != `"image /한글:%?"` || again.Record.Header.Get("X-Proof") != "response-3" || string(again.Record.Envelope) != saved {
		t.Fatal("helper return aliases an input or another result")
	}
}

func TestConnectionOwnedImagePropertiesRejectsInvalidInvocation(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/v2/"))
	th.AssertNoErr(t, err)
	marker := errors.New("property facade canceled")
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(marker)
	var absent *sdk.Connection
	for _, test := range []struct {
		name  string
		conn  *sdk.Connection
		ctx   context.Context
		cause error
	}{
		{"nil connection", absent, context.Background(), resource.ErrInvalidOption},
		{"nil context", conn, nil, resource.ErrInvalidOption},
		{"canceled context", conn, canceled, marker},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.conn.UpdateImagePropertiesRecord(test.ctx, image.ImageRecordPropertiesRequest{ID: "fixed"})
			var operation *resource.OperationError
			if result != nil || !errors.Is(err, test.cause) || !errors.As(err, &operation) || operation.Resource != "image" || operation.Operation != "UpdateImagePropertiesRecord" {
				t.Fatal(result, err)
			}
		})
	}
}
