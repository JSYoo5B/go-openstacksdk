package blockstorage_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCinderFailurePrefixAndMicroversion(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("block-storage", "/v3/project")
	client.Microversion = "3.60"
	service := blockstorage.New(client)
	cloud.Mux.HandleFunc("GET /v3/project/volumes/id", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenStack-API-Version") != "volume 3.60" {
			t.Error("Cinder microversion header missing")
		}
		testcloud.JSON(w, 200, `{"volume":{"id":"id","name":"data","status":"error_deleting","size":10}}`)
	})
	_, err := service.Volumes.Wait(context.Background(), resource.ID("id"), "available")
	var failure *resource.FailedStateError
	if !errors.Is(err, resource.ErrFailedState) || !errors.As(err, &failure) || failure.Status != "error_deleting" {
		t.Fatal(err)
	}
}
