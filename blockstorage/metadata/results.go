// Package metadata contains owned Cinder metadata results and header options.
package metadata

import (
	"encoding/json"
	"net/http"
)

// Result is the actual metadata response, independent of a Volume or Snapshot.
type Result struct {
	Metadata   map[string]string
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// DeletedKey records one successful DELETE. Key is the requested target.
type DeletedKey struct {
	Key        string
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// DeleteResult records an actual clear PUT or ordered successful key DELETEs.
// A partial result accompanies the first failed key; it is not a transaction.
type DeleteResult struct {
	Cleared *Result
	Deleted []DeletedKey
}

// Deletion is an alias for DeleteResult.
type Deletion = DeleteResult
