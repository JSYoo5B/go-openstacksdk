package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type deleteVolumeContractTransport func(*http.Request) (*http.Response, error)

func (f deleteVolumeContractTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
func deleteVolumeContractResponse(r *http.Request, code int, body io.ReadCloser, proof string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"X-Proof": {proof}}, Body: body, Request: r}
}

type deleteVolumeContractReader struct {
	data                  *strings.Reader
	readError, closeError error
	onClose               func()
	once                  sync.Once
	closes                atomic.Int32
}

func (b *deleteVolumeContractReader) Read(p []byte) (int, error) {
	if b.readError != nil {
		n, _ := b.data.Read(p)
		cause := b.readError
		b.readError = nil
		return n, cause
	}
	return b.data.Read(p)
}
func (b *deleteVolumeContractReader) Close() error {
	b.closes.Add(1)
	b.once.Do(func() {
		if b.onClose != nil {
			b.onClose()
		}
	})
	return b.closeError
}

func TestDeleteVolumeTransportAccepted404ReadCloseContextAndSourceFailuresAreTerminal(t *testing.T) {
	for _, phase := range []string{"initial", "mutation", "poll"} {
		for _, mode := range []string{"read", "close", "context", "source"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := deleteVolumeContractClient(cloud, "3.23")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("accepted404 " + phase + " " + mode + " failed")
				body := []byte{0, 0xff, 'x'}
				var calls, retries atomic.Int32
				failStep := int32(1)
				if phase == "mutation" {
					failStep = 2
				}
				if phase == "poll" {
					failStep = 3
				}
				broken := &deleteVolumeContractReader{data: strings.NewReader(string(body))}
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
					broken.onClose = func() { client.Microversion = "3.24" }
				}
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries.Add(1)
					return err
				}
				located := deleteVolumeContractBody(`"available"`)
				cloud.Provider.HTTPClient.Transport = deleteVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
					step := calls.Add(1)
					if step == failStep {
						return deleteVolumeContractResponse(r, 404, broken, phase+"-failure"), nil
					}
					if step == 1 {
						return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(located)), "located"), nil
					}
					if step == 2 {
						return deleteVolumeContractResponse(r, 202, io.NopCloser(strings.NewReader("opaque mutation")), "deletion"), nil
					}
					t.Error("404 processing failure caused poll, mutation replay or cleanup", r.Method, r.URL)
					return nil, errors.New("unexpected HTTP")
				})
				options := []blockstorage.DeleteVolumeOption{}
				if phase == "mutation" {
					options = append(options, blockstorage.WithDeleteVolumeWait(false))
				}
				result, err := blockstorage.DeleteVolume(ctx, client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, options...)
				var accepted *resource.ResponseError
				if result == nil || err == nil || result.VolumeID != "volume-1" || result.Deleted || result.Ready != nil || !errors.As(err, &accepted) || accepted.StatusCode != 404 || !bytes.Equal(accepted.Body, body) || accepted.Header.Get("X-Proof") != phase+"-failure" || calls.Load() != failStep || retries.Load() != 0 || broken.closes.Load() != 1 {
					t.Fatalf("result=%+v error=%v calls=%d retries=%d closes=%d", result, err, calls.Load(), retries.Load(), broken.closes.Load())
				}
				if mode == "source" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, cause) {
					t.Fatal("404 phase cause lost", err)
				}
				if mode == "context" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				var proofBody []byte
				var proofHeader http.Header
				switch phase {
				case "initial":
					if result.Found || result.Located == nil || result.Located.Volume != nil || result.Deletion != nil || result.LastAccepted != nil || result.Absent == nil {
						t.Fatal(result)
					}
					proofBody, proofHeader = result.Located.Body, result.Located.Header
				case "mutation":
					deleteVolumeContractLocated(t, result, located)
					if result.Deletion == nil || result.Deletion.StatusCode != 404 || result.LastAccepted != nil || result.Absent != nil {
						t.Fatal(result)
					}
					proofBody, proofHeader = result.Deletion.Body, result.Deletion.Header
				case "poll":
					deleteVolumeContractLocated(t, result, located)
					if result.Deletion == nil || result.LastAccepted == nil || result.LastAccepted.Volume != nil || result.LastAccepted.StatusCode != 404 || result.Absent == nil {
						t.Fatal(result)
					}
					proofBody, proofHeader = result.LastAccepted.Body, result.LastAccepted.Header
				}
				if !bytes.Equal(proofBody, body) || proofHeader.Get("X-Proof") != phase+"-failure" {
					t.Fatal("actual admitted404 proof lost", result)
				}
				accepted.Body[0] = '!'
				accepted.Header.Set("X-Proof", "caller")
				if !bytes.Equal(proofBody, body) || proofHeader.Get("X-Proof") != phase+"-failure" {
					t.Fatal("phase proof aliases error evidence")
				}
				if result.Absent != nil {
					result.Absent.Body[0] = '!'
					result.Absent.Header.Set("X-Proof", "caller")
					if !bytes.Equal(proofBody, body) || proofHeader.Get("X-Proof") != phase+"-failure" {
						t.Fatal("separate Absent proof aliases admitted response")
					}
				}
				deleteVolumeContractOperation(t, err)
			})
		}
	}
}

