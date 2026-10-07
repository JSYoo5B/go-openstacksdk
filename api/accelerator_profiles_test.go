package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/deviceprofiles"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const profileUUID1 = "11111111-1111-4111-8111-111111111111"
const profileUUID2 = "22222222-2222-4222-8222-222222222222"
const profileUUID3 = "33333333-3333-4333-8333-333333333333"

func TestAcceleratorProfileExactNamesAndMetadata(t *testing.T) {
	cloud := testcloud.New(t)
	var pages, gets, deletes atomic.Int32
	cloud.Mux.HandleFunc("/v2/device_profiles", func(w http.ResponseWriter, r *http.Request) {
		pages.Add(1)
		if r.URL.Query().Get("name") != "" {
			t.Error("name lookup must preserve exact local semantics")
		}
		if r.URL.Query().Get("marker") == "second" {
			testcloud.JSON(w, 200, fmt.Sprintf(`{"device_profiles":[{"uuid":%q,"name":%q,"groups":[]}]}`, profileUUID2, profileUUID1))
			return
		}
		w.Header().Set("X-Request-Id", "req-profile-list")
		testcloud.JSON(w, 200, fmt.Sprintf(`{"device_profiles":[{"uuid":%q,"name":"prefix-target","description":null,"groups":[{"resources:CUSTOM_FPGA":1234567890123456789}],"created_at":"2026-09-01 01:02:03+00:00","updated_at":null,"future":{"count":1234567890123456789}}],"next":"?marker=second"}`, profileUUID1))
	})
	cloud.Mux.HandleFunc("/v2/device_profiles/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/device_profiles/"+profileUUID2 {
			t.Errorf("name or response UUID used as endpoint: %s", r.URL.Path)
		}
		if r.Method == "DELETE" {
			deletes.Add(1)
			w.WriteHeader(204)
			return
		}
		gets.Add(1)
		w.Header().Set("X-Request-Id", "req-profile-get")
		testcloud.JSON(w, 200, fmt.Sprintf(`{"device_profile":{"uuid":%q,"name":%q,"description":"","groups":[{"trait:CUSTOM_ACCEL":"required"}],"links":[{"rel":"self","href":"/profile"}],"future":false}}`, profileUUID3, profileUUID1))
	})
	a := deviceprofiles.New(cloud.Client("accelerator", "/v2"))
	if a.Resources != a.Collection || a.RawClient().ProviderClient != cloud.Provider {
		t.Fatal("collection/client not shared")
	}
	for value, err := range a.List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		if value.Header.Get("X-Request-Id") != "req-profile-list" || value.StatusCode != 200 || value.CreatedAt == nil || value.Description != nil || string(value.Body["updated_at"]) != "null" || string(value.Body["future"]) != `{"count":1234567890123456789}` || value.Groups[0]["resources:CUSTOM_FPGA"].(json.Number).String() != "1234567890123456789" {
			t.Fatalf("list metadata: %+v", value)
		}
		break
	}
	if pages.Load() != 1 {
		t.Fatal("early iterator stop fetched another page")
	}
	id, err := a.ResolveID(context.Background(), resource.Name(profileUUID1))
	if err != nil || id != profileUUID2 {
		t.Fatalf("UUID-shaped name lookup: %q/%v", id, err)
	}
	if value, err := a.Find(context.Background(), resource.Name("target"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatalf("non-exact name matched: %+v/%v", value, err)
	}
	value, err := a.Find(context.Background(), resource.ID(profileUUID2))
	if err != nil || value.UUID != profileUUID3 || value.Header.Get("X-Request-Id") != "req-profile-get" || string(value.Body["future"]) != "false" || value.Description == nil || *value.Description != "" || len(value.Links) != 1 {
		t.Fatalf("get details: %+v/%v", value, err)
	}
	if err := a.Delete(context.Background(), resource.Name(profileUUID1)); err != nil {
		t.Fatal(err)
	}
	if gets.Load() != 1 || deletes.Load() != 1 {
		t.Fatalf("get/delete=%d/%d", gets.Load(), deletes.Load())
	}
	if _, err := a.Wait(context.Background(), resource.ID(profileUUID1), "ready"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
}

func TestAcceleratorProfileAmbiguityAndUnsafeIDs(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, deletes atomic.Int32
	cloud.Mux.HandleFunc("/v2/device_profiles", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("page") == "2" {
			testcloud.JSON(w, 200, fmt.Sprintf(`{"device_profiles":[{"uuid":%q,"name":"same"}]}`, profileUUID2))
			return
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"device_profiles":[{"uuid":%q,"name":"same"}],"next":"?page=2"}`, profileUUID1))
	})
	cloud.Mux.HandleFunc("/v2/device_profiles/", func(w http.ResponseWriter, r *http.Request) { deletes.Add(1); w.WriteHeader(204) })
	a := deviceprofiles.New(cloud.Client("accelerator", "/v2"))
	for _, id := range []string{"first,second", "named-profile", "11111111111141118111111111111111", ".."} {
		if _, err := a.Get(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("Get invalid ID %q: %v", id, err)
		}
		if err := a.Delete(context.Background(), resource.ID(id)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("Delete unsafe ID %q: %v", id, err)
		}
	}
	if calls.Load() != 0 || deletes.Load() != 0 {
		t.Fatal("invalid UUID made HTTP request")
	}
	if err := a.Delete(context.Background(), resource.Name("same")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	if calls.Load() != 2 || deletes.Load() != 0 {
		t.Fatalf("ambiguous deletion: %d/%d", calls.Load(), deletes.Load())
	}
	cloud.Mux.HandleFunc("/v2/bad/device_profiles", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"device_profiles":[{"uuid":"one,two","name":"malformed"}]}`)
	})
	bad := deviceprofiles.New(cloud.Client("accelerator", "/v2/bad"))
	if err := bad.Delete(context.Background(), resource.Name("malformed")); err == nil {
		t.Fatal("malformed response UUID permitted batch deletion")
	}
	if deletes.Load() != 0 {
		t.Fatal("malformed response UUID sent DELETE")
	}
}

