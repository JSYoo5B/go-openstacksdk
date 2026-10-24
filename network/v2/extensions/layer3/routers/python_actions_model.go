package routers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RouterActionRecord separates the pinned Source descriptor view from actual
// HTTP JSON. Resource has canonical Body names/defaults, inherited if_match,
// unknown extensions and a null location. Wire contains only returned fields;
// Envelope, Header and StatusCode preserve the complete HTTP observation.
// A failed physical read/Close may return observation evidence without Resource.
// An accepted action is an acknowledgement, not proof of backend convergence.
type RouterActionRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	source         neutronfind.SourceState
}

func routerActionInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

func routerActionRow(raw json.RawMessage) (json.RawMessage, error) {
	if !utf8.Valid(raw) {
		return nil, routerActionInvalid("response must be UTF-8 JSON")
	}
	if _, err := cloudfilter.ObjectMembers(raw); err != nil {
		return nil, routerActionInvalid("response must be a JSON object: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if nested, ok := fields["router"]; ok {
		raw = nested
	}
	if _, err := cloudfilter.ObjectMembers(raw); err != nil {
		return nil, routerActionInvalid("router response must be a JSON object: %v", err)
	}
	return bytes.Clone(raw), nil
}

func routerActionObservation(response *rest.Response) *RouterActionRecord {
	if response == nil {
		return nil
	}
	value := &RouterActionRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if raw, err := routerActionRow(response.Body); err == nil {
		var wire resource.RawResource
		if json.Unmarshal(raw, &wire) == nil {
			wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
			value.Wire = wire.Clone()
		}
	}
	return value
}

func routerActionCanonical(key string) (string, bool) {
	if _, ok := neutronBodyFieldNames[key]; ok {
		return key, true
	}
	for canonical, wire := range neutronBodyFieldNames {
		if wire == key {
			return canonical, true
		}
	}
	return key, false
}

func routerActionDescriptor(key string, raw json.RawMessage) (json.RawMessage, error) {
	raw = bytes.Clone(raw)
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return raw, nil
	}
	switch key {
	case "evpn_vni", "revision_number":
		return jsonfilter.DescriptorIntegerJSON(raw)
	case "is_admin_state_up", "is_distributed", "is_ha", "enable_ndp_proxy":
		// Ordinary Python json.loads decodes decimal/exponent tokens through
		// binary64. Underflow is false and overflow is true for bool fields;
		// integer tokens retain arbitrary-precision zero/nonzero truthiness.
		token := bytes.TrimSpace(raw)
		if len(token) > 0 && (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
			floating, err := strconv.ParseFloat(string(token), 64)
			if err != nil && !errors.Is(err, strconv.ErrRange) {
				return nil, err
			}
			return json.Marshal(floating != 0)
		}
		truth, err := cloudfilter.PythonTruthy(raw)
		if err != nil {
			return nil, err
		}
		return json.Marshal(truth)
	case "availability_zone_hints", "availability_zones", "routes", "tags":
		if len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '[' {
			return raw, nil
		}
		return json.RawMessage(append(append([]byte{'['}, raw...), ']')), nil
	case "external_gateway_info":
		if len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '{' {
			return raw, nil
		}
		return json.RawMessage("{}"), nil
	}
	return raw, nil
}

func routerActionSeed(id string, seed *resource.RawResource) (map[string]json.RawMessage, json.RawMessage, error) {
	fields := make(map[string]json.RawMessage)
	var match json.RawMessage
	if seed != nil {
		for key, raw := range seed.Body {
			if !utf8.ValidString(key) || !utf8.Valid(raw) || !json.Valid(raw) {
				return nil, nil, routerActionInvalid("seed %q must contain UTF-8 JSON", key)
			}
			switch key {
			case "location":
				continue
			case "base_path", "connection", "microversion", "_synchronized", "resource_type", "__conflicting_attrs", "self":
				return nil, nil, routerActionInvalid("seed %q is a Resource control", key)
			case "if_match", "if-match":
				if match != nil {
					return nil, nil, routerActionInvalid("seed has conflicting if_match aliases")
				}
				match = bytes.Clone(raw)
				if !bytes.Equal(bytes.TrimSpace(match), []byte("null")) && bytes.TrimSpace(match)[0] != '[' {
					match = append(append(json.RawMessage{'['}, match...), ']')
				}
				continue
			}
			canonical, _ := routerActionCanonical(key)
			if _, exists := fields[canonical]; exists {
				return nil, nil, routerActionInvalid("seed has conflicting aliases for %q", canonical)
			}
			if _, declared := neutronBodyFieldNames[canonical]; declared {
				converted, err := routerActionDescriptor(canonical, raw)
				if err != nil {
					return nil, nil, routerActionInvalid("seed %s descriptor: %v", canonical, err)
				}
				raw = converted
			}
			fields[canonical] = bytes.Clone(raw)
		}
		if match == nil {
			match = routerActionHeader(seed.Header)
		}
	}
	fields["id"], _ = json.Marshal(id)
	return fields, match, nil
}

func routerActionHeader(header http.Header) json.RawMessage {
	// Requests exposes a case-insensitive single value. Multiple HTTP values
	// are coalesced into that string before the inherited list descriptor.
	for name, values := range header {
		if (strings.EqualFold(name, "if-match") || strings.EqualFold(name, "if_match")) && len(values) > 0 {
			raw, _ := json.Marshal([]string{strings.Join(values, ", ")})
			return raw
		}
	}
	return nil
}

func routerActionFromResponse(response *rest.Response, seed map[string]json.RawMessage, seededMatch json.RawMessage) (*RouterActionRecord, error) {
	value := routerActionObservation(response)
	if value == nil {
		return nil, routerActionInvalid("action response is required")
	}
	fields := make(map[string]json.RawMessage, len(seed))
	for key, raw := range seed {
		fields[key] = bytes.Clone(raw)
	}
	// _translate_response tolerates malformed/empty JSON, but a successfully
	// parsed nonobject or null router envelope is a Resource translation error.
	if json.Valid(response.Body) {
		raw, err := routerActionRow(response.Body)
		if err != nil {
			return value, err
		}
		members, err := cloudfilter.ObjectMembers(raw)
		if err != nil {
			return value, err
		}
		for _, member := range members {
			if member.Key == "self" {
				continue
			}
			canonical, _ := routerActionCanonical(member.Key)
			fields[canonical] = bytes.Clone(member.Value)
		}
	}
	if _, present := fields["project_id"]; !present {
		if tenant, present := fields["tenant_id"]; present {
			fields["project_id"] = bytes.Clone(tenant)
		}
	}
	for canonical := range neutronBodyFieldNames {
		raw, present := fields[canonical]
		if !present {
			raw = json.RawMessage("null")
			if canonical == "tags" {
				raw = json.RawMessage("[]")
			}
		}
		projected, err := routerActionDescriptor(canonical, raw)
		if err != nil {
			return value, routerActionInvalid("%s descriptor: %v", canonical, err)
		}
		fields[canonical] = projected
	}
	// Header descriptors beat colliding unknown Body attrs in Resource.to_dict.
	// A Requests header value is a single string, converted by type=list to a
	// one-element list; a comma in that string is not split into separate tags.
	fields["if_match"] = bytes.Clone(seededMatch)
	if fields["if_match"] == nil {
		fields["if_match"] = json.RawMessage("null")
	}
	if match := routerActionHeader(value.Header); match != nil {
		fields["if_match"] = match
	}
	fields["location"] = json.RawMessage("null")
	value.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: fields, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	return value, nil
}
