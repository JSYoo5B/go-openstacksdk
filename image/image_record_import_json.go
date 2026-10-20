package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

func imageRecordImportCheck(ctx context.Context, check func(context.Context) error) error {
	if check != nil {
		return check(ctx)
	}
	return nil
}

func captureImageRecordImportJSON(ctx context.Context, check func(context.Context) error, value any) (json.RawMessage, error) {
	if err := imageRecordImportCheck(ctx, check); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	err = errors.Join(err, imageRecordImportCheck(ctx, check))
	if err != nil {
		return nil, errors.Join(uploadInvalid("image import value cannot be encoded"), err)
	}
	// encoding/json replaces invalid Go string bytes. After successful encoding
	// (which has already rejected cycles), inspect ordinary strings as well as
	// the encoded document so invalid input never selects replacement text.
	if !imageRecordImportGoStrings(reflect.ValueOf(value), make(map[imageRecordImportJSONVisit]struct{})) {
		return nil, uploadInvalid("image import strings must be UTF-8")
	}
	if err := validateImageRecordImportJSON(raw); err != nil {
		return nil, err
	}
	return bytes.Clone(raw), nil
}

type imageRecordImportJSONVisit struct {
	kind    reflect.Kind
	typ     reflect.Type
	pointer uintptr
}

func imageRecordImportGoStrings(value reflect.Value, seen map[imageRecordImportJSONVisit]struct{}) bool {
	if !value.IsValid() {
		return true
	}
	if value.CanInterface() {
		if _, ok := value.Interface().(json.Marshaler); ok {
			return true
		}
	}
	switch value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		if value.IsNil() {
			return true
		}
		visit := imageRecordImportJSONVisit{kind: value.Kind(), typ: value.Type(), pointer: uintptr(value.UnsafePointer())}
		if _, already := seen[visit]; already {
			return true
		}
		seen[visit] = struct{}{}
	}
	switch value.Kind() {
	case reflect.String:
		return utf8.ValidString(value.String())
	case reflect.Pointer, reflect.Interface:
		return value.IsNil() || imageRecordImportGoStrings(value.Elem(), seen)
	case reflect.Array, reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			if !imageRecordImportGoStrings(value.Index(index), seen) {
				return false
			}
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			if !imageRecordImportGoStrings(iter.Key(), seen) || !imageRecordImportGoStrings(iter.Value(), seen) {
				return false
			}
		}
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			if field.PkgPath != "" || strings.Split(field.Tag.Get("json"), ",")[0] == "-" {
				continue
			}
			if !imageRecordImportGoStrings(value.Field(index), seen) {
				return false
			}
		}
	}
	return true
}

// Reject invalid UTF-8 and unpaired escaped surrogates before any JSON decoder
// can replace them. This applies to nested IDs and extension strings as well.
func validateImageRecordImportJSON(raw json.RawMessage) error {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return uploadInvalid("image import values must be complete UTF-8 JSON")
	}
	for index := 0; index < len(raw); index++ {
		if raw[index] != '"' {
			continue
		}
		start := index
		for index++; index < len(raw); index++ {
			if raw[index] == '\\' {
				index++
				continue
			}
			if raw[index] == '"' {
				break
			}
		}
		if !imageRecordTagUnicodeString(raw[start : index+1]) {
			return uploadInvalid("image import strings must not contain unpaired UTF-16 surrogates")
		}
	}
	return nil
}

