package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"reflect"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type zoneWorkflow struct {
	ctx      context.Context
	api      *availabilityzones.API
	check    func(context.Context) error
	location json.RawMessage
}
type zoneWorkflowFailure struct{ error }

func (e zoneWorkflowFailure) Unwrap() error          { return e.error }
func (zoneWorkflowFailure) TerminalSDKFailure() bool { return true }

func (s *Service) captureZones(ctx context.Context) (*zoneWorkflow, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.API.AvailabilityZones == nil {
		return nil, invalid("availability zone service is required")
	}
	serviceAPI, api, client, servers := s.API, s.API.AvailabilityZones, s.client, s.Servers
	if serviceAPI.RawClient() != client || api.RawClient() != client {
		return nil, invalid("availability zone components must share their client")
	}
	source, err := cloudread.Capture(ctx, client, "compute")
	if err != nil {
		return nil, err
	}
	outer := rest.OperationGuard(ctx)
	var observed error
	check := func(ctx context.Context) error {
		if observed != nil {
			return cloudread.ContextError(ctx, observed)
		}
		var changed, parent error
		if s.API != serviceAPI || serviceAPI.AvailabilityZones != api || s.client != client || s.Servers != servers || serviceAPI.RawClient() != client || api.RawClient() != client {
			changed = invalid("availability zone service binding changed")
		}
		if outer != nil {
			parent = outer(ctx)
		}
		if err := errors.Join(source.Guard(ctx), changed, parent); err != nil {
			observed = zoneWorkflowFailure{err}
		}
		return observed
	}
	if err := check(ctx); err != nil {
		return nil, err
	}
	p := &zoneWorkflow{ctx: rest.WithOperationGuard(ctx, check), api: api, check: check, location: json.RawMessage("null")}
	if servers != nil && servers.dependencies.CloudLocation != nil {
		facts, readErr := servers.dependencies.CloudLocation()
		err = readErr
		if err == nil {
			p.location, err = facts.ForResource(nil, nil)
		}
	}
	if err = errors.Join(err, check(p.ctx)); err != nil {
		return nil, err
	}
	p.location = bytes.Clone(p.location)
	return p, nil
}
func (p *zoneWorkflow) locate(record *availabilityzones.AvailabilityZoneRecord) *availabilityzones.AvailabilityZoneRecord {
	if record == nil {
		return nil
	}
	owned := *record
	owned.Resource = record.Resource.Clone()
	if owned.Resource != nil {
		owned.Resource.Body["location"] = bytes.Clone(p.location)
	}
	return &owned
}

func (s *Service) ListAvailabilityZones(ctx context.Context, options ...availabilityzones.AvailabilityZoneListOption) iter.Seq2[*availabilityzones.AvailabilityZoneRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*availabilityzones.AvailabilityZoneRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListAvailabilityZones", "availabilityzones", cloudread.ContextError(ctx, err))
		}
		p, err := s.captureZones(ctx)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		for record, err := range p.api.ListRecords(p.ctx, owned...) {
			err = errors.Join(err, p.check(p.ctx))
			if !yield(p.locate(record), wrap(err)) || err != nil {
				return
			}
		}
		if err := p.check(p.ctx); err != nil {
			yield(nil, wrap(err))
		}
	}
}

type AvailabilityZoneNamesOpts struct {
	Unavailable  bool
	Microversion *string
}
type AvailabilityZoneNamesOption = request.Option[AvailabilityZoneNamesOpts]

func ownZoneNames(c *request.Config[AvailabilityZoneNamesOpts]) {
	cloudread.OwnReadConfig(c)
	if c.Options.Microversion != nil {
		v := *c.Options.Microversion
		c.Options.Microversion = &v
	}
}
func WithAvailabilityZoneNamesOptions(value AvailabilityZoneNamesOpts) AvailabilityZoneNamesOption {
	if value.Microversion != nil {
		v := *value.Microversion
		value.Microversion = &v
	}
	return func(c *request.Config[AvailabilityZoneNamesOpts]) error {
		c.Options = value
		ownZoneNames(c)
		return nil
	}
}
func WithUnavailableZones(value bool) AvailabilityZoneNamesOption {
	return func(c *request.Config[AvailabilityZoneNamesOpts]) error { c.Options.Unavailable = value; return nil }
}
func WithAvailabilityZoneNamesMicroversion(value string) AvailabilityZoneNamesOption {
	return func(c *request.Config[AvailabilityZoneNamesOpts]) error {
		v := value
		c.Options.Microversion = &v
		return nil
	}
}
func WithAvailabilityZoneNamesHeader(key, value string) AvailabilityZoneNamesOption {
	return request.WithHeader[AvailabilityZoneNamesOpts](key, value)
}

