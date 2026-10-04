// Adapted from github.com/jmespath/go-jmespath v0.4.0.
// Copyright 2015 James Saryerwinnie; licensed under Apache-2.0.
// See LICENSE and README.md for attribution and local changes.

package jmespath

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

type jpFunction func(arguments []interface{}) (interface{}, error)

type jpType string

const (
	jpUnknown     jpType = "unknown"
	jpNumber      jpType = "number"
	jpString      jpType = "string"
	jpArray       jpType = "array"
	jpObject      jpType = "object"
	jpArrayNumber jpType = "array[number]"
	jpArrayString jpType = "array[string]"
	jpExpref      jpType = "expref"
	jpAny         jpType = "any"
)

type functionEntry struct {
	name      string
	arguments []argSpec
	handler   jpFunction
	hasExpRef bool
}

type argSpec struct {
	types    []jpType
	variadic bool
}

type functionCaller struct {
	functionTable map[string]functionEntry
}

func newFunctionCaller() *functionCaller {
	caller := &functionCaller{}
	caller.functionTable = map[string]functionEntry{
		"length": {
			name: "length",
			arguments: []argSpec{
				{types: []jpType{jpString, jpArray, jpObject}},
			},
			handler: jpfLength,
		},
		"starts_with": {
			name: "starts_with",
			arguments: []argSpec{
				{types: []jpType{jpString}},
				{types: []jpType{jpString}},
			},
			handler: jpfStartsWith,
		},
		"abs": {
			name: "abs",
			arguments: []argSpec{
				{types: []jpType{jpNumber}},
			},
			handler: jpfAbs,
		},
		"avg": {
			name: "avg",
			arguments: []argSpec{
				{types: []jpType{jpArrayNumber}},
			},
			handler: jpfAvg,
		},
		"ceil": {
			name: "ceil",
			arguments: []argSpec{
				{types: []jpType{jpNumber}},
			},
			handler: jpfCeil,
		},
		"contains": {
			name: "contains",
			arguments: []argSpec{
				{types: []jpType{jpArray, jpString}},
				{types: []jpType{jpAny}},
			},
			handler: jpfContains,
		},
		"ends_with": {
			name: "ends_with",
			arguments: []argSpec{
				{types: []jpType{jpString}},
				{types: []jpType{jpString}},
			},
			handler: jpfEndsWith,
		},
		"floor": {
			name: "floor",
			arguments: []argSpec{
				{types: []jpType{jpNumber}},
			},
			handler: jpfFloor,
		},
		"map": {
			name: "map",
			arguments: []argSpec{
				{types: []jpType{jpExpref}},
				{types: []jpType{jpArray}},
			},
			handler:   jpfMap,
			hasExpRef: true,
		},
		"max": {
			name: "max",
			arguments: []argSpec{
				{types: []jpType{jpArrayNumber, jpArrayString}},
			},
			handler: jpfMax,
		},
		"merge": {
			name: "merge",
			arguments: []argSpec{
				{types: []jpType{jpObject}, variadic: true},
			},
			handler: jpfMerge,
		},
		"max_by": {
			name: "max_by",
			arguments: []argSpec{
				{types: []jpType{jpArray}},
				{types: []jpType{jpExpref}},
			},
			handler:   jpfMaxBy,
			hasExpRef: true,
		},
		"sum": {
			name: "sum",
			arguments: []argSpec{
				{types: []jpType{jpArrayNumber}},
			},
			handler: jpfSum,
		},
		"min": {
			name: "min",
			arguments: []argSpec{
				{types: []jpType{jpArrayNumber, jpArrayString}},
			},
			handler: jpfMin,
		},
		"min_by": {
			name: "min_by",
			arguments: []argSpec{
				{types: []jpType{jpArray}},
				{types: []jpType{jpExpref}},
			},
			handler:   jpfMinBy,
			hasExpRef: true,
		},
		"type": {
			name: "type",
			arguments: []argSpec{
				{types: []jpType{jpAny}},
			},
			handler: jpfType,
		},
		"keys": {
			name: "keys",
			arguments: []argSpec{
				{types: []jpType{jpObject}},
			},
			handler: jpfKeys,
		},
		"values": {
			name: "values",
			arguments: []argSpec{
				{types: []jpType{jpObject}},
			},
			handler: jpfValues,
		},
		"sort": {
			name: "sort",
			arguments: []argSpec{
				{types: []jpType{jpArrayString, jpArrayNumber}},
			},
			handler: jpfSort,
		},
		"sort_by": {
			name: "sort_by",
			arguments: []argSpec{
				{types: []jpType{jpArray}},
				{types: []jpType{jpExpref}},
			},
			handler:   jpfSortBy,
			hasExpRef: true,
		},
		"join": {
			name: "join",
			arguments: []argSpec{
				{types: []jpType{jpString}},
				{types: []jpType{jpArrayString}},
			},
			handler: jpfJoin,
		},
		"reverse": {
			name: "reverse",
			arguments: []argSpec{
				{types: []jpType{jpArray, jpString}},
			},
			handler: jpfReverse,
		},
		"to_array": {
			name: "to_array",
			arguments: []argSpec{
				{types: []jpType{jpAny}},
			},
			handler: jpfToArray,
		},
		"to_string": {
			name: "to_string",
			arguments: []argSpec{
				{types: []jpType{jpAny}},
			},
			handler: jpfToString,
		},
		"to_number": {
			name: "to_number",
			arguments: []argSpec{
				{types: []jpType{jpAny}},
			},
			handler: jpfToNumber,
		},
		"not_null": {
			name: "not_null",
			arguments: []argSpec{
				{types: []jpType{jpAny}, variadic: true},
			},
			handler: jpfNotNull,
		},
	}
	return caller
}

