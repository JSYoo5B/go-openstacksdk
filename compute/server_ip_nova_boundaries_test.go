package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestNovaServerIPAcceptedActionFailurePreservesPerItemHistory(t *testing.T) {
	for _, scenario := range []string{"Close", "cancel", "source"} {
		f := newNovaIPFixture(t, "")
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("accepted Nova action processing failure")
		base := f.cloud.Provider.HTTPClient.Transport
		f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
			response, err := base.RoundTrip(r)
			if err == nil && r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/action") && response.Header.Get("X-Proof") == "Nova-attached-"+dispatchAddresses["b"] {
				response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error {
					switch scenario {
					case "Close":
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
		result, err := f.service.AddIPList(ctx, automaticServer(t, "null"), []string{dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]}, append(novaIPOptions(), compute.WithServerIPWait(true))...)
		cancel(nil)
		var proof *resource.ResponseError
		if err == nil || result == nil || !errors.As(err, &proof) || proof.StatusCode != 202 || proof.Header.Get("X-Proof") != "Nova-attached-"+dispatchAddresses["b"] || len(proof.Body) != 0 || len(result.Attempts) != 2 || !result.Attempts[0].Completed || !result.Attempts[0].Observed || result.Attempts[1].Completed || result.Attempts[1].Error == nil || result.Observed {
			t.Fatal(result, err, proof, f.trace())
		}
		a := result.NovaAssignment
		if a == nil || a.FloatingIP.ID != "2" || a.FloatingIP.InstanceID != nil || !a.ActionAccepted || a.ActionResponse.StatusCode != 202 || a.ActionResponse.Header.Get("X-Proof") != proof.Header.Get("X-Proof") || result.Server.Status != "BUILD" || !reflect.DeepEqual(f.trace(), []string{"list", "get:9007199254740993", "attach:" + dispatchAddresses["a"], "raw", "list", "get:2", "attach:" + dispatchAddresses["b"]}) {
			t.Fatal(result, err, f.trace())
		}
		if scenario == "Close" && !errors.Is(err, cause) || scenario == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) || scenario == "source" && !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(scenario, err)
		}
	}
}

func TestNovaServerIPAcceptedAllocationAndCompatibilityClosePreserveProof(t *testing.T) {
	for _, phase := range []string{"allocation", "compatibility"} {
		f := newNovaIPFixture(t, "")
		cause := errors.New("accepted " + phase + " Close")
		base := f.cloud.Provider.HTTPClient.Transport
		f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
			response, err := base.RoundTrip(r)
			match := phase == "allocation" && r.Method == "POST" && r.URL.Path == "/v2.1/os-floating-ips" || phase == "compatibility" && r.Method == "GET" && r.URL.Path == "/v2.1/os-floating-ips/29"
			if err == nil && match {
				response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error { return cause }}
			}
			return response, err
		})
		result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, "null")}, novaIPOptions(compute.WithFloatingIPPool(resource.Name("public")), compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false)))...)
		var proof *resource.ResponseError
		if !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 200 || result == nil || result.NovaAssignment == nil || !result.NovaAssignment.Allocated || result.NovaAssignment.FloatingIP.ID != "29" || result.NovaAssignment.FloatingIP.Address != dispatchAddresses["pool"] || result.NovaAssignment.ActionAccepted || result.NovaAssignment.AllocationResponse.Header.Get("X-Proof") != "Nova-allocated" {
			t.Fatal(result, err, proof)
		}
		want := []string{"allocate"}
		if phase == "compatibility" {
			want = append(want, "get:29")
		}
		if !reflect.DeepEqual(f.trace(), want) {
			t.Fatal(f.trace(), want)
		}
	}
}

