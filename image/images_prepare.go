package image

import (
	"context"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (s *Service) prepareGetImage(ctx context.Context, ref resource.Ref, options []GetImageOption) (*preparedTaskSource, string, error) {
	p, err := s.captureTaskSource(ctx)
	if err == nil && ref.IsName() {
		err = ref.Validate()
	}
	if err == nil && !utf8.ValidString(ref.String()) {
		err = uploadInvalid("image reference must be valid UTF-8")
	}
	if err == nil && !ref.IsName() {
		err = validateTaskID(ref.String())
	}
	if err != nil {
		return nil, "", err
	}
	policy, err := parseGetImageOptions(options)
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, "", err
	}
	id := ref.String()
	if ref.IsName() {
		id, err = New(p.client).Images.ResolveID(ctx, ref)
		if err = checkTaskResponse(ctx, p, nil, err); err != nil {
			return nil, "", err
		}
		if err = validateTaskID(id); err != nil {
			return nil, "", err
		}
	}
	if err = p.check(ctx); err != nil {
		return nil, "", err
	}
	return p, id, nil
}
