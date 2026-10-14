package networks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WithCloudRecordListQuery snapshots JSON kwargs without evaluating Resource.list
// formals until its generator is consumed. Use this with ProxyCloudListRecords;
// the older typed record APIs retain their eager option-validation contract.
func WithCloudRecordListQuery(key string, value json.RawMessage) RecordListOption {
	owned := bytes.Clone(value)
	return func(c *RecordListOpts) error {
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}

// networkCloudListProxyOptions performs only Python argument binding and the
// Proxy's one conflicting-attrs merge. Resource.list validation remains lazy.
func networkCloudListProxyOptions(ctx context.Context, engine *neutronfind.Engine, options []RecordListOption) (RecordListOpts, error) {
	policy := RecordListOpts{}
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, findInvalid("nil networks list option")
		}
		next := copyRecordListOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = copyRecordListOpts(next)
	}
	pop := func(key string) json.RawMessage {
		v := bytes.Clone(policy.Query[key])
		delete(policy.Query, key)
		policy.order = slices.DeleteFunc(policy.order, func(k string) bool { return k == key })
		return v
	}
	for _, key := range []string{"self", "resource_type"} {
		if _, present := policy.Query[key]; present {
			return policy, findInvalid("%q collides with bound Proxy argument", key)
		}
	}
	policy.expression = pop("jmespath_filters")
	base := pop("base_path")
	conflict := pop("__conflicting_attrs")
	if conflict != nil {
		truthy, err := cloudfilter.PythonJSONTruthy(conflict)
		if err != nil {
			return policy, err
		}
		if truthy {
			members, err := cloudfilter.ObjectMembers(conflict)
			if err != nil {
				return policy, findInvalid("truthy __conflicting_attrs must be an object: %v", err)
			}
			for _, m := range members {
				if m.Key == "base_path" || m.Key == "paginated" {
					return policy, findInvalid("restored %q duplicates explicit Resource.list argument", m.Key)
				}
				if !slices.Contains(policy.order, m.Key) {
					policy.order = append(policy.order, m.Key)
				}
				policy.Query[m.Key] = bytes.Clone(m.Value)
			}
		}
	}
	pop("__conflicting_attrs")
	pop("jmespath_filters") // restored spelling is an unknown Resource kwarg
	for _, key := range []string{"cls", "session"} {
		if _, ok := policy.Query[key]; ok {
			return policy, findInvalid("%q collides with a bound Resource/Proxy list argument", key)
		}
	}
	pop("self")
	pop("resource_type") // restored keys are ordinary unknown Resource.list params
	if base != nil {
		policy.Query["base_path"] = base
		policy.order = append(policy.order, "base_path")
	}
	return copyRecordListOpts(policy), nil
}

// Keystoneauth's JSON-domain microversion normalization is a transport policy:
// falsey explicit values suppress its header; numbers and iterable version
// parts are normalized before the concrete Gophercloud string header is used.
func networkCloudListVersion(raw json.RawMessage) (*string, error) {
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	text := ""
	if !truthy {
		return &text, nil
	}
	switch bytes.TrimSpace(raw)[0] {
	case '"':
		if err = json.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
		text = strings.TrimLeft(text, "v")
	case '[':
		var parts []json.RawMessage
		if err = json.Unmarshal(raw, &parts); err != nil {
			return nil, err
		}
		items := make([]string, len(parts))
		for i, p := range parts {
			items[i], err = cloudfilter.PythonString(p)
			if err != nil {
				return nil, err
			}
		}
		text = strings.Join(items, ".")
	case '{':
		parts, err := cloudfilter.ObjectMembers(raw)
		if err != nil {
			return nil, err
		}
		items := make([]string, len(parts))
		for i, p := range parts {
			items[i] = p.Key
		}
		text = strings.Join(items, ".")
	default:
		number, err := jsonfilter.DescriptorFloatJSON(raw)
		if err != nil {
			return nil, err
		}
		f, err := strconv.ParseFloat(string(number), 64)
		if err != nil {
			return nil, err
		}
		text, err = cloudfilter.PythonString(json.RawMessage(strconv.FormatFloat(f, 'e', -1, 64)))
		if err != nil {
			return nil, err
		}
	}
	parts := strings.Split(text, ".")
	if len(parts) == 1 && parts[0] == "latest" {
		return &text, nil
	}
	if len(parts) == 1 {
		parts = append(parts, "0")
	}
	for i, p := range parts {
		if p == "latest" {
			if i > 0 && parts[0] != "latest" {
				return nil, findInvalid("major.latest microversion is not allowed")
			}
			continue
		}
		value, err := jsonfilter.PythonIntegerString(p)
		if err != nil {
			return nil, err
		}
		parts[i] = value.String()
	}
	text = strings.Join(parts, ".")
	allLatest := true
	for _, p := range parts {
		allLatest = allLatest && p == "latest"
	}
	if allLatest {
		text = "latest"
	}
	return &text, nil
}

