package resource

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"reflect"
	"testing"
)

type terminalDeletePolicyError struct{ error }

func (e terminalDeletePolicyError) Unwrap() error          { return e.error }
func (terminalDeletePolicyError) TerminalSDKFailure() bool { return true }

func deleteErrorCollection(cause error, calls *int) *Collection[terminalReadItem] {
	return NewCollection(Adapter[terminalReadItem]{
		Kind: "delete-items", ID: func(row *terminalReadItem) string { return row.ID },
		Name: func(row *terminalReadItem) string { return row.Name },
		Iterate: func(context.Context, url.Values) iter.Seq2[*terminalReadItem, error] {
			return func(yield func(*terminalReadItem, error) bool) {
				yield(&terminalReadItem{ID: "input", Name: "input"}, nil)
			}
		},
		Delete: func(context.Context, string) error { *calls++; return cause },
	})
}

func TestDeleteOnlyIgnoresOrdinaryMissingResponses(t *testing.T) {
	native := nativeRead404()
	native.Method = "DELETE"
	for _, cause := range []error{native, fmt.Errorf("service: %w", native), &OperationError{Operation: "nativeDelete", Resource: "items", Cause: native}} {
		for _, ref := range []Ref{ID("input"), Name("input")} {
			for _, strict := range []bool{false, true} {
				t.Run(fmt.Sprintf("%T/name=%t/strict=%t", cause, ref.IsName(), strict), func(t *testing.T) {
					calls := 0
					c := deleteErrorCollection(cause, &calls)
					var options []LookupOption
					if strict {
						options = []LookupOption{WithMissingError()}
					}
					err := c.Delete(context.Background(), ref, options...)
					if calls != 1 || strict && !errors.Is(err, ErrNotFound) || !strict && err != nil {
						t.Fatal(calls, err)
					}
				})
			}
		}
	}
}

func TestDeletePreservesTerminalAndJoinedFailuresUnderIgnoreMissing(t *testing.T) {
	cases := terminalReadCases()
	callback := errors.New("retry callback refused deletion")
	cases = append(cases, struct {
		name  string
		cause error
	}{"callback", errors.Join(nativeRead404(), callback)},
		struct {
			name  string
			cause error
		}{"nested callback", errors.Join(errors.Join(nativeRead404(), callback))},
		struct {
			name  string
			cause error
		}{"original status policy", terminalDeletePolicyError{nativeRead404()}})
	for _, tc := range cases {
		for _, ref := range []Ref{ID("input"), Name("input")} {
			for _, strict := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/name=%t/strict=%t", tc.name, ref.IsName(), strict), func(t *testing.T) {
					calls := 0
					c := deleteErrorCollection(tc.cause, &calls)
					var options []LookupOption
					if strict {
						options = []LookupOption{WithMissingError()}
					}
					err := c.Delete(context.Background(), ref, options...)
					var operation *OperationError
					if calls != 1 || err == nil || errors.Is(err, ErrNotFound) || !errors.As(err, &operation) || operation.Operation != "delete" || !reflect.DeepEqual(operation.Cause, tc.cause) {
						t.Fatalf("calls=%d operation=%+v error=%v cause=%v", calls, operation, err, tc.cause)
					}
				})
			}
		}
	}
}

func TestDeleteNamePreservesTerminalLookupInsteadOfIgnoringMissing(t *testing.T) {
	for _, terminal := range []error{context.Canceled, ErrInvalidOption, errors.Join(nativeRead404(), errors.New("lookup callback failed")), terminalDeletePolicyError{nativeRead404()}} {
		t.Run(fmt.Sprintf("%T", terminal), func(t *testing.T) {
			cause := &NotFoundError{Resource: "original", Reference: "input", Cause: terminal}
			deletes := 0
			c := deleteErrorCollection(nil, &deletes)
			c.binding.Iterate = func(context.Context, url.Values) iter.Seq2[*terminalReadItem, error] {
				return func(yield func(*terminalReadItem, error) bool) { yield(nil, cause) }
			}
			err := c.Delete(context.Background(), Name("input"))
			if err == nil || deletes != 0 || !errors.Is(err, cause) {
				t.Fatal(err, deletes)
			}
		})
	}
}
