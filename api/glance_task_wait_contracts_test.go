package api_test

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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/image/v2/tasks"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

// Pinned image/v2/_proxy.py wait_for_task checks fetched selected failures for
// this exact message, recreates only type/input, then pauses before a fresh GET.
const glanceTask396 = "Image cannot be imported. Error code: '396'"
const glanceTaskPrefix = "/reverse/glance/v2/"

type glanceTaskRoundTrip func(*http.Request) (*http.Response, error)

func (f glanceTaskRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func glanceTaskJSON(id, status, message, input string) string {
	return fmt.Sprintf(`{"id":%q,"type":"import","status":%q,"message":%q,"input":%s,"self":"https://foreign.invalid/tasks/decoy","schema":"https://foreign.invalid/schema","result":{"precise":9007199254740993}}`, id, status, message, input)
}

func glanceTaskFields(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err, string(body))
	}
	return fields
}

func glanceTaskNumber(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func glanceTaskHTTP(status int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Evidence": {"actual"}}, Body: body}
}

func glanceTaskFast() tasks.TaskWaitOption { return tasks.WithTaskWaitPollInterval(time.Millisecond) }

func TestGlanceTaskWaitFreshReadsAndDefaults(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("image", glanceTaskPrefix)
	client.Endpoint = cloud.Server.URL + "/catalog/unused/"
	client.ResourceBase = cloud.Server.URL + glanceTaskPrefix
	var gets, posts atomic.Int32
	var deadlines []time.Time
	var deadlineSet []bool
	transport := cloud.Provider.HTTPClient.Transport
	cloud.Provider.HTTPClient.Transport = glanceTaskRoundTrip(func(r *http.Request) (*http.Response, error) {
		d, set := r.Context().Deadline()
		deadlines = append(deadlines, d)
		deadlineSet = append(deadlineSet, set)
		return transport.RoundTrip(r)
	})
	cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/original", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.RawQuery != "" {
			t.Error("unexpected task request", r.Method, r.URL.String())
		}
		n := gets.Add(1)
		status := "success"
		if n <= 2 {
			status = []string{"pending", "processing"}[n-1]
		}
		w.Header().Set("X-Read", fmt.Sprint(n))
		testcloud.JSON(w, 200, glanceTaskJSON("response-id", status, "", `{"nested":{"exact":9007199254740993},"null":null}`))
	})
	cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); w.WriteHeader(500) })
	a := tasks.New(client)
	v, err := a.WaitForTask(context.Background(), resource.ID("original"), glanceTaskFast())
	if err != nil || v == nil || v.Task == nil || v.Task.Status != "success" || v.Task.ID != "response-id" || v.CurrentID != "original" || v.OriginalID != "original" || v.Created != nil || v.Recreated != 0 || gets.Load() != 3 || posts.Load() != 0 {
		t.Fatal(v, err, gets.Load(), posts.Load())
	}
	if v.StatusCode != 200 || v.Header.Get("X-Read") != "3" || !bytes.Contains(v.Body, []byte("9007199254740993")) {
		t.Fatal("last actual GET evidence was lost", v)
	}
	if !deadlineSet[0] || time.Until(deadlines[0]) < 100*time.Second || time.Until(deadlines[0]) > 121*time.Second || !deadlines[0].Equal(deadlines[1]) || !deadlines[1].Equal(deadlines[2]) {
		t.Fatal("default wait budget was reset", deadlines, deadlineSet)
	}
	// Options replace defaults; caller deadlines still bound an unlimited wait.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := a.WaitForTaskState(ctx, resource.ID("original"), "SUCCESS", tasks.WithTaskWaitTimeout(time.Second), tasks.WithUnlimitedTaskWait()); err != nil {
		t.Fatal(err)
	}
	parentDeadline, _ := ctx.Deadline()
	if !deadlines[3].Equal(parentDeadline) {
		t.Fatal("unlimited option replaced caller deadline", deadlines[3], parentDeadline)
	}
	if _, err := a.WaitForTask(context.Background(), resource.ID("original"), tasks.WithTaskWaitOpts(tasks.TaskWaitOpts{Timeout: new(time.Duration)})); err != nil || deadlineSet[4] {
		t.Fatal("zero explicit budget was not unlimited", err, deadlineSet)
	}
}

