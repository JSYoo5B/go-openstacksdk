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
	"gophercloudsdk/clustering/v1/policies"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestClusteringPolicyCreateEnvelopePrecisionAndOptionsSnapshot(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Vendor") != "custom" || r.URL.RawQuery != "" {
			t.Error(r.URL, r.Header)
		}
		data, _ := io.ReadAll(r.Body)
		var body map[string]map[string]json.RawMessage
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 1 || len(body["policy"]) != 3 || string(body["policy"]["name"]) != `"scaling"` || string(body["policy"]["spec"]) != `{"type":"senlin.policy.scaling","version":"1.0","properties":{"number":9007199254740993}}` || string(body["policy"]["vendor"]) != `{"enabled":false}` {
			t.Error(string(data))
		}
		w.Header().Set("X-OpenStack-Request-ID", "policy-create")
		testcloud.JSON(w, 201, `{"policy":{"id":"returned-policy","name":"scaling","type":"senlin.policy.scaling-1.0","spec":{"properties":{"number":9007199254740993}},"data":{"cost":12345678901234567890,"zero":0,"unset":null},"project":"project-1","domain":null,"user":"user-1","created_at":"2016-01-18T00:00:00Z","updated_at":null,"vendor":false}}`)
	})
	raw := json.RawMessage(`{"type":"senlin.policy.scaling","version":"1.0","properties":{"number":9007199254740993}}`)
	option := policies.WithCreateOptions(policies.CreateOpts{Name: "scaling", Spec: raw})
	raw[2] = 'X'
	extension := map[string]any{"enabled": false}
	field := policies.WithCreateField("vendor", extension)
	extension["enabled"] = true
	api := policies.New(cloud.Client("clustering", "/senlin/v1"))
	for range 2 {
		value, err := api.Create(context.Background(), policies.CreateOpts{}, option, field, policies.WithCreateHeader("X-Vendor", "custom"))
		if err != nil || value.ID != "returned-policy" || value.Name != "scaling" || value.Type != "senlin.policy.scaling-1.0" || value.ProjectID != "project-1" || value.DomainID != "" || value.UserID != "user-1" {
			t.Fatal(value, err)
		}
		if string(value.Spec["properties"]) != `{"number":9007199254740993}` || string(value.Data["cost"]) != "12345678901234567890" || string(value.Body["domain"]) != "null" || string(value.Body["vendor"]) != "false" || value.CreatedAt == nil || value.UpdatedAt != nil || value.StatusCode != 201 || value.Header.Get("X-OpenStack-Request-ID") != "policy-create" {
			t.Fatal(value)
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringPolicyCreateSpecConvenienceSnapshotsObjects(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"policy":{"name":"scale","spec":{"number":9007199254740993}}}` {
			t.Error(string(body))
		}
		testcloud.JSON(w, 201, `{"policy":{"id":"policy-id"}}`)
	})
	input := map[string]any{"number": json.Number("9007199254740993")}
	option := policies.WithCreateSpec(input)
	input["number"] = 2
	if _, err := policies.New(cloud.Client("clustering", "/senlin/v1")).Create(context.Background(), policies.CreateOpts{Name: "scale"}, option); err != nil {
		t.Fatal(err)
	}
}

func TestClusteringPolicyGetIdentityNullAndEnvelopeEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /senlin/v1/policies/short-id", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"policy":{"id":null,"name":"returned-name","spec":{},"data":null,"domain":null}}`)
	})
	value, err := policies.New(cloud.Client("clustering", "/senlin/v1")).Get(context.Background(), "short-id")
	if err != nil || value.ID != "" || value.Name != "returned-name" || value.Data != nil || value.Spec == nil || string(value.Body["id"]) != "null" || string(value.Body["domain"]) != "null" {
		t.Fatal(value, err)
	}
	if _, exists := value.Body["project"]; exists {
		t.Fatal("missing project synthesized")
	}
}

