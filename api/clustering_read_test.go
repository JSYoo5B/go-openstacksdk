package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/events"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestClusteringReadActionNumericTimesAndResponseIdentity(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /senlin/v1/actions/request-id", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "test-token" || r.URL.RawQuery != "" {
			t.Error(r.URL, r.Header)
		}
		w.Header().Set("X-Request-Id", "action-request")
		testcloud.JSON(w, 200, `{"action":{"id":"response-id","name":"create_node","target":"node-id","cluster_id":"cluster-id","action":"NODE_CREATE","status":"SUCCEEDED","owner":null,"interval":-1,"start_time":1453414055.48672001,"end_time":null,"timeout":9007199254740993,"inputs":{"capacity":9007199254740993},"outputs":{},"data":{"optional":null},"depends_on":["before"],"depended_by":[],"created_at":"2015-10-10T12:46:36.000000","updated_at":null,"vendor":false}}`)
	})
	value, err := actions.New(cloud.Client("clustering", "/senlin/v1")).Get(context.Background(), "request-id")
	if err != nil || value.ID != "response-id" || value.TargetID != "node-id" || value.ClusterID == nil || *value.ClusterID != "cluster-id" || value.OwnerID != nil || value.Interval == nil || value.Interval.String() != "-1" || value.StartAt == nil || value.StartAt.String() != "1453414055.48672001" || value.EndAt != nil || value.Timeout == nil || value.Timeout.String() != "9007199254740993" {
		t.Fatal(value, err)
	}
	if value.CreatedAt == nil || *value.CreatedAt != "2015-10-10T12:46:36.000000" || value.UpdatedAt != nil || string(value.Inputs["capacity"]) != "9007199254740993" || string(value.Data["optional"]) != "null" || string(value.Body["vendor"]) != "false" || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "action-request" {
		t.Fatal(value)
	}
}

func TestClusteringReadShortPagesContinueUsingUnchangedWireIDs(t *testing.T) {
	for _, plural := range []string{"actions", "events"} {
		t.Run(plural, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/"+plural, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("limit") != "5" {
					t.Error(r.URL)
				}
				switch calls.Add(1) {
				case 1:
					if r.URL.Query().Has("marker") {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{"`+plural+`":[{"id":"wire-id"}]}`)
				case 2:
					if r.URL.Query().Get("marker") != "wire-id" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{"`+plural+`":[]}`)
				default:
					t.Error("empty page did not stop", calls.Load())
				}
			})
			client := cloud.Client("clustering", "/senlin/v1")
			rows := 0
			if plural == "actions" {
				for value, err := range actions.New(client).List(context.Background(), actions.WithListOptions(actions.ListOpts{Limit: 5})) {
					if err != nil {
						t.Fatal(err)
					}
					rows++
					value.ID = "consumer mutation"
				}
			} else {
				for value, err := range events.New(client).List(context.Background(), events.WithListOptions(events.ListOpts{Limit: 5})) {
					if err != nil {
						t.Fatal(err)
					}
					rows++
					value.ID = "consumer mutation"
				}
			}
			if rows != 1 || calls.Load() != 2 {
				t.Fatalf("rows=%d requests=%d", rows, calls.Load())
			}
		})
	}
}

