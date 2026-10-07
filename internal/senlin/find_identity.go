package senlin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const findMicroversionArgument = "senlin.find.microversion"

type FindOpts struct {
	IgnoreMissing *bool
	Fallback      resource.FindFallbackPolicy
}

type FindOption = request.Option[FindOpts]

func WithFindIgnoreMissing(value bool) FindOption {
	return func(config *request.Config[FindOpts]) error {
		copy := value
		config.Options.IgnoreMissing = &copy
		return nil
	}
}

func WithFindFallback(value resource.FindFallbackPolicy) FindOption {
	return func(config *request.Config[FindOpts]) error {
		config.Options.Fallback = value
		return nil
	}
}

func WithFindHeader(key, value string) FindOption {
	return request.WithHeader[FindOpts](key, value)
}

// WithFindMicroversion follows the owned list request version policy. Explicit
// empty selects the server default, and omitted inherits the source selection.
func WithFindMicroversion(value string) FindOption {
	return request.WithArgument[FindOpts](findMicroversionArgument, value)
}

// FindIdentity tries the controller identity route first, then optionally
// searches all listed rows for one exact original ID or name. Explicit
// transport controls are prepared once for both phases; without these controls
// the existing source client selection remains live.
func FindIdentity[T any](ctx context.Context, source *gophercloud.ServiceClient, factory func(*gophercloud.ServiceClient) rest.CollectionSpec[T], normalize func(*T, string, string), identity string, options ...FindOption) (*T, error) {
	kind := "clustering.find"
	if factory != nil {
		kind = factory(source).Kind
	}
	fail := func(err error) (*T, error) { return nil, request.Wrap("FindIdentity", kind, err) }
	config, err := request.Apply(FindOpts{}, append([]FindOption(nil), options...)...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true, findMicroversionArgument)
	}
	if err == nil {
		err = Identifier(identity)
	}
	if err == nil && (config.Options.Fallback < resource.FindFallbackCompatible || config.Options.Fallback > resource.FindFallbackNever) {
		err = fmt.Errorf("%w: unsupported identity fallback policy", resource.ErrInvalidOption)
	}
	if err == nil && (factory == nil || normalize == nil) {
		err = fmt.Errorf("%w: identity collection factory and normalizer are required", resource.ErrInvalidOption)
	}
	if err != nil {
		return fail(err)
	}
	ignoreMissing := config.Options.IgnoreMissing == nil || *config.Options.IgnoreMissing
	fallback := config.Options.Fallback
	// Preserve the Find namespace boundary before translating the one owned
	// transport control to the existing List preparation implementation.
	transport := config
	transport.Arguments = maps.Clone(config.Arguments)
	if version, exists := transport.Arguments[findMicroversionArgument]; exists {
		delete(transport.Arguments, findMicroversionArgument)
		transport.Arguments[listMicroversionArgument] = version
	}
	client, err := PrepareListClient(ctx, source, transport)
	if err != nil {
		return fail(err)
	}
	spec := ListSpec(source, client, factory)
	kind = spec.Kind
	if err := validateFindSpec(ctx, spec); err != nil {
		return fail(err)
	}
	if spec.ValidateID != nil {
		if err := spec.ValidateID(identity); err != nil {
			return fail(err)
		}
	}
	codes := append([]int(nil), spec.GetCodes...)
	if len(codes) == 0 {
		codes = []int{http.StatusOK}
	}
	response, err := rest.DoJSON(ctx, spec.Client, http.MethodGet, spec.Client.ServiceURL(spec.Path, url.PathEscape(identity)), nil, nil, codes...)
	if err == nil {
		value, err := decodeFindIdentity(response, spec, normalize)
		if err != nil {
			return fail(err)
		}
		return value, nil
	}
	if !findCanFallback(err, fallback) {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) && response == nil && !findNonHTTPFailure(err) {
			if ignoreMissing {
				return nil, nil
			}
			return fail(&resource.NotFoundError{Resource: kind, Reference: identity, Cause: err})
		}
		return fail(err)
	}
	nameKey := spec.NameQueryKey
	if nameKey == "" {
		nameKey = "name"
	}
	nameQuery := identity
	if spec.NameQuery != nil {
		nameQuery = spec.NameQuery(identity)
	}
	query := url.Values{nameKey: {nameQuery}}
	var found *T
	var foundID string
	validateItem := spec.ValidateItem
	spec.ValidateItem = func(value *T) error {
		if validateItem != nil {
			if err := validateItem(value); err != nil {
				return err
			}
		}
		id, name, err := findRawIdentity(spec.Metadata(value))
		if err != nil {
			return err
		}
		if spec.ValidateID != nil {
			if err := spec.ValidateID(id); err != nil {
				return err
			}
		}
		normalize(value, id, name)
		if found != nil && (id == identity || name == identity) {
			return &resource.AmbiguousError{Resource: kind, Name: identity, IDs: []string{foundID, id}}
		}
		return nil
	}
	marker := spec.Paging.Marker
	if marker != nil {
		spec.Paging.Marker = func(value *T) (string, error) {
			id, name, err := findRawIdentity(spec.Metadata(value))
			if err != nil {
				return "", err
			}
			normalize(value, id, name)
			return marker(value)
		}
	}
	for value, err := range rest.ListWithControl(ctx, spec, query, rest.ListControl{}) {
		if err != nil {
			return fail(err)
		}
		id, name, err := findRawIdentity(spec.Metadata(value))
		if err != nil {
			// ValidateItem already established these fields before yielding.
			return fail(err)
		}
		if id != identity && name != identity {
			continue
		}
		if found != nil {
			return fail(&resource.AmbiguousError{Resource: kind, Name: identity, IDs: []string{foundID, id}})
		}
		found, foundID = value, id
	}
	if found == nil && !ignoreMissing {
		// The successful list search is the final absence observation. Do not
		// attach the suppressed direct 400/403 response as its cause.
		return fail(&resource.NotFoundError{Resource: kind, Reference: identity})
	}
	return found, nil
}

