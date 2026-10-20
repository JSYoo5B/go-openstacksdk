package image

import "maps"

// ImageMemberOpts supplies ordinary headers for add, get and update requests.
type ImageMemberOpts struct{ Headers map[string]string }
type ImageMemberOption func(*ImageMemberOpts) error

// RemoveImageMemberOpts controls a member deletion. Nil IgnoreMissing means true.
type RemoveImageMemberOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type RemoveImageMemberOption func(*RemoveImageMemberOpts) error

// FindImageMemberOpts controls one direct member GET. Nil IgnoreMissing means true.
type FindImageMemberOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type FindImageMemberOption func(*FindImageMemberOpts) error

// ListImageMembersOpts bounds locally consumed rows; zero MaxItems is unlimited.
// The server receives no limit or pagination query.
type ListImageMembersOpts struct {
	Headers  map[string]string
	MaxItems int
}
type ListImageMembersOption func(*ListImageMembersOpts) error

// WithImageMemberOpts snapshots and replaces the entire concrete configuration.
func WithImageMemberOpts(value ImageMemberOpts) ImageMemberOption {
	snapshot := copyImageMemberOpts(value)
	return func(config *ImageMemberOpts) error { *config = copyImageMemberOpts(snapshot); return nil }
}
func WithImageMemberHeader(key, value string) ImageMemberOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *ImageMemberOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithImageMemberHeaders(values map[string]string) ImageMemberOption {
	apply := WithImageMutationHeaders(values)
	return func(config *ImageMemberOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithRemoveImageMemberOpts(value RemoveImageMemberOpts) RemoveImageMemberOption {
	snapshot := copyRemoveImageMemberOpts(value)
	return func(config *RemoveImageMemberOpts) error { *config = copyRemoveImageMemberOpts(snapshot); return nil }
}
func WithRemoveImageMemberHeader(key, value string) RemoveImageMemberOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *RemoveImageMemberOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithRemoveImageMemberHeaders(values map[string]string) RemoveImageMemberOption {
	apply := WithImageMutationHeaders(values)
	return func(config *RemoveImageMemberOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithRemoveImageMemberIgnoreMissing(value bool) RemoveImageMemberOption {
	return func(config *RemoveImageMemberOpts) error {
		snapshot := value
		config.IgnoreMissing = &snapshot
		return nil
	}
}
func WithFindImageMemberOpts(value FindImageMemberOpts) FindImageMemberOption {
	snapshot := copyFindImageMemberOpts(value)
	return func(config *FindImageMemberOpts) error { *config = copyFindImageMemberOpts(snapshot); return nil }
}
func WithFindImageMemberHeader(key, value string) FindImageMemberOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *FindImageMemberOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithFindImageMemberHeaders(values map[string]string) FindImageMemberOption {
	apply := WithImageMutationHeaders(values)
	return func(config *FindImageMemberOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithFindImageMemberIgnoreMissing(value bool) FindImageMemberOption {
	return func(config *FindImageMemberOpts) error {
		snapshot := value
		config.IgnoreMissing = &snapshot
		return nil
	}
}
func WithListImageMembersOpts(value ListImageMembersOpts) ListImageMembersOption {
	snapshot := copyListImageMembersOpts(value)
	return func(config *ListImageMembersOpts) error { *config = copyListImageMembersOpts(snapshot); return nil }
}
func WithListImageMembersHeader(key, value string) ListImageMembersOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *ListImageMembersOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithListImageMembersHeaders(values map[string]string) ListImageMembersOption {
	apply := WithImageMutationHeaders(values)
	return func(config *ListImageMembersOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithListImageMembersMaxItems(value int) ListImageMembersOption {
	return func(config *ListImageMembersOpts) error { config.MaxItems = value; return nil }
}

func applyImageMemberHeader(headers *map[string]string, apply ImageMutationOption) error {
	config := ImageMutationOpts{Headers: *headers}
	if err := apply(&config); err != nil {
		return err
	}
	*headers = config.Headers
	return nil
}
func copyImageMemberOpts(value ImageMemberOpts) ImageMemberOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	return value
}
func copyRemoveImageMemberOpts(value RemoveImageMemberOpts) RemoveImageMemberOpts {
	value.Headers = copyImageMemberOpts(ImageMemberOpts{Headers: value.Headers}).Headers
	if value.IgnoreMissing != nil {
		snapshot := *value.IgnoreMissing
		value.IgnoreMissing = &snapshot
	}
	return value
}
func copyFindImageMemberOpts(value FindImageMemberOpts) FindImageMemberOpts {
	value.Headers = copyImageMemberOpts(ImageMemberOpts{Headers: value.Headers}).Headers
	if value.IgnoreMissing != nil {
		snapshot := *value.IgnoreMissing
		value.IgnoreMissing = &snapshot
	}
	return value
}
func copyListImageMembersOpts(value ListImageMembersOpts) ListImageMembersOpts {
	value.Headers = copyImageMemberOpts(ImageMemberOpts{Headers: value.Headers}).Headers
	return value
}

func applyImageMemberOptions[T any, O ~func(*T) error](config T, options []O, copyConfig func(T) T) (T, error) {
	config = copyConfig(config)
	for _, apply := range options {
		if apply == nil {
			return config, uploadInvalid("nil image member option")
		}
		candidate := copyConfig(config)
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config = copyConfig(candidate)
	}
	return config, nil
}
func parseImageMemberOptions(options []ImageMemberOption) (ImageMemberOpts, error) {
	config, err := applyImageMemberOptions(ImageMemberOpts{}, options, copyImageMemberOpts)
	if err != nil {
		return config, err
	}
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, err
}
func parseRemoveImageMemberOptions(options []RemoveImageMemberOption) (RemoveImageMemberOpts, error) {
	config, err := applyImageMemberOptions(RemoveImageMemberOpts{}, options, copyRemoveImageMemberOpts)
	if err != nil {
		return config, err
	}
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, err
}
func parseFindImageMemberOptions(options []FindImageMemberOption) (FindImageMemberOpts, error) {
	config, err := applyImageMemberOptions(FindImageMemberOpts{}, options, copyFindImageMemberOpts)
	if err != nil {
		return config, err
	}
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, err
}
func parseListImageMembersOptions(options []ListImageMembersOption) (ListImageMembersOpts, error) {
	config, err := applyImageMemberOptions(ListImageMembersOpts{}, options, copyListImageMembersOpts)
	if err != nil {
		return config, err
	}
	if config.MaxItems < 0 {
		return config, uploadInvalid("max items must be non-negative")
	}
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, err
}
