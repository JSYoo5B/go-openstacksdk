package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/jsonfilter"
)

// matchLocal follows Resource.list's ordered Body predicates, not cloud search
// mapping membership. Missing object fields are None; only reached dictionary
// predicates call .get, so truthy scalars match {} but fail nonempty mappings.
func matchLocal(ctx context.Context, view json.RawMessage, filters []cloudfilter.JSONMember, guard func() error) (matched bool, err error) {
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
		matched, err := snapshotLocalValue(value, filter.Value, check)
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

func snapshotLocalValue(actual, expected json.RawMessage, check func() error) (bool, error) {
	if err := check(); err != nil {
		return false, err
	}
	if err := snapshotQueryDocument(expected); err != nil {
		return false, err
	}
	if bytes.TrimSpace(expected)[0] == '{' {
		return snapshotLocalDictionary(actual, expected, check)
	}
	equal, err := jsonfilter.EqualPythonJSON(actual, expected)
	if err != nil {
		return false, err
	}
	return equal, check()
}

func snapshotLocalDictionary(actual, expected json.RawMessage, check func() error) (bool, error) {
	if err := check(); err != nil {
		return false, err
	}
	truthy, err := cloudfilter.PythonTruthy(actual)
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
		matched, err := snapshotLocalValue(value, predicate.Value, check)
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
