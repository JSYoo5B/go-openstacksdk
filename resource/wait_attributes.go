package resource

import (
	"fmt"
	"reflect"
	"strings"
)

// ValidateWaitOptionsFor checks model attributes before a workflow performs a
// mutating request. Service bindings still own the default status predicate.
func ValidateWaitOptionsFor[T any](opts ...WaitOption) error {
	o, err := parseWait(opts)
	if err != nil {
		return err
	}
	_, _, err = waitFields[T](o)
	return err
}

func waitFields[T any](o waitOptions) (status, progress *reflect.StructField, err error) {
	model := reflect.TypeFor[T]()
	if o.statusAttribute != "" {
		status, err = waitField(model, o.statusAttribute)
		if err != nil {
			return nil, nil, err
		}
		if status == nil || indirectType(status.Type).Kind() != reflect.String {
			return nil, nil, fmt.Errorf("%w: status attribute %q must be an exported string field", ErrUnsupported, o.statusAttribute)
		}
	}
	if o.progressCallback != nil {
		progress, err = waitField(model, "progress")
		if err != nil {
			return nil, nil, err
		}
		if progress != nil {
			kind := indirectType(progress.Type).Kind()
			if kind < reflect.Int || kind > reflect.Uint64 {
				return nil, nil, fmt.Errorf("%w: progress must be an exported integer field", ErrUnsupported)
			}
		}
	}
	return status, progress, nil
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func waitField(model reflect.Type, attribute string) (*reflect.StructField, error) {
	model = indirectType(model)
	if model.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: waiter attributes require a struct model", ErrUnsupported)
	}
	var match *reflect.StructField
	for _, field := range reflect.VisibleFields(model) {
		if field.PkgPath != "" {
			continue
		}
		jsonName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if jsonName == "-" {
			continue
		}
		if field.Name != attribute && jsonName != attribute && !(jsonName == "" && strings.EqualFold(field.Name, attribute)) {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("%w: attribute %q is ambiguous in the model", ErrUnsupported, attribute)
		}
		copy := field
		match = &copy
	}
	return match, nil
}

func waitFieldValue[T any](value *T, field *reflect.StructField) reflect.Value {
	if value == nil || field == nil {
		return reflect.Value{}
	}
	v := reflect.ValueOf(value)
	for _, index := range field.Index {
		for v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		v = v.Field(index)
	}
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func waitStatus[T any](value *T, field *reflect.StructField, service func(*T) string) (string, error) {
	if field == nil {
		if service == nil {
			return "", ErrUnsupported
		}
		return service(value), nil
	}
	v := waitFieldValue(value, field)
	if !v.IsValid() {
		return "", fmt.Errorf("%w: status attribute %q is nil", ErrUnsupported, field.Name)
	}
	return v.String(), nil
}

func reportWaitProgress[T any](o waitOptions, value *T, field *reflect.StructField) error {
	if o.progressCallback == nil {
		return nil
	}
	progress := 0
	v := waitFieldValue(value, field)
	if v.IsValid() {
		if v.Kind() >= reflect.Int && v.Kind() <= reflect.Int64 {
			n := v.Int()
			progress = int(n)
			if int64(progress) != n {
				return fmt.Errorf("%w: progress exceeds Go int range", ErrUnsupported)
			}
		} else {
			n := v.Uint()
			progress = int(n)
			if progress < 0 || uint64(progress) != n {
				return fmt.Errorf("%w: progress exceeds Go int range", ErrUnsupported)
			}
		}
	}
	o.progressCallback(progress)
	return nil
}
