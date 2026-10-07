package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFloatingIPDeleteNegativeAndAdditionalRetryLimits(t *testing.T) {
	for _, retries := range []int{-2, 3} {
		t.Run(fmt.Sprint(retries), func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 204, ""
				}
				return 200, `{"floatingips":[{"id":"ip","status":"ACTIVE"}]}`
			}
			result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"}, compute.WithFloatingIPDeleteRetries(retries))
			expected := max(0, retries) + 1
			var failure *compute.FloatingIPDeleteVerificationError
			if !errors.As(err, &failure) || failure.Attempts != expected || len(result.Attempts) != expected || len(state.events) != 2*expected || result.Deleted {
				t.Fatal(result, err, state)
			}
		})
	}
}
func TestFloatingIPDeleteVerificationConsumesPagesAndRejectsIncompleteSelection(t *testing.T) {
	for _, mode := range []string{"late match", "403", "malformed", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 204, ""
				}
				if r.URL.Query().Get("marker") == "" {
					first := `{"id":"other","status":"DOWN"}`
					if mode == "duplicate" {
						first = `{"id":"ip","status":"DOWN"}`
					}
					return 200, `{"floatingips":[` + first + `],"floatingips_links":[{"rel":"next","href":"?marker=next"}]}`
				}
				if mode == "403" {
					return 403, "late denied"
				}
				if mode == "malformed" {
					return 200, `{"floatingips":{}}`
				}
				return 200, `{"floatingips":[{"id":"ip","status":"DOWN"}]}`
			}
			result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"})
			if len(result.Attempts) != 1 || !result.Attempts[0].Accepted || len(state.events) != 3 {
				t.Fatal(result, err, state)
			}
			if mode == "late match" {
				if err != nil || !result.Deleted || !result.Down || len(result.LastVerification.Pages) != 2 {
					t.Fatal(result, err)
				}
			} else if err == nil || result.Deleted || result.Attempts[0].Verified || result.LastVerification == nil {
				t.Fatal(result, err)
			}
			if mode == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			if mode == "403" && (result.Failure == nil || result.Failure.StatusCode != 403) {
				t.Fatal(result)
			}
			if mode == "malformed" && (result.Failure == nil || result.Failure.StatusCode != 200 || string(result.Failure.Envelope) != `{"floatingips":{}}` || len(result.LastVerification.Pages) != 2) {
				t.Fatal(result)
			}

		})
	}
}
func TestFloatingIPDeleteDirectVerificationAndMemberNotFound(t *testing.T) {
	id := "12345678-1234-1234-1234-123456789abc"
	for _, mode := range []string{"default", "direct missing", "direct wrong ID DOWN", "nonUUID"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 204, ""
				}
				if mode == "direct missing" {
					return 404, "member absent"
				}
				if mode == "direct wrong ID DOWN" {
					return 200, `{"floatingip":{"id":"other","status":"DOWN"}}`
				}
				return 200, `{"floatingips":[]}`
			}
			target := id
			if mode == "nonUUID" {
				target = "ip"
			}
			var options []compute.FloatingIPDeleteOption
			if mode != "default" {
				options = append(options, compute.WithFloatingIPDeleteDirectGet(true))
			}
			result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: target}, options...)
			if len(state.events) != 2 || !result.Attempts[0].Accepted {
				t.Fatal(result, err, state)
			}
			if mode == "direct missing" {
				if !gophercloud.ResponseCodeIs(err, 404) || result.Deleted || result.Attempts[0].NotFound != nil || result.Failure.StatusCode != 404 {
					t.Fatal(result, err)
				}
			} else if err != nil || !result.Deleted {
				t.Fatal(result, err)
			}
			if mode == "default" || mode == "nonUUID" {
				if state.events[1] != "GET /delete-network/v2.0/floatingips" || !result.Absent {
					t.Fatal(result, state)
				}
			} else if !strings.HasSuffix(state.events[1], "/"+id) {
				t.Fatal(state.events)
			}
			if mode == "direct wrong ID DOWN" {
				if result.Absent || !result.Down {
					t.Fatal(result)
				}
				queryRaw(t, result.LastVerification.FloatingIP.Wire, "id", `"other"`)
			}
		})
	}
}
func TestFloatingIPDeleteAcceptedFailureRetainsReceiptAndStops(t *testing.T) {
	for _, mode := range []string{"close", "cancel", "source"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			cached, err := conn.Network(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted delete failure")
			state.reply = func(*http.Request) (int, string) { return 202, "passive accepted body" }
			state.afterClose = func(*http.Request) error {
				switch mode {
				case "close":
					return cause
				case "cancel":
					cancel(cause)
				case "source":
					cached.API = nil
				}
				return nil
			}
			result, err := conn.DeleteFloatingIP(ctx, compute.DeleteFloatingIPRequest{ID: "ip"})
			var proof *resource.ResponseError
			if err == nil || !errors.As(err, &proof) || proof.StatusCode != 202 || result.Deleted || len(state.events) != 1 || len(result.Attempts) != 1 || !result.Attempts[0].Accepted || result.Attempts[0].Verified || result.LastVerification != nil || string(result.Attempts[0].Response.Envelope) != "passive accepted body" {
				t.Fatal(result, err, state)
			}
			if mode != "source" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if result.Failure.StatusCode != 202 || result.Failure.Header.Get("X-Delete-Proof") != "DELETE /delete-network/v2.0/floatingips/ip" {
				t.Fatal(result)
			}
		})
	}
}
func TestFloatingIPDeleteVerificationAcceptedFailureKeepsPriorDelete(t *testing.T) {
	for _, mode := range []string{"close", "cancel", "source"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			cached, err := conn.Network(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("verification failed")
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 204, ""
				}
				return 200, `{"floatingips":[{"id":"ip","status":"DOWN"}]}`
			}
			state.afterClose = func(r *http.Request) error {
				if r.Method == "GET" {
					switch mode {
					case "close":
						return cause
					case "cancel":
						cancel(cause)
					case "source":
						cached.RawClient().Endpoint += "changed/"
					}
				}
				return nil
			}
			result, err := conn.DeleteFloatingIP(ctx, compute.DeleteFloatingIPRequest{ID: "ip"})
			var proof *resource.ResponseError
			if err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || result.Deleted || len(state.events) != 2 || !result.Attempts[0].Accepted || result.Attempts[0].Response.StatusCode != 204 || result.Attempts[0].Verified || result.LastVerification == nil || result.Failure.StatusCode != 200 {
				t.Fatal(result, err, state)
			}
			if mode != "source" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}
