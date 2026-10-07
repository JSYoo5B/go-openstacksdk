package stackevents

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/orchestration/v1/stacks"
)

// EventResource retains native event fields and server extensions. Request
// identity describes the endpoint, independently of the server's event fields.
// Detailed is true only for the resource-scoped single-event GET.
type EventResource struct {
	Event
	Detailed            bool
	RequestStack        stacks.StackIdentity
	RequestResourceName string
	RequestEventID      string
	ResourceType        *string
	Body                map[string]json.RawMessage
	Header              http.Header
}

func decodeEvent(raw json.RawMessage, header http.Header, identity stacks.StackIdentity, resourceName, eventID string, detailed bool) (*EventResource, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("event must be a JSON object")
	}
	var native Event
	if err := json.Unmarshal(raw, &native); err != nil {
		return nil, err
	}
	if native.ID == "" {
		return nil, fmt.Errorf("event response has no id")
	}
	var extra struct {
		ResourceType *string `json:"resource_type"`
	}
	if err := json.Unmarshal(raw, &extra); err != nil {
		return nil, err
	}
	return &EventResource{
		Event: native, Detailed: detailed,
		RequestStack: identity, RequestResourceName: resourceName, RequestEventID: eventID,
		ResourceType: extra.ResourceType, Body: fields, Header: header.Clone(),
	}, nil
}
