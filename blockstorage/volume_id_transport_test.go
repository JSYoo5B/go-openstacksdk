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

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestGetVolumeIDAcceptedMemberSchemaAndDescriptorErrorsNeverReturnIDOrFallback(t *testing.T) {
	for _, body := range []string{`{"Volume":{}}`, `{"volume":null}`, `{"volume":[]}`, `{"volume":{"id":"wire","bootable":"bad"}}`, `{"volume":{}} {}`, `{"volume":{"id":"` + string([]byte{0xff}) + `"}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			reader := &getVolumesContractReader{data: strings.NewReader(body)}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.Path != volumeReadContractBase+"volumes/target" {
					t.Error(r.URL)
				}
				return getVolumesContractResponse(r, 200, reader, "current-member"), nil
			})
			result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: "target"})
			var accepted *resource.ResponseError
			if result == nil || result.ID != nil || result.Value != nil || result.Volume != nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || reader.closes.Load() != 1 || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Proof") != "current-member" || string(result.Observed.Body) != body {
				t.Fatal(result, err, calls.Load())
			}
			accepted.Body[0] = '!'
			accepted.Header.Set("X-Proof", "caller")
			if string(result.Observed.Body) != body || result.Observed.Header.Get("X-Proof") != "current-member" {
				t.Fatal("returned proof borrowed ResponseError memory", result)
			}
			volumeReadContractOperation(t, err, "GetVolumeID")
		})
	}
}

func TestGetVolumeIDAcceptedReadCloseCancellationPreservesAllCausesAndProof(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	readCause, closeCause, cancelCause := errors.New("accepted ID read"), errors.New("accepted ID Close"), errors.New("cancel accepted ID Close")
	body := `{"volume":{"id":"must-not-commit"}}`
	reader := &getVolumesContractReader{data: strings.NewReader(body), readError: readCause, closeError: closeCause, onClose: func() { cancel(cancelCause) }}
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return getVolumesContractResponse(r, 200, reader, "admitted"), nil
	})
	result, err := blockstorage.GetVolumeID(ctx, client, blockstorage.GetVolumeIDRequest{NameOrID: "target"})
	var accepted *resource.ResponseError
	if result == nil || result.ID != nil || result.Value != nil || result.Volume != nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || reader.closes.Load() != 1 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || string(result.Observed.Body) != body {
		t.Fatal(result, err, calls.Load(), reader.closes.Load())
	}
	volumeReadContractOperation(t, err, "GetVolumeID")
}

func TestGetVolumeIDTerminalNativeErrorsKeepCurrentRejectionWithoutAcceptedEvidence(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "member 500", true: "list 403 after member 404"}[fallback], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if fallback && n == 1 {
					testcloud.JSON(w, 404, `{"error":"suppressed member"}`)
					return
				}
				path := volumeReadContractBase + "volumes/target"
				code := 500
				if fallback {
					path = volumeReadContractPath
					code = 403
					if r.URL.Query().Get("name") != "target" {
						t.Error(r.URL)
					}
				}
				volumeReadContractWire(t, r, path, "test-token")
				w.Header().Set("X-Proof", "current-rejected")
				testcloud.JSON(w, code, `{"error":"current rejection"}`)
			})
			result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: "target"})
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			wantCode, wantCalls := 500, int32(1)
			if fallback {
				wantCode, wantCalls = 403, 2
			}
			if result == nil || result.ID != nil || result.Value != nil || result.Volume != nil || result.Observed != nil || len(result.Pages) != 0 || calls.Load() != wantCalls || errors.As(err, &accepted) || !errors.As(err, &native) || native.Actual != wantCode || string(native.Body) != `{"error":"current rejection"}` || native.ResponseHeader.Get("X-Proof") != "current-rejected" {
				t.Fatal(result, err, calls.Load())
			}
			volumeReadContractOperation(t, err, "GetVolumeID")
		})
	}
}

func TestGetVolumeIDLaterFallbackFailuresDoNotBorrowSuppressedMemberOrCommitEarlierID(t *testing.T) {
	for _, kind := range []string{"transport", "native rejection", "accepted Close and cancellation"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			laterCause, cancelCause := errors.New("later ID list failure"), errors.New("cancel later ID list")
			first := volumeReadContractPage(`[{"id":"would-have-been-ID","name":"target"}]`, "?marker=next")
			last := volumeReadContractPage(`[]`, "")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				switch calls.Add(1) {
				case 1:
					return getVolumesContractResponse(r, 400, io.NopCloser(strings.NewReader(`{"error":"old member"}`)), "old-member"), nil
				case 2:
					return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(first)), "first-list"), nil
				case 3:
					if kind == "transport" {
						return nil, laterCause
					}
					if kind == "native rejection" {
						return getVolumesContractResponse(r, 403, io.NopCloser(strings.NewReader(`{"error":"current rejection"}`)), "current-rejected"), nil
					}
					reader := &getVolumesContractReader{data: strings.NewReader(last), closeError: laterCause, onClose: func() { cancel(cancelCause) }}
					return getVolumesContractResponse(r, 200, reader, "current-list"), nil
				default:
					t.Error("SDK resend", r.URL)
					return nil, errors.New("unexpected fourth request")
				}
			})
			result, err := blockstorage.GetVolumeID(ctx, client, blockstorage.GetVolumeIDRequest{NameOrID: "target"})
			if result == nil || result.ID != nil || result.Value != nil || result.Volume != nil || result.Observed != nil || calls.Load() != 3 || len(result.Pages) < 1 || result.Pages[0].Header.Get("X-Proof") != "first-list" {
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
				if len(result.Pages) != 1 || errors.As(err, &accepted) || !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Proof") != "current-rejected" {
					t.Fatal(result, err)
				}
			case "accepted Close and cancellation":
				if len(result.Pages) != 2 || !errors.Is(err, laterCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != last || accepted.Header.Get("X-Proof") != "current-list" || errors.As(err, &native) {
					t.Fatal(result, err)
				}
			}
			volumeReadContractOperation(t, err, "GetVolumeID")
		})
	}
}

func TestGetVolumeIDPostResponseSourceDriftKeepsActualMemberAndNoID(t *testing.T) {
	for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
		t.Run(fact, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			body := `{"volume":{"id":"must-not-commit"}}`
			reader := &getVolumesContractReader{data: strings.NewReader(body), onClose: func() {
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
			}}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return getVolumesContractResponse(r, 200, reader, "current-source"), nil
			})
			result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: "target"})
			var accepted *resource.ResponseError
			if result == nil || result.ID != nil || result.Value != nil || result.Volume != nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Proof") != "current-source" || string(result.Observed.Body) != body {
				t.Fatal(result, err, calls.Load())
			}
			volumeReadContractOperation(t, err, "GetVolumeID")
		})
	}
}

func TestGetVolumeIDNativeAuthenticationRetriesKeepOwnedBodylessMemberAndSingleID(t *testing.T) {
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
			t.Error(method, endpoint, count, opts)
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
			testcloud.JSON(w, 200, `{"volume":{"id":{"opaque":9007199254740993}}}`)
		default:
			t.Error("accepted member resent", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: "target"}, func(*blockstorage.VolumeReadOpts) error {
		callbacks.Add(1)
		client.MoreHeaders["x-source"] = "after-capture"
		cloud.Provider.SetToken("callback-live")
		return nil
	})
	if err != nil || result == nil || string(result.ID) != `{"opaque":9007199254740993}` || result.Volume == nil || result.Observed == nil || len(result.Pages) != 0 || result.Observed.Header.Get("X-Proof") != "accepted-once" || calls.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 || callbacks.Load() != 1 {
		t.Fatal(result, err, calls.Load(), reauths.Load(), backoffs.Load(), retries.Load(), callbacks.Load())
	}
}
