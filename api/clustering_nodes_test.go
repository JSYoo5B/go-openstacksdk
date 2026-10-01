package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/clustering/v1/nodes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestClusteringNodeCreateAcceptedEnvelopeSnapshotsAndSubmission(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	response := `{"node":{"id":"response-node","name":"worker","profile_id":"response-profile","profile_name":"profile","physical_id":"physical-node","cluster_id":null,"index":9007199254740993,"role":null,"project":"project-1","domain":null,"user":"user-1","init_at":"2015-03-05T08:53:15Z","created_at":null,"updated_at":null,"metadata":{"big":9007199254740993,"unset":null},"data":{"zero":0,"off":false},"details":{},"dependents":{},"tainted":false,"status":"INIT","status_reason":"accepted","future":false}}`
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		data, _ := io.ReadAll(r.Body)
		if string(data) != `{"node":{"cluster_id":null,"metadata":{"big":9007199254740993,"unset":null},"name":"worker","profile_id":"profile-name","role":"","vendor":{"off":false,"zero":0}}}` || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Vendor") != "retained" {
			t.Error(string(data), r.URL, r.Header)
		}
		w.Header().Set("Location", "actions/create-action")
		w.Header().Set("X-OpenStack-Request-ID", "node-create")
		testcloud.JSON(w, 202, response)
	})
	metadata := json.RawMessage(`{"big":9007199254740993,"unset":null}`)
	option := nodes.WithCreateOptions(nodes.CreateOpts{Name: "worker", ProfileID: "profile-name", ClusterID: request.Null[string](), Role: request.Present(""), Metadata: metadata})
	metadata[2] = 'X'
	vendor := map[string]any{"off": false, "zero": 0}
	extension := nodes.WithCreateField("vendor", vendor)
	vendor["off"] = true
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	for range 2 {
		value, err := api.Create(context.Background(), nodes.CreateOpts{}, option, extension, nodes.WithCreateHeader("X-Vendor", "retained"))
		if err != nil || value.ID != "response-node" || value.ProfileID != "response-profile" || value.PhysicalID != "physical-node" || value.ClusterID != nil || value.Role != nil || value.DomainID != nil || value.Index == nil || value.Index.String() != "9007199254740993" || value.Tainted == nil || *value.Tainted {
			t.Fatal(value, err)
		}
		if value.ProjectID != "project-1" || value.UserID != "user-1" || value.InitAt == nil || value.CreatedAt != nil || value.UpdatedAt != nil || string(value.UserMetadata["big"]) != "9007199254740993" || string(value.Data["off"]) != "false" || string(value.Body["cluster_id"]) != "null" || string(value.Body["future"]) != "false" || value.StatusCode != 202 {
			t.Fatal(value)
		}
		if value.Operation == nil || value.Operation.ActionID != "create-action" || value.Operation.Location != "actions/create-action" || value.Operation.StatusCode != 202 || string(value.Operation.Body) != response || value.Operation.Header.Get("X-OpenStack-Request-ID") != "node-create" {
			t.Fatal(value.Operation)
		}
		value.Header.Set("X-OpenStack-Request-ID", "consumer-change")
		if value.Operation.Header.Get("X-OpenStack-Request-ID") != "node-create" {
			t.Fatal("submission shares resource headers")
		}
	}
	if calls.Load() != 2 {
		t.Fatal("create performed a fetch or retry", calls.Load())
	}
}

