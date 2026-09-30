package main

import (
	"go/types"
	"strings"
)

func extractionMethod(t types.Type, name string) *types.Signature {
	object, _, _ := types.LookupFieldOrMethod(t, true, nil, name)
	if object == nil {
		object, _, _ = types.LookupFieldOrMethod(types.NewPointer(t), true, nil, name)
	}
	if fn, ok := object.(*types.Func); ok {
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() == 0 && sig.Results().Len() == 2 && isError(sig.Results().At(1).Type()) {
			return sig
		}
	}
	return nil
}

// Select a named extractor only when it matches the operation or is unambiguous.
// Shared QoS result types expose multiple extractors for different rule models.
func operationExtractor(fn *types.Func) (string, *types.Signature) {
	sig := fn.Type().(*types.Signature)
	if sig.Results().Len() != 1 {
		return "", nil
	}
	result := sig.Results().At(0).Type()
	if ex := extraction(result); ex != nil {
		return "Extract", ex
	}
	for _, verb := range []string{"Create", "Get", "Update", "Find", "List", "Set"} {
		if strings.HasPrefix(fn.Name(), verb) {
			name := "Extract" + strings.TrimPrefix(fn.Name(), verb)
			if ex := extractionMethod(result, name); ex != nil {
				return name, ex
			}
		}
	}
	var name string
	var found *types.Signature
	methods := types.NewMethodSet(types.NewPointer(result))
	for i := 0; i < methods.Len(); i++ {
		candidate := methods.At(i).Obj().Name()
		if !strings.HasPrefix(candidate, "Extract") {
			continue
		}
		if ex := extractionMethod(result, candidate); ex != nil {
			if found != nil {
				return "", nil
			}
			name, found = candidate, ex
		}
	}
	return name, found
}

func operationReturnPolicy(fn *types.Func) string {
	policy := returnPolicy(fn.Type().(*types.Signature))
	if policy == "result" {
		if _, ex := operationExtractor(fn); ex != nil {
			return "extract"
		}
	}
	return policy
}