func (e *functionEntry) resolveArgs(arguments []any) ([]any, error) {
	variadic := len(e.arguments) > 0 && e.arguments[len(e.arguments)-1].variadic
	if !variadic && len(arguments) != len(e.arguments) || variadic && len(arguments) < len(e.arguments) {
		return nil, fmt.Errorf("%s(): invalid arity: received %d arguments", e.name, len(arguments))
	}
	for i, argument := range arguments {
		index := i
		if index >= len(e.arguments) {
			index = len(e.arguments) - 1
		}
		if err := e.arguments[index].typeCheck(argument); err != nil {
			return nil, fmt.Errorf("%s(): argument %d: %w", e.name, i+1, err)
		}
	}
	return arguments, nil
}

func (a *argSpec) typeCheck(value any) error {
	for _, kind := range a.types {
		switch kind {
		case jpAny:
			return nil
		case jpNumber:
			if number, ok := value.(json.Number); ok {
				if _, err := parseNumber(number); err != nil {
					return err
				}
				return nil
			}
		case jpString:
			if _, ok := value.(string); ok {
				return nil
			}
		case jpArray:
			if _, ok := value.([]any); ok {
				return nil
			}
		case jpObject:
			if _, ok := value.(map[string]any); ok {
				return nil
			}
		case jpExpref:
			if _, ok := value.(expRef); ok {
				return nil
			}
		case jpArrayNumber:
			if numbers, ok := toArrayNum(value); ok {
				for _, number := range numbers {
					if _, err := parseNumber(number); err != nil {
						return err
					}
				}
				return nil
			}
		case jpArrayString:
			if _, ok := toArrayStr(value); ok {
				return nil
			}
		}
	}
	return fmt.Errorf("invalid type %T, expected %v", value, a.types)
}

func (f *functionCaller) CallFunction(name string, arguments []any, intr *treeInterpreter) (any, error) {
	entry, ok := f.functionTable[name]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", name)
	}
	resolved, err := entry.resolveArgs(arguments)
	if err != nil {
		return nil, err
	}
	if entry.hasExpRef {
		resolved = append([]any{intr}, resolved...)
	}
	value, err := entry.handler(resolved)
	if err != nil {
		return nil, fmt.Errorf("%s(): %w", name, err)
	}
	return value, nil
}

