package attributestags

import (
	"bytes"
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

// ResourceKind names the ten tagged resource classes in the pinned Network
// proxy. The SDK owns their tag routes; callers never supply a URL or builder.
type ResourceKind string

const (
	NetworkResource           ResourceKind = "network"
	PortResource              ResourceKind = "port"
	SubnetResource            ResourceKind = "subnet"
	SubnetPoolResource        ResourceKind = "subnet-pool"
	RouterResource            ResourceKind = "router"
	FloatingIPResource        ResourceKind = "floating-ip"
	SecurityGroupResource     ResourceKind = "security-group"
	SecurityGroupRuleResource ResourceKind = "security-group-rule"
	QoSPolicyResource         ResourceKind = "qos-policy"
	TrunkResource             ResourceKind = "trunk"
)

// TaggedResource is an immutable resource reference with a local tag snapshot.
// A fresh reference has the Source default [], without fetching its parent.
// Mutations return a new reference; that cache is distinct from HTTP evidence.
type TaggedResource struct {
	kind ResourceKind
	id   string
	tags []string
}

func NewTaggedResource(kind ResourceKind, id string) TaggedResource {
	return TaggedResource{kind: kind, id: id, tags: []string{}}
}
func (r TaggedResource) Kind() ResourceKind { return r.kind }
func (r TaggedResource) ID() string         { return r.id }
func (r TaggedResource) Tags() []string     { return slices.Clone(r.tags) }

// WithCachedTags snapshots known local tags. A nil list preserves an explicit
// null, as does a fetched tags:null; it is not replaced by the fresh [] default.
func (r TaggedResource) WithCachedTags(tags []string) TaggedResource {
	r.tags = slices.Clone(tags)
	return r
}

// TagResult separates the Source-style cache transition from the actual HTTP
// response. ObservedTags is true only when GetTags decoded a tags field.
type TagResult struct {
	Resource     TaggedResource
	ObservedTags bool
	Body         json.RawMessage
	Header       http.Header
	StatusCode   int
}

type TagOpts struct{ SourceSecurityGroupRuleTags bool }

// WithSourceSecurityGroupRuleTags enables the pinned Python SGR tag route on a
// backend that supports it. Stock Neutron 27.0.0 has no SGR tag-controller
// registration, so this compatibility capability is disabled by default.
func WithSourceSecurityGroupRuleTags() TagOption {
	return func(c *request.Config[TagOpts]) error { c.Options.SourceSecurityGroupRuleTags = true; return nil }
}

type TagOption = request.Option[TagOpts]

func WithTagHeader(key, value string) TagOption { return request.WithHeader[TagOpts](key, value) }

// WithTagField snapshots an optional body extension for SetTags/AddTags. The
// tags field itself is owned by the method argument; other methods are bodyless.
func WithTagField(key string, value any) TagOption { return request.WithField[TagOpts](key, value) }

func tagInvalid(message string) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
}
func tagWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "attributestags", cloudread.ContextError(ctx, err))
}
func tagText(value string) bool {
	return utf8.ValidString(value) && strings.IndexFunc(value, unicode.IsControl) < 0
}
func tagSegment(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !tagText(value) {
		return tagInvalid("resource ID and individual tag must be nonblank literal UTF-8 text without controls")
	}
	return nil
}
func tagPath(kind ResourceKind) (string, bool) {
	switch kind {
	case NetworkResource:
		return "networks", true
	case PortResource:
		return "ports", true
	case SubnetResource:
		return "subnets", true
	case SubnetPoolResource:
		return "subnetpools", true
	case RouterResource:
		return "routers", true
	case FloatingIPResource:
		return "floatingips", true
	case SecurityGroupResource:
		return "security-groups", true
	case SecurityGroupRuleResource:
		return "security-group-rules", true
	// Neutron's shared tag controller is registered under the parent collection
	// 'policies', without the normal QoS /qos prefix. Source set_tags overrides
	// this path correctly; its other inherited TagMixin routes omit that fix.
	case QoSPolicyResource:
		return "policies", false
	case TrunkResource:
		return "trunks", false
	default:
		return "", false
	}
}
func tagCodes() (accepted, rejected []int) {
	accepted, rejected = make([]int, 200), make([]int, 200)
	for i := range accepted {
		accepted[i] = 200 + i
		rejected[i] = 400 + i
	}
	return
}

