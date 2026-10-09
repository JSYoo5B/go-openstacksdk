package image

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecordCloudWaitOpts controls Cloud wait_for_image. Nil Timeout means
// the Source default of 3600 seconds and Unlimited means Python None. A zero
// or negative timeout expires before the first lookup. Nil PollInterval means
// the fixed two-second iterate_timeout wait; a configured interval is a Go
// extension and must be positive.
type ImageRecordCloudWaitOpts struct {
	Headers      map[string]string
	Timeout      *time.Duration
	Unlimited    bool
	PollInterval *time.Duration
}
type ImageRecordCloudWaitOption func(*ImageRecordCloudWaitOpts) error

func copyImageRecordCloudWait(value ImageRecordCloudWaitOpts) ImageRecordCloudWaitOpts {
	value.Headers = copyImageRecordHeaders(value.Headers)
	if value.Timeout != nil {
		owned := *value.Timeout
		value.Timeout = &owned
	}
	if value.PollInterval != nil {
		owned := *value.PollInterval
		value.PollInterval = &owned
	}
	return value
}
func WithImageRecordCloudWaitOpts(value ImageRecordCloudWaitOpts) ImageRecordCloudWaitOption {
	owned := copyImageRecordCloudWait(value)
	return func(config *ImageRecordCloudWaitOpts) error { *config = copyImageRecordCloudWait(owned); return nil }
}
func WithImageRecordCloudWaitHeader(key, value string) ImageRecordCloudWaitOption {
	return func(config *ImageRecordCloudWaitOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordCloudWaitHeaders(values map[string]string) ImageRecordCloudWaitOption {
	owned := copyImageRecordHeaders(values)
	return func(config *ImageRecordCloudWaitOpts) error { return mergeImageRecordHeaders(&config.Headers, owned) }
}
func WithImageRecordCloudWaitTimeout(value time.Duration) ImageRecordCloudWaitOption {
	return func(config *ImageRecordCloudWaitOpts) error {
		owned := value
		config.Timeout, config.Unlimited = &owned, false
		return nil
	}
}
func WithImageRecordCloudWaitUnlimited() ImageRecordCloudWaitOption {
	return func(config *ImageRecordCloudWaitOpts) error { config.Timeout, config.Unlimited = nil, true; return nil }
}
func WithImageRecordCloudWaitPollInterval(value time.Duration) ImageRecordCloudWaitOption {
	return func(config *ImageRecordCloudWaitOpts) error { owned := value; config.PollInterval = &owned; return nil }
}

func prepareImageRecordCloudWait(ctx context.Context, check func(context.Context) error, options []ImageRecordCloudWaitOption) (ImageRecordCloudWaitOpts, error) {
	value := copyImageRecordCloudWait(ImageRecordCloudWaitOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, uploadInvalid("nil image cloud wait option")
		}
		owned := copyImageRecordCloudWait(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return value, err
		}
		value = copyImageRecordCloudWait(owned)
	}
	if value.Unlimited && value.Timeout != nil {
		return value, uploadInvalid("image cloud wait cannot be unlimited and have a timeout")
	}
	if value.PollInterval != nil && *value.PollInterval <= 0 {
		return value, uploadInvalid("image cloud wait poll interval must be positive")
	}
	headers, err := imageMutationHeaders(value.Headers, false, "")
	value.Headers = headers
	return value, errors.Join(err, check(ctx))
}

// ImageRecordCloudWaitResult retains the last found record and the number of
// lookups. Image is the active record only when the wait succeeds.
type ImageRecordCloudWaitResult struct {
	Image   *ImageRecord
	Last    *ImageRecord
	Lookups int
}

// imageRecordStatusIs is Source `image_obj['status'] == value`: only an exact
// JSON string compares equal; null and non-string values keep polling.
func imageRecordStatusIs(record *ImageRecord, value string) bool {
	status, err := decodeImageRecordString(record.Resource.Body["status"], "Cloud wait image status")
	return err == nil && status == value
}

// WaitForCloudImageRecord implements Cloud wait_for_image. It reads the
// supplied record's string id, then repeats the Proxy find_image lookup with
// ignore_missing true: missing images and other statuses keep polling, exact
// "active" succeeds and exact "error" fails. Lookup errors stop immediately.
// The deadline is checked before each lookup and an in-flight lookup is not
// cancelled by it, matching iterate_timeout.
func (s *Service) WaitForCloudImageRecord(ctx context.Context, image *ImageRecord, options ...ImageRecordCloudWaitOption) (*ImageRecordCloudWaitResult, error) {
	const operation = "WaitForCloudImageRecord"
	fail := func(result *ImageRecordCloudWaitResult, err error) (*ImageRecordCloudWaitResult, error) {
		return result, wrapImageMutationError(ctx, operation, err)
	}
	if image == nil || image.Resource == nil {
		return fail(nil, uploadInvalid("image wait record Resource is required"))
	}
	id, err := decodeImageRecordString(image.Resource.Body["id"], "image wait id")
	if err == nil {
		err = validateImageRecordIdentity(id)
	}
	if err != nil {
		return fail(nil, err)
	}
	owned := slices.Clone(options)
	var policy ImageRecordCloudWaitOpts
	p, err := s.prepareImageRecord(ctx, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		var err error
		policy, err = prepareImageRecordCloudWait(opctx, check, owned)
		return policy.Headers, err
	})
	if err != nil {
		return fail(nil, err)
	}
	parameters, err := prepareImageRecordList(p.ctx, p.check, nil)
	if err != nil {
		return fail(nil, err)
	}
	timeout, interval := 3600*time.Second, 2*time.Second
	if policy.Timeout != nil {
		timeout = *policy.Timeout
	}
	if policy.PollInterval != nil {
		interval = *policy.PollInterval
	}
	result := &ImageRecordCloudWaitResult{}
	lastFailure := func(err error) (*ImageRecordCloudWaitResult, error) {
		if last := result.Last; last != nil {
			var accepted *resource.ResponseError
			if !errors.As(err, &accepted) {
				err = (&rest.Response{Body: last.Envelope, Header: last.Header, StatusCode: last.StatusCode}).Fail(err)
			}
		}
		return fail(result, err)
	}
	deadline := time.Now().Add(timeout)
	for {
		if err := p.check(p.ctx); err != nil {
			return lastFailure(err)
		}
		if !policy.Unlimited && !time.Now().Before(deadline) {
			return lastFailure(fmt.Errorf("%w: timeout waiting for image %q to snapshot", context.DeadlineExceeded, id))
		}
		result.Lookups++
		record, err := findPreparedImageRecord(p, id, FindImageRecordOpts{}, parameters)
		if err != nil {
			return fail(result, err)
		}
		if record != nil {
			result.Last = record
			if imageRecordStatusIs(record, "active") {
				result.Image = record
				return result, nil
			}
			if imageRecordStatusIs(record, "error") {
				return lastFailure(&resource.FailedStateError{Resource: "images", ID: id, Status: "error"})
			}
		}
		timer := time.NewTimer(interval)
		select {
		case <-p.ctx.Done():
			timer.Stop()
			err := p.check(p.ctx)
			if err == nil {
				err = context.Cause(p.ctx)
			}
			return lastFailure(err)
		case <-timer.C:
		}
	}
}
