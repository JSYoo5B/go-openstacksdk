package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
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

// pythonContains evaluates Source `exclude in value` for a decoded JSON value:
// substring for strings, string element equality for lists and key
// membership for dictionaries. Other values raise TypeError in Python.
func pythonContains(raw json.RawMessage, needle string) (bool, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return false, uploadInvalid("Cloud image name is required")
	}
	switch trimmed[0] {
	case '"':
		text, err := decodeImageRecordString(raw, "Cloud image name")
		return err == nil && strings.Contains(text, needle), err
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return false, errors.Join(uploadInvalid("Cloud image name list must be complete JSON"), err)
		}
		for _, item := range items {
			if element := bytes.TrimSpace(item); len(element) == 0 || element[0] != '"' {
				continue
			}
			text, err := decodeImageRecordString(item, "Cloud image name element")
			if err != nil {
				return false, err
			}
			if text == needle {
				return true, nil
			}
		}
		return false, nil
	case '{':
		var members map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &members); err != nil {
			return false, errors.Join(uploadInvalid("Cloud image name dictionary must be complete JSON"), err)
		}
		_, present := members[needle]
		return present, nil
	default:
		return false, uploadInvalid("Cloud image name %s is not a container for exclude", trimmed)
	}
}

// cloudImageRecordExclude is Cloud get_image_exclude: the first search_images
// row, or with truthy exclude the first reached row whose name does not
// contain it. Rows after the selected one are never inspected.
func cloudImageRecordExclude(p *preparedImageRecord, nameOrID, exclude string) (*CloudImageRecordResult, error) {
	if !utf8.ValidString(exclude) {
		return nil, uploadInvalid("Cloud image exclude must be valid UTF-8")
	}
	search, err := cloudImageRecordSearch(p, nameOrID, nil)
	result := &CloudImageRecordResult{Inventory: search.Inventory}
	if err != nil {
		return result, err
	}
	for _, record := range search.Images {
		if err := p.check(p.ctx); err != nil {
			return result, err
		}
		if exclude != "" {
			contains, err := pythonContains(record.Resource.Body["name"], exclude)
			if err != nil {
				receipt := &rest.Response{Body: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
				return result, receipt.Fail(err)
			}
			if contains {
				continue
			}
		}
		result.Image = record
		return result, p.check(p.ctx)
	}
	return result, p.check(p.ctx)
}

func (s *Service) getImageRecordExcluded(ctx context.Context, operation, nameOrID, exclude, field string, options []ImageRecordQueryOption) (*CloudImageRecordResult, error) {
	p, _, err := s.prepareImageRecordQuery(ctx, options, imageRecordQueryArguments{})
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	result, err := cloudImageRecordExclude(p, nameOrID, exclude)
	if err != nil {
		return result, wrapImageMutationError(ctx, operation, err)
	}
	if result.Image != nil {
		if field == "" {
			result.Value, err = imageRecordObject(result.Image.Resource.Body)
		} else {
			result.Value = bytes.Clone(result.Image.Resource.Body[field])
		}
		if err = errors.Join(err, p.check(p.ctx)); err != nil {
			result.Value = nil
			return result, wrapImageMutationError(ctx, operation, err)
		}
	}
	return result, nil
}

// GetImageRecordExclude implements Cloud get_image_exclude. It always uses
// the default search_images inventory; an empty exclude returns the first
// selected row. Value is the selected declared view and nil when absent.
func (s *Service) GetImageRecordExclude(ctx context.Context, nameOrID, exclude string, options ...ImageRecordQueryOption) (*CloudImageRecordResult, error) {
	return s.getImageRecordExcluded(ctx, "GetImageRecordExclude", nameOrID, exclude, "", options)
}

// GetImageRecordName implements Cloud get_image_name. Value is the selected
// row's raw name JSON, so a selected null name is present as null.
func (s *Service) GetImageRecordName(ctx context.Context, imageID, exclude string, options ...ImageRecordQueryOption) (*CloudImageRecordResult, error) {
	return s.getImageRecordExcluded(ctx, "GetImageRecordName", imageID, exclude, "name", options)
}

// GetImageRecordID implements Cloud get_image_id. Value is the selected row's
// raw id JSON; the identifier phase also matches IDs and globs.
func (s *Service) GetImageRecordID(ctx context.Context, imageName, exclude string, options ...ImageRecordQueryOption) (*CloudImageRecordResult, error) {
	return s.getImageRecordExcluded(ctx, "GetImageRecordID", imageName, exclude, "id", options)
}
