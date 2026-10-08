package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestClusteringReceiversSynchronousCRUDAndResponseEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	var creates, gets, updates, deletes atomic.Int32
	response := `{"receiver":{"id":"response-receiver","name":"webhook","type":"webhook","cluster_id":"cluster","action":"CLUSTER_SCALE_OUT","user":"owner","project":"project","domain":null,"created_at":"2015-06-27T05:09:43Z","updated_at":null,"actor":{"big":9007199254740993},"params":{"fraction":1.234567890123456789},"channel":{"alarm_url":"https://foreign.example/trigger?secret=x"},"future":false}}`
	write := func(w http.ResponseWriter, code int) {
		w.Header().Set("Location", "https://foreign.example/unrelated#trace")
		w.Header().Set("X-Request-ID", "receiver-request")
		testcloud.JSON(w, code, response)
	}
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/receivers", func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"receiver":{"action":"CLUSTER_SCALE_OUT","cluster_id":"cluster-name","name":"webhook","type":"webhook"}}` || r.Header.Get("X-Auth-Token") != "test-token" || r.URL.RawQuery != "" {
			t.Error(string(body), r.Header, r.URL)
		}
		write(w, 201)
	})
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/receivers/controller-name", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); write(w, 200) })
	cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/receivers/controller-name", func(w http.ResponseWriter, r *http.Request) {
		updates.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"receiver":{"action":"CLUSTER_SCALE_IN","params":{}}}` {
			t.Error(string(body))
		}
		write(w, 200)
	})
	cloud.Mux.HandleFunc("DELETE /reverse/senlin/v1/receivers/controller-name", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Error(string(body))
		}
		w.WriteHeader(204)
	})
	api := receivers.New(cloud.Client("clustering", "/reverse/senlin/v1"))
	created, err := api.Create(context.Background(), receivers.CreateOpts{Name: "webhook", Type: "webhook"}, receivers.WithCreateClusterID("cluster-name"), receivers.WithCreateAction("CLUSTER_SCALE_OUT"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(context.Background(), "controller-name")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := api.Update(context.Background(), resource.ID("controller-name"), receivers.UpdateOpts{}, receivers.WithUpdateAction("CLUSTER_SCALE_IN"), receivers.WithUpdateParams(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	for index, value := range []*receivers.Receiver{created, got, updated} {
		code := 200
		if index == 0 {
			code = 201
		}
		if value.ID != "response-receiver" || value.ClusterID == nil || *value.ClusterID != "cluster" || value.Action == nil || *value.Action != "CLUSTER_SCALE_OUT" || value.UserID != "owner" || value.ProjectID != "project" || value.CreatedAt == nil || value.UpdatedAt != nil || string(value.Body["domain"]) != "null" || string(value.Actor["big"]) != "9007199254740993" || string(value.Params["fraction"]) != "1.234567890123456789" || string(value.Body["future"]) != "false" || value.StatusCode != code || value.Header.Get("Location") != "https://foreign.example/unrelated#trace" {
			t.Fatal(value)
		}
	}
	created.Header.Set("X-Request-ID", "consumer")
	if got.Header.Get("X-Request-ID") != "receiver-request" {
		t.Fatal("response headers share storage")
	}
	if err := api.Delete(context.Background(), resource.ID("controller-name")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Resources.Wait(context.Background(), resource.ID("controller-name"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal("invented a status waiter", err)
	}
	if creates.Load() != 1 || gets.Load() != 1 || updates.Load() != 1 || deletes.Load() != 1 {
		t.Fatal("receiver operation followed a channel, Location or looked up a direct ID", creates.Load(), gets.Load(), updates.Load(), deletes.Load())
	}
}

func TestClusteringReceiversMessagePresenceSnapshotsAndReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /senlin/v1/receivers", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		expected := `{"receiver":{"name":"message","type":"message"}}`
		switch calls.Add(1) {
		case 2, 3:
			expected = `{"receiver":{"action":null,"actor":null,"cluster_id":null,"name":"message","params":{"big":9007199254740993,"off":false},"type":"message","vendor":{"zero":0}}}`
		case 4:
			expected = `{"receiver":{"action":"","actor":{},"cluster_id":"","name":"message","params":{},"type":"message"}}`
		case 5:
			expected = `{"receiver":{"name":"vendor","type":"vendor.receiver"}}`
		}
		if string(body) != expected || r.Header.Get("OpenStack-API-Version") != "" {
			t.Error(string(body), r.Header)
		}
		testcloud.JSON(w, 201, `{"receiver":{"id":"message-id","cluster_id":null,"action":null,"actor":null,"params":{},"channel":{"number":9007199254740993}}}`)
	})
	api := receivers.New(cloud.Client("clustering", "/senlin/v1"))
	if _, err := api.Create(context.Background(), receivers.CreateOpts{Name: "message", Type: "message"}); err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(`{"big":9007199254740993,"off":false}`)
	options := receivers.WithCreateOptions(receivers.CreateOpts{Name: "message", Type: "message", Params: raw})
	raw[2] = 'X'
	vendor := map[string]any{"zero": 0}
	extension := receivers.WithCreateField("vendor", vendor)
	vendor["zero"] = 1
	for range 2 {
		value, err := api.Create(context.Background(), receivers.CreateOpts{}, options, receivers.WithCreateClusterIDNull(), receivers.WithCreateActionNull(), receivers.WithCreateActor(nil), extension)
		if err != nil || value.ClusterID != nil || value.Action != nil || value.Actor != nil || string(value.Body["cluster_id"]) != "null" || string(value.Channel["number"]) != "9007199254740993" {
			t.Fatal(value, err)
		}
	}
	if _, err := api.Create(context.Background(), receivers.CreateOpts{Name: "message", Type: "message"}, receivers.WithCreateClusterID(""), receivers.WithCreateAction(""), receivers.WithCreateActor(map[string]any{}), receivers.WithCreateParams(map[string]any{})); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Create(context.Background(), receivers.CreateOpts{Name: "vendor", Type: "vendor.receiver"}); err != nil {
		t.Fatal("SDK invented a receiver type enum or version gate", err)
	}
	if calls.Load() != 5 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringReceiversUpdateNameLookupFreezesInputsAndRechecksSource(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(map[bool]string{false: "snapshot", true: "changed-source"}[changed], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/senlin/v1")
			var reads, patches atomic.Int32
			var captured *request.Config[receivers.UpdateOpts]
			cloud.Mux.HandleFunc("GET /senlin/v1/receivers", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if r.URL.Query().Get("name") != "selected" {
					t.Error(r.URL)
				}
				captured.Options.Name = request.Present("changed")
				captured.Options.Params[2] = 'X'
				captured.Headers["X-Vendor"] = "changed"
				captured.Fields["vendor"] = json.RawMessage(`true`)
				if changed {
					client.Type = "compute"
				}
				testcloud.JSON(w, 200, `{"receivers":[{"id":"other","name":"unrelated"},{"id":"wire-id","name":"selected"}]}`)
			})
			cloud.Mux.HandleFunc("PATCH /senlin/v1/receivers/wire-id", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				body, _ := io.ReadAll(r.Body)
				if string(body) != `{"receiver":{"action":"","name":null,"params":{"big":9007199254740993,"off":false},"vendor":false}}` || r.Header.Get("X-Vendor") != "original" {
					t.Error(string(body), r.Header)
				}
				testcloud.JSON(w, 200, `{"receiver":{"id":"response-id","action":null}}`)
			})
			capture := func(config *request.Config[receivers.UpdateOpts]) error { captured = config; return nil }
			raw := json.RawMessage(`{"big":9007199254740993,"off":false}`)
			option := receivers.WithUpdateOptions(receivers.UpdateOpts{Params: raw})
			raw[2] = 'X'
			value, err := receivers.New(client).Update(context.Background(), resource.Name("selected"), receivers.UpdateOpts{}, option, receivers.WithUpdateNameNull(), receivers.WithUpdateAction(""), receivers.WithUpdateField("vendor", false), receivers.WithUpdateHeader("X-Vendor", "original"), capture)
			if changed {
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) || patches.Load() != 0 {
					t.Fatal(value, err, patches.Load())
				}
			} else if err != nil || value.ID != "response-id" || value.Action != nil || patches.Load() != 1 {
				t.Fatal(value, err, patches.Load())
			}
			if reads.Load() != 1 {
				t.Fatal(reads.Load())
			}
		})
	}
}

func TestClusteringReceiversUserGateResourcesBypassAndPageRecheck(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/receivers", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("user") != "owner" || r.URL.Query().Has("user_id") || r.Header.Get("OpenStack-API-Version") != "clustering 1.4" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"receivers":[{"id":"first","name":"first"}],"next":"/senlin/v1/receivers?marker=first"}`)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	api := receivers.New(client)
	for _, version := range []string{"", "1.3", "latest"} {
		client.Microversion = version
		if _, err := api.All(context.Background(), receivers.WithListUserID("owner")); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
		for _, user := range []string{"owner", ""} {
			if _, err := api.Resources.All(context.Background(), resource.WithQuery("user", user)); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(version, user, err)
			}
		}
	}
	client.Microversion = "1.4"
	for _, key := range []string{"user", "user_id"} {
		if _, err := api.All(context.Background(), receivers.WithListQuery(key, "owner")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(key, err)
		}
	}
	if _, err := api.Resources.All(context.Background(), resource.WithQuery("user_id", "owner")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
	for _, generic := range []bool{false, true} {
		client.Microversion = "1.4"
		seen := 0
		var final error
		iterator := api.List(context.Background(), receivers.WithListUserID("owner"))
		if generic {
			iterator = api.Resources.List(context.Background(), resource.WithQuery("user", "owner"))
		}
		for value, err := range iterator {
			if err != nil {
				final = err
				continue
			}
			if value.ID != "first" {
				t.Fatal(value)
			}
			seen++
			client.Microversion = "1.3"
		}
		if seen != 1 || !errors.Is(final, resource.ErrUnsupported) {
			t.Fatal(generic, seen, final)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("version-gated continuation was sent", calls.Load())
	}
}

func TestClusteringReceiversLazyPagingLocalFiltersAndWireMarkers(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/receivers", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		q := r.URL.Query()
		if q.Get("limit") != "2" || q.Get("type") != "message" || q.Get("action") != "CLUSTER_SCALE_OUT" || q.Get("sort") != "name:asc" || q.Get("global_project") != "false" || q.Get("vendor") != "retained" || q.Has("params") {
			t.Error(q)
		}
		switch q.Get("marker") {
		case "":
			w.Header().Set("Link", `</senlin/v1/receivers?marker=wire-second>; rel="next"`)
			testcloud.JSON(w, 200, `{"receivers":[{"id":"wire-first","name":"match","params":{"big":9007199254740993,"off":false}},{"id":"wire-second","name":"filtered","params":{"big":9007199254740992,"off":false}}]}`)
		case "wire-second":
			testcloud.JSON(w, 200, `{"receivers":[{"id":"wire-third","name":"match2","params":{"big":9007199254740993,"off":false}}]}`)
		case "wire-third":
			testcloud.JSON(w, 200, `{"receivers":[]}`)
		default:
			t.Error("consumer mutation or filtered row changed wire marker", q)
			testcloud.JSON(w, 200, `{"receivers":[]}`)
		}
	})
	api := receivers.New(cloud.Client("clustering", "/senlin/v1"))
	global := false
	option := receivers.WithListOptions(receivers.ListOpts{Limit: 2, Type: "message", Action: "CLUSTER_SCALE_OUT", Sort: "name:asc", GlobalProject: &global})
	global = true
	filter := map[string]any{"big": json.Number("9007199254740993"), "off": false}
	filterOption := receivers.WithListFilter("params", filter)
	filter["big"] = json.Number("9007199254740992")
	iterator := api.List(context.Background(), option, receivers.WithListQuery("vendor", "retained"), filterOption)
	if calls.Load() != 0 {
		t.Fatal("list was eager")
	}
	for range 2 {
		seen := 0
		for value, err := range iterator {
			if err != nil {
				t.Fatal(err)
			}
			seen++
			value.ID = "consumer-id"
			value.Params["big"][0] = 'X'
		}
		if seen != 2 {
			t.Fatal(seen)
		}
	}
	if calls.Load() != 6 {
		t.Fatal(calls.Load())
	}
	for _, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if calls.Load() != 7 {
		t.Fatal("break fetched another page", calls.Load())
	}
}