func TestDeleteVolumeTransportNestedNative404NeverBecomesLogicalAbsence(t *testing.T) {
	for _, phase := range []string{"name", "initial", "mutation", "poll"} {
		t.Run(phase, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.23")
			var calls atomic.Int32
			nativeCause := gophercloud.ErrUnexpectedResponseCode{Method: http.MethodGet, URL: "https://nested.invalid/", Expected: []int{200}, Actual: 404, Body: []byte(`{"error":"nested404"}`)}
			wrapped := &url.Error{Op: "nested", URL: "https://nested.invalid/", Err: nativeCause}
			located := deleteVolumeContractBody(`"available"`)
			prior := deleteVolumeContractBody(`"error_deleting"`)
			failStep := int32(1)
			if phase == "mutation" {
				failStep = 2
			}
			if phase == "poll" {
				failStep = 4
			}
			cloud.Provider.HTTPClient.Transport = deleteVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
				step := calls.Add(1)
				if step == failStep {
					return nil, wrapped
				}
				if step == 1 {
					return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(located)), "located"), nil
				}
				if step == 2 {
					return deleteVolumeContractResponse(r, 202, io.NopCloser(strings.NewReader("mutation")), "deletion"), nil
				}
				if step == 3 {
					return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(prior)), "prior"), nil
				}
				t.Error("nested404 caused mutation replay or cleanup", r.Method, r.URL)
				return nil, errors.New("unexpected HTTP")
			})
			ref := resource.ID("volume-1")
			if phase == "name" {
				ref = resource.Name("worker")
			}
			interval := time.Millisecond
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: ref}, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &interval}))
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if result == nil || err == nil || !errors.As(err, &native) || native.Actual != 404 || native.URL != "https://nested.invalid/" || string(native.Body) != `{"error":"nested404"}` || errors.As(err, &accepted) || result.Deleted || result.Absent != nil || result.Ready != nil || calls.Load() != failStep {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			if phase == "name" || phase == "initial" {
				if result.Found || result.Located != nil || result.Deletion != nil {
					t.Fatal(result)
				}
			} else {
				deleteVolumeContractLocated(t, result, located)
				if phase == "poll" && (result.LastAccepted == nil || string(result.LastAccepted.Body) != prior || result.LastAccepted.Header.Get("X-Proof") != "prior") {
					t.Fatal("unaccepted transport404 replaced previous accepted evidence", result)
				}
			}
			deleteVolumeContractOperation(t, err)
		})
	}
	t.Run("accepted200 Close cause404 cannot invent absence", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := deleteVolumeContractClient(cloud, "3.23")
		cause := gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		var calls atomic.Int32
		body := deleteVolumeContractBody(`"deleted"`)
		cloud.Provider.HTTPClient.Transport = deleteVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			return deleteVolumeContractResponse(r, 200, &deleteVolumeContractReader{data: strings.NewReader(body), closeError: cause}, "actual200"), nil
		})
		result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")})
		var accepted *resource.ResponseError
		var nested gophercloud.ErrUnexpectedResponseCode
		if result == nil || result.Found || result.Deleted || result.Absent != nil || result.Located == nil || result.Located.StatusCode != 200 || result.Located.Volume != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || !errors.As(err, &nested) || nested.Actual != 404 || calls.Load() != 1 {
			t.Fatal(result, err, calls.Load())
		}
	})
}

