package request_test

import (
	"errors"
	"net/url"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type filters struct {
	Name    string   `q:"name"`
	Enabled *bool    `q:"enabled"`
	Limit   int      `q:"limit"`
	Fields  []string `q:"fields"`
}

func TestTypedQueryConversionPreservesPointersAndRejectsInvalidFilters(t *testing.T) {
	options, err := request.QueryOptions[*filters](url.Values{"name": {"worker"}, "enabled": {"false"}, "limit": {"5"}, "fields": {"id", "name"}})
	if err != nil || options.Name != "worker" || options.Enabled == nil || *options.Enabled || options.Limit != 5 || len(options.Fields) != 2 {
		t.Fatalf("options=%+v err=%v", options, err)
	}
	if _, err := request.QueryOptions[filters](url.Values{"unknown": {"value"}}); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := request.QueryOptions[filters](url.Values{"enabled": {"perhaps"}}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := request.QueryOptions[filters](url.Values{"limit": {"99999999999999999999999999999999"}}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
