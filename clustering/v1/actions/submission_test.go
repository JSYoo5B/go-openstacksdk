package actions

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestSubmissionSnapshotOwnsAcceptedResponseEvidence(t *testing.T) {
	original := &Submission{
		ActionID: "accepted-action", Location: "https://cloud.example/v1/actions/accepted-action",
		Body:   json.RawMessage(`{"future":9007199254740993}`),
		Header: http.Header{"X-Request-Id": {"first", "second"}}, StatusCode: http.StatusAccepted,
	}
	copy := original.Snapshot()
	original.ActionID = "changed-input"
	original.Body[0] = '['
	original.Header["X-Request-Id"][0] = "changed-input"
	if copy.ActionID != "accepted-action" || string(copy.Body) != `{"future":9007199254740993}` || copy.Header.Values("X-Request-ID")[0] != "first" || copy.StatusCode != http.StatusAccepted {
		t.Fatal("input mutation changed accepted evidence", copy)
	}
	copy.Location = "changed-copy"
	copy.Body[1] = '!'
	copy.Header["X-Request-Id"][1] = "changed-copy"
	if original.Location != "https://cloud.example/v1/actions/accepted-action" || original.Body[1] != '"' || original.Header.Values("X-Request-ID")[1] != "second" {
		t.Fatal("copy mutation changed original evidence", original)
	}
	var absent *Submission
	if absent.Snapshot() != nil {
		t.Fatal("nil submission created accepted evidence")
	}
	if empty := (&Submission{}).Snapshot(); empty == nil || empty.Body != nil || empty.Header != nil || empty.StatusCode != 0 {
		t.Fatal("local empty submission fabricated evidence", empty)
	}
}
