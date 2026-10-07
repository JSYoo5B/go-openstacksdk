package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/profiles"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestClusteringProfilesCRUDExactPathsBodiesAndResponseEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/catalog/v1")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/proxy/tenant/senlin/v1")
	client.Microversion = "1.2"
	var methods []string
	response := `{"profile":{"id":"response-id","name":"returned-name","type":"os.nova.server-1.0","project":"project-id","domain":null,"user":"user-id","spec":{"properties":{"large":9007199254740993,"false":false},"version":1.0000000000000001},"metadata":{"label":"original"},"created_at":"2016-01-01T12:00:00.000000","updated_at":null,"future":9007199254740995}}`
	cloud.Mux.HandleFunc("/proxy/tenant/senlin/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method != "POST" || r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "clustering 1.2" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Custom") != "captured" {
			t.Error(r.Method, r.URL, r.Header)
		}
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body) != 1 || string(body["profile"]["name"]) != `"original"` || string(body["profile"]["spec"]) != `{"properties":{"large":9007199254740993,"false":false}}` || string(body["profile"]["metadata"]) != `{"flag":false}` || string(body["profile"]["vendor"]) != `{"count":9007199254740997}` {
			t.Error(body)
		}
		w.Header().Set("X-Request-Id", "created-profile")
		testcloud.JSON(w, 201, response)
	})
	cloud.Mux.HandleFunc("/proxy/tenant/senlin/v1/profiles/short-id", func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method != "GET" {
			t.Error(r.Method)
		}
		testcloud.JSON(w, 200, response)
	})
	cloud.Mux.HandleFunc("/proxy/tenant/senlin/v1/profiles/direct-name", func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		switch r.Method {
		case "PATCH":
			var body map[string]map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body["profile"]) != 2 || string(body["profile"]["name"]) != `"renamed"` || string(body["profile"]["metadata"]) != `{}` {
				t.Error(body)
			}
			testcloud.JSON(w, 200, response)
		case "DELETE":
			w.WriteHeader(204)
		default:
			t.Error("unexpected lookup", r.Method)
		}
	})
	spec := json.RawMessage(`{"properties":{"large":9007199254740993,"false":false}}`)
	metadata := map[string]bool{"flag": false}
	option := profiles.WithCreateOptions(profiles.CreateOpts{Name: "original", Spec: spec})
	metadataOption := profiles.WithCreateMetadata(metadata)
	spec[2], metadata["flag"] = 'X', true
	api := profiles.New(client)
	value, err := api.Create(context.Background(), profiles.CreateOpts{}, option, metadataOption, profiles.WithCreateHeader("X-Custom", "captured"), profiles.WithCreateField("vendor", map[string]json.Number{"count": "9007199254740997"}))
	if err != nil || value.ID != "response-id" || value.Name != "returned-name" || value.DomainID != nil || value.ProjectID != "project-id" || value.UserID != "user-id" || string(value.Spec["version"]) != "1.0000000000000001" || string(value.UserMetadata["label"]) != `"original"` || string(value.Body["future"]) != "9007199254740995" || value.Header.Get("X-Request-Id") != "created-profile" || value.StatusCode != 201 || value.UpdatedAt != nil {
		t.Fatal(value, err)
	}
	if _, err := api.Get(context.Background(), "short-id"); err != nil {
		t.Fatal(err)
	}
	name := "renamed"
	updateOption := profiles.WithUpdateOptions(profiles.UpdateOpts{Name: &name, Metadata: json.RawMessage(`{}`)})
	name = "mutated"
	if _, err := api.Update(context.Background(), resource.ID("direct-name"), profiles.UpdateOpts{}, updateOption); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), resource.ID("direct-name")); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(methods, []string{"POST", "GET", "PATCH", "DELETE"}) {
		t.Fatal(methods)
	}
}

