package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Pinned update_receiver passes attrs to Proxy._update and Resource.commit:
// current-value comparison/sticky dirty fields (resource.py:214-246), dirty-only
// body (:1231-1249), clean no-op (:1914-1919), shallow response merge/reset
// (:1380-1387), and direct base_path forwarding (proxy.py:716-748). Receivers
// use synchronous PATCH200. Fixed scope/identity, immutable views and strict
// accepted response errors are documented Go policies, not live cloud proof.
func receiverTrackedSeed(t *testing.T, id string) *receivers.Receiver {
	t.Helper()
	var value receivers.Receiver
	body := fmt.Sprintf(`{"id":%q,"name":"A","type":"message","action":"CLUSTER_SCALE_OUT","cluster_id":"cluster","user":"cached-owner","params":{"a":1,"b":2},"actor":{"large":9007199254740993},"channel":{"alarm_url":"https://foreign.invalid/trigger"},"future":{"off":false}}`, id)
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatal(err)
	}
	value.Header, value.StatusCode = http.Header{"X-Request-Id": {"seed"}}, 200
	return &value
}

func receiverTrackedHandle(t *testing.T, client *gophercloud.ServiceClient) *receivers.TrackedReceiver {
	t.Helper()
	tracked, err := receivers.New(client).Track(receiverTrackedSeed(t, "fixed"))
	if err != nil {
		t.Fatal(err)
	}
	return tracked
}

func receiverTrackedBody(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	var body map[string]map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
	}
	if len(body) != 1 || body["receiver"] == nil {
		t.Error("missing receiver object", body)
	}
	return body["receiver"]
}

func TestClusteringReceiverTrackedCleanStickyDirtyShallowMergeAndActualResponse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("PATCH /v1/receivers/fixed", func(w http.ResponseWriter, r *http.Request) {
		fields := receiverTrackedBody(t, r)
		if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Vendor") != "kept" {
			t.Error(r.URL, r.Header)
		}
		w.Header().Set("Location", "https://foreign.invalid/not-an-action#trace")
		w.Header().Set("X-Request-ID", "last-patch")
		if calls.Add(1) == 1 {
			if len(fields) != 1 || string(fields["name"]) != `"A"` {
				t.Error(fields)
			}
			testcloud.JSON(w, 200, `{"receiver":{"name":"Server","params":{"a":3}}}`)
		} else {
			if len(fields) != 3 || string(fields["name"]) != `"B"` || string(fields["params"]) != `{"big":9007199254740993,"off":false}` || string(fields["vendor"]) != "false" {
				t.Error(fields)
			}
			testcloud.JSON(w, 200, `{"receiver":{"future":{"returned":true}}}`)
		}
	})
	tracked := receiverTrackedHandle(t, cloud.Client("clustering", "/v1"))
	for _, opts := range []receivers.UpdateOpts{{}, {Name: request.Present("A")}, {Params: json.RawMessage(`{ "b":2.0,"a":1e0 }`)}} {
		if err := tracked.Edit(opts); err != nil || tracked.Dirty() {
			t.Fatal(err, tracked.Value())
		}
	}
	if _, err := tracked.Commit(context.Background(), receivers.WithUpdateHeader("X-Vendor", "kept")); err != nil || calls.Load() != 0 {
		t.Fatal(err, calls.Load())
	}
	for _, name := range []string{"B", "A", "A"} {
		if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName(name)); err != nil {
			t.Fatal(err)
		}
	}
	if !tracked.Dirty() {
		t.Fatal("reversion cleared sticky dirty state")
	}
	value, err := tracked.Commit(context.Background(), receivers.WithUpdateHeader("X-Vendor", "kept"))
	if err != nil || tracked.Dirty() || value.ID != "fixed" || value.Name != "Server" || value.Type != "message" || len(value.Params) != 1 || string(value.Params["a"]) != "3" || value.Header.Get("Location") != "https://foreign.invalid/not-an-action#trace" {
		t.Fatal(value, err)
	}
	response := tracked.Response()
	if response.ID != "" || len(response.Body) != 2 || response.StatusCode != 200 || response.Header.Get("X-Request-ID") != "last-patch" {
		t.Fatal("actual response invented cached fields", response)
	}
	if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("B"), receivers.WithUpdateParams(map[string]any{"big": json.Number("9007199254740993"), "off": false}), receivers.WithUpdateField("vendor", false)); err != nil {
		t.Fatal(err)
	}
	value, err = tracked.Commit(context.Background(), receivers.WithUpdateHeader("X-Vendor", "kept"))
	if err != nil || tracked.Dirty() || value.Name != "B" || string(value.Params["big"]) != "9007199254740993" || string(value.Body["vendor"]) != "false" || len(tracked.Response().Body) != 1 || tracked.Response().Name != "" || calls.Load() != 2 {
		t.Fatal("omitted response fields lost submitted cache", value, err, tracked.Response(), calls.Load())
	}
}

