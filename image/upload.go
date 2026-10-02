package image

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"reflect"
	"strings"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
)

// Visibility identifies who can use an image.
type Visibility = images.ImageVisibility

const (
	VisibilityPrivate   = images.ImageVisibilityPrivate
	VisibilityPublic    = images.ImageVisibilityPublic
	VisibilityShared    = images.ImageVisibilityShared
	VisibilityCommunity = images.ImageVisibilityCommunity
)

// UploadImageRequest creates image metadata and uploads Data through Glance's
// direct-upload API. Data is read from its current position and remains owned
// by the caller, including when it implements io.Closer. Upload never closes it.
type UploadImageRequest struct {
	Name string
	Data io.Reader
}

type uploadImageOptions struct {
	base        images.CreateOpts
	properties  map[string]json.RawMessage
	minDisk     *int
	minRAM      *int
	size        *int64
	wait        bool
	waitOptions []resource.WaitOption
}

// UploadImageOption configures a direct image-upload workflow.
type UploadImageOption func(*uploadImageOptions) error

// WithDiskFormat overrides the default qcow2 format. Glance validates supported
// formats; the SDK does not infer the format from the data or a filename.
func WithDiskFormat(format string) UploadImageOption {
	return func(o *uploadImageOptions) error {
		if strings.TrimSpace(format) == "" {
			return uploadInvalid("disk format must not be empty")
		}
		o.base.DiskFormat = format
		return nil
	}
}

// WithContainerFormat overrides the default bare container format.
func WithContainerFormat(format string) UploadImageOption {
	return func(o *uploadImageOptions) error {
		if strings.TrimSpace(format) == "" {
			return uploadInvalid("container format must not be empty")
		}
		o.base.ContainerFormat = format
		return nil
	}
}

func WithVisibility(visibility Visibility) UploadImageOption {
	return func(o *uploadImageOptions) error {
		switch visibility {
		case VisibilityPrivate, VisibilityPublic, VisibilityShared, VisibilityCommunity:
		default:
			return uploadInvalid("unknown image visibility %q", visibility)
		}
		o.base.Visibility = &visibility
		return nil
	}
}

// WithMinDisk sets the minimum boot disk size in GB, including an explicit zero.
func WithMinDisk(gigabytes int) UploadImageOption {
	return func(o *uploadImageOptions) error {
		if gigabytes < 0 {
			return uploadInvalid("minimum disk size must not be negative")
		}
		o.minDisk = &gigabytes
		return nil
	}
}

// WithMinRAM sets the minimum boot memory in MB, including an explicit zero.
func WithMinRAM(megabytes int) UploadImageOption {
	return func(o *uploadImageOptions) error {
		if megabytes < 0 {
			return uploadInvalid("minimum RAM must not be negative")
		}
		o.minRAM = &megabytes
		return nil
	}
}

func WithProtected(enabled bool) UploadImageOption {
	return func(o *uploadImageOptions) error { o.base.Protected = &enabled; return nil }
}

func WithHidden(enabled bool) UploadImageOption {
	return func(o *uploadImageOptions) error { o.base.Hidden = &enabled; return nil }
}

// WithUploadSize sends X-OpenStack-Image-Size so Glance can preallocate storage.
// The value is the data size in bytes, not the image's virtual disk size. The
// SDK does not buffer the input or seek to its end to discover the size.
func WithUploadSize(bytes int64) UploadImageOption {
	return func(o *uploadImageOptions) error {
		if bytes < 0 {
			return uploadInvalid("upload size must not be negative")
		}
		o.size = &bytes
		return nil
	}
}

// WithTags snapshots the supplied tags. Glance validates their schema.
func WithTags(tags ...string) UploadImageOption {
	tags = append([]string(nil), tags...)
	return func(o *uploadImageOptions) error { o.base.Tags = append([]string(nil), tags...); return nil }
}

// WithProperty snapshots a JSON value for an additional image property. Core
// fields cannot be replaced. Glance validates the property's schema.
func WithProperty(key string, value any) UploadImageOption {
	return WithProperties(map[string]any{key: value})
}

// WithProperties snapshots JSON property values and merges them at the root of
// Glance's unwrapped metadata request. Later options replace the same property.
func WithProperties(properties map[string]any) UploadImageOption {
	snapshot := make(map[string]json.RawMessage, len(properties))
	var snapshotErr error
	for key, value := range properties {
		if strings.TrimSpace(key) == "" || reservedImageProperties[key] {
			snapshotErr = uploadInvalid("image property %q is empty or conflicts with a core field", key)
			break
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			snapshotErr = uploadInvalid("image property %q is not JSON serializable: %v", key, err)
			break
		}
		snapshot[key] = encoded
	}
	return func(o *uploadImageOptions) error {
		if snapshotErr != nil {
			return snapshotErr
		}
		maps.Copy(o.properties, snapshot)
		return nil
	}
}

