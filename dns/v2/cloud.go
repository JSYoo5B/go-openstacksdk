package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/internal/designate"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/recordsets"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zones"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// This file maps the pinned openstacksdk cloud layer (DnsCloudMixin) onto the
// Service. Zones are addressed by name or ID: a GET by identifier, then a list
// filtered by name, ID/name match, duplicate detection (resource.FindIdentity).

type rawRow struct{ designate.Evidence }

// zoneCollection builds a zone lookup collection with identity find enabled.
func (s *Service) zoneCollection() *resource.Collection[zones.Zone] {
	api := s.Zones
	return resource.NewCollection(resource.Adapter[zones.Zone]{
		Kind:         "zones",
		IdentityFind: true,
		Get:          func(ctx context.Context, id string) (*zones.Zone, error) { return api.Get(ctx, id) },
		ID:           func(v *zones.Zone) string { return v.ID },
		Name:         func(v *zones.Zone) string { return v.Name },
		NameQuery:    func(name string) string { return name },
		Iterate: func(ctx context.Context, q url.Values) iter.Seq2[*zones.Zone, error] {
			options := make([]zones.ListOption, 0, len(q))
			for key, values := range q {
				for _, value := range values {
					options = append(options, zones.WithListQuery(key, value))
				}
			}
			return api.List(ctx, options...)
		},
	})
}

// GetDNSZone is cloud get_zone: find a zone by ID or name, returning nil, nil
// when absent. Python's filters argument raises TypeError for any non-empty
// value in the pinned proxy.find_zone, so none is offered.
func (s *Service) GetDNSZone(ctx context.Context, nameOrID string, options ...resource.IdentityFindOption) (*zones.Zone, error) {
	value, err := s.zoneCollection().FindIdentity(ctx, nameOrID, options...)
	return value, request.Wrap("GetDNSZone", "dns zones", err)
}

func (s *Service) requireZone(ctx context.Context, identity string) (*zones.Zone, error) {
	zone, err := s.zoneCollection().FindIdentity(ctx, identity)
	if err != nil {
		return nil, err
	}
	if zone == nil {
		return nil, &resource.NotFoundError{Resource: "zone", Reference: identity}
	}
	return zone, nil
}

// ListDNSZones is cloud list_zones: every zone matching the typed query.
func (s *Service) ListDNSZones(ctx context.Context, options ...zones.ListOption) ([]*zones.Zone, error) {
	var out []*zones.Zone
	for zone, err := range s.Zones.List(ctx, options...) {
		if err != nil {
			return nil, request.Wrap("ListDNSZones", "dns zones", err)
		}
		out = append(out, zone)
	}
	return out, nil
}

// DNSSearchOpts configures cloud search helpers.
type DNSSearchOpts struct {
	// Filters is a flat map applied locally to the raw rows (and, for zones,
	// sent as query parameters like Python) or a JMESPath expression string
	// (record sets only).
	Filters any
}

type DNSSearchOption = request.Option[DNSSearchOpts]

// WithDNSSearchFilters supplies a map of exact-match filters or, for record
// sets, a JMESPath string. The value is snapshotted as JSON immediately.
func WithDNSSearchFilters(filters any) DNSSearchOption {
	encoded, err := json.Marshal(filters)
	return func(c *request.Config[DNSSearchOpts]) error {
		if err != nil {
			return fmt.Errorf("%w: search filters: %v", resource.ErrInvalidOption, err)
		}
		var snapshot any
		if err := json.Unmarshal(encoded, &snapshot); err != nil {
			return err
		}
		c.Options.Filters = snapshot
		return nil
	}
}

// DNSSearchResult holds the selected rows. With a JMESPath expression the
// result is arbitrary JSON in Value and Items is nil.
type DNSSearchResult[T any] struct {
	Items      []*T
	Value      json.RawMessage
	Expression bool
}

func prepareSearch(options []DNSSearchOption) (*json.RawMessage, any, error) {
	cfg, err := request.Apply(DNSSearchOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, false)
	}
	if err != nil {
		return nil, nil, err
	}
	if cfg.Options.Filters == nil {
		return nil, nil, nil
	}
	encoded, err := json.Marshal(cfg.Options.Filters)
	if err != nil {
		return nil, nil, err
	}
	raw := json.RawMessage(encoded)
	return &raw, cfg.Options.Filters, nil
}