func TestClusteringReadActionListMarkersDefaultsAndOptionsSnapshot(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/actions", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "1" || q.Get("name") != "selected" || q.Get("target") != "node-id" || q.Get("action") != "NODE_CREATE" || q.Get("status") != "SUCCEEDED" || q.Get("sort") != "created_at:desc,name" || q.Get("global_project") != "true" || q.Get("vendor") != "retained" {
			t.Error(q)
		}
		switch calls.Add(1) {
		case 1:
			if q.Has("marker") {
				t.Error(q)
			}
			w.Header().Set("X-Request-Id", "action-page1")
			testcloud.JSON(w, 200, `{"actions":[{"id":"action-1","name":"selected","target":"different-target","status":"SUCCEEDED"}]}`)
		case 2:
			if q.Get("marker") != "action-1" {
				t.Error(q)
			}
			w.Header().Set("X-Request-Id", "action-page2")
			testcloud.JSON(w, 200, `{"actions":[{"id":"action-2","name":"selected","status":"SUCCEEDED"}]}`)
		case 3:
			if q.Get("marker") != "action-2" {
				t.Error(q)
			}
			testcloud.JSON(w, 200, `{"actions":[]}`)
		default:
			t.Error("unexpected list request", calls.Load())
		}
	})
	global := true
	option := actions.WithListOptions(actions.ListOpts{Limit: 1, Name: "selected", TargetID: "node-id", Action: "NODE_CREATE", Status: "SUCCEEDED", Sort: "created_at:desc,name", GlobalProject: &global})
	global = false
	var applied *bool
	capture := func(config *request.Config[actions.ListOpts]) error {
		applied = config.Options.GlobalProject
		return nil
	}
	api := actions.New(cloud.Client("clustering", "/senlin/v1"))
	options := []actions.ListOption{option, capture, actions.WithListQuery("vendor", "retained")}
	iterator := api.List(context.Background(), options...)
	options[0] = actions.WithListGlobalProject(false)
	if calls.Load() != 0 {
		t.Fatal("eager iterator")
	}
	for range 2 {
		var values []*actions.Action
		for value, err := range iterator {
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, value)
		}
		if len(values) != 2 || values[0].ID != "action-1" || values[1].ID != "action-2" || values[0].Header.Get("X-Request-Id") != "action-page1" || values[1].Header.Get("X-Request-Id") != "action-page2" || calls.Load() != 3 || applied == nil {
			t.Fatal(values, calls.Load(), applied)
		}
		*applied = false
		calls.Store(0)
	}
}

func TestClusteringReadEventRepeatedFiltersSnapshotAndExactBody(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/events", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if !reflect.DeepEqual(q["oid"], []string{"node-1", "node-2"}) || !reflect.DeepEqual(q["oname"], []string{"first", "second"}) || !reflect.DeepEqual(q["otype"], []string{"NODE", "CLUSTER"}) || !reflect.DeepEqual(q["action"], []string{"create", "delete"}) || q.Get("global_project") != "false" || q.Get("cluster_id") != "cluster-name" || q.Get("level") != "20" || q.Get("sort") != "timestamp:desc" || q.Get("vendor") != "retained" {
			t.Error(q)
		}
		if calls.Add(1) == 1 {
			w.Header().Set("X-Request-Id", "event-page")
			testcloud.JSON(w, 200, `{"events":[{"id":"event-1","oid":"response-node","oname":"response-name","otype":"NODE","cluster_id":null,"timestamp":"2016-10-10T12:46:36.000000","level":"20","status":"START","meta_data":{"number":9007199254740993},"future":null}],"links":[{"rel":"next","href":"?marker=event-1"}]}`)
		} else {
			if q.Get("marker") != "event-1" {
				t.Error(q)
			}
			testcloud.JSON(w, 200, `{"events":[]}`)
		}
	})
	global := false
	input := events.ListOpts{ObjectIDs: []string{"node-1", "node-2"}, ObjectNames: []string{"first", "second"}, ObjectTypes: []string{"NODE", "CLUSTER"}, Actions: []string{"create", "delete"}, ClusterID: "cluster-name", Level: "20", Sort: "timestamp:desc", GlobalProject: &global}
	option := events.WithListOptions(input)
	input.ObjectIDs[0], input.ObjectNames[0], input.ObjectTypes[0], input.Actions[0] = "changed", "changed", "changed", "changed"
	global = true
	values, err := events.New(cloud.Client("clustering", "/senlin/v1")).All(context.Background(), option, events.WithListQuery("vendor", "retained"))
	if err != nil || len(values) != 1 || calls.Load() != 2 {
		t.Fatal(values, err, calls.Load())
	}
	value := values[0]
	if value.ID != "event-1" || value.ObjectID != "response-node" || value.ObjectName != "response-name" || value.Level != "20" || value.ClusterID != nil || value.GeneratedAt == nil || *value.GeneratedAt != "2016-10-10T12:46:36.000000" || string(value.MetaData) != `{"number":9007199254740993}` || string(value.Body["future"]) != "null" || value.Header.Get("X-Request-Id") != "event-page" {
		t.Fatal(value)
	}
}

