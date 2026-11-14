// Package sourcewait resolves shared resource wait options for waits whose
// timeout or pause comes from a Source (Python openstacksdk) value that the
// shared options cannot express, such as a zero or negative timeout.
package sourcewait

import (
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Result is the resolved settings. When the caller's options did not choose a
// timeout (TimeoutSet) or a pause schedule (StrategySet), that settings field
// is zero and the caller applies its Source default.
type Result struct {
	resource.WaitSettings
}

// Resolve applies opts with resource.ResolveWait.
func Resolve(supports resource.WaitSupport, opts []resource.WaitOption) (Result, error) {
	settings, err := resource.ResolveWait(resource.WaitDefaults{Timeout: time.Second, Strategy: resource.FixedWait(time.Second), Supports: supports}, opts...)
	if err != nil {
		return Result{}, err
	}
	if !settings.TimeoutSet {
		settings.Timeout = 0
	}
	if !settings.StrategySet {
		settings.Strategy = resource.WaitStrategy{}
	}
	return Result{WaitSettings: settings}, nil
}
