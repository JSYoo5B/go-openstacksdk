package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/baremetalintrospection/v1/introspection"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestIntrospectionCollectionUsesUUIDWithoutNameStatusOrDelete(t *testing.T) {
	cloud := testcloud.New(t)
	var firstPages, nextPages, gets atomic.Int32
	cloud.Mux.HandleFunc("/v1/introspection", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("limit") != "1" || r.URL.Query().Get("vendor") != "a&b" {
			t.Errorf("method=%s query=%s", r.Method, r.URL.RawQuery)
		}
		firstPages.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"introspection":       []any{map[string]any{"uuid": "node", "finished": false, "state": "processing", "error": nil}},
			"introspection_links": []any{map[string]any{"rel": "next", "href": cloud.Server.URL + "/v1/introspection/page2"}},
		})
	})
	cloud.Mux.HandleFunc("/v1/introspection/page2", func(w http.ResponseWriter, r *http.Request) {
		nextPages.Add(1)
		testcloud.JSON(w, 200, `{"introspection":[{"uuid":"other","finished":true,"state":"finished","error":null}]}`)
	})
	cloud.Mux.HandleFunc("/v1/introspection/node", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, `{"uuid":"node","finished":true,"state":"finished","error":null}`)
	})
	api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
	ctx := context.Background()
	if id, err := api.Resources.ResolveID(ctx, resource.ID("node")); err != nil || id != "node" || gets.Load() != 0 {
		t.Fatalf("id=%q gets=%d err=%v", id, gets.Load(), err)
	}
	value, err := api.Find(ctx, resource.ID("node"))
	if err != nil || value.UUID != "node" || !value.Finished || value.Error != "" || gets.Load() != 1 {
		t.Fatalf("value=%v gets=%d err=%v", value, gets.Load(), err)
	}
	options := []resource.ListOption{resource.WithPageSize(1), resource.WithQuery("vendor", "a&b")}
	values, err := api.All(ctx, options...)
	if err != nil || len(values) != 2 || values[0].UUID != "node" || values[0].Finished || values[1].UUID != "other" || !values[1].Finished {
		t.Fatalf("values=%v err=%v", values, err)
	}
	beforeNext := nextPages.Load()
	for value, err := range api.Resources.List(ctx, options...) {
		if err != nil || value.UUID != "node" {
			t.Fatalf("value=%v err=%v", value, err)
		}
		break
	}
	if nextPages.Load() != beforeNext {
		t.Fatal("list fetched after consumer break")
	}
	beforeFirst, beforeGet := firstPages.Load(), gets.Load()
	if _, err := api.Find(ctx, resource.Name("node")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("name error=%v", err)
	}
	if err := api.Remove(ctx, resource.ID("node")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("delete error=%v", err)
	}
	if _, err := api.WaitFor(ctx, resource.ID("node"), "finished"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("native status error=%v", err)
	}
	if firstPages.Load() != beforeFirst || gets.Load() != beforeGet {
		t.Fatal("unsupported resource policy made an HTTP request")
	}
}

