package shareattachments

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
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

type prepared struct {
	ctx          context.Context
	source       *cloudread.Source
	guard        func(context.Context) error
	server, path string
	discovery    []DiscoveryObservation
}

func validIdentity(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) {
		return invalid("identity must be nonblank UTF-8 literal text")
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return invalid("identity must not contain controls")
		}
	}
	return nil
}
func (a *API) capture(ctx context.Context, server string) (*prepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, invalid("share attachment API is required")
	}
	if err := validIdentity(server); err != nil {
		return nil, err
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, err
	}
	var changed error
	guard := func(check context.Context) error {
		if changed == nil && a.client != selected {
			changed = invalid("selected share attachment client changed")
		}
		return errors.Join(changed, source.Guard(check), rest.CheckOperationGuard(ctx))
	}
	p := &prepared{source: source, guard: guard, server: server, path: "servers/" + url.PathEscape(server) + "/shares"}
	p.ctx = rest.WithOperationGuard(ctx, guard)
	return p, guard(ctx)
}
func ownRead(c *request.Config[ReadOpts]) {
	c.Options.Microversion = copyPointer(c.Options.Microversion)
	cloudread.OwnReadConfig(c)
}
func ownCreate(c *request.Config[CreateOpts]) {
	c.Options = copyCreate(c.Options)
	cloudread.OwnReadConfig(c)
}
func ownDelete(c *request.Config[DeleteOpts]) {
	c.Options.Microversion = copyPointer(c.Options.Microversion)
	c.Options.IgnoreMissing = copyPointer(c.Options.IgnoreMissing)
	cloudread.OwnReadConfig(c)
}
func ownList(c *request.Config[ListOpts]) {
	c.Options = copyList(c.Options)
	cloudread.OwnReadConfig(c)
}
func codes(start int) []int {
	out := make([]int, 200)
	for i := range out {
		out[i] = start + i
	}
	return out
}
func wrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "shareattachments", cloudread.ContextError(ctx, err))
}
func (p *prepared) selectPolicy(override *string, headers map[string]string) error {
	// Invalid caller policy is rejected before version discovery can do HTTP.
	trial := *p.source
	trial.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
	preliminary := maps.Clone(headers)
	if override == nil && p.source.Client.Microversion == "" {
		for key := range preliminary {
			if isVersionHeader(key) {
				delete(preliminary, key)
			}
		}
	}
	if override != nil {
		for key := range trial.Client.MoreHeaders {
			if isVersionHeader(key) {
				delete(trial.Client.MoreHeaders, key)
			}
		}
	}
	if err := errors.Join(trial.WithPolicy(p.ctx, override, preliminary), p.guard(p.ctx)); err != nil {
		return err
	}
	selected := p.source.Client.Microversion
	if override != nil {
		selected = *override
	} else if selected == "" {
		found, err := microversions.Read(p.ctx, p.source, microversions.NovaProfile)
		for _, v := range found.Responses {
			p.discovery = append(p.discovery, DiscoveryObservation{Envelope: append(json.RawMessage(nil), v.Body...), Header: v.Header.Clone(), StatusCode: v.StatusCode})
		}
		if err = errors.Join(err, p.guard(p.ctx)); err != nil {
			return err
		}
		selected, err = microversions.SelectVersion(found.Maximum, found.Minimum, "2.97")
		if err != nil {
			return p.discoveryFailure(err)
		}
	}
	for key := range p.source.Client.MoreHeaders {
		if isVersionHeader(key) {
			delete(p.source.Client.MoreHeaders, key)
		}
	}
	return errors.Join(p.source.WithPolicy(p.ctx, &selected, headers), p.guard(p.ctx))
}
func isVersionHeader(key string) bool {
	return strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version")
}
func (p *prepared) headers(method string) rest.RequestHeaderPolicy {
	policy := rest.RequestHeaderPolicy{Values: maps.Clone(p.source.Client.MoreHeaders)}
	if policy.Values == nil {
		policy.Values = make(map[string]string)
	}
	for key := range policy.Values {
		if isVersionHeader(key) {
			delete(policy.Values, key)
		}
	}
	if version := p.source.Client.Microversion; version != "" {
		policy.Values["OpenStack-API-Version"] = "compute " + version
		policy.Values["X-OpenStack-Nova-API-Version"] = version
	} else {
		policy.Absent = []string{"OpenStack-API-Version", "X-OpenStack-Nova-API-Version"}
	}
	if method == http.MethodPost {
		if _, exists := policy.Values["Content-Type"]; !exists {
			policy.Values["Content-Type"] = "application/json"
		}
	}
	p.source.Client.MoreHeaders = maps.Clone(policy.Values)
	return policy
}
func (p *prepared) do(method, target string, body any) (*rest.Response, error) {
	return rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, method, target, body, p.headers(method), rest.RejectionPolicy{Codes: codes(400), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, codes(200)...)
}

