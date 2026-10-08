package compute

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// NovaFloatingIP retains the legacy Nova allocation fields and actual HTTP
// evidence. Nova has no floating-IP status; no synthetic ACTIVE is supplied.
type NovaFloatingIP struct {
	resource.Metadata
	ID           string  `json:"id"`
	Address      string  `json:"ip"`
	Pool         string  `json:"pool"`
	FixedAddress *string `json:"fixed_ip"`
	InstanceID   *string `json:"instance_id"`
	Owner        *string `json:"owner"`
	response     *rest.Response
}

func (ip *NovaFloatingIP) fail(cause error) error {
	if ip != nil && ip.response != nil {
		return ip.response.Fail(cause)
	}
	return cause
}

func (ip *NovaFloatingIP) UnmarshalJSON(data []byte) error {
	var wire struct {
		ID         json.RawMessage `json:"id"`
		Pool       string          `json:"pool"`
		InstanceID *string         `json:"instance_id"`
		Owner      *string         `json:"owner"`
	}
	var meta resource.Metadata
	if err := resource.DecodeObject(data, &wire, &meta); err != nil {
		return err
	}
	var id string
	raw := bytes.TrimSpace(wire.ID)
	if len(raw) > 0 && raw[0] == '"' {
		if err := json.Unmarshal(raw, &id); err != nil {
			return err
		}
	} else {
		// Nova-network uses integer IDs. Preserve all decimal digits; never
		// round through float64 or accept fractional/exponent representations.
		if len(raw) == 0 {
			return invalid("Nova floating IP response lacks an ID")
		}
		for _, c := range raw {
			if c < '0' || c > '9' {
				return invalid("Nova floating IP ID must be a string or decimal integer")
			}
		}
		id = string(raw)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	decodeAlias := func(canonical, legacy string) (*string, error) {
		value, present := meta.Body[canonical]
		if !present {
			value, present = meta.Body[legacy]
		}
		if !present {
			return nil, nil
		}
		var field *string
		err := json.Unmarshal(value, &field)
		return field, err
	}
	address, err := decodeAlias("floating_ip_address", "ip")
	if err != nil {
		return err
	}
	fixed, err := decodeAlias("fixed_ip_address", "fixed_ip")
	if err != nil {
		return err
	}
	*ip = NovaFloatingIP{Metadata: meta, ID: id, Pool: wire.Pool, FixedAddress: fixed, InstanceID: wire.InstanceID, Owner: wire.Owner}
	if address != nil {
		ip.Address = *address
	}
	return nil
}

// NovaFloatingIPResponse is an accepted allocation/action response, including
// an empty 202 action body. It does not assert later association convergence.
type NovaFloatingIPResponse struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

type NovaFloatingIPAssignment struct {
	FloatingIP         *NovaFloatingIP
	Reused, Allocated  bool
	AlreadyAttached    bool
	ActionAccepted     bool
	AllocationResponse *NovaFloatingIPResponse
	ActionResponse     *NovaFloatingIPResponse
}

// NovaFloatingIPSelection is a value diagnostic, not an executable plan.
type NovaFloatingIPSelection struct {
	ID, Address, Pool string
}

func novaIPInvalid(format string, args ...any) error {
	return fmt.Errorf("Nova floating IP: %w", invalid(format, args...))
}