func TestGlanceTaskWaitSelected396AndExactFetchedInput(t *testing.T) {
	for _, tc := range []struct {
		name, status, message, target string
		failures                      []string
		wantPost, wantFailure         bool
	}{
		{"default recreation", "failure", glanceTask396, "success", nil, true, false},
		{"custom selected failure", "BROKEN", glanceTask396, "success", []string{"broken"}, true, false},
		{"target precedes failure", "failure", glanceTask396, "FAILURE", nil, false, false},
		{"empty failures disable", "failure", glanceTask396, "success", []string{}, false, false},
		{"unselected state", "pending", glanceTask396, "success", nil, false, false},
		{"wrong message punctuation", "failure", strings.TrimSuffix(glanceTask396, "'"), "success", nil, false, true},
		{"message is case sensitive", "failure", strings.ToLower(glanceTask396), "success", nil, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var oldGets, newGets, posts atomic.Int32
			input := `{"digits":9007199254740993,"exponent":1e+123,"nested":[null,true,{"vendor":["x",{}]}]}`
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
				status, message := tc.status, tc.message
				if oldGets.Add(1) > 1 {
					status, message = "success", ""
				}
				testcloud.JSON(w, 200, glanceTaskJSON("incidental", status, message, input))
			})
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				fields := glanceTaskFields(t, body)
				if len(fields) != 2 || string(fields["type"]) != `"import"` || !reflect.DeepEqual(glanceTaskNumber(t, fields["input"]), glanceTaskNumber(t, json.RawMessage(input))) {
					t.Error("recreation copied fields or rounded fetched input", string(body))
				}
				w.Header().Set("X-Created", "real")
				testcloud.JSON(w, 201, glanceTaskJSON("new", "success", "", "null"))
			})
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/new", func(w http.ResponseWriter, r *http.Request) {
				newGets.Add(1)
				testcloud.JSON(w, 200, glanceTaskJSON("new", "SUCCESS", "", `{}`))
			})
			opts := []tasks.TaskWaitOption{glanceTaskFast()}
			if tc.failures != nil {
				opts = append(opts, tasks.WithTaskWaitFailureStates(tc.failures...))
			}
			v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTaskState(context.Background(), resource.ID("old"), tc.target, opts...)
			if tc.wantFailure {
				var failure *resource.FailedStateError
				if !errors.As(err, &failure) || v == nil || v.Task.Status != tc.status || failure.ID != "old" || posts.Load() != 0 || oldGets.Load() != 1 {
					t.Fatal(v, err, oldGets.Load(), posts.Load())
				}
				return
			}
			if err != nil || v == nil {
				t.Fatal(v, err)
			}
			if tc.wantPost {
				if posts.Load() != 1 || newGets.Load() != 1 || oldGets.Load() != 1 || v.CurrentID != "new" || v.OriginalID != "old" || v.Recreated != 1 || v.StatusCode != 200 || v.Created == nil || v.Created.StatusCode != 201 || v.Created.Task.Status != "success" || v.Created.Header.Get("X-Created") != "real" {
					t.Fatal("POST status was used instead of a fresh GET", v, oldGets.Load(), newGets.Load(), posts.Load())
				}
			} else if posts.Load() != 0 || newGets.Load() != 0 {
				t.Fatal("unselected failure recreated", v, posts.Load(), newGets.Load())
			}
		})
	}
}

