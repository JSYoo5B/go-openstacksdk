package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCyborgCloudBindAndUnbindSendUUIDKeyedPatchAndRequire202(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []string
	status := 202
	cloud.Mux.HandleFunc("PATCH /v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, string(cyborgCloudDecode(t, r)))
		if r.URL.RawQuery != "" {
			t.Errorf("query %q", r.URL.RawQuery)
		}
		w.WriteHeader(status)
	})
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	ctx := context.Background()
	meta, err := a.Bind(ctx, resource.ID("a1"), acceleratorrequests.BindOpts{Hostname: "host1", DeviceRPUUID: "rp1", InstanceUUID: "vm1"})
	if err != nil || meta == nil || meta.StatusCode != 202 || meta.Body != nil {
		t.Fatalf("bind meta=%+v err=%v", meta, err)
	}
	if _, err := a.Unbind(ctx, resource.ID("a1")); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`{"a1":[{"op":"add","path":"/hostname","value":"host1"},{"op":"add","path":"/device_rp_uuid","value":"rp1"},{"op":"add","path":"/instance_uuid","value":"vm1"}]}`,
		`{"a1":[{"op":"remove","path":"/hostname"},{"op":"remove","path":"/device_rp_uuid"},{"op":"remove","path":"/instance_uuid"}]}`,
	}
	if len(bodies) != 2 || bodies[0] != want[0] || bodies[1] != want[1] {
		t.Fatalf("bodies=%q", bodies)
	}
	for _, code := range []int{200, 201, 204, 500} {
		status = code
		before := len(bodies)
		meta, err := a.Bind(ctx, resource.ID("a1"), acceleratorrequests.BindOpts{Hostname: "host1", DeviceRPUUID: "rp1", InstanceUUID: "vm1"})
		if !cyborgCloudIsStatus(err, code) || len(bodies) != before+1 {
			t.Fatalf("status %d meta=%+v err=%v", code, meta, err)
		}
	}
}

func TestCyborgCloudPatchAcceptsOnlyAddOrRemoveOfBindingFieldsBeforeHTTP(t *testing.T) {
	cloud, calls := cyborgCloudNoHTTP(t)
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	ctx := context.Background()
	v, empty := "v", ""
	full := []acceleratorrequests.BindingOperation{{Op: "add", Path: "/hostname", Value: &v}, {Op: "add", Path: "/device_rp_uuid", Value: &v}, {Op: "add", Path: "/instance_uuid", Value: &v}}
	for name, patch := range map[string][]acceleratorrequests.BindingOperation{
		"empty":           nil,
		"replace":         {{Op: "replace", Path: "/hostname", Value: &v}},
		"test":            {{Op: "test", Path: "/hostname", Value: &v}},
		"unknown path":    {{Op: "remove", Path: "/state"}},
		"partial add":     {{Op: "add", Path: "/hostname", Value: &v}},
		"empty value":     {{Op: "add", Path: "/hostname", Value: &empty}, full[1], full[2]},
		"nil value":       {{Op: "add", Path: "/hostname"}, full[1], full[2]},
		"remove w/ value": {{Op: "remove", Path: "/hostname", Value: &v}},
		"mixed":           {{Op: "remove", Path: "/hostname"}, full[2]},
		"duplicate":       {{Op: "remove", Path: "/hostname"}, {Op: "remove", Path: "/hostname"}},
	} {
		if _, err := a.Patch(ctx, resource.ID("a1"), patch); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := a.Patch(ctx, resource.ID("a1"), full, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Errorf("nil option: %v", err)
	}
	if _, err := a.Patch(ctx, resource.ID("a1"), full, acceleratorrequests.WithBindingHeader("OpenStack-API-Version", "accelerator 2.9")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Errorf("reserved header: %v", err)
	}
	if _, err := a.Patch(ctx, resource.ID("a1,a2"), full); !errors.Is(err, resource.ErrInvalidOption) {
		t.Errorf("multi-ID ref: %v", err)
	}
	if _, err := a.Patch(ctx, resource.Name("name"), full); !errors.Is(err, resource.ErrUnsupported) {
		t.Errorf("name ref: %v", err)
	}
	if _, err := a.Patch(cyborgCloudCanceled(), resource.ID("a1"), full); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight sent HTTP")
	}
}

func TestCyborgCloudBindProjectIDNeedsMicroversion21AndPreservesValue(t *testing.T) {
	cloud := testcloud.New(t)
	var body string
	cloud.Mux.HandleFunc("PATCH /v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		body = string(cyborgCloudDecode(t, r))
		if r.Header.Get("OpenStack-API-Version") != "accelerator 2.1" {
			t.Errorf("microversion header %q", r.Header.Get("OpenStack-API-Version"))
		}
		w.WriteHeader(202)
	})
	project := "project-1"
	opts := acceleratorrequests.BindOpts{Hostname: "h", DeviceRPUUID: "rp", InstanceUUID: "vm", ProjectID: &project}
	base := cloud.Client("accelerator", "/v2")
	if _, err := acceleratorrequests.New(base).Bind(context.Background(), resource.ID("a1"), opts); !errors.Is(err, resource.ErrUnsupported) || body != "" {
		t.Fatalf("default microversion: %v body=%q", err, body)
	}
	newer := cloud.Client("accelerator", "/v2")
	newer.Microversion = "2.1"
	if _, err := acceleratorrequests.New(newer).Bind(context.Background(), resource.ID("a1"), opts); err != nil {
		t.Fatal(err)
	}
	if want := `{"a1":[{"op":"add","path":"/hostname","value":"h"},{"op":"add","path":"/device_rp_uuid","value":"rp"},{"op":"add","path":"/instance_uuid","value":"vm"},{"op":"add","path":"/project_id","value":"project-1"}]}`; body != want {
		t.Fatalf("body=%s", body)
	}
}

func TestCyborgCloudBindDeadlineKeepsCauseWithoutReplay(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cyborgCloudHang(cloud, "PATCH /v2/accelerator_requests", &calls)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := acceleratorrequests.New(cloud.Client("accelerator", "/v2")).Bind(ctx, resource.ID("a1"), acceleratorrequests.BindOpts{Hostname: "h", DeviceRPUUID: "rp", InstanceUUID: "vm"})
	if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}
