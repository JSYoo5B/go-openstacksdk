package image

import "encoding/json"

// ImageUploadOpts owns flat metadata overrides and ordinary headers for a new
// direct upload. Size is sent only on the binary PUT, including an explicit zero.
type ImageUploadOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
	Size    *int64
}

type ImageUploadOption func(*ImageUploadOpts) error

// WithImageUploadOpts snapshots and replaces the complete upload policy.
func WithImageUploadOpts(value ImageUploadOpts) ImageUploadOption {
	snapshot := copyImageUploadOpts(value)
	return func(config *ImageUploadOpts) error { *config = copyImageUploadOpts(snapshot); return nil }
}
func WithImageUploadHeader(key, value string) ImageUploadOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *ImageUploadOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithImageUploadHeaders(values map[string]string) ImageUploadOption {
	apply := WithImageMutationHeaders(values)
	return func(config *ImageUploadOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithImageUploadSize(value int64) ImageUploadOption {
	return func(config *ImageUploadOpts) error { owned := value; config.Size = &owned; return nil }
}
func WithoutImageUploadSize() ImageUploadOption {
	return func(config *ImageUploadOpts) error { config.Size = nil; return nil }
}

// WithImageUploadField snapshots standard Go JSON once at factory time. It
// replaces one literal field; name belongs exclusively to UploadImageRequest.
func WithImageUploadField(key string, value any) ImageUploadOption {
	var encoded json.RawMessage
	err := validateImageUploadField(key)
	if err == nil {
		encoded, err = marshalImagePatchField(key, value)
	}
	return func(config *ImageUploadOpts) error {
		if err != nil {
			return err
		}
		if config.Fields == nil {
			config.Fields = make(map[string]json.RawMessage)
		}
		config.Fields[key] = copyImagePatchRaw(encoded)
		return nil
	}
}

// WithImageUploadFields replaces all overrides. Nil and empty maps restore the
// default metadata fields; supplied raw values remain independent snapshots.
func WithImageUploadFields(values map[string]json.RawMessage) ImageUploadOption {
	snapshot := copyImagePatchMap(values)
	return func(config *ImageUploadOpts) error { config.Fields = copyImagePatchMap(snapshot); return nil }
}
func WithImageUploadDiskFormat(value string) ImageUploadOption {
	return WithImageUploadField("disk_format", value)
}
func WithImageUploadContainerFormat(value string) ImageUploadOption {
	return WithImageUploadField("container_format", value)
}
func WithImageUploadVisibility(value Visibility) ImageUploadOption {
	return WithImageUploadField("visibility", value)
}
func WithImageUploadOwner(value string) ImageUploadOption {
	return WithImageUploadField("owner", value)
}
func WithImageUploadProtected(value bool) ImageUploadOption {
	return WithImageUploadField("protected", value)
}
func WithImageUploadHidden(value bool) ImageUploadOption {
	return WithImageUploadField("os_hidden", value)
}
func WithImageUploadMinDisk(value int64) ImageUploadOption {
	return WithImageUploadField("min_disk", value)
}
func WithImageUploadMinRAM(value int64) ImageUploadOption {
	return WithImageUploadField("min_ram", value)
}
func WithImageUploadTags(values ...string) ImageUploadOption {
	return WithImageUploadField("tags", append([]string{}, values...))
}
func copyImageUploadOpts(value ImageUploadOpts) ImageUploadOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	value.Fields = copyImagePatchMap(value.Fields)
	if value.Fields == nil {
		value.Fields = make(map[string]json.RawMessage)
	}
	value.Size = copyCreateImportPointer(value.Size)
	return value
}
