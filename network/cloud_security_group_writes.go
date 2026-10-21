package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// SecurityGroupUpdate lists the only fields cloud update_security_group
// accepts. A nil pointer omits the field; a non-nil pointer to an empty string
// or false is sent.
type SecurityGroupUpdate struct {
	Name        *string
	Description *string
	Stateful    *bool
}

// SecurityGroupWriteResult reports the selected backend and the response of a
// security group mutation. Group is set for the Neutron backend. NovaBody holds
// the raw Nova response envelope value without Python munch normalization.
type SecurityGroupWriteResult struct {
	Backend  string
	Group    *groups.SecGroup
	Rule     *rules.SecGroupRule
	NovaBody json.RawMessage
}

// SecurityGroupRuleSpec mirrors the create_security_group_rule parameters.
// Nil pointers mean Python None. Empty Direction and Ethertype select the
// Python defaults ingress and IPv4.
type SecurityGroupRuleSpec struct {
	PortRangeMin         *int
	PortRangeMax         *int
	Protocol             *string
	RemoteIPPrefix       *string
	RemoteGroupID        *string
	RemoteAddressGroupID *string
	Direction            string
	Ethertype            string
	ProjectID            *string
	Description          *string
}

func securityGroupWriteWrap(ctx context.Context, op string, e error) error {
	return request.Wrap(op, "security groups", cloudread.ContextError(ctx, e))
}

// selectSecurityGroup runs the cloud get_security_group lookup (list all, select
// locally) and returns the selected group ID and its normalized body.
func (c *SecurityGroupCloud) selectSecurityGroup(ctx context.Context, nameOrID string) (string, map[string]json.RawMessage, error) {
	result, err := c.GetSecurityGroup(ctx, nameOrID)
	if err != nil {
		return "", nil, err
	}
	if result == nil || len(result.Value) == 0 || string(result.Value) == "null" {
		return "", nil, nil
	}
	var body map[string]json.RawMessage
	if e := json.Unmarshal(result.Value, &body); e != nil {
		return "", nil, fmt.Errorf("selected security group is not a mapping: %w", e)
	}
	var id string
	if e := json.Unmarshal(body["id"], &id); e != nil || id == "" {
		return "", nil, fmt.Errorf("selected security group has no string id")
	}
	return id, body, nil
}

func securityGroupNovaStatus() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}

// DeleteSecurityGroup deletes the group selected by exact name or ID after a
// full list lookup. It returns false without a mutation when nothing matches and
// true after a successful delete; a delete that fails (including 404 after the
// lookup) is an error. Duplicate matches fail before any mutation. The helper
// never retries the delete.
func (c *SecurityGroupCloud) DeleteSecurityGroup(ctx context.Context, nameOrID string) (bool, error) {
	const op = "DeleteSecurityGroup"
	state, err := c.start(ctx, nil)
	if err != nil {
		return false, securityGroupWriteWrap(ctx, op, err)
	}
	id, _, err := c.selectSecurityGroup(ctx, nameOrID)
	if err != nil || id == "" {
		return false, securityGroupWriteWrap(ctx, op, err)
	}
	if state.backend == "neutron" {
		err = state.api.DeleteWithOptions(ctx, id, groups.WithDeleteIgnoreMissing(false))
	} else {
		client := state.source.Client
		_, err = client.Delete(ctx, client.ServiceURL("os-security-groups", url.PathEscape(id)), &gophercloud.RequestOpts{OkCodes: securityGroupNovaStatus()})
	}
	if err != nil {
		return false, securityGroupWriteWrap(ctx, op, err)
	}
	return true, nil
}

// UpdateSecurityGroup updates the group selected by exact name or ID. A missing
// group is a resource.ErrNotFound error before any mutation. Neutron sends only
// the supplied fields (no HTTP when none are supplied); the legacy Nova branch
// fills a missing name and description from the selected group before the PUT.
func (c *SecurityGroupCloud) UpdateSecurityGroup(ctx context.Context, nameOrID string, update SecurityGroupUpdate) (*SecurityGroupWriteResult, error) {
	const op = "UpdateSecurityGroup"
	state, err := c.start(ctx, nil)
	if err != nil {
		return nil, securityGroupWriteWrap(ctx, op, err)
	}
	id, body, err := c.selectSecurityGroup(ctx, nameOrID)
	if err != nil {
		return nil, securityGroupWriteWrap(ctx, op, err)
	}
	if id == "" {
		return nil, securityGroupWriteWrap(ctx, op, &resource.NotFoundError{Resource: "security group", Reference: nameOrID})
	}
	result := &SecurityGroupWriteResult{Backend: state.backend}
	if state.backend == "neutron" {
		var options []groups.AttributeOption
		if update.Name != nil {
			options = append(options, groups.WithAttribute("name", *update.Name))
		}
		if update.Description != nil {
			options = append(options, groups.WithAttribute("description", *update.Description))
		}
		if update.Stateful != nil {
			options = append(options, groups.WithAttribute("stateful", *update.Stateful))
		}
		result.Group, err = state.api.UpdateWithAttributes(ctx, id, options...)
		if err != nil {
			return nil, securityGroupWriteWrap(ctx, op, err)
		}
		return result, nil
	}
	payload := map[string]any{}
	if update.Stateful != nil {
		payload["stateful"] = *update.Stateful
	}
	for key, value := range map[string]*string{"name": update.Name, "description": update.Description} {
		if value != nil {
			payload[key] = *value
		} else {
			payload[key] = body[key]
		}
	}
	client := state.source.Client
	var envelope map[string]json.RawMessage
	_, err = client.Put(ctx, client.ServiceURL("os-security-groups", url.PathEscape(id)), map[string]any{"security_group": payload}, &envelope, &gophercloud.RequestOpts{OkCodes: securityGroupNovaStatus()})
	if err == nil {
		if raw, ok := envelope["security_group"]; ok {
			result.NovaBody = bytes.Clone(raw)
		} else {
			err = errors.New("Nova response is missing security_group")
		}
	}
	if err != nil {
		return nil, securityGroupWriteWrap(ctx, op, err)
	}
	return result, nil
}

