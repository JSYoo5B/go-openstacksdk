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

func glResponse(r *http.Request, code int, body io.ReadCloser, proof string) *http.Response {
	return getVolumesContractResponse(r, code, body, proof)
}

func TestGetVolumeLimitsAcceptedReadCloseCancellationOwnsCurrentStageAndJoinsEveryCause(t *testing.T) {
	for _, stage := range []string{"member", "list", "limits", "member + source", "list + source", "limits + source"} {
		t.Run(stage, func(t *testing.T) {
			changedSource := strings.HasSuffix(stage, " + source")
			stage = strings.TrimSuffix(stage, " + source")
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("current read failure"), errors.New("current close failure"), errors.New("current close cancellation")
			input := "ref"
			body := `{"project":{"id":"requested"}}`
			if stage == "list" {
				input = "name/only"
				body = `{"projects":[{"name":"name/only","id":"requested"}]}`
			} else if stage == "limits" {
				body = `{"limits":{"absolute":{"maxTotalVolumes":7}}}`
			}
			reader := &getVolumesContractReader{data: strings.NewReader(body), readError: readCause, closeError: closeCause, onClose: func() {
				if changedSource {
					glMutateSource(cinder, "resource base")
				}
				cancel(cancelCause)
			}}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 && stage == "limits" {
					glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
					return glResponse(r, 200, io.NopCloser(strings.NewReader(`{"project":{"id":"requested"}}`)), "project"), nil
				}
				if stage == "limits" {
					volumeReadContractWire(t, r, glLimitsPath, "test-token")
				} else {
					path := glProjectsPath + "/ref"
					if stage == "list" {
						path = glProjectsPath
					}
					glIdentityWire(t, r, path, "test-token")
				}
				return glResponse(r, 203, reader, "current"), nil
			})
			result, err := glCall(ctx, cinder, identity, input)
			var physical *resource.ResponseError
			want := int32(1)
			if stage == "limits" {
				want = 2
			}
			if result == nil || result.Project == nil || result.Value != nil || result.Limits != nil || calls.Load() != want || reader.closes.Load() != 1 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "current" {
				t.Fatal(result, err, calls.Load(), reader.closes.Load())
			}
			if changedSource && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("simultaneous source drift was lost under cancellation", err)
			}
			var proofBody []byte
			var proofHeader http.Header
			switch stage {
			case "limits":
				if result.Project.Project == nil || result.Project.Observed == nil || result.Project.Observed.Header.Get("X-Proof") != "project" || result.Observed == nil || string(result.RequestedProjectID) != `"requested"` {
					t.Fatal(result)
				}
				proofBody, proofHeader = result.Observed.Body, result.Observed.Header
			case "member":
				if result.Project.Project != nil || result.Project.Observed == nil || result.Observed != nil || result.RequestedProjectID != nil {
					t.Fatal(result)
				}
				proofBody, proofHeader = result.Project.Observed.Body, result.Project.Observed.Header
			case "list":
				if result.Project.Project != nil || result.Project.Observed != nil || len(result.Project.Pages) != 1 || result.Observed != nil || result.RequestedProjectID != nil {
					t.Fatal(result)
				}
				proofBody, proofHeader = result.Project.Pages[0].Body, result.Project.Pages[0].Header
			}
			physical.Body[0] = '!'
			physical.Header.Set("X-Proof", "changed")
			if string(proofBody) != body || proofHeader.Get("X-Proof") != "current" {
				t.Fatal("current accepted proof aliases error")
			}
			glOperation(t, err)
		})
	}
}

