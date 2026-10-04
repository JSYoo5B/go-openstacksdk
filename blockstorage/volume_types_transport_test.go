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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestVolumeTypesAcceptedReadCloseCancellationRetainsAllCausesAndIndependentCurrentProof(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("accepted read cause"), errors.New("accepted close cause"), errors.New("caller cause on accepted close")
			body := volumeTypeContractPage(`[{"id":"target"}]`, "")
			path := volumeTypeContractPath
			if operation == "GetVolumeType" {
				body = `{"volume_type":{"id":"target"}}`
				path = volumeTypeContractMember
			}
			reader := &getVolumesContractReader{data: strings.NewReader(body), readError: readCause, closeError: closeCause, onClose: func() { cancel(cancelCause) }}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				volumeReadContractWire(t, r, path, "test-token")
				calls.Add(1)
				return getVolumesContractResponse(r, 200, reader, "current-admitted"), nil
			})
			got := volumeTypeContractCall(ctx, client, operation, nil)
			var physical *resource.ResponseError
			if got.nilResult || got.value != nil || got.types != nil || got.typ != nil || calls.Load() != 1 || reader.closes.Load() != 1 || !errors.Is(got.err, readCause) || !errors.Is(got.err, closeCause) || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cancelCause) || !errors.As(got.err, &physical) || physical.StatusCode != 200 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "current-admitted" {
				t.Fatal(got, calls.Load(), reader.closes.Load())
			}
			proof := got.observed
			if operation != "GetVolumeType" {
				if len(got.pages) != 1 || got.observed != nil {
					t.Fatal(got)
				}
				proof = got.pages[0]
			} else if len(got.pages) != 0 {
				t.Fatal(got)
			}
			if proof == nil || string(proof.Body) != body || proof.Header.Get("X-Proof") != "current-admitted" {
				t.Fatal(got)
			}
			physical.Body[0] = '!'
			physical.Header.Set("X-Proof", "error mutated")
			if string(proof.Body) != body || proof.Header.Get("X-Proof") != "current-admitted" {
				t.Fatal("accepted error aliases returned physical proof", got)
			}
			volumeTypeContractOperation(t, got.err, operation)
		})
	}
}

