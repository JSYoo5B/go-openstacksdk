package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/messaging/v2/queues"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func newZaqarQueues(cloud *testcloud.Cloud) *queues.API {
	return queues.New(cloud.Client("messaging", "/zaqar"))
}

func zaqarQueueBody(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
	}
	return body
}

func zaqarQueueBodyStrings(body map[string]json.RawMessage) map[string]string {
	got := map[string]string{}
	for key, value := range body {
		got[key] = string(value)
	}
	return got
}

func zaqarQueueNoRequests(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
}

func TestZaqarQueueCreateBodyOmitsNameAndAcceptsOnly201Or204(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {204, true}, {200, false}, {202, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("PUT /zaqar/v2/queues/jobs", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			got := zaqarQueueBodyStrings(zaqarQueueBody(t, r))
			want := map[string]string{
				"_default_message_ttl": `3600`, "_dead_letter_queue": `"dead"`, "_enable_encrypt_messages": `false`, "vendor": `{"enabled":false}`,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if id := r.Header.Get("Client-ID"); id != "" {
				t.Errorf("native queues Create must not add Client-ID, got %q", id)
			}
			w.WriteHeader(tc.status)
		})
		encrypt := false
		err := newZaqarQueues(cloud).Create(context.Background(), queues.CreateOpts{
			QueueName: "jobs", DefaultMessageTTL: 3600, DeadLetterQueue: "dead", EnableEncryptMessages: &encrypt,
		}, queues.WithCreateField("vendor", map[string]bool{"enabled": false}))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d error=%v calls=%d", tc.status, err, calls.Load())
		}
		if !tc.success {
			var unexpected gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &unexpected) || unexpected.Actual != tc.status {
				t.Fatalf("status %d error=%v", tc.status, err)
			}
		}
	}
}

