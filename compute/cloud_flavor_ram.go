package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ErrNoFlavorByRAM is a completed local no-match, not a fabricated HTTP404.
var ErrNoFlavorByRAM = errors.New("no flavor matches the requested RAM and name condition")

// FlavorRAMOpts owns JSON-domain Python arguments; callers need no builder or
// application interface. Nil Include means None; nil GetExtra defaults true.
// QueryOptions reuse the established flavor read transport/options policy.
type FlavorRAMOpts struct {
	Include, GetExtra *json.RawMessage
	QueryOptions      []FlavorQueryOption
}
type FlavorRAMOption func(*FlavorRAMOpts) error

func cloneFlavorRAMOpts(v FlavorRAMOpts) FlavorRAMOpts {
	if v.Include != nil {
		raw := json.RawMessage(bytes.Clone(*v.Include))
		v.Include = &raw
	}
	if v.GetExtra != nil {
		raw := json.RawMessage(bytes.Clone(*v.GetExtra))
		v.GetExtra = &raw
	}
	v.QueryOptions = slices.Clone(v.QueryOptions)
	return v
}
func WithFlavorRAMOpts(v FlavorRAMOpts) FlavorRAMOption {
	owned := cloneFlavorRAMOpts(v)
	return func(c *FlavorRAMOpts) error { *c = cloneFlavorRAMOpts(owned); return nil }
}
func flavorRAMArgument(v any, include bool) FlavorRAMOption {
	raw, err := json.Marshal(v)
	owned := bytes.Clone(raw)
	return func(c *FlavorRAMOpts) error {
		if err != nil {
			return invalid("RAM helper JSON argument: %v", err)
		}
		value := json.RawMessage(bytes.Clone(owned))
		if include {
			c.Include = &value
		} else {
			c.GetExtra = &value
		}
		return nil
	}
}

// WithFlavorRAMInclude snapshots a normal substring or another Source JSON
// primitive/container. Truthy wrong types fail only when membership is reached.
func WithFlavorRAMInclude(v any) FlavorRAMOption       { return flavorRAMArgument(v, true) }
func WithFlavorRAMGetExtra(v bool) FlavorRAMOption     { return flavorRAMArgument(v, false) }
func WithFlavorRAMGetExtraValue(v any) FlavorRAMOption { return flavorRAMArgument(v, false) }
func WithFlavorRAMQueryOptions(v ...FlavorQueryOption) FlavorRAMOption {
	owned := slices.Clone(v)
	return func(c *FlavorRAMOpts) error { c.QueryOptions = slices.Clone(owned); return nil }
}

type FlavorRAMResult struct {
	Flavor    *flavors.FlavorRecord
	Inventory []*flavors.FlavorRecord
}
type flavorRAMNumber struct {
	value    *big.Rat
	infinity int
}

