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

func TestUploadVolumeToImageAcceptedDiscoveryOrActionFailureKeepsCurrentProofAndNoValue(t *testing.T) {
	for _, phase := range []string{"ordinary action", "action after required discovery", "required discovery"} {
		for _, kind := range []string{"Read", "Close", "joined Read Close source cancellation"} {
			t.Run(phase+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.90")
				ctx, cancel := context.WithCancelCause(vsaContext(t))
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("volume image upload accepted Read"), errors.New("volume image upload accepted Close"), errors.New("volume image upload custom cancellation")
				support := `{"version":{"id":"v3.0","min_version":"3.0","max_version":"3.50"}}`
				reply := []byte(`{"os-volume_upload_image":`)
				code := 203
				if phase == "required discovery" {
					reply = []byte(support)
					code = 300
				}
				var calls, closes, retries atomic.Int32
				broken := &vsaFaultBody{reader: bytes.NewReader(reply)}
				if kind != "Close" {
					broken.readErr = readCause
				}
				if kind != "Read" {
					broken.closeErr = closeCause
				}
				broken.onClose = func() {
					closes.Add(1)
					if kind == "joined Read Close source cancellation" {
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
					if phase == "action after required discovery" && n == 1 {
						vsaDiscoveryRequest(t, req, vsaVersionPath)
						return vsaResponse(req, 300, []byte(support), "earlier support"), nil
					}
					if phase == "required discovery" {
						vsaDiscoveryRequest(t, req, vsaVersionPath)
					} else {
						body := vuiBody
						if phase == "action after required discovery" {
							body = `{"os-volume_upload_image":{"force":false,"image_name":"","protected":false}}`
						}
						vuiPost(t, req, "id", body, "3.90", "test-token")
					}
					response := vsaResponse(req, code, reply, "current accepted phase")
					response.Body = broken
					return response, nil
				})
				var options []blockstorage.VolumeImageUploadOption
				if phase != "ordinary action" {
					options = []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadProtected(false)}
				}
				result, err := blockstorage.UploadVolumeToImage(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "", options...)
				vsaOperation(t, err, "UploadVolumeToImage")
				if result == nil || result.VolumeID != "id" || result.Completed || result.Upload != nil {
					t.Fatal(result, err)
				}
				var current *rest.Response
				wantCalls := int32(1)
				switch phase {
				case "ordinary action":
					if len(result.Discovery) != 0 || result.Microversion != "3.90" {
						t.Fatal(result)
					}
					current = result.Applied
				case "action after required discovery":
					wantCalls = 2
					if len(result.Discovery) != 1 || result.Discovery[0].Header.Get("X-Proof") != "earlier support" || string(result.Discovery[0].Body) != support || result.Microversion != "3.90" {
						t.Fatal(result)
					}
					current = result.Applied
				case "required discovery":
					if result.Applied != nil || len(result.Discovery) != 1 {
						t.Fatal(result)
					}
					current = result.Discovery[0]
				}
				var proof *resource.ResponseError
				if current == nil || current.StatusCode != code || !bytes.Equal(current.Body, reply) || current.Header.Get("X-Proof") != "current accepted phase" || !errors.As(err, &proof) || proof.StatusCode != code || !bytes.Equal(proof.Body, reply) || proof.Header.Get("X-Proof") != "current accepted phase" || calls.Load() != wantCalls || closes.Load() != 1 || retries.Load() != 0 {
					t.Fatal(result, err, proof, calls.Load(), closes.Load(), retries.Load())
				}
				if kind != "Close" && !errors.Is(err, readCause) || kind != "Read" && !errors.Is(err, closeCause) || kind == "joined Read Close source cancellation" && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("accepted upload phase lost current causes", err)
				}
				proof.Body[0] = '!'
				proof.Header.Set("X-Proof", "changed error")
				if !bytes.Equal(current.Body, reply) || current.Header.Get("X-Proof") != "current accepted phase" {
					t.Fatal("current phase aliases error proof")
				}
				current.Body[1] = '?'
				if proof.Body[1] != reply[1] {
					t.Fatal("error proof aliases current phase")
				}
				if phase == "action after required discovery" && (string(result.Discovery[0].Body) != support || result.Discovery[0].Header.Get("X-Proof") != "earlier support") {
					t.Fatal("current failure borrowed or mutated earlier discovery")
				}
			})
		}
	}
}