func TestNovaServerIPOneHTTPBudgetAndOptionSnapshot(t *testing.T) {
	for _, mode := range []string{"default", "override", "parent", "unlimited"} {
		f := newNovaIPFixture(t, "")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		if mode == "parent" {
			cancel()
			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		}
		parentDeadline, _ := ctx.Deadline()
		var deadlines []time.Time
		base := f.cloud.Provider.HTTPClient.Transport
		f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
			d, ok := r.Context().Deadline()
			if !ok {
				t.Error("HTTP lacks shared budget")
			}
			deadlines = append(deadlines, d)
			return base.RoundTrip(r)
		})
		server := automaticServer(t, "null")
		addresses := []string{dispatchAddresses["a"], dispatchAddresses["b"]}
		inner, destination := 0, 0
		extra := []compute.AutomaticFloatingIPOption{beforeDispatchOption(func() { inner++; server.ID = "changed"; addresses[0] = dispatchAddresses["c"] }, compute.WithAutomaticIPEnabled(false)), compute.WithAutomaticEnsureOptions(beforeDispatchOption(func() { destination++ }, network.WithEnsureFixedAddress("10.0.0.10")))}
		if mode == "override" {
			extra = append(extra, compute.WithAutomaticIPTimeout(17*time.Second))
		}
		if mode == "unlimited" {
			extra = append(extra, compute.WithUnlimitedAutomaticIPTimeout())
		}
		start := time.Now()
		result, err := f.service.AddIPList(ctx, server, addresses, novaIPOptions(extra...)...)
		cancel()
		if err != nil || result == nil || len(deadlines) != 6 || inner != 1 || destination != 1 || result.Server.ID != "server" || result.Attempts[0].RequestedAddress != dispatchAddresses["a"] {
			t.Fatal(result, err, len(deadlines), inner, destination, f.trace())
		}
		for _, d := range deadlines {
			if !d.Equal(deadlines[0]) {
				t.Fatal("budget restarted", deadlines)
			}
		}
		if mode == "parent" || mode == "unlimited" {
			if !deadlines[0].Equal(parentDeadline) {
				t.Fatal(deadlines[0], parentDeadline)
			}
		} else {
			want := 60 * time.Second
			if mode == "override" {
				want = 17 * time.Second
			}
			if delta := deadlines[0].Sub(start); delta < want-time.Second || delta > want+time.Second {
				t.Fatal(mode, delta)
			}
		}
		if mode == "unlimited" && deadlines[0].Sub(start) <= 60*time.Second {
			t.Fatal("default cap remained", deadlines)
		}
	}
}

func TestNovaServerIPRetryCannotChangeOwnedActionOrSource(t *testing.T) {
	for _, mutation := range []string{"body", "source"} {
		f := newNovaIPFixture(t, "actionHTTP")
		retries := 0
		f.cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
			retries++
			if mutation == "body" {
				opts.JSONBody = json.RawMessage(`{"addFloatingIp":{"address":"198.51.100.99"}}`)
			} else {
				f.service.API = nil
			}
			return nil
		}
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, "null"), []string{dispatchAddresses["b"], dispatchAddresses["c"]}, novaIPOptions()...)
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Proof") != "Nova-denied" || !strings.Contains(string(native.Body), "second attach denied") || retries != 1 || result == nil || len(result.Attempts) != 1 || result.NovaAssignment.FloatingIP.ID != "2" || result.NovaAssignment.ActionAccepted || !reflect.DeepEqual(f.trace(), []string{"list", "get:2", "attach:" + dispatchAddresses["b"]}) {
			t.Fatal(result, err, retries, f.trace())
		}
	}
}

func TestNovaServerIPRealDeadlineStopsAfterAcceptedFirstAction(t *testing.T) {
	f := newNovaIPFixture(t, "")
	base := f.cloud.Provider.HTTPClient.Transport
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/v2.1/servers/server" {
			f.event("blocked-raw")
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		return base.RoundTrip(r)
	})
	result, err := f.service.AddIPList(context.Background(), automaticServer(t, "null"), []string{dispatchAddresses["a"], dispatchAddresses["b"]}, append(novaIPOptions(compute.WithAutomaticIPTimeout(time.Second)), compute.WithServerIPWait(true))...)
	if !errors.Is(err, context.DeadlineExceeded) || result == nil || result.NovaAssignment == nil || !result.NovaAssignment.ActionAccepted || len(result.Attempts) != 1 || result.Attempts[0].Completed || result.Attempts[0].Error == nil || result.Observed || !reflect.DeepEqual(f.trace(), []string{"list", "get:9007199254740993", "attach:" + dispatchAddresses["a"], "blocked-raw"}) {
		t.Fatal(result, err, f.trace())
	}
}