func TestGetVolumeLimitsNativeRejectionAndTransportErrorsNeverBorrowEarlierProjectProof(t *testing.T) {
	for _, stage := range []string{"member", "list", "limits"} {
		for _, transport := range []bool{false, true} {
			t.Run(stage+map[bool]string{false: "/native", true: "/transport"}[transport], func(t *testing.T) {
				cloud := testcloud.New(t)
				cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
				var calls atomic.Int32
				cause := errors.New("current transport failure")
				cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					if stage != "member" && n == 1 {
						glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
						return glResponse(r, 404, io.NopCloser(strings.NewReader(`{"error":"old missing"}`)), "suppressed member"), nil
					}
					if stage == "limits" && n == 2 {
						glIdentityWire(t, r, glProjectsPath, "test-token")
						return glResponse(r, 203, io.NopCloser(strings.NewReader(`{"projects":[{"name":"ref","id":"requested"}]}`)), "project list"), nil
					}
					if stage == "limits" {
						volumeReadContractWire(t, r, glLimitsPath, "test-token")
					} else {
						path := glProjectsPath
						if stage == "member" {
							path += "/ref"
						}
						glIdentityWire(t, r, path, "test-token")
					}
					if transport {
						return nil, cause
					}
					return glResponse(r, 429, io.NopCloser(strings.NewReader(`{"error":"current rejection"}`)), "current rejection"), nil
				})
				result, err := glCall(context.Background(), cinder, identity, "ref")
				var physical *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				want := int32(1)
				if stage == "list" {
					want = 2
				} else if stage == "limits" {
					want = 3
				}
				if result == nil || result.Project == nil || result.Observed != nil || result.Limits != nil || result.Value != nil || result.Project.Observed != nil || calls.Load() != want || errors.As(err, &physical) {
					t.Fatal(result, err, calls.Load())
				}
				if stage == "limits" {
					if result.Project.Project == nil || len(result.Project.Pages) != 1 || result.Project.Pages[0].Header.Get("X-Proof") != "project list" || string(result.RequestedProjectID) != `"requested"` {
						t.Fatal(result)
					}
				} else if result.Project.Project != nil || len(result.Project.Pages) != 0 || result.RequestedProjectID != nil {
					t.Fatal(result)
				}
				if transport {
					if !errors.Is(err, cause) || errors.As(err, &native) {
						t.Fatal(err)
					}
				} else if !errors.As(err, &native) || native.Actual != 429 || string(native.Body) != `{"error":"current rejection"}` || native.ResponseHeader.Get("X-Proof") != "current rejection" {
					t.Fatal(err)
				}
				glOperation(t, err)
			})
		}
	}
}

func TestGetVolumeLimitsAcceptedProjectSchemaErrorsAreTerminalBeforeFallbackOrLimits(t *testing.T) {
	for _, tc := range []struct {
		body string
		list bool
	}{{body: `null`}, {body: `[]`}, {body: `{"project":null}`}, {body: `{"project":false}`}, {body: string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})}, {body: `{"Projects":[]}`, list: true}, {body: `{"projects":false}`, list: true}, {body: `{"projects":[false]}`, list: true}} {
		t.Run(tc.body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			var calls atomic.Int32
			path, input := glProjectsPath+"/ref", "ref"
			if tc.list {
				path, input = glProjectsPath, "name/only"
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				glIdentityWire(t, r, path, "test-token")
				calls.Add(1)
				w.Header().Set("X-Proof", "current project")
				testcloud.JSON(w, 203, tc.body)
			})
			result, err := glCall(context.Background(), cinder, identity, input)
			var physical *resource.ResponseError
			if result == nil || result.Project == nil || result.Project.Project != nil || result.RequestedProjectID != nil || result.Observed != nil || result.Value != nil || calls.Load() != 1 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != tc.body || physical.Header.Get("X-Proof") != "current project" {
				t.Fatal(result, err, calls.Load())
			}
			if tc.list {
				if result.Project.Observed != nil || len(result.Project.Pages) != 1 {
					t.Fatal(result)
				}
			} else if result.Project.Observed == nil || len(result.Project.Pages) != 0 {
				t.Fatal(result)
			}
		})
	}
}

