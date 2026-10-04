package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type attachVolumeContractBody struct {
	data                  *strings.Reader
	readError, closeError error
	onClose               func()
	once                  sync.Once
	closes                atomic.Int32
}

func (b *attachVolumeContractBody) Read(p []byte) (int, error) {
	if b.readError != nil {
		n, _ := b.data.Read(p)
		cause := b.readError
		b.readError = nil
		return n, cause
	}
	return b.data.Read(p)
}
func (b *attachVolumeContractBody) Close() error {
	b.closes.Add(1)
	b.once.Do(func() {
		if b.onClose != nil {
			b.onClose()
		}
	})
	return b.closeError
}

func TestAttachVolumeContractsAcceptedBodyFailuresRetainPhaseProofWithoutResend(t *testing.T) {
	for _, phase := range []string{"checked", "created", "poll"} {
		for _, mode := range []string{"read", "close", "context", "source"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				nova, cinder := attachVolumeContractClients(cloud)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("accepted " + phase + " " + mode + " failure")
				var calls, retries atomic.Int32
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries.Add(1)
					return err
				}
				bodyText := attachVolumeContractObservation("available", `[]`)
				failStep := int32(1)
				if phase == "created" {
					bodyText, failStep = attachVolumeContractCreated(), 2
				}
				if phase == "poll" {
					bodyText, failStep = attachVolumeContractObservation("in-use", `null`), 3
				}
				broken := &attachVolumeContractBody{data: strings.NewReader(bodyText)}
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
				cloud.Provider.HTTPClient.Transport = attachVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
					step := calls.Add(1)
					if step == failStep {
						return attachVolumeContractResponse(200, broken, phase+"-failure"), nil
					}
					if step == 1 {
						return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractObservation("available", `[]`))), "checked"), nil
					}
					if step == 2 {
						return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractCreated())), "created"), nil
					}
					t.Error("accepted response failure caused resend or cleanup", r.Method, r.URL)
					return attachVolumeContractResponse(500, io.NopCloser(strings.NewReader(`{}`)), "unexpected"), nil
				})
				result, err := blockstorage.AttachVolume(ctx, nova, cinder, attachVolumeContractInput())
				var accepted *resource.ResponseError
				if err == nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != bodyText || accepted.Header.Get("X-Proof") != phase+"-failure" || result == nil || calls.Load() != failStep || retries.Load() != 0 || broken.closes.Load() != 1 || result.Ready != nil {
					t.Fatalf("result=%+v error=%v calls=%d retries=%d closes=%d", result, err, calls.Load(), retries.Load(), broken.closes.Load())
				}
				if mode == "source" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, cause) {
					t.Fatal("accepted failure cause lost", err)
				}
				if mode == "context" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				switch phase {
				case "checked":
					if result.Checked == nil || string(result.Checked.Body) != bodyText || result.Checked.Volume != nil || result.Created != nil || result.LastAccepted != nil {
						t.Fatal(result)
					}
				case "created":
					if result.Created == nil || string(result.Created.Body) != bodyText || result.Created.Attachment != nil || result.LastAccepted != nil {
						t.Fatal(result)
					}
				case "poll":
					attachVolumeContractCreatedProof(t, result)
					if result.LastAccepted == nil || string(result.LastAccepted.Body) != bodyText || result.LastAccepted.Volume != nil {
						t.Fatal(result)
					}
				}
				accepted.Body[0] = '!'
				accepted.Header.Set("X-Proof", "caller")
				var proofBody json.RawMessage
				var proofHeader http.Header
				switch phase {
				case "checked":
					proofBody, proofHeader = result.Checked.Body, result.Checked.Header
				case "created":
					proofBody, proofHeader = result.Created.Body, result.Created.Header
				case "poll":
					proofBody, proofHeader = result.LastAccepted.Body, result.LastAccepted.Header
				}
				if string(proofBody) != bodyText || proofHeader.Get("X-Proof") != phase+"-failure" {
					t.Fatal("error evidence aliases result proof")
				}
			})
		}
	}
}

