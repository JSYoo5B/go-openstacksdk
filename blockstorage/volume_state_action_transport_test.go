package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const vstPath = bmwBase + "volumes/requested/action"

type vstAction func(context.Context, *gophercloud.ServiceClient, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error)

func vstClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := bmwClient(cloud)
	client.Microversion = "3.80"
	return client
}
func vstPost(t *testing.T, req *http.Request, body, token string) {
	t.Helper()
	actual := bmwRequest(t, req, http.MethodPost, vstPath, "3.80", token)
	if string(actual) != body || req.URL.RawQuery != "" || req.ContentLength != int64(len(body)) || len(req.TransferEncoding) != 0 || req.Header.Get("X-OpenStack-Volume-API-Version") != "3.80" {
		t.Error("volume action changed literal body, scope, framing or version", string(actual), req.URL, req.Header, req.ContentLength, req.TransferEncoding)
	}
	if req.Body != nil {
		if err := req.Body.Close(); err != nil {
			t.Error(err)
		}
	}
}
func vstPage(t *testing.T, page *blockstorage.VolumeActionPage, code int, body, proof string) {
	t.Helper()
	if page == nil || page.StatusCode != code || string(page.Body) != body || page.Header.Get("X-Proof") != proof {
		t.Fatal("volume action acknowledgement", page, code, body, proof)
	}
}
func vstOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "volumes" {
		t.Fatal("volume action operation", err, wrapped)
	}
}
func vstNoApplied(t *testing.T, result *blockstorage.VolumeActionResult, err error, operation string) {
	t.Helper()
	var accepted *resource.ResponseError
	if result == nil || result.VolumeID != "requested" || result.Microversion != "3.80" || result.Applied != nil || result.Completed || len(result.Discovery) != 0 || err == nil || errors.As(err, &accepted) {
		t.Fatal("unaccepted attempt became acknowledgement", result, err, accepted)
	}
	vstOperation(t, err, operation)
}

func TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion(t *testing.T) {
	cases := []struct {
		name, operation, body       string
		call                        vstAction
		read, close, source, cancel bool
	}{
		{"reserve-read", "ReserveVolume", `{"os-reserve":null}`, blockstorage.ReserveVolume, true, false, false, false},
		{"unreserve-close", "UnreserveVolume", `{"os-unreserve":null}`, blockstorage.UnreserveVolume, false, true, false, false},
		{"begin-source", "BeginVolumeDetaching", `{"os-begin_detaching":null}`, blockstorage.BeginVolumeDetaching, false, false, true, false},
		{"abort-joined", "AbortVolumeDetaching", `{"os-roll_detaching":null}`, blockstorage.AbortVolumeDetaching, true, true, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vstClient(cloud)
			ctx, cancel := context.WithCancelCause(bmwContext(t))
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("volume acknowledgement Read"), errors.New("volume acknowledgement Close"), errors.New("volume action custom cancellation")
			reply := "opaque accepted acknowledgement"
			broken := &snapshotReadTransportBody{data: strings.NewReader(reply)}
			if tc.read {
				broken.readError = readCause
			}
			if tc.close {
				broken.closeError = closeCause
			}
			broken.onClose = func() {
				if tc.source {
					client.ResourceBase += "changed/"
				}
				if tc.cancel {
					cancel(cancelCause)
				}
			}
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("accepted volume response cannot retry")
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				vstPost(t, req, tc.body, "test-token")
				return snapshotReadTransportResponse(req, 203, broken, "actual volume action"), nil
			})
			result, err := tc.call(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "requested"})
			var proof *resource.ResponseError
			if result == nil || result.VolumeID != "requested" || result.Microversion != "3.80" || result.Completed || len(result.Discovery) != 0 || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != reply || proof.Header.Get("X-Proof") != "actual volume action" || calls.Load() != 1 || retries.Load() != 0 || broken.closes.Load() != 1 {
				t.Fatal(result, err, proof, calls.Load(), retries.Load(), broken.closes.Load())
			}
			if tc.read && !errors.Is(err, readCause) || tc.close && !errors.Is(err, closeCause) || tc.source && !errors.Is(err, resource.ErrInvalidOption) || tc.cancel && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
				t.Fatal("accepted volume action cause lost", err)
			}
			vstOperation(t, err, tc.operation)
			vstPage(t, result.Applied, 203, reply, "actual volume action")
			proof.Body[0] = '!'
			proof.Header.Set("X-Proof", "changed error")
			vstPage(t, result.Applied, 203, reply, "actual volume action")
			result.Applied.Body[1] = '?'
			if proof.Body[1] != reply[1] {
				t.Fatal("applied acknowledgement aliases response error")
			}
		})
	}
}