func TestDeleteVolumeTransportCapturedHeadersLiveAuthAndSourceGuards(t *testing.T) {
	t.Run("ordinary headers are captured but provider token is live", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := deleteVolumeContractClient(cloud, "3.23")
		var calls, options atomic.Int32
		var retained *blockstorage.DeleteVolumeOpts
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", "3.23", "live-token")
				w.Header().Set("X-Proof", "located")
				testcloud.JSON(w, 200, deleteVolumeContractBody(`null`))
			case 2:
				deleteVolumeContractWire(t, r, http.MethodDelete, deleteVolumeContractPath, "cascade=false&force=false", "3.23", "live-token")
				w.WriteHeader(204)
			default:
				t.Error("retained config changed wait or force", r.Method, r.URL)
				w.WriteHeader(500)
			}
		})
		result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, func(v *blockstorage.DeleteVolumeOpts) error {
			options.Add(1)
			value := false
			v.Wait = &value
			retained = v
			client.MoreHeaders["x-source"] = "changed"
			cloud.Provider.SetToken("live-token")
			return nil
		}, func(*blockstorage.DeleteVolumeOpts) error {
			options.Add(1)
			*retained.Wait = true
			retained.Force = true
			return nil
		})
		if err != nil || result == nil || !result.Deleted || calls.Load() != 2 || options.Load() != 2 || client.MoreHeaders["x-source"] != "changed" {
			t.Fatal(result, err, calls.Load(), options.Load())
		}
	})
	t.Run("clean Name absence cannot suppress source mutation", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := deleteVolumeContractClient(cloud, "3.23")
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = deleteVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractListPath, "name=worker", "3.23", "test-token")
			client.Microversion = "3.24"
			return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(`{"volumes":[]}`)), "name"), nil
		})
		result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.Name("worker")})
		if result == nil || result.VolumeID != "" || result.Found || result.Deleted || result.Located != nil || result.Deletion != nil || result.Absent != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
			t.Fatal(result, err, calls.Load())
		}
	})
	for _, change := range []struct {
		name   string
		mutate func(*gophercloud.ServiceClient)
	}{{"provider", func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }}, {"endpoint", func(c *gophercloud.ServiceClient) { c.Endpoint += "changed/" }}, {"base", func(c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" }}, {"type", func(c *gophercloud.ServiceClient) { c.Type = "image" }}, {"microversion", func(c *gophercloud.ServiceClient) { c.Microversion = "3.24" }}} {
		for _, during := range []string{"option", "accepted initial", "progress"} {
			t.Run(change.name+"/"+during, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := deleteVolumeContractClient(cloud, "3.23")
				var calls atomic.Int32
				located := deleteVolumeContractBody(`"available"`)
				cloud.Provider.HTTPClient.Transport = deleteVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
					step := calls.Add(1)
					if step == 1 {
						if during == "accepted initial" {
							change.mutate(client)
						}
						return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(located)), "located"), nil
					}
					if step == 2 {
						return deleteVolumeContractResponse(r, 202, io.NopCloser(strings.NewReader("mutation")), "deletion"), nil
					}
					if step == 3 {
						return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(deleteVolumeContractBody(`"deleting"`))), "progress"), nil
					}
					t.Error("changed source allowed another exchange", r.Method, r.URL)
					return nil, errors.New("unexpected HTTP")
				})
				options := []blockstorage.DeleteVolumeOption{}
				if during == "option" {
					options = append(options, func(*blockstorage.DeleteVolumeOpts) error { change.mutate(client); return nil })
				}
				if during == "progress" {
					options = append(options, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{ProgressCallback: func(int) error { change.mutate(client); return nil }}))
				}
				result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, options...)
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(result, err)
				}
				switch during {
				case "option":
					if result != nil || calls.Load() != 0 {
						t.Fatal(result, calls.Load())
					}
				case "accepted initial":
					var accepted *resource.ResponseError
					if result == nil || result.Found || result.Located == nil || result.Located.Volume != nil || result.Deletion != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || calls.Load() != 1 {
						t.Fatal(result, err, calls.Load())
					}
				case "progress":
					deleteVolumeContractLocated(t, result, located)
					var accepted *resource.ResponseError
					if result.Deleted || result.Ready != nil || result.LastAccepted == nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || calls.Load() != 3 {
						t.Fatal(result, err, calls.Load())
					}
				}
			})
		}
	}
}

