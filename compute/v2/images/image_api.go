package images

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// API is explicitly bound to Nova's legacy image proxy. Construct it with a
// compute client selected at 2.1..2.35 (metadata alone supports through 2.38).
// An empty selected microversion uses Nova's default 2.1. It never downgrades.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

type preparedImage struct {
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
}

func imageCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
func imageWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "nova-images", cloudread.ContextError(ctx, err))
}
func imageID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return imageInvalid("identifier must be nonblank UTF-8 literal text")
	}
	for _, c := range id {
		if unicode.IsControl(c) {
			return imageInvalid("identifier has controls")
		}
	}
	return nil
}
func requireLegacyVersion(version, ceiling string) error {
	if version == "" {
		version = "2.1"
	}
	if version == "latest" {
		return fmt.Errorf("%w: legacy Nova images require 2.1..%s; latest is not supported", resource.ErrUnsupported, ceiling)
	}
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return imageInvalid("Nova microversion must have two decimal components")
	}
	for _, part := range parts {
		if part == "" || len(part) > 1 && part[0] == '0' {
			return imageInvalid("invalid Nova microversion")
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return imageInvalid("invalid Nova microversion")
			}
		}
	}
	candidate, err := microversions.ParseVersion(version)
	if err != nil {
		return err
	}
	low, _ := microversions.ParseVersion("2.1")
	high, _ := microversions.ParseVersion(ceiling)
	if candidate.Compare(low) < 0 || candidate.Compare(high) > 0 {
		return fmt.Errorf("%w: legacy Nova images require 2.1..%s (selected %s)", resource.ErrUnsupported, ceiling, version)
	}
	return nil
}
func (a *API) capture(ctx context.Context, ceiling string) (*preparedImage, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, imageInvalid("image API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, err
	}
	if err := requireLegacyVersion(source.Client.Microversion, ceiling); err != nil {
		return nil, err
	}
	var changed error
	guard := func(ctx context.Context) error {
		if changed == nil && a.client != selected {
			changed = imageInvalid("selected image client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	p := &preparedImage{ctx: ctx, source: source, guard: guard}
	return p, guard(ctx)
}
func (p *preparedImage) headers(headers map[string]string) error {
	return errors.Join(p.source.WithPolicy(p.ctx, nil, headers), p.guard(p.ctx))
}
func (p *preparedImage) get(id string) (*ImageRecord, error) {
	if err := imageID(id); err != nil {
		return nil, err
	}
	r, err := rest.DoJSONGuardedRejections(p.ctx, &p.source.Client, p.guard, http.MethodGet, p.source.Client.ServiceURL("images", url.PathEscape(id)), nil, nil, rest.RejectionPolicy{Codes: []int{400, 403, 404}, PreserveCleanRetry: true}, imageCodes()...)
	if err != nil {
		return imageReceipt(r), err
	}
	v, err := imageFromResponse(r, id)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = r.Fail(err)
	}
	return v, err
}
func (a *API) GetRecord(ctx context.Context, id string, options ...ReadOption) (*ImageRecord, error) {
	const op = "GetRecord"
	p, err := a.capture(ctx, "2.35")
	if err != nil {
		return nil, imageWrap(ctx, op, err)
	}
	opts, err := applyOptions(ctx, p.guard, ReadOpts{}, copyRead, slices.Clone(options))
	if err == nil {
		err = p.headers(opts.Headers)
	}
	if err != nil {
		return nil, imageWrap(ctx, op, err)
	}
	v, err := p.get(id)
	return v, imageWrap(ctx, op, err)
}
func (a *API) DeleteRecord(ctx context.Context, id string, options ...DeleteOption) error {
	const op = "DeleteRecord"
	p, err := a.capture(ctx, "2.35")
	if err != nil {
		return imageWrap(ctx, op, err)
	}
	opts, err := applyOptions(ctx, p.guard, DeleteOpts{}, copyDelete, slices.Clone(options))
	if err == nil {
		err = imageID(id)
	}
	if err == nil {
		err = p.headers(opts.Headers)
	}
	if err != nil {
		return imageWrap(ctx, op, err)
	}
	_, err = rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard, http.MethodDelete, p.source.Client.ServiceURL("images", url.PathEscape(id)), nil, nil, rest.RejectionPolicy{Codes: []int{404}, PreserveCleanRetry: true}, imageCodes()...)
	if opts.IgnoreMissing == nil || *opts.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return imageWrap(ctx, op, err)
}
func (p *preparedImage) list(path string, query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*ImageRecord, error] {
	return func(yield func(*ImageRecord, error) bool) {
		var origin *rest.Response
		spec := rest.CollectionSpec[ImageRecord]{Client: &p.source.Client, Path: path, Kind: "nova-images", PluralKey: "images", Metadata: imageMetadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: imageCodes(), ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) }, ValidateItem: func(v *ImageRecord) error {
			fields, err := normalizedImageFields(v.row)
			if err != nil {
				return err
			}
			v.Header, v.StatusCode = v.Wire.Header.Clone(), v.Wire.StatusCode
			v.Envelope = bytes.Clone(origin.Body)
			return errors.Join(projectImage(v, fields), p.guard(p.ctx))
		}, Paging: rest.PagePolicy[ImageRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: imageMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		for v, err := range rest.ListWithControl(p.ctx, spec, query, control) {
			if err != nil {
				yield(nil, err)
				return
			}
			matched, err := jsonfilter.MatchFilters(v.Resource.Body, filters)
			if err = errors.Join(err, p.guard(p.ctx)); err != nil {
				yield(nil, origin.Fail(err))
				return
			}
			if matched && !yield(v, nil) {
				return
			}
		}
	}
}

// ListRecords defaults to /images/detail. Summary mode is /images.
func (a *API) ListRecords(ctx context.Context, options ...ListOption) iter.Seq2[*ImageRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ImageRecord, error) bool) {
		const op = "ListRecords"
		p, err := a.capture(ctx, "2.35")
		if err != nil {
			yield(nil, imageWrap(ctx, op, err))
			return
		}
		opts, err := applyOptions(ctx, p.guard, ListOpts{}, copyList, owned)
		if err == nil {
			err = p.headers(opts.Headers)
		}
		if err != nil {
			yield(nil, imageWrap(ctx, op, err))
			return
		}
		if _, ok := p.source.Client.MoreHeaders["Accept"]; !ok {
			p.source.Client.MoreHeaders["Accept"] = "application/json"
		}
		query, filters, control, err := prepareList(opts)
		if err != nil {
			yield(nil, imageWrap(ctx, op, err))
			return
		}
		path := "images/detail"
		if opts.Details != nil && !*opts.Details {
			path = "images"
		}
		for v, err := range p.list(path, query, filters, control) {
			if !yield(v, imageWrap(ctx, op, err)) {
				return
			}
		}
	}
}

