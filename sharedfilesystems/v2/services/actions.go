package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/manilaown"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const (
	// MinStateMicroversion is the first version with PUT /services/enable and /disable.
	MinStateMicroversion = 7
	// MinDisabledReasonMicroversion is the first version that accepts disabled_reason.
	MinDisabledReasonMicroversion = 83
	// MinEnsureSharesMicroversion is the first version with POST /services/ensure-shares.
	MinEnsureSharesMicroversion = 86
)

// ActionEvidence is the actual wire body, headers and status of an action.
type ActionEvidence = manilaown.Evidence

// ServiceState is the typed view of an enable/disable response. Manila returns
// host, binary and the disabled flag; the service object is also accepted
// under a "service" member, as the Python Resource does.
type ServiceState struct {
	Host           string         `json:"host"`
	Binary         string         `json:"binary"`
	Status         string         `json:"status"`
	Disabled       *bool          `json:"disabled"`
	DisabledReason string         `json:"disabled_reason"`
	Evidence       ActionEvidence `json:"-"`
}

func stateWrap(op string, err error) error { return request.Wrap(op, "services", err) }

func (a *API) validateTarget(ctx context.Context, host, binary string) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("%w: services API is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(host) == "" || strings.TrimSpace(binary) == "" {
		return fmt.Errorf("%w: host and binary are required", resource.ErrInvalidOption)
	}
	return nil
}

func (a *API) state(ctx context.Context, op, action string, floor int, body map[string]any) (*ServiceState, error) {
	resp, err := manilaown.Call{Client: a.client, Floor: floor, Method: http.MethodPut, URL: a.client.ServiceURL("services", action), Body: body, OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, stateWrap(op, err)
	}
	value, err := manilaown.Flat[ServiceState](resp, "service")
	return value, stateWrap(op, err)
}

// Enable sends PUT /services/enable {"host","binary"} (microversion 2.7 and
// later). Only HTTP 200 succeeds and nothing is retried.
func (a *API) Enable(ctx context.Context, host, binary string) (*ServiceState, error) {
	if err := a.validateTarget(ctx, host, binary); err != nil {
		return nil, stateWrap("Enable", err)
	}
	return a.state(ctx, "Enable", "enable", MinStateMicroversion, map[string]any{"host": host, "binary": binary})
}

// DisableOpts carries the optional reason. A supplied empty string is sent;
// Python omits a falsy reason.
type DisableOpts struct {
	DisabledReason request.Optional[string] `json:"disabled_reason,omitzero"`
}

type DisableOption = request.Option[DisableOpts]

func WithDisableOptions(value DisableOpts) DisableOption { return request.WithOptions(value) }

// WithDisableField adds a JSON member the typed options do not own.
func WithDisableField(key string, value any) DisableOption {
	return request.WithField[DisableOpts](key, value)
}

type protectedDisable struct {
	DisableOpts
	Host   string `json:"host"`
	Binary string `json:"binary"`
}

// Disable sends PUT /services/disable. A reason needs microversion 2.83 and
// is checked locally; without a reason 2.7 suffices.
func (a *API) Disable(ctx context.Context, host, binary string, opts DisableOpts, options ...DisableOption) (*ServiceState, error) {
	if err := a.validateTarget(ctx, host, binary); err != nil {
		return nil, stateWrap("Disable", err)
	}
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	if err != nil {
		return nil, stateWrap("Disable", err)
	}
	body, err := manilaown.Body(cfg.Options)
	if err == nil {
		body, err = request.MergeFieldsFor(body, cfg.Fields, protectedDisable{})
	}
	if err != nil {
		return nil, stateWrap("Disable", err)
	}
	floor := MinStateMicroversion
	if !cfg.Options.DisabledReason.IsZero() {
		floor = MinDisabledReasonMicroversion
	}
	body["host"], body["binary"] = host, binary
	return a.state(ctx, "Disable", "disable", floor, body)
}

// EnsureShares sends POST /services/ensure-shares {"host":...} (2.86 and
// later). Success is 200, 202 or 204 and means only that the request was
// accepted: the back end may still be ensuring shares. Every error response is
// reported, unlike Python, which ignores the response.
func (a *API) EnsureShares(ctx context.Context, host string) error {
	if a == nil || a.client == nil {
		return stateWrap("EnsureShares", fmt.Errorf("%w: services API is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return stateWrap("EnsureShares", err)
	}
	if strings.TrimSpace(host) == "" {
		return stateWrap("EnsureShares", fmt.Errorf("%w: host is required", resource.ErrInvalidOption))
	}
	_, err := manilaown.Call{Client: a.client, Floor: MinEnsureSharesMicroversion, Method: http.MethodPost, URL: a.client.ServiceURL("services", "ensure-shares"), Body: map[string]any{"host": host}, OK: []int{200, 202, 204}}.Do(ctx)
	return stateWrap("EnsureShares", err)
}
