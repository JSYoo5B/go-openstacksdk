package tasks

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

func TestTaskWaitDecodeCanonicalFieldsAndExactNumbers(t *testing.T) {
	raw := json.RawMessage(`{"id":"canonical","ID":{},"status":"success","STATUS":[],"input":{"n":9007199254740993,"exponent":1e1000,"nested":[null,{"n":-9007199254740993}]},"INPUT":false,"result":{"n":9007199254740995},"created_at":"2024-01-02T03:04:05Z","CREATED_AT":false,"vendor":{"keep":true}}`)
	response := &rest.Response{Body: raw, Header: http.Header{"X-Proof": {"actual"}}, StatusCode: 200}
	task, fields, err := decodeTaskWait(response)
	if err != nil || task.ID != "canonical" || task.Status != "success" {
		t.Fatalf("decode = %#v %v", task, err)
	}
	if task.Input["n"] != json.Number("9007199254740993") || task.Input["exponent"] != json.Number("1e1000") || task.Result["n"] != json.Number("9007199254740995") {
		t.Fatalf("precision lost: %#v %#v", task.Input, task.Result)
	}
	if string(fields["vendor"]) != `{"keep":true}` || string(response.Body) != string(raw) {
		t.Fatalf("raw fields changed: %#v", fields)
	}
	result := &TaskWaitResult{}
	result.observed(response, task)
	created := taskWaitResponse(response, task)
	task.Input["n"] = "mutated"
	task.Input["nested"].([]any)[1].(map[string]any)["n"] = "mutated"
	if created.Task.Input["n"] != json.Number("9007199254740993") || created.Task.Input["nested"].([]any)[1].(map[string]any)["n"] != json.Number("-9007199254740993") {
		t.Fatal("latest created typed proof aliases outer task maps")
	}
	response.Body[0], response.Header["X-Proof"][0] = '!', "changed"
	if result.Body[0] != '{' || result.Header.Get("X-Proof") != "actual" || created.Body[0] != '{' || created.Header.Get("X-Proof") != "actual" {
		t.Fatal("retained HTTP evidence aliases its source")
	}
}

func TestTaskWaitDecodeRejectsNativeShapesWithAcceptedEvidence(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{]`, `{"input":[]}`, `{"input":3}`, `{"result":false}`, `{"status":{}}`, `{"created_at":"old-no-z"}`, "{\"input\":{\"key\":\"\xff\"}}"} {
		t.Run(raw, func(t *testing.T) {
			response := &rest.Response{Body: json.RawMessage(raw), Header: http.Header{"X-Proof": {"actual"}}, StatusCode: 201}
			_, _, err := decodeTaskWait(response)
			var evidence *resource.ResponseError
			if !errors.As(err, &evidence) || evidence.StatusCode != 201 || string(evidence.Body) != raw || evidence.Header.Get("X-Proof") != "actual" {
				t.Fatalf("accepted evidence missing: %v", err)
			}
			if raw == `{"created_at":"old-no-z"}` {
				var cause *time.ParseError
				if !errors.As(err, &cause) {
					t.Fatalf("native timestamp cause missing: %v", err)
				}
			}
		})
	}
	for _, fields := range []map[string]json.RawMessage{{}, {"status": json.RawMessage(`null`)}, {"status": json.RawMessage(`3`)}} {
		if _, err := taskWaitString(fields, "status"); err == nil {
			t.Fatal("missing/null/nonstring decision accepted")
		}
	}
	if value, err := taskWaitString(map[string]json.RawMessage{"status": json.RawMessage(`""`)}, "status"); err != nil || value != "" {
		t.Fatalf("empty source string collapsed: %q %v", value, err)
	}
}