func TestVolumeStateActionLiveReauthAndBoundedRetryKeepLiteralNullBody(t *testing.T) {
	cloud := testcloud.New(t)
	client := vstClient(cloud)
	const body = `{"os-roll_detaching":null}`
	var calls, reauths, retries atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error {
		if reauths.Add(1) > 1 {
			return errors.New("bounded volume reauth fixture exhausted")
		}
		cloud.Provider.SetToken("reauth-token")
		client.MoreHeaders["x-source"] = "later ordinary header"
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, original error, _ uint) error {
		if !gophercloud.ResponseCodeIs(original, 503) {
			return original
		}
		if retries.Add(1) > 1 {
			return errors.Join(original, errors.New("bounded volume retry fixture exhausted"))
		}
		old, ok := options.JSONBody.(json.RawMessage)
		if !ok || string(old) != body || method != http.MethodPost || !strings.HasSuffix(target, vstPath) || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody {
			t.Error("volume retry acquired alternate body ownership", method, target, options, string(old))
		}
		options.JSONBody = json.RawMessage(bytes.Clone(old))
		if len(old) != 0 {
			old[0] = '!'
		}
		options.MoreHeaders["X-Native"] = "allowed retry header"
		cloud.Provider.SetToken("retry-token")
		return nil
	}
	reply := "opaque acknowledgement without model JSON"
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		n := calls.Add(1)
		vstPost(t, req, body, map[int32]string{1: "test-token", 2: "reauth-token", 3: "retry-token"}[n])
		switch n {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 503, `{"error":"retry once"}`)
		case 3:
			if req.Header.Get("X-Native") != "allowed retry header" {
				t.Error(req.Header)
			}
			w.Header().Set("X-Proof", "accepted once")
			testcloud.JSON(w, 203, reply)
		default:
			t.Error("volume action replayed", req.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.AbortVolumeDetaching(bmwContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "requested"})
	if err != nil || result == nil || !result.Completed || result.VolumeID != "requested" || result.Microversion != "3.80" || len(result.Discovery) != 0 || calls.Load() != 3 || reauths.Load() != 1 || retries.Load() != 1 || client.Microversion != "3.80" || client.MoreHeaders["x-source"] != "later ordinary header" {
		t.Fatal(result, err, calls.Load(), reauths.Load(), retries.Load())
	}
	vstPage(t, result.Applied, 203, reply, "accepted once")
}

func TestVolumeStateActionRetryMutationStopsBodyVersionAndSourceBeforeResend(t *testing.T) {
	for _, kind := range []string{"body", "version", "source", "joined"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vstClient(cloud)
			ctx, cancel := context.WithCancelCause(bmwContext(t))
			defer cancel(nil)
			callbackCause, cancelCause := errors.New("volume retry callback"), errors.New("volume retry custom cancellation")
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
				if !gophercloud.ResponseCodeIs(original, 503) {
					return original
				}
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("bounded volume mutation fixture exhausted"))
				}
				switch kind {
				case "body", "joined":
					options.JSONBody = json.RawMessage(`{"os-begin_detaching":{}}`)
				case "version":
					options.MoreHeaders["x-openstack-volume-api-version"] = "3.71"
				case "source":
					client.ResourceBase += "changed/"
				}
				if kind == "joined" {
					cancel(cancelCause)
					return callbackCause
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				vstPost(t, req, `{"os-begin_detaching":null}`, "test-token")
				w.Header().Set("X-Proof", "native rejection")
				testcloud.JSON(w, 503, `{"error":"retry once"}`)
			})
			result, err := blockstorage.BeginVolumeDetaching(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "requested"})
			vstNoApplied(t, result, err, "BeginVolumeDetaching")
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Proof") != "native rejection" || calls.Load() != 1 || retries.Load() != 1 {
				t.Fatal(result, err, native, calls.Load(), retries.Load())
			}
			if kind == "joined" && (!errors.Is(err, callbackCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
				t.Fatal("joined volume retry causes lost", err)
			}
		})
	}
}

