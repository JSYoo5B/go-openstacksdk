package image

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

func createImageRecordTask(p *preparedImageRecord, attrs map[string]json.RawMessage) (*ImageRecordCreateTask, *ImageUploadResponse, error) {
	if err := p.check(p.ctx); err != nil {
		return nil, nil, err
	}
	raw := copyTaskRawMap(attrs)
	// Capture every raw input before HTTP, including discarded extensions.
	if _, err := imageRecordObject(raw); err != nil {
		return nil, nil, err
	}
	for key := range raw {
		if err := validateImageRecordCreateControl(key, true); err != nil {
			return nil, nil, err
		}
	}
	if hook, present := raw["__conflicting_attrs"]; present {
		truthy, err := cloudfilter.PythonTruthy(hook)
		if err != nil {
			return nil, nil, err
		}
		if truthy {
			if bytes.TrimSpace(hook)[0] != '{' {
				return nil, nil, uploadInvalid("truthy task __conflicting_attrs must be a JSON object")
			}
			var overrides map[string]json.RawMessage
			if err := json.Unmarshal(hook, &overrides); err != nil {
				return nil, nil, err
			}
			for key, rawValue := range overrides {
				if err := validateImageRecordCreateControl(key, false); err != nil {
					return nil, nil, err
				}
				raw[key] = bytes.Clone(rawValue)
			}
			delete(raw, "__conflicting_attrs")
		}
	}
	body, err := normalizeImageRecordCreateTask(raw, nil)
	if err != nil {
		return nil, nil, err
	}
	seed, err := projectImageRecordCreateTask(body, p.location, nil)
	if err != nil {
		return nil, nil, err
	}
	wire := make(map[string]json.RawMessage, len(body))
	for _, field := range imageRecordCreateTaskFields {
		if raw, present := body[field.canonical]; present {
			wire[field.wire] = bytes.Clone(raw)
		}
	}
	encoded, err := imageRecordObject(wire)
	if err != nil {
		return nil, nil, err
	}
	response, err := rest.DoJSONGuardedRejectionsHeaders(p.ctx, p.client, p.check, http.MethodPost, p.base+"tasks", encoded,
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
	receipt := imageUploadResponse(response)
	if err != nil {
		return imageRecordCreateTaskReceipt(seed, response), receipt, err
	}
	value, err := imageRecordCreateTaskFromResponse(p, seed, response)
	if err != nil {
		if value == nil {
			value = imageRecordCreateTaskReceipt(seed, response)
		}
		return value, receipt, err
	}
	return value, receipt, nil
}

func imageRecordCreateTaskIdentity(task *ImageRecordCreateTask) (string, error) {
	raw := imageRecordCreateTaskRaw(task, "id")
	truthy, err := cloudfilter.PythonTruthy(raw)
	if err != nil {
		return "", err
	}
	if !truthy {
		return "", uploadInvalid("task GET requires a nonempty literal identity")
	}
	// Source urljoin stringifies raw Task IDs. Preserve that JSON-domain branch
	// while keeping the resulting identity in one safe literal Go URI segment.
	id, err := cloudfilter.PythonString(raw)
	if err != nil {
		return "", err
	}
	if err := validateImageRecordIdentity(id); err != nil {
		return "", err
	}
	return id, nil
}

func getCreatedImageRecordTask(p *preparedImageRecord, task *ImageRecordCreateTask) (*ImageRecordCreateTask, *ImageUploadResponse, error) {
	if err := p.check(p.ctx); err != nil {
		return nil, nil, err
	}
	seed := cloneImageRecordCreateTask(task)
	if _, err := imageRecordCreateTaskBody(seed); err != nil {
		return nil, nil, err
	}
	id, err := imageRecordCreateTaskIdentity(seed)
	if err != nil {
		return nil, nil, err
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, p.client, p.check, http.MethodGet, p.base+"tasks/"+url.PathEscape(id), nil, nil,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if response == nil {
		return nil, nil, err
	}
	receipt := imageUploadResponse(response)
	if err != nil {
		return imageRecordCreateTaskReceipt(seed, response), receipt, err
	}
	value, err := imageRecordCreateTaskFromResponse(p, seed, response)
	if err != nil && value == nil {
		value = imageRecordCreateTaskReceipt(seed, response)
	}
	return value, receipt, err
}
