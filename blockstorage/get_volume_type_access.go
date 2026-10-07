package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// GetVolumeTypeAccess resolves the type, then reads its access response once.
// The source field is untyped JSON; advertised links never trigger pagination.
func GetVolumeTypeAccess(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumeTypeAccessRequest, options ...VolumeTypeReadOption) (*GetVolumeTypeAccessResult, error) {
	p, resolved, id, err := prepareVolumeTypeAccess(ctx, cinder, input.NameOrID, options)
	if p == nil {
		return nil, wrapVolumeTypeError(ctx, "GetVolumeTypeAccess", err)
	}
	result := &GetVolumeTypeAccessResult{Resolved: resolved, TypeID: id}
	if err != nil {
		return result, wrapVolumeTypeError(ctx, "GetVolumeTypeAccess", err)
	}
	response, err := p.accessExchange(ctx, http.MethodGet, id, "os-volume-type-access", nil)
	result.Observed = volumeTypeAccessProof(response)
	if err != nil {
		return result, wrapVolumeTypeError(ctx, "GetVolumeTypeAccess", err)
	}
	fields, err := attachmentObject(response.Body)
	if err != nil {
		return result, wrapVolumeTypeError(ctx, "GetVolumeTypeAccess", response.Fail(err))
	}
	value, present := fields["volume_type_access"]
	if !present {
		value = json.RawMessage("[]")
	}
	accesses, err := volumeTypeAccessRows(value, response)
	if err != nil {
		return result, wrapVolumeTypeError(ctx, "GetVolumeTypeAccess", response.Fail(err))
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, wrapVolumeTypeError(ctx, "GetVolumeTypeAccess", err)
	}
	result.Value, result.Accesses = bytes.Clone(value), accesses
	return result, nil
}

// Optional convenience projection never narrows Value. Arbitrary array items
// or a non-array field remain successful values with no invented raw resources.
func volumeTypeAccessRows(value json.RawMessage, response *rest.Response) ([]*resource.RawResource, error) {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, nil
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(value, &rows); err != nil {
		return nil, err
	}
	accesses := make([]*resource.RawResource, 0, len(rows))
	for _, row := range rows {
		if item := bytes.TrimSpace(row); len(item) == 0 || item[0] != '{' {
			return nil, nil
		}
		var access resource.RawResource
		if err := json.Unmarshal(row, &access); err != nil {
			return nil, err
		}
		access.Header, access.StatusCode = response.Header.Clone(), response.StatusCode
		accesses = append(accesses, &access)
	}
	return accesses, nil
}
