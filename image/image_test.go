package image_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"gophercloudsdk/image"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestFlatGlanceResponseAndFailureStates(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("GET /v2/images/ready", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"ready","name":"ubuntu","status":"active","size":1024,"hw_architecture":"aarch64"}`)
	})
	cloud.Mux.HandleFunc("GET /v2/images/failed", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"failed","status":"killed"}`)
	})
	ctx := context.Background()
	ready, err := service.Images.Wait(ctx, resource.ID("ready"), "ACTIVE")
	if err != nil || ready.SizeBytes != 1024 || ready.Properties["hw_architecture"] != "aarch64" {
		t.Fatalf("ready=%v err=%v", ready, err)
	}
	if _, err := service.Images.Wait(ctx, resource.ID("failed"), "active"); !errors.Is(err, resource.ErrFailedState) {
		t.Fatal(err)
	}
}
