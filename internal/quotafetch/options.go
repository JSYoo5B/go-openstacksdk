// Package quotafetch owns the common QuotaSet fetch contract. Service profiles
// are SDK data; callers supply concrete options, never a builder or decoder.
package quotafetch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type Options struct {
	Query        map[string]json.RawMessage
	Usage        bool
	RequiresID   bool
	BasePath     *string
	ErrorMessage *string
	SkipCache    bool
	ResponseKey  *string
	Microversion *string
	Location     *resource.CloudLocation
	usageValue   json.RawMessage
}
type Option func(*Options) error

func CloneOptions(v Options) Options {
	v.Query = cloneFields(v.Query)
	v.usageValue = bytes.Clone(v.usageValue)
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	for _, p := range []**string{&v.BasePath, &v.ErrorMessage, &v.ResponseKey, &v.Microversion} {
		if *p != nil {
			x := **p
			*p = &x
		}
	}
	return v
}
func WithOptions(v Options) Option {
	owned := CloneOptions(v)
	return func(c *Options) error { *c = CloneOptions(owned); return nil }
}
func WithQuery(key string, value any) Option {
	// Bound positional arguments cannot become fetch kwargs. Reject their
	// presence without invoking a caller marshaler, even for a null value.
	if key == "self" || key == "session" || key == "project" {
		return func(*Options) error { return invalid("quota query %q collides with a bound Source argument", key) }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *Options) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithUsage(v bool) Option {
	return func(c *Options) error { c.Usage = v; delete(c.Query, "usage"); return nil }
}
func WithMicroversion(v string) Option {
	return func(c *Options) error { x := v; c.Microversion = &x; delete(c.Query, "microversion"); return nil }
}
func WithBasePath(v string) Option {
	return func(c *Options) error { x := v; c.BasePath = &x; delete(c.Query, "base_path"); return nil }
}
func WithRequiresID(v bool) Option {
	return func(c *Options) error { c.RequiresID = v; delete(c.Query, "requires_id"); return nil }
}
func WithSkipCache(v bool) Option {
	return func(c *Options) error { c.SkipCache = v; delete(c.Query, "skip_cache"); return nil }
}
func WithErrorMessage(v string) Option {
	return func(c *Options) error { x := v; c.ErrorMessage = &x; delete(c.Query, "error_message"); return nil }
}
func WithResponseKey(v string) Option {
	return func(c *Options) error {
		x := v
		c.ResponseKey = &x
		delete(c.Query, "resource_response_key")
		return nil
	}
}

func applyOptions(profile Profile, userID string, options []Option, check func() error) (Options, error) {
	var c Options
	for _, apply := range options {
		if err := check(); err != nil {
			return c, err
		}
		if apply == nil {
			return c, invalid("nil quota read option")
		}
		c = CloneOptions(c)
		optionErr := apply(&c)
		if err := errors.Join(optionErr, check()); err != nil {
			return c, err
		}
		c = CloneOptions(c)
	}
	if err := check(); err != nil {
		return c, err
	}
	c = CloneOptions(c)
	keys := make([]string, 0, len(c.Query))
	for key := range c.Query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !text(key) || strings.TrimSpace(key) == "" {
			return c, invalid("quota query key must be nonblank UTF8 without controls")
		}
		switch key {
		case "self", "session", "project":
			return c, invalid("quota query %q collides with a bound Source argument", key)
		}
		if profile.NovaDetail && key == "base_path" {
			return c, invalid("Nova get_quota_set supplies base_path itself")
		}
		if userID != "" && key == "user_id" {
			return c, invalid("user_id is fixed by UserQuotaScope")
		}
		raw := c.Query[key]
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return c, invalid("quota query %q requires complete UTF8 JSON", key)
		}
		switch key {
		case "usage", "requires_id", "skip_cache":
			if key == "usage" && !profile.NovaDetail {
				c.usageValue = bytes.Clone(raw)
				delete(c.Query, key)
				continue
			}
			truth, err := cloudfilter.PythonJSONTruthy(raw)
			if err != nil {
				return c, err
			}
			switch key {
			case "usage":
				c.Usage = truth
			case "requires_id":
				c.RequiresID = truth
			case "skip_cache":
				c.SkipCache = truth
			}
			delete(c.Query, key)
		case "resource_response_key":
			// common.QuotaSet._translate_response accepts this argument but ignores it.
			delete(c.Query, key)
		case "base_path", "microversion", "error_message":
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				switch key {
				case "base_path":
					c.BasePath = nil
				case "microversion":
					c.Microversion = nil
				case "error_message":
					c.ErrorMessage = nil
				}
				delete(c.Query, key)
				continue
			}
			var value string
			if err := json.Unmarshal(raw, &value); err != nil {
				return c, invalid("quota fetch %q requires string or null", key)
			}
			switch key {
			case "base_path":
				c.BasePath = &value
			case "microversion":
				c.Microversion = &value
			case "error_message":
				c.ErrorMessage = &value
			}
			delete(c.Query, key)
		}
	}
	if c.RequiresID {
		return c, invalid("fresh QuotaSet fetch requires_id=True has no resource ID")
	}
	if profile.NovaDetail && c.BasePath != nil {
		return c, invalid("Nova get_quota_set supplies base_path itself")
	}
	for _, value := range []*string{c.BasePath, c.Microversion, c.ErrorMessage, c.ResponseKey} {
		if value != nil && !text(*value) {
			return c, invalid("quota fetch control requires UTF8 without controls")
		}
	}
	if c.Location != nil {
		if _, err := c.Location.ForResource(nil, nil); err != nil {
			return c, err
		}
	}
	return c, nil
}
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func text(v string) bool { return utf8.ValidString(v) && strings.IndexFunc(v, unicode.IsControl) < 0 }
func cloneFields(v map[string]json.RawMessage) map[string]json.RawMessage {
	if v == nil {
		return nil
	}
	c := maps.Clone(v)
	for key, raw := range c {
		c[key] = bytes.Clone(raw)
	}
	return c
}

func WithLocation(v resource.CloudLocation) Option {
	owned := v.Clone()
	return func(c *Options) error { x := owned.Clone(); c.Location = &x; return nil }
}
