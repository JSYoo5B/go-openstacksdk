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

func TestVolumeResizeAcceptedFaultsKeepActualOpaqueProofAndEveryCause(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		call       vstAction
	}{
		{"ExtendVolume", `{"os-extend":{"new_size":0}}`, func(ctx context.Context, c *gophercloud.ServiceClient, in blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ExtendVolume(ctx, c, in, 0)
		}},
		{"RetypeVolume", `{"os-retype":{"migration_policy":"never","new_type":"new-type"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, in blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.RetypeVolume(ctx, c, in, "new-type")
		}},
		{"CompleteVolumeExtend", `{"os-extend_volume_completion":{"error":false}}`, func(ctx context.Context, c *gophercloud.ServiceClient, in blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeExtend(ctx, c, in)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vstClient(cloud)
			ctx, cancel := context.WithCancelCause(bmwContext(t))
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("resize Read"), errors.New("resize Close"), errors.New("resize cancellation")
			const reply = "opaque accepted resize"
			body := &snapshotReadTransportBody{data: strings.NewReader(reply), readError: readCause, closeError: closeCause, onClose: func() { client.ResourceBase += "changed/"; cancel(cancelCause) }}
			var wire, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("accepted response cannot retry")
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
				wire.Add(1)
				vstPost(t, req, tc.body, "test-token")
				return snapshotReadTransportResponse(req, 203, body, "current resize response"), nil
			})
			result, err := tc.call(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "requested"})
			var proof *resource.ResponseError
			if result == nil || result.Completed || result.VolumeID != "requested" || len(result.Discovery) != 0 || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != reply || proof.Header.Get("X-Proof") != "current resize response" || wire.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(result, err, proof, wire.Load(), retries.Load(), body.closes.Load())
			}
			for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
				if !errors.Is(err, cause) {
					t.Fatal(cause, err)
				}
			}
			vstOperation(t, err, tc.name)
			vstPage(t, result.Applied, 203, reply, "current resize response")
			proof.Body[0] = '!'
			proof.Header.Set("X-Proof", "caller changed")
			vstPage(t, result.Applied, 203, reply, "current resize response")
		})
	}
}

func TestVolumeRetypeReauthRetryKeepsOwnedPolicyAndRejectsChangedBody(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(map[bool]string{false: "live reauth and bounded retry", true: "changed policy rejected"}[mutate], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vstClient(cloud)
			var wire, retries, reauths, options atomic.Int32
			var retained *blockstorage.VolumeRetypeOpts
			const body = `{"os-retype":{"migration_policy":"never","new_type":"new-type"}}`
			cloud.Provider.ReauthFunc = func(context.Context) error {
				if reauths.Add(1) > 1 {
					return errors.New("reauth exhausted")
				}
				cloud.Provider.SetToken("retype-live")
				*retained.MigrationPolicy = "on-demand"
				client.MoreHeaders["x-source"] = "later ordinary header"
				return nil
			}
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, opts *gophercloud.RequestOpts, original error, _ uint) error {
				if !gophercloud.ResponseCodeIs(original, 503) {
					return original
				}
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("retry exhausted"))
				}
				raw, ok := opts.JSONBody.(json.RawMessage)
				if !ok || string(raw) != body || method != http.MethodPost || !strings.HasSuffix(target, vstPath) || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody {
					t.Error(method, target, opts, string(raw))
				}
				if mutate {
					opts.JSONBody = json.RawMessage(`{"os-retype":{"migration_policy":"on-demand","new_type":"new-type"}}`)
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := wire.Add(1)
				token := "test-token"
				if n > 1 {
					token = "retype-live"
				}
				vstPost(t, req, body, token)
				if n == 1 {
					w.WriteHeader(401)
					return
				}
				if n == 2 {
					w.Header().Set("X-Proof", "original retype rejection")
					testcloud.JSON(w, 503, `{"error":"retry retype"}`)
					return
				}
				w.Header().Set("X-Proof", "third retype response")
				w.WriteHeader(203)
				_, _ = w.Write([]byte("opaque retry"))
			})
			result, err := blockstorage.RetypeVolume(bmwContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "requested"}, "new-type", func(next *blockstorage.VolumeRetypeOpts) error {
				options.Add(1)
				policy := "never"
				next.MigrationPolicy = &policy
				retained = next
				return nil
			})
			if options.Load() != 1 || reauths.Load() != 1 || retries.Load() != 1 || client.Microversion != "3.80" {
				t.Fatal(result, err, options.Load(), reauths.Load(), retries.Load())
			}
			if mutate {
				vstNoApplied(t, result, err, "RetypeVolume")
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || native.Method != http.MethodPost || !strings.HasSuffix(native.URL, vstPath) || string(native.Body) != `{"error":"retry retype"}` || native.ResponseHeader.Get("X-Proof") != "original retype rejection" || wire.Load() != 2 {
					t.Fatal(result, err, native, wire.Load())
				}
			} else {
				if err != nil || result == nil || !result.Completed || wire.Load() != 3 {
					t.Fatal(result, err, wire.Load())
				}
				vstPage(t, result.Applied, 203, "opaque retry", "third retype response")
			}
		})
	}
}
