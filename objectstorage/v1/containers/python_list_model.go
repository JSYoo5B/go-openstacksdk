package containers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

func listRecordProject(row json.RawMessage, page *rest.Response) (*ContainerRecord, error) {
	members, err := cloudfilter.ObjectMembers(row)
	if err != nil {
		return nil, err
	}
	body, headers := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	metadata := make(map[string]string)
	for _, member := range members {
		switch member.Key {
		case "cls", "connection", "microversion", "_synchronized":
			return nil, metadataInvalid("container list constructor collides with %q", member.Key)
		case "self":
			continue
		case "metadata":
			raw := bytes.TrimSpace(member.Value)
			if bytes.Equal(raw, []byte("null")) {
				continue
			}
			values, err := cloudfilter.ObjectMembers(raw)
			if err != nil {
				return nil, err
			}
			for _, pair := range values {
				text, err := cloudfilter.PythonString(pair.Value)
				if err != nil {
					return nil, err
				}
				key := pair.Key
				if !strings.HasPrefix(strings.ToLower(key), "x-container-meta-") {
					key = "X-Container-Meta-" + key
				}
				metadata[key] = text
			}
		default:
			if listRecordBodyKey(member.Key) {
				body[member.Key] = bytes.Clone(member.Value)
				continue
			}
			if decl, known := recordHeader(member.Key); known {
				headers[decl.field] = bytes.Clone(member.Value)
			}
		}
	}
	// Resource.list constructs from each row. It does not translate account HTTP
	// headers into container Header descriptors or observed custom metadata.
	value, err := recordProjection(&rest.Response{Body: bytes.Clone(row), Header: make(http.Header), StatusCode: page.StatusCode}, body, headers)
	if err != nil {
		return value, err
	}
	if _, present := body["name"]; !present {
		if raw, exists := body["id"]; exists {
			value.Resource.Body["name"] = bytes.Clone(raw)
		}
	}
	value.Metadata = metadata
	value.Envelope, value.Header, value.StatusCode = bytes.Clone(page.Body), page.Header.Clone(), page.StatusCode
	value.Resource.Header, value.Resource.StatusCode = page.Header.Clone(), page.StatusCode
	value.Wire.Header, value.Wire.StatusCode = page.Header.Clone(), page.StatusCode
	return value, nil
}
func listRecordMatch(actual, filter json.RawMessage) (bool, error) {
	raw := bytes.TrimSpace(filter)
	if len(raw) == 0 || raw[0] != '{' {
		return jsonfilter.EqualPythonJSON(actual, filter)
	}
	truthy, err := cloudfilter.PythonTruthy(actual)
	if err != nil || !truthy {
		return false, err
	}
	members, err := cloudfilter.ObjectMembers(filter)
	if err != nil {
		return false, err
	}
	if len(members) == 0 {
		return true, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(actual, &fields); err != nil {
		return false, metadataInvalid("truthy container local-filter value has no mapping get")
	}
	for _, member := range members {
		value, exists := fields[member.Key]
		if !exists {
			value = json.RawMessage("null")
		}
		matched, err := listRecordMatch(value, member.Value)
		if err != nil || !matched {
			return matched, err
		}
	}
	return true, nil
}
func listRecordMarker(raw json.RawMessage) ([]string, error) {
	return cloudfilter.RequestQueryValues(raw)
}
func listRecordRows(page *rest.Response) ([]json.RawMessage, error) {
	// Decode the complete actual response before lazily constructing rows. A
	// singleton object is accepted by Source; null/scalars fail as a consumed row.
	raw := bytes.TrimSpace(page.Body)
	if len(raw) == 0 || !json.Valid(raw) {
		return nil, metadataInvalid("container listing requires JSON")
	}
	if raw[0] == '[' {
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, err
		}
		return rows, nil
	}
	return []json.RawMessage{bytes.Clone(raw)}, nil
}
func listRecordReceipt(page *rest.Response) *ContainerRecord {
	if page == nil {
		return nil
	}
	return &ContainerRecord{Envelope: bytes.Clone(page.Body), Header: page.Header.Clone(), StatusCode: page.StatusCode}
}