func TestClusteringReceiversFindDeleteMissingAmbiguousAndInvalidResolvedIDs(t *testing.T) {
	cloud := testcloud.New(t)
	var reads, deletes atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/receivers", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		body := `{"receivers":[]}`
		switch r.URL.Query().Get("name") {
		case "selected":
			body = `{"receivers":[{"id":"wire-id","name":"selected"}]}`
		case "duplicate":
			body = `{"receivers":[{"id":"first","name":"duplicate"},{"id":"second","name":"duplicate"}]}`
		case "invalid":
			body = `{"receivers":[{"id":"bad/id","name":"invalid"}]}`
		case "forbidden":
			testcloud.JSON(w, 403, `{"error":"forbidden"}`)
			return
		}
		testcloud.JSON(w, 200, body)
	})
	cloud.Mux.HandleFunc("GET /senlin/v1/receivers/missing", func(w http.ResponseWriter, r *http.Request) { reads.Add(1); w.WriteHeader(404) })
	cloud.Mux.HandleFunc("DELETE /senlin/v1/receivers/{id}", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		switch r.PathValue("id") {
		case "missing":
			w.WriteHeader(404)
		case "conflict":
			w.WriteHeader(409)
		default:
			w.WriteHeader(204)
		}
	})
	api := receivers.New(cloud.Client("clustering", "/senlin/v1"))
	for _, ref := range []resource.Ref{resource.ID("missing"), resource.Name("missing")} {
		if value, err := api.Find(context.Background(), ref); value != nil || err != nil {
			t.Fatal(value, err)
		}
		if _, err := api.Find(context.Background(), ref, resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
		if _, err := api.Resources.Find(context.Background(), ref); !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
		if err := api.Delete(context.Background(), ref); err != nil {
			t.Fatal(err)
		}
		if err := api.Delete(context.Background(), ref, resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
	}
	if value, err := api.Find(context.Background(), resource.Name("selected")); err != nil || value.ID != "wire-id" {
		t.Fatal(value, err)
	}
	if err := api.Delete(context.Background(), resource.Name("selected")); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"find", "delete"} {
		for _, name := range []string{"duplicate", "forbidden"} {
			var err error
			if operation == "find" {
				_, err = api.Find(context.Background(), resource.Name(name))
			} else {
				err = api.Delete(context.Background(), resource.Name(name))
			}
			if name == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) || name == "forbidden" && !gophercloud.ResponseCodeIs(err, 403) {
				t.Fatal(operation, name, err)
			}
		}
	}
	before := deletes.Load()
	if err := api.Delete(context.Background(), resource.Name("invalid")); !errors.Is(err, resource.ErrInvalidOption) || deletes.Load() != before {
		t.Fatal("invalid resolved ID reached DELETE", err, deletes.Load())
	}
	if err := api.Delete(context.Background(), resource.ID("conflict")); !gophercloud.ResponseCodeIs(err, 409) {
		t.Fatal(err)
	}
	if deletes.Load() != 4 || reads.Load() != 15 {
		t.Fatal(reads.Load(), deletes.Load())
	}
}

