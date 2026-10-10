package resource

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

func TestIgnoreMissingDropsOnlyPlain404(t *testing.T) {
	notFound := gophercloud.ErrUnexpectedResponseCode{Actual: http.StatusNotFound, Expected: []int{204}}
	conflict := gophercloud.ErrUnexpectedResponseCode{Actual: http.StatusConflict, Expected: []int{204}}
	wrapped := &OperationError{Operation: "Delete", Resource: "traits", Cause: notFound}
	for name, tc := range map[string]struct {
		err  error
		keep bool
	}{
		"nil":               {nil, false},
		"native 404":        {notFound, false},
		"wrapped 404":       {wrapped, false},
		"409":               {conflict, true},
		"canceled with 404": {errors.Join(context.Canceled, notFound), true},
		"two joined 404s":   {errors.Join(notFound, notFound), true},
		"invalid option":    {ErrInvalidOption, true},
		"plain error":       {errors.New("boom"), true},
	} {
		got := IgnoreMissing(tc.err)
		// ErrUnexpectedResponseCode holds slices and is not comparable.
		if tc.keep && (got == nil || got.Error() != tc.err.Error()) || !tc.keep && got != nil {
			t.Fatal(name, got)
		}
	}
}