func jpfAbs(arguments []any) (any, error)   { return absNumber(arguments[0].(json.Number)) }
func jpfCeil(arguments []any) (any, error)  { return roundNumber(arguments[0].(json.Number), true) }
func jpfFloor(arguments []any) (any, error) { return roundNumber(arguments[0].(json.Number), false) }

func jpfLength(arguments []any) (any, error) {
	length := 0
	switch value := arguments[0].(type) {
	case string:
		length = utf8.RuneCountInString(value)
	case []any:
		length = len(value)
	case map[string]any:
		length = len(value)
	default:
		return nil, errors.New("cannot compute length")
	}
	return json.Number(strconv.Itoa(length)), nil
}

func jpfStartsWith(arguments []any) (any, error) {
	return strings.HasPrefix(arguments[0].(string), arguments[1].(string)), nil
}
func jpfEndsWith(arguments []any) (any, error) {
	return strings.HasSuffix(arguments[0].(string), arguments[1].(string)), nil
}

func jpfAvg(arguments []any) (any, error) {
	numbers, _ := toArrayNum(arguments[0])
	if len(numbers) == 0 {
		return nil, nil
	}
	sum, err := addNumbers(numbers)
	if err != nil {
		return nil, err
	}
	numerator, err := numberToFloat(sum)
	if err != nil {
		return nil, err
	}
	return numberFromFloat(numerator / float64(len(numbers)))
}

func jpfSum(arguments []any) (any, error) {
	numbers, _ := toArrayNum(arguments[0])
	return addNumbers(numbers)
}

func jpfContains(arguments []any) (any, error) {
	if text, ok := arguments[0].(string); ok {
		search, ok := arguments[1].(string)
		if !ok {
			return nil, fmt.Errorf("string contains requires a string search value")
		}
		return strings.Contains(text, search), nil
	}
	for _, item := range arguments[0].([]any) {
		equal, err := jsonEqual(item, arguments[1], true)
		if err != nil {
			return nil, err
		}
		if equal {
			return true, nil
		}
	}
	return false, nil
}

func jpfMap(arguments []any) (any, error) {
	intr := arguments[0].(*treeInterpreter)
	expression := arguments[1].(expRef)
	values := arguments[2].([]any)
	result := make([]any, 0, len(values))
	for _, value := range values {
		selected, err := intr.Execute(expression.ref, value)
		if err != nil {
			return nil, err
		}
		result = append(result, selected)
	}
	return result, nil
}

func jpfMax(arguments []any) (any, error) { return extreme(arguments[0].([]any), true) }
func jpfMin(arguments []any) (any, error) { return extreme(arguments[0].([]any), false) }

func extreme(values []any, maximum bool) (any, error) {
	if len(values) == 0 {
		return nil, nil
	}
	best := values[0]
	for _, value := range values[1:] {
		comparison, ok, err := orderedCompare(value, best)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("extreme values must be numbers or strings")
		}
		if maximum && comparison > 0 || !maximum && comparison < 0 {
			best = value
		}
	}
	return best, nil
}

func jpfMerge(arguments []any) (any, error) {
	result := make(map[string]any)
	for _, value := range arguments {
		for key, item := range value.(map[string]any) {
			result[key] = item
		}
	}
	return result, nil
}

func jpfMaxBy(arguments []any) (any, error) { return extremeBy(arguments, true) }
func jpfMinBy(arguments []any) (any, error) { return extremeBy(arguments, false) }

func extremeBy(arguments []any, maximum bool) (any, error) {
	intr := arguments[0].(*treeInterpreter)
	values := arguments[1].([]any)
	expression := arguments[2].(expRef)
	if len(values) == 0 {
		return nil, nil
	}
	best := values[0]
	bestKey, err := intr.Execute(expression.ref, best)
	if err != nil {
		return nil, err
	}
	if err := comparableKey(bestKey); err != nil {
		return nil, err
	}
	for _, value := range values[1:] {
		key, err := intr.Execute(expression.ref, value)
		if err != nil {
			return nil, err
		}
		if err := comparableKey(key); err != nil {
			return nil, err
		}
		comparison, ok, err := orderedCompare(key, bestKey)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("by-key values must be numbers or strings")
		}
		if maximum && comparison > 0 || !maximum && comparison < 0 {
			best, bestKey = value, key
		}
	}
	return best, nil
}

