package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestVolumeSnapshotMutationAcceptedFailuresRetainCurrentPhaseAndCauses(t *testing.T) {
	for _, phase := range []string{"create", "create poll", "delete", "delete poll"} {
		for _, fault := range []string{"read", "close", "context", "source"} {
			t.Run(phase+"/"+fault, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := snapshotReadContractClient(cloud)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("snapshot " + phase + " " + fault)
				step, code := int32(1), 203
				body := `{"snapshot":{"id":"snap","status":"available"}}`
				if phase == "create poll" {
					step, code = 2, 200
				}
				if phase == "delete" {
					step, code, body = 2, 202, string([]byte{0xff, 0, 'x'})
				}
				if phase == "delete poll" {
					step, code, body = 3, 200, `{"snapshot":{"status":"deleted"}}`
				}
				broken := &snapshotReadTransportBody{data: strings.NewReader(body)}
				switch fault {
				case "read":
					broken.readError = cause
				case "close":
					broken.closeError = cause
				case "context":
					broken.onClose = func() { cancel(cause) }
				case "source":
					broken.onClose = func() { client.Microversion = "3.99" }
				}
				var calls, retries atomic.Int32
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries.Add(1)
					return err
				}
				cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
					current := calls.Add(1)
					if current == step {
						return snapshotReadTransportResponse(r, code, broken, "current"), nil
					}
					if current == 1 {
						return snapshotReadTransportResponse(r, 202, io.NopCloser(strings.NewReader(`{"snapshot":{"id":"snap","status":"creating"}}`)), "prior"), nil
					}
					if current == 2 && phase == "delete poll" {
						return snapshotReadTransportResponse(r, 204, io.NopCloser(strings.NewReader("")), "applied"), nil
					}
					t.Error("failed accepted phase was replayed", r.Method, r.URL)
					return nil, errors.New("unexpected request")
				})
				var proof *blockstorage.VolumeSnapshotMutationPage
				var err error
				if strings.HasPrefix(phase, "create") {
					result, failed := blockstorage.CreateVolumeSnapshot(ctx, client, blockstorage.CreateVolumeSnapshotRequest{VolumeID: "vol"})
					err = failed
					if result == nil || result.Value != nil || result.Ready != nil || result.Created == nil {
						t.Fatal(result, err)
					}
					proof = result.LastAccepted
					if phase == "create poll" && (result.CreatedValue == nil || result.Created.Header.Get("X-Proof") != "prior") {
						t.Fatal("lost completed create", result)
					}
				} else {
					result, failed := blockstorage.DeleteVolumeSnapshot(ctx, client, blockstorage.DeleteVolumeSnapshotRequest{NameOrID: "snap"}, blockstorage.WithDeleteVolumeSnapshotWait(true))
					err = failed
					if result == nil || result.Deleted != nil || result.Ready != nil || result.Resolved == nil || result.Resolved.Value == nil || result.Applied == nil {
						t.Fatal(result, err)
					}
					proof = result.LastAccepted
					if phase == "delete poll" && result.Applied.Header.Get("X-Proof") != "applied" {
						t.Fatal("lost completed delete acknowledgement", result)
					}
				}
				var physical *resource.ResponseError
				if err == nil || !errors.As(err, &physical) || physical.StatusCode != code || !bytes.Equal(physical.Body, []byte(body)) || physical.Header.Get("X-Proof") != "current" || proof == nil || proof.StatusCode != code || !bytes.Equal(proof.Body, []byte(body)) || proof.Header.Get("X-Proof") != "current" || calls.Load() != step || retries.Load() != 0 || broken.closes.Load() != 1 {
					t.Fatal(err, proof, calls.Load(), retries.Load(), broken.closes.Load())
				}
				if fault == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if fault != "source" && !errors.Is(err, cause) {
					t.Fatal("lost physical/context cause", err)
				}
				if fault == "context" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				proof.Body[0] = '!'
				proof.Header.Set("X-Proof", "changed")
				if !bytes.Equal(physical.Body, []byte(body)) || physical.Header.Get("X-Proof") != "current" {
					t.Fatal("response error aliases result proof", physical)
				}
			})
		}
	}
}

