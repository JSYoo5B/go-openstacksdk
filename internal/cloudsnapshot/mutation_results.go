package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// MutationPage owns actual physical bytes, including opaque acknowledgements
// and a clean polling404. Each result phase has an independent copy.
type MutationPage struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// CreateResult keeps merged logical stages separate from physical responses.
// Value/Snapshot are committed only after the requested workflow succeeds.
// A tolerated empty/malformed body has a Page and no physical RawResource.
type CreateResult struct {
	SnapshotID                 json.RawMessage
	Created, LastAccepted      *MutationPage
	CreatedValue, Value, Ready json.RawMessage
	CreatedSnapshot, Snapshot  *resource.RawResource
	ReadySnapshot              *resource.RawResource
}

// DeleteResult maps successful absence to false and completed mutation to
// true. Deleted remains nil on error; Resolved/Applied retain earlier phases.
// Absent is clean GET404 proof, never evidence from a rejected DELETE404.
type DeleteResult struct {
	SnapshotID                    json.RawMessage
	Deleted                       *bool
	Resolved                      *Result
	Applied, LastAccepted, Absent *MutationPage
	Ready                         json.RawMessage
	ReadySnapshot                 *resource.RawResource
}

// WaitTimeoutError reports the SDK loop deadline without deriving an HTTP
// context deadline. Parent context cancellation is still honored separately.
type WaitTimeoutError struct{ Timeout time.Duration }

func (e *WaitTimeoutError) Error() string {
	return fmt.Sprintf("volume snapshot wait exceeded %s", e.Timeout)
}
func (e *WaitTimeoutError) Unwrap() error { return context.DeadlineExceeded }

func mutationProof(wire *rest.Response) *MutationPage {
	if wire == nil {
		return nil
	}
	return &MutationPage{Body: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
}

func cloneMutationProof(page *MutationPage) *MutationPage {
	if page == nil {
		return nil
	}
	return &MutationPage{Body: bytes.Clone(page.Body), Header: page.Header.Clone(), StatusCode: page.StatusCode}
}
