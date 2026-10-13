package groups

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordListGenerator is the opaque Source Proxy generator, never an eager
// JSON list. Its records retain the captured context and operation source guard.
type RecordListGenerator struct {
	Records iter.Seq2[*SecurityGroupRecord, error]
}

func (g *RecordListGenerator) JMESPathIdentity() any { return g }
func (g *RecordListGenerator) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("%w: opaque security group generator", resource.ErrUnsupported)
}

type RecordProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

func (a *API) securityGroupSourceListEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if e := cloudread.Context(ctx); e != nil {
		return nil, e
	}
	if a == nil {
		return nil, findInvalid("configured Security Groups API required")
	}
	selected := a.client
	profile := securityGroupListProfile()
	profile.SourceStatusPolicy, profile.SourceLinkPolicy, profile.PythonLoadedEquality = true, true, true
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected Security Groups API changed")
		}
		return nil
	})
}
func WithRecordProxyQuery(key string, value any) RecordListOption {
	raw, e := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if e != nil {
			return e
		}
		if c.Query == nil {
			c.Query = map[string]json.RawMessage{}
		}
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordListBasePath(value string) RecordListOption {
	return func(c *RecordListOpts) error { x := value; c.BasePath = &x; return nil }
}

// ProxyListRecords consumes Source formal kwargs without changing the existing
// typed ListRecords status, reserved-argument or equality policies.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	const op = "ProxyListRecords"
	wrap := func(e error) error { return request.Wrap(op, "security groups", cloudread.ContextError(ctx, e)) }
	engine, e := a.securityGroupSourceListEngine(ctx)
	if e != nil {
		return nil, wrap(e)
	}
	cfg := copyRecordListOpts(RecordListOpts{})
	for _, apply := range slices.Clone(options) {
		if e := engine.Check(ctx); e != nil {
			return nil, wrap(e)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil Source security group list option"))
		}
		next := copyRecordListOpts(cfg)
		if e := errors.Join(apply(&next), engine.Check(ctx)); e != nil {
			return nil, wrap(e)
		}
		cfg = copyRecordListOpts(next)
	}
	normalized, expression, e := neutronfind.NormalizeProxyListOptions(recordListOptions(cfg))
	if e = errors.Join(e, engine.Check(ctx)); e != nil {
		return nil, wrap(e)
	}
	if normalized.BasePath != nil {
		path, e := securityGroupProxyPath(*normalized.BasePath, normalized.Query)
		if e != nil {
			return nil, wrap(e)
		}
		normalized.BasePath = &path
	}
	if raw, present := normalized.Query["max_items"]; present {
		truthy, e := cloudfilter.PythonJSONTruthy(raw)
		if e != nil {
			return nil, wrap(e)
		}
		if !truthy {
			delete(normalized.Query, "max_items")
			normalized.Order = slices.DeleteFunc(normalized.Order, func(k string) bool { return k == "max_items" })
		} else if bytes.Equal(bytes.TrimSpace(raw), []byte("true")) {
			normalized.Query["max_items"] = json.RawMessage("1")
		}
	}
	generator := &RecordListGenerator{Records: func(yield func(*SecurityGroupRecord, error) bool) {
		if normalized.BasePath != nil && *normalized.BasePath == "" {
			return
		}
		for value, e := range engine.List(ctx, normalized) {
			if !yield(securityGroupListRecord(value), wrap(e)) {
				return
			}
		}
	}}
	var value any = generator
	if expression != nil {
		truthy, e := cloudfilter.PythonJSONTruthy(expression)
		if e != nil {
			return nil, wrap(e)
		}
		if truthy && len(bytes.TrimSpace(expression)) > 0 && bytes.TrimSpace(expression)[0] == '"' {
			var text string
			if e := json.Unmarshal(expression, &text); e != nil {
				return nil, wrap(e)
			}
			value, e = jmespath.Search(text, generator)
			if e != nil {
				return nil, wrap(e)
			}
		}
	}
	result := &RecordProxyListResult{Value: value}
	result.JSON, _ = json.Marshal(value)
	if e := engine.Check(ctx); e != nil {
		return nil, wrap(e)
	}
	return result, nil
}

// GetSourceRecord is the Source named direct-ID getter, with no Find fallback.
func (a *API) GetSourceRecord(ctx context.Context, id string, options ...RecordListOption) (*SecurityGroupRecord, error) {
	const op = "GetSourceRecord"
	wrap := func(e error) error { return request.Wrap(op, "security groups", cloudread.ContextError(ctx, e)) }
	engine, e := a.securityGroupSourceListEngine(ctx)
	if e != nil {
		return nil, wrap(e)
	}
	cfg := copyRecordListOpts(RecordListOpts{})
	for _, apply := range slices.Clone(options) {
		if e := engine.Check(ctx); e != nil {
			return nil, wrap(e)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil member option"))
		}
		next := copyRecordListOpts(cfg)
		if e := errors.Join(apply(&next), engine.Check(ctx)); e != nil {
			return nil, wrap(e)
		}
		cfg = copyRecordListOpts(next)
	}
	if len(cfg.Query) > 0 || cfg.BasePath != nil {
		return nil, wrap(findInvalid("Source named member getter accepts no filter kwargs"))
	}
	value, e := engine.Get(ctx, id, recordListOptions(cfg))
	return securityGroupListRecord(value), wrap(e)
}

func securityGroupProxyPath(path string, attrs map[string]json.RawMessage) (string, error) {
	var out strings.Builder
	for i := 0; i < len(path); {
		if path[i] != '%' {
			out.WriteByte(path[i])
			i++
			continue
		}
		if i+1 < len(path) && path[i+1] == '%' {
			out.WriteString("%25")
			i += 2
			continue
		}
		if i+1 >= len(path) || path[i+1] != '(' {
			return "", fmt.Errorf("%w: Source route requires %%(key)s", resource.ErrUnsupported)
		}
		end := strings.IndexByte(path[i+2:], ')')
		if end < 0 {
			return "", findInvalid("malformed Source route")
		}
		end += i + 2
		if end+1 >= len(path) || path[end+1] != 's' {
			return "", fmt.Errorf("%w: Source route supports string formatting", resource.ErrUnsupported)
		}
		key := path[i+2 : end]
		raw, ok := attrs[key]
		if !ok {
			return "", findInvalid("missing route parameter %q", key)
		}
		text, e := cloudfilter.PythonString(raw)
		if e != nil {
			return "", e
		}
		out.WriteString(url.PathEscape(text))
		i = end + 2
	}
	return out.String(), nil
}