func TestGlanceTaskWaitRepeated396KeepsBudgetAndPausesAfterCreate(t *testing.T) {
	t.Run("repeated recreations share one budget", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, posts atomic.Int32
		var deadlines []time.Time
		transport := cloud.Provider.HTTPClient.Transport
		cloud.Provider.HTTPClient.Transport = glanceTaskRoundTrip(func(r *http.Request) (*http.Response, error) {
			d, ok := r.Context().Deadline()
			if !ok {
				t.Error("bounded budget absent", r.Method, r.URL.Path)
			}
			deadlines = append(deadlines, d)
			return transport.RoundTrip(r)
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			id := strings.TrimPrefix(r.URL.Path, glanceTaskPrefix+"tasks/")
			status, message := "failure", glanceTask396
			if id == "new-2" {
				status, message = "success", ""
			}
			testcloud.JSON(w, 200, glanceTaskJSON(id, status, message, `{"exact":9007199254740993}`))
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) {
			id := fmt.Sprintf("new-%d", posts.Add(1))
			testcloud.JSON(w, 201, glanceTaskJSON(id, "success", "", `{}`))
		})
		v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(context.Background(), resource.ID("old"), tasks.WithTaskWaitTimeout(10*time.Second), glanceTaskFast())
		if err != nil || v == nil || v.CurrentID != "new-2" || v.OriginalID != "old" || v.Recreated != 2 || v.Created == nil || v.Created.Task.ID != "new-2" || calls.Load() != 3 || posts.Load() != 2 || len(deadlines) != 5 {
			t.Fatal(v, err, calls.Load(), posts.Load(), deadlines)
		}
		for _, d := range deadlines {
			if !d.Equal(deadlines[0]) {
				t.Fatal("ID transition reset budget", deadlines)
			}
		}
	})
	t.Run("successful POST is not immediate completion", func(t *testing.T) {
		cloud := testcloud.New(t)
		var gets, posts atomic.Int32
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			testcloud.JSON(w, 200, glanceTaskJSON("old", "failure", glanceTask396, "null"))
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) {
			posts.Add(1)
			testcloud.JSON(w, 201, glanceTaskJSON("new", "success", "", "null"))
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/new", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			testcloud.JSON(w, 200, glanceTaskJSON("new", "success", "", "null"))
		})
		v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(context.Background(), resource.ID("old"), tasks.WithTaskWaitPollInterval(time.Hour), tasks.WithTaskWaitTimeout(2*time.Second))
		if !errors.Is(err, context.DeadlineExceeded) || v == nil || v.CurrentID != "new" || v.Task.Status != "success" || v.StatusCode != 201 || v.Created == nil || v.Recreated != 1 || gets.Load() != 1 || posts.Load() != 1 {
			t.Fatal("POST shortcut or replay", v, err, gets.Load(), posts.Load())
		}
	})
}

