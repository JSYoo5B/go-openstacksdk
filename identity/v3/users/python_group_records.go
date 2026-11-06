package users

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/keystonecalls"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GroupRecordListOpts reuses User declarations while keeping the Group route
// fixed. BasePath is rejected, including a present null raw base_path kwarg.
type GroupRecordListOpts RecordListOpts

// GroupRecordListOption is one option of ListUsersInGroupRecords and
// AllUsersInGroupRecords: WithGroupRecordListOpts, WithGroupRecordListQuery,
// WithGroupRecordListLocation, WithGroupRecordListJMESPath, request.Custom
// over *GroupRecordListOpts, request.Headers (replaces the headers),
// request.Microversion, request.Limit, request.Marker, request.MaxItems or
// request.Paginated (each a Source keyword argument). request.Header and
// request.Query fail with resource.ErrInvalidOption before HTTP:
// WithGroupRecordListQuery rejects the bound base_path and group arguments
// and discards ignored kwargs before marshaling the value.
type GroupRecordListOption = request.ListCall

var groupRecordList = keystonecalls.SourceList[GroupRecordListOpts]{
	State: func(c *GroupRecordListOpts) keystonecalls.ListState {
		return keystonecalls.ListState{Query: &c.Query, Headers: &c.Headers, Microversion: &c.Microversion, BasePath: &c.BasePath, Order: &c.order}
	},
	Control: recordListControl,
	Ignores: recordListIgnores,
	Invalid: recordInvalid,
}

// applyGroupRecordListQuery rejects the bound group_users arguments before
// applying a Source keyword argument.
func applyGroupRecordListQuery(c *GroupRecordListOpts, key string, raw json.RawMessage, err error) error {
	if key == "base_path" || key == "group" {
		return recordInvalid("group_users argument %q is already bound", key)
	}
	return groupRecordList.Apply(c, key, raw, err)
}

var groupRecordListCalls = func() calls.Spec[GroupRecordListOpts] {
	spec := groupRecordList.ControlSpec()
	spec.Paging = calls.JSONPaging(applyGroupRecordListQuery)
	spec.BasePath = nil
	return spec
}()

func copyGroupRecordOpts(v GroupRecordListOpts) GroupRecordListOpts {
	return GroupRecordListOpts(copyRecordListOpts(RecordListOpts(v)))
}
func groupFixedPath(v RecordListOpts) error {
	if v.BasePath != nil {
		return recordInvalid("group_users already binds base_path")
	}
	for _, key := range []string{"base_path", "group"} {
		if _, present := v.Query[key]; present {
			return recordInvalid("group_users argument %q is already bound", key)
		}
	}
	return nil
}
func WithGroupRecordListOpts(v GroupRecordListOpts) GroupRecordListOption {
	owned := copyGroupRecordOpts(v)
	return request.Custom(func(c *GroupRecordListOpts) error {
		*c = copyGroupRecordOpts(owned)
		return groupFixedPath(RecordListOpts(*c))
	})
}

// WithGroupRecordListQuery sets one Source keyword argument, as
// WithRecordListQuery does; the group route binds base_path and group.
func WithGroupRecordListQuery(key string, value any) GroupRecordListOption {
	if key == "base_path" || key == "group" {
		err := recordInvalid("group_users argument %q is already bound", key)
		return request.Custom(func(*GroupRecordListOpts) error { return err })
	}
	return groupRecordList.Query(key, value)
}
func WithGroupRecordListLocation(v resource.CloudLocation) GroupRecordListOption {
	owned := v.Clone()
	return request.Custom(func(c *GroupRecordListOpts) error { location := owned.Clone(); c.Location = &location; return nil })
}
func WithGroupRecordListJMESPath(v string) GroupRecordListOption {
	return keystonecalls.Keyword(applyGroupRecordListQuery, "jmespath_filters", v)
}
func groupIdentity(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return recordInvalid("group identity must be a nonblank UTF-8 literal without controls or traversal")
	}
	return nil
}
func (a *API) groupRecordEngine(ctx context.Context, groupID string) (*neutronfind.Engine, error) {
	if err := errors.Join(cloudread.Context(ctx), groupIdentity(groupID)); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("configured users API required")
	}
	selected := a.client
	profile := recordProfile
	// Source group_users uses User, not a reduced GroupUser subclass. Only its
	// collection route changes, and Engine.New owns copied descriptor slices.
	profile.Path = "groups/" + url.PathEscape(groupID) + "/users"
	profile.VersionedPath = "/v3/" + profile.Path
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return recordInvalid("selected group users client changed")
		}
		return nil
	})
}
func applyGroupRecordOptions(ctx context.Context, engine *neutronfind.Engine, options []GroupRecordListOption) (RecordListOpts, error) {
	converted := make([]func(*RecordListOpts) error, 0, len(options))
	for _, apply := range calls.Translate(options, groupRecordListCalls) {
		if apply == nil {
			converted = append(converted, nil)
			continue
		}
		converted = append(converted, func(c *RecordListOpts) error {
			next := copyGroupRecordOpts(GroupRecordListOpts(*c))
			if err := apply(&next); err != nil {
				return err
			}
			v := RecordListOpts(next)
			if err := groupFixedPath(v); err != nil {
				return err
			}
			*c = copyRecordListOpts(v)
			return nil
		})
	}
	return applyRecordListFuncs(ctx, engine, converted)
}

// ListUsersInGroupRecords mirrors Proxy.group_users using the complete User profile.
// HTTP is lazy; the immutable literal group ID owns every continuation route.
func (a *API) ListUsersInGroupRecords(ctx context.Context, groupID string, options ...GroupRecordListOption) iter.Seq2[*UserRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*UserRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListUsersInGroupRecords", "users", cloudread.ContextError(ctx, err))
		}
		engine, err := a.groupRecordEngine(ctx, groupID)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		policy, err := applyGroupRecordOptions(ctx, engine, owned)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		expression, err := separateRecordListExpression(&policy)
		if err == nil && expression != nil {
			err = recordInvalid("JMESPath changes the return type; use ProxyUsersInGroupRecords")
		}
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		for value, err := range engine.List(ctx, recordListOptions(policy)) {
			row, projectErr := recordListRecord(value, policy)
			if !yield(row, wrap(errors.Join(err, projectErr, engine.Check(ctx)))) {
				return
			}
		}
	}
}
func (a *API) AllUsersInGroupRecords(ctx context.Context, groupID string, options ...GroupRecordListOption) ([]*UserRecord, error) {
	rows := make([]*UserRecord, 0)
	for row, err := range a.ListUsersInGroupRecords(ctx, groupID, options...) {
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// ProxyUsersInGroupRecords preserves Source's opaque generator JMESPath behavior.
func (a *API) ProxyUsersInGroupRecords(ctx context.Context, groupID string, options ...GroupRecordListOption) (*RecordProxyListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ProxyUsersInGroupRecords", "users", cloudread.ContextError(ctx, err))
	}
	engine, err := a.groupRecordEngine(ctx, groupID)
	if err != nil {
		return nil, wrap(err)
	}
	policy, err := applyGroupRecordOptions(ctx, engine, slices.Clone(options))
	if err != nil {
		return nil, wrap(err)
	}
	value, err := recordProxyListValue(ctx, engine, policy, "ProxyUsersInGroupRecords")
	if err != nil {
		return nil, wrap(err)
	}
	result := &RecordProxyListResult{Value: value}
	if recordProxyJSONValue(value) {
		result.JSON, err = json.Marshal(value)
	}
	return result, wrap(errors.Join(err, engine.Check(ctx)))
}
