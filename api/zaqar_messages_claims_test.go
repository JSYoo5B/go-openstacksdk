package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/messaging/v2/claims"
	"github.com/JSYoo5B/go-openstacksdk/messaging/v2/messages"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func newZaqarMessages(cloud *testcloud.Cloud) *messages.API {
	return messages.New(cloud.Client("messaging", "/zaqar"))
}

func newZaqarClaims(cloud *testcloud.Cloud) *claims.API {
	return claims.New(cloud.Client("messaging", "/zaqar"))
}

func zaqarMessageRaw(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("request body: %v", err)
	}
	return string(data)
}

func zaqarMessageUnexpected(t *testing.T, err error, status int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != status {
		t.Fatalf("want unexpected response code %d, got %v", status, err)
	}
}

func TestZaqarMessageCreateSendsBatchEnvelopePreservingZeroValuesAndRequires201(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /zaqar/v2/queues/jobs/messages", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			var body map[string]json.RawMessage
			if err := json.Unmarshal([]byte(zaqarMessageRaw(t, r)), &body); err != nil {
				t.Errorf("body: %v", err)
			}
			if len(body) != 2 || string(body["vendor"]) != `{"flag":false}` {
				t.Errorf("body=%v", body)
			}
			var items []map[string]json.RawMessage
			if err := json.Unmarshal(body["messages"], &items); err != nil || len(items) != 2 {
				t.Errorf("messages=%s error=%v", body["messages"], err)
			}
			if len(items) == 2 {
				if string(items[0]["ttl"]) != `300` || string(items[0]["delay"]) != `5` || string(items[0]["body"]) != `{"event":"a","flag":false}` {
					t.Errorf("first=%v", items[0])
				}
				if _, ok := items[1]["ttl"]; ok {
					t.Errorf("zero ttl is omitted by the native model: %v", items[1])
				}
				if _, ok := items[1]["delay"]; ok {
					t.Errorf("zero delay is omitted by the native model: %v", items[1])
				}
				if string(items[1]["body"]) != `{}` {
					t.Errorf("empty body must stay an object: %v", items[1])
				}
			}
			if id := r.Header.Get("Client-ID"); id != "" {
				t.Errorf("native messages Create must not add Client-ID, got %q", id)
			}
			testcloud.JSON(w, tc.status, `{"resources":["/v2/queues/jobs/messages/m1","/v2/queues/jobs/messages/m2"]}`)
		})
		list, err := newZaqarMessages(cloud).Create(context.Background(), "jobs", messages.BatchCreateOpts{
			{TTL: 300, Delay: 5, Body: map[string]any{"event": "a", "flag": false}},
			{Body: map[string]any{}},
		}, messages.WithCreateField("vendor", map[string]bool{"flag": false}))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d error=%v calls=%d", tc.status, err, calls.Load())
		}
		if tc.success && len(list.Resources) != 2 {
			t.Fatalf("list=%+v", list)
		}
		if !tc.success {
			zaqarMessageUnexpected(t, err, tc.status)
		}
	}
}

