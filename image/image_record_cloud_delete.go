package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

const (
	imageRecordObjectKey      = "owner_specified.openstack.object"
	imageRecordShadeObjectKey = "owner_specified.shade.object"
)

// ImageRecordCloudDeleteOpts controls Cloud delete_image. Wait enables the
// post-delete absence loop, using the same Timeout, Unlimited and PollInterval
// rules as WaitForCloudImageRecord. Headers apply to Glance requests only.
type ImageRecordCloudDeleteOpts struct {
	Headers      map[string]string
	Wait         bool
	Timeout      *time.Duration
	Unlimited    bool
	PollInterval *time.Duration
}
type ImageRecordCloudDeleteOption func(*ImageRecordCloudDeleteOpts) error

func copyImageRecordCloudDelete(value ImageRecordCloudDeleteOpts) ImageRecordCloudDeleteOpts {
	wait := copyImageRecordCloudWait(ImageRecordCloudWaitOpts{Headers: value.Headers, Timeout: value.Timeout, PollInterval: value.PollInterval})
	value.Headers, value.Timeout, value.PollInterval = wait.Headers, wait.Timeout, wait.PollInterval
	return value
}
func WithImageRecordCloudDeleteOpts(value ImageRecordCloudDeleteOpts) ImageRecordCloudDeleteOption {
	owned := copyImageRecordCloudDelete(value)
	return func(config *ImageRecordCloudDeleteOpts) error {
		*config = copyImageRecordCloudDelete(owned)
		return nil
	}
}
func WithImageRecordCloudDeleteHeader(key, value string) ImageRecordCloudDeleteOption {
	return func(config *ImageRecordCloudDeleteOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordCloudDeleteHeaders(values map[string]string) ImageRecordCloudDeleteOption {
	owned := copyImageRecordHeaders(values)
	return func(config *ImageRecordCloudDeleteOpts) error { return mergeImageRecordHeaders(&config.Headers, owned) }
}
func WithImageRecordCloudDeleteWait(value bool) ImageRecordCloudDeleteOption {
	return func(config *ImageRecordCloudDeleteOpts) error { config.Wait = value; return nil }
}
func WithImageRecordCloudDeleteTimeout(value time.Duration) ImageRecordCloudDeleteOption {
	return func(config *ImageRecordCloudDeleteOpts) error {
		owned := value
		config.Timeout, config.Unlimited = &owned, false
		return nil
	}
}
func WithImageRecordCloudDeleteUnlimited() ImageRecordCloudDeleteOption {
	return func(config *ImageRecordCloudDeleteOpts) error {
		config.Timeout, config.Unlimited = nil, true
		return nil
	}
}
func WithImageRecordCloudDeletePollInterval(value time.Duration) ImageRecordCloudDeleteOption {
	return func(config *ImageRecordCloudDeleteOpts) error {
		owned := value
		config.PollInterval = &owned
		return nil
	}
}

func prepareImageRecordCloudDelete(ctx context.Context, check func(context.Context) error, options []ImageRecordCloudDeleteOption) (ImageRecordCloudDeleteOpts, ImageRecordCloudWaitOpts, error) {
	value := copyImageRecordCloudDelete(ImageRecordCloudDeleteOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return value, ImageRecordCloudWaitOpts{}, err
		}
		if apply == nil {
			return value, ImageRecordCloudWaitOpts{}, uploadInvalid("nil image cloud delete option")
		}
		owned := copyImageRecordCloudDelete(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return value, ImageRecordCloudWaitOpts{}, err
		}
		value = copyImageRecordCloudDelete(owned)
	}
	// The wait policy validation and header rules are shared with
	// WaitForCloudImageRecord, applied once to an owned snapshot.
	wait, err := prepareImageRecordCloudWait(ctx, check, []ImageRecordCloudWaitOption{WithImageRecordCloudWaitOpts(ImageRecordCloudWaitOpts{
		Headers: value.Headers, Timeout: value.Timeout, Unlimited: value.Unlimited, PollInterval: value.PollInterval,
	})})
	value.Headers = wait.Headers
	return value, wait, err
}

// ImageRecordCloudDeleteResult keeps each Cloud delete_image phase. Deleted is
// the Source boolean: false only when the initial find selected nothing.
type ImageRecordCloudDeleteResult struct {
	Deleted     bool
	Found       *ImageRecord
	Delete      *ImageRecordDeleteResult
	Object      *objects.DeleteObjectResult
	WaitLookups int
	WaitLast    *ImageRecord
}

// cloudImageRecordUploadedObject evaluates the Source Task-upload cleanup
// condition on image.properties and splits the stored "container/name" value.
// Python raises TypeError/AttributeError/ValueError after the image delete for
// null properties, non-string values or a value without a slash.
func cloudImageRecordUploadedObject(record *ImageRecord) (string, string, bool, error) {
	raw := record.Resource.Body["properties"]
	present, err := pythonContains(raw, imageRecordObjectKey)
	if err == nil && !present {
		// Python's `or` evaluates the shade key only after a miss.
		present, err = pythonContains(raw, imageRecordShadeObjectKey)
	}
	if err != nil || !present {
		return "", "", false, err
	}
	var properties map[string]json.RawMessage
	if err := json.Unmarshal(raw, &properties); err != nil {
		return "", "", false, uploadInvalid("image Task object properties must be a dictionary")
	}
	value, present := properties[imageRecordObjectKey]
	if !present {
		value = properties[imageRecordShadeObjectKey]
	}
	if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return "", "", false, uploadInvalid("image Task object location is null")
	}
	location, err := decodeImageRecordString(value, "image Task object location")
	if err != nil {
		return "", "", false, err
	}
	container, name, found := strings.Cut(location, "/")
	if !found {
		return "", "", false, uploadInvalid("image Task object location %q has no container separator", location)
	}
	return container, name, true, nil
}

