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

func TestConnectionOwnedImageDownloadSharesSourceAndRetainsBorrowedDataThroughImportUpdateAndStage(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const initial = "image /한글:%?"
	const fetched = "retarget /한글:%?"
	const checksum = "900150983cd24fb0d6963f7d28e17f72"
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("download-0")
	calls := 0
	reader := strings.NewReader("headtail")
	var source *gophercloud.ServiceClient
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		if index > 9 || req.Header.Get("X-Auth-Token") != fmt.Sprintf("download-%d", index) || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
			t.Fatal(index, req.Header)
		}
		id := fetched
		if index <= 2 {
			id = initial
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
		case 3, 9:
			path += "/file"
		case 4:
			path += "/import"
			method = http.MethodPost
		case 5:
			method = http.MethodPatch
		case 6:
			path += "/stage"
			method = http.MethodPut
		}
		query := ""
		if index == 3 || index == 9 {
			query = (url.Values{"prefer": {"fast,fast, a,b "}}).Encode()
		}
		expectedURL := path
		if query != "" {
			expectedURL += "?" + query
		}
		if req.Method != method || req.URL.String() != expectedURL {
			t.Fatal(index, req.Method, req.URL)
		}
		header := "source"
		if index >= 4 {
			header = "later"
		}
		th.AssertEquals(t, header, req.Header.Get("X-Source"))
		if index == 2 {
			source.MoreHeaders["X-Source"] = "later"
		}
		switch index {
		case 1:
			th.AssertEquals(t, "8", req.Header.Get("X-OpenStack-Image-Size"))
			data := make([]byte, 4)
			n, err := io.ReadFull(req.Body, data)
			if err != nil || n != 4 || string(data) != "head" {
				t.Fatal(n, err, string(data))
			}
			_ = req.Body.Close()
		case 4:
			var body map[string]json.RawMessage
			th.AssertNoErr(t, json.NewDecoder(req.Body).Decode(&body))
			if len(body) != 1 || string(body["method"]) != `{"name":"glance-direct"}` {
				t.Fatal(body)
			}
		case 5:
			var patches []struct {
				Op, Path string
				Value    json.RawMessage
			}
			th.AssertNoErr(t, json.NewDecoder(req.Body).Decode(&patches))
			if len(patches) != 1 || patches[0].Op != "add" || patches[0].Path != "/name" || string(patches[0].Value) != `"after"` {
				t.Fatal(patches)
			}
		case 6:
			th.AssertEquals(t, "8", req.Header.Get("X-OpenStack-Image-Size"))
			data, err := io.ReadAll(req.Body)
			th.AssertNoErr(t, err)
			th.AssertEquals(t, "tail", string(data))
			_ = req.Body.Close()
		}
		if method == http.MethodGet && req.Body != nil {
			t.Fatal("read workflow emitted body", index)
		}
		calls++
		provider.SetToken(fmt.Sprintf("download-%d", calls))
		code, body := 203, `{"id":"retarget /한글:%?","status":"queued","container_format":"bare","disk_format":"qcow2","checksum":"`+checksum+`"}`
		headerResponse := http.Header{"X-Proof": {fmt.Sprintf("phase-%d", index)}}
		switch index {
		case 0:
			body = `{"id":"image /한글:%?","status":"queued"}`
		case 1:
			code, body = 299, "opaque upload"
		case 2:
			headerResponse.Set("OpenStack-image-import-methods", "fetch, only")
		case 3, 9:
			code, body = 206, "abc"
			headerResponse.Set("Content-Range", "bytes 0-2/3")
			headerResponse.Set("Content-MD5", "binary decoy")
			headerResponse.Set("OpenStack-image-import-methods", "must not consume")
		case 4, 6:
			code, body = 204, ""
		case 5:
			body = `{"name":"after"}`
		case 7:
			body = `{"name":"staged","status":"queued"}`
		}
		return &http.Response{Request: req, StatusCode: code, Header: headerResponse, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	cloud := "connection cloud"
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(resource.CloudLocation{Cloud: &cloud}))
	th.AssertNoErr(t, err)
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source = versioned.RawClient()
	source.Microversion = "2.10"
	source.MoreHeaders = map[string]string{"X-Source": "source"}
	if service.RawClient() != source || service.API.RawClient() != source {
		t.Fatal("owned download replaced connection source")
	}
	uploaded, err := conn.UploadImageRecord(context.Background(), image.ImageRecordUploadRequest{Data: reader}, image.WithImageRecordUploadContainerFormat("bare"), image.WithImageRecordUploadDiskFormat("qcow2"))
	th.AssertNoErr(t, err)
	if uploaded == nil || uploaded.Record == nil || calls != 2 || reader.Len() != 4 {
		t.Fatal(uploaded, calls, reader.Len())
	}
	uploaded.Record.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
	uploaded.Record.Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
	downloaded, err := conn.DownloadImageRecord(context.Background(), image.ImageRecordDownloadRequest{Record: uploaded.Record}, image.WithImageRecordDownloadStorePreferences("fast", "fast", " a,b "), image.WithImageRecordDownloadChunkSize(2))
	if err != nil || downloaded == nil || downloaded.Record == nil || downloaded.Metadata == nil || downloaded.Downloaded == nil || downloaded.Metadata.StatusCode != 203 || downloaded.Downloaded.StatusCode != 206 || string(downloaded.Downloaded.Body) != "abc" || downloaded.BytesWritten != 3 || downloaded.Checksum == nil || !downloaded.Checksum.Verified || calls != 4 || reader.Len() != 4 || string(downloaded.Record.Resource.Body["id"]) != `"retarget /한글:%?"` || downloaded.Record.Header.Get("X-Proof") != "phase-2" || downloaded.Downloaded.Header.Get("X-Proof") != "phase-3" {
		t.Fatal(downloaded, err, calls, reader.Len())
	}
	th.CheckDeepEquals(t, []string{"fetch", " only"}, downloaded.Record.ImportMethods)
	var location resource.CloudLocation
	th.AssertNoErr(t, json.Unmarshal(downloaded.Record.Resource.Body["location"], &location))
	th.AssertEquals(t, "connection cloud", *location.Cloud)
	imported, err := conn.ImportImageRecord(context.Background(), image.ImageRecordImportRequest{Record: downloaded.Record})
	th.AssertNoErr(t, err)
	if imported == nil || imported.Record == nil || imported.Acknowledgement.ImageID != fetched || calls != 5 || reader.Len() != 4 {
		t.Fatal(imported, calls, reader.Len())
	}
	updated, err := conn.UpdateImageRecord(context.Background(), image.ImageRecordUpdateRequest{Record: imported.Record, Attributes: map[string]any{"name": "after"}})
	th.AssertNoErr(t, err)
	if updated == nil || calls != 6 || reader.Len() != 4 {
		t.Fatal(updated, calls, reader.Len())
	}
	staged, err := conn.StageImageRecord(context.Background(), image.ImageRecordStageRequest{Record: updated})
	th.AssertNoErr(t, err)
	if staged == nil || staged.Record == nil || calls != 8 || reader.Len() != 0 || string(staged.Record.Resource.Body["name"]) != `"staged"` {
		t.Fatal(staged, calls, reader.Len())
	}
	streamed, err := conn.DownloadImageRecord(context.Background(), image.ImageRecordDownloadRequest{Record: staged.Record}, image.WithImageRecordDownloadStorePreferences("fast", "fast", " a,b "), image.WithImageRecordDownloadStream(true))
	if err != nil || streamed == nil || streamed.Downloaded.Stream == nil || calls != 10 || streamed.Checksum == nil || streamed.Checksum.Complete || streamed.BytesWritten != 0 || string(streamed.Downloaded.ContentMD5) != `"`+checksum+`"` || streamed.Downloaded.Header.Get("Content-MD5") != "binary decoy" || streamed.Downloaded.CompatibilityHeader.Get("Content-MD5") != checksum {
		t.Fatal(streamed, err, calls)
	}
	data, err := io.ReadAll(streamed.Downloaded.Stream)
	th.AssertNoErr(t, err)
	th.AssertEquals(t, "abc", string(data))
	th.AssertNoErr(t, streamed.Downloaded.Stream.Close())
	if streamed.Checksum.Complete || streamed.Checksum.Verified || streamed.BytesWritten != 0 || reader.Len() != 0 {
		t.Fatal(streamed, reader.Len())
	}
	downloaded.Metadata.Body[0] = '!'
	downloaded.Downloaded.Header.Set("X-Proof", "changed")
	downloaded.Record.Envelope[0] = '!'
	if imported.Record.Header.Get("X-Proof") != "phase-2" || string(imported.Record.Envelope)[0] != '{' || streamed.Record.Header.Get("X-Proof") != "phase-8" || source.MoreHeaders["X-Source"] != "later" {
		t.Fatal("workflow evidence aliases", imported, streamed)
	}
}

func TestConnectionOwnedImageDownloadRejectsInvalidInvocation(t *testing.T) {
	conn, err := sdk.FromProvider(&gophercloud.ProviderClient{}, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/v2/"))
	th.AssertNoErr(t, err)
	marker := errors.New("download facade canceled")
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(marker)
	var absent *sdk.Connection
	for _, test := range []struct {
		name  string
		conn  *sdk.Connection
		ctx   context.Context
		cause error
	}{{"nil connection", absent, context.Background(), resource.ErrInvalidOption}, {"nil context", conn, nil, resource.ErrInvalidOption}, {"canceled context", conn, canceled, marker}} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.conn.DownloadImageRecord(test.ctx, image.ImageRecordDownloadRequest{ID: "fixed"})
			var operation *resource.OperationError
			if got != nil || !errors.Is(err, test.cause) || !errors.As(err, &operation) || operation.Resource != "image" || operation.Operation != "DownloadImageRecord" {
				t.Fatal(got, err)
			}
		})
	}
}
