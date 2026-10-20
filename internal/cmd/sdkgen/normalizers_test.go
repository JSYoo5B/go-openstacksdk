package main

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestResultNormalizerRequiresAuditedSourceAndResultType(t *testing.T) {
	newFunction := func(path, name, result string) *types.Func {
		pkg := types.NewPackage(path, "fixture")
		model := types.NewNamed(types.NewTypeName(token.NoPos, pkg, result, nil), types.NewStruct(nil, nil), nil)
		sig := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, pkg, "", model)), false)
		return types.NewFunc(token.NoPos, pkg, name, sig)
	}
	for key, expected := range resultNormalizers {
		separator := strings.LastIndex(key, ".")
		path, name := key[:separator], key[separator+1:]
		fn := newFunction(path, name, expected.resultType)
		if normalizerFor(fn) == nil || operationReturnPolicy(fn) != "normalize" {
			t.Errorf("%s has no response policy", key)
		}
		for _, other := range []*types.Func{
			newFunction(path, name, "DifferentResult"),
			newFunction(path+"/other", name, expected.resultType),
			newFunction(path, name+"Other", expected.resultType),
		} {
			if normalizerFor(other) != nil {
				t.Errorf("unexpected normalizer for %s", other)
			}
		}
	}
}
