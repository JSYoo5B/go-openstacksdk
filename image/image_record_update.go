package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonpatch"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecordUpdateRequest selects a literal ID or an SDK-produced record.
// Attributes are raw resource changes. No GET or name lookup is performed.
// The private original/current state is copied; exposed view and receipt edits
// do not become changes. Options overlay attributes with the same name.
type ImageRecordUpdateRequest struct {
	ID         string
	Record     *ImageRecord
	Attributes map[string]any
}

type preparedImageRecordUpdate struct {
	*preparedImageRecord
	seed    *ImageRecord
	id      string
	patches []jsonpatch.Operation
}

const imageRecordPatchContentType = "application/openstack-images-v2.1-json-patch"

// UpdateImageRecord compares raw components and submits an automatic JSON Patch.
// Missing values differ from explicit null and descriptor defaults. A clean
// record causes no HTTP; a pending record may produce an empty PATCH. A valid
// object response overlays the submitted state and establishes a clean baseline.
// Tolerated invalid JSON keeps pending changes for an explicit subsequent call.
// Returned records are independent copies, including on accepted HTTP failures.
func (s *Service) UpdateImageRecord(ctx context.Context, input ImageRecordUpdateRequest, options ...ImageRecordOption) (*ImageRecord, error) {
	fail := func(record *ImageRecord, err error) (*ImageRecord, error) {
		return record, wrapImageMutationError(ctx, "UpdateImageRecord", err)
	}
	p, err := s.prepareImageRecordUpdate(ctx, input, slices.Clone(options))
	if err != nil {
		return fail(nil, err)
	}
	record, err := commitImageRecordUpdate(p)
	return fail(record, err)
}

// Already-prepared callers share the commit without recapturing source,
// options or current location between discovery and PATCH.
func commitImageRecordUpdate(p *preparedImageRecordUpdate) (*ImageRecord, error) {
	if len(p.seed.bodyState.dirty) == 0 {
		if err := p.check(p.ctx); err != nil {
			return nil, err
		}
		return p.seed, nil
	}
	response, err := rest.DoJSONGuardedRejectionsHeaders(p.ctx, p.client, p.check,
		http.MethodPatch, imageRecordEndpoint(p.preparedImageRecord, p.id), p.patches,
		map[string]string{"Content-Type": imageRecordPatchContentType, "Accept": ""},
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if response == nil {
		return nil, err
	}
	if err != nil {
		return imageRecordUpdateReceipt(p.seed, response), err
	}
	record, err := imageRecordFromResponse(p.ctx, p.check, p.seed.bodyState.current, p.location, response)
	if err != nil {
		return imageRecordUpdateReceipt(p.seed, response), err
	}
	if !json.Valid(response.Body) {
		// Source ValueError tolerance does not clean the pending components or
		// update the original body. This state survives immutable Go returns.
		record.bodyState = cloneImageRecordBodyState(p.seed.bodyState)
	}
	record.data = p.seed.data
	return record, nil
}

func (s *Service) prepareImageRecordUpdate(ctx context.Context, input ImageRecordUpdateRequest, options []ImageRecordOption) (*preparedImageRecordUpdate, error) {
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return nil, err
	}
	if input.ID != "" && input.Record != nil {
		return nil, uploadInvalid("select image ID or Record, not both")
	}
	seed := cloneImageRecord(input.Record)
	if seed == nil {
		if err := validateImageRecordIdentity(input.ID); err != nil {
			return nil, err
		}
		rawID, _ := json.Marshal(input.ID)
		seed = &ImageRecord{bodyState: newImageRecordBodyState(map[string]json.RawMessage{"id": rawID})}
	} else if seed.bodyState == nil {
		return nil, uploadInvalid("image Record must retain SDK-produced raw body state")
	}
	if seed.bodyState.dirty == nil {
		seed.bodyState.dirty = make(map[string]struct{})
	}
	attrs, err := captureImageRecordAttributes(p.ctx, p.check, input.Attributes)
	if err != nil {
		return nil, err
	}
	if err := validateImageRecordUpdateAttributes(attrs, input.ID != ""); err != nil {
		return nil, err
	}
	var prepared *preparedImageRecordUpdate
	err = p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		optionAttrs, headers, err := prepareImageRecordGet(opctx, check, options)
		if err != nil {
			return nil, err
		}
		if err := validateImageRecordUpdateAttributes(optionAttrs, input.ID != ""); err != nil {
			return nil, err
		}
		for key, raw := range optionAttrs {
			attrs[key] = bytes.Clone(raw)
		}
		// Keep direct-update supplied-ID validation ahead of normalization.
		if raw, present := attrs["id"]; present {
			if _, err := decodeImageRecordString(raw, "image update identity"); err != nil {
				return nil, err
			}
		}
		updates, methods, err := normalizeImageRecord(attrs, nil, false, true)
		if err != nil {
			return nil, err
		}
		prepared, err = prepareImageRecordNormalizedUpdate(p, seed, updates, methods)
		return headers, errors.Join(err, check(opctx))
	})
	if err != nil {
		return nil, err
	}
	installImageRecordUpdateHeaders(p)
	return prepared, p.check(p.ctx)
}