func TestNovaServerIPAutomaticDefaultPoolAndKnownSkips(t *testing.T) {
	f := newNovaIPFixture(t, "")
	server := automaticServer(t, autoFixed)
	server.Status = "BUILD"
	result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, novaIPOptions()...)
	if err != nil || result == nil || result.Mode != compute.ServerIPAutomatic || result.NovaAssignment == nil || result.NovaAssignment.FloatingIP.ID != "2" || !result.NovaAssignment.ActionAccepted || len(result.Attempts) != 0 || result.Observed || !reflect.DeepEqual(f.trace(), []string{"pools", "list", "get:2", "attach:" + dispatchAddresses["b"]}) {
		t.Fatal(result, err, f.trace())
	}
	for _, skip := range []string{"disabled", "private", "floating", "empty"} {
		f = newNovaIPFixture(t, "")
		server = automaticServer(t, autoFixed)
		server.Status = "BUILD"
		var extra []compute.AutomaticFloatingIPOption
		switch skip {
		case "disabled":
			extra = append(extra, compute.WithAutomaticIPEnabled(false))
		case "private":
			extra = append(extra, compute.WithAutomaticAddressOptions(compute.WithPrivateCloud(true)))
		case "floating":
			server = automaticServer(t, autoFloating)
		case "empty":
			server = automaticServer(t, `{}`)
		}
		result, err = f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, novaIPOptions(extra...)...)
		if err != nil || result == nil || result.Decision.Needed || result.NovaAssignment != nil || len(f.trace()) != 0 {
			t.Fatal(skip, result, err, f.trace())
		}
	}
}

func TestNovaServerIPReadinessAndCreationConsumeLegacyBackend(t *testing.T) {
	for _, mode := range []string{"Ensure", "Get async", "Get sync", "Wait", "Create"} {
		f := newNovaIPFixture(t, "")
		f.status = "ACTIVE"
		common := []compute.AutomaticFloatingIPOption{compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(compute.FloatingIPNova), compute.WithAddressReachability(false)), compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"]), compute.WithAutomaticEnsureOptions(network.WithEnsureFixedAddress("10.0.0.10")), compute.WithAutomaticIPPollInterval(time.Millisecond)}
		input := compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}
		ready := []compute.ServerReadyOption{compute.WithServerReadyAutomaticIPOptions(common...), compute.WithServerReadyWaitOptions(resource.WithPollInterval(time.Millisecond))}
		var result *compute.AutomaticServerIPResult
		var err error
		switch mode {
		case "Ensure":
			result, err = f.service.EnsureServerFloatingIP(context.Background(), input, common...)
		case "Get async":
			result, err = f.service.GetActiveServer(context.Background(), input, ready...)
		case "Get sync":
			result, err = f.service.GetActiveServer(context.Background(), input, append(ready, compute.WithActiveServerWait(true))...)
		case "Wait":
			input.Server.Status = "ERROR"
			result, err = f.service.WaitForServer(context.Background(), input, ready...)
		case "Create":
			f.cloud.Mux.HandleFunc("POST /v2.1/servers", func(w http.ResponseWriter, r *http.Request) {
				f.event("create")
				testServerCreateBody(t, r)
				testcloud.JSON(w, 202, `{"server":{"id":"server","adminPass":"initial"}}`)
			})
			created, createErr := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), compute.AutomaticServerCreateOptions{Server: automaticCreateOptions().Server, AutomaticIP: common})
			err = createErr
			if created == nil || created.Creation.AdminPass != "initial" || created.Server.AdminPass != "" {
				t.Fatal(created, err, f.trace())
			}
			result = created.Automatic
		}
		wait := mode != "Get async"
		if err != nil || result == nil || result.Assignment != nil || result.NovaAssignment == nil || len(result.Attempts) != 2 || result.Observed != wait || result.Server.Status != "ACTIVE" {
			t.Fatal(mode, result, err, f.trace())
		}
		for _, attempt := range result.Attempts {
			if !attempt.Completed || attempt.Observed != wait || attempt.NovaAssignment == nil || !attempt.NovaAssignment.ActionAccepted {
				t.Fatal(mode, attempt)
			}
		}
		wantRaw := 2
		if !wait {
			wantRaw = 0
		}
		if mode == "Wait" || mode == "Create" {
			wantRaw++
		}
		if f.gets != wantRaw {
			t.Fatal(mode, f.gets, f.trace())
		}
	}
}

