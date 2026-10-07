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

// member owns one logical bodyless volume GET. Native authentication and retry
// can repeat this same route; only admitted responses become Observed evidence.
func (p *preparedVolumeSearch) member(ctx context.Context, id string, result *GetVolumeResult) (record *volumeIdentityRecord, err error) {
	defer func() { p.memberFailure = err }()
	if err := p.reader.guard(ctx); err != nil {
		return nil, err
	}
	target := p.reader.cinder.client.ServiceURL("volumes", url.PathEscape(id))
	if err := validateAttachTarget(&p.reader.cinder.client, target); err != nil {
		return nil, err
	}
	response, err := rest.DoJSON(ctx, &p.reader.cinder.client, http.MethodGet, target, nil, nil, http.StatusOK)
	if response != nil {
		result.Observed = &GetVolumesPage{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
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
	row, present := fields["volume"]
	if !present {
		return nil, response.Fail(attachInvalid("member response must contain canonical volume object"))
	}
	var volume resource.RawResource
	if err := json.Unmarshal(row, &volume); err != nil {
		return nil, response.Fail(err)
	}
	volume.Header, volume.StatusCode = response.Header.Clone(), response.StatusCode
	return p.identityRecord(ctx, getVolumesEntry{volume: &volume, origin: response, raw: bytes.Clone(row)})
}
