package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deployables"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/devices"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestAcceleratorConnectionAndMicroversion(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if opts.Type != "accelerator" || opts.Region != "region" {
			t.Errorf("catalog options: %+v", opts)
		}
		return cloud.Server.URL + "/proxy/accelerator/v2/", nil
	}
	cloud.Mux.HandleFunc("/proxy/accelerator/v2/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/proxy/accelerator/v2/" {
			testcloud.JSON(w, 200, `{"id":"v2.0","min_version":"2.0","max_version":"2.12","version":"2.12"}`)
			return
		}
		if r.URL.Path != "/proxy/accelerator/v2/devices/d1" {
			t.Errorf("URL: %s", r.URL)
		}
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "accelerator 2.10" {
			t.Errorf("headers: %v", r.Header)
		}
		testcloud.JSON(w, 200, `{"uuid":"d1","id":17,"status":"enabled"}`)
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("region"), sdk.WithMicroversionRange(sdk.Accelerator, "2.3", "2.10"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := c.Accelerator(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.AcceleratorV2(context.Background())
	if err != nil || a != b || lookups.Load() != 1 {
		t.Fatalf("cache: %v/%d", err, lookups.Load())
	}
	v, err := a.Devices.Get(context.Background(), "d1")
	if err != nil || v.UUID != "d1" || v.ID.String() != "17" {
		t.Fatalf("device: %+v/%v", v, err)
	}
	for _, suffix := range []string{"", "/v2", "/v2/"} {
		c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Accelerator, cloud.Server.URL+"/proxy/accelerator"+suffix), sdk.WithMicroversion(sdk.Accelerator, "2.10"))
		if err != nil {
			t.Fatal(err)
		}
		a, err := c.Accelerator(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := a.Devices.Get(context.Background(), "d1"); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Accelerator(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestAcceleratorReadResourcesAndMetadata(t *testing.T) {
	cloud := testcloud.New(t)
	var pages atomic.Int32
	cloud.Mux.HandleFunc("/v2/deployables", func(w http.ResponseWriter, r *http.Request) {
		pages.Add(1)
		if r.URL.Query().Get("marker") == "next" {
			testcloud.JSON(w, 200, `{"deployables":[{"uuid":"p2","name":"same","root_id":3}]}`)
			return
		}
		w.Header().Set("X-Request-Id", "req-page")
		testcloud.JSON(w, 200, fmt.Sprintf(`{"deployables":[{"uuid":"p1","name":"same","parent_id":null,"device_id":17,"created_at":"2026-09-01 03:04:05+00:00","updated_at":null,"future":{"count":1234567890123456789}}],"links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2/deployables?marker=next"))
	})
	cloud.Mux.HandleFunc("/v2/deployables/p1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-get")
		testcloud.JSON(w, 200, `{"deployable":{"uuid":"p1","name":"same","parent_id":null,"bitstream_id":null,"future":false}}`)
	})
	a := deployables.New(cloud.Client("accelerator", "/v2"))
	for v, err := range a.List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		if v.UUID != "p1" || v.Header.Get("X-Request-Id") != "req-page" || string(v.Body["future"]) != `{"count":1234567890123456789}` || v.CreatedAt == nil {
			t.Fatalf("model: %+v", v)
		}
		break
	}
	if pages.Load() != 1 {
		t.Fatal("iterator fetched after break")
	}
	if _, err := a.Find(context.Background(), resource.Name("same")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	got, err := a.Get(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 200 || got.Header.Get("X-Request-Id") != "req-get" || string(got.Body["parent_id"]) != "null" || string(got.Body["future"]) != "false" {
		t.Fatalf("metadata: %+v", got)
	}
	if _, ok := got.Body["updated_at"]; ok {
		t.Fatal("omitted value invented")
	}
	if err := a.Delete(context.Background(), resource.ID("p1")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := a.Wait(context.Background(), resource.ID("p1"), "active"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
}

func TestAcceleratorReadFailuresAndIdentity(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/devices/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch strings.TrimPrefix(r.URL.Path, "/v2/devices/") {
		case "gone":
			testcloud.JSON(w, 404, `{"error":"missing"}`)
		case "denied":
			w.Header().Set("X-Request-Id", "req-error")
			testcloud.JSON(w, 403, `{"error":"denied"}`)
		case "bad":
			testcloud.JSON(w, 200, `null`)
		default:
			testcloud.JSON(w, 200, `{"uuid":"response-different","id":"17","status":"enabled"}`)
		}
	})
	client := cloud.Client("accelerator", "/v2")
	client.Microversion = "2.3"
	a := devices.New(client)
	if _, err := a.Find(context.Background(), resource.Name("host")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := a.Get(context.Background(), ".."); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("invalid identity sent HTTP")
	}
	if v, err := a.Find(context.Background(), resource.ID("gone"), resource.WithIgnoreMissing()); err != nil || v != nil {
		t.Fatalf("missing: %v/%v", v, err)
	}
	_, err := a.Get(context.Background(), "denied")
	var response gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &response) || response.Actual != 403 || response.ResponseHeader.Get("X-Request-Id") != "req-error" || !strings.Contains(string(response.Body), "denied") {
		t.Fatalf("lost response: %v", err)
	}
	if _, err := a.Get(context.Background(), "bad"); err == nil {
		t.Fatal("null accepted")
	}
	v, err := a.Wait(context.Background(), resource.ID("request-fixed"), "enabled", resource.WithTimeout(time.Second))
	if err != nil || v.UUID != "response-different" {
		t.Fatalf("wait: %+v/%v", v, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Get(ctx, "valid"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestAcceleratorPaginationCycleAndOrigin(t *testing.T) {
	for _, next := range []string{"?marker=again", "https://other.example/v2/deployables"} {
		t.Run(next, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/v2/deployables", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{"deployables":[{"uuid":"p1","name":"item"}],"next":%q}`, next))
			})
			_, err := deployables.New(cloud.Client("accelerator", "/v2")).All(context.Background())
			if err == nil {
				t.Fatal("invalid next accepted")
			}
			if next[0] == '?' {
				if !errors.Is(err, resource.ErrPaginationCycle) || calls.Load() != 2 {
					t.Fatalf("cycle: %v/%d", err, calls.Load())
				}
			} else if calls.Load() != 1 {
				t.Fatalf("origin: %d", calls.Load())
			}
		})
	}
}
