package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ImageMemberRecordRequest selects one literal member ID or a Record identity.
// A present Resource.id takes precedence over alternate member_id, including
// explicit null. Other record fields do not become fresh request attributes.
// The image parent remains a separate required reference.
type ImageMemberRecordRequest struct {
	ID     string
	Record *ImageMemberRecord
}

// AddImageMemberRecord creates a fresh Member using supplied declared raw
// attributes. An absent member attribute is permitted and sends flat POST {};
// required fields, enum/schema validation and sharing policy are server-owned.
func (s *Service) AddImageMemberRecord(ctx context.Context, parent resource.Ref, options ...ImageMemberRecordWriteOption) (*ImageMemberRecord, error) {
	return s.writeImageMemberRecord(ctx, parent, "", slices.Clone(options), false)
}

// UpdateImageMemberRecord constructs a fresh Member from the selected ID and
// supplied attributes, rather than committing an existing mutable instance.
// Bound member_id is dirty even with no attributes, so it always sends PUT.
// Direct id may select the route but is excluded from the PUT body.
func (s *Service) UpdateImageMemberRecord(ctx context.Context, parent resource.Ref, input ImageMemberRecordRequest, options ...ImageMemberRecordWriteOption) (*ImageMemberRecord, error) {
	owned := slices.Clone(options)
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrapImageMutationError(ctx, "UpdateImageMemberRecord", err)
	}
	identity, err := imageMemberRecordRequestIdentity(input)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "UpdateImageMemberRecord", err)
	}
	return s.writeImageMemberRecord(ctx, parent, identity, owned, true)
}

func (s *Service) writeImageMemberRecord(ctx context.Context, parent resource.Ref, identity string, options []ImageMemberRecordWriteOption, update bool) (*ImageMemberRecord, error) {
	operation, method := "AddImageMemberRecord", http.MethodPost
	if update {
		operation, method = "UpdateImageMemberRecord", http.MethodPut
	}
	fail := func(record *ImageMemberRecord, err error) (*ImageMemberRecord, error) {
		return record, wrapImageMutationError(ctx, operation, err)
	}
	var seed map[string]json.RawMessage
	route := identity
	p, err := s.prepareImageMemberRecord(ctx, parent, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		attributes, headers, err := prepareImageMemberRecordWrite(opctx, check, options, update)
		if err != nil {
			return nil, err
		}
		seed = attributes
		if update {
			seed["member_id"], _ = json.Marshal(identity)
			if raw, present := seed["id"]; present {
				if !utf8.Valid(raw) {
					return nil, uploadInvalid("member route identity must be UTF-8")
				}
				var replacement string
				if err := json.Unmarshal(raw, &replacement); err != nil {
					return nil, errors.Join(uploadInvalid("member route identity must be a string"), err)
				}
				if err := validateImageMemberRecordIdentity(replacement); err != nil {
					return nil, err
				}
				route = replacement
			}
		}
		return headers, check(opctx)
	})
	if err != nil {
		return fail(nil, err)
	}
	submitted, err := imageMemberRecordSubmitted(seed, p.id, p.location)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return fail(nil, err)
	}
	body := make(map[string]json.RawMessage)
	for _, field := range imageMemberRecordFields {
		if update && field.canonical == "id" {
			continue
		}
		if raw, present := seed[field.canonical]; present {
			body[field.wire] = bytes.Clone(raw)
		}
	}
	var child *string
	if update {
		child = &route
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, p.client, p.check, method, imageMemberEndpoint(p.preparedImageMutation, child), body, nil,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	return finishImageMemberRecord(ctx, operation, p, seed, submitted, response, err)
}

// GetImageMemberRecord performs one literal member GET without discovery.
// Supplied records select an ID only; the fresh Member's raw seed is member_id.
func (s *Service) GetImageMemberRecord(ctx context.Context, parent resource.Ref, input ImageMemberRecordRequest, options ...ImageMemberOption) (*ImageMemberRecord, error) {
	const operation = "GetImageMemberRecord"
	fail := func(err error) (*ImageMemberRecord, error) { return nil, wrapImageMutationError(ctx, operation, err) }
	owned := slices.Clone(options)
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	identity, err := imageMemberRecordRequestIdentity(input)
	if err != nil {
		return fail(err)
	}
	p, err := s.prepareImageMemberRecord(ctx, parent, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		return prepareImageMemberRecordHeaders(opctx, check, owned)
	})
	if err != nil {
		return fail(err)
	}
	rawID, _ := json.Marshal(identity)
	seed := map[string]json.RawMessage{"member_id": rawID}
	submitted, err := imageMemberRecordSubmitted(seed, p.id, p.location)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return fail(err)
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, p.client, p.check, http.MethodGet, imageMemberEndpoint(p.preparedImageMutation, &identity), nil, nil,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	return finishImageMemberRecord(ctx, operation, p, seed, submitted, response, err)
}