func TestClusteringNodeCreateDefaultsDoNotInventClusterOrRole(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"node":{"name":"worker","profile_id":"profile"}}` {
			t.Error(string(body))
		}
		w.Header().Set("Location", "actions/default-create")
		testcloud.JSON(w, 202, `{"node":{"id":null,"name":"response-name","index":-1,"role":""}}`)
	})
	value, err := nodes.New(cloud.Client("clustering", "/senlin/v1")).Create(context.Background(), nodes.CreateOpts{Name: "worker", ProfileID: "profile"})
	if err != nil || value.ID != "" || value.Name != "response-name" || value.Operation == nil || value.Operation.ActionID != "default-create" || value.Index == nil || value.Index.String() != "-1" || value.Role == nil || *value.Role != "" || string(value.Body["id"]) != "null" {
		t.Fatal(value, err)
	}
}

func TestClusteringNodeGetDirectIdentityAndPhysicalDetails(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes/short-name", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch calls.Add(1) {
		case 1:
			if len(q) != 0 {
				t.Error(q)
			}
		case 2:
			if q.Get("show_details") != "true" || q.Get("vendor") != "retained" || r.Header.Get("X-Vendor") != "custom" {
				t.Error(q, r.Header)
			}
		case 3:
			if q.Get("show_details") != "false" {
				t.Error(q)
			}
		default:
			t.Error(calls.Load())
		}
		w.Header().Set("X-Request-ID", "read-node")
		testcloud.JSON(w, 200, `{"node":{"id":"response-node","physical_id":"physical","cluster_id":"cluster","role":"worker","index":0,"details":{"capacity":9007199254740993},"metadata":{},"tainted":false}}`)
	})
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	for _, options := range [][]nodes.GetOption{nil, {nodes.WithGetDetails(true), nodes.WithGetQuery("vendor", "retained"), nodes.WithGetHeader("X-Vendor", "custom")}, {nodes.WithGetDetails(false)}} {
		value, err := api.Get(context.Background(), "short-name", options...)
		if err != nil || value.ID != "response-node" || value.ClusterID == nil || *value.ClusterID != "cluster" || value.Index == nil || value.Index.String() != "0" || string(value.Details["capacity"]) != "9007199254740993" || value.Header.Get("X-Request-ID") != "read-node" || value.Operation != nil {
			t.Fatal(value, err)
		}
	}
	if calls.Load() != 3 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeUpdateNameResolutionFreezesNullableInputsAndHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.13"
	var reads, updates atomic.Int32
	var captured *request.Config[nodes.UpdateOpts]
	metadata := json.RawMessage(`{"number":9007199254740993,"off":false}`)
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		if r.URL.Query().Get("name") != "old-name" {
			t.Error(r.URL)
		}
		captured.Options.Name = request.Present("mutated")
		captured.Options.Metadata[2] = 'X'
		captured.Options.Tainted = request.Present(true)
		captured.Headers["X-Vendor"] = "mutated"
		captured.Fields["vendor"] = json.RawMessage(`true`)
		testcloud.JSON(w, 200, `{"nodes":[{"id":"wrong-id","name":"unrelated"},{"id":"resolved-node","name":"old-name"}]}`)
	})
	cloud.Mux.HandleFunc("PATCH /senlin/v1/nodes/resolved-node", func(w http.ResponseWriter, r *http.Request) {
		updates.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"node":{"metadata":{"number":9007199254740993,"off":false},"name":null,"profile_id":"new-profile","role":"","tainted":false,"vendor":false}}` || r.Header.Get("X-Vendor") != "original" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "/senlin/v1/actions/update-action")
		testcloud.JSON(w, 202, `{"node":{"id":"response-node","tainted":false,"role":"","metadata":{"number":9007199254740993}}}`)
	})
	option := nodes.WithUpdateOptions(nodes.UpdateOpts{Name: request.Null[string](), ProfileID: request.Present("new-profile"), Role: request.Present(""), Metadata: metadata, Tainted: request.Present(false)})
	metadata[2] = 'Y'
	capture := func(config *request.Config[nodes.UpdateOpts]) error { captured = config; return nil }
	value, err := nodes.New(client).Update(context.Background(), resource.Name("old-name"), nodes.UpdateOpts{}, option, nodes.WithUpdateField("vendor", false), nodes.WithUpdateHeader("X-Vendor", "original"), capture)
	if err != nil || value.ID != "response-node" || value.Operation == nil || value.Operation.ActionID != "update-action" || value.Tainted == nil || *value.Tainted || reads.Load() != 1 || updates.Load() != 1 {
		t.Fatal(value, err, reads.Load(), updates.Load())
	}
}