func TestGetVolumeLimitsSelectedSourcesRemainFrozenAcrossBothAcceptedStages(t *testing.T) {
	for _, stage := range []string{"member", "limits"} {
		for _, selected := range []string{"cinder", "identity"} {
			for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
				t.Run(stage+"/"+selected+"/"+fact, func(t *testing.T) {
					cloud := testcloud.New(t)
					cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
					var calls atomic.Int32
					source := cinder
					if selected == "identity" {
						source = identity
					}
					body := `{"project":{"id":"requested"}}`
					if stage == "limits" {
						body = `{"limits":{}}`
					}
					reader := &getVolumesContractReader{data: strings.NewReader(body), onClose: func() { glMutateSource(source, fact) }}
					cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
						if calls.Add(1) == 1 && stage == "limits" {
							return glResponse(r, 200, io.NopCloser(strings.NewReader(`{"project":{"id":"requested"}}`)), "project"), nil
						}
						return glResponse(r, 203, reader, "current"), nil
					})
					result, err := glCall(context.Background(), cinder, identity, "ref")
					var physical *resource.ResponseError
					want := int32(1)
					if stage == "limits" {
						want = 2
					}
					if result == nil || result.Project == nil || result.Value != nil || result.Limits != nil || calls.Load() != want || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || reader.closes.Load() != 1 {
						t.Fatal(result, err, calls.Load())
					}
					if stage == "limits" {
						if result.Project.Project == nil || result.Observed == nil {
							t.Fatal(result)
						}
					} else if result.Project.Project != nil || result.Project.Observed == nil || result.Observed != nil {
						t.Fatal(result)
					}
				})
			}
		}
	}
}

func TestGetVolumeLimitsBothStagesHonorNativeReauthAndBoundedRetryWithLiveTokens(t *testing.T) {
	for _, stage := range []string{"member", "limits"} {
		t.Run(stage, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			var calls, reauths, retries atomic.Int32
			cloud.Provider.ReauthFunc = func(context.Context) error {
				reauths.Add(1)
				cloud.Provider.SetToken("reauth-token")
				cinder.MoreHeaders["x-source"] = "reauth changed"
				identity.MoreHeaders["x-identity"] = "reauth changed"
				return nil
			}
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, opts *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if retries.Add(1) > 1 {
					return errors.New("bounded native fixture exhausted")
				}
				if method != http.MethodGet || opts.JSONBody != nil || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody || len(opts.OkCodes) != 300 || opts.OkCodes[0] != 100 || opts.OkCodes[299] != 399 {
					t.Error("owned native GET changed", method, target, opts)
				}
				opts.MoreHeaders["X-Native"] = "retry"
				cloud.Provider.SetToken("retry-token")
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				first := int32(1)
				if stage == "limits" {
					first = 2
					if n == 1 {
						glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
						testcloud.JSON(w, 200, `{"project":{"id":"requested"}}`)
						return
					}
				}
				if stage == "member" && n == 4 {
					volumeReadContractWire(t, r, glLimitsPath, "retry-token")
					testcloud.JSON(w, 200, `{"limits":{}}`)
					return
				}
				token := "test-token"
				if n == first+1 {
					token = "reauth-token"
				} else if n == first+2 {
					token = "retry-token"
				}
				if stage == "member" {
					glIdentityWire(t, r, glProjectsPath+"/ref", token)
				} else {
					volumeReadContractWire(t, r, glLimitsPath, token)
					if r.URL.Query().Get("project_id") != "requested" {
						t.Error(r.URL)
					}
				}
				switch n - first {
				case 0:
					testcloud.JSON(w, 401, `{"error":"expired"}`)
				case 1:
					testcloud.JSON(w, 503, `{"error":"retry"}`)
				case 2:
					if r.Header.Get("X-Native") != "retry" {
						t.Error(r.Header)
					}
					w.Header().Set("X-Proof", "accepted once")
					if stage == "member" {
						testcloud.JSON(w, 203, `{"project":{"id":"requested"}}`)
					} else {
						testcloud.JSON(w, 203, `{"limits":{}}`)
					}
				default:
					t.Error("workflow resent/restarted", r.URL)
					testcloud.JSON(w, 500, `{}`)
				}
			})
			result, err := glCall(context.Background(), cinder, identity, "ref")
			if err != nil || result == nil || result.Project == nil || result.Project.Project == nil || result.Limits == nil || result.Observed == nil || calls.Load() != 4 || reauths.Load() != 1 || retries.Load() != 1 {
				t.Fatal(result, err, calls.Load(), reauths.Load(), retries.Load())
			}
			proof := result.Observed
			if stage == "member" {
				proof = result.Project.Observed
			}
			if proof == nil || proof.StatusCode != 203 || proof.Header.Get("X-Proof") != "accepted once" {
				t.Fatal(result)
			}
		})
	}
}

