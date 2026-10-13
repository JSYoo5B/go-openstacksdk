package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// BulkPortResult retains the one real POST receipt. Records projects only
// returned response rows, lazily, with no merge from submitted attributes.
type BulkPortResult struct {
	Records    iter.Seq2[*PortRecord, error]
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

// All collects this already-submitted response iterator. It never sends a new
// request and discards partial collected rows when projection fails.
func (result *BulkPortResult) All() ([]*PortRecord, error) {
	if result == nil || result.Records == nil {
		return nil, findInvalid("completed bulk response iterator required")
	}
	records := make([]*PortRecord, 0)
	for record, err := range result.Records {
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// CreatePorts performs Source create_ports' single eager POST {ports:[attrs]}.
// Every row's raw Body and Header are validated first; only the last row Header
// applies to the request. Returned rows are projected on iteration, retaining
// Source constructor collisions and the actual completed POST on all errors.
func (a *API) CreatePorts(ctx context.Context, data []BulkPortInput, options ...BulkPortOption) (*BulkPortResult, error) {
	const op = "CreatePorts"
	wrap := func(err error) error { return request.Wrap(op, "ports", cloudread.ContextError(ctx, err)) }
	profile := portListProfile()
	profile.SourceStatusPolicy = true
	engine, err := a.portRecordEngine(ctx, profile)
	if err != nil {
		return nil, wrap(err)
	}
	if len(data) == 0 {
		return nil, wrap(findInvalid("bulk create requires nonempty port attribute objects"))
	}
	inputs := make([]BulkPortInput, len(data))
	for i, input := range data {
		inputs[i] = cloneBulkPortInput(input)
	}
	cfg := cloneBulkPortOpts(BulkPortOpts{})
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil bulk port option"))
		}
		next := cloneBulkPortOpts(cfg)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = cloneBulkPortOpts(next)
	}
	rows := make([]map[string]json.RawMessage, len(inputs))
	var lastHeaders map[string]string
	for i, input := range inputs {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		body, headers, err := prepareBulkPort(input)
		if err != nil {
			return nil, wrap(err)
		}
		rows[i], lastHeaders = body, headers
	}
	headers := maps.Clone(cfg.Headers)
	for key, value := range lastHeaders {
		headers[key] = value
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return nil, wrap(err)
	}
	response, err := engine.PostCollection(ctx, map[string]json.RawMessage{"ports": encoded}, neutronfind.Options{Headers: headers, Microversion: cfg.Microversion})
	result := bulkPortReceipt(response)
	if err != nil {
		return result, wrap(err)
	}
	rowsRaw, err := bulkResponseRows(response.Body)
	if err != nil {
		return result, wrap(response.Fail(err))
	}
	retained := &rest.Response{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	result.Records = func(yield func(*PortRecord, error) bool) {
		for _, row := range rowsRaw {
			if err := engine.Check(ctx); err != nil {
				yield(nil, wrap(retained.Fail(err)))
				return
			}
			value, err := engine.ProjectExistingRow(ctx, bytes.Clone(row), retained, neutronfind.ExistingRowPolicy{Bulk: true})
			if err != nil {
				yield(nil, wrap(err))
				return
			}
			if !yield(portListRecord(value), nil) {
				return
			}
		}
	}
	return result, wrap(engine.Check(ctx))
}
func bulkPortReceipt(response *rest.Response) *BulkPortResult {
	if response == nil {
		return nil
	}
	return &BulkPortResult{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func bulkResponseRows(body json.RawMessage) ([]json.RawMessage, error) {
	if !utf8.Valid(body) || !json.Valid(body) {
		return nil, findInvalid("bulk response must contain complete UTF-8 JSON")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil || envelope == nil {
		return nil, errors.Join(findInvalid("bulk response must be an object with ports"), err)
	}
	raw, present := envelope["ports"]
	if !present {
		return nil, findInvalid("bulk response missing ports")
	}
	token := bytes.TrimSpace(raw)
	switch token[0] {
	case '[':
		var rows []json.RawMessage
		if err := json.Unmarshal(token, &rows); err != nil {
			return nil, err
		}
		return rows, nil
	case '{':
		members, err := cloudfilter.ObjectMembers(token)
		if err != nil {
			return nil, err
		}
		rows := make([]json.RawMessage, len(members))
		for i, member := range members {
			rows[i], _ = json.Marshal(member.Key)
		}
		return rows, nil
	case '"':
		var text string
		if err := json.Unmarshal(token, &text); err != nil {
			return nil, err
		}
		rows := make([]json.RawMessage, 0)
		for _, char := range text {
			row, _ := json.Marshal(string(char))
			rows = append(rows, row)
		}
		return rows, nil
	default:
		return nil, findInvalid("bulk ports response must be iterable")
	}
}