func TestClusteringPolicyUpdateNameResolvedOnceAndInputsFrozen(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, updates atomic.Int32
	var captured *request.Config[policies.UpdateOpts]
	name := "new-name"
	cloud.Mux.HandleFunc("GET /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if r.URL.Query().Get("name") != "old-name" {
			t.Error(r.URL)
		}
		name = "mutated-name"
		*captured.Options.Name = "mutated-config"
		captured.Fields["vendor"] = json.RawMessage(`true`)
		captured.Headers["X-Vendor"] = "mutated-header"
		testcloud.JSON(w, 200, `{"policies":[{"id":"other","name":"other"},{"id":"policy-id","name":"old-name"}]}`)
	})
	cloud.Mux.HandleFunc("PATCH /senlin/v1/policies/policy-id", func(w http.ResponseWriter, r *http.Request) {
		updates.Add(1)
		data, _ := io.ReadAll(r.Body)
		if string(data) != `{"policy":{"name":"new-name","vendor":false}}` || r.Header.Get("X-Vendor") != "original" {
			t.Error(string(data), r.Header)
		}
		testcloud.JSON(w, 200, `{"policy":{"id":"response-id","name":"new-name"}}`)
	})
	option := func(config *request.Config[policies.UpdateOpts]) error { captured = config; return nil }
	api := policies.New(cloud.Client("clustering", "/senlin/v1"))
	value, err := api.Update(context.Background(), resource.Name("old-name"), policies.UpdateOpts{Name: &name}, policies.WithUpdateField("vendor", false), policies.WithUpdateHeader("X-Vendor", "original"), option)
	if err != nil || value.ID != "response-id" || lookups.Load() != 1 || updates.Load() != 1 {
		t.Fatal(value, err, lookups.Load(), updates.Load())
	}
}

