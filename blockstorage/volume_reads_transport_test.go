package blockstorage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestVolumeReadsAcceptedReadCloseAndCancellationKeepAllCausesAndCurrentProof(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("read accepted bytes"), errors.New("close accepted body"), errors.New("cancel on accepted Close")
			body := `{"volume":{"id":"wire"}}`
			if operation == "ListVolumes" {
				body = `{"volumes":[{"id":"wire"}]}`
			}
			reader := &getVolumesContractReader{data: strings.NewReader(body), readError: readCause, closeError: closeCause, onClose: func() { cancel(cancelCause) }}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return getVolumesContractResponse(r, 200, reader, "physical-current"), nil
			})
			got := volumeReadContractCall(ctx, client, operation)
			var accepted *resource.ResponseError
			if got.nilResult || got.value != nil || got.volume != nil || got.volumes != nil || got.exists != nil || calls.Load() != 1 || reader.closes.Load() != 1 || !errors.Is(got.err, readCause) || !errors.Is(got.err, closeCause) || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cancelCause) || !errors.As(got.err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Proof") != "physical-current" {
				t.Fatal(got, calls.Load(), reader.closes.Load())
			}
			var proof *blockstorage.GetVolumesPage
			if operation == "ListVolumes" {
				if len(got.pages) != 1 || got.observed != nil {
					t.Fatal(got)
				}
				proof = got.pages[0]
			} else {
				if got.observed == nil || len(got.pages) != 0 {
					t.Fatal(got)
				}
				proof = got.observed
			}
			if string(proof.Body) != body || proof.StatusCode != 200 || proof.Header.Get("X-Proof") != "physical-current" {
				t.Fatal(proof)
			}
			accepted.Body[0] = '!'
			accepted.Header.Set("X-Proof", "error mutation")
			if string(proof.Body) != body || proof.Header.Get("X-Proof") != "physical-current" {
				t.Fatal("error and returned proof share memory", proof)
			}
			volumeReadContractOperation(t, got.err, operation)
		})
	}
}