func TestClusteringProfilesMetadataPresenceAndOptionsReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var present []string
	cloud.Mux.HandleFunc("POST /v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		raw, exists := body["profile"]["metadata"]
		if !exists {
			present = append(present, "omitted")
		} else {
			present = append(present, string(raw))
		}
		testcloud.JSON(w, 201, `{"profile":{"id":"created","name":"name"}}`)
	})
	cloud.Mux.HandleFunc("PATCH /v1/profiles/profile", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		raw, exists := body["profile"]["metadata"]
		if !exists {
			present = append(present, "omitted")
		} else {
			present = append(present, string(raw))
		}
		testcloud.JSON(w, 200, `{"profile":{"id":"profile","metadata":null}}`)
	})
	api := profiles.New(cloud.Client("clustering", "/v1"))
	base := profiles.CreateOpts{Name: "name", Spec: json.RawMessage(`{}`)}
	for _, option := range []profiles.CreateOption{profiles.WithCreateOptions(base), profiles.WithCreateMetadata(nil), profiles.WithCreateMetadata(map[string]any{})} {
		if _, err := api.Create(context.Background(), base, option); err != nil {
			t.Fatal(err)
		}
	}
	name := "renamed"
	for _, option := range []profiles.UpdateOption{profiles.WithUpdateOptions(profiles.UpdateOpts{Name: &name}), profiles.WithUpdateMetadata(nil), profiles.WithUpdateMetadata(map[string]any{})} {
		if _, err := api.Update(context.Background(), resource.ID("profile"), profiles.UpdateOpts{Name: &name}, option); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(present, []string{"omitted", "null", "{}", "omitted", "null", "{}"}) {
		t.Fatal(present)
	}
	var applied *string
	option := profiles.WithUpdateOptions(profiles.UpdateOpts{Name: &name})
	capture := func(config *request.Config[profiles.UpdateOpts]) error { applied = config.Options.Name; return nil }
	for range 2 {
		if _, err := api.Update(context.Background(), resource.ID("profile"), profiles.UpdateOpts{}, option, capture); err != nil {
			t.Fatal(err)
		}
		if applied == nil || *applied != "renamed" {
			t.Fatal(applied)
		}
		*applied = "caller-mutation"
	}
}

