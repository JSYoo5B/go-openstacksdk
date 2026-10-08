package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type preparedImageRecordTag struct {
	*preparedImageRecord
	seed *ImageRecord
	id   string
}

// AddImageRecordTag sends one fixed single-tag PUT and appends the tag to a
// private local Image only after accepted HTTP completes. Duplicate local tags
// remain duplicates. Source Proxy returns None and TagMixin returns self; this
// owned result exposes the local view and actual acknowledgement separately.
func (s *Service) AddImageRecordTag(ctx context.Context, input ImageRecordTagRequest, tag string, options ...ImageMutationOption) (*ImageRecordTagResult, error) {
	return s.mutateImageRecordTag(ctx, input, tag, slices.Clone(options), false)
}

// RemoveImageRecordTag sends one fixed single-tag DELETE and removes the first
// matching local tag after accepted HTTP. An absent local tag is a no-op; an
// HTTP rejection, including a missing server tag, remains an error.
func (s *Service) RemoveImageRecordTag(ctx context.Context, input ImageRecordTagRequest, tag string, options ...ImageMutationOption) (*ImageRecordTagResult, error) {
	return s.mutateImageRecordTag(ctx, input, tag, slices.Clone(options), true)
}

func (s *Service) prepareImageRecordTag(ctx context.Context, input ImageRecordTagRequest, tag string, options []ImageMutationOption) (*preparedImageRecordTag, error) {
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return nil, err
	}
	if input.ID != "" && input.Record != nil {
		return nil, uploadInvalid("select image ID or Record, not both")
	}
	seed := cloneImageRecord(input.Record)
	identity := input.ID
	if input.Record != nil {
		if seed.Resource == nil {
			return nil, uploadInvalid("image tag seed Resource is required")
		}
		if err := json.Unmarshal(seed.Resource.Body["id"], &identity); err != nil {
			return nil, errors.Join(uploadInvalid("image tag seed id must be a string"), err)
		}
	}
	if err := validateImageRecordIdentity(identity); err != nil {
		return nil, err
	}
	if err := validateImageRecordTag(tag); err != nil {
		return nil, err
	}
	// Every caller channel is already privately owned when current-location or
	// ordinary-header callbacks run. The operation's fixed identity cannot move.
	if err := p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		return prepareImageRecordTagHeaders(opctx, check, options)
	}); err != nil {
		return nil, err
	}
	if seed == nil {
		rawID, _ := json.Marshal(identity)
		view, err := projectImageRecord(map[string]json.RawMessage{"id": rawID}, p.location, resource.Metadata{})
		if err != nil {
			return nil, err
		}
		seed = &ImageRecord{Resource: view, ImportMethods: make([]string, 0),
			bodyState: newImageRecordBodyState(map[string]json.RawMessage{"id": rawID})}
	}
	if err := p.check(p.ctx); err != nil {
		return nil, err
	}
	return &preparedImageRecordTag{preparedImageRecord: p, seed: seed, id: identity}, nil
}

func prepareImageRecordTagHeaders(ctx context.Context, check func(context.Context) error, options []ImageMutationOption) (map[string]string, error) {
	config := copyImageMutationOpts(ImageMutationOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, uploadInvalid("nil image mutation option")
		}
		candidate := copyImageMutationOpts(config)
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return nil, err
		}
		config = copyImageMutationOpts(candidate)
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	return headers, errors.Join(err, check(ctx))
}

// Source urljoin strips outer slashes and does not validate tag strings. The Go
// profile instead keeps any space, slash, backslash or length as literal text
// in one escaped segment. Empty, exact dot segments, controls and invalid UTF-8
// cannot select a safe literal endpoint; these are explicit Go URI bounds.
func validateImageRecordTag(tag string) error {
	if tag == "" || !utf8.ValidString(tag) || tag == "." || tag == ".." {
		return uploadInvalid("image tag must be nonempty valid UTF-8 literal text")
	}
	for _, char := range tag {
		if unicode.IsControl(char) {
			return uploadInvalid("image tag must not contain controls")
		}
	}
	return nil
}

