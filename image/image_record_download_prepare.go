package image

import (
	"context"
	"encoding/json"
	"errors"
	"unicode/utf8"
)

type preparedImageRecordDownload struct {
	*preparedImageRecord
	seed     *ImageRecord
	identity string
	input    ImageRecordDownloadRequest
	policy   ImageRecordDownloadOpts
}

func (s *Service) prepareImageRecordDownload(ctx context.Context, input ImageRecordDownloadRequest, options []ImageRecordDownloadOption) (*preparedImageRecordDownload, error) {
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return nil, err
	}
	if input.Output != nil && downloadNilWriter(input.Output) {
		return nil, uploadInvalid("image download writer must not be typed nil")
	}
	if input.Output != nil && input.Filename != "" {
		return nil, uploadInvalid("select image download Output or Filename, not both")
	}
	if !utf8.ValidString(input.Filename) {
		return nil, uploadInvalid("image download filename must be valid UTF-8")
	}
	count := 0
	if input.ID != "" {
		count++
	}
	if input.Record != nil {
		count++
	}
	if input.Resource != nil {
		count++
	}
	if count != 1 {
		return nil, uploadInvalid("select exactly one image ID, Record or Resource")
	}
	var seed *ImageRecord
	var identity string
	if input.Resource == nil {
		seed, identity, err = imageRecordMutationSeed(input.ID, input.Record)
	} else {
		raw := input.Resource.Clone().Body
		for _, key := range []string{"self", "connection", "_synchronized", "microversion"} {
			if _, present := raw[key]; present {
				return nil, uploadInvalid("image download seed %q collides with a source constructor argument", key)
			}
		}
		var normalized map[string]json.RawMessage
		normalized, _, err = normalizeImageRecord(raw, nil, false, true)
		if err == nil {
			seed = &ImageRecord{bodyState: newImageRecordBodyState(normalized)}
			identity, err = decodeImageRecordString(normalized["id"], "image download seed identity")
		}
		if err == nil {
			err = validateImageRecordIdentity(identity)
		}
	}
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, err
	}
	prepared := &preparedImageRecordDownload{preparedImageRecord: p, seed: seed, identity: identity, input: input}
	// No caller-owned raw map or mutable view is accessed after callbacks begin.
	prepared.input.Record, prepared.input.Resource = nil, nil
	if err := p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		if err := errors.Join(projectImageRecordMutationSeed(seed, p.location), check(opctx)); err != nil {
			return nil, err
		}
		prepared.policy, err = prepareImageRecordDownloadOptions(opctx, check, options)
		return prepared.policy.Headers, errors.Join(err, check(opctx))
	}); err != nil {
		return nil, err
	}
	return prepared, p.check(p.ctx)
}
