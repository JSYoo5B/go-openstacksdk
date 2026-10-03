package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// ImageTasks lazily fetches the finite task list for one image. MaxItems limits
// local consumption; links and paging hints are passive response data.
func (s *Service) ImageTasks(ctx context.Context, ref resource.Ref, options ...ListImageTasksOption) iter.Seq2[*ImageTaskInfo, error] {
	options = append([]ListImageTasksOption(nil), options...)
	return func(yield func(*ImageTaskInfo, error) bool) {
		fail := func(err error) { yield(nil, wrapImageMutationError(ctx, "ImageTasks", err)) }
		p, id, policy, err := s.prepareImageTasks(ctx, ref, options)
		if err != nil {
			fail(err)
			return
		}
		response, err := rest.DoJSON(ctx, p.client, http.MethodGet, p.base+"images/"+url.PathEscape(id)+"/tasks", nil, nil, http.StatusOK)
		if err = checkTaskResponse(ctx, p, response, err); err != nil {
			fail(err)
			return
		}
		fields, err := schemaObject(response.Body)
		if err != nil {
			fail(response.Fail(err))
			return
		}
		raw, exists := schemaField(fields, "tasks")
		if !exists || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '[' {
			fail(response.Fail(fmt.Errorf("image tasks response requires a nonnull tasks array")))
			return
		}
		var rows []json.RawMessage
		if err = json.Unmarshal(raw, &rows); err != nil {
			fail(response.Fail(err))
			return
		}
		for index, raw := range rows {
			if policy.MaxItems > 0 && index >= policy.MaxItems {
				return
			}
			if err = p.check(ctx); err != nil {
				fail(response.Fail(err))
				return
			}
			var value ImageTaskInfo
			if err = json.Unmarshal(raw, &value); err != nil {
				fail(response.Fail(err))
				return
			}
			value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
			if err = p.check(ctx); err != nil {
				fail(response.Fail(err))
				return
			}
			if !yield(&value, nil) {
				return
			}
			if err = p.check(ctx); err != nil {
				fail(response.Fail(err))
				return
			}
		}
		if err = p.check(ctx); err != nil {
			fail(response.Fail(err))
		}
	}
}

// AllImageTasks collects one finite response and discards partial rows on an
// error. A successful empty collection returns a nonnil empty slice.
func (s *Service) AllImageTasks(ctx context.Context, ref resource.Ref, options ...ListImageTasksOption) ([]*ImageTaskInfo, error) {
	values := make([]*ImageTaskInfo, 0)
	for value, err := range s.ImageTasks(ctx, ref, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
