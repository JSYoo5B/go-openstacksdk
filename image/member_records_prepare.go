package image

import (
	"context"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type preparedImageMemberRecord struct {
	*preparedImageMutation
	ctx      context.Context
	check    func(context.Context) error
	location json.RawMessage
}

// Capture source identity, bindings, outer guard and Connection facts before
// options or the explicit parent-name resolver can execute caller callbacks.
func (s *Service) prepareImageMemberRecord(ctx context.Context, parent resource.Ref, apply func(context.Context, func(context.Context) error) (map[string]string, error)) (*preparedImageMemberRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	source, err := s.captureTaskSource(ctx)
	if err != nil {
		return nil, err
	}
	if err := parent.Validate(); err != nil {
		return nil, err
	}
	if !utf8.ValidString(parent.String()) {
		return nil, uploadInvalid("member image reference must be UTF-8")
	}
	if !parent.IsName() {
		if err := createImportID(parent.String()); err != nil {
			return nil, err
		}
	}
	api, images := s.API, s.Images
	resourceBase := source.source.ResourceBase
	outer := rest.OperationGuard(ctx)
	var observed error
	check := func(checkCtx context.Context) error {
		if observed != nil {
			return cloudread.ContextError(checkCtx, observed)
		}
		var binding, ancestor error
		if s.API != api || s.Images != images || source.source.ResourceBase != resourceBase || api != nil && api.RawClient() != source.source {
			binding = uploadInvalid("member service binding changed")
		}
		if outer != nil {
			ancestor = outer(checkCtx)
		}
		observed = errors.Join(source.check(checkCtx), binding, ancestor)
		return cloudread.ContextError(checkCtx, observed)
	}
	opctx := rest.WithOperationGuard(ctx, check)
	if err := check(opctx); err != nil {
		return nil, err
	}
	location := json.RawMessage("null")
	if s.dependencies.CloudLocation != nil {
		facts, readErr := s.dependencies.CloudLocation()
		err = readErr
		if err == nil {
			location, err = facts.ForResource(nil, facts.Zone)
		}
	}
	if err = errors.Join(err, check(opctx)); err != nil {
		return nil, err
	}
	// Reuse the member preparation with the original header/client snapshot.
	// A location callback may change valid ordinary source headers; those
	// changes belong to the next operation, not this captured request.
	captured := *s
	captured.client = source.client
	prepared, err := captured.prepareImageMember(opctx, parent, nil, nil, func() (map[string]string, error) {
		if err := check(opctx); err != nil {
			return nil, err
		}
		headers, err := apply(opctx, check)
		return headers, errors.Join(err, check(opctx))
	})
	if err = errors.Join(err, check(opctx)); err != nil {
		return nil, err
	}
	prepared.service, prepared.source = s, source.source
	return &preparedImageMemberRecord{preparedImageMutation: prepared, ctx: opctx, check: check, location: location}, nil
}
