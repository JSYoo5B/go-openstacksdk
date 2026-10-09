package image

import (
	"context"
	"encoding/json"
	"errors"
)

// ExistingImageRecord is Python Image.existing(id=...) as used by the Proxy
// _existing_image helper: a synchronized Record holding only the literal id,
// projected with the current Connection location and no HTTP request. Its raw
// body state is clean, so a later owned update starts from that id alone.
func (s *Service) ExistingImageRecord(ctx context.Context, id string) (*ImageRecord, error) {
	const operation = "ExistingImageRecord"
	if err := validateImageRecordIdentity(id); err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	p, err := s.prepareImageRecord(ctx, func(context.Context, func(context.Context) error) (map[string]string, error) {
		return nil, nil
	})
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	raw, _ := json.Marshal(id)
	record := &ImageRecord{bodyState: newImageRecordBodyState(map[string]json.RawMessage{"id": raw})}
	if err := errors.Join(projectImageRecordMutationSeed(record, p.location), p.check(p.ctx)); err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	return record, nil
}
