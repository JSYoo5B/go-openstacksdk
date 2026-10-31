package connectiontest_test

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
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionOwnedImageCreateComposesPolicyAndKeepsSwiftLazy(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	calls := 0
	provider.HTTPClient.Transport = testcloud.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		index := calls
		calls++
		if index == 0 {
			if request.Method != http.MethodPost {
				t.Fatalf("request.Method = %v, want %v", request.Method, http.MethodPost)
			}
			if got := request.URL.String(); got != "https://cloud.test/glance/v2/images" {
				t.Fatalf("request.URL.String() = %v, want %v", got, "https://cloud.test/glance/v2/images")
			}
			var body map[string]json.RawMessage
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(body["name"]) != `"image"` || string(body["disk_format"]) != `"raw"` || string(body["container_format"]) != `"bare"` || string(body["vendor"]) != `"False"` || string(body["min_disk"]) != "3" || string(body["owner_specified.openstack.object"]) != `"images/image"` {
				t.Fatal(body)
			}
			return &http.Response{Request: request, StatusCode: 203, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"id":"created"}`))}, nil
		}
		if index != 1 {
			t.Fatal("unexpected Swift, wait or metadata request", index, request.Method, request.URL)
		}
		if request.Method != http.MethodPut {
			t.Fatalf("request.Method = %v, want %v", request.Method, http.MethodPut)
		}
		if got := request.URL.String(); got != "https://cloud.test/glance/v2/images/created/file" {
			t.Fatalf("request.URL.String() = %v, want %v", got, "https://cloud.test/glance/v2/images/created/file")
		}
		if got := request.Header.Get("X-OpenStack-Image-Size"); got != "" {
			t.Fatalf("request.Header.Get(\"X-OpenStack-Image-Size\") = %v, want %v", got, "")
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(body) != "payload" {
			t.Fatalf("string(body) = %v, want %v", string(body), "payload")
		}
		return &http.Response{Request: request, StatusCode: 299, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("actual upload"))}, nil
	})
	connection, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/glance/v2/"), sdk.WithImageCreatePolicy(image.WithImageCreatePolicyFormat("raw"), image.WithImageCreatePolicyVendorAgent(map[string]any{"vendor": false})))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := connection.CreateImageRecord(context.Background(), image.ImageRecordCreateRequest{Name: "image", Data: image.ImageRecordCreateBytes([]byte("payload")), Attributes: map[string]any{"min_disk": "3"}}, image.WithImageRecordCreateAllowDuplicates(true), image.WithImageRecordCreateWait(true), image.WithImageRecordCreateTimeout("ignored on direct branch"))
	if err != nil || result == nil || result.Outcome != "uploaded" || result.Record == nil || result.Created == nil || result.Uploaded == nil || result.Created.StatusCode != 203 || result.Uploaded.StatusCode != 299 || calls != 2 {
		t.Fatal(result, err, calls)
	}
	service, err := connection.Image(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	versioned, err := connection.ImageV2(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if service.RawClient() != versioned.RawClient() {
		t.Fatal("whole create replaced the connection image source")
	}
}

func TestConnectionOwnedImageCreateRejectsInvalidInvocation(t *testing.T) {
	connection, err := sdk.FromProvider(&gophercloud.ProviderClient{}, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/v2/"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
