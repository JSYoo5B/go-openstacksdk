package compute

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gophercloudsdk/resource"
	"net/url"
	"strconv"

	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/internal/rest"
)

type floatingIPQueryParameters struct {
	query   url.Values
	local   []cloudfilter.JSONMember
	control rest.ListControl
}

var floatingIPQueryKeys = map[string]string{
	"description": "description", "fixed_ip_address": "fixed_ip_address", "floating_ip_address": "floating_ip_address",
	"floating_network_id": "floating_network_id", "id": "id", "fields": "fields", "port_id": "port_id",
	"router_id": "router_id", "status": "status", "subnet_id": "subnet_id", "project_id": "project_id",
	"tenant_id": "project_id", "sort_key": "sort_key", "sort_dir": "sort_dir", "limit": "limit", "marker": "marker",
	"tags": "tags", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any",
}
var floatingIPBodyKeys = []string{
	"id", "name", "created_at", "description", "dns_domain", "dns_name", "fixed_ip_address", "floating_ip_address",
	"floating_network_id", "port_details", "port_id", "qos_policy_id", "project_id", "tenant_id", "router_id",
	"status", "updated_at", "subnet_id", "revision_number", "tags",
}

func floatingIPQueryFilter(raw *json.RawMessage) (json.RawMessage, bool, bool, error) {
	if raw == nil {
		return nil, false, false, nil
	}
	truthy, err := cloudfilter.PythonTruthy(*raw)
	if err != nil {
		return nil, false, false, floatingIPQueryInputError("filters", err)
	}
	value := bytes.TrimSpace(*raw)
	return bytes.Clone(value), len(value) > 0 && value[0] == '{', truthy, nil
}

func prepareFloatingIPParameters(raw json.RawMessage) (floatingIPQueryParameters, error) {
	p := floatingIPQueryParameters{query: url.Values{}}
	if raw == nil {
		return p, nil
	}
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return p, floatingIPQueryInputError("Neutron filters must be a dictionary", err)
	}
	values := map[string]json.RawMessage{}
	for _, member := range members {
		values[member.Key] = member.Value
	}
	for key, wire := range floatingIPQueryKeys {
		value, present := values[key]
		if !present {
			value, present = values[wire]
		}
		if !present {
			continue
		}
		if wire == "project_id" {
			if tenant, exists := values["tenant_id"]; exists {
				value = tenant
			}
		}
		encoded, err := cloudfilter.RequestQueryValues(value)
		if err != nil {
			return p, floatingIPQueryInputError("query "+key, err)
		}
		if len(encoded) > 0 {
			p.query[wire] = encoded
		}
	}
	for _, member := range members {
		if _, query := floatingIPQueryKeys[member.Key]; query {
			continue
		}
		for _, key := range floatingIPBodyKeys {
			if member.Key == key {
				p.local = append(p.local, member)
				break
			}
		}
	}
	if value, present := values["paginated"]; present {
		truthy, err := cloudfilter.PythonTruthy(value)
		if err != nil {
			return p, err
		}
		p.control.SinglePage = !truthy
	}
	if value, present := values["max_items"]; present {
		truthy, err := cloudfilter.PythonTruthy(value)
		if err != nil {
			return p, err
		}
		if truthy {
			if bytes.Equal(value, []byte("true")) {
				p.control.MaxItems = 1
			} else {
				p.control.MaxItems, err = strconv.Atoi(string(value))
				if err != nil || p.control.MaxItems < 0 {
					return p, invalid("floating IP max_items must be a nonnegative integer")
				}
			}
		}
	}
	p.control.LimitHint = true
	return p, nil
}

func floatingIPQueryInputError(label string, cause error) error {
	return fmt.Errorf("%w: floating IP %s: %w", resource.ErrInvalidOption, label, cause)
}
