// Package jsonpatch constructs deterministic RFC 6902 patches for owned JSON
// resource bodies. It does not depend on a Python jsonpatch runtime version.
package jsonpatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

// Operation is one owned JSON Patch operation. Diff emits add, remove and
// replace; it does not infer move or copy operations.
type Operation struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
	From  string          `json:"from,omitempty"`
}

// Diff compares two complete UTF-8 JSON documents. Returned values are owned,
// and an unchanged document produces a non-nil empty slice (JSON []). JSON
// equality is type-aware and compares numbers exactly, without float64 rounding.
//
// Object members are visited in escaped JSON Pointer token order. Arrays retain
// a deterministic longest common subsequence, recursively edit unmatched pairs,
// and insert/remove remaining elements in executable index order. Operations
// must be applied as returned: globally sorting paths would change array meaning.
// The result need not be a shortest patch and is not byte-equivalent to an
// arbitrary Python jsonpatch version. Decoding follows encoding/json, including
// its last duplicate object member and escaped-surrogate behavior.
func Diff(original, current json.RawMessage) ([]Operation, error) {
	before, err := decode(original)
	if err != nil {
		return nil, fmt.Errorf("original JSON: %w", err)
	}
	after, err := decode(current)
	if err != nil {
		return nil, fmt.Errorf("current JSON: %w", err)
	}
	builder := diffBuilder{operations: make([]Operation, 0)}
	builder.compare("", before, after)
	return builder.operations, nil
}

func decode(raw json.RawMessage) (any, error) {
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("JSON document must be valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}

type diffBuilder struct {
	operations []Operation
}

func (builder *diffBuilder) compare(path string, before, after any) {
	if equal(before, after) {
		return
	}
	if oldObject, ok := before.(map[string]any); ok {
		if newObject, ok := after.(map[string]any); ok {
			builder.object(path, oldObject, newObject)
			return
		}
	}
	if oldArray, ok := before.([]any); ok {
		if newArray, ok := after.([]any); ok {
			builder.array(path, oldArray, newArray)
			return
		}
	}
	builder.value("replace", path, after)
}

func (builder *diffBuilder) object(path string, before, after map[string]any) {
	keys := make([]string, 0, len(before)+len(after))
	for key := range before {
		keys = append(keys, key)
	}
	for key := range after {
		if _, exists := before[key]; !exists {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return token(keys[i]) < token(keys[j]) })
	for _, key := range keys {
		oldValue, oldExists := before[key]
		newValue, newExists := after[key]
		child := path + "/" + token(key)
		switch {
		case !newExists:
			builder.remove(child)
		case !oldExists:
			builder.value("add", child, newValue)
		default:
			builder.compare(child, oldValue, newValue)
		}
	}
}

func (builder *diffBuilder) array(path string, before, after []any) {
	// Equal prefixes and suffixes avoid quadratic alignment for the common
	// small edit in an otherwise large array.
	prefix := 0
	for prefix < len(before) && prefix < len(after) && equal(before[prefix], after[prefix]) {
		prefix++
	}
	oldEnd, newEnd := len(before), len(after)
	for oldEnd > prefix && newEnd > prefix && equal(before[oldEnd-1], after[newEnd-1]) {
		oldEnd--
		newEnd--
	}
	oldMiddle, newMiddle := before[prefix:oldEnd], after[prefix:newEnd]
	anchors := make([]anchor, 0)
	findAnchors(oldMiddle, newMiddle, 0, 0, &anchors)
	oldCursor, newCursor, index := 0, 0, prefix
	for _, match := range anchors {
		builder.gap(path, index, oldMiddle[oldCursor:match.old], newMiddle[newCursor:match.current])
		index += match.current - newCursor + 1
		oldCursor, newCursor = match.old+1, match.current+1
	}
	builder.gap(path, index, oldMiddle[oldCursor:], newMiddle[newCursor:])
}

func (builder *diffBuilder) gap(path string, index int, before, after []any) {
	paired := min(len(before), len(after))
	for offset := 0; offset < paired; offset++ {
		builder.compare(path+"/"+strconv.Itoa(index+offset), before[offset], after[offset])
	}
	// Each removal shifts the next surplus element into the same position.
	for offset := paired; offset < len(before); offset++ {
		builder.remove(path + "/" + strconv.Itoa(index+paired))
	}
	for offset := paired; offset < len(after); offset++ {
		builder.value("add", path+"/"+strconv.Itoa(index+offset), after[offset])
	}
}

func (builder *diffBuilder) remove(path string) {
	builder.operations = append(builder.operations, Operation{Op: "remove", Path: path})
}

func (builder *diffBuilder) value(op, path string, value any) {
	// Values originate exclusively from the successful UseNumber JSON decode;
	// each is marshalable, and Marshal allocates independent operation bytes.
	raw, _ := json.Marshal(value)
	builder.operations = append(builder.operations, Operation{Op: op, Path: path, Value: raw})
}

func token(key string) string {
	return strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

func equal(left, right any) bool {
	switch value := left.(type) {
	case nil:
		return right == nil
	case bool:
		other, ok := right.(bool)
		return ok && value == other
	case string:
		other, ok := right.(string)
		return ok && value == other
	case json.Number:
		other, ok := right.(json.Number)
		if !ok {
			return false
		}
		result, _ := jsonfilter.EqualJSON(json.RawMessage(value.String()), json.RawMessage(other.String()))
		return result
	case []any:
		other, ok := right.([]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for index, item := range value {
			if !equal(item, other[index]) {
				return false
			}
		}
		return true
	case map[string]any:
		other, ok := right.(map[string]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for key, item := range value {
			otherItem, exists := other[key]
			if !exists || !equal(item, otherItem) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

type anchor struct {
	old, current int
}

// Hirschberg alignment keeps linear live row storage instead of a quadratic
// edit matrix. Ties choose the earliest destination split; singleton matches
// choose the first destination occurrence. Equal anchors remain in sequence.
func findAnchors(before, after []any, oldOffset, newOffset int, matches *[]anchor) {
	if len(before) == 0 || len(after) == 0 {
		return
	}
	if len(before) == 1 {
		for index, item := range after {
			if equal(before[0], item) {
				*matches = append(*matches, anchor{old: oldOffset, current: newOffset + index})
				return
			}
		}
		return
	}
	middle := len(before) / 2
	split := destinationSplit(before[:middle], before[middle:], after)
	findAnchors(before[:middle], after[:split], oldOffset, newOffset, matches)
	findAnchors(before[middle:], after[split:], oldOffset+middle, newOffset+split, matches)
}

func destinationSplit(left, right, after []any) int {
	forward, backward := lcsLengths(left, after, false), lcsLengths(right, after, true)
	split, best := 0, -1
	for index := 0; index <= len(after); index++ {
		length := forward[index] + backward[len(after)-index]
		if length > best {
			split, best = index, length
		}
	}
	return split
}

func lcsLengths(before, after []any, reverse bool) []int {
	row := make([]int, len(after)+1)
	for oldIndex := range before {
		diagonal := 0
		for newIndex := range after {
			above := row[newIndex+1]
			i, j := oldIndex, newIndex
			if reverse {
				i, j = len(before)-1-oldIndex, len(after)-1-newIndex
			}
			if equal(before[i], after[j]) {
				row[newIndex+1] = diagonal + 1
			} else {
				row[newIndex+1] = max(row[newIndex], above)
			}
			diagonal = above
		}
	}
	return row
}
