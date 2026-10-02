package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/image/v2/imagedata"
	"gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

// Pinned Proxy.stage_image/Image.stage require exact queued status, stage binary
// data, then fetch metadata. These explicit Ref/reader calls retain their own
// response proof and deliberately avoid Python filename/size inference/cache.
const glanceStagePrefix = "/reverse/glance/v2/"

type glanceStageRoundTrip func(*http.Request) (*http.Response, error)

func (f glanceStageRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func glanceStageHTTP(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Evidence": {"actual"}}, Body: body}
}
func glanceStageKnown() *images.Image {
	return &images.Image{ID: "selected", Status: images.ImageStatusQueued}
}
func glanceStageJSON(id, status string) string {
	return fmt.Sprintf(`{"id":%q,"name":"worker","status":%q,"container_format":"bare","disk_format":"qcow2","created_at":"2024-01-02T03:04:05Z","vendor":{"precise":9007199254740993}}`, id, status)
}

type glanceStageReader struct {
	data                          *strings.Reader
	reads, closes, seeks, lengths atomic.Int32
}

func glanceStageData(value string) *glanceStageReader {
	return &glanceStageReader{data: strings.NewReader(value)}
}
func (r *glanceStageReader) Read(p []byte) (int, error) { r.reads.Add(1); return r.data.Read(p) }
func (r *glanceStageReader) Close() error               { r.closes.Add(1); return nil }
func (r *glanceStageReader) Seek(offset int64, whence int) (int64, error) {
	r.seeks.Add(1)
	return r.data.Seek(offset, whence)
}
func (r *glanceStageReader) Len() int { r.lengths.Add(1); return r.data.Len() }

func TestGlanceStageFreshKnownAndBorrowedReader(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(fmt.Sprint("known=", known), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + glanceStagePrefix
			data := glanceStageData("prefix-payload")
			if _, err := data.Seek(7, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			data.seeks.Store(0)
			var gets, puts, wrong atomic.Int32
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
				n := gets.Add(1)
				status, id := "queued", "initial-response-decoy"
				if known || n > 1 {
					status, id = "uploading", "fresh-response-decoy"
					w.Header().Set("X-Metadata", "fresh")
				}
				testcloud.JSON(w, 200, glanceStageJSON(id, status))
			})
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) {
				puts.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil || r.Method != http.MethodPut || string(body) != "payload" || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("Accept") != "" || r.ContentLength != -1 || r.Header.Get("X-OpenStack-Image-Size") != "" {
					t.Error("borrowed stream or owned binary headers changed", r.Method, string(body), r.Header, r.ContentLength, err)
				}
				w.Header().Set("X-Stage", "actual")
				w.Header().Set("Location", "https://foreign.invalid/tasks/not-followed")
				w.WriteHeader(204)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
			a := imagedata.New(client)
			var v *imagedata.StageImageResult
			var err error
			if known {
				v, err = a.StageKnownImage(context.Background(), glanceStageKnown(), data)
			} else {
				v, err = a.StageImage(context.Background(), resource.ID("selected"), data)
			}
			wantGets := int32(2)
			if known {
				wantGets = 1
			}
			if err != nil || v == nil || v.ImageID != "selected" || v.Image == nil || v.Image.ID != "fresh-response-decoy" || v.Image.Status != "uploading" || v.StatusCode != 200 || v.Header.Get("X-Metadata") != "fresh" || !bytes.Contains(v.Body, []byte("9007199254740993")) || v.Acknowledgement == nil || v.Acknowledgement.StatusCode != 204 || v.Acknowledgement.Header.Get("X-Stage") != "actual" || gets.Load() != wantGets || puts.Load() != 1 || wrong.Load() != 0 || data.closes.Load() != 0 || data.seeks.Load() != 0 || data.lengths.Load() != 0 {
				t.Fatal(v, err, gets.Load(), puts.Load(), wrong.Load(), data.closes.Load(), data.seeks.Load(), data.lengths.Load())
			}
			// Metadata and staging proof are separate actual responses.
			v.Header.Set("X-Stage", "caller")
			v.Body[0] = '!'
			v.Image.Status = "caller"
			if v.Acknowledgement.Header.Get("X-Stage") != "actual" || len(v.Acknowledgement.Body) != 0 {
				t.Fatal(v.Acknowledgement)
			}
			if err := data.Close(); err != nil || data.closes.Load() != 1 {
				t.Fatal("caller lost reader ownership", err, data.closes.Load())
			}
			if client.ResourceBase != cloud.Server.URL+glanceStagePrefix || client.Endpoint != cloud.Server.URL+"/catalog/unused/" {
				t.Fatal("source client changed", client)
			}
		})
	}
}

func TestGlanceStageSizeAndOptionSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name, payload, header string
		options               []imagedata.StageOption
	}{
		{"absent", "bytes", "", nil}, {"explicit zero", "", "0", []imagedata.StageOption{imagedata.WithStageSize(0)}}, {"size does not trim or frame", "payload", "999", []imagedata.StageOption{imagedata.WithStageSize(999)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var puts, gets atomic.Int32
			data := glanceStageData(tc.payload)
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) {
				puts.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != tc.payload || r.ContentLength != -1 || r.Header.Get("X-OpenStack-Image-Size") != tc.header {
					t.Error(string(body), err, r.Header, r.ContentLength)
				}
				_, present := r.Header["X-Openstack-Image-Size"]
				if present != (tc.header != "") {
					t.Error("size presence changed", r.Header)
				}
				w.WriteHeader(204)
			})
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, glanceStageJSON("selected", "uploading"))
			})
			if v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageKnownImage(context.Background(), glanceStageKnown(), data, tc.options...); err != nil || v == nil || puts.Load() != 1 || gets.Load() != 1 || data.closes.Load() != 0 || data.seeks.Load() != 0 || data.lengths.Load() != 0 {
				t.Fatal(v, err, puts.Load(), gets.Load(), data.closes.Load(), data.seeks.Load(), data.lengths.Load())
			}
		})
	}
	t.Run("bulk option reuse and last wins", func(t *testing.T) {
		cloud := testcloud.New(t)
		var puts atomic.Int32
		size := int64(7)
		headers := map[string]string{"X-Trace": "snapshot"}
		option := imagedata.WithStageOpts(imagedata.StageOpts{Size: &size, Headers: headers})
		size = -1
		headers["X-Trace"] = "mutated"
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) {
			puts.Add(1)
			_, _ = io.Copy(io.Discard, r.Body)
			if r.Header.Get("X-OpenStack-Image-Size") != "7" || r.Header.Get("X-Trace") != "final" {
				t.Error(r.Header)
			}
			w.WriteHeader(204)
		})
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, 200, glanceStageJSON("selected", "uploading"))
		})
		a := imagedata.New(cloud.Client("image", glanceStagePrefix))
		for range 2 {
			if v, err := a.StageKnownImage(context.Background(), glanceStageKnown(), strings.NewReader("payload"), imagedata.WithStageSize(99), option, imagedata.WithStageHeader("X-Trace", "earlier"), imagedata.WithStageHeader("x-trace", "final")); err != nil || v == nil {
				t.Fatal(v, err)
			}
		}
		if puts.Load() != 2 {
			t.Fatal(puts.Load())
		}
	})
	t.Run("known seed and retained custom options are isolated", func(t *testing.T) {
		cloud := testcloud.New(t)
		seed := glanceStageKnown()
		var retained *imagedata.StageOpts
		var puts, gets atomic.Int32
		option := imagedata.StageOption(func(o *imagedata.StageOpts) error {
			seed.ID = "other"
			seed.Status = "active"
			o.Size = new(int64)
			*o.Size = 7
			o.Headers = map[string]string{"X-Trace": "captured"}
			retained = o
			return nil
		})
		cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(r *http.Request) (*http.Response, error) {
			retained.Headers["X-Trace"] = "late"
			*retained.Size = 999
			if r.Header.Get("X-Trace") != "captured" {
				t.Error(r.Header)
			}
			if r.Method == http.MethodPut {
				puts.Add(1)
				if r.URL.Path != glanceStagePrefix+"images/selected/stage" || r.Header.Get("X-OpenStack-Image-Size") != "7" {
					t.Error(r.URL.String(), r.Header)
				}
				_, _ = io.Copy(io.Discard, r.Body)
				return glanceStageHTTP(204, io.NopCloser(strings.NewReader(""))), nil
			}
			gets.Add(1)
			if r.URL.Path != glanceStagePrefix+"images/selected" {
				t.Error(r.URL.String())
			}
			return glanceStageHTTP(200, io.NopCloser(strings.NewReader(glanceStageJSON("selected", "uploading")))), nil
		})
		if v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageKnownImage(context.Background(), seed, strings.NewReader("payload"), option); err != nil || v == nil || v.ImageID != "selected" || puts.Load() != 1 || gets.Load() != 1 {
			t.Fatal(v, err, puts.Load(), gets.Load())
		}
	})
}

