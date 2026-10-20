package cloudfilter_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

func TestCloudFilterOrderedMappingConsumesOnlyReachedKeysAndKeepsDuplicateFirstPosition(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"v","status":"bad","metadata":{"role":"db"}}]`)
	for _, raw := range []string{`{"status":"good","unknown":null}`, `{"metadata":{"role":"web","unknown":null},"unknown":null}`, `{"status":"bad","unknown":null,"status":"good"}`} {
		cloudFilterContractSelect(t, rows, "", cloudFilterContractFilter(raw), []int{})
	}
	for _, raw := range []string{`{"unknown":null,"status":"good"}`, `{"status":"good","unknown":null,"status":"bad"}`, `{"metadata":{"role":"db","unknown":null}}`} {
		result, err := cloudfilter.Select(rows, "", cloudFilterContractFilter(raw), nil)
		if err == nil || !strings.Contains(err.Error(), "unknown") || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal(result, err)
		}
	}
	cloudFilterContractSelect(t, cloudFilterContractRows(t, `[{"known":null}]`), "", cloudFilterContractFilter(`{"known":null}`), []int{0})
	result, err := cloudfilter.Select(cloudFilterContractRows(t, `[{"known":null}]`), "", cloudFilterContractFilter(`{"missing":null}`), nil)
	if err == nil || !strings.Contains(err.Error(), "missing") || result.Value != nil {
		t.Fatal("missing field became null", result, err)
	}
}

func TestCloudFilterNestedEmptyObjectsUseActualTruthinessBeforeShapeOrMissingKeys(t *testing.T) {
	for _, tc := range []struct {
		value   string
		matches bool
	}{{`null`, false}, {`false`, false}, {`0`, false}, {`-0e99999`, false}, {`""`, false}, {`[]`, false}, {`{}`, false}, {`true`, true}, {`1`, true}, {`-0.0001`, true}, {`"x"`, true}, {`[1]`, true}, {`{"x":1}`, true}} {
		t.Run(tc.value, func(t *testing.T) {
			rows := cloudFilterContractRows(t, `[{"metadata":`+tc.value+`}]`)
			indices := []int{}
			if tc.matches {
				indices = []int{0}
			}
			cloudFilterContractSelect(t, rows, "", cloudFilterContractFilter(`{"metadata":{}}`), indices)
		})
	}
	for _, value := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`} {
		cloudFilterContractSelect(t, cloudFilterContractRows(t, `[{"metadata":`+value+`}]`), "", cloudFilterContractFilter(`{"metadata":{"unknown":true}}`), []int{})
	}
	for _, value := range []string{`true`, `1`, `"x"`, `"missing"`, `[1]`, `["missing"]`} {
		result, err := cloudfilter.Select(cloudFilterContractRows(t, `[{"metadata":`+value+`}]`), "", cloudFilterContractFilter(`{"metadata":{"missing":true}}`), nil)
		if err == nil || !strings.Contains(err.Error(), "metadata") || result.Value != nil || result.Indices != nil {
			t.Fatal(result, err)
		}
	}
}

func TestCloudFilterLeafEqualityKeepsExactArrayShapeAndPythonBooleanNumberPrecision(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"v","leaf":[true,{"a":false,"large":9007199254740993}],"metadata":{"a":true,"extra":1}}]`)
	cloudFilterContractSelect(t, rows, "", cloudFilterContractFilter(`{"leaf":[1,{"a":0,"large":9007199254740993.0}],"metadata":{"a":1}}`), []int{0})
	for _, filter := range []string{`{"leaf":[1,{"a":0,"large":9007199254740992}]}`, `{"leaf":[1,{"a":0}]}`, `{"leaf":[{"a":0,"large":9007199254740993},1]}`} {
		cloudFilterContractSelect(t, rows, "", cloudFilterContractFilter(filter), []int{})
	}
	cloudFilterContractSelect(t, cloudFilterContractRows(t, `[{"leaf":1e999999999999999999999999999999}]`), "", cloudFilterContractFilter(`{"leaf":10e999999999999999999999999999998}`), []int{0})
}

func TestCloudFilterNonmappingFilterShapeIsLazyAfterIdentifierAndActualTruthiness(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `-0.0e99999`, `""`, `[]`, `{}`} {
		cloudFilterContractSelect(t, cloudFilterContractRows(t, `[{}, {"id":"v"}]`), "", cloudFilterContractFilter(raw), []int{0, 1})
	}
	for _, raw := range []string{`true`, `1`, `[1]`} {
		cloudFilterContractSelect(t, []json.RawMessage{}, "", cloudFilterContractFilter(raw), []int{})
		cloudFilterContractSelect(t, cloudFilterContractRows(t, `[{}]`), "", cloudFilterContractFilter(raw), []int{})
		cloudFilterContractSelect(t, cloudFilterContractRows(t, `[{"id":"v","name":"other"}]`), "no-match", cloudFilterContractFilter(raw), []int{})
		result, err := cloudfilter.Select(cloudFilterContractRows(t, `[{"id":"v"}]`), "", cloudFilterContractFilter(raw), nil)
		if err == nil || !strings.Contains(err.Error(), "mapping") || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal(result, err)
		}
	}
}

func TestCloudFilterInvalidJSONIsLocalAtomicAndStillValidatedAfterEmptyIdentifierSelection(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"v","name":"other"}]`)
	for _, raw := range []json.RawMessage{nil, {}, json.RawMessage(`{"x":`), json.RawMessage(`false true`), json.RawMessage([]byte{'"', 0xff, '"'})} {
		result, err := cloudfilter.Select(rows, "no-match", &raw, nil)
		if err == nil || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal(result, err)
		}
	}
	result, err := cloudfilter.Select(rows, "", cloudFilterContractFilter(`{"x":true,}`), nil)
	var syntax *json.SyntaxError
	if err == nil || !errors.As(err, &syntax) || result.Value != nil {
		t.Fatal("JSON conversion cause was lost", result, err)
	}
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`[]`), json.RawMessage(`false`), json.RawMessage(`{} {}`), json.RawMessage([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		result, err := cloudfilter.Select([]json.RawMessage{raw}, "", nil, nil)
		if err == nil || !reflect.DeepEqual(result, cloudfilter.Result{}) {
			t.Fatal(result, err)
		}
	}
	result, err = cloudfilter.Select(rows, string([]byte{0xff}), nil, nil)
	if err == nil || result.Value != nil {
		t.Fatal("invalid Unicode request pattern silently changed", result, err)
	}
}