// Raw normalized components are applied to an already-owned seed. The
// helper is shared by direct updates and the whole property-helper workflow.
func prepareImageRecordNormalizedUpdate(p *preparedImageRecord, seed *ImageRecord, updates map[string]json.RawMessage, methods []string) (*preparedImageRecordUpdate, error) {
	// Validate before equality can hide an unpaired surrogate as equal to an
	// existing replacement character.
	if raw, present := updates["id"]; present {
		if _, err := decodeImageRecordString(raw, "image update identity"); err != nil {
			return nil, err
		}
	}
	if seed.bodyState.dirty == nil {
		seed.bodyState.dirty = make(map[string]struct{})
	}
	seed.ImportMethods = slices.Clone(methods)
	if err := updateImageRecordBody(seed.bodyState, updates); err != nil {
		return nil, err
	}
	// Discarding dirtiness retains the new current ID. Another dirty component
	// can therefore commit an /id change while routing through the current ID.
	delete(seed.bodyState.dirty, "id")
	identity, err := decodeImageRecordString(seed.bodyState.current["id"], "image record identity")
	if err != nil {
		return nil, err
	}
	if err := validateImageRecordIdentity(identity); err != nil {
		return nil, err
	}
	seed.Resource, err = projectImageRecord(seed.bodyState.current, p.location,
		resource.Metadata{Header: seed.Header.Clone(), StatusCode: seed.StatusCode})
	if err != nil {
		return nil, err
	}
	var patches []jsonpatch.Operation
	if len(seed.bodyState.dirty) != 0 {
		original, err := imageRecordPatchBody(seed.bodyState.original)
		if err != nil {
			return nil, err
		}
		current, err := imageRecordPatchBody(seed.bodyState.current)
		if err != nil {
			return nil, err
		}
		patches, err = jsonpatch.Diff(original, current)
		if err != nil {
			return nil, errors.Join(uploadInvalid("image body diff failed"), err)
		}
	}
	return &preparedImageRecordUpdate{preparedImageRecord: p, seed: seed, id: identity, patches: patches}, p.check(p.ctx)
}

func installImageRecordUpdateHeaders(p *preparedImageRecord) {
	// Keep both native service defaults and operation-owned request headers so
	// native header precedence cannot replace the image PATCH representation.
	p.client.MoreHeaders["Content-Type"] = imageRecordPatchContentType
	p.client.MoreHeaders["Accept"] = ""
}

func validateImageRecordUpdateAttributes(attrs map[string]json.RawMessage, literal bool) error {
	for key := range attrs {
		switch key {
		case "self", "image", "connection", "_synchronized", "microversion", "base_path":
			return uploadInvalid("image update attribute %q requires a source control outside the fixed owned profile", key)
		case "id":
			if literal {
				return uploadInvalid("literal image input already binds id")
			}
		}
	}
	return nil
}

func updateImageRecordBody(state *imageRecordBodyState, updates map[string]json.RawMessage) error {
	for key, raw := range updates {
		previous, present := state.current[key]
		equal := false
		if present {
			var err error
			equal, err = jsonfilter.EqualPythonJSON(previous, raw)
			if err != nil {
				return errors.Join(uploadInvalid("image attribute %q dirty comparison failed", key), err)
			}
		}
		if !present || !equal {
			state.current[key] = bytes.Clone(raw)
			state.dirty[key] = struct{}{}
		}
	}
	return nil
}

// Transpose raw canonical components to wire names, then unpack properties.
// The properties component replaces the previous one; when it is a truthy
// dictionary its keys overwrite declared wire keys only in the PATCH body.
func imageRecordPatchBody(fields map[string]json.RawMessage) (json.RawMessage, error) {
	body := make(map[string]json.RawMessage)
	for _, field := range imageRecordFields {
		if field.canonical == "properties" {
			continue
		}
		if raw, present := fields[field.canonical]; present {
			body[field.wire] = bytes.Clone(raw)
		}
	}
	if properties, present := fields["properties"]; present {
		truthy, err := cloudfilter.PythonTruthy(properties)
		if err != nil {
			return nil, err
		}
		if truthy {
			switch bytes.TrimSpace(properties)[0] {
			case '{':
				var props map[string]json.RawMessage
				if err := json.Unmarshal(properties, &props); err != nil {
					return nil, err
				}
				for key, raw := range props {
					body[key] = bytes.Clone(raw)
				}
			case '"':
				body["properties"] = bytes.Clone(properties)
			}
		}
	}
	return imageRecordObject(body)
}

// A failed accepted response proves HTTP, not translation success. Keep the
// submitted state pending and expose the actual passive response independently.
func imageRecordUpdateReceipt(seed *ImageRecord, response *rest.Response) *ImageRecord {
	record := cloneImageRecord(seed)
	record.Envelope = bytes.Clone(response.Body)
	record.Header = response.Header.Clone()
	record.StatusCode = response.StatusCode
	record.ImportMethods = imageRecordHTTPImportMethods(response.Header)
	record.Wire = nil
	if json.Valid(response.Body) {
		wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		if json.Unmarshal(response.Body, wire) == nil {
			record.Wire = wire
		}
	}
	if record.Resource != nil {
		record.Resource.Header = response.Header.Clone()
		record.Resource.StatusCode = response.StatusCode
	}
	return record
}