func TestGlanceStageQueuedNativePreflightAndNameBinding(t *testing.T) {
	for _, body := range []string{`{`, `null`, `[]`, `{"id":"selected"}`, `{"status":null}`, `{"status":7}`, `{"status":"Queued"}`, `{"status":"active"}`, `{"STATUS":"queued"}`, `{"status":"queued","min_ram":"bad"}`, `{"status":"queued","created_at":"not-a-date"}`, `{"status":"queued","tags":[{}]}`, string([]byte{'{', '"', 's', 't', 'a', 't', 'u', 's', '"', ':', '"', 0xff, '"', '}'})} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, puts atomic.Int32
			data := glanceStageData("payload")
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				w.Header().Set("X-Initial", "actual")
				testcloud.JSON(w, 200, body)
			})
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) { puts.Add(1); w.WriteHeader(204) })
			v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageImage(context.Background(), resource.ID("selected"), data)
			var response *resource.ResponseError
			if v != nil || !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != body || response.Header.Get("X-Initial") != "actual" || gets.Load() != 1 || puts.Load() != 0 || data.reads.Load() != 0 || data.closes.Load() != 0 {
				t.Fatal(v, err, response, gets.Load(), puts.Load(), data.reads.Load(), data.closes.Load())
			}
			if strings.Contains(body, `"min_ram"`) {
				var decode *json.UnmarshalTypeError
				if !errors.As(err, &decode) {
					t.Fatal("native decoder cause lost", err)
				}
			}
		})
	}
	t.Run("exact name all pages then fresh queued check", func(t *testing.T) {
		cloud := testcloud.New(t)
		var lists, gets, puts atomic.Int32
		data := glanceStageData("payload")
		cloud.Mux.HandleFunc(glanceStagePrefix+"images", func(w http.ResponseWriter, r *http.Request) {
			lists.Add(1)
			if r.URL.Query().Get("name") != "worker" || r.Header.Get("X-Trace") != "prepared" {
				t.Error(r.URL.String(), r.Header)
			}
			if r.URL.Query().Get("marker") == "" {
				testcloud.JSON(w, 200, `{"images":[{"id":"case-decoy","name":"Worker"}],"next":"/v2/images?marker=second&name=worker"}`)
			} else {
				testcloud.JSON(w, 200, `{"images":[{"id":"selected","name":"worker","status":"active"}]}`)
			}
		})
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
			n := gets.Add(1)
			status := "queued"
			if n > 1 {
				status = "uploading"
			}
			testcloud.JSON(w, 200, glanceStageJSON("response-decoy", status))
		})
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) {
			puts.Add(1)
			body, _ := io.ReadAll(r.Body)
			if string(body) != "payload" {
				t.Error(string(body))
			}
			w.WriteHeader(204)
		})
		v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageImage(context.Background(), resource.Name("worker"), data, imagedata.WithStageHeader("X-Trace", "prepared"))
		if err != nil || v == nil || v.ImageID != "selected" || lists.Load() != 2 || gets.Load() != 2 || puts.Load() != 1 {
			t.Fatal(v, err, lists.Load(), gets.Load(), puts.Load())
		}
	})
	for _, mode := range []string{"missing", "duplicate", "late error"} {
		t.Run("name "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, wrong atomic.Int32
			data := glanceStageData("payload")
			cloud.Mux.HandleFunc(glanceStagePrefix+"images", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if mode == "missing" {
					testcloud.JSON(w, 200, `{"images":[]}`)
					return
				}
				if r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 200, `{"images":[{"id":"selected","name":"worker"}],"next":"/v2/images?marker=second&name=worker"}`)
					return
				}
				if mode == "duplicate" {
					testcloud.JSON(w, 200, `{"images":[{"id":"other","name":"worker"}]}`)
				} else {
					testcloud.JSON(w, 403, `{"denied":true}`)
				}
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
			v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageImage(context.Background(), resource.Name("worker"), data)
			if v != nil || err == nil || wrong.Load() != 0 || data.reads.Load() != 0 || (mode == "missing" && !errors.Is(err, resource.ErrNotFound)) || (mode == "duplicate" && !errors.Is(err, resource.ErrAmbiguous)) || (mode == "late error" && !gophercloud.ResponseCodeIs(err, 403)) {
				t.Fatal(v, err, lists.Load(), wrong.Load(), data.reads.Load())
			}
		})
	}
}

func TestGlanceStagePutNeverReplaysOrInvokesProviderCallbacks(t *testing.T) {
	for _, code := range []int{401, 429, 498, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var puts, gets, reauth, retries, backoffs atomic.Int32
			data := glanceStageData("payload")
			cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); return nil }
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return nil
			}
			cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoffs.Add(1)
				return nil
			}
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) {
				n := puts.Add(1)
				buf := make([]byte, 3)
				_, _ = io.ReadFull(r.Body, buf)
				if n > 1 {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, code, `{"failure":true}`)
			})
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, glanceStageJSON("selected", "uploading"))
			})
			v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageKnownImage(context.Background(), glanceStageKnown(), data)
			if v != nil || !gophercloud.ResponseCodeIs(err, code) || puts.Load() != 1 || gets.Load() != 0 || reauth.Load() != 0 || retries.Load() != 0 || backoffs.Load() != 0 || data.closes.Load() != 0 || data.seeks.Load() != 0 || cloud.Provider.ReauthFunc == nil || cloud.Provider.RetryFunc == nil || cloud.Provider.RetryBackoffFunc == nil {
				t.Fatal(v, err, puts.Load(), gets.Load(), reauth.Load(), retries.Load(), backoffs.Load(), data.closes.Load(), data.seeks.Load())
			}
		})
	}
	for _, code := range []int{301, 302, 303, 307, 308} {
		t.Run(fmt.Sprint("redirect", code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var puts, wrong, policy atomic.Int32
			data := glanceStageData("payload")
			cloud.Provider.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { policy.Add(1); return nil }
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) {
				puts.Add(1)
				http.Redirect(w, r, glanceStagePrefix+"images/other/stage", code)
			})
			cloud.Mux.HandleFunc(glanceStagePrefix+"images/other/stage", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(204) })
			v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageKnownImage(context.Background(), glanceStageKnown(), data)
			if v != nil || !gophercloud.ResponseCodeIs(err, code) || puts.Load() != 1 || wrong.Load() != 0 || policy.Load() != 0 || data.closes.Load() != 0 || data.seeks.Load() != 0 || cloud.Provider.HTTPClient.CheckRedirect == nil {
				t.Fatal(v, err, puts.Load(), wrong.Load(), policy.Load(), data.closes.Load(), data.seeks.Load())
			}
		})
	}
}

type glanceStageBrokenBody struct {
	first []byte
	cause error
}

