package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestImageLocationsOptionsSnapshotsAndReplacement(t *testing.T) {
	pair := &ImageLocationValidation{OSHashAlgo: "before", OSHashValue: "owned"}
	headers := map[string]string{"X-Original": "before"}
	helper := WithAddImageLocationOpts(AddImageLocationOpts{ValidationData: pair, Headers: headers})
	pair.OSHashAlgo = "after"
	headers["X-Original"] = "after"
	policy, err := parseAddImageLocationOptions([]AddImageLocationOption{WithAddImageLocationHeader("X-Discard", "before"), WithImageLocationValidation("discard", "discard"), helper, WithAddImageLocationHeader("x-original", "last"), WithAddImageLocationHeaders(map[string]string{"X-Extra": "owned"})})
	if err != nil || policy.ValidationData.OSHashAlgo != "before" || policy.Headers["X-Original"] != "last" || policy.Headers["X-Extra"] != "owned" || policy.Headers["X-Discard"] != "" {
		t.Fatal("replacement/snapshot failed", policy, err)
	}
	policy.ValidationData.OSHashValue = "late mutation"
	policy.Headers["X-Original"] = "late mutation"
	policy, err = parseAddImageLocationOptions([]AddImageLocationOption{helper})
	if err != nil || policy.ValidationData.OSHashValue != "owned" || policy.Headers["X-Original"] != "before" {
		t.Fatal("reusable helper aliases application", policy, err)
	}
	policy, err = parseAddImageLocationOptions([]AddImageLocationOption{helper, WithAddImageLocationOpts(AddImageLocationOpts{})})
	if err != nil || policy.ValidationData != nil || len(policy.Headers) != 0 || policy.Headers == nil {
		t.Fatal("complete replacement retained fields", policy, err)
	}
	getHeaders := map[string]string{"X-Before": "owned"}
	getHelper := WithGetImageLocationsOpts(GetImageLocationsOpts{Headers: getHeaders})
	getHeaders["X-Before"] = "late"
	get, err := parseGetImageLocationsOptions([]GetImageLocationsOption{WithGetImageLocationsHeader("X-Discard", "before"), getHelper, WithGetImageLocationsHeader("x-before", "last")})
	if err != nil || get.Headers["X-Before"] != "last" || get.Headers["X-Discard"] != "" {
		t.Fatal("Get replacement failed", get, err)
	}
}

func TestImageLocationsOptionsOwnCallbackHandlesAndSlice(t *testing.T) {
	var retained *AddImageLocationOpts
	var applies, calls int
	options := make([]AddImageLocationOption, 2)
	options[0] = func(config *AddImageLocationOpts) error {
		applies++
		if config.Headers == nil {
			t.Fatal("default Headers not initialized")
		}
		config.Headers["X-Extra"] = "owned"
		config.ValidationData = &ImageLocationValidation{OSHashAlgo: "literal", OSHashValue: "owned"}
		retained = config
		options[1] = func(*AddImageLocationOpts) error { t.Fatal("option slice was not captured"); return nil }
		return nil
	}
	options[1] = func(config *AddImageLocationOpts) error {
		applies++
		retained.Headers["X-Extra"] = "retained mutation"
		retained.ValidationData.OSHashValue = "retained mutation"
		if config.Headers["X-Extra"] != "owned" || config.ValidationData.OSHashValue != "owned" {
			t.Fatal("retained config aliases next callback", config)
		}
		return nil
	}
	client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		var body struct {
			Validation map[string]string `json:"validation_data"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Validation["os_hash_value"] != "owned" || req.Header.Get("X-Extra") != "owned" {
			t.Fatal("prepared callback snapshot changed", body, err, req.Header)
		}
		return deleteCoreHTTP(202, http.NoBody, nil), nil
	})
	ack, err := New(client).AddImageLocation(context.Background(), resource.ID("fixed"), "literal", options...)
	if err != nil || ack == nil || applies != 2 || calls != 1 {
		t.Fatalf("ack=%v err=%v applies=%d calls=%d", ack, err, applies, calls)
	}
	cause := errors.New("option failure")
	for _, add := range []bool{false, true} {
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("option failure sent HTTP"); return nil, nil })
		var err error
		if add {
			_, err = New(client).AddImageLocation(context.Background(), resource.Name("exact"), "literal", func(*AddImageLocationOpts) error { return cause })
		} else {
			_, err = New(client).GetImageLocations(context.Background(), resource.Name("exact"), func(config *GetImageLocationsOpts) error {
				if config.Headers == nil {
					t.Fatal("Get Headers nil")
				}
				return cause
			})
		}
		if !errors.Is(err, cause) {
			t.Fatal("option cause lost", err)
		}
	}
}

func TestImageLocationsOptionsRejectOwnedHeadersAndAliases(t *testing.T) {
	for _, header := range []map[string]string{{"X-Extra": "before", "x-extra": "after"}, {"X-Auth-Token": "override"}, {"Host": "foreign.test"}, {"Content-Type": "text/plain"}, {"Accept": "text/plain"}, {"OpenStack-API-Version": "image 2.17"}, {"X-OpenStack-Image-Size": "1"}, {"Content-Length": "1"}, {"X-Bad": "bad\nvalue"}, {"X-Bad": "bad\xff"}} {
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("invalid header sent HTTP"); return nil, nil })
		ack, addErr := New(client).AddImageLocation(context.Background(), resource.Name("exact"), "literal", WithAddImageLocationHeaders(header))
		result, getErr := New(client).GetImageLocations(context.Background(), resource.Name("exact"), WithGetImageLocationsHeaders(header))
		if ack != nil || result != nil || !errors.Is(addErr, resource.ErrInvalidOption) || !errors.Is(getErr, resource.ErrInvalidOption) {
			t.Fatalf("headers=%v add=%v get=%v", header, addErr, getErr)
		}
	}
	for _, add := range []bool{false, true} {
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("nil callback sent HTTP"); return nil, nil })
		var err error
		if add {
			_, err = New(client).AddImageLocation(context.Background(), resource.ID("fixed"), "literal", nil)
		} else {
			_, err = New(client).GetImageLocations(context.Background(), resource.ID("fixed"), nil)
		}
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("nil callback accepted", err)
		}
	}
}

func TestImageLocationsOptionsParallelReusableHelpers(t *testing.T) {
	headers := map[string]string{"X-Shared": "owned"}
	addHeader := WithAddImageLocationHeaders(headers)
	getHeader := WithGetImageLocationsHeaders(headers)
	hash := WithImageLocationValidation("literal", "owned")
	headers["X-Shared"] = "after creation"
	for index := 0; index < 12; index++ {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			t.Parallel()
			add, err := parseAddImageLocationOptions([]AddImageLocationOption{addHeader, hash})
			if err != nil || add.Headers["X-Shared"] != "owned" || add.ValidationData.OSHashValue != "owned" {
				t.Fatal("shared Add helper changed", add, err)
			}
			add.Headers["X-Shared"] = "application mutation"
			add.ValidationData.OSHashValue = "application mutation"
			get, err := parseGetImageLocationsOptions([]GetImageLocationsOption{getHeader})
			if err != nil || get.Headers["X-Shared"] != "owned" {
				t.Fatal("shared Get helper changed", get, err)
			}
			get.Headers["X-Shared"] = "application mutation"
		})
	}
}
