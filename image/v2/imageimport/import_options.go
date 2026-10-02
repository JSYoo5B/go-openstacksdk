package imageimport

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"unicode/utf8"

	"gophercloudsdk/resource"
)

const (
	GlanceDownloadMethod ImportMethod = "glance-download"
	CopyImageMethod      ImportMethod = "copy-image"
)

// ImportOpts owns the import POST body. Empty Method selects glance-direct.
// Nil booleans are omitted; explicit false is sent. Store selects the legacy
// header, while Stores selects the root JSON array. Fields and MethodFields
// extend their respective JSON objects without overriding built-in fields.
type ImportOpts struct {
	Method                 ImportMethod
	URI                    string
	RemoteRegion           string
	RemoteImageID          string
	RemoteServiceInterface string
	Store                  *string
	Stores                 []string
	AllStores              *bool
	AllStoresMustSucceed   *bool
	Headers                map[string]string
	Fields                 map[string]any
	MethodFields           map[string]any
}

// ImportOption configures the SDK-owned workflow. Later options replace earlier values.
type ImportOption func(*ImportOpts) error

// WithImportOpts snapshots and replaces the complete options value.
func WithImportOpts(value ImportOpts) ImportOption {
	snapshot, err := copyImportOpts(value)
	return func(config *ImportOpts) error {
		if err != nil {
			return err
		}
		owned, err := copyImportOpts(snapshot)
		if err == nil {
			*config = owned
		}
		return err
	}
}

func WithImportMethod(value ImportMethod) ImportOption {
	return func(config *ImportOpts) error { config.Method = value; return nil }
}
func WithImportURI(value string) ImportOption {
	return func(config *ImportOpts) error { config.URI = value; return nil }
}
func WithImportRemoteRegion(value string) ImportOption {
	return func(config *ImportOpts) error { config.RemoteRegion = value; return nil }
}
func WithImportRemoteImageID(value string) ImportOption {
	return func(config *ImportOpts) error { config.RemoteImageID = value; return nil }
}
func WithImportRemoteServiceInterface(value string) ImportOption {
	return func(config *ImportOpts) error { config.RemoteServiceInterface = value; return nil }
}
func WithImportStore(value string) ImportOption {
	return func(config *ImportOpts) error { owned := value; config.Store = &owned; return nil }
}
func WithImportStores(values ...string) ImportOption {
	snapshot := append(make([]string, 0, len(values)), values...)
	return func(config *ImportOpts) error {
		config.Stores = append(make([]string, 0, len(snapshot)), snapshot...)
		return nil
	}
}
func WithImportAllStores(value bool) ImportOption {
	return func(config *ImportOpts) error { owned := value; config.AllStores = &owned; return nil }
}
func WithImportAllStoresMustSucceed(value bool) ImportOption {
	return func(config *ImportOpts) error { owned := value; config.AllStoresMustSucceed = &owned; return nil }
}

// WithImportHeader adds an ordinary header. Case-insensitive later options win.
// Authentication, transport, version and store selection remain SDK-owned.
func WithImportHeader(key, value string) ImportOption {
	return func(config *ImportOpts) error {
		canonical, err := importHeaders(map[string]string{key: value}, false, "")
		if err != nil {
			return err
		}
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		for existing := range config.Headers {
			if strings.EqualFold(existing, key) {
				delete(config.Headers, existing)
			}
		}
		for key, value := range canonical {
			config.Headers[key] = value
		}
		return nil
	}
}

// WithImportHeaders snapshots and merges headers, rejecting conflicting case aliases.
func WithImportHeaders(headers map[string]string) ImportOption {
	snapshot := maps.Clone(headers)
	return func(config *ImportOpts) error {
		canonical, err := importHeaders(snapshot, false, "")
		if err != nil {
			return err
		}
		for key, value := range canonical {
			if err := WithImportHeader(key, value)(config); err != nil {
				return err
			}
		}
		return nil
	}
}

