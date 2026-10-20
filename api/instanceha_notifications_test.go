package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/notifications"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const notificationUUID = "44444444-4444-4444-8444-444444444444"

func TestInstanceHANotificationCreateSnapshotsPayloadAndPreservesWorkflowJSON(t *testing.T) {
	cloud := testcloud.New(t)
	var creates atomic.Int32
	client := cloud.Client("instance-ha", "/instance-ha/v1/tenant")
	client.Microversion = "1.1"
	response := `{"notification":{"notification_uuid":"` + notificationUUID + `","id":9007199254740993,"status":"new","source_host_uuid":"` + hostUUID + `","generated_time":"2026-10-01 09:10:11","payload":{"event":"STOPPED","large":9007199254740995},"recovery_workflow_details":[{"name":"Evacuate","state":"RUNNING","progress":0.5,"custom":9007199254740997,"progress_details":[{"timestamp":"2026-10-01 09:10:11.123456","message":null,"progress":"0.5","extra":null}]}],"updated_at":null}}`
	cloud.Mux.HandleFunc("/instance-ha/v1/tenant/notifications", func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		if r.Method != "POST" || r.Header.Get("OpenStack-API-Version") != "instance-ha 1.1" {
			t.Errorf("%s %v", r.Method, r.Header)
		}
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body["notification"]["hostname"]) != `"compute-01"` || body["notification"]["host_name"] != nil || string(body["notification"]["payload"]) != `{"event":"STOPPED","large":9007199254740995}` {
			t.Errorf("body=%v", body)
		}
		w.Header().Set("X-Request-ID", "notification-accepted")
		testcloud.JSON(w, 202, response)
	})
	cloud.Mux.HandleFunc("/instance-ha/v1/tenant/notifications/"+notificationUUID, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, response) })
	payload := map[string]any{"event": "STOPPED", "large": json.Number("9007199254740995")}
	option := notifications.WithCreatePayload(payload)
	payload["event"] = "mutated"
	payload["large"] = 1
	api := notifications.New(client)
	value, err := api.Create(context.Background(), notifications.CreateOpts{Type: "COMPUTE_HOST", Hostname: "compute-01", GeneratedTime: "2026-10-01 09:10:11"}, option)
	if err != nil {
		t.Fatal(err)
	}
	if value.UUID != notificationUUID || string(value.ID) != "9007199254740993" || value.StatusCode != 202 || value.Header.Get("X-Request-ID") != "notification-accepted" || string(value.Payload) != `{"event":"STOPPED","large":9007199254740995}` || value.UpdatedAt != nil {
		t.Fatalf("value=%+v", value)
	}
	workflow := value.RecoveryWorkflowDetails[0]
	if string(workflow.Progress) != "0.5" || string(workflow.Body["custom"]) != "9007199254740997" || string(workflow.ProgressDetails[0].Progress) != `"0.5"` || string(workflow.ProgressDetails[0].Body["extra"]) != "null" || workflow.ProgressDetails[0].Message != nil {
		t.Fatalf("workflow=%+v", workflow)
	}
	if _, err := api.Get(context.Background(), value.UUID); err != nil {
		t.Fatal(err)
	}
	if creates.Load() != 1 || client.Microversion != "1.1" {
		t.Fatal("mutation replayed or source upgraded")
	}
}

func TestInstanceHANotificationListTypedAliasesAndHTTPPagination(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v1/tenant/notifications", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		q := r.URL.Query()
		if q.Get("generated-since") != "2026-10-01 00:00:00" || q.Has("generated_since") || q.Get("source_host_uuid") != hostUUID || q.Get("status") != "new" {
			t.Errorf("query=%v", q)
		}
		if q.Get("marker") == "next-page" {
			testcloud.JSON(w, 200, `{"notifications":[{"notification_uuid":"`+segmentUUID+`","id":2,"status":"new"}]}`)
			return
		}
		w.Header().Set("Link", `</v1/tenant/notifications?marker=next-page>; rel="next"`)
		testcloud.JSON(w, 200, `{"notifications":[{"notification_uuid":"`+notificationUUID+`","id":1,"status":"new"}]}`)
	})
	since, host, status := "2026-10-01 00:00:00", hostUUID, "new"
	option := notifications.WithListOptions(notifications.ListOpts{GeneratedSince: &since, SourceHostUUID: &host, Status: &status})
	since = "mutated"
	values, err := notifications.New(cloud.Client("instance-ha", "/v1/tenant")).All(context.Background(), option)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].UUID != notificationUUID || values[1].UUID != segmentUUID || calls.Load() != 2 {
		t.Fatalf("values=%+v calls=%d", values, calls.Load())
	}
}