func TestVolumeReadStrictAcceptedEnvelopesKeepPhysicalEvidenceWithoutLogicalValues(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		cases := []string{`{"Volume":{}}`, `{"volume":null}`, `{"volume":[]}`, `{"volume":false}`, `{"volume":{}} {}`, `{"volume":{"id":"` + string([]byte{0xff}) + `"}}`}
		if operation == "ListVolumes" {
			cases = []string{`{"Volumes":[]}`, `{"volumes":null}`, `{"volumes":{}}`, `{"volumes":[`, `{"volumes":[{"id":"` + string([]byte{0xff}) + `"}]}`}
		}
		for _, body := range cases {
			t.Run(operation+"/"+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				reader := &getVolumesContractReader{data: strings.NewReader(body)}
				cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					return getVolumesContractResponse(r, 200, reader, "schema-origin"), nil
				})
				got := volumeReadContractCall(context.Background(), client, operation)
				var proof *resource.ResponseError
				if got.nilResult || got.value != nil || got.volume != nil || got.volumes != nil || got.exists != nil || calls.Load() != 1 || reader.closes.Load() != 1 || !errors.As(got.err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Proof") != "schema-origin" {
					t.Fatal(got, calls.Load())
				}
				if operation == "ListVolumes" {
					if len(got.pages) != 1 || string(got.pages[0].Body) != body {
						t.Fatal(got)
					}
				} else if got.observed == nil || string(got.observed.Body) != body || len(got.pages) != 0 {
					t.Fatal(got)
				}
				volumeReadContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeExistsFallbackFailureNeverBorrowsEarlierRejectedMember(t *testing.T) {
	for _, kind := range []string{"transport", "native rejection", "accepted close plus cancel"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			laterCause, cancelCause := errors.New("current fallback failed"), errors.New("cancel current fallback")
			first := volumeReadContractPage(`[{"id":"different","name":"other"}]`, "?marker=next")
			last := volumeReadContractPage(`[{"id":"last","name":"target"}]`, "")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				switch calls.Add(1) {
				case 1:
					if r.URL.Path != volumeReadContractBase+"volumes/target" {
						t.Error(r.URL)
					}
					return getVolumesContractResponse(r, 400, io.NopCloser(strings.NewReader(`{"error":"old suppressed member"}`)), "old-member"), nil
				case 2:
					if r.URL.Path != volumeReadContractPath || r.URL.Query().Get("name") != "target" {
						t.Error(r.URL)
					}
					return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(first)), "first-list"), nil
				case 3:
					if r.URL.Path != volumeReadContractPath || r.URL.RawQuery != "marker=next" {
						t.Error(r.URL)
					}
					if kind == "transport" {
						return nil, laterCause
					}
					if kind == "native rejection" {
						return getVolumesContractResponse(r, 403, io.NopCloser(strings.NewReader(`{"error":"current rejected page"}`)), "current-rejected"), nil
					}
					reader := &getVolumesContractReader{data: strings.NewReader(last), closeError: laterCause, onClose: func() { cancel(cancelCause) }}
					return getVolumesContractResponse(r, 200, reader, "current-list"), nil
				default:
					t.Error("SDK resend or suppressed old member resurrected", r.URL)
					return nil, errors.New("extra request")
				}
			})
			result, err := blockstorage.VolumeExists(ctx, client, blockstorage.VolumeExistsRequest{NameOrID: "target"})
			if result == nil || result.Exists != nil || result.Value != nil || result.Volume != nil || result.Observed != nil || calls.Load() != 3 || len(result.Pages) < 1 || result.Pages[0].Header.Get("X-Proof") != "first-list" {
				t.Fatal(result, err, calls.Load())
			}
			var accepted *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			switch kind {
			case "transport":
				if len(result.Pages) != 1 || !errors.Is(err, laterCause) || errors.As(err, &accepted) || errors.As(err, &native) {
					t.Fatal(result, err)
				}
			case "native rejection":
				if len(result.Pages) != 1 || errors.As(err, &accepted) || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"current rejected page"}` || native.ResponseHeader.Get("X-Proof") != "current-rejected" {
					t.Fatal(result, err)
				}
			case "accepted close plus cancel":
				if len(result.Pages) != 2 || !errors.Is(err, laterCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != last || accepted.Header.Get("X-Proof") != "current-list" || errors.As(err, &native) {
					t.Fatal(result, err)
				}
			}
			volumeReadContractOperation(t, err, "VolumeExists")
		})
	}
}

func TestListVolumesLaterNativeOrTransportErrorKeepsOnlyActualPriorPages(t *testing.T) {
	for _, rejected := range []bool{false, true} {
		t.Run(map[bool]string{false: "transport", true: "native rejection"}[rejected], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			cause := errors.New("second page transport")
			first := volumeReadContractPage(`[{"id":"partial"}]`, "?marker=later")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(first)), "accepted-first"), nil
				}
				if rejected {
					return getVolumesContractResponse(r, 404, io.NopCloser(strings.NewReader(`{"error":"later missing"}`)), "rejected-later"), nil
				}
				return nil, cause
			})
			result, err := blockstorage.ListVolumes(context.Background(), client)
			var accepted *resource.ResponseError
			if result == nil || result.Value != nil || result.Volumes != nil || len(result.Pages) != 1 || string(result.Pages[0].Body) != first || result.Pages[0].Header.Get("X-Proof") != "accepted-first" || calls.Load() != 2 || err == nil || errors.As(err, &accepted) {
				t.Fatal(result, err, calls.Load())
			}
			if rejected {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Proof") != "rejected-later" {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal(err)
			}
			volumeReadContractOperation(t, err, "ListVolumes")
		})
	}
}

func TestVolumeReadsPostResponseSourceChangesStopWithCurrentAdmittedProof(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
			t.Run(operation+"/"+fact, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				body := `{"volume":{"id":"wire"}}`
				if operation == "ListVolumes" {
					body = volumeReadContractPage(`[{"id":"wire"}]`, "?marker=must-not-fetch")
				}
				reader := &getVolumesContractReader{data: strings.NewReader(body)}
				reader.onClose = func() {
					switch fact {
					case "provider":
						client.ProviderClient = &gophercloud.ProviderClient{}
					case "endpoint":
						client.Endpoint = cloud.Server.URL + "/changed/"
					case "resource base":
						client.ResourceBase = cloud.Server.URL + "/changed/"
					case "type":
						client.Type = "volume"
					case "microversion":
						client.Microversion = "3.61"
					}
				}
				var calls atomic.Int32
				cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					return getVolumesContractResponse(r, 200, reader, "changed-current"), nil
				})
				got := volumeReadContractCall(context.Background(), client, operation)
				var accepted *resource.ResponseError
				if got.nilResult || got.value != nil || got.volume != nil || got.volumes != nil || got.exists != nil || calls.Load() != 1 || !errors.Is(got.err, resource.ErrInvalidOption) || !errors.As(got.err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Proof") != "changed-current" {
					t.Fatal(got, calls.Load())
				}
				if operation == "ListVolumes" {
					if len(got.pages) != 1 {
						t.Fatal(got)
					}
				} else if got.observed == nil || len(got.pages) != 0 {
					t.Fatal(got)
				}
				volumeReadContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeReadBlockedContinuationCancellationRetainsActualPagesAndCause(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "VolumeExists"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("cancel blocked ordinary read")
			entered := make(chan struct{})
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.Path != volumeReadContractPath {
					return getVolumesContractResponse(r, 404, io.NopCloser(strings.NewReader(`{"error":"missing"}`)), "rejected-member"), nil
				}
				if r.URL.Query().Get("marker") == "" {
					return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(volumeReadContractPage(`[{"id":"other","name":"other"}]`, "?marker=next"))), "actual-prior"), nil
				}
				close(entered)
				<-r.Context().Done()
				return nil, r.Context().Err()
			})
			done := make(chan volumeReadContractOutcome, 1)
			go func() { done <- volumeReadContractCall(ctx, client, operation) }()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("second list request did not start")
			}
			cancel(cause)
			select {
			case got := <-done:
				var accepted *resource.ResponseError
				want := int32(2)
				if operation == "VolumeExists" {
					want = 3
				}
				if got.nilResult || got.value != nil || got.volume != nil || got.volumes != nil || got.exists != nil || got.observed != nil || len(got.pages) != 1 || got.pages[0].Header.Get("X-Proof") != "actual-prior" || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || errors.As(got.err, &accepted) || calls.Load() != want {
					t.Fatal(got, calls.Load())
				}
			case <-time.After(3 * time.Second):
				t.Fatal("cancellation did not release ordinary read")
			}
		})
	}
}

func TestGetVolumeByIDNativeAuthenticationAndRetryKeepFixedOwnedRequest(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	target := cloud.Server.URL + volumeReadContractBase + "volumes/target"
	var calls, reauths, backoffs, retries, callbacks atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("refreshed"); return nil }
	cloud.Provider.RetryBackoffFunc = func(_ context.Context, code *gophercloud.ErrUnexpectedResponseCode, _ error, count uint) error {
		backoffs.Add(1)
		if code.Actual != 429 || count != 1 {
			t.Error(code, count)
		}
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, endpoint string, opts *gophercloud.RequestOpts, err error, count uint) error {
		retries.Add(1)
		if !gophercloud.ResponseCodeIs(err, 503) {
			return err
		}
		if method != http.MethodGet || endpoint != target || count != 2 || opts.JSONBody != nil || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody || !reflect.DeepEqual(opts.OkCodes, []int{200}) {
			t.Error(method, endpoint, opts, count)
		}
		if opts.MoreHeaders == nil {
			opts.MoreHeaders = map[string]string{}
		}
		opts.MoreHeaders["X-Retry"] = "native"
		return nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		token := "refreshed"
		if n == 1 {
			token = "callback-live"
		}
		volumeReadContractWire(t, r, volumeReadContractBase+"volumes/target", token)
		if r.URL.RawQuery != "" {
			t.Error("retry changed literal route query", r.URL)
		}
		switch n {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 429, `{"error":"backoff"}`)
		case 3:
			testcloud.JSON(w, 503, `{"error":"retry"}`)
		case 4:
			if r.Header.Get("X-Retry") != "native" {
				t.Error(r.Header)
			}
			w.Header().Set("X-Proof", "accepted-once")
			testcloud.JSON(w, 200, `{"volume":{"id":"wire","unknown":9007199254740993}}`)
		default:
			t.Error("SDK resent accepted GET", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumeByID(context.Background(), client, blockstorage.GetVolumeByIDRequest{ID: "target"}, func(*blockstorage.VolumeReadOpts) error {
		callbacks.Add(1)
		client.MoreHeaders["x-source"] = "after-capture"
		cloud.Provider.SetToken("callback-live")
		return nil
	})
	if err != nil || result == nil || result.Volume == nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 || callbacks.Load() != 1 || result.Observed.Header.Get("X-Proof") != "accepted-once" || string(result.Volume.Body["unknown"]) != "9007199254740993" || string(result.Volume.Body["id"]) != `"wire"` {
		t.Fatal(result, err, calls.Load(), reauths.Load(), backoffs.Load(), retries.Load(), callbacks.Load())
	}
}

func TestVolumeReadNativeRetryCannotChangeBodyDecoderOrAcceptedStatus(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		for _, mode := range []string{"body", "response decoder", "status expansion"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, err error, _ uint) error {
					if !gophercloud.ResponseCodeIs(err, 503) {
						return err
					}
					switch mode {
					case "body":
						opts.JSONBody = map[string]any{"bad": true}
					case "response decoder":
						opts.JSONResponse = new(any)
					case "status expansion":
						opts.OkCodes = append(opts.OkCodes, 201)
					}
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					path := volumeReadContractBase + "volumes/target"
					if operation == "ListVolumes" {
						path = volumeReadContractPath
					}
					volumeReadContractWire(t, r, path, "test-token")
					w.Header().Set("X-Proof", "native-rejected")
					if calls.Add(1) == 1 {
						testcloud.JSON(w, 503, `{"error":"retry"}`)
					} else if operation == "ListVolumes" {
						testcloud.JSON(w, 201, `{"volumes":[]}`)
					} else {
						testcloud.JSON(w, 201, `{"volume":{"id":"wire"}}`)
					}
				})
				got := volumeReadContractCall(context.Background(), client, operation)
				var accepted *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				wantCode, wantCalls := 503, int32(1)
				if mode == "status expansion" {
					wantCode, wantCalls = 201, 2
				}
				if got.nilResult || got.value != nil || got.volume != nil || got.volumes != nil || got.exists != nil || got.observed != nil || len(got.pages) != 0 || calls.Load() != wantCalls || errors.As(got.err, &accepted) || !errors.As(got.err, &native) || native.Actual != wantCode || !reflect.DeepEqual(native.Expected, []int{200}) {
					t.Fatal(got, calls.Load())
				}
				volumeReadContractOperation(t, got.err, operation)
			})
		}
	}
}
