package rules

import (
	"context"
	"errors"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Shared request calls are sealed and cannot run caller code, so the
// option-phase guards of each apply loop are exercised here with internal
// state callbacks: cancellation and client changes between callbacks, nil
// callbacks, and callbacks that retain a pointer to the state.
func checkQoSApplyLoop[T any](t *testing.T, name string, newEngine func(context.Context, *API) (*neutronfind.Engine, error), apply func(context.Context, *neutronfind.Engine, []func(*T) error) (T, error), headers func(*T) *map[string]string) {
	t.Helper()
	for _, mode := range []string{"cancel", "client", "nil", "escape"} {
		cloud := testcloud.New(t)
		api := New(cloud.Client("network", "/catalog/unused/"))
		ctx, cancel := context.WithCancelCause(context.Background())
		engine, err := newEngine(ctx, api)
		if err != nil {
			t.Fatal(name, mode, err)
		}
		cause, optionCause := errors.New("cancelled"), errors.New("option error")
		var retained *T
		callback := func(s *T) error {
			switch mode {
			case "cancel":
				cancel(cause)
				return optionCause
			case "client":
				api.client.ResourceBase += "changed/"
				return optionCause
			}
			*headers(s) = map[string]string{"X-Owned": "yes"}
			retained = s
			return nil
		}
		callbacks := []func(*T) error{callback}
		if mode == "nil" {
			callbacks = append(callbacks, nil)
		}
		policy, err := apply(ctx, engine, callbacks)
		cancel(nil)
		switch mode {
		case "cancel":
			if !errors.Is(err, optionCause) || !errors.Is(err, cause) {
				t.Fatal(name, mode, err)
			}
		case "client":
			if !errors.Is(err, optionCause) || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(name, mode, err)
			}
		case "nil":
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(name, mode, err)
			}
		case "escape":
			(*headers(retained))["X-Owned"] = "late"
			if err != nil || (*headers(&policy))["X-Owned"] != "yes" {
				t.Fatal(name, mode, err, policy)
			}
		}
	}
}

func TestQoSRuleApplyLoopsGuardCallbacks(t *testing.T) {
	checkQoSApplyLoop(t, "bandwidth limit find", func(ctx context.Context, a *API) (*neutronfind.Engine, error) {
		return a.bandwidthLimitEngine(ctx, "parent")
	}, func(ctx context.Context, e *neutronfind.Engine, c []func(*bandwidthLimitFindState) error) (bandwidthLimitFindState, error) {
		return applyBandwidthLimitOptions(ctx, e, bandwidthLimitFindState{}, copyBandwidthLimitFind, c)
	}, func(s *bandwidthLimitFindState) *map[string]string { return &s.Headers })
	checkQoSApplyLoop(t, "DSCP marking list", func(ctx context.Context, a *API) (*neutronfind.Engine, error) {
		return a.dscpMarkingEngine(ctx, "parent")
	}, func(ctx context.Context, e *neutronfind.Engine, c []func(*dscpMarkingListState) error) (dscpMarkingListState, error) {
		return applyDSCPMarkingOptions(ctx, e, dscpMarkingListState{}, copyDSCPMarkingList, c)
	}, func(s *dscpMarkingListState) *map[string]string { return &s.Headers })
	checkQoSApplyLoop(t, "minimum bandwidth get", func(ctx context.Context, a *API) (*neutronfind.Engine, error) {
		return a.minimumBandwidthEngine(ctx, "parent")
	}, func(ctx context.Context, e *neutronfind.Engine, c []func(*minimumBandwidthGetState) error) (minimumBandwidthGetState, error) {
		return applyMinimumBandwidthOptions(ctx, e, minimumBandwidthGetState{}, copyMinimumBandwidthGet, c)
	}, func(s *minimumBandwidthGetState) *map[string]string { return &s.Headers })
	checkQoSApplyLoop(t, "minimum packet rate mutation", func(ctx context.Context, a *API) (*neutronfind.Engine, error) {
		return a.minimumPacketRateEngine(ctx, "parent")
	}, func(ctx context.Context, e *neutronfind.Engine, c []func(*packetRateMutationState) error) (packetRateMutationState, error) {
		return applyMinimumPacketRateOptions(ctx, e, packetRateMutationState{}, copyPacketRateMutation, c)
	}, func(s *packetRateMutationState) *map[string]string { return &s.Headers })
	checkQoSApplyLoop(t, "packet rate limit get", func(ctx context.Context, a *API) (*neutronfind.Engine, error) {
		return a.packetRateLimitEngine(ctx, "parent")
	}, func(ctx context.Context, e *neutronfind.Engine, c []func(*packetRateLimitGetState) error) (packetRateLimitGetState, error) {
		return applyPacketRateLimitOptions(ctx, e, packetRateLimitGetState{}, copyPacketRateLimitGet, c)
	}, func(s *packetRateLimitGetState) *map[string]string { return &s.Headers })
	checkQoSApplyLoop(t, "QoS rule delete", func(ctx context.Context, a *API) (*neutronfind.Engine, error) {
		return a.bandwidthLimitEnginePolicy(ctx, "parent", true)
	}, applyQoSRuleDelete, func(s *qosRuleDeleteState) *map[string]string { return &s.Headers })
}
