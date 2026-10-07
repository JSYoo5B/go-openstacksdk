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

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestVolumeImageMetadataAcceptedFailuresRetainOnlyCurrentPhaseProofAndJoinedCauses(t *testing.T) {
	for _, phase := range []string{"set", "member", "first deletion after member", "second explicit deletion"} {
		for _, kind := range []string{"read", "close", "joined IO source cancellation"} {
			t.Run(phase+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				ctx, cancel := context.WithCancelCause(vsaContext(t))
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("image metadata accepted Read"), errors.New("image metadata accepted Close"), errors.New("image metadata caller cancellation")
				reply := []byte{0xff, 0x00, '{', '!'}
				member := `{"volume":{"volume_image_metadata":{"a":"1","b":"2"}}}`
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
					if kind == "joined IO source cancellation" {
						client.ResourceBase += "changed/"
						cancel(cancelCause)
					}
				}
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					retries.Add(1)
					return original
				}
				cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					if phase == "first deletion after member" && n == 1 {
						vimRequest(t, req, "id", "", "3.60", "test-token")
						return vsaResponse(req, 200, []byte(member), "earlier member"), nil
					}
					if phase == "second explicit deletion" && n == 1 {
						vimRequest(t, req, "id", `{"os-unset_image_metadata":{"key":"a"}}`, "3.60", "test-token")
						return vsaResponse(req, 200, []byte("earlier acknowledgement"), "earlier key"), nil
					}
					body := `{"os-set_image_metadata":{"metadata":{"name":"value"}}}`
					if phase == "member" {
						body = ""
					}
					if phase == "first deletion after member" {
						body = `{"os-unset_image_metadata":{"key":"a"}}`
					}
					if phase == "second explicit deletion" {
						body = `{"os-unset_image_metadata":{"key":"b"}}`
					}
					vimRequest(t, req, "id", body, "3.60", "test-token")
					response := vsaResponse(req, 203, reply, "current accepted phase")
					response.Body = broken
					return response, nil
				})
				var current *rest.Response
				var err error
				if phase == "set" {
					result, callErr := blockstorage.SetVolumeImageMetadata(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "id"}, blockstorage.WithVolumeImageMetadataValue("name", "value"))
					err = callErr
					if result == nil || result.Completed || result.VolumeID != "id" || result.Microversion != "3.60" || len(result.Discovery) != 0 {
						t.Fatal(result, err)
					}
					current = result.Applied
					vsaOperation(t, err, "SetVolumeImageMetadata")
				} else {
					option := blockstorage.WithVolumeImageMetadataDeleteAll()
					if phase == "second explicit deletion" {
						option = blockstorage.WithVolumeImageMetadataDeleteKeys("a", "b", "later")
					}
					result, callErr := blockstorage.DeleteVolumeImageMetadata(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "id"}, option)
					err = callErr
					if result == nil || result.Completed || result.VolumeID != "id" || result.Microversion != "3.60" || len(result.Discovery) != 0 {
						t.Fatal(result, err)
					}
					vsaOperation(t, err, "DeleteVolumeImageMetadata")
					switch phase {
					case "member":
						current = result.Observed
						if result.Failed != nil || len(result.Deleted) != 0 {
							t.Fatal("member failure invented current deletion", result)
						}
					case "first deletion after member":
						if result.Observed == nil || string(result.Observed.Body) != member || result.Observed.Header.Get("X-Proof") != "earlier member" || len(result.Deleted) != 0 || result.Failed == nil || result.Failed.Key != "a" {
							t.Fatal(result)
						}
						current = result.Failed.Response
					case "second explicit deletion":
						if result.Observed != nil || len(result.Deleted) != 1 || result.Deleted[0].Key != "a" || result.Deleted[0].Response == nil || string(result.Deleted[0].Response.Body) != "earlier acknowledgement" || result.Failed == nil || result.Failed.Key != "b" {
							t.Fatal(result)
						}
						current = result.Failed.Response
					}
				}
				var proof *resource.ResponseError
				wantCalls := int32(1)
				if phase == "first deletion after member" || phase == "second explicit deletion" {
					wantCalls = 2
				}
				if current == nil || current.StatusCode != 203 || !bytes.Equal(current.Body, reply) || current.Header.Get("X-Proof") != "current accepted phase" || !errors.As(err, &proof) || proof.StatusCode != 203 || !bytes.Equal(proof.Body, reply) || proof.Header.Get("X-Proof") != "current accepted phase" || calls.Load() != wantCalls || closes.Load() != 1 || retries.Load() != 0 {
					t.Fatal(current, err, proof, calls.Load(), closes.Load(), retries.Load())
				}
				if kind != "close" && !errors.Is(err, readCause) || kind != "read" && !errors.Is(err, closeCause) || kind == "joined IO source cancellation" && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("current accepted phase lost IO/source/context causes", err)
				}
				proof.Body[0] = '!'
				proof.Header.Set("X-Proof", "changed error")
				if !bytes.Equal(current.Body, reply) || current.Header.Get("X-Proof") != "current accepted phase" {
					t.Fatal("current response aliases error proof", current, proof)
				}
				current.Body[1] = '?'
				if proof.Body[1] != reply[1] {
					t.Fatal("error proof aliases result response")
				}
			})
		}
	}
}

