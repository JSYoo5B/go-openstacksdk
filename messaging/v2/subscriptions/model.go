// Package subscriptions implements queue-scoped Zaqar subscription operations.
package subscriptions

import (
	"encoding/json"

	"gophercloudsdk/resource"
)

// Subscription retains exact raw numeric/metadata values and HTTP evidence.
// ID prefers a present canonical id over the alternate subscription_id.
type Subscription struct {
	resource.Metadata
	ID             string          `json:"-"`
	SubscriptionID string          `json:"subscription_id"`
	Subscriber     string          `json:"subscriber"`
	Source         string          `json:"source"`
	TTL            json.RawMessage `json:"ttl"`
	Age            json.RawMessage `json:"age"`
	Options        json.RawMessage `json:"options"`
}

func (value *Subscription) UnmarshalJSON(data []byte) error {
	var metadata resource.Metadata
	if err := resource.DecodeObject(data, &struct{}{}, &metadata); err != nil {
		return err
	}
	type plain Subscription
	fields := make(map[string]json.RawMessage)
	for _, key := range []string{"subscription_id", "subscriber", "source", "ttl", "age", "options"} {
		if raw, present := metadata.Body[key]; present {
			fields[key] = raw
		}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	var decoded plain
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return err
	}
	decoded.Metadata = metadata
	decoded.ID = decoded.SubscriptionID
	if raw, present := metadata.Body["id"]; present {
		var id *string
		if err := json.Unmarshal(raw, &id); err != nil {
			return err
		}
		decoded.ID = ""
		if id != nil {
			decoded.ID = *id
		}
	}
	*value = Subscription(decoded)
	return nil
}
