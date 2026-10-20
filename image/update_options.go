package image

import "encoding/json"

// UpdateImageOpts supplies ordered concrete changes and ordinary headers.
type UpdateImageOpts struct {
	Headers map[string]string
	Changes []ImagePatch
}
type UpdateImageOption func(*UpdateImageOpts) error

// SetImagePropertiesOpts supplies literal root values. Nil and empty maps
// produce an empty PATCH; no aliases or Python value conversions are applied.
type SetImagePropertiesOpts struct {
	Headers    map[string]string
	Properties map[string]json.RawMessage
}
type SetImagePropertiesOption func(*SetImagePropertiesOpts) error

func WithUpdateImageOpts(value UpdateImageOpts) UpdateImageOption {
	snapshot := copyUpdateImageOpts(value)
	return func(config *UpdateImageOpts) error { *config = copyUpdateImageOpts(snapshot); return nil }
}
func WithUpdateImageHeader(key, value string) UpdateImageOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *UpdateImageOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithUpdateImageHeaders(values map[string]string) UpdateImageOption {
	apply := WithImageMutationHeaders(values)
	return func(config *UpdateImageOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithUpdateImageChange(change ImagePatch) UpdateImageOption {
	snapshot := copyImagePatch(change)
	return func(config *UpdateImageOpts) error {
		config.Changes = append(config.Changes, copyImagePatch(snapshot))
		return nil
	}
}
func WithUpdateImageChanges(changes ...ImagePatch) UpdateImageOption {
	snapshot := copyImagePatches(changes)
	return func(config *UpdateImageOpts) error { config.Changes = copyImagePatches(snapshot); return nil }
}
func WithUpdateImageName(value string) UpdateImageOption { return WithUpdateImageField("name", value) }
func WithUpdateImageVisibility(value string) UpdateImageOption {
	return WithUpdateImageField("visibility", value)
}
func WithUpdateImageProtected(value bool) UpdateImageOption {
	return WithUpdateImageField("protected", value)
}
func WithUpdateImageHidden(value bool) UpdateImageOption {
	return WithUpdateImageField("os_hidden", value)
}
func WithUpdateImageOwner(value string) UpdateImageOption {
	return WithUpdateImageField("owner", value)
}
func WithUpdateImageContainerFormat(value string) UpdateImageOption {
	return WithUpdateImageField("container_format", value)
}
func WithUpdateImageDiskFormat(value string) UpdateImageOption {
	return WithUpdateImageField("disk_format", value)
}
func WithUpdateImageMinDisk(value int64) UpdateImageOption {
	return WithUpdateImageField("min_disk", value)
}
func WithUpdateImageMinRAM(value int64) UpdateImageOption {
	return WithUpdateImageField("min_ram", value)
}
func WithUpdateImageTags(values ...string) UpdateImageOption {
	snapshot := append([]string{}, values...)
	return WithUpdateImageField("tags", snapshot)
}

// WithUpdateImageField snapshots standard Go JSON at factory time and appends
// one add/upsert. Factory failures surface when the returned option is applied.
func WithUpdateImageField(key string, value any) UpdateImageOption {
	encoded, err := marshalImagePatchField(key, value)
	return func(config *UpdateImageOpts) error {
		if err != nil {
			return err
		}
		config.Changes = append(config.Changes, ImagePatch{Op: "add", Path: imagePatchFieldPath(key), Value: copyImagePatchRaw(encoded)})
		return nil
	}
}
func WithUpdateImageFields(values map[string]json.RawMessage) UpdateImageOption {
	snapshot, err := imagePatchFields(copyImagePatchMap(values))
	return func(config *UpdateImageOpts) error {
		if err != nil {
			return err
		}
		config.Changes = append(config.Changes, copyImagePatches(snapshot)...)
		return nil
	}
}
func WithUpdateImageRemoveField(key string) UpdateImageOption {
	err := validateImagePatchField(key)
	return func(config *UpdateImageOpts) error {
		if err != nil {
			return err
		}
		config.Changes = append(config.Changes, ImagePatch{Op: "remove", Path: imagePatchFieldPath(key)})
		return nil
	}
}
func WithSetImagePropertiesOpts(value SetImagePropertiesOpts) SetImagePropertiesOption {
	snapshot := copySetImagePropertiesOpts(value)
	return func(config *SetImagePropertiesOpts) error { *config = copySetImagePropertiesOpts(snapshot); return nil }
}
func WithSetImagePropertiesHeader(key, value string) SetImagePropertiesOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *SetImagePropertiesOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithSetImagePropertiesHeaders(values map[string]string) SetImagePropertiesOption {
	apply := WithImageMutationHeaders(values)
	return func(config *SetImagePropertiesOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithSetImagePropertiesProperty(key string, value any) SetImagePropertiesOption {
	encoded, err := marshalImagePatchField(key, value)
	return func(config *SetImagePropertiesOpts) error {
		if err != nil {
			return err
		}
		if config.Properties == nil {
			config.Properties = make(map[string]json.RawMessage)
		}
		config.Properties[key] = copyImagePatchRaw(encoded)
		return nil
	}
}
func WithSetImagePropertiesProperties(values map[string]json.RawMessage) SetImagePropertiesOption {
	snapshot := copyImagePatchMap(values)
	return func(config *SetImagePropertiesOpts) error {
		config.Properties = copyImagePatchMap(snapshot)
		return nil
	}
}
func copyUpdateImageOpts(value UpdateImageOpts) UpdateImageOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	value.Changes = copyImagePatches(value.Changes)
	return value
}
func copySetImagePropertiesOpts(value SetImagePropertiesOpts) SetImagePropertiesOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	value.Properties = copyImagePatchMap(value.Properties)
	if value.Properties == nil {
		value.Properties = make(map[string]json.RawMessage)
	}
	return value
}
func parseUpdateImageOptions(options []UpdateImageOption) (UpdateImageOpts, error) {
	value, err := applyTaskOptions(UpdateImageOpts{}, options, copyUpdateImageOpts)
	if err != nil {
		return value, err
	}
	for _, change := range value.Changes {
		if err := validateImagePatch(change); err != nil {
			return value, err
		}
	}
	if value.Changes == nil {
		value.Changes = make([]ImagePatch, 0)
	}
	value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	return value, err
}
func parseSetImagePropertiesOptions(options []SetImagePropertiesOption) (SetImagePropertiesOpts, []ImagePatch, error) {
	value, err := applyTaskOptions(SetImagePropertiesOpts{}, options, copySetImagePropertiesOpts)
	if err != nil {
		return value, nil, err
	}
	changes, err := imagePatchFields(value.Properties)
	if err == nil {
		value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	}
	return value, changes, err
}
