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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestVolumeTypeAccessAcceptedReadCloseCancellationRetainsCorrectStageAndAllCauses(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, second := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: " lookup", true: " access action"}[second], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("cancel accepted close")
				lookup := `{"volume_type":{"id":"returned-한글"}}`
				body := lookup
				code := 200
				if second {
					body = `{"volume_type_access":[{}]}`
					if operation != "GetVolumeTypeAccess" {
						body = string([]byte{0xff, 0x00})
						code = 202
					}
				}
				reader := &getVolumesContractReader{data: strings.NewReader(body), readError: readCause, closeError: closeCause, onClose: func() { cancel(cancelCause) }}
				var calls atomic.Int32
				cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					if n == 1 {
						volumeTypeAccessContractLookup(t, r, "test-token")
						if second {
							return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(lookup)), "lookup"), nil
						}
					} else {
						volumeTypeAccessContractSecond(t, r, operation, "project", "test-token")
					}
					return getVolumesContractResponse(r, code, reader, "current-stage"), nil
				})
				got := volumeTypeAccessContractCall(ctx, client, operation, "target", "project")
				var physical *resource.ResponseError
				wantCalls := int32(1)
				if second {
					wantCalls = 2
				}
				if got.nilResult || got.resolved == nil || got.value != nil || got.accesses != nil || calls.Load() != wantCalls || reader.closes.Load() != 1 || !errors.Is(got.err, readCause) || !errors.Is(got.err, closeCause) || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cancelCause) || !errors.As(got.err, &physical) || physical.StatusCode != code || string(physical.Body) != body || physical.Header.Get("X-Proof") != "current-stage" {
					t.Fatal(got, calls.Load(), reader.closes.Load())
				}
				proof := got.resolved.Observed
				if second {
					proof = got.phase
					if got.resolved.Type == nil || got.resolved.Observed == nil || got.resolved.Observed.Header.Get("X-Proof") != "lookup" {
						t.Fatal("lookup proof replaced by current phase", got)
					}
				} else if got.phase != nil || got.resolved.Type != nil {
					t.Fatal("lookup failure reached second stage", got)
				}
				if proof == nil || string(proof.Body) != body || proof.Header.Get("X-Proof") != "current-stage" {
					t.Fatal(got)
				}
				physical.Body[0] = '!'
				physical.Header.Set("X-Proof", "error changed")
				if string(proof.Body) != body || proof.Header.Get("X-Proof") != "current-stage" {
					t.Fatal("error/current response alias", got)
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestGetVolumeTypeAccessInvalidJSONAndTopLevelShapeRetainOnlyActualAccessProof(t *testing.T) {
	for _, body := range []string{"", `null`, `false`, `[]`, `{} {}`, `{"volume_type_access":`, `{"volume_type_access":{"unknown":"` + string([]byte{0xff}) + `"}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					volumeTypeAccessContractLookup(t, r, "test-token")
					return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(`{"volume_type":{"id":"returned-한글"}}`)), "lookup"), nil
				}
				volumeTypeAccessContractSecond(t, r, "GetVolumeTypeAccess", "", "test-token")
				return getVolumesContractResponse(r, 203, io.NopCloser(strings.NewReader(body)), "access-JSON"), nil
			})
			result, err := blockstorage.GetVolumeTypeAccess(context.Background(), client, blockstorage.GetVolumeTypeAccessRequest{NameOrID: "target"})
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Accesses != nil || result.Resolved == nil || result.Resolved.Type == nil || result.Resolved.Observed == nil || result.Resolved.Observed.Header.Get("X-Proof") != "lookup" || result.Observed == nil || result.Observed.StatusCode != 203 || string(result.Observed.Body) != body || calls.Load() != 2 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "access-JSON" {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestVolumeTypeAccessCurrentNativeRejectionNeverBorrowsSuppressedMemberOrLookupProof(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, transport := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: " native rejection", true: " transport"}[transport], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				cause := errors.New("current second-stage transport")
				cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
					switch calls.Add(1) {
					case 1:
						volumeTypeAccessContractLookup(t, r, "test-token")
						return getVolumesContractResponse(r, 404, io.NopCloser(strings.NewReader(`{"error":"old missing member"}`)), "old-member"), nil
					case 2:
						volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
						if r.URL.RawQuery != "is_public=none" {
							t.Error(r.URL)
						}
						return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(volumeTypeContractPage(`[{"id":"returned-한글","name":"target"}]`, ""))), "lookup-list"), nil
					case 3:
						volumeTypeAccessContractSecond(t, r, operation, "project", "test-token")
						if transport {
							return nil, cause
						}
						return getVolumesContractResponse(r, 403, io.NopCloser(strings.NewReader(`{"error":"current rejected phase"}`)), "current-rejected"), nil
					default:
						t.Error("SDK retry/relookup/cleanup", r.URL)
						return nil, errors.New("extra request")
					}
				})
				got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project")
				var physical *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				if got.nilResult || got.resolved == nil || got.resolved.Type == nil || got.resolved.Observed != nil || len(got.resolved.Pages) != 1 || got.resolved.Pages[0].Header.Get("X-Proof") != "lookup-list" || got.phase != nil || got.value != nil || got.accesses != nil || calls.Load() != 3 || errors.As(got.err, &physical) {
					t.Fatal(got, calls.Load())
				}
				if transport {
					if !errors.Is(got.err, cause) || errors.As(got.err, &native) {
						t.Fatal(got.err)
					}
				} else if !errors.As(got.err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"current rejected phase"}` || native.ResponseHeader.Get("X-Proof") != "current-rejected" {
					t.Fatal(got.err)
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeTypeAccessSourceFactsRemainFrozenAfterLookupAndAcceptedSecondResponse(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, second := range []bool{false, true} {
			for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
				t.Run(operation+map[bool]string{false: " lookup/", true: " second/"}[second]+fact, func(t *testing.T) {
					cloud := testcloud.New(t)
					client := volumeReadContractClient(cloud)
					var calls atomic.Int32
					lookup := `{"volume_type":{"id":"returned-한글"}}`
					body := lookup
					code := 200
					if second {
						body = `{"volume_type_access":[{}]}`
						if operation != "GetVolumeTypeAccess" {
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
							volumeTypeAccessContractLookup(t, r, "test-token")
							if second {
								return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(lookup)), "lookup"), nil
							}
						} else {
							volumeTypeAccessContractSecond(t, r, operation, "project", "test-token")
						}
						return getVolumesContractResponse(r, code, reader, "source-changed"), nil
					})
					got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project")
					var physical *resource.ResponseError
					want := int32(1)
					if second {
						want = 2
					}
					if got.nilResult || got.resolved == nil || got.value != nil || got.accesses != nil || calls.Load() != want || !errors.Is(got.err, resource.ErrInvalidOption) || !errors.As(got.err, &physical) || physical.StatusCode != code || string(physical.Body) != body || physical.Header.Get("X-Proof") != "source-changed" {
						t.Fatal(got, calls.Load())
					}
					if second {
						if got.phase == nil || got.resolved.Type == nil || got.resolved.Observed.Header.Get("X-Proof") != "lookup" {
							t.Fatal(got)
						}
					} else if got.phase != nil || got.resolved.Type != nil || got.resolved.Observed == nil {
						t.Fatal(got)
					}
				})
			}
		}
	}
}