func TestGlanceTaskWaitFetchedSchemaAndInputPresence(t *testing.T) {
	for _, body := range []string{
		`{`, `null`, `[]`, `{"id":"old","type":"import","status":null}`, `{"id":"old","type":"import"}`, `{"id":"old","status":7}`,
		`{"id":"old","status":"success","input":[]}`, `{"id":"old","status":"success","result":"bad"}`, `{"id":"old","status":"success","created_at":"not-a-date"}`,
	} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Accepted", "original")
				testcloud.JSON(w, 200, body)
			})
			v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(context.Background(), resource.ID("old"))
			var response *resource.ResponseError
			if v != nil || !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != body || response.Header.Get("X-Accepted") != "original" || calls.Load() != 1 {
				t.Fatal(v, err, response, calls.Load())
			}
		})
	}
	for _, tc := range []struct{ name, field, want string }{
		{"null", `,"input":null`, "null"}, {"empty", `,"input":{}`, "{}"},
	} {
		t.Run("recreation input "+tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var posts atomic.Int32
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, fmt.Sprintf(`{"id":"old","type":"import","status":"failure","message":%q%s}`, glanceTask396, tc.field))
			})
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				body, _ := io.ReadAll(r.Body)
				if got := string(glanceTaskFields(t, body)["input"]); got != tc.want {
					t.Error("input presence changed", got, tc.want)
				}
				testcloud.JSON(w, 201, glanceTaskJSON("new", "pending", "", "null"))
			})
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/new", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, glanceTaskJSON("new", "success", "", "null"))
			})
			if v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(context.Background(), resource.ID("old"), glanceTaskFast()); err != nil || v == nil || posts.Load() != 1 {
				t.Fatal(v, err, posts.Load())
			}
		})
	}
	t.Run("missing selected input cannot fabricate null", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			testcloud.JSON(w, 200, fmt.Sprintf(`{"id":"old","type":"import","status":"failure","message":%q}`, glanceTask396))
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(context.Background(), resource.ID("old"))
		var response *resource.ResponseError
		if v == nil || v.Task == nil || v.Task.Status != "failure" || !errors.As(err, &response) || response.StatusCode != 200 || calls.Load() != 1 {
			t.Fatal(v, err, calls.Load())
		}
	})
	for _, field := range []string{``, `,"type":""`, `,"type":null`} {
		t.Run("invalid selected type "+field, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{"id":"old","status":"failure","message":%q,"input":null%s}`, glanceTask396, field))
			})
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(context.Background(), resource.ID("old"))
			if err == nil || v == nil || v.Task == nil || v.Task.Status != "failure" || calls.Load() != 1 {
				t.Fatal("invalid type reached POST or lost observed task", v, err, calls.Load())
			}
		})
	}
}

func TestGlanceTaskWaitRecreationErrorsKeepActualEvidence(t *testing.T) {
	for _, tc := range []struct {
		name     string
		code     int
		body     string
		accepted bool
	}{
		{"native forbidden", 403, `{"denied":true}`, false}, {"unexpected create200", 200, `{"id":"new","status":"success"}`, false},
		{"malformed", 201, `{`, true}, {"null", 201, `null`, true}, {"missing ID", 201, `{"status":"success"}`, true},
		{"null ID", 201, `{"id":null,"status":"success"}`, true}, {"unsafe ID", 201, `{"id":"new/unsafe","status":"success"}`, true}, {"case-only ID", 201, `{"ID":"new","status":"success"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, posts atomic.Int32
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, glanceTaskJSON("old", "failure", glanceTask396, `{}`))
			})
			cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				w.Header().Set("X-POST", "accepted-or-native")
				testcloud.JSON(w, tc.code, tc.body)
			})
			v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(context.Background(), resource.ID("old"), glanceTaskFast())
			if err == nil || v == nil || v.CurrentID != "old" || v.Task.ID != "old" || v.StatusCode != 200 || v.Task.Status != "failure" || gets.Load() != 1 || posts.Load() != 1 {
				t.Fatal("failed create replaced fetched evidence or replayed", v, err, gets.Load(), posts.Load())
			}
			if tc.accepted {
				var response *resource.ResponseError
				if !errors.As(err, &response) || response.StatusCode != 201 || string(response.Body) != tc.body || response.Header.Get("X-POST") != "accepted-or-native" || v.Created == nil || v.Created.StatusCode != 201 || string(v.Created.Body) != tc.body {
					t.Fatal("accepted create proof missing", v, err, response)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != tc.code || v.Created != nil {
					t.Fatal("native create failure reclassified", v, err)
				}
			}
		})
	}
	t.Run("failed followup retains valid POST result", func(t *testing.T) {
		cloud := testcloud.New(t)
		var gets, posts atomic.Int32
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			testcloud.JSON(w, 200, glanceTaskJSON("old", "failure", glanceTask396, `{}`))
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) {
			posts.Add(1)
			testcloud.JSON(w, 201, glanceTaskJSON("new", "pending", "", `{"create":true}`))
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/new", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 404, `{"missing":true}`) })
		v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(context.Background(), resource.ID("old"), glanceTaskFast())
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 404 || v == nil || v.CurrentID != "new" || v.Task.ID != "new" || v.Task.Status != "pending" || v.StatusCode != 201 || v.Created == nil || gets.Load() != 2 || posts.Load() != 1 {
			t.Fatal(v, err, gets.Load(), posts.Load())
		}
		v.Task.Input["create"] = false
		v.Header.Set("Content-Type", "changed")
		v.Body[0] = '!'
		if v.Created.Task.Input["create"] != true || v.Created.Header.Get("Content-Type") != "application/json" || v.Created.Body[0] != '{' {
			t.Fatal("outer model mutation rewrote actual POST proof", v.Created)
		}
	})
}

type glanceTaskBrokenBody struct {
	first []byte
	cause error
}

