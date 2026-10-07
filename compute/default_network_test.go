package compute_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestServerDefaultNetworkDependencyFailureCancellationAndValidation(t *testing.T) {
	for _, scenario := range []string{"failure", "cancel", "invalid ID", "invalid name"} {
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
			if scenario == "cancel" {
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
