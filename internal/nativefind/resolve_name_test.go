package nativefind_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/nativefind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestGuardedCreationNameResolverPagesAndRetainsSource(t *testing.T) {
	type row struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	for _, scenario := range []string{"success", "late 403", "duplicate", "retry source", "204"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/v2")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /v2/images", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("name") != "wanted" {
					t.Error(r.URL)
				}
				if scenario == "retry source" {
					testcloud.JSON(w, 503, `{"error":"unavailable"}`)
					return
				}
				if scenario == "204" {
					w.WriteHeader(204)
					return
				}
				if r.URL.Query().Get("marker") != "" {
					if scenario == "late 403" {
						testcloud.JSON(w, 403, `{"error":"denied"}`)
						return
					}
					if scenario == "duplicate" {
						testcloud.JSON(w, 200, `{"images":[{"id":"second","name":"wanted"}]}`)
						return
					}
					testcloud.JSON(w, 300, `{"images":[]}`)
					return
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{"images":[{"id":"first","name":"wanted"}],"images_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2/images?marker=next"))
			})
			if scenario == "retry source" {
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					client.Endpoint = cloud.Server.URL + "/changed/"
					return nil
				}
			}
			ctx := rest.WithOperationSources(context.Background())
			id, err := nativefind.ResolveName(ctx, client, "images", "image", "images", resource.Name("wanted"), func(r *row) string { return r.ID }, func(r *row) string { return r.Name }, true, nil)
			switch scenario {
			case "success":
				if err != nil || id != "first" || calls.Load() != 2 {
					t.Fatal(id, err, calls.Load())
				}
				client.Endpoint = cloud.Server.URL + "/changed/"
				if !errors.Is(rest.CheckOperationGuard(ctx), resource.ErrInvalidOption) {
					t.Fatal("source binding discarded after resolution")
				}
			case "late 403":
				var unexpected gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &unexpected) || unexpected.Actual != 403 || id != "" || calls.Load() != 2 {
					t.Fatal(id, err, calls.Load())
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) || id != "" || calls.Load() != 2 {
					t.Fatal(id, err, calls.Load())
				}
			case "retry source":
				var unexpected gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &unexpected) || unexpected.Actual != 503 || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
					t.Fatal(id, err, calls.Load())
				}
			case "204":
				if !errors.Is(err, resource.ErrNotFound) || calls.Load() != 1 {
					t.Fatal(id, err, calls.Load())
				}
			}
		})
	}
}
