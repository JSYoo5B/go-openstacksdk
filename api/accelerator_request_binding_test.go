package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestAcceleratorRequestBindingCollectionURL(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		var body map[string][]acceleratorrequests.BindingOperation
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || r.Method != "PATCH" || r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "accelerator 2.10" || r.Header.Get("X-Service-Token") != "service-token" {
			t.Errorf("request %s %s %+v / %v", r.Method, r.URL, body, err)
		}
		patch := body["a1"]
		if calls.Add(1) == 1 {
			if len(patch) != 4 || patch[0].Op != "add" || patch[0].Path != "/hostname" || patch[0].Value == nil || *patch[0].Value != "host1" || *patch[3].Value != "project1" {
				t.Errorf("bind patch %+v", patch)
			}
		} else {
			for _, op := range patch {
				if op.Op != "remove" || op.Value != nil {
					t.Errorf("unbind patch %+v", patch)
				}
			}
		}
		w.Header().Set("X-Request-Id", "binding-accepted")
		w.WriteHeader(202)
	})
	client := cloud.Client("accelerator", "/v2")
	client.Microversion = "2.10"
	a := acceleratorrequests.New(client)
	project := "project1"
	meta, err := a.Bind(context.Background(), resource.ID("a1"), acceleratorrequests.BindOpts{Hostname: "host1", DeviceRPUUID: "rp1", InstanceUUID: "instance1", ProjectID: &project}, acceleratorrequests.WithBindingHeader("X-Service-Token", "service-token"))
	if err != nil || meta.StatusCode != 202 || meta.Header.Get("X-Request-Id") != "binding-accepted" || meta.Body != nil {
		t.Fatalf("bind %+v/%v", meta, err)
	}
	if _, err := a.Unbind(context.Background(), resource.ID("a1"), acceleratorrequests.WithBindingHeader("X-Service-Token", "service-token")); err != nil || calls.Load() != 2 {
		t.Fatalf("unbind %v/%d", err, calls.Load())
	}
}

func TestAcceleratorRequestBindingSnapshotAndBatch(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("accelerator", "/v2")
	var calls atomic.Int32
	value := "before"
	rp := "rp"
	instance := "instance"
	patches := map[string][]acceleratorrequests.BindingOperation{
		"a1": {{Op: "add", Path: "/hostname", Value: &value}, {Op: "add", Path: "/device_rp_uuid", Value: &rp}, {Op: "add", Path: "/instance_uuid", Value: &instance}},
		"a2": {{Op: "add", Path: "/hostname", Value: &value}, {Op: "add", Path: "/device_rp_uuid", Value: &rp}, {Op: "add", Path: "/instance_uuid", Value: &instance}},
	}
	client.ReauthFunc = func(ctx context.Context) error {
		value = "after"
		delete(patches, "a2")
		client.SetToken("fresh")
		return nil
	}
	cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		var body map[string][]acceleratorrequests.BindingOperation
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 2 || *body["a1"][0].Value != "before" || *body["a2"][0].Value != "before" {
			t.Errorf("snapshot %+v/%v", body, err)
		}
		if calls.Add(1) == 1 {
			testcloud.JSON(w, 401, `{"error":"expired"}`)
			return
		}
		w.WriteHeader(202)
	})
	meta, err := acceleratorrequests.New(client).PatchMany(context.Background(), patches)
	if err != nil || meta.StatusCode != 202 || calls.Load() != 2 || value != "after" {
		t.Fatalf("batch %+v/%v/%d", meta, err, calls.Load())
	}
}

func TestAcceleratorRequestBindingPreflightAndAuthorization(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Request-Id", "service-required")
		testcloud.JSON(w, 403, `{"error":"service token required"}`)
	})
	client := cloud.Client("accelerator", "/v2")
	a := acceleratorrequests.New(client)
	ctx := context.Background()
	value := "value"
	for _, patch := range [][]acceleratorrequests.BindingOperation{
		nil, {{Op: "replace", Path: "/hostname", Value: &value}}, {{Op: "add", Path: "/hostname", Value: &value}},
		{{Op: "remove", Path: "/hostname", Value: &value}}, {{Op: "remove", Path: "/unknown"}},
		{{Op: "remove", Path: "/hostname"}, {Op: "add", Path: "/instance_uuid", Value: &value}},
		{{Op: "remove", Path: "/hostname"}, {Op: "remove", Path: "/hostname"}},
	} {
		if _, err := a.Patch(ctx, resource.ID("a1"), patch); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := a.PatchMany(ctx, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := a.Patch(ctx, resource.Name("name"), []acceleratorrequests.BindingOperation{{Op: "remove", Path: "/hostname"}}); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	project := "project"
	if _, err := a.Bind(ctx, resource.ID("a1"), acceleratorrequests.BindOpts{Hostname: "host", DeviceRPUUID: "rp", InstanceUUID: "instance", ProjectID: &project}); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	for _, opt := range []acceleratorrequests.BindingOption{nil, acceleratorrequests.WithBindingHeader("OpenStack-API-Version", "accelerator 2.10"), request.WithField[struct{}]("unknown", true), request.WithQuery[struct{}]("unknown", "value"), request.WithArgument[struct{}]("unknown", true)} {
		if _, err := a.Unbind(ctx, resource.ID("a1"), opt); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid binding sent HTTP")
	}
	_, err := a.Unbind(ctx, resource.ID("a1"))
	var cause gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &cause) || cause.Actual != 403 || cause.ResponseHeader.Get("X-Request-Id") != "service-required" || calls.Load() != 1 {
		t.Fatalf("auth cause %v", err)
	}
}

func TestAcceleratorRequestDeleteSelectorsAndPolicy(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "DELETE" || r.ContentLength > 0 || len(r.URL.Query()) != 1 {
			t.Errorf("delete %s %s", r.Method, r.URL)
		}
		switch {
		case r.URL.Query().Get("arqs") == "a1":
			w.WriteHeader(204)
		case r.URL.Query().Get("arqs") == "a1,a2":
			if r.Header.Get("X-Service-Token") != "service-token" {
				t.Error("missing service token")
			}
			testcloud.JSON(w, 404, `{"error":"one missing"}`)
		case r.URL.Query().Get("arqs") == "missing":
			testcloud.JSON(w, 404, `{"error":"missing"}`)
		case r.URL.Query().Get("instance") == "instance":
			if r.Header.Get("X-Service-Token") != "service-token" {
				t.Error("missing service token")
			}
			w.WriteHeader(204)
		default:
			t.Errorf("unknown selector %s", r.URL)
			w.WriteHeader(500)
		}
	})
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	ctx := context.Background()
	if err := a.Delete(ctx, resource.ID("a1")); err != nil {
		t.Fatal(err)
	}
	if err := a.Delete(ctx, resource.ID("missing")); err != nil {
		t.Fatal(err)
	}
	if err := a.Delete(ctx, resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	if _, err := a.DeleteMany(ctx, []string{"a1", "a2"}, acceleratorrequests.WithDeleteHeader("X-Service-Token", "service-token")); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if meta, err := a.DeleteByInstance(ctx, "instance", acceleratorrequests.WithDeleteHeader("X-Service-Token", "service-token")); err != nil || meta.StatusCode != 204 {
			t.Fatalf("instance: %+v/%v", meta, err)
		}
	}
	before := calls.Load()
	for _, ids := range [][]string{nil, {"a1,a2"}, {"a1", "a1"}, {""}} {
		if _, err := a.DeleteMany(ctx, ids); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if err := a.Delete(ctx, resource.ID("a1,a2")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != before {
		t.Fatal("invalid delete sent HTTP")
	}
}