func TestNovaServerIPReadOnlyPlanAndExplicitBackendChoice(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone, compute.FloatingIPNeutron} {
		f := newNovaIPFixture(t, "")
		options := []compute.AutomaticFloatingIPOption{compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(source), compute.WithAddressReachability(false)), compute.WithFloatingIPAddresses(dispatchAddresses["a"]), compute.WithAutomaticEnsureOptions(network.WithEnsureFixedAddress("10.0.0.10"))}
		decision, err := f.service.PlanServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, "null")}, options...)
		if err != nil || decision == nil || !decision.Needed || decision.Backend != compute.FloatingIPNova || len(decision.NovaSelections) != 0 || len(f.trace()) != 0 {
			t.Fatal(source, decision, err, f.trace())
		}
		result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, "null")}, compute.WithServerIPAutomaticOptions(options...))
		if err != nil || result.NovaAssignment == nil || !result.Attempts[0].Completed || !result.NovaAssignment.ActionAccepted {
			t.Fatal(source, result, err, f.trace())
		}
	}
}

func TestNovaServerIPSelectedDecoderAndPoolResponseBoundaries(t *testing.T) {
	for _, scenario := range []string{"canonicalNull", "fractionalID", "missingAssociation", "duplicate", "pool204", "poolEmpty"} {
		f := newNovaIPFixture(t, "")
		var options []compute.ServerIPOption
		switch scenario {
		case "canonicalNull":
			f.rows = novaIPRow("9007199254740993", dispatchAddresses["a"], "public", `"foreign"`) + `,{"id":8,"floating_ip_address":null,"ip":"198.51.100.10","pool":"public","instance_id":null}`
			options = novaIPOptions()
		case "fractionalID":
			f.rows = novaIPRow("1.5", dispatchAddresses["a"], "public", "null")
			options = novaIPOptions()
		case "missingAssociation":
			f.rows = strings.Replace(novaIPRow("9007199254740993", dispatchAddresses["a"], "public", "null"), `"instance_id":null,`, "", 1)
			options = novaIPOptions()
		case "duplicate":
			f.rows = novaIPRow("1", dispatchAddresses["a"], "public", "null") + "," + novaIPRow("2", dispatchAddresses["a"], "public", "null")
			options = novaIPOptions()
		default:
			base := f.cloud.Provider.HTTPClient.Transport
			f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/v2.1/os-floating-ip-pools" {
					f.event("synthetic-pools")
					body := `{"floating_ip_pools":[]}`
					code := 200
					if scenario == "pool204" {
						body = ""
						code = 204
					}
					return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				}
				return base.RoundTrip(r)
			})
			options = novaIPOptions()
		}
		var result *compute.AutomaticServerIPResult
		var err error
		if strings.HasPrefix(scenario, "pool") {
			server := automaticServer(t, autoFixed)
			server.Status = "BUILD"
			result, err = f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, options...)
		} else {
			result, err = f.service.AddIPList(context.Background(), automaticServer(t, "null"), []string{dispatchAddresses["a"]}, options...)
		}
		if scenario == "canonicalNull" {
			if err != nil || result.NovaAssignment.FloatingIP.ID != "9007199254740993" {
				t.Fatal(result, err)
			}
			continue
		}
		if err == nil || result == nil || result.NovaAssignment != nil || strings.Contains(strings.Join(f.trace(), ","), "allocate") || strings.Contains(strings.Join(f.trace(), ","), "attach:") {
			t.Fatal(scenario, result, err, f.trace())
		}
		if scenario == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) {
			t.Fatal(err)
		}
		if scenario == "poolEmpty" && !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
	}
}
