package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// GetSchemaRecord reads one fixed schema using its pinned Schema/MetadefSchema
// descriptor rules. HTTP 200..399 is accepted; JSON syntax errors leave bare
// resource defaults, while a parsed nonobject remains an observable error.
func (s *Service) GetSchemaRecord(ctx context.Context, kind SchemaKind, options ...GetSchemaOption) (*SchemaRecord, error) {
	const operation = "GetSchemaRecord"
	fail := func(err error) (*SchemaRecord, error) {
		return nil, wrapImageMutationError(ctx, operation, cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	path, metadef, err := schemaRecordPath(kind)
	if err != nil {
		return fail(err)
	}
	// This existing image source capture also serves task, image, update and
	// upload operations. The schema reader adds no new transport or fault engine.
	source, err := s.captureTaskSource(ctx)
	if err != nil {
		return fail(err)
	}
	api, images := s.API, s.Images
	outer := rest.OperationGuard(ctx)
	var observed error
	check := func(ctx context.Context) error {
		if observed != nil {
			return cloudread.ContextError(ctx, observed)
		}
		var binding, parent error
		if s.API != api || s.Images != images || api != nil && api.RawClient() != source.source {
			binding = uploadInvalid("schema service binding changed")
		}
		if outer != nil {
			parent = outer(ctx)
		}
		observed = errors.Join(source.check(ctx), binding, parent)
		return cloudread.ContextError(ctx, observed)
	}
	opctx := rest.WithOperationGuard(ctx, check)
	if err := check(opctx); err != nil {
		return fail(err)
	}
	location := json.RawMessage("null")
	if s.dependencies.CloudLocation != nil {
		facts, readErr := s.dependencies.CloudLocation()
		err = readErr
		if err == nil {
			// Schema has no project/zone descriptors: preserve the bare current
			// location, including any explicitly configured zone.
			location, err = facts.ForResource(nil, facts.Zone)
		}
	}
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(err)
	}
	guarded := slices.Clone(options)
	for i, apply := range guarded {
		if apply == nil {
			continue
		}
		guarded[i] = func(config *GetSchemaOpts) error {
			if err := check(opctx); err != nil {
				return err
			}
			return errors.Join(apply(config), check(opctx))
		}
	}
	prepared, err := s.prepareSchema(opctx, guarded)
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(err)
	}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	response, err := rest.DoJSONGuarded(opctx, prepared.client, check, http.MethodGet, prepared.base+path, nil, nil, codes...)
	if err != nil {
		return fail(err)
	}
	record, err := decodeSchemaRecord(kind, metadef, location, response)
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(response.Fail(err))
	}
	return record, nil
}
