package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonUserAction struct {
	ctx      context.Context
	source   *cloudread.Source
	guard    func(context.Context) error
	serverID string
}

func userActionInvalid(message string) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
}
func userActionWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "servers", cloudread.ContextError(ctx, err))
}
func (a *API) prepareUserAction(ctx context.Context, id string) (*pythonUserAction, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" || id == "." || id == ".." {
		return nil, userActionInvalid("server ID must be nonblank literal text")
	}
	if !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return nil, userActionInvalid("server ID must be valid text without controls")
	}
	if a == nil {
		return nil, userActionInvalid("server API is required")
	}
	original := a.client
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return nil, err
	}
	var observed error
	guard := func(ctx context.Context) error {
		if a.client != original && observed == nil {
			observed = userActionInvalid("server API source changed")
		}
		return errors.Join(observed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &pythonUserAction{ctx: ctx, source: source, guard: guard, serverID: id}, guard(ctx)
}
func ownUserActionConfig[T any](c *request.Config[T], clone func(T) T) {
	c.Options = clone(c.Options)
	fields := make(map[string]json.RawMessage, len(c.Fields))
	for key, value := range c.Fields {
		fields[key] = bytes.Clone(value)
	}
	c.Fields = fields
	c.Headers = maps.Clone(c.Headers)
	if c.Headers == nil {
		c.Headers = make(map[string]string)
	}
	query := make(url.Values, len(c.Query))
	for key, value := range c.Query {
		query[key] = slices.Clone(value)
	}
	c.Query = query
	c.Arguments = maps.Clone(c.Arguments)
	if c.Arguments == nil {
		c.Arguments = make(map[string]any)
	}
}
func applyUserActionOptions[T any](p *pythonUserAction, initial T, clone func(T) T, body bool, options []request.Option[T]) (request.Config[T], error) {
	owned := slices.Clone(options)
	guarded := make([]request.Option[T], len(owned))
	for i, option := range owned {
		apply := option
		guarded[i] = func(c *request.Config[T]) error {
			if err := p.guard(p.ctx); err != nil {
				return err
			}
			if apply == nil {
				return userActionInvalid("nil server action option")
			}
			ownUserActionConfig(c, clone)
			err := apply(c)
			ownUserActionConfig(c, clone)
			return errors.Join(err, p.guard(p.ctx))
		}
	}
	c, err := request.Apply(initial, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(c, body, false, true)
	}
	if err == nil {
		for key, value := range c.Fields {
			if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 || !utf8.Valid(value) || !json.Valid(value) {
				err = userActionInvalid("server action extension requires valid UTF-8 JSON and a nonblank text key")
				break
			}
		}
	}
	if err == nil {
		err = p.source.WithPolicy(p.ctx, nil, c.Headers)
	}
	return c, errors.Join(err, p.guard(p.ctx))
}
func (p *pythonUserAction) version(minimum, maximum string) error {
	selected := p.source.Client.Microversion
	if selected == "" {
		selected = "2.1"
	}
	if selected != "latest" {
		parts := strings.Split(selected, ".")
		if len(parts) != 2 || parts[0] != "2" || parts[1] == "" || len(parts[1]) > 1 && parts[1][0] == '0' {
			return userActionInvalid("Nova microversion must be 2 followed by a decimal component")
		}
		for _, digit := range parts[1] {
			if digit < '0' || digit > '9' {
				return userActionInvalid("invalid Nova microversion")
			}
		}
	}
	candidate, err := microversions.ParseVersion(selected)
	if err != nil {
		return userActionInvalid("invalid Nova microversion")
	}
	low, _ := microversions.ParseVersion(minimum)
	unsupported := candidate.Compare(low) < 0
	if maximum != "" {
		high, _ := microversions.ParseVersion(maximum)
		unsupported = unsupported || candidate.Compare(high) > 0
	}
	if unsupported {
		return fmt.Errorf("%w: server action requires Nova %s..%s (selected %s)", resource.ErrUnsupported, minimum, maximum, selected)
	}
	return nil
}
func (p *pythonUserAction) send(method, suffix string, body any) error {
	return p.sendWithFaultPolicy(method, suffix, body, false)
}

// New owned actions can stop native retry/reauth callbacks after physical faults
// while existing action bindings retain their reviewed rejection policy.
func (p *pythonUserAction) sendWithFaultPolicy(method, suffix string, body any, stopFaultCallbacks bool) error {
	accepted := make([]int, 200)
	rejected := make([]int, 200)
	for i := range accepted {
		accepted[i] = 200 + i
		rejected[i] = 400 + i
	}
	var headers map[string]string
	if method == http.MethodPost {
		// Server._action owns the empty Accept header. Remove application
		// aliases on the captured client before fixing it across retries.
		p.source.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
		for key := range p.source.Client.MoreHeaders {
			if strings.EqualFold(key, "Accept") {
				delete(p.source.Client.MoreHeaders, key)
			}
		}
		headers = map[string]string{"Accept": ""}
	}
	_, err := rest.DoJSONGuardedRejectionsHeaders(p.ctx, &p.source.Client, p.guard, method, p.source.Client.ServiceURL("servers", url.PathEscape(p.serverID), suffix), body, headers, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: stopFaultCallbacks}, accepted...)
	return err
}
func userActionBody[T any](name string, value map[string]any, c request.Config[T]) (map[string]any, error) {
	for _, item := range value {
		if text, ok := item.(string); ok && !utf8.ValidString(text) {
			return nil, userActionInvalid("server action input must be valid UTF-8")
		}
	}
	if len(c.Fields) != 0 && value == nil {
		value = make(map[string]any)
	}
	// A nil map stored in an interface still serializes as the Source's null.
	return request.MergeFieldsFor(map[string]any{name: value}, c.Fields, c.Options)
}
func (a *API) simpleUserAction(ctx context.Context, op, id, name string, fields map[string]any, legacy bool, options []UserActionOption) error {
	p, err := a.prepareUserAction(ctx, id)
	if err != nil {
		return userActionWrap(ctx, op, err)
	}
	c, err := applyUserActionOptions(p, UserActionOpts{}, func(v UserActionOpts) UserActionOpts { return v }, name != "", options)
	if err == nil {
		err = p.version("2.1", "")
	}
	if err == nil && legacy {
		err = p.version("2.1", "2.43")
	}
	var body any
	method, suffix := http.MethodDelete, "os-server-password"
	if err == nil && name != "" {
		method, suffix = http.MethodPost, "action"
		body, err = userActionBody(name, fields, c)
	}
	if err == nil {
		err = p.send(method, suffix, body)
	}
	return userActionWrap(ctx, op, err)
}

