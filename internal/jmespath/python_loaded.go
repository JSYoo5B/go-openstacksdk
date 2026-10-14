package jmespath

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// SearchLoaded rounds finite floating JSON data and literals as CPython JSON
// loading does, while retaining integer precision. Search remains exact decimal.
// Exact binary operands preserve mixed integer/float comparisons.
// Arithmetic, serialization and the nonfinite namespace are separate contracts.
func SearchLoaded(expression string, data any) (any, error) {
	ast, err := NewParser().Parse(expression)
	if err != nil {
		return nil, err
	}
	ast, err = loadedAST(ast)
	if err != nil {
		return nil, err
	}
	owned, err := loadedNumbers(data)
	if err != nil {
		return nil, err
	}
	return newInterpreter().Execute(ast, owned)
}
func loadedAST(n ASTNode) (ASTNode, error) {
	if n.nodeType == ASTLiteral {
		v, err := loadedNumbers(n.value)
		if err != nil {
			return ASTNode{}, err
		}
		n.value = v
	}
	children := make([]ASTNode, len(n.children))
	for i, c := range n.children {
		v, err := loadedAST(c)
		if err != nil {
			return ASTNode{}, err
		}
		children[i] = v
	}
	n.children = children
	return n, nil
}
func loadedNumbers(v any) (any, error) {
	switch value := v.(type) {
	case json.Number:
		if !strings.ContainsAny(string(value), ".eE") {
			return value, nil
		}
		number, err := strconv.ParseFloat(string(value), 64)
		if math.IsInf(number, 0) || math.IsNaN(number) || err != nil {
			return nil, fmt.Errorf("Python-loaded nonfinite number is unsupported: %s", value)
		}
		// A shortest decimal can equal a large integer even when its binary
		// float does not. Preserve the exact finite binary value for comparison.
		rational := new(big.Rat).SetFloat64(number)
		scale := rational.Denom().BitLen() - 1
		text := rational.FloatString(scale)
		if strings.Contains(text, ".") {
			text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
		}
		return json.Number(text), nil
	case []any:
		out := make([]any, len(value))
		for i, x := range value {
			v, err := loadedNumbers(x)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, x := range value {
			v, err := loadedNumbers(x)
			if err != nil {
				return nil, err
			}
			out[key] = v
		}
		return out, nil
	default:
		return v, nil
	}
}