func TestClusteringNodeUpdateIDNoLookupAndOptionalNullEmptyObject(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("PATCH /senlin/v1/nodes/name-shaped-id", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"node":{"metadata":{},"profile_id":null,"role":null}}` {
			t.Error(string(body))
		}
		w.Header().Set("Location", "actions/change")
		testcloud.JSON(w, 202, `{"node":{"id":"response-node"}}`)
	})
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	option := nodes.WithUpdateOptions(nodes.UpdateOpts{ProfileID: request.Null[string](), Role: request.Null[string](), Metadata: json.RawMessage(`{}`)})
	for range 2 {
		if _, err := api.Update(context.Background(), resource.ID("name-shaped-id"), nodes.UpdateOpts{}, option); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeTaintedFalseAndNullRequireExplicitVersion(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("PATCH /senlin/v1/nodes/node-id", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"node":{"tainted":false}}` || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "actions/taint-change")
		testcloud.JSON(w, 202, `{"node":{"id":"node-id","tainted":false}}`)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	api := nodes.New(client)
	for _, version := range []string{"", "1.12", "latest"} {
		client.Microversion = version
		for _, value := range []request.Optional[bool]{request.Present(false), request.Present(true), request.Null[bool]()} {
			if _, err := api.Update(context.Background(), resource.Name("selected"), nodes.UpdateOpts{Tainted: value}); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(version, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
	client.Microversion = "1.13"
	if _, err := api.Update(context.Background(), resource.ID("node-id"), nodes.UpdateOpts{Tainted: request.Present(false)}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeVersionRevalidatedAfterNameLookup(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.13"
	var reads, writes atomic.Int32
	var captured *request.Config[nodes.UpdateOpts]
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		client.Microversion = "1.12"
		captured.Options.Tainted = request.Optional[bool]{}
		testcloud.JSON(w, 200, `{"nodes":[{"id":"node-id","name":"selected"}]}`)
	})
	cloud.Mux.HandleFunc("PATCH /senlin/v1/nodes/node-id", func(w http.ResponseWriter, r *http.Request) { writes.Add(1); t.Error("version-gated mutation sent") })
	capture := func(config *request.Config[nodes.UpdateOpts]) error { captured = config; return nil }
	_, err := nodes.New(client).Update(context.Background(), resource.Name("selected"), nodes.UpdateOpts{Tainted: request.Present(false)}, capture)
	if !errors.Is(err, resource.ErrUnsupported) || reads.Load() != 1 || writes.Load() != 0 {
		t.Fatal(err, reads.Load(), writes.Load())
	}
}

func TestClusteringNodeDeleteActionDefaultForceAndResponseOwnership(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("DELETE /senlin/v1/nodes/node-id", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch calls.Add(1) {
		case 1, 3:
			if len(body) != 0 {
				t.Error(string(body))
			}
		case 2:
			if string(body) != `{"force":true}` {
				t.Error(string(body))
			}
		}
		w.Header().Set("Location", "actions/delete-action")
		w.Header().Set("X-Request-ID", "delete-node")
		w.WriteHeader(202)
	})
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	force := true
	option := nodes.WithDeleteOptions(nodes.DeleteOpts{Force: force})
	force = false
	for _, options := range [][]nodes.DeleteOption{nil, {option}, {nodes.WithDeleteForce(false)}} {
		value, err := api.Delete(context.Background(), resource.ID("node-id"), options...)
		if err != nil || value == nil || value.ActionID != "delete-action" || value.Location != "actions/delete-action" || value.StatusCode != 202 || len(value.Body) != 0 || value.Header.Get("X-Request-ID") != "delete-node" {
			t.Fatal(value, err)
		}
	}
	if calls.Load() != 3 {
		t.Fatal("delete performed automatic fetch or poll", calls.Load())
	}
	if err := api.Resources.Delete(context.Background(), resource.ID("node-id")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal("shared Delete discarded submission", err)
	}
}

func TestClusteringNodeDeleteNameLookupDefaultsMissingAndConflict(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, deletes atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		switch r.URL.Query().Get("name") {
		case "selected":
			testcloud.JSON(w, 200, `{"nodes":[{"id":"resolved-id","name":"selected"}]}`)
		case "no-id":
			testcloud.JSON(w, 200, `{"nodes":[{"id":null,"name":"no-id"}]}`)
		default:
			testcloud.JSON(w, 200, `{"nodes":[]}`)
		}
	})
	cloud.Mux.HandleFunc("DELETE /senlin/v1/nodes/resolved-id", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.Header().Set("X-Request-ID", "delete-conflict")
		testcloud.JSON(w, 409, `{"error":"node busy"}`)
	})
	cloud.Mux.HandleFunc("DELETE /senlin/v1/nodes/missing", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		testcloud.JSON(w, 404, `{"error":"missing"}`)
	})
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	if value, err := api.Delete(context.Background(), resource.Name("missing")); value != nil || err != nil {
		t.Fatal(value, err)
	}
	if _, err := api.Delete(context.Background(), resource.Name("missing"), nodes.WithDeleteIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Delete(context.Background(), resource.Name("no-id")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.Delete(context.Background(), resource.Name("selected")); !gophercloud.ResponseCodeIs(err, 409) || !strings.Contains(err.Error(), "node busy") {
		t.Fatal(err)
	}
	if _, err := api.Delete(context.Background(), resource.ID("missing"), nodes.WithDeleteForce(true)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if value, err := api.Delete(context.Background(), resource.ID("missing"), nodes.WithDeleteForce(true), nodes.WithDeleteIgnoreMissing(true)); value != nil || err != nil {
		t.Fatal(value, err)
	}
	if _, err := api.Delete(context.Background(), resource.Name("missing"), nodes.WithDeleteForce(true)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Delete(context.Background(), resource.ID("missing"), nodes.WithDeleteIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if lists.Load() != 5 || deletes.Load() != 4 {
		t.Fatal(lists.Load(), deletes.Load())
	}
}

func TestClusteringNodeAcceptedMalformedEvidenceNeverResends(t *testing.T) {
	for _, operation := range []string{"Create", "Get", "Update"} {
		for _, body := range []string{`{}`, `{"node":null}`, `{"node":[]}`, `{"node":{"details":[]}}`, `{"node":`} {
			t.Run(operation+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				code := 202
				if operation == "Get" {
					code = 200
				}
				cloud.Mux.HandleFunc("/senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Location", "actions/accepted-action")
					w.Header().Set("X-Request-ID", "accepted")
					testcloud.JSON(w, code, body)
				})
				cloud.Mux.HandleFunc("/senlin/v1/nodes/node-id", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Location", "actions/accepted-action")
					w.Header().Set("X-Request-ID", "accepted")
					testcloud.JSON(w, code, body)
				})
				api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
				var value *nodes.Node
				var err error
				switch operation {
				case "Create":
					value, err = api.Create(context.Background(), nodes.CreateOpts{Name: "worker", ProfileID: "profile"})
				case "Get":
					value, err = api.Get(context.Background(), "node-id")
				case "Update":
					value, err = api.Update(context.Background(), resource.ID("node-id"), nodes.UpdateOpts{Role: request.Present("new-role")})
				}
				var evidence *resource.ResponseError
				if value != nil || !errors.As(err, &evidence) || evidence.StatusCode != code || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "accepted" || calls.Load() != 1 {
					t.Fatal(value, err, evidence, calls.Load())
				}
			})
		}
	}
}

