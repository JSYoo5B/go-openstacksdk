package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func observed(response *rest.Response) *Page {
	if response == nil {
		return nil
	}
	return &Page{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}

func sourceCodes() []int {
	codes := make([]int, 300)
	for i := range codes {
		codes[i] = 100 + i
	}
	return codes
}

// Resource.fetch tolerates a JSON ValueError only. A valid non-object and an
// invalid UTF-8 response cannot be promoted to an empty successful resource.
func memberObject(response *rest.Response) (json.RawMessage, error) {
	return memberObjectFor(response, snapshotReadSchema())
}

func memberObjectFor(response *rest.Response, schema readSchema) (json.RawMessage, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(fmt.Errorf("Cinder resource response must be UTF-8 JSON"))
	}
	if !json.Valid(response.Body) {
		return json.RawMessage("{}"), nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &fields); err != nil {
		return nil, response.Fail(err)
	}
	if fields == nil {
		return nil, response.Fail(fmt.Errorf("Cinder resource response must be a nonnull JSON object"))
	}
	raw, present := fields[schema.singular]
	if !present {
		raw = response.Body
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, response.Fail(err)
	}
	if object == nil {
		return nil, response.Fail(fmt.Errorf("Cinder resource member must be a nonnull JSON object"))
	}
	return bytes.Clone(raw), nil
}

func listObjects(response *rest.Response) (map[string]json.RawMessage, []json.RawMessage, error) {
	return listObjectsFor(response, snapshotReadSchema())
}

func listObjectsFor(response *rest.Response, schema readSchema) (map[string]json.RawMessage, []json.RawMessage, error) {
	if !utf8.Valid(response.Body) {
		return nil, nil, response.Fail(fmt.Errorf("Cinder resource list response must be UTF-8 JSON"))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &fields); err != nil {
		return nil, nil, response.Fail(err)
	}
	raw, present := fields[schema.plural]
	if !present {
		return nil, nil, response.Fail(fmt.Errorf("Cinder resource list response must contain %s", schema.plural))
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) != 0 && trimmed[0] == '[' {
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, nil, response.Fail(err)
		}
		return fields, rows, nil
	}
	// Source wraps any non-list payload exactly once; construction is lazy.
	return fields, []json.RawMessage{bytes.Clone(raw)}, nil
}

func (p *reader) materialize(ctx context.Context, raw json.RawMessage, seed *string, list bool, wire *rest.Response) (*entry, error) {
	view, seeded, err := p.schema.normalize(raw, seed, list, p.location)
	if err != nil {
		var own *locationError
		if errors.As(err, &own) {
			return nil, err
		}
		return nil, wire.Fail(err)
	}
	var value resource.RawResource
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, wire.Fail(err)
	}
	value.Header, value.StatusCode = wire.Header.Clone(), wire.StatusCode
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(view, &fields); err != nil {
		return nil, wire.Fail(err)
	}
	text := func(raw json.RawMessage) string {
		var value string
		if len(raw) != 0 && raw[0] == '"' {
			_ = json.Unmarshal(raw, &value)
		}
		return value
	}
	if err := p.source.Guard(ctx); err != nil {
		return nil, wire.Fail(err)
	}
	return &entry{view: view, resource: &value, id: text(fields["id"]), name: text(fields["name"]), seeded: seeded}, nil
}
