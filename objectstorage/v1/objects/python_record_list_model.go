package objects

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ObjectRecord is one row of the named Source objects(container) listing.
// Resource is the Source Object to_dict view (Body attributes, Header
// attributes with their aliases, the updated_at aka and location); Wire keeps
// every raw row member. Container is the URI value Source sets from the call
// argument. Envelope, Header and StatusCode are the actual container page
// receipt that carried the row.
type ObjectRecord struct {
	Resource, Wire *resource.RawResource
	Container      string
	Metadata       map[string]json.RawMessage
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

type objectListHeaderField struct{ field, wire, kind, alias string }

// objectListHeaderFields are the Object Header descriptors in class order.
// manifest and object_manifest share x-object-manifest; the later
// object_manifest owns the constructor attribute-name mapping.
var objectListHeaderFields = []objectListHeaderField{
	{"is_newest", "x-newest", "bool", ""}, {"range", "range", "dict", ""},
	{"if_match", "if-match", "list", ""}, {"if_none_match", "if-none-match", "list", ""},
	{"if_modified_since", "if-modified-since", "str", ""}, {"if_unmodified_since", "if-unmodified-since", "str", ""},
	{"signature", "signature", "", ""}, {"expires_at", "expires", "", ""},
	{"manifest", "x-object-manifest", "", ""}, {"multipart_manifest", "multipart-manifest", "", ""},
	{"content_length", "content-length", "int", "_bytes"}, {"content_type", "content-type", "", "_content_type"},
	{"accept_ranges", "accept-ranges", "", ""}, {"etag", "etag", "", "_hash"},
	{"is_static_large_object", "x-static-large-object", "bool", ""},
	{"content_encoding", "content-encoding", "", ""}, {"content_disposition", "content-disposition", "", ""},
	{"delete_after", "x-delete-after", "int", ""}, {"delete_at", "x-delete-at", "", ""},
	{"object_manifest", "x-object-manifest", "", ""}, {"timestamp", "x-timestamp", "", ""},
	{"last_modified_at", "last-modified", "", "_last_modified"},
	{"transfer_encoding", "transfer-encoding", "", ""},
	{"is_content_type_detected", "x-detect-content-type", "bool", ""},
	{"copy_from", "x-copy-from", "", ""}, {"symlink_target", "x-symlink-target", "", ""},
	{"symlink_target_account", "x-symlink-target-account", "", ""},
	{"access_control_allow_origin", "access-control-allow-origin", "", ""},
}

// objectListBodyFields maps Object Body attribute names to server names.
var objectListBodyFields = [][2]string{
	{"name", "name"}, {"_hash", "hash"}, {"_bytes", "bytes"}, {"_last_modified", "last_modified"},
	{"_content_type", "content_type"}, {"id", "id"},
}

func objectListBodyServer(key string) (string, bool) {
	for _, pair := range objectListBodyFields {
		if key == pair[0] || key == pair[1] {
			return pair[1], true
		}
	}
	return "", false
}
func objectListHeaderWire(key string) (string, bool) {
	for _, decl := range objectListHeaderFields {
		if strings.EqualFold(key, decl.wire) || key == decl.field && decl.field != "manifest" {
			return decl.wire, true
		}
	}
	return "", false
}

// objectListConvert applies the Source descriptor type to a present value.
func objectListConvert(raw json.RawMessage, kind string) (json.RawMessage, error) {
	trim := bytes.TrimSpace(raw)
	if bytes.Equal(trim, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	switch kind {
	case "int":
		return jsonfilter.DescriptorIntegerJSON(trim)
	case "bool":
		truthy, err := cloudfilter.PythonJSONTruthy(trim)
		if err != nil {
			return nil, err
		}
		return json.Marshal(truthy)
	case "list":
		if trim[0] == '[' {
			return bytes.Clone(trim), nil
		}
		return json.RawMessage("[" + string(trim) + "]"), nil
	case "dict":
		if trim[0] == '{' {
			return bytes.Clone(trim), nil
		}
		return json.RawMessage("{}"), nil
	case "str":
		if trim[0] == '"' {
			return bytes.Clone(trim), nil
		}
		text, err := cloudfilter.PythonString(trim)
		if err != nil {
			return nil, err
		}
		return json.Marshal(text)
	}
	return bytes.Clone(trim), nil
}

// objectListProject reproduces Object.existing(**row, container=container):
// constructor collisions fail, Body attributes are consumed before Header
// attributes, unknown members (subdir, symlink_path, ...) are dropped from the
// view and remain in Wire, and to_dict resolves Header aliases to Body values.
func objectListProject(container string, row json.RawMessage, page *rest.Response) (*ObjectRecord, error) {
	members, err := cloudfilter.ObjectMembers(row)
	if err != nil {
		return nil, err
	}
	wire := make(map[string]json.RawMessage, len(members))
	body, headers := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	metadata := make(map[string]json.RawMessage)
	location := json.RawMessage("null")
	for _, member := range members {
		wire[member.Key] = bytes.Clone(member.Value)
		switch member.Key {
		case "cls", "connection", "microversion", "_synchronized":
			return nil, metadataInvalid("object list constructor collides with %q", member.Key)
		case "self", "container", "data":
			// Resource.list pops self, the URI argument replaces container and
			// Object.__init__ keeps data outside the attribute view.
			continue
		case "location":
			// Resource._collect_attrs discards unknown attrs before computing
			// location when the container URI is present. SDK location is owned
			// and remains null; Wire preserves the remote field.
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
				key := pair.Key
				if !strings.HasPrefix(strings.ToLower(key), "x-object-meta-") {
					key = "X-Object-Meta-" + key
				}
				metadata[key] = bytes.Clone(pair.Value)
			}
			continue
		}
		if server, known := objectListBodyServer(member.Key); known {
			body[server] = bytes.Clone(member.Value)
			continue
		}
		if name, known := objectListHeaderWire(member.Key); known {
			headers[name] = bytes.Clone(member.Value)
		}
	}
	view := make(map[string]json.RawMessage)
	for _, pair := range objectListBodyFields {
		value := json.RawMessage("null")
		if raw, present := body[pair[1]]; present {
			value = bytes.Clone(raw)
			if pair[0] == "_bytes" {
				if value, err = objectListConvert(raw, "int"); err != nil {
					return nil, err
				}
			}
		}
		view[pair[0]] = value
	}
	if _, present := body["id"]; !present {
		view["id"] = bytes.Clone(view["name"])
	}
	for _, decl := range objectListHeaderFields {
		value := json.RawMessage("null")
		if raw, present := headers[decl.wire]; present {
			if value, err = objectListConvert(raw, decl.kind); err != nil {
				return nil, err
			}
		} else if decl.alias != "" {
			value = bytes.Clone(view[decl.alias])
		}
		view[decl.field] = value
	}
	view["updated_at"] = bytes.Clone(view["last_modified_at"])
	view["location"] = location
	record := &ObjectRecord{
		Resource:   &resource.RawResource{Metadata: resource.Metadata{Body: view, Header: page.Header.Clone(), StatusCode: page.StatusCode}},
		Wire:       &resource.RawResource{Metadata: resource.Metadata{Body: wire, Header: page.Header.Clone(), StatusCode: page.StatusCode}},
		Container:  container,
		Metadata:   metadata,
		Envelope:   bytes.Clone(page.Body),
		Header:     page.Header.Clone(),
		StatusCode: page.StatusCode,
	}
	return record, nil
}

// objectListMatch is Resource.list client filtering: plain values use Python
// equality and dictionaries use the recursive _dict_filter.
func objectListMatch(actual, filter json.RawMessage) (bool, error) {
	raw := bytes.TrimSpace(filter)
	if len(raw) == 0 || raw[0] != '{' {
		return jsonfilter.EqualPythonLoadedJSON(actual, filter)
	}
	truthy, err := cloudfilter.PythonJSONTruthy(actual)
	if err != nil || !truthy {
		return false, err
	}
	members, err := cloudfilter.ObjectMembers(filter)
	if err != nil {
		return false, err
	}
	// Source calls d.get only while iterating a nonempty filter. An empty
	// filter matches every truthy value, including a scalar or a list.
	if len(members) == 0 {
		return true, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(actual, &fields); err != nil {
		return false, metadataInvalid("truthy object local-filter value has no mapping get")
	}
	for _, member := range members {
		value, exists := fields[member.Key]
		if !exists {
			value = json.RawMessage("null")
		}
		matched, err := objectListMatch(value, member.Value)
		if err != nil || !matched {
			return matched, err
		}
	}
	return true, nil
}

// objectListRows decodes the whole page before lazy row construction. A
// singleton object is accepted like Source; null and scalars fail as rows.
func objectListRows(page *rest.Response) ([]json.RawMessage, error) {
	raw := bytes.TrimSpace(page.Body)
	if len(raw) == 0 || !json.Valid(raw) {
		return nil, metadataInvalid("object listing requires JSON")
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
func objectListReceipt(container string, page *rest.Response) *ObjectRecord {
	if page == nil {
		return nil
	}
	return &ObjectRecord{Container: container, Envelope: bytes.Clone(page.Body), Header: page.Header.Clone(), StatusCode: page.StatusCode}
}
