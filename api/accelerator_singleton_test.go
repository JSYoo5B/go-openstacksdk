package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/common"
	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/devices"
	"github.com/JSYoo5B/gophercloudsdk/internal/cyborg"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestAcceleratorSingletonEnvelopeCardinality(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"flat", `{"uuid":"d1"}`, true},
		{"singular", `{"device":{"uuid":"d1"}}`, true},
		{"plural-one", `{"devices":[{"uuid":"d1"}]}`, true},
		{"plural-empty", `{"devices":[]}`, false},
		{"plural-many", `{"devices":[{"uuid":"d1"},{"uuid":"d2"}]}`, false},
		{"plural-null", `{"devices":null}`, false},
		{"singular-null", `{"device":null}`, false},
		{"both", `{"device":{"uuid":"d1"},"devices":[{"uuid":"d2"}]}`, false},
		{"array", `[{"uuid":"d1"}]`, false},
		{"null", `null`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/v2/devices/d1", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-Id", "cardinality")
				testcloud.JSON(w, 200, tc.body)
			})
			v, err := devices.New(cloud.Client("accelerator", "/v2")).Get(context.Background(), "d1")
			if tc.valid {
				if err != nil || v.UUID != "d1" || v.Header.Get("X-Request-Id") != "cardinality" {
					t.Fatalf("value=%+v error=%v", v, err)
				}
			} else if err == nil || v != nil {
				t.Fatalf("invalid shape accepted: %+v/%v", v, err)
			}
		})
	}
}

func TestAcceleratorBindingIDValidatorBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/devices/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) })
	client := cloud.Client("accelerator", "/v2")
	validate := func(id string) error {
		if id != "safe" {
			return resource.ErrInvalidOption
		}
		return nil
	}
	c := cyborg.Collection(client, "devices", "device", "devices", func(v *devices.Device) string { return v.UUID }, nil, nil,
		func(v *devices.Device) *common.Metadata { return &v.Metadata }, func(id string) string { return client.ServiceURL("devices", id) }, validate)
	for _, ref := range []resource.Ref{resource.ID("a,b"), resource.ID("unsafe")} {
		if err := c.Delete(context.Background(), ref); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if _, err := c.Get(context.Background(), ref.String()); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid ID reached HTTP")
	}
	if err := c.Delete(context.Background(), resource.ID("safe")); err != nil || calls.Load() != 1 {
		t.Fatalf("valid delete: %v/%d", err, calls.Load())
	}
}
