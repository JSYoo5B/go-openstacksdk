package request

import (
	"encoding"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"gophercloudsdk/resource"
)

// QueryOptions is used by SDK-owned collection bindings when an upstream API
// accepts concrete list inputs without an extension builder. Unknown fields are
// rejected rather than silently ignored. Callers normally use typed WithOptions.
func QueryOptions[T any](query url.Values) (T, error) {
	var result T
	value := reflect.ValueOf(&result).Elem()
	if value.Kind() == reflect.Pointer {
		value.Set(reflect.New(value.Type().Elem()))
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return result, fmt.Errorf("%w: list input is not a struct", resource.ErrUnsupported)
	}
	fields := map[string]reflect.Value{}
	var visit func(reflect.Value)
	visit = func(value reflect.Value) {
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if field.Anonymous {
				nested := value.Field(i)
				if nested.Kind() == reflect.Pointer && nested.Type().Elem().Kind() == reflect.Struct {
					nested.Set(reflect.New(nested.Type().Elem()))
					nested = nested.Elem()
				}
				if nested.Kind() == reflect.Struct {
					visit(nested)
				}
			}
			key := strings.Split(field.Tag.Get("q"), ",")[0]
			if key != "" && key != "-" && value.Field(i).CanSet() {
				fields[key] = value.Field(i)
			}
		}
	}
	visit(value)
	for key, values := range query {
		field, ok := fields[key]
		if !ok {
			return result, fmt.Errorf("%w: unsupported list filter %q", resource.ErrUnsupported, key)
		}
		if err := setQueryField(field, values); err != nil {
			return result, fmt.Errorf("%w: filter %q: %v", resource.ErrInvalidOption, key, err)
		}
	}
	return result, nil
}

func setQueryField(field reflect.Value, values []string) error {
	if field.Kind() == reflect.Pointer {
		field.Set(reflect.New(field.Type().Elem()))
		field = field.Elem()
	}
	text := ""
	if len(values) > 0 {
		text = values[len(values)-1]
	}
	if field.CanAddr() {
		if decoder, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return decoder.UnmarshalText([]byte(text))
		}
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString(text)
	case reflect.Bool:
		value, err := strconv.ParseBool(text)
		if err != nil {
			return err
		}
		field.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(text, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(text, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(value)
	case reflect.Slice:
		if field.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported filter type %s", field.Type())
		}
		value := reflect.MakeSlice(field.Type(), len(values), len(values))
		for i, text := range values {
			value.Index(i).SetString(text)
		}
		field.Set(value)
	default:
		return fmt.Errorf("unsupported filter type %s", field.Type())
	}
	return nil
}