func (r *glanceStageBrokenBody) Read(p []byte) (int, error) {
	if len(r.first) > 0 {
		n := copy(p, r.first)
		r.first = r.first[n:]
		return n, nil
	}
	return 0, r.cause
}
func (*glanceStageBrokenBody) Close() error { return nil }

type glanceStageBrokenReader struct {
	glanceStageBrokenBody
	closes, seeks atomic.Int32
}

func (r *glanceStageBrokenReader) Close() error { r.closes.Add(1); return nil }
func (r *glanceStageBrokenReader) Seek(int64, int) (int64, error) {
	r.seeks.Add(1)
	return 0, errors.New("caller reader cannot seek")
}

func TestGlanceStageStreamFailuresAndCallerCancellation(t *testing.T) {
	t.Run("reader failure remains original and is not replayed", func(t *testing.T) {
		cloud := testcloud.New(t)
		cause := errors.New("caller stream failed after prefix")
		data := &glanceStageBrokenReader{glanceStageBrokenBody: glanceStageBrokenBody{first: []byte("prefix"), cause: cause}}
		var attempts, callbacks atomic.Int32
		cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			callbacks.Add(1)
			return nil
		}
		cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(r *http.Request) (*http.Response, error) {
			attempts.Add(1)
			body, err := io.ReadAll(r.Body)
			if r.Method != http.MethodPut || string(body) != "prefix" || !errors.Is(err, cause) || r.GetBody != nil {
				t.Error(r.Method, string(body), err, r.GetBody != nil)
			}
			return nil, err
		})
		v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageKnownImage(context.Background(), glanceStageKnown(), data)
		if v != nil || !errors.Is(err, cause) || attempts.Load() != 1 || callbacks.Load() != 0 || data.closes.Load() != 0 || data.seeks.Load() != 0 {
			t.Fatal(v, err, attempts.Load(), callbacks.Load(), data.closes.Load(), data.seeks.Load())
		}
	})
	t.Run("transport404 is terminal and not missing", func(t *testing.T) {
		cloud := testcloud.New(t)
		cause := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		var attempts atomic.Int32
		cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(r *http.Request) (*http.Response, error) { attempts.Add(1); return nil, cause })
		v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageKnownImage(context.Background(), glanceStageKnown(), strings.NewReader("payload"))
		if v != nil || !errors.Is(err, cause) || errors.Is(err, resource.ErrNotFound) || attempts.Load() != 1 {
			t.Fatal(v, err, attempts.Load())
		}
	})
	t.Run("caller cancellation after actual stage arrival", func(t *testing.T) {
		cloud := testcloud.New(t)
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("caller stopped upload")
		data := glanceStageData("payload")
		entered, release := make(chan struct{}), make(chan struct{})
		t.Cleanup(func() { close(release) })
		var puts, gets atomic.Int32
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) {
			puts.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "payload" {
				t.Error(string(body), err)
			}
			close(entered)
			select {
			case <-r.Context().Done():
			case <-release:
			}
		})
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			testcloud.JSON(w, 200, glanceStageJSON("selected", "uploading"))
		})
		type answer struct {
			value *imagedata.StageImageResult
			err   error
		}
		done := make(chan answer, 1)
		go func() {
			v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageKnownImage(ctx, glanceStageKnown(), data)
			done <- answer{v, err}
		}()
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("PUT did not arrive")
		}
		cancel(cause)
		select {
		case got := <-done:
			if got.value != nil || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || puts.Load() != 1 || gets.Load() != 0 || data.closes.Load() != 0 || data.seeks.Load() != 0 {
				t.Fatal(got.value, got.err, puts.Load(), gets.Load(), data.closes.Load(), data.seeks.Load())
			}
		case <-time.After(5 * time.Second):
			t.Fatal("cancellation did not reach PUT")
		}
	})
}

