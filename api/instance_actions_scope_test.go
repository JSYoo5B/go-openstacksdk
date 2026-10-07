package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/instanceactions"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const actionSummary = `{"action":"reboot","instance_uuid":"wire-server","message":null,"project_id":"project","request_id":"req-one","start_time":"2026-10-01T00:00:00.123456","updated_at":"2026-10-01T00:01:00.000000","user_id":"user","status":"extension-list","vendor":{"attempt":1}}`

func newActionScope(t *testing.T, cloud *testcloud.Cloud) *instanceactions.ActionScope {
	t.Helper()
	client := cloud.Client("compute", "/v2.1")
	client.Microversion = "2.84"
	scope, err := instanceactions.New(client).InServer(context.Background(), resource.ID("server-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestInstanceActionScopePreservesSummaryDetailEventsAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, details atomic.Int32
	cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.84" || r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("vendor") != "a&b" {
			t.Errorf("method=%s version=%s query=%s", r.Method, r.Header.Get("X-OpenStack-Nova-API-Version"), r.URL.RawQuery)
		}
		w.Header().Set("X-Openstack-Request-Id", "req-page")
		if r.URL.Query().Get("marker") == "req-two" {
			testcloud.JSON(w, 200, `{"instanceActions":[{"action":"reboot","request_id":"req-three","message":null}]}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"instanceActions": []json.RawMessage{json.RawMessage(actionSummary), json.RawMessage(`{"action":"reboot","request_id":"req-two","message":"failed"}`)},
			"links":           []map[string]string{{"rel": "next", "href": cloud.Server.URL + "/v2.1/servers/server-fixed/os-instance-actions?marker=req-two&limit=2&vendor=a%26b"}},
		})
	})
	cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions/req-one", func(w http.ResponseWriter, r *http.Request) {
		details.Add(1)
		w.Header().Set("X-Openstack-Request-Id", "req-detail")
		testcloud.JSON(w, 200, `{"instanceAction":{"action":"reboot","instance_uuid":"wire-server","message":"failed","project_id":"project","request_id":"req-one","start_time":"2026-10-01T00:00:00.123456","updated_at":"2026-10-01T00:01:00.000000","user_id":"user","status":"extension-detail","events":[{"event":"compute_reboot_instance","start_time":"2026-10-01T00:00:00.456789","finish_time":"2026-10-01T00:01:00.000000","result":"Error","traceback":"trace","host":"compute-a","hostId":"hashed","details":"fault explanation","status":{"vendor":"failed"}}]}}`)
	})
	scope := newActionScope(t, cloud)
	if scope.ServerID() != "server-fixed" || lists.Load() != 0 || details.Load() != 0 {
		t.Fatal("explicit parent ID triggered lookup")
	}
	if id, err := scope.ResolveID(context.Background(), resource.ID("req-one")); err != nil || id != "req-one" || lists.Load() != 0 || details.Load() != 0 {
		t.Fatalf("id=%q err=%v", id, err)
	}
	values, err := scope.All(context.Background(), resource.WithPageSize(2), resource.WithQuery("vendor", "a&b"))
	if err != nil || len(values) != 3 || lists.Load() != 2 || details.Load() != 0 || values[2].RequestID != "req-three" {
		t.Fatalf("values=%v lists=%d details=%d err=%v", values, lists.Load(), details.Load(), err)
	}
	first := values[0]
	if first.ServerID != "server-fixed" || first.InstanceUUID != "wire-server" || first.Action != "reboot" || first.ProjectID != "project" || first.UserID != "user" || first.StartTime.IsZero() || first.UpdatedAt == nil || first.UpdatedAt.IsZero() || first.Details != nil || first.Events != nil || string(first.Body["status"]) != `"extension-list"` || string(first.Body["message"]) != "null" || string(first.Body["vendor"]) != `{"attempt":1}` || first.Header.Get("X-Openstack-Request-Id") != "req-page" {
		t.Fatalf("summary=%+v", first)
	}
	first.Header.Set("X-Openstack-Request-Id", "caller-change")
	if values[1].Header.Get("X-Openstack-Request-Id") != "req-page" {
		t.Fatal("listed records share mutable response headers")
	}
	value, err := scope.Find(context.Background(), resource.ID("req-one"))
	if err != nil || value == nil || value.Details == nil || value.Details.Events == nil || value.Events == nil || len(*value.Events) != 1 || details.Load() != 1 || lists.Load() != 2 {
		t.Fatalf("detail=%+v err=%v", value, err)
	}
	event := (*value.Events)[0]
	if value.ServerID != "server-fixed" || value.InstanceUUID != "wire-server" || value.RequestID != "req-one" || value.Details.RequestID != value.RequestID || value.UpdatedAt == nil || value.Details.UpdatedAt == nil || string(value.Body["status"]) != `"extension-detail"` || value.Header.Get("X-Openstack-Request-Id") != "req-detail" || event.Event != "compute_reboot_instance" || event.Details == nil || *event.Details != "fault explanation" || event.Host == nil || *event.Host != "compute-a" || event.HostID == nil || *event.HostID != "hashed" || event.Traceback != "trace" || event.StartTime.IsZero() || event.FinishTime.IsZero() || string(event.Body["status"]) != `{"vendor":"failed"}` {
		t.Fatalf("value=%+v event=%+v", value, event)
	}
}

func TestInstanceActionScopeDistinguishesUnavailableEmptyAndNullEvents(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		present     bool
		empty       bool
	}{
		{"not returned by policy", "", false, false},
		{"null events", `,"events":null`, true, false},
		{"empty events", `,"events":[]`, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions/req-one", func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-OpenStack-Nova-API-Version") != "2.1" {
					t.Errorf("scope changed selected microversion: %s", r.Header.Get("X-OpenStack-Nova-API-Version"))
				}
				testcloud.JSON(w, 200, `{"instanceAction":{"request_id":"req-one"`+tc.field+`}}`)
			})
			client := cloud.Client("compute", "/v2.1")
			client.Microversion = "2.1"
			scope, err := instanceactions.New(client).InServer(context.Background(), resource.ID("server-fixed"))
			if err != nil {
				t.Fatal(err)
			}
			value, err := scope.Get(context.Background(), "req-one")
			if err != nil || value == nil || value.Details == nil || (value.Events != nil) != tc.empty || (value.Details.Events != nil) != tc.empty {
				t.Fatalf("value=%+v err=%v", value, err)
			}
			if _, present := value.Body["events"]; present != tc.present {
				t.Fatalf("event field presence lost: %s", value.Body["events"])
			}
		})
	}
}

func TestInstanceActionScopePreservesParentLookupFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		cause      error
	}{
		{"missing", `{"servers":[]}`, 200, resource.ErrNotFound},
		{"ambiguous", `{"servers":[{"id":"one","name":"worker"},{"id":"two","name":"worker"}]}`, 200, resource.ErrAmbiguous},
		{"forbidden", `{"forbidden":{"message":"parent denied"}}`, 403, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/v2.1/servers/detail", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, tc.status, tc.body)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("failed parent lookup issued action request: %s", r.URL.Path)
				testcloud.JSON(w, 404, "{}")
			})
			api := instanceactions.New(cloud.Client("compute", "/v2.1"))
			scope, err := api.InServer(context.Background(), resource.Name("worker"))
			if scope != nil || err == nil || calls.Load() != 1 || tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Fatalf("scope=%v calls=%d err=%v", scope, calls.Load(), err)
			}
			if tc.status == 403 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 || string(response.Body) != tc.body {
					t.Fatalf("response=%+v err=%v", response, err)
				}
			}
		})
	}
}

func TestInstanceActionScopeResolvesExactServerNameOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var parentCalls, actionCalls atomic.Int32
	cloud.Mux.HandleFunc("/v2.1/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		parentCalls.Add(1)
		if r.URL.Query().Get("name") != "^worker$" {
			t.Errorf("name query=%q", r.URL.Query().Get("name"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"servers": []map[string]string{{"id": "wrong", "name": "worker-extra"}}, "servers_links": []map[string]string{{"rel": "next", "href": cloud.Server.URL + "/v2.1/servers/page2"}}})
	})
	cloud.Mux.HandleFunc("/v2.1/servers/page2", func(w http.ResponseWriter, r *http.Request) {
		parentCalls.Add(1)
		testcloud.JSON(w, 200, `{"servers":[{"id":"server-fixed","name":"worker"}]}`)
	})
	cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions/req-one", func(w http.ResponseWriter, r *http.Request) {
		actionCalls.Add(1)
		testcloud.JSON(w, 200, `{"instanceAction":{"request_id":"req-one","instance_uuid":"different-wire-parent"}}`)
	})
	api := instanceactions.New(cloud.Client("compute", "/v2.1"))
	scope, err := api.InServer(context.Background(), resource.Name("worker"))
	if err != nil || scope == nil || scope.ServerID() != "server-fixed" || parentCalls.Load() != 2 {
		t.Fatalf("scope=%v parents=%d err=%v", scope, parentCalls.Load(), err)
	}
	for range 2 {
		value, err := scope.Get(context.Background(), "req-one")
		if err != nil || value.ServerID != "server-fixed" || value.InstanceUUID != "different-wire-parent" {
			t.Fatalf("value=%v err=%v", value, err)
		}
	}
	if parentCalls.Load() != 2 || actionCalls.Load() != 2 {
		t.Fatal("action operations resolved the parent again")
	}
}

func TestInstanceActionScopeRejectsUnsupportedPoliciesBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 404, "{}") })
	api := instanceactions.New(cloud.Client("compute", "/v2.1"))
	ctx := context.Background()
	if _, err := api.InServer(ctx, resource.ID("..")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.InServer(canceled, resource.ID("server-fixed")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := instanceactions.New(nil).InServer(ctx, resource.ID("server-fixed")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	scope := newActionScope(t, cloud)
	if _, err := scope.Find(ctx, resource.Name("reboot")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := scope.ResolveID(ctx, resource.Name("reboot")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID("req-one")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := scope.Wait(ctx, resource.ID("req-one"), "complete"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := scope.Get(ctx, "../req-one"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, option := range []resource.ListOption{resource.WithName("reboot"), resource.WithStatus("complete"), resource.WithPageSize(0), nil} {
		count := 0
		for value, err := range scope.List(ctx, option) {
			count++
			if value != nil || err == nil {
				t.Fatalf("value=%v err=%v", value, err)
			}
		}
		if count != 1 {
			t.Fatalf("error count=%d", count)
		}
	}
	if _, err := scope.Get(canceled, "req-one"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.All(canceled); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

func TestInstanceActionScopeStopsPaginationOnBreakCancelAndCycle(t *testing.T) {
	for _, mode := range []string{"break", "cancel", "cycle"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				next := cloud.Server.URL + "/v2.1/servers/server-fixed/os-instance-actions?marker=req-one"
				if mode == "cycle" {
					next = cloud.Server.URL + r.URL.String()
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"instanceActions": []json.RawMessage{json.RawMessage(actionSummary)}, "links": []map[string]string{{"rel": "next", "href": next}}})
			})
			scope := newActionScope(t, cloud)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			values, failures := 0, 0
			for value, err := range scope.List(ctx) {
				if err != nil {
					failures++
					if value != nil || mode == "cancel" && !errors.Is(err, context.Canceled) || mode == "cycle" && !errors.Is(err, resource.ErrPaginationCycle) {
						t.Fatalf("value=%v err=%v", value, err)
					}
					continue
				}
				values++
				if mode == "break" {
					break
				}
				if mode == "cancel" {
					cancel()
				}
			}
			if calls.Load() != 1 || values != 1 || mode == "break" && failures != 0 || mode != "break" && failures != 1 {
				t.Fatalf("calls=%d values=%d failures=%d", calls.Load(), values, failures)
			}
		})
	}
}

func TestInstanceActionScopePreservesGetErrorsAndMissingPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"missing", `{"itemNotFound":{"message":"no action"}}`, 404},
		{"forbidden", `{"forbidden":{"message":"policy denied"}}`, 403},
		{"invalid timestamp", `{"instanceAction":{"request_id":"req-one","start_time":"not-a-time"}}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions/req-one", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "req-error")
				testcloud.JSON(w, tc.status, tc.body)
			})
			scope := newActionScope(t, cloud)
			value, err := scope.Find(context.Background(), resource.ID("req-one"))
			if value != nil || err == nil {
				t.Fatalf("value=%v err=%v", value, err)
			}
			if tc.status != 200 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != tc.status || string(response.Body) != tc.body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "req-error" {
					t.Fatalf("response=%+v err=%v", response, err)
				}
				if tc.status == 404 && !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
				value, err = scope.Find(context.Background(), resource.ID("req-one"), resource.WithIgnoreMissing())
				if tc.status == 404 && (value != nil || err != nil) || tc.status == 403 && err == nil {
					t.Fatalf("ignore-missing value=%v err=%v", value, err)
				}
			} else {
				var parse *time.ParseError
				if !errors.As(err, &parse) {
					t.Fatalf("parse error cause lost: %v", err)
				}
			}
		})
	}
}