// RemoveImageMemberRecord deletes one fixed member and returns the actual
// opaque acknowledgement for accepted200..399. Default ignore-missing only
// suppresses a clean final native404; handling, source and callback failures
// remain errors. An accepted failure can return partial acknowledgement bytes.
func (s *Service) RemoveImageMemberRecord(ctx context.Context, parent resource.Ref, input ImageMemberRecordRequest, options ...RemoveImageMemberOption) (*ImageMemberAcknowledgement, error) {
	const operation = "RemoveImageMemberRecord"
	fail := func(err error) (*ImageMemberAcknowledgement, error) {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	owned := slices.Clone(options)
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	identity, err := imageMemberRecordRequestIdentity(input)
	if err != nil {
		return fail(err)
	}
	var policy RemoveImageMemberOpts
	p, err := s.prepareImageMemberRecord(ctx, parent, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		var err error
		policy, err = prepareImageMemberRecordRemove(opctx, check, owned)
		return policy.Headers, err
	})
	if err != nil {
		return fail(err)
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, p.client, p.check, http.MethodDelete, imageMemberEndpoint(p.preparedImageMutation, &identity), nil, nil,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if guardErr := p.check(p.ctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	if response == nil {
		if native, clean := err.(gophercloud.ErrUnexpectedResponseCode); clean && native.Actual == http.StatusNotFound && (policy.IgnoreMissing == nil || *policy.IgnoreMissing) {
			return nil, nil
		}
		return fail(err)
	}
	ack := &ImageMemberAcknowledgement{ImageID: p.id, MemberID: identity, Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	return ack, wrapImageMutationError(ctx, operation, err)
}

func imageMemberRecordRequestIdentity(input ImageMemberRecordRequest) (string, error) {
	if input.ID != "" && input.Record != nil {
		return "", uploadInvalid("select member ID or Record, not both")
	}
	identity := input.ID
	if input.Record != nil {
		if input.Record.Resource == nil {
			return "", uploadInvalid("member Record Resource is required")
		}
		fields := input.Record.Resource.Clone().Body
		raw, present := fields["id"]
		if !present {
			raw = fields["member_id"]
		}
		if !utf8.Valid(raw) {
			return "", uploadInvalid("member Record identity must be UTF-8")
		}
		if err := json.Unmarshal(raw, &identity); err != nil {
			return "", errors.Join(uploadInvalid("member Record identity must be a string"), err)
		}
	}
	if err := validateImageMemberRecordIdentity(identity); err != nil {
		return "", err
	}
	return identity, nil
}

func imageMemberRecordSubmitted(fields map[string]json.RawMessage, imageID string, location json.RawMessage) (*ImageMemberRecord, error) {
	view, err := projectImageMemberRecord(fields, imageID, location, resource.Metadata{})
	if err != nil {
		return nil, err
	}
	parent := imageID
	return &ImageMemberRecord{Resource: view, ImageID: &parent}, nil
}

func finishImageMemberRecord(ctx context.Context, operation string, p *preparedImageMemberRecord, seed map[string]json.RawMessage, submitted *ImageMemberRecord, response *rest.Response, err error) (*ImageMemberRecord, error) {
	fail := func(record *ImageMemberRecord, err error) (*ImageMemberRecord, error) {
		return record, wrapImageMutationError(ctx, operation, err)
	}
	if response == nil {
		return fail(nil, err)
	}
	if err != nil {
		return fail(imageMemberRecordReceipt(submitted, response), err)
	}
	record, err := imageMemberRecordFromResponse(p.ctx, p.check, seed, p.id, p.location, response)
	if err != nil {
		return fail(imageMemberRecordReceipt(submitted, response), err)
	}
	return record, nil
}

// Translation failure retains the submitted view and actual passive receipt
// separately. It never substitutes an accepted error for a clean record.
func imageMemberRecordReceipt(submitted *ImageMemberRecord, response *rest.Response) *ImageMemberRecord {
	parent := *submitted.ImageID
	record := &ImageMemberRecord{Resource: submitted.Resource.Clone(), ImageID: &parent,
		Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	record.Resource.Header = response.Header.Clone()
	record.Resource.StatusCode = response.StatusCode
	if json.Valid(response.Body) {
		wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		if json.Unmarshal(response.Body, wire) == nil {
			record.Wire = wire
		}
	}
	return record
}
