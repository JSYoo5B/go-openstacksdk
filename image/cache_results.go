package image

import (
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ImageCacheAcknowledgement retains actual accepted response evidence. Queueing
// does not prove completed caching. Target is present only for ClearCache.
type ImageCacheAcknowledgement struct {
	ImageID    string
	Target     *CacheTarget
	Body       []byte
	Header     http.Header
	StatusCode int
}

// CachedImageNodesResult preserves a finite list of passive node reference URLs.
type CachedImageNodesResult struct {
	ImageID    string
	Nodes      []string
	Body       []byte
	Header     http.Header
	StatusCode int
}

func cacheAcknowledgement(id string, target *CacheTarget, response *rest.Response, status int) *ImageCacheAcknowledgement {
	if response == nil || response.StatusCode != status {
		return nil
	}
	result := &ImageCacheAcknowledgement{ImageID: id, Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if target != nil {
		owned := *target
		result.Target = &owned
	}
	return result
}
