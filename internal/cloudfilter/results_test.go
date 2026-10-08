package cloudfilter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

func TestCloudFilterExpressionBridgeReturnsArbitraryOwnedJSONWithoutInventedRowIndices(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"a","size":0,"metadata":{"role":"db"}},{"id":"b","size":9007199254740993,"metadata":{"role":"web"}}]`)
	for _, tc := range []struct{ expression, expected string }{
		{`[].id`, `["a","b"]`}, {`[].size`, `[0,9007199254740993]`}, {`length(@)`, `2`}, {`@ | [0].metadata`, `{"role":"db"}`},
		{`[?metadata.role=='db'].id`, `["a"]`}, {`[?size].id`, `["a","b"]`}, {`'string'`, `"string"`}, {"`false`", `false`}, {"`null`", `null`}, {"`0`", `0`},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			encoded, err := json.Marshal(tc.expression)
			if err != nil {
				t.Fatal(err)
			}
			filter := json.RawMessage(encoded)
			result, err := cloudfilter.Select(rows, "", &filter, nil)
			if err != nil || !result.Expression || result.Indices != nil || result.Value == nil {
				t.Fatal(result, err)
			}
			cloudFilterContractEqual(t, result.Value, tc.expected)
		})
	}
	encoded, _ := json.Marshal(`@`)
	filter := json.RawMessage(encoded)
	result, err := cloudfilter.Select(rows, "no-match", &filter, nil)
	if err != nil || !result.Expression || result.Indices != nil {
		t.Fatal(result, err)
	}
	cloudFilterContractEqual(t, result.Value, `[]`)
}

func TestCloudFilterExpressionsStillParseEmptySelectionAndKeepRuntimeFailuresAtomic(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"v","name":"other"}]`)
	for _, expression := range []string{`[?`, `unknown(@)`, `length(missing)[]`, `length(missing)[?@]`} {
		encoded, _ := json.Marshal(expression)
		raw := json.RawMessage(encoded)
		result, err := cloudfilter.Select(rows, "no-match", &raw, nil)
		if err == nil || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal("expression error was swallowed or gained partial output", result, err)
		}
	}
	for _, rows := range [][]json.RawMessage{nil, cloudFilterContractRows(t, `[{"id":"v","name":"other"}]`)} {
		encoded, _ := json.Marshal(`[].unknown(@)`)
		raw := json.RawMessage(encoded)
		result, err := cloudfilter.Select(rows, "no-match", &raw, nil)
		if err != nil || !result.Expression || result.Indices != nil {
			t.Fatal("unvisited expression function was evaluated", result, err)
		}
		cloudFilterContractEqual(t, result.Value, `[]`)
	}
}

func TestCloudFilterFirstFollowsTruthyLengthAndIntegerIndexWithoutRetestingSelectedFalsyValue(t *testing.T) {
	for _, tc := range []struct {
		raw, expected string
		absent        bool
		multiple      int
		shapeError    bool
	}{
		{raw: `null`, absent: true}, {raw: `false`, absent: true}, {raw: `0`, absent: true}, {raw: `-0.0e99999`, absent: true}, {raw: `""`, absent: true}, {raw: `[]`, absent: true}, {raw: `{}`, absent: true},
		{raw: `[0]`, expected: `0`}, {raw: `[false]`, expected: `false`}, {raw: `[""]`, expected: `""`}, {raw: `[null]`, absent: true}, {raw: `[{"id":false}]`, expected: `{"id":false}`},
		{raw: `"중"`, expected: `"중"`}, {raw: `"중a"`, multiple: 2}, {raw: `[1,2]`, multiple: 2}, {raw: `{"a":1,"b":2}`, multiple: 2},
		{raw: `true`, shapeError: true}, {raw: `1`, shapeError: true}, {raw: `{"0":"literal string key"}`, shapeError: true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			value, err := cloudfilter.First(json.RawMessage(tc.raw))
			var ambiguity *cloudfilter.MultipleError
			if tc.multiple != 0 {
				if value != nil || !errors.As(err, &ambiguity) || ambiguity.Length != tc.multiple {
					t.Fatal(value, err)
				}
				return
			}
			if tc.shapeError {
				if value != nil || err == nil || errors.As(err, &ambiguity) {
					t.Fatal(value, err)
				}
				return
			}
			if err != nil || (value == nil) != tc.absent {
				t.Fatal(value, err)
			}
			if !tc.absent {
				cloudFilterContractEqual(t, value, tc.expected)
			}
		})
	}
	if value, err := cloudfilter.First(nil); value != nil || err != nil {
		t.Fatal(value, err)
	}
	source := json.RawMessage(`[ "\u0061" ]`)
	value, err := cloudfilter.First(source)
	if err != nil || string(value) != `"\u0061"` {
		t.Fatal("escaped selected JSON string changed", string(value), err)
	}
	value[0] = '!'
	if string(source) != `[ "\u0061" ]` {
		t.Fatal("selected result aliases source", string(source))
	}
	for _, invalid := range []json.RawMessage{{}, json.RawMessage(`[] true`), json.RawMessage([]byte{'"', 0xff, '"'})} {
		value, err := cloudfilter.First(invalid)
		if value != nil || err == nil {
			t.Fatal(value, err)
		}
	}
}

