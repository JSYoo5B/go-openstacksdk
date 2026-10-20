package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestConnectionOwnedImageImportSharesSourceAndPreservesBorrowedData(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "image /한글:%?"
	const fetched = `{"id":"image /한글:%?","status":"queued","name":"seed","container_format":"bare","disk_format":"qcow2","properties":{"team":"keep"}}`
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("import-0")
	calls := 0
	reader := strings.NewReader("binary")
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		if index > 9 || req.Header.Get("X-Auth-Token") != fmt.Sprintf("import-%d", index) || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
			t.Fatal(index, req.Method, req.URL, req.Header)
		}
		path := endpoint + "images/" + url.PathEscape(id)
		method := http.MethodGet
		if index == 1 || index == 8 {
			path += "/stage"
			method = http.MethodPut
		}
		if index == 3 {
			path = endpoint + "info/stores"
		}
		if index >= 4 && index <= 7 {
			path += "/import"
			method = http.MethodPost
		}
		if req.Method != method || req.URL.String() != path {
			t.Fatal(index, req.Method, req.URL)
		}
		if method == http.MethodPut {
			th.AssertEquals(t, "6", req.Header.Get("X-OpenStack-Image-Size"))
			body, err := io.ReadAll(req.Body)
			th.AssertNoErr(t, err)
			expected := "binary"
			if index == 8 {
				expected = ""
			}
			th.AssertEquals(t, expected, string(body))
		} else if method == http.MethodPost {
			var body map[string]json.RawMessage
			th.AssertNoErr(t, json.NewDecoder(req.Body).Decode(&body))
			var m map[string]json.RawMessage
			th.AssertNoErr(t, json.Unmarshal(body["method"], &m))
			names := []string{`"glance-direct"`, `"web-download"`, `"glance-download"`, `"copy-image"`}
			th.AssertEquals(t, names[index-4], string(m["name"]))
			storeHeader := req.Header.Get("X-Image-Meta-Store")
			if index == 4 {
				th.AssertEquals(t, `["fast"]`, string(body["stores"]))
				th.AssertEquals(t, "fast", storeHeader)
			} else if storeHeader != "" {
				t.Fatal("configured store expanded selection", index, req.Header)
			}
			if index == 5 {
				if len(m) != 1 || string(body["all_stores"]) != "false" || string(body["all_stores_must_succeed"]) != "false" {
					t.Fatal(body, m)
				}
			}
			if index == 6 {
				if string(m["glance_region"]) != `{"region":"r1"}` || string(m["glance_image_id"]) != "42" || string(m["glance_service_interface"]) != `["public"]` {
					t.Fatal(m)
				}
			}
			if index == 7 {
				th.AssertEquals(t, `["cold",null,"cold"]`, string(body["stores"]))
			}
			expectedHeader := "source"
			if index >= 5 {
				expectedHeader = "later"
			}
			th.AssertEquals(t, expectedHeader, req.Header.Get("X-Source"))
			if req.Header.Get("X-OpenStack-Image-Size") != "" {
				t.Fatal("binary size entered import", req.Header)
			}
		} else if req.Body != nil {
			t.Fatal(index, "unexpected GET body")
		}
		calls++
		provider.SetToken(fmt.Sprintf("import-%d", calls))
		header := http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}}
		body, status := fetched, 201
		if index == 1 || index == 8 {
			body, status = "", 204
		}
		if index == 2 {
			body, status = `{"id":"image /한글:%?","status":"queued"}`, 203
			header.Set("OpenStack-Image-Import-Methods", "direct,copy")
		}
		if index == 3 {
			body, status = `{"stores":[{"id":"fast","name":"store"}]}`, 200
		}
		if index >= 4 && index <= 7 {
			body, status = "opaque import\xff", 299
			header.Set("OpenStack-Image-Import-Methods", "do,not,consume")
			header.Set("Location", "https://foreign.test/task/decoy")
		}
		if index == 9 {
			body, status = "invalid metadata JSON", 299
		}
		return &http.Response{Request: req, StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint))
	th.AssertNoErr(t, err)
	service, err := conn.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := conn.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	source := versioned.RawClient()
	source.MoreHeaders = map[string]string{"X-Source": "source", "X-Image-Meta-Store": "configured-decoy"}
	source.Microversion = "2.10"
	if service.RawClient() != source || service.API.RawClient() != source {
		t.Fatal("import replaced shared source")
	}
	seed, err := service.GetImageRecord(context.Background(), image.ImageRecordRequest{ID: id})
	th.AssertNoErr(t, err)
	staged, err := conn.StageImageRecord(context.Background(), image.ImageRecordStageRequest{Record: seed, Data: reader})
	th.AssertNoErr(t, err)
	stores, err := service.API.ServiceInfo.AllStoreRecords(context.Background())
	th.AssertNoErr(t, err)
	if len(stores) != 1 || calls != 4 {
		t.Fatal(stores, calls)
	}
	stores[0].Resource.Body["id"] = json.RawMessage(`"public-store-decoy"`)
	staged.Record.Resource.Body["id"] = json.RawMessage(`"public-image-decoy"`)
	first, err := conn.ImportImageRecord(context.Background(), image.ImageRecordImportRequest{Record: staged.Record}, image.WithImageRecordImportStore(image.ImageRecordImportStore{Record: stores[0]}), func(*image.ImageRecordImportOpts) error { source.MoreHeaders["X-Source"] = "later"; return nil })
	th.AssertNoErr(t, err)
	if first == nil || first.Record == nil || first.Acknowledgement == nil || first.Acknowledgement.ImageID != id || first.Acknowledgement.StatusCode != 299 || string(first.Acknowledgement.Body) != "opaque import\xff" || first.Record.StatusCode != 203 || first.Record.Header.Get("X-Proof") != "response-2" || len(first.Record.ImportMethods) != 0 || string(first.Record.Resource.Body["id"]) != `"image /한글:%?"` {
		t.Fatal(first)
	}
	second, err := service.ImportImageRecord(context.Background(), image.ImageRecordImportRequest{Record: first.Record}, image.WithImageRecordImportMethod(imageimport.WebDownloadMethod), image.WithImageRecordImportAllStores(false), image.WithImageRecordImportAllStoresMustSucceed(false))
	th.AssertNoErr(t, err)
	third, err := conn.ImportImageRecord(context.Background(), image.ImageRecordImportRequest{Record: second.Record}, image.WithImageRecordImportMethod(imageimport.GlanceDownloadMethod), image.WithImageRecordImportRemoteRegion(map[string]any{"region": "r1"}), image.WithImageRecordImportRemoteImageID(42), image.WithImageRecordImportRemoteServiceInterface([]string{"public"}))
	th.AssertNoErr(t, err)
	fourth, err := service.ImportImageRecord(context.Background(), image.ImageRecordImportRequest{Record: third.Record}, image.WithImageRecordImportMethod(imageimport.CopyImageMethod), image.WithImageRecordImportStores(image.ImageRecordImportStore{ID: "cold"}, image.ImageRecordImportStore{RawID: json.RawMessage("null")}, image.ImageRecordImportStore{ID: "cold"}))
	th.AssertNoErr(t, err)
	if calls != 8 || !reflect.DeepEqual(first.Record.Envelope, fourth.Record.Envelope) || fourth.Record.StatusCode != 203 || string(fourth.Record.Resource.Body["status"]) != `"queued"` || source.MoreHeaders["X-Image-Meta-Store"] != "configured-decoy" {
		t.Fatal(first, fourth, calls, source.MoreHeaders)
	}
	// Import performs no binary read and keeps the caller's plain data reference.
	// After the initial stage its cursor is EOF: nil Data here sends zero bytes.
	again, err := service.StageImageRecord(context.Background(), image.ImageRecordStageRequest{Record: fourth.Record})
	th.AssertNoErr(t, err)
	if calls != 10 || again.Record == nil || again.Staged.StatusCode != 204 || again.Metadata.StatusCode != 299 || again.Record.Wire != nil {
		t.Fatal(again, calls)
	}
	first.Acknowledgement.Body[0] = '!'
	first.Acknowledgement.Header.Set("X-Proof", "changed")
	first.Record.Envelope[0] = '!'
	if string(staged.Record.Resource.Body["id"]) != `"public-image-decoy"` || string(seed.Envelope) != fetched || fourth.Record.Header.Get("X-Proof") != "response-2" || string(fourth.Record.Envelope) == string(first.Record.Envelope) || fourth.Acknowledgement.Header.Get("X-Proof") != "response-7" {
		t.Fatal("import receipts alias one another", staged, first, fourth)
	}
}

func TestConnectionOwnedImageImportRejectsInvalidInvocation(t *testing.T) {
	conn, err := sdk.FromProvider(&gophercloud.ProviderClient{}, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/v2/"))
	th.AssertNoErr(t, err)
	marker := errors.New("import facade canceled")
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
			got, err := test.conn.ImportImageRecord(test.ctx, image.ImageRecordImportRequest{ID: "fixed"})
			var operation *resource.OperationError
			if got != nil || !errors.Is(err, test.cause) || !errors.As(err, &operation) || operation.Resource != "image" || operation.Operation != "ImportImageRecord" {
				t.Fatal(got, err)
			}
		})
	}
}
