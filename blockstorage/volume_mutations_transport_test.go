package blockstorage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestVolumeMutationAcceptedReadCloseCancellationRetainsCurrentStageAndAllCauses(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, second := range []bool{false, true} {
			t.Run(op+map[bool]string{false: " lookup", true: " mutation"}[second], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("read accepted response"), errors.New("close accepted response"), errors.New("cancel accepted Close")
				lookup := `{"volume":{"id":"returned-한글"}}`
				body, code := lookup, 200
				if second {
					body, code = `{"volume":{"description":"server"}}`, 203
					if op == "SetVolumeBootable" {
						body = string([]byte{0xff, 0, '!'})
						code = 202
					}
				}
				reader := &getVolumesContractReader{data: strings.NewReader(body), readError: readCause, closeError: closeCause, onClose: func() { cancel(cancelCause) }}
				var calls atomic.Int32
				cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
					if calls.Add(1) == 1 {
						vmLookup(t, r, "test-token")
						if second {
							return vmResponse(r, 200, io.NopCloser(strings.NewReader(lookup)), "lookup"), nil
						}
					} else {
						vmMutation(t, r, op, "test-token")
					}
					return vmResponse(r, code, reader, "current"), nil
				})
				got := vmCall(ctx, client, op, "ref", vmChanged(), nil)
				var physical *resource.ResponseError
				wantCalls := int32(1)
				if second {
					wantCalls = 2
				}
				if got.nilResult || got.resolved == nil || got.value != nil || got.volume != nil || calls.Load() != wantCalls || reader.closes.Load() != 1 || !errors.Is(got.err, readCause) || !errors.Is(got.err, closeCause) || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cancelCause) || !errors.As(got.err, &physical) || physical.StatusCode != code || string(physical.Body) != body || physical.Header.Get("X-Proof") != "current" {
					t.Fatal(got, calls.Load(), reader.closes.Load())
				}
				var proofBody []byte
				var proofHeader http.Header
				if second {
					if got.applied == nil || got.resolved.Volume == nil || got.resolved.Observed == nil || got.resolved.Observed.Header.Get("X-Proof") != "lookup" {
						t.Fatal(got)
					}
					proofBody, proofHeader = got.applied.Body, got.applied.Header
				} else {
					if got.applied != nil || got.resolved.Volume != nil || got.resolved.Observed == nil {
						t.Fatal(got)
					}
					proofBody, proofHeader = got.resolved.Observed.Body, got.resolved.Observed.Header
				}
				physical.Body[0] = '!'
				physical.Header.Set("X-Proof", "error mutation")
				if string(proofBody) != body || proofHeader.Get("X-Proof") != "current" {
					t.Fatal("error proof aliases physical page", got)
				}
				volumeReadContractOperation(t, got.err, op)
			})
		}
	}
}