func TestVolumeSnapshotMutationDelete404RequiresCleanReadCloseAndGuard(t *testing.T) {
	for _, fault := range []string{"clean", "read", "read joined EOF", "close", "close EOF", "context", "source", "wrapped callback"} {
		t.Run(fault, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("rejected404 " + fault)
			body := `{"error":"gone"}`
			rejected := &snapshotReadTransportBody{data: strings.NewReader(body)}
			switch fault {
			case "read":
				rejected.readError = cause
			case "read joined EOF":
				rejected.readError = errors.Join(io.EOF, cause)
			case "close":
				rejected.closeError = cause
			case "close EOF":
				rejected.closeError = io.EOF
			case "context":
				rejected.onClose = func() { cancel(cause) }
			case "source":
				rejected.onClose = func() { client.ResourceBase += "changed/" }
			case "wrapped callback":
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					return fmt.Errorf("native wrapper: %w", err)
				}
			}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				switch calls.Add(1) {
				case 1:
					return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"snapshot":{"id":"snap","status":"available"}}`)), "resolved"), nil
				case 2:
					return snapshotReadTransportResponse(r, 204, io.NopCloser(strings.NewReader("")), "applied"), nil
				case 3:
					return snapshotReadTransportResponse(r, 404, rejected, "absent"), nil
				default:
					t.Error("faulty404 replayed")
					return nil, errors.New("unexpected request")
				}
			})
			result, err := blockstorage.DeleteVolumeSnapshot(ctx, client, blockstorage.DeleteVolumeSnapshotRequest{NameOrID: "snap"}, blockstorage.WithDeleteVolumeSnapshotWait(true))
			if result == nil || result.Resolved == nil || result.Applied == nil || result.LastAccepted == nil || result.LastAccepted.StatusCode != 204 || result.LastAccepted.Header.Get("X-Proof") != "applied" || calls.Load() != 3 || rejected.closes.Load() != 1 {
				t.Fatal(result, err, calls.Load(), rejected.closes.Load())
			}
			if fault == "clean" {
				if err != nil || result.Deleted == nil || !*result.Deleted || result.Absent == nil || result.Absent.StatusCode != 404 || string(result.Absent.Body) != body || result.Absent.Header.Get("X-Proof") != "absent" || result.Ready != nil {
					t.Fatal(result, err)
				}
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || result.Deleted != nil || result.Absent != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body {
				t.Fatal("faulty404 established absence or lost native proof", result, err)
			}
			if (fault == "read" || fault == "read joined EOF" || fault == "close" || fault == "context") && !errors.Is(err, cause) {
				t.Fatal("lost rejected-body cause", err)
			}
			if fault == "close EOF" && !errors.Is(err, io.EOF) {
				t.Fatal("ignored Close EOF", err)
			}
			if fault == "context" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if fault == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestVolumeSnapshotMutationCreateRetryCannotReplaceOwnedBodyOrDecoder(t *testing.T) {
	for _, change := range []string{"body", "raw body", "decoder", "retention", "source"} {
		t.Run(change, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls, retries atomic.Int32
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				data, err := io.ReadAll(r.Body)
				if err != nil || r.Method != http.MethodPost || r.URL.Path != snapshotReadContractBasic || string(data) != `{"snapshot":{"force":false,"name":"owned","volume_id":"vol"}}` {
					t.Error("literal request changed", r.Method, r.URL, string(data), err)
				}
				return snapshotReadTransportResponse(r, 500, io.NopCloser(strings.NewReader(`{"error":"retry"}`)), "rejected"), nil
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				switch change {
				case "body":
					options.JSONBody = json.RawMessage(`{"snapshot":{"force":true,"volume_id":"different"}}`)
				case "raw body":
					options.RawBody = strings.NewReader("changed")
				case "decoder":
					options.JSONResponse = &map[string]any{}
				case "retention":
					options.KeepResponseBody = false
				case "source":
					client.Endpoint += "changed/"
				}
				return nil
			}
			result, err := blockstorage.CreateVolumeSnapshot(context.Background(), client, blockstorage.CreateVolumeSnapshotRequest{VolumeID: "vol"}, blockstorage.WithCreateVolumeSnapshotName("owned"))
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 500 || result == nil || result.Created != nil || result.Value != nil || calls.Load() != 1 || retries.Load() != 1 {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
		})
	}
}

func TestVolumeSnapshotMutationDeletePollingPreservesLiveAuthAndNative404Retry(t *testing.T) {
	for _, mode := range []string{"reauth", "retry404", "expanded404"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls, reauths, retries atomic.Int32
			cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("reauth-token"); return nil }
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				if mode == "expanded404" {
					options.OkCodes = append(options.OkCodes, 404)
				}
				cloud.Provider.SetToken("retry-token")
				return nil
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				step := calls.Add(1)
				if r.URL.Path != snapshotReadContractBasic+"/snap" || r.Header.Get("OpenStack-API-Version") != "volume 3.60" || r.Header.Get("X-Source") != "entry" {
					t.Error("selected source changed", r.Method, r.URL, r.Header)
				}
				if step <= 2 && r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error("initial token changed", r.Header)
				}
				if step == 3 && r.Header.Get("X-Auth-Token") != "after-delete" {
					t.Error("poll froze authentication", r.Header)
				}
				if step == 4 {
					want := "retry-token"
					if mode == "reauth" {
						want = "reauth-token"
					}
					if r.Header.Get("X-Auth-Token") != want {
						t.Error("native callback token lost", r.Header, want)
					}
				}
				switch step {
				case 1:
					return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"snapshot":{"id":"snap","status":"available"}}`)), "resolved"), nil
				case 2:
					cloud.Provider.SetToken("after-delete")
					return snapshotReadTransportResponse(r, 204, io.NopCloser(strings.NewReader("")), "applied"), nil
				case 3:
					code := 404
					if mode == "reauth" {
						code = 401
					}
					if mode == "expanded404" {
						code = 500
					}
					return snapshotReadTransportResponse(r, code, io.NopCloser(strings.NewReader(`{"error":"retry"}`)), "retry"), nil
				case 4:
					if mode == "expanded404" {
						return snapshotReadTransportResponse(r, 404, io.NopCloser(strings.NewReader(`{"error":"expanded"}`)), "expanded"), nil
					}
					return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"snapshot":{"status":"deleted"}}`)), "ready"), nil
				default:
					return nil, errors.New("unexpected request")
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			result, err := blockstorage.DeleteVolumeSnapshot(ctx, client, blockstorage.DeleteVolumeSnapshotRequest{NameOrID: "snap"}, blockstorage.WithDeleteVolumeSnapshotWait(true))
			if result == nil || result.Applied == nil || result.Absent != nil || calls.Load() != 4 {
				t.Fatal(result, err, calls.Load())
			}
			if mode == "expanded404" {
				var native gophercloud.ErrUnexpectedResponseCode
				if err == nil || !errors.As(err, &native) || native.Actual != 404 || result.Deleted != nil || result.LastAccepted.StatusCode != 204 || retries.Load() != 1 {
					t.Fatal("expanded policy established absence", result, err, retries.Load())
				}
			} else if err != nil || result.Deleted == nil || !*result.Deleted || result.Ready == nil || result.ReadySnapshot == nil || result.LastAccepted.StatusCode != 200 || (mode == "reauth" && (reauths.Load() != 1 || retries.Load() != 0)) || (mode == "retry404" && (reauths.Load() != 0 || retries.Load() != 1)) {
				t.Fatal(result, err, reauths.Load(), retries.Load())
			}
		})
	}
}
