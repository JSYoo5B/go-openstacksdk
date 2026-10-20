package jmespath_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
)

type opaqueGenerator struct{ marker int }

func (g *opaqueGenerator) JMESPathIdentity() any { return g }
func (*opaqueGenerator) MarshalJSON() ([]byte, error) {
	return nil, errors.New("Python generator is not JSON serializable")
}

type invalidOpaqueMarker struct{ key any }

func (g invalidOpaqueMarker) JMESPathIdentity() any { return g.key }

func TestOpaqueGeneratorExpressionsMatchPythonUnconsumedGenerator(t *testing.T) {
	generator := &opaqueGenerator{marker: 1}
	for _, test := range []struct {
		expression string
		want       any
	}{
		{"[*].id", nil}, {"[0]", nil}, {"[?status=='ACTIVE']", nil}, {"type(@)", nil}, {"`true`", true}, {"@ == @", true}, {"[ @ ] == [ @ ]", true}, {"{x: @} == {x: @}", true}, {"@ == `null`", false}, {"@ != @", false},
	} {
		t.Run(test.expression, func(t *testing.T) {
			value, err := jmespath.Search(test.expression, generator)
			if err != nil || value != test.want {
				t.Fatal(value, err, test.want)
			}
		})
	}
	for _, expression := range []string{"length(@)", "sort_by(@,&id)", "contains(@,'x')", "to_string(@)", "unknown_function(@)", "["} {
		if value, err := jmespath.Search(expression, generator); err == nil {
			t.Fatal(expression, value)
		}
	}
	for _, expression := range []string{"@", "not_null(@,`[]`)", "@ || `[]`"} {
		if value, err := jmespath.Search(expression, generator); err != nil || value != generator {
			t.Fatal(expression, value, err)
		}
	}
	value, err := jmespath.Search("to_array(@)", generator)
	array, ok := value.([]any)
	if err != nil || !ok || len(array) != 1 || array[0] != generator {
		t.Fatal(value, err)
	}
}

func TestOpaqueIdentityIsOptInAndKeepsLegacyJSONEquality(t *testing.T) {
	first, second := &opaqueGenerator{marker: 1}, &opaqueGenerator{marker: 1}
	for _, test := range []struct {
		input any
		want  bool
	}{
		{map[string]any{"left": first, "right": first}, true}, {map[string]any{"left": first, "right": second}, false},
		{map[string]any{"left": []any{first}, "right": []any{first}}, true}, {map[string]any{"left": []any{first}, "right": []any{second}}, false},
		{map[string]any{"left": map[string]any{"x": first}, "right": map[string]any{"x": first}}, true},
		{map[string]any{"left": json.Number("1"), "right": true}, false},
		{map[string]any{"left": []any{json.Number("1")}, "right": []any{true}}, true},
	} {
		value, err := jmespath.Search("left == right", test.input)
		if err != nil || value != test.want {
			t.Fatal(value, err, test.want)
		}
	}
}

func TestOpaqueIdentityRejectsInvalidKeysWithoutPanicking(t *testing.T) {
	for _, key := range []any{nil, "key", 1, []int{1}, map[string]int{"key": 1}, func() {}} {
		marker := invalidOpaqueMarker{key: key}
		if value, err := jmespath.Search("@ == @", marker); err == nil {
			t.Fatal(key, value)
		}
		if value, err := jmespath.Search("left == right", map[string]any{"left": []any{marker}, "right": []any{marker}}); err == nil {
			t.Fatal(key, value)
		}
	}
}
