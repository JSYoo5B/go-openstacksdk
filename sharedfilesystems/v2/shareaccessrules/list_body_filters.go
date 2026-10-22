package shareaccessrules

import (
	"encoding/json"
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

const listBodyFiltersArgument = "shareaccessrules.body_filters"

// accessRuleBodyFilterFields lists the pinned Python ShareAccessRule Body
// attributes (openstack/shared_file_system/v2/share_access_rule.py plus the
// inherited id and name) that Resource.list treats as client-side filters.
// share_id is a query parameter there and is owned by the share scope.
var accessRuleBodyFilterFields = map[string]bool{
	"id": true, "name": true, "access_key": true, "access_level": true, "access_list": true,
	"access_to": true, "access_type": true, "created_at": true, "metadata": true, "state": true,
	"updated_at": true, "lock_visibility": true, "lock_deletion": true, "lock_reason": true,
}

// WithListBodyFilter adds a local filter like the client-side Body filters of
// pinned Python Resource.list. name is a Python ShareAccessRule attribute (id,
// name, access_key, access_level, access_list, access_to, access_type,
// created_at, metadata, state, updated_at, lock_visibility, lock_deletion or
// lock_reason); share_id is fixed by the scope. value is encoded to JSON when
// the option is created and is never sent to the server. A rule is yielded
// only when every filter matches the raw response: object filters match a
// subset of a nonempty response object, other values compare exactly and a
// missing field compares as null (internal/jsonfilter.MatchFilters). Unknown
// names and encoding errors fail before HTTP. Repeating a name replaces its
// value. WithListMaxItems counts raw rows before filtering, as Python's
// max_items does.
func WithListBodyFilter(name string, value any) ListOption {
	encoded, err := json.Marshal(value)
	return func(cfg *request.Config[ListOpts]) error {
		if err != nil {
			return invalid("access rule body filter %q: %v", name, err)
		}
		if !accessRuleBodyFilterFields[name] {
			return invalid("unknown access rule body filter %q", name)
		}
		filters, _, argErr := request.Argument[map[string]json.RawMessage](*cfg, listBodyFiltersArgument)
		if argErr != nil {
			return argErr
		}
		owned := maps.Clone(filters)
		if owned == nil {
			owned = make(map[string]json.RawMessage)
		}
		owned[name] = append(json.RawMessage(nil), encoded...)
		if cfg.Arguments == nil {
			cfg.Arguments = make(map[string]any)
		}
		cfg.Arguments[listBodyFiltersArgument] = owned
		return nil
	}
}

func accessRuleBodyFilters(cfg request.Config[ListOpts]) (map[string]json.RawMessage, error) {
	filters, _, err := request.Argument[map[string]json.RawMessage](cfg, listBodyFiltersArgument)
	if err != nil {
		return nil, err
	}
	for name, raw := range filters {
		if !accessRuleBodyFilterFields[name] || !json.Valid(raw) {
			return nil, invalid("invalid access rule body filter %q", name)
		}
	}
	return filters, nil
}

func matchAccessRule(rule *AccessRule, filters map[string]json.RawMessage) (bool, error) {
	if len(filters) == 0 {
		return true, nil
	}
	return jsonfilter.MatchFilters(rule.Body, filters)
}