func TestClusteringReadServicesVersionGateAndLocalStatusFilter(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/services", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "clustering 1.7" || r.URL.Query().Has("status") {
			t.Error(r.Header, r.URL)
		}
		testcloud.JSON(w, 200, `{"services":[{"id":"engine-1","binary":"senlin-engine","topic":"senlin-engine","host":"host-1","status":"enabled","state":"up","disabled_reason":null,"updated_at":"2017-04-24T07:43:12","future":9007199254740993},{"id":"engine-2","status":"disabled"}]}`)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	api := services.New(client)
	for _, version := range []string{"", "1.0", "1.6", "latest"} {
		client.Microversion = version
		if _, err := api.All(context.Background()); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
		if _, err := api.Resources.All(context.Background(), resource.WithQuery("vendor", "requested")); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal("shared list bypassed gate", version, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("version gate performed HTTP")
	}
	client.Microversion = "1.7"
	values, err := api.Resources.All(context.Background(), resource.WithStatus("enabled"))
	if err != nil || len(values) != 1 || values[0].ID != "engine-1" || values[0].DisabledReason != nil || values[0].UpdatedAt == nil || *values[0].UpdatedAt != "2017-04-24T07:43:12" || string(values[0].Body["future"]) != "9007199254740993" || values[0].Topic != "senlin-engine" || calls.Load() != 1 {
		t.Fatal(values, err, calls.Load())
	}
	if _, err := api.Resources.Get(context.Background(), "engine-1"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := api.Resources.Wait(context.Background(), resource.ID("engine-1"), "enabled"); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
}

func TestClusteringReadPaginationMissingRowIDCycleAndCancellation(t *testing.T) {
	for _, mode := range []string{"missing-id", "cycle", "cancel", "break"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/actions", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-Id", "pagination-evidence")
				if mode == "missing-id" {
					testcloud.JSON(w, 200, `{"actions":[{"target":"target-is-not-marker"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"actions":[{"id":"same-id"}]}`)
				}
			})
			api := actions.New(cloud.Client("clustering", "/senlin/v1"))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var resultErr error
			for _, err := range api.List(ctx, actions.WithListOptions(actions.ListOpts{Limit: 1})) {
				if err != nil {
					resultErr = err
					break
				}
				if mode == "cancel" {
					cancel()
				} else if mode == "break" {
					break
				}
			}
			switch mode {
			case "missing-id":
				var responseErr *resource.ResponseError
				if !errors.Is(resultErr, resource.ErrInvalidOption) || !errors.As(resultErr, &responseErr) || responseErr.Header.Get("X-Request-Id") != "pagination-evidence" || calls.Load() != 1 {
					t.Fatal(resultErr, calls.Load())
				}
			case "cycle":
				var cycle *resource.PaginationCycleError
				if !errors.As(resultErr, &cycle) || calls.Load() != 2 {
					t.Fatal(resultErr, calls.Load())
				}
			case "cancel":
				if !errors.Is(resultErr, context.Canceled) || calls.Load() != 1 {
					t.Fatal(resultErr, calls.Load())
				}
			case "break":
				if resultErr != nil || calls.Load() != 1 {
					t.Fatal(resultErr, calls.Load())
				}
			}
		})
	}
}

func TestClusteringReadDefaultsCoreExtensionsAndSortValidation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/actions", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Has("limit") || r.URL.Query().Has("marker") || r.URL.Query().Has("global_project") || r.URL.Query().Has("sort") {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"actions":[]}`)
	})
	api := actions.New(cloud.Client("clustering", "/senlin/v1"))
	values, err := api.All(context.Background())
	if err != nil || values == nil || len(values) != 0 || calls.Load() != 1 {
		t.Fatal(values, err, calls.Load())
	}
	for _, option := range []actions.ListOption{
		nil,
		actions.WithListOptions(actions.ListOpts{Sort: "name:sideways"}),
		actions.WithListOptions(actions.ListOpts{Sort: "unknown:asc"}),
		actions.WithListQuery("target", "hidden"),
		actions.WithListQuery("global_project", "true"),
		request.WithField[actions.ListOpts]("field", true),
	} {
		if _, err := api.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.Resources.All(context.Background(), resource.WithQuery("sort", "invalid:asc")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("shared collection bypassed sort validation", err)
	}
	eventAPI := events.New(api.RawClient())
	for _, option := range []events.ListOption{
		events.WithListOptions(events.ListOpts{ObjectIDs: []string{""}}),
		events.WithListQuery("oid", "hidden"),
		events.WithListQuery("obj_name", "hidden"),
		events.WithListOptions(events.ListOpts{Sort: "created_at:asc"}),
	} {
		if _, err := eventAPI.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("invalid options performed HTTP", calls.Load())
	}
}