func TestClusteringReceiverTrackedEmptyNullAndRemovalPresence(t *testing.T) {
	for _, mode := range []string{"empty", "null", "removal", "absent-null"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			seed := receiverTrackedSeed(t, "fixed")
			if mode == "absent-null" {
				delete(seed.Body, "name")
				delete(seed.Body, "action")
				delete(seed.Body, "params")
			}
			tracked, err := receivers.New(cloud.Client("clustering", "/v1")).Track(seed)
			if err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("PATCH /v1/receivers/fixed", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				fields := receiverTrackedBody(t, r)
				wantName, wantAction, wantParams := "null", "null", "null"
				if mode == "empty" {
					wantName, wantAction, wantParams = `""`, `""`, "{}"
				}
				if len(fields) != 3 || string(fields["name"]) != wantName || string(fields["action"]) != wantAction || string(fields["params"]) != wantParams {
					t.Error(fields)
				}
				testcloud.JSON(w, 200, `{"receiver":{}}`)
			})
			if mode == "removal" || mode == "absent-null" {
				for _, remove := range []func() error{tracked.RemoveName, tracked.RemoveAction, tracked.RemoveParams} {
					if err := remove(); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "absent-null" && tracked.Dirty() {
					t.Fatal("absent removal created dirty state")
				}
			}
			if mode == "empty" {
				err = tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName(""), receivers.WithUpdateAction(""), receivers.WithUpdateParams(map[string]any{}))
			} else if mode != "removal" {
				err = tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateNameNull(), receivers.WithUpdateActionNull(), receivers.WithUpdateParams(nil))
			}
			if err != nil || !tracked.Dirty() {
				t.Fatal(err, tracked.Value())
			}
			if _, err := tracked.Commit(context.Background()); err != nil || tracked.Dirty() || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
			for _, key := range []string{"name", "action", "params"} {
				_, present := tracked.Value().Body[key]
				if present == (mode == "removal") {
					t.Fatal(mode, key, tracked.Value().Body)
				}
			}
		})
	}
}

func TestClusteringReceiverTrackedAtomicEditAndHeaderOnlyCommitProtection(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	tracked := receiverTrackedHandle(t, cloud.Client("clustering", "/v1"))
	invalid := []receivers.UpdateOption{
		nil,
		receivers.WithUpdateParams([]any{}),
		receivers.WithUpdateHeader("X-Vendor", "value"),
		request.WithQuery[receivers.UpdateOpts]("vendor", "value"),
		request.WithArgument[receivers.UpdateOpts]("foreign", true),
	}
	for _, key := range []string{"id", "ID", "Name", "ACTION", "PARAMS", "type", "TYPE", "cluster_id", "CLUSTER_ID", "actor", "channel", "USER_ID", "created_at", "links"} {
		invalid = append(invalid, receivers.WithUpdateField(key, false))
	}
	for index, option := range invalid {
		if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("WouldApply"), option); !errors.Is(err, resource.ErrInvalidOption) || tracked.Dirty() || tracked.Value().Name != "A" {
			t.Fatalf("option %d partly applied: %v, %#v", index, err, tracked.Value())
		}
	}
	if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("WouldApply"), func(config *request.Config[receivers.UpdateOpts]) error {
		config.Fields["vendor"] = json.RawMessage(`{`)
		return nil
	}); err == nil || tracked.Dirty() || tracked.Value().Name != "A" {
		t.Fatal("invalid raw JSON partly applied", err, tracked.Value())
	}
	if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("Pending")); err != nil {
		t.Fatal(err)
	}
	invalidCommit := []receivers.UpdateOption{nil, receivers.WithUpdateNameNull(), receivers.WithUpdateAction(""), receivers.WithUpdateParams(nil), receivers.WithUpdateField("vendor", false), request.WithQuery[receivers.UpdateOpts]("vendor", "value"), request.WithArgument[receivers.UpdateOpts]("foreign", true), receivers.WithUpdateHeader("X-Auth-Token", "override"), receivers.WithUpdateHeader("OpenStack-API-Version", "clustering 1.4")}
	for index, option := range invalidCommit {
		if _, err := tracked.Commit(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) || !tracked.Dirty() || tracked.Value().Name != "Pending" || calls.Load() != 0 {
			t.Fatalf("commit option %d: %v, %#v, calls=%d", index, err, tracked.Value(), calls.Load())
		}
	}
}

