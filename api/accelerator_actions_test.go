package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/deployables"
	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/devices"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestAcceleratorDeviceActions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/devices/d1/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.Header.Get("OpenStack-API-Version") != "accelerator 2.10" || r.Header.Get("X-Custom") != "value" || r.ContentLength > 0 {
			t.Errorf("request: %s %s %v", r.Method, r.URL, r.Header)
		}
		w.Header().Set("X-Request-Id", "req-action")
		if r.URL.Path == "/v2/devices/d1/disable" {
			testcloud.JSON(w, 403, `{"error":"denied"}`)
			return
		}
		w.WriteHeader(200)
	})
	client := cloud.Client("accelerator", "/v2")
	client.Microversion = "2.10"
	a := devices.New(client)
	meta, err := a.Enable(context.Background(), resource.ID("d1"), devices.WithActionHeader("X-Custom", "value"))
	if err != nil || meta.StatusCode != 200 || meta.Header.Get("X-Request-Id") != "req-action" {
		t.Fatalf("enable: %+v/%v", meta, err)
	}
	_, err = a.Disable(context.Background(), resource.ID("d1"), devices.WithActionHeader("X-Custom", "value"))
	var response gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &response) || response.Actual != 403 || response.ResponseHeader.Get("X-Request-Id") != "req-action" {
		t.Fatalf("disable: %v", err)
	}
	for _, version := range []string{"latest", "2.bad", "2.99999999999999999999999", "3.10"} {
		client.Microversion = version
		if _, err := a.Enable(context.Background(), resource.ID("d1")); err == nil {
			t.Fatalf("version accepted: %s", version)
		}
	}
	client.Microversion = "2.10"
	for _, option := range []devices.ActionOption{nil, devices.WithActionHeader("openstack-api-version", "accelerator 2.0"), request.WithField[struct{}]("field", true), request.WithQuery[struct{}]("query", "value"), request.WithArgument[struct{}]("argument", true)} {
		if _, err := a.Enable(context.Background(), resource.ID("d1"), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid option: %v", err)
		}
	}
	if _, err := a.Enable(context.Background(), resource.Name("compute-host")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("invalid action sent HTTP: %d", calls.Load())
	}
}

func TestAcceleratorProgrammingSnapshotAndURL(t *testing.T) {
	cloud := testcloud.New(t)
	patch := []deployables.ProgramOperation{{Op: "replace", Path: "/program", Value: []deployables.ProgramImage{{ImageUUID: "image-before"}}}}
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/deployables", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		patch[0].Value[0].ImageUUID = "image-after"
		testcloud.JSON(w, 200, `{"deployables":[{"uuid":"p1","name":"fpga"}]}`)
	})
	cloud.Mux.HandleFunc("/v2/deployables/p1/program", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "PATCH" || r.Header.Get("X-Custom") != "value" {
			t.Errorf("request: %s/%v", r.Method, r.Header)
		}
		var got []deployables.ProgramOperation
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil || len(got) != 1 || got[0].Value[0].ImageUUID != "image-before" || got[0].Path != "/program" || got[0].Op != "replace" {
			t.Errorf("patch: %+v/%v", got, err)
		}
		w.Header().Set("X-Request-Id", "req-program")
		testcloud.JSON(w, 200, `{"uuid":"p1","name":"fpga","bitstream_id":"image-before","future":"retained"}`)
	})
	a := deployables.New(cloud.Client("accelerator", "/v2"))
	got, err := a.Patch(context.Background(), resource.Name("fpga"), patch, deployables.WithProgramHeader("X-Custom", "value"))
	if err != nil || got.UUID != "p1" || got.Header.Get("X-Request-Id") != "req-program" || string(got.Body["future"]) != `"retained"` {
		t.Fatalf("program: %+v/%v", got, err)
	}
	for _, input := range [][]deployables.ProgramOperation{nil, {{Op: "add", Path: "/program", Value: []deployables.ProgramImage{{ImageUUID: "image"}}}}, {{Op: "replace", Path: "/other", Value: []deployables.ProgramImage{{ImageUUID: "image"}}}}, {{Op: "replace", Path: "/program", Value: []deployables.ProgramImage{{ImageUUID: ""}}}}} {
		if _, err := a.Patch(context.Background(), resource.Name("fpga"), input); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid patch: %v", err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("invalid patch sent HTTP: %d", calls.Load())
	}
}

func TestAcceleratorProgrammingHTTPError(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v2/deployables/p1/program", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 500, `{"error":"program failed"}`) })
	_, err := deployables.New(cloud.Client("accelerator", "/v2")).Program(context.Background(), resource.ID("p1"), "image")
	if !gophercloud.ResponseCodeIs(err, 500) {
		t.Fatalf("program error lost: %v", err)
	}
}
