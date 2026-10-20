package keypairs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// KeypairDeleteOpts selects an optional owner and the missing-resource policy.
// An omitted IgnoreMissing defaults to true; Present(false) requires existence.
// UserID is sent literally when nonempty. Nova enforces owner permissions and
// the selected microversion; this operation does not change that version.
type KeypairDeleteOpts struct {
	UserID        string
	IgnoreMissing request.Optional[bool]
}

type KeypairDeleteOption = request.Option[KeypairDeleteOpts]

func WithKeypairDeleteOptions(value KeypairDeleteOpts) KeypairDeleteOption {
	return request.WithOptions(value)
}

func WithKeypairDeleteUserID(value string) KeypairDeleteOption {
	return func(config *request.Config[KeypairDeleteOpts]) error {
		config.Options.UserID = value
		return nil
	}
}

func WithKeypairDeleteIgnoreMissing(value bool) KeypairDeleteOption {
	return func(config *request.Config[KeypairDeleteOpts]) error {
		config.Options.IgnoreMissing = request.Present(value)
		return nil
	}
}

// WithKeypairDeleteQuery adds a wire extension. The owner and local missing
// policy require their concrete options and cannot be replaced by extensions.
func WithKeypairDeleteQuery(key, value string) KeypairDeleteOption {
	return request.WithQuery[KeypairDeleteOpts](key, value)
}

func WithKeypairDeleteHeader(key, value string) KeypairDeleteOption {
	return request.WithHeader[KeypairDeleteOpts](key, value)
}

// DeleteKeypair deletes a keypair directly by its name, which is Nova's logical
// keypair ID. It ignores ordinary missing responses by default and performs no
// preliminary lookup or wait. Accepted response bodies are passive bytes.
// Generated Delete and Remove retain their existing native behavior.
func (a *API) DeleteKeypair(ctx context.Context, name string, options ...KeypairDeleteOption) error {
	fail := func(err error) error {
		return request.Wrap("DeleteKeypair", "keypairs", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	if err := resource.ID(name).Validate(); err != nil {
		return fail(err)
	}
	if !utf8.ValidString(name) || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fail(fmt.Errorf("%w: keypair name must be valid text without controls", resource.ErrInvalidOption))
	}
	if a == nil {
		return fail(fmt.Errorf("%w: keypair API is required", resource.ErrInvalidOption))
	}
	original := a.client
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return fail(err)
	}
	var changed error
	guard := func(ctx context.Context) error {
		if a.client != original && changed == nil {
			changed = fmt.Errorf("%w: keypair API source changed", resource.ErrInvalidOption)
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	owned := slices.Clone(options)
	guarded := make([]KeypairDeleteOption, len(owned))
	for index, option := range owned {
		apply := option
		guarded[index] = func(config *request.Config[KeypairDeleteOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil keypair delete option", resource.ErrInvalidOption)
			}
			ownKeypairDeleteConfig(config)
			applyErr := apply(config)
			ownKeypairDeleteConfig(config)
			return errors.Join(applyErr, guard(ctx))
		}
	}
	config, err := request.Apply(KeypairDeleteOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, true)
	}
	if err == nil && config.Options.IgnoreMissing.IsNull() {
		err = fmt.Errorf("%w: ignore-missing must be a boolean", resource.ErrInvalidOption)
	}
	query := make(url.Values, len(config.Query)+1)
	if err == nil {
		for key, values := range config.Query {
			if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
				err = fmt.Errorf("%w: invalid keypair delete query key", resource.ErrInvalidOption)
				break
			}
			if strings.EqualFold(key, "user_id") || strings.EqualFold(key, "ignore_missing") {
				err = fmt.Errorf("%w: query %q requires its concrete keypair delete option", resource.ErrInvalidOption, key)
				break
			}
			query[key] = slices.Clone(values)
		}
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return fail(err)
	}
	if config.Options.UserID != "" {
		query.Set("user_id", config.Options.UserID)
	}
	target := source.Client.ServiceURL("os-keypairs", url.PathEscape(name))
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = index + 200
	}
	collection := resource.NewCollection(resource.Adapter[KeyPair]{
		Kind: "keypairs",
		Delete: func(ctx context.Context, _ string) error {
			response, err := rest.DoJSONGuarded(ctx, &source.Client, guard, http.MethodDelete, target, nil, nil, codes...)
			if final := guard(ctx); final != nil {
				if response != nil && err == nil {
					err = response.Fail(final)
				} else {
					err = errors.Join(err, final)
				}
			}
			return cloudread.ContextError(ctx, err)
		},
	})
	var lookup []resource.LookupOption
	if ignore, supplied := config.Options.IgnoreMissing.Get(); supplied && !ignore {
		lookup = []resource.LookupOption{resource.WithMissingError()}
	}
	err = collection.Delete(ctx, resource.ID(name), lookup...)
	return fail(errors.Join(err, guard(ctx)))
}

func ownKeypairDeleteConfig(config *request.Config[KeypairDeleteOpts]) {
	query := make(url.Values, len(config.Query))
	for key, values := range config.Query {
		query[key] = slices.Clone(values)
	}
	config.Query = query
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	config.Fields = maps.Clone(config.Fields)
	if config.Fields == nil {
		config.Fields = make(map[string]json.RawMessage)
	}
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
}
