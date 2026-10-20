package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/db/v1/databases"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/instances"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const troveInstanceBody = `{"instance":{"id":"inst-1","name":"db","status":"BUILD","hostname":"db.example.invalid","created":"2026-01-02T03:04:05","updated":"2026-01-02T03:04:06","flavor":{"id":"7"},"volume":{"size":0,"used":0.5},"datastore":{"type":"mysql","version":"8.0","version_id":"ver-1"},"ip":["192.0.2.1"],"addresses":[{"type":"private","address":"198.51.100.1"}],"vendor_extra":1}}`

func troveInstanceAPI(cloud *testcloud.Cloud) *instances.API {
	return instances.New(cloud.Client("database", "/trove"))
}

func TestTroveInstanceCreateSendsCompleteBodyAndAcknowledgesWithoutWaiting(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/trove/instances" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
			return
		}
		got := troveRawInner(t, r, "instance")
		want := map[string]string{
			"flavorRef":         `"7"`,
			"volume":            `{"size":5,"type":"ssd"}`,
			"name":              `"db"`,
			"availability_zone": `"az1"`,
			"configuration":     `"cfg-1"`,
			"datastore":         `{"type":"mysql","version":"8.0"}`,
			"nics":              `[{"net-id":"net-1","v4-fixed-ip":"192.0.2.9"}]`,
			"databases":         `[{"character_set":"utf8","name":"app"}]`,
			"users":             `[{"name":"writer","password":"example-password"}]`,
			"vendor":            `{"enabled":false}`,
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 200, troveInstanceBody)
	})
	value, err := troveInstanceAPI(cloud).Create(context.Background(), instances.CreateOpts{
		FlavorRef: "7", Size: 5, VolumeType: "ssd", Name: "db", AvailabilityZone: "az1", Configuration: "cfg-1",
		Datastore: &instances.DatastoreOpts{Type: "mysql", Version: "8.0"},
		Networks:  []instances.NetworkOpts{{UUID: "net-1", V4FixedIP: "192.0.2.9"}},
		Databases: databases.BatchCreateOpts{{Name: "app", CharSet: "utf8"}},
		Users:     users.BatchCreateOpts{{Name: "writer", Password: "example-password"}},
	}, instances.WithCreateField("vendor", map[string]bool{"enabled": false}))
	if err != nil || value == nil || value.ID != "inst-1" || value.Status != "BUILD" || value.Flavor.ID != "7" || value.Volume.Size != 0 || value.Volume.Used != 0.5 ||
		value.Datastore.VersionID != "ver-1" || value.Created.IsZero() || len(value.Addresses) != 1 || value.Addresses[0].Address != "198.51.100.1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("acknowledgement must not trigger status polling: calls=%d", calls.Load())
	}
}

