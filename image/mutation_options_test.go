package image

import (
	"errors"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageMutationOptionsSnapshotReplacementAndLastWins(t *testing.T) {
	input := map[string]string{"X-Extra": "snapshot"}
	replace := WithImageMutationOpts(ImageMutationOpts{Headers: input})
	merge := WithImageMutationHeaders(input)
	input["X-Extra"] = "caller mutation"
	value, err := parseImageMutationOptions([]ImageMutationOption{request.Header("X-Old", "discarded"), replace, request.Header("x-extra", "last"), request.Header("X-Other", "retained")})
	if err != nil || value.Headers["X-Extra"] != "last" || value.Headers["X-Old"] != "" || value.Headers["X-Other"] != "retained" {
		t.Fatalf("value=%+v err=%v", value, err)
	}
	value, err = parseImageMutationOptions([]ImageMutationOption{request.Header("X-Old", "retained"), merge})
	if err != nil || value.Headers["X-Extra"] != "snapshot" || value.Headers["X-Old"] != "retained" {
		t.Fatalf("map snapshot/merge changed: value=%+v err=%v", value, err)
	}
	value, err = parseImageMutationOptions([]ImageMutationOption{replace, WithImageMutationOpts(ImageMutationOpts{})})
	if err != nil || value.Headers == nil || len(value.Headers) != 0 {
		t.Fatalf("empty replacement value=%+v err=%v", value, err)
	}
	var config ImageMutationOpts
	if err := applyCall(imageMutationCalls, request.Header("X-Direct", "initialized"), &config); err != nil || config.Headers["X-Direct"] != "initialized" {
		t.Fatal("helper did not initialize nil header map", config, err)
	}
}

func TestImageMutationOptionsCallbacksOwnHandlesAndCauses(t *testing.T) {
	var retained []*ImageMutationOpts
	var callbacks int
	value, err := parseImageMutationOptions([]ImageMutationOption{
		request.Custom(func(value *ImageMutationOpts) error {
			callbacks++
			if value.Headers == nil {
				t.Fatal("callback default Headers is nil")
			}
			value.Headers["X-Extra"] = "owned"
			retained = append(retained, value)
			return nil
		}),
		request.Custom(func(value *ImageMutationOpts) error {
			callbacks++
			retained[0].Headers["X-Extra"] = "earlier retained mutation"
			if value.Headers["X-Extra"] != "owned" {
				t.Fatal("later callback aliases retained handle", value)
			}
			retained = append(retained, value)
			return nil
		}),
	})
	if err != nil || callbacks != 2 || retained[0] == retained[1] {
		t.Fatal("callback count/identity changed", callbacks, retained, err)
	}
	for _, retained := range retained {
		retained.Headers["X-Extra"] = "after preparation"
	}
	if value.Headers["X-Extra"] != "owned" {
		t.Fatal("prepared options alias caller", value)
	}
	cause := errors.New("mutation option cause")
	_, err = parseImageMutationOptions([]ImageMutationOption{request.Custom(func(*ImageMutationOpts) error { return cause })})
	if err != cause {
		t.Fatal("single callback cause changed", err)
	}
	value, err = parseImageMutationOptions([]ImageMutationOption{request.Custom(func(value *ImageMutationOpts) error { value.Headers = nil; return nil }), request.Custom(func(value *ImageMutationOpts) error { value.Headers["X-Extra"] = "owned"; return nil })})
	if err != nil || value.Headers["X-Extra"] != "owned" {
		t.Fatal("nil callback map was not normalized", value, err)
	}
}

func TestImageMutationOptionsRejectHeaderOwnershipAndAliases(t *testing.T) {
	for _, option := range []ImageMutationOption{
		nil, request.Header("Authorization", "replacement"), request.Header("X-Auth-Token", "replacement"), request.Header("Host", "foreign.test"), request.Header("Cookie", "secret"),
		request.Header("Content-Length", "1"), request.Header("Connection", "close"), request.Header("Accept", "text/plain"), request.Header("Content-Type", "text/plain"), request.Header("OpenStack-API-Version", "image 2.8"),
		request.Header("X-OpenStack-Image-Size", "0"), request.Header("Bad Name", "value"), request.Header("X-Extra", "bad\x00value"), request.Header("X-Extra", "bad\r\nvalue"), request.Header("X-Extra", "bad\xffvalue"),
		WithImageMutationHeaders(map[string]string{"X-Extra": "one", "x-extra": "two"}), WithImageMutationOpts(ImageMutationOpts{Headers: map[string]string{"X-Extra": "one", "x-extra": "two"}}),
	} {
		if _, err := parseImageMutationOptions([]ImageMutationOption{option}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("header option accepted or invalid cause lost", err)
		}
	}
	value, err := parseImageMutationOptions([]ImageMutationOption{WithImageMutationHeaders(map[string]string{"X-Extra": "same", "x-extra": "same"})})
	if err != nil || len(value.Headers) != 1 || value.Headers["X-Extra"] != "same" {
		t.Fatal("identical aliases not canonicalized", value, err)
	}
}

func TestImageMutationOptionsParallelReusableHelpers(t *testing.T) {
	options := []ImageMutationOption{WithImageMutationOpts(ImageMutationOpts{Headers: map[string]string{"X-Base": "owned"}}), WithImageMutationHeaders(map[string]string{"X-Extra": "owned"}), request.Header("X-Last", "owned")}
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			for range 20 {
				value, err := parseImageMutationOptions(options)
				if err != nil || len(value.Headers) != 3 || value.Headers["X-Base"] != "owned" || value.Headers["X-Extra"] != "owned" || value.Headers["X-Last"] != "owned" {
					t.Errorf("reusable headers changed: value=%+v err=%v", value, err)
					return
				}
				value.Headers["X-Base"] = "caller mutation"
			}
		})
	}
	group.Wait()
}