func TestAttachVolumeContractsSourceSnapshotsLiveTokenAndConsistentVersions(t *testing.T) {
	cloud := testcloud.New(t)
	nova, cinder := attachVolumeContractClients(cloud)
	nova.MoreHeaders["openstack-api-version"] = "compute 2.79"
	cinder.MoreHeaders["openstack-api-version"] = "volume 3.60"
	var calls, options atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "option-token")
			testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
		case 2:
			attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, "option-token")
			attachVolumeContractPostBody(t, r, "/dev/vdb")
			w.Header().Set("X-Proof", "created")
			testcloud.JSON(w, 200, attachVolumeContractCreated())
		case 3:
			attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "option-token")
			testcloud.JSON(w, 200, attachVolumeContractObservation("in-use", `[]`))
		default:
			t.Error("unexpected source snapshot phase", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	var retained *blockstorage.AttachVolumeOpts
	result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(),
		func(value *blockstorage.AttachVolumeOpts) error {
			options.Add(1)
			value.Device = "/dev/vdb"
			retained = value
			nova.MoreHeaders["x-source"] = "changed-nova"
			cinder.MoreHeaders["x-source"] = "changed-cinder"
			cloud.Provider.SetToken("option-token")
			return nil
		},
		func(*blockstorage.AttachVolumeOpts) error {
			options.Add(1)
			retained.Device = "late change"
			return nil
		},
	)
	attachVolumeContractCreatedProof(t, result)
	if err != nil || result.Ready == nil || calls.Load() != 3 || options.Load() != 2 || nova.Endpoint != cloud.Server.URL+"/unused/nova/" || cinder.ResourceBase != cloud.Server.URL+attachVolumeContractCinderBase || nova.MoreHeaders["x-source"] != "changed-nova" {
		t.Fatalf("result=%+v error=%v HTTP=%d options=%d", result, err, calls.Load(), options.Load())
	}
}

func TestAttachVolumeContractsRejectSourceChangesAtOptionsAndAfterMutation(t *testing.T) {
	mutations := []struct {
		name   string
		change func(*gophercloud.ServiceClient)
	}{
		{"provider", func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }},
		{"endpoint", func(c *gophercloud.ServiceClient) { c.Endpoint += "changed/" }},
		{"resource base", func(c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" }},
		{"type", func(c *gophercloud.ServiceClient) { c.Type = "image" }},
		{"microversion", func(c *gophercloud.ServiceClient) { c.Microversion = "9.99" }},
	}
	for _, mutation := range mutations {
		t.Run("option/"+mutation.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			var calls, options atomic.Int32
			cloud.Provider.HTTPClient.Transport = attachVolumeContractTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), func(*blockstorage.AttachVolumeOpts) error { options.Add(1); mutation.change(nova); return nil }, func(*blockstorage.AttachVolumeOpts) error { options.Add(1); return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || options.Load() != 1 {
				t.Fatalf("result=%+v error=%v HTTP=%d options=%d", result, err, calls.Load(), options.Load())
			}
		})
		t.Run("accepted mutation/"+mutation.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = attachVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractObservation("available", `[]`))), "checked"), nil
				}
				attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, "test-token")
				mutation.change(cinder)
				return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractCreated())), "created"), nil
			})
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput())
			var accepted *resource.ResponseError
			if result == nil || result.Created == nil || result.Created.StatusCode != 200 || string(result.Created.Body) != attachVolumeContractCreated() || result.Created.Attachment != nil || result.LastAccepted != nil || result.Ready != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &accepted) || accepted.Header.Get("X-Proof") != "created" || calls.Load() != 2 {
				t.Fatalf("result=%+v error=%v HTTP=%d", result, err, calls.Load())
			}
		})
	}
	t.Run("progress callback changes source", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := attachVolumeContractClients(cloud)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = attachVolumeContractTransport(func(*http.Request) (*http.Response, error) {
			switch calls.Add(1) {
			case 1:
				return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractObservation("available", `[]`))), "checked"), nil
			case 2:
				return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractCreated())), "created"), nil
			default:
				return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractObservation("attaching", `[]`))), "poll"), nil
			}
		})
		result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{ProgressCallback: func(int) error { nova.Microversion = "2.99"; return nil }}))
		attachVolumeContractCreatedProof(t, result)
		if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 3 || result.LastAccepted == nil || result.Ready != nil {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
}

