package image

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GetImageRecord fetches a literal Image identity using a private seed snapshot.
// It follows Source descriptor/default/unknown-property response rules without
// mutating a supplied Resource or enabling Python's optional adapter cache.
func (s *Service) GetImageRecord(ctx context.Context, input ImageRecordRequest, options ...ImageRecordOption) (*ImageRecord, error) {
	fail := func(err error) (*ImageRecord, error) { return nil, wrapImageMutationError(ctx, "GetImageRecord", err) }
	owned := slices.Clone(options)
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return fail(err)
	}
	if input.ID != "" && input.Resource != nil {
		return fail(uploadInvalid("select image ID or Resource, not both"))
	}
	if input.ID != "" {
		if err := validateImageRecordIdentity(input.ID); err != nil {
			return fail(err)
		}
	}
	seed := make(map[string]json.RawMessage)
	if input.Resource != nil {
		seed = input.Resource.Clone().Body
		if seed == nil {
			seed = make(map[string]json.RawMessage)
		}
	}
	for _, key := range []string{"connection", "_synchronized", "microversion"} {
		if _, present := seed[key]; present {
			return fail(uploadInvalid("image seed %q collides with a source constructor argument", key))
		}
	}
	if input.ID != "" {
		seed["id"], _ = json.Marshal(input.ID)
	}
	// Constructor snapshots and attribute encodings finish before a location
	// callback can modify caller maps, raw fields or ordinary source headers.
	if _, err := imageRecordObject(seed); err != nil {
		return fail(err)
	}
	if err := p.check(p.ctx); err != nil {
		return fail(err)
	}
	attrs, err := captureImageRecordAttributes(p.ctx, p.check, input.Attributes)
	if err != nil {
		return fail(err)
	}
	if _, present := attrs["id"]; present && input.ID != "" {
		return fail(uploadInvalid("literal image input already binds id"))
	}
	for key, raw := range attrs {
		seed[key] = bytes.Clone(raw)
	}
	var identity string
	err = p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		attrs, headers, err := prepareImageRecordGet(opctx, check, owned)
		if err != nil {
			return nil, err
		}
		if _, present := attrs["id"]; present && input.ID != "" {
			return nil, uploadInvalid("literal image input already binds id")
		}
		for key, raw := range attrs {
			seed[key] = bytes.Clone(raw)
		}
		// This Go request combines names before one constructor projection. It
		// preserves unrelated unknown attributes across request/option maps; it is
		// not Python's mutable Resource _update component replacement lifecycle.
		seed, _, err = normalizeImageRecord(seed, nil, false, true)
		if err != nil {
			return nil, err
		}
		identity, err = decodeImageRecordString(seed["id"], "image seed identity")
		if err != nil {
			return nil, err
		}
		if err := validateImageRecordIdentity(identity); err != nil {
			return nil, err
		}
		// Do not let a server overwrite invalid descriptor input and hide a local
		// constructor failure. Every seed conversion is a preflight operation.
		if _, err := projectImageRecord(seed, p.location, resource.Metadata{}); err != nil {
			return nil, err
		}
		return headers, check(opctx)
	})
	if err != nil {
		return fail(err)
	}
	response, err := rest.DoJSONGuarded(p.ctx, p.client, p.check, http.MethodGet, imageRecordEndpoint(p, identity), nil, nil, imageRecordCodes()...)
	if err != nil {
		return fail(err)
	}
	record, err := imageRecordFromResponse(p.ctx, p.check, seed, p.location, response)
	if err != nil {
		return fail(err)
	}
	if !json.Valid(response.Body) {
		// An invalid response does not clean an unsynchronized constructor.
		// Preserve raw presence so a later owned update can commit its seed.
		record.bodyState = pendingImageRecordBodyState(seed)
	}
	return record, nil
}
func imageRecordEndpoint(p *preparedImageRecord, id string) string {
	return p.base + "images/" + url.PathEscape(id)
}