func TestClusteringPolicyUpdateIDSkipsLookupAndSnapshotCanBeReused(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("PATCH /senlin/v1/policies/looks-like-name", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"policy":{"name":"original"}}` {
			t.Error(string(body))
		}
		testcloud.JSON(w, 200, `{"policy":{"id":"looks-like-name","name":"original"}}`)
	})
	name := "original"
	option := policies.WithUpdateOptions(policies.UpdateOpts{Name: &name})
	name = "later"
	api := policies.New(cloud.Client("clustering", "/senlin/v1"))
	for range 2 {
		if _, err := api.Update(context.Background(), resource.ID("looks-like-name"), policies.UpdateOpts{}, option); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringPolicyValidateVersionGateAndPrecision(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /senlin/v1/policies/validate", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		data, _ := io.ReadAll(r.Body)
		if string(data) != `{"policy":{"spec":{"number":9007199254740993},"vendor":false}}` || r.Header.Get("OpenStack-API-Version") != "clustering 1.2" {
			t.Error(string(data), r.Header)
		}
		testcloud.JSON(w, 200, `{"policy":{"id":null,"spec":{"number":9007199254740993}}}`)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	api := policies.New(client)
	for _, selected := range []string{"", "1.0", "1.1", "latest"} {
		client.Microversion = selected
		if _, err := api.Validate(context.Background(), policies.ValidateOpts{Spec: json.RawMessage(`{}`)}); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(selected, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("version gate sent HTTP", calls.Load())
	}
	client.Microversion = "1.2"
	value, err := api.Validate(context.Background(), policies.ValidateOpts{}, policies.WithValidateSpec(map[string]any{"number": json.Number("9007199254740993")}), policies.WithValidateField("vendor", false))
	if err != nil || value.ID != "" || string(value.Spec["number"]) != "9007199254740993" || string(value.Body["id"]) != "null" || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
}

func TestClusteringPolicyMutationPreflightRejectsInvalidCoreAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("unexpected HTTP", r.Method, r.URL)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.2"
	api := policies.New(client)
	base := policies.CreateOpts{Name: "policy", Spec: json.RawMessage(`{}`)}
	tests := []struct {
		name string
		call func() error
	}{
		{"missing-name", func() error {
			_, e := api.Create(context.Background(), policies.CreateOpts{Spec: json.RawMessage(`{}`)})
			return e
		}},
		{"missing-spec", func() error { _, e := api.Create(context.Background(), policies.CreateOpts{Name: "policy"}); return e }},
		{"null-spec", func() error {
			_, e := api.Create(context.Background(), policies.CreateOpts{Name: "policy", Spec: json.RawMessage(`null`)})
			return e
		}},
		{"array-spec", func() error {
			_, e := api.Create(context.Background(), policies.CreateOpts{Name: "policy", Spec: json.RawMessage(`[]`)})
			return e
		}},
		{"core-name", func() error {
			_, e := api.Create(context.Background(), base, policies.WithCreateField("name", "replacement"))
			return e
		}},
		{"core-spec", func() error {
			_, e := api.Create(context.Background(), base, policies.WithCreateField("spec", map[string]any{}))
			return e
		}},
		{"response-id", func() error {
			_, e := api.Create(context.Background(), base, policies.WithCreateField("id", "unsafe"))
			return e
		}},
		{"owned-version", func() error {
			_, e := api.Create(context.Background(), base, policies.WithCreateHeader("openstack-api-version", "clustering 1.9"))
			return e
		}},
		{"owned-auth", func() error {
			_, e := api.Create(context.Background(), base, policies.WithCreateHeader("X-Auth-Token", "override"))
			return e
		}},
		{"query", func() error {
			_, e := api.Create(context.Background(), base, request.WithQuery[policies.CreateOpts]("name", "other"))
			return e
		}},
		{"nil-option", func() error { _, e := api.Create(context.Background(), base, nil); return e }},
		{"empty-update", func() error {
			_, e := api.Update(context.Background(), resource.Name("name"), policies.UpdateOpts{})
			return e
		}},
		{"update-spec", func() error {
			_, e := api.Update(context.Background(), resource.Name("name"), policies.UpdateOpts{}, policies.WithUpdateField("spec", map[string]any{}))
			return e
		}},
		{"empty-name", func() error {
			_, e := api.Update(context.Background(), resource.Name("name"), policies.UpdateOpts{}, policies.WithUpdateName(""))
			return e
		}},
		{"validate-null", func() error {
			_, e := api.Validate(context.Background(), policies.ValidateOpts{}, policies.WithValidateSpec(nil))
			return e
		}},
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

func TestClusteringPolicyAcceptedMalformedMutationDoesNotResend(t *testing.T) {
	for _, response := range []string{`{}`, `{"policy":null}`, `{"policy":[]}`, `{"policy":{"spec":[]}}`, `{"policy":`} {
		t.Run(response, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("POST /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-OpenStack-Request-ID", "accepted")
				testcloud.JSON(w, 201, response)
			})
			value, err := policies.New(cloud.Client("clustering", "/senlin/v1")).Create(context.Background(), policies.CreateOpts{Name: "policy", Spec: json.RawMessage(`{}`)})
			var evidence *resource.ResponseError
			var operation *resource.OperationError
			if value != nil || !errors.As(err, &evidence) || !errors.As(err, &operation) || evidence.StatusCode != 201 || evidence.Header.Get("X-OpenStack-Request-ID") != "accepted" || string(evidence.Body) != response || calls.Load() != 1 {
				t.Fatal(value, err, evidence, calls.Load())
			}
		})
	}
}

func TestClusteringPolicyRejectsWrongSuccessCodeAndPreservesHTTPError(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"policy":{"id":"wrong-code"}}`) })
	_, err := policies.New(cloud.Client("clustering", "/senlin/v1")).Create(context.Background(), policies.CreateOpts{Name: "policy", Spec: json.RawMessage(`{}`)})
	if !gophercloud.ResponseCodeIs(err, 200) {
		t.Fatal(err)
	}
}

