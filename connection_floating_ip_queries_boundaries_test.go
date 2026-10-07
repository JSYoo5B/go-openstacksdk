package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute"
	"gophercloudsdk/resource"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFloatingIPQueryGetZeroOneMultipleAndExpressionValues(t *testing.T) {
	_, conn, state := floatingQueryFixture(t, compute.FloatingIPNova)
	state.reply = func(r *http.Request) (int, string) { return 200, `{"floating_ips":[{"id":"a"},{"id":"b"}]}` }
	result, err := conn.GetFloatingIP(context.Background(), compute.GetFloatingIPRequest{ID: "absent"})
	if err != nil || result.FloatingIP != nil || result.Value != nil {
		t.Fatal(result, err)
	}
	result, err = conn.GetFloatingIP(context.Background(), compute.GetFloatingIPRequest{ID: "a"})
	if err != nil || result.FloatingIP == nil {
		t.Fatal(result, err)
	}
	_, err = conn.GetFloatingIP(context.Background(), compute.GetFloatingIPRequest{})
	if !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	for expression, want := range map[string]string{"`[false]`": "false", "`[0]`": "0", "`[\"\"]`": `""`, "`null`": "", "[].id": ""} {
		result, err = conn.GetFloatingIP(context.Background(), compute.GetFloatingIPRequest{}, compute.WithFloatingIPQueryExpression(expression))
		if expression == "[].id" {
			if !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(result, err)
			}
			continue
		}
		if err != nil || result.FloatingIP != nil || string(result.Value) != want {
			t.Fatal(expression, result, err)
		}
	}
}
func TestFloatingIPQueryExistingBypassesOptionsAndDiscovery(t *testing.T) {
	_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
	existing := &compute.FloatingIPRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage("null")}}}}
	callbacks := 0
	result, err := conn.GetFloatingIP(context.Background(), compute.GetFloatingIPRequest{Existing: existing}, func(*compute.FloatingIPQueryOpts) error { callbacks++; return errors.New("unused") }, nil)
	if err != nil || result.FloatingIP != existing || callbacks != 0 || len(state.locators) != 0 || len(state.paths) != 0 {
		t.Fatal(result, err, state)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("adoption cancelled")
	cancel(cause)
	_, err = conn.GetFloatingIP(ctx, compute.GetFloatingIPRequest{Existing: existing})
	if !errors.Is(err, cause) || callbacks != 0 {
		t.Fatal(err)
	}
}
func TestFloatingIPQueryMemberProjectionAndUUIDDirectGet(t *testing.T) {
	for _, id := range []string{"12345678-1234-1234-1234-123456789abc", "12345678123412341234123456789abc", "{12345678-1234-1234-1234-123456789ABC}", "urn:uuid:12345678-1234-1234-1234-123456789abc"} {
		t.Run(id, func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				return 200, `{"floatingip":{"floating_ip_address":null,"extension":true}}`
			}
			result, err := conn.GetFloatingIP(context.Background(), compute.GetFloatingIPRequest{ID: id}, compute.WithFloatingIPQueryDirectGet(true), compute.WithFloatingIPQueryFilters(json.RawMessage(`invalid`)))
			if err != nil || result.Observed.StatusCode != 200 || len(state.paths) != 1 || !strings.Contains(state.paths[0], "/floatingips/") {
				t.Fatal(result, err, state)
			}
			queryRaw(t, result.FloatingIP.Resource, "id", fmt.Sprintf("%q", id))
			queryRaw(t, result.FloatingIP.Resource, "if_match", `["actual-tag"]`)
			if _, exists := result.FloatingIP.Wire.Body["id"]; exists {
				t.Fatal("invented wire ID")
			}
		})
	}
	for _, body := range []string{`{"floatingip":{"id":null}}`, `{"floatingip":{"id":"different"}}`} {
		_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
		state.reply = func(*http.Request) (int, string) { return 200, body }
		result, err := conn.GetFloatingIPByID(context.Background(), compute.GetFloatingIPByIDRequest{ID: "requested"})
		if err != nil || string(result.FloatingIP.Resource.Body["id"]) == `"requested"` {
			t.Fatal(result, err)
		}
	}
}
func TestFloatingIPQueryMemberFailuresNeverFallback(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNeutron, compute.FloatingIPNova} {
		for _, code := range []int{404, 403, 204} {
			t.Run(fmt.Sprint(source, code), func(t *testing.T) {
				_, conn, state := floatingQueryFixture(t, source)
				state.reply = func(*http.Request) (int, string) { return code, `{"error":"actual"}` }
				result, err := conn.GetFloatingIPByID(context.Background(), compute.GetFloatingIPByIDRequest{ID: "row"})
				if !gophercloud.ResponseCodeIs(err, code) || len(state.paths) != 1 || result.FallbackError != nil || result.SuppressedNotFound != nil || result.Failure.StatusCode != code {
					t.Fatal(result, err, state)
				}
			})
		}
	}
	_, conn, state := floatingQueryFixture(t, compute.FloatingIPNova)
	state.reply = func(*http.Request) (int, string) { return 200, `{"floating_ip":{"ip":"known","extension":1}}` }
	result, err := conn.GetFloatingIPByID(context.Background(), compute.GetFloatingIPByIDRequest{ID: "row"})
	var accepted *resource.ResponseError
	if !errors.As(err, &accepted) || result.Observed == nil || result.FloatingIP.Wire == nil || result.FloatingIP.Resource != nil || result.Value != nil || accepted.StatusCode != 200 {
		t.Fatal(result, err)
	}
	queryRaw(t, result.FloatingIP.Wire, "ip", `"known"`)
}
func TestFloatingIPQueryPoolsProjectionSearchAndErrors(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNeutron, compute.FloatingIPNova, compute.FloatingIPNone} {
		t.Run(string(source), func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, source)
			state.reply = func(*http.Request) (int, string) {
				return 200, `{"floating_ip_pools":[{"name":"public","vendor":1},{"name":"private","vendor":2},{"name":null}]}`
			}
			result, err := conn.ListFloatingIPPools(context.Background())
			if err != nil || len(result.Pools) != 3 || len(result.Pools[0].Body) != 1 || !reflect.DeepEqual(state.locators, []string{"compute"}) || !strings.Contains(string(result.Pages[0].Envelope), "vendor") {
				t.Fatal(result, err, state)
			}
			result, err = conn.SearchFloatingIPPools(context.Background(), compute.SearchFloatingIPPoolsRequest{Name: "pub*"}, compute.WithFloatingIPQueryFilters(json.RawMessage(`{"name":"public"}`)))
			if err != nil || len(result.Pools) != 1 {
				t.Fatal(result, err)
			}
			result, err = conn.SearchFloatingIPPools(context.Background(), compute.SearchFloatingIPPoolsRequest{}, compute.WithFloatingIPQueryExpression("[].name"))
			if err != nil || result.Pools != nil || string(result.Value) != `["public","private"]` {
				t.Fatal(result, err)
			}
			result, err = conn.ListFloatingIPPools(context.Background(), compute.WithFloatingIPQueryFilters(json.RawMessage(`{}`)))
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(result, err)
			}
			state.reply = func(*http.Request) (int, string) { return 404, `{"missing":"pools"}` }
			result, err = conn.ListFloatingIPPools(context.Background())
			if !gophercloud.ResponseCodeIs(err, 404) || result.Failure.StatusCode != 404 {
				t.Fatal(result, err)
			}
			state.reply = func(*http.Request) (int, string) { return 200, `{"floating_ip_pools":[{"other":"missing"}]}` }
			result, err = conn.ListFloatingIPPools(context.Background())
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || result.Value != nil || result.Pools != nil || len(result.Pages) != 1 || proof.StatusCode != 200 {
				t.Fatal(result, err)
			}
		})
	}
}
func TestFloatingIPQueryNeutronPagesControlsAndRawLimit(t *testing.T) {
	for _, control := range []string{"pages", "single", "max", "numeric-marker", "empty-link"} {
		t.Run(control, func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.URL.Query().Get("marker") != "" {
					return 200, `{"floatingips":[]}`
				}
				if control == "empty-link" {
					return 200, `{"floatingips":[],"floatingips_links":[{"rel":"next","href":"?marker=next"}]}`
				}
				if control == "numeric-marker" {
					return 200, `{"floatingips":[{"id":9007199254740993}]}`
				}
				return 200, `{"floatingips":[{"id":"a","dns_name":"keep"},{"id":"b","dns_name":"drop"}],"floatingips_links":[{"rel":"next","href":"?marker=next"}]}`
			}
			filters := "{}"
			switch control {
			case "single":
				filters = `{"paginated":false}`
			case "max":
				filters = `{"max_items":1,"dns_name":"drop"}`
			case "numeric-marker":
				filters = `{"limit":1}`
			}
			result, err := conn.ListFloatingIPs(context.Background(), compute.WithFloatingIPQueryFilters(json.RawMessage(filters)))
			if err != nil {
				t.Fatal(result, err, state)
			}
			expected := 2
			if control == "single" || control == "max" || control == "empty-link" {
				expected = 1
			}
			if len(state.paths) != expected || len(result.Pages) != expected {
				t.Fatal(result, state)
			}
			if control == "max" && (len(result.FloatingIPs) != 0 || string(result.Value) != "[]" || state.queries[0].Get("limit") != "1") {
				t.Fatal(result, state)
			}
			if control == "numeric-marker" && state.queries[1].Get("marker") != "9007199254740993" {
				t.Fatal(state.queries)
			}
		})
	}
}
func TestFloatingIPQueryLatePageFailureNoLogicalPrefix(t *testing.T) {
	for _, late := range []string{"403", "404", "malformed", "cycle"} {
		t.Run(late, func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if strings.Contains(r.URL.Path, "compute") {
					return 200, `{"floating_ips":[{"id":"fallback"}]}`
				}
				if r.URL.Query().Get("marker") == "" {
					return 200, `{"floatingips":[{"id":"prefix"}],"floatingips_links":[{"rel":"next","href":"?marker=next"}]}`
				}
				switch late {
				case "403":
					return 403, `{"denied":true}`
				case "404":
					return 404, `{"late":true}`
				case "malformed":
					return 200, `{"floatingips":{}}`
				default:
					return 200, `{"floatingips":[{"id":"repeated"}],"floatingips_links":[{"rel":"next","href":"?marker=next"}]}`
				}
			}
			result, err := conn.ListFloatingIPs(context.Background())
			if late == "404" {
				if err != nil || result.FallbackError == nil || len(result.FloatingIPs) != 1 {
					t.Fatal(result, err)
				}
				queryRaw(t, result.FloatingIPs[0].Resource, "id", `"fallback"`)
			} else if err == nil || result.Value != nil || result.FloatingIPs != nil || len(result.Pages) < 1 {
				t.Fatal(result, err)
			}
			if late == "403" && (result.Failure == nil || result.Failure.StatusCode != 403) {
				t.Fatal(result)
			}
		})
	}
}
func TestFloatingIPQueryOptionsBudgetOwnershipAndSourceCapture(t *testing.T) {
	for _, mode := range []string{"default", "timeout", "unlimited", "parent"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if strings.Contains(r.URL.Path, "network") {
					return 404, `{"missing":true}`
				}
				return 200, `{"floating_ips":[{"id":1}]}`
			}
			ctx := context.Background()
			var options []compute.FloatingIPQueryOption
			var parent time.Time
			if mode == "parent" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				parent, _ = ctx.Deadline()
			}
			raw := json.RawMessage(`{}`)
			locationName := "owned"
			location := resource.CloudLocation{Cloud: &locationName}
			source := compute.FloatingIPNeutron
			options = append(options, compute.WithFloatingIPQueryOptions(compute.FloatingIPQueryOpts{Source: &source, Filters: &raw, Location: &location}))
			if mode != "default" {
				options = append(options, compute.WithFloatingIPQueryTimeout(time.Minute))
			}
			if mode == "unlimited" {
				options = append(options, compute.WithUnlimitedFloatingIPQueryTimeout())
			}
			source = compute.FloatingIPNova
			locationName = "changed"
			raw[0] = '['
			calls := 0
			options = append(options, func(*compute.FloatingIPQueryOpts) error { calls++; return nil })
			result, err := conn.ListFloatingIPs(ctx, options...)
			if err != nil || calls != 1 || len(state.deadlines) != 2 || !state.deadlines[0].Equal(state.deadlines[1]) {
				t.Fatal(result, err, state, calls)
			}
			if (mode == "default" || mode == "unlimited") && !state.deadlines[0].IsZero() {
				t.Fatal(state.deadlines)
			}
			if mode == "timeout" && state.deadlines[0].IsZero() || mode == "parent" && !state.deadlines[0].Equal(parent) {
				t.Fatal(state.deadlines, parent)
			}
			if !strings.Contains(string(result.FloatingIPs[0].Resource.Body["location"]), `"owned"`) {
				t.Fatal(result)
			}
		})
	}
	_, conn, state := floatingQueryFixture(t, compute.FloatingIPNova)
	service, err := conn.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ListFloatingIPs(context.Background(), beforeAvailableOption(func() { service.Servers = nil }, compute.WithFloatingIPQuerySource(compute.FloatingIPNova)))
	if !errors.Is(err, resource.ErrInvalidOption) || len(state.paths) != 0 {
		t.Fatal(err, state)
	}
}
func TestFloatingIPQueryAcceptedCloseAndCancellationRetainWire(t *testing.T) {
	for _, mode := range []string{"close", "cancel", "source"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, compute.FloatingIPNova)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("query close/cancel")
			service, err := conn.Compute(ctx)
			if err != nil {
				t.Fatal(err)
			}
			state.reply = func(*http.Request) (int, string) { return 200, `{"floating_ip":{"id":9,"ip":"known"}}` }
			state.afterClose = func() error {
				switch mode {
				case "cancel":
					cancel(cause)
				case "source":
					service.API = nil
				default:
					return cause
				}
				return nil
			}
			result, err := service.GetFloatingIPByID(ctx, compute.GetFloatingIPByIDRequest{ID: "9"})
			var proof *resource.ResponseError
			if err == nil || result.Observed == nil || result.FloatingIP == nil || result.FloatingIP.Wire == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || len(state.paths) != 1 {
				t.Fatal(result, err, state)
			}
			queryRaw(t, result.FloatingIP.Wire, "id", "9")
			if mode != "source" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode != "close" && (result.FloatingIP.Resource != nil || result.Value != nil) {
				t.Fatal(result)
			}
		})
	}
}
func TestFloatingIPQueryRetryCannotExpandAcceptedCodesOrFallback(t *testing.T) {
	cloud, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
	state.reply = func(*http.Request) (int, string) { return 404, `{"original":"not found"}` }
	retries := 0
	cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
		retries++
		options.OkCodes = append(options.OkCodes, 404)
		return nil
	}
	result, err := conn.ListFloatingIPs(context.Background())
	var terminal interface{ TerminalSDKFailure() bool }
	if !errors.As(err, &terminal) || !terminal.TerminalSDKFailure() {
		t.Fatal(err)
	}
	if err == nil || retries != 1 || len(state.paths) != 2 || result.FallbackError != nil || result.SuppressedNotFound != nil || result.Failure == nil || result.Failure.StatusCode != 404 || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(result, err, state, retries)
	}
}
func TestFloatingIPQueryPreflightAndMissingNetworkFallback(t *testing.T) {
	_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("early cancellation")
	cancel(cause)
	if _, err := conn.ListFloatingIPs(ctx); !errors.Is(err, cause) || len(state.locators) != 0 {
		t.Fatal(err, state)
	}
	if _, err := conn.GetFloatingIPByID(context.Background(), compute.GetFloatingIPByIDRequest{ID: "bad/id"}); !errors.Is(err, resource.ErrInvalidOption) || len(state.locators) != 0 {
		t.Fatal(err, state)
	}
	if _, err := conn.ListFloatingIPs(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) || len(state.locators) != 0 {
		t.Fatal(err, state)
	}
	state.catalogError = &gophercloud.ErrEndpointNotFound{}
	state.reply = func(*http.Request) (int, string) { return 200, `{"floating_ips":[{"id":"nova","status":"ERROR"}]}` }
	result, err := conn.ListFloatingIPs(context.Background())
	if err != nil || result.Backend != compute.FloatingIPNova || result.FloatingIPs[0].NormalizationSource != compute.FloatingIPNova || len(state.paths) != 1 {
		t.Fatal(result, err, state)
	}
	queryRaw(t, result.FloatingIPs[0].Resource, "status", `"ACTIVE"`)
}