func TestDeleteVolumeTransportWaitBudgetStartsAfterLookupAndMutationAndZeroIsUnlimited(t *testing.T) {
	cloud := testcloud.New(t)
	client := deleteVolumeContractClient(cloud, "3.23")
	var calls atomic.Int32
	located := deleteVolumeContractBody(`null`)
	cloud.Provider.HTTPClient.Transport = deleteVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
		switch calls.Add(1) {
		case 1, 2:
			if _, bounded := r.Context().Deadline(); bounded {
				t.Error("status-wait budget leaked into lookup/mutation")
			}
			code, body, proof := 200, located, "located"
			if r.Method == http.MethodDelete {
				code, body, proof = 202, "opaque", "deletion"
			}
			return deleteVolumeContractResponse(r, code, &deleteVolumeContractReader{data: strings.NewReader(body), onClose: func() { time.Sleep(100 * time.Millisecond) }}, proof), nil
		case 3:
			if _, bounded := r.Context().Deadline(); !bounded {
				t.Error("positive wait budget missing")
			}
			return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(deleteVolumeContractBody(`"deleted"`))), "ready"), nil
		default:
			t.Error("unexpected wait boundary request", r.URL)
			return nil, errors.New("unexpected HTTP")
		}
	})
	timeout := 50 * time.Millisecond
	result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{Timeout: &timeout}))
	deleteVolumeContractLocated(t, result, located)
	if err != nil || !result.Deleted || result.Ready == nil || calls.Load() != 3 {
		t.Fatal(result, err, calls.Load())
	}
	t.Run("explicit zero waits past a nonterminal result", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := deleteVolumeContractClient(cloud, "3.23")
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				w.Header().Set("X-Proof", "located")
				testcloud.JSON(w, 200, located)
			case 2:
				w.WriteHeader(202)
			case 3:
				testcloud.JSON(w, 200, deleteVolumeContractBody(`"error_deleting"`))
			case 4:
				w.WriteHeader(404)
				_, _ = w.Write([]byte{0, 0xff})
			default:
				t.Error("zero budget issued extra mutation/poll", r.Method, r.URL)
				w.WriteHeader(500)
			}
		})
		zero, interval := time.Duration(0), time.Millisecond
		result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{Timeout: &zero, PollInterval: &interval, ProgressCallback: func(progress int) error {
			callbacks.Add(1)
			if progress != 0 {
				t.Error(progress)
			}
			return nil
		}}))
		deleteVolumeContractLocated(t, result, located)
		if err != nil || !result.Deleted || result.Absent == nil || result.Ready != nil || calls.Load() != 4 || callbacks.Load() != 1 {
			t.Fatal(result, err, calls.Load(), callbacks.Load())
		}
	})
}

