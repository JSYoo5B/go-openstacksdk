package image

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FindImageRecordOpts owns one GET-first discovery operation. IgnoreMissing
// defaults to true. List controls and filters are explicit Go extensions to the
// pinned query-free proxy. Its headers also apply to the initial GET; Headers
// override the nested list headers for all phases.
type FindImageRecordOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
	List          ImageRecordListOpts
	listOptions   []ImageRecordListOption
}
type FindImageRecordOption func(*FindImageRecordOpts) error

func copyFindImageRecord(value FindImageRecordOpts) FindImageRecordOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	value.List = copyImageRecordList(value.List)
	value.listOptions = slices.Clone(value.listOptions)
	return value
}
func WithFindImageRecordOpts(value FindImageRecordOpts) FindImageRecordOption {
	owned := copyFindImageRecord(value)
	return func(config *FindImageRecordOpts) error { *config = copyFindImageRecord(owned); return nil }
}
func WithFindImageRecordHeader(key, value string) FindImageRecordOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *FindImageRecordOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithFindImageRecordHeaders(values map[string]string) FindImageRecordOption {
	apply := WithImageMutationHeaders(values)
	return func(config *FindImageRecordOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithFindImageRecordIgnoreMissing(value bool) FindImageRecordOption {
	return func(config *FindImageRecordOpts) error { owned := value; config.IgnoreMissing = &owned; return nil }
}

// WithFindImageRecordListOptions appends options to the owned list profile.
// They are applied once with context and source checks between callbacks.
func WithFindImageRecordListOptions(options ...ImageRecordListOption) FindImageRecordOption {
	owned := slices.Clone(options)
	return func(config *FindImageRecordOpts) error {
		config.listOptions = append(config.listOptions, owned...)
		return nil
	}
}

func prepareFindImageRecord(ctx context.Context, check func(context.Context) error, options []FindImageRecordOption) (FindImageRecordOpts, imageRecordListParameters, error) {
	value := copyFindImageRecord(FindImageRecordOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return value, imageRecordListParameters{}, err
		}
		if apply == nil {
			return value, imageRecordListParameters{}, uploadInvalid("nil find image record option")
		}
		owned := copyFindImageRecord(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return value, imageRecordListParameters{}, err
		}
		value = copyFindImageRecord(owned)
	}
	listOptions := append([]ImageRecordListOption{WithImageRecordListOpts(value.List)}, value.listOptions...)
	parameters, err := prepareImageRecordList(ctx, check, listOptions)
	if err != nil {
		return value, parameters, err
	}
	headers := maps.Clone(parameters.headers)
	if headers == nil {
		headers = make(map[string]string)
	}
	findHeaders, headerErr := imageMutationHeaders(value.Headers, false, "")
	if headerErr != nil {
		return value, parameters, errors.Join(headerErr, check(ctx))
	}
	for key, val := range findHeaders {
		headers[key] = val
	}
	parameters.headers, err = imageMutationHeaders(headers, false, "")
	return value, parameters, errors.Join(err, check(ctx))
}

// FindImageRecord follows the pinned Image search: literal GET, a complete
// exact ID-or-name search with an automatic name hint, then a hidden-image
// search after successful absence. The second list starts from the original
// query. Clean native 400/403/404 permit fallback; IO, hook, source and accepted
// response errors are terminal. All phases share one prepared operation.
func (s *Service) FindImageRecord(ctx context.Context, nameOrID string, options ...FindImageRecordOption) (*ImageRecord, error) {
	const operation = "FindImageRecord"
	fail := func(err error) (*ImageRecord, error) {
		return nil, wrapImageMutationError(ctx, operation, cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	if err := validateImageRecordIdentity(nameOrID); err != nil {
		return fail(err)
	}
	owned := slices.Clone(options)
	var policy FindImageRecordOpts
	var parameters imageRecordListParameters
	p, err := s.prepareImageRecord(ctx, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		var err error
		policy, parameters, err = prepareFindImageRecord(opctx, check, owned)
		return parameters.headers, err
	})
	if err != nil {
		return fail(err)
	}
	result, err := findPreparedImageRecord(p, nameOrID, policy, parameters)
	if err != nil {
		return fail(err)
	}
	return result, nil
}

// Compound image creation uses the same complete discovery engine with the
// original source, location and headers captured for the whole workflow.
func findPreparedImageRecord(p *preparedImageRecord, nameOrID string, policy FindImageRecordOpts, parameters imageRecordListParameters) (*ImageRecord, error) {
	get := func(readCtx context.Context, identity string) (*ImageRecord, error) {
		response, err := rest.DoJSONGuardedRejections(readCtx, p.client, p.check, http.MethodGet, imageRecordEndpoint(p, identity), nil, nil, rest.RejectionPolicy{Codes: []int{http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound}, PreserveCleanRetry: true}, imageRecordCodes()...)
		if err != nil {
			return nil, err
		}
		id, _ := json.Marshal(identity)
		return imageRecordFromResponse(readCtx, p.check, map[string]json.RawMessage{"id": id}, p.location, response)
	}
	collection := resource.NewCollection(resource.Adapter[ImageRecord]{
		Kind: "images", IdentityFind: true, ValidateID: validateImageRecordIdentity, IdentityDirectGet: validateImageRecordIdentity,
		Get: get,
		// Find's explicit Go query controls are list-only. The Python proxy has
		// no query kwargs and its default direct GET is query-free.
		GetIdentityQuery: func(readCtx context.Context, id string, _ url.Values) (*ImageRecord, error) { return get(readCtx, id) },
		IdentityResponseID: func(value *ImageRecord) (string, error) {
			if value == nil || value.Resource == nil {
				return "", uploadInvalid("image Resource is required")
			}
			return imageRecordText(value, "id"), nil
		},
		Name: func(value *ImageRecord) string { return imageRecordText(value, "name") }, NameQuery: func(value string) string { return value },
		IdentityMissingListQuery: url.Values{"os_hidden": {"True"}},
		Iterate: func(_ context.Context, q url.Values) iter.Seq2[*ImageRecord, error] {
			phase := parameters
			phase.query = q
			return listImageRecordsPrepared(p, phase)
		},
	})
	result, err := collection.FindIdentity(p.ctx, nameOrID, resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: parameters.query, IgnoreMissing: policy.IgnoreMissing}))
	return result, errors.Join(err, p.check(p.ctx))
}
