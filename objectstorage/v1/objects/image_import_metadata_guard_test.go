package objects

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestImageImportObjectMetadataOuterGuardStopsNativeRetryAndReauthentication(t *testing.T) {
	for _, policy := range []string{"retry", "reauthentication"} {
		t.Run(policy, func(t *testing.T) {
			c := objectMetadataClient()
			calls, hooks := 0, 0
			changed := false
			outer := errors.New("image source changed in native callback")
			status := 503
			if policy == "retry" {
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					changed = true
					return nil
				}
			} else {
				status = 401
				c.ReauthFunc = func(context.Context) error { hooks++; changed = true; return nil }
			}
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return objectMetadataWire(r, status, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("rejected"))), nil
			})
			ctx := rest.WithOperationSources(rest.WithOperationGuard(context.Background(), func(context.Context) error {
				if changed {
					return outer
				}
				return nil
			}))
			result, err := New(c).GetMetadata(ctx, "images", "image")
			// Gophercloud retains two independent reauthentication failures
			// in fields instead of unwrapping its original HTTP rejection.
			nativeErr := err
			if policy == "reauthentication" {
				var reauth *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &reauth) || !errors.Is(reauth.ErrReauth, outer) {
					t.Fatal(result, err, reauth, calls, hooks)
				}
				nativeErr = reauth.ErrOriginal
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if result != nil || !errors.Is(err, outer) || !errors.As(nativeErr, &native) || native.Actual != status || string(native.Body) != "rejected" || calls != 1 || hooks != 1 {
				t.Fatal(result, err, native, calls, hooks)
			}
		})
	}
}

func TestImageImportObjectMetadataReadGuardKeepsAcceptedEvidenceBeforeCloseRestores(t *testing.T) {
	c := objectMetadataClient()
	outer := errors.New("image source changed while reading metadata")
	changed := false
	read := false
	calls := 0
	body := &objectMetadataBody{Reader: objectMetadataReader(func(data []byte) (int, error) {
		if read {
			return 0, io.EOF
		}
		read = true
		changed = true
		return copy(data, "actual"), io.EOF
	}), onClose: func() { changed = false }}
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return objectMetadataWire(r, 200, http.Header{"X-Proof": {"kept"}}, body), nil
	})
	ctx := rest.WithOperationSources(rest.WithOperationGuard(context.Background(), func(context.Context) error {
		if changed {
			return outer
		}
		return nil
	}))
	result, err := New(c).GetMetadata(ctx, "images", "image")
	if result == nil || result.StatusCode != 200 || string(result.Body) != "actual" || result.Metadata != nil || !errors.Is(err, outer) || calls != 1 || body.closes.Load() != 1 || changed {
		t.Fatal(result, err, calls, body.closes.Load(), changed)
	}
	objectMetadataProof(t, err, 200, "actual")
}

func TestImageImportObjectSourceReadMutationRemainsRegisteredAfterCloseRestores(t *testing.T) {
	c := objectMetadataClient()
	endpoint := c.Endpoint
	read := false
	body := &objectMetadataBody{Reader: objectMetadataReader(func(data []byte) (int, error) {
		if read {
			return 0, io.EOF
		}
		read = true
		c.Endpoint = "http://changed.invalid/v1/AUTH_a/"
		return copy(data, "actual"), io.EOF
	}), onClose: func() { c.Endpoint = endpoint }}
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		return objectMetadataWire(r, 200, http.Header{"X-Proof": {"kept"}}, body), nil
	})
	ctx := rest.WithOperationSources(context.Background())
	result, err := New(c).GetMetadata(ctx, "images", "image")
	if result == nil || !errors.Is(err, resource.ErrInvalidOption) || c.Endpoint != endpoint || !errors.Is(rest.CheckOperationGuard(ctx), resource.ErrInvalidOption) {
		t.Fatal(result, err, c.Endpoint, rest.CheckOperationGuard(ctx))
	}
	objectMetadataProof(t, err, 200, "actual")
}
