package image_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/image"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"

	"github.com/gophercloud/gophercloud/v2"
)

func TestUploadUnauthorizedKeepsHTTPErrorWithoutReauthentication(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	var attempts, reauth atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("new-token"); return nil }
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"created","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "image-data" {
			t.Errorf("body=%q err=%v", body, err)
		}
		testcloud.JSON(w, http.StatusUnauthorized, `{"message":"token expired"}`)
	})
	reader := &ownedUploadReader{Reader: strings.NewReader("prefix-image-data")}
	_, _ = reader.Seek(int64(len("prefix-")), io.SeekStart)
	created, err := service.Upload(context.Background(), uploadInput(reader))
	var responseError gophercloud.ErrUnexpectedResponseCode
	if created == nil || !errors.As(err, &responseError) || responseError.Actual != 401 || attempts.Load() != 1 || reauth.Load() != 0 || reader.closed.Load() != 0 {
		t.Fatalf("created=%v err=%v attempts=%d reauth=%d close calls=%d", created, err, attempts.Load(), reauth.Load(), reader.closed.Load())
	}
}

type nonseekableUploadReader struct {
	reader io.Reader
	closed atomic.Int32
}

func (r *nonseekableUploadReader) Read(p []byte) (int, error) { return r.reader.Read(p) }
func (r *nonseekableUploadReader) Close() error               { r.closed.Add(1); return nil }

type uploadRoundTripper func(*http.Request) (*http.Response, error)

func (f uploadRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func uploadResponse(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

func TestUploadNeverReplaysPrefixOrRacesWithAsynchronousClose(t *testing.T) {
	for _, status := range []int{401, 429, 500} {
		for _, seekable := range []bool{false, true} {
			t.Run(http.StatusText(status)+" seekable="+map[bool]string{false: "false", true: "true"}[seekable], func(t *testing.T) {
				provider := &gophercloud.ProviderClient{}
				provider.UseTokenLock()
				provider.SetToken("test-token")
				provider.UserAgent.Prepend("upload-contract")
				var callbacks, attempts atomic.Int32
				provider.ReauthFunc = func(context.Context) error { callbacks.Add(1); return nil }
				provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					callbacks.Add(1)
					return nil
				}
				provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					callbacks.Add(1)
					return nil
				}
				closeGate, closeDone := make(chan struct{}), make(chan struct{})
				provider.HTTPClient.Transport = uploadRoundTripper(func(r *http.Request) (*http.Response, error) {
					if r.Method == http.MethodPost {
						_ = r.Body.Close()
						return uploadResponse(r, 201, `{"id":"created","status":"queued"}`), nil
					}
					attempt := attempts.Add(1)
					if r.Method != http.MethodPut || r.URL.Path != "/v2/images/created/file" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Test-Header") != "preserved" || !strings.Contains(r.Header.Get("User-Agent"), "upload-contract") {
						t.Errorf("request=%s %s headers=%v", r.Method, r.URL, r.Header)
					}
					if attempt == 1 {
						if seekable && status == 401 {
							_, _ = io.Copy(io.Discard, r.Body)
						} else {
							prefix := make([]byte, 1)
							if n, err := r.Body.Read(prefix); n != 1 || err != nil || string(prefix) != "i" {
								t.Errorf("prefix=%q n=%d err=%v", prefix, n, err)
							}
						}
						// RoundTripper may close a request body asynchronously, including
						// after a caller has already started another request.
						go func() { <-closeGate; _ = r.Body.Close(); close(closeDone) }()
						return uploadResponse(r, status, `{"message":"first upload failed"}`), nil
					}
					body, _ := io.ReadAll(r.Body)
					t.Errorf("binary upload was retried with %q", body)
					_ = r.Body.Close()
					return uploadResponse(r, 204, ""), nil
				})
				client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "image", Endpoint: "http://image.invalid/v2/", MoreHeaders: map[string]string{"X-Test-Header": "preserved"}}
				service := image.New(client)
				var data io.Reader
				var closeCount *atomic.Int32
				if seekable {
					reader := &ownedUploadReader{Reader: strings.NewReader("image-data")}
					data, closeCount = reader, &reader.closed
				} else {
					reader := &nonseekableUploadReader{reader: strings.NewReader("image-data")}
					data, closeCount = reader, &reader.closed
				}
				created, err := service.Upload(context.Background(), uploadInput(data))
				close(closeGate)
				select {
				case <-closeDone:
				case <-time.After(time.Second):
					t.Fatal("delayed request close did not finish")
				}
				var responseError gophercloud.ErrUnexpectedResponseCode
				if created == nil || created.ID != "created" || !errors.As(err, &responseError) || responseError.Actual != status || attempts.Load() != 1 || callbacks.Load() != 0 || closeCount.Load() != 0 || provider.Token() != "test-token" {
					t.Fatalf("created=%v err=%v attempts=%d callbacks=%d close calls=%d token=%q", created, err, attempts.Load(), callbacks.Load(), closeCount.Load(), provider.Token())
				}
			})
		}
	}
}