func TestVolumeTypeAccessNativeReauthenticationAndBoundedRetryKeepPayloadTargetAndLiveHeaders(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls, reauths, retries, callbacks atomic.Int32
			method, phase := http.MethodGet, "os-volume-type-access"
			if operation != "GetVolumeTypeAccess" {
				method, phase = http.MethodPost, "action"
			}
			target := cloud.Server.URL + volumeTypeContractPath + "/" + url.PathEscape(volumeTypeAccessContractID) + "/" + phase
			cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("refreshed"); return nil }
			cloud.Provider.RetryFunc = func(_ context.Context, gotMethod, endpoint string, opts *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if retries.Add(1) > 1 {
					return errors.New("bounded fixture retry exhausted")
				}
				if gotMethod != method || endpoint != target || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody || len(opts.OkCodes) != 300 || opts.OkCodes[0] != 100 || opts.OkCodes[299] != 399 {
					t.Error(gotMethod, endpoint, opts)
				}
				if operation == "GetVolumeTypeAccess" && opts.JSONBody != nil {
					t.Error("GET acquired request body", opts)
				}
				cloud.Provider.SetToken("retry-live")
				if opts.MoreHeaders == nil {
					opts.MoreHeaders = map[string]string{}
				}
				opts.MoreHeaders["X-Retry"] = "native"
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if n == 1 {
					volumeTypeAccessContractLookup(t, r, "callback-live")
					testcloud.JSON(w, 200, `{"volume_type":{"id":"returned-한글"}}`)
					return
				}
				token := "callback-live"
				if n == 3 {
					token = "refreshed"
				}
				if n == 4 {
					token = "retry-live"
				}
				volumeTypeAccessContractSecond(t, r, operation, "literal / project", token)
				switch n {
				case 2:
					testcloud.JSON(w, 401, `{"error":"expired"}`)
				case 3:
					testcloud.JSON(w, 503, `{"error":"retry"}`)
				case 4:
					if r.Header.Get("X-Retry") != "native" {
						t.Error(r.Header)
					}
					w.Header().Set("X-Proof", "accepted-once")
					if operation == "GetVolumeTypeAccess" {
						testcloud.JSON(w, 203, `{"volume_type_access":false}`)
					} else {
						w.WriteHeader(202)
						_, _ = w.Write([]byte("opaque"))
					}
				default:
					t.Error("unbounded/native workflow replay", r.URL)
					testcloud.JSON(w, 500, `{"error":"extra request"}`)
				}
			})
			got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "literal / project", func(*blockstorage.VolumeTypeReadOpts) error {
				callbacks.Add(1)
				client.MoreHeaders["x-source"] = "after capture"
				cloud.Provider.SetToken("callback-live")
				return nil
			})
			if got.err != nil || got.nilResult || got.resolved == nil || got.resolved.Observed == nil || got.phase == nil || got.phase.Header.Get("X-Proof") != "accepted-once" || calls.Load() != 4 || reauths.Load() != 1 || retries.Load() != 1 || callbacks.Load() != 1 {
				t.Fatal(got, calls.Load(), reauths.Load(), retries.Load(), callbacks.Load())
			}
		})
	}
}

