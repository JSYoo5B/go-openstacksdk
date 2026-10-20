package objects

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// DeleteObject discovers unknown SLO state with fresh HEAD, then deletes the
// object or its manifest and segments. Only clean physical 404 responses may
// be ignored. An SLO 200 report is checked for embedded failures; a 202/204
// acknowledgement retains unknown segment counts and completion state.
func (a *API) DeleteObject(ctx context.Context, container, object string, options ...DeleteObjectOption) (*DeleteObjectResult, error) {
	p, err := a.captureDeleteObject(ctx, container, object)
	if err != nil {
		return nil, request.Wrap("DeleteObject", "objects", err)
	}
	cfg, err := p.applyOptions(ctx, options)
	if err = p.finish(ctx, cfg, err); err != nil {
		return nil, request.Wrap("DeleteObject", "objects", err)
	}
	var result *DeleteObjectResult
	var discovery *rest.Response
	if p.staticLargeObject == nil {
		codes := []int{http.StatusOK, http.StatusNoContent}
		if p.ignoreMissing {
			codes = append(codes, http.StatusNotFound)
		}
		discovery, err = p.read(ctx, http.MethodHead, p.target, false, codes...)
		if discovery != nil {
			result = &DeleteObjectResult{Discovery: objectDeleteResponse(discovery)}
		}
		if err != nil {
			return result, request.Wrap("DeleteObject", "objects", err)
		}
		if discovery.StatusCode == http.StatusNotFound {
			result.IgnoredMissing = true
			return result, nil
		}
		flag, projectionErr := objectDeleteSLOFlag(discovery.Header)
		if err = joinMetadataErrors(projectionErr, p.guard(ctx)); err != nil {
			return result, request.Wrap("DeleteObject", "objects", discovery.Fail(err))
		}
		p.staticLargeObject = &flag
		result.StaticLargeObject = cloneDeleteObjectBool(&flag)
	}
	if err = p.guard(ctx); err != nil {
		if discovery != nil {
			err = discovery.Fail(err)
		}
		return result, request.Wrap("DeleteObject", "objects", err)
	}
	slo := *p.staticLargeObject
	target := p.target
	codes := []int{http.StatusAccepted, http.StatusNoContent}
	if slo {
		parsed, _ := url.Parse(target)
		query := parsed.Query()
		query.Set("multipart-manifest", "delete")
		parsed.RawQuery = query.Encode()
		target = parsed.String()
		codes = append(codes, http.StatusOK)
	}
	if p.ignoreMissing {
		codes = append(codes, http.StatusNotFound)
	}
	deletion, err := p.read(ctx, http.MethodDelete, target, slo, codes...)
	if deletion != nil {
		if result == nil {
			result = &DeleteObjectResult{StaticLargeObject: cloneDeleteObjectBool(p.staticLargeObject)}
		}
		result.Deletion = objectDeleteResponse(deletion)
	}
	if err != nil {
		return result, request.Wrap("DeleteObject", "objects", err)
	}
	if deletion.StatusCode == http.StatusNotFound {
		result.IgnoredMissing = true
		return result, nil
	}
	if slo && deletion.StatusCode == http.StatusOK {
		var bulk ObjectDeleteBulkInfo
		if err = json.Unmarshal(deletion.Body, &bulk); err == nil {
			bulk.Header, bulk.StatusCode = deletion.Header.Clone(), deletion.StatusCode
			result.Bulk = &bulk
			if bulk.ResponseCode < 200 || bulk.ResponseCode >= 300 || len(bulk.Errors) != 0 {
				err = &ObjectDeleteBulkError{
					ResponseStatus: bulk.ResponseStatus, ResponseBody: bulk.ResponseBody, ResponseCode: bulk.ResponseCode,
					Errors: append([]ObjectDeleteBulkFailure{}, bulk.Errors...),
				}
			}
		}
		if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
			return result, request.Wrap("DeleteObject", "objects", deletion.Fail(err))
		}
	}
	return result, nil
}

func objectDeleteResponse(response *rest.Response) *DeleteObjectResponse {
	return &DeleteObjectResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