func TestClusteringProfilesValidateVersionBodyAndNullableIdentity(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v1/profiles/validate", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "clustering 1.2" {
			t.Error(r.Header)
		}
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body["profile"]) != 1 || string(body["profile"]["spec"]) != `{"properties":{"count":9007199254740993},"type":"os.nova.server","version":"1.0"}` {
			t.Error(body)
		}
		w.Header().Set("X-Request-Id", "validation-evidence")
		testcloud.JSON(w, 200, `{"profile":{"id":null,"name":"validated_profile","metadata":null,"domain":null,"spec":{"version":1.0,"properties":{"count":9007199254740993}},"created_at":null,"updated_at":null}}`)
	})
	client := cloud.Client("clustering", "/v1")
	api := profiles.New(client)
	input := map[string]any{"type": "os.nova.server", "version": "1.0", "properties": map[string]json.Number{"count": "9007199254740993"}}
	option := profiles.WithValidateSpec(input)
	input["type"] = "mutated"
	for _, version := range []string{"", "1.0", "1.1", "latest"} {
		client.Microversion = version
		if _, err := api.Validate(context.Background(), profiles.ValidateOpts{}, option); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
	}
	for _, version := range []string{"2.0", "1.02"} {
		client.Microversion = version
		if _, err := api.Validate(context.Background(), profiles.ValidateOpts{}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(version, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("version gates performed HTTP")
	}
	client.Microversion = "1.2"
	value, err := api.Validate(context.Background(), profiles.ValidateOpts{}, option)
	if err != nil || value.ID != "" || string(value.Body["id"]) != "null" || value.CreatedAt != nil || value.UserMetadata != nil || string(value.Spec["properties"]) != `{"count":9007199254740993}` || value.Header.Get("X-Request-Id") != "validation-evidence" || value.StatusCode != 200 || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
	client.MoreHeaders = map[string]string{"openstack-api-version": "clustering 1.1"}
	if _, err := api.Validate(context.Background(), profiles.ValidateOpts{}, option); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
}

func TestClusteringProfilesNameUpdateFindDefaultsAndAmbiguity(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, updates atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		switch r.URL.Query().Get("name") {
		case "compute.a":
			testcloud.JSON(w, 200, `{"profiles":[{"id":"wrong","name":"computeXa"},{"id":"stable","name":"compute.a"}]}`)
		case "duplicate":
			testcloud.JSON(w, 200, `{"profiles":[{"id":"a","name":"duplicate"},{"id":"b","name":"duplicate"}]}`)
		default:
			testcloud.JSON(w, 200, `{"profiles":[]}`)
		}
	})
	cloud.Mux.HandleFunc("PATCH /v1/profiles/stable", func(w http.ResponseWriter, r *http.Request) {
		updates.Add(1)
		testcloud.JSON(w, 200, `{"profile":{"id":"stable","name":"renamed"}}`)
	})
	api := profiles.New(cloud.Client("clustering", "/v1"))
	name := "renamed"
	value, err := api.Update(context.Background(), resource.Name("compute.a"), profiles.UpdateOpts{Name: &name})
	if err != nil || value.ID != "stable" || lookups.Load() != 1 || updates.Load() != 1 {
		t.Fatal(value, err, lookups.Load(), updates.Load())
	}
	if value, err := api.Find(context.Background(), resource.Name("missing")); err != nil || value != nil {
		t.Fatal(value, err)
	}
	if _, err := api.Find(context.Background(), resource.Name("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Resources.Find(context.Background(), resource.Name("missing")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Update(context.Background(), resource.Name("duplicate"), profiles.UpdateOpts{Name: &name}); !errors.Is(err, resource.ErrAmbiguous) || updates.Load() != 1 {
		t.Fatal(err, updates.Load())
	}
	if _, err := api.Find(context.Background(), resource.Name("duplicate")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
}

func TestClusteringProfilesDeleteIgnoresOnlyAbsence(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/profiles/{identity}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("identity") {
		case "missing":
			testcloud.JSON(w, 404, `{"error":{"message":"missing"}}`)
		case "denied":
			testcloud.JSON(w, 403, `{"error":{"message":"denied"}}`)
		case "used":
			testcloud.JSON(w, 409, `{"error":{"message":"in use"}}`)
		default:
			w.WriteHeader(204)
		}
	})
	api := profiles.New(cloud.Client("clustering", "/v1"))
	if err := api.Delete(context.Background(), resource.ID("missing")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if value, err := api.Find(context.Background(), resource.ID("missing")); value != nil || err != nil {
		t.Fatal(value, err)
	}
	for identity, code := range map[string]int{"denied": 403, "used": 409} {
		if err := api.Delete(context.Background(), resource.ID(identity)); !gophercloud.ResponseCodeIs(err, code) {
			t.Fatal(identity, err)
		}
		if _, err := api.Get(context.Background(), identity); !gophercloud.ResponseCodeIs(err, code) {
			t.Fatal(identity, err)
		}
	}
}

func TestClusteringProfilesListShortPagesLocalFiltersAndSnapshot(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "5" || q.Get("name") != "selected" || q.Get("type") != "os.nova.server" || q.Get("sort") != "created_at:desc,name" || q.Get("global_project") != "false" || q.Get("vendor") != "retained" || q.Has("metadata") || q.Has("spec") || q.Has("project_id") {
			t.Error(q)
		}
		switch calls.Add(1) {
		case 1:
			if q.Has("marker") {
				t.Error(q)
			}
			w.Header().Set("X-Request-Id", "first-page")
			testcloud.JSON(w, 200, `{"profiles":[{"id":"selected-1","name":"selected","project":"p","metadata":{"sub":{"large":9007199254740993,"ratio":1.0,"extra":false}}},{"id":"filtered-1","name":"selected","project":"p","metadata":{"sub":{"large":9007199254740992,"ratio":1.0}}}]}`)
		case 2:
			if q.Get("marker") != "filtered-1" {
				t.Error(q)
			}
			w.Header().Set("X-Request-Id", "second-page")
			testcloud.JSON(w, 200, `{"profiles":[{"id":"selected-2","name":"selected","project":"p","metadata":{"sub":{"large":9007199254740993,"ratio":1e0}}}]}`)
		case 3:
			if q.Get("marker") != "selected-2" {
				t.Error(q)
			}
			testcloud.JSON(w, 200, `{"profiles":[]}`)
		default:
			t.Error("unexpected request", calls.Load())
		}
	})
	global := false
	filter := map[string]any{"sub": map[string]json.Number{"large": "9007199254740993", "ratio": "1"}}
	option := profiles.WithListOptions(profiles.ListOpts{Limit: 5, Name: "selected", Type: "os.nova.server", Sort: "created_at:desc,name", GlobalProject: &global})
	filterOption := profiles.WithListFilter("metadata", filter)
	global = true
	filter["sub"] = map[string]any{"large": 0}
	var applied *bool
	capture := func(config *request.Config[profiles.ListOpts]) error {
		applied = config.Options.GlobalProject
		return nil
	}
	options := []profiles.ListOption{option, filterOption, profiles.WithListFilter("project_id", "p"), profiles.WithListQuery("vendor", "retained"), capture}
	api := profiles.New(cloud.Client("clustering", "/v1"))
	iterator := api.List(context.Background(), options...)
	options[0] = profiles.WithListGlobalProject(true)
	if calls.Load() != 0 {
		t.Fatal("eager list")
	}
	for range 2 {
		var ids []string
		for value, err := range iterator {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
			if value.Header.Get("X-Request-Id") == "" {
				t.Fatal(value)
			}
			value.ID = "consumer-mutation"
		}
		if !reflect.DeepEqual(ids, []string{"selected-1", "selected-2"}) || calls.Load() != 3 || applied == nil || *applied {
			t.Fatal(ids, calls.Load(), applied)
		}
		*applied = true
		calls.Store(0)
	}
}

