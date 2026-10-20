package attachments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type pythonAttachmentPrepared struct {
	ctx       context.Context
	source    *cloudread.Source
	guard     func(context.Context) error
	discovery []DiscoveryObservation
}

func pythonAttachmentCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
func pythonAttachmentRejections() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 400 + i
	}
	return codes
}
func pythonAttachmentWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "attachments", cloudread.ContextError(ctx, err))
}
func (a *API) pythonAttachmentCapture(ctx context.Context) (*pythonAttachmentPrepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, pythonAttachmentInvalid("attachment API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = pythonAttachmentInvalid("selected attachment client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &pythonAttachmentPrepared{ctx: ctx, source: source, guard: guard}, guard(ctx)
}
func pythonAttachmentVolumeID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return pythonAttachmentInvalid("volume ID must be a nonblank UTF-8 string")
	}
	for _, character := range id {
		if unicode.IsControl(character) {
			return pythonAttachmentInvalid("volume ID has controls")
		}
	}
	return nil
}
func pythonAttachmentCreateFields(policy RecordCreateOpts, volume string) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage)
	for key, raw := range policy.Fields {
		wire, known := pythonAttachmentField(key)
		if !known {
			continue
		}
		if wire == "volume_id" {
			return nil, pythonAttachmentInvalid("volume_id is supplied by the volume argument")
		}
		if _, duplicate := fields[wire]; duplicate {
			return nil, pythonAttachmentInvalid("conflicting attribute aliases for %q", wire)
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, pythonAttachmentInvalid("attribute %q must be UTF-8 JSON", key)
		}
		fields[wire] = bytes.Clone(raw)
	}
	raw, _ := json.Marshal(volume)
	fields["volume_id"] = raw
	return fields, nil
}

// modeSupport keeps the advertised range test separate from the selected
// default, as Source supports_microversion does. A finite default is no proof
// that the server advertises the required version within both range bounds.
func (p *pythonAttachmentPrepared) modeSupport(explicit *bool) (bool, error) {
	if explicit != nil {
		return *explicit, p.guard(p.ctx)
	}
	selected := p.source.Client.Microversion
	discovered, err := microversions.Read(p.ctx, p.source, microversions.CinderProfile)
	for _, response := range discovered.Responses {
		p.discovery = append(p.discovery, DiscoveryObservation{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode})
	}
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		return false, err
	}
	supported, err := microversions.Supports(discovered.Maximum, discovered.Minimum, selected, "3.54")
	if err != nil {
		return false, p.discoveryFailure(err)
	}
	if selected == "" {
		version, selectErr := microversions.SelectVersion(discovered.Maximum, discovered.Minimum, "3.71")
		if selectErr != nil {
			return false, p.discoveryFailure(selectErr)
		}
		if err := p.source.WithPolicy(p.ctx, &version, nil); err != nil {
			return false, p.discoveryFailure(err)
		}
	}
	if err := p.guard(p.ctx); err != nil {
		return false, p.discoveryFailure(err)
	}
	return supported, nil
}
func (p *pythonAttachmentPrepared) discoveryFailure(err error) error {
	if len(p.discovery) == 0 {
		return err
	}
	observed := p.discovery[len(p.discovery)-1]
	return (&rest.Response{Body: bytes.Clone(observed.Envelope), Header: observed.Header.Clone(), StatusCode: observed.StatusCode}).Fail(err)
}
func (p *pythonAttachmentPrepared) proof(value *AttachmentRecord) *AttachmentRecord {
	if value == nil {
		return nil
	}
	value.Discovery = make([]DiscoveryObservation, len(p.discovery))
	for i, observed := range p.discovery {
		value.Discovery[i] = DiscoveryObservation{Envelope: bytes.Clone(observed.Envelope), Header: observed.Header.Clone(), StatusCode: observed.StatusCode}
	}
	return value
}
func (p *pythonAttachmentPrepared) discoveryResult(err error) (*AttachmentRecord, error) {
	if len(p.discovery) == 0 {
		return nil, err
	}
	return p.proof(&AttachmentRecord{}), err
}
func (p *pythonAttachmentPrepared) create(payload map[string]json.RawMessage) (*rest.Response, error) {
	headers := rest.RequestHeaderPolicy{Values: make(map[string]string)}
	if version := p.source.Client.Microversion; version != "" {
		headers.Values["OpenStack-API-Version"] = "volume " + version
		headers.Values["X-OpenStack-Volume-API-Version"] = version
	} else {
		headers.Absent = []string{"OpenStack-API-Version", "X-OpenStack-Volume-API-Version"}
	}
	return rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, http.MethodPost, p.source.Client.ServiceURL("attachments"), map[string]any{"attachment": payload}, headers, rest.RejectionPolicy{Codes: pythonAttachmentRejections(), PreserveCleanRetry: true}, pythonAttachmentCodes()...)
}

