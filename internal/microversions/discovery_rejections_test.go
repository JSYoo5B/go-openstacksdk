package microversions

import (
	"context"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"io"
	"net/http"
	"strings"
	"testing"
)

type discoveryFaultBody struct {
	reader                io.Reader
	readErr, closeErr     error
	afterRead, afterClose func()
	closes                int
}

func (b *discoveryFaultBody) Read(data []byte) (int, error) {
	n, err := b.reader.Read(data)
	if b.afterRead != nil {
		b.afterRead()
	}
	if b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *discoveryFaultBody) Close() error {
	b.closes++
	if b.afterClose != nil {
		b.afterClose()
	}
	return b.closeErr
}

func TestDiscoveryRejectedStatusesRetainPhysicalFaultsAndTransientSourceChanges(t *testing.T) {
	for _, status := range []int{304, 400, 403, 404, 405, 500, 503} {
		for _, fault := range []string{"read", "close", "mixed", "source"} {
			t.Run(http.StatusText(status)+"/"+fault, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := &gophercloud.ServiceClient{ProviderClient: cloud.Provider, Type: "volumev3", Endpoint: cloud.Server.URL + "/v3/project/", ResourceBase: cloud.Server.URL + "/v3/project/"}
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				source, err := cloudread.Capture(ctx, client, "volume")
				if err != nil {
					t.Fatal(err)
				}
				readErr := errors.New("discovery read failure")
				closeErr := errors.New("discovery close failure")
				cause := errors.New("discovery cancellation")
				body := &discoveryFaultBody{reader: strings.NewReader("partial-discovery")}
				if fault == "read" || fault == "mixed" {
					body.readErr = readErr
				}
				if fault == "close" || fault == "mixed" {
					body.closeErr = closeErr
				}
				if fault == "source" || fault == "mixed" {
					original := client.ResourceBase
					body.afterRead = func() { client.ResourceBase = cloud.Server.URL + "/changed/"; cancel(cause) }
					body.afterClose = func() { client.ResourceBase = original }
				}
				calls := 0
				cloud.Provider.HTTPClient.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: status, Header: http.Header{"X-Discovery-Proof": []string{"actual"}}, Body: body, Request: req}, nil
				})
				result, err := Read(ctx, source, CinderProfile)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != status || string(native.Body) != "partial-discovery" || native.ResponseHeader.Get("X-Discovery-Proof") != "actual" || native.Method != "GET" || native.URL != cloud.Server.URL+"/v3/" {
					t.Fatal(result, err, native)
				}
				if body.readErr != nil && !errors.Is(err, readErr) {
					t.Fatal(err)
				}
				if body.closeErr != nil && !errors.Is(err, closeErr) {
					t.Fatal(err)
				}
				if body.afterRead != nil && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
					t.Fatal(err)
				}
				if calls != 1 || body.closes != 1 || len(result.Responses) != 0 {
					t.Fatal(calls, body.closes, result)
				}
			})
		}
	}
}
