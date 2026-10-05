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
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestCinderVolumeAttachmentAcceptedFailuresKeepIndependentActualProofAndJoinedCauses(t *testing.T) {
	for _, action := range cdaActions() {
		for _, kind := range []string{"read", "close", "read and close", "read close source and cancellation"} {
			t.Run(action.name+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				ctx, cancel := context.WithCancelCause(vsaContext(t))
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("accepted attachment Read"), errors.New("accepted attachment Close"), errors.New("attachment caller cancellation")
				reply := []byte{0xff, 0x00, '{', '!'}
				var calls, closes, retries atomic.Int32
				broken := &vsaFaultBody{reader: bytes.NewReader(reply)}
				if kind != "close" {
					broken.readErr = readCause
				}
				if kind != "read" {
					broken.closeErr = closeCause
				}
				broken.onClose = func() {
					closes.Add(1)
					if kind == "read close source and cancellation" {
						client.ResourceBase += "changed/"
						cancel(cancelCause)
					}
				}
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					retries.Add(1)
					return original
				}
				cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
					calls.Add(1)
					cdaPost(t, req, "id", action.body, "3.60", "test-token")
					response := vsaResponse(req, 203, reply, "actual attachment acknowledgement")
					response.Body = broken
					return response, nil
				})
				result, err := action.call(ctx, client, "id")
				vsaOperation(t, err, action.operation)
				var proof *resource.ResponseError
				if result == nil || result.VolumeID != "id" || result.Microversion != "3.60" || result.Completed || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || !bytes.Equal(result.Applied.Body, reply) || result.Applied.Header.Get("X-Proof") != "actual attachment acknowledgement" || !errors.As(err, &proof) || proof.StatusCode != 203 || !bytes.Equal(proof.Body, reply) || proof.Header.Get("X-Proof") != "actual attachment acknowledgement" || calls.Load() != 1 || closes.Load() != 1 || retries.Load() != 0 {
					t.Fatal(result, err, proof, calls.Load(), closes.Load(), retries.Load())
				}
				if kind != "close" && !errors.Is(err, readCause) || kind != "read" && !errors.Is(err, closeCause) || kind == "read close source and cancellation" && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("accepted action lost current IO/source/context cause", err)
				}
				proof.Body[0] = '!'
				proof.Header.Set("X-Proof", "mutated error")
				if !bytes.Equal(result.Applied.Body, reply) || result.Applied.Header.Get("X-Proof") != "actual attachment acknowledgement" {
					t.Fatal("result proof aliases error proof", result.Applied, proof)
				}
				result.Applied.Body[1] = '?'
				result.Applied.Header.Set("X-Proof", "mutated result")
				if proof.Body[1] != reply[1] || proof.Header.Get("X-Proof") != "mutated error" {
					t.Fatal("error proof aliases result proof", result.Applied, proof)
				}
			})
		}
	}
}

