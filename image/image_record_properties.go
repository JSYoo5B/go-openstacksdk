package image

import (
	"context"
	"errors"
	"slices"
)

// ImageRecordPropertiesRequest selects a literal ID or an SDK-produced Image
// record. A literal ID is not fetched: its missing cached properties fail after
// any requested kernel/ramdisk discovery. Public view edits do not become data.
type ImageRecordPropertiesRequest struct {
	ID     string
	Record *ImageRecord
}

// ImageRecordPropertiesResult keeps the Source helper boolean distinct from
// its immutable record. Updated means a nonempty prepared property map selected
// the update workflow; it can be true without a PATCH. It does not assert a
// changed field or server save. Updated is false on an error with a receipt.
type ImageRecordPropertiesResult struct {
	Updated bool
	Record  *ImageRecord
}

// UpdateImagePropertiesRecord implements the complete owned property helper:
// cached property copy, kernel/ramdisk discovery, int/raw/string conversion,
// raw meta precedence, canonical comparison and the shared automatic commit.
// It does not fetch the target, find by name or poll. Noop success returns a
// prepared independent record; accepted PATCH failures retain its real receipt.
func (s *Service) UpdateImagePropertiesRecord(ctx context.Context, input ImageRecordPropertiesRequest, options ...ImageRecordPropertiesOption) (*ImageRecordPropertiesResult, error) {
	fail := func(record *ImageRecord, err error) (*ImageRecordPropertiesResult, error) {
		var result *ImageRecordPropertiesResult
		if record != nil {
			result = &ImageRecordPropertiesResult{Record: record}
		}
		return result, wrapImageMutationError(ctx, "UpdateImagePropertiesRecord", err)
	}
	owned := slices.Clone(options)
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return fail(nil, err)
	}
	seed, _, err := imageRecordMutationSeed(input.ID, input.Record)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return fail(nil, err)
	}
	var policy ImageRecordPropertiesOpts
	if err := p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		// Source _get_resource materializes all descriptors before kwargs and
		// cached-property processing, including when the helper will be a noop.
		if err := errors.Join(projectImageRecordMutationSeed(seed, p.location), check(opctx)); err != nil {
			return nil, err
		}
		var err error
		policy, err = prepareImageRecordPropertiesOptions(opctx, check, owned)
		return policy.Headers, err
	}); err != nil {
		return fail(nil, err)
	}
	prepared, selected, err := prepareImageRecordPropertiesUpdate(p, seed, policy)
	if err != nil {
		return fail(nil, err)
	}
	if !selected {
		if err := p.check(p.ctx); err != nil {
			return fail(nil, err)
		}
		return &ImageRecordPropertiesResult{Record: seed}, nil
	}
	installImageRecordUpdateHeaders(p)
	record, err := commitImageRecordUpdate(prepared)
	if err != nil {
		return fail(record, err)
	}
	return &ImageRecordPropertiesResult{Updated: true, Record: record}, nil
}