func TestAcceleratorProfileCreateContractAndSnapshots(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/device_profiles", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.Header.Get("X-Profile-Extension") != "enabled" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("request: %s/%v", r.Method, r.Header)
		}
		var body []map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 {
			t.Fatalf("profile must be one-element array: %v/%v", body, err)
		}
		if string(body[0]["name"]) != `"original"` || string(body[0]["description"]) != `""` || string(body[0]["groups"]) != `[{"resources:CUSTOM_FPGA":1234567890123456789,"trait:CUSTOM_ACCEL":"required"}]` || string(body[0]["future"]) != `{"list":["original"]}` {
			t.Errorf("snapshot: %s", body)
		}
		if _, ok := body[0]["uuid"]; ok {
			t.Error("nil UUID serialized")
		}
		w.Header().Set("X-Request-Id", "req-create-profile")
		testcloud.JSON(w, 201, fmt.Sprintf(`{"uuid":%q,"name":"original","description":"","groups":[{"resources:CUSTOM_FPGA":1234567890123456789}],"future":null}`, profileUUID1))
	})
	description := ""
	groups := []deviceprofiles.Group{{"resources:CUSTOM_FPGA": json.Number("1234567890123456789"), "trait:CUSTOM_ACCEL": "required"}}
	typed := deviceprofiles.WithCreateOptions(deviceprofiles.CreateOpts{Name: "original", Groups: groups, Description: &description})
	extension := map[string]any{"list": []string{"original"}}
	field := deviceprofiles.WithCreateField("future", extension)
	groups[0]["trait:CUSTOM_ACCEL"] = "forbidden"
	description = "changed"
	extension["list"].([]string)[0] = "changed"
	a := deviceprofiles.New(cloud.Client("accelerator", "/v2"))
	for i := 0; i < 2; i++ {
		value, err := a.Create(context.Background(), deviceprofiles.CreateOpts{}, typed, field, deviceprofiles.WithCreateHeader("X-Profile-Extension", "enabled"))
		if err != nil || value.UUID != profileUUID1 || value.Header.Get("X-Request-Id") != "req-create-profile" || value.StatusCode != 201 || string(value.Body["future"]) != "null" || value.Groups[0]["resources:CUSTOM_FPGA"].(json.Number).String() != "1234567890123456789" {
			t.Fatalf("create: %+v/%v", value, err)
		}
	}
	if calls.Load() != 2 || groups[0]["trait:CUSTOM_ACCEL"] != "forbidden" || description != "changed" {
		t.Fatal("options were consumed or caller input mutated")
	}
}

func TestAcceleratorProfileCreateValidationBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	a := deviceprofiles.New(cloud.Client("accelerator", "/v2"))
	valid := deviceprofiles.CreateOpts{Name: "valid", Groups: []deviceprofiles.Group{{"resources:CUSTOM_FPGA": "1"}}}
	for _, key := range []string{"name", "groups", "uuid", "description"} {
		if _, err := a.Create(context.Background(), valid, deviceprofiles.WithCreateField(key, "override")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("core %s: %v", key, err)
		}
	}
	for _, header := range []string{"x-auth-token", "OpenStack-API-Version", "content-type", "Host"} {
		if _, err := a.Create(context.Background(), valid, deviceprofiles.WithCreateHeader(header, "override")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("core header %s: %v", header, err)
		}
	}
	badUUID := "one,two"
	for _, opts := range []deviceprofiles.CreateOpts{{Name: "", Groups: valid.Groups}, {Name: "bad/name", Groups: valid.Groups}, {Name: "valid"}, {Name: "valid", Groups: []deviceprofiles.Group{nil}}, {Name: "valid", Groups: valid.Groups, UUID: &badUUID}} {
		if _, err := a.Create(context.Background(), opts); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid create: %+v/%v", opts, err)
		}
	}
	if _, err := a.Create(context.Background(), valid, deviceprofiles.WithCreateField("future", make(chan int))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := a.Create(context.Background(), valid, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Create(ctx, valid); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("invalid input made HTTP request")
	}
}

func TestAcceleratorProfileRejectsMalformedResponseGroups(t *testing.T) {
	for name, groups := range map[string]string{"array item": `[[]]`, "null item": `[null]`, "scalar item": `[17]`, "object instead of array": `{}`, "string": `"not-groups"`} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/v2/device_profiles/"+profileUUID1, func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, fmt.Sprintf(`{"device_profile":{"uuid":%q,"name":"bad","groups":%s}}`, profileUUID1, groups))
			})
			if value, err := deviceprofiles.New(cloud.Client("accelerator", "/v2")).Get(context.Background(), profileUUID1); err == nil || value != nil {
				t.Fatalf("invalid groups accepted: %+v/%v", value, err)
			}
		})
	}
}

func TestAcceleratorProfileDeleteWaitAndHTTPFailures(t *testing.T) {
	cloud := testcloud.New(t)
	var polls atomic.Int32
	cloud.Mux.HandleFunc("/v2/device_profiles/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/v2/device_profiles/")
		if id == profileUUID3 {
			w.Header().Set("X-Request-Id", "req-profile-denied")
			testcloud.JSON(w, 403, `{"error":"forbidden"}`)
			return
		}
		if id == profileUUID2 || r.Method == "DELETE" || polls.Add(1) > 1 {
			testcloud.JSON(w, 404, `{"error":"gone"}`)
			return
		}
		// A differing response UUID must not change the next poll's identifier.
		testcloud.JSON(w, 200, fmt.Sprintf(`{"uuid":%q,"name":"pending"}`, profileUUID3))
	})
	a := deviceprofiles.New(cloud.Client("accelerator", "/v2"))
	if err := a.Delete(context.Background(), resource.ID(profileUUID2)); err != nil {
		t.Fatal(err)
	}
	if err := a.Delete(context.Background(), resource.ID(profileUUID2), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	if err := a.WaitDeleted(context.Background(), resource.ID(profileUUID1), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second)); err != nil || polls.Load() != 2 {
		t.Fatalf("stable waiter: %v/%d", err, polls.Load())
	}
	err := a.Delete(context.Background(), resource.ID(profileUUID3))
	var response gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &response) || response.Actual != 403 || response.ResponseHeader.Get("X-Request-Id") != "req-profile-denied" || string(response.Body) != `{"error":"forbidden"}` {
		t.Fatalf("HTTP cause lost: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.WaitDeleted(ctx, resource.ID(profileUUID1)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