func TestAttachVolumeContractsWaitBudgetStartsAfterCreatedResponse(t *testing.T) {
	cloud := testcloud.New(t)
	nova, cinder := attachVolumeContractClients(cloud)
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = attachVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
		switch calls.Add(1) {
		case 1:
			return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractObservation("available", `[]`))), "checked"), nil
		case 2:
			time.Sleep(100 * time.Millisecond)
			if err := r.Context().Err(); err != nil {
				t.Errorf("wait-only timeout leaked into mutation: %v", err)
			}
			return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractCreated())), "created"), nil
		default:
			return attachVolumeContractResponse(200, io.NopCloser(strings.NewReader(attachVolumeContractObservation("in-use", `null`))), "ready"), nil
		}
	})
	timeout := 50 * time.Millisecond
	result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{Timeout: &timeout}))
	attachVolumeContractCreatedProof(t, result)
	if err != nil || result.Ready == nil || calls.Load() != 3 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
	t.Run("explicit zero is unlimited through a nonterminal observation", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := attachVolumeContractClients(cloud)
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
			case 2:
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 200, attachVolumeContractCreated())
			case 3:
				attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
				testcloud.JSON(w, 200, attachVolumeContractObservation("attaching", `[]`))
			case 4:
				attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
				testcloud.JSON(w, 200, attachVolumeContractObservation("in-use", `null`))
			default:
				t.Error("unexpected explicit-zero phase", r.Method, r.URL)
				w.WriteHeader(500)
			}
		})
		zero, interval := time.Duration(0), time.Millisecond
		result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{Timeout: &zero, PollInterval: &interval, ProgressCallback: func(int) error { callbacks.Add(1); return nil }}))
		attachVolumeContractCreatedProof(t, result)
		if err != nil || result.Ready == nil || calls.Load() != 4 || callbacks.Load() != 1 {
			t.Fatalf("result=%+v error=%v calls=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
		}
	})
}

