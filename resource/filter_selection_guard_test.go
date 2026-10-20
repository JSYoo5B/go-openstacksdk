package resource

import (
	"context"
	"errors"
	"testing"
)

func TestFilterSelectionGuardStopsBeforeRestoringOption(t *testing.T) {
	changed := false
	restored := false
	cause := errors.New("source changed")
	check := func() error {
		if changed {
			return cause
		}
		return nil
	}
	options := []ListOption{
		func(config *listOptions) error { changed = true; return nil },
		func(config *listOptions) error { changed = false; restored = true; return nil },
	}
	selection, err := PrepareFilterSelectionGuarded(nil, check, options...)
	if selection != nil || !errors.Is(err, cause) || restored {
		t.Fatal(selection, err, restored)
	}
}

func TestFilterSelectionGuardPreservesCausesAndSkipsCanceledOptions(t *testing.T) {
	for _, early := range []bool{false, true} {
		t.Run(map[bool]string{true: "before callback", false: "after callback"}[early], func(t *testing.T) {
			callbackCause := errors.New("callback cause")
			contextCause := errors.New("operation cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			if early {
				cancel(contextCause)
			} else {
				defer cancel(nil)
			}
			called := false
			option := func(config *listOptions) error { called = true; cancel(contextCause); return callbackCause }
			check := func() error { return errors.Join(ctx.Err(), context.Cause(ctx)) }
			selected, err := PrepareFilterSelectionGuarded(nil, check, option)
			if selected != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, contextCause) || called == early {
				t.Fatal(selected, err, called)
			}
			if !early && !errors.Is(err, callbackCause) {
				t.Fatal("callback cause lost", err)
			}
		})
	}
}

func TestFilterSelectionGuardRetainsSelectionBoundaries(t *testing.T) {
	for _, test := range []struct {
		name    string
		guard   func() error
		options []ListOption
		want    error
	}{
		{"nil guard same values", nil, []ListOption{WithFilter("status", nil), WithFilter("vendor", make(chan int))}, nil},
		{"unknown bad value ignored", func() error { return nil }, []ListOption{WithFilter("status", "pending"), WithFilter("vendor", make(chan int))}, nil},
		{"nil option invalid", func() error { return nil }, []ListOption{nil}, ErrInvalidOption},
		{"nonsemantic option unsupported", func() error { return nil }, []ListOption{WithName("name")}, ErrUnsupported},
	} {
		t.Run(test.name, func(t *testing.T) {
			selected, err := PrepareFilterSelectionGuarded(&FilterDescriptor{Body: map[string]string{"status": "status"}}, test.guard, test.options...)
			if test.want != nil {
				if selected != nil || !errors.Is(err, test.want) {
					t.Fatal(selected, err)
				}
				return
			}
			if err != nil || selected == nil {
				t.Fatal(selected, err)
			}
			raw, present, err := selected.Attribute("status")
			if err != nil || !present || len(raw) == 0 {
				t.Fatal(string(raw), present, err)
			}
		})
	}
}