type preparedTag struct {
	ctx      context.Context
	source   *cloudread.Source
	guard    func(context.Context) error
	target   string
	resource TaggedResource
	config   request.Config[TagOpts]
}

func (a *API) prepareTag(ctx context.Context, input TaggedResource, tag *string, body, bulkAdd bool, options []TagOption) (*preparedTag, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, tagInvalid("configured Neutron tag API is required")
	}
	path, canAdd := tagPath(input.kind)
	if path == "" {
		return nil, fmt.Errorf("%w: resource kind %q has no pinned tag capability", resource.ErrUnsupported, input.kind)
	}
	if bulkAdd && !canAdd {
		return nil, fmt.Errorf("%w: pinned %s Resource has no add_tags method", resource.ErrUnsupported, input.kind)
	}
	if err := tagSegment(input.id); err != nil {
		return nil, err
	}
	if tag != nil {
		if err := tagSegment(*tag); err != nil {
			return nil, err
		}
	}
	input.tags = slices.Clone(input.tags)
	for _, cached := range input.tags {
		if !utf8.ValidString(cached) {
			return nil, tagInvalid("cached tags must be UTF-8 strings")
		}
	}
	original := a.client
	source, err := cloudread.Capture(ctx, original, "network")
	if err != nil {
		return nil, err
	}
	var observed error
	guard := func(ctx context.Context) error {
		if observed == nil && a.client != original {
			observed = tagInvalid("selected Neutron tag API changed")
		}
		return errors.Join(cloudread.ContextError(ctx, observed), source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err := guard(ctx); err != nil {
		return nil, err
	}
	copied := slices.Clone(options)
	guarded := make([]TagOption, len(copied))
	for index, option := range copied {
		apply := option
		guarded[index] = func(c *request.Config[TagOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return tagInvalid("nil tag option")
			}
			cloneTagConfig(c)
			err := apply(c)
			cloneTagConfig(c)
			return errors.Join(err, guard(ctx))
		}
	}
	config, err := request.Apply(TagOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, body, false, true)
	}
	if err == nil && config.Options.SourceSecurityGroupRuleTags && input.kind != SecurityGroupRuleResource {
		err = tagInvalid("Source security-group-rule tag capability requires that resource kind")
	}
	if err == nil && input.kind == SecurityGroupRuleResource && !config.Options.SourceSecurityGroupRuleTags {
		err = fmt.Errorf("%w: stock Neutron 27.0.0 has no security-group-rule tag controller; opt into the explicit Source compatibility capability", resource.ErrUnsupported)
	}
	if err == nil {
		for key, value := range config.Fields {
			if key == "tags" {
				err = tagInvalid("tags body field is owned by the method argument")
				break
			}
			if strings.TrimSpace(key) == "" || !tagText(key) || !utf8.Valid(value) || !json.Valid(value) {
				err = tagInvalid("tag extension needs a nonblank UTF-8 key and JSON value")
				break
			}
		}
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return nil, err
	}
	target := source.Client.ServiceURL(path, url.PathEscape(input.id), "tags")
	if tag != nil {
		target += "/" + url.PathEscape(*tag)
	}
	return &preparedTag{ctx: ctx, source: source, guard: guard, target: target, resource: input, config: config}, nil
}
func cloneTagConfig(c *request.Config[TagOpts]) {
	fields := make(map[string]json.RawMessage, len(c.Fields))
	for key, value := range c.Fields {
		fields[key] = bytes.Clone(value)
	}
	c.Fields = fields
	c.Headers = maps.Clone(c.Headers)
	if c.Headers == nil {
		c.Headers = make(map[string]string)
	}
	query := make(url.Values, len(c.Query))
	for key, value := range c.Query {
		query[key] = slices.Clone(value)
	}
	c.Query = query
	c.Arguments = maps.Clone(c.Arguments)
	if c.Arguments == nil {
		c.Arguments = make(map[string]any)
	}
}