func TestTroveInstanceCreateMinimalBodyOmitsOptionalFields(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /trove/instances", func(w http.ResponseWriter, r *http.Request) {
		got := troveRawInner(t, r, "instance")
		if want := map[string]string{"flavorRef": `"1"`, "volume": `{"size":1}`}; !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 200, troveInstanceBody)
	})
	if _, err := troveInstanceAPI(cloud).Create(context.Background(), instances.CreateOpts{FlavorRef: "1", Size: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestTroveInstanceCreateAcceptsOnly200(t *testing.T) {
	troveStatusMatrix(t, "POST", "/trove/instances", troveInstanceBody, []int{200}, func(cloud *testcloud.Cloud) error {
		_, err := troveInstanceAPI(cloud).Create(context.Background(), instances.CreateOpts{FlavorRef: "1", Size: 1})
		return err
	})
}

func TestTroveInstanceCreateRejectsInvalidInputsAndExtensionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := troveInstanceAPI(cloud)
	ctx := context.Background()
	var invalid gophercloud.ErrInvalidInput
	var missing gophercloud.ErrMissingInput
	if _, err := api.Create(ctx, instances.CreateOpts{FlavorRef: "1", Size: 0}); !errors.As(err, &invalid) {
		t.Fatalf("zero size: %v", err)
	}
	if _, err := api.Create(ctx, instances.CreateOpts{FlavorRef: "1", Size: -1}); !errors.As(err, &invalid) {
		t.Fatalf("negative size: %v", err)
	}
	if _, err := api.Create(ctx, instances.CreateOpts{Size: 1}); !errors.As(err, &missing) {
		t.Fatalf("missing flavor: %v", err)
	}
	if _, err := api.Create(ctx, instances.CreateOpts{FlavorRef: "1", Size: 1, Name: "db"}, instances.WithCreateField("name", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected name: %v", err)
	}
	if _, err := api.Create(ctx, instances.CreateOpts{FlavorRef: "1", Size: 1}, instances.WithCreateField("flavorRef", "2")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected flavorRef: %v", err)
	}
	if _, err := api.Create(ctx, instances.CreateOpts{FlavorRef: "1", Size: 1}, instances.WithCreateField("volume", map[string]int{"size": 9})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected volume: %v", err)
	}
	if _, err := api.Create(ctx, instances.CreateOpts{FlavorRef: "1", Size: 1}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
}

func TestTroveInstanceGetAndDeleteStatusesAndMissingPolicies(t *testing.T) {
	troveStatusMatrix(t, "GET", "/trove/instances/inst-1", troveInstanceBody, []int{200}, func(cloud *testcloud.Cloud) error {
		value, err := troveInstanceAPI(cloud).Get(context.Background(), "inst-1")
		if err == nil && (value == nil || value.Name != "db" || value.Hostname != "db.example.invalid") {
			t.Errorf("value=%+v", value)
		}
		return err
	})
	troveStatusMatrix(t, "DELETE", "/trove/instances/inst-1", `{}`, []int{202, 204}, func(cloud *testcloud.Cloud) error {
		return troveInstanceAPI(cloud).Delete(context.Background(), "inst-1")
	})
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/trove/instances/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := troveInstanceAPI(cloud)
	ctx := context.Background()
	if _, err := api.Get(ctx, "missing"); !troveIsNative404(err) || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if err := api.Delete(ctx, "missing"); !troveIsNative404(err) || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Remove with missing error: %v", err)
	}
}

func TestTroveInstanceActionsSendExactBodiesAndAccept201Or202(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
		call func(*instances.API) error
	}{
		{"restart", `{"restart":{}}`, func(api *instances.API) error { return api.Restart(context.Background(), "inst-1") }},
		{"resize", `{"resize":{"flavorRef":"9"}}`, func(api *instances.API) error { return api.Resize(context.Background(), "inst-1", "9") }},
		{"resize volume", `{"resize":{"volume":{"size":10}}}`, func(api *instances.API) error { return api.ResizeVolume(context.Background(), "inst-1", 10) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var got string
			cloud.Mux.HandleFunc("POST /trove/instances/inst-1/action", func(w http.ResponseWriter, r *http.Request) {
				data, _ := io.ReadAll(r.Body)
				got = string(data)
				w.WriteHeader(202)
			})
			if err := tc.call(troveInstanceAPI(cloud)); err != nil || got != tc.want {
				t.Fatalf("body=%q want=%q error=%v", got, tc.want, err)
			}
			troveStatusMatrix(t, "POST", "/trove/instances/inst-1/action", `{}`, []int{201, 202}, func(cloud *testcloud.Cloud) error {
				return tc.call(troveInstanceAPI(cloud))
			})
		})
	}
}

func TestTroveInstanceResizeVolumeSendsZeroAndNegativeSizesWithoutValidation(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []string
	cloud.Mux.HandleFunc("POST /trove/instances/inst-1/action", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(data))
		testcloud.JSON(w, 400, `{"badRequest":{"message":"invalid size"}}`)
	})
	api := troveInstanceAPI(cloud)
	for _, size := range []int{0, -1} {
		var unexpected gophercloud.ErrUnexpectedResponseCode
		if err := api.ResizeVolume(context.Background(), "inst-1", size); !errors.As(err, &unexpected) || unexpected.Actual != 400 {
			t.Fatalf("size %d error=%v", size, err)
		}
	}
	if want := []string{`{"resize":{"volume":{"size":0}}}`, `{"resize":{"volume":{"size":-1}}}`}; !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%v: server owns size validation", bodies)
	}
}

func TestTroveInstanceConfigurationGroupAttachAndDetach(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
		call func(*instances.API) error
	}{
		{"attach", `{"instance":{"configuration":"cfg-1"}}`, func(api *instances.API) error {
			return api.AttachConfigurationGroup(context.Background(), "inst-1", "cfg-1")
		}},
		{"detach", `{"instance":{}}`, func(api *instances.API) error {
			return api.DetachConfigurationGroup(context.Background(), "inst-1")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var got string
			cloud.Mux.HandleFunc("PUT /trove/instances/inst-1", func(w http.ResponseWriter, r *http.Request) {
				data, _ := io.ReadAll(r.Body)
				got = string(data)
				w.WriteHeader(202)
			})
			if err := tc.call(troveInstanceAPI(cloud)); err != nil || got != tc.want {
				t.Fatalf("body=%q want=%q error=%v", got, tc.want, err)
			}
			troveStatusMatrix(t, "PUT", "/trove/instances/inst-1", `{}`, []int{202}, func(cloud *testcloud.Cloud) error {
				return tc.call(troveInstanceAPI(cloud))
			})
		})
	}
}