func WithImportField(key string, value any) ImportOption {
	return withImportField(key, value, false)
}
func WithImportMethodField(key string, value any) ImportOption {
	return withImportField(key, value, true)
}
func withImportField(key string, value any, method bool) ImportOption {
	snapshot, err := importJSONValue(value)
	return func(config *ImportOpts) error {
		if err != nil {
			return err
		}
		if err := importFieldKey(key, method); err != nil {
			return err
		}
		fields := &config.Fields
		if method {
			fields = &config.MethodFields
		}
		if *fields == nil {
			*fields = make(map[string]any)
		}
		(*fields)[key] = append(json.RawMessage(nil), snapshot...)
		return nil
	}
}

func copyImportOpts(value ImportOpts) (ImportOpts, error) {
	if value.Store != nil {
		owned := *value.Store
		value.Store = &owned
	}
	if value.AllStores != nil {
		owned := *value.AllStores
		value.AllStores = &owned
	}
	if value.AllStoresMustSucceed != nil {
		owned := *value.AllStoresMustSucceed
		value.AllStoresMustSucceed = &owned
	}
	if value.Stores != nil {
		value.Stores = append(make([]string, 0, len(value.Stores)), value.Stores...)
	}
	value.Headers = maps.Clone(value.Headers)
	var err error
	value.Fields, err = copyImportFields(value.Fields)
	if err == nil {
		value.MethodFields, err = copyImportFields(value.MethodFields)
	}
	return value, err
}

func copyImportFields(fields map[string]any) (map[string]any, error) {
	if fields == nil {
		return nil, nil
	}
	result := make(map[string]any, len(fields))
	for key, value := range fields {
		body, err := importJSONValue(value)
		if err != nil {
			return nil, err
		}
		result[key] = body
	}
	return result, nil
}

func importJSONValue(value any) (json.RawMessage, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: import extension: %w", resource.ErrInvalidOption, err)
	}
	if !utf8.Valid(body) {
		return nil, importInvalid("import extension must be valid UTF-8")
	}
	return append(json.RawMessage(nil), body...), nil
}

func parseImportOpts(options []ImportOption) (ImportOpts, error) {
	var config ImportOpts
	for _, apply := range options {
		if apply == nil {
			return config, importInvalid("nil import option")
		}
		if err := apply(&config); err != nil {
			return config, err
		}
	}
	// A custom option can retain config. Return a separate owned snapshot.
	snapshot, err := copyImportOpts(config)
	if err != nil {
		return snapshot, err
	}
	if snapshot.Method == "" {
		snapshot.Method = GlanceDirectMethod
	}
	for _, value := range []string{string(snapshot.Method), snapshot.URI, snapshot.RemoteRegion,
		snapshot.RemoteImageID, snapshot.RemoteServiceInterface} {
		if !utf8.ValidString(value) {
			return snapshot, importInvalid("import values must be valid UTF-8")
		}
	}
	if snapshot.Method == WebDownloadMethod {
		if snapshot.URI == "" {
			return snapshot, importInvalid("web-download requires URI")
		}
	} else if snapshot.URI != "" {
		return snapshot, importInvalid("URI requires web-download")
	}
	remote := snapshot.RemoteRegion != "" || snapshot.RemoteImageID != "" || snapshot.RemoteServiceInterface != ""
	if snapshot.Method == GlanceDownloadMethod {
		if snapshot.RemoteRegion == "" || snapshot.RemoteImageID == "" {
			return snapshot, importInvalid("glance-download requires remote region and image ID")
		}
	} else if remote {
		return snapshot, importInvalid("remote values require glance-download")
	}
	if snapshot.Store != nil {
		if *snapshot.Store == "" || !utf8.ValidString(*snapshot.Store) {
			return snapshot, importInvalid("store must be nonempty valid UTF-8")
		}
		if len(snapshot.Stores) > 0 {
			return snapshot, importInvalid("Store and Stores are mutually exclusive")
		}
		// The legacy value becomes an HTTP header, rather than a URL segment.
		if _, err := importHeaders(map[string]string{"X-Image-Meta-Store": *snapshot.Store}, true, ""); err != nil {
			return snapshot, err
		}
	}
	for _, value := range snapshot.Stores {
		if value == "" || !utf8.ValidString(value) {
			return snapshot, importInvalid("stores must be nonempty valid UTF-8")
		}
	}
	if snapshot.AllStores != nil && *snapshot.AllStores && (snapshot.Store != nil || len(snapshot.Stores) > 0) {
		return snapshot, importInvalid("AllStores is mutually exclusive with Store and Stores")
	}
	for key := range snapshot.Fields {
		if err := importFieldKey(key, false); err != nil {
			return snapshot, err
		}
	}
	for key := range snapshot.MethodFields {
		if err := importFieldKey(key, true); err != nil {
			return snapshot, err
		}
	}
	snapshot.Headers, err = importHeaders(snapshot.Headers, false, "")
	return snapshot, err
}

