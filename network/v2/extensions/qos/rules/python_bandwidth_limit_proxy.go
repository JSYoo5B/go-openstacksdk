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
	"github.com/JSYoo5B/go-openstacksdk/request"
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
//
// Deprecated: use request.Query.
func WithBandwidthLimitProxyQuery(key string, value any) BandwidthLimitListOption {
	if bandwidthLimitProxyIgnores(key) {
		return request.Custom(func(*BandwidthLimitListOpts) error { return nil })
	}
	raw, err := json.Marshal(value)
	return request.Custom(bandwidthLimitProxyQuery(key, bytes.Clone(raw), err))
}

// bandwidthLimitProxyQuery applies one marshaled value for WithBandwidthLimitProxyQuery and the shared
// request calls that carry the same key.
func bandwidthLimitProxyQuery(key string, owned json.RawMessage, err error) func(*BandwidthLimitListOpts) error {
	if bandwidthLimitProxyIgnores(key) {
		return func(*BandwidthLimitListOpts) error { return nil }
	}
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
//
// Calls: request.Header, request.Headers, request.Microversion, request.Query,
// request.BasePath, request.Limit, request.Marker, request.MaxItems and
// request.Paginated. Here request.Query and the paging calls are Proxy kwargs,
// so Query "headers", "microversion" and "base_path" replace those controls at
// their position.
func (a *API) ProxyBandwidthLimitRecords(ctx context.Context, parent string, options ...BandwidthLimitListOption) (*BandwidthLimitProxyResult, error) {
	const op = "ProxyBandwidthLimitRecords"
	engine, err := a.bandwidthLimitEnginePolicy(ctx, parent, true)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, op, err)
	}
	callbacks := bandwidthLimitProxyCalls(slices.Clone(options))
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

// bandwidthLimitProxyIgnores reports whether Proxy discards a kwarg before
// evaluating its value; jmespath_filters is a formal Proxy argument.
func bandwidthLimitProxyIgnores(key string) bool {
	return key != "jmespath_filters" && bandwidthLimitListIgnores(key)
}

// bandwidthLimitProxyCalls translates the options of one call into BandwidthLimitListOpts callbacks.
func bandwidthLimitProxyCalls(calls []request.ListCall) []func(*BandwidthLimitListOpts) error {
	return qosCalls(calls, func(s *BandwidthLimitListOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, basePath: &s.BasePath}
	}, bandwidthLimitProxyQuery, true)
}