// headerPolicy keeps protocol and auth/framing exclusions through native retry
// hooks. The shared live X-Auth-Token stays owned by fixedrequest authentication;
// ordinary headers retain native retry policy instead of being frozen wholesale.
func (p *preparedTag) headerPolicy(body bool) rest.RequestHeaderPolicy {
	policy := rest.RequestHeaderPolicy{
		Values: make(map[string]string),
		Absent: []string{
			"Authorization", "X-Service-Token", "Cookie", "Host",
			"Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade",
			"X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version",
		},
	}
	if p.source.Client.Microversion == "" {
		policy.Absent = append(policy.Absent, "OpenStack-API-Version")
	} else {
		policy.Values["OpenStack-API-Version"] = "network " + p.source.Client.Microversion
	}
	if body {
		contentType := "application/json"
		for key, value := range p.source.Client.MoreHeaders {
			if strings.EqualFold(key, "Content-Type") {
				contentType = value
			}
		}
		policy.Values["Content-Type"] = contentType
	}
	return policy
}

func (p *preparedTag) do(method string, tags []string, body bool) (*TagResult, error) {
	var payload any
	if body {
		for _, tag := range tags {
			if !utf8.ValidString(tag) {
				return nil, tagInvalid("tag list contains non-UTF-8 text")
			}
		}
		fields := map[string]any{"tags": slices.Clone(tags)}
		value, err := request.MergeFields(fields, p.config.Fields)
		if err != nil {
			return nil, errors.Join(err, p.guard(p.ctx))
		}
		payload = value
	}
	if err := p.guard(p.ctx); err != nil {
		return nil, err
	}
	accepted, rejected := tagCodes()
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, method, p.target, payload, p.headerPolicy(body), rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true}, accepted...)
	if response == nil {
		return nil, err
	}
	result := &TagResult{Resource: p.resource, Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	return result, err
}
func tagResponseFailure(ctx context.Context, result *TagResult, err error) error {
	err = cloudread.ContextError(ctx, err)
	if result == nil {
		return err
	}
	return (&rest.Response{Body: result.Body, Header: result.Header, StatusCode: result.StatusCode}).Fail(err)
}

// GetTags executes one GET and returns the Source cached/default tags when the
// JSON object omits tags. Present [] and null are distinct and update the cache.
func (a *API) GetTags(ctx context.Context, input TaggedResource, options ...TagOption) (*TagResult, error) {
	const op = "GetTags"
	p, err := a.prepareTag(ctx, input, nil, false, false, options)
	if err != nil {
		return nil, tagWrap(ctx, op, err)
	}
	result, err := p.do(http.MethodGet, nil, false)
	if err != nil {
		return result, tagWrap(ctx, op, err)
	}
	var fields map[string]json.RawMessage
	if !utf8.Valid(result.Body) {
		err = tagInvalid("tag response must be UTF-8 JSON")
	} else {
		err = json.Unmarshal(result.Body, &fields)
	}
	if err == nil && fields == nil {
		err = tagInvalid("tag response must be a JSON object")
	}
	if err == nil {
		if raw, present := fields["tags"]; present {
			var tags []string
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				tags = nil
			} else {
				var elements []json.RawMessage
				err = json.Unmarshal(raw, &elements)
				if err == nil && elements == nil {
					err = tagInvalid("tags must be an array or null")
				}
				if err == nil {
					tags = make([]string, len(elements))
					for i, element := range elements {
						trimmed := bytes.TrimSpace(element)
						if len(trimmed) == 0 || trimmed[0] != '"' {
							err = tagInvalid("tag response array elements must be strings")
							break
						}
						if err = json.Unmarshal(trimmed, &tags[i]); err != nil {
							break
						}
					}
				}
			}
			if err == nil {
				result.Resource.tags = tags
				result.ObservedTags = true
			}
		}
	}
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return result, tagWrap(ctx, op, tagResponseFailure(ctx, result, err))
	}
	return result, nil
}

// SetTags replaces every remote tag in one PUT. Nil is explicit JSON null;
// an empty nonnil list is []. The returned cache records caller tags, while Body
// retains the server response independently (Source ignores mutation bodies).
func (a *API) SetTags(ctx context.Context, input TaggedResource, tags []string, options ...TagOption) (*TagResult, error) {
	return a.changeTags(ctx, "SetTags", http.MethodPut, input, slices.Clone(tags), false, options)
}