func TestZaqarMessageRejectsCollisionsAndNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newZaqarMessages(cloud)
	ctx := context.Background()
	batch := messages.BatchCreateOpts{{Body: map[string]any{"a": 1}}}
	if _, err := api.Create(ctx, "jobs", batch, messages.WithCreateField("messages", []int{})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("envelope collision: %v", err)
	}
	if _, err := api.Create(ctx, "jobs", batch, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Create(ctx, "jobs", messages.BatchCreateOpts{{TTL: 1}}); err == nil {
		t.Fatal("a message without body must fail before HTTP")
	}
	if err := api.Delete(ctx, "jobs", "m1", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil delete option: %v", err)
	}
	if err := api.DeleteMessages(ctx, "jobs", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil delete messages option: %v", err)
	}
	if _, err := api.GetMessages(ctx, "jobs", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil get messages option: %v", err)
	}
	if _, err := api.PopMessages(ctx, "jobs", messages.PopMessagesOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil pop option: %v", err)
	}
	for _, err := range api.List(ctx, "jobs", nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestZaqarMessageListFiltersFollowLinksAndStatus204(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/jobs/messages", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		switch r.URL.Query().Get("marker") {
		case "":
			testcloud.JSON(w, 200, `{"messages":[{"id":"m1","ttl":60,"age":1,"body":{"flag":false},"href":"/v2/queues/jobs/messages/m1","checksum":"c"}],"links":[{"rel":"next","href":"/v2/queues/jobs/messages?marker=m1&limit=1"}]}`)
		case "m1":
			testcloud.JSON(w, 200, `{"messages":[{"id":"m2","body":{}}],"links":[{"rel":"next","href":"/v2/queues/jobs/messages?marker=m2&limit=1"}]}`)
		default:
			w.WriteHeader(204)
		}
	})
	var ids []string
	var first *messages.Message
	for value, err := range newZaqarMessages(cloud).List(context.Background(), "jobs",
		messages.WithListOptions(messages.ListOpts{Limit: 1, Echo: true, IncludeClaimed: true, IncludeDelayed: true}), messages.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = value
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"m1", "m2"}) || len(requests) != 3 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	if first.TTL != 60 || first.Body["flag"] != false || first.Checksum != "c" {
		t.Fatalf("first=%+v", first)
	}
	q := requests[0]
	if q.Get("limit") != "1" || q.Get("echo") != "true" || q.Get("include_claimed") != "true" || q.Get("include_delayed") != "true" || q.Get("vendor") != "x" {
		t.Fatalf("first query=%v", q)
	}
	if requests[1].Get("marker") != "m1" || requests[2].Get("marker") != "m2" {
		t.Fatalf("follow-up queries=%v", requests[1:])
	}
}

func TestZaqarMessageGetAndGetMessagesQueriesAndMissingError(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/jobs/messages/m1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"m1","ttl":0,"age":2,"body":{"flag":false},"href":"/h","checksum":"c"}`)
	})
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/jobs/messages/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/jobs/messages", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"messages":[{"id":"m1","body":{"flag":false}},{"id":"m2","body":{}}]}`)
	})
	api := newZaqarMessages(cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "jobs", "m1")
	if err != nil || value.ID != "m1" || value.TTL != 0 || value.Body["flag"] != false || value.Age != 2 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Get(ctx, "jobs", "missing"); err == nil {
		t.Fatal("missing message must be an error")
	} else {
		zaqarMessageUnexpected(t, err, 404)
	}
	list, err := api.GetMessages(ctx, "jobs", messages.WithGetMessagesOptions(messages.GetMessagesOpts{IDs: []string{"m1", "m2"}}), messages.WithGetMessagesQuery("vendor", "x"))
	if err != nil || len(list) != 2 || list[1].ID != "m2" {
		t.Fatalf("list=%+v error=%v", list, err)
	}
	if _, err := api.GetMessages(ctx, "jobs"); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 || !reflect.DeepEqual(queries[0]["ids"], []string{"m1", "m2"}) || queries[0].Get("vendor") != "x" || len(queries[1]) != 0 {
		t.Fatalf("queries=%v", queries)
	}
}

func TestZaqarMessageDeleteClaimQueryAndRequires204(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("DELETE /zaqar/v2/queues/jobs/messages/{id}", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "odd":
			w.WriteHeader(200)
		default:
			w.WriteHeader(204)
		}
	})
	api := newZaqarMessages(cloud)
	ctx := context.Background()
	if err := api.Delete(ctx, "jobs", "m1", messages.WithDeleteOptions(messages.DeleteOpts{ClaimID: "c1"}), messages.WithDeleteQuery("vendor", "x")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "jobs", "m2"); err != nil {
		t.Fatal(err)
	}
	if want := []url.Values{{"claim_id": {"c1"}, "vendor": {"x"}}, {}}; !reflect.DeepEqual(queries, want) {
		t.Fatalf("queries=%v", queries)
	}
	zaqarMessageUnexpected(t, api.Delete(ctx, "jobs", "missing"), 404)
	zaqarMessageUnexpected(t, api.Delete(ctx, "jobs", "odd"), 200)
	if got, want := api.DeleteMessageURL(ctx, "jobs", "m1"), cloud.Server.URL+"/zaqar/v2/queues/jobs/messages/m1"; got != want {
		t.Fatalf("DeleteMessageURL=%q want %q", got, want)
	}
}