func TestDeleteVolumeTransportWaitCancellationAndLaterRejectionsRetainPreviousProof(t *testing.T) {
	for _, mode := range []string{"timer", "callback cancel", "blocked", "native rejection", "transport cause"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.23")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("waiting stopped")
			callbackCause := errors.New("callback stopped")
			var calls atomic.Int32
			located := deleteVolumeContractBody(`"in-use"`)
			prior := deleteVolumeContractBody(`null`)
			cloud.Provider.HTTPClient.Transport = deleteVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
				switch calls.Add(1) {
				case 1:
					return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(located)), "located"), nil
				case 2:
					return deleteVolumeContractResponse(r, 202, io.NopCloser(strings.NewReader("mutation")), "deletion"), nil
				case 3:
					return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(prior)), "prior"), nil
				case 4:
					if mode == "blocked" {
						<-r.Context().Done()
						return nil, r.Context().Err()
					}
					if mode == "native rejection" {
						return deleteVolumeContractResponse(r, 403, io.NopCloser(strings.NewReader(`{"error":"later rejected"}`)), "rejected"), nil
					}
					if mode == "transport cause" {
						return nil, cause
					}
					t.Error("wait failure caused later exchange", r.URL)
					return nil, errors.New("unexpected HTTP")
				default:
					t.Error("wait failure caused cleanup/replay", r.URL)
					return nil, errors.New("unexpected HTTP")
				}
			})
			interval, timeout := time.Hour, 200*time.Millisecond
			policy := blockstorage.DeleteVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval}
			if mode == "callback cancel" {
				policy.Timeout = nil
				policy.ProgressCallback = func(int) error { cancel(cause); return callbackCause }
			}
			if mode == "blocked" || mode == "native rejection" || mode == "transport cause" {
				interval = time.Millisecond
			}
			if mode == "native rejection" || mode == "transport cause" {
				policy.Timeout = nil
			}
			result, err := blockstorage.DeleteVolume(ctx, client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeWaitPolicy(policy))
			deleteVolumeContractLocated(t, result, located)
			var accepted *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || result.Deleted || result.Deletion == nil || result.LastAccepted == nil || string(result.LastAccepted.Body) != prior || result.LastAccepted.Header.Get("X-Proof") != "prior" || result.Ready != nil || result.Absent != nil {
				t.Fatal(result, err)
			}
			switch mode {
			case "timer":
				if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &accepted) || string(accepted.Body) != prior || calls.Load() != 3 {
					t.Fatal(err, calls.Load())
				}
			case "callback cancel":
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callbackCause) || !errors.As(err, &accepted) || calls.Load() != 3 {
					t.Fatal(err, calls.Load())
				}
			case "blocked":
				if !errors.Is(err, context.DeadlineExceeded) || errors.As(err, &accepted) || calls.Load() != 4 {
					t.Fatal(err, calls.Load())
				}
			case "native rejection":
				if !errors.As(err, &native) || native.Actual != 403 || errors.As(err, &accepted) || calls.Load() != 4 {
					t.Fatal(err, calls.Load())
				}
			case "transport cause":
				if !errors.Is(err, cause) || errors.As(err, &accepted) || errors.Is(err, context.Canceled) || calls.Load() != 4 {
					t.Fatal(err, calls.Load())
				}
			}
			deleteVolumeContractOperation(t, err)
		})
	}
}

func TestDeleteVolumeTransportNativeAuthBackoffRetryPreservesNormalAndForceRequests(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprint(force), func(t *testing.T) {
			cloud := testcloud.New(t)
			version := "3.23"
			if force {
				version = "3.22"
			}
			client := deleteVolumeContractClient(cloud, version)
			var calls, mutations, reauths, backoffs, retries atomic.Int32
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
				wantMethod, wantTarget, wantCodes := http.MethodDelete, cloud.Server.URL+deleteVolumeContractPath+"?cascade=false&force=false", []int{202, 204, 404}
				if force {
					wantMethod, wantTarget, wantCodes = http.MethodPost, cloud.Server.URL+deleteVolumeContractActionPath, []int{201, 202, 404}
				}
				if method != wantMethod || target != wantTarget || count != 2 || !reflect.DeepEqual(options.OkCodes, wantCodes) || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody || !force && options.JSONBody != nil {
					t.Error(method, target, count, options)
				}
				if force {
					raw, err := json.Marshal(options.JSONBody)
					if err != nil || string(raw) != `{"os-force_delete":null}` {
						t.Error(string(raw), err)
					}
				}
				if options.MoreHeaders == nil {
					options.MoreHeaders = map[string]string{}
				}
				options.MoreHeaders["X-Retry"] = "native"
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				step := calls.Add(1)
				if step == 1 {
					deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", version, "test-token")
					w.Header().Set("X-Proof", "located")
					testcloud.JSON(w, 200, deleteVolumeContractBody(`"available"`))
					return
				}
				if r.Method == http.MethodGet {
					deleteVolumeContractWire(t, r, http.MethodGet, deleteVolumeContractPath, "", version, "refreshed-token")
					w.Header().Set("X-Proof", "absent")
					w.WriteHeader(404)
					_, _ = w.Write([]byte{0, 0xff})
					return
				}
				count := mutations.Add(1)
				token := "refreshed-token"
				if count == 1 {
					token = "test-token"
				}
				method, path, query := http.MethodDelete, deleteVolumeContractPath, "cascade=false&force=false"
				if force {
					method, path, query = http.MethodPost, deleteVolumeContractActionPath, ""
				}
				deleteVolumeContractWire(t, r, method, path, query, version, token)
				if force {
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 || string(body["os-force_delete"]) != "null" {
						t.Error(body, err)
					}
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
					w.Header().Set("X-Proof", "deletion")
					w.WriteHeader(202)
					_, _ = w.Write([]byte{0, 0xff})
				}
			})
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeForce(force))
			if err != nil || result == nil || !result.Found || !result.Deleted || result.Deletion == nil || result.Absent == nil || calls.Load() != 6 || mutations.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d mutations=%d auth=%d backoff=%d retry=%d", result, err, calls.Load(), mutations.Load(), reauths.Load(), backoffs.Load(), retries.Load())
			}
		})
	}
}

