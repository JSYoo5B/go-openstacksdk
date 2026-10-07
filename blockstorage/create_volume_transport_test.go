package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type createVolumeContractTransport func(*http.Request) (*http.Response, error)

func (f createVolumeContractTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
func createVolumeContractResponse(code int, body io.ReadCloser, proof string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"X-Proof": {proof}}, Body: body}
}

type createVolumeContractReader struct {
	data                  *strings.Reader
	readError, closeError error
	onClose               func()
	once                  sync.Once
	closes                atomic.Int32
}

func (b *createVolumeContractReader) Read(p []byte) (int, error) {
	if b.readError != nil {
		n, _ := b.data.Read(p)
		cause := b.readError
		b.readError = nil
		return n, cause
	}
	return b.data.Read(p)
}
func (b *createVolumeContractReader) Close() error {
	b.closes.Add(1)
	b.once.Do(func() {
		if b.onClose != nil {
			b.onClose()
		}
	})
	return b.closeError
}

func TestCreateVolumeTransportAcceptedPhaseReadCloseContextAndSourceFailuresKeepProof(t *testing.T) {
	for _, phase := range []string{"create", "poll", "bootable"} {
		for _, mode := range []string{"read", "close", "context", "source"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				cinder, _ := createVolumeContractClients(cloud)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("accepted " + phase + " " + mode + " failed")
				var calls, retries atomic.Int32
				body := []byte(createVolumeContractBody("creating"))
				failStep, code := int32(1), 202
				if phase == "create" && mode == "read" {
					body = []byte(`{"volume":{"id":"partial`)
				}
				if phase == "poll" {
					body = []byte(createVolumeContractBody("available"))
					failStep, code = 2, 200
				}
				if phase == "bootable" {
					body = []byte{0, 0xff, 'x'}
					failStep, code = 3, 200
				}
				broken := &createVolumeContractReader{data: strings.NewReader(string(body))}
				if mode == "read" {
					broken.readError = cause
				}
				if mode == "close" {
					broken.closeError = cause
				}
				if mode == "context" {
					broken.onClose = func() { cancel(cause) }
				}
				if mode == "source" {
					broken.onClose = func() { cinder.Microversion = "3.99" }
				}
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries.Add(1)
					return err
				}
				cloud.Provider.HTTPClient.Transport = createVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
					step := calls.Add(1)
					if step == failStep {
						return createVolumeContractResponse(code, broken, phase+"-failure"), nil
					}
					if step == 1 {
						return createVolumeContractResponse(202, io.NopCloser(strings.NewReader(createVolumeContractBody("creating"))), "created"), nil
					}
					if step == 2 {
						return createVolumeContractResponse(200, io.NopCloser(strings.NewReader(createVolumeContractBody("available"))), "ready"), nil
					}
					t.Error("accepted error caused another create, action, poll or cleanup", r.Method, r.URL)
					return nil, errors.New("unexpected HTTP")
				})
				options := []blockstorage.CreateVolumeOption{}
				if phase == "bootable" {
					options = append(options, blockstorage.WithCreateVolumeBootable(true))
				}
				result, err := blockstorage.CreateVolume(ctx, cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, options...)
				var accepted *resource.ResponseError
				if result == nil || err == nil || !errors.As(err, &accepted) || accepted.StatusCode != code || !bytes.Equal(accepted.Body, body) || accepted.Header.Get("X-Proof") != phase+"-failure" || calls.Load() != failStep || retries.Load() != 0 || broken.closes.Load() != 1 {
					t.Fatalf("result=%+v error=%v calls=%d retries=%d closes=%d", result, err, calls.Load(), retries.Load(), broken.closes.Load())
				}
				if mode == "source" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, cause) {
					t.Fatal("accepted phase cause lost", err)
				}
				if mode == "context" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				var proofBody []byte
				var proofHeader http.Header
				switch phase {
				case "create":
					if result.Created == nil || result.Created.Volume != nil || result.VolumeID != "" || result.LastAccepted != nil || result.Ready != nil || result.BootableSet != nil {
						t.Fatal(result)
					}
					proofBody, proofHeader = result.Created.Body, result.Created.Header
				case "poll":
					createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
					if result.LastAccepted == nil || result.LastAccepted.Volume != nil || result.Ready != nil || result.BootableSet != nil {
						t.Fatal(result)
					}
					proofBody, proofHeader = result.LastAccepted.Body, result.LastAccepted.Header
				case "bootable":
					createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
					if result.Ready == nil || result.LastAccepted == nil || result.BootableSet == nil || *result.Ready.IsBootable {
						t.Fatal(result)
					}
					proofBody, proofHeader = result.BootableSet.Body, result.BootableSet.Header
				}
				if !bytes.Equal(proofBody, body) || proofHeader.Get("X-Proof") != phase+"-failure" {
					t.Fatal("phase raw proof lost", result)
				}
				accepted.Body[0] = '!'
				accepted.Header.Set("X-Proof", "caller")
				if !bytes.Equal(proofBody, body) || proofHeader.Get("X-Proof") != phase+"-failure" {
					t.Fatal("result proof aliases accepted error")
				}
				createVolumeContractOperation(t, err)
			})
		}
	}
}