func TestClusteringNodeMalformedActionLocationsRetainAcceptedEvidence(t *testing.T) {
	for _, operation := range []string{"Create", "Update", "Delete"} {
		locations := []string{"/senlin/v1/actions/", "https://foreign.example/actions/id", "/senlin/v1/nodes/id", "actions/id?token=bad"}
		locations = append(locations, "")
		for _, location := range locations {
			t.Run(operation+location, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				body := `{"node":{"id":"accepted-node"}}`
				handler := func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if location != "" {
						w.Header().Set("Location", location)
					}
					w.Header().Set("X-Request-ID", "accepted")
					testcloud.JSON(w, 202, body)
				}
				cloud.Mux.HandleFunc("/senlin/v1/nodes", handler)
				cloud.Mux.HandleFunc("/senlin/v1/nodes/node-id", handler)
				api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
				var err error
				var node *nodes.Node
				var action *actions.Submission
				switch operation {
				case "Create":
					node, err = api.Create(context.Background(), nodes.CreateOpts{Name: "worker", ProfileID: "profile"})
				case "Update":
					node, err = api.Update(context.Background(), resource.ID("node-id"), nodes.UpdateOpts{Role: request.Present("role")})
				case "Delete":
					action, err = api.Delete(context.Background(), resource.ID("node-id"))
				}
				var evidence *resource.ResponseError
				if node != nil || action != nil || !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != body || calls.Load() != 1 {
					t.Fatal(node, action, err, evidence, calls.Load())
				}
			})
		}
	}
}

func TestClusteringNodeWrongSuccessCodesRemainHTTPErrors(t *testing.T) {
	for _, operation := range []string{"Create", "Get", "Update", "Delete"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			wrong := 201
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "actions/wrong-code")
				testcloud.JSON(w, wrong, `{"node":{"id":"node-id"}}`)
			}
			cloud.Mux.HandleFunc("/senlin/v1/nodes", handler)
			cloud.Mux.HandleFunc("/senlin/v1/nodes/node-id", handler)
			api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
			var err error
			switch operation {
			case "Create":
				_, err = api.Create(context.Background(), nodes.CreateOpts{Name: "worker", ProfileID: "profile"})
			case "Get":
				_, err = api.Get(context.Background(), "node-id")
			case "Update":
				_, err = api.Update(context.Background(), resource.ID("node-id"), nodes.UpdateOpts{Role: request.Present("role")})
			case "Delete":
				_, err = api.Delete(context.Background(), resource.ID("node-id"))
			}
			if !gophercloud.ResponseCodeIs(err, wrong) {
				t.Fatal(err)
			}
			var accepted *resource.ResponseError
			if errors.As(err, &accepted) {
				t.Fatal("HTTP failure became accepted decode error", err)
			}
		})
	}
}

