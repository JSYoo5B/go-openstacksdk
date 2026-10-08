package image

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

func checkTaskResponse(ctx context.Context, p *preparedTaskSource, response *rest.Response, err error) error {
	if sourceErr := p.check(ctx); sourceErr != nil {
		if response != nil {
			sourceErr = response.Fail(sourceErr)
		}
		if err == nil {
			return sourceErr
		}
		return errors.Join(err, sourceErr)
	}
	return err
}

func decodeTaskInfoResponse(ctx context.Context, p *preparedTaskSource, operation string, response *rest.Response, err error) (*TaskInfo, error) {
	if err = checkTaskResponse(ctx, p, response, err); err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	var value TaskInfo
	if err = json.Unmarshal(response.Body, &value); err != nil {
		return nil, wrapImageMutationError(ctx, operation, response.Fail(err))
	}
	if err = p.check(ctx); err != nil {
		return nil, wrapImageMutationError(ctx, operation, response.Fail(err))
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	return &value, nil
}
