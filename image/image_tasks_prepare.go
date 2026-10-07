package image

import (
	"context"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func (s *Service) prepareImageTasks(ctx context.Context, ref resource.Ref, options []ListImageTasksOption) (*preparedTaskSource, string, ListImageTasksOpts, error) {
	p, err := s.captureTaskSource(ctx)
	if err == nil && ref.IsName() {
		err = ref.Validate()
	}
	if err == nil && !utf8.ValidString(ref.String()) {
		err = uploadInvalid("image task reference must be valid UTF-8")
	}
	if err == nil && !ref.IsName() {
		err = validateImageTasksID(ref.String())
	}
	if err != nil {
		return nil, "", ListImageTasksOpts{}, err
	}
	policy, err := parseListImageTasksOptions(options)
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, "", policy, err
	}
	id := ref.String()
	if ref.IsName() {
		id, err = New(p.client).Images.ResolveID(ctx, ref)
		if err = checkTaskResponse(ctx, p, nil, err); err != nil {
			return nil, "", policy, err
		}
		if err = validateImageTasksID(id); err != nil {
			return nil, "", policy, err
		}
	}
	if err = p.check(ctx); err != nil {
		return nil, "", policy, err
	}
	return p, id, policy, nil
}

func validateImageTasksID(value string) error {
	return validateTaskID(value)
}