func TestIntrospectionWaiterPollsStableInputUUIDDespiteResponseIdentity(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v1/introspection/node", func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			testcloud.JSON(w, 200, `{"uuid":"different-wire-uuid","finished":false,"state":"processing","error":null}`)
			return
		}
		testcloud.JSON(w, 200, `{"finished":true,"state":"finished","error":null,"finished_at":"2026-10-01T00:00:00"}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("response redirected the stable UUID poll to %s", r.URL.Path)
		testcloud.JSON(w, 404, "{}")
	})
	api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
	value, err := api.WaitUntilFinished(context.Background(), resource.ID("node"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
	if err != nil || value == nil || !value.Finished || value.UUID != "" || value.FinishedAt.IsZero() || calls.Load() != 2 {
		t.Fatalf("value=%v calls=%d err=%v", value, calls.Load(), err)
	}
}

func TestIntrospectionWaiterReturnsImmediatelyForFinishedOrFailedResponse(t *testing.T) {
	for _, tc := range []struct {
		name, body, message, state string
		failed                     bool
	}{
		{"finished", `{"uuid":"node","finished":true,"state":"finished","error":null}`, "", "finished", false},
		{"failed before finished", `{"uuid":"node","finished":false,"state":"processing","error":"hardware inspection failed"}`, "hardware inspection failed", "processing", true},
		{"failure wins over finished", `{"uuid":"node","finished":true,"state":"error","error":"operator canceled"}`, "operator canceled", "error", true},
		{"error state without message", `{"uuid":"node","finished":false,"state":"error","error":null}`, "introspection reported an error state", "error", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/v1/introspection/node", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, tc.body)
			})
			api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
			value, err := api.WaitUntilFinished(context.Background(), resource.ID("node"), resource.WithPollInterval(time.Hour), resource.WithTimeout(time.Second))
			if calls.Load() != 1 {
				t.Fatalf("calls=%d err=%v", calls.Load(), err)
			}
			if !tc.failed {
				if err != nil || value == nil || !value.Finished {
					t.Fatalf("value=%v err=%v", value, err)
				}
				return
			}
			var failure *introspection.IntrospectionFailureError
			if value != nil || !errors.Is(err, resource.ErrFailedState) || !errors.As(err, &failure) || failure.ID != "node" || failure.Message != tc.message || failure.State != tc.state || failure.Details == nil || failure.Details.UUID != "node" {
				t.Fatalf("value=%v failure=%+v err=%v", value, failure, err)
			}
		})
	}
}

func TestIntrospectionWaiterValidatesBeforeRequests(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{}`) })
	api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
	ctx := context.Background()
	for _, options := range [][]resource.WaitOption{{nil}, {resource.WithTimeout(0)}, {resource.WithPollInterval(0)}} {
		if _, err := api.WaitUntilFinished(ctx, resource.ID("node"), options...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid wait option error=%v", err)
		}
	}
	if _, err := api.WaitUntilFinished(ctx, resource.ID("..")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("invalid ID error=%v", err)
	}
	if _, err := api.WaitUntilFinished(ctx, resource.Name("node")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("name error=%v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.WaitUntilFinished(canceled, resource.ID("node")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error=%v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("validation sent %d HTTP requests", calls.Load())
	}
}

func TestIntrospectionWaiterPreservesCancellationAndBothDeadlines(t *testing.T) {
	for _, mode := range []string{"parent cancellation", "parent deadline", "wait timeout"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			firstRequest := make(chan struct{})
			cloud.Mux.HandleFunc("/v1/introspection/node", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					close(firstRequest)
				}
				testcloud.JSON(w, 200, `{"uuid":"node","finished":false,"state":"processing","error":null}`)
			})
			api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			options := []resource.WaitOption{resource.WithPollInterval(time.Hour), resource.WithTimeout(time.Second)}
			want := context.DeadlineExceeded
			if mode == "parent deadline" {
				var stop context.CancelFunc
				ctx, stop = context.WithTimeout(ctx, 100*time.Millisecond)
				defer stop()
			} else if mode == "wait timeout" {
				options = append(options, resource.WithTimeout(100*time.Millisecond))
			} else {
				want = context.Canceled
				go func() {
					select {
					case <-firstRequest:
						cancel()
					case <-ctx.Done():
					}
				}()
			}
			value, err := api.WaitUntilFinished(ctx, resource.ID("node"), options...)
			if value != nil || !errors.Is(err, want) || calls.Load() != 1 {
				t.Fatalf("value=%v calls=%d err=%v", value, calls.Load(), err)
			}
		})
	}
}

func TestIntrospectionWaiterPreservesHTTPAndDecodeFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
	}{
		{"missing", `{}`, 404},
		{"forbidden", `{}`, 403},
		{"invalid timestamp", `{"uuid":"node","finished":true,"started_at":"invalid"}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/v1/introspection/node", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, tc.code, tc.body) })
			api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
			value, err := api.WaitUntilFinished(context.Background(), resource.ID("node"), resource.WithPollInterval(time.Hour))
			if err == nil || value != nil || calls.Load() != 1 {
				t.Fatalf("value=%v calls=%d err=%v", value, calls.Load(), err)
			}
			if tc.code != 200 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != tc.code || errors.Is(err, resource.ErrNotFound) != (tc.code == 404) {
					t.Fatalf("HTTP error=%v", err)
				}
			} else {
				var parse *time.ParseError
				if !errors.As(err, &parse) || errors.Is(err, resource.ErrFailedState) {
					t.Fatalf("decode error=%v", err)
				}
			}
		})
	}
}
