package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// imageRecordWaitObservation keeps Python lower() comparisons outside the
// shared string adapter. The shared engine sees only an SDK-owned sentinel.
type imageRecordWaitObservation struct {
	Status string
	record *ImageRecord
}

const imageRecordWaitReached = "image-record-wait-reached"
const imageRecordWaitPending = "image-record-wait-pending"

type preparedImageRecordWait struct {
	*preparedImageRecord
	seed   *ImageRecord
	body   map[string]json.RawMessage
	id     string
	config ImageRecordWaitOpts
	last   *ImageRecord
}

func (s *Service) prepareImageRecordWaitInput(ctx context.Context, seed *ImageRecord, options []ImageRecordWaitOption, deleting bool) (*preparedImageRecordWait, error) {
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return nil, err
	}
	owned := cloneImageRecord(seed)
	if owned == nil || owned.Resource == nil {
		return nil, uploadInvalid("image wait seed Resource is required")
	}
	identity, err := decodeImageRecordString(owned.Resource.Body["id"], "image wait seed id")
	if err != nil {
		return nil, err
	}
	if err := validateImageRecordIdentity(identity); err != nil {
		return nil, err
	}
	body := imageRecordBodySnapshot(owned)
	body, _, err = normalizeImageRecord(body, nil, false, false)
	if err != nil {
		return nil, err
	}
	result := &preparedImageRecordWait{preparedImageRecord: p, seed: owned, body: body, id: identity}
	err = p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		config, err := prepareImageRecordWait(opctx, check, options, deleting)
		result.config = config
		if err != nil {
			return nil, err
		}
		// Validate all finite descriptor inputs before HTTP. Preserve the input
		// record's own channels for cached-target and first404 return values.
		location := owned.Resource.Body["location"]
		if location == nil {
			location = p.location
		}
		if _, err := projectImageRecord(body, location, owned.Resource.Metadata); err != nil {
			return nil, err
		}
		return config.Headers, check(opctx)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// A null descriptor is distinct from a string containing zero characters.
func imageRecordWaitState(value *ImageRecord, attribute string) (string, bool, error) {
	var raw json.RawMessage
	if attribute == "image_import_methods" {
		raw, _ = json.Marshal(value.ImportMethods)
	} else {
		raw = value.Resource.Body[attribute]
	}
	if raw == nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", true, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return "", false, errors.Join(uploadInvalid("image wait attribute %q must be a string or null", attribute), err)
	}
	return text, false, nil
}

func (p *preparedImageRecordWait) initialState() (string, bool, error) {
	location := p.seed.Resource.Body["location"]
	if location == nil {
		location = p.location
	}
	view, err := projectImageRecord(p.body, location, p.seed.Resource.Metadata)
	if err != nil {
		return "", false, err
	}
	value := *p.seed
	value.Resource = view
	return imageRecordWaitState(&value, p.config.Attribute)
}

func (p *preparedImageRecordWait) fetch(ctx context.Context, target string, deleting bool) (*imageRecordWaitObservation, error) {
	if err := p.check(ctx); err != nil {
		return nil, err
	}
	response, err := rest.DoJSONGuardedRejections(ctx, p.client, p.check, http.MethodGet, imageRecordEndpoint(p.preparedImageRecord, p.id), nil, nil,
		rest.RejectionPolicy{Codes: []int{http.StatusNotFound}, PreserveCleanRetry: true}, imageRecordCodes()...)
	if err != nil {
		return nil, err
	}
	value, err := imageRecordFromResponse(ctx, p.check, p.body, p.location, response)
	if err != nil {
		return nil, err
	}
	value.data = p.seed.data
	p.last = value
	p.body = imageRecordBodySnapshot(value)
	state, null, err := imageRecordWaitState(value, p.config.Attribute)
	if err != nil {
		return nil, response.Fail(err)
	}
	if deleting && null {
		return nil, response.Fail(uploadInvalid("image deletion wait status must be a string"))
	}
	observed := &imageRecordWaitObservation{Status: imageRecordWaitPending, record: value}
	if !null && cloudfilter.PythonLower(state) == cloudfilter.PythonLower(target) {
		observed.Status = imageRecordWaitReached
		if deleting {
			observed.Status = "deleted"
		}
		return observed, nil
	}
	if !deleting && !null {
		failures := p.config.FailureStates
		if failures == nil {
			failures = []string{"ERROR"}
		}
		for _, failure := range failures {
			if cloudfilter.PythonLower(state) == cloudfilter.PythonLower(failure) {
				return nil, response.Fail(&resource.FailedStateError{Resource: "images", ID: p.id, Status: state})
			}
		}
	}
	return observed, p.check(ctx)
}