func TestInstanceHANotificationRejectsNamesMutationsAndInvalidPayloadBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("unsupported request %s", r.URL) })
	api := notifications.New(cloud.Client("instance-ha", "/v1"))
	if _, err := api.Find(context.Background(), resource.Name("notification-name")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), resource.ID(notificationUUID)); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	for _, payload := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`"string"`), json.RawMessage(`[]`), json.RawMessage(`{"bad"`)} {
		if _, err := api.Create(context.Background(), notifications.CreateOpts{Type: "VM", Hostname: "host", GeneratedTime: "time", Payload: payload}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", payload, err)
		}
	}
	for _, key := range []string{"payload", "hostname", "notification_uuid", "status", "recovery_workflow_details"} {
		if _, err := api.Create(context.Background(), notifications.CreateOpts{Type: "VM", Hostname: "host", GeneratedTime: "time", Payload: json.RawMessage(`{}`)}, notifications.WithCreateField(key, "override")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", key, err)
		}
	}
}

func TestInstanceHANotificationWaitUsesRealStatusAndContext(t *testing.T) {
	for _, state := range []string{"finished", "failed", "error"} {
		t.Run(state, func(t *testing.T) {
			cloud := testcloud.New(t)
			var polls atomic.Int32
			cloud.Mux.HandleFunc("/v1/notifications/"+notificationUUID, func(w http.ResponseWriter, r *http.Request) {
				status := "running"
				if polls.Add(1) > 1 {
					status = state
				}
				testcloud.JSON(w, 200, `{"notification":{"notification_uuid":"`+notificationUUID+`","status":"`+status+`","recovery_workflow_details":[{"progress":0.75}]}}`)
			})
			var progress []int
			value, err := notifications.New(cloud.Client("instance-ha", "/v1")).Wait(context.Background(), resource.ID(notificationUUID), "finished", resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(value int) { progress = append(progress, value) }))
			if state == "finished" {
				if err != nil || value.Status != state {
					t.Fatal(err)
				}
			} else {
				var failed *resource.FailedStateError
				if !errors.As(err, &failed) {
					t.Fatal(err)
				}
			}
			if polls.Load() != 2 || !reflect.DeepEqual(progress, []int{0}) {
				t.Fatalf("polls=%d progress=%v", polls.Load(), progress)
			}
		})
	}
	cloud := testcloud.New(t)
	api := notifications.New(cloud.Client("instance-ha", "/v1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Wait(ctx, resource.ID(notificationUUID), "finished"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestInstanceHANotificationAcceptedDecodeAndHTTPErrorsKeepEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	var creates atomic.Int32
	cloud.Mux.HandleFunc("/v1/notifications", func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		w.Header().Set("X-Request-ID", "accepted")
		testcloud.JSON(w, 202, `{"notification":{"id":9007199254740993,"recovery_workflow_details":"invalid"}}`)
	})
	cloud.Mux.HandleFunc("/v1/notifications/"+notificationUUID, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, `{"error":"forbidden"}`) })
	api := notifications.New(cloud.Client("instance-ha", "/v1"))
	_, err := api.Create(context.Background(), notifications.CreateOpts{Type: "VM", Hostname: "h", GeneratedTime: "time", Payload: json.RawMessage(`{}`)})
	var evidence *resource.ResponseError
	if !errors.As(err, &evidence) || evidence.StatusCode != 202 || evidence.Header.Get("X-Request-ID") != "accepted" || creates.Load() != 1 {
		t.Fatalf("evidence=%+v err=%v", evidence, err)
	}
	if _, err := api.Get(context.Background(), notificationUUID); !gophercloud.ResponseCodeIs(err, 403) {
		t.Fatal(err)
	}
}

func TestInstanceHANotificationIteratorSnapshotsOptionSlice(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("status") != "new" {
			t.Errorf("query=%v", r.URL.Query())
		}
		testcloud.JSON(w, 200, `{"notifications":[{"notification_uuid":"`+notificationUUID+`","status":"new"}]}`)
	})
	status := "new"
	options := []notifications.ListOption{notifications.WithListOptions(notifications.ListOpts{Status: &status})}
	iterator := notifications.New(cloud.Client("instance-ha", "/v1")).List(context.Background(), options...)
	options[0] = notifications.WithListQuery("status", "mutated")
	status = "also-mutated"
	for value, err := range iterator {
		if err != nil || value.Status != "new" {
			t.Fatalf("value=%+v err=%v", value, err)
		}
	}
}
