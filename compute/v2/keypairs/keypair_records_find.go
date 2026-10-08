package keypairs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// KeypairFindOpts supplies an optional owner and missing-result policy. Nil
// IgnoreMissing defaults to true. Nil Microversion retains a selected version
// or discovers at most 2.10; an explicit empty value selects versionless reads.
type KeypairFindOpts struct {
	UserID        string
	IgnoreMissing *bool
	Microversion  *string
}

type KeypairFindOption = request.Option[KeypairFindOpts]

func copyKeypairFindOptions(value KeypairFindOpts) KeypairFindOpts {
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	if value.Microversion != nil {
		owned := *value.Microversion
		value.Microversion = &owned
	}
	return value
}

// WithKeypairFindOptions replaces typed options, retaining additional headers.
func WithKeypairFindOptions(value KeypairFindOpts) KeypairFindOption {
	owned := copyKeypairFindOptions(value)
	return func(config *request.Config[KeypairFindOpts]) error {
		config.Options = copyKeypairFindOptions(owned)
		return nil
	}
}
func WithKeypairFindUserID(value string) KeypairFindOption {
	return func(config *request.Config[KeypairFindOpts]) error { config.Options.UserID = value; return nil }
}
func WithKeypairFindIgnoreMissing(value bool) KeypairFindOption {
	return func(config *request.Config[KeypairFindOpts]) error {
		owned := value
		config.Options.IgnoreMissing = &owned
		return nil
	}
}
func WithKeypairFindMicroversion(value string) KeypairFindOption {
	return func(config *request.Config[KeypairFindOpts]) error {
		owned := value
		config.Options.Microversion = &owned
		return nil
	}
}
func WithKeypairFindHeader(key, value string) KeypairFindOption {
	return request.WithHeader[KeypairFindOpts](key, value)
}

func prepareKeypairFind(ctx context.Context, guard func(context.Context) error, options []KeypairFindOption) (request.Config[KeypairFindOpts], error) {
	guarded := make([]KeypairFindOption, len(options))
	for index, option := range options {
		apply := option
		guarded[index] = func(config *request.Config[KeypairFindOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil keypair-find option", resource.ErrInvalidOption)
			}
			config.Options = copyKeypairFindOptions(config.Options)
			copyKeypairReadConfig(config)
			applyErr := apply(config)
			config.Options = copyKeypairFindOptions(config.Options)
			copyKeypairReadConfig(config)
			return errors.Join(applyErr, guard(ctx))
		}
	}
	config, err := request.Apply(KeypairFindOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	return config, err
}

// Resource identity fields remain passive. A numeric/null response name does
// not equal the caller's string identity and never becomes another request URL.
func keypairRecordName(value *KeypairRecord) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var name string
	if err := json.Unmarshal(value.Resource.Body["name"], &name); err != nil {
		return ""
	}
	return name
}

// The fixed member reader escapes this literal text once. Names containing
// spaces or reserved URL characters remain one member, never another route.
func validateKeypairRecordIdentity(identity string) error {
	if strings.TrimSpace(identity) == "" || !utf8.ValidString(identity) || identity == "." || identity == ".." {
		return fmt.Errorf("%w: keypair identity must be nonempty UTF-8 literal text", resource.ErrInvalidOption)
	}
	for _, char := range identity {
		if unicode.IsControl(char) {
			return fmt.Errorf("%w: keypair identity must not contain controls", resource.ErrInvalidOption)
		}
	}
	return nil
}

