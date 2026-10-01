package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/instanceha/v1/segments"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const segmentUUID = "11111111-1111-4111-8111-111111111111"
const otherSegmentUUID = "22222222-2222-4222-8222-222222222222"

func TestInstanceHASegmentCRUDPreservesPrefixUUIDAndTypedBody(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/instance-ha/v1/catalog-tenant")
	client.Microversion = "1.2"
	var methods []string
	response := `{"segment":{"uuid":"` + segmentUUID + `","id":9007199254740993,"name":"compute-a","description":null,"updated_at":null,"enabled":false,"extension":9007199254740995}}`
	cloud.Mux.HandleFunc("/instance-ha/v1/catalog-tenant/segments", func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method != "POST" || r.Header.Get("OpenStack-API-Version") != "instance-ha 1.2" {
			t.Errorf("request %s %v", r.Method, r.Header)
		}
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body["segment"]["enabled"]) != "false" || string(body["segment"]["description"]) != `"snapshot"` || string(body["segment"]["custom"]) != `{"large":9007199254740997}` {
			t.Errorf("body=%v", body)
		}
		w.Header().Set("X-Request-ID", "create-evidence")
		testcloud.JSON(w, 202, response)
	})
	cloud.Mux.HandleFunc("/instance-ha/v1/catalog-tenant/segments/"+segmentUUID, func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		switch r.Method {
		case "GET":
			testcloud.JSON(w, 200, response)
		case "PUT":
			var body map[string]map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if string(body["segment"]["enabled"]) != "false" || string(body["segment"]["description"]) != `""` || len(body["segment"]) != 2 {
				t.Errorf("update=%v", body)
			}
			testcloud.JSON(w, 200, response)
		case "DELETE":
			w.WriteHeader(204)
		default:
			t.Errorf("method=%s", r.Method)
		}
	})
	description, disabled := "snapshot", false
	option := segments.WithCreateOptions(segments.CreateOpts{Name: "compute-a", RecoveryMethod: "auto", ServiceType: "COMPUTE", Description: &description, Enabled: &disabled})
	description, disabled = "mutated", true
	api := segments.New(client)
	value, err := api.Create(context.Background(), segments.CreateOpts{}, option, segments.WithCreateField("custom", map[string]json.Number{"large": "9007199254740997"}))
	if err != nil {
		t.Fatal(err)
	}
	if value.UUID != segmentUUID || string(value.ID) != "9007199254740993" || value.StatusCode != 202 || value.Header.Get("X-Request-ID") != "create-evidence" || string(value.Body["extension"]) != "9007199254740995" || value.UpdatedAt != nil || string(value.Body["description"]) != "null" {
		t.Fatalf("value=%+v", value)
	}
	if _, err := api.Get(context.Background(), value.UUID); err != nil {
		t.Fatal(err)
	}
	empty, disabled := "", false
	if _, err := api.Update(context.Background(), resource.ID(value.UUID), segments.UpdateOpts{Description: &empty, Enabled: &disabled}); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), resource.ID(value.UUID)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(methods, []string{"POST", "GET", "PUT", "DELETE"}) {
		t.Fatal(methods)
	}
}

func TestInstanceHASegmentListLinksAreLazyAndKeepFilters(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/v1/tenant")
	client.Microversion = "1.2"
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v1/tenant/segments", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("limit") != "1" || r.URL.Query().Get("enabled") != "false" || r.URL.Query().Get("service_type") != "COMPUTE" {
			t.Errorf("query=%s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("marker") == "next-page" {
			testcloud.JSON(w, 200, `{"segments":[{"uuid":"`+otherSegmentUUID+`","id":2,"name":"second"}]}`)
		} else {
			testcloud.JSON(w, 200, `{"segments":[{"uuid":"`+segmentUUID+`","id":1,"name":"first"}],"segments_links":[{"rel":"next","href":"?marker=next-page"}]}`)
		}
	})
	limit, enabled, service := 1, false, "COMPUTE"
	iterator := segments.New(client).List(context.Background(), segments.WithListOptions(segments.ListOpts{Limit: &limit, Enabled: &enabled, ServiceType: &service}))
	if calls.Load() != 0 {
		t.Fatal("iterator made eager HTTP")
	}
	for _, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if calls.Load() != 1 {
		t.Fatal("break fetched next page")
	}
	var ids []string
	for value, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID)
	}
	if !reflect.DeepEqual(ids, []string{segmentUUID, otherSegmentUUID}) || calls.Load() != 3 {
		t.Fatalf("ids=%v calls=%d", ids, calls.Load())
	}
}