func TestVolumeMutationCurrentNativeRejectionNeverBorrowsSuppressedMemberOrLookupProof(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, transport := range []bool{false, true} {
			t.Run(op+map[bool]string{false: " rejected", true: " transport"}[transport], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
				var calls atomic.Int32
				cause := errors.New("current mutation transport failure")
				cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
					switch calls.Add(1) {
					case 1:
						vmLookup(t, r, "test-token")
						return vmResponse(r, 404, io.NopCloser(strings.NewReader(`{"error":"old missing"}`)), "old missing"), nil
					case 2:
						volumeReadContractWire(t, r, volumeReadContractPath, "test-token")
						if r.URL.Query().Get("name") != "ref" {
							t.Error(r.URL)
						}
						return vmResponse(r, 200, io.NopCloser(strings.NewReader(volumeReadContractPage(`[{"id":"returned-한글","name":"ref"}]`, ""))), "lookup list"), nil
					case 3:
						vmMutation(t, r, op, "test-token")
						if transport {
							return nil, cause
						}
						return vmResponse(r, 403, io.NopCloser(strings.NewReader(`{"error":"current mutation"}`)), "current rejection"), nil
					default:
						t.Error("SDK replay/cleanup", r.URL)
						return nil, errors.New("extra request")
					}
				})
				got := vmCall(context.Background(), client, op, "ref", vmChanged(), nil)
				var physical *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				if got.nilResult || got.resolved == nil || got.resolved.Volume == nil || got.resolved.Observed != nil || len(got.resolved.Pages) != 1 || got.resolved.Pages[0].Header.Get("X-Proof") != "lookup list" || got.applied != nil || got.value != nil || got.volume != nil || calls.Load() != 3 || errors.As(got.err, &physical) {
					t.Fatal(got, calls.Load())
				}
				if transport {
					if !errors.Is(got.err, cause) || errors.As(got.err, &native) {
						t.Fatal(got.err)
					}
				} else if !errors.As(got.err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"current mutation"}` || native.ResponseHeader.Get("X-Proof") != "current rejection" {
					t.Fatal(got.err)
				}
				volumeReadContractOperation(t, got.err, op)
			})
		}
	}
}

func TestVolumeMutationSourceFactsStayFrozenAfterAcceptedLookupOrMutation(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, second := range []bool{false, true} {
			for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
				t.Run(op+map[bool]string{false: " lookup/", true: " mutation/"}[second]+fact, func(t *testing.T) {
					cloud := testcloud.New(t)
					client := vmClient(cloud)
					var calls atomic.Int32
					lookup := `{"volume":{"id":"returned-한글"}}`
					body, code := lookup, 200
					if second {
						body, code = `{"volume":{}}`, 203
						if op == "SetVolumeBootable" {
							body = "opaque"
							code = 202
						}
					}
					reader := &getVolumesContractReader{data: strings.NewReader(body), onClose: func() {
						switch fact {
						case "provider":
							client.ProviderClient = &gophercloud.ProviderClient{}
						case "endpoint":
							client.Endpoint = cloud.Server.URL + "/changed/"
						case "resource base":
							client.ResourceBase = cloud.Server.URL + "/changed/"
						case "type":
							client.Type = "compute"
						case "microversion":
							client.Microversion = "3.61"
						}
					}}
					cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
						if calls.Add(1) == 1 {
							vmLookup(t, r, "test-token")
							if second {
								return vmResponse(r, 200, io.NopCloser(strings.NewReader(lookup)), "lookup"), nil
							}
						} else {
							vmMutation(t, r, op, "test-token")
						}
						return vmResponse(r, code, reader, "source current"), nil
					})
					got := vmCall(context.Background(), client, op, "ref", vmChanged(), nil)
					var physical *resource.ResponseError
					want := int32(1)
					if second {
						want = 2
					}
					if got.nilResult || got.resolved == nil || got.value != nil || got.volume != nil || calls.Load() != want || !errors.Is(got.err, resource.ErrInvalidOption) || !errors.As(got.err, &physical) || physical.StatusCode != code || string(physical.Body) != body || physical.Header.Get("X-Proof") != "source current" {
						t.Fatal(got, calls.Load())
					}
					if second {
						if got.applied == nil || got.resolved.Volume == nil || got.resolved.Observed.Header.Get("X-Proof") != "lookup" {
							t.Fatal(got)
						}
					} else if got.applied != nil || got.resolved.Volume != nil || got.resolved.Observed == nil {
						t.Fatal(got)
					}
				})
			}
		}
	}
}

func TestVolumeMutationNativeReauthAndBoundedRetryPreservePayloadTargetAndLiveHeaders(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		t.Run(op, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls, reauths, retries atomic.Int32
			method, target := http.MethodPut, cloud.Server.URL+volumeReadContractBase+"volumes/"+url.PathEscape(vmID)
			if op == "SetVolumeBootable" {
				method = http.MethodPost
				target += "/action"
			}
			cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("refreshed"); return nil }
			cloud.Provider.RetryFunc = func(_ context.Context, m, u string, opts *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if retries.Add(1) > 1 {
					return errors.New("bounded fixture retry exhausted")
				}
				if m != method || u != target || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody || len(opts.OkCodes) != 300 || opts.OkCodes[0] != 100 || opts.OkCodes[299] != 399 {
					t.Error(m, u, opts)
				}
				cloud.Provider.SetToken("retry-live")
				if opts.MoreHeaders == nil {
					opts.MoreHeaders = map[string]string{}
				}
				opts.MoreHeaders["X-Native"] = "retry"
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if n == 1 {
					vmLookup(t, r, "test-token")
					testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
					return
				}
				token := "test-token"
				if n == 3 {
					token = "refreshed"
				}
				if n == 4 {
					token = "retry-live"
				}
				body := vmMutation(t, r, op, token)
				if op == "UpdateVolume" {
					if string(vmFields(t, body["volume"])["description"]) != `"requested"` {
						t.Error(body)
					}
				} else if string(vmFields(t, body["os-set_bootable"])["bootable"]) != "true" {
					t.Error(body)
				}
				switch n {
				case 2:
					testcloud.JSON(w, 401, `{"error":"expired"}`)
				case 3:
					testcloud.JSON(w, 503, `{"error":"retry"}`)
				case 4:
					if r.Header.Get("X-Native") != "retry" {
						t.Error(r.Header)
					}
					w.Header().Set("X-Proof", "accepted once")
					if op == "UpdateVolume" {
						testcloud.JSON(w, 203, `{"volume":{"description":"server"}}`)
					} else {
						w.WriteHeader(202)
						_, _ = w.Write([]byte{0xff, 0})
					}
				default:
					t.Error("unbounded workflow retry", r.URL)
					testcloud.JSON(w, 500, `{"error":"extra"}`)
				}
			})
			got := vmCall(context.Background(), client, op, "ref", vmChanged(), nil)
			if got.err != nil || got.nilResult || got.resolved == nil || got.resolved.Observed == nil || got.applied == nil || got.applied.Header.Get("X-Proof") != "accepted once" || calls.Load() != 4 || reauths.Load() != 1 || retries.Load() != 1 {
				t.Fatal(got, calls.Load(), reauths.Load(), retries.Load())
			}
		})
	}
}

func TestVolumeMutationNativeRetryProtectsOwnedBodyDecoderAndOriginalAcceptedStatus(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, kind := range []string{"body", "raw body", "decoder", "discard response", "status expansion"} {
			t.Run(op+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
				var calls, retries atomic.Int32
				cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, err error, _ uint) error {
					if !gophercloud.ResponseCodeIs(err, 503) {
						return err
					}
					if retries.Add(1) > 1 {
						return errors.New("bounded fixture retry exhausted")
					}
					switch kind {
					case "body":
						opts.JSONBody = map[string]any{"changed": true}
					case "raw body":
						opts.RawBody = strings.NewReader("changed")
					case "decoder":
						opts.JSONResponse = new(any)
					case "discard response":
						opts.KeepResponseBody = false
					case "status expansion":
						opts.OkCodes = append(opts.OkCodes, 400)
					}
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					switch calls.Add(1) {
					case 1:
						vmLookup(t, r, "test-token")
						testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
					case 2:
						vmMutation(t, r, op, "test-token")
						w.Header().Set("X-Proof", "native current")
						testcloud.JSON(w, 503, `{"error":"retry"}`)
					case 3:
						vmMutation(t, r, op, "test-token")
						w.Header().Set("X-Proof", "native current")
						testcloud.JSON(w, 400, `{"error":"widened status"}`)
					default:
						t.Error("unbounded native fixture", r.URL)
						testcloud.JSON(w, 500, `{"error":"extra"}`)
					}
				})
				got := vmCall(context.Background(), client, op, "ref", vmChanged(), nil)
				var physical *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				wantCalls, wantStatus := int32(2), 503
				if kind == "status expansion" {
					wantCalls, wantStatus = 3, 400
				}
				if got.nilResult || got.resolved == nil || got.resolved.Volume == nil || got.applied != nil || got.value != nil || got.volume != nil || calls.Load() != wantCalls || retries.Load() != 1 || errors.As(got.err, &physical) || !errors.As(got.err, &native) || native.Actual != wantStatus || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || native.ResponseHeader.Get("X-Proof") != "native current" {
					t.Fatal(got, calls.Load(), retries.Load())
				}
				if kind != "status expansion" && !errors.Is(got.err, resource.ErrInvalidOption) {
					t.Fatal(got.err)
				}
			})
		}
	}
}

func TestVolumeMutationBlockedHTTPStageCancellationRetainsLookupWithoutInventedApplied(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		t.Run(op, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller canceled blocked mutation")
			started := make(chan struct{})
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					vmLookup(t, r, "test-token")
					w.Header().Set("X-Proof", "lookup")
					testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
					return
				}
				vmMutation(t, r, op, "test-token")
				close(started)
				<-r.Context().Done()
			})
			done := make(chan vmOutcome, 1)
			go func() { done <- vmCall(ctx, client, op, "ref", vmChanged(), nil) }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("mutation never started")
			}
			cancel(cause)
			select {
			case got := <-done:
				var physical *resource.ResponseError
				if got.nilResult || got.resolved == nil || got.resolved.Volume == nil || got.resolved.Observed == nil || got.resolved.Observed.Header.Get("X-Proof") != "lookup" || got.applied != nil || got.value != nil || got.volume != nil || calls.Load() != 2 || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || errors.As(got.err, &physical) {
					t.Fatal(got, calls.Load())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("mutation ignored cancellation")
			}
		})
	}
}

func TestVolumeMutationLookupSchemaAndUnexpectedNativeStatusStopBeforeMutation(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, tc := range []struct {
			code int
			body string
		}{{200, `{}`}, {200, `{"volume":null}`}, {200, `{"volume":{"id":"returned-한글","bootable":"invalid"}}`}, {201, `{"volume":{"id":"returned-한글"}}`}} {
			t.Run(op+tc.body+http.StatusText(tc.code), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					vmLookup(t, r, "test-token")
					if calls.Add(1) != 1 {
						t.Error("lookup failure caused fallback/mutation", r.URL)
					}
					w.Header().Set("X-Proof", "lookup failure")
					testcloud.JSON(w, tc.code, tc.body)
				})
				got := vmCall(context.Background(), client, op, "ref", vmChanged(), nil)
				var physical *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				if got.nilResult || got.resolved == nil || got.resolved.Volume != nil || got.applied != nil || got.volume != nil || got.value != nil || calls.Load() != 1 {
					t.Fatal(got, calls.Load())
				}
				if tc.code == 200 {
					if got.resolved.Observed == nil || !errors.As(got.err, &physical) || physical.StatusCode != 200 || string(physical.Body) != tc.body {
						t.Fatal(got)
					}
				} else if got.resolved.Observed != nil || errors.As(got.err, &physical) || !errors.As(got.err, &native) || native.Actual != 201 {
					t.Fatal(got)
				}
			})
		}
	}
}

func TestVolumeMutationRedirectCannotChangeFixedTargetMethodOrSendSharedCredentials(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		t.Run(op, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					vmLookup(t, r, "test-token")
					testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
				case 2:
					vmMutation(t, r, op, "test-token")
					w.Header().Set("Location", cloud.Server.URL+"/changed/route")
					w.WriteHeader(302)
				default:
					t.Error("redirect received shared credentials", r.URL, r.Header)
					w.WriteHeader(500)
				}
			})
			got := vmCall(context.Background(), client, op, "ref", vmChanged(), nil)
			var physical *resource.ResponseError
			if got.nilResult || got.resolved == nil || got.resolved.Volume == nil || got.applied != nil || got.value != nil || got.volume != nil || calls.Load() != 2 || !errors.Is(got.err, resource.ErrInvalidOption) || errors.As(got.err, &physical) {
				t.Fatal(got, calls.Load())
			}
		})
	}
}
