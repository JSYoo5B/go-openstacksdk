package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// mutationState contains known logical inputs, never an HTTP response record.
// Overlay preserves prior fields and resolves wire/attribute aliases in parsed
// dictionary order. Descriptor conversion happens only when a view is built.
type mutationState [len(descriptors)]json.RawMessage

func (s *mutationState) overlay(raw json.RawMessage) error {
	return overlayMutationFields(s[:], descriptors[:], raw)
}
func (s *mutationState) object() json.RawMessage {
	return mutationFieldsObject(s[:], descriptors[:])
}
func (s *mutationState) view(location resource.CloudLocation) (json.RawMessage, error) {
	view, _, err := normalize(s.object(), nil, false, location)
	return view, err
}
func (s *mutationState) id() json.RawMessage { return mutationField(s[12]) }

// Route validation is reached only when a physical member request is needed.
// No-wait and cached-ready results can expose an arbitrary nullable source ID.
func (s *mutationState) routeID() (string, error) {
	return mutationRouteID(s.id(), "snapshot")
}
func (s *mutationState) status(nullable bool) (string, error) {
	return mutationStatus(s[8], "snapshot", nullable)
}

// An empty or malformed UTF-8 JSON reply leaves the Resource unchanged. It
// does not manufacture an empty actual object. Valid wrong-shaped JSON fails.
func mutationObject(wire *rest.Response) (json.RawMessage, *resource.RawResource, error) {
	return mutationObjectFor(wire, "snapshot")
}

func mutationObjectFor(wire *rest.Response, singular string) (json.RawMessage, *resource.RawResource, error) {
	if !utf8.Valid(wire.Body) {
		return nil, nil, wire.Fail(fmt.Errorf("%s response must be UTF-8 JSON", singular))
	}
	if !json.Valid(wire.Body) {
		return nil, nil, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(wire.Body, &envelope); err != nil {
		return nil, nil, wire.Fail(err)
	}
	if envelope == nil {
		return nil, nil, wire.Fail(fmt.Errorf("%s response must be a nonnull JSON object", singular))
	}
	raw, present := envelope[singular]
	if !present {
		raw = wire.Body
	}
	var actual resource.RawResource
	if err := json.Unmarshal(raw, &actual); err != nil {
		return nil, nil, wire.Fail(err)
	}
	actual.Header, actual.StatusCode = wire.Header.Clone(), wire.StatusCode
	return bytes.Clone(raw), &actual, nil
}