func TestVolumeStateActionPhysicalRedirectMutationRemainsStickyAfterRepair(t *testing.T) {
	const body = `{"os-reserve":null}`
	for _, kind := range []string{"body", "framing", "header", "target", "body-fault-repair", "header-fault-repair"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vstClient(cloud)
			var calls, redirects, retries atomic.Int32
			var redirected *http.Request
			var firstRejected error
			readCause, closeCause := errors.New("redirected volume body Read"), errors.New("redirected volume body Close")
			broken := &snapshotReadTransportBody{data: strings.NewReader(body), readError: readCause, closeError: closeCause}
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				redirects.Add(1)
				redirected = next
				if len(via) != 1 {
					t.Error("unexpected volume redirect chain", len(via))
				}
				switch kind {
				case "header", "header-fault-repair":
					next.Header.Del("OpenStack-API-Version")
				case "target":
					next.URL.Path += "/changed"
				case "framing":
					next.TransferEncoding = []string{"chunked"}
				case "body", "body-fault-repair":
					if next.Body != nil {
						_ = next.Body.Close()
					}
					if kind == "body" {
						next.Body = io.NopCloser(strings.NewReader(`{"os-reserve":{}}`))
					} else {
						next.Body = broken
					}
				}
				return nil
			}
			if strings.HasSuffix(kind, "fault-repair") {
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					if retries.Add(1) > 1 {
						return errors.Join(original, errors.New("bounded volume physical repair fixture exhausted"))
					}
					firstRejected = original
					if redirected == nil {
						t.Error("no rejected physical volume request to repair")
					} else {
						redirected.Body = io.NopCloser(strings.NewReader(body))
						redirected.ContentLength = int64(len(body))
						redirected.TransferEncoding = nil
						redirected.Header.Set("OpenStack-API-Version", "volume 3.80")
						redirected.Header.Set("X-OpenStack-Volume-API-Version", "3.80")
					}
					return nil
				}
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
				if calls.Add(1) != 1 {
					t.Error("changed physical volume request reached transport", req.Method, req.URL, req.Header)
					return snapshotReadTransportResponse(req, 203, io.NopCloser(strings.NewReader("must not arrive")), "unexpected"), nil
				}
				vstPost(t, req, body, "test-token")
				return &http.Response{StatusCode: 307, Header: http.Header{"Location": {req.URL.String()}}, Body: io.NopCloser(strings.NewReader("redirect")), Request: req}, nil
			})
			result, err := blockstorage.ReserveVolume(bmwContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "requested"})
			vstNoApplied(t, result, err, "ReserveVolume")
			if kind != "body-fault-repair" && !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 || redirects.Load() != 1 {
				t.Fatal(result, err, calls.Load(), redirects.Load())
			}
			if strings.HasSuffix(kind, "fault-repair") && (retries.Load() != 1 || firstRejected == nil || !errors.Is(err, firstRejected)) {
				t.Fatal("physical volume rejection stopped being sticky", err, firstRejected, retries.Load())
			}
			if kind == "body-fault-repair" && (!errors.Is(err, readCause) || !errors.Is(err, closeCause) || broken.closes.Load() != 1) {
				t.Fatal("physical volume IO causes or Close count lost", err, broken.closes.Load())
			}
		})
	}
}
