package objects

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ObjectWaitOpts is the full object wait policy. Wait holds the shared wait
// options (see WithObjectWait); StatusHeader selects a response header as the
// status instead of a resource.WithStatusAttribute attribute, and the later of
// the two wins. A struct that sets StatusHeader while its own Wait selects a
// status attribute is rejected before HTTP.
type ObjectWaitOpts struct {
	Headers      map[string]string
	Wait         []resource.WaitOption
	StatusHeader string
	// statusHeaderAt is len(Wait) when WithObjectWaitStatusHeader applied, so
	// attributes selected earlier lose to the header.
	statusHeaderAt int
}

// ObjectWaitOption is one option of WaitForStatus and WaitForDelete:
// WithObjectWaitOpts, WithObjectWaitHeaders, WithObjectWait,
// WithObjectWaitStatusHeader, request.Header (validated and merged like a
// one-entry WithObjectWaitHeaders) or request.Custom over *ObjectWaitOpts.
// Other shared calls, including request.Headers (the WithObjectWaitHeaders
// map merges instead of replacing), fail with resource.ErrInvalidOption
// before HTTP.
type ObjectWaitOption = request.GetCall

var objectWaitCalls = calls.Spec[ObjectWaitOpts]{Header: calls.MergeHeader[ObjectWaitOpts](WithObjectWaitHeaders)}

func WithObjectWaitOpts(value ObjectWaitOpts) ObjectWaitOption {
	snapshot := cloneObjectWaitOpts(value)
	return request.Custom(func(cfg *ObjectWaitOpts) error { *cfg = cloneObjectWaitOpts(snapshot); return nil })
}
func WithObjectWaitHeaders(values map[string]string) ObjectWaitOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *ObjectWaitOpts) error {
		if _, err := validateObjectWaitHeaders(snapshot); err != nil {
			return err
		}
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	})
}

// WithObjectWait appends shared wait options; repeated calls append and the
// later option wins per control. WaitForStatus defaults to no SDK timeout,
// a fixed two-second interval and ERROR failure detection on the status
// attribute; WaitForDelete defaults to a 120-second timeout and the same
// interval. Both accept resource.WithTimeout, WithUnlimitedWait,
// WithPollInterval, WithWaitStrategy, WithFailureStates, WithStatusAttribute
// (a Swift object attribute; a single field name), WithProgressCallback and
// WithProgress (both report zero after each nonterminal observation; the
// callback runs first and its error stops the wait with nothing sent for
// that step). WaitForDelete validates but ignores failure states and the
// status attribute. A selected attribute replaces an earlier
// WithObjectWaitStatusHeader.
func WithObjectWait(opts ...resource.WaitOption) ObjectWaitOption {
	snapshot := append([]resource.WaitOption(nil), opts...)
	selects := objectWaitSelectsAttribute(snapshot)
	return request.Custom(func(cfg *ObjectWaitOpts) error {
		cfg.Wait = append(append([]resource.WaitOption(nil), cfg.Wait...), snapshot...)
		if selects {
			cfg.StatusHeader = ""
		}
		return nil
	})
}

// WithObjectWaitStatusHeader selects a response header as the status. It
// replaces a status attribute selected earlier through WithObjectWait.
func WithObjectWaitStatusHeader(value string) ObjectWaitOption {
	return request.Custom(func(cfg *ObjectWaitOpts) error {
		cfg.StatusHeader, cfg.statusHeaderAt = value, len(cfg.Wait)
		return nil
	})
}

// objectWaitSelectsAttribute reports whether opts select a status attribute.
// Invalid options report false here and fail when the wait resolves them.
func objectWaitSelectsAttribute(opts []resource.WaitOption) bool {
	probe, err := resource.ResolveWait(resource.WaitDefaults{Timeout: time.Second, Strategy: resource.FixedWait(time.Second), Supports: objectWaitSupport}, opts...)
	return err == nil && probe.StatusAttribute != ""
}

const objectWaitSupport = resource.WaitSupportFailureStates | resource.WaitSupportStatusAttribute | resource.WaitSupportProgress

func cloneObjectWaitOpts(value ObjectWaitOpts) ObjectWaitOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	if value.Wait != nil {
		value.Wait = append([]resource.WaitOption(nil), value.Wait...)
	}
	return value
}
func (p *preparedCreateObject) applyObjectWaitOptions(ctx context.Context, options []ObjectWaitOption) (ObjectWaitOpts, error) {
	cfg := ObjectWaitOpts{}
	for _, option := range calls.Translate(options, objectWaitCalls) {
		if err := p.guard(ctx); err != nil {
			return cfg, err
		}
		if option == nil {
			return cfg, metadataInvalid("nil object wait option")
		}
		callback := cloneObjectWaitOpts(cfg)
		err := option(&callback)
		cfg = cloneObjectWaitOpts(callback)
		if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