// Names and Value are completed results. SuppressedError explains a source-
// compatible empty result; Inventory retains the consumed diagnostic records.
type AvailabilityZoneNamesResult struct {
	Names           []json.RawMessage
	Value           json.RawMessage
	Inventory       []*availabilityzones.AvailabilityZoneRecord
	SuppressedError error
}

func suppressZoneListFailure(err error) bool {
	// A callback/source/accepted/transport failure must never masquerade as a
	// native rejection merely because a nested cause carries an HTTP status.
	var terminal interface{ TerminalSDKFailure() bool }
	var transport *url.Error
	if errors.As(err, &terminal) && terminal.TerminalSDKFailure() || errors.As(err, &transport) || errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, resource.ErrUnsupported) {
		return false
	}
	for current := err; current != nil; {
		if joined, ok := current.(interface{ Unwrap() []error }); ok {
			parts := joined.Unwrap()
			if len(parts) == 0 {
				return false
			}
			// A retry callback may return the exact original rejection. The
			// common request layer retains both occurrences; this adds no new
			// failure. Any different callback cause still prevents suppression.
			for _, duplicate := range parts[1:] {
				if !reflect.DeepEqual(parts[0], duplicate) {
					return false
				}
			}
			current = parts[0]
		} else {
			current = errors.Unwrap(current)
		}
	}
	var cycle *resource.PaginationCycleError
	if errors.As(err, &cycle) {
		return true
	}
	var accepted *resource.ResponseError
	if errors.As(err, &accepted) {
		return false
	}
	var httpErr gophercloud.ErrUnexpectedResponseCode
	return errors.As(err, &httpErr) && httpErr.Actual >= 400 && httpErr.Actual < 600
}

func (s *Service) ListAvailabilityZoneNames(ctx context.Context, options ...AvailabilityZoneNamesOption) (*AvailabilityZoneNamesResult, error) {
	owned := slices.Clone(options)
	wrap := func(err error) error {
		return request.Wrap("ListAvailabilityZoneNames", "availabilityzones", cloudread.ContextError(ctx, err))
	}
	p, err := s.captureZones(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	config, err := cloudread.ApplyReadOptions(p.ctx, AvailabilityZoneNamesOpts{}, owned, ownZoneNames, p.check)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, wrap(err)
	}
	leaf := []availabilityzones.AvailabilityZoneListOption{}
	if config.Options.Microversion != nil {
		leaf = append(leaf, availabilityzones.WithAvailabilityZoneMicroversion(*config.Options.Microversion))
	}
	for key, value := range config.Headers {
		leaf = append(leaf, availabilityzones.WithAvailabilityZoneHeader(key, value))
	}
	result := &AvailabilityZoneNamesResult{}
	names := []json.RawMessage{}
	for record, readErr := range p.api.ListRecords(p.ctx, leaf...) {
		record = p.locate(record)
		if record != nil {
			result.Inventory = append(result.Inventory, record)
		}
		if err := p.check(p.ctx); err != nil {
			return result, wrap(errors.Join(readErr, err))
		}
		if readErr != nil {
			if suppressZoneListFailure(readErr) {
				result.Names = []json.RawMessage{}
				result.Value = json.RawMessage("[]")
				result.SuppressedError = wrap(readErr)
				return result, nil
			}
			return result, wrap(readErr)
		}
		state := bytes.TrimSpace(record.Resource.Body["state"])
		var members map[string]json.RawMessage
		rowFailure := func(err error) error {
			return wrap(&resource.ResponseError{Body: bytes.Clone(record.Envelope), Header: record.Header.Clone(), StatusCode: record.StatusCode, Cause: err})
		}
		if len(state) == 0 || state[0] != '{' {
			return result, rowFailure(fmt.Errorf("%w: availability zone state must be an object", resource.ErrInvalidOption))
		}
		if err := json.Unmarshal(state, &members); err != nil {
			return result, rowFailure(err)
		}
		available, exists := members["available"]
		if !exists {
			return result, rowFailure(fmt.Errorf("%w: availability zone state lacks available", resource.ErrInvalidOption))
		}
		truthy, err := cloudfilter.PythonTruthy(available)
		if err != nil {
			return result, rowFailure(err)
		}
		if truthy || config.Options.Unavailable {
			names = append(names, bytes.Clone(record.Resource.Body["name"]))
		}
	}
	if err := p.check(p.ctx); err != nil {
		return result, wrap(err)
	}
	result.Value, err = json.Marshal(names)
	if err != nil {
		return result, wrap(err)
	}
	result.Names = names
	return result, nil
}
