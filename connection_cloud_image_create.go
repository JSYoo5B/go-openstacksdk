package openstack

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
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CloudImageCreateRequest mirrors Cloud create_image. An empty VolumeID uses
// the Glance CreateImageRecord workflow with Image; otherwise the volume branch
// uses Image.Name with the Cinder Proxy create_image. PollInterval is a Go
// extension for the Cloud wait loop and its timeout cleanup; zero means the
// fixed two-second iterate_timeout interval.
type CloudImageCreateRequest struct {
	Image        image.ImageRecordCreateRequest
	VolumeID     string
	PollInterval time.Duration
}

// CloudImageCreateResult keeps every Cloud create_image phase. Image is the
// Source return value: the created record without wait, or the first found
// record whose status is neither queued nor saving.
type CloudImageCreateResult struct {
	Image       *image.ImageRecord
	Created     *image.ImageRecordCreateResult
	Volume      *VolumeImageCreateResult
	WaitLookups int
	WaitLast    *image.ImageRecord
	Cleanup     *image.ImageRecordCloudDeleteResult
}

// cloudCreateTimeout converts Source timeout values used by iterate_timeout:
// nil is the 3600 second default, null waits forever, numbers and Python bools
// are seconds. Other JSON values raise TypeError in Python.
func cloudCreateTimeout(raw json.RawMessage) (time.Duration, bool, error) {
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

// cloudCreateFormat applies the Cinder Proxy `if not value` default check to a
// raw create option; truthy values must be strings for the typed volume action.
func cloudCreateFormat(raw json.RawMessage, label string) (string, error) {
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

func cloudCreateTruthy(raw json.RawMessage) (bool, error) {
	if raw == nil {
		return false, nil
	}
	return cloudfilter.PythonTruthy(raw)
}

// CreateCloudImageRecord implements Cloud create_image. Without a volume it
// runs CreateImageRecord with the same options; with a volume it runs the
// Cinder Proxy create_image using the allow_duplicates and format options.
// When wait is truthy it then repeats Cloud get_image on the created id until
// a found status is neither queued nor saving. On timeout it runs Cloud
// delete_image(id, wait=True) and returns the timeout with that cleanup.
func (c *Connection) CreateCloudImageRecord(ctx context.Context, input CloudImageCreateRequest, options ...image.ImageRecordCreateOption) (*CloudImageCreateResult, error) {
	const operation = "CreateCloudImageRecord"
	fail := func(result *CloudImageCreateResult, err error) (*CloudImageCreateResult, error) {
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
	wait, err := cloudCreateTruthy(policy.Wait)
	if err != nil {
		return fail(nil, err)
	}
	timeout, unlimited, err := cloudCreateTimeout(policy.Timeout)
	if err != nil && wait {
		return fail(nil, err)
	}
	service, err := c.Image(ctx)
	if err != nil {
		return fail(nil, err)
	}
	result := &CloudImageCreateResult{}
	if input.VolumeID == "" {
		result.Created, err = service.CreateImageRecord(ctx, input.Image, image.WithImageRecordCreateOpts(policy))
		if result.Created != nil {
			result.Image = result.Created.Record
		}
	} else {
		request := VolumeImageCreateRequest{Name: input.Image.Name, VolumeID: input.VolumeID}
		request.AllowDuplicates, err = cloudCreateTruthy(policy.AllowDuplicates)
		if err == nil {
			request.ContainerFormat, err = cloudCreateFormat(policy.ContainerFormat, "container format")
		}
		if err == nil {
			request.DiskFormat, err = cloudCreateFormat(policy.DiskFormat, "disk format")
		}
		if err != nil {
			return fail(nil, err)
		}
		result.Volume, err = c.CreateVolumeImageRecord(ctx, request)
		if result.Volume != nil {
			result.Image = result.Volume.Image
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