func validateFindSpec[T any](ctx context.Context, spec rest.CollectionSpec[T]) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if spec.Client == nil || spec.Client.ProviderClient == nil || strings.TrimSpace(spec.Path) == "" || spec.Metadata == nil {
		return fmt.Errorf("%w: identity collection client, path and metadata are required", resource.ErrInvalidOption)
	}
	if !spec.Get {
		return fmt.Errorf("%w: identity lookup requires a direct GET", resource.ErrUnsupported)
	}
	if spec.Validate != nil {
		return spec.Validate(ctx)
	}
	return nil
}

func decodeFindIdentity[T any](response *rest.Response, spec rest.CollectionSpec[T], normalize func(*T, string, string)) (*T, error) {
	body, err := response.Object(spec.SingleKey)
	if err != nil {
		return nil, err
	}
	var value T
	if err := resource.DecodeObject(body, &value, spec.Metadata(&value)); err != nil {
		return nil, response.Fail(err)
	}
	metadata := spec.Metadata(&value)
	if metadata == nil {
		return nil, response.Fail(fmt.Errorf("%w: identity response metadata is required", resource.ErrInvalidOption))
	}
	metadata.Header, metadata.StatusCode = response.Header.Clone(), response.StatusCode
	if spec.ValidateItem != nil {
		if err := spec.ValidateItem(&value); err != nil {
			return nil, response.Fail(err)
		}
	}
	id, name, err := findRawIdentity(metadata)
	if err != nil {
		return nil, response.Fail(err)
	}
	if spec.ValidateID != nil {
		if err := spec.ValidateID(id); err != nil {
			return nil, response.Fail(err)
		}
	}
	normalize(&value, id, name)
	return &value, nil
}

func findRawIdentity(metadata *resource.Metadata) (string, string, error) {
	if metadata == nil {
		return "", "", fmt.Errorf("%w: identity response metadata is required", resource.ErrInvalidOption)
	}
	var id string
	if raw, exists := metadata.Body["id"]; !exists || json.Unmarshal(raw, &id) != nil {
		return "", "", fmt.Errorf("%w: identity response requires a string id", resource.ErrInvalidOption)
	}
	if err := Identifier(id); err != nil {
		return "", "", err
	}
	var name string
	if raw, exists := metadata.Body["name"]; exists && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if err := json.Unmarshal(raw, &name); err != nil {
			return "", "", fmt.Errorf("%w: identity response name must be a string or null", resource.ErrInvalidOption)
		}
	}
	return id, name, nil
}

func findNonHTTPFailure(err error) bool {
	var accepted *resource.ResponseError
	var transport *url.Error
	return errors.As(err, &accepted) || errors.As(err, &transport) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func findCanFallback(err error, policy resource.FindFallbackPolicy) bool {
	if policy == resource.FindFallbackNever || findNonHTTPFailure(err) {
		return false
	}
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return true
	}
	return policy == resource.FindFallbackCompatible && (gophercloud.ResponseCodeIs(err, http.StatusBadRequest) || gophercloud.ResponseCodeIs(err, http.StatusForbidden))
}