func TestCloudFilterPythonTruthyPreservesExactDecimalZeroPolicyAndNullableResults(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want bool
	}{{`null`, false}, {`false`, false}, {`0`, false}, {`-0e999999999999999999999999999`, false}, {`""`, false}, {`[]`, false}, {`{}`, false}, {`true`, true}, {`1e-10000`, true}, {`9007199254740993`, true}, {`"false"`, true}, {`[false]`, true}, {`{"x":null}`, true}} {
		got, err := cloudfilter.PythonTruthy(json.RawMessage(tc.raw))
		if err != nil || got != tc.want {
			t.Fatal(tc, got, err)
		}
	}
	if got, err := cloudfilter.PythonTruthy(nil); got || err != nil {
		t.Fatal(got, err)
	}
	if got, err := cloudfilter.PythonTruthy(json.RawMessage{}); got || err == nil {
		t.Fatal("empty supplied JSON confused with absent result", got, err)
	}
}

func TestCloudFilterSelectOwnsViewsFiltersIndicesAndResultsBeforeGuardCallbacks(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"v","nested":{"number":9007199254740993}},{"id":"other"}]`)
	original := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		original[i] = bytes.Clone(row)
	}
	filter := json.RawMessage(`{"id":"v"}`)
	changed := false
	result, err := cloudfilter.Select(rows, "", &filter, func() error {
		if !changed {
			changed = true
			rows[0][0] = '!'
			rows[1] = json.RawMessage(`null`)
			filter[0] = '!'
			filter = json.RawMessage(`true`)
		}
		return nil
	})
	if err != nil || !changed || !reflect.DeepEqual(result.Indices, []int{0}) {
		t.Fatal(result, err)
	}
	cloudFilterContractEqual(t, result.Value, `[{"id":"v","nested":{"number":9007199254740993}}]`)
	sibling, err := cloudfilter.Select(original, "", cloudFilterContractFilter(`{"id":"v"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	result.Value[0] = '!'
	result.Indices[0] = 999
	if !reflect.DeepEqual(sibling.Indices, []int{0}) || original[0][0] != '{' {
		t.Fatal("owned selections share indices or source bytes", sibling)
	}
	cloudFilterContractEqual(t, sibling.Value, `[{"id":"v","nested":{"number":9007199254740993}}]`)
	reusable := cloudFilterContractFilter(`{"id":"v"}`)
	var wait sync.WaitGroup
	for range 12 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := cloudfilter.Select(original, "", reusable, nil)
			if err != nil || !reflect.DeepEqual(value.Indices, []int{0}) {
				t.Error(value, err)
				return
			}
			value.Value[0] = '!'
			value.Indices[0] = 999
		}()
	}
	wait.Wait()
	if string(*reusable) != `{"id":"v"}` || original[0][0] != '{' {
		t.Fatal("reused input views/filters were mutated")
	}
}

func TestCloudFilterGuardInterruptsTraversalAndRetainsCancellationCauseWithoutPartialSelection(t *testing.T) {
	cause := errors.New("source no longer matches")
	result, err := cloudfilter.Select([]json.RawMessage{json.RawMessage(`broken`)}, "", cloudFilterContractFilter(`broken`), func() error { return cause })
	if !errors.Is(err, cause) || !reflect.DeepEqual(result, cloudfilter.Result{}) {
		t.Fatal("guard failure lost or invalid caller JSON was consumed first", result, err)
	}
	rows := make([]json.RawMessage, 100)
	for i := range rows {
		rows[i] = json.RawMessage(`{"id":"v","name":"other"}`)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	observations := 0
	result, err = cloudfilter.Select(rows, "v", nil, func() error {
		observations++
		if observations >= 10 {
			cancel(cause)
		}
		if ctx.Err() != nil {
			return errors.Join(ctx.Err(), context.Cause(ctx))
		}
		return nil
	})
	if observations < 10 || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !reflect.DeepEqual(result, cloudfilter.Result{}) {
		t.Fatal("interrupted traversal exposed partial rows/indices or lost caller cause", result, err, observations)
	}
}