func flavorRAMNumeric(raw json.RawMessage) (flavorRAMNumber, error) {
	text := string(bytes.TrimSpace(raw))
	switch text {
	case "true":
		text = "1"
	case "false":
		text = "0"
	}
	if text == "" || text == "null" || text[0] == '"' || text[0] == '[' || text[0] == '{' {
		return flavorRAMNumber{}, invalid("RAM comparison requires numbers, got %s", raw)
	}
	if strings.ContainsAny(text, ".eE") {
		v, err := strconv.ParseFloat(text, 64)
		if math.IsInf(v, 1) {
			return flavorRAMNumber{infinity: 1}, nil
		}
		if math.IsInf(v, -1) {
			return flavorRAMNumber{infinity: -1}, nil
		}
		if err != nil || math.IsNaN(v) {
			return flavorRAMNumber{}, invalid("RAM float: %v", err)
		}
		return flavorRAMNumber{value: new(big.Rat).SetFloat64(v)}, nil
	}
	value, ok := new(big.Int).SetString(text, 10)
	if !ok {
		return flavorRAMNumber{}, invalid("RAM comparison requires a JSON number")
	}
	return flavorRAMNumber{value: new(big.Rat).SetInt(value)}, nil
}
func flavorRAMCompare(left, right json.RawMessage) (int, error) {
	a, err := flavorRAMNumeric(left)
	if err != nil {
		return 0, err
	}
	b, err := flavorRAMNumeric(right)
	if err != nil {
		return 0, err
	}
	if a.infinity != 0 || b.infinity != 0 {
		if a.infinity < b.infinity {
			return -1, nil
		}
		if a.infinity > b.infinity {
			return 1, nil
		}
		return 0, nil
	}
	return a.value.Cmp(b.value), nil
}
func flavorRAMContains(name, include json.RawMessage) (bool, error) {
	name = bytes.TrimSpace(name)
	include = bytes.TrimSpace(include)
	if len(name) == 0 {
		return false, invalid("include membership requires an iterable flavor name")
	}
	switch name[0] {
	case '"':
		var haystack, needle string
		if len(include) == 0 || include[0] != '"' {
			return false, invalid("include in a string name requires a string")
		}
		if err := json.Unmarshal(name, &haystack); err != nil {
			return false, err
		}
		if err := json.Unmarshal(include, &needle); err != nil {
			return false, err
		}
		return strings.Contains(haystack, needle), nil
	case '[':
		var values []json.RawMessage
		if err := json.Unmarshal(name, &values); err != nil {
			return false, err
		}
		for _, value := range values {
			equal, err := jsonfilter.EqualPythonLoadedJSON(value, include)
			if err != nil {
				return false, err
			}
			if equal {
				return true, nil
			}
		}
		return false, nil
	case '{':
		if len(include) > 0 && (include[0] == '[' || include[0] == '{') {
			return false, invalid("include is an unhashable dictionary key")
		}
		if len(include) == 0 || include[0] != '"' {
			return false, nil
		}
		var key string
		if err := json.Unmarshal(include, &key); err != nil {
			return false, err
		}
		var values map[string]json.RawMessage
		if err := json.Unmarshal(name, &values); err != nil {
			return false, err
		}
		_, found := values[key]
		return found, nil
	default:
		return false, invalid("include membership requires an iterable flavor name")
	}
}
func cloneFlavorRAMRecord(v *flavors.FlavorRecord) *flavors.FlavorRecord {
	if v == nil {
		return nil
	}
	out := *v
	out.Resource = v.Resource.Clone()
	out.Wire = v.Wire.Clone()
	out.Envelope = bytes.Clone(v.Envelope)
	out.PageEnvelope = bytes.Clone(v.PageEnvelope)
	out.Header = v.Header.Clone()
	if v.Enrichment != nil {
		e := *v.Enrichment
		e.Resource = v.Enrichment.Resource.Clone()
		e.Wire = v.Enrichment.Wire.Clone()
		e.ExtraSpecs = bytes.Clone(v.Enrichment.ExtraSpecs)
		e.Envelope = bytes.Clone(v.Enrichment.Envelope)
		e.Header = v.Enrichment.Header.Clone()
		out.Enrichment = &e
	}
	return &out
}
func flavorRAMProof(inventory []*flavors.FlavorRecord, cause error) error {
	if len(inventory) == 0 {
		return cause
	}
	row := inventory[len(inventory)-1]
	body := row.Envelope
	if row.PageEnvelope != nil {
		body = row.PageEnvelope
	}
	return &resource.ResponseError{Body: bytes.Clone(body), Header: row.Header.Clone(), StatusCode: row.StatusCode, Cause: cause}
}

