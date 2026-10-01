package resource

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// IdentityFindOpts configures automatic ID-or-name lookup. A nil IgnoreMissing
// defaults to true. Query contains only service-specific list query fields;
// transport, pagination controls and local filters are not FindIdentity inputs.
type IdentityFindOpts struct {
	IgnoreMissing *bool
	Fallback      FindFallbackPolicy
	Query         url.Values
}

// IdentityFindOption configures an identity lookup. Later options win.
// The SDK copies the resulting configuration before either HTTP phase.
type IdentityFindOption func(*IdentityFindOpts) error

// WithIdentityFindOptions snapshots the supplied options now and takes a fresh
// copy each time the option is applied. A later bulk option replaces all fields.
func WithIdentityFindOptions(value IdentityFindOpts) IdentityFindOption {
	snapshot := cloneIdentityFindOptions(value)
	return func(options *IdentityFindOpts) error {
		*options = cloneIdentityFindOptions(snapshot)
		return nil
	}
}

func WithIdentityFindIgnoreMissing(value bool) IdentityFindOption {
	return func(options *IdentityFindOpts) error {
		copy := value
		options.IgnoreMissing = &copy
		return nil
	}
}

func WithIdentityFindFallback(value FindFallbackPolicy) IdentityFindOption {
	return func(options *IdentityFindOpts) error {
		options.Fallback = value
		return nil
	}
}

// WithIdentityFindQuery supplies a wire query field to the list phase only.
// An explicit name-query key takes precedence over the automatic name hint;
// local matching still compares the original identity against each ID or name.
func WithIdentityFindQuery(key, value string) IdentityFindOption {
	return func(options *IdentityFindOpts) error {
		if err := validateIdentityFindQueryKey(key); err != nil {
			return err
		}
		if options.Query == nil {
			options.Query = make(url.Values)
		}
		options.Query.Set(key, value)
		return nil
	}
}

func cloneIdentityFindOptions(value IdentityFindOpts) IdentityFindOpts {
	copy := value
	if value.IgnoreMissing != nil {
		ignored := *value.IgnoreMissing
		copy.IgnoreMissing = &ignored
	}
	copy.Query = make(url.Values, len(value.Query))
	for key, values := range value.Query {
		copy.Query[key] = append([]string(nil), values...)
	}
	return copy
}

func parseIdentityFindOptions(options []IdentityFindOption) (IdentityFindOpts, error) {
	value := IdentityFindOpts{Query: make(url.Values)}
	for _, apply := range append([]IdentityFindOption(nil), options...) {
		if apply == nil {
			return IdentityFindOpts{}, invalid("nil identity find option")
		}
		if err := apply(&value); err != nil {
			return IdentityFindOpts{}, err
		}
	}
	// A custom option may retain the pointer or supply caller-owned maps. Do not
	// let a GET callback mutate the configuration of the later list phase.
	snapshot := cloneIdentityFindOptions(value)
	if snapshot.Fallback < FindFallbackCompatible || snapshot.Fallback > FindFallbackNever {
		return IdentityFindOpts{}, invalid("unsupported identity fallback policy")
	}
	for key := range snapshot.Query {
		if err := validateIdentityFindQueryKey(key); err != nil {
			return IdentityFindOpts{}, err
		}
	}
	return snapshot, nil
}

func validateIdentityFindQueryKey(key string) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return invalid("identity find query key must be nonempty UTF-8")
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return invalid("identity find query key must not contain control characters")
		}
	}
	switch strings.ToLower(key) {
	case "max_items", "paginated", "base_path", "list_base_path", "jmespath_filters", "headers", "microversion", "allow_unknown_params", "ignore_missing", "fallback":
		return invalid("identity find query %q is an SDK control", key)
	}
	return nil
}
