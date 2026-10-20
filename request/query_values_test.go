package request_test

import (
	"errors"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestQueryValuesRepeatKeysAndExtendTypedQuery(t *testing.T) {
	values := []string{"a", "b"}
	option := request.WithQueryValues[struct{}]("member_of1", values...)
	values[0] = "changed"
	c, err := request.Apply(struct{}{}, request.WithQuery[struct{}]("member_of1", "old"), option,
		request.WithQueryValues[struct{}]("gone", "x"), request.WithQueryValues[struct{}]("gone"))
	if err != nil || !reflect.DeepEqual(c.Query, url.Values{"member_of1": {"a", "b"}}) {
		t.Fatal(c.Query, err)
	}
	query, err := request.ExtendQuery("?member_of1=typed&limit=1", c.Query)
	if err != nil || query != "?limit=1&member_of1=a&member_of1=b" {
		t.Fatal(query, err)
	}
	if _, err := request.Apply(struct{}{}, request.WithQueryValues[struct{}](" ", "a")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