func TestUploadVolumeToImageLiveNativeRetriesAndPhysicalGuardsKeepBodyAndOriginalAdmission(t *testing.T) {
	for _, required := range []bool{false, true} {
		for _, kind := range []string{"success", "expanded rejection", "changed retry body", "removed retry version", "source changed by retry"} {
			t.Run(map[bool]string{false: "ordinary", true: "required"}[required]+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.90")
				var calls, posts, reauths, retries atomic.Int32
				support := `{"version":{"id":"v3.0","min_version":"3.0","max_version":"3.50"}}`
				body := vuiBody
				if required {
					body = `{"os-volume_upload_image":{"force":false,"image_name":"","protected":false}}`
				}
				cloud.Provider.ReauthFunc = func(context.Context) error {
					if reauths.Add(1) > 1 {
						return errors.New("bounded volume upload reauth exhausted")
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
						return errors.Join(original, errors.New("bounded volume upload retry exhausted"))
					}
					old, ok := options.JSONBody.(json.RawMessage)
					if !ok || string(old) != body || method != http.MethodPost || !strings.HasSuffix(target, "/volumes/id/action") || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody {
						t.Error("volume upload retry changed owned action JSON", method, target, options, string(old))
					}
					switch kind {
					case "expanded rejection":
						options.OkCodes = append(options.OkCodes, 400)
					case "changed retry body":
						options.JSONBody = json.RawMessage(`{"os-volume_upload_image":{"force":true,"image_name":""}}`)
					case "removed retry version":
						for key := range options.MoreHeaders {
							if strings.EqualFold(key, "OpenStack-API-Version") {
								delete(options.MoreHeaders, key)
							}
						}
					case "source changed by retry":
						client.ResourceBase += "changed/"
					default:
						options.JSONBody = json.RawMessage(bytes.Clone(old))
						old[0] = '!'
						options.MoreHeaders["X-Native"] = "allowed retry header"
					}
					cloud.Provider.SetToken("retry-token")
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					calls.Add(1)
					if req.Method == http.MethodGet {
						if !required || posts.Load() != 0 {
							t.Error("unexpected extra upload discovery", req.URL)
						}
						vsaDiscoveryRequest(t, req, vsaVersionPath)
						w.Header().Set("X-Proof", "earlier upload support")
						testcloud.JSON(w, 300, support)
						return
					}
					n := posts.Add(1)
					vuiPost(t, req, "id", body, "3.90", map[int32]string{1: "test-token", 2: "reauth-token", 3: "retry-token"}[n])
					switch n {
					case 1:
						testcloud.JSON(w, 401, `{"error":"expired"}`)
					case 2:
						w.Header().Set("X-Proof", "native unavailable")
						testcloud.JSON(w, 503, `{"error":"retry once"}`)
					case 3:
						if kind == "expanded rejection" {
							w.Header().Set("X-Proof", "original-policy native rejection")
							testcloud.JSON(w, 400, `{"error":"not accepted"}`)
							return
						}
						if kind != "success" {
							t.Error("changed upload request reached HTTP", kind)
						}
						if req.Header.Get("X-Native") != "allowed retry header" {
							t.Error(req.Header)
						}
						w.Header().Set("X-Proof", "accepted native retry")
						testcloud.JSON(w, 203, vuiReply)
					default:
						t.Error("upload replayed beyond bounded native retry", n)
						w.WriteHeader(500)
					}
				})
				var options []blockstorage.VolumeImageUploadOption
				if required {
					options = []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadProtected(false)}
				}
				result, err := blockstorage.UploadVolumeToImage(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "", options...)
				wantPages := 0
				extraCalls := int32(0)
				if required {
					wantPages = 1
					extraCalls = 1
				}
				if result == nil || result.VolumeID != "id" || result.Microversion != "3.90" || len(result.Discovery) != wantPages || reauths.Load() != 1 || retries.Load() != 1 || client.MoreHeaders["x-source"] != "later ordinary header" {
					t.Fatal(result, err, calls.Load(), posts.Load(), reauths.Load(), retries.Load())
				}
				if required && (string(result.Discovery[0].Body) != support || result.Discovery[0].Header.Get("X-Proof") != "earlier upload support") {
					t.Fatal(result.Discovery)
				}
				if kind == "success" {
					if err != nil || !result.Completed || result.Applied == nil || result.Applied.StatusCode != 203 || string(result.Applied.Body) != vuiReply || result.Applied.Header.Get("X-Proof") != "accepted native retry" || string(result.Upload) != `{"image_id":"returned/unsafe","large":9007199254740993,"unknown":[null,false]}` || posts.Load() != 3 || calls.Load() != 3+extraCalls {
						t.Fatal(result, err, calls.Load(), posts.Load())
					}
					return
				}
				vsaOperation(t, err, "UploadVolumeToImage")
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				wantCode, wantPosts, wantBody, wantProof := 503, int32(2), `{"error":"retry once"}`, "native unavailable"
				if kind == "expanded rejection" {
					wantCode, wantPosts, wantBody, wantProof = 400, 3, `{"error":"not accepted"}`, "original-policy native rejection"
				}
				if result.Completed || result.Upload != nil || result.Applied != nil || errors.As(err, &proof) || !errors.As(err, &native) || native.Actual != wantCode || string(native.Body) != wantBody || native.ResponseHeader.Get("X-Proof") != wantProof || posts.Load() != wantPosts || calls.Load() != wantPosts+extraCalls {
					t.Fatal(result, err, native, proof, calls.Load(), posts.Load())
				}
				if kind == "expanded rejection" {
					if len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 {
						t.Fatal("native hook broadened original upload admission", native.Expected)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("mutated retry lost local policy cause", err)
				}
			})
		}
	}
	for _, required := range []bool{false, true} {
		for _, kind := range []string{"body", "header", "target", "framing"} {
			t.Run(map[bool]string{false: "ordinary", true: "required"}[required]+" physical "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.90")
				var calls, posts, redirects atomic.Int32
				support := `{"version":{"id":"v3.0","min_version":"3.0","max_version":"3.50"}}`
				body := vuiBody
				if required {
					body = `{"os-volume_upload_image":{"force":false,"image_name":"","protected":false}}`
				}
				cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
					redirects.Add(1)
					if len(via) != 1 {
						t.Error("unexpected upload redirect chain", len(via))
					}
					switch kind {
					case "body":
						if next.Body != nil {
							_ = next.Body.Close()
						}
						next.Body = io.NopCloser(strings.NewReader(`{"changed":true}`))
					case "header":
						next.Header.Del("OpenStack-API-Version")
					case "target":
						next.URL.Path += "/changed"
					case "framing":
						next.TransferEncoding = []string{"chunked"}
					}
					return nil
				}
				cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
					calls.Add(1)
					if req.Method == http.MethodGet {
						if !required || posts.Load() != 0 {
							t.Error("extra physical upload discovery", req.URL)
						}
						vsaDiscoveryRequest(t, req, vsaVersionPath)
						return vsaResponse(req, 300, []byte(support), "earlier discovery"), nil
					}
					if posts.Add(1) != 1 {
						t.Error("changed upload action reached physical transport", req.URL)
						return vsaResponse(req, 203, []byte(vuiReply), "unexpected"), nil
					}
					vuiPost(t, req, "id", body, "3.90", "test-token")
					response := vsaResponse(req, 307, []byte("redirect"), "not an admitted upload action")
					response.Header.Set("Location", req.URL.String())
					return response, nil
				})
				var options []blockstorage.VolumeImageUploadOption
				if required {
					options = []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadProtected(false)}
				}
				result, err := blockstorage.UploadVolumeToImage(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "", options...)
				vsaOperation(t, err, "UploadVolumeToImage")
				var proof *resource.ResponseError
				wantCalls, wantPages := int32(1), 0
				if required {
					wantCalls = 2
					wantPages = 1
				}
				if result == nil || result.Completed || result.Upload != nil || result.Applied != nil || len(result.Discovery) != wantPages || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls.Load() != wantCalls || posts.Load() != 1 || redirects.Load() != 1 {
					t.Fatal("physical mutation sent or borrowed discovery proof", result, err, proof, calls.Load(), posts.Load(), redirects.Load())
				}
				if required && result.Discovery[0].Header.Get("X-Proof") != "earlier discovery" {
					t.Fatal(result.Discovery)
				}
			})
		}
	}
}
