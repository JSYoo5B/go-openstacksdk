package image

import (
	"context"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (s *Service) prepareImageUpdateReference(ctx context.Context, ref resource.Ref) (*preparedTaskSource, error) {
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
	return p, err
}
func finishImageUpdateReference(ctx context.Context, p *preparedTaskSource, ref resource.Ref, headers map[string]string, err error) (string, error) {
	if err = p.finish(ctx, headers, err); err != nil {
		return "", err
	}
	id := ref.String()
	if ref.IsName() {
		id, err = New(p.client).Images.ResolveID(ctx, ref)
		if err = checkTaskResponse(ctx, p, nil, err); err != nil {
			return "", err
		}
		if err = validateTaskID(id); err != nil {
			return "", err
		}
	}
	if err = p.check(ctx); err != nil {
		return "", err
	}
	// Native ServiceClient.Request overlays service headers on RequestOpts. Set
	// these only on the captured client after Name lookup; the source is borrowed.
	p.client.MoreHeaders["Content-Type"] = "application/openstack-images-v2.1-json-patch"
	p.client.MoreHeaders["Accept"] = "application/json"
	return id, nil
}
