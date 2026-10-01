package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/accelerator/v2/acceleratorrequests"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestAcceleratorRequestCreateRetainsBatch(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || r.Method != "POST" || string(body["device_profile_name"]) != `"fpga"` || string(body["future"]) != `{"count":9007199254740993}` || r.Header.Get("X-Custom") != "create" {
			t.Errorf("request %s/%+v/%v", r.Method, body, err)
		}
		w.Header().Set("X-Request-Id", "req-batch")
		testcloud.JSON(w, 201, `{"arqs":[{"uuid":"a1","state":"Initial","device_profile_group_id":0,"hostname":null,"future":9007199254740993},{"uuid":"a2","state":"Initial","device_profile_group_id":1,"hostname":""}],"future_envelope":null}`)
	})
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	v, err := a.Create(context.Background(), acceleratorrequests.CreateOpts{DeviceProfileName: "fpga"}, acceleratorrequests.WithCreateField("future", map[string]any{"count": json.Number("9007199254740993")}), acceleratorrequests.WithCreateHeader("X-Custom", "create"))
	if err != nil || v.StatusCode != 201 || len(v.Requests) != 2 || v.Requests[0].UUID != "a1" || v.Requests[1].UUID != "a2" || v.Requests[0].Hostname != nil || v.Requests[1].Hostname == nil || *v.Requests[1].Hostname != "" || string(v.Requests[0].Body["future"]) != "9007199254740993" || string(v.Body["future_envelope"]) != "null" {
		t.Fatalf("created: %+v/%v", v, err)
	}
	v.Header.Set("X-Request-Id", "changed")
	if v.Requests[0].Header.Get("X-Request-Id") != "req-batch" {
		t.Fatal("envelope and resource headers alias")
	}
	v.Requests[0].Header.Set("X-Request-Id", "changed")
	if v.Requests[1].Header.Get("X-Request-Id") != "req-batch" {
		t.Fatal("batch resource headers alias")
	}
}

func TestAcceleratorRequestCreatePartialAndErrors(t *testing.T) {
	for _, tc := range []struct {
		body  string
		count int
		valid bool
	}{
		{`{"arqs":[]}`, 0, true},
		{`{"arqs":[{"uuid":"a1"},null]}`, 1, false},
		{`{"arqs":[{"uuid":"a1"},{"state":"Initial"}]}`, 1, false},
		{`{"arqs":null}`, 0, false},
		{`{"uuid":"a1"}`, 0, false},
		{`null`, 0, false},
		{`{"arqs":`, 0, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-Id", "accepted")
				testcloud.JSON(w, 201, tc.body)
			})
			v, err := acceleratorrequests.New(cloud.Client("accelerator", "/v2")).Create(context.Background(), acceleratorrequests.CreateOpts{DeviceProfileName: "fpga"})
			if (err == nil) != tc.valid || v == nil || v.Requests == nil || len(v.Requests) != tc.count || v.Header.Get("X-Request-Id") != "accepted" || v.StatusCode != 201 || calls.Load() != 1 {
				t.Fatalf("partial result=%+v err=%v calls=%d", v, err, calls.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "denied")
		testcloud.JSON(w, 403, `{"error":"denied"}`)
	})
	v, err := acceleratorrequests.New(cloud.Client("accelerator", "/v2")).Create(context.Background(), acceleratorrequests.CreateOpts{DeviceProfileName: "fpga"})
	var cause gophercloud.ErrUnexpectedResponseCode
	if v != nil || !errors.As(err, &cause) || cause.Actual != 403 || cause.ResponseHeader.Get("X-Request-Id") != "denied" {
		t.Fatalf("cause: %+v/%v", v, err)
	}
}

func TestAcceleratorRequestReadAndStatePolicy(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/accelerator_requests/a1", func(w http.ResponseWriter, r *http.Request) {
		state := "Initial"
		if calls.Add(1) > 1 {
			state = "Bound"
		}
		testcloud.JSON(w, 200, `{"uuid":"a1","state":"`+state+`","instance_uuid":null,"attach_handle_info":{"bus":"0000:01"}}`)
	})
	cloud.Mux.HandleFunc("/v2/accelerator_requests/a2", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"uuid":"a2","state":"BindFailed"}`)
	})
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	v, err := a.Wait(context.Background(), resource.ID("a1"), "bound", resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
	if err != nil || v.State != "Bound" || v.InstanceUUID != nil || calls.Load() != 2 || v.AttachHandleInfo["bus"] != "0000:01" {
		t.Fatalf("wait: %+v/%v", v, err)
	}
	if _, err := a.Wait(context.Background(), resource.ID("a2"), "Bound"); !errors.Is(err, resource.ErrFailedState) {
		t.Fatal(err)
	}
	if _, err := a.Find(context.Background(), resource.Name("fpga")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := a.Get(context.Background(), "a1,a2"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestAcceleratorRequestPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Errorf("unexpected request: %s", r.URL)
		w.WriteHeader(500)
	})
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	ctx := context.Background()
	for _, opt := range []acceleratorrequests.CreateOption{nil, acceleratorrequests.WithCreateField("device_profile_name", "other"), acceleratorrequests.WithCreateHeader("x-auth-token", "token"), request.WithQuery[acceleratorrequests.CreateOpts]("ignored", "value"), request.WithArgument[acceleratorrequests.CreateOpts]("unknown", true)} {
		if _, err := a.Create(ctx, acceleratorrequests.CreateOpts{DeviceProfileName: "fpga"}, opt); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := a.Create(ctx, acceleratorrequests.CreateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := a.Create(cancelled, acceleratorrequests.CreateOpts{DeviceProfileName: "fpga"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight sent HTTP")
	}
}
