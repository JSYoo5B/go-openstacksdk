package senlin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// TrackedState owns a cached body and a sticky dirty set. Equality is checked
// against the current value at assignment, rather than a historical baseline.
// New edits made during an HTTP request survive acceptance of its response.
// The SDK's resource packages own mutation permissions and HTTP requests.
type TrackedState struct {
	mu       sync.RWMutex
	body     map[string]json.RawMessage
	dirty    map[string]uint64
	revision uint64
}

// TrackedPatch is an owned request snapshot and its internal revision token.
// Body contains only dirty fields; deletion is represented by explicit null.
type TrackedPatch struct {
	Body      map[string]json.RawMessage
	owner     *TrackedState
	revisions map[string]uint64
}

// ValidateTrackedFields keeps extension fields out of known typed properties,
// including case variants that encoding/json would otherwise treat as aliases.
// Mutable typed properties must use the resource's concrete UpdateOpts.
func ValidateTrackedFields[T any](fields map[string]json.RawMessage) error {
	for _, field := range reflect.VisibleFields(reflect.TypeFor[T]()) {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if field.PkgPath != "" || name == "" || name == "-" {
			continue
		}
		for key := range fields {
			if strings.EqualFold(key, name) {
				return fmt.Errorf("%w: tracked extension %q is a typed property", resource.ErrInvalidOption, key)
			}
		}
	}
	return nil
}

func NewTrackedState(body map[string]json.RawMessage) (*TrackedState, error) {
	copied, err := copyTrackedBody(body)
	if err != nil {
		return nil, err
	}
	return &TrackedState{body: copied, dirty: make(map[string]uint64)}, nil
}

func (state *TrackedState) Body() map[string]json.RawMessage {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return cloneTrackedBody(state.body)
}

func (state *TrackedState) Dirty() bool {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return len(state.dirty) > 0
}

// Edit is atomic: invalid JSON does not apply a partial edit. Reassigning the
// current value does not create or clear dirty state. JSON numbers compare by
// exact decimal value; booleans remain distinct from numbers in this Go API.
func (state *TrackedState) Edit(fields map[string]json.RawMessage) error {
	copied, err := copyTrackedBody(fields)
	if err != nil {
		return err
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	for key, value := range copied {
		if current, exists := state.body[key]; exists && equalTrackedJSON(current, value) {
			continue
		}
		state.body[key] = value
		state.revision++
		state.dirty[key] = state.revision
	}
	return nil
}

// Remove removes a cached field and records a null mutation. Removing an absent
// field is a no-op, including when a previous removal is still dirty.
func (state *TrackedState) Remove(key string) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if _, exists := state.body[key]; !exists {
		return
	}
	delete(state.body, key)
	state.revision++
	state.dirty[key] = state.revision
}

func (state *TrackedState) Pending() TrackedPatch {
	state.mu.RLock()
	defer state.mu.RUnlock()
	patch := TrackedPatch{Body: make(map[string]json.RawMessage, len(state.dirty)), owner: state, revisions: make(map[string]uint64, len(state.dirty))}
	for key, revision := range state.dirty {
		value, exists := state.body[key]
		if !exists {
			value = json.RawMessage("null")
		}
		patch.Body[key] = append(json.RawMessage(nil), value...)
		patch.revisions[key] = revision
	}
	return patch
}

// Accept merges a valid response by whole fields and cleans the submitted
// revisions, including fields omitted by the response. It never clears or
// overwrites newer edits. HTTP failure must not call Accept.
func (state *TrackedState) Accept(patch TrackedPatch, response map[string]json.RawMessage) error {
	if patch.owner != state {
		return fmt.Errorf("%w: tracked revision belongs to another resource", resource.ErrInvalidOption)
	}
	copied, err := copyTrackedBody(response)
	if err != nil {
		return err
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	for key, value := range copied {
		if current, dirty := state.dirty[key]; dirty && current != patch.revisions[key] {
			continue
		}
		state.body[key] = value
	}
	for key, revision := range patch.revisions {
		if state.dirty[key] == revision {
			delete(state.dirty, key)
		}
	}
	return nil
}

// TrackedSeed prefers original response fields so omitted and explicit-null
// fields survive wrapping a decoded SDK model. A manually constructed model
// without Body is serialized as a local cache, with no fabricated HTTP code.
func TrackedSeed[T any](value *T, metadata *resource.Metadata) (map[string]json.RawMessage, error) {
	if value == nil || metadata == nil {
		return nil, fmt.Errorf("%w: tracked model is required", resource.ErrInvalidOption)
	}
	if metadata.Body != nil {
		return copyTrackedBody(metadata.Body)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: tracked model: %v", resource.ErrInvalidOption, err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &body); err != nil || body == nil {
		return nil, fmt.Errorf("%w: tracked model must be an object", resource.ErrInvalidOption)
	}
	return copyTrackedBody(body)
}

// TrackedDecode creates an owned typed view. Its Body is the supplied cache or
// actual response snapshot; Header and StatusCode identify the last successful
// HTTP response. Callers expose the two views through distinct lifecycle APIs.
func TrackedDecode[T any](body map[string]json.RawMessage, evidence *resource.Metadata, metadata func(*T) *resource.Metadata) (*T, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var value T
	if err := json.Unmarshal(encoded, &value); err != nil {
		return nil, err
	}
	if evidence != nil {
		info := metadata(&value)
		info.Header = evidence.Header.Clone()
		info.StatusCode = evidence.StatusCode
	}
	return &value, nil
}

func cloneTrackedBody(body map[string]json.RawMessage) map[string]json.RawMessage {
	copied := make(map[string]json.RawMessage, len(body))
	for key, value := range body {
		copied[key] = append(json.RawMessage(nil), value...)
	}
	return copied
}

func copyTrackedBody(body map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	for key, value := range body {
		if !json.Valid(value) {
			return nil, fmt.Errorf("%w: tracked field %q is not JSON", resource.ErrInvalidOption, key)
		}
	}
	return cloneTrackedBody(body), nil
}

func equalTrackedJSON(left, right json.RawMessage) bool {
	return EqualJSON(left, right)
}
