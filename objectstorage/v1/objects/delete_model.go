package objects

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// DeleteObjectResponse retains the actual response of one completed phase.
type DeleteObjectResponse struct {
	Body       []byte
	Header     http.Header
	StatusCode int
}

// DeleteObjectResult keeps discovery and deletion observations separate.
// StaticLargeObject is unknown after a missing discovery response. A nil Bulk
// on an SLO 202/204 acknowledgement means segment counts are unknown.
type DeleteObjectResult struct {
	Discovery, Deletion *DeleteObjectResponse
	StaticLargeObject   *bool
	IgnoredMissing      bool
	Bulk                *ObjectDeleteBulkInfo
}

// ObjectDeleteBulkFailure preserves the encoded object name and arbitrary
// server message. Error need not be an HTTP status line.
type ObjectDeleteBulkFailure struct {
	Name, Error string
}

// ObjectDeleteBulkInfo retains an SLO deletion's embedded report. Its counts
// describe subrequests; NumberNotFound does not mean the top-level request was
// ignored. Inherited dates and links remain passive raw fields in Body.
type ObjectDeleteBulkInfo struct {
	resource.Metadata
	ResponseStatus string
	ResponseBody   string
	ResponseCode   int
	NumberDeleted  int64
	NumberNotFound int64
	Errors         []ObjectDeleteBulkFailure
}

// UnmarshalJSON atomically projects the five required bulk report fields.
// Unknown fields and exact number tokens remain independently owned in Body.
func (b *ObjectDeleteBulkInfo) UnmarshalJSON(data []byte) error {
	if b == nil {
		return fmt.Errorf("object delete bulk decoder requires a destination")
	}
	data = bytes.TrimSpace(data)
	if !utf8.Valid(data) || len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("object delete bulk report must be a UTF-8 JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	value := ObjectDeleteBulkInfo{Metadata: resource.Metadata{Body: fields}}
	var err error
	if value.ResponseStatus, err = objectDeleteBulkString(fields, "Response Status"); err != nil {
		return err
	}
	if value.ResponseCode, err = objectDeleteBulkStatus(value.ResponseStatus); err != nil {
		return err
	}
	if value.ResponseBody, err = objectDeleteBulkString(fields, "Response Body"); err != nil {
		return err
	}
	if value.NumberDeleted, err = objectDeleteBulkCount(fields, "Number Deleted"); err != nil {
		return err
	}
	if value.NumberNotFound, err = objectDeleteBulkCount(fields, "Number Not Found"); err != nil {
		return err
	}
	raw := bytes.TrimSpace(fields["Errors"])
	if len(raw) == 0 || raw[0] != '[' {
		return fmt.Errorf("object delete bulk Errors must be an array")
	}
	var pairs [][]json.RawMessage
	if err := json.Unmarshal(raw, &pairs); err != nil {
		return err
	}
	value.Errors = make([]ObjectDeleteBulkFailure, len(pairs))
	for n, pair := range pairs {
		if len(pair) != 2 {
			return fmt.Errorf("object delete bulk Errors entry must contain two strings")
		}
		parts := map[string]json.RawMessage{"name": pair[0], "error": pair[1]}
		if value.Errors[n].Name, err = objectDeleteBulkString(parts, "name"); err != nil {
			return err
		}
		if value.Errors[n].Error, err = objectDeleteBulkString(parts, "error"); err != nil {
			return err
		}
	}
	*b = value
	return nil
}

func objectDeleteBulkString(fields map[string]json.RawMessage, name string) (string, error) {
	raw := bytes.TrimSpace(fields[name])
	if len(raw) == 0 || raw[0] != '"' {
		return "", fmt.Errorf("object delete bulk %q must be a string", name)
	}
	var value string
	err := json.Unmarshal(raw, &value)
	return value, err
}
func objectDeleteBulkCount(fields map[string]json.RawMessage, name string) (int64, error) {
	value, err := strconv.ParseInt(string(bytes.TrimSpace(fields[name])), 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("object delete bulk %q must be a nonnegative int64", name)
	}
	return value, nil
}
func objectDeleteBulkStatus(value string) (int, error) {
	invalid := func() (int, error) { return 0, fmt.Errorf("invalid object delete bulk Response Status") }
	if len(value) < 3 {
		return invalid()
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return invalid()
		}
	}
	for n, b := range []byte(value) {
		if b < 32 || b == 127 || n < 3 && (b < '0' || b > '9') {
			return invalid()
		}
	}
	code, _ := strconv.Atoi(value[:3])
	if code < 100 || code > 599 || len(value) > 3 && (value[3] != ' ' || strings.TrimSpace(value[4:]) == "") {
		return invalid()
	}
	return code, nil
}

// ObjectDeleteBulkError reports an embedded failure in an actual HTTP 200
// response. Its data is independent of the caller's mutable Bulk result.
type ObjectDeleteBulkError struct {
	ResponseStatus string
	ResponseBody   string
	ResponseCode   int
	Errors         []ObjectDeleteBulkFailure
}

func (e *ObjectDeleteBulkError) Error() string {
	return fmt.Sprintf("object delete bulk %s: %s (%d errors)", e.ResponseStatus, e.ResponseBody, len(e.Errors))
}