func (r *glanceTaskBrokenBody) Read(p []byte) (int, error) {
	if len(r.first) != 0 {
		n := copy(p, r.first)
		r.first = r.first[n:]
		return n, nil
	}
	return 0, r.cause
}
func (*glanceTaskBrokenBody) Close() error { return nil }

func TestGlanceTaskWaitTerminalReadsAndCancellationNeverReplay(t *testing.T) {
	for _, phase := range []string{"GET", "POST"} {
		for _, outcome := range []string{"read", "transport404", "canceled404", "canceled200"} {
			t.Run(phase+"/"+outcome, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var attempted, gets, posts, retries atomic.Int32
				cause := errors.New("accepted body read stopped")
				transportCause := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
				cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, 200, glanceTaskJSON("old", "failure", glanceTask396, `{}`))
				})
				transport := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = glanceTaskRoundTrip(func(r *http.Request) (*http.Response, error) {
					if r.Method != phase {
						return transport.RoundTrip(r)
					}
					attempted.Add(1)
					if phase == "POST" {
						posts.Add(1)
					}
					code := 200
					if phase == "POST" {
						code = 201
					}
					switch outcome {
					case "read":
						return glanceTaskHTTP(code, &glanceTaskBrokenBody{first: []byte(`{"accepted":`), cause: cause}), nil
					case "transport404":
						return nil, transportCause
					case "canceled404":
						cancel()
						return glanceTaskHTTP(404, io.NopCloser(strings.NewReader(`{"missing":true}`))), nil
					default:
						cancel()
						return glanceTaskHTTP(code, io.NopCloser(strings.NewReader(glanceTaskJSON("new", "success", "", `{}`)))), nil
					}
				})
				// Native status failures retain configured retry policy; accepted
				// read failures happen after it and must not trigger a resend.
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries.Add(1)
					return err
				}
				v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(ctx, resource.ID("old"), glanceTaskFast())
				if err == nil || attempted.Load() != 1 || (phase == "GET" && v != nil) || (phase == "POST" && (v == nil || v.Task.ID != "old" || v.CurrentID != "old" || gets.Load() != 1 || posts.Load() != 1)) {
					t.Fatal(v, err, attempted.Load(), gets.Load(), posts.Load())
				}
				if outcome == "read" {
					var response *resource.ResponseError
					wantCode := 200
					if phase == "POST" {
						wantCode = 201
					}
					if !errors.Is(err, cause) || !errors.As(err, &response) || response.StatusCode != wantCode || string(response.Body) != `{"accepted":` || response.Header.Get("X-Evidence") != "actual" || retries.Load() != 0 {
						t.Fatal("accepted failure replayed or proof changed", v, err, response, retries.Load())
					}
					if phase == "POST" && (v.Created == nil || v.Created.Task != nil || string(v.Created.Body) != `{"accepted":`) {
						t.Fatal("partial POST evidence invented a task", v)
					}
				} else if outcome == "transport404" {
					if !errors.Is(err, transportCause) || errors.Is(err, resource.ErrNotFound) {
						t.Fatal("transport error became missing", err)
					}
				} else if !errors.Is(err, context.Canceled) {
					t.Fatal("caller cancellation hidden by response status", err)
				}
			})
		}
	}
}