func TestClusteringReceiverTrackedOwnsInputOptionsMapsPointersAndHTTPViews(t *testing.T) {
	cloud := testcloud.New(t)
	seed := receiverTrackedSeed(t, "fixed")
	tracked, err := receivers.New(cloud.Client("clustering", "/v1")).Track(seed)
	if err != nil {
		t.Fatal(err)
	}
	seed.Body["id"] = json.RawMessage(`"retarget"`)
	seed.Header.Set("X-Request-ID", "caller")
	*seed.Action, *seed.ClusterID = "caller", "caller"
	seed.Actor["large"][0] = '0'
	seed.Params["a"] = json.RawMessage("0")
	seed.Channel["alarm_url"][1] = 'X'
	for _, view := range []*receivers.Receiver{tracked.Value(), tracked.Response()} {
		if view.ID != "fixed" || *view.Action != "CLUSTER_SCALE_OUT" || *view.ClusterID != "cluster" || view.Header.Get("X-Request-ID") != "seed" || string(view.Actor["large"]) != "9007199254740993" || string(view.Params["a"]) != "1" || string(view.Channel["alarm_url"]) != `"https://foreign.invalid/trigger"` {
			t.Fatal(view)
		}
		view.Body["params"][2] = 'Y'
		view.Header.Set("X-Request-ID", "view")
		*view.Action, *view.ClusterID = "view", "view"
		view.Actor["large"][0] = '1'
		view.Params["a"] = json.RawMessage("9")
		view.Channel["alarm_url"][1] = 'Y'
	}
	if value := tracked.Value(); *value.Action != "CLUSTER_SCALE_OUT" || value.Header.Get("X-Request-ID") != "seed" || string(value.Params["a"]) != "1" || string(value.Actor["large"]) != "9007199254740993" {
		t.Fatal("getter storage leaked", value)
	}
	raw := json.RawMessage(`{"big":9007199254740993,"off":false}`)
	option := receivers.WithUpdateOptions(receivers.UpdateOpts{Action: request.Present(""), Params: raw})
	raw[2] = 'Z'
	if err := tracked.Edit(receivers.UpdateOpts{}, option); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	cloud.Mux.HandleFunc("PATCH /v1/receivers/fixed", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fields := receiverTrackedBody(t, r)
		if len(fields) != 2 || string(fields["action"]) != `""` || string(fields["params"]) != `{"big":9007199254740993,"off":false}` {
			t.Error(fields)
		}
		w.Header().Set("X-Request-ID", "actual")
		testcloud.JSON(w, 200, `{"receiver":{"id":"fixed","params":{"big":9007199254740995}}}`)
	})
	if _, err := tracked.Commit(context.Background()); err != nil || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
	actual := tracked.Response()
	actual.Body["params"][2] = 'Q'
	actual.Header.Set("X-Request-ID", "outside")
	actual.Params["big"][0] = '0'
	if value := tracked.Response(); value.Header.Get("X-Request-ID") != "actual" || string(value.Params["big"]) != "9007199254740995" || string(value.Body["params"]) != `{"big":9007199254740995}` {
		t.Fatal("last successful response storage leaked", value)
	}
}

