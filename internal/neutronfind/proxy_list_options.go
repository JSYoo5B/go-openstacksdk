package neutronfind

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

func removeProxyListKey(value *Options, key string) {
	delete(value.Query, key)
	value.Order = slices.DeleteFunc(value.Order, func(existing string) bool { return existing == key })
}
func proxyListNullableString(raw json.RawMessage, key string) (*string, error) {
	if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
		return nil, errors.Join(findInvalid("%s must contain complete UTF-8 JSON", key), err)
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil, errors.Join(findInvalid("%s must be a JSON string or null", key), err)
	}
	return &text, nil
}
func proxyListArgumentCollision(value Options) error {
	for _, key := range []string{"cls", "session", "self", "resource_type"} {
		if _, present := value.Query[key]; present {
			return findInvalid("%q collides with a bound Resource/Proxy list argument", key)
		}
	}
	return nil
}

// NormalizeProxyListOptions consumes the pinned Proxy._list/Resource.list
// formal arguments without changing resource-specific Body/Query declarations.
// It snapshots every input. The second result is the original Proxy JMESPath
// argument; a restored jmespath_filters kwarg is an unknown Resource.list
// parameter and cannot replace it. Unknown JSON kwargs are left for the exact
// resource classifier, except Source controls that the older Engine reserves.
func NormalizeProxyListOptions(value Options) (Options, json.RawMessage, error) {
	value = CloneOptions(value)
	if err := proxyListArgumentCollision(value); err != nil {
		return value, nil, err
	}
	var expression json.RawMessage
	if raw, present := value.Query["jmespath_filters"]; present {
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return value, nil, errors.Join(findInvalid("jmespath_filters must contain complete UTF-8 JSON"), err)
		}
		expression = bytes.Clone(raw)
		removeProxyListKey(&value, "jmespath_filters")
	}
	// Proxy passes base_path explicitly to Resource.list, even when omitted.
	// Its original value is bound before conflicting attrs are restored.
	if raw, present := value.Query["base_path"]; present {
		path, err := proxyListNullableString(raw, "base_path")
		if err != nil {
			return value, expression, err
		}
		value.BasePath = path
		removeProxyListKey(&value, "base_path")
	}
	if raw, present := value.Query["__conflicting_attrs"]; present {
		removeProxyListKey(&value, "__conflicting_attrs")
		truthy, err := cloudfilter.PythonJSONTruthy(raw)
		if err != nil {
			return value, expression, errors.Join(findInvalid("__conflicting_attrs must contain complete UTF-8 JSON"), err)
		}
		if truthy {
			members, err := cloudfilter.ObjectMembers(raw)
			if err != nil {
				return value, expression, errors.Join(findInvalid("truthy __conflicting_attrs must be an object"), err)
			}
			for _, member := range members {
				switch member.Key {
				case "paginated", "base_path":
					return value, expression, findInvalid("restored %q duplicates the explicit Resource.list argument", member.Key)
				}
				if !slices.Contains(value.Order, member.Key) {
					value.Order = append(value.Order, member.Key)
				}
				value.Query[member.Key] = bytes.Clone(member.Value)
			}
			// Source pops this key after its one restoration pass, including an
			// injected key with the same spelling. It does not restore recursively.
			removeProxyListKey(&value, "__conflicting_attrs")
		}
	}
	if err := proxyListArgumentCollision(value); err != nil {
		return value, expression, err
	}
	// JMESPath restored as an attr belongs to Resource.list and is unknown.
	removeProxyListKey(&value, "jmespath_filters")
	if raw, present := value.Query["microversion"]; present {
		version, err := proxyListNullableString(raw, "microversion")
		if err != nil {
			return value, expression, err
		}
		value.Microversion = version
		removeProxyListKey(&value, "microversion")
	}
	if raw, present := value.Query["headers"]; present {
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return value, expression, errors.Join(findInvalid("headers must contain complete UTF-8 JSON"), err)
		}
		headers := make(map[string]string)
		truthy, _ := cloudfilter.PythonJSONTruthy(raw)
		if truthy {
			members, err := cloudfilter.ObjectMembers(raw)
			if err != nil {
				return value, expression, errors.Join(findInvalid("headers must be a string-valued JSON object or falsey value"), err)
			}
			for _, member := range members {
				var text string
				if trim := bytes.TrimSpace(member.Value); len(trim) == 0 || trim[0] != '"' {
					return value, expression, findInvalid("header %q must be a JSON string", member.Key)
				}
				if err := json.Unmarshal(member.Value, &text); err != nil {
					return value, expression, errors.Join(findInvalid("header %q must be a JSON string", member.Key), err)
				}
				headers[member.Key] = text
			}
		}
		value.Headers = headers
		removeProxyListKey(&value, "headers")
	}
	// Resource.list's allow_unknown_params formal argument is deliberately
	// unused: _validate always receives True. Other names are ordinary attrs,
	// not Body descriptors, and Source discards them without value evaluation.
	for _, key := range []string{"allow_unknown_params", "list_base_path", "connection", "_synchronized", "requires_id", "error_message", "skip_cache", "resource_response_key", "location"} {
		removeProxyListKey(&value, key)
	}
	return value, expression, nil
}