func TestGlanceTaskWaitPreflightAndClosedOptions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = glanceTaskRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return glanceTaskHTTP(200, io.NopCloser(strings.NewReader(glanceTaskJSON("old", "success", "", `{}`)))), nil
	})
	client := cloud.Client("image", glanceTaskPrefix)
	a := tasks.New(client)
	for _, ref := range []resource.Ref{resource.Ref{}, resource.ID("bad/id"), resource.ID("%2F"), resource.ID(".."), resource.Name("named")} {
		_, err := a.WaitForTask(context.Background(), ref)
		if err == nil || calls.Load() != 0 {
			t.Fatal("unsafe or unsupported ref reached HTTP", ref, err, calls.Load())
		}
	}
	for _, tc := range []struct {
		name   string
		option tasks.TaskWaitOption
	}{
		{"nil option", nil}, {"zero timeout helper", tasks.WithTaskWaitTimeout(0)}, {"negative timeout", tasks.WithTaskWaitTimeout(-time.Second)},
		{"zero poll", tasks.WithTaskWaitPollInterval(0)}, {"blank failure", tasks.WithTaskWaitFailureStates(" ")},
		{"auth", tasks.WithTaskWaitHeader("X-Auth-Token", "forged")}, {"content type", tasks.WithTaskWaitHeader("Content-Type", "text/plain")},
		{"conflicting case aliases", tasks.WithTaskWaitHeaders(map[string]string{"X-Trace": "first", "x-trace": "second"})},
	} {
		if _, err := a.WaitForTask(context.Background(), resource.ID("old"), tc.option); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(tc.name, err, calls.Load())
		}
	}
	if _, err := a.WaitForTaskState(context.Background(), resource.ID("old"), " "); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := a.WaitForTask(nil, resource.ID("old")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.WaitForTask(ctx, resource.ID("old")); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal(err, calls.Load())
	}
	var nilAPI *tasks.API
	for _, api := range []*tasks.API{nilAPI, tasks.New(nil), tasks.New(&gophercloud.ServiceClient{Type: "image"})} {
		if _, err := api.WaitForTask(context.Background(), resource.ID("old")); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(err, calls.Load())
		}
	}
	for index, mutation := range []func(){func() { client.Type = "compute" }, func() { client.Type = "image"; client.ResourceBase = "https://foreign.invalid/v2/" }, func() {
		client.ResourceBase = ""
		client.MoreHeaders = map[string]string{"X-Trace": "first", "x-trace": "second"}
	}} {
		mutation()
		want := resource.ErrInvalidOption
		if index == 0 {
			want = resource.ErrUnsupported
		}
		if _, err := a.WaitForTask(context.Background(), resource.ID("old")); !errors.Is(err, want) || calls.Load() != 0 {
			t.Fatal("invalid source reached HTTP", err, calls.Load())
		}
	}
}

func TestGlanceTaskWaitOptionSnapshotsAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("X-Trace") != "owned" {
			t.Error("header alias/input mutation reached request", r.Header)
		}
		w.Header().Set("X-Owned", "actual")
		testcloud.JSON(w, 200, glanceTaskJSON("old", "failure", glanceTask396, `{"nested":{"value":1}}`))
	})
	timeout, interval := 10*time.Second, time.Millisecond
	failures := []string{}
	headers := map[string]string{"X-Trace": "owned"}
	option := tasks.WithTaskWaitOpts(tasks.TaskWaitOpts{Timeout: &timeout, PollInterval: &interval, FailureStates: failures, Headers: headers})
	timeout, interval = -time.Second, 0
	headers["X-Trace"] = "caller changed"
	// Explicit empty failure states allow the exact 396 response to be the
	// requested target without recreation, and the reusable option owns inputs.
	a := tasks.New(cloud.Client("image", glanceTaskPrefix))
	var wg sync.WaitGroup
	errorsOut := make(chan error, 6)
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := a.WaitForTaskState(context.Background(), resource.ID("old"), "failure", option)
			if err != nil || v == nil || v.Task == nil || v.Created != nil {
				errorsOut <- fmt.Errorf("snapshot result=%v error=%v", v, err)
				return
			}
			v.Task.Input["nested"].(map[string]any)["value"] = "changed"
			v.Header.Set("X-Owned", "changed")
			v.Body[0] = '!'
		}()
	}
	wg.Wait()
	close(errorsOut)
	for err := range errorsOut {
		t.Error(err)
	}
	v, err := a.WaitForTaskState(context.Background(), resource.ID("old"), "failure", option)
	if err != nil || v == nil || v.Header.Get("X-Owned") != "actual" || v.Body[0] != '{' || v.Task.Input["nested"].(map[string]any)["value"] != json.Number("1") || gets.Load() != 7 {
		t.Fatal("result ownership or option reuse failed", v, err, gets.Load())
	}
	// A retained custom option object is no longer the parsed request policy.
	var retained *tasks.TaskWaitOpts
	custom := tasks.TaskWaitOption(func(o *tasks.TaskWaitOpts) error {
		o.Headers = map[string]string{"X-Trace": "owned"}
		retained = o
		return nil
	})
	transport := cloud.Provider.HTTPClient.Transport
	cloud.Provider.HTTPClient.Transport = glanceTaskRoundTrip(func(r *http.Request) (*http.Response, error) {
		retained.Headers["X-Trace"] = "late mutation"
		return transport.RoundTrip(r)
	})
	if v, err := a.WaitForTaskState(context.Background(), resource.ID("old"), "failure", custom); err != nil || v == nil {
		t.Fatal(v, err)
	}
}