func (s *Service) mutateImageRecordTag(ctx context.Context, input ImageRecordTagRequest, tag string, options []ImageMutationOption, removing bool) (*ImageRecordTagResult, error) {
	operation, method := "AddImageRecordTag", http.MethodPut
	if removing {
		operation, method = "RemoveImageRecordTag", http.MethodDelete
	}
	fail := func(result *ImageRecordTagResult, err error) (*ImageRecordTagResult, error) {
		return result, wrapImageMutationError(ctx, operation, err)
	}
	p, err := s.prepareImageRecordTag(ctx, input, tag, options)
	if err != nil {
		return fail(nil, err)
	}
	endpoint := imageRecordEndpoint(p.preparedImageRecord, p.id) + "/tags/" + url.PathEscape(tag)
	response, err := rest.DoJSONGuardedRejections(p.ctx, p.client, p.check, method, endpoint, nil, nil,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if response == nil {
		return fail(nil, err)
	}
	result := &ImageRecordTagResult{Acknowledgement: &ImageRecordTagAcknowledgement{
		ImageID: p.id, Tag: tag, Body: bytes.Clone(response.Body),
		Header: response.Header.Clone(), StatusCode: response.StatusCode,
	}}
	if err != nil {
		return fail(result, err)
	}
	if err := p.check(p.ctx); err != nil {
		return fail(result, response.Fail(err))
	}
	record := cloneImageRecord(p.seed)
	if err := mutateImageRecordTags(record, tag, removing); err != nil {
		return fail(result, response.Fail(err))
	}
	if err := p.check(p.ctx); err != nil {
		return fail(result, response.Fail(err))
	}
	result.Record = record
	return result, nil
}

// TagMixin reads only tags after HTTP; unrelated fields and all fetch/Wire
// receipts remain unchanged. Its list descriptor wraps non-list scalar/object
// values, preserves untyped members, defaults missing tags to [] and keeps null
// as null, whose append/remove is a post-acknowledgement local model failure.
func mutateImageRecordTags(record *ImageRecord, tag string, removing bool) error {
	raw, present := record.Resource.Body["tags"]
	if !present {
		raw = json.RawMessage("[]")
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return uploadInvalid("image tags must be complete UTF-8 JSON")
	}
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return uploadInvalid("image tags must not be null for a local tag mutation")
	}
	var tags []json.RawMessage
	if trimmed[0] == '[' {
		if err := json.Unmarshal(raw, &tags); err != nil {
			return errors.Join(uploadInvalid("image tags list could not be decoded"), err)
		}
	} else {
		tags = []json.RawMessage{bytes.Clone(raw)}
	}
	if removing {
		for index, raw := range tags {
			item := bytes.TrimSpace(raw)
			if len(item) == 0 || item[0] != '"' || !imageRecordTagUnicodeString(item) {
				continue
			}
			var text string
			if err := json.Unmarshal(item, &text); err != nil {
				return errors.Join(uploadInvalid("image tag string could not be decoded"), err)
			}
			if text == tag {
				tags = append(tags[:index], tags[index+1:]...)
				break
			}
		}
	} else {
		rawTag, _ := json.Marshal(tag)
		tags = append(tags, rawTag)
	}
	// Rebuild only the list delimiters. Existing untyped member bytes, including
	// exact large numbers and nested objects, remain passive owned JSON values.
	updated := json.RawMessage{'['}
	for index, raw := range tags {
		if index != 0 {
			updated = append(updated, ',')
		}
		updated = append(updated, raw...)
	}
	updated = append(updated, ']')
	record.Resource.Body["tags"] = updated
	if record.bodyState != nil {
		// TagMixin updates raw attributes directly; it does not reset the original
		// commit baseline or create a Resource dirty flag.
		record.bodyState.current["tags"] = bytes.Clone(updated)
	}
	return nil
}

func imageRecordTagRejectionCodes() []int {
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = 400 + index
	}
	return codes
}

// encoding/json replaces an unpaired UTF-16 surrogate with U+FFFD. Python
// retains the surrogate, which cannot equal a valid UTF-8 literal tag. Keep
// such passive members unmatched rather than removing a different string.
// Valid pairs and escaped backslash literals still use json.Unmarshal below.
// The caller has already validated a complete JSON string; this checks only
// its Unicode escape pairing, without parsing or normalizing JSON values.
func imageRecordTagUnicodeString(raw json.RawMessage) bool {
	for index := 1; index < len(raw)-1; index++ {
		if raw[index] != '\\' {
			continue
		}
		index++
		if raw[index] != 'u' {
			continue
		}
		unit, _ := strconv.ParseUint(string(raw[index+1:index+5]), 16, 16)
		index += 4
		if unit >= 0xD800 && unit <= 0xDBFF {
			if index+7 > len(raw)-1 || raw[index+1] != '\\' || raw[index+2] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(raw[index+3:index+7]), 16, 16)
			if err != nil || low < 0xDC00 || low > 0xDFFF {
				return false
			}
			index += 6
		} else if unit >= 0xDC00 && unit <= 0xDFFF {
			return false
		}
	}
	return true
}