func TestVolumeImageMetadataNativeRetriesAndPhysicalMutationGuardsKeepBatchPolicy(t *testing.T) {
	for _, mode := range []string{"set", "explicit batch", "all batch"} {
		for _, kind := range []string{"success", "expanded rejection", "changed body", "removed version"} {
			t.Run(mode+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.90")
				var calls, reauths, retries atomic.Int32
				body := `{"os-set_image_metadata":{"metadata":{"name":"value"}}}`
				operation := "SetVolumeImageMetadata"
				if mode == "explicit batch" {
					body = `{"os-unset_image_metadata":{"key":"a"}}`
					operation = "DeleteVolumeImageMetadata"
				}
				if mode == "all batch" {
					body = ""
					operation = "DeleteVolumeImageMetadata"
				}
				member := `{"volume":{"volume_image_metadata":{"a":null,"b":false}}}`
				cloud.Provider.ReauthFunc = func(context.Context) error {
					if reauths.Add(1) > 1 {
						return errors.New("bounded image metadata reauth exhausted")
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
						return errors.Join(original, errors.New("bounded image metadata retry exhausted"))
					}
					old, ok := options.JSONBody.(json.RawMessage)
					if mode == "all batch" {
						if options.JSONBody != nil || method != http.MethodGet || !strings.HasSuffix(target, "/volumes/id") || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody {
							t.Error("member retry acquired body or alternate target", method, target, options)
						}
					} else if !ok || string(old) != body || method != http.MethodPost || !strings.HasSuffix(target, "/volumes/id/action") || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody {
						t.Error("native retry changed JSON ownership", method, target, options, string(old))
					}
					switch kind {
					case "expanded rejection":
						options.OkCodes = append(options.OkCodes, 400)
					case "changed body":
						options.JSONBody = json.RawMessage(`{"os-unset_image_metadata":{"key":"changed"}}`)
					case "removed version":
						for key := range options.MoreHeaders {
							if strings.EqualFold(key, "OpenStack-API-Version") {
								delete(options.MoreHeaders, key)
							}
						}
					default:
						if mode != "all batch" {
							options.JSONBody = json.RawMessage(bytes.Clone(old))
							old[0] = '!'
						}
						options.MoreHeaders["X-Native"] = "owned retry header"
					}
					cloud.Provider.SetToken("retry-token")
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					n := calls.Add(1)
					token := map[int32]string{1: "test-token", 2: "reauth-token", 3: "retry-token", 4: "retry-token", 5: "retry-token"}[n]
					currentBody := body
					if n == 4 && mode == "all batch" {
						currentBody = `{"os-unset_image_metadata":{"key":"a"}}`
					}
					if n == 5 && mode == "all batch" {
						currentBody = `{"os-unset_image_metadata":{"key":"b"}}`
					}
					if n == 4 && mode == "explicit batch" {
						currentBody = `{"os-unset_image_metadata":{"key":"b"}}`
					}
					vimRequest(t, req, "id", currentBody, "3.90", token)
					switch n {
					case 1:
						testcloud.JSON(w, 401, `{"error":"expired"}`)
					case 2:
						w.Header().Set("X-Proof", "native unavailable")
						testcloud.JSON(w, 503, `{"error":"retry once"}`)
					case 3:
						if kind == "expanded rejection" {
							w.Header().Set("X-Proof", "native original policy")
							testcloud.JSON(w, 400, `{"error":"rejected"}`)
							return
						}
						if kind != "success" {
							t.Error("changed image metadata request reached HTTP", kind)
						}
						if req.Header.Get("X-Native") != "owned retry header" {
							t.Error(req.Header)
						}
						w.Header().Set("X-Proof", "current retried key")
						if mode == "all batch" {
							testcloud.JSON(w, 203, member)
						} else {
							w.WriteHeader(203)
							_, _ = w.Write([]byte{0xff, 0x00})
						}
					case 4, 5:
						if kind != "success" || mode == "set" || n == 5 && mode != "all batch" {
							t.Error("batch continued after failure", kind, mode)
						}
						w.Header().Set("X-Proof", "next batch key")
						w.WriteHeader(399)
						_, _ = w.Write([]byte{0xff, '!'})
					default:
						t.Error("image metadata exceeded bounded native requests", n)
						w.WriteHeader(500)
					}
				})
				var err error
				var current *rest.Response
				var completed bool
				var deleted *blockstorage.VolumeImageMetadataDeleteResult
				if mode == "set" {
					result, callErr := blockstorage.SetVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, blockstorage.WithVolumeImageMetadataValue("name", "value"))
					err = callErr
					if result == nil || len(result.Discovery) != 0 || result.Microversion != "3.90" {
						t.Fatal(result, err)
					}
					current, completed = result.Applied, result.Completed
				} else {
					option := blockstorage.WithVolumeImageMetadataDeleteKeys("a", "b")
					if mode == "all batch" {
						option = blockstorage.WithVolumeImageMetadataDeleteAll()
					}
					result, callErr := blockstorage.DeleteVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, option)
					err = callErr
					deleted = result
					if result == nil || len(result.Discovery) != 0 || mode != "all batch" && result.Observed != nil || result.Microversion != "3.90" {
						t.Fatal(result, err)
					}
					completed = result.Completed
					if mode == "all batch" {
						current = result.Observed
					} else if result.Failed != nil {
						current = result.Failed.Response
					} else if len(result.Deleted) > 0 {
						current = result.Deleted[0].Response
					}
				}
				if reauths.Load() != 1 || retries.Load() != 1 || client.Microversion != "3.90" || client.MoreHeaders["x-source"] != "later ordinary header" {
					t.Fatal(err, reauths.Load(), retries.Load(), client.Microversion, client.MoreHeaders)
				}
				if kind == "success" {
					wantCalls := int32(3)
					if mode != "set" {
						wantCalls = 4
						if mode == "all batch" {
							wantCalls = 5
						}
						if deleted.Failed != nil || len(deleted.Deleted) != 2 || deleted.Deleted[0].Key != "a" || deleted.Deleted[1].Key != "b" || deleted.Deleted[1].Response == nil || deleted.Deleted[1].Response.StatusCode != 399 || deleted.Deleted[1].Response.Header.Get("X-Proof") != "next batch key" {
							t.Fatal(deleted)
						}
					}
					wantBody := []byte{0xff, 0x00}
					if mode == "all batch" {
						wantBody = []byte(member)
					}
					if err != nil || !completed || current == nil || current.StatusCode != 203 || !bytes.Equal(current.Body, wantBody) || current.Header.Get("X-Proof") != "current retried key" || calls.Load() != wantCalls {
						t.Fatal(current, err, completed, calls.Load())
					}
					return
				}
				vsaOperation(t, err, operation)
				var proof *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				wantCode, wantCalls := 503, int32(2)
				if kind == "expanded rejection" {
					wantCode, wantCalls = 400, 3
				}
				if completed || current != nil || errors.As(err, &proof) || !errors.As(err, &native) || native.Actual != wantCode || calls.Load() != wantCalls {
					t.Fatal(current, deleted, err, proof, native, calls.Load())
				}
				if deleted != nil {
					if len(deleted.Deleted) != 0 || mode == "all batch" && deleted.Failed != nil || mode != "all batch" && (deleted.Failed == nil || deleted.Failed.Key != "a") {
						t.Fatal("native failed-key proof borrowed earlier state", deleted)
					}
				}
				if kind == "expanded rejection" {
					if len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || native.ResponseHeader.Get("X-Proof") != "native original policy" || string(native.Body) != `{"error":"rejected"}` {
						t.Fatal(native)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("mutated retry lost local cause", err)
				}
			})
		}
	}
	for _, mode := range []string{"member", "set", "explicit deletion"} {
		for _, kind := range []string{"header", "target", "body or framing", "injected GET body Close fault"} {
			if kind == "injected GET body Close fault" && mode != "member" {
				continue
			}
			t.Run(mode+" physical "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.90")
				var calls, redirects, closes atomic.Int32
				closeCause := errors.New("injected member GET body Close")
				injected := &vsaFaultBody{reader: bytes.NewReader([]byte(`{}`)), closeErr: closeCause, onClose: func() { closes.Add(1) }}
				body := ""
				operation := "DeleteVolumeImageMetadata"
				if mode == "set" {
					body = `{"os-set_image_metadata":{"metadata":{}}}`
					operation = "SetVolumeImageMetadata"
				}
				if mode == "explicit deletion" {
					body = `{"os-unset_image_metadata":{"key":"a"}}`
				}
				cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
					redirects.Add(1)
					if len(via) != 1 {
						t.Error("unexpected physical metadata redirect chain", len(via))
					}
					switch kind {
					case "injected GET body Close fault":
						next.Body = injected
						next.ContentLength = 2
					case "header":
						next.Header.Del("OpenStack-API-Version")
					case "target":
						next.URL.Path += "/changed"
					case "body or framing":
						if mode == "member" {
							next.Body = io.NopCloser(strings.NewReader(`{}`))
							next.ContentLength = 2
						} else {
							if next.Body != nil {
								_ = next.Body.Close()
							}
							next.Body = io.NopCloser(strings.NewReader(`{"changed":true}`))
						}
					}
					return nil
				}
				cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
					if calls.Add(1) != 1 {
						t.Error("changed physical metadata request reached transport", req.URL, req.Header)
						return vsaResponse(req, 203, []byte("unexpected"), "unexpected"), nil
					}
					vimRequest(t, req, "id", body, "3.90", "test-token")
					response := vsaResponse(req, 307, []byte("redirect"), "not admitted phase")
					response.Header.Set("Location", req.URL.String())
					return response, nil
				})
				var err error
				var current *rest.Response
				var completed bool
				if mode == "set" {
					result, callErr := blockstorage.SetVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"})
					err = callErr
					if result == nil || len(result.Discovery) != 0 {
						t.Fatal(result, err)
					}
					current, completed = result.Applied, result.Completed
				} else {
					option := blockstorage.WithVolumeImageMetadataDeleteAll()
					if mode == "explicit deletion" {
						option = blockstorage.WithVolumeImageMetadataDeleteKeys("a", "later")
					}
					result, callErr := blockstorage.DeleteVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, option)
					err = callErr
					if result == nil || len(result.Discovery) != 0 || len(result.Deleted) != 0 {
						t.Fatal(result, err)
					}
					completed = result.Completed
					if mode == "member" {
						current = result.Observed
						if result.Failed != nil {
							t.Fatal(result)
						}
					} else {
						if result.Observed != nil || result.Failed == nil || result.Failed.Key != "a" {
							t.Fatal(result)
						}
						current = result.Failed.Response
					}
				}
				vsaOperation(t, err, operation)
				var proof *resource.ResponseError
				if completed || current != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls.Load() != 1 || redirects.Load() != 1 {
					t.Fatal("physical metadata mutation sent or borrowed phase proof", current, err, proof, calls.Load(), redirects.Load())
				}
				if kind == "injected GET body Close fault" && (!errors.Is(err, closeCause) || closes.Load() != 1) {
					t.Fatal("injected member body leaked or lost Close cause", err, closes.Load())
				}
			})
		}
	}
}