func importFieldKey(key string, method bool) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return importInvalid("invalid import extension key")
	}
	alias := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
	if method {
		switch alias {
		case "name", "method", "uri", "glanceregion", "glanceimageid", "glanceserviceinterface", "remoteregion", "remoteimageid", "remoteserviceinterface":
			return importInvalid("method field %q is owned by the SDK", key)
		}
	} else {
		switch alias {
		case "method", "store", "storeid", "stores", "allstores", "allstoresmustsucceed":
			return importInvalid("field %q is owned by the SDK", key)
		}
	}
	return nil
}

func importHeaders(headers map[string]string, source bool, version string) (map[string]string, error) {
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		if key == "" || !utf8.ValidString(value) {
			return nil, importInvalid("invalid import header")
		}
		for _, c := range []byte(key) {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
				continue
			}
			return nil, importInvalid("invalid import header %q", key)
		}
		for _, c := range []byte(value) {
			if (c < 32 && c != '\t') || c == 127 {
				return nil, importInvalid("invalid import header value")
			}
		}
		name := http.CanonicalHeaderKey(key)
		if old, exists := result[name]; exists && old != value {
			return nil, importInvalid("conflicting import header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade", "x-openstack-glance-api-version":
			return nil, importInvalid("header %q is owned by the SDK", key)
		case "x-image-meta-store":
			if !source {
				return nil, importInvalid("store header is owned by Store")
			}
		case "openstack-api-version":
			if !source || version == "" || value != "image "+version {
				return nil, importInvalid("image version header conflicts with selected microversion")
			}
		}
		result[name] = value
	}
	return result, nil
}

func importBody(policy ImportOpts) map[string]any {
	method := maps.Clone(policy.MethodFields)
	if method == nil {
		method = make(map[string]any)
	}
	method["name"] = policy.Method
	if policy.URI != "" {
		method["uri"] = policy.URI
	}
	if policy.Method == GlanceDownloadMethod {
		method["glance_region"], method["glance_image_id"] = policy.RemoteRegion, policy.RemoteImageID
		if policy.RemoteServiceInterface != "" {
			method["glance_service_interface"] = policy.RemoteServiceInterface
		}
	}
	body := maps.Clone(policy.Fields)
	if body == nil {
		body = make(map[string]any)
	}
	body["method"] = method
	if len(policy.Stores) > 0 {
		body["stores"] = append([]string(nil), policy.Stores...)
	}
	if policy.AllStores != nil {
		body["all_stores"] = *policy.AllStores
	}
	if policy.AllStoresMustSucceed != nil {
		body["all_stores_must_succeed"] = *policy.AllStoresMustSucceed
	}
	return body
}

func importInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

func importObject(body json.RawMessage) (map[string]json.RawMessage, error) {
	if !utf8.Valid(body) {
		return nil, fmt.Errorf("response must be valid UTF-8")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("response must be a JSON object")
	}
	return fields, nil
}