// FindRecord uses literal GET first, then /images?name=..., not detail mode.
func (a *API) FindRecord(ctx context.Context, nameOrID string, options ...FindOption) (*ImageRecord, error) {
	const op = "FindRecord"
	p, err := a.capture(ctx, "2.35")
	if err != nil {
		return nil, imageWrap(ctx, op, err)
	}
	opts, err := applyOptions(ctx, p.guard, FindOpts{}, copyFind, slices.Clone(options))
	if err != nil {
		return nil, imageWrap(ctx, op, err)
	}
	collection := resource.NewCollection(resource.Adapter[ImageRecord]{Kind: "nova-images", IdentityFind: true, ValidateID: imageID, IdentityDirectGet: imageID, IdentityResponseID: func(v *ImageRecord) (string, error) {
		if v == nil {
			return "", imageInvalid("find response missing")
		}
		return imageText(v, "id"), nil
	}, Name: func(v *ImageRecord) string { return imageText(v, "name") }, NameQuery: func(name string) string { return name }, NameQueryKey: "name", Get: func(_ context.Context, id string) (*ImageRecord, error) { return p.get(id) }, Iterate: func(_ context.Context, query url.Values) iter.Seq2[*ImageRecord, error] {
		return p.list("images", query, nil, rest.ListControl{})
	}})
	v, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(opts.IgnoreMissing == nil || *opts.IgnoreMissing))
	return v, imageWrap(ctx, op, errors.Join(err, p.guard(ctx)))
}