func networkCloudListResourceOptions(policy RecordListOpts) (RecordListOpts, error) {
	// This call is inside the generator. It must never reject unused raw
	// headers/base_path/microversion on a literal or nested opaque JMES result.
	if raw, ok := policy.Query["microversion"]; ok {
		version, err := networkCloudListVersion(raw)
		if err != nil {
			return policy, findInvalid("microversion: %v", err)
		}
		policy.Microversion = version
		delete(policy.Query, "microversion")
		policy.order = slices.DeleteFunc(policy.order, func(k string) bool { return k == "microversion" })
	}
	normalized, _, err := neutronfind.NormalizeProxyListOptions(recordListOptions(policy))
	if err != nil {
		return policy, err
	}
	policy.Query, policy.Headers, policy.Microversion, policy.BasePath, policy.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.BasePath, normalized.Order
	return copyRecordListOpts(policy), nil
}

func networkCloudListLocation(record *NetworkRecord, location *resource.CloudLocation) error {
	if record == nil || record.Resource == nil {
		return nil
	}
	if location == nil {
		record.Resource.Body["location"] = json.RawMessage("null")
		return nil
	}
	project := record.Resource.Body["project_id"]
	if project != nil {
		truthy, err := cloudfilter.PythonJSONTruthy(project)
		if err != nil {
			return err
		}
		if !truthy {
			project = nil
		} else {
			current := location.Project.ID
			if current == nil {
				current = json.RawMessage("null")
			}
			equal, err := jsonfilter.EqualPythonLoadedJSON(project, current)
			if err != nil {
				return err
			}
			if equal {
				project = nil
			}
		}
	}
	// Network has no tenant_id alias. Unknown availability_zone is still an
	// instance attribute used by Resource._update_location through hasattr.
	raw, err := location.ForResource(project, record.Resource.Body["availability_zone"])
	if err != nil {
		return err
	}
	record.Resource.Body["location"] = bytes.Clone(raw)
	return nil
}

// ProxyCloudListRecords is the connected Source list lane. Existing typed and
// record APIs keep their baseline; opaque JMES remains a lazy generator value.
func (a *API) ProxyCloudListRecords(ctx context.Context, location *resource.CloudLocation, options ...RecordListOption) (*RecordProxyListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ProxyCloudListRecords", "networks", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("configured networks API required"))
	}
	selected := a.client
	profile := networkListProfile()
	profile.SourceStatusPolicy, profile.SourceLinkPolicy = true, true
	engine, err := neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected networks client changed")
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	policy, err := networkCloudListProxyOptions(ctx, engine, slices.Clone(options))
	if err != nil {
		return nil, wrap(err)
	}
	expression, err := recordListExpression(&policy)
	if err != nil {
		return nil, wrap(err)
	}
	var ownedLocation *resource.CloudLocation
	if location != nil {
		copy := location.Clone()
		ownedLocation = &copy
	}
	generator := &RecordListGenerator{Records: func(yield func(*NetworkRecord, error) bool) {
		if cause := engine.Check(ctx); cause != nil {
			yield(nil, wrap(cause))
			return
		}
		ownedPolicy, cause := networkCloudListResourceOptions(copyRecordListOpts(policy))
		if cause != nil {
			yield(nil, wrap(cause))
			return
		}
		if ownedPolicy.BasePath != nil && *ownedPolicy.BasePath == "" {
			return
		} // Source while uri is false: no HTTP.
		for value, cause := range engine.List(ctx, recordListOptions(ownedPolicy)) {
			record := networkListRecord(value)
			if cause == nil {
				cause = networkCloudListLocation(record, ownedLocation)
				if cause != nil && record != nil {
					cause = (&rest.Response{Body: bytes.Clone(record.Envelope), Header: record.Header.Clone(), StatusCode: record.StatusCode}).Fail(cause)
				}
			}
			cause = errors.Join(cause, engine.Check(ctx))
			if !yield(record, wrap(cause)) {
				return
			}
			if cause != nil {
				return
			}
		}
	}}
	result := &RecordProxyListResult{Value: generator}
	if expression != "" {
		result.Value, err = jmespath.Search(expression, generator)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	if !networkProxyHasGenerator(result.Value) {
		result.JSON, err = json.Marshal(result.Value)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	return result, wrap(engine.Check(ctx))
}