func TestClusteringReceiverTrackedCanonicalNameAllPagesAndExplicitFixedID(t *testing.T) {
	for _, mode := range []string{"name", "missing", "duplicate", "changed-id", "null-id", "omitted-id"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets, patches atomic.Int32
			fixedID := "canonical"
			if strings.HasSuffix(mode, "-id") {
				fixedID = "request-id"
			}
			cloud.Mux.HandleFunc("GET /v1/receivers", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "A" || r.URL.Query().Has("user") {
					t.Error(r.URL)
				}
				if r.URL.Query().Get("marker") == "" {
					w.Header().Set("Link", `</v1/receivers?name=A&marker=next>; rel="next"`)
					body := `{"receivers":[{"id":"decoy","Name":"A"}]}`
					if mode == "duplicate" {
						body = `{"receivers":[{"id":"first","name":"A"}]}`
					}
					testcloud.JSON(w, 200, body)
				} else if mode == "missing" {
					testcloud.JSON(w, 200, `{"receivers":[{"id":"other","name":"Elsewhere"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"receivers":[{"id":"canonical","ID":"shadow","name":"A","Name":"shadow","user":"cached-owner"}]}`)
				}
			})
			cloud.Mux.HandleFunc("GET /v1/receivers/"+fixedID, func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				fields := `"id":"changed-id","name":"A"`
				if mode == "null-id" {
					fields = `"id":null,"ID":"shadow","name":"A"`
				} else if mode == "omitted-id" {
					fields = `"ID":"shadow","name":"A"`
				}
				testcloud.JSON(w, 200, `{"receiver":{`+fields+`}}`)
			})
			cloud.Mux.HandleFunc("PATCH /v1/receivers/"+fixedID, func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				testcloud.JSON(w, 200, `{"receiver":{"id":"response-retarget","name":"Accepted"}}`)
			})
			api := receivers.New(cloud.Client("clustering", "/v1"))
			ref := resource.Name("A")
			if fixedID == "request-id" {
				ref = resource.ID(fixedID)
			}
			tracked, err := api.Load(context.Background(), ref)
			if mode == "missing" || mode == "duplicate" {
				want := resource.ErrNotFound
				if mode == "duplicate" {
					want = resource.ErrAmbiguous
				}
				if !errors.Is(err, want) || tracked != nil || lists.Load() != 2 || gets.Load() != 0 || patches.Load() != 0 {
					t.Fatal(tracked, err, lists.Load(), gets.Load(), patches.Load())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tracked.Value().Body["id"] = json.RawMessage(`"caller-retarget"`)
			if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("Requested")); err != nil {
				t.Fatal(err)
			}
			if _, err := tracked.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}
			if _, err := tracked.Refresh(context.Background()); err != nil || patches.Load() != 1 {
				t.Fatal(err, patches.Load())
			}
			if mode == "name" && (lists.Load() != 2 || gets.Load() != 1) || mode != "name" && (lists.Load() != 0 || gets.Load() != 2) {
				t.Fatal("lookup repeated or retargeted", lists.Load(), gets.Load())
			}
		})
	}
}

func TestClusteringReceiverTrackedNativeAndMalformedResponseEvidenceNoResend(t *testing.T) {
	for _, mode := range []string{"native-404", "native-409", "unexpected-202", "empty", "array", "wrong-envelope", "invalid-id", "invalid-name"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			status, body := 200, `{"receiver":{}}`
			switch mode {
			case "native-404":
				status, body = 404, `{"error":"missing"}`
			case "native-409":
				status, body = 409, `{"error":"conflict"}`
			case "unexpected-202":
				status = 202
			case "empty":
				body = ""
			case "array":
				body = `{"receiver":[]}`
			case "wrong-envelope":
				body = `{"wrong":{"id":"fixed"}}`
			case "invalid-id":
				body = `{"receiver":{"id":"bad/id","name":"Server"}}`
			case "invalid-name":
				body = `{"receiver":{"id":"fixed","name":false}}`
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("PATCH /v1/receivers/fixed", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-ID", "failed-response")
				w.Header().Set("Location", "https://foreign.invalid/ignored")
				testcloud.JSON(w, status, body)
			})
			tracked := receiverTrackedHandle(t, cloud.Client("clustering", "/v1"))
			if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("Pending")); err != nil {
				t.Fatal(err)
			}
			_, err := tracked.Commit(context.Background())
			if err == nil || calls.Load() != 1 || !tracked.Dirty() || tracked.Value().Name != "Pending" || tracked.Response().Name != "A" || tracked.Response().Header.Get("X-Request-ID") != "seed" {
				t.Fatal(err, calls.Load(), tracked.Value(), tracked.Response())
			}
			var accepted *resource.ResponseError
			if status != 200 {
				if !gophercloud.ResponseCodeIs(err, status) || errors.As(err, &accepted) {
					t.Fatal("native error masked", err)
				}
			} else if !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Request-ID") != "failed-response" || accepted.Cause == nil {
				t.Fatal("accepted evidence lost", err, accepted)
			}
		})
	}
}

