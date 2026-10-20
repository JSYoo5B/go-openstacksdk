package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

// MatchLocal shares ordered Cinder Resource Body predicates with owned leaf APIs.
// Its reviewed exact-decimal policy is unchanged; new Source-shaped readers can
// explicitly select MatchLocalLoaded without changing existing callers.
func MatchLocal(ctx context.Context, view json.RawMessage, filters []cloudfilter.JSONMember, guard func() error) (bool, error) {
	return matchLocal(ctx, view, filters, guard)
}

// MatchLocalLoaded applies CPython JSON-loaded numeric equality and truthiness
// to the same ordered Resource.list predicates and guarded recursive flow.
// Integer tokens stay exact; decimal/exponent tokens use IEEE-754 floats.
func MatchLocalLoaded(ctx context.Context, view json.RawMessage, filters []cloudfilter.JSONMember, guard func() error) (bool, error) {
	return matchLocalWithPolicy(ctx, view, filters, guard, snapshotLocalPolicy{loaded: true})
}

type snapshotLocalPolicy struct{ loaded bool }

func matchLocal(ctx context.Context, view json.RawMessage, filters []cloudfilter.JSONMember, guard func() error) (bool, error) {
	return matchLocalWithPolicy(ctx, view, filters, guard, snapshotLocalPolicy{})
}

// Missing object fields are None; only reached dictionary predicates call .get,
// so truthy scalars match {} but fail nonempty mappings. Both policies use this
// flow, including snapshots, short-circuit order and final source/context guard.
func matchLocalWithPolicy(ctx context.Context, view json.RawMessage, filters []cloudfilter.JSONMember, guard func() error, policy snapshotLocalPolicy) (matched bool, err error) {
	if err := cloudread.Context(ctx); err != nil {
		return false, err
	}
	ownedView := json.RawMessage(bytes.Clone(view))
	ownedFilters := make([]cloudfilter.JSONMember, len(filters))
	for i, filter := range filters {
		ownedFilters[i] = cloudfilter.JSONMember{Key: filter.Key, Value: bytes.Clone(filter.Value)}
	}
	check := func() error {
		var err error
		if guard != nil {
			err = guard()
		}
		return cloudread.ContextError(ctx, err)
	}
	defer func() {
		err = errors.Join(err, check())
		if err != nil {
			matched = false
		}
	}()
	if err := check(); err != nil {
		return false, err
	}
	actual, err := cloudfilter.ObjectMembers(ownedView)
	if err != nil {
		return false, fmt.Errorf("Cinder resource local-filter view: %w", err)
	}
	values := make(map[string]json.RawMessage, len(actual))
	for _, member := range actual {
		values[member.Key] = member.Value
	}
	for _, filter := range ownedFilters {
		if err := check(); err != nil {
			return false, err
		}
		value, exists := values[filter.Key]
		if !exists {
			value = json.RawMessage("null")
		}
		matched, err := policy.value(value, filter.Value, check)
		if err != nil {
			return false, fmt.Errorf("Cinder resource local filter %q: %w", filter.Key, err)
		}
		if err := check(); err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}
	return true, check()
}

func (policy snapshotLocalPolicy) value(actual, expected json.RawMessage, check func() error) (bool, error) {
	if err := check(); err != nil {
		return false, err
	}
	if err := policy.document(expected); err != nil {
		return false, err
	}
	if bytes.TrimSpace(expected)[0] == '{' {
		return policy.dictionary(actual, expected, check)
	}
	equal, err := policy.equal(actual, expected)
	if err != nil {
		return false, err
	}
	return equal, check()
}

func (policy snapshotLocalPolicy) dictionary(actual, expected json.RawMessage, check func() error) (bool, error) {
	if err := check(); err != nil {
		return false, err
	}
	truthy, err := policy.truthy(actual)
	if err != nil {
		return false, err
	}
	if !truthy {
		return false, check()
	}
	predicates, err := cloudfilter.ObjectMembers(expected)
	if err != nil {
		return false, err
	}
	if len(predicates) == 0 {
		return true, check()
	}
	values, err := cloudfilter.ObjectMembers(actual)
	if err != nil {
		return false, fmt.Errorf("truthy local-filter value has no mapping get: %w", err)
	}
	fields := make(map[string]json.RawMessage, len(values))
	for _, value := range values {
		fields[value.Key] = value.Value
	}
	for _, predicate := range predicates {
		if err := check(); err != nil {
			return false, err
		}
		value, exists := fields[predicate.Key]
		if !exists {
			value = json.RawMessage("null")
		}
		matched, err := policy.value(value, predicate.Value, check)
		if err != nil {
			return false, fmt.Errorf("nested local filter %q: %w", predicate.Key, err)
		}
		if err := check(); err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}
	return true, check()
}

func (policy snapshotLocalPolicy) equal(actual, expected json.RawMessage) (bool, error) {
	if policy.loaded {
		return jsonfilter.EqualPythonLoadedJSON(actual, expected)
	}
	return jsonfilter.EqualPythonJSON(actual, expected)
}
func (policy snapshotLocalPolicy) truthy(actual json.RawMessage) (bool, error) {
	if policy.loaded {
		return cloudfilter.PythonJSONTruthy(actual)
	}
	return cloudfilter.PythonTruthy(actual)
}
func (policy snapshotLocalPolicy) document(raw json.RawMessage) error {
	if !policy.loaded {
		return snapshotQueryDocument(raw)
	}
	if len(raw) == 0 {
		return fmt.Errorf("explicit raw value must be complete JSON")
	}
	_, err := cloudfilter.PythonJSONTruthy(raw)
	return err
}
