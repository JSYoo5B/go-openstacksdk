package glancecalls

import (
	"errors"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type opts struct{ Headers map[string]string }

func TestHeaderSpecReplacesAliasesInCallerOrder(t *testing.T) {
	spec := HeaderSpec(func(o *opts) *map[string]string { return &o.Headers })
	var o opts
	for _, apply := range calls.Translate([]request.GetCall{request.Header("x-trace", "1"), request.Header("X-TRACE", "2"), request.Header("x-other", "3")}, spec) {
		if err := apply(&o); err != nil {
			t.Fatal(err)
		}
	}
	if want := map[string]string{"X-Trace": "2", "X-Other": "3"}; !reflect.DeepEqual(o.Headers, want) {
		t.Fatalf("headers = %v, want %v", o.Headers, want)
	}
}

func TestHeaderSpecRejectsOtherControls(t *testing.T) {
	spec := HeaderSpec(func(o *opts) *map[string]string { return &o.Headers })
	for _, call := range []request.GetCall{request.Headers(map[string]string{"A": "1"}), request.Microversion("2.1")} {
		var o opts
		if err := calls.Translate([]request.GetCall{call}, spec)[0](&o); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("call %v: err = %v", call, err)
		}
	}
}
