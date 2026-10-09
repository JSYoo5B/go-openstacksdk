package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
)

type preparedImageRecordUpload struct {
	*preparedImageRecordCreate
	data   io.Reader
	policy ImageRecordUploadOpts
}

func (s *Service) prepareImageRecordUpload(ctx context.Context, input ImageRecordUploadRequest, options []ImageRecordUploadOption) (*preparedImageRecordUpload, error) {
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return nil, err
	}
	if input.Data != nil && isNilReader(input.Data) {
		return nil, uploadInvalid("image upload reader must not be typed nil")
	}
	// Caller metadata and binary reference are captured before options and the
	// current-location callback. No reader Read/Seek/Close is performed here.
	attrs, err := captureImageRecordUploadAttributes(p.ctx, p.check, input.Attributes)
	if err != nil {
		return nil, err
	}
	prepared := &preparedImageRecordUpload{data: input.Data}
	if err := p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		policy, err := prepareImageRecordUploadOptions(opctx, check, options)
		if err != nil {
			return nil, err
		}
		prepared.policy = policy
		raw := make(map[string]json.RawMessage, len(attrs)+len(policy.Attributes)+2)
		for key, value := range attrs {
			raw[key] = bytes.Clone(value.(json.RawMessage))
		}
		for key, value := range policy.Attributes {
			raw[key] = bytes.Clone(value.(json.RawMessage))
		}
		// Python's named arguments bind these before private _create. The Go
		// map cannot represent duplicate Python argument binding, so formal
		// formats own these initial keys; the Source hook may override them.
		raw["container_format"] = bytes.Clone(policy.ContainerFormat)
		raw["disk_format"] = bytes.Clone(policy.DiskFormat)
		create, err := prepareImageRecordCreate(p, raw)
		if err != nil {
			return nil, errors.Join(err, check(opctx))
		}
		prepared.preparedImageRecordCreate = create
		return policy.Headers, check(opctx)
	}); err != nil {
		return nil, err
	}
	// Metadata media belongs to this private prepared client. The shared
	// binary engine independently installs octet-stream and empty Accept.
	p.client.MoreHeaders["Content-Type"], p.client.MoreHeaders["Accept"] = "application/json", "application/json"
	return prepared, p.check(p.ctx)
}
