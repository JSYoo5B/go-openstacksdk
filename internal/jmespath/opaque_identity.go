package jmespath

import (
	"fmt"
	"reflect"
)

// OpaqueIdentity represents a non-JSON value whose Python identity is observable
// to expressions. It is opt-in: ordinary JSON retains its existing equality.
// The key must be a comparable pointer, never a mutable slice, map or function.
type OpaqueIdentity interface {
	JMESPathIdentity() any
}

func opaqueIdentityKey(value OpaqueIdentity) (any, error) {
	key := value.JMESPathIdentity()
	kind := reflect.TypeOf(key)
	if kind == nil || kind.Kind() != reflect.Pointer || !kind.Comparable() {
		return nil, fmt.Errorf("opaque JMESPath identity must be a comparable pointer")
	}
	return key, nil
}

func opaqueEqual(left, right any) (equal bool, opaque bool, err error) {
	l, leftOpaque := left.(OpaqueIdentity)
	r, rightOpaque := right.(OpaqueIdentity)
	if !leftOpaque && !rightOpaque {
		return false, false, nil
	}
	var leftKey, rightKey any
	if leftOpaque {
		leftKey, err = opaqueIdentityKey(l)
		if err != nil {
			return false, true, err
		}
	}
	if rightOpaque {
		rightKey, err = opaqueIdentityKey(r)
		if err != nil {
			return false, true, err
		}
	}
	if !leftOpaque || !rightOpaque {
		return false, true, nil
	}
	return leftKey == rightKey, true, nil
}
