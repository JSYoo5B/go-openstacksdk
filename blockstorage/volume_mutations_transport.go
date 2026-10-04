package blockstorage

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"

	"gophercloudsdk/internal/rest"
)

func volumeMutationSuccessCodes() []int {
	codes := make([]int, 300)
	for index := range codes {
		codes[index] = 100 + index
	}
	return codes
}

func (p *preparedVolumeSearch) mutationExchange(ctx context.Context, method, id string, body any) (*rest.Response, error) {
	if err := p.reader.guard(ctx); err != nil {
		return nil, err
	}
	segments := []string{"volumes", url.PathEscape(id)}
	switch method {
	case http.MethodPut:
	case http.MethodPost:
		segments = append(segments, "action")
	default:
		return nil, attachInvalid("volume mutation changes its fixed phase method")
	}
	target := p.reader.cinder.client.ServiceURL(segments...)
	if err := validateAttachTarget(&p.reader.cinder.client, target); err != nil {
		return nil, err
	}
	response, err := rest.DoJSON(ctx, &p.reader.cinder.client, method, target, body, nil, volumeMutationSuccessCodes()...)
	if observed := p.reader.guard(ctx); observed != nil {
		err = errors.Join(err, observed)
		if response != nil {
			err = response.Fail(err)
		}
	}
	return response, attachContextError(ctx, err)
}

func volumeMutationProof(response *rest.Response) *VolumeMutationPage {
	if response == nil {
		return nil
	}
	return &VolumeMutationPage{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
