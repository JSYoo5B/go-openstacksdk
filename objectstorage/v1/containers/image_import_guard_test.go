package containers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestImageImportContainerOuterGuardChecksBeforeOptionsAndHTTP(t *testing.T) {
	outer := errors.New("image source changed")
	calls, options := 0, 0
	api, _ := lifecycleAPI(lifecycleTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("forbidden") }))
	ctx := rest.WithOperationGuard(context.Background(), func(context.Context) error { return outer })
	result, err := api.CreateContainer(ctx, "images", func(*CreateContainerOpts) error { options++; return nil })
	if result != nil || !errors.Is(err, outer) || calls != 0 || options != 0 {
		t.Fatal(result, err, calls, options)
	}
}

func TestImageImportContainerMetadataOuterGuardStopsNativeReplay(t *testing.T) {
	for _, operation := range []string{"HEAD", "PUT", "DELETE"} {
		t.Run(operation, func(t *testing.T) {
			calls, hooks := 0, 0
			changed := false
			outer := errors.New("image source changed in retry")
			api, c := lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return lifecycleWire(r, 503, &lifecycleBody{data: "rejected"}), nil
			}))
			c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				changed = true
				return nil
			}
			ctx := rest.WithOperationSources(rest.WithOperationGuard(context.Background(), func(context.Context) error {
				if changed {
					return outer
				}
				return nil
			}))
			var err error
			switch operation {
			case "HEAD":
				_, err = api.GetMetadata(ctx, "images")
			case "PUT":
				_, err = api.CreateContainer(ctx, "images")
			case "DELETE":
				_, err = api.DeleteContainer(ctx, "images")
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, outer) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "rejected" || calls != 1 || hooks != 1 {
				t.Fatal(err, native, calls, hooks)
			}
		})
	}
}

func TestImageImportContainerOuterGuardReadFailureRemainsAfterCloseRestore(t *testing.T) {
	for _, operation := range []string{"HEAD", "PUT", "DELETE"} {
		t.Run(operation, func(t *testing.T) {
			outer := errors.New("image source changed during read")
			changed := false
			calls := 0
			body := &lifecycleBody{data: "actual", onRead: func() { changed = true }, onClose: func() { changed = false }}
			code := 204
			if operation == "PUT" {
				code = 201
			}
			api, _ := lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) { calls++; return lifecycleWire(r, code, body), nil }))
			ctx := rest.WithOperationSources(rest.WithOperationGuard(context.Background(), func(context.Context) error {
				if changed {
					return outer
				}
				return nil
			}))
			var err error
			if operation == "HEAD" {
				var result *GetMetadataResult
				result, err = api.GetMetadata(ctx, "images")
				if result == nil || result.StatusCode != 204 || string(result.Body) != "actual" || result.Metadata != nil {
					t.Fatal(result, err)
				}
			} else {
				var result *ContainerResponse
				if operation == "PUT" {
					result, err = api.CreateContainer(ctx, "images")
				} else {
					result, err = api.DeleteContainer(ctx, "images")
				}
				lifecycleProof(t, result, err, code, "actual")
			}
			var proof *resource.ResponseError
			if !errors.Is(err, outer) || !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != "actual" || calls != 1 || body.closes != 1 || changed {
				t.Fatal(err, proof, calls, body.closes, changed)
			}
		})
	}
}

func TestImageImportContainerSourceOnlyRegistrationPersistsWithoutRecursion(t *testing.T) {
	for _, field := range []string{"endpoint", "base", "type", "provider", "API"} {
		t.Run(field, func(t *testing.T) {
			outerCalls := 0
			ctx := rest.WithOperationSources(rest.WithOperationGuard(context.Background(), func(context.Context) error {
				outerCalls++
				if outerCalls > 100 {
					t.Fatal("recursive full guard")
				}
				return nil
			}))
			api, c := lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) {
				return lifecycleWire(r, 201, io.NopCloser(http.NoBody)), nil
			}))
			result, err := api.CreateContainer(ctx, "images")
			if err != nil || result == nil || outerCalls == 0 {
				t.Fatal(result, err, outerCalls)
			}
			switch field {
			case "endpoint":
				c.Endpoint = "https://changed.invalid/v1/AUTH_account/"
			case "base":
				c.ResourceBase = "https://swift.invalid/other/"
			case "type":
				c.Type = "image"
			case "provider":
				c.ProviderClient = &gophercloud.ProviderClient{}
			case "API":
				api.client = &gophercloud.ServiceClient{}
			}
			if err := rest.CheckOperationGuard(ctx); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("Swift binding disappeared", err)
			}
		})
	}
}
