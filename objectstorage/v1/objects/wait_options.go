package objects

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type ObjectWaitOpts struct {
	Headers                       map[string]string
	Interval, Timeout             *time.Duration
	StatusAttribute, StatusHeader string
	FailureStates                 []string
	ProgressCallback              func(int) error
}

// ObjectWaitOption is one option of WaitForStatus and WaitForDelete: the
// WithObjectWait* options, request.Header (validated and merged like a
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
func WithObjectWaitPollInterval(value time.Duration) ObjectWaitOption {
	return request.Custom(func(cfg *ObjectWaitOpts) error {
		if value <= 0 {
			return metadataInvalid("object polling interval must be positive")
		}
		owned := value
		cfg.Interval = &owned
		return nil
	})
}
func WithObjectWaitTimeout(value time.Duration) ObjectWaitOption {
	return request.Custom(func(cfg *ObjectWaitOpts) error {
		if value <= 0 {
			return metadataInvalid("object wait timeout must be positive")
		}
		owned := value
		cfg.Timeout = &owned
		return nil
	})
}

// WithObjectWaitUnlimitedWait removes the SDK timeout. The caller's context
// still controls cancellation and deadlines.
func WithObjectWaitUnlimitedWait() ObjectWaitOption {
	return request.Custom(func(cfg *ObjectWaitOpts) error { owned := time.Duration(0); cfg.Timeout = &owned; return nil })
}
func WithObjectWaitStatusAttribute(value string) ObjectWaitOption {
	return request.Custom(func(cfg *ObjectWaitOpts) error { cfg.StatusAttribute, cfg.StatusHeader = value, ""; return nil })
}
func WithObjectWaitStatusHeader(value string) ObjectWaitOption {
	return request.Custom(func(cfg *ObjectWaitOpts) error { cfg.StatusHeader, cfg.StatusAttribute = value, ""; return nil })
}

// WithObjectWaitFailureStates replaces failure matching. Calling it without
// states installs an explicit empty list; nil full options restore ERROR.
func WithObjectWaitFailureStates(values ...string) ObjectWaitOption {
	snapshot := make([]string, len(values))
	copy(snapshot, values)
	return request.Custom(func(cfg *ObjectWaitOpts) error {
		cfg.FailureStates = make([]string, len(snapshot))
		copy(cfg.FailureStates, snapshot)
		return nil
	})
}
func WithObjectWaitProgressCallback(callback func(int) error) ObjectWaitOption {
	return request.Custom(func(cfg *ObjectWaitOpts) error { cfg.ProgressCallback = callback; return nil })
}
func WithoutObjectWaitProgressCallback() ObjectWaitOption {
	return request.Custom(func(cfg *ObjectWaitOpts) error { cfg.ProgressCallback = nil; return nil })
}
func cloneObjectWaitOpts(value ObjectWaitOpts) ObjectWaitOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	if value.Interval != nil {
		owned := *value.Interval
		value.Interval = &owned
	}
	if value.Timeout != nil {
		owned := *value.Timeout
		value.Timeout = &owned
	}
	if value.FailureStates != nil {
		owned := make([]string, len(value.FailureStates))
		copy(owned, value.FailureStates)
		value.FailureStates = owned
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
