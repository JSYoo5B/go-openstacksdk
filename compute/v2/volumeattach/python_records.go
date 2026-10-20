package volumeattach

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AttachmentRecord retains the Source declared view separately from the actual
// HTTP row. ServerID is the URI parent, not Nova's unrelated serverId body key.
type AttachmentRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	ServerID       string
	row            json.RawMessage
}

func (v *AttachmentRecord) UnmarshalJSON(data []byte) error {
	v.Wire = &resource.RawResource{}
	if err := json.Unmarshal(data, v.Wire); err != nil {
		return err
	}
	v.row = bytes.Clone(data)
	return nil
}

var attachmentFields = []string{"id", "name", "device", "volume_id", "attachment_id", "bdm_id", "tag", "delete_on_termination"}

func attachmentCanonical(key string) (string, bool) {
	switch key {
	case "volumeId":
		return "volume_id", true
	case "bdm_uuid":
		return "bdm_id", true
	}
	for _, field := range attachmentFields {
		if field == key {
			return key, true
		}
	}
	return "", false
}
func attachmentInvalid(format string, values ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, values...))
}
func attachmentID(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) {
		return attachmentInvalid("identifier must be nonblank UTF-8 text")
	}
	for _, c := range value {
		if unicode.IsControl(c) {
			return attachmentInvalid("identifier has controls")
		}
	}
	return nil
}
func attachmentClone(values map[string]json.RawMessage) map[string]json.RawMessage {
	owned := make(map[string]json.RawMessage, len(values))
	for k, v := range values {
		owned[k] = bytes.Clone(v)
	}
	return owned
}
func attachmentNormalized(data json.RawMessage) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(data)
	if err != nil {
		return nil, err
	}
	fields := map[string]json.RawMessage{}
	for _, member := range members {
		if key, ok := attachmentCanonical(member.Key); ok {
			fields[key] = bytes.Clone(member.Value)
		}
	}
	return fields, nil
}
func attachmentProject(v *AttachmentRecord, fields map[string]json.RawMessage) error {
	body := map[string]json.RawMessage{}
	for _, field := range attachmentFields {
		value, err := resource.BodyRecordField(fields, field, resource.BodyFieldJSON)
		if err != nil {
			return err
		}
		body[field] = value
	}
	// Resource.id falls back to alternate_id only when id is absent.
	if _, present := fields["id"]; !present {
		body["id"] = bytes.Clone(body["volume_id"])
	}
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func attachmentMetadata(v *AttachmentRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func attachmentReceipt(r *rest.Response, serverID string) *AttachmentRecord {
	if r == nil {
		return nil
	}
	return &AttachmentRecord{ServerID: serverID, Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
}
func attachmentFromResponse(r *rest.Response, serverID string, seed map[string]json.RawMessage) (*AttachmentRecord, error) {
	v := attachmentReceipt(r, serverID)
	if v == nil {
		return nil, nil
	}
	fields := attachmentClone(seed)
	// Source ignores an absent/non-JSON success body and retains seeded attrs.
	if json.Valid(r.Body) {
		if !utf8.Valid(r.Body) {
			return v, attachmentInvalid("response must be UTF-8 JSON")
		}
		var root resource.RawResource
		if err := json.Unmarshal(r.Body, &root); err != nil {
			return v, err
		}
		raw := r.Body
		if nested, ok := root.Body["volumeAttachment"]; ok {
			raw = nested
		}
		var wire resource.RawResource
		if err := json.Unmarshal(raw, &wire); err != nil {
			return v, err
		}
		wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
		v.Wire = wire.Clone()
		observed, err := attachmentNormalized(raw)
		if err != nil {
			return v, err
		}
		for k, value := range observed {
			fields[k] = value
		}
	}
	return v, attachmentProject(v, fields)
}
func attachmentMarker(v *AttachmentRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", attachmentInvalid("attachment pagination marker missing")
	}
	fields, err := attachmentNormalized(v.row)
	if err != nil {
		return "", err
	}
	raw, present := fields["id"]
	if !present {
		raw = fields["volume_id"]
	}
	var marker string
	if err := json.Unmarshal(raw, &marker); err != nil || marker == "" {
		return "", attachmentInvalid("attachment marker must be a nonempty string")
	}
	return marker, nil
}

type preparedAttachment struct {
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
}

func attachmentCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
func attachmentWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volumeattach", cloudread.ContextError(ctx, err))
}
func (a *API) captureAttachment(ctx context.Context) (*preparedAttachment, error) {
	if a == nil {
		return nil, attachmentInvalid("attachment API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, err
	}
	var changed error
	guard := func(ctx context.Context) error {
		if changed == nil && a.client != selected {
			changed = attachmentInvalid("selected attachment client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &preparedAttachment{ctx: ctx, source: source, guard: guard}, guard(ctx)
}
func (p *preparedAttachment) policy(headers map[string]string) error {
	if p.source.Client.Microversion == "" {
		discovery, err := microversions.Read(p.ctx, p.source, microversions.NovaProfile)
		if err != nil {
			return err
		}
		version, err := microversions.SelectVersion(discovery.Maximum, discovery.Minimum, "2.89")
		if err != nil {
			return err
		}
		if version != "" {
			if err := p.source.WithPolicy(p.ctx, &version, nil); err != nil {
				return err
			}
		}
	}
	return errors.Join(p.source.WithPolicy(p.ctx, nil, headers), p.guard(p.ctx))
}
func attachmentApply[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, clone func(T) T, options []O) (T, error) {
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, attachmentInvalid("nil attachment option")
		}
		next := clone(value)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return value, err
		}
		value = clone(next)
	}
	return value, guard(ctx)
}

// AttachmentListOpts separates pagination controls from Source Body filters.
// Nil Paginated means true. MaxItems counts received rows before local filters.
type AttachmentListOpts struct {
	Headers   map[string]string
	Query     map[string]json.RawMessage
	Limit     *int
	Offset    *int
	MaxItems  int
	Paginated *bool
}
type AttachmentListOption func(*AttachmentListOpts) error

func copyAttachmentList(v AttachmentListOpts) AttachmentListOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Query = attachmentClone(v.Query)
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	if v.Offset != nil {
		x := *v.Offset
		v.Offset = &x
	}
	if v.Paginated != nil {
		x := *v.Paginated
		v.Paginated = &x
	}
	return v
}
func WithAttachmentListOpts(value AttachmentListOpts) AttachmentListOption {
	owned := copyAttachmentList(value)
	return func(v *AttachmentListOpts) error { *v = copyAttachmentList(owned); return nil }
}
func WithAttachmentLimit(value int) AttachmentListOption {
	return func(v *AttachmentListOpts) error { x := value; v.Limit = &x; return nil }
}
func WithAttachmentOffset(value int) AttachmentListOption {
	return func(v *AttachmentListOpts) error { x := value; v.Offset = &x; return nil }
}
func WithAttachmentMaxItems(value int) AttachmentListOption {
	return func(v *AttachmentListOpts) error { v.MaxItems = value; return nil }
}
func WithAttachmentPaginated(value bool) AttachmentListOption {
	return func(v *AttachmentListOpts) error { x := value; v.Paginated = &x; return nil }
}
func WithAttachmentListHeaders(value map[string]string) AttachmentListOption {
	owned := maps.Clone(value)
	return func(v *AttachmentListOpts) error { v.Headers = maps.Clone(owned); return nil }
}

// Unknown query keys and wire-only Body aliases are ignored before marshaling,
// matching Resource.list's canonical attribute filters.
func WithAttachmentQuery(key string, value any) AttachmentListOption {
	known := key == "limit" || key == "offset" || key == "marker"
	for _, field := range attachmentFields {
		if key == field {
			known = true
		}
	}
	if !known {
		return func(*AttachmentListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(v *AttachmentListOpts) error {
		if err != nil {
			return err
		}
		if v.Query == nil {
			v.Query = map[string]json.RawMessage{}
		}
		v.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func prepareAttachmentList(v AttachmentListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 || v.Offset != nil && *v.Offset < 0 {
		return nil, nil, rest.ListControl{}, attachmentInvalid("pagination values must be nonnegative")
	}
	query := url.Values{}
	filters := map[string]json.RawMessage{}
	for key, raw := range v.Query {
		if key == "limit" || key == "offset" || key == "marker" {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, rest.ListControl{}, err
			}
			if len(values) != 0 {
				query[key] = values
			}
		} else {
			for _, field := range attachmentFields {
				if key == field {
					if !json.Valid(raw) || !utf8.Valid(raw) {
						return nil, nil, rest.ListControl{}, attachmentInvalid("filter must be UTF-8 JSON")
					}
					filters[key] = bytes.Clone(raw)
				}
			}
		}
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Offset != nil {
		query.Set("offset", strconv.Itoa(*v.Offset))
	}
	// Nova offset remains a server filter while Source pagination uses marker.
	if query.Has("offset") {
		offset, err := strconv.Atoi(query.Get("offset"))
		if err != nil || len(query["offset"]) != 1 || offset < 0 {
			return nil, nil, rest.ListControl{}, attachmentInvalid("offset must be one nonnegative integer")
		}
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}
func (p *preparedAttachment) list(serverID string, query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*AttachmentRecord, error] {
	return func(yield func(*AttachmentRecord, error) bool) {
		var origin *rest.Response
		spec := rest.CollectionSpec[AttachmentRecord]{Client: &p.source.Client, Path: "servers/" + url.PathEscape(serverID) + "/os-volume_attachments", Kind: "volumeattach", PluralKey: "volumeAttachments", Metadata: attachmentMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: attachmentCodes(), ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) }, ValidateItem: func(v *AttachmentRecord) error {
			v.ServerID = serverID
			v.Envelope = bytes.Clone(origin.Body)
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			fields, err := attachmentNormalized(v.row)
			if err != nil {
				return err
			}
			return errors.Join(attachmentProject(v, fields), p.guard(p.ctx))
		}, Paging: rest.PagePolicy[AttachmentRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: attachmentMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		for value, err := range rest.ListWithControl(p.ctx, spec, query, control) {
			if err != nil {
				yield(nil, err)
				return
			}
			matched, err := jsonfilter.MatchFilters(value.Resource.Body, filters)
			if err = errors.Join(err, p.guard(p.ctx)); err != nil {
				yield(nil, origin.Fail(err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

// ListRecords owns Source limit/offset, Body filters and lazy paging for a fixed
// server. It does not call Nova before the returned iterator is consumed.
func (a *API) ListRecords(ctx context.Context, serverID string, options ...AttachmentListOption) iter.Seq2[*AttachmentRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*AttachmentRecord, error) bool) {
		const op = "ListRecords"
		p, err := a.captureAttachment(ctx)
		if err == nil {
			err = attachmentID(serverID)
		}
		if err != nil {
			yield(nil, attachmentWrap(ctx, op, err))
			return
		}
		opts, err := attachmentApply(ctx, p.guard, AttachmentListOpts{}, copyAttachmentList, owned)
		var query url.Values
		var filters map[string]json.RawMessage
		var control rest.ListControl
		if err == nil {
			query, filters, control, err = prepareAttachmentList(opts)
		}
		if err == nil {
			err = p.policy(opts.Headers)
		}
		if err != nil {
			yield(nil, attachmentWrap(ctx, op, err))
			return
		}
		if _, ok := p.source.Client.MoreHeaders["Accept"]; !ok {
			p.source.Client.MoreHeaders["Accept"] = "application/json"
		}
		for value, err := range p.list(serverID, query, filters, control) {
			if !yield(value, attachmentWrap(ctx, op, err)) {
				return
			}
		}
	}
}
func (s *VolumeAttachmentScope) ListRecords(ctx context.Context, options ...AttachmentListOption) iter.Seq2[*AttachmentRecord, error] {
	return s.api.ListRecords(ctx, s.parentID, options...)
}

// AttachmentUpdateOpts owns replacement and declared Body attrs; omitted swap
// sends the existing volume ID. Extra unknown attrs are ignored by Source.
type AttachmentUpdateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}
type AttachmentUpdateOption func(*AttachmentUpdateOpts) error

func copyAttachmentUpdate(v AttachmentUpdateOpts) AttachmentUpdateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = attachmentClone(v.Fields)
	return v
}
func WithAttachmentUpdateOpts(value AttachmentUpdateOpts) AttachmentUpdateOption {
	owned := copyAttachmentUpdate(value)
	return func(v *AttachmentUpdateOpts) error { *v = copyAttachmentUpdate(owned); return nil }
}
func WithAttachmentUpdateHeaders(value map[string]string) AttachmentUpdateOption {
	owned := maps.Clone(value)
	return func(v *AttachmentUpdateOpts) error { v.Headers = maps.Clone(owned); return nil }
}
func WithAttachmentSwapTo(volumeID string) AttachmentUpdateOption {
	return WithAttachmentAttribute("volume_id", volumeID)
}
func WithAttachmentAttribute(key string, value any) AttachmentUpdateOption {
	canonical, known := attachmentCanonical(key)
	if key == "server_id" || key == "id" {
		return func(*AttachmentUpdateOpts) error { return attachmentInvalid("update target %q is fixed", key) }
	}
	if !known {
		return func(*AttachmentUpdateOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(v *AttachmentUpdateOpts) error {
		if err != nil {
			return err
		}
		if v.Fields == nil {
			v.Fields = map[string]json.RawMessage{}
		}
		// The named volume_id=None argument means retain the original volume. A
		// wire alias volumeId=None is a Body attr and must remain explicit null.
		if key == "volume_id" && bytes.Equal(bytes.TrimSpace(owned), []byte("null")) {
			delete(v.Fields, canonical)
		} else {
			v.Fields[canonical] = bytes.Clone(owned)
		}
		return nil
	}
}

// UpdateRecord uses the current volume ID in the route, never attachment_id.
// Source defaults volumeId to that ID even when only other attrs are changed.
func (a *API) UpdateRecord(ctx context.Context, serverID, volumeID string, options ...AttachmentUpdateOption) (*AttachmentRecord, error) {
	const op = "UpdateRecord"
	p, err := a.captureAttachment(ctx)
	if err == nil {
		err = errors.Join(attachmentID(serverID), attachmentID(volumeID))
	}
	if err != nil {
		return nil, attachmentWrap(ctx, op, err)
	}
	opts, err := attachmentApply(ctx, p.guard, AttachmentUpdateOpts{}, copyAttachmentUpdate, slices.Clone(options))
	fields := map[string]json.RawMessage{}
	fields["id"], _ = json.Marshal(volumeID)
	fields["volume_id"], _ = json.Marshal(volumeID)
	if err == nil {
		for key, raw := range opts.Fields {
			canonical, known := attachmentCanonical(key)
			if !known {
				continue
			}
			if canonical == "id" {
				err = attachmentInvalid("update target id is fixed")
				break
			}
			if !json.Valid(raw) || !utf8.Valid(raw) {
				err = attachmentInvalid("attribute %q must be UTF-8 JSON", key)
				break
			}
			fields[canonical] = bytes.Clone(raw)
		}
	}
	if err == nil {
		err = p.policy(opts.Headers)
	}
	if err != nil {
		return nil, attachmentWrap(ctx, op, err)
	}
	body := map[string]json.RawMessage{}
	for key, raw := range fields {
		if key == "id" {
			continue
		}
		wire := key
		if key == "volume_id" {
			wire = "volumeId"
		}
		if key == "bdm_id" {
			wire = "bdm_uuid"
		}
		body[wire] = bytes.Clone(raw)
	}
	response, err := rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, http.MethodPut, p.source.Client.ServiceURL("servers", url.PathEscape(serverID), "os-volume_attachments", url.PathEscape(volumeID)), map[string]any{"volumeAttachment": body}, nil, attachmentCodes()...)
	if err != nil {
		return attachmentReceipt(response, serverID), attachmentWrap(ctx, op, err)
	}
	value, err := attachmentFromResponse(response, serverID, fields)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, attachmentWrap(ctx, op, err)
}
func (s *VolumeAttachmentScope) UpdateRecord(ctx context.Context, volumeID string, options ...AttachmentUpdateOption) (*AttachmentRecord, error) {
	return s.api.UpdateRecord(ctx, s.parentID, volumeID, options...)
}