func TestClusteringNodePreflightRejectsImmutableFieldsAndWrongOptions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("unexpected HTTP", r.Method, r.URL)
	})
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	base := nodes.CreateOpts{Name: "worker", ProfileID: "profile"}
	tests := []struct {
		name string
		call func() error
	}{
		{"name-required", func() error {
			_, e := api.Create(context.Background(), nodes.CreateOpts{ProfileID: "profile"})
			return e
		}},
		{"profile-required", func() error { _, e := api.Create(context.Background(), nodes.CreateOpts{Name: "worker"}); return e }},
		{"name-grammar", func() error {
			_, e := api.Create(context.Background(), nodes.CreateOpts{Name: "1wrong", ProfileID: "profile"})
			return e
		}},
		{"name-length", func() error {
			_, e := api.Create(context.Background(), nodes.CreateOpts{Name: strings.Repeat("a", 255), ProfileID: "profile"})
			return e
		}},
		{"metadata-array", func() error {
			_, e := api.Create(context.Background(), base, nodes.WithCreateMetadata([]int{}))
			return e
		}},
		{"core-profile", func() error {
			_, e := api.Create(context.Background(), base, nodes.WithCreateField("profile_id", "other"))
			return e
		}},
		{"response-id", func() error {
			_, e := api.Create(context.Background(), base, nodes.WithCreateField("id", "unsafe"))
			return e
		}},
		{"create-tainted", func() error {
			_, e := api.Create(context.Background(), base, nodes.WithCreateField("tainted", false))
			return e
		}},
		{"empty-update", func() error {
			_, e := api.Update(context.Background(), resource.Name("selected"), nodes.UpdateOpts{})
			return e
		}},
		{"update-cluster", func() error {
			_, e := api.Update(context.Background(), resource.Name("selected"), nodes.UpdateOpts{}, nodes.WithUpdateField("cluster_id", "cluster"))
			return e
		}},
		{"update-status", func() error {
			_, e := api.Update(context.Background(), resource.Name("selected"), nodes.UpdateOpts{}, nodes.WithUpdateField("status", "ACTIVE"))
			return e
		}},
		{"auth-header", func() error {
			_, e := api.Create(context.Background(), base, nodes.WithCreateHeader("X-Auth-Token", "replace"))
			return e
		}},
		{"version-header", func() error {
			_, e := api.Get(context.Background(), "node-id", nodes.WithGetHeader("OpenStack-API-Version", "clustering 1.13"))
			return e
		}},
		{"details-query", func() error {
			_, e := api.Get(context.Background(), "node-id", nodes.WithGetQuery("show_details", "true"))
			return e
		}},
		{"mutation-query", func() error {
			_, e := api.Create(context.Background(), base, request.WithQuery[nodes.CreateOpts]("vendor", "ignored"))
			return e
		}},
		{"delete-fields", func() error {
			_, e := api.Delete(context.Background(), resource.Name("selected"), request.WithField[nodes.DeleteOpts]("vendor", false))
			return e
		}},
		{"delete-version-header", func() error {
			_, e := api.Delete(context.Background(), resource.Name("selected"), nodes.WithDeleteHeader("OpenStack-API-Version", "clustering 1.13"))
			return e
		}},
		{"nil-option", func() error { _, e := api.Get(context.Background(), "node-id", nil); return e }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeFindExplicitNamesDuplicatesAndMissingDefaults(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		switch r.URL.Query().Get("name") {
		case "duplicate":
			testcloud.JSON(w, 200, `{"nodes":[{"id":"one","name":"duplicate"},{"id":"two","name":"duplicate"}]}`)
		case "12345678-1234-1234-1234-123456789012":
			testcloud.JSON(w, 200, `{"nodes":[{"id":"response-id","name":"12345678-1234-1234-1234-123456789012"}]}`)
		default:
			testcloud.JSON(w, 200, `{"nodes":[]}`)
		}
	})
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes/missing", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 404, `{"error":"missing"}`)
	})
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	for _, ref := range []resource.Ref{resource.ID("missing"), resource.Name("missing")} {
		if value, err := api.Find(context.Background(), ref); value != nil || err != nil {
			t.Fatal(value, err)
		}
		if _, err := api.Find(context.Background(), ref, resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
	}
	var ambiguous *resource.AmbiguousError
	if _, err := api.Find(context.Background(), resource.Name("duplicate")); !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"one", "two"}) {
		t.Fatal(err, ambiguous)
	}
	if value, err := api.Find(context.Background(), resource.Name("12345678-1234-1234-1234-123456789012")); err != nil || value.ID != "response-id" {
		t.Fatal(value, err)
	}
	if lists.Load() != 4 || gets.Load() != 2 {
		t.Fatal(lists.Load(), gets.Load())
	}
}