func TestDeleteVolumeTransportNativeRetryCannotChangeOwnedBodyResponseOrSuccessCodes(t *testing.T) {
	for _, force := range []bool{false, true} {
		for _, mode := range []string{"JSON body", "raw body", "response target", "response ownership", "status expansion"} {
			t.Run(fmt.Sprintf("%v/%s", force, mode), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := deleteVolumeContractClient(cloud, "3.22")
				var calls, mutations atomic.Int32
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
					if !gophercloud.ResponseCodeIs(err, 503) {
						return err
					}
					switch mode {
					case "JSON body":
						options.JSONBody = map[string]any{"os-force_delete": "changed"}
					case "raw body":
						options.RawBody = strings.NewReader("changed")
					case "response target":
						options.JSONResponse = new(any)
					case "response ownership":
						options.KeepResponseBody = false
					case "status expansion":
						options.OkCodes = append(options.OkCodes, 200)
					}
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method == http.MethodGet {
						w.Header().Set("X-Proof", "located")
						testcloud.JSON(w, 200, deleteVolumeContractBody(`null`))
						return
					}
					if mutations.Add(1) == 1 {
						testcloud.JSON(w, 503, `{"error":"retry"}`)
					} else {
						w.Header().Set("X-Proof", "expanded-current")
						testcloud.JSON(w, 200, `{"accepted":true}`)
					}
				})
				result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeForce(force), blockstorage.WithDeleteVolumeWait(false))
				if result == nil || !result.Found || result.Deleted || result.Deletion != nil || result.LastAccepted != nil || result.Absent != nil || result.Ready != nil || err == nil {
					t.Fatal(result, err)
				}
				if mode == "status expansion" {
					var native gophercloud.ErrUnexpectedResponseCode
					codes := []int{202, 204, 404}
					if force {
						codes = []int{201, 202, 404}
					}
					if !errors.As(err, &native) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, codes) || native.ResponseHeader.Get("X-Proof") != "expanded-current" || calls.Load() != 3 || mutations.Load() != 2 {
						t.Fatal(result, err, calls.Load())
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 || mutations.Load() != 1 {
					t.Fatal(result, err, calls.Load())
				}
			})
		}
	}
}

func TestDeleteVolumeTransportRedirectCannotChangeOwnedFiniteQueryOrActionRoute(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprint(force), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := deleteVolumeContractClient(cloud, "3.22")
			var calls, redirects atomic.Int32
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				next.URL.RawQuery = "cascade=true&force=true"
				next.Method = http.MethodDelete
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method == http.MethodGet {
					w.Header().Set("X-Proof", "located")
					testcloud.JSON(w, 200, deleteVolumeContractBody(`null`))
					return
				}
				w.Header().Set("Location", deleteVolumeContractPath+"?cascade=false")
				w.WriteHeader(307)
			})
			result, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, blockstorage.WithDeleteVolumeWait(false), blockstorage.WithDeleteVolumeForce(force))
			if result == nil || !result.Found || result.Deleted || result.Deletion != nil || result.LastAccepted != nil || result.Absent != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 || redirects.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d redirects=%d", result, err, calls.Load(), redirects.Load())
			}
		})
	}
}
