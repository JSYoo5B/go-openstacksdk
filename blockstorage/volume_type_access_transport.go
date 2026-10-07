package blockstorage

import (
	"bytes"
	"context"
	"errors"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
)

// Type access's concrete Python exception gate accepts every final HTTP status
// below 400. A fresh slice owns this policy through native retries; lookup keeps
// its separate ordinary type GET policy. Informational responses follow net/http.
func volumeTypeAccessSuccessCodes() []int {
	codes := make([]int, 300)
	for index := range codes {
		codes[index] = 100 + index
	}
	return codes
}

func (p *preparedVolumeTypes) accessExchange(ctx context.Context, method, id, phase string, body any) (*rest.Response, error) {
	if err := p.reader.guard(ctx); err != nil {
		return nil, err
	}
	target := p.reader.cinder.client.ServiceURL("types", url.PathEscape(id), phase)
	if err := validateAttachTarget(&p.reader.cinder.client, target); err != nil {
		return nil, err
	}
	response, err := rest.DoJSON(ctx, &p.reader.cinder.client, method, target, body, nil, volumeTypeAccessSuccessCodes()...)
	if observed := p.reader.guard(ctx); observed != nil {
		err = errors.Join(err, observed)
		if response != nil {
			err = response.Fail(err)
		}
	}
	return response, attachContextError(ctx, err)
}

func volumeTypeAccessProof(response *rest.Response) *VolumeTypesPage {
	if response == nil {
		return nil
	}
	return &VolumeTypesPage{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