func TestClusteringReceiverTrackedRefreshResetsOlderDirtyAndPreservesFailures(t *testing.T) {
	cloud := testcloud.New(t)
	var mode atomic.Int32
	var gets, patches atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/receivers/fixed", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		switch mode.Load() {
		case 0:
			w.Header().Set("X-Request-ID", "refresh")
			w.Header().Set("Location", "https://foreign.invalid/ignored")
			testcloud.JSON(w, 200, `{"receiver":{"id":null,"ID":"shadow","name":null,"Name":"shadow","action":null,"params":{"observed":9007199254740993}}}`)
		case 1:
			testcloud.JSON(w, 404, `{"error":"missing"}`)
		case 2:
			w.Header().Set("X-Request-ID", "bad-refresh")
			testcloud.JSON(w, 200, `{"receiver":{"id":false}}`)
		}
	})
	cloud.Mux.HandleFunc("PATCH /v1/receivers/fixed", func(w http.ResponseWriter, r *http.Request) { patches.Add(1); w.WriteHeader(500) })
	tracked := receiverTrackedHandle(t, cloud.Client("clustering", "/v1"))
	if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("Discarded"), receivers.WithUpdateParams(map[string]any{"pending": true})); err != nil {
		t.Fatal(err)
	}
	value, err := tracked.Refresh(context.Background())
	if err != nil || tracked.Dirty() || value.ID != "" || value.Name != "" || value.Action != nil || len(value.Params) != 1 || string(value.Params["observed"]) != "9007199254740993" || value.Header.Get("X-Request-ID") != "refresh" {
		t.Fatal(value, err)
	}
	if _, err := tracked.Commit(context.Background()); err != nil || patches.Load() != 0 {
		t.Fatal(err, patches.Load())
	}
	if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("Keep")); err != nil {
		t.Fatal(err)
	}
	mode.Store(1)
	if _, err := tracked.Refresh(context.Background()); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) || !tracked.Dirty() || tracked.Value().Name != "Keep" || tracked.Response().Header.Get("X-Request-ID") != "refresh" {
		t.Fatal(err, tracked.Value(), tracked.Response())
	}
	mode.Store(2)
	_, err = tracked.Refresh(context.Background())
	var accepted *resource.ResponseError
	if !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != `{"receiver":{"id":false}}` || accepted.Header.Get("X-Request-ID") != "bad-refresh" || !tracked.Dirty() || tracked.Value().Name != "Keep" || tracked.Response().Header.Get("X-Request-ID") != "refresh" || gets.Load() != 3 || patches.Load() != 0 {
		t.Fatal(err, accepted, gets.Load(), patches.Load())
	}
}

func TestClusteringReceiverTrackedNewEditsSurviveCommitAndRefreshInFlight(t *testing.T) {
	for _, operation := range []string{"Commit", "Refresh"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			started, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			defer once.Do(func() { close(release) })
			var patches atomic.Int32
			write := func(w http.ResponseWriter, r *http.Request) {
				close(started)
				<-release
				w.Header().Set("X-Request-ID", "observed")
				testcloud.JSON(w, 200, `{"receiver":{"name":"ServerOld","action":"SERVER_OLD","params":{"server":true}}}`)
			}
			cloud.Mux.HandleFunc("GET /v1/receivers/fixed", write)
			cloud.Mux.HandleFunc("PATCH /v1/receivers/fixed", func(w http.ResponseWriter, r *http.Request) {
				fields := receiverTrackedBody(t, r)
				call := patches.Add(1)
				if operation == "Commit" && call == 1 {
					if len(fields) != 2 || string(fields["name"]) != `"Submitted"` || string(fields["params"]) != `{"submitted":true}` {
						t.Error(fields)
					}
					write(w, r)
					return
				}
				if len(fields) != 2 || string(fields["name"]) != `"Newer"` || string(fields["action"]) != `""` {
					t.Error("new revisions were cleared or overwritten", fields)
				}
				testcloud.JSON(w, 200, `{"receiver":{}}`)
			})
			tracked := receiverTrackedHandle(t, cloud.Client("clustering", "/v1"))
			if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("Submitted"), receivers.WithUpdateParams(map[string]any{"submitted": true})); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				var err error
				if operation == "Commit" {
					_, err = tracked.Commit(context.Background())
				} else {
					_, err = tracked.Refresh(context.Background())
				}
				done <- err
			}()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("HTTP did not start")
			}
			if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("Newer"), receivers.WithUpdateAction("")); err != nil {
				t.Fatal(err)
			}
			once.Do(func() { close(release) })
			if err := <-done; err != nil || !tracked.Dirty() || tracked.Value().Name != "Newer" || tracked.Value().Action == nil || *tracked.Value().Action != "" || tracked.Response().Name != "ServerOld" || string(tracked.Value().Params["server"]) != "true" {
				t.Fatal(err, tracked.Value(), tracked.Response())
			}
			if _, err := tracked.Commit(context.Background()); err != nil || tracked.Dirty() {
				t.Fatal(err, tracked.Value())
			}
			want := int32(1)
			if operation == "Commit" {
				want = 2
			}
			if patches.Load() != want {
				t.Fatal(patches.Load())
			}
		})
	}
}

