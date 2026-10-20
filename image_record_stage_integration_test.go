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

func TestConnectionOwnedImageStageSharesSourceAndRetainsBorrowedDataThroughProperties(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	const id = "image /한글:%?"
	const fetched = `{"id":"image /한글:%?","status":"queued","name":"seed","properties":{"team":"keep"}}`
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("stage-0")
	calls := 0
	reader := strings.NewReader("skip-body")
	_, err := reader.Seek(5, io.SeekStart)
	th.AssertNoErr(t, err)
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := calls
		if index > 5 || req.Header.Get("X-Auth-Token") != fmt.Sprintf("stage-%d", index) || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
			t.Fatal(index, req.Method, req.URL, req.Header)
		}
		expectedHeader := "source"
		if index >= 3 {
			expectedHeader = "mutated"
		}
		if req.Header.Get("X-Source") != expectedHeader {
			t.Fatal("ordinary headers were recaptured within a workflow", index, req.Header)
		}
		path := endpoint + "images/" + url.PathEscape(id)
		method := http.MethodGet
		if index == 1 || index == 4 {
			path += "/stage"
			method = http.MethodPut
		}
		if index == 3 {
			method = http.MethodPatch
		}
		if req.URL.String() != path || req.Method != method {
			t.Fatal(index, req.Method, req.URL)
		}
		if method == http.MethodPut {
			if req.Header.Get("Content-Type") != "application/octet-stream" || req.Header.Get("Accept") != "" || req.Header.Get("X-OpenStack-Image-Size") != "9" || req.GetBody != nil {
				t.Fatal(index, req.Header, req.GetBody != nil)
			}
			body, err := io.ReadAll(req.Body)
			th.AssertNoErr(t, err)
			expected := "body"
			if index == 4 {
				expected = ""
			}
			th.AssertEquals(t, expected, string(body))
		} else if method == http.MethodPatch {
			var patch []struct {
				Op, Path string
				Value    json.RawMessage
			}
			th.AssertNoErr(t, json.NewDecoder(req.Body).Decode(&patch))
			if len(patch) != 1 || patch[0].Op != "add" || patch[0].Path != "/gpu" || string(patch[0].Value) != `"True"` {
				t.Fatal(patch)
			}
		} else if req.Body != nil || req.Header.Get("X-OpenStack-Image-Size") != "" {
			t.Fatal("stage size escaped into metadata GET", index, req.Body, req.Header)
		}
		calls++
		provider.SetToken(fmt.Sprintf("stage-%d", calls))
		header := http.Header{"X-Proof": {fmt.Sprintf("response-%d", index)}}
		body, status := fetched, 201
		if index == 1 {
			body, status = "opaque stage\xff", 299
			header.Set("OpenStack-Image-Import-Methods", "stage, method")
		}
		if index == 2 {
			body = `{"id":"image /한글:%?","status":"queued","size":9,"properties":{"team":"keep"}}`
			header.Set("OpenStack-Image-Import-Methods", "final, store")
		}
		if index == 3 {
			body, status = `{"id":"image /한글:%?","status":"queued","team":"keep","gpu":"True"}`, 202
		}
		if index == 4 {
			body, status = "", 204
		}
		if index == 5 {
			body, status = "opaque final JSON", 299
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
	source.MoreHeaders = map[string]string{"X-Source": "source"}
	source.Microversion = "2.10"
	if service.RawClient() != source || service.API.RawClient() != source {
		t.Fatal("stage replaced shared source")
	}
	seed, err := service.GetImageRecord(context.Background(), image.ImageRecordRequest{ID: id})
	th.AssertNoErr(t, err)
	seed.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
	first, err := conn.StageImageRecord(context.Background(), image.ImageRecordStageRequest{Record: seed, Data: reader}, func(*image.ImageRecordStageOpts) error {
		source.MoreHeaders["X-Source"] = "mutated"
		return nil
	})
	th.AssertNoErr(t, err)
	if first == nil || first.Record == nil || first.Staged == nil || first.Metadata == nil || calls != 3 || first.Staged.StatusCode != 299 || first.Metadata.StatusCode != 201 || string(first.Staged.Body) != "opaque stage\xff" || string(first.Record.Resource.Body["name"]) != `"seed"` || string(first.Record.Resource.Body["id"]) != `"image /한글:%?"` || len(first.Record.ImportMethods) != 2 {
		t.Fatal(first, calls)
	}
	updated, err := conn.UpdateImagePropertiesRecord(context.Background(), image.ImageRecordPropertiesRequest{Record: first.Record}, image.WithImageRecordProperty("gpu", true))
	th.AssertNoErr(t, err)
	if updated == nil || !updated.Updated || updated.Record == nil || calls != 4 {
		t.Fatal(updated, calls)
	}
	// The plain borrowed data survives a translated property PATCH. Its cursor
	// is now EOF, so this second stage sends zero bytes while inferring total9.
	second, err := service.StageImageRecord(context.Background(), image.ImageRecordStageRequest{Record: updated.Record})
	th.AssertNoErr(t, err)
	if second == nil || second.Record == nil || second.Staged.StatusCode != 204 || second.Metadata.StatusCode != 299 || calls != 6 || string(second.Record.Resource.Body["properties"]) != `{"gpu":"True","team":"keep"}` || second.Record.Wire != nil {
		t.Fatal(second, calls)
	}
	first.Metadata.Body[0] = '!'
	first.Staged.Header.Set("X-Proof", "changed")
	second.Record.Header.Set("X-Proof", "changed")
	if first.Record.Header.Get("X-Proof") != "response-2" || string(first.Record.Envelope) == string(first.Metadata.Body) || second.Metadata.Header.Get("X-Proof") != "response-5" || seed.Header.Get("X-Proof") != "response-0" || string(seed.Envelope) != fetched || string(seed.Resource.Body["id"]) != `"public decoy"` {
		t.Fatal("stage evidence or records alias one another", first, second, seed)
	}
}

func TestConnectionOwnedImageStageRejectsInvalidInvocation(t *testing.T) {
	conn, err := sdk.FromProvider(&gophercloud.ProviderClient{}, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/v2/"))
	th.AssertNoErr(t, err)
	marker := errors.New("stage facade canceled")
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
			got, err := test.conn.StageImageRecord(test.ctx, image.ImageRecordStageRequest{ID: "fixed"})
			var operation *resource.OperationError
			if got != nil || !errors.Is(err, test.cause) || !errors.As(err, &operation) || operation.Resource != "image" || operation.Operation != "StageImageRecord" {
				t.Fatal(got, err)
			}
		})
	}
}
