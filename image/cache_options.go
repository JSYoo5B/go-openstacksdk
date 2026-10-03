package image

import "maps"

// CacheOpts supplies ordinary headers for cache reads, queueing and maintenance.
type CacheOpts struct{ Headers map[string]string }
type CacheOption func(*CacheOpts) error

// CacheDeleteOpts controls deletion of one cached or queued image. A nil
// IgnoreMissing uses the default true; name lookup errors are always returned.
type CacheDeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type CacheDeleteOption func(*CacheDeleteOpts) error

// CacheTarget selects the part of the cache cleared by ClearCache.
type CacheTarget uint8

const (
	CacheBoth CacheTarget = iota
	CacheOnly
	QueueOnly
)

// ClearCacheOpts uses CacheBoth by default. The target header is SDK-owned.
type ClearCacheOpts struct {
	Headers map[string]string
	Target  CacheTarget
}
type ClearCacheOption func(*ClearCacheOpts) error

func WithCacheOpts(value CacheOpts) CacheOption {
	snapshot := copyCacheOpts(value)
	return func(config *CacheOpts) error { *config = copyCacheOpts(snapshot); return nil }
}
func WithCacheHeader(key, value string) CacheOption {
	return cacheHeaderOption(WithImageMutationHeader(key, value))
}
func WithCacheHeaders(values map[string]string) CacheOption {
	return cacheHeaderOption(WithImageMutationHeaders(values))
}
func WithCacheDeleteOpts(value CacheDeleteOpts) CacheDeleteOption {
	snapshot := copyCacheDeleteOpts(value)
	return func(config *CacheDeleteOpts) error { *config = copyCacheDeleteOpts(snapshot); return nil }
}
func WithCacheDeleteHeader(key, value string) CacheDeleteOption {
	apply := WithCacheHeader(key, value)
	return func(config *CacheDeleteOpts) error {
		value := CacheOpts{Headers: config.Headers}
		err := apply(&value)
		config.Headers = value.Headers
		return err
	}
}
func WithCacheDeleteHeaders(values map[string]string) CacheDeleteOption {
	apply := WithCacheHeaders(values)
	return func(config *CacheDeleteOpts) error {
		value := CacheOpts{Headers: config.Headers}
		err := apply(&value)
		config.Headers = value.Headers
		return err
	}
}
func WithCacheDeleteIgnoreMissing(value bool) CacheDeleteOption {
	return func(config *CacheDeleteOpts) error { snapshot := value; config.IgnoreMissing = &snapshot; return nil }
}
func WithClearCacheOpts(value ClearCacheOpts) ClearCacheOption {
	snapshot := copyClearCacheOpts(value)
	return func(config *ClearCacheOpts) error { *config = copyClearCacheOpts(snapshot); return nil }
}
func WithClearCacheHeader(key, value string) ClearCacheOption {
	apply := WithCacheHeader(key, value)
	return func(config *ClearCacheOpts) error {
		value := CacheOpts{Headers: config.Headers}
		err := apply(&value)
		config.Headers = value.Headers
		return err
	}
}
func WithClearCacheHeaders(values map[string]string) ClearCacheOption {
	apply := WithCacheHeaders(values)
	return func(config *ClearCacheOpts) error {
		value := CacheOpts{Headers: config.Headers}
		err := apply(&value)
		config.Headers = value.Headers
		return err
	}
}
func WithClearCacheTarget(value CacheTarget) ClearCacheOption {
	return func(config *ClearCacheOpts) error { config.Target = value; return nil }
}

func cacheHeaderOption(apply ImageMutationOption) CacheOption {
	return func(config *CacheOpts) error {
		value := ImageMutationOpts{Headers: config.Headers}
		err := apply(&value)
		config.Headers = value.Headers
		if err != nil {
			return err
		}
		_, err = cacheHeaders(config.Headers, false, "")
		return err
	}
}
func copyCacheOpts(value CacheOpts) CacheOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	return value
}
func copyCacheDeleteOpts(value CacheDeleteOpts) CacheDeleteOpts {
	value.Headers = copyCacheOpts(CacheOpts{Headers: value.Headers}).Headers
	missing := true
	if value.IgnoreMissing != nil {
		missing = *value.IgnoreMissing
	}
	value.IgnoreMissing = &missing
	return value
}
func copyClearCacheOpts(value ClearCacheOpts) ClearCacheOpts {
	value.Headers = copyCacheOpts(CacheOpts{Headers: value.Headers}).Headers
	return value
}
func parseCacheOptions(options []CacheOption) (CacheOpts, error) {
	config := copyCacheOpts(CacheOpts{})
	for _, apply := range options {
		if apply == nil {
			return config, uploadInvalid("nil cache option")
		}
		candidate := copyCacheOpts(config)
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config = copyCacheOpts(candidate)
	}
	headers, err := cacheHeaders(config.Headers, false, "")
	config.Headers = headers
	return config, err
}
func parseCacheDeleteOptions(options []CacheDeleteOption) (CacheDeleteOpts, error) {
	config := copyCacheDeleteOpts(CacheDeleteOpts{})
	for _, apply := range options {
		if apply == nil {
			return config, uploadInvalid("nil cache delete option")
		}
		candidate := copyCacheDeleteOpts(config)
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config = copyCacheDeleteOpts(candidate)
	}
	headers, err := cacheHeaders(config.Headers, false, "")
	config.Headers = headers
	return config, err
}
func parseClearCacheOptions(options []ClearCacheOption) (ClearCacheOpts, error) {
	config := copyClearCacheOpts(ClearCacheOpts{})
	for _, apply := range options {
		if apply == nil {
			return config, uploadInvalid("nil clear cache option")
		}
		candidate := copyClearCacheOpts(config)
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config = copyClearCacheOpts(candidate)
	}
	if config.Target != CacheBoth && config.Target != CacheOnly && config.Target != QueueOnly {
		return config, uploadInvalid("invalid cache target")
	}
	headers, err := cacheHeaders(config.Headers, false, "")
	config.Headers = headers
	return config, err
}