func TestClusteringProfilesLocalJSONFilterTypesArraysAndNull(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"profiles":[{"id":"match","metadata":{"flag":false,"array":[{"x":1}],"nested":{"key":null,"extra":true}}},{"id":"extra-array-key","metadata":{"flag":false,"array":[{"x":1,"extra":2}],"nested":{"key":null}}},{"id":"wrong-type","metadata":{"flag":0,"array":[{"x":1}],"nested":{"key":null}}},{"id":"empty","metadata":{}},{"id":"null","metadata":null},{"id":"missing"}]}`)
	})
	api := profiles.New(cloud.Client("clustering", "/v1"))
	filter := json.RawMessage(`{"flag":false,"array":[{"x":1.0}],"nested":{"key":null}}`)
	values, err := api.All(context.Background(), profiles.WithListFilter("metadata", filter))
	if err != nil || len(values) != 1 || values[0].ID != "match" {
		t.Fatal(values, err)
	}
	values, err = api.All(context.Background(), profiles.WithListFilter("metadata", nil))
	if err != nil || len(values) != 2 || values[0].ID != "null" || values[1].ID != "missing" {
		t.Fatal(values, err)
	}
	values, err = api.All(context.Background(), profiles.WithListFilter("metadata", map[string]any{}))
	if err != nil || len(values) != 3 || values[0].ID != "match" || values[2].ID != "wrong-type" {
		t.Fatal(values, err)
	}
}

func TestClusteringProfilesDefaultsInvalidOptionsAndEmptyUpdatePreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"profiles":[]}`)
	})
	api := profiles.New(cloud.Client("clustering", "/v1"))
	values, err := api.All(context.Background())
	if err != nil || values == nil || len(values) != 0 || calls.Load() != 1 {
		t.Fatal(values, err, calls.Load())
	}
	for _, option := range []profiles.ListOption{
		nil, profiles.WithListOptions(profiles.ListOpts{Limit: -1}), profiles.WithListOptions(profiles.ListOpts{Marker: "bad/marker"}),
		profiles.WithListOptions(profiles.ListOpts{Sort: "name:sideways"}), profiles.WithListOptions(profiles.ListOpts{Sort: "name,,type"}),
		profiles.WithListQuery("metadata", "wire-filter"), profiles.WithListQuery("global_project", "true"), profiles.WithListQuery("name", "hidden"),
		profiles.WithListFilter("unknown", true), profiles.WithListFilter("metadata", make(chan int)),
		request.WithField[profiles.ListOpts]("vendor", true), request.WithHeader[profiles.ListOpts]("X-Auth-Token", "value"), request.WithArgument[profiles.ListOpts]("unknown", true),
	} {
		if _, err := api.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.Resources.All(context.Background(), resource.WithQuery("sort", "name:sideways")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.Resources.All(context.Background(), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := api.Update(context.Background(), resource.Name("would-require-lookup"), profiles.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("invalid options performed HTTP", calls.Load())
	}
}

func TestClusteringProfilesMutationSchemaAndProtectedExtensionsPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("unexpected request", r.URL) })
	client := cloud.Client("clustering", "/v1")
	client.Microversion = "1.2"
	api := profiles.New(client)
	base := profiles.CreateOpts{Name: "valid", Spec: json.RawMessage(`{}`)}
	for _, value := range []profiles.CreateOpts{
		{}, {Name: "valid", Spec: json.RawMessage(`null`)}, {Name: "valid", Spec: json.RawMessage(`[]`)}, {Name: "valid", Spec: json.RawMessage(`{`)},
		{Name: "1invalid", Spec: json.RawMessage(`{}`)}, {Name: strings.Repeat("a", 255), Spec: json.RawMessage(`{}`)}, {Name: "valid", Spec: json.RawMessage(`{}`), Metadata: json.RawMessage(`false`)},
	} {
		if _, err := api.Create(context.Background(), value); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	for _, option := range []profiles.CreateOption{
		nil, profiles.WithCreateField("metadata", nil), profiles.WithCreateField("spec", map[string]any{}), profiles.WithCreateField("id", "owned"), profiles.WithCreateField("project_id", "owned"),
		profiles.WithCreateHeader("OpenStack-API-Version", "clustering 1.2"), profiles.WithCreateHeader("X-Auth-Token", "owned"), profiles.WithCreateHeader("Content-Type", "text/plain"),
		request.WithQuery[profiles.CreateOpts]("query", "owned"), request.WithArgument[profiles.CreateOpts]("unknown", true),
	} {
		if _, err := api.Create(context.Background(), base, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	name := "renamed"
	for _, option := range []profiles.UpdateOption{profiles.WithUpdateField("metadata", nil), profiles.WithUpdateField("spec", map[string]any{}), profiles.WithUpdateField("type", "newtype"), profiles.WithUpdateOptions(profiles.UpdateOpts{Metadata: json.RawMessage(`1`)})} {
		if _, err := api.Update(context.Background(), resource.Name("needs-lookup"), profiles.UpdateOpts{Name: &name}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.Validate(context.Background(), profiles.ValidateOpts{Spec: json.RawMessage(`null`)}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.Validate(context.Background(), profiles.ValidateOpts{Spec: json.RawMessage(`{}`)}, profiles.WithValidateField("spec", map[string]any{})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringProfilesAcceptedMalformedMutationNeverResends(t *testing.T) {
	for _, operation := range []string{"Create", "Update", "Validate", "Get"} {
		for _, body := range []string{`{`, `{}`, `{"profile":null}`, `{"profile":[]}`, `{"profile":{"spec":true}}`} {
			t.Run(operation+"/"+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				code := 200
				if operation == "Create" {
					code = 201
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-Id", "accepted-mutation")
					testcloud.JSON(w, code, body)
				})
				client := cloud.Client("clustering", "/v1")
				client.Microversion = "1.2"
				api := profiles.New(client)
				var err error
				switch operation {
				case "Create":
					_, err = api.Create(context.Background(), profiles.CreateOpts{Name: "valid", Spec: json.RawMessage(`{}`)})
				case "Update":
					name := "valid"
					_, err = api.Update(context.Background(), resource.ID("id"), profiles.UpdateOpts{Name: &name})
				case "Validate":
					_, err = api.Validate(context.Background(), profiles.ValidateOpts{Spec: json.RawMessage(`{}`)})
				case "Get":
					_, err = api.Get(context.Background(), "id")
				}
				var responseErr *resource.ResponseError
				if !errors.As(err, &responseErr) || responseErr.StatusCode != code || string(responseErr.Body) != body || responseErr.Header.Get("X-Request-Id") != "accepted-mutation" || calls.Load() != 1 {
					t.Fatal(err, responseErr, calls.Load())
				}
			})
		}
	}
}

func TestClusteringProfilesExplicitSuccessCodesAndNativeErrors(t *testing.T) {
	for _, code := range []int{200, 202, 400, 403, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("POST /v1/profiles", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, code, `{"profile":{"id":"id"}}`)
			})
			_, err := profiles.New(cloud.Client("clustering", "/v1")).Create(context.Background(), profiles.CreateOpts{Name: "valid", Spec: json.RawMessage(`{}`)})
			var responseErr *resource.ResponseError
			if !gophercloud.ResponseCodeIs(err, code) || errors.As(err, &responseErr) || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
		})
	}
}

func TestClusteringProfilesPaginationGuardsAndConsumerControl(t *testing.T) {
	for _, mode := range []string{"follow", "break", "cancel", "cycle", "foreign", "filter-change", "missing-marker"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /v1/profiles", func(w http.ResponseWriter, r *http.Request) {
				count := calls.Add(1)
				w.Header().Set("X-Request-Id", "paging-evidence")
				if mode == "missing-marker" {
					testcloud.JSON(w, 200, `{"profiles":[{"name":"cannot-use-name-as-marker"}]}`)
					return
				}
				if mode == "cycle" {
					testcloud.JSON(w, 200, `{"profiles":[{"id":"same"}]}`)
					return
				}
				if count == 1 {
					link := "?marker=next"
					if mode == "foreign" {
						link = "https://foreign.invalid/v1/profiles?marker=next"
					}
					if mode == "filter-change" {
						link = "?marker=next&vendor=changed"
					}
					w.Header().Set("Link", "<"+link+">; rel=\"next\"")
					testcloud.JSON(w, 200, `{"profiles":[{"id":"first"}]}`)
					return
				}
				if r.URL.Query().Get("vendor") != "retained" || r.URL.Query().Get("marker") != "next" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"profiles":[]}`)
			})
			api := profiles.New(cloud.Client("clustering", "/v1"))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var resultErr error
			for _, err := range api.List(ctx, profiles.WithListOptions(profiles.ListOpts{Limit: 5}), profiles.WithListQuery("vendor", "retained")) {
				if err != nil {
					resultErr = err
					break
				}
				if mode == "break" {
					break
				}
				if mode == "cancel" {
					cancel()
				}
			}
			switch mode {
			case "follow":
				if resultErr != nil || calls.Load() != 2 {
					t.Fatal(resultErr, calls.Load())
				}
			case "break":
				if resultErr != nil || calls.Load() != 1 {
					t.Fatal(resultErr, calls.Load())
				}
			case "cancel":
				if !errors.Is(resultErr, context.Canceled) || calls.Load() != 1 {
					t.Fatal(resultErr, calls.Load())
				}
			case "cycle":
				var cycle *resource.PaginationCycleError
				if !errors.As(resultErr, &cycle) || calls.Load() != 2 {
					t.Fatal(resultErr, calls.Load())
				}
			default:
				var responseErr *resource.ResponseError
				if !errors.Is(resultErr, resource.ErrInvalidOption) || !errors.As(resultErr, &responseErr) || responseErr.Header.Get("X-Request-Id") != "paging-evidence" || calls.Load() != 1 {
					t.Fatal(resultErr, calls.Load())
				}
			}
		})
	}
}

func TestClusteringProfilesContextAndClientValidationBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("unexpected request") })
	client := cloud.Client("clustering", "/v1")
	client.Microversion = "1.2"
	api := profiles.New(client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	name := "valid"
	operations := []func() error{
		func() error { _, err := api.Get(ctx, "id"); return err },
		func() error {
			_, err := api.Create(ctx, profiles.CreateOpts{Name: name, Spec: json.RawMessage(`{}`)})
			return err
		},
		func() error {
			_, err := api.Update(ctx, resource.Name("name"), profiles.UpdateOpts{Name: &name})
			return err
		},
		func() error {
			_, err := api.Validate(ctx, profiles.ValidateOpts{Spec: json.RawMessage(`{}`)})
			return err
		},
		func() error { _, err := api.All(ctx); return err },
		func() error { return api.Delete(ctx, resource.ID("id")) },
	}
	for _, operation := range operations {
		if err := operation(); !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	client.Type = "compute"
	if _, err := api.Get(context.Background(), "id"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := profiles.New(nil).Create(context.Background(), profiles.CreateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	var absent *profiles.API
	if absent.RawClient() != nil {
		t.Fatal("nil API client")
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}