func TestGetVolumeLimitsNativeRetryProtectsBodyDecoderRetentionAndOriginalStatusPolicy(t *testing.T) {
	for _, stage := range []string{"member", "limits"} {
		for _, kind := range []string{"body", "raw body", "decoder", "discard response", "source", "status expansion"} {
			t.Run(stage+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
				var calls, retries atomic.Int32
				cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, err error, _ uint) error {
					if !gophercloud.ResponseCodeIs(err, 503) {
						return err
					}
					if retries.Add(1) > 1 {
						return errors.New("bounded fixture exhausted")
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
					case "source":
						glMutateSource(cinder, "resource base")
					case "status expansion":
						opts.OkCodes = append(opts.OkCodes, 400)
					}
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					n := calls.Add(1)
					first := int32(1)
					if stage == "limits" {
						first = 2
						if n == 1 {
							glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
							testcloud.JSON(w, 200, `{"project":{"id":"requested"}}`)
							return
						}
					}
					if stage == "member" {
						glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
					} else {
						volumeReadContractWire(t, r, glLimitsPath, "test-token")
					}
					w.Header().Set("X-Proof", "current native")
					if n == first {
						testcloud.JSON(w, 503, `{"error":"retry"}`)
					} else if n == first+1 {
						testcloud.JSON(w, 400, `{"error":"expanded native status"}`)
					} else {
						t.Error("unbounded fixture", r.URL)
						testcloud.JSON(w, 500, `{}`)
					}
				})
				result, err := glCall(context.Background(), cinder, identity, "ref")
				var physical *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				want := int32(1)
				if stage == "limits" {
					want++
				}
				status := 503
				if kind == "status expansion" {
					want++
					status = 400
				}
				if result == nil || result.Project == nil || result.Limits != nil || result.Value != nil || result.Observed != nil || calls.Load() != want || retries.Load() != 1 || errors.As(err, &physical) || !errors.As(err, &native) || native.Actual != status || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || native.ResponseHeader.Get("X-Proof") != "current native" {
					t.Fatal(result, err, calls.Load(), retries.Load())
				}
				if stage == "limits" {
					if result.Project.Project == nil || result.Project.Observed == nil {
						t.Fatal(result)
					}
				} else if result.Project.Project != nil || result.Project.Observed != nil || len(result.Project.Pages) != 0 {
					t.Fatal(result)
				}
				if kind != "status expansion" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestGetVolumeLimitsBlockedHTTPStageCancellationRetainsOnlyCompletedProject(t *testing.T) {
	for _, stage := range []string{"member", "limits"} {
		t.Run(stage, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller canceled blocked GET")
			started := make(chan struct{})
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 && stage == "limits" {
					glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
					w.Header().Set("X-Proof", "project")
					testcloud.JSON(w, 200, `{"project":{"id":"requested"}}`)
					return
				}
				if stage == "member" {
					glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
				} else {
					volumeReadContractWire(t, r, glLimitsPath, "test-token")
				}
				close(started)
				<-r.Context().Done()
			})
			type outcome struct {
				result *blockstorage.GetVolumeLimitsResult
				err    error
			}
			done := make(chan outcome, 1)
			go func() { result, err := glCall(ctx, cinder, identity, "ref"); done <- outcome{result, err} }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("stage never started")
			}
			cancel(cause)
			select {
			case got := <-done:
				var physical *resource.ResponseError
				want := int32(1)
				if stage == "limits" {
					want = 2
				}
				if got.result == nil || got.result.Project == nil || got.result.Value != nil || got.result.Limits != nil || got.result.Observed != nil || calls.Load() != want || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || errors.As(got.err, &physical) {
					t.Fatal(got, calls.Load())
				}
				if stage == "limits" {
					if got.result.Project.Project == nil || got.result.Project.Observed == nil || got.result.Project.Observed.Header.Get("X-Proof") != "project" {
						t.Fatal(got)
					}
				} else if got.result.Project.Project != nil || got.result.Project.Observed != nil {
					t.Fatal(got)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("stage ignored cancellation")
			}
		})
	}
}