func TestClusteringReceiverUpdateScopeRoutesSnapshotsLiveTokensAndIsolation(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/catalog/v1")
	wirePrefix, decodedPrefix := "/reverse/team%20alpha/senlin/v1", "/reverse/team alpha/senlin/v1"
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + wirePrefix)
	client.Microversion = "1.0"
	client.MoreHeaders = map[string]string{"X-Configured": "kept"}
	endpoint, resourceBase, headers := client.Endpoint, client.ResourceBase, maps.Clone(client.MoreHeaders)
	api := receivers.New(client)
	leftPath, rightPath := "tenants/租户/receivers", "alternate/receivers"
	left, err := api.AtBasePath(leftPath)
	if err != nil {
		t.Fatal(err)
	}
	right, err := api.AtBasePath(rightPath)
	if err != nil {
		t.Fatal(err)
	}
	var gets, lists, patches atomic.Int32
	var captured *request.Config[receivers.UpdateOpts]
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Configured") != "kept" || r.Header.Get("OpenStack-API-Version") != "clustering 1.0" {
			t.Error(r.Header)
		}
		if strings.Contains(r.URL.Path, "/actions/") || strings.Contains(r.URL.Path, "/trigger") {
			t.Error("followed receiver response data", r.URL)
		}
		w.Header().Set("Location", "https://foreign.invalid/actions/ignored#trace")
		w.Header().Set("X-Request-ID", "scope")
		if r.Method == http.MethodGet {
			if r.Header.Get("X-Auth-Token") != "fresh-token" {
				t.Error(r.Header)
			}
			if r.URL.Path == decodedPrefix+"/"+rightPath {
				lists.Add(1)
				if r.URL.Query().Get("name") != "A" || r.URL.Query().Has("user") {
					t.Error(r.URL)
				}
				if r.URL.Query().Get("marker") == "" {
					if captured != nil {
						captured.Options.Name = request.Present("Altered")
						captured.Options.Params[2] = 'X'
						captured.Headers["X-Snapshot"] = "altered"
					}
					w.Header().Set("Link", fmt.Sprintf(`<%s/%s?name=A&marker=next>; rel="next"`, wirePrefix, rightPath))
					testcloud.JSON(w, 200, `{"receivers":[{"id":"decoy","name":"Elsewhere"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"receivers":[{"id":"canonical","name":"A","user":"cached-user"}]}`)
				}
				return
			}
			gets.Add(1)
			if r.URL.Path != decodedPrefix+"/"+leftPath+"/fixed" && r.URL.Path != decodedPrefix+"/"+rightPath+"/request-id" {
				t.Error("unexpected detail GET", r.URL)
			}
			testcloud.JSON(w, 200, `{"receiver":{"id":null,"ID":"shadow","name":"A","channel":{"alarm_url":"https://foreign.invalid/trigger"}}}`)
			return
		}
		if r.Method != http.MethodPatch || r.URL.RawQuery != "" {
			t.Error(r.Method, r.URL)
		}
		fields := receiverTrackedBody(t, r)
		count := patches.Add(1)
		if count == 1 && r.Header.Get("X-Auth-Token") != "test-token" || count > 1 && r.Header.Get("X-Auth-Token") != "fresh-token" {
			t.Error(r.Header)
		}
		valid := false
		for _, target := range []string{leftPath + "/fixed", rightPath + "/request-id", rightPath + "/canonical", "receivers/base-id", leftPath + "/concurrent", rightPath + "/concurrent"} {
			if r.URL.Path == decodedPrefix+"/"+target {
				valid = true
				parts := strings.Split(target, "/")
				for i := range parts {
					parts[i] = url.PathEscape(parts[i])
				}
				if r.URL.EscapedPath() != wirePrefix+"/"+strings.Join(parts, "/") {
					t.Error("route escaped twice", r.URL.EscapedPath())
				}
			}
		}
		if !valid {
			t.Error("unselected PATCH route", r.URL)
		}
		if count == 3 && (len(fields) != 3 || string(fields["name"]) != "null" || string(fields["action"]) != `""` || string(fields["params"]) != `{"large":9007199254740993}` || r.Header.Get("X-Snapshot") != "captured") {
			t.Error("name resolution changed prepared request", fields, r.Header)
		}
		testcloud.JSON(w, 200, `{"receiver":{"id":"changed-response","action":"CLUSTER_SCALE_IN","channel":{"alarm_url":"https://foreign.invalid/trigger"}}}`)
	})
	tracked, err := left.Track(receiverTrackedSeed(t, "fixed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tracked.Commit(context.Background()); err != nil || patches.Load() != 0 {
		t.Fatal(err, patches.Load())
	}
	if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateParams(nil)); err != nil {
		t.Fatal(err)
	}
	if value, err := tracked.Commit(context.Background()); err != nil || value.Header.Get("Location") != "https://foreign.invalid/actions/ignored#trace" || *value.Action != "CLUSTER_SCALE_IN" {
		t.Fatal(value, err)
	}
	cloud.Provider.SetToken("fresh-token")
	if _, err := tracked.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	loaded, err := right.Load(context.Background(), resource.ID("request-id"))
	if err != nil {
		t.Fatal(err)
	}
	if err := loaded.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("Loaded")); err != nil {
		t.Fatal(err)
	}
	if _, err := loaded.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err = right.Update(context.Background(), resource.Name("A"), receivers.UpdateOpts{}, receivers.WithUpdateNameNull(), receivers.WithUpdateAction(""), receivers.WithUpdateParams(map[string]json.Number{"large": "9007199254740993"}), receivers.WithUpdateHeader("X-Snapshot", "captured"), func(config *request.Config[receivers.UpdateOpts]) error { captured = config; return nil })
	if err != nil {
		t.Fatal(err)
	}
	captured = nil
	byName, err := right.Load(context.Background(), resource.Name("A"))
	if err != nil {
		t.Fatal(err)
	}
	if err := byName.Edit(receivers.UpdateOpts{}, receivers.WithUpdateAction("")); err != nil {
		t.Fatal(err)
	}
	if _, err := byName.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(context.Background(), resource.ID("base-id"), receivers.UpdateOpts{}, receivers.WithUpdateName("Default")); err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	for _, scope := range []*receivers.UpdateScope{left, right} {
		wait.Add(1)
		go func(scope *receivers.UpdateScope) {
			defer wait.Done()
			if _, err := scope.Update(context.Background(), resource.ID("concurrent"), receivers.UpdateOpts{}, receivers.WithUpdateAction("CLUSTER_SCALE_OUT")); err != nil {
				t.Error(err)
			}
		}(scope)
	}
	wait.Wait()
	if gets.Load() != 2 || lists.Load() != 4 || patches.Load() != 7 || client.Endpoint != endpoint || client.ResourceBase != resourceBase || !reflect.DeepEqual(client.MoreHeaders, headers) {
		t.Fatal(gets.Load(), lists.Load(), patches.Load(), client)
	}
}