// ClearPassword deletes Nova's stored encrypted password without changing the
// guest administrator password. It sends one DELETE and does not ignore 404.
func (a *API) ClearPassword(ctx context.Context, serverID string, options ...UserActionOption) error {
	return a.simpleUserAction(ctx, "ClearPassword", serverID, "", nil, false, options)
}

// Restore submits restore for a soft-deleted server without waiting for state.
func (a *API) Restore(ctx context.Context, serverID string, options ...UserActionOption) error {
	return a.simpleUserAction(ctx, "Restore", serverID, "restore", nil, false, options)
}

// LockWithOptions preserves the Python default lock:null and optional reason,
// including an empty string. A supplied reason requires selected Nova 2.73+.
func (a *API) LockWithOptions(ctx context.Context, serverID string, options ...LockServerOption) error {
	p, err := a.prepareUserAction(ctx, serverID)
	if err != nil {
		return userActionWrap(ctx, "LockWithOptions", err)
	}
	c, err := applyUserActionOptions(p, LockServerOpts{}, func(v LockServerOpts) LockServerOpts {
		if v.LockedReason != nil {
			copy := *v.LockedReason
			v.LockedReason = &copy
		}
		return v
	}, true, options)
	var fields map[string]any
	if err == nil {
		err = p.version("2.1", "")
	}
	if err == nil && c.Options.LockedReason != nil {
		err = p.version("2.73", "")
		fields = map[string]any{"locked_reason": *c.Options.LockedReason}
	}
	var body map[string]any
	if err == nil {
		body, err = userActionBody("lock", fields, c)
	}
	if err == nil {
		err = p.send(http.MethodPost, "action", body)
	}
	return userActionWrap(ctx, "LockWithOptions", err)
}

