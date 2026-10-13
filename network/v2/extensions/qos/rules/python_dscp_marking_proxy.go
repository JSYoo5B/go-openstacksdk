package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type DSCPMarkingGenerator struct {
	Records iter.Seq2[*DSCPMarkingRecord, error]
}

func (g *DSCPMarkingGenerator) JMESPathIdentity() any { return g }
func (g *DSCPMarkingGenerator) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("%w: opaque QoS rule generator", resource.ErrUnsupported)
}

type DSCPMarkingProxyResult struct {
	Value any
	JSON  json.RawMessage
}

// WithDSCPMarkingProxyQuery consumes Source formal values at their option
// position, so later concrete headers/version/base-path choices remain last-win.
func WithDSCPMarkingProxyQuery(key string, value any) DSCPMarkingListOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *DSCPMarkingListOpts) error {
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

// ProxyDSCPMarkingRecords preserves Source Proxy's formal kwargs and opaque
// generator JMESPath evaluation. Existing typed list keeps its legacy policy.
func (a *API) ProxyDSCPMarkingRecords(ctx context.Context, parent string, options ...DSCPMarkingListOption) (*DSCPMarkingProxyResult, error) {
	const op = "ProxyDSCPMarkingRecords"
	engine, err := a.dscpMarkingEnginePolicy(ctx, parent, true)
	if err != nil {
		return nil, dscpMarkingWrap(ctx, op, err)
	}
	cfg, err := applyDSCPMarkingOptions(ctx, engine, DSCPMarkingListOpts{}, copyDSCPMarkingList, slices.Clone(options))
	if err != nil {
		return nil, dscpMarkingWrap(ctx, op, err)
	}
	owned, err := dscpMarkingBound(dscpMarkingListOptions(cfg), parent, false)
	if err != nil {
		return nil, dscpMarkingWrap(ctx, op, err)
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(owned)
	if err = errors.Join(err, engine.Check(ctx)); err != nil {
		return nil, dscpMarkingWrap(ctx, op, err)
	}
	if raw, present := normalized.Query["qos_policy_id"]; present {
		var selected string
		if err := json.Unmarshal(raw, &selected); err != nil {
			return nil, dscpMarkingWrap(ctx, op, dscpMarkingInvalid("restored qos_policy_id must be a Go string"))
		}
		parent = selected
		delete(normalized.Query, "qos_policy_id")
		normalized.Order = slices.DeleteFunc(normalized.Order, func(k string) bool { return k == "qos_policy_id" })
		engine, err = a.dscpMarkingEnginePolicy(ctx, parent, true)
		if err != nil {
			return nil, dscpMarkingWrap(ctx, op, err)
		}
	}
	if normalized.BasePath != nil {
		path := *normalized.BasePath
		// URI formatting uses the selected parent; the static route remains
		// service-relative and paging stays confined to that collection.
		path, err = dscpMarkingProxyPath(path, parent, normalized.Query)
		if err != nil {
			return nil, dscpMarkingWrap(ctx, op, err)
		}
		normalized.BasePath = &path
	}
	if raw, present := normalized.Query["max_items"]; present {
		truthy, e := cloudfilter.PythonJSONTruthy(raw)
		if e != nil {
			return nil, dscpMarkingWrap(ctx, op, e)
		}
		if !truthy {
			delete(normalized.Query, "max_items")
			normalized.Order = slices.DeleteFunc(normalized.Order, func(k string) bool { return k == "max_items" })
		} else if bytes.Equal(bytes.TrimSpace(raw), []byte("true")) {
			normalized.Query["max_items"] = json.RawMessage("1")
		}
	}
	generator := &DSCPMarkingGenerator{Records: func(yield func(*DSCPMarkingRecord, error) bool) {
		if normalized.BasePath != nil && *normalized.BasePath == "" {
			return
		}
		for value, err := range engine.List(ctx, normalized) {
			if !yield(dscpMarkingRecord(parent, value), dscpMarkingWrap(ctx, op, err)) {
				return
			}
		}
	}}
	var value any = generator
	if expression != nil {
		truthy, err := cloudfilter.PythonJSONTruthy(expression)
		if err != nil {
			return nil, dscpMarkingWrap(ctx, op, err)
		}
		token := bytes.TrimSpace(expression)
		if truthy && len(token) > 0 && token[0] == '"' {
			var text string
			if err := json.Unmarshal(token, &text); err != nil {
				return nil, dscpMarkingWrap(ctx, op, err)
			}
			value, err = jmespath.Search(text, generator)
			if err != nil {
				return nil, dscpMarkingWrap(ctx, op, err)
			}
		}
	}
	result := &DSCPMarkingProxyResult{Value: value}
	result.JSON, _ = json.Marshal(value)
	if err := engine.Check(ctx); err != nil {
		return nil, dscpMarkingWrap(ctx, op, err)
	}
	return result, nil
}

// GetSourceDSCPMarkingRecord opts the named cloud getter into the pinned Source
// status and projection policy while preserving the existing typed API lane.
func (a *API) GetSourceDSCPMarkingRecord(ctx context.Context, parent, id string) (*DSCPMarkingRecord, error) {
	const op = "GetSourceDSCPMarkingRecord"
	engine, err := a.dscpMarkingEnginePolicy(ctx, parent, true)
	if err != nil {
		return nil, dscpMarkingWrap(ctx, op, err)
	}
	value, err := engine.Get(ctx, id, neutronfind.Options{})
	return dscpMarkingRecord(parent, value), dscpMarkingWrap(ctx, op, err)
}

// Source %-mapping string routes retain JSON-domain argument binding; the
// existing Engine enforces service-relative paths and paging confinement.
func dscpMarkingProxyPath(path, parent string, attrs map[string]json.RawMessage) (string, error) {
	var out strings.Builder
	for i := 0; i < len(path); {
		if path[i] != '%' {
			out.WriteByte(path[i])
			i++
			continue
		}
		if i+1 < len(path) && path[i+1] == '%' {
			out.WriteString("%25")
			i += 2
			continue
		}
		if i+1 >= len(path) || path[i+1] != '(' {
			return "", fmt.Errorf("%w: Source route requires %%(key)s formatting", resource.ErrUnsupported)
		}
		end := strings.IndexByte(path[i+2:], ')')
		if end < 0 {
			return "", fmt.Errorf("%w: malformed Source route", resource.ErrInvalidOption)
		}
		end += i + 2
		if end+1 >= len(path) || path[end+1] != 's' {
			return "", fmt.Errorf("%w: Source route supports %%(key)s string formatting", resource.ErrUnsupported)
		}
		key := path[i+2 : end]
		text := parent
		if key != "qos_policy_id" {
			raw, ok := attrs[key]
			if !ok {
				return "", fmt.Errorf("%w: missing route parameter %q", resource.ErrInvalidOption, key)
			}
			var err error
			text, err = cloudfilter.PythonString(raw)
			if err != nil {
				return "", err
			}
		}
		out.WriteString(url.PathEscape(text))
		i = end + 2
	}
	return out.String(), nil
}
