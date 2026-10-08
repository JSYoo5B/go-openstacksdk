package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type flavorWorkflowFailure struct{ error }

func (e flavorWorkflowFailure) Unwrap() error          { return e.error }
func (flavorWorkflowFailure) TerminalSDKFailure() bool { return true }

type flavorRecordWorkflow struct {
	ctx      context.Context
	api      *flavors.API
	check    func(context.Context) error
	location json.RawMessage
}

// Capture the service and Connection's existing location reader before any
// leaf options, discovery, or physical reads. The leaf still owns its reader.
func (s *Service) captureFlavorRecords(ctx context.Context) (*flavorRecordWorkflow, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.API.Flavors == nil {
		return nil, invalid("flavor service is required")
	}
	serviceAPI, api, client := s.API, s.API.Flavors, s.client
	resources, flavorsCollection, servers := api.Resources, s.Flavors, s.Servers
	if serviceAPI.RawClient() != client || api.RawClient() != client {
		return nil, invalid("flavor components must share their selected client")
	}
	source, err := cloudread.Capture(ctx, client, "compute")
	if err != nil {
		return nil, err
	}
	outer := rest.OperationGuard(ctx)
	var observed error
	guard := func(checkCtx context.Context) error {
		if observed != nil {
			return cloudread.ContextError(checkCtx, observed)
		}
		var outerErr error
		if outer != nil {
			outerErr = outer(checkCtx)
		}
		var changed error
		if s.API != serviceAPI || serviceAPI.Flavors != api || s.client != client || s.Flavors != flavorsCollection || s.Servers != servers || api.Resources != resources || serviceAPI.RawClient() != client || api.RawClient() != client {
			changed = invalid("flavor service binding changed")
		}
		if err := errors.Join(source.Guard(checkCtx), outerErr, changed); err != nil {
			observed = flavorWorkflowFailure{err}
		}
		return observed
	}
	if err := guard(ctx); err != nil {
		return nil, err
	}
	p := &flavorRecordWorkflow{ctx: rest.WithOperationGuard(ctx, guard), api: api, check: guard}
	location := json.RawMessage("null")
	if servers != nil && servers.dependencies.CloudLocation != nil {
		reader := servers.dependencies.CloudLocation
		facts, readErr := reader()
		err = readErr
		if err == nil {
			location, err = facts.ForResource(nil, nil)
		}
	}
	if err = errors.Join(err, guard(p.ctx)); err != nil {
		return nil, err
	}
	p.location = bytes.Clone(location)
	return p, nil
}

// Only the owned views receive Connection location. Physical Wire objects and
// each request's envelope, headers, and status retain the leaf's exact evidence.
func (p *flavorRecordWorkflow) locate(record *flavors.FlavorRecord) *flavors.FlavorRecord {
	if record == nil {
		return nil
	}
	owned := *record
	if record.Resource != nil {
		owned.Resource = record.Resource.Clone()
		owned.Resource.Body["location"] = bytes.Clone(p.location)
	}
	if record.Enrichment != nil {
		enrichment := *record.Enrichment
		if enrichment.Resource != nil {
			enrichment.Resource = enrichment.Resource.Clone()
			enrichment.Resource.Body["location"] = bytes.Clone(p.location)
		}
		owned.Enrichment = &enrichment
	}
	return &owned
}

// ListFlavors adds Connection location to the owned leaf's summary/detail
// records. Paging, raw row caps, filters and conditional extra-specs enrichment
// remain library-owned leaf operations. Each iteration captures a fresh source
// and one location snapshot before applying any options.
func (s *Service) ListFlavors(ctx context.Context, options ...flavors.FlavorListOption) iter.Seq2[*flavors.FlavorRecord, error] {
	ownedOptions := slices.Clone(options)
	return func(yield func(*flavors.FlavorRecord, error) bool) {
		fail := func(err error) error {
			return request.Wrap("ListFlavors", "flavors", cloudread.ContextError(ctx, err))
		}
		p, err := s.captureFlavorRecords(ctx)
		if err != nil {
			yield(nil, fail(err))
			return
		}
		for record, readErr := range p.api.ListRecords(p.ctx, ownedOptions...) {
			err := errors.Join(readErr, p.check(p.ctx))
			if !yield(p.locate(record), fail(err)) || err != nil {
				return
			}
			if err := p.check(p.ctx); err != nil {
				yield(nil, fail(err))
				return
			}
		}
		if err := p.check(p.ctx); err != nil {
			yield(nil, fail(err))
		}
	}
}

// FindFlavor composes the leaf's fixed member GET and detailed list fallback
// with one recorded Connection location. Accepted partial records and
// their physical errors are returned together without another lookup.
func (s *Service) FindFlavor(ctx context.Context, identity string, options ...flavors.FlavorFindOption) (*flavors.FlavorRecord, error) {
	ownedOptions := slices.Clone(options)
	fail := func(err error) error {
		return request.Wrap("FindFlavor", "flavors", cloudread.ContextError(ctx, err))
	}
	p, err := s.captureFlavorRecords(ctx)
	if err != nil {
		return nil, fail(err)
	}
	record, err := p.api.FindFlavor(p.ctx, identity, ownedOptions...)
	err = errors.Join(err, p.check(p.ctx))
	return p.locate(record), fail(err)
}
