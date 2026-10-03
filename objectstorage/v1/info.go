package v1

import (
	"context"
	"encoding/json"
	"net/http"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/swiftinfo"
	"gophercloudsdk/request"
)

// GetInfo reads fresh capabilities from the catalog-derived /info endpoint.
// Only an actual HTTP 200 with a complete capability object returns Info.
func (s *Service) GetInfo(ctx context.Context, options ...GetInfoOption) (*Info, error) {
	p, err := s.captureInfo(ctx)
	if err != nil {
		return nil, request.Wrap("GetInfo", "objectstorage", err)
	}
	cfg, err := p.applyGetInfoOptions(ctx, options)
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("GetInfo", "objectstorage", err)
	}
	response, err := p.read(ctx, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("GetInfo", "objectstorage", err)
	}
	info, err := decodeInfo(response)
	if guardErr := p.guard(ctx); guardErr != nil {
		err = response.Fail(joinTempURLKeyErrors(err, guardErr))
	}
	if err != nil {
		return nil, request.Wrap("GetInfo", "objectstorage", err)
	}
	return info, nil
}

// GetObjectSegmentSize clamps the requested size using fresh Swift/SLO bounds.
// A nil size requests 1 GiB. Only a clean actual 404 or 412 selects the fallback
// maximum 2684354561; accepted-response failures retain proof with Size zero.
func (s *Service) GetObjectSegmentSize(ctx context.Context, options ...ObjectSegmentSizeOption) (*ObjectSegmentSizeResult, error) {
	p, err := s.captureInfo(ctx)
	if err != nil {
		return nil, request.Wrap("GetObjectSegmentSize", "objectstorage", err)
	}
	cfg, err := p.applySegmentOptions(ctx, options)
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("GetObjectSegmentSize", "objectstorage", err)
	}
	requested := swiftinfo.DefaultSegmentSize
	if cfg.Size != nil {
		requested = *cfg.Size
	}
	if requested < 0 {
		return nil, request.Wrap("GetObjectSegmentSize", "objectstorage", tempURLKeyContextError(ctx, tempURLKeyInvalid("negative object segment size")))
	}
	response, err := p.read(ctx, http.StatusOK, http.StatusNotFound, http.StatusPreconditionFailed)
	var result *ObjectSegmentSizeResult
	if response != nil {
		result = &ObjectSegmentSizeResult{RequestedSize: requested, Header: response.Header.Clone(), StatusCode: response.StatusCode, Body: append([]byte(nil), response.Body...)}
	}
	if err != nil {
		return result, request.Wrap("GetObjectSegmentSize", "objectstorage", err)
	}
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusPreconditionFailed {
		result.MaxFileSize, result.UsedFallback = swiftinfo.FallbackMaxFileSize, true
	} else {
		result.Info, err = decodeInfo(response)
		if err == nil {
			var maximum, minimum int64
			maximum, err = infoBound(result.Info.Swift, "max_file_size")
			if err == nil {
				minimum, err = infoBound(result.Info.SLO, "min_segment_size")
			}
			if err == nil {
				result.MaxFileSize, result.MinSegmentSize = maximum, minimum
			} else {
				err = response.Fail(err)
			}
		}
	}
	if guardErr := p.guard(ctx); guardErr != nil {
		err = response.Fail(joinTempURLKeyErrors(err, guardErr))
	}
	if err != nil {
		result.MaxFileSize, result.MinSegmentSize, result.UsedFallback = 0, 0, false
		return result, request.Wrap("GetObjectSegmentSize", "objectstorage", err)
	}
	result.Size = swiftinfo.Select(requested, result.MaxFileSize, result.MinSegmentSize)
	return result, nil
}

func decodeInfo(response *rest.Response) (*Info, error) {
	var info Info
	if err := json.Unmarshal(response.Body, &info); err != nil {
		return nil, response.Fail(err)
	}
	info.Header, info.StatusCode = response.Header.Clone(), response.StatusCode
	return &info, nil
}

func infoBound(section map[string]json.RawMessage, name string) (int64, error) {
	return swiftinfo.Bound(section, name)
}