func TestGetVolumeLimitsRedirectCannotChangeFixedStageTargetOrForwardCredentials(t *testing.T) {
	for _, stage := range []string{"member", "limits"} {
		t.Run(stage, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if n == 1 && stage == "limits" {
					testcloud.JSON(w, 200, `{"project":{"id":"requested"}}`)
					return
				}
				want := int32(1)
				if stage == "limits" {
					want = 2
				}
				if n != want {
					t.Error("changed target received credentials", r.URL, r.Header)
					w.WriteHeader(500)
					return
				}
				w.Header().Set("Location", cloud.Server.URL+"/changed/target")
				w.WriteHeader(302)
			})
			result, err := glCall(context.Background(), cinder, identity, "ref")
			var physical *resource.ResponseError
			want := int32(1)
			if stage == "limits" {
				want = 2
			}
			if result == nil || result.Project == nil || result.Limits != nil || result.Value != nil || result.Observed != nil || calls.Load() != want || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) {
				t.Fatal(result, err, calls.Load())
			}
			if stage == "limits" && result.Project.Project == nil {
				t.Fatal(result)
			}
		})
	}
}

func TestGetVolumeLimitsProjectPagingRejectsUnsafeContinuationWithoutBorrowedProof(t *testing.T) {
	for _, next := range []string{"https://foreign.invalid/projects", "/changed/projects", "?name=name%2Fonly", "?marker=next"} {
		t.Run(next, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if n > 2 {
					t.Error("cycle repeated physical query", r.URL)
					w.WriteHeader(500)
					return
				}
				glIdentityWire(t, r, glProjectsPath, "test-token")
				href := next
				if next == "?marker=next" && n == 2 {
					href = "?marker=next"
				}
				raw, _ := json.Marshal(href)
				w.Header().Set("X-Proof", "page")
				testcloud.JSON(w, 203, `{"projects":[{"name":"other"}],"links":{"next":`+string(raw)+`}}`)
			})
			result, err := glCall(context.Background(), cinder, identity, "name/only")
			var physical *resource.ResponseError
			want := int32(1)
			cause := resource.ErrInvalidOption
			if next == "?marker=next" {
				want = 2
				cause = resource.ErrPaginationCycle
			} else if next == "?name=name%2Fonly" {
				cause = resource.ErrPaginationCycle
			}
			if result == nil || result.Project == nil || result.Project.Project != nil || len(result.Project.Pages) != int(want) || result.Project.Observed != nil || result.RequestedProjectID != nil || result.Observed != nil || result.Value != nil || calls.Load() != want || !errors.Is(err, cause) || !errors.As(err, &physical) || physical.StatusCode != 203 || physical.Header.Get("X-Proof") != "page" {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}
