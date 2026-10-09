package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// cloudImageRecordSearch is Cloud search_images: complete default list_images
// before Python-shaped identifier glob and dictionary/JMESPath selection.
func cloudImageRecordSearch(p *preparedImageRecord, nameOrID string, filters *json.RawMessage) (*ImageRecordQueryResult, error) {
	rows, err := cloudImageRecordRows(p, true, false)
	result := &ImageRecordQueryResult{Inventory: rows.inventory}
	if err != nil {
		return result, err
	}
	selected, err := cloudfilter.Select(rows.views, nameOrID, filters, func() error { return p.check(p.ctx) })
	if err != nil {
		return result, errors.Join(fmt.Errorf("%w: image local search: %w", resource.ErrInvalidOption, err), p.check(p.ctx))
	}
	if err := p.check(p.ctx); err != nil {
		return result, err
	}
	result.Value = bytes.Clone(selected.Value)
	if !selected.Expression {
		result.Images = make([]*ImageRecord, len(selected.Indices))
		for index, source := range selected.Indices {
			result.Images[index] = rows.kept[source]
		}
	}
	return result, nil
}

func (s *Service) prepareImageRecordQuery(ctx context.Context, options []ImageRecordQueryOption, accepted imageRecordQueryArguments) (*preparedImageRecord, ImageRecordQueryOpts, error) {
	owned := slices.Clone(options)
	var policy ImageRecordQueryOpts
	p, err := s.prepareImageRecord(ctx, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		var err error
		policy, err = prepareImageRecordQuery(opctx, check, owned, accepted)
		return policy.Headers, err
	})
	return p, policy, err
}

// AllCloudImageRecords implements Cloud list_images. It eagerly consumes the
// complete default image list and removes rows whose status lowers to deleted
// unless FilterDeleted is false. ShowAll adds member_status=all and disables
// the status filter. ListImageRecords remains the lazy Proxy images API.
func (s *Service) AllCloudImageRecords(ctx context.Context, options ...ImageRecordQueryOption) (*ImageRecordQueryResult, error) {
	const operation = "AllCloudImageRecords"
	p, policy, err := s.prepareImageRecordQuery(ctx, options, imageRecordQueryArguments{listControls: true})
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	filterDeleted := policy.FilterDeleted == nil || *policy.FilterDeleted
	showAll := policy.ShowAll != nil && *policy.ShowAll
	rows, err := cloudImageRecordRows(p, filterDeleted, showAll)
	result := &ImageRecordQueryResult{Inventory: rows.inventory}
	if err != nil {
		return result, wrapImageMutationError(ctx, operation, err)
	}
	selected, err := cloudfilter.Select(rows.views, "", nil, func() error { return p.check(p.ctx) })
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return result, wrapImageMutationError(ctx, operation, err)
	}
	result.Value = bytes.Clone(selected.Value)
	result.Images = slices.Clone(rows.kept)
	return result, nil
}

// SearchImageRecords implements Cloud search_images. It always uses the
// default list_images inventory, then applies the identifier phase and
// optional dictionary or JMESPath filters to the declared Resource views.
func (s *Service) SearchImageRecords(ctx context.Context, nameOrID string, options ...ImageRecordQueryOption) (*ImageRecordQueryResult, error) {
	const operation = "SearchImageRecords"
	p, policy, err := s.prepareImageRecordQuery(ctx, options, imageRecordQueryArguments{filters: true})
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	result, err := cloudImageRecordSearch(p, nameOrID, policy.Filters)
	return result, wrapImageMutationError(ctx, operation, err)
}

// GetCloudImageRecord implements Cloud get_image. Omitted or null filters use
// the Proxy find_image engine with ignore_missing true. Every other filter
// value, including falsey JSON, uses the deprecated search path followed by
// Python's truthiness, len and index-0 selection.
func (s *Service) GetCloudImageRecord(ctx context.Context, nameOrID string, options ...ImageRecordQueryOption) (*CloudImageRecordResult, error) {
	const operation = "GetCloudImageRecord"
	p, policy, err := s.prepareImageRecordQuery(ctx, options, imageRecordQueryArguments{filters: true})
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	result := &CloudImageRecordResult{}
	if policy.Filters != nil && !bytes.Equal(bytes.TrimSpace(*policy.Filters), []byte("null")) {
		search, err := cloudImageRecordSearch(p, nameOrID, policy.Filters)
		result.Inventory = search.Inventory
		if err != nil {
			return result, wrapImageMutationError(ctx, operation, err)
		}
		selected, err := cloudfilter.First(search.Value)
		if err != nil {
			var multiple *cloudfilter.MultipleError
			if errors.As(err, &multiple) {
				err = &ImageRecordSelectionError{NameOrID: nameOrID, Length: multiple.Length}
			} else {
				err = fmt.Errorf("%w: local image selection: %w", resource.ErrInvalidOption, err)
			}
			return result, wrapImageMutationError(ctx, operation, err)
		}
		if err := p.check(p.ctx); err != nil {
			return result, wrapImageMutationError(ctx, operation, err)
		}
		result.Value = bytes.Clone(selected)
		if selected != nil && len(search.Images) == 1 {
			result.Image = search.Images[0]
		}
		return result, nil
	}
	if err := validateImageRecordIdentity(nameOrID); err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	parameters, err := prepareImageRecordList(p.ctx, p.check, nil)
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	record, err := findPreparedImageRecord(p, nameOrID, FindImageRecordOpts{}, parameters)
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	if record != nil {
		result.Image = record
		result.Value, err = imageRecordObject(record.Resource.Body)
		if err = errors.Join(err, p.check(p.ctx)); err != nil {
			return result, wrapImageMutationError(ctx, operation, err)
		}
	}
	return result, nil
}

// GetImageRecordByID implements Cloud get_image_by_id: one strict literal
// GetImageRecord without Find fallback, missing-as-nil or list requests.
func (s *Service) GetImageRecordByID(ctx context.Context, id string, options ...ImageRecordQueryOption) (*ImageRecord, error) {
	const operation = "GetImageRecordByID"
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	if err := validateImageRecordIdentity(id); err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	// Each Cloud option becomes one Get option, so the reused preparation checks
	// context and source drift between callbacks. The final step admits only
	// headers because the Source helper has no list or filter arguments.
	state := copyImageRecordQuery(ImageRecordQueryOpts{})
	converted := make([]ImageRecordOption, 0, len(options)+1)
	for _, apply := range slices.Clone(options) {
		converted = append(converted, func(*ImageRecordOpts) error {
			if apply == nil {
				return uploadInvalid("nil image record query option")
			}
			next := copyImageRecordQuery(state)
			if err := apply(&next); err != nil {
				return err
			}
			state = copyImageRecordQuery(next)
			return nil
		})
	}
	converted = append(converted, func(config *ImageRecordOpts) error {
		policy, err := prepareImageRecordQuery(ctx, cloudread.Context, []ImageRecordQueryOption{WithImageRecordQueryOpts(state)}, imageRecordQueryArguments{})
		if err != nil {
			return err
		}
		return mergeImageRecordHeaders(&config.Headers, policy.Headers)
	})
	record, err := s.GetImageRecord(ctx, ImageRecordRequest{ID: id}, converted...)
	if failure, ok := err.(*resource.OperationError); ok {
		// Name the Cloud entry point without nesting the delegated operation.
		renamed := *failure
		renamed.Operation = operation
		return record, &renamed
	}
	return record, err
}