func TestClusteringReceiversAcceptedMalformedAndWrongCodesKeepEvidence(t *testing.T) {
	for _, operation := range []string{"Create", "Get", "Update", "List"} {
		for _, malformed := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: "WrongCode", true: "Malformed"}[malformed], func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				code, body := 201, `{"receiver":{"id":"valid"}}`
				if operation == "Create" {
					code = 200
				}
				if malformed {
					code, body = 200, `{"receiver":null}`
					if operation == "Create" {
						code = 201
					} else if operation == "List" {
						body = `{"receivers":[false]}`
					}
				}
				handler := func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-ID", "response-evidence")
					testcloud.JSON(w, code, body)
				}
				cloud.Mux.HandleFunc("/senlin/v1/receivers", handler)
				cloud.Mux.HandleFunc("/senlin/v1/receivers/wire-id", handler)
				api := receivers.New(cloud.Client("clustering", "/senlin/v1"))
				var err error
				switch operation {
				case "Create":
					_, err = api.Create(context.Background(), receivers.CreateOpts{Name: "message", Type: "message"})
				case "Get":
					_, err = api.Get(context.Background(), "wire-id")
				case "Update":
					_, err = api.Update(context.Background(), resource.ID("wire-id"), receivers.UpdateOpts{}, receivers.WithUpdateName("changed"))
				case "List":
					_, err = api.All(context.Background())
				}
				var accepted *resource.ResponseError
				if malformed {
					if !errors.As(err, &accepted) || accepted.StatusCode != code || string(accepted.Body) != body || accepted.Header.Get("X-Request-ID") != "response-evidence" {
						t.Fatal(err, accepted)
					}
				} else {
					var original gophercloud.ErrUnexpectedResponseCode
					if !gophercloud.ResponseCodeIs(err, code) || !errors.As(err, &original) || string(original.Body) != body || errors.As(err, &accepted) {
						t.Fatal(err)
					}
				}
				if calls.Load() != 1 {
					t.Fatal("response failure resent or fetched", calls.Load())
				}
			})
		}
	}
}

