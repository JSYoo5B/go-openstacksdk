// Package serverview renders the JSON mappings shared by the cloud server
// reads and the hostvars expansion: the record or typed-server base view, the
// interface fields of the address expansion, and the record-to-server
// projection. It depends only on the Nova server models so that both the
// compute package and Connection can use it without an import cycle.
package serverview

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Interfaces carries the address expansion fields set over a base view.
// Addresses holds the owned JSON fields of each address row per network.
type Interfaces struct {
	Addresses                           map[string][]map[string]json.RawMessage
	PublicIPv4, PublicIPv6, PrivateIPv4 string
	InterfaceIP, AccessIPv4, AccessIPv6 string
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// ObjectWith sets key to value in the JSON object raw. A missing or null raw
// starts an empty object; any other nonobject is rejected.
func ObjectWith(raw json.RawMessage, key string, value json.RawMessage) (json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
		if err := json.Unmarshal(trimmed, &fields); err != nil {
			return nil, fmt.Errorf("%w: hostvars %s base is not an object", resource.ErrInvalidOption, key)
		}
	}
	fields[key] = bytes.Clone(value)
	return json.Marshal(fields)
}

// Base returns the Source Resource view of record when present, else the
// typed server JSON (with its wire names) plus a computed location and the
// server image object.
func Base(server *servers.Server, record *servers.ServerFindRecord, location resource.CloudLocation) (map[string]json.RawMessage, error) {
	body := map[string]json.RawMessage{}
	if record != nil && record.Resource != nil {
		for key, value := range record.Resource.Body {
			body[key] = bytes.Clone(value)
		}
		return body, nil
	}
	if server == nil {
		return nil, invalid("server is required")
	}
	raw, err := json.Marshal(server)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	project, _ := json.Marshal(server.TenantID)
	zone, _ := json.Marshal(server.AvailabilityZone)
	if server.TenantID == "" {
		project = nil
	}
	if server.AvailabilityZone == "" {
		zone = nil
	}
	if body["location"], err = location.ForResource(project, zone); err != nil {
		return nil, err
	}
	if server.Image != nil {
		if body["image"], err = json.Marshal(server.Image); err != nil {
			return nil, err
		}
	}
	return body, nil
}

// Overlay sets addresses, public_v4, public_v6, private_v4, interface_ip,
// access_ipv4 and access_ipv6 from view over body. A nil view changes nothing.
func Overlay(body map[string]json.RawMessage, view *Interfaces) error {
	if view == nil {
		return nil
	}
	addresses := make(map[string][]json.RawMessage, len(view.Addresses))
	for network, rows := range view.Addresses {
		list := make([]json.RawMessage, 0, len(rows))
		for _, row := range rows {
			raw, err := json.Marshal(row)
			if err != nil {
				return err
			}
			list = append(list, raw)
		}
		addresses[network] = list
	}
	var err error
	if body["addresses"], err = json.Marshal(addresses); err != nil {
		return err
	}
	for key, value := range map[string]string{
		"public_v4": view.PublicIPv4, "public_v6": view.PublicIPv6, "private_v4": view.PrivateIPv4,
		"interface_ip": view.InterfaceIP, "access_ipv4": view.AccessIPv4, "access_ipv6": view.AccessIPv6,
	} {
		body[key], _ = json.Marshal(value)
	}
	return nil
}

// FromRecord projects the received row of a record lookup onto the typed Nova
// server. A nil record is (nil, nil).
func FromRecord(record *servers.ServerFindRecord) (*servers.Server, error) {
	if record == nil {
		return nil, nil
	}
	source := record.Wire
	if source == nil {
		source = record.Resource
	}
	if source == nil {
		return nil, invalid("server record has no body")
	}
	raw, err := json.Marshal(source.Body)
	if err != nil {
		return nil, err
	}
	var server servers.Server
	if err := json.Unmarshal(raw, &server); err != nil {
		return nil, fmt.Errorf("%w: decode server record: %w", resource.ErrInvalidOption, err)
	}
	return &server, nil
}