func TestVolumeTypesStrictCanonicalAcceptedEnvelopesNeverProduceLogicalValues(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		bodies := []string{`{"Volume_Types":[]}`, `{"volume_types":null}`, `{"volume_types":{}}`, `{"volume_types":[false]}`, `{"volume_types":[`, `{"volume_types":[]} {}`, `{"volume_types":[{"unknown":"` + string([]byte{0xff}) + `"}]}`}
		if operation == "GetVolumeType" {
			bodies = []string{`{}`, `{"Volume_Type":{}}`, `{"volume_type":null}`, `{"volume_type":[]}`, `{"volume_type":false}`, `{"volume_type":{}} {}`, `{"volume_type":{"unknown":"` + string([]byte{0xff}) + `"}}`}
		}
		for _, body := range bodies {
			t.Run(operation+"/"+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				reader := &getVolumesContractReader{data: strings.NewReader(body)}
				cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					return getVolumesContractResponse(r, 200, reader, "canonical-origin"), nil
				})
				got := volumeTypeContractCall(context.Background(), client, operation, nil)
				var physical *resource.ResponseError
				if got.nilResult || got.value != nil || got.types != nil || got.typ != nil || calls.Load() != 1 || reader.closes.Load() != 1 || !errors.As(got.err, &physical) || physical.StatusCode != 200 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "canonical-origin" {
					t.Fatal(got, calls.Load())
				}
				if operation == "GetVolumeType" {
					if got.observed == nil || len(got.pages) != 0 || string(got.observed.Body) != body {
						t.Fatal(got)
					}
				} else if len(got.pages) != 1 || string(got.pages[0].Body) != body {
					t.Fatal(got)
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeTypesPostResponseSourceChangesKeepCurrentAdmittedProofAndStop(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
			t.Run(operation+"/"+fact, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				body := volumeTypeContractPage(`[{"id":"target"}]`, "?marker=must-not-follow")
				if operation == "GetVolumeType" {
					body = `{"volume_type":{"id":"target"}}`
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
					calls.Add(1)
					return getVolumesContractResponse(r, 200, reader, "source-changed"), nil
				})
				got := volumeTypeContractCall(context.Background(), client, operation, nil)
				var physical *resource.ResponseError
				if got.nilResult || got.value != nil || got.types != nil || got.typ != nil || calls.Load() != 1 || !errors.Is(got.err, resource.ErrInvalidOption) || !errors.As(got.err, &physical) || physical.StatusCode != 200 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "source-changed" {
					t.Fatal(got, calls.Load())
				}
				if operation == "GetVolumeType" {
					if got.observed == nil || len(got.pages) != 0 {
						t.Fatal(got)
					}
				} else if len(got.pages) != 1 {
					t.Fatal(got)
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestGetVolumeTypeFallbackFailuresKeepOnlyActualCurrentPhaseEvidence(t *testing.T) {
	for _, kind := range []string{"transport", "native rejection", "accepted close cancel"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			currentCause, cancelCause := errors.New("current fallback failed"), errors.New("cancel current fallback")
			first := volumeTypeContractPage(`[{"id":"other"}]`, "?marker=last")
			last := volumeTypeContractPage(`[{"id":"actual","name":"target"}]`, "")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				path := volumeTypeContractPath
				if n == 1 {
					path = volumeTypeContractMember
				}
				volumeReadContractWire(t, r, path, "test-token")
				if r.URL.Query().Get("is_public") != "none" || r.URL.Query().Has("name") {
					t.Error(r.URL)
				}
				switch n {
				case 1:
					return getVolumesContractResponse(r, 400, io.NopCloser(strings.NewReader(`{"error":"old rejected member"}`)), "old-member"), nil
				case 2:
					return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(first)), "first-list"), nil
				case 3:
					if r.URL.Query().Get("marker") != "last" {
						t.Error(r.URL)
					}
					if kind == "transport" {
						return nil, currentCause
					}
					if kind == "native rejection" {
						return getVolumesContractResponse(r, 403, io.NopCloser(strings.NewReader(`{"error":"current rejected"}`)), "current-rejected"), nil
					}
					reader := &getVolumesContractReader{data: strings.NewReader(last), closeError: currentCause, onClose: func() { cancel(cancelCause) }}
					return getVolumesContractResponse(r, 200, reader, "current-admitted"), nil
				default:
					t.Error("SDK resent or revived member proof", r.URL)
					return nil, errors.New("extra request")
				}
			})
			result, err := blockstorage.GetVolumeType(ctx, client, blockstorage.GetVolumeTypeRequest{NameOrID: "target"})
			var physical *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.Value != nil || result.Type != nil || result.Observed != nil || calls.Load() != 3 || len(result.Pages) < 1 || result.Pages[0].Header.Get("X-Proof") != "first-list" {
				t.Fatal(result, err, calls.Load())
			}
			switch kind {
			case "transport":
				if len(result.Pages) != 1 || !errors.Is(err, currentCause) || errors.As(err, &physical) || errors.As(err, &native) {
					t.Fatal(result, err)
				}
			case "native rejection":
				if len(result.Pages) != 1 || errors.As(err, &physical) || !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Proof") != "current-rejected" {
					t.Fatal(result, err)
				}
			case "accepted close cancel":
				if len(result.Pages) != 2 || !errors.Is(err, currentCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.As(err, &physical) || physical.StatusCode != 200 || string(physical.Body) != last || physical.Header.Get("X-Proof") != "current-admitted" || errors.As(err, &native) {
					t.Fatal(result, err)
				}
			}
			volumeTypeContractOperation(t, err, "GetVolumeType")
		})
	}
}

