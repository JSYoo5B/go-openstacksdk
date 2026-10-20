package image

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// This internal metadata constructor/create engine implements private
// Proxy._create(Image), not the modern public create_image workflow. It can be
// reused by later workflows without recapturing source or invoking a public API.
type preparedImageRecordCreate struct {
	*preparedImageRecord
	seed *ImageRecord
	body json.RawMessage
}

// Direct controls cannot change the captured Go source or request path. The
// Source conflicting hook runs after _create binding: inner base_path and
// resource_type therefore remain ordinary metadata, while constructor argument
// collisions still fail. Python's mutable class/session controls are not exposed.
func validateImageRecordCreateControl(key string, top bool) error {
	switch key {
	case "self", "connection", "_synchronized", "microversion":
		return uploadInvalid("image constructor attribute %q conflicts with the owned source", key)
	case "base_path", "resource_type":
		if top {
			return uploadInvalid("image constructor attribute %q conflicts with the fixed create target", key)
		}
	}
	return nil
}

func prepareImageRecordCreate(p *preparedImageRecord, attributes map[string]json.RawMessage) (*preparedImageRecordCreate, error) {
	if err := p.check(p.ctx); err != nil {
		return nil, err
	}
	raw := copyTaskRawMap(attributes)
	for key := range raw {
		if err := validateImageRecordCreateControl(key, true); err != nil {
			return nil, err
		}
	}
	if hook, present := raw["__conflicting_attrs"]; present {
		truthy, err := cloudfilter.PythonTruthy(hook)
		if err != nil {
			return nil, err
		}
		if truthy {
			if bytes.TrimSpace(hook)[0] != '{' {
				return nil, uploadInvalid("truthy __conflicting_attrs must be a JSON object")
			}
			var conflicting map[string]json.RawMessage
			if err := json.Unmarshal(hook, &conflicting); err != nil {
				return nil, err
			}
			for key, value := range conflicting {
				if err := validateImageRecordCreateControl(key, false); err != nil {
					return nil, err
				}
				raw[key] = bytes.Clone(value)
			}
			// A hook overriding its own name is also discarded by Source.
			delete(raw, "__conflicting_attrs")
		}
		// A falsey hook is not popped and becomes an unknown property.
	}
	fields, methods, err := normalizeImageRecord(raw, nil, false, true)
	if err != nil {
		return nil, err
	}
	view, err := projectImageRecord(fields, p.location, resource.Metadata{})
	if err != nil {
		return nil, errors.Join(err, p.check(p.ctx))
	}
	// Source stores dirty raw Body separately from getter projection. Every
	// constructor Body attribute, including id, feeds POST. A truthy id alone
	// enters the original patch baseline; no identity validation precedes POST.
	state := &imageRecordBodyState{current: copyTaskRawMap(fields), original: make(map[string]json.RawMessage), dirty: make(map[string]struct{})}
	for key := range fields {
		state.dirty[key] = struct{}{}
	}
	if id, present := fields["id"]; present {
		truthy, err := cloudfilter.PythonTruthy(id)
		if err != nil {
			return nil, err
		}
		if truthy {
			state.original["id"] = bytes.Clone(id)
		}
	}
	seed := &ImageRecord{Resource: view, ImportMethods: methods, bodyState: state}
	// Existing raw wire transposition/Source property flattening also applies
	// to creation, with ALL current raw fields rather than a dirty subset.
	body, err := imageRecordPatchBody(state.current)
	if err != nil {
		return nil, err
	}
	if err := p.check(p.ctx); err != nil {
		return nil, err
	}
	return &preparedImageRecordCreate{preparedImageRecord: p, seed: seed, body: body}, nil
}

func createPreparedImageRecord(p *preparedImageRecordCreate) (*ImageRecord, *rest.Response, error) {
	if err := p.check(p.ctx); err != nil {
		return nil, nil, err
	}
	response, err := rest.DoJSONGuardedRejectionsHeaders(p.ctx, p.client, p.check, http.MethodPost,
		p.base+"images", p.body,
		map[string]string{"Content-Type": "application/json", "Accept": "application/json"},
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if guardErr := p.check(p.ctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	if response == nil {
		return nil, nil, err
	}
	if err != nil {
		return imageRecordUpdateReceipt(p.seed, response), response, err
	}
	record, err := imageRecordFromResponse(p.ctx, p.check, p.seed.bodyState.current, p.location, response)
	if err != nil {
		return imageRecordUpdateReceipt(p.seed, response), response, err
	}
	if !json.Valid(response.Body) {
		record.bodyState = cloneImageRecordBodyState(p.seed.bodyState)
	}
	if err := p.check(p.ctx); err != nil {
		return record, response, response.Fail(err)
	}
	return record, response, nil
}
