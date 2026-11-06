package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// SubnetPortTarget is a concrete Source subnet selector. IDs are literal;
// selecting an owned/native subnet reads its ID without fetching the subnet.
type SubnetPortTarget struct {
	value json.RawMessage
	err   error
}

func SubnetPortID(id string) SubnetPortTarget {
	raw, err := json.Marshal(id)
	return SubnetPortTarget{value: raw, err: err}
}

// SubnetPortValue preserves Resource._get_id's non-Resource JSON-domain input.
// Explicit JSON null and arbitrary JSON identifiers are not treated as names.
func SubnetPortValue(value json.RawMessage) SubnetPortTarget {
	return SubnetPortTarget{value: bytes.Clone(value)}
}

func SubnetPortRecord(value *subnets.SubnetRecord) SubnetPortTarget {
	if value == nil || value.Resource == nil {
		return SubnetPortTarget{err: findInvalid("owned subnet record required")}
	}
	raw := bytes.Clone(value.Resource.Body["id"])
	if raw == nil {
		raw = json.RawMessage("null")
	}
	return SubnetPortTarget{value: raw}
}

func SubnetPortNative(value *subnets.Subnet) SubnetPortTarget {
	if value == nil {
		return SubnetPortTarget{err: findInvalid("native subnet required")}
	}
	return SubnetPortID(value.ID)
}

// SubnetPortFields selects an owned Resource-like ID, with absent ID equal to
// Source's None. The fields are evidence-free input; no response is invented.
func SubnetPortFields(fields map[string]json.RawMessage) SubnetPortTarget {
	raw := bytes.Clone(fields["id"])
	if raw == nil {
		raw = json.RawMessage("null")
	}
	return SubnetPortTarget{value: raw}
}

type SubnetPortOpts struct {
	Headers      map[string]string
	Microversion *string
}

// SubnetPortOption is one option of GetSubnetPorts: WithSubnetPortOpts,
// request.Headers (replaces the headers), request.Microversion or
// request.Custom over *SubnetPortOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type SubnetPortOption = request.GetCall

func copySubnetPortOpts(value SubnetPortOpts) SubnetPortOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	if value.Microversion != nil {
		version := *value.Microversion
		value.Microversion = &version
	}
	return value
}
func WithSubnetPortOpts(value SubnetPortOpts) SubnetPortOption {
	owned := copySubnetPortOpts(value)
	return request.Custom(func(c *SubnetPortOpts) error { *c = copySubnetPortOpts(owned); return nil })
}
func withSubnetPortHeaders(value map[string]string) func(*SubnetPortOpts) error {
	owned := maps.Clone(value)
	return func(c *SubnetPortOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func withSubnetPortMicroversion(value string) func(*SubnetPortOpts) error {
	return func(c *SubnetPortOpts) error { version := value; c.Microversion = &version; return nil }
}

// SubnetPortsResult retains every observed port and the selected pointers.
// Multiple fixed IP matches append the same pointer, matching Source's loop.
type SubnetPortsResult struct {
	Ports     []*PortRecord
	Inventory []*PortRecord
}

// GetSubnetPorts performs the Source unfiltered ports() scan. It never looks
// up a subnet, pushes a subnet filter, deduplicates ports or rewrites receipts.
// Local errors retain the actual row/page and all previously observed records.
func (a *API) GetSubnetPorts(ctx context.Context, target SubnetPortTarget, options ...SubnetPortOption) (*SubnetPortsResult, error) {
	const op = "GetSubnetPorts"
	wrap := func(err error) error { return request.Wrap(op, "ports", cloudread.ContextError(ctx, err)) }
	engine, err := a.portListEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	id := bytes.Clone(target.value)
	if target.err != nil {
		return nil, wrap(target.err)
	}
	if !utf8.Valid(id) || !json.Valid(id) {
		return nil, wrap(findInvalid("subnet identifier must contain complete UTF-8 JSON"))
	}
	policy := copySubnetPortOpts(SubnetPortOpts{})
	for _, apply := range calls.Translate(options, subnetPortCalls) {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil subnet port option"))
		}
		next := copySubnetPortOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copySubnetPortOpts(next)
	}
	list := recordListOptions(RecordListOpts{Headers: policy.Headers, Microversion: policy.Microversion})
	result := &SubnetPortsResult{Ports: make([]*PortRecord, 0), Inventory: make([]*PortRecord, 0)}
	for value, err := range engine.List(ctx, list) {
		if err != nil {
			return result, wrap(err)
		}
		record := portListRecord(value)
		result.Inventory = append(result.Inventory, record)
		raw := record.Resource.Body["fixed_ips"]
		var fixed []json.RawMessage
		if err := json.Unmarshal(raw, &fixed); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			cause := errors.Join(findInvalid("fixed_ips must be an iterable list"), err)
			return result, wrap(portSubnetProof(record, cause))
		}
		for _, row := range fixed {
			if err := engine.Check(ctx); err != nil {
				return result, wrap(portSubnetProof(record, err))
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(row, &fields); err != nil || fields == nil {
				return result, wrap(portSubnetProof(record, errors.Join(findInvalid("fixed_ip must contain subnet_id"), err)))
			}
			subnetID, present := fields["subnet_id"]
			if !present {
				return result, wrap(portSubnetProof(record, findInvalid("fixed_ip missing subnet_id")))
			}
			matches, err := jsonfilter.EqualPythonLoadedJSON(subnetID, id)
			if err != nil {
				return result, wrap(portSubnetProof(record, err))
			}
			if matches {
				result.Ports = append(result.Ports, record)
			}
		}
	}
	return result, wrap(engine.Check(ctx))
}

func portSubnetProof(record *PortRecord, cause error) error {
	if record == nil {
		return cause
	}
	return (&rest.Response{Body: bytes.Clone(record.Envelope), Header: record.Header.Clone(), StatusCode: record.StatusCode}).Fail(cause)
}

// subnetPortCalls interprets the shared calls of SubnetPortOption.
var subnetPortCalls = calls.Spec[SubnetPortOpts]{
	Headers:      neutronfind.Use(withSubnetPortHeaders),
	Microversion: neutronfind.Use(withSubnetPortMicroversion),
}
