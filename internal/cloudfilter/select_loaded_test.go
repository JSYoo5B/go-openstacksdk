package cloudfilter_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

func loadedSelectFilter(raw string) *json.RawMessage { v := json.RawMessage(raw); return &v }
func assertLoadedSelect(t *testing.T, rows []json.RawMessage, filter string, want []int) {
	t.Helper()
	result, err := cloudfilter.SelectLoaded(rows, "", loadedSelectFilter(filter), nil)
	if err != nil || !reflect.DeepEqual(result.Indices, want) || result.Expression || result.Value == nil {
		t.Fatal(filter, result, err, want)
	}
}

func TestCloudSelectLoadedRoundedFloatEqualityKeepsIntegerTokensAndExactDefault(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"float","value":9007199254740993.0},{"id":"original-int","value":9007199254740993},{"id":"rounded-int","value":9007199254740992}]`)
	assertLoadedSelect(t, rows, `{"value":9007199254740993}`, []int{1})
	assertLoadedSelect(t, rows, `{"value":9007199254740992}`, []int{0, 2})
	baseline, err := cloudfilter.Select(rows, "", loadedSelectFilter(`{"value":9007199254740993}`), nil)
	if err != nil || !reflect.DeepEqual(baseline.Indices, []int{0, 1}) {
		t.Fatal("default exact policy changed", baseline, err)
	}
}
func TestCloudSelectLoadedUnderflowMatchesZeroAndFalseWithoutRoundingBigIntegers(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"value":1e-400},{"value":-1e-400},{"value":0},{"value":false},{"value":9007199254740993}]`)
	for _, filter := range []string{`{"value":0}`, `{"value":false}`, `{"value":1e-400}`} {
		assertLoadedSelect(t, rows, filter, []int{0, 1, 2, 3})
	}
	assertLoadedSelect(t, rows, `{"value":9007199254740993}`, []int{4})
	baseline, err := cloudfilter.Select(rows, "", loadedSelectFilter(`{"value":0}`), nil)
	if err != nil || !reflect.DeepEqual(baseline.Indices, []int{2, 3}) {
		t.Fatal("existing default was migrated", baseline, err)
	}
}
func TestCloudSelectLoadedOverflowInfinityDoesNotEqualFiniteBigInteger(t *testing.T) {
	huge := strings.Repeat("9", 401)
	rows := cloudFilterContractRows(t, `[{"value":1e400},{"value":10e399},{"value":-1e400},{"value":`+huge+`}]`)
	assertLoadedSelect(t, rows, `{"value":1e400}`, []int{0, 1})
	assertLoadedSelect(t, rows, `{"value":-10e399}`, []int{2})
	assertLoadedSelect(t, rows, `{"value":`+huge+`}`, []int{3})
}
func TestCloudSelectLoadedBooleanNumericAliasesAndDecimalFloatEquality(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"value":true},{"value":1.0},{"value":false},{"value":-0.0},{"value":null},{"value":"1"},{"value":0.1},{"value":0.10000000000000001}]`)
	assertLoadedSelect(t, rows, `{"value":1}`, []int{0, 1})
	assertLoadedSelect(t, rows, `{"value":0}`, []int{2, 3})
	assertLoadedSelect(t, rows, `{"value":null}`, []int{4})
	assertLoadedSelect(t, rows, `{"value":"1"}`, []int{5})
	assertLoadedSelect(t, rows, `{"value":0.1}`, []int{6, 7})
}
func TestCloudSelectLoadedNestedEmptyDictionaryUsesLoadedActualTruthiness(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"metadata":1e-400},{"metadata":0},{"metadata":false},{"metadata":null},{"metadata":{}},{"metadata":[]},{"metadata":true},{"metadata":1e400},{"metadata":"x"},{"metadata":[false]}]`)
	assertLoadedSelect(t, rows, `{"metadata":{}}`, []int{6, 7, 8, 9})
	baseline, err := cloudfilter.Select(rows, "", loadedSelectFilter(`{"metadata":{}}`), nil)
	if err != nil || !reflect.DeepEqual(baseline.Indices, []int{0, 6, 7, 8, 9}) {
		t.Fatal("default truthiness changed", baseline, err)
	}
}
func TestCloudSelectLoadedFalseyScalarFilterSkipsMappingWhileTruthyScalarsFail(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"a"},{"id":"b"}]`)
	for _, filter := range []string{`1e-400`, `-1e-400`, `false`, `0`, `null`, `""`, `[]`, `{}`} {
		assertLoadedSelect(t, rows, filter, []int{0, 1})
	}
	for _, filter := range []string{`true`, `1e400`, `[false]`} {
		result, err := cloudfilter.SelectLoaded(rows, "", loadedSelectFilter(filter), nil)
		if err == nil || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal(filter, result, err)
		}
	}
	// The exact default still treats a nonzero underflow token as a truthy,
	// non-mapping filter, which is intentionally separate from this Source opt-in.
	baseline, err := cloudfilter.Select(rows, "", loadedSelectFilter(`1e-400`), nil)
	if err == nil || !reflect.DeepEqual(baseline, cloudfilter.Result{}) {
		t.Fatal(baseline, err)
	}
}
func TestCloudSelectLoadedOrderedPredicatesKeepUnknownRuntimeFailureBoundary(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"state":"keep","metadata":true}]`)
	for _, filter := range []string{`{"state":"drop","unknown":null}`, `{"state":"keep","unknown":null,"state":"drop"}`} {
		assertLoadedSelect(t, rows, filter, []int{})
	}
	for _, filter := range []string{`{"unknown":null,"state":"drop"}`, `{"metadata":{"x":1}}`} {
		result, err := cloudfilter.SelectLoaded(rows, "", loadedSelectFilter(filter), nil)
		if err == nil || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal(filter, result, err)
		}
	}
	assertLoadedSelect(t, cloudFilterContractRows(t, `[{}]`), `{"unknown":null}`, []int{})
}
func TestCloudSelectLoadedRecursiveMappingsAndArrayLeavesKeepDictionaryWidth(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"metadata":{"nested":{"rounded":9007199254740993.0,"tiny":1e-400,"extra":9},"rows":[true,{"x":1e-400}]}},{"metadata":{"nested":{"rounded":9007199254740993.0,"tiny":1e-400},"rows":[true,{"x":1e-400,"extra":9}]}}]`)
	assertLoadedSelect(t, rows, `{"metadata":{"nested":{"rounded":9007199254740992,"tiny":false}}}`, []int{0, 1})
	assertLoadedSelect(t, rows, `{"metadata":{"rows":[1,{"x":false}]}}`, []int{0})
	assertLoadedSelect(t, rows, `{"metadata":{"rows":[{"x":false},1]}}`, []int{})
}
func TestCloudSelectLoadedIdentifierPhaseAndRawOutputRemainOwned(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"selected-a","name":"alpha","value":1e-400},{"id":"selected-b","name":"beta","value":0},{"id":"other","value":0}]`)
	result, err := cloudfilter.SelectLoaded(rows, "selected-*", loadedSelectFilter(`{"value":false}`), nil)
	if err != nil || !reflect.DeepEqual(result.Indices, []int{0, 1}) || !strings.Contains(string(result.Value), "1e-400") {
		t.Fatal(result, err)
	}
	result.Value[0] = '!'
	result.Indices[0] = 9
	sibling, err := cloudfilter.SelectLoaded(rows, "selected-*", loadedSelectFilter(`{"value":false}`), nil)
	if err != nil || !reflect.DeepEqual(sibling.Indices, []int{0, 1}) || string(rows[0]) != `{"id":"selected-a","name":"alpha","value":1e-400}` {
		t.Fatal(sibling, err, rows)
	}
}
func TestCloudSelectLoadedExpressionBridgePreservesArbitraryOutputAndAtomicRuntimeErrors(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"a","value":1e-400},{"id":"b","value":0}]`)
	for _, tc := range []struct{ expression, want string }{{"[].id", `["a","b"]`}, {"length(@)", "2"}, {"`false`", "false"}} {
		raw, _ := json.Marshal(tc.expression)
		filter := json.RawMessage(raw)
		result, err := cloudfilter.SelectLoaded(rows, "", &filter, nil)
		if err != nil || !result.Expression || result.Indices != nil || string(result.Value) != tc.want {
			t.Fatal(result, err, tc)
		}
	}
	for _, expression := range []string{"[?", "unknown(@)", "length(missing)[]"} {
		raw, _ := json.Marshal(expression)
		filter := json.RawMessage(raw)
		result, err := cloudfilter.SelectLoaded(rows, "no-match", &filter, nil)
		if err == nil || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal(expression, result, err)
		}
	}
}
func TestCloudSelectLoadedOwnsInputsBeforeMutatingGuardCallbacks(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"metadata":{"value":9007199254740993.0}},{"metadata":{"value":9007199254740993}}]`)
	filter := json.RawMessage(`{"metadata":{"value":9007199254740992}}`)
	changed := false
	result, err := cloudfilter.SelectLoaded(rows, "", &filter, func() error {
		if !changed {
			changed = true
			rows[0][0] = '!'
			rows[1] = json.RawMessage("null")
			filter[0] = '!'
		}
		return nil
	})
	if err != nil || !changed || !reflect.DeepEqual(result.Indices, []int{0}) || !strings.Contains(string(result.Value), "9007199254740993.0") {
		t.Fatal(result, err)
	}
}
func TestCloudSelectLoadedOneShotGuardFailureCannotReturnPartialRows(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"metadata":{"tiny":1e-400}},{"metadata":{"tiny":0}}]`)
	cause := errors.New("selected source changed once")
	calls := 0
	result, err := cloudfilter.SelectLoaded(rows, "", loadedSelectFilter(`{"metadata":{"tiny":false}}`), func() error {
		calls++
		if calls == 5 {
			return cause
		}
		return nil
	})
	if !errors.Is(err, cause) || !reflect.DeepEqual(result, cloudfilter.Result{}) || calls < 6 {
		t.Fatal(result, err, calls)
	}
	// A failure at the first boundary still reaches final observation without
	// allowing a later nil-returning guard to turn it into a successful result.
	calls = 0
	result, err = cloudfilter.SelectLoaded(rows, "", nil, func() error {
		calls++
		if calls == 1 {
			return cause
		}
		return nil
	})
	if !errors.Is(err, cause) || !reflect.DeepEqual(result, cloudfilter.Result{}) || calls != 2 {
		t.Fatal(result, err, calls)
	}
}
func TestCloudSelectLoadedInvalidJSONUTF8AndPatternsRemainAtomic(t *testing.T) {
	for _, row := range []json.RawMessage{nil, json.RawMessage(`[]`), json.RawMessage(`{} []`), json.RawMessage(`{"x":"` + string([]byte{0xff}) + `"}`)} {
		result, err := cloudfilter.SelectLoaded([]json.RawMessage{row}, "", nil, nil)
		if err == nil || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal(string(row), result, err)
		}
	}
	rows := cloudFilterContractRows(t, `[{"value":1}]`)
	for _, filter := range []json.RawMessage{nil, json.RawMessage(``), json.RawMessage(`false true`), json.RawMessage(`"` + string([]byte{0xff}) + `"`), json.RawMessage(`NaN`)} {
		result, err := cloudfilter.SelectLoaded(rows, "", &filter, nil)
		if err == nil || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal(string(filter), result, err)
		}
	}
	result, err := cloudfilter.SelectLoaded(rows, string([]byte{0xff}), nil, nil)
	if err == nil || !reflect.DeepEqual(result, cloudfilter.Result{}) {
		t.Fatal(result, err)
	}
}
