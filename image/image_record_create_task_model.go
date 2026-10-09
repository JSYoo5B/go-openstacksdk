package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecordCreateTask preserves the Task's raw descriptor view and actual
// HTTP evidence independently. Missing Body descriptors read as JSON null.
// Input, result, status, and id are not coerced on the no-wait create path.
type ImageRecordCreateTask struct {
	Resource   *resource.RawResource
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
	body       map[string]json.RawMessage
}

// ImageRecordCreateTaskWaitResult retains the latest observed Task and each
// actual accepted fetch/recreation receipt, including handling failures.
// Counts match their receipt slices; rejected requests do not invent receipts.
type ImageRecordCreateTaskWaitResult struct {
	Task *ImageRecordCreateTask
	// OriginalTask follows the first Source object until recreation replaces the
	// waiter's local object. Outer failure diagnostics still use that object.
	OriginalTask                        *ImageRecordCreateTask
	Fetches, Recreations                int
	FetchResponses, RecreationResponses []*ImageUploadResponse
}

// ImageRecordCreateTaskFailureError marks only Source ResourceFailure branches.
// HTTP, representation, timeout, and source-guard failures retain their own
// causes and do not request the outer workflow's failure-diagnostic Task GET.
type ImageRecordCreateTaskFailureError struct {
	Task       *ImageRecordCreateTask
	ID, Status string
}

func (e *ImageRecordCreateTaskFailureError) Error() string {
	return fmt.Sprintf("Task:%s transitioned to failure state %s", e.ID, e.Status)
}
func (e *ImageRecordCreateTaskFailureError) Unwrap() error { return resource.ErrFailedState }

// Task declares ten Body fields; inherited id/name add two. It has no tags
// mixin. owner_id is its canonical descriptor for the owner wire attribute.
var imageRecordCreateTaskFields = [...]struct{ canonical, wire string }{
	{"created_at", "created_at"}, {"expires_at", "expires_at"},
	{"input", "input"}, {"message", "message"}, {"owner_id", "owner"},
	{"result", "result"}, {"schema", "schema"}, {"status", "status"},
	{"type", "type"}, {"updated_at", "updated_at"}, {"id", "id"}, {"name", "name"},
}

func normalizeImageRecordCreateTask(fields map[string]json.RawMessage, envelope json.RawMessage) (map[string]json.RawMessage, error) {
	if envelope == nil {
		var err error
		envelope, err = imageRecordObject(fields)
		if err != nil {
			return nil, err
		}
	}
	members, err := cloudfilter.ObjectMembers(envelope)
	if err != nil {
		return nil, err
	}
	result := make(map[string]json.RawMessage)
	for _, member := range members {
		for _, descriptor := range imageRecordCreateTaskFields {
			if member.Key == descriptor.canonical || member.Key == descriptor.wire {
				result[descriptor.canonical] = bytes.Clone(member.Value)
				break
			}
		}
	}
	return result, nil
}

func projectImageRecordCreateTask(body map[string]json.RawMessage, location json.RawMessage, response *rest.Response) (*ImageRecordCreateTask, error) {
	view := make(map[string]json.RawMessage, len(imageRecordCreateTaskFields)+1)
	for _, field := range imageRecordCreateTaskFields {
		raw := body[field.canonical]
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, uploadInvalid("task descriptor %q must be complete UTF-8 JSON", field.canonical)
		}
		view[field.canonical] = bytes.Clone(raw)
	}
	if location == nil {
		location = json.RawMessage("null")
	}
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, uploadInvalid("task location must be complete UTF-8 JSON")
	}
	view["location"] = bytes.Clone(location)
	value := &ImageRecordCreateTask{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: view}}, body: copyTaskRawMap(body)}
	if response != nil {
		value.Envelope, value.Header, value.StatusCode = bytes.Clone(response.Body), response.Header.Clone(), response.StatusCode
		value.Resource.Header, value.Resource.StatusCode = response.Header.Clone(), response.StatusCode
	}
	return value, nil
}

func cloneImageRecordCreateTask(value *ImageRecordCreateTask) *ImageRecordCreateTask {
	if value == nil {
		return nil
	}
	return &ImageRecordCreateTask{Resource: value.Resource.Clone(), Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode, body: copyTaskRawMap(value.body)}
}

func imageRecordCreateTaskBody(value *ImageRecordCreateTask) (map[string]json.RawMessage, error) {
	if value == nil || value.Resource == nil {
		return nil, uploadInvalid("task Resource is required")
	}
	if value.body != nil {
		return copyTaskRawMap(value.body), nil
	}
	return normalizeImageRecordCreateTask(value.Resource.Body, nil)
}

func imageRecordCreateTaskRaw(value *ImageRecordCreateTask, key string) json.RawMessage {
	var raw json.RawMessage
	if value != nil {
		if value.body != nil {
			raw = value.body[key]
		} else if value.Resource != nil {
			raw = value.Resource.Body[key]
		}
	}
	if raw == nil {
		return json.RawMessage("null")
	}
	return bytes.Clone(raw)
}

func imageRecordCreateTaskReceipt(seed *ImageRecordCreateTask, response *rest.Response) *ImageRecordCreateTask {
	value := cloneImageRecordCreateTask(seed)
	if value == nil || response == nil {
		return value
	}
	value.Envelope, value.Header, value.StatusCode = bytes.Clone(response.Body), response.Header.Clone(), response.StatusCode
	if value.Resource != nil {
		value.Resource.Header, value.Resource.StatusCode = response.Header.Clone(), response.StatusCode
	}
	return value
}

func imageRecordCreateTaskFromResponse(p *preparedImageRecord, seed *ImageRecordCreateTask, response *rest.Response) (*ImageRecordCreateTask, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(uploadInvalid("task response must be UTF-8"))
	}
	body, err := imageRecordCreateTaskBody(seed)
	if err != nil {
		return nil, response.Fail(err)
	}
	// Resource tolerates JSON syntax failures without replacing its current
	// raw Body. A syntactically valid non-object remains a translation failure.
	if json.Valid(response.Body) {
		var wire resource.RawResource
		if err := json.Unmarshal(response.Body, &wire); err != nil {
			return nil, response.Fail(err)
		}
		normalized, err := normalizeImageRecordCreateTask(wire.Body, response.Body)
		if err != nil {
			return nil, response.Fail(err)
		}
		for key, raw := range normalized {
			body[key] = bytes.Clone(raw)
		}
	}
	value, err := projectImageRecordCreateTask(body, p.location, response)
	if err != nil {
		return nil, response.Fail(err)
	}
	if err := p.check(p.ctx); err != nil {
		return value, response.Fail(err)
	}
	return value, nil
}