func TestGlanceTaskWaitLiveSourceFixedRoutesAndReauthentication(t *testing.T) {
	t.Run("provider middleware headers and configured retry remain live", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/catalog/unused/")
		client.ResourceBase = cloud.Server.URL + glanceTaskPrefix
		client.MoreHeaders = map[string]string{"X-Source": "configured"}
		client.Microversion = "2.17"
		cloud.Provider.SetToken("initial")
		var gets, posts, middleware, reauth, retries atomic.Int32
		type key struct{}
		transport := cloud.Provider.HTTPClient.Transport
		cloud.Provider.HTTPClient.Transport = glanceTaskRoundTrip(func(r *http.Request) (*http.Response, error) {
			middleware.Add(1)
			if r.Context().Value(key{}) != "caller" || r.Header.Get("X-Source") != "configured" || r.Header.Get("X-Trace") != "final" || r.Header.Get("OpenStack-API-Version") != "image 2.17" {
				t.Error("shared client policy lost", r.Header)
			}
			return transport.RoundTrip(r)
		})
		cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("reauthenticated"); return nil }
		cloud.Provider.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			retries.Add(1)
			if method != http.MethodPost || target != cloud.Server.URL+glanceTaskPrefix+"tasks" || !gophercloud.ResponseCodeIs(err, 503) {
				return err
			}
			return nil
		}
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			if r.Header.Get("X-Auth-Token") != "initial" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("between-phases")
			testcloud.JSON(w, 200, glanceTaskJSON("response-decoy", "failure", glanceTask396, `{"raw":9007199254740993}`))
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) {
			n := posts.Add(1)
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"input":{"raw":9007199254740993},"type":"import"}` {
				t.Error("request replay changed input", string(body))
			}
			if n == 1 {
				if r.Header.Get("X-Auth-Token") != "between-phases" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 401, `{}`)
				return
			}
			if r.Header.Get("X-Auth-Token") != "reauthenticated" {
				t.Error(r.Header)
			}
			if n == 2 {
				testcloud.JSON(w, 503, `{}`)
				return
			}
			testcloud.JSON(w, 201, glanceTaskJSON("new", "pending", "", `{}`))
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/new", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			if r.Header.Get("X-Auth-Token") != "reauthenticated" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, glanceTaskJSON("new", "success", "", `{}`))
		})
		v, err := tasks.New(client).WaitForTask(context.WithValue(context.Background(), key{}, "caller"), resource.ID("old"), glanceTaskFast(), tasks.WithTaskWaitHeader("X-Trace", "earlier"), tasks.WithTaskWaitHeader("x-trace", "final"))
		if err != nil || v == nil || v.CurrentID != "new" || posts.Load() != 3 || gets.Load() != 2 || middleware.Load() != 5 || reauth.Load() != 1 || retries.Load() != 1 || client.ResourceBase != cloud.Server.URL+glanceTaskPrefix || client.MoreHeaders["X-Source"] != "configured" {
			t.Fatal(v, err, posts.Load(), gets.Load(), middleware.Load(), reauth.Load(), retries.Load())
		}
	})
	t.Run("source changes are checked after options and fetch", func(t *testing.T) {
		for _, when := range []string{"option", "after GET"} {
			t.Run(when, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("image", glanceTaskPrefix)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 200, glanceTaskJSON("old", "failure", glanceTask396, `{}`))
				})
				opts := []tasks.TaskWaitOption{}
				if when == "option" {
					opts = append(opts, func(*tasks.TaskWaitOpts) error { client.Type = "compute"; return nil })
				} else {
					transport := cloud.Provider.HTTPClient.Transport
					cloud.Provider.HTTPClient.Transport = glanceTaskRoundTrip(func(r *http.Request) (*http.Response, error) {
						response, err := transport.RoundTrip(r)
						client.Type = "compute"
						return response, err
					})
				}
				v, err := tasks.New(client).WaitForTask(context.Background(), resource.ID("old"), opts...)
				if !errors.Is(err, resource.ErrUnsupported) || (when == "option" && (v != nil || calls.Load() != 0)) || (when == "after GET" && (v == nil || v.Task.Status != "failure" || calls.Load() != 1)) {
					t.Fatal(v, err, calls.Load())
				}
			})
		}
	})
	t.Run("redirect cannot change task identity", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, wrong atomic.Int32
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			http.Redirect(w, r, glanceTaskPrefix+"tasks/other", 307)
		})
		cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/other", func(w http.ResponseWriter, r *http.Request) {
			wrong.Add(1)
			testcloud.JSON(w, 200, glanceTaskJSON("other", "success", "", `{}`))
		})
		v, err := tasks.New(cloud.Client("image", glanceTaskPrefix)).WaitForTask(context.Background(), resource.ID("old"))
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 || wrong.Load() != 0 {
			t.Fatal(v, err, calls.Load(), wrong.Load())
		}
	})
}

func TestGlanceTaskWaitNativeSurfacesAndCanonicalEvidenceStaySeparate(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("image", glanceTaskPrefix)
	var gets, posts, deletes atomic.Int32
	cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/old", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Method != http.MethodGet {
			deletes.Add(1)
			w.WriteHeader(500)
			return
		}
		testcloud.JSON(w, 200, `{"id":"old","type":"import","status":"success","input":{"large":9007199254740993},"result":{"null":null},"STATUS":false,"ID":[],"INPUT":"extension"}`)
	})
	cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		body, _ := io.ReadAll(r.Body)
		fields := glanceTaskFields(t, body)
		if string(fields["input"]) != "null" || string(fields["type"]) != `"import"` {
			t.Error(string(body))
		}
		testcloud.JSON(w, 201, glanceTaskJSON("native-created", "success", "", `{"large":9007199254740993}`))
	})
	a := tasks.New(client)
	v, err := a.WaitForTask(context.Background(), resource.ID("old"))
	if err != nil || v == nil || v.Task.ID != "old" || v.Task.Status != "success" || v.Task.Input["large"] != json.Number("9007199254740993") || string(glanceTaskFields(t, v.Body)["STATUS"]) != "false" || posts.Load() != 0 || deletes.Load() != 0 {
		t.Fatal("case aliases changed canonical decisions or raw proof", v, err)
	}
	// Existing native Get preserves its own case-folded decoder. Use a separate
	// unambiguous native response to prove its signature and float64 projection.
	cloud.Mux.HandleFunc(glanceTaskPrefix+"tasks/native", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, glanceTaskJSON("native", "success", "", `{"large":9007199254740993}`))
	})
	get := a.Get
	create := a.Create
	if native, err := get(context.Background(), "native"); err != nil || native == nil || native.Input["large"] != float64(9007199254740992) {
		t.Fatal("native Get ABI/number projection changed", native, err)
	}
	if native, err := create(context.Background(), tasks.CreateOpts{Type: "import"}); err != nil || native == nil || native.ID != "native-created" || native.Input["large"] != float64(9007199254740992) {
		t.Fatal("native Create ABI/default input changed", native, err)
	}
	if _, err := a.Find(context.Background(), resource.Name("named")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal("task Name capability was invented", err)
	}
	if err := a.Remove(context.Background(), resource.ID("old")); !errors.Is(err, resource.ErrUnsupported) || deletes.Load() != 0 || posts.Load() != 1 || gets.Load() != 2 {
		t.Fatal("wait introduced cleanup/create requests", err, deletes.Load(), posts.Load(), gets.Load())
	}
}