func fetchKeypairRecord(ctx context.Context, source *cloudread.Source, identity, owner, version string, check func(context.Context) error) (*KeypairRecord, error) {
	target := source.Client.ServiceURL("os-keypairs", url.PathEscape(identity))
	if owner != "" {
		target += "?" + url.Values{"user_id": []string{owner}}.Encode()
	}
	response, prior := microversions.MemberGet(ctx, source, target, version, microversions.NovaProfile, keypairRecordCodes()...)
	if response == nil {
		return nil, prior
	}
	result := &KeypairRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	fail := func(err error) (*KeypairRecord, error) {
		return result, response.Fail(cloudread.ContextError(ctx, err))
	}
	if err := errors.Join(prior, check(ctx)); err != nil {
		return fail(err)
	}
	seedName, _ := json.Marshal(identity)
	seed := map[string]json.RawMessage{"name": seedName}
	if owner != "" {
		seed["user_id"], _ = json.Marshal(owner)
	}
	// Resource.fetch tolerates response.json ValueError, but a parsed root or
	// keypair envelope must be an object. Physical failures never reach here.
	if utf8.Valid(response.Body) && json.Valid(response.Body) {
		root, err := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
		if err != nil {
			return result, err
		}
		wire := root
		if _, present := root.Body["keypair"]; present {
			wire, err = rest.Decode(response, "keypair", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
			if err != nil {
				return result, err
			}
		}
		result.Wire = wire.Clone()
		maps.Copy(seed, normalizedKeypairCreateFields(wire.Body))
	}
	view, err := projectKeypairRecord(seed, resource.Metadata{Header: response.Header, StatusCode: response.StatusCode})
	if err != nil {
		return fail(err)
	}
	if err := check(ctx); err != nil {
		return fail(err)
	}
	result.Resource = view
	return result, nil
}

// FindKeypair first reads the fixed member route, then falls back to the same
// owner's complete list only after a clean native 400, 403 or 404. Missing is
// ignored by default; duplicate and late list failures are always returned.
// Native Get, Find and their typed KeyPair results remain unchanged.
func (a *API) FindKeypair(ctx context.Context, nameOrID string, options ...KeypairFindOption) (*KeypairRecord, error) {
	fail := func(value *KeypairRecord, err error) (*KeypairRecord, error) {
		return value, request.Wrap("FindKeypair", "keypairs", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if err := validateKeypairRecordIdentity(nameOrID); err != nil {
		return fail(nil, err)
	}
	source, guard, err := keypairRecordSource(ctx, a)
	if err != nil {
		return fail(nil, err)
	}
	operationCtx := rest.WithOperationGuard(ctx, guard)
	check := func(checkCtx context.Context) error {
		return errors.Join(guard(checkCtx), rest.CheckOperationGuard(checkCtx))
	}
	owned := append([]KeypairFindOption(nil), options...)
	config, err := prepareKeypairFind(ctx, check, owned)
	if err != nil {
		return fail(nil, err)
	}
	value := copyKeypairFindOptions(config.Options)
	if !hasKeypairHeader(config.Headers, "Accept") {
		config.Headers["Accept"] = "application/json"
	}
	chosen, err := selectKeypairReadVersion(operationCtx, source, value.Microversion, config.Headers)
	if err != nil {
		return fail(nil, err)
	}
	if err := check(operationCtx); err != nil {
		return fail(nil, err)
	}
	query := make(url.Values)
	if value.UserID != "" {
		query.Set("user_id", value.UserID)
	}
	var acceptedPartial *KeypairRecord
	collection := resource.NewCollection(resource.Adapter[KeypairRecord]{
		Kind: "keypairs", IdentityFind: true, ValidateID: validateKeypairRecordIdentity, IdentityDirectGet: validateKeypairRecordIdentity,
		Get: func(ctx context.Context, identity string) (*KeypairRecord, error) {
			result, err := fetchKeypairRecord(ctx, source, identity, value.UserID, chosen, check)
			if err != nil {
				acceptedPartial = result
			}
			return result, err
		},
		IdentityResponseID: func(value *KeypairRecord) (string, error) {
			if value == nil || value.Resource == nil {
				return "", fmt.Errorf("%w: keypair Resource is required", resource.ErrInvalidOption)
			}
			return keypairRecordName(value), nil
		},
		Name: keypairRecordName,
		Iterate: func(ctx context.Context, _ url.Values) iter.Seq2[*KeypairRecord, error] {
			return listKeypairRecordsPrepared(ctx, source, guard, keypairListParameters{query: query}, true)
		},
	})
	ignoreMissing := value.IgnoreMissing == nil || *value.IgnoreMissing
	result, err := collection.FindIdentity(operationCtx, nameOrID, resource.WithIdentityFindIgnoreMissing(ignoreMissing))
	if err != nil {
		return fail(acceptedPartial, err)
	}
	if err := check(operationCtx); err != nil {
		return fail(result, err)
	}
	return result, nil
}
