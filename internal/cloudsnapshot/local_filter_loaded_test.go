package cloudsnapshot_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Expectations follow the pinned Resource.list _dict_filter over CPython3.14
// json.loads values; they do not use either Go equality implementation as oracle.
func loadedSnapshotFilters(t *testing.T, raw string) []cloudfilter.JSONMember {
	t.Helper()
	members, err := cloudfilter.ObjectMembers(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	return members
}
func assertLoadedSnapshot(t *testing.T, view, filters string, want bool) {
	t.Helper()
	got, err := cloudsnapshot.MatchLocalLoaded(context.Background(), json.RawMessage(view), loadedSnapshotFilters(t, filters), nil)
	if err != nil || got != want {
		t.Fatal(view, filters, got, err, want)
	}
}

func TestSnapshotLoadedLocalRoundedFloatAndBigIntegerEquality(t *testing.T) {
	for _, tc := range []struct {
		actual, expected string
		want             bool
	}{{"9007199254740993.0", "9007199254740993", false}, {"9007199254740993.0", "9007199254740992", true}, {"9007199254740993", "9007199254740992", false}, {"9007199254740993", "9007199254740993", true}, {"0.1", "0.10000000000000001", true}, {"9007199254740992", "9007199254740993.0", true}} {
		t.Run(tc.actual+tc.expected, func(t *testing.T) {
			assertLoadedSnapshot(t, `{"value":`+tc.actual+`}`, `{"value":`+tc.expected+`}`, tc.want)
		})
	}
}
func TestSnapshotLoadedLocalUnderflowEqualsZeroAndBooleanWithoutChangingExactDefault(t *testing.T) {
	for _, tc := range []struct{ actual, expected string }{{"1e-400", "0"}, {"-1e-400", "false"}, {"0", "1e-400"}, {"false", "-1e-400"}} {
		view, filter := `{"value":`+tc.actual+`}`, `{"value":`+tc.expected+`}`
		assertLoadedSnapshot(t, view, filter, true)
		matched, err := cloudsnapshot.MatchLocal(context.Background(), json.RawMessage(view), loadedSnapshotFilters(t, filter), nil)
		if err != nil || matched {
			t.Fatal("exact default changed", matched, err)
		}
	}
}
func TestSnapshotLoadedLocalOverflowInfinityAndFiniteIntegerRemainDistinct(t *testing.T) {
	huge := strings.Repeat("9", 401)
	for _, tc := range []struct {
		actual, expected string
		want             bool
	}{{"1e400", "10e399", true}, {"1e400", "-1e400", false}, {"1e400", huge, false}, {huge, "1e400", false}, {"-1e400", "-10e399", true}} {
		assertLoadedSnapshot(t, `{"value":`+tc.actual+`}`, `{"value":`+tc.expected+`}`, tc.want)
	}
}
func TestSnapshotLoadedLocalBooleanNumbersAndNullableLeaves(t *testing.T) {
	for _, tc := range []struct {
		actual, expected string
		want             bool
	}{{"true", "1.0", true}, {"false", "-0.0", true}, {"false", "null", false}, {"null", "null", true}, {`"1"`, "1", false}, {"1.0", "true", true}, {"2", "true", false}} {
		assertLoadedSnapshot(t, `{"value":`+tc.actual+`}`, `{"value":`+tc.expected+`}`, tc.want)
	}
	assertLoadedSnapshot(t, `{}`, `{"missing":null}`, true)
}
func TestSnapshotLoadedLocalEmptyDictionaryUsesLoadedScalarTruthiness(t *testing.T) {
	for _, tc := range []struct {
		actual string
		want   bool
	}{{"1e-400", false}, {"-1e-400", false}, {"0", false}, {"false", false}, {"null", false}, {`""`, false}, {`[]`, false}, {`{}`, false}, {"1e400", true}, {"true", true}, {`"truthy"`, true}, {`[0]`, true}, {`{"extra":null}`, true}} {
		assertLoadedSnapshot(t, `{"value":`+tc.actual+`}`, `{"value":{}}`, tc.want)
	}
	matched, err := cloudsnapshot.MatchLocal(context.Background(), json.RawMessage(`{"value":1e-400}`), loadedSnapshotFilters(t, `{"value":{}}`), nil)
	if err != nil || !matched {
		t.Fatal("default truthiness changed", matched, err)
	}
}
func TestSnapshotLoadedLocalTruthyScalarNonemptyDictionaryIsReachedShapeFailure(t *testing.T) {
	for _, actual := range []string{"true", "1e400", `"x"`, `[0]`} {
		got, err := cloudsnapshot.MatchLocalLoaded(context.Background(), json.RawMessage(`{"metadata":`+actual+`}`), loadedSnapshotFilters(t, `{"metadata":{"missing":null}}`), nil)
		var proof *resource.ResponseError
		if got || err == nil || !strings.Contains(err.Error(), "metadata") || errors.As(err, &proof) {
			t.Fatal(got, err)
		}
	}
	// Underflow is falsey after loading; it never attempts mapping.get.
	assertLoadedSnapshot(t, `{"metadata":1e-400}`, `{"metadata":{"missing":null}}`, false)
}
func TestSnapshotLoadedLocalNestedSubsetPreservesMissingNoneAndFloatPolicy(t *testing.T) {
	assertLoadedSnapshot(t, `{"metadata":{"nested":{"rounded":9007199254740993.0,"tiny":1e-400,"bool":true,"extra":9},"unused":1}}`, `{"metadata":{"nested":{"rounded":9007199254740992,"tiny":false,"bool":1,"missing":null}}}`, true)
	assertLoadedSnapshot(t, `{"metadata":{}}`, `{"metadata":{"missing":null}}`, false)
	assertLoadedSnapshot(t, `{"metadata":{"extra":1}}`, `{"metadata":{"missing":null}}`, true)
}
func TestSnapshotLoadedLocalArrayLeavesKeepExactDictionaryWidthAndOrder(t *testing.T) {
	assertLoadedSnapshot(t, `{"metadata":{"rows":[true,{"x":1e-400,"large":9007199254740993.0}]}}`, `{"metadata":{"rows":[1,{"large":9007199254740992,"x":false}]}}`, true)
	assertLoadedSnapshot(t, `{"metadata":{"rows":[{"x":1e-400,"extra":1}]}}`, `{"metadata":{"rows":[{"x":false}]}}`, false)
	assertLoadedSnapshot(t, `{"metadata":{"rows":[1,2]}}`, `{"metadata":{"rows":[2,1]}}`, false)
}
func TestSnapshotLoadedLocalOrderedShortCircuitKeepsUnreachedFailuresUntouched(t *testing.T) {
	view := json.RawMessage(`{"state":"keep","metadata":true}`)
	for _, tc := range []struct {
		filter string
		fail   bool
	}{{`{"state":"drop","metadata":{"x":1}}`, false}, {`{"metadata":{"x":1},"state":"drop"}`, true}, {`{"state":"keep","metadata":{"x":1},"state":"drop"}`, false}} {
		matched, err := cloudsnapshot.MatchLocalLoaded(context.Background(), view, loadedSnapshotFilters(t, tc.filter), nil)
		if matched || (err != nil) != tc.fail {
			t.Fatal(tc, matched, err)
		}
	}
	filters := []cloudfilter.JSONMember{{Key: "state", Value: json.RawMessage(`"drop"`)}, {Key: "metadata", Value: json.RawMessage(`{broken`)}}
	if matched, err := cloudsnapshot.MatchLocalLoaded(context.Background(), view, filters, nil); matched || err != nil {
		t.Fatal(matched, err)
	}
	filters[0].Value = json.RawMessage(`"keep"`)
	if matched, err := cloudsnapshot.MatchLocalLoaded(context.Background(), view, filters, nil); matched || err == nil {
		t.Fatal(matched, err)
	}
}
func TestSnapshotLoadedLocalInvalidUTF8AndJSONDoNotReturnPartialSuccess(t *testing.T) {
	for _, view := range []json.RawMessage{nil, json.RawMessage(`[]`), json.RawMessage(`{} []`), json.RawMessage(`{"value":"` + string([]byte{0xff}) + `"}`)} {
		matched, err := cloudsnapshot.MatchLocalLoaded(context.Background(), view, nil, nil)
		if matched || err == nil {
			t.Fatal(string(view), matched, err)
		}
	}
	for _, filter := range []json.RawMessage{nil, json.RawMessage(``), json.RawMessage(`false true`), json.RawMessage(`"` + string([]byte{0xff}) + `"`), json.RawMessage(`NaN`)} {
		matched, err := cloudsnapshot.MatchLocalLoaded(context.Background(), json.RawMessage(`{"value":1}`), []cloudfilter.JSONMember{{Key: "value", Value: filter}}, nil)
		if matched || err == nil {
			t.Fatal(string(filter), matched, err)
		}
	}
}
func TestSnapshotLoadedLocalSnapshotsBeforeGuardAndKeepsCallerCancellation(t *testing.T) {
	view := json.RawMessage(`{"metadata":{"tiny":1e-400}}`)
	filters := loadedSnapshotFilters(t, `{"metadata":{"tiny":false}}`)
	calls := 0
	matched, err := cloudsnapshot.MatchLocalLoaded(context.Background(), view, filters, func() error {
		calls++
		if calls == 1 {
			view[0] = '!'
			filters[0].Key = "late"
			filters[0].Value[0] = '!'
		}
		return nil
	})
	if err != nil || !matched || calls == 0 {
		t.Fatal(matched, err, calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("loaded filter canceled")
	cancel(cause)
	calls = 0
	matched, err = cloudsnapshot.MatchLocalLoaded(ctx, json.RawMessage(`{}`), nil, func() error { calls++; return nil })
	if matched || calls != 0 || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(matched, err, calls)
	}
}
func TestSnapshotLoadedLocalEightResourceRowsRetainOneShotGuardFailures(t *testing.T) {
	// Eight normalized row shapes exercise the shared guard layer independently
	// of a leaf's schema, pagination or HTTP API support classification.
	for _, kind := range []string{"volume", "snapshot", "backup", "attachment", "volume-type", "group", "group-snapshot", "consistency-group-snapshot"} {
		t.Run(kind, func(t *testing.T) {
			view := json.RawMessage(fmt.Sprintf(`{"id":%q,"metadata":{"rounded":9007199254740993.0,"bool":true}}`, kind))
			filters := loadedSnapshotFilters(t, `{"metadata":{"rounded":9007199254740992,"bool":1}}`)
			if matched, err := cloudsnapshot.MatchLocalLoaded(context.Background(), view, filters, nil); err != nil || !matched {
				t.Fatal(matched, err)
			}
			calls := 0
			cause := errors.New("selected resource changed once")
			matched, err := cloudsnapshot.MatchLocalLoaded(context.Background(), view, filters, func() error {
				calls++
				if calls == 3 {
					return cause
				}
				return nil
			})
			var proof *resource.ResponseError
			if matched || !errors.Is(err, cause) || calls < 4 || errors.As(err, &proof) {
				t.Fatal(matched, err, calls)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			parent := errors.New("parent canceled")
			calls = 0
			matched, err = cloudsnapshot.MatchLocalLoaded(ctx, view, filters, func() error {
				calls++
				if calls == 3 {
					cancel(parent)
					return cause
				}
				return nil
			})
			if matched || !errors.Is(err, cause) || !errors.Is(err, parent) || !errors.Is(err, context.Canceled) {
				t.Fatal(matched, err, calls)
			}
		})
	}
}
