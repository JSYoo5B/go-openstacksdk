package senlin

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestTrackedCurrentComparisonStickyDirtyAndRemoval(t *testing.T) {
	state, err := NewTrackedState(map[string]json.RawMessage{"name": json.RawMessage(`"A"`), "metadata": json.RawMessage(`{"big":9007199254740993,"off":false}`)})
	if err != nil {
		t.Fatal(err)
	}
	for _, fields := range []map[string]json.RawMessage{{}, {"name": json.RawMessage(`"A"`)}, {"metadata": json.RawMessage(`{ "off":false,"big":9007199254740993.0 }`)}} {
		if err := state.Edit(fields); err != nil || state.Dirty() {
			t.Fatal("same value created dirty state", err, state.Pending())
		}
	}
	for _, name := range []string{`"B"`, `"A"`, `"A"`} {
		if err := state.Edit(map[string]json.RawMessage{"name": json.RawMessage(name)}); err != nil {
			t.Fatal(err)
		}
	}
	if !state.Dirty() || string(state.Pending().Body["name"]) != `"A"` {
		t.Fatal("reversion cleared sticky dirty state", state.Pending())
	}
	state.Remove("metadata")
	state.Remove("missing")
	if _, exists := state.Body()["metadata"]; exists || string(state.Pending().Body["metadata"]) != "null" || len(state.Pending().Body) != 2 {
		t.Fatal(state.Body(), state.Pending())
	}
	if err := state.Accept(state.Pending(), map[string]json.RawMessage{}); err != nil || state.Dirty() {
		t.Fatal("valid empty response did not clean submitted fields", err, state.Pending())
	}
	state.Remove("metadata")
	if state.Dirty() {
		t.Fatal("removing missing field created new dirty state")
	}
}

func TestTrackedOwnsAllSnapshotsAndRejectedEditIsAtomic(t *testing.T) {
	raw := json.RawMessage(`{"big":9007199254740993}`)
	seed := map[string]json.RawMessage{"vendor": raw}
	state, err := NewTrackedState(seed)
	if err != nil {
		t.Fatal(err)
	}
	raw[2] = 'X'
	seed["name"] = json.RawMessage(`"outside"`)
	if string(state.Body()["vendor"]) != `{"big":9007199254740993}` || len(state.Body()) != 1 {
		t.Fatal(state.Body())
	}
	owned := state.Body()
	owned["vendor"][2] = 'Y'
	if err := state.Edit(map[string]json.RawMessage{"name": json.RawMessage(`"A"`), "invalid": json.RawMessage(`{`)}); !errors.Is(err, resource.ErrInvalidOption) || state.Dirty() {
		t.Fatal("invalid edit partly applied", err, state.Body())
	}
	input := map[string]json.RawMessage{"vendor": json.RawMessage(`{"big":9007199254740995}`)}
	if err := state.Edit(input); err != nil {
		t.Fatal(err)
	}
	input["vendor"][2] = 'Z'
	pending := state.Pending()
	pending.Body["vendor"][2] = 'Q'
	if string(state.Body()["vendor"]) != `{"big":9007199254740995}` {
		t.Fatal("mutable snapshot aliases state", state.Body())
	}
}

func TestTrackedAcceptShallowMergeAndPreservesNewerEdits(t *testing.T) {
	state, _ := NewTrackedState(map[string]json.RawMessage{"name": json.RawMessage(`"A"`), "spec": json.RawMessage(`{"old":true}`), "metadata": json.RawMessage(`{"a":1,"b":2}`)})
	_ = state.Edit(map[string]json.RawMessage{"name": json.RawMessage(`"submitted"`), "metadata": json.RawMessage(`{"a":3,"b":4}`)})
	pending := state.Pending()
	_ = state.Edit(map[string]json.RawMessage{"name": json.RawMessage(`"newer"`), "vendor": json.RawMessage(`false`)})
	response := map[string]json.RawMessage{"name": json.RawMessage(`"server"`), "metadata": json.RawMessage(`{"a":5}`), "vendor": json.RawMessage(`true`)}
	if err := state.Accept(pending, response); err != nil {
		t.Fatal(err)
	}
	response["metadata"][2] = 'X'
	body, dirty := state.Body(), state.Pending().Body
	if string(body["name"]) != `"newer"` || string(body["spec"]) != `{"old":true}` || string(body["metadata"]) != `{"a":5}` || string(body["vendor"]) != "false" || len(dirty) != 2 {
		t.Fatal(body, dirty)
	}
	if _, exists := dirty["metadata"]; exists {
		t.Fatal("accepted old revision stayed dirty", dirty)
	}
	other, _ := NewTrackedState(nil)
	if err := state.Accept(other.Pending(), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("accepted another handle's revision token", err)
	}
	if err := state.Accept(state.Pending(), map[string]json.RawMessage{"bad": json.RawMessage(`{`)}); !errors.Is(err, resource.ErrInvalidOption) || !state.Dirty() {
		t.Fatal("invalid response cleared changes", err, state.Pending())
	}
}