// GetFlavorByRAM materializes all detailed flavors and requested enrichment,
// then stably sorts by Source RAM and selects the first >= threshold whose
// name passes include. JSON-domain any inputs are snapshotted by the SDK;
// invalid Python-like comparison/membership is deferred until actually used.
func (s *Service) GetFlavorByRAM(ctx context.Context, ram any, options ...FlavorRAMOption) (*FlavorRAMResult, error) {
	const op = "GetFlavorByRAM"
	p, err := s.captureFlavorRecords(ctx)
	if err != nil {
		return nil, flavorCloudFailure(op, err)
	}
	requested, err := json.Marshal(ram)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, flavorCloudFailure(op, errors.Join(invalid("RAM requires a JSON-domain value"), err))
	}
	cfg := cloneFlavorRAMOpts(FlavorRAMOpts{})
	for _, apply := range slices.Clone(options) {
		if err := p.check(p.ctx); err != nil {
			return nil, flavorCloudFailure(op, err)
		}
		if apply == nil {
			return nil, flavorCloudFailure(op, invalid("nil flavor RAM option"))
		}
		next := cloneFlavorRAMOpts(cfg)
		if err := errors.Join(apply(&next), p.check(p.ctx)); err != nil {
			return nil, flavorCloudFailure(op, err)
		}
		cfg = cloneFlavorRAMOpts(next)
	}
	include := json.RawMessage("null")
	if cfg.Include != nil {
		include = bytes.Clone(*cfg.Include)
	}
	includeTruthy, err := cloudfilter.PythonJSONTruthy(include)
	if err != nil {
		return nil, flavorCloudFailure(op, invalid("include requires complete UTF-8 JSON: %v", err))
	}
	queryOptions := []FlavorQueryOption{WithFlavorQueryExtraSpecs(true)}
	queryOptions = append(queryOptions, cfg.QueryOptions...)
	if cfg.GetExtra != nil {
		extra, cause := cloudfilter.PythonJSONTruthy(*cfg.GetExtra)
		if cause != nil {
			return nil, flavorCloudFailure(op, invalid("get_extra requires JSON: %v", cause))
		}
		queryOptions = append(queryOptions, WithFlavorQueryExtraSpecs(extra))
	}
	prepared, err := p.prepareQuery(queryOptions)
	if err == nil && prepared.Options.Filters != nil {
		err = invalid("RAM helper has no local/list filters argument")
	}
	if err != nil {
		return nil, flavorCloudFailure(op, err)
	}
	listed, err := p.queryInventorySource(prepared, true, true)
	result := &FlavorRAMResult{}
	if listed != nil {
		for _, row := range listed.Inventory {
			result.Inventory = append(result.Inventory, cloneFlavorRAMRecord(row))
		}
	}
	if err != nil {
		return result, flavorCloudFailure(op, err)
	}
	ordered := slices.Clone(result.Inventory)
	var compareErr error
	slices.SortStableFunc(ordered, func(a, b *flavors.FlavorRecord) int {
		if compareErr != nil {
			return 0
		}
		if a == nil || a.Resource == nil || b == nil || b.Resource == nil {
			compareErr = invalid("flavor inventory Resource required")
			return 0
		}
		comparison, cause := flavorRAMCompare(a.Resource.Body["ram"], b.Resource.Body["ram"])
		compareErr = cause
		return comparison
	})
	if compareErr != nil {
		return result, flavorCloudFailure(op, flavorRAMProof(result.Inventory, compareErr))
	}
	for _, row := range ordered {
		if err := p.check(p.ctx); err != nil {
			return result, flavorCloudFailure(op, flavorRAMProof(result.Inventory, err))
		}
		if row == nil || row.Resource == nil {
			return result, flavorCloudFailure(op, flavorRAMProof(result.Inventory, invalid("flavor Resource required")))
		}
		comparison, cause := flavorRAMCompare(row.Resource.Body["ram"], requested)
		if cause != nil {
			return result, flavorCloudFailure(op, flavorRAMProof(result.Inventory, cause))
		}
		if comparison < 0 {
			continue
		}
		if includeTruthy {
			matches, cause := flavorRAMContains(row.Resource.Body["name"], include)
			if cause != nil {
				return result, flavorCloudFailure(op, flavorRAMProof(result.Inventory, cause))
			}
			if !matches {
				continue
			}
		}
		if err := p.check(p.ctx); err != nil {
			return result, flavorCloudFailure(op, flavorRAMProof(result.Inventory, err))
		}
		result.Flavor = cloneFlavorRAMRecord(row)
		return result, nil
	}
	err = fmt.Errorf("%w: minimum RAM %s include %s", ErrNoFlavorByRAM, requested, include)
	return result, flavorCloudFailure(op, flavorRAMProof(result.Inventory, err))
}
