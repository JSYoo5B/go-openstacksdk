package compute_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestServerDefaultNetworkDependencyFailureCancellationAndValidation(t *testing.T) {
	for _, scenario := range []string{"failure", "cancel", "cancel with absent default", "invalid ID", "invalid name"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, creates atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				http.Error(w, "must not create", 500)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("default network unavailable")
			want := failure
			if scenario == "cancel" || scenario == "cancel with absent default" {
				want = context.Canceled
			} else if scenario != "failure" {
				want = resource.ErrInvalidOption
			}
			service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{
				DefaultNetwork: func(context.Context) (resource.Ref, error) {
					calls.Add(1)
					switch scenario {
					case "failure":
						return resource.Ref{}, failure
					case "cancel":
						cancel()
						return resource.ID("default-id"), nil
					case "cancel with absent default":
						cancel()
						return resource.Ref{}, nil
					case "invalid ID":
						return resource.ID("unsafe/path"), nil
					default:
						return resource.Name(""), nil
					}
				},
			})
			row, err := service.Servers.Create(ctx, nicCreateRequest())
			if row != nil || !errors.Is(err, want) || calls.Load() != 1 || creates.Load() != 0 {
				t.Fatalf("row=%v err=%v calls=%d creates=%d", row, err, calls.Load(), creates.Load())
			}
		})
	}
}

func TestServerDefaultNetworkDoesNotLeakBetweenRequestsAndPreservesWaitFailure(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, creates, waits atomic.Int32
	client := cloud.Client("compute", "/compute")
	client.Microversion = "2.37"
	service := compute.New(client, compute.Dependencies{
		DefaultNetwork: func(context.Context) (resource.Ref, error) {
			if calls.Add(1) == 1 {
				return resource.ID("default-id"), nil
			}
			return resource.Ref{}, nil
		},
	})
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		want := any("auto")
		if creates.Add(1) == 1 {
			want = []any{map[string]any{"uuid": "default-id"}}
		}
		checkNICServerBody(t, r, want)
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/servers/created", func(w http.ResponseWriter, r *http.Request) {
		waits.Add(1)
		testcloud.JSON(w, 200, `{"server":{"id":"created","status":"ERROR"}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected cleanup/lookup: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	row, err := service.Servers.Create(context.Background(), nicCreateRequest(), compute.WithWait(resource.WithTimeout(time.Second)))
	if !errors.Is(err, resource.ErrFailedState) || row == nil || row.ID != "created" || row.Status != "BUILD" {
		t.Fatalf("row=%v err=%v", row, err)
	}
	row, err = service.Servers.Create(context.Background(), nicCreateRequest())
	if err != nil || row == nil || row.ID != "created" || calls.Load() != 2 || creates.Load() != 2 || waits.Load() != 1 {
		t.Fatalf("row=%v err=%v calls=%d creates=%d waits=%d", row, err, calls.Load(), creates.Load(), waits.Load())
	}
}

func TestServerDefaultNetworkIsSkippedByExplicitNICsAndPreflight(t *testing.T) {
	for _, scenario := range []string{"empty NIC", "none", "invalid image", "invalid boot policy", "invalid mode version"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var creates atomic.Int32
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
			})
			client := cloud.Client("compute", "/compute")
			client.Microversion = "2.37"
			service := compute.New(client, compute.Dependencies{
				DefaultNetwork: func(context.Context) (resource.Ref, error) {
					t.Error("must not call default selection")
					return resource.Ref{}, errors.New("default lookup unexpectedly called")
				},
			})
			req := nicCreateRequest()
			var opts []compute.CreateServerOption
			switch scenario {
			case "empty NIC":
				opts = append(opts, compute.WithNetworkInterfaces(compute.ServerNetworkInterface{}))
			case "none":
				opts = append(opts, compute.WithNetworkMode("none"))
			case "invalid image":
				req.Image = resource.ID("unsafe/path")
			case "invalid boot policy":
				opts = append(opts, compute.WithDeleteBootVolumeOnTermination(true))
			default:
				opts = append(opts, compute.WithNetworkMode("auto"))
				client.Microversion = "2.36"
			}
			row, err := service.Servers.Create(context.Background(), req, opts...)
			if scenario == "empty NIC" || scenario == "none" {
				if err != nil || row == nil || creates.Load() != 1 {
					t.Fatalf("row=%v err=%v creates=%d", row, err, creates.Load())
				}
			} else if err == nil || row != nil || creates.Load() != 0 {
				t.Fatalf("row=%v err=%v creates=%d", row, err, creates.Load())
			}
		})
	}
}
