package rest

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"gophercloudsdk/resource"
)

func TestResponseValidationChecksWholeBodyBeforeGetAndListDecoders(t *testing.T) {
	for _, operation := range []string{"get", "list"} {
		t.Run(operation, func(t *testing.T) {
			body := "{\"items\":[],\"unknown\":\"\xff\"}"
			var calls, checks atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				writeList(w, body)
			})
			cause := errors.New("invalid whole response UTF-8")
			spec.ValidateResponse = func(response *Response) error {
				checks.Add(1)
				if utf8.Valid(response.Body) || string(response.Body) != body || response.StatusCode != 200 || response.Header.Get("X-Page") != "kept" {
					t.Fatalf("missing wire evidence: %+v", response)
				}
				return cause
			}
			var err error
			if operation == "get" {
				value, failure := Collection(spec).Get(context.Background(), "id")
				if value != nil {
					t.Fatal("invalid response produced an object")
				}
				err = failure
			} else {
				values, failure := collectList(context.Background(), spec, nil)
				if len(values) != 0 {
					t.Fatal("invalid response yielded rows")
				}
				err = failure
			}
			var evidence *resource.ResponseError
			if !errors.Is(err, cause) || !errors.As(err, &evidence) || string(evidence.Body) != body || evidence.Header.Get("X-Page") != "kept" || evidence.StatusCode != 200 || calls.Load() != 1 || checks.Load() != 1 {
				t.Fatalf("error=%v evidence=%+v calls/checks=%d/%d", err, evidence, calls.Load(), checks.Load())
			}
		})
	}
}

func TestResponseValidationRunsPerPageAndPreservesLazyBreak(t *testing.T) {
	for _, stopEarly := range []bool{false, true} {
		t.Run(map[bool]string{false: "second page failure", true: "consumer break"}[stopEarly], func(t *testing.T) {
			var calls, checks atomic.Int32
			const first = `{"items":[{"id":"first"}],"next":"?marker=second"}`
			const second = `{"items":[{"id":"second"}],"vendor":"invalid"}`
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("marker") == "" {
					writeList(w, first)
				} else {
					writeList(w, second)
				}
			})
			cause := errors.New("invalid second whole page")
			spec.ValidateResponse = func(response *Response) error {
				if checks.Add(1) == 2 {
					return cause
				}
				return nil
			}
			stream := List(context.Background(), spec, nil)
			if calls.Load() != 0 || checks.Load() != 0 {
				t.Fatal("stream validation was eager")
			}
			var rows int
			var failure error
			for value, err := range stream {
				if err != nil {
					failure = err
					break
				}
				rows++
				if value.ID != "first" {
					t.Fatal("invalid second page yielded an object")
				}
				if stopEarly {
					break
				}
			}
			if rows != 1 {
				t.Fatalf("rows=%d", rows)
			}
			if stopEarly {
				if failure != nil || calls.Load() != 1 || checks.Load() != 1 {
					t.Fatalf("failure=%v calls/checks=%d/%d", failure, calls.Load(), checks.Load())
				}
			} else {
				var evidence *resource.ResponseError
				if !errors.Is(failure, cause) || !errors.As(failure, &evidence) || string(evidence.Body) != second || calls.Load() != 2 || checks.Load() != 2 {
					t.Fatalf("failure=%v evidence=%+v calls/checks=%d/%d", failure, evidence, calls.Load(), checks.Load())
				}
			}
		})
	}
}
