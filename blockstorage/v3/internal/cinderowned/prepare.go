package cinderowned

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cinderrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Version selects how one call chooses its Cinder microversion.
//
// Default sends the configured client microversion without discovery, as
// Python does when it passes session.default_microversion. Otherwise the call
// follows Resource._get_microversion: the configured microversion, or
// min(Ceiling, server maximum) from discovery. Required additionally runs
// utils.supports_microversion first and fails when it is false. Minimum
// follows Resource._assert_microversion_for and fails when the selected
// version is absent or lower.
type Version struct {
	Default  bool
	Ceiling  string
	Required string
	Minimum  string
}

// Target is one collection route: Path is the escaped collection path below
// the Cinder project endpoint.
type Target struct {
	Spec   *Spec
	Client func() *gophercloud.ServiceClient
	Path   string
}

// Prepared is one operation's captured Cinder client and source guard.
type Prepared struct {
	target    Target
	ctx       context.Context
	source    *cloudread.Source
	guard     func(context.Context) error
	discovery []Observation
	headers   map[string]string
}

// Codes is Python raise_from_response: every status from 200 through 399.
func Codes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}

func rejectedCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 400 + i
	}
	return codes
}

// Wrap adds operation context and any context cancellation cause.
func (t Target) Wrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, t.Spec.Kind, cloudread.ContextError(ctx, err))
}

// Wrapper returns a function that keeps a record and wraps its error.
func (t Target) Wrapper(ctx context.Context, op string) func(*Record, error) (*Record, error) {
	return func(r *Record, err error) (*Record, error) { return r, t.Wrap(ctx, op, err) }
}

