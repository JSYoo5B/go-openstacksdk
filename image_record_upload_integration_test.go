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

func TestConnectionOwnedImageUploadSharesSourceAndBorrowedDataThroughImportUpdateAndStage(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "image /한글:%?"
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("upload-0")
	calls := 0
	reader := strings.NewReader("headtail")
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		if index > 5 || req.Header.Get("X-Auth-Token") != fmt.Sprintf("upload-%d", index) || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
			t.Fatal(index, req.Method, req.URL, req.Header)
		}
		path := endpoint + "images/" + url.PathEscape(id)
		method := http.MethodGet
		switch index {
		case 0:
			path = endpoint + "images"
			method = http.MethodPost
		case 1:
			path += "/file"
			method = http.MethodPut
		case 2:
			path += "/import"
			method = http.MethodPost
		case 3:
			method = http.MethodPatch
		case 4:
			path += "/stage"
			method = http.MethodPut
		}
		if req.Method != method || req.URL.String() != path || req.URL.RawQuery != "" {
			t.Fatal(index, req.Method, req.URL)
		}
		header := "source"
		if index >= 2 {
			header = "later"
		}
		th.AssertEquals(t, header, req.Header.Get("X-Source"))
		switch index {
		case 0:
			var body map[string]json.RawMessage
			th.AssertNoErr(t, json.NewDecoder(req.Body).Decode(&body))
			if len(body) != 4 || string(body["container_format"]) != `"bare"` || string(body["disk_format"]) != `"qcow2"` || string(body["name"]) != `"seed"` || string(body["status"]) != `"queued"` {
				t.Fatal(body)
			}
			th.AssertEquals(t, "application/json", req.Header.Get("Content-Type"))
			th.AssertEquals(t, "application/json", req.Header.Get("Accept"))
		case 1:
			th.AssertEquals(t, "8", req.Header.Get("X-OpenStack-Image-Size"))
			th.AssertEquals(t, "application/octet-stream", req.Header.Get("Content-Type"))
			th.AssertEquals(t, "", req.Header.Get("Accept"))
			bytes := make([]byte, 4)
			n, err := io.ReadFull(req.Body, bytes)
			if err != nil || n != 4 || string(bytes) != "head" {
				t.Fatal(n, err, string(bytes))
			}
			_ = req.Body.Close()
		case 2:
			var body map[string]json.RawMessage
			th.AssertNoErr(t, json.NewDecoder(req.Body).Decode(&body))
			if len(body) != 1 || string(body["method"]) != `{"name":"glance-direct"}` {
				t.Fatal(body)
			}
		case 3:
			var patches []struct {
				Op, Path string
				Value    json.RawMessage
			}
			th.AssertNoErr(t, json.NewDecoder(req.Body).Decode(&patches))
			if len(patches) != 1 || patches[0].Op != "replace" || patches[0].Path != "/name" || string(patches[0].Value) != `"after"` {
				t.Fatal(patches)
			}
		case 4:
			th.AssertEquals(t, "8", req.Header.Get("X-OpenStack-Image-Size"))
			body, err := io.ReadAll(req.Body)
			th.AssertNoErr(t, err)
			th.AssertEquals(t, "tail", string(body))
			_ = req.Body.Close()
		case 5:
			if req.Body != nil {
				t.Fatal("fetch carried binary body")
			}
		}
		calls++
		provider.SetToken(fmt.Sprintf("upload-%d", calls))
		status, body := 203, `{"id":"image /한글:%?","status":"queued"}`
		responseHeader := http.Header{"X-Proof": {fmt.Sprintf("phase-%d", index)}}
		switch index {
		case 0:
			body = `{"id":"image /한글:%?","status":"queued"}`
			responseHeader.Set("OpenStack-image-import-methods", "direct,web")
		case 1:
			status, body = 299, "opaque upload\xff"
			responseHeader.Set("OpenStack-image-import-methods", "must not consume")
		case 2:
			status, body = 204, ""
		case 3:
			body = `{"name":"after"}`
		case 4:
			status, body = 204, ""
		case 5:
			body = `{"name":"staged","status":"queued"}`
		}
		return &http.Response{Request: req, StatusCode: status, Header: responseHeader, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	cloud := "connection cloud"
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(resource.CloudLocation{Cloud: &cloud}))
	th.AssertNoErr(t, err)
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source := versioned.RawClient()
	source.Microversion = "2.10"
	source.MoreHeaders = map[string]string{"X-Source": "source"}
	if service.RawClient() != source || service.API.RawClient() != source {
		t.Fatal("owned upload replaced connection source")
	}
	uploaded, err := conn.UploadImageRecord(context.Background(), image.ImageRecordUploadRequest{Data: reader, Attributes: map[string]any{"name": "seed", "status": "queued"}}, image.WithImageRecordUploadContainerFormat("bare"), image.WithImageRecordUploadDiskFormat("qcow2"), func(*image.ImageRecordUploadOpts) error { source.MoreHeaders["X-Source"] = "later"; return nil })
	if err != nil || uploaded == nil || uploaded.Record == nil || uploaded.Metadata == nil || uploaded.Uploaded == nil || calls != 2 || uploaded.Metadata.StatusCode != 203 || uploaded.Uploaded.StatusCode != 299 || string(uploaded.Uploaded.Body) != "opaque upload\xff" || string(uploaded.Record.Envelope) != `{"id":"image /한글:%?","status":"queued"}` || uploaded.Record.Header.Get("X-Proof") != "phase-0" || len(uploaded.Record.ImportMethods) != 2 || reader.Len() != 4 {
		t.Fatal(uploaded, err, calls, reader.Len())
	}
	var location resource.CloudLocation
	th.AssertNoErr(t, json.Unmarshal(uploaded.Record.Resource.Body["location"], &location))
	th.AssertEquals(t, "connection cloud", *location.Cloud)
	uploaded.Record.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
	uploaded.Record.Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
	imported, err := service.ImportImageRecord(context.Background(), image.ImageRecordImportRequest{Record: uploaded.Record})
	th.AssertNoErr(t, err)
	if imported == nil || imported.Record == nil || imported.Acknowledgement.ImageID != id || reader.Len() != 4 || calls != 3 {
		t.Fatal(imported, calls, reader.Len())
	}
	updated, err := conn.UpdateImageRecord(context.Background(), image.ImageRecordUpdateRequest{Record: imported.Record, Attributes: map[string]any{"name": "after"}})
	th.AssertNoErr(t, err)
	if updated == nil || calls != 4 || reader.Len() != 4 {
		t.Fatal(updated, calls, reader.Len())
	}
	staged, err := service.StageImageRecord(context.Background(), image.ImageRecordStageRequest{Record: updated})
	th.AssertNoErr(t, err)
	if staged == nil || staged.Record == nil || staged.Staged == nil || staged.Metadata == nil || calls != 6 || reader.Len() != 0 || string(staged.Record.Resource.Body["name"]) != `"staged"` || staged.Metadata.StatusCode != 203 {
		t.Fatal(staged, calls, reader.Len())
	}
	uploaded.Metadata.Body[0] = '!'
	uploaded.Uploaded.Header.Set("X-Proof", "changed")
	uploaded.Record.Envelope[0] = '!'
	if imported.Record.Header.Get("X-Proof") != "phase-0" || string(imported.Record.Envelope) != `{"id":"image /한글:%?","status":"queued"}` || staged.Record.Header.Get("X-Proof") != "phase-5" || source.MoreHeaders["X-Source"] != "later" {
		t.Fatal("workflow receipts alias", uploaded, imported, staged)
	}
}

func TestConnectionOwnedImageUploadRejectsInvalidInvocation(t *testing.T) {
	conn, err := sdk.FromProvider(&gophercloud.ProviderClient{}, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/v2/"))
	th.AssertNoErr(t, err)
	marker := errors.New("upload facade canceled")
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(marker)
	var absent *sdk.Connection
	for _, test := range []struct {
		name  string
		conn  *sdk.Connection
		ctx   context.Context
		cause error
	}{
		{"nil connection", absent, context.Background(), resource.ErrInvalidOption}, {"nil context", conn, nil, resource.ErrInvalidOption}, {"canceled context", conn, canceled, marker},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.conn.UploadImageRecord(test.ctx, image.ImageRecordUploadRequest{}, image.WithImageRecordUploadContainerFormat("bare"), image.WithImageRecordUploadDiskFormat("qcow2"))
			var operation *resource.OperationError
			if got != nil || !errors.Is(err, test.cause) || !errors.As(err, &operation) || operation.Resource != "image" || operation.Operation != "UploadImageRecord" {
				t.Fatal(got, err)
			}
		})
	}
}
