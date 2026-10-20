package floatingips

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AvailableIPOpts contains optional Go transport overrides, not server filters.
// The pinned Source method has no filter kwargs and always lists unfiltered.
type AvailableIPOpts struct {
	Headers      map[string]string
	Microversion *string
}
type AvailableIPOption func(*AvailableIPOpts) error

func copyAvailableIP(v AvailableIPOpts) AvailableIPOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithAvailableIPOpts(v AvailableIPOpts) AvailableIPOption {
	owned := copyAvailableIP(v)
	return func(c *AvailableIPOpts) error { *c = copyAvailableIP(owned); return nil }
}
func WithAvailableIPHeaders(v map[string]string) AvailableIPOption {
	owned := maps.Clone(v)
	return func(c *AvailableIPOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithAvailableIPMicroversion(v string) AvailableIPOption {
	return func(c *AvailableIPOpts) error { x := v; c.Microversion = &x; return nil }
}

// AvailableIPRecord owns the Source resource view and original page evidence.
// It represents an observed available IP, never a reservation or allocation.
type AvailableIPRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func availableIPProfile() neutronfind.Profile {
	p := neutronfind.Profile{Kind: "floatingips", Path: "floatingips", Singular: "floatingip", Plural: "floatingips", IDKey: "id", NameKey: "floating_ip_address", VersionedPath: "/v2.0/floatingips", PreserveUnknownBody: true, SourceFindPolicy: true, SourceLinkPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true}
	for _, key := range []string{"id", "created_at", "description", "dns_domain", "dns_name", "fixed_ip_address", "floating_ip_address", "floating_network_id", "port_id", "qos_policy_id", "project_id", "tenant_id", "router_id", "status", "updated_at", "subnet_id"} {
		d := neutronfind.Descriptor{Name: key, Wire: key, Kind: neutronfind.Untyped, Missing: "null"}
		if key == "project_id" {
			d.Alias = "tenant_id"
		}
		p.Body = append(p.Body, d)
	}
	p.Body = append(p.Body, neutronfind.Descriptor{Name: "name", Wire: "floating_ip_address", Kind: neutronfind.Untyped, Missing: "null"}, neutronfind.Descriptor{Name: "port_details", Wire: "port_details", Kind: neutronfind.Dictionary, Missing: "null"}, neutronfind.Descriptor{Name: "tags", Wire: "tags", Kind: neutronfind.List, Missing: "[]"}, neutronfind.Descriptor{Name: "revision_number", Wire: "revision_number", Kind: neutronfind.Integer, Missing: "null"})
	p.Headers = []neutronfind.Descriptor{{Name: "if_match", Wire: "if-match", Kind: neutronfind.List, Missing: "null"}}
	return p
}

// availableIPRow postpones Source Resource construction until the row is
// consumed. Later malformed rows must not prevent an earlier free-IP return.
type availableIPRow struct {
	raw      json.RawMessage
	metadata resource.Metadata
	record   *neutronfind.Record
}

func (v *availableIPRow) UnmarshalJSON(raw []byte) error { v.raw = bytes.Clone(raw); return nil }

// FindAvailableIPRecord lists floating IPs without a server query and returns
// the first Source-falsey port_id. It stops the pager immediately at that row.
// Exhaustion returns nil, nil. Callers do not need their own selection loop.
func (a *API) FindAvailableIPRecord(ctx context.Context, options ...AvailableIPOption) (*AvailableIPRecord, error) {
	wrap := func(err error) error { return request.Wrap("FindAvailableIPRecord", "floatingips", err) }
	if a == nil {
		return nil, wrap(errors.Join(resource.ErrInvalidOption, errors.New("floating IP API required")))
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, availableIPProfile(), func(context.Context) error {
		if a.client != selected {
			return errors.Join(resource.ErrInvalidOption, errors.New("selected floating IP client changed"))
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, wrap(err)
	}
	cfg := AvailableIPOpts{}
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(errors.Join(resource.ErrInvalidOption, errors.New("nil available IP option")))
		}
		next := copyAvailableIP(cfg)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyAvailableIP(next)
	}
	if err := source.WithPolicy(ctx, cfg.Microversion, cfg.Headers); err != nil {
		return nil, wrap(err)
	}
	codes := make([]int, 300)
	for i := range codes {
		codes[i] = 100 + i
	}
	rejected := make([]int, 200)
	for i := range rejected {
		rejected[i] = 400 + i
	}
	var origin *rest.Response
	spec := rest.CollectionSpec[availableIPRow]{Client: &source.Client, Path: "floatingips", Kind: "floatingips", PluralKey: "floatingips", Validate: engine.Check, SourceGuard: engine.Check, ListCodes: codes,
		Metadata: func(v *availableIPRow) *resource.Metadata { return &v.metadata },
		ReadPage: func(ctx context.Context, target string, codes ...int) (*rest.Response, error) {
			return rest.DoJSONGuardedRejections(ctx, &source.Client, engine.Check, http.MethodGet, target, nil, nil, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, codes...)
		},
		ValidateResponse: func(response *rest.Response) error { origin = response; return engine.Check(ctx) },
		Paging: rest.PagePolicy[availableIPRow]{HTTPLink: true, DictionaryLinks: true, SourceLinkPolicy: true, IgnoreFalseyNext: true, NextTruthy: cloudfilter.PythonJSONTruthy, MarkerFallback: true, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, VersionedPath: "/v2.0/floatingips", Marker: func(v *availableIPRow) (string, error) {
			if v == nil || v.record == nil || v.record.Wire == nil {
				return "", resource.ErrInvalidOption
			}
			values, err := cloudfilter.RequestQueryValues(v.record.Wire.Body["id"])
			if err != nil || len(values) != 1 || values[0] == "" {
				return "", errors.Join(resource.ErrInvalidOption, err)
			}
			return values[0], nil
		}},
	}
	for row, err := range rest.ListWithControl(ctx, spec, nil, rest.ListControl{}) {
		if err != nil {
			return nil, wrap(err)
		}
		value, err := engine.ProjectExistingRow(ctx, row.raw, origin, neutronfind.ExistingRowPolicy{})
		row.record = value

		if err != nil {
			return nil, wrap(err)
		}
		if value == nil || value.Resource == nil {
			return nil, wrap(errors.Join(resource.ErrInvalidOption, errors.New("floating IP list returned no resource")))
		}
		raw := value.Resource.Body["port_id"]
		if raw == nil {
			raw = json.RawMessage("null")
		}
		truthy, cause := cloudfilter.PythonJSONTruthy(raw)
		if cause = errors.Join(cause, engine.Check(ctx)); cause != nil {
			return nil, wrap(cause)
		}
		if !truthy {
			return &AvailableIPRecord{Resource: value.Resource.Clone(), Wire: value.Wire.Clone(), Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}, nil
		}
	}
	return nil, wrap(engine.Check(ctx))
}