func TestZaqarMessageBulkDeleteSendsIDsAndAcceptsOnly200Or204(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {204, true}, {202, false}} {
		cloud := testcloud.New(t)
		var queries []url.Values
		cloud.Mux.HandleFunc("DELETE /zaqar/v2/queues/jobs/messages", func(w http.ResponseWriter, r *http.Request) {
			queries = append(queries, r.URL.Query())
			w.WriteHeader(tc.status)
		})
		api := newZaqarMessages(cloud)
		err := api.DeleteMessages(context.Background(), "jobs", messages.WithDeleteMessagesOptions(messages.DeleteMessagesOpts{IDs: []string{"m1", "m2"}}), messages.WithDeleteMessagesQuery("vendor", "x"))
		if tc.success != (err == nil) {
			t.Fatalf("status %d error=%v", tc.status, err)
		}
		if !tc.success {
			zaqarMessageUnexpected(t, err, tc.status)
		}
		if len(queries) != 1 || !reflect.DeepEqual(queries[0]["ids"], []string{"m1", "m2"}) || queries[0].Get("vendor") != "x" {
			t.Fatalf("queries=%v", queries)
		}
	}
}

func TestZaqarMessagePopSendsPopQueryAndDecodesClaimFields(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("DELETE /zaqar/v2/queues/jobs/messages", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.URL.Query().Get("pop") == "9" {
			w.WriteHeader(204)
			return
		}
		testcloud.JSON(w, 200, `{"messages":[{"id":"m1","ttl":0,"age":3,"claim_count":1,"claim_id":"c1","body":{"flag":false}}]}`)
	})
	api := newZaqarMessages(cloud)
	ctx := context.Background()
	popped, err := api.PopMessages(ctx, "jobs", messages.PopMessagesOpts{Pop: 2}, messages.WithPopMessagesQuery("vendor", "x"))
	if err != nil || len(popped) != 1 || popped[0].ClaimID != "c1" || popped[0].ClaimCount != 1 || popped[0].Body["flag"] != false {
		t.Fatalf("popped=%+v error=%v", popped, err)
	}
	empty, err := api.PopMessages(ctx, "jobs", messages.PopMessagesOpts{Pop: 9})
	if err != nil || len(empty) != 0 {
		t.Fatalf("204 means no messages: %+v %v", empty, err)
	}
	if _, err := api.PopMessages(ctx, "jobs", messages.PopMessagesOpts{}); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 3 || queries[0].Get("pop") != "2" || queries[0].Get("vendor") != "x" || queries[1].Get("pop") != "9" || queries[2].Has("pop") {
		t.Fatalf("queries=%v", queries)
	}
}

func TestZaqarMessageCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newZaqarMessages(cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	batch := messages.BatchCreateOpts{{Body: map[string]any{"a": 1}}}
	if _, err := api.Create(canceled, "jobs", batch); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Get(canceled, "jobs", "m1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Delete(canceled, "jobs", "m1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	if err := api.DeleteMessages(canceled, "jobs"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled bulk delete: %v", err)
	}
	if _, err := api.PopMessages(canceled, "jobs", messages.PopMessagesOpts{Pop: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled pop: %v", err)
	}
	if _, err := api.GetMessages(canceled, "jobs"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get messages: %v", err)
	}
	for _, err := range api.List(canceled, "jobs") {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, "jobs", batch); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

func TestZaqarClaimCreateSendsTTLGraceAndLimitQuery(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {204, true}, {200, false}, {202, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /zaqar/v2/queues/jobs/claims", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if got := r.URL.Query(); got.Get("limit") != "5" || got.Get("vendor") != "x" {
				t.Errorf("query=%v", got)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal([]byte(zaqarMessageRaw(t, r)), &body); err != nil {
				t.Errorf("body: %v", err)
			}
			if len(body) != 3 || string(body["ttl"]) != `300` || string(body["grace"]) != `60` || string(body["vendor"]) != `false` {
				t.Errorf("body=%v", body)
			}
			if id := r.Header.Get("Client-ID"); id != "" {
				t.Errorf("native claims Create must not add Client-ID, got %q", id)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"messages":[{"href":"/v2/queues/jobs/messages/m1?claim_id=c1","age":1.5,"ttl":0,"body":{"flag":false}}]}`)
		})
		got, err := newZaqarClaims(cloud).Create(context.Background(), "jobs", claims.CreateOpts{TTL: 300, Grace: 60, Limit: 5},
			claims.WithCreateField("vendor", false), claims.WithCreateQuery("vendor", "x"))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d error=%v calls=%d", tc.status, err, calls.Load())
		}
		switch {
		case tc.status == 201 && (len(got) != 1 || got[0].Body["flag"] != false || got[0].Age != 1.5):
			t.Fatalf("claimed=%+v", got)
		case tc.status == 204 && len(got) != 0:
			t.Fatalf("204 means nothing to claim: %+v", got)
		case !tc.success:
			zaqarMessageUnexpected(t, err, tc.status)
		}
	}
}

func TestZaqarClaimGetUpdateDeleteWireShapes(t *testing.T) {
	cloud := testcloud.New(t)
	var updateStatus, deleteStatus atomic.Int32
	updateStatus.Store(204)
	deleteStatus.Store(204)
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/jobs/claims/c1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"age":2.5,"href":"/v2/queues/jobs/claims/c1","ttl":300,"messages":[{"href":"/m","age":1,"ttl":0,"body":{"flag":false}}]}`)
	})
	cloud.Mux.HandleFunc("GET /zaqar/v2/queues/jobs/claims/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("PATCH /zaqar/v2/queues/jobs/claims/c1", func(w http.ResponseWriter, r *http.Request) {
		if body := zaqarMessageRaw(t, r); body != `{"grace":60,"ttl":120,"vendor":false}` {
			t.Errorf("update body=%s", body)
		}
		w.WriteHeader(int(updateStatus.Load()))
	})
	cloud.Mux.HandleFunc("DELETE /zaqar/v2/queues/jobs/claims/c1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(deleteStatus.Load()))
	})
	api := newZaqarClaims(cloud)
	ctx := context.Background()
	claim, err := api.Get(ctx, "jobs", "c1")
	if err != nil || claim == nil || claim.TTL != 300 || claim.Age != 2.5 || len(claim.Messages) != 1 || claim.Messages[0].Body["flag"] != false {
		t.Fatalf("claim=%+v error=%v", claim, err)
	}
	if _, err := api.Get(ctx, "jobs", "missing"); err == nil {
		t.Fatal("missing claim must be an error")
	} else {
		zaqarMessageUnexpected(t, err, 404)
	}
	if err := api.Update(ctx, "jobs", "c1", claims.UpdateOpts{TTL: 120, Grace: 60}, claims.WithUpdateField("vendor", false)); err != nil {
		t.Fatal(err)
	}
	updateStatus.Store(200)
	zaqarMessageUnexpected(t, api.Update(ctx, "jobs", "c1", claims.UpdateOpts{TTL: 120, Grace: 60}, claims.WithUpdateField("vendor", false)), 200)
	if err := api.Delete(ctx, "jobs", "c1"); err != nil {
		t.Fatal(err)
	}
	deleteStatus.Store(200)
	zaqarMessageUnexpected(t, api.Delete(ctx, "jobs", "c1"), 200)
}

func TestZaqarClaimRejectsCollisionsAndNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newZaqarClaims(cloud)
	ctx := context.Background()
	if _, err := api.Create(ctx, "jobs", claims.CreateOpts{TTL: 1}, claims.WithCreateField("ttl", 2)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected create field: %v", err)
	}
	if _, err := api.Create(ctx, "jobs", claims.CreateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if err := api.Update(ctx, "jobs", "c1", claims.UpdateOpts{}, claims.WithUpdateField("grace", 2)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected update field: %v", err)
	}
	if err := api.Update(ctx, "jobs", "c1", claims.UpdateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil update option: %v", err)
	}
}

func TestZaqarClaimCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newZaqarClaims(cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Create(canceled, "jobs", claims.CreateOpts{TTL: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Get(canceled, "jobs", "c1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Update(canceled, "jobs", "c1", claims.UpdateOpts{TTL: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled update: %v", err)
	}
	if err := api.Delete(canceled, "jobs", "c1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, "jobs", claims.CreateOpts{TTL: 1}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
