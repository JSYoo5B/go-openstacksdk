package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
)

// ImageRecordImportStore selects a literal string ID, raw JSON ID, constructor
// attributes, or the private identity of an SDK-produced StoreRecord. Nonempty
// ID, Record, nonnil RawID and nonnil Attributes are mutually exclusive. The zero
// selector is the empty literal string ID; an empty attributes map constructs a
// null ID. Only canonical id is consumed, with no name fallback or discovery.
// Plural stores retain any JSON ID. Singular Store also becomes an HTTP header
// and therefore requires a complete valid UTF-8 string, including empty text.
type ImageRecordImportStore struct {
	ID         string
	Record     *serviceinfo.StoreRecord
	RawID      json.RawMessage
	Attributes map[string]any

	identity           json.RawMessage
	capturedID         string
	capturedRecord     *serviceinfo.StoreRecord
	capturedRawID      json.RawMessage
	capturedAttributes json.RawMessage
	captured           bool
}

// ImageRecordImportOpts owns all request values. Nil Method defaults to
// glance-direct; raw null is an explicit method name None. Other nil/null values
// are omitted. Falsy URI is omitted; a truthy URI requires exact web-download.
// Remote values are included only when all three are truthy, for any method.
// Fields and MethodFields add JSON extensions without replacing core controls.
type ImageRecordImportOpts struct {
	Method                 json.RawMessage
	URI                    json.RawMessage
	RemoteRegion           json.RawMessage
	RemoteImageID          json.RawMessage
	RemoteServiceInterface json.RawMessage
	AllStores              json.RawMessage
	AllStoresMustSucceed   json.RawMessage
	Store                  *ImageRecordImportStore
	Stores                 []ImageRecordImportStore
	Headers                map[string]string
	Fields                 map[string]any
	MethodFields           map[string]any
}

type ImageRecordImportOption func(*ImageRecordImportOpts) error

// WithImageRecordImportOpts captures values and store identities immediately and
// replaces the complete policy when applied. With helpers may then overlay it.
func WithImageRecordImportOpts(value ImageRecordImportOpts) ImageRecordImportOption {
	snapshot, captureErr := copyImageRecordImportOpts(nil, nil, value)
	return func(config *ImageRecordImportOpts) error {
		if captureErr != nil {
			return captureErr
		}
		owned, err := copyImageRecordImportOpts(nil, nil, snapshot)
		if err == nil {
			*config = owned
		}
		return err
	}
}

func WithImageRecordImportMethod(value any) ImageRecordImportOption {
	return withImageRecordImportValue(value, func(config *ImageRecordImportOpts, raw json.RawMessage) { config.Method = raw })
}
func WithImageRecordImportURI(value any) ImageRecordImportOption {
	return withImageRecordImportValue(value, func(config *ImageRecordImportOpts, raw json.RawMessage) { config.URI = raw })
}
func WithImageRecordImportRemoteRegion(value any) ImageRecordImportOption {
	return withImageRecordImportValue(value, func(config *ImageRecordImportOpts, raw json.RawMessage) { config.RemoteRegion = raw })
}
func WithImageRecordImportRemoteImageID(value any) ImageRecordImportOption {
	return withImageRecordImportValue(value, func(config *ImageRecordImportOpts, raw json.RawMessage) { config.RemoteImageID = raw })
}
func WithImageRecordImportRemoteServiceInterface(value any) ImageRecordImportOption {
	return withImageRecordImportValue(value, func(config *ImageRecordImportOpts, raw json.RawMessage) { config.RemoteServiceInterface = raw })
}
func WithImageRecordImportAllStores(value any) ImageRecordImportOption {
	return withImageRecordImportValue(value, func(config *ImageRecordImportOpts, raw json.RawMessage) { config.AllStores = raw })
}
func WithImageRecordImportAllStoresMustSucceed(value any) ImageRecordImportOption {
	return withImageRecordImportValue(value, func(config *ImageRecordImportOpts, raw json.RawMessage) { config.AllStoresMustSucceed = raw })
}