// CreateSecurityGroupRule creates a rule in the group selected by exact name or
// ID. Neutron converts port -1 to null and always sends the Python body keys
// (unset values as null; a falsy remote_address_group_id is dropped by the
// native rule mutation). The legacy Nova branch requires a protocol, rejects
// egress, and fills ICMP -1 ports or the TCP/UDP 1-65535 range.
func (c *SecurityGroupCloud) CreateSecurityGroupRule(ctx context.Context, nameOrID string, spec SecurityGroupRuleSpec) (*SecurityGroupWriteResult, error) {
	const op = "CreateSecurityGroupRule"
	state, err := c.start(ctx, nil)
	if err != nil {
		return nil, securityGroupWriteWrap(ctx, op, err)
	}
	id, _, err := c.selectSecurityGroup(ctx, nameOrID)
	if err != nil {
		return nil, securityGroupWriteWrap(ctx, op, err)
	}
	if id == "" {
		return nil, securityGroupWriteWrap(ctx, op, &resource.NotFoundError{Resource: "security group", Reference: nameOrID})
	}
	direction, ethertype := spec.Direction, spec.Ethertype
	if direction == "" {
		direction = "ingress"
	}
	if ethertype == "" {
		ethertype = "IPv4"
	}
	min, max := spec.PortRangeMin, spec.PortRangeMax
	result := &SecurityGroupWriteResult{Backend: state.backend}
	if state.backend == "neutron" {
		port := func(p *int) any {
			if p == nil || *p == -1 {
				return nil
			}
			return *p
		}
		str := func(p *string) any {
			if p == nil {
				return nil
			}
			return *p
		}
		options := []rules.AttributeOption{
			rules.WithAttribute("security_group_id", id),
			rules.WithAttribute("port_range_min", port(min)),
			rules.WithAttribute("port_range_max", port(max)),
			rules.WithAttribute("protocol", str(spec.Protocol)),
			rules.WithAttribute("remote_ip_prefix", str(spec.RemoteIPPrefix)),
			rules.WithAttribute("remote_group_id", str(spec.RemoteGroupID)),
			rules.WithAttribute("remote_address_group_id", str(spec.RemoteAddressGroupID)),
			rules.WithAttribute("direction", direction),
			rules.WithAttribute("ethertype", ethertype),
		}
		if spec.ProjectID != nil {
			options = append(options, rules.WithAttribute("tenant_id", *spec.ProjectID))
		}
		if spec.Description != nil {
			options = append(options, rules.WithAttribute("description", *spec.Description))
		}
		result.Rule, err = rules.New(&state.source.Client).CreateWithAttributes(ctx, options...)
		if err != nil {
			return nil, securityGroupWriteWrap(ctx, op, err)
		}
		return result, nil
	}
	if spec.Protocol == nil {
		return nil, securityGroupWriteWrap(ctx, op, fmt.Errorf("%w: protocol must be specified", resource.ErrInvalidOption))
	}
	if direction == "egress" {
		return nil, securityGroupWriteWrap(ctx, op, fmt.Errorf("%w: no support for egress rules", resource.ErrInvalidOption))
	}
	var from, to any
	if min != nil {
		from = *min
	}
	if max != nil {
		to = *max
	}
	switch *spec.Protocol {
	case "icmp":
		if min == nil {
			from = -1
		}
		if max == nil {
			to = -1
		}
	case "tcp", "udp":
		if min == nil && max == nil {
			from, to = 1, 65535
		}
	}
	rule := map[string]any{"parent_group_id": id, "ip_protocol": *spec.Protocol, "from_port": from, "to_port": to}
	rule["cidr"] = nilString(spec.RemoteIPPrefix)
	rule["group_id"] = nilString(spec.RemoteGroupID)
	if spec.ProjectID != nil {
		rule["tenant_id"] = *spec.ProjectID
	}
	client := state.source.Client
	var envelope map[string]json.RawMessage
	_, err = client.Post(ctx, client.ServiceURL("os-security-group-rules"), map[string]any{"security_group_rule": rule}, &envelope, &gophercloud.RequestOpts{OkCodes: securityGroupNovaStatus()})
	if err == nil {
		if raw, ok := envelope["security_group_rule"]; ok {
			result.NovaBody = bytes.Clone(raw)
		} else {
			err = errors.New("Nova response is missing security_group_rule")
		}
	}
	if err != nil {
		return nil, securityGroupWriteWrap(ctx, op, err)
	}
	return result, nil
}

func nilString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
