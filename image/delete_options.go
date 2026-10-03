package image

import "unicode/utf8"

// DeleteImageOpts selects whole-image deletion or one store's copy.
// Nil StoreID deletes the whole image. Nil IgnoreMissing ignores absence.
// An explicitly empty StoreID is invalid and never expands the deletion scope.
type DeleteImageOpts struct {
	StoreID       *string
	IgnoreMissing *bool
}

type DeleteImageOption func(*DeleteImageOpts) error

// WithDeleteImageOpts snapshots and replaces the complete deletion policy.
func WithDeleteImageOpts(value DeleteImageOpts) DeleteImageOption {
	snapshot := copyDeleteImageOpts(value)
	return func(config *DeleteImageOpts) error { *config = copyDeleteImageOpts(snapshot); return nil }
}

// WithDeleteImageStore deletes the image copy from the specified store.
func WithDeleteImageStore(value string) DeleteImageOption {
	return func(config *DeleteImageOpts) error { owned := value; config.StoreID = &owned; return nil }
}

func WithDeleteImageIgnoreMissing(value bool) DeleteImageOption {
	return func(config *DeleteImageOpts) error { owned := value; config.IgnoreMissing = &owned; return nil }
}

func copyDeleteImageOpts(value DeleteImageOpts) DeleteImageOpts {
	value.StoreID = copyCreateImportPointer(value.StoreID)
	value.IgnoreMissing = copyCreateImportPointer(value.IgnoreMissing)
	return value
}

func parseDeleteImageOptions(options []DeleteImageOption) (DeleteImageOpts, error) {
	var config DeleteImageOpts
	for _, apply := range options {
		if apply == nil {
			return config, uploadInvalid("nil delete option")
		}
		// Each callback gets its own config; retained handles cannot affect the
		// next callback or the policy eventually used by the HTTP request.
		candidate := copyDeleteImageOpts(config)
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config = copyDeleteImageOpts(candidate)
	}
	if config.IgnoreMissing == nil {
		value := true
		config.IgnoreMissing = &value
	}
	if config.StoreID != nil {
		if !utf8.ValidString(*config.StoreID) {
			return config, uploadInvalid("delete store ID must be valid UTF-8")
		}
		if err := createImportID(*config.StoreID); err != nil {
			return config, err
		}
	}
	return config, nil
}