func TestClusteringReadEventGetHTTPAndDecodeEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /senlin/v1/events/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "event-error")
		switch strings.TrimPrefix(r.URL.Path, "/senlin/v1/events/") {
		case "gone":
			testcloud.JSON(w, 404, `{"error":"missing event"}`)
		case "denied":
			testcloud.JSON(w, 403, `{"error":"denied event"}`)
		case "bad":
			testcloud.JSON(w, 200, `{"event":{"level":true}}`)
		default:
			testcloud.JSON(w, 200, `{"event":{"id":"event-response","level":"20","oid":"node-1","timestamp":"2016-10-10T12:46:36.000000","meta_data":null}}`)
		}
	})
	api := events.New(cloud.Client("clustering", "/senlin/v1"))
	if _, err := api.Get(context.Background(), "gone"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	_, err := api.Get(context.Background(), "denied")
	var statusError gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &statusError) || statusError.Actual != 403 || statusError.ResponseHeader.Get("X-Request-Id") != "event-error" || string(statusError.Body) != `{"error":"denied event"}` {
		t.Fatal(err)
	}
	_, err = api.Get(context.Background(), "bad")
	var responseError *resource.ResponseError
	if !errors.As(err, &responseError) || responseError.StatusCode != 200 || responseError.Header.Get("X-Request-Id") != "event-error" || string(responseError.Body) != `{"event":{"level":true}}` {
		t.Fatal(err)
	}
	value, err := api.Get(context.Background(), "request-id")
	if err != nil || value.ID != "event-response" || value.Level != "20" || string(value.MetaData) != "null" || value.GeneratedAt == nil {
		t.Fatal(value, err)
	}
}

func TestClusteringReadEventLevelsPreserveStringNumberNullAndOmission(t *testing.T) {
	rows := []struct{ id, body, level, raw string }{
		{"string", `{"id":"string","level":"20"}`, "20", `"20"`},
		{"number", `{"id":"number","level":20}`, "20", "20"},
		{"large", `{"id":"large","level":9007199254740993}`, "9007199254740993", "9007199254740993"},
		{"null", `{"id":"null","level":null}`, "", "null"},
		{"omitted", `{"id":"omitted"}`, "", ""},
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		var bodies []string
		for _, row := range rows {
			bodies = append(bodies, row.body)
		}
		testcloud.JSON(w, 200, `{"events":[`+strings.Join(bodies, ",")+`]}`)
	})
	cloud.Mux.HandleFunc("/v1/events/", func(w http.ResponseWriter, r *http.Request) {
		for _, row := range rows {
			if strings.TrimPrefix(r.URL.Path, "/v1/events/") == row.id {
				testcloud.JSON(w, 200, `{"event":`+row.body+`}`)
				return
			}
		}
		http.NotFound(w, r)
	})
	api := events.New(cloud.Client("clustering", "/v1"))
	all, err := api.All(context.Background())
	if err != nil || len(all) != len(rows) {
		t.Fatalf("list: %#v %v", all, err)
	}
	for i, row := range rows {
		value, err := api.Get(context.Background(), row.id)
		if err != nil || value.Level != row.level || string(value.Body["level"]) != row.raw || all[i].Level != row.level || string(all[i].Body["level"]) != row.raw {
			t.Fatalf("%s: get=%#v list=%#v err=%v", row.id, value, all[i], err)
		}
	}
}

func TestClusteringReadServicesFullPageNoFallbackAndCanceledList(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/services", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("limit") != "1" || r.URL.Query().Has("marker") {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"services":[{"id":"engine-1","status":"enabled"}]}`)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.7"
	api := services.New(client)
	values, err := api.All(context.Background(), services.WithListOptions(services.ListOpts{Limit: 1}))
	if err != nil || len(values) != 1 || calls.Load() != 1 {
		t.Fatal(values, err, calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.All(ctx); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
	var nilAction *actions.API
	if _, err := nilAction.Get(context.Background(), "action-id"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	var nilEvent *events.API
	if _, err := nilEvent.All(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
