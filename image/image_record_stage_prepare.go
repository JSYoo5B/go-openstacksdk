package image

import (
	"context"
	"errors"
	"io"
)

type preparedImageRecordStage struct {
	*preparedImageRecord
	seed     *ImageRecord
	identity string
	data     io.Reader
	filename string
	policy   ImageRecordStageOpts
}

// Preparation captures data references without opening, reading or seeking.
// The private body, option slice and source are owned before any callback.
func (s *Service) prepareImageRecordStage(ctx context.Context, input ImageRecordStageRequest, options []ImageRecordStageOption) (*preparedImageRecordStage, error) {
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return nil, err
	}
	seed, identity, err := imageRecordMutationSeed(input.ID, input.Record)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, err
	}
	if input.Filename != "" && input.Data != nil {
		return nil, uploadInvalid("image stage filename and Data are mutually exclusive")
	}
	data := seed.data
	if input.Data != nil {
		data = input.Data
	}
	if input.Filename == "" && data != nil && isNilReader(data) {
		return nil, uploadInvalid("image stage reader must not be typed nil")
	}
	prepared := &preparedImageRecordStage{preparedImageRecord: p, seed: seed,
		identity: identity, data: data, filename: input.Filename}
	if err := p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		if err := errors.Join(projectImageRecordMutationSeed(seed, p.location), check(opctx)); err != nil {
			return nil, err
		}
		status, err := decodeImageRecordString(seed.Resource.Body["status"], "image stage status")
		if err != nil {
			return nil, errors.Join(err, check(opctx))
		}
		if status != "queued" {
			return nil, uploadInvalid("image stage requires queued status, got %q", status)
		}
		prepared.policy, err = prepareImageRecordStageOptions(opctx, check, options)
		return prepared.policy.Headers, errors.Join(err, check(opctx))
	}); err != nil {
		return nil, err
	}
	seed.data = data
	return prepared, p.check(p.ctx)
}
