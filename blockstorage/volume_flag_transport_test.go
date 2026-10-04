package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		call       vstAction
	}{
		{"SetVolumeBootableStatus", `{"os-set_bootable":{"bootable":false}}`, func(ctx context.Context, c *gophercloud.ServiceClient, in blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.SetVolumeBootableStatus(ctx, c, in, false)
		}},
		{"SetVolumeReadonly", `{"os-update_readonly_flag":{"readonly":false}}`, func(ctx context.Context, c *gophercloud.ServiceClient, in blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.SetVolumeReadonly(ctx, c, in, blockstorage.WithVolumeReadonly(false))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vstClient(cloud)
			ctx, cancel := context.WithCancelCause(bmwContext(t))
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("flag Read"), errors.New("flag Close"), errors.New("flag canceled")
			const reply = "opaque flag acknowledgement"
			body := &snapshotReadTransportBody{data: strings.NewReader(reply), readError: readCause, closeError: closeCause, onClose: func() { client.ResourceBase += "changed/"; cancel(cancelCause) }}
			var wire, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("accepted flag cannot retry")
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
				wire.Add(1)
				vstPost(t, req, tc.body, "test-token")
				return snapshotReadTransportResponse(req, 203, body, "flag current response"), nil
			})
			result, err := tc.call(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "requested"})
			var proof *resource.ResponseError
			if result == nil || result.Completed || result.VolumeID != "requested" || len(result.Discovery) != 0 || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != reply || proof.Header.Get("X-Proof") != "flag current response" || wire.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(result, err, proof, wire.Load(), retries.Load(), body.closes.Load())
			}
			for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
				if !errors.Is(err, cause) {
					t.Fatal("lost flag cause", cause, err)
				}
			}
			vstOperation(t, err, tc.name)
			vstPage(t, result.Applied, 203, reply, "flag current response")
			proof.Body[0] = '!'
			proof.Header.Set("X-Proof", "changed")
			vstPage(t, result.Applied, 203, reply, "flag current response")
		})
	}
}

func TestVolumeReadonlyRetryPreservesFalseAndRejectsChangedFlag(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(map[bool]string{false: "bounded unchanged retry", true: "changed bool rejected"}[mutate], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vstClient(cloud)
			const body = `{"os-update_readonly_flag":{"readonly":false}}`
			const changed = `{"os-update_readonly_flag":{"readonly":true}}`
			var wire, retries, options atomic.Int32
			var original error
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, opts *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				original = err
				if retries.Add(1) > 1 {
					return errors.Join(err, errors.New("bounded retry exhausted"))
				}
				raw, ok := opts.JSONBody.(json.RawMessage)
				if !ok || string(raw) != body || method != http.MethodPost || !strings.HasSuffix(target, vstPath) || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody {
					t.Error(method, target, opts, string(raw))
				}
				if mutate {
					opts.JSONBody = json.RawMessage(changed)
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := wire.Add(1)
				vstPost(t, r, body, "test-token")
				if n == 1 {
					w.Header().Set("X-Proof", "original flag rejection")
					testcloud.JSON(w, 503, `{"error":"flag retry"}`)
					return
				}
				w.Header().Set("X-Proof", "second flag response")
				w.WriteHeader(203)
				_, _ = w.Write([]byte("opaque retry"))
			})
			result, err := blockstorage.SetVolumeReadonly(bmwContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "requested"}, func(next *blockstorage.VolumeReadonlyOpts) error {
				options.Add(1)
				flag := false
				next.Readonly = &flag
				return nil
			})
			if options.Load() != 1 || retries.Load() != 1 {
				t.Fatal(options.Load(), retries.Load())
			}
			if mutate {
				vstNoApplied(t, result, err, "SetVolumeReadonly")
				var native, first gophercloud.ErrUnexpectedResponseCode
				if wire.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || !errors.As(original, &first) || native.Actual != 503 || native.Method != first.Method || native.URL != first.URL || string(native.Body) != `{"error":"flag retry"}` || native.ResponseHeader.Get("X-Proof") != "original flag rejection" {
					t.Fatal(result, err, original, wire.Load())
				}
			} else {
				if err != nil || result == nil || !result.Completed || wire.Load() != 2 || len(result.Discovery) != 0 {
					t.Fatal(result, err, wire.Load())
				}
				vstPage(t, result.Applied, 203, "opaque retry", "second flag response")
			}
		})
	}
}
