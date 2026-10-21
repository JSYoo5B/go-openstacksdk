package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// ConsoleState is the console state of a node. Info keeps the console
// connection information exactly as the server returned it, and Body keeps the
// complete response document.
type ConsoleState struct {
	Enabled bool            `json:"console_enabled"`
	Info    json.RawMessage `json:"console_info"`
	Header  http.Header     `json:"-"`
	Body    json.RawMessage `json:"-"`
}

func nodeSubPath(operation, id string, segments ...string) ([]string, error) {
	if id == "" {
		return nil, request.Wrap(operation, "nodes", errors.New("node ID or name must not be empty"))
	}
	return append([]string{"nodes", url.PathEscape(id)}, segments...), nil
}

// GetConsole reads GET /nodes/{id}/states/console. Only 200 succeeds. The
// call sends the client microversion only; it never selects one itself.
func (a *API) GetConsole(ctx context.Context, id string) (*ConsoleState, error) {
	segments, err := nodeSubPath("GetConsole", id, "states", "console")
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	resp, err := a.client.Get(ctx, a.client.ServiceURL(segments...), &raw, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return nil, request.Wrap("GetConsole", "nodes", err)
	}
	state := ConsoleState{Header: resp.Header.Clone(), Body: raw}
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, request.Wrap("GetConsole", "nodes", err)
	}
	return &state, nil
}

// SetConsoleMode sends PUT /nodes/{id}/states/console with {"enabled": value}.
// The explicit false value is always sent. 200, 202 and 204 are accepted; the
// call does not wait for the console to change state.
func (a *API) SetConsoleMode(ctx context.Context, id string, enabled bool) error {
	segments, err := nodeSubPath("SetConsoleMode", id, "states", "console")
	if err != nil {
		return err
	}
	_, err = a.client.Put(ctx, a.client.ServiceURL(segments...), map[string]any{"enabled": enabled}, nil,
		&gophercloud.RequestOpts{OkCodes: []int{200, 202, 204}})
	return request.Wrap("SetConsoleMode", "nodes", err)
}

// AddTrait sends PUT /nodes/{id}/traits/{trait} without a body. Only 204
// succeeds. The trait is path-escaped. A failed or uncertain call is never
// replayed automatically.
func (a *API) AddTrait(ctx context.Context, id, trait string) error {
	if trait == "" {
		return request.Wrap("AddTrait", "nodes", errors.New("trait must not be empty"))
	}
	segments, err := nodeSubPath("AddTrait", id, "traits", url.PathEscape(trait))
	if err != nil {
		return err
	}
	_, err = a.client.Put(ctx, a.client.ServiceURL(segments...), nil, nil, &gophercloud.RequestOpts{OkCodes: []int{204}})
	return request.Wrap("AddTrait", "nodes", err)
}