// UnshelveWithOptions handles host-only and explicit null-zone bodies, unlike
// the generated native Unshelve method. It never changes the selected version.
func (a *API) UnshelveWithOptions(ctx context.Context, serverID string, options ...UnshelveServerOption) error {
	p, err := a.prepareUserAction(ctx, serverID)
	if err != nil {
		return userActionWrap(ctx, "UnshelveWithOptions", err)
	}
	c, err := applyUserActionOptions(p, UnshelveServerOpts{}, func(v UnshelveServerOpts) UnshelveServerOpts {
		if v.AvailabilityZone != nil {
			copy := *v.AvailabilityZone
			v.AvailabilityZone = &copy
		}
		return v
	}, true, options)
	fields := make(map[string]any)
	if err == nil {
		err = p.version("2.1", "")
	}
	if err == nil && c.Options.Host != "" {
		err = p.version("2.91", "")
		fields["host"] = c.Options.Host
	}
	if err == nil && c.Options.AvailabilityZone != nil && !c.Options.ZoneSet {
		err = userActionInvalid("availability zone value requires ZoneSet")
	}
	if err == nil && c.Options.ZoneSet {
		if c.Options.AvailabilityZone == nil {
			err = p.version("2.91", "")
			fields["availability_zone"] = nil
		} else {
			err = p.version("2.77", "")
			fields["availability_zone"] = *c.Options.AvailabilityZone
		}
	}
	if len(fields) == 0 {
		fields = nil
	}
	var body map[string]any
	if err == nil {
		body, err = userActionBody("unshelve", fields, c)
	}
	if err == nil {
		err = p.send(http.MethodPost, "action", body)
	}
	return userActionWrap(ctx, "UnshelveWithOptions", err)
}

// AddFixedIP sends the direct legacy Nova addFixedIp action with a literal
// network ID. It does not resolve the network or call Neutron; Nova <=2.43 only.
func (a *API) AddFixedIP(ctx context.Context, serverID, networkID string, options ...UserActionOption) error {
	return a.simpleUserAction(ctx, "AddFixedIP", serverID, "addFixedIp", map[string]any{"networkId": networkID}, true, options)
}
func (a *API) RemoveFixedIP(ctx context.Context, serverID, address string, options ...UserActionOption) error {
	return a.simpleUserAction(ctx, "RemoveFixedIP", serverID, "removeFixedIp", map[string]any{"address": address}, true, options)
}

// AddFloatingIP only submits the direct action. It does not list, allocate,
// re-read or wait for a floating IP, and supports selected Nova <=2.43.
func (a *API) AddFloatingIP(ctx context.Context, serverID, address string, options ...FloatingIPServerOption) error {
	p, err := a.prepareUserAction(ctx, serverID)
	if err != nil {
		return userActionWrap(ctx, "AddFloatingIP", err)
	}
	c, err := applyUserActionOptions(p, FloatingIPServerOpts{}, func(v FloatingIPServerOpts) FloatingIPServerOpts {
		if v.FixedAddress != nil {
			copy := *v.FixedAddress
			v.FixedAddress = &copy
		}
		return v
	}, true, options)
	if err == nil {
		err = p.version("2.1", "2.43")
	}
	fields := map[string]any{"address": address}
	if c.Options.FixedAddress != nil {
		fields["fixed_address"] = *c.Options.FixedAddress
	}
	var body map[string]any
	if err == nil {
		body, err = userActionBody("addFloatingIp", fields, c)
	}
	if err == nil {
		err = p.send(http.MethodPost, "action", body)
	}
	return userActionWrap(ctx, "AddFloatingIP", err)
}
func (a *API) RemoveFloatingIP(ctx context.Context, serverID, address string, options ...UserActionOption) error {
	return a.simpleUserAction(ctx, "RemoveFloatingIP", serverID, "removeFloatingIp", map[string]any{"address": address}, true, options)
}
