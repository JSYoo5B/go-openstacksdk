package jsonpatch_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonpatch"
)

func TestDiffAppliesNestedAndArrayEdits(t *testing.T) {
	for _, test := range []struct{ name, before, after string }{
		{"object membership", `{"old":null,"keep":1}`, `{"new":null,"keep":1}`},
		{"nested object", `{"meta":{"count":2,"old":[1]},"keep":true}`, `{"meta":{"count":3,"new":{"x":1}},"keep":true}`},
		{"pointer escaped keys", `{"a/b":{"~x":1,"":3},"~1":0}`, `{"a/b":{"~x":2,"":null},"~1":1}`},
		{"unusual keys", `{" ":1,"한글":2,"\u0000":3}`, `{" ":null,"한글":[2],"\u0000":4,"":true}`},
		{"array insertion front", `["a","b"]`, `[null,"a","b"]`},
		{"array insertion middle", `["a","c"]`, `["a","b","c"]`},
		{"array insertion tail", `[1,2]`, `[1,2,3,4]`},
		{"array removal front", `[0,1,2]`, `[1,2]`},
		{"array removal middle", `[0,1,2,3]`, `[0,3]`},
		{"array removal tail", `[1,2,3,4]`, `[1,2]`},
		{"array clear", `[1,null,{"x":2}]`, `[]`},
		{"array populate", `[]`, `[1,null,{"x":2}]`},
		{"array reorder", `["first","second","third"]`, `["third","first","second"]`},
		{"duplicate array anchors", `[1,2,1,2,1]`, `[2,1,2,2,1]`},
		{"nested array object", `[{"id":1,"name":"before"},{"id":2}]`, `[{"id":1,"name":"after"},{"id":2}]`},
		{"nested arrays", `[[1,3],["a","b"],[]]`, `[[1,2,3],["b"],[true]]`},
		{"mixed structural changes", `{"a":[1,{"b":2},3,4],"z":[true]}`, `{"a":[0,1,{"b":3},4,5],"z":[1]}`},
		{"root scalar", `"before"`, `false`},
		{"root null to object", `null`, `{"key":null}`},
		{"root object to array", `{"key":1}`, `[1]`},
		{"member type change", `{"x":{"a":1},"y":1}`, `{"x":[{"a":1}],"y":"1"}`},
		{"empty shapes", `{"x":[],"y":{}}`, `{"x":{},"y":[]}`},
	} {
		t.Run(test.name, func(t *testing.T) { assertDiffApplies(t, test.before, test.after) })
	}
}

