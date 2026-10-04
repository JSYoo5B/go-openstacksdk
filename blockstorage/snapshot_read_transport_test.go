package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type snapshotReadTransport func(*http.Request) (*http.Response, error)

func (f snapshotReadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type snapshotReadTransportBody struct {
	data                  io.Reader
	readError, closeError error
	onClose               func()
	closes                atomic.Int32
}

func (b *snapshotReadTransportBody) Read(p []byte) (int, error) {
	n, err := b.data.Read(p)
	if err == io.EOF && b.readError != nil {
		return n, b.readError
	}
	return n, err
}
func (b *snapshotReadTransportBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeError
}
func snapshotReadTransportResponse(r *http.Request, code int, body io.ReadCloser, proof string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {proof}}, Body: body, Request: r}
}
func snapshotReadTransportMutate(client *gophercloud.ServiceClient, fact string) {
	switch fact {
	case "provider":
		client.ProviderClient = &gophercloud.ProviderClient{}
	case "endpoint":
		client.Endpoint += "changed/"
	case "resource base":
		client.ResourceBase += "changed/"
	case "type":
		client.Type = "compute"
	case "microversion":
		client.Microversion = "3.61"
	}
}

func TestVolumeSnapshotReadOriginalOptionsOwnEachSnapshotAndSourceOrdinaryHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls, callbacks atomic.Int32
	filters := json.RawMessage(`{"status":"available","description":"wanted"}`)
	headers := map[string]string{"x-extra": "owned"}
	version := "latest"
	first := func(o *blockstorage.VolumeSnapshotListOpts) error {
		callbacks.Add(1)
		o.Filters = &filters
		o.Headers = headers
		o.Microversion = &version
		return nil
	}
	second := func(o *blockstorage.VolumeSnapshotListOpts) error {
		callbacks.Add(1)
		filters = json.RawMessage(`{"status":"changed","description":"changed"}`)
		headers["x-extra"] = "changed"
		version = "3.61"
		client.MoreHeaders["x-source"] = "later ordinary change"
		return nil
	}
	third := func(o *blockstorage.VolumeSnapshotListOpts) error {
		callbacks.Add(1)
		if o.Filters == nil || string(*o.Filters) != `{"status":"available","description":"wanted"}` || o.Headers["x-extra"] != "owned" || o.Microversion == nil || *o.Microversion != "latest" {
			t.Error("previous callback retained mutable inputs", o)
		}
		return nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != snapshotReadContractDetail || r.URL.Query().Get("status") != "available" || len(r.URL.Query()) != 1 || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Extra") != "owned" || r.Header.Get("OpenStack-API-Version") != "volume latest" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"owned","description":"wanted"}]`, ""))
	})
	result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, first, second, third)
	if err != nil || result == nil || len(result.Snapshots) != 1 || calls.Load() != 1 || callbacks.Load() != 3 || client.Microversion != "3.60" || client.MoreHeaders["x-source"] != "later ordinary change" {
		t.Fatal(result, err, calls.Load(), callbacks.Load())
	}
}

func TestVolumeSnapshotReadOriginalCallbackFailureJoinsSourceAndContextBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls, callbacks atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	callbackCause, cancelCause := errors.New("original option error"), errors.New("original option cancellation")
	option := func(o *blockstorage.VolumeSnapshotReadOpts) error {
		callbacks.Add(1)
		client.ResourceBase += "changed/"
		cancel(cancelCause)
		return callbackCause
	}
	result, err := blockstorage.GetVolumeSnapshotByID(ctx, client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "literal"}, option)
	var physical *resource.ResponseError
	if result != nil || calls.Load() != 0 || callbacks.Load() != 1 || !errors.Is(err, callbackCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) {
		t.Fatal(result, err, calls.Load(), callbacks.Load())
	}
}

func TestVolumeSnapshotReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage(t *testing.T) {
	for _, stage := range []string{"member", "list", "member + source", "list + source"} {
		t.Run(stage, func(t *testing.T) {
			changed := strings.HasSuffix(stage, " + source")
			stage = strings.TrimSuffix(stage, " + source")
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("current read"), errors.New("current close"), errors.New("current cancellation")
			body := `{"snapshot":{"id":"wire"}}`
			if stage == "list" {
				body = snapshotReadContractPage(`[{"id":"wire"}]`, "")
			}
			reader := &snapshotReadTransportBody{data: strings.NewReader(body), readError: readCause, closeError: closeCause, onClose: func() {
				if changed {
					client.ResourceBase += "changed/"
				}
				cancel(cancelCause)
			}}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return snapshotReadTransportResponse(r, 203, reader, "current"), nil
			})
			var observed *blockstorage.VolumeSnapshotsPage
			var pages []*blockstorage.VolumeSnapshotsPage
			var value json.RawMessage
			var err error
			if stage == "member" {
				result, e := blockstorage.GetVolumeSnapshotByID(ctx, client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "literal"})
				err = e
				if result == nil || result.Snapshot != nil {
					t.Fatal(result, e)
				}
				observed, pages, value = result.Observed, result.Pages, result.Value
			} else {
				result, e := blockstorage.ListVolumeSnapshots(ctx, client)
				err = e
				if result == nil || result.Snapshots != nil {
					t.Fatal(result, e)
				}
				pages, value = result.Pages, result.Value
			}
			var physical *resource.ResponseError
			if value != nil || calls.Load() != 1 || reader.closes.Load() != 1 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "current" {
				t.Fatal(err, calls.Load(), reader.closes.Load())
			}
			if changed && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("source drift lost under context failure", err)
			}
			proof := observed
			if stage == "list" {
				if observed != nil || len(pages) != 1 {
					t.Fatal(observed, pages)
				}
				proof = pages[0]
			} else if proof == nil || len(pages) != 0 {
				t.Fatal(proof, pages)
			}
			physical.Body[0] = '!'
			physical.Header.Set("X-Proof", "error mutated")
			if string(proof.Body) != body || proof.Header.Get("X-Proof") != "current" {
				t.Fatal("error aliases admitted proof")
			}
		})
	}
}

