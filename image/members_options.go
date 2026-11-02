package image

import (
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ImageMemberOpts supplies ordinary headers for add, get and update requests.
type ImageMemberOpts struct{ Headers map[string]string }

// ImageMemberOption is one option of AddImageMember, GetImageMember and
// UpdateImageMember: WithImageMemberOpts, WithImageMemberHeaders (merges),
// request.Header (merges one header) or request.Custom over *ImageMemberOpts.
// Other shared calls, including request.Headers, fail with
// resource.ErrInvalidOption before HTTP.
type ImageMemberOption = request.GetCall

// RemoveImageMemberOpts controls a member deletion. Nil IgnoreMissing means true.
type RemoveImageMemberOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// RemoveImageMemberOption is one option of RemoveImageMember:
// WithRemoveImageMemberOpts, WithRemoveImageMemberHeaders (merges),
// request.Header (merges one header), request.IgnoreMissing or request.Custom
// over *RemoveImageMemberOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RemoveImageMemberOption = request.DeleteCall

// FindImageMemberOpts controls one direct member GET. Nil IgnoreMissing means true.
type FindImageMemberOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// FindImageMemberOption is one option of FindImageMember:
// WithFindImageMemberOpts, WithFindImageMemberHeaders (merges),
// request.Header (merges one header), request.IgnoreMissing or request.Custom
// over *FindImageMemberOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type FindImageMemberOption = request.FindCall

// ListImageMembersOpts bounds locally consumed rows; zero MaxItems is unlimited.
// The server receives no limit or pagination query.
type ListImageMembersOpts struct {
	Headers  map[string]string
	MaxItems int
}

// ListImageMembersOption is one option of ListImageMembers:
// WithListImageMembersOpts, WithListImageMembersHeaders (merges),
// request.Header (merges one header), request.MaxItems or request.Custom over
// *ListImageMembersOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListImageMembersOption = request.ListCall

// WithImageMemberOpts snapshots and replaces the entire concrete configuration.
func WithImageMemberOpts(value ImageMemberOpts) ImageMemberOption {
	snapshot := copyImageMemberOpts(value)
	return request.Custom(func(config *ImageMemberOpts) error { *config = copyImageMemberOpts(snapshot); return nil })
}
func WithImageMemberHeaders(values map[string]string) ImageMemberOption {
	apply := mergeImageMutationHeaders(values)
	return request.Custom(func(config *ImageMemberOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}
func WithRemoveImageMemberOpts(value RemoveImageMemberOpts) RemoveImageMemberOption {
	snapshot := copyRemoveImageMemberOpts(value)
	return request.Custom(func(config *RemoveImageMemberOpts) error { *config = copyRemoveImageMemberOpts(snapshot); return nil })
}
func WithRemoveImageMemberHeaders(values map[string]string) RemoveImageMemberOption {
	apply := mergeImageMutationHeaders(values)
	return request.Custom(func(config *RemoveImageMemberOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}
func WithFindImageMemberOpts(value FindImageMemberOpts) FindImageMemberOption {
	snapshot := copyFindImageMemberOpts(value)
	return request.Custom(func(config *FindImageMemberOpts) error { *config = copyFindImageMemberOpts(snapshot); return nil })
}
func WithFindImageMemberHeaders(values map[string]string) FindImageMemberOption {
	apply := mergeImageMutationHeaders(values)
	return request.Custom(func(config *FindImageMemberOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}
func WithListImageMembersOpts(value ListImageMembersOpts) ListImageMembersOption {
	snapshot := copyListImageMembersOpts(value)
	return request.Custom(func(config *ListImageMembersOpts) error { *config = copyListImageMembersOpts(snapshot); return nil })
}
func WithListImageMembersHeaders(values map[string]string) ListImageMembersOption {
	apply := mergeImageMutationHeaders(values)
	return request.Custom(func(config *ListImageMembersOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}

var (
	imageMemberCalls       = calls.Spec[ImageMemberOpts]{Header: calls.MergeHeader[ImageMemberOpts](WithImageMemberHeaders)}
	removeImageMemberCalls = calls.Spec[RemoveImageMemberOpts]{
		Header:        calls.MergeHeader[RemoveImageMemberOpts](WithRemoveImageMemberHeaders),
		IgnoreMissing: calls.Pointer(func(c *RemoveImageMemberOpts) **bool { return &c.IgnoreMissing }),
	}
	findImageMemberCalls = calls.Spec[FindImageMemberOpts]{
		Header:        calls.MergeHeader[FindImageMemberOpts](WithFindImageMemberHeaders),
		IgnoreMissing: calls.Pointer(func(c *FindImageMemberOpts) **bool { return &c.IgnoreMissing }),
	}
	listImageMembersCalls = calls.Spec[ListImageMembersOpts]{
		Header: calls.MergeHeader[ListImageMembersOpts](WithListImageMembersHeaders),
		Paging: calls.Paging[ListImageMembersOpts]{MaxItems: calls.Value(func(c *ListImageMembersOpts) *int { return &c.MaxItems })},
	}
)

// applyImageMemberHeader merges headers with one image mutation header option.
func applyImageMemberHeader(headers *map[string]string, apply func(*ImageMutationOpts) error) error {
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
	config, err := applyImageMemberOptions(ImageMemberOpts{}, calls.Translate(options, imageMemberCalls), copyImageMemberOpts)
	if err != nil {
		return config, err
	}
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, err
}
func parseRemoveImageMemberOptions(options []RemoveImageMemberOption) (RemoveImageMemberOpts, error) {
	config, err := applyImageMemberOptions(RemoveImageMemberOpts{}, calls.Translate(options, removeImageMemberCalls), copyRemoveImageMemberOpts)
	if err != nil {
		return config, err
	}
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, err
}
func parseFindImageMemberOptions(options []FindImageMemberOption) (FindImageMemberOpts, error) {
	config, err := applyImageMemberOptions(FindImageMemberOpts{}, calls.Translate(options, findImageMemberCalls), copyFindImageMemberOpts)
	if err != nil {
		return config, err
	}
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, err
}
func parseListImageMembersOptions(options []ListImageMembersOption) (ListImageMembersOpts, error) {
	config, err := applyImageMemberOptions(ListImageMembersOpts{}, calls.Translate(options, listImageMembersCalls), copyListImageMembersOpts)
	if err != nil {
		return config, err
	}
	if config.MaxItems < 0 {
		return config, uploadInvalid("max items must be non-negative")
	}
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, err
}
