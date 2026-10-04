package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/resource"
)

func snapshotLocalContractMembers(t *testing.T, raw string) []cloudfilter.JSONMember {
	t.Helper()
	members, err := cloudfilter.ObjectMembers(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	return members
}

func TestSnapshotLocalFiltersUseMissingNoneTruthyDictionaryAndExactLeafEquality(t *testing.T) {
	for _, tc := range []struct {
		view, filters   string
		matched, failed bool
	}{
		{`{}`, `{"missing":null}`, true, false},
		{`{}`, `{"missing":{}}`, false, false},
		{`{"metadata":{}}`, `{"metadata":{}}`, false, false},
		{`{"description":false}`, `{"description":{}}`, false, false},
		{`{"description":0}`, `{"description":{}}`, false, false},
		{`{"description":[]}`, `{"description":{}}`, false, false},
		{`{"description":true}`, `{"description":{}}`, true, false},
		{`{"description":"truthy"}`, `{"description":{}}`, true, false},
		{`{"description":[1]}`, `{"description":{}}`, true, false},
		{`{"description":true}`, `{"description":{"missing":null}}`, false, true},
		{`{"metadata":{"extra":1}}`, `{"metadata":{"missing":null}}`, true, false},
		{`{"metadata":{"nested":{"a":true,"large":900719925474099312345,"extra":1},"array":[{"x":false},1]}}`, `{"metadata":{"nested":{"a":1,"large":900719925474099312345},"array":[{"x":0},true]}}`, true, false},
		{`{"metadata":{"array":[{"x":false,"extra":1}]}}`, `{"metadata":{"array":[{"x":0}]}}`, false, false},
		{`{"metadata":{"large":900719925474099312346}}`, `{"metadata":{"large":900719925474099312345}}`, false, false},
	} {
		t.Run(tc.filters+tc.view, func(t *testing.T) {
			matched, err := matchLocal(context.Background(), json.RawMessage(tc.view), snapshotLocalContractMembers(t, tc.filters), nil)
			if (err != nil) != tc.failed || matched != tc.matched {
				t.Fatal(matched, err, tc)
			}
		})
	}
}

func TestSnapshotLocalFiltersRetainParsedOrderAndDoNotConsumeUnusedShapeErrors(t *testing.T) {
	for _, tc := range []struct {
		filters string
		failed  bool
	}{
		{`{"description":"drop","metadata":{"x":1}}`, false},
		{`{"metadata":{"x":1},"description":"drop"}`, true},
		{`{"description":"original","metadata":{"x":1},"description":"drop"}`, false},
		{`{"metadata":{"first":false,"later":{"x":1}}}`, false},
		{`{"metadata":{"later":{"x":1},"first":false}}`, true},
	} {
		view := json.RawMessage(`{"description":"keep","metadata":true}`)
		if bytes.Contains([]byte(tc.filters), []byte(`"later"`)) {
			view = json.RawMessage(`{"description":"keep","metadata":{"first":true,"later":true}}`)
		}
		policy := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(tc.filters)})
		matched, err := matchLocal(context.Background(), view, policy.local, nil)
		if matched || (err != nil) != tc.failed {
			t.Fatal(tc, matched, err)
		}
	}
	malformedTail := []cloudfilter.JSONMember{{Key: "description", Value: json.RawMessage(`"drop"`)}, {Key: "metadata", Value: json.RawMessage(`{`)}}
	matched, err := matchLocal(context.Background(), json.RawMessage(`{"description":"keep"}`), malformedTail, nil)
	if matched || err != nil {
		t.Fatal("unreached predicate was consumed", matched, err)
	}
	malformedTail[0].Value = json.RawMessage(`"keep"`)
	matched, err = matchLocal(context.Background(), json.RawMessage(`{"description":"keep"}`), malformedTail, nil)
	if matched || err == nil {
		t.Fatal("reached invalid predicate did not fail", matched, err)
	}
}

func TestSnapshotLocalFiltersOwnInputsBeforeGuardAndJoinContextWithSourceFailureAtomically(t *testing.T) {
	view := json.RawMessage(`{"metadata":{"nested":true}}`)
	filters := snapshotLocalContractMembers(t, `{"metadata":{"nested":1}}`)
	calls := 0
	matched, err := matchLocal(context.Background(), view, filters, func() error {
		calls++
		if calls == 1 {
			for i := range view {
				view[i] = '!'
			}
			filters[0].Key = "changed"
			filters[0].Value[0] = '!'
		}
		return nil
	})
	if err != nil || !matched || calls == 0 {
		t.Fatal("guard mutated snapshotted inputs", matched, err, calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause, source := errors.New("local filter parent cause"), errors.New("selected source changed")
	calls = 0
	matched, err = matchLocal(ctx, json.RawMessage(`{"metadata":{"nested":true}}`), snapshotLocalContractMembers(t, `{"metadata":{"nested":1}}`), func() error {
		calls++
		if calls > 1 {
			cancel(cause)
			return source
		}
		return nil
	})
	if matched || !errors.Is(err, source) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal("guard/context causes or atomic false result lost", matched, err)
	}
	calls = 0
	matched, err = matchLocal(nil, json.RawMessage(`{}`), nil, func() error { calls++; return nil })
	if matched || calls != 0 || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(matched, err, calls)
	}
	for _, invalid := range []json.RawMessage{json.RawMessage(`[]`), json.RawMessage(`{} []`), json.RawMessage([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		matched, err = matchLocal(context.Background(), invalid, nil, nil)
		if matched || err == nil {
			t.Fatal(string(invalid), matched, err)
		}
	}
}