func TestAttachVolumeContractsDistinctProvidersAndNullableOptionalServer(t *testing.T) {
	for _, serverField := range []string{"", `,"serverId":null`} {
		t.Run(serverField, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			provider := &gophercloud.ProviderClient{HTTPClient: *cloud.Server.Client()}
			provider.UseTokenLock()
			provider.SetToken("cinder-token")
			cinder.ProviderClient = provider
			// Endpoint need not have a trailing slash when the effective base does.
			nova.Endpoint = strings.TrimSuffix(nova.Endpoint, "/")
			var calls atomic.Int32
			created := `{"volumeAttachment":{"volumeId":"vol-1"` + serverField + `,"vendor":9007199254740993}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "cinder-token")
					testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
					return
				}
				attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, "test-token")
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 200, created)
			})
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeWait(false))
			if err != nil || result == nil || result.Created == nil || result.Created.Attachment == nil || result.Created.Attachment.VolumeID == nil || *result.Created.Attachment.VolumeID != "vol-1" || result.Created.Attachment.ServerID != nil || string(result.Created.Body) != created || calls.Load() != 2 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}

func TestAttachVolumeContractsNativeReauthenticationBackoffAndOwnedRetry(t *testing.T) {
	cloud := testcloud.New(t)
	nova, cinder := attachVolumeContractClients(cloud)
	var calls, posts, reauths, backoffs, retries atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("refreshed-token"); return nil }
	cloud.Provider.RetryBackoffFunc = func(_ context.Context, code *gophercloud.ErrUnexpectedResponseCode, _ error, count uint) error {
		backoffs.Add(1)
		if code.Actual != 429 || count != 1 {
			t.Errorf("backoff status=%d count=%d", code.Actual, count)
		}
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, err error, count uint) error {
		retries.Add(1)
		if !gophercloud.ResponseCodeIs(err, 503) {
			return err
		}
		if method != http.MethodPost || target != cloud.Server.URL+attachVolumeContractCreatePath || count != 2 || options.JSONBody == nil || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody {
			t.Errorf("retry method=%s URL=%s count=%d options=%+v", method, target, count, options)
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
			if posts.Load() == 0 {
				attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
				testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
			} else {
				attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "refreshed-token")
				testcloud.JSON(w, 200, attachVolumeContractObservation("in-use", `[]`))
			}
			return
		}
		count := posts.Add(1)
		token := "refreshed-token"
		if count == 1 {
			token = "test-token"
		}
		attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, token)
		attachVolumeContractPostBody(t, r, "/dev/vdb")
		switch count {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 429, `{"error":"throttled"}`)
		case 3:
			testcloud.JSON(w, 503, `{"error":"retry"}`)
		default:
			if r.Header.Get("X-Retry") != "native" {
				t.Error("native retry headers lost")
			}
			w.Header().Set("X-Proof", "created")
			testcloud.JSON(w, 200, attachVolumeContractCreated())
		}
	})
	result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeDevice("/dev/vdb"))
	attachVolumeContractCreatedProof(t, result)
	if err != nil || result.Ready == nil || calls.Load() != 6 || posts.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 || cloud.Provider.Token() != "refreshed-token" {
		t.Fatalf("result=%+v error=%v calls=%d posts=%d reauths=%d backoffs=%d retries=%d", result, err, calls.Load(), posts.Load(), reauths.Load(), backoffs.Load(), retries.Load())
	}
}

func TestAttachVolumeContractsNativeRetryCannotReplaceOwnedRequestOrStatusPolicy(t *testing.T) {
	for _, tc := range []struct {
		name         string
		mutate       func(*gophercloud.RequestOpts)
		secondStatus int
	}{
		{"JSON request", func(o *gophercloud.RequestOpts) {
			o.JSONBody = map[string]any{"volumeAttachment": map[string]any{"volumeId": "other"}}
		}, 200},
		{"raw request", func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("changed") }, 200},
		{"response target", func(o *gophercloud.RequestOpts) { o.JSONResponse = new(any) }, 200},
		{"response ownership", func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }, 200},
		{"success policy expansion", func(o *gophercloud.RequestOpts) { o.OkCodes = append(o.OkCodes, 201) }, 201},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			var calls, posts atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				tc.mutate(options)
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method == http.MethodGet {
					if posts.Load() != 0 {
						t.Error("failed creation triggered poll")
					}
					testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
					return
				}
				attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, "test-token")
				attachVolumeContractPostBody(t, r, "")
				if posts.Add(1) == 1 {
					testcloud.JSON(w, 503, `{"error":"retry"}`)
				} else {
					testcloud.JSON(w, tc.secondStatus, attachVolumeContractCreated())
				}
			})
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput())
			if err == nil || result == nil || result.Created != nil || result.LastAccepted != nil || result.Ready != nil {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if tc.secondStatus == 201 {
				if !errors.As(err, &native) || native.Actual != 201 || !reflect.DeepEqual(native.Expected, []int{200}) || calls.Load() != 3 || posts.Load() != 2 {
					t.Fatalf("expanded policy error=%v calls=%d posts=%d", err, calls.Load(), posts.Load())
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 || posts.Load() != 1 {
				t.Fatalf("owned request changed error=%v calls=%d posts=%d", err, calls.Load(), posts.Load())
			}
		})
	}
}