var uploadReaderFailure = errors.New("source image read failed")

type failedUploadReader struct{}

func (failedUploadReader) Read([]byte) (int, error) { return 0, uploadReaderFailure }

func TestUploadSourceReadErrorRemainsInspectable(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"created","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	})
	created, err := service.Upload(context.Background(), uploadInput(failedUploadReader{}))
	if created == nil || created.ID != "created" || !errors.Is(err, uploadReaderFailure) {
		t.Fatalf("created=%v err=%v", created, err)
	}
}

type noSeekUploadReader struct {
	*strings.Reader
	seeks atomic.Int32
}

func (r *noSeekUploadReader) Seek(int64, int) (int64, error) {
	r.seeks.Add(1)
	return 0, uploadReaderFailure
}

func TestUploadDoesNotSeekCallerReader(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"created","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	})
	reader := &noSeekUploadReader{Reader: strings.NewReader("data")}
	created, err := service.Upload(context.Background(), uploadInput(reader))
	if created == nil || err != nil || reader.seeks.Load() != 0 {
		t.Fatalf("created=%v err=%v seeks=%d", created, err, reader.seeks.Load())
	}
}

type blockingOwnedUploadReader struct {
	started chan struct{}
	release chan struct{}
	start   sync.Once
	close   sync.Once
	closed  atomic.Int32
}

func (r *blockingOwnedUploadReader) Read([]byte) (int, error) {
	r.start.Do(func() { close(r.started) })
	<-r.release
	return 0, io.EOF
}

func (r *blockingOwnedUploadReader) Close() error {
	r.close.Do(func() { r.closed.Add(1); close(r.release) })
	return nil
}

func TestUploadCallerCanReleaseBlockedReaderAfterCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"created","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	})
	reader := &blockingOwnedUploadReader{started: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() { _ = reader.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		created *image.Image
		err     error
	}
	finished := make(chan result, 1)
	go func() {
		created, err := service.Upload(ctx, uploadInput(reader))
		finished <- result{created: created, err: err}
	}()
	select {
	case <-reader.started:
	case <-time.After(time.Second):
		t.Fatal("upload did not start reading")
	}
	cancel()
	if reader.closed.Load() != 0 {
		t.Fatal("SDK closed caller input on cancellation")
	}
	// An arbitrary io.Reader may block in Read. Its owner must release that Read.
	_ = reader.Close()
	select {
	case got := <-finished:
		if got.created == nil || got.created.ID != "created" || !errors.Is(got.err, context.Canceled) || reader.closed.Load() != 1 {
			t.Fatalf("created=%v err=%v close calls=%d", got.created, got.err, reader.closed.Load())
		}
	case <-time.After(time.Second):
		t.Fatal("upload did not finish after cancellation and caller release")
	}
}

func TestUploadDoesNotFollowRedirectOrMutateProviderSettings(t *testing.T) {
	cloud := testcloud.New(t)
	var redirects atomic.Int32
	cloud.Provider.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"created","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Location", "/redirected")
		w.WriteHeader(http.StatusSeeOther)
	})
	cloud.Mux.HandleFunc("/redirected", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upload redirect must not discard PUT data: %s", r.Method)
		w.WriteHeader(http.StatusNoContent)
	})
	created, err := service.Upload(context.Background(), uploadInput(strings.NewReader("image-data")))
	var responseError gophercloud.ErrUnexpectedResponseCode
	if created == nil || !errors.As(err, &responseError) || responseError.Actual != 303 || redirects.Load() != 0 {
		t.Fatalf("created=%v err=%v redirects=%d", created, err, redirects.Load())
	}
	if err := cloud.Provider.HTTPClient.CheckRedirect(nil, nil); err != nil || redirects.Load() != 1 {
		t.Fatal("upload changed the original provider's redirect policy")
	}
}

func TestUploadPreservesHTTPClientTimeout(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Timeout = 100 * time.Millisecond
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"created","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(io.Discard, r.Body); <-r.Context().Done() })
	created, err := service.Upload(context.Background(), uploadInput(strings.NewReader("image-data")))
	if created == nil || created.ID != "created" || !errors.Is(err, context.DeadlineExceeded) || cloud.Provider.HTTPClient.Timeout != 100*time.Millisecond {
		t.Fatalf("created=%v err=%v timeout=%s", created, err, cloud.Provider.HTTPClient.Timeout)
	}
}
