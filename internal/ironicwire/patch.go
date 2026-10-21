package ironicwire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PatchOp is one RFC 6902 operation. Value is raw JSON and is omitted from the
// wire form only when it is nil, so an explicit JSON null, false, zero or empty
// value is sent unchanged. From is sent only for move and copy operations.
type PatchOp struct {
	Op    string
	Path  string
	From  string
	Value json.RawMessage
}

// MarshalJSON writes op, path, from (when set) and value (when non-nil) in the
// key order used by the pinned Python jsonpatch library.
func (p PatchOp) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(`{`)
	first := true
	add := func(key string, raw []byte) {
		if !first {
			out.WriteByte(',')
		}
		first = false
		k, _ := json.Marshal(key)
		out.Write(k)
		out.WriteByte(':')
		out.Write(raw)
	}
	op, _ := json.Marshal(p.Op)
	add("op", op)
	path, _ := json.Marshal(p.Path)
	add("path", path)
	if p.From != "" {
		from, _ := json.Marshal(p.From)
		add("from", from)
	}
	if p.Value != nil {
		if !json.Valid(p.Value) {
			return nil, fmt.Errorf("%w: patch value for %q is not valid JSON", resource.ErrInvalidOption, p.Path)
		}
		add("value", p.Value)
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Names maps the attribute names a caller may use to the server field names.
// Both the attribute name and the server name resolve to the server name.
type Names struct {
	// Attributes lists attribute name to server name pairs.
	Attributes map[string]string
	// ID is the server name of the identity field (for example "uuid").
	ID string
}

// Server returns the server name for an attribute or server name and whether
// it is a declared body field.
func (n Names) Server(name string) (string, bool) {
	if wire, ok := n.Attributes[name]; ok {
		return wire, true
	}
	for _, wire := range n.Attributes {
		if wire == name {
			return wire, true
		}
	}
	return "", false
}

// Attribute is one value of an attribute-style update.
type Attribute struct {
	Name  string
	Value json.RawMessage
}

// ResolveAttributes maps attribute names to server names. Identity attributes,
// unknown names and the same field named twice are rejected, because the pinned
// Resource constructor silently drops unknown names and takes the identity from
// the reference.
func ResolveAttributes(names Names, attrs map[string]any) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(attrs))
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		wire, ok := names.Server(key)
		if !ok {
			return nil, invalid("attribute %q is not a declared body field", key)
		}
		if wire == names.ID {
			return nil, invalid("attribute %q is the resource identity; it comes from the reference", key)
		}
		if _, dup := out[wire]; dup {
			return nil, invalid("attribute %q repeats server field %q", key, wire)
		}
		raw, err := json.Marshal(attrs[key])
		if err != nil {
			return nil, errors.Join(invalid("attribute %q is not JSON", key), err)
		}
		out[wire] = raw
	}
	return out, nil
}

// AttributePatch derives the patch the pinned Python Resource.commit sends for
// a resource built from an identity and attributes: the original body holds only
// the identity, so every attribute becomes an "add" operation. Operations are
// ordered by path.
func AttributePatch(attrs map[string]json.RawMessage) []PatchOp {
	ops := make([]PatchOp, 0, len(attrs))
	for wire, raw := range attrs {
		ops = append(ops, PatchOp{Op: "add", Path: "/" + wire, Value: slices.Clone(raw)})
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].Path < ops[j].Path })
	return ops
}

// ConvertPatch rewrites the first path segment of every operation from an
// attribute name to its server name, like Resource._convert_patch. A path that
// is empty or "/" fails. Unknown first segments are kept unchanged.
func ConvertPatch(names Names, patch []PatchOp) ([]PatchOp, error) {
	out := make([]PatchOp, len(patch))
	for i, item := range patch {
		if item.Op == "" {
			return nil, invalid("patch operation %d has no op", i)
		}
		trimmed := strings.TrimLeft(item.Path, "/")
		if item.Path == "" || trimmed == "" {
			return nil, invalid("patch operation %d has a malformed or missing path", i)
		}
		parts := strings.SplitN(trimmed, "/", 2)
		field := parts[0]
		if wire, ok := names.Attributes[field]; ok {
			field = wire
		}
		item.Path = "/" + field
		if len(parts) > 1 {
			item.Path += "/" + parts[1]
		}
		item.Value = slices.Clone(item.Value)
		out[i] = item
	}
	return out, nil
}

// Truthy applies the CPython truth value to a JSON document.
func Truthy(raw json.RawMessage) bool {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}