func TestZaqarQueueRejectsCollisionsAndNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	zaqarQueueNoRequests(t, cloud)
	api := newZaqarQueues(cloud)
	ctx := context.Background()
	if err := api.Create(ctx, queues.CreateOpts{QueueName: "jobs", DefaultMessageTTL: 60}, queues.WithCreateField("_default_message_ttl", 1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected create field: %v", err)
	}
	if err := api.Create(ctx, queues.CreateOpts{QueueName: "jobs", DefaultMessageTTL: 60}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if err := api.Create(ctx, queues.CreateOpts{DefaultMessageTTL: 60}); err == nil {
		t.Fatal("missing queue name must fail before HTTP")
	}
	if err := api.Purge(ctx, "jobs", queues.PurgeOpts{ResourceTypes: []queues.PurgeResource{queues.ResourceMessages}}, queues.WithPurgeField("resource_types", []string{"x"})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected purge field: %v", err)
	}
	if err := api.Purge(ctx, "jobs", queues.PurgeOpts{}); err == nil {
		t.Fatal("empty purge resource types must fail before HTTP")
	}
	if _, err := api.Share(ctx, "jobs", queues.ShareOpts{}, queues.WithShareField("paths", []string{"x"}), queues.WithShareField("vendor", 1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected share field: %v", err)
	}
	if _, err := api.Share(ctx, "jobs", queues.ShareOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil share option: %v", err)
	}
	if _, err := api.Update(ctx, "jobs", queues.BatchUpdateOpts{{Op: queues.ReplaceOp, Path: "/metadata/x", Value: 1}}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil update option: %v", err)
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestZaqarQueueGetDecodesMetadataAndKeepsNative404(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/jobs", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"_default_message_ttl":3600,"_max_claim_count":0,"_enable_encrypt_messages":false,"_dead_letter_queue":"dead","owner":"ops"}`)
	})
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{"title":"gone"}`)
	})
	api := newZaqarQueues(cloud)
	value, err := api.Get(context.Background(), "jobs")
	if err != nil || value.DefaultMessageTTL != 3600 || value.DeadLetterQueue != "dead" || value.EnableEncryptMessages || value.MaxClaimCount != 0 || value.Extra["owner"] != "ops" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(context.Background(), "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
}

const zaqarWantPatchBody = `[{"op":"replace","path":"/metadata/_max_claim_count","value":0},{"op":"add","path":"/metadata/flag","value":false},{"op":"remove","path":"/metadata/old","value":""}]`

// zaqarPatchBody normalizes the request body to compact JSON without
// reordering keys, so the assertion covers the exact patch list.
func zaqarPatchBody(t *testing.T, r *http.Request) string {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("request body: %v", err)
	}
	var list []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Errorf("request body is not a JSON patch list: %s", raw)
		return string(raw)
	}
	out := "["
	for i, entry := range list {
		if i > 0 {
			out += ","
		}
		out += `{"op":` + string(entry["op"]) + `,"path":` + string(entry["path"]) + `,"value":` + string(entry["value"]) + `}`
		if len(entry) != 3 {
			t.Errorf("patch entry %d has keys %v", i, entry)
		}
	}
	return out + "]"
}

func TestZaqarQueueUpdateRejectsInvalidPatchBeforeRequest(t *testing.T) {
	var requests atomic.Int32
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PATCH /zaqar/v2/queues/jobs", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 200, `{}`)
	})
	api := newZaqarQueues(cloud)
	ctx := context.Background()
	// The upstream builder requires op, path and value; a nil value (for
	// example on a remove operation) is therefore rejected locally, as is an
	// empty path.
	for name, batch := range map[string]queues.BatchUpdateOpts{
		"nil value":  {{Op: queues.RemoveOp, Path: "/metadata/old", Value: nil}},
		"empty path": {{Op: queues.AddOp, Path: "", Value: 1}},
		"empty op":   {{Path: "/metadata/x", Value: 1}},
	} {
		if _, err := api.Update(ctx, "jobs", batch); err == nil || !strings.Contains(err.Error(), "Missing input") {
			t.Fatalf("%s: error=%v", name, err)
		}
	}
	// Update accepts only WithUpdateOptions; nil options fail before a request.
	if _, err := api.Update(ctx, "jobs", queues.BatchUpdateOpts{{Op: queues.AddOp, Path: "/metadata/x", Value: 1}}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option error=%v", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("requests=%d", got)
	}
}

func TestZaqarQueueUpdateOptionsReplaceBatchAndEmptyBatchSendsEmptyList(t *testing.T) {
	var bodies []string
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PATCH /zaqar/v2/queues/jobs", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(raw))
		testcloud.JSON(w, 200, `{}`)
	})
	api := newZaqarQueues(cloud)
	ctx := context.Background()
	if _, err := api.Update(ctx, "jobs", queues.BatchUpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "jobs", queues.BatchUpdateOpts{{Op: queues.AddOp, Path: "/metadata/ignored", Value: 1}},
		queues.WithUpdateOptions(queues.BatchUpdateOpts{{Op: queues.ReplaceOp, Path: "/metadata/kept", Value: "v"}})); err != nil {
		t.Fatal(err)
	}
	want := []string{`[]`, `[{"op":"replace","path":"/metadata/kept","value":"v"}]`}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%q want %q", bodies, want)
	}
}

func TestZaqarQueueUpdateUsesJSONPatchContentTypeAndAccepts200201204(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {201, true}, {204, true}, {202, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PATCH /zaqar/v2/queues/jobs", func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Content-Type"); got != "application/openstack-messaging-v2.0-json-patch" {
				t.Errorf("content type=%q", got)
			}
			if body := zaqarPatchBody(t, r); !reflect.DeepEqual(body, zaqarWantPatchBody) {
				t.Errorf("body=%s", body)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"_max_claim_count":0,"flag":false}`)
		})
		_, err := newZaqarQueues(cloud).Update(context.Background(), "jobs", queues.BatchUpdateOpts{
			{Op: queues.ReplaceOp, Path: "/metadata/_max_claim_count", Value: 0},
			{Op: queues.AddOp, Path: "/metadata/flag", Value: false},
			{Op: queues.RemoveOp, Path: "/metadata/old", Value: ""},
		})
		if tc.success != (err == nil) {
			t.Fatalf("status %d error=%v", tc.status, err)
		}
		if !tc.success {
			var unexpected gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &unexpected) || unexpected.Actual != tc.status {
				t.Fatalf("status %d error=%v", tc.status, err)
			}
		}
	}
}

