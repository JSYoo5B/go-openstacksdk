package actions

import (
	"encoding/json"
	"net/http"
)

// Submission records an accepted asynchronous request. ActionID identifies an
// action that can be fetched or waited on through API; submission alone does
// not imply completion or fabricate action status. Body and Header belong to
// this request rather than a subsequently fetched Action.
type Submission struct {
	ActionID   string
	Location   string
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}
