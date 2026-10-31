// Package cloudimage implements the image-side helpers of Connection: the
// Cloud create_image workflow (openstack/cloud/_image.py create_image with its
// wait and timeout cleanup), the guarded Glance service resolution shared by
// the Connection image record and member record methods, and the Swift source
// registration used by the configured image create Task branch. Connection
// supplies its lazy services, provider and volume image create through
// Dependencies, so this package never imports the root.
package cloudimage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvolume"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	objectapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// Dependencies are the Connection facilities the image helpers use. The zero
// value stands for a nil Connection.
type Dependencies struct {
	// Image is the lazy shared Glance service.
	Image func(context.Context) (*image.Service, error)
	// ObjectStorage is the lazy shared Swift service.
	ObjectStorage func(context.Context) (*objectapi.Service, error)
	// Provider returns the Connection's current provider.
	Provider func() *gophercloud.ProviderClient
}

// CreateRequest mirrors the root CloudImageCreateRequest field for field.
type CreateRequest struct {
	Image        image.ImageRecordCreateRequest
	VolumeID     string
	PollInterval time.Duration
}

// CreateResult mirrors the root CloudImageCreateResult field for field; V is
// the root volume image create result type, which stays generic because the
// root result's Volume field must keep the root pointer type for a direct
// conversion.
type CreateResult[V any] struct {
	Image       *image.ImageRecord
	Created     *image.ImageRecordCreateResult
	Volume      V
	WaitLookups int
	WaitLast    *image.ImageRecord
	Cleanup     *image.ImageRecordCloudDeleteResult
}

// CreateVolume runs the Cinder Proxy create_image and returns its result
// together with the image record it carries (nil when it carries none). The
// request is the cloudvolume mirror of the root VolumeImageCreateRequest.
type CreateVolume[V any] func(context.Context, cloudvolume.ImageCreateRequest) (V, *image.ImageRecord, error)

func invalid(message string) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
}

