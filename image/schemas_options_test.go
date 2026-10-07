package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestImageSchemasOptionsSnapshotReplacementAndLastWins(t *testing.T) {
	headers := map[string]string{"X-Source": "captured"}
	full := WithGetSchemaOpts(GetSchemaOpts{Headers: headers})
	merge := WithGetSchemaHeaders(headers)
	headers["X-Source"] = "mutated"
	policy, err := parseGetSchemaOptions([]GetSchemaOption{WithGetSchemaHeader("X-Discard", "before"), full, WithGetSchemaHeader("x-source", "last"), WithGetSchemaHeaders(map[string]string{"X-Extra": "owned"})})
	if err != nil || policy.Headers["X-Source"] != "last" || policy.Headers["X-Extra"] != "owned" || policy.Headers["X-Discard"] != "" {
		t.Fatal("replacement/last-wins failed", policy, err)
	}
	policy.Headers["X-Source"] = "application mutation"
	policy, err = parseGetSchemaOptions([]GetSchemaOption{full})
	if err != nil || policy.Headers["X-Source"] != "captured" {
		t.Fatal("full helper did not snapshot", policy, err)
	}
	policy, err = parseGetSchemaOptions([]GetSchemaOption{merge})
	if err != nil || policy.Headers["X-Source"] != "captured" {
		t.Fatal("map helper did not snapshot", policy, err)
	}
	policy, err = parseGetSchemaOptions([]GetSchemaOption{full, WithGetSchemaOpts(GetSchemaOpts{})})
	if err != nil || policy.Headers == nil || len(policy.Headers) != 0 {
		t.Fatal("empty full replacement retained headers", policy, err)
	}
}

func TestImageSchemasOptionsOwnRetainedCallbacksAndSlice(t *testing.T) {
	var retained *GetSchemaOpts
	var applies, calls int
	options := make([]GetSchemaOption, 2)
	options[0] = func(config *GetSchemaOpts) error {
		applies++
		if config.Headers == nil {
			t.Fatal("default map not initialized")
		}
		config.Headers["X-Extra"] = "owned"
		retained = config
		options[1] = func(*GetSchemaOpts) error { t.Fatal("option slice not captured"); return nil }
		return nil
	}
	options[1] = func(config *GetSchemaOpts) error {
		applies++
		retained.Headers["X-Extra"] = "late mutation"
		if config.Headers["X-Extra"] != "owned" {
			t.Fatal("callback config aliases earlier handle", config)
		}
		retained = config
		return nil
	}
	client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		retained.Headers["X-Extra"] = "transport-time mutation"
		if req.Header.Get("X-Extra") != "owned" {
			t.Fatal("prepared map aliases retained config", req.Header)
		}
		return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{}`)), nil), nil
	})
	value, err := New(client).GetImageSchema(context.Background(), options...)
	if err != nil || value == nil || applies != 2 || calls != 1 {
		t.Fatal(value, err, applies, calls)
	}
	cause := errors.New("custom option cause")
	client = deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("failed option emitted GET"); return nil, nil })
	value, err = New(client).GetImageSchema(context.Background(), func(*GetSchemaOpts) error { return cause })
	if value != nil || !errors.Is(err, cause) {
		t.Fatal("option cause lost", value, err)
	}
}

func TestImageSchemasOptionsHeaderAuthorityAndAliases(t *testing.T) {
	for _, headers := range []map[string]string{{"X-Alias": "one", "x-alias": "two"}, {"X-Auth-Token": "token"}, {"X-Service-Token": "token"}, {"Host": "foreign.test"}, {"Cookie": "private"}, {"Content-Length": "1"}, {"Transfer-Encoding": "chunked"}, {"Connection": "close"}, {"Accept": "text/plain"}, {"Content-Type": "text/plain"}, {"OpenStack-API-Version": "image 2.17"}, {"X-OpenStack-Image-Size": "10"}, {"X-Bad": "line\nfeed"}, {"X-Bad": "bad\xff"}, {"bad key": "value"}} {
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("invalid option emitted GET"); return nil, nil })
		value, err := New(client).GetImageSchema(context.Background(), WithGetSchemaHeaders(headers))
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("invalid header accepted", headers, value, err)
		}
	}
	policy, err := parseGetSchemaOptions([]GetSchemaOption{WithGetSchemaHeaders(map[string]string{"X-Alias": "same", "x-alias": "same"}), WithGetSchemaHeader("x-alias", "last")})
	if err != nil || len(policy.Headers) != 1 || policy.Headers["X-Alias"] != "last" {
		t.Fatal("safe alias merge failed", policy, err)
	}
	client := deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("nil option emitted GET"); return nil, nil })
	value, err := New(client).GetImageSchema(context.Background(), nil)
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("nil callback accepted", value, err)
	}
}

func TestImageSchemasOptionsParallelReusableHelpers(t *testing.T) {
	headers := map[string]string{"X-Shared": "owned"}
	full := WithGetSchemaOpts(GetSchemaOpts{Headers: headers})
	merge := WithGetSchemaHeaders(headers)
	one := WithGetSchemaHeader("X-Extra", "owned")
	headers["X-Shared"] = "late"
	for index := 0; index < 12; index++ {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			t.Parallel()
			policy, err := parseGetSchemaOptions([]GetSchemaOption{full, merge, one})
			if err != nil || policy.Headers["X-Shared"] != "owned" || policy.Headers["X-Extra"] != "owned" {
				t.Fatal("reusable helper changed", policy, err)
			}
			policy.Headers["X-Shared"] = "application mutation"
		})
	}
}