// WithWait waits for active after upload. The shared five-minute timeout and
// two-second interval apply unless overridden. Options are checked before POST.
func WithWait(options ...resource.WaitOption) UploadImageOption {
	options = append([]resource.WaitOption(nil), options...)
	return func(o *uploadImageOptions) error {
		if err := resource.ValidateWaitOptionsFor[images.Image](options...); err != nil {
			return err
		}
		o.wait = true
		o.waitOptions = options
		return nil
	}
}

var reservedImageProperties = func() map[string]bool {
	reserved := map[string]bool{"size": true, "self": true, "properties": true, "os_hash_algo": true, "os_hash_value": true}
	for _, typ := range []reflect.Type{reflect.TypeOf(images.CreateOpts{}), reflect.TypeOf(Image{})} {
		for i := 0; i < typ.NumField(); i++ {
			key := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
			if key != "" && key != "-" {
				reserved[key] = true
			}
		}
	}
	return reserved
}()

type uploadMetadata struct{ options uploadImageOptions }

// prepareUploadMetadata shares the existing concrete metadata builder with
// workflows that freeze the JSON request before starting another phase.
func prepareUploadMetadata(options uploadImageOptions) (map[string]any, error) {
	return (uploadMetadata{options: options}).ToImageCreateMap()
}

func (b uploadMetadata) ToImageCreateMap() (map[string]any, error) {
	body, err := b.options.base.ToImageCreateMap()
	if err != nil {
		return nil, err
	}
	if b.options.minDisk != nil {
		body["min_disk"] = *b.options.minDisk
	}
	if b.options.minRAM != nil {
		body["min_ram"] = *b.options.minRAM
	}
	return request.MergeFieldsFor(body, b.options.properties, b.options.base)
}

// Upload creates metadata, streams Data via PUT /images/{id}/file, and optionally
// waits for active. It defaults to qcow2/bare and private visibility. Without
// WithWait, the returned model is the metadata creation response, whose status
// may still be queued. Upload itself waits for the PUT request to finish.
// Upload and waiter failures return the created image alongside a wrapped cause;
// the SDK never deletes it automatically. The caller owns and closes Data.
// The binary PUT uses one attempt without reauthentication or retry callbacks.
// Its 401/429 errors retain the created image; use API.ImageData.Upload to retry
// an existing image after refreshing authentication or rewinding the input.
func (s *Service) Upload(ctx context.Context, input UploadImageRequest, options ...UploadImageOption) (*Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, uploadInvalid("image name must not be empty")
	}
	if input.Data == nil || isNilReader(input.Data) {
		return nil, uploadInvalid("image data reader must not be nil")
	}
	visibility := images.ImageVisibilityPrivate
	o := uploadImageOptions{
		base:       images.CreateOpts{Name: input.Name, DiskFormat: "qcow2", ContainerFormat: "bare", Visibility: &visibility},
		properties: make(map[string]json.RawMessage),
	}
	for _, apply := range options {
		if apply == nil {
			return nil, uploadInvalid("nil upload option")
		}
		if err := apply(&o); err != nil {
			return nil, err
		}
	}
	metadata := uploadMetadata{options: o}
	if _, err := prepareUploadMetadata(o); err != nil {
		return nil, uploadWrap("prepare upload", err)
	}
	created, err := images.Create(ctx, s.client, metadata).Extract()
	if err != nil {
		if created != nil && created.ID != "" {
			return created, uploadWrap("create metadata", err)
		}
		return nil, uploadWrap("create metadata", err)
	}
	if err := resource.ID(created.ID).Validate(); err != nil {
		return created, uploadWrap("upload", err)
	}
	if err := s.uploadDataOnce(ctx, created.ID, input.Data, o.size); err != nil {
		return created, uploadWrap("upload data", err)
	}
	if !o.wait {
		return created, nil
	}
	ready, err := s.Images.Wait(ctx, resource.ID(created.ID), "active", o.waitOptions...)
	if err != nil {
		return created, uploadWrap("upload/wait", err)
	}
	return ready, nil
}

func isNilReader(reader io.Reader) bool {
	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}

func uploadInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

func uploadWrap(operation string, err error) error {
	return &resource.OperationError{Operation: operation, Resource: "image", Cause: err}
}

var _ images.CreateOptsBuilder = uploadMetadata{}