func (p *preparedImageRecordWait) collection(target string, deleting bool) *resource.Collection[imageRecordWaitObservation] {
	return resource.NewCollection(resource.Adapter[imageRecordWaitObservation]{
		Kind: "images", ValidateID: validateImageRecordIdentity,
		ID:        func(*imageRecordWaitObservation) string { return p.id },
		Status:    func(value *imageRecordWaitObservation) string { return value.Status },
		WaitGuard: p.check,
		Get: func(ctx context.Context, _ string) (*imageRecordWaitObservation, error) {
			return p.fetch(ctx, target, deleting)
		},
	})
}

func (p *preparedImageRecordWait) failure(ctx context.Context, operation string, err error) (*ImageRecord, error) {
	// Budget/progress/guard failures retain the last actual accepted receipt.
	// A later physical failure already has newer evidence and must keep it.
	var accepted *resource.ResponseError
	var rejected gophercloud.ErrUnexpectedResponseCode
	if p.last != nil && !errors.As(err, &accepted) && !errors.As(err, &rejected) {
		err = (&rest.Response{Body: bytes.Clone(p.last.Envelope), Header: p.last.Header.Clone(), StatusCode: p.last.StatusCode}).Fail(err)
	}
	return cloneImageRecord(p.last), wrapImageMutationError(ctx, operation, err)
}

// WaitForImageRecordStatus first checks the supplied Image descriptor and then
// polls fresh fixed-ID GETs. The seed and all receipts are independently owned.
// After a successfully projected GET, a later error returns the last record
// with that error; only a nil error indicates the requested transition completed.
func (s *Service) WaitForImageRecordStatus(ctx context.Context, seed *ImageRecord, target string, options ...ImageRecordWaitOption) (*ImageRecord, error) {
	const operation = "WaitForImageRecordStatus"
	if !utf8.ValidString(target) {
		return nil, wrapImageMutationError(ctx, operation, uploadInvalid("image wait target must be valid UTF-8"))
	}
	owned := slices.Clone(options)
	p, err := s.prepareImageRecordWaitInput(ctx, seed, owned, false)
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	initial, null, err := p.initialState()
	if err != nil {
		return p.failure(ctx, operation, err)
	}
	if !null && cloudfilter.PythonLower(initial) == cloudfilter.PythonLower(target) {
		if err := p.check(p.ctx); err != nil {
			return p.failure(ctx, operation, err)
		}
		return cloneImageRecord(p.seed), nil
	}
	policy, expired, err := imageRecordWaitPolicy(p.config, false)
	if err != nil {
		return p.failure(ctx, operation, err)
	}
	if expired {
		return p.failure(ctx, operation, context.DeadlineExceeded)
	}
	observation, err := p.collection(target, false).Wait(p.ctx, resource.ID(p.id), imageRecordWaitReached, resource.WithWaitPolicy(policy))
	if err != nil {
		return p.failure(ctx, operation, err)
	}
	if err := p.check(p.ctx); err != nil {
		return p.failure(ctx, operation, err)
	}
	return cloneImageRecord(observation.record), nil
}

// WaitForImageRecordDelete always starts with a GET, even for a cached deleted
// status. Clean404 returns the original seed or the last successful overlay.
// It observes deletion without sending DELETE; late errors retain the last
// successfully projected record and its actual HTTP receipt.
func (s *Service) WaitForImageRecordDelete(ctx context.Context, seed *ImageRecord, options ...ImageRecordWaitOption) (*ImageRecord, error) {
	const operation = "WaitForImageRecordDelete"
	p, err := s.prepareImageRecordWaitInput(ctx, seed, slices.Clone(options), true)
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	policy, expired, err := imageRecordWaitPolicy(p.config, true)
	if err != nil {
		return p.failure(ctx, operation, err)
	}
	if expired {
		return p.failure(ctx, operation, context.DeadlineExceeded)
	}
	err = p.collection("deleted", true).WaitDeleted(p.ctx, resource.ID(p.id), resource.WithWaitPolicy(policy))
	if err != nil {
		return p.failure(ctx, operation, err)
	}
	if err := p.check(p.ctx); err != nil {
		return p.failure(ctx, operation, err)
	}
	if p.last != nil {
		return cloneImageRecord(p.last), nil
	}
	return cloneImageRecord(p.seed), nil
}