func TestFloatingIPDeleteHTTPFailureAndRetryTerminalAreNotMissing(t *testing.T) {
	for _, mode := range []string{"403", "503", "expanded404", "source retry"} {
		t.Run(mode, func(t *testing.T) {
			cloud, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			cached, err := conn.Network(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			code := 403
			if mode == "503" || mode == "source retry" {
				code = 503
			}
			if mode == "expanded404" {
				code = 404
			}
			state.reply = func(*http.Request) (int, string) { return code, "physical original" }
			callbacks := 0
			if mode == "expanded404" || mode == "source retry" {
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
					callbacks++
					if mode == "expanded404" {
						o.OkCodes = append(o.OkCodes, 404)
					} else {
						cached.RawClient().Endpoint += "changed/"
					}
					return nil
				}
			}
			result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"})
			if err == nil || !gophercloud.ResponseCodeIs(err, code) || result.Deleted || len(result.Attempts) != 1 || result.Attempts[0].Accepted || result.Attempts[0].NotFound != nil || result.Failure.StatusCode != code || string(result.Failure.Envelope) != "physical original" {
				t.Fatal(result, err, state)
			}
			expected := 1
			if mode == "expanded404" {
				expected = 2
				var terminal interface{ TerminalSDKFailure() bool }
				if !errors.As(err, &terminal) || !terminal.TerminalSDKFailure() {
					t.Fatal(err)
				}
			}
			if len(state.events) != expected || ((mode == "expanded404" || mode == "source retry") && callbacks != 1) {
				t.Fatal(state, callbacks)
			}
		})
	}
}
func TestFloatingIPDeleteWholeBudgetOptionsAndParentDeadline(t *testing.T) {
	for _, mode := range []string{"default", "timeout", "unlimited", "parent"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			gets := 0
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					return 204, ""
				}
				gets++
				if gets == 1 {
					return 200, `{"floatingips":[{"id":"ip","status":"ACTIVE"}]}`
				}
				return 200, `{"floatingips":[]}`
			}
			input := compute.DeleteFloatingIPRequest{ID: "ip"}
			calls := 0
			ctx := context.Background()
			var parent time.Time
			var options []compute.FloatingIPDeleteOption
			if mode != "default" {
				options = append(options, compute.WithFloatingIPDeleteTimeout(time.Minute))
			}
			if mode == "unlimited" {
				options = append(options, compute.WithUnlimitedFloatingIPDeleteTimeout())
			}
			if mode == "parent" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				parent, _ = ctx.Deadline()
			}
			options = append(options, func(*compute.FloatingIPDeleteOpts) error { calls++; input.ID = "outside"; return nil })
			result, err := conn.DeleteFloatingIP(ctx, input, options...)
			if err != nil || !result.Deleted || result.ID != "ip" || calls != 1 || len(state.deadlines) != 4 {
				t.Fatal(result, err, state, calls)
			}
			for _, deadline := range state.deadlines {
				if !deadline.Equal(state.deadlines[0]) {
					t.Fatal("reset budget", state.deadlines)
				}
			}
			switch mode {
			case "default", "unlimited":
				if !state.deadlines[0].IsZero() {
					t.Fatal(state.deadlines)
				}
			case "timeout":
				if state.deadlines[0].IsZero() {
					t.Fatal(state.deadlines)
				}
			case "parent":
				if !state.deadlines[0].Equal(parent) {
					t.Fatal(state.deadlines, parent)
				}
			}
		})
	}
}
func TestFloatingIPDeleteRealDeadlineStopsAfterAcceptedDelete(t *testing.T) {
	_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
	state.reply = func(r *http.Request) (int, string) {
		if r.Method == "DELETE" {
			return 204, ""
		}
		<-r.Context().Done()
		return 503, "deadline during verification"
	}
	result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"}, compute.WithFloatingIPDeleteTimeout(60*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) || len(state.events) != 2 || len(result.Attempts) != 1 || !result.Attempts[0].Accepted || result.Deleted || result.Attempts[0].Response.StatusCode != 204 {
		t.Fatal(result, err, state)
	}
}
func TestFloatingIPDeletePreflightCapturedServiceAndLegacyVersion(t *testing.T) {
	for _, mode := range []string{"nil connection", "nil context", "cancelled", "unsafeID", "nil option", "source change", "legacy version"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNova)
			ctx := context.Background()
			input := compute.DeleteFloatingIPRequest{ID: "ip"}
			var options []compute.FloatingIPDeleteOption
			cause := errors.New("preflight cancel")
			if mode == "nil connection" {
				conn = (*sdk.Connection)(nil)
			}
			if mode == "nil context" {
				ctx = nil
			}
			if mode == "cancelled" {
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(cause)
			}
			if mode == "unsafeID" {
				input.ID = "bad/id"
			}
			if mode == "nil option" {
				options = append(options, nil)
			}
			if mode == "source change" || mode == "legacy version" {
				service, err := conn.Compute(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "legacy version" {
					service.RawClient().Microversion = "2.36"
				} else {
					options = append(options, func(*compute.FloatingIPDeleteOpts) error { service.Servers = nil; return nil })
					_, err = service.DeleteFloatingIP(ctx, input, options...)
					if !errors.Is(err, resource.ErrInvalidOption) || len(state.events) != 0 {
						t.Fatal(err, state)
					}
					return
				}
			}
			result, err := conn.DeleteFloatingIP(ctx, input, options...)
			if err == nil || len(state.events) != 0 {
				t.Fatal(result, err, state)
			}
			if mode == "cancelled" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode == "legacy version" && !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(err)
			}
			if mode != "legacy version" && !reflect.DeepEqual(state.locators, []string(nil)) {
				t.Fatal(state.locators)
			}
		})
	}
}

