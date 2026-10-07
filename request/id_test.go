package request_test

import (
	"errors"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"testing"
)

type nativeID int64

func TestNumericIDRespectsNativeTypeAndRange(t *testing.T) {
	if id, err := request.NumericID[nativeID]("123"); err != nil || id != 123 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	if _, err := request.NumericID[int8]("128"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := request.NumericID[uint]("-1"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := request.NumericID[int]("not-numeric"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