// CreateRecord posts the explicit share ID and only supplied declared Source
// attributes. Resource defaults belong to the returned view, not the request.
func (a *API) CreateRecord(ctx context.Context, server, share string, options ...CreateOption) (*Record, error) {
	p, err := a.capture(ctx, server)
	if err != nil {
		return nil, wrap(ctx, "CreateRecord", err)
	}
	fail := func(err error) (*Record, error) { return p.proof(nativeResponse(err)), wrap(ctx, "CreateRecord", err) }
	if err := validIdentity(share); err != nil {
		return nil, wrap(ctx, "CreateRecord", err)
	}
	cfg, err := cloudread.ApplyReadOptions(p.ctx, CreateOpts{}, options, ownCreate, p.guard)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, true)
	}
	if err != nil {
		return nil, wrap(ctx, "CreateRecord", err)
	}
	fields := make(map[string]json.RawMessage)
	for key, raw := range cfg.Options.Attributes {
		if control(key) {
			return nil, wrap(ctx, "CreateRecord", unsupportedControl(key))
		}
		if !knownField(key) {
			continue
		}
		if key == "share_id" {
			return nil, wrap(ctx, "CreateRecord", invalid("share_id is supplied by the share argument"))
		}
		fields[key], err = validateRaw(key, raw)
		if err != nil {
			return nil, wrap(ctx, "CreateRecord", err)
		}
	}
	fields["share_id"], _ = json.Marshal(share)
	if err = p.selectPolicy(cfg.Options.Microversion, cfg.Headers); err != nil {
		return p.proof(nil), wrap(ctx, "CreateRecord", err)
	}
	response, err := p.do(http.MethodPost, p.source.Client.ServiceURL(p.path), map[string]any{"share": fields})
	if err != nil {
		if response != nil {
			return p.proof(response), wrap(ctx, "CreateRecord", err)
		}
		return fail(err)
	}
	result, err := p.translate(response, fields)
	return result, wrap(ctx, "CreateRecord", err)
}

// GetRecord fixes both URI IDs. A missing or null response share_id cannot
// redirect the next operation or erase the member ID seeded by the Proxy.
func (a *API) GetRecord(ctx context.Context, server, share string, options ...ReadOption) (*Record, error) {
	p, err := a.capture(ctx, server)
	if err != nil {
		return nil, wrap(ctx, "GetRecord", err)
	}
	if err := validIdentity(share); err != nil {
		return nil, wrap(ctx, "GetRecord", err)
	}
	cfg, err := cloudread.ApplyReadOptions(p.ctx, ReadOpts{}, options, ownRead, p.guard)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, true)
	}
	if err != nil {
		return nil, wrap(ctx, "GetRecord", err)
	}
	if err = p.selectPolicy(cfg.Options.Microversion, cfg.Headers); err != nil {
		return p.proof(nil), wrap(ctx, "GetRecord", err)
	}
	response, err := p.do(http.MethodGet, p.source.Client.ServiceURL(p.path, url.PathEscape(share)), nil)
	if err != nil {
		if response == nil {
			response = nativeResponse(err)
		}
		return p.proof(response), wrap(ctx, "GetRecord", err)
	}
	result, err := p.translate(response, seedID(share))
	return result, wrap(ctx, "GetRecord", err)
}

// DeleteRecord is bodyless and ignores only a clean terminal HTTP 404 by
// default. Physical read/Close, callback, source and context faults survive.
func (a *API) DeleteRecord(ctx context.Context, server, share string, options ...DeleteOption) (*DeleteResult, error) {
	p, err := a.capture(ctx, server)
	if err != nil {
		return nil, wrap(ctx, "DeleteRecord", err)
	}
	if err := validIdentity(share); err != nil {
		return nil, wrap(ctx, "DeleteRecord", err)
	}
	cfg, err := cloudread.ApplyReadOptions(p.ctx, DeleteOpts{}, options, ownDelete, p.guard)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, true)
	}
	if err != nil {
		return nil, wrap(ctx, "DeleteRecord", err)
	}
	if err = p.selectPolicy(cfg.Options.Microversion, cfg.Headers); err != nil {
		return &DeleteResult{Discovery: p.discoveryCopy()}, wrap(ctx, "DeleteRecord", err)
	}
	response, err := p.do(http.MethodDelete, p.source.Client.ServiceURL(p.path, url.PathEscape(share)), nil)
	if response == nil {
		response = nativeResponse(err)
	}
	ignored := false
	if err != nil && (cfg.Options.IgnoreMissing == nil || *cfg.Options.IgnoreMissing) {
		clean := resource.IgnoreMissing(err)
		ignored = clean == nil
		err = clean
	}
	result := &DeleteResult{IgnoredMissing: ignored, Discovery: p.discoveryCopy()}
	if response != nil {
		result.Envelope = append(json.RawMessage(nil), response.Body...)
		result.Header = response.Header.Clone()
		result.StatusCode = response.StatusCode
	}
	return result, wrap(ctx, "DeleteRecord", err)
}

func hasHeader(headers map[string]string, key string) bool {
	for name := range headers {
		if strings.EqualFold(name, key) {
			return true
		}
	}
	return false
}