func withImageRecordImportValue(value any, set func(*ImageRecordImportOpts, json.RawMessage)) ImageRecordImportOption {
	snapshot, captureErr := captureImageRecordImportJSON(nil, nil, value)
	return func(config *ImageRecordImportOpts) error {
		if captureErr != nil {
			return captureErr
		}
		set(config, bytes.Clone(snapshot))
		return nil
	}
}

func WithImageRecordImportStore(value ImageRecordImportStore) ImageRecordImportOption {
	snapshot, captureErr := copyImageRecordImportStore(nil, nil, value)
	return func(config *ImageRecordImportOpts) error {
		if captureErr != nil {
			return captureErr
		}
		owned, err := copyImageRecordImportStore(nil, nil, snapshot)
		if err == nil {
			config.Store = &owned
		}
		return err
	}
}

// WithImageRecordImportStores replaces the complete plural selection. An empty
// invocation clears it; it does not clear a separately selected singular Store.
func WithImageRecordImportStores(values ...ImageRecordImportStore) ImageRecordImportOption {
	snapshot := slices.Clone(values)
	var captureErr error
	for index := range snapshot {
		snapshot[index], captureErr = copyImageRecordImportStore(nil, nil, snapshot[index])
		if captureErr != nil {
			break
		}
	}
	return func(config *ImageRecordImportOpts) error {
		if captureErr != nil {
			return captureErr
		}
		owned := slices.Clone(snapshot)
		for index := range owned {
			var err error
			owned[index], err = copyImageRecordImportStore(nil, nil, owned[index])
			if err != nil {
				return err
			}
		}
		config.Stores = owned
		return nil
	}
}

func WithImageRecordImportHeader(key, value string) ImageRecordImportOption {
	return func(config *ImageRecordImportOpts) error {
		return mergeImageRecordImportHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordImportHeaders(values map[string]string) ImageRecordImportOption {
	snapshot := copyImageRecordHeaders(values)
	return func(config *ImageRecordImportOpts) error {
		return mergeImageRecordImportHeaders(&config.Headers, snapshot)
	}
}
func WithImageRecordImportField(key string, value any) ImageRecordImportOption {
	return WithImageRecordImportFields(map[string]any{key: value})
}
func WithImageRecordImportFields(values map[string]any) ImageRecordImportOption {
	return withImageRecordImportFields(values, false)
}
func WithImageRecordImportMethodField(key string, value any) ImageRecordImportOption {
	return WithImageRecordImportMethodFields(map[string]any{key: value})
}
func WithImageRecordImportMethodFields(values map[string]any) ImageRecordImportOption {
	return withImageRecordImportFields(values, true)
}

func withImageRecordImportFields(values map[string]any, method bool) ImageRecordImportOption {
	snapshot, captureErr := captureImageRecordImportMap(nil, nil, values, func(key string) error { return imageRecordImportFieldKey(key, method) })
	return func(config *ImageRecordImportOpts) error {
		if captureErr != nil {
			return captureErr
		}
		target := &config.Fields
		if method {
			target = &config.MethodFields
		}
		if *target == nil {
			*target = make(map[string]any)
		}
		for key, value := range snapshot {
			(*target)[key] = json.RawMessage(bytes.Clone(value.(json.RawMessage)))
		}
		return nil
	}
}

func imageRecordImportFieldKey(key string, method bool) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return uploadInvalid("invalid image import extension key")
	}
	alias := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
	if method {
		switch alias {
		case "name", "method", "uri", "glanceregion", "glanceimageid", "glanceserviceinterface", "remoteregion", "remoteimageid", "remoteserviceinterface":
			return uploadInvalid("image import method field %q is owned by the SDK", key)
		}
	} else {
		switch alias {
		case "method", "store", "storeid", "stores", "allstores", "allstoresmustsucceed":
			return uploadInvalid("image import field %q is owned by the SDK", key)
		}
	}
	return nil
}

