package image

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/pollschedule"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/sourcewait"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ImageRecordCloudWaitOpts controls Cloud wait_for_image. Wait holds shared
// resource wait options: WithTimeout (default 3600 seconds), WithUnlimitedWait
// (Python None), and WithPollInterval or WithWaitStrategy, which replace the
// fixed two-second iterate_timeout pause as a Go extension. A zero or negative
// timeout or poll interval and other shared controls fail with
// resource.ErrInvalidOption before HTTP.
type ImageRecordCloudWaitOpts struct {
	Headers map[string]string
	Wait    []resource.WaitOption
}

// ImageRecordCloudWaitOption is one option of the Cloud image waits:
// WithImageRecordCloudWaitOpts, WithImageRecordCloudWaitHeaders (merges),
// WithImageRecordCloudWait, request.Header
// (merges one header) or request.Custom over *ImageRecordCloudWaitOpts. Other
// shared calls, including request.Headers, fail with resource.ErrInvalidOption
// before HTTP.
type ImageRecordCloudWaitOption = request.GetCall

var imageRecordCloudWaitCalls = calls.Spec[ImageRecordCloudWaitOpts]{
	Header: calls.MergeHeader[ImageRecordCloudWaitOpts](WithImageRecordCloudWaitHeaders),
}

func copyImageRecordCloudWait(value ImageRecordCloudWaitOpts) ImageRecordCloudWaitOpts {
	value.Headers = copyImageRecordHeaders(value.Headers)
	value.Wait = slices.Clone(value.Wait)
	return value
}
func WithImageRecordCloudWaitOpts(value ImageRecordCloudWaitOpts) ImageRecordCloudWaitOption {
	owned := copyImageRecordCloudWait(value)
	return request.Custom(func(config *ImageRecordCloudWaitOpts) error { *config = copyImageRecordCloudWait(owned); return nil })
}
func WithImageRecordCloudWaitHeaders(values map[string]string) ImageRecordCloudWaitOption {
	owned := copyImageRecordHeaders(values)
	return request.Custom(func(config *ImageRecordCloudWaitOpts) error { return mergeImageRecordHeaders(&config.Headers, owned) })
}

// WithImageRecordCloudWait appends shared resource wait options to Wait: the
// timeout, unlimited, poll interval and wait strategy controls. Repeated calls
// append; for each control the later option wins.
func WithImageRecordCloudWait(options ...resource.WaitOption) ImageRecordCloudWaitOption {
	owned := slices.Clone(options)
	return request.Custom(func(config *ImageRecordCloudWaitOpts) error {
		config.Wait = append(slices.Clone(config.Wait), owned...)
		return nil
	})
}

// imageRecordCloudWaitPolicy is a validated Cloud wait policy.
type imageRecordCloudWaitPolicy struct {
	Headers   map[string]string
	Timeout   time.Duration
	Unlimited bool
	Strategy  resource.WaitStrategy
}

// resolveImageRecordCloudWait resolves the shared options over the defaults.
func resolveImageRecordCloudWait(options []resource.WaitOption) (imageRecordCloudWaitPolicy, error) {
	settings, err := sourcewait.Resolve(0, options)
	if err != nil {
		return imageRecordCloudWaitPolicy{}, err
	}
	policy := imageRecordCloudWaitPolicy{Timeout: 3600 * time.Second, Strategy: resource.FixedWait(2 * time.Second)}
	if settings.TimeoutSet {
		policy.Timeout, policy.Unlimited = settings.Timeout, settings.Unlimited
	}
	if settings.StrategySet {
		policy.Strategy = settings.Strategy
	}
	return policy, nil
}

func prepareImageRecordCloudWait(ctx context.Context, check func(context.Context) error, options []ImageRecordCloudWaitOption) (imageRecordCloudWaitPolicy, error) {
	value := copyImageRecordCloudWait(ImageRecordCloudWaitOpts{})
	for _, apply := range calls.Translate(options, imageRecordCloudWaitCalls) {
		if err := check(ctx); err != nil {
			return imageRecordCloudWaitPolicy{}, err
		}
		if apply == nil {
			return imageRecordCloudWaitPolicy{}, uploadInvalid("nil image cloud wait option")
		}
		owned := copyImageRecordCloudWait(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return imageRecordCloudWaitPolicy{}, err
		}
		value = copyImageRecordCloudWait(owned)
	}
	policy, err := resolveImageRecordCloudWait(value.Wait)
	if err != nil {
		return policy, err
	}
	policy.Headers, err = imageMutationHeaders(value.Headers, false, "")
	return policy, errors.Join(err, check(ctx))
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
	var policy imageRecordCloudWaitPolicy
	p, err := s.prepareImageRecord(ctx, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		var err error
		policy, err = prepareImageRecordCloudWait(opctx, check, owned)
		return policy.Headers, err
	})
	if err != nil {
		return fail(nil, err)
	}
	result := &ImageRecordCloudWaitResult{}
	result.Lookups, result.Last, err = cloudImageRecordPoll(p, id, policy, "to snapshot", func(record *ImageRecord) (bool, error) {
		if record == nil {
			return false, nil
		}
		if imageRecordStatusIs(record, "active") {
			result.Image = record
			return true, nil
		}
		if imageRecordStatusIs(record, "error") {
			return false, &resource.FailedStateError{Resource: "images", ID: id, Status: "error"}
		}
		return false, nil
	})
	if err != nil {
		return fail(result, err)
	}
	return result, nil
}

// cloudImageRecordPoll is the Cloud iterate_timeout loop around repeated
// ignore-missing find_image lookups. The deadline is checked before each
// lookup, an in-flight lookup is not cancelled by it and the full interval is
// slept after every unfinished lookup. Errors after a found record carry that
// record's actual receipt unless they already hold newer response evidence.
func cloudImageRecordPoll(p *preparedImageRecord, id string, policy imageRecordCloudWaitPolicy, purpose string, done func(*ImageRecord) (bool, error)) (int, *ImageRecord, error) {
	lookups := 0
	var last *ImageRecord
	lastFailure := func(err error) (int, *ImageRecord, error) {
		var accepted *resource.ResponseError
		var rejected gophercloud.ErrUnexpectedResponseCode
		if last != nil && !errors.As(err, &accepted) && !errors.As(err, &rejected) {
			err = (&rest.Response{Body: last.Envelope, Header: last.Header, StatusCode: last.StatusCode}).Fail(err)
		}
		return lookups, last, err
	}
	parameters, err := prepareImageRecordList(p.ctx, p.check, nil)
	if err != nil {
		return lookups, last, err
	}
	pacer := pollschedule.NewPacer(policy.Strategy.Delay)
	deadline := time.Now().Add(policy.Timeout)
	for {
		if err := p.check(p.ctx); err != nil {
			return lastFailure(err)
		}
		if !policy.Unlimited && !time.Now().Before(deadline) {
			return lastFailure(fmt.Errorf("%w: timeout waiting for image %q %s", context.DeadlineExceeded, id, purpose))
		}
		lookups++
		record, err := findPreparedImageRecord(p, id, FindImageRecordOpts{}, parameters)
		if err != nil {
			return lookups, last, err
		}
		if record != nil {
			last = record
		}
		finished, err := done(record)
		if err != nil {
			return lastFailure(err)
		}
		if finished {
			return lookups, last, nil
		}
		timer := time.NewTimer(pacer.Next())
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