func TestInstanceActionScopeReturnsContinuationErrorOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("marker") != "" {
			testcloud.JSON(w, 403, `{"forbidden":{"message":"page denied"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"instanceActions": []json.RawMessage{json.RawMessage(actionSummary)}, "links": []map[string]string{{"rel": "next", "href": cloud.Server.URL + "/v2.1/servers/server-fixed/os-instance-actions?marker=req-one"}}})
	})
	scope := newActionScope(t, cloud)
	values, failures := 0, 0
	for value, err := range scope.List(context.Background()) {
		if err != nil {
			failures++
			var response gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.As(err, &response) || response.Actual != 403 || string(response.Body) != `{"forbidden":{"message":"page denied"}}` {
				t.Fatalf("value=%v response=%+v err=%v", value, response, err)
			}
			continue
		}
		values++
	}
	if calls.Load() != 2 || values != 1 || failures != 1 {
		t.Fatalf("calls=%d values=%d failures=%d", calls.Load(), values, failures)
	}
}

func TestInstanceActionScopeRejectsNonObjectActionsAndInvalidEvents(t *testing.T) {
	for _, body := range []string{
		`null`, `[]`, `1`, `"action"`,
		`{"request_id":"req-one","events":"not-an-array"}`,
		`{"request_id":"req-one","events":{}}`,
		`{"request_id":"req-one","events":[null]}`,
		`{"request_id":"req-one","events":[[]]}`,
		`{"request_id":"req-one","events":[1]}`,
		`{"request_id":"req-one","events":[{"details":123}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"instanceActions":[`+body+`]}`)
			})
			cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions/req-one", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"instanceAction":`+body+`}`)
			})
			scope := newActionScope(t, cloud)
			if value, err := scope.Get(context.Background(), "req-one"); value != nil || err == nil {
				t.Fatalf("non-object or malformed detail accepted: value=%+v err=%v", value, err)
			}
			failures := 0
			for value, err := range scope.List(context.Background()) {
				failures++
				if value != nil || err == nil {
					t.Fatalf("non-object or malformed listing accepted: value=%+v err=%v", value, err)
				}
			}
			if failures != 1 {
				t.Fatalf("decode errors=%d", failures)
			}
		})
	}
}

func TestInstanceActionScopeRequiresArrayListingAndPreservesEmptyArray(t *testing.T) {
	for _, body := range []string{
		`null`, `[]`, `1`, `{"instanceActions":null}`, `{"instanceActions":{}}`, `{}`,
		`{"instanceActions":[]}`,
	} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/v2.1/servers/server-fixed/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, body)
			})
			scope := newActionScope(t, cloud)
			values, err := scope.All(context.Background())
			if body == `{"instanceActions":[]}` {
				if err != nil || values == nil || len(values) != 0 {
					t.Fatalf("empty listing values=%v err=%v", values, err)
				}
			} else if err == nil || values != nil {
				t.Fatalf("malformed listing values=%v err=%v", values, err)
			}
		})
	}
}