func copyImageRecordImportStore(ctx context.Context, check func(context.Context) error, value ImageRecordImportStore) (ImageRecordImportStore, error) {
	if err := imageRecordImportCheck(ctx, check); err != nil {
		return value, err
	}
	value.RawID = bytes.Clone(value.RawID)
	value.identity = bytes.Clone(value.identity)
	value.capturedRawID = bytes.Clone(value.capturedRawID)
	value.capturedAttributes = bytes.Clone(value.capturedAttributes)
	choices := 0
	if value.ID != "" {
		choices++
	}
	if value.Record != nil {
		choices++
	}
	if value.RawID != nil {
		choices++
	}
	if value.Attributes != nil {
		choices++
	}
	if choices > 1 {
		return value, uploadInvalid("select only one image import store ID, Record, RawID or Attributes")
	}
	if !utf8.ValidString(value.ID) {
		return value, uploadInvalid("image import store ID must be UTF-8")
	}
	var attributes json.RawMessage
	var err error
	if value.Attributes != nil {
		value.Attributes, err = captureImageRecordImportMap(ctx, check, value.Attributes, nil)
		if err != nil {
			return value, err
		}
		attributes, err = json.Marshal(value.Attributes)
		if err != nil {
			return value, err
		}
	}
	if value.RawID != nil {
		if err := validateImageRecordImportJSON(value.RawID); err != nil {
			return value, err
		}
	}
	// Reusing a captured selector never consults caller-edited public views or a
	// record pointer whose complete value the caller later overwrote.
	same := value.captured && value.ID == value.capturedID && value.Record == value.capturedRecord &&
		bytes.Equal(value.RawID, value.capturedRawID) && (value.RawID == nil) == (value.capturedRawID == nil) &&
		bytes.Equal(attributes, value.capturedAttributes) && (attributes == nil) == (value.capturedAttributes == nil)
	if same {
		return value, imageRecordImportCheck(ctx, check)
	}
	switch {
	case value.Record != nil:
		value.identity, err = value.Record.ImportIdentity()
	case value.RawID != nil:
		value.identity = bytes.Clone(value.RawID)
	case value.Attributes != nil:
		value.identity, err = serviceinfo.StoreImportIdentity(attributes)
	default:
		value.identity, err = json.Marshal(value.ID)
	}
	if err = errors.Join(err, imageRecordImportCheck(ctx, check)); err != nil {
		return value, err
	}
	if err := validateImageRecordImportJSON(value.identity); err != nil {
		return value, err
	}
	value.identity = bytes.Clone(value.identity)
	value.capturedID, value.capturedRecord, value.captured = value.ID, value.Record, true
	value.capturedRawID, value.capturedAttributes = bytes.Clone(value.RawID), bytes.Clone(attributes)
	return value, nil
}

func compileImageRecordImport(config ImageRecordImportOpts) (map[string]any, map[string]string, error) {
	methodName := config.Method
	if methodName == nil {
		methodName = json.RawMessage(`"glance-direct"`)
	}
	method := make(map[string]any, len(config.MethodFields)+5)
	for key, value := range config.MethodFields {
		method[key] = value
	}
	method["name"] = json.RawMessage(bytes.Clone(methodName))
	uri, err := cloudfilter.PythonTruthy(config.URI)
	if err != nil {
		return nil, nil, err
	}
	if uri {
		text, decodeErr := decodeImageRecordString(methodName, "image import method")
		if decodeErr != nil || text != "web-download" {
			return nil, nil, uploadInvalid("URI is only supported with image import method web-download")
		}
		method["uri"] = json.RawMessage(bytes.Clone(config.URI))
	}
	remote := true
	for _, value := range []json.RawMessage{config.RemoteRegion, config.RemoteImageID, config.RemoteServiceInterface} {
		truthy, err := cloudfilter.PythonTruthy(value)
		if err != nil {
			return nil, nil, err
		}
		remote = remote && truthy
	}
	if remote {
		method["glance_region"], method["glance_image_id"], method["glance_service_interface"] =
			json.RawMessage(bytes.Clone(config.RemoteRegion)), json.RawMessage(bytes.Clone(config.RemoteImageID)), json.RawMessage(bytes.Clone(config.RemoteServiceInterface))
	}
	if err := validateImageRecordImportStores(config); err != nil {
		return nil, nil, err
	}
	body := make(map[string]any, len(config.Fields)+4)
	for key, value := range config.Fields {
		body[key] = value
	}
	body["method"] = method
	for key, value := range map[string]json.RawMessage{"all_stores": config.AllStores, "all_stores_must_succeed": config.AllStoresMustSucceed} {
		if value != nil && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			body[key] = json.RawMessage(bytes.Clone(value))
		}
	}
	headers := copyImageRecordHeaders(config.Headers)
	stores := config.Stores
	if config.Store != nil {
		text, err := decodeImageRecordString(config.Store.identity, "singular image import store identity")
		if err != nil {
			return nil, nil, err
		}
		if !downloadHeaderValue(text) {
			return nil, nil, uploadInvalid("singular image import store identity must be a valid HTTP header value")
		}
		headers["X-Image-Meta-Store"] = text
		stores = []ImageRecordImportStore{*config.Store}
	}
	if len(stores) > 0 {
		identities := make([]json.RawMessage, len(stores))
		for index := range stores {
			identities[index] = bytes.Clone(stores[index].identity)
		}
		body["stores"] = identities
	}
	return body, headers, nil
}

func validateImageRecordImportStores(config ImageRecordImportOpts) error {
	allStores, err := cloudfilter.PythonTruthy(config.AllStores)
	if err != nil {
		return err
	}
	if allStores && (config.Store != nil || len(config.Stores) > 0) {
		return uploadInvalid("all_stores is mutually exclusive with Store and Stores")
	}
	if config.Store != nil && len(config.Stores) > 0 {
		return uploadInvalid("Store and Stores are mutually exclusive")
	}
	return nil
}
