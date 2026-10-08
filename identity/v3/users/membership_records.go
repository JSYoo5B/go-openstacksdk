package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// These fixed hooks are supplied only by the two owned membership APIs. The
// shared raw container retains the public UserProjectRecord's type identity.
type membershipRecordRead struct {
	operation, kind, plural string
	prepare                 func(context.Context, func(context.Context) error) (projectListParameters, error)
	project                 func(*UserProjectRecord, string) error
	marker                  func(*UserProjectRecord) (string, error)
	iterate                 func(context.Context, rest.CollectionSpec[UserProjectRecord], projectListParameters) iter.Seq2[*UserProjectRecord, error]
}

// Preserve the pointer already supplied to rest's metadata hook. RawResource's
// decoder updates it atomically and owns the response's complete JSON fields.
func unmarshalMembershipRecord(data []byte, wire **resource.RawResource) error {
	if *wire == nil {
		*wire = &resource.RawResource{}
	}
	return json.Unmarshal(data, *wire)
}

func cloneMembershipRecordView(wire *resource.RawResource, userID string) (*resource.RawResource, error) {
	if wire == nil || wire.Body == nil {
		return nil, fmt.Errorf("%w: membership response fields are required", resource.ErrInvalidOption)
	}
	view := wire.Clone()
	parent, err := json.Marshal(userID)
	if err != nil {
		return nil, err
	}
	view.Body["user_id"] = parent
	return view, nil
}

// The runner owns source, parent URI, accepted response and continuation policy.
// Project options/filtering stay in their existing prepare/iterator hooks;
// Group's absent hooks leave its initial query and local controls empty.
func (a *API) listMembershipRecords(ctx context.Context, userID string, policy membershipRecordRead) iter.Seq2[*UserProjectRecord, error] {
	return func(yield func(*UserProjectRecord, error) bool) {
		wrap := func(err error) error {
			if err != nil {
				err = cloudread.ContextError(ctx, err)
			}
			return request.Wrap(policy.operation, policy.kind, err)
		}
		if err := cloudread.Context(ctx); err != nil {
			yield(nil, wrap(err))
			return
		}
		if err := resource.ID(userID).Validate(); err != nil {
			yield(nil, wrap(err))
			return
		}
		var client *gophercloud.ServiceClient
		if a != nil {
			client = a.client
		}
		source, err := cloudread.Capture(ctx, client, "identity")
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		guard := func(ctx context.Context) error {
			var replacement error
			if a == nil || a.client != client {
				replacement = fmt.Errorf("%w: selected users API client changed", resource.ErrInvalidOption)
			}
			return errors.Join(replacement, source.Guard(ctx), rest.CheckOperationGuard(ctx))
		}
		parameters := projectListParameters{query: make(url.Values)}
		if policy.prepare != nil {
			parameters, err = policy.prepare(ctx, guard)
		}
		if err == nil {
			err = source.WithPolicy(ctx, parameters.microversion, parameters.headers)
		}
		if err == nil {
			err = guard(ctx)
		}
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		selected := rest.CollectionSpec[UserProjectRecord]{
			Client: &source.Client, Path: "users/" + url.PathEscape(userID) + "/" + policy.plural,
			Kind: policy.kind, PluralKey: policy.plural,
			Metadata: userProjectMetadata, Validate: guard, SourceGuard: guard,
			ListCodes:    []int{http.StatusOK, http.StatusNoContent},
			ValidateItem: func(value *UserProjectRecord) error { return policy.project(value, userID) },
			Paging: rest.PagePolicy[UserProjectRecord]{
				LinkKeys: []string{"links", policy.plural + "_links"}, NextKey: "next", DictionaryLinks: true,
				HTTPLink: true, MarkerFallback: policy.marker != nil, Marker: policy.marker,
				AllowFirstServerLimit: true,
				MarkerOnShortPage:     true, MaxItemsLimitHint: true, StopOnEmptyPage: true,
			},
		}
		var sequence iter.Seq2[*UserProjectRecord, error]
		if policy.iterate != nil {
			sequence = policy.iterate(ctx, selected, parameters)
		} else {
			sequence = rest.ListWithControl(ctx, selected, parameters.query, parameters.control)
		}
		for value, err := range sequence {
			if !yield(value, wrap(err)) {
				return
			}
		}
	}
}