func TestCreateVolumeTransportEntrySnapshotsLiveImageTokenAndSourceMutationGuards(t *testing.T) {
	t.Run("name lookup and captured headers use independent live providers", func(t *testing.T) {
		cloud := testcloud.New(t)
		cinder, glance := createVolumeContractClients(cloud)
		imageProvider := &gophercloud.ProviderClient{HTTPClient: *cloud.Server.Client()}
		imageProvider.UseTokenLock()
		imageProvider.SetToken("image-initial")
		glance.ProviderClient = imageProvider
		image := resource.Name("worker")
		var calls, options atomic.Int32
		var retained *blockstorage.CreateVolumeOpts
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				createVolumeContractWire(t, r, http.MethodGet, createVolumeContractGlanceBase+"images", "image-live")
				testcloud.JSON(w, 200, `{"images":[{"id":"selected-image","name":"worker"}]}`)
			case 2:
				createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, "cinder-live")
				body := createVolumeContractEnvelope(t, r)
				if string(body["name"]) != `"original"` || string(body["imageRef"]) != `"selected-image"` {
					t.Error(body)
				}
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 202, createVolumeContractBody("creating"))
			case 3:
				createVolumeContractWire(t, r, http.MethodGet, createVolumeContractPollPath, "cinder-live")
				testcloud.JSON(w, 200, createVolumeContractBody("available"))
			default:
				t.Error("unexpected image or volume phase", r.Method, r.URL)
				w.WriteHeader(500)
			}
		})
		result, err := blockstorage.CreateVolume(context.Background(), cinder, glance, blockstorage.CreateVolumeRequest{Size: 2, Image: &image}, func(v *blockstorage.CreateVolumeOpts) error {
			options.Add(1)
			name := "original"
			v.Attributes.Name = &name
			retained = v
			cinder.MoreHeaders["x-source"] = "changed-cinder"
			glance.MoreHeaders["x-source"] = "changed-image"
			cloud.Provider.SetToken("cinder-live")
			imageProvider.SetToken("image-live")
			return nil
		}, func(*blockstorage.CreateVolumeOpts) error {
			options.Add(1)
			*retained.Attributes.Name = "late mutation"
			return nil
		})
		createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
		if err != nil || result.Ready == nil || calls.Load() != 3 || options.Load() != 2 || glance.MoreHeaders["x-source"] != "changed-image" {
			t.Fatalf("result=%+v error=%v calls=%d options=%d", result, err, calls.Load(), options.Load())
		}
	})
	t.Run("source changes after image name resolution prevent creation", func(t *testing.T) {
		cloud := testcloud.New(t)
		cinder, glance := createVolumeContractClients(cloud)
		image := resource.Name("worker")
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = createVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			createVolumeContractWire(t, r, http.MethodGet, createVolumeContractGlanceBase+"images", "test-token")
			cinder.ResourceBase += "changed/"
			response := createVolumeContractResponse(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"selected-image","name":"worker"}]}`)), "image")
			response.Request = r
			return response, nil
		})
		result, err := blockstorage.CreateVolume(context.Background(), cinder, glance, blockstorage.CreateVolumeRequest{Size: 2, Image: &image})
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
	for _, mutation := range []struct {
		name   string
		change func(*gophercloud.ServiceClient)
	}{{"provider", func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }}, {"endpoint", func(c *gophercloud.ServiceClient) { c.Endpoint += "changed/" }}, {"resource base", func(c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" }}, {"type", func(c *gophercloud.ServiceClient) { c.Type = "image" }}, {"microversion", func(c *gophercloud.ServiceClient) { c.Microversion = "3.99" }}} {
		t.Run("accepted create/"+mutation.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, _ := createVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = createVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, "test-token")
				mutation.change(cinder)
				return createVolumeContractResponse(202, io.NopCloser(strings.NewReader(createVolumeContractBody("creating"))), "created"), nil
			})
			result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2})
			var accepted *resource.ResponseError
			if result == nil || result.Created == nil || string(result.Created.Body) != createVolumeContractBody("creating") || result.Created.Volume != nil || result.LastAccepted != nil || result.Ready != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &accepted) || accepted.StatusCode != 202 || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}

func TestCreateVolumeTransportWaitBudgetExcludesCreateAndBootableUsesParentContext(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, _ := createVolumeContractClients(cloud)
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = createVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
		switch calls.Add(1) {
		case 1:
			if _, bounded := r.Context().Deadline(); bounded {
				t.Error("wait deadline leaked into creation")
			}
			return createVolumeContractResponse(202, &createVolumeContractReader{data: strings.NewReader(createVolumeContractBody("creating")), onClose: func() { time.Sleep(100 * time.Millisecond) }}, "created"), nil
		case 2:
			if _, bounded := r.Context().Deadline(); !bounded {
				t.Error("positive status-wait budget missing")
			}
			return createVolumeContractResponse(200, io.NopCloser(strings.NewReader(createVolumeContractBody("available"))), "ready"), nil
		case 3:
			createVolumeContractWire(t, r, http.MethodPost, createVolumeContractActionPath, "test-token")
			if _, bounded := r.Context().Deadline(); bounded {
				t.Error("wait context leaked into post-ready action")
			}
			time.Sleep(100 * time.Millisecond)
			if r.Context().Err() != nil {
				t.Error("expired status budget canceled action", r.Context().Err())
			}
			return createVolumeContractResponse(200, io.NopCloser(strings.NewReader("opaque")), "bootable"), nil
		default:
			t.Error("unexpected budget-boundary request", r.URL)
			return nil, errors.New("unexpected HTTP")
		}
	})
	timeout := 50 * time.Millisecond
	result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeBootable(true), blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{Timeout: &timeout}))
	createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
	if err != nil || result.Ready == nil || result.BootableSet == nil || string(result.BootableSet.Body) != "opaque" || calls.Load() != 3 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
	t.Run("zero budget permits a nonterminal observation and next poll", func(t *testing.T) {
		cloud := testcloud.New(t)
		cinder, _ := createVolumeContractClients(cloud)
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 202, createVolumeContractBody("creating"))
			case 2:
				testcloud.JSON(w, 200, createVolumeContractBody("creating"))
			case 3:
				testcloud.JSON(w, 200, createVolumeContractBody("available"))
			default:
				t.Error("unexpected zero-budget phase", r.Method, r.URL)
				w.WriteHeader(500)
			}
		})
		zero, interval := time.Duration(0), time.Millisecond
		result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{Timeout: &zero, PollInterval: &interval, ProgressCallback: func(progress int) error {
			callbacks.Add(1)
			if progress != 0 {
				t.Error(progress)
			}
			return nil
		}}))
		createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
		if err != nil || result.Ready == nil || calls.Load() != 3 || callbacks.Load() != 1 {
			t.Fatalf("result=%+v error=%v calls=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
		}
	})
}

func TestCreateVolumeTransportWaitCancellationAndLaterFailuresRetainAcceptedCreation(t *testing.T) {
	for _, mode := range []string{"timer timeout", "callback cancel", "blocked timeout", "later native denial", "later transport cause", "invalid poll identity"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, _ := createVolumeContractClients(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller canceled create waiting")
			callbackCause := errors.New("progress stopped create")
			var calls atomic.Int32
			pollBody := createVolumeContractBody("creating")
			cloud.Provider.HTTPClient.Transport = createVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
				switch calls.Add(1) {
				case 1:
					return createVolumeContractResponse(202, io.NopCloser(strings.NewReader(createVolumeContractBody("creating"))), "created"), nil
				case 2:
					body := pollBody
					if mode == "invalid poll identity" {
						body = `{"volume":{"id":"different","status":"available"}}`
					}
					return createVolumeContractResponse(200, io.NopCloser(strings.NewReader(body)), "last-accepted"), nil
				case 3:
					if mode == "blocked timeout" {
						<-r.Context().Done()
						return nil, r.Context().Err()
					}
					if mode == "later native denial" {
						return createVolumeContractResponse(403, io.NopCloser(strings.NewReader(`{"error":"later denial"}`)), "rejected"), nil
					}
					if mode == "later transport cause" {
						return nil, cause
					}
					t.Error("wait failure issued another request", r.Method, r.URL)
					return nil, errors.New("unexpected HTTP")
				default:
					t.Error("cleanup or bootable after failed wait", r.Method, r.URL)
					return nil, errors.New("unexpected HTTP")
				}
			})
			interval, timeout := time.Hour, 200*time.Millisecond
			policy := blockstorage.CreateVolumeWaitOpts{PollInterval: &interval, Timeout: &timeout}
			if mode == "callback cancel" {
				policy.Timeout = nil
				policy.ProgressCallback = func(int) error { cancel(cause); return callbackCause }
			}
			if mode == "blocked timeout" || mode == "later native denial" || mode == "later transport cause" {
				interval = time.Millisecond
			}
			if mode == "later native denial" || mode == "later transport cause" || mode == "invalid poll identity" {
				policy.Timeout = nil
			}
			result, err := blockstorage.CreateVolume(ctx, cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeBootable(true), blockstorage.WithCreateVolumeWaitPolicy(policy))
			createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
			var accepted *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || result.LastAccepted == nil || result.Ready != nil || result.BootableSet != nil {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			switch mode {
			case "timer timeout":
				if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &accepted) || string(accepted.Body) != pollBody || calls.Load() != 2 {
					t.Fatal(result, err, calls.Load())
				}
			case "callback cancel":
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callbackCause) || !errors.As(err, &accepted) || calls.Load() != 2 {
					t.Fatal(result, err, calls.Load())
				}
			case "blocked timeout":
				if !errors.Is(err, context.DeadlineExceeded) || errors.As(err, &accepted) || string(result.LastAccepted.Body) != pollBody || calls.Load() != 3 {
					t.Fatal(result, err, calls.Load())
				}
			case "later native denial":
				if !errors.As(err, &native) || native.Actual != 403 || errors.As(err, &accepted) || string(result.LastAccepted.Body) != pollBody || calls.Load() != 3 {
					t.Fatal(result, err, calls.Load())
				}
			case "later transport cause":
				if !errors.Is(err, cause) || errors.Is(err, context.Canceled) || errors.As(err, &accepted) || string(result.LastAccepted.Body) != pollBody || result.LastAccepted.Header.Get("X-Proof") != "last-accepted" || calls.Load() != 3 {
					t.Fatal(result, err, calls.Load())
				}
			case "invalid poll identity":
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(result.LastAccepted.Body) != `{"volume":{"id":"different","status":"available"}}` || calls.Load() != 2 {
					t.Fatal(result, err, calls.Load())
				}
			}
			createVolumeContractOperation(t, err)
		})
	}
}

func TestCreateVolumeTransportNativeAuthBackoffRetryOwnsCreateAndBootableJSON(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, _ := createVolumeContractClients(cloud)
	var calls, creates, actions, reauths, backoffs, retries atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("refreshed-token"); return nil }
	cloud.Provider.RetryBackoffFunc = func(_ context.Context, code *gophercloud.ErrUnexpectedResponseCode, _ error, count uint) error {
		backoffs.Add(1)
		if code.Actual != 429 || count != 1 {
			t.Error(code, count)
		}
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, err error, count uint) error {
		retries.Add(1)
		if !gophercloud.ResponseCodeIs(err, 503) {
			return err
		}
		if method != http.MethodPost || options.JSONBody == nil || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody {
			t.Error(method, target, options)
		}
		if target == cloud.Server.URL+createVolumeContractCreatePath {
			if count != 2 || !reflect.DeepEqual(options.OkCodes, []int{202}) {
				t.Error(count, options.OkCodes)
			}
		} else if target == cloud.Server.URL+createVolumeContractActionPath {
			if count != 1 || !reflect.DeepEqual(options.OkCodes, []int{200}) {
				t.Error(count, options.OkCodes)
			}
		} else {
			t.Error("retry changed phase route", target)
		}
		if options.MoreHeaders == nil {
			options.MoreHeaders = map[string]string{}
		}
		options.MoreHeaders["X-Retry"] = "native"
		return nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method == http.MethodGet {
			createVolumeContractWire(t, r, http.MethodGet, createVolumeContractPollPath, "refreshed-token")
			testcloud.JSON(w, 200, createVolumeContractBody("available"))
			return
		}
		if r.URL.Path == createVolumeContractCreatePath {
			count := creates.Add(1)
			token := "refreshed-token"
			if count == 1 {
				token = "test-token"
			}
			createVolumeContractWire(t, r, http.MethodPost, createVolumeContractCreatePath, token)
			body := createVolumeContractEnvelope(t, r)
			if !reflect.DeepEqual(body, map[string]json.RawMessage{"size": json.RawMessage(`2`), "name": json.RawMessage(`"worker"`)}) {
				t.Error(body)
			}
			switch count {
			case 1:
				testcloud.JSON(w, 401, `{"error":"expired"}`)
			case 2:
				testcloud.JSON(w, 429, `{"error":"throttled"}`)
			case 3:
				testcloud.JSON(w, 503, `{"error":"retry"}`)
			default:
				if r.Header.Get("X-Retry") != "native" {
					t.Error("native retry header missing")
				}
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 202, createVolumeContractBody("creating"))
			}
			return
		}
		createVolumeContractWire(t, r, http.MethodPost, createVolumeContractActionPath, "refreshed-token")
		fields := createVolumeContractFields(t, r)
		if len(fields) != 1 || string(fields["os-set_bootable"]) != `{"bootable":true}` {
			t.Error(fields)
		}
		if actions.Add(1) == 1 {
			testcloud.JSON(w, 503, `{"error":"action retry"}`)
		} else {
			w.Header().Set("X-Proof", "bootable")
			w.WriteHeader(200)
			_, _ = w.Write([]byte{0, 0xff, 'x'})
		}
	})
	result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeName("worker"), blockstorage.WithCreateVolumeBootable(true))
	createVolumeContractCreated(t, result, createVolumeContractBody("creating"))
	if err != nil || result.Ready == nil || result.BootableSet == nil || calls.Load() != 7 || creates.Load() != 4 || actions.Load() != 2 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 2 {
		t.Fatalf("result=%+v error=%v calls=%d creates=%d actions=%d reauths=%d backoffs=%d retries=%d", result, err, calls.Load(), creates.Load(), actions.Load(), reauths.Load(), backoffs.Load(), retries.Load())
	}
}

func TestCreateVolumeTransportNativeRetryCannotRewritePhaseBodyOrPromoteStatus(t *testing.T) {
	for _, phase := range []string{"create", "bootable"} {
		for _, mode := range []string{"body", "raw body", "response target", "response ownership", "status expansion"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				cinder, _ := createVolumeContractClients(cloud)
				var calls, failedPhase atomic.Int32
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
					if !gophercloud.ResponseCodeIs(err, 503) {
						return err
					}
					switch mode {
					case "body":
						options.JSONBody = map[string]any{"changed": true}
					case "raw body":
						options.RawBody = strings.NewReader("changed")
					case "response target":
						options.JSONResponse = new(any)
					case "response ownership":
						options.KeepResponseBody = false
					case "status expansion":
						code := 201
						if phase == "bootable" {
							code = 202
						}
						options.OkCodes = append(options.OkCodes, code)
					}
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if phase == "bootable" && r.URL.Path == createVolumeContractCreatePath {
						w.Header().Set("X-Proof", "created")
						testcloud.JSON(w, 202, createVolumeContractBody("creating"))
						return
					}
					if r.Method == http.MethodGet {
						if phase != "bootable" {
							t.Error("failed create caused poll")
						}
						testcloud.JSON(w, 200, createVolumeContractBody("available"))
						return
					}
					path := createVolumeContractCreatePath
					if phase == "bootable" {
						path = createVolumeContractActionPath
					}
					createVolumeContractWire(t, r, http.MethodPost, path, "test-token")
					_ = createVolumeContractFields(t, r)
					if failedPhase.Add(1) == 1 {
						testcloud.JSON(w, 503, `{"error":"retry"}`)
						return
					}
					code := 201
					if phase == "bootable" {
						code = 202
					}
					testcloud.JSON(w, code, `{"accepted":true}`)
				})
				options := []blockstorage.CreateVolumeOption{blockstorage.WithCreateVolumeWait(false)}
				if phase == "bootable" {
					options = append(options, blockstorage.WithCreateVolumeBootable(true))
				}
				result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, options...)
				if err == nil || result == nil || result.BootableSet != nil || phase == "create" && (result.Created != nil || result.Ready != nil) || phase == "bootable" && (result.Created == nil || result.Ready == nil) {
					t.Fatalf("result=%+v error=%v", result, err)
				}
				base := int32(0)
				if phase == "bootable" {
					base = 2
				}
				if mode == "status expansion" {
					var native gophercloud.ErrUnexpectedResponseCode
					want := 201
					expected := []int{202}
					if phase == "bootable" {
						want = 202
						expected = []int{200}
					}
					if !errors.As(err, &native) || native.Actual != want || !reflect.DeepEqual(native.Expected, expected) || calls.Load() != base+2 || failedPhase.Load() != 2 {
						t.Fatalf("phase=%s native=%+v error=%v calls=%d", phase, native, err, calls.Load())
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != base+1 || failedPhase.Load() != 1 {
					t.Fatalf("owned phase request changed error=%v calls=%d", err, calls.Load())
				}
			})
		}
	}
}

func TestCreateVolumeTransportUnacceptedCauseAndCanceledContextDoNotInventProof(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, _ := createVolumeContractClients(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("creation transport failed")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = createVolumeContractTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if cancelled {
					cancel(cause)
				}
				return nil, cause
			})
			result, err := blockstorage.CreateVolume(ctx, cinder, nil, blockstorage.CreateVolumeRequest{Size: 2})
			var accepted *resource.ResponseError
			if result == nil || result.Created != nil || result.LastAccepted != nil || result.Ready != nil || result.BootableSet != nil || !errors.Is(err, cause) || cancelled && !errors.Is(err, context.Canceled) || errors.As(err, &accepted) || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			createVolumeContractOperation(t, err)
		})
	}
}