// Prepare captures the selected Cinder client once. Later replacement of the
// API's client, provider, endpoint, microversion or headers fails the call.
func (t Target) Prepare(ctx context.Context) (*Prepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if t.Client == nil {
		return nil, Invalid("%s API is required", t.Spec.Kind)
	}
	selected := t.Client()
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	if source.Client.Microversion != "" {
		if _, err := microversions.ParseVersion(source.Client.Microversion); err != nil {
			return nil, Invalid("invalid selected %s microversion: %v", t.Spec.Kind, err)
		}
	}
	guard := func(ctx context.Context) error {
		var replaced error
		if t.Client() != selected {
			replaced = Invalid("selected %s client changed", t.Spec.Kind)
		}
		return errors.Join(replaced, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &Prepared{target: t, ctx: ctx, source: source, guard: guard, headers: map[string]string{}}, guard(ctx)
}

// Guard reports source changes, cancellation and operation guard faults.
func (p *Prepared) Guard(ctx context.Context) error { return p.guard(ctx) }

// Headers applies caller headers after the SDK-owned header policy.
func (p *Prepared) Headers(values map[string]string) error {
	return errors.Join(p.source.WithPolicy(p.ctx, nil, values), p.guard(p.ctx))
}

// Accept fixes the Accept request header. Resource.list sends
// application/json unless the caller supplied one; an explicit value always
// wins when force is true.
func (p *Prepared) Accept(value string, force bool) {
	if !force {
		for key := range p.source.Client.MoreHeaders {
			if http.CanonicalHeaderKey(key) == "Accept" {
				return
			}
		}
	}
	p.headers["Accept"] = value
}

func (p *Prepared) observe(responses []*rest.Response) {
	for _, r := range responses {
		if r != nil {
			p.discovery = append(p.discovery, Observation{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode})
		}
	}
}

// Discovery returns copies of the actual discovery responses so far.
func (p *Prepared) Discovery() []Observation { return cloneObservations(p.discovery) }

// Microversion returns the microversion the call sends ("" means none).
func (p *Prepared) Microversion() string { return p.source.Client.Microversion }

// Select applies the version policy. It must run after Headers.
func (p *Prepared) Select(v Version) error {
	if v.Default {
		return p.guard(p.ctx)
	}
	var version string
	var pages []*rest.Response
	var err error
	if v.Required != "" {
		version, pages, err = cinderrequest.NegotiateRequired(p.ctx, p.source, v.Ceiling, v.Required)
	} else {
		version, pages, err = cinderrequest.Negotiate(p.ctx, p.source, v.Ceiling)
	}
	p.observe(pages)
	if err != nil {
		return errors.Join(err, p.guard(p.ctx))
	}
	if v.Minimum != "" {
		if version == "" {
			return fmt.Errorf("%w: API version %s is required, but the default version will be used", resource.ErrUnsupported, v.Minimum)
		}
		actual, err := microversions.ParseVersion(version)
		if err != nil {
			return Invalid("selected %s microversion: %v", p.target.Spec.Kind, err)
		}
		expected, err := microversions.ParseVersion(v.Minimum)
		if err != nil {
			return err
		}
		if actual.Compare(expected) < 0 {
			return fmt.Errorf("%w: API version %s is required, but %s will be used", resource.ErrUnsupported, v.Minimum, version)
		}
	}
	return errors.Join(p.source.WithPolicy(p.ctx, &version, nil), p.guard(p.ctx))
}

// Supports follows utils.supports_microversion: one discovery read, the
// advertised range check and, with a configured microversion, version_match.
// Missing discovery data means false.
func (p *Prepared) Supports(required string) (bool, error) {
	found, err := microversions.Read(p.ctx, p.source, microversions.CinderProfile)
	p.observe(found.Responses)
	if err != nil {
		return false, errors.Join(err, p.guard(p.ctx))
	}
	if !found.Found {
		return false, p.guard(p.ctx)
	}
	supported, err := microversions.Supports(found.Maximum, found.Minimum, p.source.Client.Microversion, required)
	if err != nil && len(found.Responses) != 0 {
		err = found.Responses[len(found.Responses)-1].Fail(err)
	}
	return supported, errors.Join(err, p.guard(p.ctx))
}

// URL joins escaped segments below the collection path.
func (p *Prepared) URL(segments ...string) string {
	parts := []string{p.target.Path}
	for _, segment := range segments {
		parts = append(parts, url.PathEscape(segment))
	}
	return p.source.Client.ServiceURL(parts...)
}

// ServiceURL joins already escaped parts below the Cinder project endpoint.
func (p *Prepared) ServiceURL(parts ...string) string { return p.source.Client.ServiceURL(parts...) }

// Do sends one request with the selected microversion headers (or none),
// JSON Content-Type when a body is sent, and Python's status policy: every
// status below 400 is accepted. Rejected bodies are observed so that a clean
// 400, 403 or 404 can drive the find fallback.
func (p *Prepared) Do(method, target string, body any) (*rest.Response, error) {
	policy := rest.RequestHeaderPolicy{Values: maps.Clone(p.headers), Absent: []string{
		"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version",
	}}
	if version := p.source.Client.Microversion; version != "" {
		policy.Values["OpenStack-API-Version"] = "volume " + version
		policy.Values["X-OpenStack-Volume-API-Version"] = version
	} else {
		policy.Absent = append(policy.Absent, "OpenStack-API-Version", "X-OpenStack-Volume-API-Version")
	}
	if body != nil {
		value, supplied := p.source.Client.MoreHeaders["Content-Type"]
		if !supplied {
			value = "application/json"
		}
		policy.Values["Content-Type"] = value
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, method, target, body, policy, rest.RejectionPolicy{Codes: rejectedCodes(), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, Codes()...)
	return response, errors.Join(err, p.guard(p.ctx))
}

// Evidence keeps an accepted or native rejected response as a record without
// a declared view.
func (p *Prepared) Evidence(response *rest.Response, err error) *Record {
	if response == nil && err != nil {
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &native) {
			response = &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		}
	}
	if response == nil {
		if len(p.discovery) == 0 {
			return nil
		}
		return &Record{Discovery: p.Discovery()}
	}
	return &Record{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, Discovery: p.Discovery()}
}

// Record projects an accepted response over seed. A projection failure keeps
// the accepted response as proof.
func (p *Prepared) Record(response *rest.Response, seed map[string]json.RawMessage) (*Record, error) {
	value, err := p.target.Spec.FromResponse(response, seed)
	value.Discovery = p.Discovery()
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, err
}