func TestClusteringPolicyListShortPagesRetainFiltersAndSnapshot(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "2" || q.Get("name") != "selected" || q.Get("type") != "senlin.policy.scaling-1.0" || q.Get("sort") != "created_at:desc,name" || q.Get("global_project") != "false" || q.Get("vendor") != "retain" {
			t.Error(q)
		}
		switch calls.Add(1) {
		case 1:
			if q.Has("marker") {
				t.Error(q)
			}
			w.Header().Set("X-Page", "one")
			testcloud.JSON(w, 200, `{"policies":[{"id":"p1","name":"selected","spec":{"number":9007199254740993}}]}`)
		case 2:
			if q.Get("marker") != "p1" {
				t.Error(q)
			}
			testcloud.JSON(w, 200, `{"policies":[{"id":"p2","name":"selected"}]}`)
		case 3:
			if q.Get("marker") != "p2" {
				t.Error(q)
			}
			testcloud.JSON(w, 200, `{"policies":[]}`)
		default:
			t.Error("extra request", calls.Load())
		}
	})
	global := false
	option := policies.WithListOptions(policies.ListOpts{Limit: 2, Name: "selected", Type: "senlin.policy.scaling-1.0", Sort: "created_at:desc,name", GlobalProject: &global})
	global = true
	options := []policies.ListOption{option, policies.WithListQuery("vendor", "retain")}
	iterator := policies.New(cloud.Client("clustering", "/senlin/v1")).List(context.Background(), options...)
	options[0] = policies.WithListGlobalProject(true)
	if calls.Load() != 0 {
		t.Fatal("eager list")
	}
	for range 2 {
		var values []*policies.Policy
		for value, err := range iterator {
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, value)
			if value.ID == "p1" {
				value.ID = "consumer-mutated"
			}
		}
		if len(values) != 2 || values[1].ID != "p2" || values[0].Header.Get("X-Page") != "one" || calls.Load() != 3 {
			t.Fatal(values, calls.Load())
		}
		calls.Store(0)
	}
}