func TestZaqarQueueDeleteRequires204AndKeepsMissingError(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /zaqar/v2/queues/{name}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("name") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "odd":
			w.WriteHeader(200)
		default:
			w.WriteHeader(204)
		}
	})
	api := newZaqarQueues(cloud)
	ctx := context.Background()
	if err := api.Delete(ctx, "jobs"); err != nil {
		t.Fatal(err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.Delete(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Delete(ctx, "odd"); !errors.As(err, &unexpected) || unexpected.Actual != 200 {
		t.Fatalf("delete must reject 200: %v", err)
	}
}

func TestZaqarQueueStatsDecodeMessagesObject(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/jobs/stats", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"messages":{"claimed":0,"free":3,"total":3,"oldest":{"age":1}}}`)
	})
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/bad/stats", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"messages":{}}`)
	})
	api := newZaqarQueues(cloud)
	stats, err := api.GetStats(context.Background(), "jobs")
	if err != nil || stats != (queues.Stats{Claimed: 0, Free: 3, Total: 3}) {
		t.Fatalf("stats=%+v error=%v", stats, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.GetStats(context.Background(), "bad"); !errors.As(err, &unexpected) || unexpected.Actual != 201 {
		t.Fatalf("stats must require 200: %v", err)
	}
}

func TestZaqarQueueShareAndPurgeWireShapes(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /zaqar/v2/queues/jobs/share", func(w http.ResponseWriter, r *http.Request) {
		got := zaqarQueueBodyStrings(zaqarQueueBody(t, r))
		want := map[string]string{"paths": `["messages","claims"]`, "methods": `["GET","POST"]`, "expires": `"2026-12-31T00:00:00"`, "vendor": `false`}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("share body=%v", got)
		}
		testcloud.JSON(w, 200, `{"project":"p1","paths":["/v2/queues/jobs/messages"],"expires":"2026-12-31T00:00:00","methods":["GET"],"signature":"sig"}`)
	})
	var purgeStatus atomic.Int32
	purgeStatus.Store(204)
	cloud.Mux.HandleFunc("POST /zaqar/v2/queues/jobs/purge", func(w http.ResponseWriter, r *http.Request) {
		got := zaqarQueueBodyStrings(zaqarQueueBody(t, r))
		if want := (map[string]string{"resource_types": `["messages","subscriptions"]`, "vendor": `0`}); !reflect.DeepEqual(got, want) {
			t.Errorf("purge body=%v", got)
		}
		w.WriteHeader(int(purgeStatus.Load()))
	})
	api := newZaqarQueues(cloud)
	ctx := context.Background()
	share, err := api.Share(ctx, "jobs", queues.ShareOpts{
		Paths: []queues.SharePath{queues.PathMessages, queues.PathClaims}, Methods: []queues.ShareMethod{queues.MethodGet, queues.MethodPost}, Expires: "2026-12-31T00:00:00",
	}, queues.WithShareField("vendor", false))
	if err != nil || share.Project != "p1" || share.Signature != "sig" || !reflect.DeepEqual(share.Methods, []string{"GET"}) {
		t.Fatalf("share=%+v error=%v", share, err)
	}
	purge := queues.PurgeOpts{ResourceTypes: []queues.PurgeResource{queues.ResourceMessages, queues.ResourceSubscriptions}}
	if err := api.Purge(ctx, "jobs", purge, queues.WithPurgeField("vendor", 0)); err != nil {
		t.Fatal(err)
	}
	purgeStatus.Store(200)
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.Purge(ctx, "jobs", purge, queues.WithPurgeField("vendor", 0)); !errors.As(err, &unexpected) || unexpected.Actual != 200 {
		t.Fatalf("purge must require 204: %v", err)
	}
}

func TestZaqarQueueListFiltersFollowLinksAndStatus204(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		switch r.URL.Query().Get("marker") {
		case "":
			testcloud.JSON(w, 200, `{"queues":[{"name":"a","href":"/v2/queues/a","metadata":{"_default_message_ttl":60}}],"links":[{"rel":"next","href":"/v2/queues?marker=a&limit=1"}]}`)
		case "a":
			testcloud.JSON(w, 200, `{"queues":[{"name":"b"}],"links":[{"rel":"next","href":"/v2/queues?marker=b&limit=1"}]}`)
		default:
			w.WriteHeader(204)
		}
	})
	var names []string
	for value, err := range newZaqarQueues(cloud).List(context.Background(),
		queues.WithListOptions(queues.ListOpts{Limit: 1, Detailed: true, WithCount: true}), queues.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name)
	}
	if !reflect.DeepEqual(names, []string{"a", "b"}) || len(requests) != 3 {
		t.Fatalf("names=%v requests=%v", names, requests)
	}
	first := requests[0]
	if first.Get("limit") != "1" || first.Get("detailed") != "true" || first.Get("with_count") != "true" || first.Get("vendor") != "x" || first.Has("name") || first.Has("marker") {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "a" || requests[2].Get("marker") != "b" {
		t.Fatalf("follow-up queries=%v", requests[1:])
	}
}

func TestZaqarQueueCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newZaqarQueues(cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := api.Create(canceled, queues.CreateOpts{QueueName: "jobs", DefaultMessageTTL: 60}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.GetStats(canceled, "jobs"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled stats: %v", err)
	}
	if err := api.Purge(canceled, "jobs", queues.PurgeOpts{ResourceTypes: []queues.PurgeResource{queues.ResourceMessages}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled purge: %v", err)
	}
	if err := api.Delete(canceled, "jobs"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := api.Create(deadline, queues.CreateOpts{QueueName: "jobs", DefaultMessageTTL: 60}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
