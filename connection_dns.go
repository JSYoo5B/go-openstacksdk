package openstack

import (
	"context"

	dnsv2 "github.com/JSYoo5B/go-openstacksdk/dns/v2"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/recordsets"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zones"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// The DNS cloud helpers authenticate through Connection.DNS and then run the
// dns/v2 Service helpers of the same names; see dns/v2/cloud.go for contracts.

func (c *Connection) ListDNSZones(ctx context.Context, options ...zones.ListOption) ([]*zones.Zone, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.ListDNSZones(ctx, options...)
}

func (c *Connection) GetDNSZone(ctx context.Context, nameOrID string, options ...resource.IdentityFindOption) (*zones.Zone, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.GetDNSZone(ctx, nameOrID, options...)
}

func (c *Connection) SearchDNSZones(ctx context.Context, nameOrID string, options ...dnsv2.DNSSearchOption) (*dnsv2.DNSSearchResult[zones.Zone], error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.SearchDNSZones(ctx, nameOrID, options...)
}

func (c *Connection) CreateDNSZone(ctx context.Context, name string, options ...dnsv2.DNSZoneCreateOption) (*zones.Zone, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.CreateDNSZone(ctx, name, options...)
}

func (c *Connection) UpdateDNSZone(ctx context.Context, nameOrID string, opts zones.UpdateOpts, options ...zones.UpdateOption) (*zones.Zone, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.UpdateDNSZone(ctx, nameOrID, opts, options...)
}

func (c *Connection) DeleteDNSZone(ctx context.Context, nameOrID string) (bool, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return false, err
	}
	return s.DeleteDNSZone(ctx, nameOrID)
}

func (c *Connection) ListDNSRecordSets(ctx context.Context, zone string) ([]*recordsets.RecordSet, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.ListDNSRecordSets(ctx, zone)
}

func (c *Connection) GetDNSRecordSet(ctx context.Context, zone, nameOrID string, options ...resource.IdentityFindOption) (*recordsets.RecordSet, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.GetDNSRecordSet(ctx, zone, nameOrID, options...)
}

func (c *Connection) SearchDNSRecordSets(ctx context.Context, zone, nameOrID string, options ...dnsv2.DNSSearchOption) (*dnsv2.DNSSearchResult[recordsets.RecordSet], error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.SearchDNSRecordSets(ctx, zone, nameOrID, options...)
}

func (c *Connection) CreateDNSRecordSet(ctx context.Context, zone, name, recordsetType string, records []string, description string, ttl int) (*recordsets.RecordSet, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.CreateDNSRecordSet(ctx, zone, name, recordsetType, records, description, ttl)
}

func (c *Connection) UpdateDNSRecordSet(ctx context.Context, zone, nameOrID string, opts recordsets.UpdateOpts, options ...recordsets.UpdateOption) (*recordsets.RecordSet, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return nil, err
	}
	return s.UpdateDNSRecordSet(ctx, zone, nameOrID, opts, options...)
}

func (c *Connection) DeleteDNSRecordSet(ctx context.Context, zone, nameOrID string) (bool, error) {
	s, err := c.DNS(ctx)
	if err != nil {
		return false, err
	}
	return s.DeleteDNSRecordSet(ctx, zone, nameOrID)
}
