// Package quotas reads effective Barbican quotas and manages project overrides.
package quotas

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Quota preserves exact response values, including omitted fields, JSON null,
// large integers and extensions. It is not a mutable cached Python Resource.
type Quota struct {
	resource.Metadata
	Secrets    json.RawMessage            `json:"secrets,omitempty"`
	Orders     json.RawMessage            `json:"orders,omitempty"`
	Containers json.RawMessage            `json:"containers,omitempty"`
	Consumers  json.RawMessage            `json:"consumers,omitempty"`
	CAs        json.RawMessage            `json:"cas,omitempty"`
	Data       map[string]json.RawMessage `json:"-"`
}

func (value *Quota) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("quota must be a JSON object")
	}
	copyRaw := func(raw json.RawMessage) json.RawMessage { return append(json.RawMessage(nil), raw...) }
	decoded := Quota{Metadata: resource.Metadata{Body: fields}, Data: make(map[string]json.RawMessage)}
	decoded.Secrets, decoded.Orders = copyRaw(fields["secrets"]), copyRaw(fields["orders"])
	decoded.Containers, decoded.Consumers, decoded.CAs = copyRaw(fields["containers"]), copyRaw(fields["consumers"]), copyRaw(fields["cas"])
	for key, raw := range fields {
		switch key {
		case "secrets", "orders", "containers", "consumers", "cas":
		default:
			decoded.Data[key] = copyRaw(raw)
		}
	}
	*value = decoded
	return nil
}

// UpdateResult is the actual accepted PUT acknowledgement. Barbican returns
// HTTP 204, so no configured quota is synthesized and no GET is submitted.
type UpdateResult struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}