func TestCinderVolumeAttachmentNativeAuthRetriesAndPhysicalGuardsPreserveOriginalActionPolicy(t *testing.T) {
	for _, action := range cdaActions() {
		for _, kind := range []string{"success", "expanded rejection", "changed retry body", "removed retry version", "changed source"} {
			t.Run(action.name+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.90")
				var calls, reauths, retries atomic.Int32
				cloud.Provider.ReauthFunc = func(context.Context) error {
					if reauths.Add(1) > 1 {
						return errors.New("bounded attachment reauth exhausted")
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
						return errors.Join(original, errors.New("bounded attachment retry exhausted"))
					}
					old, ok := options.JSONBody.(json.RawMessage)
					if !ok || string(old) != action.body || method != http.MethodPost || !strings.HasSuffix(target, "/volumes/id/action") || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody {
						t.Error("retry lost original action JSON ownership", method, target, options, string(old))
					}
					switch kind {
					case "expanded rejection":
						options.OkCodes = append(options.OkCodes, 400)
					case "changed retry body":
						options.JSONBody = json.RawMessage(`{"os-detach":{"attachment_id":"changed"}}`)
					case "removed retry version":
						for key := range options.MoreHeaders {
							if strings.EqualFold(key, "OpenStack-API-Version") {
								delete(options.MoreHeaders, key)
							}
						}
					case "changed source":
						client.Endpoint += "changed/"
					default:
						options.JSONBody = json.RawMessage(bytes.Clone(old))
						old[0] = '!'
						options.MoreHeaders["X-Native"] = "allowed retry header"
					}
					cloud.Provider.SetToken("retry-token")
					return nil
				}
				reply := []byte{0xff, 0x00, '!'}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					n := calls.Add(1)
					token := map[int32]string{1: "test-token", 2: "reauth-token", 3: "retry-token"}[n]
					cdaPost(t, req, "id", action.body, "3.90", token)
					switch n {
					case 1:
						testcloud.JSON(w, 401, `{"error":"expired"}`)
					case 2:
						w.Header().Set("X-Proof", "native unavailable")
						testcloud.JSON(w, 503, `{"error":"retry once"}`)
					case 3:
						if kind == "expanded rejection" {
							w.Header().Set("X-Proof", "native original-policy rejection")
							testcloud.JSON(w, 400, `{"error":"not accepted"}`)
							return
						}
						if kind != "success" {
							t.Error("changed native request reached HTTP", kind)
						}
						if req.Header.Get("X-Native") != "allowed retry header" {
							t.Error(req.Header)
						}
						w.Header().Set("X-Proof", "accepted attachment retry")
						w.WriteHeader(203)
						_, _ = w.Write(reply)
					default:
						t.Error("attachment replayed beyond bounded native retry", n)
						w.WriteHeader(500)
					}
				})
				result, err := action.call(vsaContext(t), client, "id")
				if reauths.Load() != 1 || retries.Load() != 1 || client.Microversion != "3.90" || client.MoreHeaders["x-source"] != "later ordinary header" {
					t.Fatal(result, err, reauths.Load(), retries.Load(), client.Microversion, client.MoreHeaders)
				}
				if kind == "success" {
					if err != nil || result == nil || !result.Completed || result.Microversion != "3.90" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || !bytes.Equal(result.Applied.Body, reply) || result.Applied.Header.Get("X-Proof") != "accepted attachment retry" || calls.Load() != 3 {
						t.Fatal(result, err, calls.Load())
					}
					return
				}
				cdaNoAdmitted(t, result, err, action.operation)
				var native gophercloud.ErrUnexpectedResponseCode
				wantCode, wantCalls, wantBody, wantProof := 503, int32(2), `{"error":"retry once"}`, "native unavailable"
				if kind == "expanded rejection" {
					wantCode, wantCalls, wantBody, wantProof = 400, 3, `{"error":"not accepted"}`, "native original-policy rejection"
				}
				if !errors.As(err, &native) || native.Actual != wantCode || string(native.Body) != wantBody || native.ResponseHeader.Get("X-Proof") != wantProof || calls.Load() != wantCalls {
					t.Fatal(result, err, native, calls.Load())
				}
				if kind == "expanded rejection" {
					if len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 {
						t.Fatal("retry widened original native status policy", native.Expected)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("mutated native request lost local rejection", err)
				}
			})
		}
	}
	for _, action := range []cdaCase{cdaActions()[0], cdaActions()[2]} {
		for _, kind := range []string{"body", "header", "framing", "target"} {
			t.Run(action.name+" physical "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.90")
				var calls, redirects atomic.Int32
				cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
					redirects.Add(1)
					if len(via) != 1 {
						t.Error("unexpected physical redirect chain", len(via))
					}
					switch kind {
					case "body":
						if next.Body != nil {
							_ = next.Body.Close()
						}
						next.Body = io.NopCloser(strings.NewReader(`{"changed":true}`))
					case "header":
						next.Header.Del("OpenStack-API-Version")
					case "framing":
						next.TransferEncoding = []string{"chunked"}
					case "target":
						next.URL.Path += "/changed"
					}
					return nil
				}
				cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
					if calls.Add(1) != 1 {
						t.Error("changed physical request reached transport", req.URL, req.Header)
						return vsaResponse(req, 203, []byte("unexpected"), "unexpected"), nil
					}
					cdaPost(t, req, "id", action.body, "3.90", "test-token")
					response := vsaResponse(req, 307, []byte("redirect"), "redirect is not an admitted action")
					response.Header.Set("Location", req.URL.String())
					return response, nil
				})
				result, err := action.call(vsaContext(t), client, "id")
				cdaNoAdmitted(t, result, err, action.operation)
				if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 || redirects.Load() != 1 {
					t.Fatal("physical action mutation sent or fabricated acknowledgement", result, err, calls.Load(), redirects.Load())
				}
			})
		}
	}
}
