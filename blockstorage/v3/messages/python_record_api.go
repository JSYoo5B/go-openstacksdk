package messages

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// API shares the selected authenticated Cinder client without taking ownership.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

type recordPrepared struct {
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
}

func (a *API) capture(ctx context.Context) (*recordPrepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("message API required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = recordInvalid("selected Cinder message client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &recordPrepared{ctx: ctx, source: source, guard: guard}, guard(ctx)
}
func applyRecord[T any, O ~func(*T) error](p *recordPrepared, zero T, copy func(T) T, options []O) (T, error) {
	value := copy(zero)
	for _, apply := range options {
		if err := p.guard(p.ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, recordInvalid("nil message option")
		}
		next := copy(value)
		if err := errors.Join(apply(&next), p.guard(p.ctx)); err != nil {
			return value, err
		}
		value = copy(next)
	}
	return value, p.guard(p.ctx)
}
func recordID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return recordInvalid("nonblank UTF-8 message identity required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return recordInvalid("message identity has controls")
		}
	}
	return nil
}
func recordCodes(first int) []int {
	values := make([]int, 200)
	for i := range values {
		values[i] = first + i
	}
	return values
}
func recordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "messages", cloudread.ContextError(ctx, err))
}
func (p *recordPrepared) read(method, target string) (*rest.Response, error) {
	policy := rest.RequestHeaderPolicy{Values: make(map[string]string)}
	if version := p.source.Client.Microversion; version != "" {
		policy.Values["OpenStack-API-Version"] = "volume " + version
		policy.Values["X-OpenStack-Volume-API-Version"] = version
	} else {
		policy.Absent = []string{"OpenStack-API-Version", "X-OpenStack-Volume-API-Version"}
	}
	return rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, method, target, nil, policy, rest.RejectionPolicy{Codes: recordCodes(400), PreserveCleanRetry: true}, recordCodes(200)...)
}

// GetRecord reads one literal ID without list fallback. The default recognizes
// both the pinned Source plural key and Cinder's actual singular message key.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordGetOption) (*MessageRecord, error) {
	const op = "GetRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecord(p, RecordGetOpts{}, copyGet, slices.Clone(options))
	if err == nil {
		err = p.source.WithPolicy(ctx, policy.Microversion, policy.Headers)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	r, err := p.read(http.MethodGet, p.source.Client.ServiceURL("messages", url.PathEscape(id)))
	if err != nil {
		return recordReceipt(r), recordWrap(ctx, op, err)
	}
	value, err := recordResponse(r, id, policy.SourceEnvelope != nil && *policy.SourceEnvelope)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = r.Fail(err)
	}
	return value, recordWrap(ctx, op, err)
}

// DeleteRecord defaults to ignoring a clean404 and does not decode delete body.
func (a *API) DeleteRecord(ctx context.Context, id string, options ...RecordDeleteOption) error {
	const op = "DeleteRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return recordWrap(ctx, op, err)
	}
	if err = recordID(id); err != nil {
		return recordWrap(ctx, op, err)
	}
	policy, err := applyRecord(p, RecordDeleteOpts{}, copyDelete, slices.Clone(options))
	if err == nil {
		err = p.source.WithPolicy(ctx, policy.Microversion, policy.Headers)
	}
	if err != nil {
		return recordWrap(ctx, op, err)
	}
	_, err = p.read(http.MethodDelete, p.source.Client.ServiceURL("messages", url.PathEscape(id)))
	err = errors.Join(err, p.guard(ctx))
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return recordWrap(ctx, op, err)
}

// ListRecords lazily yields owned Source-shaped rows with raw-row cap and
// declared Body filtering. The server retains authorization/project scoping.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*MessageRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*MessageRecord, error) bool) {
		const op = "ListRecords"
		fail := func(err error) { yield(nil, recordWrap(ctx, op, err)) }
		p, err := a.capture(ctx)
		if err != nil {
			fail(err)
			return
		}
		policy, err := applyRecord(p, RecordListOpts{}, copyList, owned)
		var query url.Values
		var filters map[string]json.RawMessage
		var control rest.ListControl
		if err == nil {
			query, filters, control, err = prepareList(policy)
		}
		if err == nil {
			err = p.source.WithPolicy(ctx, policy.Microversion, policy.Headers)
		}
		if err != nil {
			fail(err)
			return
		}
		headers := p.source.Client.MoreHeaders
		if _, exists := headers["Accept"]; !exists {
			headers["Accept"] = "application/json"
		}
		offset := query.Has("offset") && !query.Has("marker")
		var page *rest.Response
		spec := rest.CollectionSpec[MessageRecord]{Client: &p.source.Client, Path: "messages", Kind: "messages", PluralKey: "messages", Metadata: recordMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: recordCodes(200), ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) {
			return p.read(http.MethodGet, target)
		}, ValidateResponse: func(r *rest.Response) error { page = r; return p.guard(ctx) }, ValidateItem: func(v *MessageRecord) error {
			v.Envelope = bytes.Clone(page.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			return errors.Join(recordProject(v, v.fields), p.guard(ctx))
		}, Paging: rest.PagePolicy[MessageRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, OffsetPagination: offset, MarkerFallback: !offset, Marker: recordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		keys := make([]string, 0, len(filters))
		for key := range filters {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for value, err := range rest.ListWithControl(ctx, spec, query, control) {
			if err != nil {
				fail(err)
				return
			}
			matched := true
			for _, key := range keys {
				match, cause := recordMatch(value.Resource.Body[key], filters[key])
				if cause = errors.Join(cause, p.guard(ctx)); cause != nil {
					fail(page.Fail(cause))
					return
				}
				if !match {
					matched = false
					break
				}
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}
