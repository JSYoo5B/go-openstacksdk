package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type BandwidthLimitGenerator struct {
	Records iter.Seq2[*BandwidthLimitRecord, error]
}

func (g *BandwidthLimitGenerator) JMESPathIdentity() any { return g }
func (g *BandwidthLimitGenerator) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("%w: opaque bandwidth generator", resource.ErrUnsupported)
}

type BandwidthLimitProxyResult struct {
	Value any
	JSON  json.RawMessage
}

// WithBandwidthLimitProxyQuery consumes Source formal values at their option
// position, so later concrete headers/version/base-path choices remain last-win.
func WithBandwidthLimitProxyQuery(key string, value any) BandwidthLimitListOption {
	if key != "jmespath_filters" && bandwidthLimitListIgnores(key) {
		return func(*BandwidthLimitListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *BandwidthLimitListOpts) error {
		if err != nil {
			return err
		}
		switch key {
		case "headers", "microversion", "base_path":
			normalized, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{key: json.RawMessage(bytes.Clone(owned))}, Order: []string{key}})
			if err != nil {
				return err
			}
			switch key {
			case "headers":
				c.Headers = normalized.Headers
			case "microversion":
				c.Microversion = normalized.Microversion
			case "base_path":
				c.BasePath = normalized.BasePath
			}
			return nil
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}

// ProxyBandwidthLimitRecords preserves Source Proxy's formal kwargs and opaque
// generator JMESPath evaluation. Existing typed list keeps its legacy policy.
func (a *API) ProxyBandwidthLimitRecords(ctx context.Context, parent string, options ...BandwidthLimitListOption) (*BandwidthLimitProxyResult, error) {
	const op = "ProxyBandwidthLimitRecords"
	engine, err := a.bandwidthLimitEnginePolicy(ctx, parent, true)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, op, err)
	}
	callbacks := make([]func(*BandwidthLimitListOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	cfg, err := applyBandwidthLimitOptions(ctx, engine, BandwidthLimitListOpts{}, copyBandwidthLimitList, callbacks)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, op, err)
	}
	owned, err := bandwidthLimitParentQuery(bandwidthLimitListOptions(cfg), parent, false)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, op, err)
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(owned)
	if err = errors.Join(err, engine.Check(ctx)); err != nil {
		return nil, bandwidthLimitWrap(ctx, op, err)
	}
	generator := &BandwidthLimitGenerator{Records: func(yield func(*BandwidthLimitRecord, error) bool) {
		for value, err := range engine.List(ctx, normalized) {
			if !yield(bandwidthLimitRecord(value, parent), bandwidthLimitWrap(ctx, op, err)) {
				return
			}
		}
	}}
	var value any = generator
	if expression != nil {
		truthy, err := cloudfilter.PythonJSONTruthy(expression)
		if err != nil {
			return nil, bandwidthLimitWrap(ctx, op, err)
		}
		token := bytes.TrimSpace(expression)
		if truthy && len(token) > 0 && token[0] == '"' {
			var text string
			if err := json.Unmarshal(token, &text); err != nil {
				return nil, bandwidthLimitWrap(ctx, op, err)
			}
			value, err = jmespath.Search(text, generator)
			if err != nil {
				return nil, bandwidthLimitWrap(ctx, op, err)
			}
		}
	}
	result := &BandwidthLimitProxyResult{Value: value}
	result.JSON, _ = json.Marshal(value)
	if err := engine.Check(ctx); err != nil {
		return nil, bandwidthLimitWrap(ctx, op, err)
	}
	return result, nil
}