func TestClusteringReceiversFindContinuationKeepsNameAndLaterPageOutcomes(t *testing.T) {
	for _, fixture := range []struct {
		name, firstName, expectedID string
		secondCode                  int
		want                        error
	}{
		{name: "later-match", firstName: "unrelated", expectedID: "second", secondCode: 200},
		{name: "cross-page-duplicate", firstName: "selected", secondCode: 200, want: resource.ErrAmbiguous},
		{name: "later-error-after-match", firstName: "selected", secondCode: 403},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/receivers", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("name") != "selected" || r.URL.Query().Has("limit") || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error("receiver Find continuation lost its name query", r.URL, r.Header)
				}
				switch r.URL.Query().Get("marker") {
				case "":
					testcloud.JSON(w, 200, `{"receivers":[{"id":"first","name":"`+fixture.firstName+`"}],"receivers_links":[{"rel":"next","href":"/senlin/v1/receivers?marker=first"}]}`)
				case "first":
					if fixture.secondCode != 200 {
						testcloud.JSON(w, fixture.secondCode, `{"error":"later-page-forbidden"}`)
						return
					}
					testcloud.JSON(w, 200, `{"receivers":[{"id":"second","name":"selected","channel":{}}]}`)
				default:
					t.Error("unexpected receiver Find continuation", r.URL)
					w.WriteHeader(500)
				}
			})
			value, err := receivers.New(cloud.Client("clustering", "/senlin/v1")).Find(context.Background(), resource.Name("selected"))
			if fixture.secondCode == 403 {
				if value != nil || !gophercloud.ResponseCodeIs(err, 403) {
					t.Fatal("Find returned an earlier match or ignored a later HTTP failure", value, err)
				}
			} else if fixture.want != nil {
				var duplicate *resource.AmbiguousError
				if value != nil || !errors.Is(err, fixture.want) || !errors.As(err, &duplicate) || len(duplicate.IDs) != 2 || duplicate.IDs[0] != "first" || duplicate.IDs[1] != "second" {
					t.Fatal(value, err, duplicate)
				}
			} else if err != nil || value == nil || value.ID != fixture.expectedID || value.Name != "selected" || value.StatusCode != 200 {
				t.Fatal(value, err)
			}
			if calls.Load() != 2 {
				t.Fatal("Find skipped or refetched its receiver continuation", calls.Load())
			}
		})
	}
}