// DeleteCloudImageRecord implements Cloud delete_image: an ignore-missing
// find, the owned whole-image delete of the found record, Task-upload Swift
// object cleanup when image_api_use_tasks is truthy and the found properties
// name an object, then an optional find loop until the image is absent. The
// pinned source accepts delete_objects but never reads it, so Go omits it.
func (s *Service) DeleteCloudImageRecord(ctx context.Context, nameOrID string, options ...ImageRecordCloudDeleteOption) (*ImageRecordCloudDeleteResult, error) {
	const operation = "DeleteCloudImageRecord"
	fail := func(result *ImageRecordCloudDeleteResult, err error) (*ImageRecordCloudDeleteResult, error) {
		return result, wrapImageMutationError(ctx, operation, err)
	}
	if err := validateImageRecordIdentity(nameOrID); err != nil {
		return fail(nil, err)
	}
	owned := slices.Clone(options)
	var policy ImageRecordCloudDeleteOpts
	var waitPolicy ImageRecordCloudWaitOpts
	var useTasks bool
	p, err := s.prepareImageRecord(ctx, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		// image_api_use_tasks is captured from cloud configuration once.
		cloud, err := copyImageCreatePolicy(opctx, check, s.dependencies.CreatePolicy)
		if err == nil && cloud.UseTasks != nil {
			useTasks, err = imageCreateTruthy(cloud.UseTasks)
		}
		if err != nil {
			return nil, err
		}
		policy, waitPolicy, err = prepareImageRecordCloudDelete(opctx, check, owned)
		return policy.Headers, err
	})
	if err != nil {
		return fail(nil, err)
	}
	parameters, err := prepareImageRecordList(p.ctx, p.check, nil)
	if err != nil {
		return fail(nil, err)
	}
	found, err := findPreparedImageRecord(p, nameOrID, FindImageRecordOpts{}, parameters)
	if err != nil {
		return fail(nil, err)
	}
	result := &ImageRecordCloudDeleteResult{Found: found}
	if found == nil {
		return result, nil
	}
	result.Delete, err = s.DeleteImageRecord(ctx, ImageRecordDeleteRequest{Record: found}, WithImageRecordDeleteHeaders(policy.Headers))
	if err != nil {
		return result, renameImageOperation(err, operation)
	}
	if useTasks {
		container, name, present, err := cloudImageRecordUploadedObject(found)
		if err != nil {
			receipt := &rest.Response{Body: found.Envelope, Header: found.Header, StatusCode: found.StatusCode}
			return fail(result, receipt.Fail(err))
		}
		if present {
			if s.dependencies.ObjectStorage == nil {
				return fail(result, uploadInvalid("image Task object cleanup requires an object-store service"))
			}
			service, err := s.dependencies.ObjectStorage(p.ctx)
			if err = errors.Join(err, p.check(p.ctx)); err != nil {
				return fail(result, err)
			}
			if service == nil || service.Objects == nil {
				return fail(result, uploadInvalid("image Task object cleanup requires an object-store service"))
			}
			result.Object, err = service.Objects.DeleteObject(p.ctx, container, name)
			if err = errors.Join(err, p.check(p.ctx)); err != nil {
				return fail(result, err)
			}
		}
	}
	if policy.Wait {
		id, err := decodeImageRecordString(found.Resource.Body["id"], "deleted image id")
		if err == nil {
			err = validateImageRecordIdentity(id)
		}
		if err != nil {
			return fail(result, err)
		}
		result.WaitLookups, result.WaitLast, err = cloudImageRecordPoll(p, id, waitPolicy, "to be deleted", func(record *ImageRecord) (bool, error) {
			return record == nil, nil
		})
		if err != nil {
			return fail(result, err)
		}
	}
	result.Deleted = true
	return result, nil
}
