package secretconsumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "keymanager.secret_consumers"

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// SecretScope fixes the parent once. No response reference or association field
// can retarget it. Explicit ID construction performs no HTTP.
type SecretScope struct {
	client   *gophercloud.ServiceClient
	secretID string
}

func (s *SecretScope) RawClient() *gophercloud.ServiceClient {
	if s == nil {
		return nil
	}
	return s.client
}
func (s *SecretScope) SecretID() string {
	if s == nil {
		return ""
	}
	return s.secretID
}

// InSecret accepts a safe explicit ID or resolves an explicit Name once through
// the existing Secrets collection. Name lookup is an explicit Go convenience;
// the Python consumer proxy treats its string argument as an ID.
func (a *API) InSecret(ctx context.Context, ref resource.Ref) (*SecretScope, error) {
	client := a.RawClient()
	if err := validate(ctx, client); err != nil {
		return nil, request.Wrap("InSecret", kind, err)
	}
	if err := ref.Validate(); err != nil {
		return nil, request.Wrap("InSecret", kind, err)
	}
	id, err := secrets.New(client).Resources.ResolveID(ctx, ref)
	if err == nil {
		err = validate(ctx, client)
	}
	if err == nil {
		err = secretID(id)
	}
	if err != nil {
		return nil, request.Wrap("InSecret", kind, err)
	}
	return &SecretScope{client: client, secretID: id}, nil
}

func validate(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: key-manager service client is required", resource.ErrInvalidOption)
	}
	if client.Type != "key-manager" {
		return fmt.Errorf("%w: expected key-manager service client", resource.ErrUnsupported)
	}
	return validateHeaders(client.MoreHeaders)
}
func secretID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return fmt.Errorf("%w: secret ID must be valid UTF-8", resource.ErrInvalidOption)
	}
	for _, r := range id {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return fmt.Errorf("%w: invalid secret ID", resource.ErrInvalidOption)
		}
	}
	return nil
}
func (s *SecretScope) validate(ctx context.Context) error {
	if err := validate(ctx, s.RawClient()); err != nil {
		return err
	}
	return secretID(s.secretID)
}
func (s *SecretScope) endpoint() string {
	return s.client.ServiceURL("secrets", url.PathEscape(s.secretID), "consumers")
}

func (s *SecretScope) Create(ctx context.Context, opts ConsumerOpts, options ...CreateOption) (*SecretResponse, error) {
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	config, err := request.Apply(opts, options...)
	var body []byte
	if err == nil {
		body, err = consumerBody(config, config.Options)
	}
	if err == nil {
		err = s.validate(ctx)
	}
	if err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	response, err := rest.DoJSON(ctx, s.client, http.MethodPost, s.endpoint(), json.RawMessage(body), maps.Clone(config.Headers), http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	value, err := rest.Decode(response, "", func(value *SecretResponse) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap("Create", kind, err)
}

// Delete returns the actual secret response. A suppressed missing association
// returns nil,nil; no request-seeded Consumer or replacement secret is created.
func (s *SecretScope) Delete(ctx context.Context, opts ConsumerOpts, options ...DeleteOption) (*SecretResponse, error) {
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap("Delete", kind, err)
	}
	config, err := request.Apply(DeleteOpts{ConsumerOpts: opts}, options...)
	var body []byte
	if err == nil {
		body, err = consumerBody(config, config.Options.ConsumerOpts)
	}
	ignoreMissing := config.Options.IgnoreMissing == nil || *config.Options.IgnoreMissing
	if err == nil {
		err = s.validate(ctx)
	}
	if err != nil {
		return nil, request.Wrap("Delete", kind, err)
	}
	response, err := rest.DoJSON(ctx, s.client, http.MethodDelete, s.endpoint(), json.RawMessage(body), maps.Clone(config.Headers), http.StatusOK)
	// A native HTTP error can arrive after a custom transport cancels the
	// caller. Preserve both observations before missing suppression.
	if err != nil && ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	var transport *url.Error
	var accepted *resource.ResponseError
	if ignoreMissing && !errors.As(err, &transport) && !errors.As(err, &accepted) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, request.Wrap("Delete", kind, err)
	}
	value, err := rest.Decode(response, "", func(value *SecretResponse) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap("Delete", kind, err)
}

func (s *SecretScope) spec() rest.CollectionSpec[Consumer] {
	return rest.CollectionSpec[Consumer]{Client: s.client, Path: "secrets/" + url.PathEscape(s.secretID) + "/consumers", Kind: kind, PluralKey: "consumers", ListCodes: []int{http.StatusOK}, Metadata: func(value *Consumer) *resource.Metadata { return &value.Metadata }, Validate: s.validate,
		Paging: rest.PagePolicy[Consumer]{OffsetPagination: true, StopOnEmptyPage: true}}
}

func (s *SecretScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*Consumer, error] {
	owned := append([]ListOption(nil), options...)
	return func(yield func(*Consumer, error) bool) {
		if err := s.validate(ctx); err != nil {
			yield(nil, request.Wrap("List", kind, err))
			return
		}
		query, control, err := prepareList(owned)
		if err == nil {
			err = s.validate(ctx)
		}
		if err != nil {
			yield(nil, request.Wrap("List", kind, err))
			return
		}
		for value, err := range rest.ListWithControl(ctx, s.spec(), query, control) {
			if !yield(value, request.Wrap("List", kind, err)) {
				return
			}
		}
	}
}
func (s *SecretScope) All(ctx context.Context, options ...ListOption) ([]*Consumer, error) {
	var values []*Consumer
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