func TestClusteringReceiversPreflightOwnedFieldsOptionsAndContext(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("invalid receiver reached HTTP", r.URL)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	api := receivers.New(client)
	for _, value := range []receivers.CreateOpts{
		{}, {Name: "message"}, {Name: "receiver", Type: "  "}, {Name: "webhook", Type: "webhook"},
		{Name: "webhook", Type: "webhook", ClusterID: request.Present("cluster"), Action: request.Null[string]()},
		{Name: "message", Type: "message", Actor: json.RawMessage(`[]`)}, {Name: "message", Type: "message", Params: json.RawMessage(`false`)},
	} {
		if _, err := api.Create(context.Background(), value); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	for _, option := range []receivers.CreateOption{nil, receivers.WithCreateField("cluster_id", "override"), receivers.WithCreateField("channel", nil), receivers.WithCreateHeader("X-Auth-Token", "foreign"), receivers.WithCreateHeader("OpenStack-API-Version", "clustering 1.99"), request.WithQuery[receivers.CreateOpts]("vendor", "unsupported")} {
		if _, err := api.Create(context.Background(), receivers.CreateOpts{Name: "message", Type: "message"}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.Update(context.Background(), resource.Name("selected"), receivers.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("empty stateless update accepted", err)
	}
	for _, field := range []string{"type", "cluster_id", "actor", "channel", "id", "user", "name", "action", "params"} {
		if _, err := api.Update(context.Background(), resource.Name("selected"), receivers.UpdateOpts{Name: request.Present("changed")}, receivers.WithUpdateField(field, nil)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(field, err)
		}
	}
	for _, option := range []receivers.ListOption{nil, receivers.WithListFilter("unknown", nil), receivers.WithListQuery("params", "local"), receivers.WithListOptions(receivers.ListOpts{Limit: -1}), receivers.WithListOptions(receivers.ListOpts{Sort: "invalid:desc"}), request.WithField[receivers.ListOpts]("vendor", false)} {
		if _, err := api.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.Resources.All(context.Background(), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	for _, api := range []*receivers.API{nil, receivers.New(nil), receivers.New(&gophercloud.ServiceClient{})} {
		if _, err := api.Get(context.Background(), "id"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if _, err := api.Create(context.Background(), receivers.CreateOpts{Name: "message", Type: "message"}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Create(ctx, receivers.CreateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := api.Get(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, resource.ID("id"), receivers.UpdateOpts{Name: request.Present("changed")}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, resource.ID("id")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := api.All(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	change := func(config *request.Config[receivers.CreateOpts]) error { client.Type = "compute"; return nil }
	if _, err := api.Create(context.Background(), receivers.CreateOpts{Name: "message", Type: "message"}, change); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}