func TestVolumeTypeAccessNativeRetryCannotReplaceOwnedBodiesDecoderOrBroadenOriginalStatus(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, kind := range []string{"body", "raw body", "decoder", "discard response", "status expansion"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
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
						volumeTypeAccessContractLookup(t, r, "test-token")
						testcloud.JSON(w, 200, `{"volume_type":{"id":"returned-한글"}}`)
					case 2:
						volumeTypeAccessContractSecond(t, r, operation, "project", "test-token")
						w.Header().Set("X-Proof", "native-current")
						testcloud.JSON(w, 503, `{"error":"retry"}`)
					case 3:
						volumeTypeAccessContractSecond(t, r, operation, "project", "test-token")
						w.Header().Set("X-Proof", "native-current")
						testcloud.JSON(w, 400, `{"error":"widened current status"}`)
					default:
						t.Error("unbounded native fixture", r.URL)
						testcloud.JSON(w, 500, `{"error":"extra request"}`)
					}
				})
				got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project")
				var physical *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				wantCalls, wantStatus := int32(2), 503
				if kind == "status expansion" {
					wantCalls, wantStatus = 3, 400
				}
				if got.nilResult || got.resolved == nil || got.resolved.Type == nil || got.resolved.Observed == nil || got.phase != nil || got.value != nil || got.accesses != nil || calls.Load() != wantCalls || retries.Load() != 1 || errors.As(got.err, &physical) || !errors.As(got.err, &native) || native.Actual != wantStatus || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || native.ResponseHeader.Get("X-Proof") != "native-current" {
					t.Fatal(got, calls.Load(), retries.Load())
				}
				if kind != "status expansion" && !errors.Is(got.err, resource.ErrInvalidOption) {
					t.Fatal("mutated ownership reached resend", got.err)
				}
			})
		}
	}
}

func TestVolumeTypeAccessBlockedSecondStageCancellationKeepsLookupWithoutInventedPhaseProof(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller canceled blocked access stage")
			started := make(chan struct{})
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					volumeTypeAccessContractLookup(t, r, "test-token")
					w.Header().Set("X-Proof", "lookup")
					testcloud.JSON(w, 200, `{"volume_type":{"id":"returned-한글"}}`)
					return
				}
				volumeTypeAccessContractSecond(t, r, operation, "project", "test-token")
				close(started)
				<-r.Context().Done()
			})
			done := make(chan volumeTypeAccessContractOutcome, 1)
			go func() { done <- volumeTypeAccessContractCall(ctx, client, operation, "target", "project") }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("second stage never started")
			}
			cancel(cause)
			select {
			case got := <-done:
				var physical *resource.ResponseError
				if got.nilResult || got.resolved == nil || got.resolved.Type == nil || got.resolved.Observed == nil || got.resolved.Observed.Header.Get("X-Proof") != "lookup" || got.phase != nil || got.value != nil || got.accesses != nil || calls.Load() != 2 || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || errors.As(got.err, &physical) {
					t.Fatal(got, calls.Load())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("second stage ignored cancellation")
			}
		})
	}
}

func TestVolumeTypeAccessRedirectCannotChangeFixedSecondStageTargetOrMethod(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					volumeTypeAccessContractLookup(t, r, "test-token")
					testcloud.JSON(w, 200, `{"volume_type":{"id":"returned-한글"}}`)
				case 2:
					volumeTypeAccessContractSecond(t, r, operation, "project", "test-token")
					w.Header().Set("Location", cloud.Server.URL+"/changed/route")
					w.WriteHeader(302)
				default:
					t.Error("redirect received shared token", r.URL, r.Header)
					w.WriteHeader(500)
				}
			})
			got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project")
			var physical *resource.ResponseError
			if got.nilResult || got.resolved == nil || got.resolved.Type == nil || got.resolved.Observed == nil || got.phase != nil || got.value != nil || got.accesses != nil || calls.Load() != 2 || !errors.Is(got.err, resource.ErrInvalidOption) || errors.As(got.err, &physical) {
				t.Fatal(got, calls.Load())
			}
		})
	}
}
