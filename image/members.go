package image

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AddImageMember adds a literal member ID to an image. The server owns sharing
// policy; no image-state prefetch or project-name lookup is performed.
func (s *Service) AddImageMember(ctx context.Context, parent resource.Ref, memberID string, options ...ImageMemberOption) (*ImageMember, error) {
	return s.requestImageMember(ctx, parent, memberID, nil, http.MethodPost, "AddImageMember", options)
}

// GetImageMember fetches one member using the fixed image and member IDs.
func (s *Service) GetImageMember(ctx context.Context, parent resource.Ref, memberID string, options ...ImageMemberOption) (*ImageMember, error) {
	return s.requestImageMember(ctx, parent, memberID, nil, http.MethodGet, "GetImageMember", options)
}

// UpdateImageMember sends exactly pending, accepted or rejected as the status.
func (s *Service) UpdateImageMember(ctx context.Context, parent resource.Ref, memberID, status string, options ...ImageMemberOption) (*ImageMember, error) {
	return s.requestImageMember(ctx, parent, memberID, &status, http.MethodPut, "UpdateImageMember", options)
}

func (s *Service) requestImageMember(ctx context.Context, parent resource.Ref, memberID string, status *string, method, operation string, options []ImageMemberOption) (*ImageMember, error) {
	options = append([]ImageMemberOption(nil), options...)
	prepared, err := s.prepareImageMember(ctx, parent, &memberID, status, func() (map[string]string, error) {
		policy, err := parseImageMemberOptions(options)
		return policy.Headers, err
	})
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	var body any
	endpointID := &memberID
	if method == http.MethodPost {
		body = map[string]string{"member": memberID}
		endpointID = nil
	}
	if method == http.MethodPut {
		body = map[string]string{"status": *status}
	}
	response, err := rest.DoJSON(ctx, prepared.client, method, imageMemberEndpoint(prepared, endpointID), body, nil, http.StatusOK)
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	return decodeImageMemberResponse(ctx, prepared, operation, response)
}

// FindImageMember performs one direct member GET. By default, only a fully
// handled actual 404 returns nil, nil; there is no list or Name fallback.
func (s *Service) FindImageMember(ctx context.Context, parent resource.Ref, memberID string, options ...FindImageMemberOption) (*ImageMember, error) {
	options = append([]FindImageMemberOption(nil), options...)
	var policy FindImageMemberOpts
	prepared, err := s.prepareImageMember(ctx, parent, &memberID, nil, func() (map[string]string, error) {
		var err error
		policy, err = parseFindImageMemberOptions(options)
		return policy.Headers, err
	})
	if err != nil {
		return nil, wrapImageMutationError(ctx, "FindImageMember", err)
	}
	codes := []int{http.StatusOK}
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		codes = append(codes, http.StatusNotFound)
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodGet, imageMemberEndpoint(prepared, &memberID), nil, nil, codes...)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "FindImageMember", err)
	}
	if err := prepared.check(ctx); err != nil {
		return nil, wrapImageMutationError(ctx, "FindImageMember", response.Fail(err))
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return decodeImageMemberResponse(ctx, prepared, "FindImageMember", response)
}

// RemoveImageMember deletes one fixed member. Default missing handling applies
// only to a fully read and closed actual member-route 404, never parent lookup.
func (s *Service) RemoveImageMember(ctx context.Context, parent resource.Ref, memberID string, options ...RemoveImageMemberOption) (*ImageMemberAcknowledgement, error) {
	options = append([]RemoveImageMemberOption(nil), options...)
	var policy RemoveImageMemberOpts
	prepared, err := s.prepareImageMember(ctx, parent, &memberID, nil, func() (map[string]string, error) {
		var err error
		policy, err = parseRemoveImageMemberOptions(options)
		return policy.Headers, err
	})
	if err != nil {
		return nil, wrapImageMutationError(ctx, "RemoveImageMember", err)
	}
	codes := []int{http.StatusNoContent}
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		codes = append(codes, http.StatusNotFound)
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodDelete, imageMemberEndpoint(prepared, &memberID), nil, nil, codes...)
	if err == nil {
		if sourceErr := prepared.check(ctx); sourceErr != nil {
			err = response.Fail(sourceErr)
		}
	}
	return imageMemberAcknowledgement(prepared.id, memberID, response), wrapImageMutationError(ctx, "RemoveImageMember", err)
}

func decodeImageMemberResponse(ctx context.Context, prepared *preparedImageMutation, operation string, response *rest.Response) (*ImageMember, error) {
	if err := prepared.check(ctx); err != nil {
		return nil, wrapImageMutationError(ctx, operation, response.Fail(err))
	}
	var value ImageMember
	if err := json.Unmarshal(response.Body, &value); err != nil {
		return nil, wrapImageMutationError(ctx, operation, response.Fail(err))
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	if err := prepared.check(ctx); err != nil {
		return nil, wrapImageMutationError(ctx, operation, response.Fail(err))
	}
	return &value, nil
}

// ListImageMembers lazily fetches one complete collection response. MaxItems
// limits local row consumption; links and pagination hints remain passive.
func (s *Service) ListImageMembers(ctx context.Context, parent resource.Ref, options ...ListImageMembersOption) iter.Seq2[*ImageMember, error] {
	options = append([]ListImageMembersOption(nil), options...)
	return func(yield func(*ImageMember, error) bool) {
		var policy ListImageMembersOpts
		prepared, err := s.prepareImageMember(ctx, parent, nil, nil, func() (map[string]string, error) {
			var err error
			policy, err = parseListImageMembersOptions(options)
			return policy.Headers, err
		})
		fail := func(err error) { yield(nil, wrapImageMutationError(ctx, "ListImageMembers", err)) }
		if err != nil {
			fail(err)
			return
		}
		response, err := rest.DoJSON(ctx, prepared.client, http.MethodGet, imageMemberEndpoint(prepared, nil), nil, nil, http.StatusOK)
		if err != nil {
			fail(err)
			return
		}
		fields, err := schemaObject(response.Body)
		if err != nil {
			fail(response.Fail(err))
			return
		}
		raw, exists := schemaField(fields, "members")
		if !exists || len(raw) == 0 || raw[0] != '[' {
			fail(response.Fail(fmt.Errorf("image members response requires a nonnull members array")))
			return
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			fail(response.Fail(err))
			return
		}
		for index, raw := range rows {
			if policy.MaxItems > 0 && index >= policy.MaxItems {
				return
			}
			if err := prepared.check(ctx); err != nil {
				fail(response.Fail(err))
				return
			}
			var value ImageMember
			if err := json.Unmarshal(raw, &value); err != nil {
				fail(response.Fail(err))
				return
			}
			value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
			if !yield(&value, nil) {
				return
			}
		}
		if err := prepared.check(ctx); err != nil {
			fail(response.Fail(err))
		}
	}
}

// AllImageMembers collects the finite iterator and discards partial rows on an
// error. A successful empty collection returns a nonnil empty slice.
func (s *Service) AllImageMembers(ctx context.Context, parent resource.Ref, options ...ListImageMembersOption) ([]*ImageMember, error) {
	values := make([]*ImageMember, 0)
	for value, err := range s.ListImageMembers(ctx, parent, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
