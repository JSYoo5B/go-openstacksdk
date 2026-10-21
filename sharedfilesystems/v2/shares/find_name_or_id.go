package shares

import (
	"context"
	"iter"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/manilafind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// FindOpts controls FindByNameOrID. Python's find_*: ignore_missing defaults
// to true (a missing resource returns nil, nil).
type FindOpts struct {
	IgnoreMissing *bool
	AllProjects   *bool
}

type FindOption = request.Option[FindOpts]

func WithFindIgnoreMissing(value bool) FindOption {
	return func(c *request.Config[FindOpts]) error { c.Options.IgnoreMissing = &value; return nil }
}

// WithFindAllProjects sends all_tenants=true or false on the list fallback only.
func WithFindAllProjects(value bool) FindOption {
	return func(c *request.Config[FindOpts]) error { c.Options.AllProjects = &value; return nil }
}

// WithFindQuery adds a list-fallback filter. A supplied "name" replaces the
// default name=<identity> filter, as Python does.
func WithFindQuery(key, value string) FindOption { return request.WithQuery[FindOpts](key, value) }

// FindByNameOrID reproduces Python Resource.find: GET /shares/{identity} when
// the identity is a safe path segment; after HTTP 400, 403 or 404 (and for
// identities that cannot be a segment) list GET /shares with name=<identity>
// and match the exact ID or name. Two matches fail with resource.ErrAmbiguous.
// Other GET errors are returned. No mutation is involved and nothing replays.
func (a *API) FindByNameOrID(ctx context.Context, identity string, options ...FindOption) (*Share, error) {
	config, err := request.Apply(FindOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, false)
	}
	if err != nil {
		return nil, request.Wrap("FindByNameOrID", "shares", err)
	}
	query := url.Values(config.Query)
	if config.Options.AllProjects != nil {
		query.Set("all_tenants", strconv.FormatBool(*config.Options.AllProjects))
	}
	ignore := config.Options.IgnoreMissing == nil || *config.Options.IgnoreMissing
	value, err := manilafind.Find(ctx, manilafind.Spec[Share]{
		Kind: "shares", Identity: identity, IgnoreMissing: ignore, NameQuery: true, Query: query,
		Get: a.Get,
		List: func(ctx context.Context, q url.Values) iter.Seq2[*Share, error] {
			options := make([]ListOption, 0, len(q))
			for key, values := range q {
				for _, v := range values {
					options = append(options, WithListQuery(key, v))
				}
			}
			return a.List(ctx, options...)
		},
		ID: func(v *Share) string { return v.ID }, Name: func(v *Share) string { return v.Name },
	})
	return value, request.Wrap("FindByNameOrID", "shares", err)
}
