package compute_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestAutomaticIPObservationCapabilityAndReuseOwnerPreflight(t *testing.T) {
	for _, scenario := range []string{"missing Compute", "invalid Compute URL", "unscoped reuse"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			options := automaticOptions()
			switch scenario {
			case "missing Compute":
				f.service = compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }})
			case "invalid Compute URL":
				f.service.RawClient().Endpoint = ""
			case "unscoped reuse":
				options = append(options, compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(true)))
			}
			result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
			if err == nil || result == nil || !result.Decision.Needed || result.Assignment != nil || f.posts.Load() != 0 || f.raw.Load() != 0 {
				t.Fatal(result, err, f.posts.Load(), f.raw.Load())
			}
			if scenario != "invalid Compute URL" && !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(err)
			}
		})
	}
}

func TestAutomaticIPComputedAddressAndRoleUseFlagsPreserveNeedPolicy(t *testing.T) {
	for _, scenario := range []string{"untagged private", "global public", "external disabled", "both roles disabled"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			server := automaticServer(t, autoFixed)
			request := compute.AutomaticFloatingIPRequest{Server: server, Network: resource.ID("external")}
			switch scenario {
			case "untagged private":
				server.Addresses["private"] = []any{addressRow(4, "10.0.0.10", "")}
			case "global public":
				server.Addresses["private"] = []any{addressRow(4, "8.8.4.4", "fixed")}
			default:
				roleOptions := []network.NetworkRoleOption{network.WithExternalNetworkDiscovery(false)}
				if scenario == "both roles disabled" {
					roleOptions = append(roleOptions, network.WithInternalNetworkDiscovery(false))
				}
				policy, err := network.PrepareNetworkRoleOptions(roleOptions...)
				if err != nil {
					t.Fatal(err)
				}
				f.service = compute.New(f.service.RawClient(), compute.Dependencies{NetworkPolicy: policy, AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }})
				server.AccessIPv4 = "8.8.8.8"
			}
			result, err := f.service.EnsureServerFloatingIP(context.Background(), request, automaticOptions()...)
			if err != nil || result == nil {
				t.Fatal(result, err)
			}
			if scenario == "global public" {
				if result.Decision.Reason != compute.AutomaticIPExistingPublicIPv4 || result.Assignment != nil || f.posts.Load() != 0 {
					t.Fatal(result)
				}
			} else if !result.Decision.Needed || !result.Observed || result.Assignment == nil || f.posts.Load() != 1 {
				t.Fatal(result)
			}
		})
	}
}

type automaticTransport func(*http.Request) (*http.Response, error)

func (fn automaticTransport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

type automaticCloseBody struct {
	io.ReadCloser
	close func() error
}

func (body automaticCloseBody) Close() error {
	return errors.Join(body.ReadCloser.Close(), body.close())
}

func TestAutomaticIPAcceptedRawCloseFailuresRetainEvidenceWithoutRetry(t *testing.T) {
	for _, scenario := range []string{"close error", "cancel", "source"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted raw failure")
			base := f.cloud.Provider.HTTPClient.Transport
			f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
				response, err := base.RoundTrip(r)
				if err == nil && r.URL.Path == "/v2.1/servers/server" {
					response.Header.Set("X-Raw-Proof", "accepted")
					response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error {
						switch scenario {
						case "close error":
							return cause
						case "cancel":
							cancel(cause)
						case "source":
							f.service.API = nil
						}
						return nil
					}}
				}
				return response, err
			})
			server := automaticServer(t, autoFixed)
			server.Name = "original known"
			result, err := f.service.EnsureServerFloatingIP(ctx, compute.AutomaticFloatingIPRequest{Server: server}, automaticOptions()...)
			var proof *resource.ResponseError
			if err == nil || result == nil || result.Assignment == nil || !result.Assignment.Allocated || result.Server.Name != "original known" || result.Observed || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Raw-Proof") != "accepted" || len(proof.Body) == 0 || f.raw.Load() != 1 || f.posts.Load() != 1 {
				t.Fatal(result, err, proof)
			}
			if scenario == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal(err)
			}
			if scenario == "close error" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if scenario == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestAutomaticIPSelectionAbsenceWithAcceptedCancellationIsFailure(t *testing.T) {
	f := newAutomaticFixture(t)
	f.portRows = ""
	server := automaticServer(t, autoFixed)
	server.Status = "BUILD"
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("empty inventory canceled")
	base := f.cloud.Provider.HTTPClient.Transport
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		response, err := base.RoundTrip(r)
		if err == nil && r.URL.Path == "/v2.0/ports" {
			response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error { cancel(cause); return nil }}
		}
		return response, err
	})
	decision, err := f.service.PlanServerFloatingIP(ctx, compute.AutomaticFloatingIPRequest{Server: server}, automaticOptions()...)
	var proof *resource.ResponseError
	if decision == nil || decision.Reason != compute.AutomaticIPUndetermined || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 200 || f.posts.Load() != 0 {
		t.Fatal(decision, err, proof)
	}
}

func TestAutomaticIPRawRetrySourceChangePreservesAssignmentAndOriginalHTTPError(t *testing.T) {
	f := newAutomaticFixture(t)
	f.rawBody = func(int32) (int, string) { return 503, `{"error":"raw retry"}` }
	f.cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		f.service.Servers = compute.New(f.service.RawClient(), compute.Dependencies{}).Servers
		return nil
	}
	result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, automaticOptions()...)
	var original gophercloud.ErrUnexpectedResponseCode
	if result == nil || result.Assignment == nil || !result.Assignment.Allocated || result.Server.ID != "server" || result.Observed || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &original) || original.Actual != 503 || f.raw.Load() != 1 || f.posts.Load() != 1 {
		t.Fatal(result, err, original)
	}
}
