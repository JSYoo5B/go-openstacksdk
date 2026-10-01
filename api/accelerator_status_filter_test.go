package api_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"gophercloudsdk/accelerator/v2/acceleratorrequests"
	"gophercloudsdk/accelerator/v2/devices"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestAcceleratorStatusFilterStaysLocal(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v2/accelerator_requests", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("status") || r.URL.Query().Get("instance") != "instance" {
			t.Errorf("ARQ query %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"arqs":[{"uuid":"a1","state":"Initial"},{"uuid":"a2","state":"Bound"}]}`)
	})
	cloud.Mux.HandleFunc("/v2/devices", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("status") || r.URL.Query().Get("hostname") != "compute" {
			t.Errorf("device query %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"devices":[{"uuid":"d1","status":"enabled"},{"uuid":"d2","status":"maintaining"}]}`)
	})
	client := cloud.Client("accelerator", "/v2")
	client.Microversion = "2.3"
	for _, option := range []resource.ListOption{resource.WithPageSize(10), resource.WithQuery("marker", "a1")} {
		if _, err := acceleratorrequests.New(client).All(context.Background(), option); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatalf("unsupported pagination: %v", err)
		}
	}
	arqs, err := acceleratorrequests.New(client).All(context.Background(), resource.WithStatus("bound"), resource.WithQuery("instance", "instance"))
	if err != nil || len(arqs) != 1 || arqs[0].UUID != "a2" {
		t.Fatalf("ARQ filter %+v/%v", arqs, err)
	}
	d, err := devices.New(client).All(context.Background(), resource.WithStatus("ENABLED"), resource.WithQuery("hostname", "compute"))
	if err != nil || len(d) != 1 || d[0].UUID != "d1" {
		t.Fatalf("device filter %+v/%v", d, err)
	}
}