func comparableKey(value any) error {
	if number, ok := value.(json.Number); ok {
		_, err := parseNumber(number)
		return err
	}
	if _, ok := value.(string); ok {
		return nil
	}
	return fmt.Errorf("by-key value has invalid type %T; expected number or string", value)
}

func jpfType(arguments []any) (any, error) {
	switch arguments[0].(type) {
	case nil:
		return "null", nil
	case bool:
		return "boolean", nil
	case string:
		return "string", nil
	case json.Number:
		return "number", nil
	case []any:
		return "array", nil
	case map[string]any:
		return "object", nil
	default:
		return nil, nil
	}
}

func jpfKeys(arguments []any) (any, error) {
	keys := sortedKeys(arguments[0].(map[string]any))
	result := make([]any, len(keys))
	for i, key := range keys {
		result[i] = key
	}
	return result, nil
}

func jpfValues(arguments []any) (any, error) {
	value := arguments[0].(map[string]any)
	keys := sortedKeys(value)
	result := make([]any, len(keys))
	for i, key := range keys {
		result[i] = value[key]
	}
	return result, nil
}

func jpfSort(arguments []any) (any, error) {
	values := arguments[0].([]any)
	result := append([]any{}, values...)
	var comparisonError error
	sort.SliceStable(result, func(i, j int) bool {
		comparison, _, err := orderedCompare(result[i], result[j])
		if err != nil {
			comparisonError = err
			return false
		}
		return comparison < 0
	})
	if comparisonError != nil {
		return nil, comparisonError
	}
	return result, nil
}

func jpfSortBy(arguments []any) (any, error) {
	intr := arguments[0].(*treeInterpreter)
	values := arguments[1].([]any)
	expression := arguments[2].(expRef)
	if len(values) == 0 {
		return []any{}, nil
	}
	first, err := intr.Execute(expression.ref, values[0])
	if err != nil {
		return nil, err
	}
	if err := comparableKey(first); err != nil {
		return nil, err
	}
	firstNumber := isNumber(first)
	type keyedValue struct {
		value any
		key   any
	}
	keyed := make([]keyedValue, len(values))
	// Python sorted evaluates keys in source order before any comparisons;
	// the first key is visited again after the required-type observation.
	for i, value := range values {
		key, err := intr.Execute(expression.ref, value)
		if err != nil {
			return nil, err
		}
		if err := comparableKey(key); err != nil {
			return nil, err
		}
		if isNumber(key) != firstNumber {
			return nil, fmt.Errorf("sort_by key types must match the first key")
		}
		keyed[i] = keyedValue{value: value, key: key}
	}
	var comparisonError error
	sort.SliceStable(keyed, func(i, j int) bool {
		comparison, _, err := orderedCompare(keyed[i].key, keyed[j].key)
		if err != nil {
			comparisonError = err
			return false
		}
		return comparison < 0
	})
	if comparisonError != nil {
		return nil, comparisonError
	}
	result := make([]any, len(keyed))
	for i, item := range keyed {
		result[i] = item.value
	}
	return result, nil
}

func jpfJoin(arguments []any) (any, error) {
	stringsArray, _ := toArrayStr(arguments[1])
	return strings.Join(stringsArray, arguments[0].(string)), nil
}

func jpfReverse(arguments []any) (any, error) {
	if text, ok := arguments[0].(string); ok {
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), nil
	}
	values := arguments[0].([]any)
	result := make([]any, len(values))
	for i, value := range values {
		result[len(values)-1-i] = value
	}
	return result, nil
}

func jpfToArray(arguments []any) (any, error) {
	if _, ok := arguments[0].([]any); ok {
		return arguments[0], nil
	}
	return []any{arguments[0]}, nil
}

