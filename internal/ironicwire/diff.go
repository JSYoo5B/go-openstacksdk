package ironicwire

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func pointerEscape(s string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(s)
}

func decode(raw json.RawMessage) (any, bool) {
	var v any
	if raw == nil {
		return nil, false
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false
	}
	return v, true
}

// Diff builds the JSON Patch that turns src into dst, following the pinned
// jsonpatch from_diff rules for objects (removed, added and recursively
// changed keys) and arrays (index-wise comparison, additions at the end and
// repeated removals at the shortened length). Unlike the pinned library it
// never emits move operations, so a patch is semantically equal but may have
// more operations. The result is sorted by path.
func Diff(src, dst map[string]json.RawMessage) []PatchOp {
	var ops []PatchOp
	diffObjects("", toAny(src), toAny(dst), &ops)
	sort.SliceStable(ops, func(i, j int) bool { return ops[i].Path < ops[j].Path })
	return ops
}

func toAny(m map[string]json.RawMessage) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		x, _ := decode(v)
		out[k] = x
	}
	return out
}

func raw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func diffObjects(path string, src, dst map[string]any, ops *[]PatchOp) {
	keys := map[string]bool{}
	for k := range src {
		keys[k] = true
	}
	for k := range dst {
		keys[k] = true
	}
	ordered := make([]string, 0, len(keys))
	for k := range keys {
		ordered = append(ordered, k)
	}
	sort.Strings(ordered)
	for _, k := range ordered {
		p := path + "/" + pointerEscape(k)
		old, inSrc := src[k]
		next, inDst := dst[k]
		switch {
		case inSrc && !inDst:
			*ops = append(*ops, PatchOp{Op: "remove", Path: p})
		case !inSrc && inDst:
			*ops = append(*ops, PatchOp{Op: "add", Path: p, Value: raw(next)})
		default:
			diffValues(p, old, next, ops)
		}
	}
}

func diffValues(p string, old, next any, ops *[]PatchOp) {
	if reflect.DeepEqual(old, next) {
		return
	}
	switch o := old.(type) {
	case map[string]any:
		if n, ok := next.(map[string]any); ok {
			diffObjects(p, o, n, ops)
			return
		}
	case []any:
		if n, ok := next.([]any); ok {
			diffArrays(p, o, n, ops)
			return
		}
	}
	*ops = append(*ops, PatchOp{Op: "replace", Path: p, Value: raw(next)})
}

func diffArrays(path string, src, dst []any, ops *[]PatchOp) {
	max := len(src)
	if len(dst) > max {
		max = len(dst)
	}
	for i := 0; i < max; i++ {
		switch {
		case i < len(src) && i < len(dst):
			diffValues(path+"/"+strconv.Itoa(i), src[i], dst[i], ops)
		case len(src) > len(dst):
			*ops = append(*ops, PatchOp{Op: "remove", Path: path + "/" + strconv.Itoa(len(dst))})
		default:
			*ops = append(*ops, PatchOp{Op: "add", Path: path + "/" + strconv.Itoa(i), Value: raw(dst[i])})
		}
	}
}

// ChangedPaths lists the paths a patch touches, in patch order.
func ChangedPaths(ops []PatchOp) []string {
	out := make([]string, len(ops))
	for i, o := range ops {
		out[i] = o.Path
	}
	return out
}
