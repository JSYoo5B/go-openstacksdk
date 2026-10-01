package shareaccessrules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"gophercloudsdk/resource"
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
	// Only exact wire identity keys establish IDs. Keep a valid created ID even
	// when a later field fails; never let encoding/json's case folding select it.
	if err := json.Unmarshal(body["id"], &value.ID); err != nil {
		return value, fmt.Errorf("access rule id must be a JSON string: %w", err)
	}
	if err := resource.ID(value.ID).Validate(); err != nil {
		return value, fmt.Errorf("access rule response has an invalid id: %w", err)
	}
	if rawShare, exists := body["share_id"]; exists {
		if bytes.Equal(bytes.TrimSpace(rawShare), []byte("null")) {
			return value, fmt.Errorf("access rule share_id must be a JSON string")
		}
		if err := json.Unmarshal(rawShare, &value.ShareID); err != nil {
			return value, fmt.Errorf("access rule share_id must be a JSON string: %w", err)
		}
		if value.ShareID == "" {
			return value, &ParentMismatchError{ShareID: parent, AccessID: value.ID}
		}
		if err := resource.ID(value.ShareID).Validate(); err != nil {
			return value, fmt.Errorf("access rule response has an invalid share_id: %w", err)
		}
	}
	nativeBody, err := withoutRuleIdentity(raw)
	if err != nil {
		return value, err
	}
	var native ShareAccess
	if err := json.Unmarshal(nativeBody, &native); err != nil {
		return value, err
	}
	native.ID, native.ShareID = value.ID, value.ShareID
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
	return value, nil
}

// Preserve field order and native decoding for non-identity fields, while
// excluding both canonical and case-variant identities from native decoding.
// Body retains every original field, including ignored identity aliases.
func withoutRuleIdentity(raw json.RawMessage) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	result := json.RawMessage{'{'}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("access rule property must have a string key")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		if strings.EqualFold(key, "id") || strings.EqualFold(key, "share_id") {
			continue
		}
		if len(result) != 1 {
			result = append(result, ',')
		}
		encodedKey, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		result = append(result, encodedKey...)
		result = append(result, ':')
		result = append(result, value...)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return append(result, '}'), nil
}