func jpfToString(arguments []any) (any, error) {
	if text, ok := arguments[0].(string); ok {
		return text, nil
	}
	// Match the selected Python reference's compact ASCII JSON string form;
	// decimal spellings remain exact json.Number and object keys are lexical.
	var buffer strings.Builder
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(arguments[0]); err != nil {
		return nil, err
	}
	text := strings.TrimSuffix(buffer.String(), "\n")
	var result strings.Builder
	for _, r := range text {
		if r < 127 {
			result.WriteRune(r)
			continue
		}
		if r <= 0xffff {
			fmt.Fprintf(&result, "\\u%04x", r)
		} else {
			first, second := utf16.EncodeRune(r)
			fmt.Fprintf(&result, "\\u%04x\\u%04x", first, second)
		}
	}
	return result.String(), nil
}

func jpfToNumber(arguments []any) (any, error) {
	if number, ok := arguments[0].(json.Number); ok {
		if _, err := parseNumber(number); err != nil {
			return nil, err
		}
		return number, nil
	}
	text, ok := arguments[0].(string)
	if !ok {
		return nil, nil
	}
	text = strings.TrimSpace(text)
	if integer, ok := pythonIntegerText(text); ok {
		value := new(big.Int)
		if _, ok := value.SetString(integer, 10); ok {
			return json.Number(value.String()), nil
		}
	}
	text = pythonNumericDigits(text)
	finiteWord := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(text, "+"), "-"))
	if finiteWord == "nan" || finiteWord == "inf" || finiteWord == "infinity" {
		return nil, fmt.Errorf("to_number produced a nonfinite JSON number")
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			if character != '.' && character != 'e' && character != 'E' && character != '+' && character != '-' {
				return nil, nil
			}
		}
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		if numericError, ok := err.(*strconv.NumError); ok && numericError.Err == strconv.ErrRange {
			return nil, fmt.Errorf("to_number is outside finite IEEE-754 arithmetic: %w", err)
		}
		return nil, nil
	}
	return numberFromFloat(value)
}

// Python int/float accept Unicode decimal digits and separators between
// digits. Numeric string conversion remains distinct from JSON literals.
func pythonIntegerText(text string) (string, bool) {
	runes := []rune(text)
	if len(runes) == 0 {
		return "", false
	}
	var result strings.Builder
	index := 0
	if runes[0] == '+' || runes[0] == '-' {
		result.WriteRune(runes[0])
		index++
	}
	previousDigit := false
	for ; index < len(runes); index++ {
		if digit, ok := decimalDigit(runes[index]); ok {
			result.WriteByte('0' + byte(digit))
			previousDigit = true
			continue
		}
		if runes[index] == '_' && previousDigit && index+1 < len(runes) {
			if _, ok := decimalDigit(runes[index+1]); ok {
				previousDigit = false
				continue
			}
		}
		return "", false
	}
	return result.String(), previousDigit
}

func pythonNumericDigits(text string) string {
	runes := []rune(text)
	var result strings.Builder
	for i, r := range runes {
		if digit, ok := decimalDigit(r); ok {
			result.WriteByte('0' + byte(digit))
			continue
		}
		if r == '_' && i > 0 && i+1 < len(runes) {
			_, before := decimalDigit(runes[i-1])
			_, after := decimalDigit(runes[i+1])
			if before && after {
				continue
			}
		}
		result.WriteRune(r)
	}
	return result.String()
}

func decimalDigit(value rune) (int, bool) {
	for _, span := range unicode.Digit.R16 {
		if value >= rune(span.Lo) && value <= rune(span.Hi) && (value-rune(span.Lo))%rune(span.Stride) == 0 {
			return int((value-rune(span.Lo))/rune(span.Stride)) % 10, true
		}
	}
	for _, span := range unicode.Digit.R32 {
		if value >= rune(span.Lo) && value <= rune(span.Hi) && (value-rune(span.Lo))%rune(span.Stride) == 0 {
			return int((value-rune(span.Lo))/rune(span.Stride)) % 10, true
		}
	}
	return 0, false
}

func jpfNotNull(arguments []any) (any, error) {
	for _, value := range arguments {
		if value != nil {
			return value, nil
		}
	}
	return nil, nil
}
