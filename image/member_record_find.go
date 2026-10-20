package image

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FindImageMemberRecord attempts the fixed image/member GET first. A clean
// native 400, 403 or 404 falls back to the member list with exact ID-or-name
// matching. Missing is ignored by default; duplicates and list errors remain
// errors. Source defaults, options and location belong to one operation.
func (s *Service) FindImageMemberRecord(ctx context.Context, parent resource.Ref, nameOrID string, options ...FindImageMemberRecordOption) (*ImageMemberRecord, error) {
	const operation = "FindImageMemberRecord"
	fail := func(err error) (*ImageMemberRecord, error) {
		return nil, wrapImageMutationError(ctx, operation, cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	if err := validateImageMemberRecordIdentity(nameOrID); err != nil {
		return fail(err)
	}
	owned := slices.Clone(options)
	var policy FindImageMemberRecordOpts
	p, err := s.prepareImageMemberRecord(ctx, parent, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		var err error
		policy, err = prepareFindImageMemberRecord(opctx, check, owned)
		return policy.Headers, err
	})
	if err != nil {
		return fail(err)
	}
	collection := resource.NewCollection(resource.Adapter[ImageMemberRecord]{
		Kind: "image members", IdentityFind: true,
		ValidateID:        validateImageMemberRecordIdentity,
		IdentityDirectGet: validateImageMemberRecordIdentity,
		Get: func(readCtx context.Context, identity string) (*ImageMemberRecord, error) {
			codes := make([]int, 200)
			for i := range codes {
				codes[i] = 200 + i
			}
			response, err := rest.DoJSONGuardedRejections(readCtx, p.client, p.check, http.MethodGet, imageMemberEndpoint(p.preparedImageMutation, &identity), nil, nil, rest.RejectionPolicy{Codes: []int{http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound}, PreserveCleanRetry: true}, codes...)
			if err != nil {
				return nil, err
			}
			seed := make(map[string]json.RawMessage)
			seed["id"], _ = json.Marshal(identity)
			return imageMemberRecordFromResponse(readCtx, p.check, seed, p.id, p.location, response)
		},
		IdentityResponseID: func(value *ImageMemberRecord) (string, error) {
			if value == nil || value.Resource == nil {
				return "", uploadInvalid("member Resource is required")
			}
			return imageMemberRecordText(value, "id"), nil
		},
		Name: func(value *ImageMemberRecord) string { return imageMemberRecordText(value, "name") },
		Iterate: func(context.Context, url.Values) iter.Seq2[*ImageMemberRecord, error] {
			return listImageMemberRecordsPrepared(p, imageMemberRecordListParameters{})
		},
	})
	result, err := collection.FindIdentity(p.ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return fail(err)
	}
	return result, nil
}

func validateImageMemberRecordIdentity(value string) error {
	if strings.TrimSpace(value) == "" || !utf8.ValidString(value) || value == "." || value == ".." {
		return uploadInvalid("member identity must be nonblank valid UTF-8 literal text")
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return uploadInvalid("member identity must not contain controls")
		}
	}
	return nil
}

func imageMemberRecordText(value *ImageMemberRecord, field string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	text, err := decodeImageRecordString(value.Resource.Body[field], "member record "+field)
	if err != nil {
		return ""
	}
	return text
}
