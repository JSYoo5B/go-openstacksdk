package shareaccessrules

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrParentMismatch    = errors.New("access rule belongs to a different share")
	ErrParentUnavailable = errors.New("access rule parent share is unavailable")
)

type ParentMismatchError struct {
	ShareID, AccessID, ActualShareID string
}

func (e *ParentMismatchError) Error() string {
	return fmt.Sprintf("%v: rule %s belongs to %q, scope is %q", ErrParentMismatch, e.AccessID, e.ActualShareID, e.ShareID)
}
func (e *ParentMismatchError) Unwrap() error { return ErrParentMismatch }

// ParentError keeps a parent lookup failure distinct from a missing rule.
type ParentError struct {
	ShareID string
	Cause   error
}

func (e *ParentError) Error() string {
	return fmt.Sprintf("share %s: %v: %v", e.ShareID, ErrParentUnavailable, e.Cause)
}
func (e *ParentError) Is(target error) bool { return target == ErrParentUnavailable }
func (e *ParentError) Unwrap() error        { return e.Cause }

// AccessRule preserves native fields, optional 2.82 locks, and the raw response.
// ParentShareID is the scope's ID; ShareID remains the actual response field,
// which Manila can omit in list responses.
type AccessRule struct {
	ShareAccess
	ParentShareID  string
	LockVisibility *bool
	LockDeletion   *bool
	LockReason     *string
	Body           map[string]json.RawMessage
	Header         http.Header
}

func decodeRule(raw json.RawMessage, header http.Header, parent string) (*AccessRule, error) {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("access rule must be a JSON object")
	}
	value := &AccessRule{ParentShareID: parent, Body: body, Header: header.Clone()}
	// Preserve a known identity even if decoding a later field fails after allow.
	_ = json.Unmarshal(body["id"], &value.ID)
	_ = json.Unmarshal(body["share_id"], &value.ShareID)
	var native ShareAccess
	if err := json.Unmarshal(raw, &native); err != nil {
		return value, err
	}
	value.ShareAccess = native
	var locks struct {
		Visibility *bool   `json:"lock_visibility"`
		Deletion   *bool   `json:"lock_deletion"`
		Reason     *string `json:"lock_reason"`
	}
	if err := json.Unmarshal(raw, &locks); err != nil {
		return value, err
	}
	value.LockVisibility, value.LockDeletion, value.LockReason = locks.Visibility, locks.Deletion, locks.Reason
	if value.ID == "" {
		return value, fmt.Errorf("access rule response has no id")
	}
	return value, nil
}