func TestTroveInstanceEnableRootUserSendsNoBodyAndAccepts200(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /trove/instances/inst-1/root", func(w http.ResponseWriter, r *http.Request) {
		if data, _ := io.ReadAll(r.Body); len(data) != 0 {
			t.Errorf("root body=%q", data)
		}
		testcloud.JSON(w, 200, `{"user":{"name":"root","password":"example-secret"}}`)
	})
	user, err := troveInstanceAPI(cloud).EnableRootUser(context.Background(), "inst-1")
	if err != nil || user == nil || user.Name != "root" || user.Password != "example-secret" {
		t.Fatalf("user=%+v error=%v", user, err)
	}
	troveStatusMatrix(t, "POST", "/trove/instances/inst-1/root", `{"user":{"name":"root","password":"x"}}`, []int{200}, func(cloud *testcloud.Cloud) error {
		_, err := troveInstanceAPI(cloud).EnableRootUser(context.Background(), "inst-1")
		return err
	})
}

func TestTroveInstanceEnableRootUserDoesNotRetryAfterFailure(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /trove/instances/inst-1/root", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 500, `{}`)
	})
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := troveInstanceAPI(cloud).EnableRootUser(context.Background(), "inst-1"); !errors.As(err, &unexpected) || unexpected.Actual != 500 {
		t.Fatalf("error=%v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("uncertain mutation was replayed: calls=%d", calls.Load())
	}
}

func TestTroveInstanceListFollowsLinksAndAppliesCommonPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /trove/instances", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "inst-1" {
			testcloud.JSON(w, 200, `{"instances":[{"id":"inst-2","name":"two","status":"ACTIVE"},{"id":"inst-3","name":"dup","status":"ERROR"},{"id":"inst-4","name":"dup","status":"ACTIVE"}],"instances_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/trove/instances?marker=inst-1"
		testcloud.JSON(w, 200, `{"instances":[{"id":"inst-1","name":"one","status":"BUILD"}],"instances_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := troveInstanceAPI(cloud)
	ctx := context.Background()
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"inst-1", "inst-2", "inst-3", "inst-4"}) || len(requests) != 2 || len(requests[0]) != 0 || requests[1].Get("marker") != "inst-1" {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	requests = nil
	active, err := api.All(ctx, resource.WithStatus("ACTIVE"))
	if err != nil || len(active) != 2 || active[0].ID != "inst-2" || active[1].ID != "inst-4" {
		t.Fatalf("status filter: %v %v", active, err)
	}
	for _, query := range requests {
		if query.Has("status") {
			t.Fatalf("status must be applied locally, not sent: %v", requests)
		}
	}
	limited, err := api.All(ctx, resource.WithMaxItems(2))
	if err != nil || len(limited) != 2 {
		t.Fatalf("limited=%v error=%v", limited, err)
	}
	if _, err := api.Find(ctx, resource.Name("dup")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("duplicate name: %v", err)
	}
	if value, err := api.Find(ctx, resource.Name("two")); err != nil || value == nil || value.ID != "inst-2" {
		t.Fatalf("name find: %+v %v", value, err)
	}
	if _, err := api.Find(ctx, resource.Name("absent")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
	requests = nil
	if _, err := api.All(ctx, resource.WithQuery("vendor", "x")); !errors.Is(err, resource.ErrUnsupported) || len(requests) != 0 {
		t.Fatalf("extension query: %v requests=%v", err, requests)
	}
}

func TestTroveInstanceListNoContentAndHTTPFailure(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /trove/instances", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	if got, err := troveInstanceAPI(cloud).All(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("204 list: %v %v", got, err)
	}
	cloud = testcloud.New(t)
	cloud.Mux.HandleFunc("GET /trove/instances", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	for _, err := range troveInstanceAPI(cloud).List(context.Background()) {
		if !troveIsNative404(err) {
			t.Fatalf("list 404: %v", err)
		}
	}
}

func TestTroveInstanceCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := troveInstanceAPI(cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	checks := map[string]error{}
	_, checks["Create"] = api.Create(canceled, instances.CreateOpts{FlavorRef: "1", Size: 1})
	_, checks["Get"] = api.Get(canceled, "inst-1")
	_, checks["EnableRootUser"] = api.EnableRootUser(canceled, "inst-1")
	checks["Delete"] = api.Delete(canceled, "inst-1")
	checks["Restart"] = api.Restart(canceled, "inst-1")
	checks["Resize"] = api.Resize(canceled, "inst-1", "2")
	checks["ResizeVolume"] = api.ResizeVolume(canceled, "inst-1", 2)
	checks["AttachConfigurationGroup"] = api.AttachConfigurationGroup(canceled, "inst-1", "cfg-1")
	checks["DetachConfigurationGroup"] = api.DetachConfigurationGroup(canceled, "inst-1")
	for _, err := range api.List(canceled) {
		checks["List"] = err
	}
	for label, err := range checks {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled %s: %v", label, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := api.Restart(deadline, "inst-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("deadline restart must not be replayed: calls=%d", calls.Load())
	}
}
