package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
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

func TestConnectionOwnedImageCreateComposesPolicyAndKeepsSwiftLazy(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(request *http.Request) (*http.Response, error) {
		index := calls
		calls++
		if index == 0 {
			th.AssertEquals(t, http.MethodPost, request.Method)
			th.AssertEquals(t, "https://cloud.test/glance/v2/images", request.URL.String())
			var body map[string]json.RawMessage
			th.AssertNoErr(t, json.NewDecoder(request.Body).Decode(&body))
			if string(body["name"]) != `"image"` || string(body["disk_format"]) != `"raw"` || string(body["container_format"]) != `"bare"` || string(body["vendor"]) != `"False"` || string(body["min_disk"]) != "3" || string(body["owner_specified.openstack.object"]) != `"images/image"` {
				t.Fatal(body)
			}
			return &http.Response{Request: request, StatusCode: 203, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"id":"created"}`))}, nil
		}
		if index != 1 {
			t.Fatal("unexpected Swift, wait or metadata request", index, request.Method, request.URL)
		}
		th.AssertEquals(t, http.MethodPut, request.Method)
		th.AssertEquals(t, "https://cloud.test/glance/v2/images/created/file", request.URL.String())
		th.AssertEquals(t, "", request.Header.Get("X-OpenStack-Image-Size"))
		body, err := io.ReadAll(request.Body)
		th.AssertNoErr(t, err)
		th.AssertEquals(t, "payload", string(body))
		return &http.Response{Request: request, StatusCode: 299, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("actual upload"))}, nil
	})
	connection, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/glance/v2/"), sdk.WithImageCreatePolicy(image.WithImageCreatePolicyFormat("raw"), image.WithImageCreatePolicyVendorAgent(map[string]any{"vendor": false})))
	th.AssertNoErr(t, err)
	result, err := connection.CreateImageRecord(context.Background(), image.ImageRecordCreateRequest{Name: "image", Data: image.ImageRecordCreateBytes([]byte("payload")), Attributes: map[string]any{"min_disk": "3"}}, image.WithImageRecordCreateAllowDuplicates(true), image.WithImageRecordCreateWait(true), image.WithImageRecordCreateTimeout("ignored on direct branch"))
	if err != nil || result == nil || result.Outcome != "uploaded" || result.Record == nil || result.Created == nil || result.Uploaded == nil || result.Created.StatusCode != 203 || result.Uploaded.StatusCode != 299 || calls != 2 {
		t.Fatal(result, err, calls)
	}
	service, err := connection.Image(context.Background())
	th.AssertNoErr(t, err)
	versioned, err := connection.ImageV2(context.Background())
	th.AssertNoErr(t, err)
	if service.RawClient() != versioned.RawClient() {
		t.Fatal("whole create replaced the connection image source")
	}
}

func TestConnectionOwnedImageCreateRejectsInvalidInvocation(t *testing.T) {
	connection, err := sdk.FromProvider(&gophercloud.ProviderClient{}, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/v2/"))
	th.AssertNoErr(t, err)
	marker := errors.New("whole create canceled")
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(marker)
	var absent *sdk.Connection
	for _, test := range []struct {
		name       string
		connection *sdk.Connection
		ctx        context.Context
		cause      error
	}{{"nil connection", absent, context.Background(), resource.ErrInvalidOption}, {"nil context", connection, nil, resource.ErrInvalidOption}, {"canceled context", connection, canceled, marker}} {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.connection.CreateImageRecord(test.ctx, image.ImageRecordCreateRequest{})
			var operation *resource.OperationError
			if result != nil || !errors.Is(err, test.cause) || !errors.As(err, &operation) || operation.Operation != "CreateImageRecord" || operation.Resource != "image" {
				t.Fatal(result, err)
			}
		})
	}
}