func imageRecordImportHeaders(values map[string]string) (map[string]string, error) {
	headers, err := imageMutationHeaders(values, false, "")
	if err != nil {
		return nil, err
	}
	for key := range headers {
		if strings.EqualFold(key, "X-Image-Meta-Store") {
			return nil, uploadInvalid("image import store header is owned by Store")
		}
	}
	return headers, nil
}
func mergeImageRecordImportHeaders(target *map[string]string, values map[string]string) error {
	headers, err := imageRecordImportHeaders(values)
	if err != nil {
		return err
	}
	return mergeImageRecordHeaders(target, headers)
}

func copyImageRecordImportOpts(ctx context.Context, check func(context.Context) error, value ImageRecordImportOpts) (ImageRecordImportOpts, error) {
	value.Headers = copyImageRecordHeaders(value.Headers)
	rawValues := []*json.RawMessage{&value.Method, &value.URI, &value.RemoteRegion, &value.RemoteImageID, &value.RemoteServiceInterface, &value.AllStores, &value.AllStoresMustSucceed}
	for _, target := range rawValues {
		*target = bytes.Clone(*target)
		if *target != nil {
			if err := validateImageRecordImportJSON(*target); err != nil {
				return value, err
			}
		}
	}
	// Clone selector containers before any extension marshaler can run. A store
	// record's captured identity is reused while its selector remains unchanged.
	if value.Store != nil {
		owned := *value.Store
		value.Store = &owned
	}
	value.Stores = slices.Clone(value.Stores)
	var err error
	if value.Store != nil {
		*value.Store, err = copyImageRecordImportStore(ctx, check, *value.Store)
		if err != nil {
			return value, err
		}
	}
	for index := range value.Stores {
		value.Stores[index], err = copyImageRecordImportStore(ctx, check, value.Stores[index])
		if err != nil {
			return value, err
		}
	}
	value.Fields, err = captureImageRecordImportMap(ctx, check, value.Fields, func(key string) error { return imageRecordImportFieldKey(key, false) })
	if err == nil {
		value.MethodFields, err = captureImageRecordImportMap(ctx, check, value.MethodFields, func(key string) error { return imageRecordImportFieldKey(key, true) })
	}
	return value, errors.Join(err, imageRecordImportCheck(ctx, check))
}

func prepareImageRecordImportOptions(ctx context.Context, check func(context.Context) error, options []ImageRecordImportOption) (ImageRecordImportOpts, error) {
	config, err := copyImageRecordImportOpts(ctx, check, ImageRecordImportOpts{})
	if err != nil {
		return config, err
	}
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, uploadInvalid("nil image record import option")
		}
		candidate, err := copyImageRecordImportOpts(ctx, check, config)
		if err != nil {
			return config, err
		}
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return config, err
		}
		config, err = copyImageRecordImportOpts(ctx, check, candidate)
		if err != nil {
			return config, err
		}
	}
	config.Headers, err = imageRecordImportHeaders(config.Headers)
	return config, errors.Join(err, check(ctx))
}

func captureImageRecordImportMap(ctx context.Context, check func(context.Context) error, values map[string]any, validateKey func(string) error) (map[string]any, error) {
	if values == nil {
		return nil, nil
	}
	values = maps.Clone(values)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	owned := make(map[string]any, len(values))
	for _, key := range keys {
		if err := imageRecordImportCheck(ctx, check); err != nil {
			return nil, err
		}
		if !utf8.ValidString(key) {
			return nil, uploadInvalid("image import JSON key must be UTF-8")
		}
		if validateKey != nil {
			if err := validateKey(key); err != nil {
				return nil, err
			}
		}
		raw, err := captureImageRecordImportJSON(ctx, check, values[key])
		if err != nil {
			return nil, err
		}
		owned[key] = raw
	}
	return owned, nil
}
