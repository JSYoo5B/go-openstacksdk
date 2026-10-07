package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func (p *preparedVolumeTypes) member(ctx context.Context, id string, query url.Values, result *GetVolumeTypeResult) (record *volumeTypeIdentityRecord, err error) {
	defer func() { p.memberFailure = err }()
	if err := p.reader.guard(ctx); err != nil {
		return nil, err
	}
	target := p.reader.cinder.client.ServiceURL("types", url.PathEscape(id))
	if err := validateAttachTarget(&p.reader.cinder.client, target); err != nil {
		return nil, err
	}
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	response, err := rest.DoJSON(ctx, &p.reader.cinder.client, http.MethodGet, target, nil, nil, http.StatusOK)
	if response != nil {
		result.Observed = &VolumeTypesPage{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	}
	if observed := p.reader.guard(ctx); observed != nil {
		err = errors.Join(err, observed)
		if response != nil {
			err = response.Fail(err)
		}
	}
	if err != nil {
		return nil, attachContextError(ctx, err)
	}
	fields, err := attachmentObject(response.Body)
	if err != nil {
		return nil, response.Fail(err)
	}
	row, present := fields["volume_type"]
	if !present {
		return nil, response.Fail(attachInvalid("member response must contain canonical volume_type object"))
	}
	var value resource.RawResource
	if err := json.Unmarshal(row, &value); err != nil {
		return nil, response.Fail(err)
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	return p.identityRecord(ctx, volumeTypeEntry{value: &value, origin: response, raw: bytes.Clone(row)}, true)
}
