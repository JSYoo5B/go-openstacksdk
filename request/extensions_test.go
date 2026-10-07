package request_test

import (
	"errors"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestHeadersProtectConcreteInputsAndRejectInvalidValues(t *testing.T) {
	type opts struct {
		ContentType string `h:"Content-Type"`
	}
	c, err := request.Apply(opts{}, request.WithHeader[opts]("x-vendor-setting", "false"))
	if err != nil {
		t.Fatal(err)
	}
	headers, err := request.MergeHeadersFor(map[string]string{"X-Object-Meta-Owner": "sdk"}, c.Headers, c.Options)
	if err != nil || headers["X-Vendor-Setting"] != "false" {
		t.Fatalf("headers=%v err=%v", headers, err)
	}
	for _, key := range []string{"content-type", "x-object-meta-owner"} {
		if _, err := request.MergeHeadersFor(map[string]string{"X-Object-Meta-Owner": "sdk"}, map[string]string{key: "overwrite"}, opts{}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := request.Apply(opts{}, request.WithHeader[opts]("X-Vendor", "value\r\nInjected: header")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := request.ValidateCapabilities(c, false, false, false); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestSecondaryArgumentsCannotPanicOrDisappear(t *testing.T) {
	c, err := request.Apply(input{}, request.WithArgument[input]("hints", 123))
	if err != nil {
		t.Fatal(err)
	}
	if _, exists, err := request.Argument[string](c, "hints"); !exists || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
	if err := request.ValidateCapabilities(c, false, false, false); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := request.ValidateCapabilities(c, false, false, false, "hints"); err != nil {
		t.Fatal(err)
	}
	if value, exists, err := request.Argument[int](c, "hints"); err != nil || !exists || value != 123 {
		t.Fatalf("value=%v exists=%v err=%v", value, exists, err)
	}
}