func TestFloatingIPQueryNovaMemberStrictAndActualResponse(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprint(source, strict), func(t *testing.T) {
				_, conn, state := floatingQueryFixture(t, source)
				state.reply = func(*http.Request) (int, string) {
					return 200, `{"floating_ip":{"id":9007199254740993,"ip":"known","pool":"public","status":"ERROR","vendor":true}}`
				}
				result, err := conn.GetFloatingIPByID(context.Background(), compute.GetFloatingIPByIDRequest{ID: "9007199254740993"}, compute.WithFloatingIPQueryStrict(strict))
				if err != nil || result.FloatingIP == nil || result.Observed == nil || result.Failure != nil || result.Observed.StatusCode != 200 || len(state.paths) != 1 || result.FloatingIP.Backend != compute.FloatingIPNova {
					t.Fatal(result, err, state)
				}
				queryRaw(t, result.FloatingIP.Resource, "id", "9007199254740993")
				queryRaw(t, result.FloatingIP.Resource, "status", `"ACTIVE"`)
				queryRaw(t, result.FloatingIP.Wire, "status", `"ERROR"`)
				_, alias := result.FloatingIP.Resource.Body["floating_network_id"]
				_, vendor := result.FloatingIP.Resource.Body["vendor"]
				if alias == strict || vendor == strict {
					t.Fatal(result.FloatingIP.Resource)
				}
			})
		}
	}
}
func TestFloatingIPQueryRetrySourceChangeStopsBeforeSecondRequest(t *testing.T) {
	cloud, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
	cached, cacheErr := conn.Network(context.Background())
	if cacheErr != nil {
		t.Fatal(cacheErr)
	}
	state.reply = func(*http.Request) (int, string) { return 503, `{"original":"unavailable"}` }
	retries := 0
	cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error {
		retries++
		cached.RawClient().Endpoint += "changed/"
		return nil
	}
	result, err := conn.ListFloatingIPs(context.Background())
	if err == nil || !errors.Is(err, resource.ErrInvalidOption) || retries != 1 || len(state.paths) != 1 || result.Failure == nil || result.Failure.StatusCode != 503 || !gophercloud.ResponseCodeIs(err, 503) {
		t.Fatal(result, err, state, retries)
	}
}