func TestGlanceStageAcceptedAndFollowupEvidence(t *testing.T) {
	for _, tc := range []struct {
		name             string
		putCode, getCode int
		body             string
		readPhase        string
	}{
		{"PUT200", 200, 0, `{}`, ""}, {"PUT202", 202, 0, `{}`, ""}, {"PUT400", 400, 0, `{}`, ""},
		{"PUT204 read failure", 204, 0, `stage-prefix`, "PUT"},
		{"GET404", 204, 404, `{"missing":true}`, ""}, {"GET203", 204, 203, `{}`, ""},
		{"GET malformed", 204, 200, `{`, ""}, {"GET null", 204, 200, `null`, ""},
		{"GET native field", 204, 200, `{"min_ram":"bad"}`, ""}, {"GET date", 204, 200, `{"created_at":"bad"}`, ""},
		{"GET invalid UTF8", 204, 200, string([]byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xff, '"', '}'}), ""},
		{"GET read failure", 204, 200, `{"partial":`, "GET"},
		{"GET empty actual object", 204, 200, `{}`, ""}, {"GET null ID", 204, 200, `{"id":null,"status":"saving"}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cause := errors.New("actual response body stopped")
			var puts, gets, retries atomic.Int32
			var putHeader, getHeader http.Header
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(r *http.Request) (*http.Response, error) {
				code, body := tc.getCode, tc.body
				phase := "GET"
				if r.Method == http.MethodPut {
					puts.Add(1)
					phase, code = "PUT", tc.putCode
					_, _ = io.Copy(io.Discard, r.Body)
					if code == 204 && tc.readPhase != "PUT" {
						body = "ack-bytes"
					}
				} else {
					gets.Add(1)
				}
				if code == 0 {
					t.Error("unexpected follow-up GET")
					code = 500
				}
				var wireBody io.ReadCloser = io.NopCloser(strings.NewReader(body))
				if tc.readPhase == phase {
					wireBody = &glanceStageBrokenBody{first: []byte(body), cause: cause}
				}
				wire := glanceStageHTTP(code, wireBody)
				wire.Header.Set("X-Phase", phase)
				if phase == "PUT" {
					putHeader = wire.Header
				} else {
					getHeader = wire.Header
				}
				return wire, nil
			})
			v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageKnownImage(context.Background(), glanceStageKnown(), strings.NewReader("payload"))
			if puts.Load() != 1 {
				t.Fatal(v, err, puts.Load())
			}
			if tc.putCode != 204 {
				if v != nil || !gophercloud.ResponseCodeIs(err, tc.putCode) || gets.Load() != 0 || retries.Load() != 0 {
					t.Fatal(v, err, gets.Load(), retries.Load())
				}
				return
			}
			if v == nil || v.ImageID != "selected" || v.Acknowledgement == nil || v.Acknowledgement.StatusCode != 204 || v.Acknowledgement.Header.Get("X-Phase") != "PUT" {
				t.Fatal(v, err)
			}
			putHeader.Set("X-Phase", "wire-mutated")
			if v.Acknowledgement.Header.Get("X-Phase") != "PUT" {
				t.Fatal("ack shares response header", v)
			}
			if tc.readPhase == "PUT" {
				var proof *resource.ResponseError
				if !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != tc.body || string(v.Acknowledgement.Body) != tc.body || gets.Load() != 0 || v.StatusCode != 0 || v.Body != nil || v.Header != nil || v.Image != nil || retries.Load() != 0 {
					t.Fatal(v, err, proof, gets.Load(), retries.Load())
				}
				v.Acknowledgement.Body[0] = '!'
				if string(proof.Body) != tc.body {
					t.Fatal("ack body shares error proof")
				}
				return
			}
			if string(v.Acknowledgement.Body) != "ack-bytes" || gets.Load() != 1 {
				t.Fatal(v, err, gets.Load())
			}
			if tc.getCode != 200 {
				if !gophercloud.ResponseCodeIs(err, tc.getCode) || v.Image != nil || v.StatusCode != 0 || v.Body != nil || v.Header != nil {
					t.Fatal(v, err)
				}
				return
			}
			if v.StatusCode != 200 || string(v.Body) != tc.body || v.Header.Get("X-Phase") != "GET" {
				t.Fatal(v, err)
			}
			getHeader.Set("X-Phase", "wire-mutated")
			if v.Header.Get("X-Phase") != "GET" || v.Acknowledgement.Header.Get("X-Phase") != "PUT" {
				t.Fatal(v)
			}
			if tc.name == "GET empty actual object" || tc.name == "GET null ID" {
				if err != nil || v.Image == nil || v.Image.ID != "" {
					t.Fatal("response ID was fabricated", v, err)
				}
				return
			}
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != tc.body || proof.Header.Get("X-Phase") != "GET" || v.Image != nil || retries.Load() != 0 {
				t.Fatal(v, err, proof, retries.Load())
			}
			if tc.readPhase == "GET" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			v.Body[0] = '!'
			v.Header.Set("X-Phase", "caller")
			if string(proof.Body) != tc.body || proof.Header.Get("X-Phase") != "GET" {
				t.Fatal("metadata proof shares result", proof)
			}
		})
	}
	for _, phase := range []string{"PUT", "GET"} {
		for _, accepted := range []bool{false, true} {
			t.Run(fmt.Sprint("canceled ", phase, " accepted=", accepted), func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("caller canceled at response")
				var calls atomic.Int32
				cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Method == http.MethodPut {
						_, _ = io.Copy(io.Discard, r.Body)
					}
					code, body := 200, `{}`
					if r.Method == http.MethodPut {
						code, body = 204, "ack"
					}
					if r.Method == phase {
						cancel(cause)
						if !accepted {
							code, body = 404, `{"missing":true}`
						}
					}
					return glanceStageHTTP(code, io.NopCloser(strings.NewReader(body))), nil
				})
				v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageKnownImage(ctx, glanceStageKnown(), strings.NewReader("payload"))
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal(v, err)
				}
				want := int32(1)
				if phase == "GET" {
					want = 2
				}
				if calls.Load() != want {
					t.Fatal(v, err, calls.Load())
				}
				if phase == "PUT" && !accepted {
					if v != nil {
						t.Fatal(v)
					}
					return
				}
				if v == nil || v.Acknowledgement == nil || v.Acknowledgement.StatusCode != 204 {
					t.Fatal(v, err)
				}
				var proof *resource.ResponseError
				if accepted {
					code := 204
					if phase == "GET" {
						code = 200
					}
					if !errors.As(err, &proof) || proof.StatusCode != code || (phase == "GET" && (v.StatusCode != 200 || string(v.Body) != `{}`)) {
						t.Fatal(v, err, proof)
					}
				} else if !gophercloud.ResponseCodeIs(err, 404) || v.StatusCode != 0 {
					t.Fatal(v, err)
				}
			})
		}
	}
}

func TestGlanceStageClosedOptionsAndPreflight(t *testing.T) {
	type invocation func(*testcloud.Cloud, *glanceStageReader) (*imagedata.StageImageResult, error)
	cases := []struct {
		name string
		call invocation
		want error
	}{
		{"nil API", func(_ *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			var a *imagedata.API
			return a.StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"nil client", func(_ *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			return imagedata.New(nil).StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"nil provider", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			client := c.Client("image", glanceStagePrefix)
			client.ProviderClient = nil
			return imagedata.New(client).StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"nil context", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			return imagedata.New(c.Client("image", glanceStagePrefix)).StageImage(nil, resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"nil reader", func(c *testcloud.Cloud, _ *glanceStageReader) (*imagedata.StageImageResult, error) {
			return imagedata.New(c.Client("image", glanceStagePrefix)).StageImage(context.Background(), resource.ID("selected"), nil)
		}, resource.ErrInvalidOption},
		{"typed nil reader", func(c *testcloud.Cloud, _ *glanceStageReader) (*imagedata.StageImageResult, error) {
			var d *glanceStageReader
			return imagedata.New(c.Client("image", glanceStagePrefix)).StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"nil known image", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			return imagedata.New(c.Client("image", glanceStagePrefix)).StageKnownImage(context.Background(), nil, d)
		}, resource.ErrInvalidOption},
		{"known not queued", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			return imagedata.New(c.Client("image", glanceStagePrefix)).StageKnownImage(context.Background(), &images.Image{ID: "selected", Status: "Queued"}, d)
		}, resource.ErrInvalidOption},
		{"wrong service", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			return imagedata.New(c.Client("compute", glanceStagePrefix)).StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrUnsupported},
		{"foreign resource base", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			client := c.Client("image", glanceStagePrefix)
			client.ResourceBase = "https://foreign.invalid/v2/"
			return imagedata.New(client).StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"query resource base", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			client := c.Client("image", glanceStagePrefix)
			client.ResourceBase = c.Server.URL + glanceStagePrefix + "?wrong=true"
			return imagedata.New(client).StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"source size", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			client := c.Client("image", glanceStagePrefix)
			client.MoreHeaders = map[string]string{"X-OpenStack-Image-Size": "7"}
			return imagedata.New(client).StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"source alias conflict", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			client := c.Client("image", glanceStagePrefix)
			client.MoreHeaders = map[string]string{"X-Trace": "first", "x-trace": "second"}
			return imagedata.New(client).StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"version header conflict", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			client := c.Client("image", glanceStagePrefix)
			client.Microversion = "2.6"
			client.MoreHeaders = map[string]string{"OpenStack-API-Version": "image 2.7"}
			return imagedata.New(client).StageImage(context.Background(), resource.ID("selected"), d)
		}, resource.ErrInvalidOption},
		{"source changed by option", func(c *testcloud.Cloud, d *glanceStageReader) (*imagedata.StageImageResult, error) {
			client := c.Client("image", glanceStagePrefix)
			return imagedata.New(client).StageImage(context.Background(), resource.ID("selected"), d, func(*imagedata.StageOpts) error { client.Type = "compute"; return nil })
		}, resource.ErrUnsupported},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			data := glanceStageData("payload")
			cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return glanceStageHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
			})
			v, err := tc.call(cloud, data)
			if v != nil || !errors.Is(err, tc.want) || calls.Load() != 0 || data.reads.Load() != 0 || data.closes.Load() != 0 {
				t.Fatal(v, err, calls.Load(), data.reads.Load(), data.closes.Load())
			}
		})
	}
	for _, id := range []string{"", ".", "..", "a/b", "%2F", "with space", "a:b", "a\\b", string([]byte{0xff})} {
		t.Run("unsafe ID "+id, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			data := glanceStageData("payload")
			cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return glanceStageHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
			})
			v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageImage(context.Background(), resource.ID(id), data)
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || data.reads.Load() != 0 {
				t.Fatal(v, err, calls.Load(), data.reads.Load())
			}
		})
	}
	options := []struct {
		name   string
		values []imagedata.StageOption
	}{
		{"negative size", []imagedata.StageOption{imagedata.WithStageSize(-1)}}, {"nil option", []imagedata.StageOption{nil}},
		{"caller alias conflict", []imagedata.StageOption{imagedata.WithStageOpts(imagedata.StageOpts{Headers: map[string]string{"X-Trace": "first", "x-trace": "second"}})}},
		{"bulk header alias conflict", []imagedata.StageOption{imagedata.WithStageHeaders(map[string]string{"X-Trace": "first", "x-trace": "second"})}},
		{"bad header value", []imagedata.StageOption{imagedata.WithStageHeader("X-Trace", "bad\r\nvalue")}},
	}
	for _, key := range []string{"Accept", "Content-Type", "X-OpenStack-Image-Size", "X-Auth-Token", "X-Service-Token", "Authorization", "OpenStack-API-Version", "Content-Length", "Transfer-Encoding"} {
		options = append(options, struct {
			name   string
			values []imagedata.StageOption
		}{key, []imagedata.StageOption{imagedata.WithStageHeader(key, "override")}})
	}
	for _, tc := range options {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			data := glanceStageData("payload")
			cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return glanceStageHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
			})
			v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageImage(context.Background(), resource.ID("selected"), data, tc.values...)
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || data.reads.Load() != 0 {
				t.Fatal(v, err, calls.Load(), data.reads.Load())
			}
		})
	}
	t.Run("already canceled preserves custom cause", func(t *testing.T) {
		cloud := testcloud.New(t)
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("caller canceled before request")
		cancel(cause)
		var calls atomic.Int32
		data := glanceStageData("payload")
		cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return glanceStageHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
		})
		v, err := imagedata.New(cloud.Client("image", glanceStagePrefix)).StageImage(ctx, resource.ID("selected"), data)
		if v != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls.Load() != 0 || data.reads.Load() != 0 {
			t.Fatal(v, err, calls.Load(), data.reads.Load())
		}
	})
}

func TestGlanceStageLiveProviderFixedRoutesAndNativeABI(t *testing.T) {
	t.Run("current tokens captured headers metadata retry", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/catalog/unused/")
		client.ResourceBase = cloud.Server.URL + glanceStagePrefix
		client.MoreHeaders = map[string]string{"X-Source": "configured", "Accept": "application/json", "Content-Type": "application/metadata"}
		client.Microversion = "2.6"
		cloud.Provider.SetToken("initial")
		var gets, puts, middleware, reauth, retries atomic.Int32
		type callerKey struct{}
		transport := cloud.Provider.HTTPClient.Transport
		cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(r *http.Request) (*http.Response, error) {
			middleware.Add(1)
			if r.Context().Value(callerKey{}) != "caller" || r.Header.Get("X-Source") != "configured" || r.Header.Get("X-Trace") != "final" || r.Header.Get("OpenStack-API-Version") != "image 2.6" {
				t.Error(r.Context(), r.Header)
			}
			return transport.RoundTrip(r)
		})
		cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("reauthenticated"); return nil }
		cloud.Provider.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			retries.Add(1)
			if method != http.MethodGet || target != cloud.Server.URL+glanceStagePrefix+"images/selected" || !gophercloud.ResponseCodeIs(err, 503) {
				return err
			}
			return nil
		}
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
			n := gets.Add(1)
			if r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/metadata" {
				t.Error(r.Header)
			}
			switch n {
			case 1:
				if r.Header.Get("X-Auth-Token") != "initial" {
					t.Error(r.Header)
				}
				cloud.Provider.SetToken("for-stage")
				testcloud.JSON(w, 200, glanceStageJSON("decoy", "queued"))
			case 2:
				if r.Header.Get("X-Auth-Token") != "for-fetch" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 401, `{}`)
			case 3:
				if r.Header.Get("X-Auth-Token") != "reauthenticated" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 503, `{}`)
			default:
				if r.Header.Get("X-Auth-Token") != "reauthenticated" {
					t.Error(r.Header)
				}
				w.Header().Set("X-Metadata", "final")
				w.Header().Set("OpenStack-Image-Import-Methods", "glance-direct,web-download")
				w.Header().Set("OpenStack-Image-Store-Ids", "fast,slow")
				testcloud.JSON(w, 200, glanceStageJSON("actual-response-id", "uploading"))
			}
		})
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) {
			puts.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "payload" || r.Header.Get("X-Auth-Token") != "for-stage" || r.Header.Get("Accept") != "" || r.Header.Get("Content-Type") != "application/octet-stream" {
				t.Error(string(body), err, r.Header)
			}
			cloud.Provider.SetToken("for-fetch")
			w.WriteHeader(204)
		})
		v, err := imagedata.New(client).StageImage(context.WithValue(context.Background(), callerKey{}, "caller"), resource.ID("selected"), strings.NewReader("payload"), imagedata.WithStageHeader("X-Trace", "earlier"), imagedata.WithStageHeader("x-trace", "final"))
		if err != nil || v == nil || v.ImageID != "selected" || v.Image == nil || v.Image.ID != "actual-response-id" || v.Header.Get("X-Metadata") != "final" || gets.Load() != 4 || puts.Load() != 1 || middleware.Load() != 5 || reauth.Load() != 1 || retries.Load() != 1 || client.Endpoint != cloud.Server.URL+"/catalog/unused/" || client.ResourceBase != cloud.Server.URL+glanceStagePrefix || client.MoreHeaders["Content-Type"] != "application/metadata" || cloud.Provider.ReauthFunc == nil || cloud.Provider.RetryFunc == nil {
			t.Fatal(v, err, gets.Load(), puts.Load(), middleware.Load(), reauth.Load(), retries.Load(), client)
		}
		if strings.Join(v.Image.OpenStackImageImportMethods, ",") != "glance-direct,web-download" || strings.Join(v.Image.OpenStackImageStoreIDs, ",") != "fast,slow" || v.Header.Get("OpenStack-Image-Store-Ids") != "fast,slow" || string(v.Body) != glanceStageJSON("actual-response-id", "uploading") {
			t.Fatal("native header projection altered actual JSON proof", v.Image, string(v.Body), v.Header)
		}
	})
	t.Run("captured path and headers survive later valid source configuration", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", glanceStagePrefix)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.Microversion = "2.6"
		var gets, puts atomic.Int32
		cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Source") != "captured" || r.Header.Get("OpenStack-API-Version") != "image 2.6" {
				t.Error(r.Header)
			}
			if r.Method == http.MethodPut {
				puts.Add(1)
				if r.URL.Path != glanceStagePrefix+"images/selected/stage" {
					t.Error(r.URL.String())
				}
				_, _ = io.Copy(io.Discard, r.Body)
				return glanceStageHTTP(204, io.NopCloser(strings.NewReader(""))), nil
			}
			n := gets.Add(1)
			if r.URL.Path != glanceStagePrefix+"images/selected" {
				t.Error(r.URL.String())
			}
			status := "uploading"
			if n == 1 {
				status = "queued"
				client.ResourceBase = cloud.Server.URL + "/changed/v2/"
				client.MoreHeaders = map[string]string{"X-Source": "later"}
				client.Microversion = "2.9"
			}
			return glanceStageHTTP(200, io.NopCloser(strings.NewReader(glanceStageJSON("decoy", status)))), nil
		})
		v, err := imagedata.New(client).StageImage(context.Background(), resource.ID("selected"), strings.NewReader("payload"))
		if err != nil || v == nil || gets.Load() != 2 || puts.Load() != 1 || client.ResourceBase != cloud.Server.URL+"/changed/v2/" || client.MoreHeaders["X-Source"] != "later" {
			t.Fatal(v, err, gets.Load(), puts.Load(), client)
		}
	})
	for _, phase := range []string{"initial GET", "PUT"} {
		t.Run("source recheck after "+phase, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", glanceStagePrefix)
			var gets, puts atomic.Int32
			cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodPut {
					puts.Add(1)
					_, _ = io.Copy(io.Discard, r.Body)
					client.Type = "compute"
					return glanceStageHTTP(204, io.NopCloser(strings.NewReader("ack"))), nil
				}
				gets.Add(1)
				if phase == "initial GET" {
					client.Type = "compute"
				}
				return glanceStageHTTP(200, io.NopCloser(strings.NewReader(glanceStageJSON("selected", "queued")))), nil
			})
			v, err := imagedata.New(client).StageImage(context.Background(), resource.ID("selected"), strings.NewReader("payload"))
			if !errors.Is(err, resource.ErrUnsupported) || gets.Load() != 1 {
				t.Fatal(v, err, gets.Load(), puts.Load())
			}
			if phase == "initial GET" {
				if v != nil || puts.Load() != 0 {
					t.Fatal(v, err, puts.Load())
				}
			} else if v == nil || v.Image != nil || v.StatusCode != 0 || v.Acknowledgement == nil || string(v.Acknowledgement.Body) != "ack" || puts.Load() != 1 {
				t.Fatal(v, err, puts.Load())
			}
		})
	}
	t.Run("concurrent option reuse equal header aliases", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", glanceStagePrefix)
		client.MoreHeaders = map[string]string{"X-Source": "equal", "x-source": "equal"}
		size := int64(7)
		headers := map[string]string{"X-Trace": "snapshot", "x-trace": "snapshot"}
		option := imagedata.WithStageOpts(imagedata.StageOpts{Size: &size, Headers: headers})
		size = -1
		headers["X-Trace"] = "changed"
		var puts, gets atomic.Int32
		cloud.Provider.HTTPClient.Transport = glanceStageRoundTrip(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Source") != "equal" || r.Header.Get("X-Trace") != "snapshot" {
				t.Error(r.Header)
			}
			if r.Method == http.MethodPut {
				puts.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != "payload" || r.Header.Get("X-OpenStack-Image-Size") != "7" {
					t.Error(string(body), err, r.Header)
				}
				return glanceStageHTTP(204, io.NopCloser(strings.NewReader("ack"))), nil
			}
			gets.Add(1)
			return glanceStageHTTP(200, io.NopCloser(strings.NewReader(glanceStageJSON("selected", "uploading")))), nil
		})
		a := imagedata.New(client)
		done := make(chan error, 2)
		for range 2 {
			go func() {
				v, err := a.StageKnownImage(context.Background(), glanceStageKnown(), strings.NewReader("payload"), option)
				if err == nil && (v == nil || v.Acknowledgement == nil || v.Image == nil) {
					err = errors.New("actual workflow proof missing")
				}
				done <- err
			}()
		}
		for range 2 {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
		if puts.Load() != 2 || gets.Load() != 2 {
			t.Fatal(puts.Load(), gets.Load())
		}
	})
	t.Run("native Stage ABI remains a direct PUT", func(t *testing.T) {
		cloud := testcloud.New(t)
		var puts, gets atomic.Int32
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected/stage", func(w http.ResponseWriter, r *http.Request) {
			puts.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || r.Method != http.MethodPut || string(body) != "native" || r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("X-OpenStack-Image-Size") != "" {
				t.Error(r.Method, string(body), err, r.Header)
			}
			w.WriteHeader(204)
		})
		cloud.Mux.HandleFunc(glanceStagePrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			testcloud.JSON(w, 200, glanceStageJSON("selected", "active"))
		})
		if err := imagedata.New(cloud.Client("image", glanceStagePrefix)).Stage(context.Background(), "selected", strings.NewReader("native")); err != nil || puts.Load() != 1 || gets.Load() != 0 {
			t.Fatal(err, puts.Load(), gets.Load())
		}
	})
}