func TestClusteringPolicyListDefaultBreakAndInvalidExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"policies":[{"id":"p1"},{"id":"p2"}],"links":[{"rel":"next","href":"?marker=p2"}]}`)
	})
	api := policies.New(cloud.Client("clustering", "/senlin/v1"))
	for value, err := range api.List(context.Background()) {
		if err != nil || value.ID != "p1" {
			t.Fatal(value, err)
		}
		break
	}
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
	invalid := []policies.ListOption{policies.WithListQuery("name", "override"), policies.WithListOptions(policies.ListOpts{Limit: -1}), policies.WithListOptions(policies.ListOpts{Sort: "name:sideways"}), request.WithField[policies.ListOpts]("vendor", false), request.WithHeader[policies.ListOpts]("X-Auth-Token", "ignored")}
	for _, option := range invalid {
		if _, err := api.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringPolicyFindDefaultsAmbiguityAndDeleteConflict(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists, deletes atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/policies/missing", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 404, `{"error":"missing"}`)
	})
	cloud.Mux.HandleFunc("GET /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		switch r.URL.Query().Get("name") {
		case "duplicate":
			testcloud.JSON(w, 200, `{"policies":[{"id":"one","name":"duplicate"},{"id":"two","name":"duplicate"}]}`)
		case "selected":
			testcloud.JSON(w, 200, `{"policies":[{"id":"returned-id","name":"selected"}]}`)
		default:
			testcloud.JSON(w, 200, `{"policies":[]}`)
		}
	})
	cloud.Mux.HandleFunc("DELETE /senlin/v1/policies/returned-id", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.Header().Set("X-Request-ID", "conflict")
		testcloud.JSON(w, 409, `{"error":"policy attached"}`)
	})
	cloud.Mux.HandleFunc("DELETE /senlin/v1/policies/missing", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		testcloud.JSON(w, 404, `{"error":"missing"}`)
	})
	api := policies.New(cloud.Client("clustering", "/senlin/v1"))
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
	if err := api.Delete(context.Background(), resource.Name("selected")); !gophercloud.ResponseCodeIs(err, 409) || !strings.Contains(err.Error(), "policy attached") {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), resource.ID("missing")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if gets.Load() != 2 || lists.Load() != 4 || deletes.Load() != 3 {
		t.Fatal(gets.Load(), lists.Load(), deletes.Load())
	}
}

func TestClusteringPolicyContextAndVersionHeadersPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("unexpected HTTP", r.URL) })
	client := cloud.Client("clustering", "/senlin/v1")
	api := policies.New(client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Create(ctx, policies.CreateOpts{Name: "policy", Spec: json.RawMessage(`{}`)}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := api.Get(ctx, "policy-id"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := api.All(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	client.Microversion = "1.2"
	client.MoreHeaders = map[string]string{"openstack-api-version": "clustering 1.1"}
	if _, err := api.Validate(context.Background(), policies.ValidateOpts{Spec: json.RawMessage(`{}`)}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringPolicyLocalFiltersSnapshotExactNumbersAndNull(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "" {
			t.Error("local filters leaked to wire", r.URL)
		}
		testcloud.JSON(w, 200, `{"policies":[{"id":"yes","project":"project-1","domain":null,"spec":{"properties":{"number":9007199254740993,"enabled":false,"extra":1}},"data":{"ordered":[1,2]}},{"id":"rounded","project":"project-1","spec":{"properties":{"number":9007199254740992,"enabled":false}},"data":{"ordered":[1,2]}},{"id":"numeric-bool","project":"project-1","spec":{"properties":{"number":9007199254740993,"enabled":0}},"data":{"ordered":[1,2]}},{"id":"reversed","project":"project-1","spec":{"properties":{"number":9007199254740993,"enabled":false}},"data":{"ordered":[2,1]}},{"id":"missing-null","project":"project-1","spec":{"properties":{"number":9007199254740993,"enabled":false}},"data":{"ordered":[1,2]}}]}`)
	})
	input := map[string]any{"properties": map[string]any{"number": json.Number("9007199254740993.0"), "enabled": false}}
	option := policies.WithListFilter("spec", input)
	input["properties"].(map[string]any)["number"] = 2
	api := policies.New(cloud.Client("clustering", "/senlin/v1"))
	for range 2 {
		values, err := api.All(context.Background(), option, policies.WithListFilter("project_id", "project-1"), policies.WithListFilter("domain_id", nil), policies.WithListFilter("data", map[string]any{"ordered": []int{1, 2}}))
		if err != nil || len(values) != 2 || values[0].ID != "yes" || values[1].ID != "missing-null" {
			t.Fatal(values, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
	invalid := []policies.ListOption{policies.WithListFilter("unknown", 1), policies.WithListFilter("spec", make(chan int)), policies.WithListQuery("spec", "wire"), request.WithArgument[policies.ListOpts]("policies.local_filters", "wrong-type"), request.WithArgument[policies.ListOpts]("policies.local_filters", map[string]json.RawMessage{"spec": json.RawMessage(`{`)})}
	for _, option := range invalid {
		if _, err := api.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringPolicyDeleteSuccessAndMissingResponseIDCannotBeUsed(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, deletes atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		testcloud.JSON(w, 200, `{"policies":[{"id":null,"name":"null-id"}]}`)
	})
	cloud.Mux.HandleFunc("DELETE /senlin/v1/policies/explicit-id", func(w http.ResponseWriter, r *http.Request) { deletes.Add(1); w.WriteHeader(204) })
	api := policies.New(cloud.Client("clustering", "/senlin/v1"))
	if err := api.Delete(context.Background(), resource.ID("explicit-id")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), resource.Name("null-id")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.Update(context.Background(), resource.Name("null-id"), policies.UpdateOpts{}, policies.WithUpdateName("new-name")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if lists.Load() != 2 || deletes.Load() != 1 {
		t.Fatal(lists.Load(), deletes.Load())
	}
}

func TestClusteringPolicyNameGrammarAndInFlightListCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cloud.Mux.HandleFunc("GET /senlin/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"policies":[{"id":"one"}],"links":[{"rel":"next","href":"?marker=one"}]}`)
	})
	api := policies.New(cloud.Client("clustering", "/senlin/v1"))
	for _, name := range []string{"1first", "한글", "space name", "bad/name", strings.Repeat("a", 255)} {
		if _, err := api.Create(context.Background(), policies.CreateOpts{Name: name, Spec: json.RawMessage(`{}`)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(name, err)
		}
	}
	var yielded int
	for value, err := range api.List(ctx) {
		if yielded == 0 {
			if err != nil || value.ID != "one" {
				t.Fatal(value, err)
			}
			cancel()
		} else if value != nil || !errors.Is(err, context.Canceled) {
			t.Fatal(value, err)
		}
		yielded++
	}
	if yielded != 2 || calls.Load() != 1 {
		t.Fatal(yielded, calls.Load())
	}
}