// AddTags posts additional tags in one POST on the eight TagMixinNetwork kinds.
// Nil becomes [], as in its tags-or-[] default. Source updates the local cache
// to the supplied list, not a union; fetch again to observe all remote tags.
func (a *API) AddTags(ctx context.Context, input TaggedResource, tags []string, options ...TagOption) (*TagResult, error) {
	owned := slices.Clone(tags)
	if owned == nil {
		owned = []string{}
	}
	return a.changeTags(ctx, "AddTags", http.MethodPost, input, owned, true, options)
}
func (a *API) changeTags(ctx context.Context, op, method string, input TaggedResource, tags []string, add bool, options []TagOption) (*TagResult, error) {
	p, err := a.prepareTag(ctx, input, nil, true, add, options)
	if err != nil {
		return nil, tagWrap(ctx, op, err)
	}
	result, err := p.do(method, tags, true)
	if err != nil {
		return result, tagWrap(ctx, op, err)
	}
	result.Resource.tags = slices.Clone(tags)
	if err := p.guard(ctx); err != nil {
		return result, tagWrap(ctx, op, tagResponseFailure(ctx, result, err))
	}
	return result, nil
}

// AddTag appends to the returned local cache, including a duplicate. It sends
// one bodyless PUT; the server controls uniqueness, length and authorization.
func (a *API) AddTag(ctx context.Context, input TaggedResource, tag string, options ...TagOption) (*TagResult, error) {
	return a.singleTag(ctx, "AddTag", http.MethodPut, input, tag, true, options)
}

// RemoveTag sends one strict bodyless DELETE and removes only the first cached
// occurrence. An absent local tag is harmless; HTTP 404 remains an error.
func (a *API) RemoveTag(ctx context.Context, input TaggedResource, tag string, options ...TagOption) (*TagResult, error) {
	return a.singleTag(ctx, "RemoveTag", http.MethodDelete, input, tag, false, options)
}
func (a *API) singleTag(ctx context.Context, op, method string, input TaggedResource, tag string, add bool, options []TagOption) (*TagResult, error) {
	p, err := a.prepareTag(ctx, input, &tag, false, false, options)
	if err != nil {
		return nil, tagWrap(ctx, op, err)
	}
	result, err := p.do(method, nil, false)
	if err != nil {
		return result, tagWrap(ctx, op, err)
	}
	// Source's .append/.remove also fails after a successful HTTP action when a
	// cached tags:null is used. Keep the actual acknowledgement in this error.
	if result.Resource.tags == nil {
		return result, tagWrap(ctx, op, tagResponseFailure(ctx, result, tagInvalid("individual tag cache transition requires a list; cached tags is null")))
	}
	if add {
		result.Resource.tags = append(slices.Clone(result.Resource.tags), tag)
	} else {
		for i, value := range result.Resource.tags {
			if value == tag {
				result.Resource.tags = append(slices.Clone(result.Resource.tags[:i]), result.Resource.tags[i+1:]...)
				break
			}
		}
	}
	if err := p.guard(ctx); err != nil {
		return result, tagWrap(ctx, op, tagResponseFailure(ctx, result, err))
	}
	return result, nil
}

// RemoveAllTags clears the returned cache after a strict bodyless DELETE.
func (a *API) RemoveAllTags(ctx context.Context, input TaggedResource, options ...TagOption) (*TagResult, error) {
	const op = "RemoveAllTags"
	p, err := a.prepareTag(ctx, input, nil, false, false, options)
	if err != nil {
		return nil, tagWrap(ctx, op, err)
	}
	result, err := p.do(http.MethodDelete, nil, false)
	if err != nil {
		return result, tagWrap(ctx, op, err)
	}
	result.Resource.tags = []string{}
	if err := p.guard(ctx); err != nil {
		return result, tagWrap(ctx, op, tagResponseFailure(ctx, result, err))
	}
	return result, nil
}

// CheckTag preserves the actual Source behavior: return the resource on an
// accepted bodyless GET and raise HTTP 404, rather than inventing false/nil from
// the proxy's inaccurate bool annotation. Native Confirm remains separate.
func (a *API) CheckTag(ctx context.Context, input TaggedResource, tag string, options ...TagOption) (*TagResult, error) {
	const op = "CheckTag"
	p, err := a.prepareTag(ctx, input, &tag, false, false, options)
	if err != nil {
		return nil, tagWrap(ctx, op, err)
	}
	result, err := p.do(http.MethodGet, nil, false)
	return result, tagWrap(ctx, op, err)
}