// CreateRecord reserves a volume without an instance by default. It accepts
// declared Source attributes with omission and explicit JSON presence intact.
func (a *API) CreateRecord(ctx context.Context, volumeID string, options ...RecordCreateOption) (*AttachmentRecord, error) {
	const op = "CreateRecord"
	p, err := a.pythonAttachmentCapture(ctx)
	if err != nil {
		return nil, pythonAttachmentWrap(ctx, op, err)
	}
	policy, err := pythonAttachmentApply(ctx, p.guard, RecordCreateOpts{}, pythonAttachmentCopyCreate, slices.Clone(options))
	if err == nil {
		err = pythonAttachmentVolumeID(volumeID)
	}
	if err == nil {
		err = errors.Join(p.source.WithPolicy(ctx, nil, policy.Headers), p.guard(ctx))
	}
	if err != nil {
		return nil, pythonAttachmentWrap(ctx, op, err)
	}
	fields, err := pythonAttachmentCreateFields(policy, volumeID)
	if err != nil {
		return nil, pythonAttachmentWrap(ctx, op, err)
	}
	payload := pythonAttachmentClone(fields)
	for _, pair := range [][2]string{{"volume_id", "volume_uuid"}, {"instance", "instance_uuid"}} {
		raw, exists := payload[pair[0]]
		if !exists {
			continue
		}
		truthy, truthErr := cloudfilter.PythonTruthy(raw)
		if truthErr != nil {
			return nil, pythonAttachmentWrap(ctx, op, truthErr)
		}
		if truthy {
			payload[pair[1]] = raw
			delete(payload, pair[0])
		}
	}
	if raw, exists := payload["mode"]; exists {
		truthy, truthErr := cloudfilter.PythonTruthy(raw)
		if truthErr != nil {
			return nil, pythonAttachmentWrap(ctx, op, truthErr)
		}
		if !truthy {
			supported, supportErr := p.modeSupport(policy.ModeSupported)
			if supportErr != nil {
				value, failure := p.discoveryResult(supportErr)
				return value, pythonAttachmentWrap(ctx, op, failure)
			}
			if supported {
				delete(payload, "mode")
			}
		}
	}
	response, err := p.create(payload)
	if err != nil {
		if response == nil {
			value, failure := p.discoveryResult(err)
			return value, pythonAttachmentWrap(ctx, op, failure)
		}
		v := &AttachmentRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
		return p.proof(v), pythonAttachmentWrap(ctx, op, err)
	}
	value, err := pythonAttachmentFromResponse(response, fields)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return p.proof(value), pythonAttachmentWrap(ctx, op, err)
}

// ListRecords uses the Source summary /attachments route and follows
// attachments_links, HTTP links and bounded marker fallback lazily.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*AttachmentRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*AttachmentRecord, error) bool) {
		const op = "ListRecords"
		p, err := a.pythonAttachmentCapture(ctx)
		if err != nil {
			yield(nil, pythonAttachmentWrap(ctx, op, err))
			return
		}
		policy, err := pythonAttachmentApply(ctx, p.guard, RecordListOpts{}, pythonAttachmentCopyList, owned)
		if err == nil {
			err = errors.Join(p.source.WithPolicy(ctx, policy.Microversion, policy.Headers), p.guard(ctx))
		}
		if err != nil {
			yield(nil, pythonAttachmentWrap(ctx, op, err))
			return
		}
		if _, ok := p.source.Client.MoreHeaders["Accept"]; !ok {
			p.source.Client.MoreHeaders["Accept"] = "application/json"
		}
		query, filters, control, err := pythonAttachmentPrepareList(policy)
		if err != nil {
			yield(nil, pythonAttachmentWrap(ctx, op, err))
			return
		}
		var origin *rest.Response
		spec := rest.CollectionSpec[AttachmentRecord]{ReadPage: func(_ context.Context, target string, codes ...int) (*rest.Response, error) {
			return rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodGet, target, nil, nil, rest.RejectionPolicy{Codes: pythonAttachmentRejections(), PreserveCleanRetry: true}, codes...)
		}, Client: &p.source.Client, Path: "attachments", Kind: "attachments", PluralKey: "attachments", Metadata: pythonAttachmentMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: pythonAttachmentCodes(), ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(ctx) }, ValidateItem: func(v *AttachmentRecord) error {
			v.Envelope = bytes.Clone(origin.Body)
			v.Header = v.Wire.Header.Clone()
			v.StatusCode = v.Wire.StatusCode
			pythonAttachmentProject(v, v.fields)
			return p.guard(ctx)
		}, Paging: rest.PagePolicy[AttachmentRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: pythonAttachmentMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		for value, listErr := range rest.ListWithControl(ctx, spec, query, control) {
			if listErr != nil {
				yield(nil, pythonAttachmentWrap(ctx, op, listErr))
				return
			}
			matched, matchErr := jsonfilter.MatchFilters(value.Resource.Body, filters)
			if matchErr = errors.Join(matchErr, p.guard(ctx)); matchErr != nil {
				yield(nil, pythonAttachmentWrap(ctx, op, origin.Fail(matchErr)))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}