func TestInstanceHASegmentNameUpdateUsesExactUUIDAndRejectsAmbiguity(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, updates atomic.Int32
	cloud.Mux.HandleFunc("/v1/segments", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if r.URL.Query().Has("name") {
			t.Fatal("unsupported name query sent")
		}
		testcloud.JSON(w, 200, `{"segments":[{"uuid":"`+otherSegmentUUID+`","id":2,"name":"computeXa"},{"uuid":"`+segmentUUID+`","id":1,"name":"compute.a"}]}`)
	})
	cloud.Mux.HandleFunc("/v1/segments/"+segmentUUID, func(w http.ResponseWriter, r *http.Request) {
		updates.Add(1)
		testcloud.JSON(w, 200, `{"segment":{"uuid":"`+segmentUUID+`","name":"renamed"}}`)
	})
	name := "renamed"
	if _, err := segments.New(cloud.Client("instance-ha", "/v1")).Update(context.Background(), resource.Name("compute.a"), segments.UpdateOpts{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if lookups.Load() != 1 || updates.Load() != 1 {
		t.Fatalf("lookups=%d updates=%d", lookups.Load(), updates.Load())
	}
}

func TestInstanceHASegmentTypedAllAndAmbiguousName(t *testing.T) {
	cloud := testcloud.New(t)
	requests := 0
	cloud.Mux.HandleFunc("/v1/segments", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "GET" {
			t.Errorf("ambiguous name caused mutation: %s", r.Method)
		}
		if requests == 1 && r.URL.Query().Get("service_type") != "COMPUTE" {
			t.Errorf("typed All lost query: %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"segments":[{"uuid":"`+segmentUUID+`","name":"duplicate"},{"uuid":"`+otherSegmentUUID+`","name":"duplicate"}]}`)
	})
	api := segments.New(cloud.Client("instance-ha", "/v1"))
	serviceType := "COMPUTE"
	values, err := api.All(context.Background(), segments.WithListOptions(segments.ListOpts{ServiceType: &serviceType}))
	if err != nil || len(values) != 2 {
		t.Fatalf("typed All: %#v %v", values, err)
	}
	if _, err := api.Resources.Find(context.Background(), resource.Name("duplicate")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("duplicate name: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests=%d", requests)
	}
}

func TestInstanceHASegmentVersionPreflightAppliesToEveryEntryPoint(t *testing.T) {
	for _, tc := range []struct {
		name, version, kind string
		headers             map[string]string
		want                error
	}{
		{"enabled below minimum", "1.1", "instance-ha", nil, resource.ErrUnsupported},
		{"latest", "latest", "instance-ha", nil, resource.ErrUnsupported},
		{"foreign major", "2.0", "instance-ha", nil, resource.ErrInvalidOption},
		{"blank type", "1.2", "", nil, resource.ErrInvalidOption},
		{"foreign type", "1.2", "compute", nil, resource.ErrInvalidOption},
		{"header downgrade", "1.2", "instance-ha", map[string]string{"OpenStack-API-Version": "instance-ha 1.0"}, resource.ErrInvalidOption},
		{"duplicate casing", "1.2", "instance-ha", map[string]string{"OpenStack-API-Version": "instance-ha 1.2", "openstack-api-version": "instance-ha 1.0"}, resource.ErrInvalidOption},
		{"implicit upgrade", "", "instance-ha", map[string]string{"OpenStack-API-Version": "instance-ha 1.2"}, resource.ErrInvalidOption},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("preflight made HTTP %s", r.URL) })
			client := cloud.Client(tc.kind, "/v1")
			client.Microversion = tc.version
			client.MoreHeaders = tc.headers
			api := segments.New(client)
			disabled := false
			_, err := api.Create(context.Background(), segments.CreateOpts{Name: "a", RecoveryMethod: "auto", ServiceType: "COMPUTE", Enabled: &disabled})
			if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
			_, err = api.Update(context.Background(), resource.Name("name"), segments.UpdateOpts{Enabled: &disabled})
			if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
			_, err = api.Resources.All(context.Background(), resource.WithQuery("enabled", "false"))
			if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
		})
	}
}

func TestInstanceHASegmentProtectedExtensionsFailBeforeNameLookup(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid option made HTTP %s", r.URL) })
	client := cloud.Client("instance-ha", "/v1")
	client.Microversion = "1.2"
	api := segments.New(client)
	name := "a"
	for _, key := range []string{"name", "enabled", "uuid", "id", "created_at"} {
		if _, err := api.Update(context.Background(), resource.Name("lookup"), segments.UpdateOpts{Name: &name}, segments.WithUpdateField(key, "override")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", key, err)
		}
	}
	if _, err := api.Create(context.Background(), segments.CreateOpts{Name: "a", RecoveryMethod: "auto", ServiceType: "COMPUTE"}, segments.WithCreateHeader("OpenStack-API-Version", "instance-ha 1.0")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, err := range api.List(context.Background(), segments.WithListQuery("enabled", "false")) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
}

func TestInstanceHASegmentAcceptedDecodeErrorRetainsMutationEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v1/segments", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Request-ID", "accepted")
		testcloud.JSON(w, 202, `{"segment":null}`)
	})
	_, err := segments.New(cloud.Client("instance-ha", "/v1")).Create(context.Background(), segments.CreateOpts{Name: "a", RecoveryMethod: "auto", ServiceType: "COMPUTE"})
	var evidence *resource.ResponseError
	if !errors.As(err, &evidence) || evidence.StatusCode != 202 || evidence.Header.Get("X-Request-ID") != "accepted" || string(evidence.Body) != `{"segment":null}` || calls.Load() != 1 {
		t.Fatalf("evidence=%+v err=%v calls=%d", evidence, err, calls.Load())
	}
}

func TestInstanceHASegmentMissingPolicyAndOriginalHTTPCause(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/segments/"+segmentUUID, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{"error":"missing"}`) })
	cloud.Mux.HandleFunc("/v1/segments/"+otherSegmentUUID, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 409, `{"error":"in use"}`) })
	api := segments.New(cloud.Client("instance-ha", "/v1"))
	if err := api.Delete(context.Background(), resource.ID(segmentUUID)); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), resource.ID(segmentUUID), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), resource.ID(otherSegmentUUID)); !gophercloud.ResponseCodeIs(err, 409) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(ctx, segmentUUID); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := api.Get(context.Background(), "9007199254740993"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(fmt.Sprintf("numeric database ID accepted: %v", err))
	}
}
