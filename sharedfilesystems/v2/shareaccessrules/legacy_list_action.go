package shareaccessrules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/manilaversion"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Manila share action names for the legacy access listing. Manila routes
// os-access_list for 2.0-2.6 and access_list for 2.7-2.44; from 2.45 the
// action is removed in favor of GET /share-access-rules?share_id=.
const (
	LegacyAccessListActionV1 = "os-access_list"
	LegacyAccessListAction   = "access_list"
)

// AccessListActionResult is one accepted legacy listing. Rules keep the scope
// share in ParentShareID; Manila's legacy rows carry no share_id, so each
// AccessRule.ShareID stays empty unless the server actually returned one.
// Body, Header and StatusCode are the whole accepted HTTP response.
type AccessListActionResult struct {
	ShareID    string
	Action     string
	Rules      []*AccessRule
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// ListAccessAction lists a share's access rules through the legacy share
// action POST /shares/{id}/action with a null action value. The selected
// client microversion chooses the action name before HTTP: empty (2.0) to 2.6
// sends os-access_list, 2.7 to 2.44 sends access_list, and 2.45 or newer fails
// with resource.ErrUnsupported because Manila removed the action; use
// API.InShare and AccessRuleScope.List there. Symbolic latest, conflicting
// version headers and non-Manila client types fail before HTTP (see
// internal/manilaversion). The share must be given by ID; no parent lookup is
// made. Only HTTP 200 succeeds. The action is a read and is sent once per call
// without local paging, filtering or caps; the whole collection is decoded
// before it is returned. A row whose share_id names another share is rejected
// with ErrParentMismatch, and decode failures keep the accepted response.
func (a *API) ListAccessAction(ctx context.Context, shareID string) (*AccessListActionResult, error) {
	const op, kind = "ListAccessAction", "share_access_rules"
	if err := a.actionPreflight(ctx, shareID); err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	action, err := legacyAccessListActionName(a)
	if err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	response, err := rest.DoJSON(ctx, a.client, http.MethodPost, a.client.ServiceURL("shares", url.PathEscape(shareID), "action"), map[string]any{action: nil}, nil, 200)
	if err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	result, err := decodeAccessListAction(response, shareID, action)
	if err != nil {
		return nil, request.Wrap(op, kind, response.Fail(err))
	}
	return result, nil
}

func legacyAccessListActionName(a *API) (string, error) {
	minor, err := manilaversion.Minor(a.client)
	if err != nil {
		return "", err
	}
	switch {
	case minor <= 6:
		return LegacyAccessListActionV1, nil
	case minor <= 44:
		return LegacyAccessListAction, nil
	default:
		return "", fmt.Errorf("%w: the legacy access_list share action was removed in Manila 2.45, selected %s; use InShare(...).List", resource.ErrUnsupported, a.client.Microversion)
	}
}

func decodeAccessListAction(response *rest.Response, shareID, action string) (*AccessListActionResult, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &envelope); err != nil {
		return nil, err
	}
	if envelope == nil {
		return nil, fmt.Errorf("access_list response must be a JSON object")
	}
	rawRows, exists := envelope["access_list"]
	if !exists || bytes.Equal(bytes.TrimSpace(rawRows), []byte("null")) {
		return nil, fmt.Errorf("access_list response must contain an access_list array")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(rawRows, &rows); err != nil {
		return nil, fmt.Errorf("access_list must be an array: %w", err)
	}
	result := &AccessListActionResult{
		ShareID: shareID, Action: action, Rules: make([]*AccessRule, 0, len(rows)),
		Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode,
	}
	for _, row := range rows {
		rule, err := decodeRule(row, response.Header, shareID)
		if err != nil {
			return nil, err
		}
		if rule.ShareID != "" && rule.ShareID != shareID {
			return nil, &ParentMismatchError{ShareID: shareID, AccessID: rule.ID, ActualShareID: rule.ShareID}
		}
		result.Rules = append(result.Rules, rule)
	}
	return result, nil
}
