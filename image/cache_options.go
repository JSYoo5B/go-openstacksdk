package image

import (
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CacheOpts supplies ordinary headers for cache reads, queueing and maintenance.
type CacheOpts struct{ Headers map[string]string }

// CacheOption is one option of the cache reads, queueing and maintenance
// operations: WithCacheOpts, WithCacheHeaders (merges), request.Header (merges
// one header) or request.Custom over *CacheOpts. Other shared calls, including
// request.Headers, fail with resource.ErrInvalidOption before HTTP.
type CacheOption = request.GetCall

// CacheDeleteOpts controls deletion of one cached or queued image. A nil
// IgnoreMissing uses the default true; name lookup errors are always returned.
type CacheDeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// CacheDeleteOption is one option of the cached and queued image deletions:
// WithCacheDeleteOpts, WithCacheDeleteHeaders (merges), request.Header (merges
// one header), request.IgnoreMissing or request.Custom over *CacheDeleteOpts.
// Other shared calls, including request.Headers, fail with
// resource.ErrInvalidOption before HTTP.
type CacheDeleteOption = request.DeleteCall

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

// ClearCacheOption is one option of ClearCache: WithClearCacheOpts,
// WithClearCacheHeaders (merges), WithClearCacheTarget, request.Header (merges
// one header) or request.Custom over *ClearCacheOpts. Other shared calls,
// including request.Headers and request.IgnoreMissing, fail with
// resource.ErrInvalidOption before HTTP.
type ClearCacheOption = request.DeleteCall

var (
	cacheCalls       = calls.Spec[CacheOpts]{Header: calls.MergeHeader[CacheOpts](WithCacheHeaders)}
	cacheDeleteCalls = calls.Spec[CacheDeleteOpts]{
		Header:        calls.MergeHeader[CacheDeleteOpts](WithCacheDeleteHeaders),
		IgnoreMissing: calls.Pointer(func(c *CacheDeleteOpts) **bool { return &c.IgnoreMissing }),
	}
	clearCacheCalls = calls.Spec[ClearCacheOpts]{Header: calls.MergeHeader[ClearCacheOpts](WithClearCacheHeaders)}
)

func WithCacheOpts(value CacheOpts) CacheOption {
	snapshot := copyCacheOpts(value)
	return request.Custom(func(config *CacheOpts) error { *config = copyCacheOpts(snapshot); return nil })
}
func WithCacheHeaders(values map[string]string) CacheOption {
	return request.Custom(cacheHeaderOption(mergeImageMutationHeaders(values)))
}
func WithCacheDeleteOpts(value CacheDeleteOpts) CacheDeleteOption {
	snapshot := copyCacheDeleteOpts(value)
	return request.Custom(func(config *CacheDeleteOpts) error { *config = copyCacheDeleteOpts(snapshot); return nil })
}
func WithCacheDeleteHeaders(values map[string]string) CacheDeleteOption {
	apply := cacheHeaderOption(mergeImageMutationHeaders(values))
	return request.Custom(func(config *CacheDeleteOpts) error {
		value := CacheOpts{Headers: config.Headers}
		err := apply(&value)
		config.Headers = value.Headers
		return err
	})
}
func WithClearCacheOpts(value ClearCacheOpts) ClearCacheOption {
	snapshot := copyClearCacheOpts(value)
	return request.Custom(func(config *ClearCacheOpts) error { *config = copyClearCacheOpts(snapshot); return nil })
}
func WithClearCacheHeaders(values map[string]string) ClearCacheOption {
	apply := cacheHeaderOption(mergeImageMutationHeaders(values))
	return request.Custom(func(config *ClearCacheOpts) error {
		value := CacheOpts{Headers: config.Headers}
		err := apply(&value)
		config.Headers = value.Headers
		return err
	})
}
func WithClearCacheTarget(value CacheTarget) ClearCacheOption {
	return request.Custom(func(config *ClearCacheOpts) error { config.Target = value; return nil })
}

func cacheHeaderOption(apply func(*ImageMutationOpts) error) func(*CacheOpts) error {
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
	for _, apply := range calls.Translate(options, cacheCalls) {
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
	for _, apply := range calls.Translate(options, cacheDeleteCalls) {
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
	for _, apply := range calls.Translate(options, clearCacheCalls) {
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