func pythonString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case bool:
		if v {
			return "True", nil
		}
		return "False", nil
	case json.Number:
		return v.String(), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	}
	return "", fmt.Errorf("%w: only scalar filters can become query parameters", resource.ErrInvalidOption)
}

func selectRows[T any](rows []*rawRow, nameOrID string, filters *json.RawMessage) (*DNSSearchResult[T], error) {
	encoded := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		object, err := json.Marshal(row.Body)
		if err != nil {
			return nil, err
		}
		encoded[i] = object
	}
	result, err := cloudfilter.Select(encoded, nameOrID, filters, nil)
	if err != nil {
		return nil, err
	}
	if result.Expression {
		return &DNSSearchResult[T]{Value: result.Value, Expression: true}, nil
	}
	out := &DNSSearchResult[T]{Value: result.Value}
	for _, index := range result.Indices {
		item := new(T)
		if err := json.Unmarshal(encoded[index], item); err != nil {
			return nil, err
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

func (s *Service) rawRows(ctx context.Context, path []string, plural string, query url.Values) ([]*rawRow, error) {
	var rows []*rawRow
	for row, err := range designate.List[rawRow](ctx, s.client, designate.ListSpec{Path: path, PluralKey: plural}, query, nil, resource.ListControl{}) {
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// SearchDNSZones is cloud search_zones: list zones (the filter map is also sent
// as query parameters, as in Python), then select by name/ID glob and filters.
func (s *Service) SearchDNSZones(ctx context.Context, nameOrID string, options ...DNSSearchOption) (*DNSSearchResult[zones.Zone], error) {
	wrap := func(err error) error { return request.Wrap("SearchDNSZones", "dns zones", err) }
	filters, parsed, err := prepareSearch(options)
	if err != nil {
		return nil, wrap(err)
	}
	query := url.Values{}
	switch f := parsed.(type) {
	case nil:
	case map[string]any:
		for key, value := range f {
			text, err := pythonString(value)
			if err != nil {
				return nil, wrap(fmt.Errorf("filter %q: %w", key, err))
			}
			query.Set(key, text)
		}
	default:
		return nil, wrap(fmt.Errorf("%w: zone filters must be a map", resource.ErrInvalidOption))
	}
	rows, err := s.rawRows(ctx, []string{"zones"}, "zones", query)
	if err != nil {
		return nil, wrap(err)
	}
	result, err := selectRows[zones.Zone](rows, nameOrID, filters)
	return result, wrap(err)
}

// DNSZoneCreateOpts holds cloud create_zone's optional arguments. A nil Email
// or Description is sent as JSON null, like Python; nil TTL, ZoneType and
// Masters are omitted. ZoneType is upper-cased and must be PRIMARY or
// SECONDARY; SECONDARY zones drop email and ttl as the pinned proxy does.
type DNSZoneCreateOpts struct {
	ZoneType    *string
	Email       *string
	Description *string
	TTL         *int
	Masters     []string
}

type DNSZoneCreateOption = request.Option[DNSZoneCreateOpts]

func WithDNSZoneCreateOptions(value DNSZoneCreateOpts) DNSZoneCreateOption {
	snapshot := cloneZoneCreate(value)
	return func(c *request.Config[DNSZoneCreateOpts]) error {
		c.Options = cloneZoneCreate(snapshot)
		return nil
	}
}

func cloneZoneCreate(v DNSZoneCreateOpts) DNSZoneCreateOpts {
	cp := func(p *string) *string {
		if p == nil {
			return nil
		}
		x := *p
		return &x
	}
	v.ZoneType, v.Email, v.Description = cp(v.ZoneType), cp(v.Email), cp(v.Description)
	if v.TTL != nil {
		t := *v.TTL
		v.TTL = &t
	}
	if v.Masters != nil {
		v.Masters = append([]string{}, v.Masters...)
	}
	return v
}

// CreateDNSZone is cloud create_zone. A 201 or 202 only acknowledges creation;
// wait with Zones.WaitFor(..., "ACTIVE"). Failures keep their cause.
func (s *Service) CreateDNSZone(ctx context.Context, name string, options ...DNSZoneCreateOption) (*zones.Zone, error) {
	wrap := func(err error) error { return request.Wrap("CreateDNSZone", "dns zones", err) }
	cfg, err := request.Apply(DNSZoneCreateOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, false)
	}
	if err != nil {
		return nil, wrap(err)
	}
	opts := cloneZoneCreate(cfg.Options)
	body := map[string]any{"name": name, "email": nil, "description": nil}
	if opts.Email != nil {
		body["email"] = *opts.Email
	}
	if opts.Description != nil {
		body["description"] = *opts.Description
	}
	if opts.TTL != nil {
		body["ttl"] = *opts.TTL
	}
	secondary := false
	if opts.ZoneType != nil {
		zoneType := strings.ToUpper(*opts.ZoneType)
		if zoneType != "PRIMARY" && zoneType != "SECONDARY" {
			return nil, wrap(fmt.Errorf("%w: invalid type %s, valid choices are PRIMARY or SECONDARY", resource.ErrInvalidOption, zoneType))
		}
		body["type"] = zoneType
		secondary = zoneType == "SECONDARY"
	}
	if opts.Masters != nil {
		body["masters"] = opts.Masters
	}
	if secondary {
		delete(body, "email")
		delete(body, "ttl")
	}
	response, err := designate.Do(ctx, s.client, http.MethodPost, s.client.ServiceURL("zones"), body, nil, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		return nil, wrap(fmt.Errorf("unable to create zone %s: %w", name, err))
	}
	var zone zones.Zone
	if err := json.Unmarshal(response.Body, &zone); err != nil {
		return nil, wrap(response.Fail(err))
	}
	return &zone, nil
}

// UpdateDNSZone is cloud update_zone: resolve the zone, then PATCH only the
// supplied email, description, ttl and masters. A missing zone is an error
// wrapping resource.ErrNotFound.
func (s *Service) UpdateDNSZone(ctx context.Context, nameOrID string, opts zones.UpdateOpts, options ...zones.UpdateOption) (*zones.Zone, error) {
	zone, err := s.requireZone(ctx, nameOrID)
	if err != nil {
		return nil, request.Wrap("UpdateDNSZone", "dns zones", err)
	}
	value, err := s.Zones.Update(ctx, zone.ID, opts, options...)
	return value, request.Wrap("UpdateDNSZone", "dns zones", err)
}

// DeleteDNSZone is cloud delete_zone: false, nil when no zone matches, else
// DELETE (202 acknowledgement) and true. A concurrent 404 after lookup is
// ignored like Python's ignore_missing default.
func (s *Service) DeleteDNSZone(ctx context.Context, nameOrID string) (bool, error) {
	zone, err := s.zoneCollection().FindIdentity(ctx, nameOrID)
	if err != nil {
		return false, request.Wrap("DeleteDNSZone", "dns zones", err)
	}
	if zone == nil {
		return false, nil
	}
	if err := s.Zones.Remove(ctx, resource.ID(zone.ID)); err != nil {
		return false, request.Wrap("DeleteDNSZone", "dns zones", err)
	}
	return true, nil
}

// ListDNSRecordSets is cloud list_recordsets for a zone name or ID.
func (s *Service) ListDNSRecordSets(ctx context.Context, zone string) ([]*recordsets.RecordSet, error) {
	wrap := func(err error) error { return request.Wrap("ListDNSRecordSets", "dns recordsets", err) }
	found, err := s.requireZone(ctx, zone)
	if err != nil {
		return nil, wrap(err)
	}
	var out []*recordsets.RecordSet
	for value, err := range s.RecordSets.ListByZone(ctx, found.ID) {
		if err != nil {
			return nil, wrap(err)
		}
		out = append(out, value)
	}
	return out, nil
}

func (s *Service) recordSetScope(ctx context.Context, zone string) (*recordsets.RecordSetScope, error) {
	found, err := s.requireZone(ctx, zone)
	if err != nil {
		return nil, err
	}
	return s.RecordSets.InZone(ctx, resource.ID(found.ID))
}

// GetDNSRecordSet is cloud get_recordset: nil, nil when no record set matches.
func (s *Service) GetDNSRecordSet(ctx context.Context, zone, nameOrID string, options ...resource.IdentityFindOption) (*recordsets.RecordSet, error) {
	wrap := func(err error) error { return request.Wrap("GetDNSRecordSet", "dns recordsets", err) }
	scope, err := s.recordSetScope(ctx, zone)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := scope.FindIdentity(ctx, nameOrID, options...)
	return value, wrap(err)
}

// SearchDNSRecordSets is cloud search_recordsets: list the zone's record sets,
// then select by name/ID glob and filters (a map or JMESPath string).
func (s *Service) SearchDNSRecordSets(ctx context.Context, zone, nameOrID string, options ...DNSSearchOption) (*DNSSearchResult[recordsets.RecordSet], error) {
	wrap := func(err error) error { return request.Wrap("SearchDNSRecordSets", "dns recordsets", err) }
	filters, _, err := prepareSearch(options)
	if err != nil {
		return nil, wrap(err)
	}
	found, err := s.requireZone(ctx, zone)
	if err != nil {
		return nil, wrap(err)
	}
	rows, err := s.rawRows(ctx, []string{"zones", found.ID, "recordsets"}, "recordsets", nil)
	if err != nil {
		return nil, wrap(err)
	}
	result, err := selectRows[recordsets.RecordSet](rows, nameOrID, filters)
	return result, wrap(err)
}

// CreateDNSRecordSet is cloud create_recordset. recordsetType is upper-cased;
// an empty description and a zero ttl are omitted (Python truthiness).
func (s *Service) CreateDNSRecordSet(ctx context.Context, zone, name, recordsetType string, records []string, description string, ttl int) (*recordsets.RecordSet, error) {
	wrap := func(err error) error { return request.Wrap("CreateDNSRecordSet", "dns recordsets", err) }
	found, err := s.requireZone(ctx, zone)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := s.RecordSets.Create(ctx, found.ID, recordsets.CreateOpts{Name: name, Type: strings.ToUpper(recordsetType), Records: records, Description: description, TTL: ttl})
	return value, wrap(err)
}

// UpdateDNSRecordSet is cloud update_recordset: resolve zone and record set,
// then PATCH only the supplied description, ttl and records.
func (s *Service) UpdateDNSRecordSet(ctx context.Context, zone, nameOrID string, opts recordsets.UpdateOpts, options ...recordsets.UpdateOption) (*recordsets.RecordSet, error) {
	wrap := func(err error) error { return request.Wrap("UpdateDNSRecordSet", "dns recordsets", err) }
	scope, err := s.recordSetScope(ctx, zone)
	if err != nil {
		return nil, wrap(err)
	}
	found, err := scope.FindIdentity(ctx, nameOrID)
	if err != nil {
		return nil, wrap(err)
	}
	if found == nil {
		return nil, wrap(&resource.NotFoundError{Resource: "recordset", Reference: nameOrID})
	}
	value, err := scope.Update(ctx, resource.ID(found.ID), opts, options...)
	return value, wrap(err)
}

// DeleteDNSRecordSet is cloud delete_recordset: false, nil when no record set
// matches; otherwise a strict DELETE (a 404 after lookup is an error).
func (s *Service) DeleteDNSRecordSet(ctx context.Context, zone, nameOrID string) (bool, error) {
	wrap := func(err error) error { return request.Wrap("DeleteDNSRecordSet", "dns recordsets", err) }
	scope, err := s.recordSetScope(ctx, zone)
	if err != nil {
		return false, wrap(err)
	}
	found, err := scope.FindIdentity(ctx, nameOrID)
	if err != nil {
		return false, wrap(err)
	}
	if found == nil {
		return false, nil
	}
	if err := scope.Collection.Delete(ctx, resource.ID(found.ID), resource.WithMissingError()); err != nil {
		return false, wrap(err)
	}
	return true, nil
}