func TestFloatingIPDeleteAcceptedReadErrorPreservesBodyAndCause(t *testing.T) {
	_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
	cause := errors.New("DELETE response read failure")
	state.readError = cause
	state.reply = func(*http.Request) (int, string) { return 202, "partial delete receipt" }
	result, err := conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"})
	var proof *resource.ResponseError
	if !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 202 || result.Deleted || len(result.Attempts) != 1 || !result.Attempts[0].Accepted || len(state.events) != 1 || result.LastVerification != nil || string(result.Attempts[0].Response.Envelope) != "partial delete receipt" || string(proof.Body) != "partial delete receipt" {
		t.Fatal(result, err, state)
	}
}

func TestFloatingIPDeleteServiceUsesCachedConnectionDependencies(t *testing.T) {
	_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
	service, err := conn.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state.reply = func(*http.Request) (int, string) { return 204, "" }
	result, err := service.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"}, compute.WithFloatingIPDeleteRetries(0))
	if err != nil || !result.Deleted || result.Backend != compute.FloatingIPNeutron || !reflect.DeepEqual(state.locators, []string{"compute", "network"}) || len(state.events) != 1 {
		t.Fatal(result, err, state)
	}
	result, err = conn.DeleteFloatingIP(context.Background(), compute.DeleteFloatingIPRequest{ID: "ip"}, compute.WithFloatingIPDeleteRetries(0))
	if err != nil || !result.Deleted || len(state.locators) != 2 || len(state.events) != 2 {
		t.Fatal(result, err, state)
	}
}