func TestDiffHasDeterministicConcreteOperations(t *testing.T) {
	for _, test := range []struct{ name, before, after, patch string }{
		{"empty slice", `{"x":1}`, `{"x":1.00}`, `[]`},
		{"membership", `{"old":null}`, `{"new":null}`, `[{"op":"add","path":"/new","value":null},{"op":"remove","path":"/old"}]`},
		{"pointer escapes", `{"a/b":{"~":1,"":2}}`, `{"a/b":{"~":3,"":null}}`, `[{"op":"replace","path":"/a~1b/","value":null},{"op":"replace","path":"/a~1b/~0","value":3}]`},
		{"middle insertion", `[1,3]`, `[1,2,3]`, `[{"op":"add","path":"/1","value":2}]`},
		{"middle removal", `[1,2,3]`, `[1,3]`, `[{"op":"remove","path":"/1"}]`},
		{"nested object edit", `[{"x":1}]`, `[{"x":2}]`, `[{"op":"replace","path":"/0/x","value":2}]`},
		{"root replacement", `true`, `1`, `[{"op":"replace","path":"","value":1}]`},
		{"array boolean number distinction", `[true,false]`, `[1,0]`, `[{"op":"replace","path":"/0","value":1},{"op":"replace","path":"/1","value":0}]`},
		{"no move optimization", `[1,2]`, `[2,1]`, `[{"op":"remove","path":"/0"},{"op":"add","path":"/1","value":1}]`},
		{"escaped object ordering", `{"/":1,"~":1,"0":1}`, `{"/":2,"~":2,"0":2}`, `[{"op":"replace","path":"/0","value":2},{"op":"replace","path":"/~0","value":2},{"op":"replace","path":"/~1","value":2}]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			operations := assertDiffApplies(t, test.before, test.after)
			actual, err := json.Marshal(operations)
			if err != nil {
				t.Fatal(err)
			}
			equal, err := jsonfilter.EqualJSON(actual, json.RawMessage(test.patch))
			if err != nil || !equal {
				t.Fatalf("patch=%s want=%s error=%v", actual, test.patch, err)
			}
			if operations == nil {
				t.Fatal("successful Diff returned nil patch")
			}
		})
	}
}

func TestDiffKeepsArrayIndicesExecutableAcrossTenAndRepeatedRemoval(t *testing.T) {
	before := `[0,1,2,3,4,5,6,7,8,9,10,11,12]`
	after := `[0,1,10,11,12,20,21,22,23,24,25,26,27]`
	operations := assertDiffApplies(t, before, after)
	removals := 0
	for _, operation := range operations {
		if operation.Op == "remove" {
			removals++
			if operation.Path != "/2" {
				t.Fatalf("surplus removal should retain shifting index: %+v", operation)
			}
		}
	}
	if removals != 8 {
		t.Fatalf("retained array anchors not used, removals=%d patch=%+v", removals, operations)
	}
	// In particular /10 must come after /9 when appending to an array. A
	// lexical path sort would attempt the /10 add before enough elements exist.
	for index, operation := range operations {
		if operation.Op == "add" && operation.Path == "/10" {
			if index == 0 || operations[index-1].Path != "/9" {
				t.Fatalf("array append sequence changed: %+v", operations)
			}
			return
		}
	}
	t.Fatal("expected array append at /10")
}

func TestDiffUsesExactJSONNumberEquality(t *testing.T) {
	for _, test := range []struct {
		name, before, after string
		changed             bool
	}{
		{"decimal equivalent", `1`, `1.000e0`, false},
		{"negative zero", `-0.00e99999999999999999999999`, `0`, false},
		{"large equivalent", `900719925474099300`, `9007199254740993e2`, false},
		{"large different", `9007199254740993`, `9007199254740992`, true},
		{"tiny equivalent", `1e-999999999999999999999`, `0.1e-999999999999999999998`, false},
		{"large exponent equivalent", `1e999999999999999999999`, `10e999999999999999999998`, false},
		{"fraction precision", `1.000000000000000000000000001`, `1`, true},
		{"boolean distinct", `true`, `1`, true},
		{"nested boolean distinct", `{"a":[false]}`, `{"a":[0]}`, true},
		{"string distinct", `"1"`, `1`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			operations := assertDiffApplies(t, test.before, test.after)
			if (len(operations) != 0) != test.changed {
				t.Fatalf("unexpected numeric patch %+v", operations)
			}
		})
	}
}

func TestDiffOwnsInputsAndOperationValues(t *testing.T) {
	before, after := json.RawMessage(`{"old":[1],"keep":0}`), json.RawMessage(`{"new":[2],"second":[2],"keep":0}`)
	beforeCopy, afterCopy := bytes.Clone(before), bytes.Clone(after)
	operations, err := jsonpatch.Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(operations)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, beforeCopy) || !bytes.Equal(after, afterCopy) {
		t.Fatal("Diff mutated input JSON")
	}
	for index := range before {
		before[index] = ' '
	}
	for index := range after {
		after[index] = ' '
	}
	owned, err := json.Marshal(operations)
	if err != nil || !bytes.Equal(encoded, owned) {
		t.Fatalf("operation retained caller bytes: %s error=%v", owned, err)
	}
	var first, second *jsonpatch.Operation
	for index := range operations {
		if operations[index].Path == "/new" {
			first = &operations[index]
		}
		if operations[index].Path == "/second" {
			second = &operations[index]
		}
	}
	if first == nil || second == nil {
		t.Fatal("missing expected owned values")
	}
	secondCopy := bytes.Clone(second.Value)
	first.Value[1] = '9'
	if !bytes.Equal(second.Value, secondCopy) {
		t.Fatal("operations share mutable value bytes")
	}
	again, err := jsonpatch.Diff(beforeCopy, afterCopy)
	if err != nil {
		t.Fatal(err)
	}
	againEncoded, err := json.Marshal(again)
	if err != nil || !bytes.Equal(encoded, againEncoded) {
		t.Fatalf("repeat patch drift: %s %v", againEncoded, err)
	}
}

func TestDiffRejectsIncompleteOrInvalidUTF8JSON(t *testing.T) {
	invalid := []json.RawMessage{
		nil, {}, json.RawMessage(" \n\t"), json.RawMessage(`true false`),
		json.RawMessage(`{"x":true,}`), json.RawMessage(`[1,]`),
		json.RawMessage(`01`), json.RawMessage(`1e`), json.RawMessage(`{"x":`),
		json.RawMessage([]byte{'"', 0xff, '"'}),
	}
	for index, value := range invalid {
		for _, side := range []string{"original", "current"} {
			t.Run(side+"/"+strconv.Itoa(index), func(t *testing.T) {
				before, after := value, json.RawMessage(`null`)
				if side == "current" {
					before, after = after, before
				}
				operations, err := jsonpatch.Diff(before, after)
				if err == nil || operations != nil || !strings.Contains(err.Error(), side+" JSON") {
					t.Fatalf("invalid document returned patch=%+v error=%v", operations, err)
				}
			})
		}
	}
	_, err := jsonpatch.Diff(json.RawMessage(`{"x":true,}`), json.RawMessage(`null`))
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("lost decoder cause: %v", err)
	}
}

func TestDiffCompleteDocumentMatrixAppliesDeterministically(t *testing.T) {
	documents := []string{
		`null`, `true`, `false`, `0`, `1`, `1.0`, `9007199254740993`, `""`, `"a/b~c"`,
		`[]`, `{}`, `[1]`, `[null]`, `[true,1]`, `[1,true]`, `[1,1,2]`, `[2,1,1]`,
		`{"a":null}`, `{"a":1}`, `{"a":[]}`, `{"a":[1,2]}`, `{"a":[2,1]}`,
		`[{"a":1},{"a":2}]`, `[{"a":2},{"a":1}]`, `[[1,2],[3]]`, `[[2],[3,4]]`,
		`{"":1,"/":2,"~":3}`, `{"":null,"/":3,"~":4}`, `{"a":{"b":[1,true]}}`,
		`{"a":{"b":[0,false]},"z":[]}`,
	}
	for oldIndex, before := range documents {
		for newIndex, after := range documents {
			operations := assertDiffApplies(t, before, after)
			encoded, err := json.Marshal(operations)
			if err != nil {
				t.Fatal(err)
			}
			repeated, err := jsonpatch.Diff(json.RawMessage(before), json.RawMessage(after))
			if err != nil {
				t.Fatal(err)
			}
			encodedAgain, err := json.Marshal(repeated)
			if err != nil || !bytes.Equal(encoded, encodedAgain) {
				t.Fatalf("pair %d/%d nondeterministic: %s / %s error=%v", oldIndex, newIndex, encoded, encodedAgain, err)
			}
		}
	}
}

func FuzzDiffAppliesToTarget(f *testing.F) {
	for _, pair := range [][2]string{
		{`{}`, `{"a":[1,2]}`}, {`[1,2,3]`, `[0,1,3,4]`},
		{`{"a/b":{"~":1}}`, `{"a/b":{"~":2,"":true}}`},
		{`[1,true,null]`, `[true,1,false]`}, {`9007199254740993`, `9007199254740992`},
		{`{"a":{"b":[1,2]}}`, `{"a":{"b":[2,3]}}`},
	} {
		f.Add(pair[0], pair[1])
	}
	f.Fuzz(func(t *testing.T, before, after string) {
		if len(before) > 16384 || len(after) > 16384 {
			t.Skip()
		}
		operations, err := jsonpatch.Diff(json.RawMessage(before), json.RawMessage(after))
		if err != nil {
			return
		}
		assertPatchApplies(t, before, after, operations)
	})
}

func assertDiffApplies(t testing.TB, before, after string) []jsonpatch.Operation {
	t.Helper()
	operations, err := jsonpatch.Diff(json.RawMessage(before), json.RawMessage(after))
	if err != nil {
		t.Fatalf("Diff(%s, %s): %v", before, after, err)
	}
	assertPatchApplies(t, before, after, operations)
	return operations
}

func assertPatchApplies(t testing.TB, before, after string, operations []jsonpatch.Operation) {
	t.Helper()
	actual, err := applyPatch(before, operations)
	if err != nil {
		t.Fatalf("patch not executable: before=%s after=%s patch=%+v error=%v", before, after, operations, err)
	}
	raw, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	equal, err := jsonfilter.EqualJSON(raw, json.RawMessage(after))
	if err != nil || !equal {
		t.Fatalf("patch did not reach target: before=%s want=%s actual=%s patch=%+v error=%v", before, after, raw, operations, err)
	}
}

// This test-only interpreter implements RFC 6902 add/remove/replace directly.
// It shares neither the generator's traversal/alignment nor its pointer encoder.
func applyPatch(before string, operations []jsonpatch.Operation) (any, error) {
	document, err := decodeDocument(before)
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if operation.From != "" {
			return nil, fmt.Errorf("unexpected from member")
		}
		var value any
		if operation.Op == "add" || operation.Op == "replace" {
			value, err = decodeDocument(string(operation.Value))
			if err != nil {
				return nil, err
			}
		} else if operation.Op != "remove" || operation.Value != nil {
			return nil, fmt.Errorf("invalid operation %+v", operation)
		}
		var parts []string
		if operation.Path != "" {
			if operation.Path[0] != '/' {
				return nil, fmt.Errorf("invalid pointer")
			}
			parts = strings.Split(operation.Path[1:], "/")
			for index, part := range parts {
				parts[index] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
			}
		}
		document, err = applyAt(document, parts, operation.Op, value)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", operation.Op, operation.Path, err)
		}
	}
	return document, nil
}

func decodeDocument(raw string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON")
	}
	return value, nil
}

func applyAt(document any, parts []string, op string, value any) (any, error) {
	if len(parts) == 0 {
		if op == "remove" {
			return nil, fmt.Errorf("root removal not emitted")
		}
		return value, nil
	}
	last := len(parts) == 1
	switch container := document.(type) {
	case map[string]any:
		old, exists := container[parts[0]]
		if !last {
			if !exists {
				return nil, fmt.Errorf("missing object parent")
			}
			updated, err := applyAt(old, parts[1:], op, value)
			if err != nil {
				return nil, err
			}
			container[parts[0]] = updated
		} else {
			if op != "add" && !exists {
				return nil, fmt.Errorf("missing object target")
			}
			if op == "remove" {
				delete(container, parts[0])
			} else {
				container[parts[0]] = value
			}
		}
		return container, nil
	case []any:
		index, err := strconv.Atoi(parts[0])
		if last && op == "add" && parts[0] == "-" {
			index, err = len(container), nil
		}
		if err != nil || index < 0 || index > len(container) || index == len(container) && !(last && op == "add") {
			return nil, fmt.Errorf("array index outside target")
		}
		if !last {
			updated, err := applyAt(container[index], parts[1:], op, value)
			if err != nil {
				return nil, err
			}
			container[index] = updated
		} else {
			switch op {
			case "add":
				container = append(container, nil)
				copy(container[index+1:], container[index:])
				container[index] = value
			case "remove":
				container = append(container[:index], container[index+1:]...)
			case "replace":
				container[index] = value
			}
		}
		return container, nil
	default:
		return nil, fmt.Errorf("pointer parent is not a container")
	}
}
