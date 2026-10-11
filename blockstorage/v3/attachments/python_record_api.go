package attachments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type pythonAttachmentPrepared struct {
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
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
func pythonAttachmentModeSupport(version string, explicit *bool) (bool, error) {
	if explicit != nil {
		return *explicit, nil
	}
	parts := strings.Split(version, ".")
	if len(parts) == 2 {
		major, e1 := strconv.Atoi(parts[0])
		minor, e2 := strconv.Atoi(parts[1])
		if e1 == nil && e2 == nil && major == 3 && minor >= 0 {
			return minor >= 54, nil
		}
	}
	return false, pythonAttachmentInvalid("a falsey attachment mode needs a numeric Cinder microversion or explicit ModeSupported capability")
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
			supported, supportErr := pythonAttachmentModeSupport(p.source.Client.Microversion, policy.ModeSupported)
			if supportErr != nil {
				return nil, pythonAttachmentWrap(ctx, op, supportErr)
			}
			if supported {
				delete(payload, "mode")
			}
		}
	}
	response, err := rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodPost, p.source.Client.ServiceURL("attachments"), map[string]any{"attachment": payload}, nil, rest.RejectionPolicy{Codes: pythonAttachmentRejections(), PreserveCleanRetry: true}, pythonAttachmentCodes()...)
	if err != nil {
		if response == nil {
			return nil, pythonAttachmentWrap(ctx, op, err)
		}
		v := &AttachmentRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
		return v, pythonAttachmentWrap(ctx, op, err)
	}
	value, err := pythonAttachmentFromResponse(response, fields)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, pythonAttachmentWrap(ctx, op, err)
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
