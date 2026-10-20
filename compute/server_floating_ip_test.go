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
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestServerFloatingIPPreflightPrecedesCreationAndLookups(t *testing.T) {
	getterError := errors.New("network unavailable")
	for _, scenario := range []string{"nil context", "canceled", "nil option", "nil server option", "nil IP option", "zero timeout", "name", "image", "external", "boot conflict", "network version", "server wait", "IP wait", "fixed IPv6", "project", "missing getter", "nil service", "getter error", "unknown scope"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("preflight made HTTP request", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			var getters atomic.Int32
			deps := compute.Dependencies{FloatingIPs: func(context.Context) (*network.FloatingIPs, error) {
				getters.Add(1)
				if scenario == "getter error" {
					return nil, getterError
				}
				if scenario == "nil service" {
					return nil, nil
				}
				return network.New(cloud.Client("network", "/v2.0")).FloatingIPs, nil
			}}
			if scenario == "missing getter" {
				deps.FloatingIPs = nil
			}
			request := compute.CreateServerWithFloatingIPRequest{Server: nicCreateRequest(), FloatingIPNetwork: resource.ID("external")}
			ctx := context.Background()
			var options []compute.CreateServerWithFloatingIPOption
			want := resource.ErrInvalidOption
			wantGetters := int32(0)
			switch scenario {
			case "nil context":
				ctx = nil
			case "canceled":
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx, want = canceled, context.Canceled
			case "nil option":
				options = []compute.CreateServerWithFloatingIPOption{nil}
			case "nil server option":
				options = []compute.CreateServerWithFloatingIPOption{compute.WithServerOptions(nil)}
			case "nil IP option":
				options = []compute.CreateServerWithFloatingIPOption{compute.WithFloatingIPOptions(nil)}
			case "zero timeout":
				options = []compute.CreateServerWithFloatingIPOption{compute.WithWorkflowTimeout(0)}
			case "name":
				request.Server.Name = " "
			case "image":
				request.Server.Image = resource.Name("")
			case "external":
				request.FloatingIPNetwork = resource.Name("")
			case "boot conflict":
				options = []compute.CreateServerWithFloatingIPOption{compute.WithServerOptions(compute.WithBootVolume(resource.ID("volume")))}
			case "network version":
				options = []compute.CreateServerWithFloatingIPOption{compute.WithServerOptions(compute.WithNetworkMode("auto"))}
				want = resource.ErrUnsupported
			case "server wait":
				options = []compute.CreateServerWithFloatingIPOption{compute.WithServerOptions(compute.WithWait(resource.WithStatusAttribute("missing")))}
				want = resource.ErrUnsupported
			case "IP wait":
				options = []compute.CreateServerWithFloatingIPOption{compute.WithFloatingIPOptions(network.WithEnsureWait(resource.WithPollInterval(-time.Second)))}
			case "fixed IPv6":
				options = []compute.CreateServerWithFloatingIPOption{compute.WithFloatingIPOptions(network.WithEnsureFixedAddress("2001:db8::1"))}
			case "project":
				options = []compute.CreateServerWithFloatingIPOption{compute.WithFloatingIPOptions(network.WithEnsureProject("bad/id"))}
			case "missing getter":
				want = resource.ErrUnsupported
			case "nil service", "unknown scope":
				want, wantGetters = resource.ErrUnsupported, 1
			case "getter error":
				want, wantGetters = getterError, 1
			}
			result, err := compute.New(cloud.Client("compute", "/compute"), deps).Servers.CreateWithFloatingIP(ctx, request, options...)
			if result != nil || !errors.Is(err, want) || getters.Load() != wantGetters {
				t.Fatalf("result=%+v err=%v getters=%d", result, err, getters.Load())
			}
		})
	}
}