func TestVolumeSnapshotReadAcceptedSourceFactsAreFrozenThroughBodyClose(t *testing.T) {
	for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
		t.Run(fact, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := `{"snapshot":{"id":"wire"}}`
			reader := &snapshotReadTransportBody{data: strings.NewReader(body), onClose: func() { snapshotReadTransportMutate(client, fact) }}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return snapshotReadTransportResponse(r, 203, reader, "drift"), nil
			})
			result, err := blockstorage.GetVolumeSnapshot(context.Background(), client, blockstorage.GetVolumeSnapshotRequest{NameOrID: "literal"})
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Snapshot != nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || reader.closes.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &physical) || string(physical.Body) != body || result.Observed.Header.Get("X-Proof") != "drift" {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadNativeReauthRetryKeepLiveTokenAndOwnedSourceOnce(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls, reauths, retries atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cloud.Provider.ReauthFunc = func(context.Context) error {
		reauths.Add(1)
		cloud.Provider.SetToken("reauth-token")
		client.MoreHeaders["x-source"] = "changed ordinary"
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, err error, _ uint) error {
		if !gophercloud.ResponseCodeIs(err, 503) {
			return err
		}
		if retries.Add(1) > 1 {
			return errors.New("bounded retry fixture exhausted")
		}
		if method != http.MethodGet || !strings.HasSuffix(target, "snapshots/literal") || o.JSONBody != nil || o.RawBody != nil || o.JSONResponse != nil || !o.KeepResponseBody || len(o.OkCodes) != 300 || o.OkCodes[0] != 100 || o.OkCodes[299] != 399 {
			t.Error("owned request changed", method, target, o)
		}
		o.MoreHeaders["X-Native"] = "retry"
		cloud.Provider.SetToken("retry-token")
		return nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		token := "test-token"
		if n == 2 {
			token = "reauth-token"
		} else if n == 3 {
			token = "retry-token"
		}
		snapshotReadContractWire(t, r, snapshotReadContractBasic+"/literal", token)
		switch n {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 503, `{"error":"retry"}`)
		case 3:
			if r.Header.Get("X-Native") != "retry" {
				t.Error(r.Header)
			}
			w.Header().Set("X-Proof", "accepted once")
			testcloud.JSON(w, 203, `{"snapshot":{"id":"wire"}}`)
		default:
			t.Error("workflow replay", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumeSnapshotByID(ctx, client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "literal"})
	if err != nil || result == nil || result.Snapshot == nil || result.Observed == nil || calls.Load() != 3 || reauths.Load() != 1 || retries.Load() != 1 || result.Observed.StatusCode != 203 || result.Observed.Header.Get("X-Proof") != "accepted once" {
		t.Fatal(result, err, calls.Load(), reauths.Load(), retries.Load())
	}
}

func TestVolumeSnapshotReadNativeExpandedFallbackStatusIsTerminalOriginalPolicyRejection(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls, retries atomic.Int32
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if retries.Add(1) > 1 {
					return errors.New("bounded fixture exhausted")
				}
				o.OkCodes = append(o.OkCodes, code)
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				snapshotReadContractWire(t, r, snapshotReadContractBasic+"/literal", "test-token")
				n := calls.Add(1)
				if n == 1 {
					testcloud.JSON(w, 503, `{"error":"retry"}`)
					return
				}
				if n != 2 {
					t.Error("expanded status borrowed as clean fallback", r.URL)
				}
				w.Header().Set("X-Proof", "expanded current")
				testcloud.JSON(w, code, `{"error":"expanded policy"}`)
			})
			result, err := blockstorage.GetVolumeSnapshot(ctx, client, blockstorage.GetVolumeSnapshotRequest{NameOrID: "literal"})
			var physical *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.Value != nil || result.Snapshot != nil || result.Observed != nil || len(result.Pages) != 0 || calls.Load() != 2 || retries.Load() != 1 || errors.As(err, &physical) || !errors.As(err, &native) || native.Actual != code || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || native.ResponseHeader.Get("X-Proof") != "expanded current" {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadNativeRetryCannotReplaceBodyDecoderRetentionOrSource(t *testing.T) {
	for _, kind := range []string{"body", "raw body", "decoder", "retention", "source"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls, retries atomic.Int32
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if retries.Add(1) > 1 {
					return errors.New("bounded fixture exhausted")
				}
				switch kind {
				case "body":
					o.JSONBody = map[string]any{"changed": true}
				case "raw body":
					o.RawBody = strings.NewReader("changed")
				case "decoder":
					o.JSONResponse = new(any)
				case "retention":
					o.KeepResponseBody = false
				case "source":
					client.Microversion = "3.61"
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "native rejected")
				testcloud.JSON(w, 503, `{"error":"retry once"}`)
			})
			result, err := blockstorage.GetVolumeSnapshotByID(ctx, client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "literal"})
			var native gophercloud.ErrUnexpectedResponseCode
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Snapshot != nil || result.Observed != nil || len(result.Pages) != 0 || calls.Load() != 1 || retries.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Proof") != "native rejected" || errors.As(err, &physical) {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadBlockedContinuationCancellationKeepsOnlyCompletedPage(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("caller canceled continuation")
	started := make(chan struct{})
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("X-Proof", "first")
			testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"first"}]`, "?marker=blocked"))
			return
		}
		if n != 2 {
			t.Error("replayed continuation", r.URL)
		}
		close(started)
		<-r.Context().Done()
	})
	type outcome struct {
		result *blockstorage.ListVolumeSnapshotsResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() { result, err := blockstorage.ListVolumeSnapshots(ctx, client); done <- outcome{result, err} }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("continuation never started")
	}
	cancel(cause)
	select {
	case got := <-done:
		var physical *resource.ResponseError
		if got.result == nil || got.result.Value != nil || got.result.Snapshots != nil || len(got.result.Pages) != 1 || got.result.Pages[0].Header.Get("X-Proof") != "first" || calls.Load() != 2 || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || errors.As(got.err, &physical) {
			t.Fatal(got, calls.Load())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("continuation ignored cancellation")
	}
}

func TestVolumeSnapshotReadRedirectCannotSendTokenToChangedFixedTarget(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n != 1 {
			t.Error("redirect target received credentials", r.URL, r.Header)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Location", cloud.Server.URL+"/changed/target")
		w.WriteHeader(302)
	})
	result, err := blockstorage.GetVolumeSnapshotByID(context.Background(), client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "literal"})
	var physical *resource.ResponseError
	if result == nil || result.Value != nil || result.Snapshot != nil || result.Observed != nil || len(result.Pages) != 0 || calls.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) {
		t.Fatal(result, err, calls.Load())
	}
}