// Service checks the Connection and context before resolving the shared
// Glance service, wrapping failures under operation.
func Service(ctx context.Context, deps Dependencies, operation string) (*image.Service, error) {
	fail := func(err error) (*image.Service, error) {
		return nil, &resource.OperationError{Resource: "image", Operation: operation, Cause: err}
	}
	if ctx == nil || deps.Image == nil {
		return fail(fmt.Errorf("%w: Connection and context are required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return fail(errors.Join(err, context.Cause(ctx)))
	}
	service, err := deps.Image(ctx)
	if err != nil {
		return fail(err)
	}
	return service, nil
}

// SwiftService resolves Swift for the configured Task branch only. Source-only
// registration lets the Glance operation observe later Swift binding changes
// without recursion.
func SwiftService(ctx context.Context, deps Dependencies) (*objectapi.Service, error) {
	service, err := deps.ObjectStorage(ctx)
	if err != nil {
		return nil, err
	}
	if service == nil || service.RawClient() == nil || service.Containers == nil || service.Objects == nil {
		return nil, invalid("image task Swift service is required")
	}
	client, containers, objects, provider := service.RawClient(), service.Containers, service.Objects, deps.Provider()
	rest.RegisterOperationSource(ctx, func(context.Context) error {
		if deps.Provider() != provider || service.RawClient() != client || service.Containers != containers || service.Objects != objects || containers.RawClient() != client || objects.RawClient() != client {
			return invalid("image task Swift service source changed")
		}
		return nil
	})
	return service, nil
}

// createTimeout converts Source timeout values used by iterate_timeout: nil is
// the 3600 second default, null waits forever, numbers and Python bools are
// seconds. Other JSON values raise TypeError in Python.
func createTimeout(raw json.RawMessage) (time.Duration, bool, error) {
	if raw == nil {
		return 3600 * time.Second, false, nil
	}
	switch trimmed := bytes.TrimSpace(raw); string(trimmed) {
	case "null":
		return 0, true, nil
	case "true":
		return time.Second, false, nil
	case "false":
		return 0, false, nil
	}
	var seconds float64
	if err := json.Unmarshal(raw, &seconds); err != nil || math.IsNaN(seconds) {
		return 0, false, fmt.Errorf("%w: Cloud image create timeout must be a number or null", resource.ErrInvalidOption)
	}
	if seconds > float64(math.MaxInt64)/float64(time.Second) {
		return 0, true, nil
	}
	return time.Duration(seconds * float64(time.Second)), false, nil
}

// createFormat applies the Cinder Proxy `if not value` default check to a raw
// create option; truthy values must be strings for the typed volume action.
func createFormat(raw json.RawMessage, label string) (string, error) {
	if raw == nil {
		return "", nil
	}
	truthy, err := cloudfilter.PythonTruthy(raw)
	if err != nil || !truthy {
		return "", err
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%w: Cloud volume image %s must be a string", resource.ErrInvalidOption, label)
	}
	return value, nil
}

func createTruthy(raw json.RawMessage) (bool, error) {
	if raw == nil {
		return false, nil
	}
	return cloudfilter.PythonTruthy(raw)
}

// Create implements Cloud create_image. Without a volume it runs
// CreateImageRecord with the same options; with a volume it runs the Cinder
// Proxy create_image using the allow_duplicates and format options. When wait
// is truthy it then repeats Cloud get_image on the created id until a found
// status is neither queued nor saving. On timeout it runs Cloud
// delete_image(id, wait=True) and returns the timeout with that cleanup.
func Create[V any](ctx context.Context, deps Dependencies, createVolume CreateVolume[V], input CreateRequest, options ...image.ImageRecordCreateOption) (*CreateResult[V], error) {
	const operation = "CreateCloudImageRecord"
	fail := func(result *CreateResult[V], err error) (*CreateResult[V], error) {
		if err != nil && ctx != nil && ctx.Err() != nil && !errors.Is(err, context.Cause(ctx)) {
			err = errors.Join(err, context.Cause(ctx))
		}
		return result, request.Wrap(operation, "image", err)
	}
	if ctx == nil {
		return fail(nil, fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if input.PollInterval < 0 {
		return fail(nil, fmt.Errorf("%w: Cloud image create poll interval must not be negative", resource.ErrInvalidOption))
	}
	policy, err := image.PrepareImageRecordCreateOptions(ctx, options...)
	if err != nil {
		return fail(nil, err)
	}
	wait, err := createTruthy(policy.Wait)
	if err != nil {
		return fail(nil, err)
	}
	timeout, unlimited, err := createTimeout(policy.Timeout)
	if err != nil && wait {
		return fail(nil, err)
	}
	service, err := deps.Image(ctx)
	if err != nil {
		return fail(nil, err)
	}
	result := &CreateResult[V]{}
	if input.VolumeID == "" {
		result.Created, err = service.CreateImageRecord(ctx, input.Image, image.WithImageRecordCreateOpts(policy))
		if result.Created != nil {
			result.Image = result.Created.Record
		}
	} else {
		request := cloudvolume.ImageCreateRequest{Name: input.Image.Name, VolumeID: input.VolumeID}
		request.AllowDuplicates, err = createTruthy(policy.AllowDuplicates)
		if err == nil {
			request.ContainerFormat, err = createFormat(policy.ContainerFormat, "container format")
		}
		if err == nil {
			request.DiskFormat, err = createFormat(policy.DiskFormat, "disk format")
		}
		if err != nil {
			return fail(nil, err)
		}
		var volumeImage *image.ImageRecord
		result.Volume, volumeImage, err = createVolume(ctx, request)
		if volumeImage != nil {
			result.Image = volumeImage
		}
	}
	if err != nil {
		return fail(result, err)
	}
	if !wait {
		return result, nil
	}
	if result.Image == nil || result.Image.Resource == nil {
		return fail(result, fmt.Errorf("%w: Cloud image create returned no image to wait for", resource.ErrInvalidOption))
	}
	var id string
	if err := json.Unmarshal(result.Image.Resource.Body["id"], &id); err != nil || id == "" {
		return fail(result, fmt.Errorf("%w: created image id must be a nonempty string", resource.ErrInvalidOption))
	}
	interval := input.PollInterval
	if interval == 0 {
		interval = 2 * time.Second
	}
	created := result.Image
	deadline := time.Now().Add(timeout)
	for {
		if !unlimited && !time.Now().Before(deadline) {
			break
		}
		result.WaitLookups++
		found, err := service.GetCloudImageRecord(ctx, id)
		if err != nil {
			return fail(result, err)
		}
		if found.Image != nil {
			result.WaitLast = found.Image
			var status string
			raw := found.Image.Resource.Body["status"]
			isString := json.Unmarshal(raw, &status) == nil && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
			if !isString || status != "queued" && status != "saving" {
				result.Image = found.Image
				return result, nil
			}
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fail(result, context.Cause(ctx))
		case <-timer.C:
		}
	}
	timedOut := fmt.Errorf("%w: timeout waiting for the image %q to finish", context.DeadlineExceeded, id)
	cleanupOptions := []image.ImageRecordCloudDeleteOption{image.WithImageRecordCloudDeleteWait(true)}
	if input.PollInterval != 0 {
		cleanupOptions = append(cleanupOptions, image.WithImageRecordCloudDeletePollInterval(input.PollInterval))
	}
	var cleanupErr error
	result.Cleanup, cleanupErr = service.DeleteCloudImageRecord(ctx, id, cleanupOptions...)
	result.Image = created
	// Python raises the cleanup failure instead; Go keeps both causes.
	return fail(result, errors.Join(timedOut, cleanupErr))
}