func TestClusteringReceiverTrackedAndUpdateScopeOfflineSourceAndCleanValidation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("clustering", "/v1")
	api := receivers.New(client)
	tracked := receiverTrackedHandle(t, client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tracked.Commit(ctx); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal(err, calls.Load())
	}
	if _, err := tracked.Refresh(ctx); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal(err, calls.Load())
	}
	if _, err := tracked.Commit(context.Background(), receivers.WithUpdateHeader("X-Auth-Token", "override")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, version := range []string{"latest", "2.0", "1.01"} {
		client.Microversion = version
		if _, err := tracked.Commit(context.Background()); err == nil || calls.Load() != 0 {
			t.Fatal(version, err, calls.Load())
		}
	}
	client.Microversion = "1.0"
	client.MoreHeaders = map[string]string{"openstack-api-version": "clustering 1.4"}
	if _, err := tracked.Commit(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	client.MoreHeaders = nil
	if _, err := tracked.Commit(context.Background()); err != nil || calls.Load() != 0 {
		t.Fatal("cached user invented a 1.4 query gate", err, calls.Load())
	}
	client.Type = "compute"
	if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateName("Offline")); err != nil {
		t.Fatal("offline edit validated source", err)
	}
	if _, err := tracked.Commit(context.Background()); !errors.Is(err, resource.ErrInvalidOption) || !tracked.Dirty() {
		t.Fatal(err)
	}
	client.Type = "clustering"
	for _, path := range []string{"", "/receivers", "a/../receivers", "a%2freceivers", "%(id)s/receivers", "receivers?limit=1"} {
		if _, err := api.AtBasePath(path); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(path, err)
		}
	}
	for _, source := range []*gophercloud.ServiceClient{nil, {Type: "clustering", Endpoint: client.Endpoint}} {
		if _, err := receivers.New(source).AtBasePath("receivers"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	var nilAPI *receivers.API
	if _, err := nilAPI.AtBasePath("receivers"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	var nilScope *receivers.UpdateScope
	if _, err := nilScope.Update(context.Background(), resource.ID("fixed"), receivers.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := nilScope.Load(context.Background(), resource.ID("fixed")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := nilScope.Track(receiverTrackedSeed(t, "fixed")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	var nilHandle *receivers.TrackedReceiver
	if nilHandle.Value() != nil || nilHandle.Response() != nil || nilHandle.Dirty() {
		t.Fatal("nil handle invented state")
	}
	for _, mutate := range []func() error{nilHandle.RemoveName, nilHandle.RemoveAction, nilHandle.RemoveParams, func() error { return nilHandle.Edit(receivers.UpdateOpts{}) }} {
		if err := mutate(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := nilHandle.Commit(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := nilHandle.Refresh(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.Track(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	manual, err := receivers.New(nil).Track(&receivers.Receiver{ID: "manual", Name: "Local"})
	if err != nil || manual.Value().StatusCode != 0 || manual.Value().Header != nil {
		t.Fatal(manual, err)
	}
	if _, err := manual.Commit(context.Background()); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal(err, calls.Load())
	}
	t.Run("source-rechecked-after-option-and-name-lookup", func(t *testing.T) {
		lookupCloud := testcloud.New(t)
		lookupClient := lookupCloud.Client("clustering", "/v1")
		scope, err := receivers.New(lookupClient).AtBasePath("alternate/receivers")
		if err != nil {
			t.Fatal(err)
		}
		var reads, patches atomic.Int32
		lookupCloud.Mux.HandleFunc("GET /v1/alternate/receivers", func(w http.ResponseWriter, r *http.Request) {
			reads.Add(1)
			lookupClient.Type = "compute"
			testcloud.JSON(w, 200, `{"receivers":[{"id":"canonical","name":"A"}]}`)
		})
		lookupCloud.Mux.HandleFunc("PATCH /v1/alternate/receivers/canonical", func(w http.ResponseWriter, r *http.Request) { patches.Add(1); w.WriteHeader(500) })
		_, err = scope.Update(context.Background(), resource.ID("canonical"), receivers.UpdateOpts{}, receivers.WithUpdateName("Updated"), func(config *request.Config[receivers.UpdateOpts]) error { lookupClient.Type = "compute"; return nil })
		if !errors.Is(err, resource.ErrInvalidOption) || reads.Load() != 0 || patches.Load() != 0 {
			t.Fatal(err, reads.Load(), patches.Load())
		}
		lookupClient.Type = "clustering"
		_, err = scope.Update(context.Background(), resource.Name("A"), receivers.UpdateOpts{}, receivers.WithUpdateName("Updated"))
		if !errors.Is(err, resource.ErrInvalidOption) || reads.Load() != 1 || patches.Load() != 0 {
			t.Fatal(err, reads.Load(), patches.Load())
		}
	})
}