func TestGetVolumeTypeAcceptedReadContainingNative404IsTerminalWithoutFallback(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	var calls atomic.Int32
	body := `{"volume_type":{"id":"target"}}`
	nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte(`{"error":"nested"}`)}
	reader := &getVolumesContractReader{data: strings.NewReader(body), readError: nested}
	cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
		volumeReadContractWire(t, r, volumeTypeContractMember, "test-token")
		calls.Add(1)
		return getVolumesContractResponse(r, 200, reader, "accepted-200"), nil
	})
	result, err := blockstorage.GetVolumeType(context.Background(), client, blockstorage.GetVolumeTypeRequest{NameOrID: "target"})
	var physical *resource.ResponseError
	var native gophercloud.ErrUnexpectedResponseCode
	if result == nil || result.Value != nil || result.Type != nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || !errors.As(err, &physical) || physical.StatusCode != 200 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "accepted-200" || !errors.As(err, &native) || native.Actual != 404 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestGetVolumeTypeNativeAuthenticationRetriesKeepNoneQueryAndOneAcceptedProof(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	target := cloud.Server.URL + volumeTypeContractMember + "?is_public=none"
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
			token = "live-entry"
		}
		volumeReadContractWire(t, r, volumeTypeContractMember, token)
		if r.URL.RawQuery != "is_public=none" {
			t.Error("native retry changed fixed query", r.URL)
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
			w.Header().Set("X-Proof", "accepted-member")
			testcloud.JSON(w, 200, `{"volume_type":{"id":"wire","unknown":9007199254740993}}`)
		default:
			t.Error("orchestration replay", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumeType(context.Background(), client, blockstorage.GetVolumeTypeRequest{NameOrID: "target"}, func(*blockstorage.VolumeTypeSearchOpts) error {
		callbacks.Add(1)
		client.MoreHeaders["x-source"] = "after"
		cloud.Provider.SetToken("live-entry")
		return nil
	})
	if err != nil || result == nil || result.Type == nil || result.Observed == nil || result.Observed.StatusCode != 200 || result.Observed.Header.Get("X-Proof") != "accepted-member" || len(result.Pages) != 0 || calls.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 || callbacks.Load() != 1 {
		t.Fatal(result, err, calls.Load(), reauths.Load(), backoffs.Load(), retries.Load(), callbacks.Load())
	}
	if string(result.Type.Body["unknown"]) != "9007199254740993" || string(volumeReadContractFields(t, result.Value)["id"]) != `"wire"` {
		t.Fatal(result)
	}
}

func TestGetVolumeTypeNativeRetryPreservesBodyOwnershipAndOriginalAcceptedStatus(t *testing.T) {
	for _, kind := range []string{"body", "response target", "status expansion"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, method, endpoint string, opts *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if retries.Add(1) > 1 {
					return errors.New("fixture native retry budget exceeded")
				}
				switch kind {
				case "body":
					opts.JSONBody = map[string]any{"changed": true}
				case "response target":
					opts.JSONResponse = new(any)
				case "status expansion":
					opts.OkCodes = append(opts.OkCodes, 201)
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractMember, "test-token")
				if r.URL.RawQuery != "is_public=none" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Proof", "native-rejected")
				switch calls.Add(1) {
				case 1:
					testcloud.JSON(w, 503, `{"error":"retry"}`)
				case 2:
					testcloud.JSON(w, 201, `{"volume_type":{"id":"wire"}}`)
				default:
					t.Error("native fixture exceeded bounded attempts", r.URL)
					testcloud.JSON(w, 500, `{"error":"unexpected retry"}`)
				}
			})
			result, err := blockstorage.GetVolumeType(context.Background(), client, blockstorage.GetVolumeTypeRequest{NameOrID: "target"})
			var physical *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			wantCalls, wantStatus := int32(1), 503
			if kind == "status expansion" {
				wantCalls, wantStatus = 2, 201
			}
			if result == nil || result.Value != nil || result.Type != nil || result.Observed != nil || len(result.Pages) != 0 || calls.Load() != wantCalls || retries.Load() != 1 || errors.As(err, &physical) || !errors.As(err, &native) || native.Actual != wantStatus || !reflect.DeepEqual(native.Expected, []int{200}) || native.ResponseHeader.Get("X-Proof") != "native-rejected" {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
			if kind != "status expansion" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("body/decoder mutation was not rejected before resend", err)
			}
			volumeTypeContractOperation(t, err, "GetVolumeType")
		})
	}
}

func TestVolumeTypesBlockedContinuationCancellationKeepsPriorPageAndCallerCause(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	started := make(chan struct{})
	cause := errors.New("caller interrupted next type page")
	var calls atomic.Int32
	first := volumeTypeContractPage(`[{"id":"partial"}]`, "?marker=blocked")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
		if calls.Add(1) == 1 {
			testcloud.JSON(w, 200, first)
			return
		}
		close(started)
		<-r.Context().Done()
	})
	done := make(chan volumeTypeContractOutcome, 1)
	go func() { done <- volumeTypeContractCall(ctx, client, "SearchVolumeTypes", nil) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("continuation never started")
	}
	cancel(cause)
	select {
	case got := <-done:
		var physical *resource.ResponseError
		if got.nilResult || got.value != nil || got.types != nil || len(got.pages) != 1 || string(got.pages[0].Body) != first || calls.Load() != 2 || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || errors.As(got.err, &physical) {
			t.Fatal(got, calls.Load())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("blocked continuation ignored cancellation")
	}
}
