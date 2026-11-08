// Package sourcewait resolves shared resource wait options for waits whose
// timeout or pause comes from a Source (Python openstacksdk) value that the
// shared options cannot express, such as a zero or negative timeout.
package sourcewait

import (
	"math"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// placeholder stands for "not set by the caller"; no real wait uses it.
const placeholder = time.Duration(math.MaxInt64)

var placeholderStrategy = resource.FixedWait(placeholder)

// Result is the resolved settings plus whether the caller's options chose a
// timeout (WithTimeout or WithUnlimitedWait) or a pause schedule
// (WithPollInterval or WithWaitStrategy). When a control was not chosen its
// settings field is zero and the caller applies its Source default.
type Result struct {
	resource.WaitSettings
	TimeoutSet, StrategySet bool
}

// Resolve applies opts with resource.ResolveWait and reports which of the
// timeout and pause controls they set.
func Resolve(supports resource.WaitSupport, opts []resource.WaitOption) (Result, error) {
	settings, err := resource.ResolveWait(resource.WaitDefaults{Timeout: placeholder, Strategy: placeholderStrategy, Supports: supports}, opts...)
	if err != nil {
		return Result{}, err
	}
	result := Result{WaitSettings: settings, TimeoutSet: settings.Unlimited || settings.Timeout != placeholder, StrategySet: settings.Strategy != placeholderStrategy}
	if !result.TimeoutSet {
		result.Timeout = 0
	}
	if !result.StrategySet {
		result.Strategy = resource.WaitStrategy{}
	}
	return result, nil
}
