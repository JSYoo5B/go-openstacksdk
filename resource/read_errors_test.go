package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
)

type terminalReadItem struct{ ID, Name, Status string }

func readErrorCollection(cause error, calls *int, lists *int) *Collection[terminalReadItem] {
	get := func(context.Context, string) (*terminalReadItem, error) { *calls++; return nil, cause }
	return NewCollection(Adapter[terminalReadItem]{
		Kind: "read-items", IdentityFind: true,
		Get:              get,
		GetIdentityQuery: func(ctx context.Context, id string, _ url.Values) (*terminalReadItem, error) { return get(ctx, id) },
		Iterate: func(context.Context, url.Values) iter.Seq2[*terminalReadItem, error] {
			return func(yield func(*terminalReadItem, error) bool) {
				*lists++
				yield(&terminalReadItem{ID: "found", Name: "input"}, nil)
			}
		},
		ID: func(v *terminalReadItem) string { return v.ID }, Name: func(v *terminalReadItem) string { return v.Name }, Status: func(v *terminalReadItem) string { return v.Status },
	})
}

func nativeRead404() gophercloud.ErrUnexpectedResponseCode {
	return gophercloud.ErrUnexpectedResponseCode{URL: "http://original.example/items/input", Method: "GET", Expected: []int{200}, Actual: 404, Body: []byte(`{"error":"actual missing"}`), ResponseHeader: http.Header{"X-Request-Id": {"native-proof"}}}
}

func getReadError(t *testing.T, c *Collection[terminalReadItem], query bool) error {
	t.Helper()
	var value *terminalReadItem
	var err error
	if query {
		value, err = c.getIdentity(context.Background(), "input", url.Values{"domain_id": {"owned"}})
	} else {
		value, err = c.Get(context.Background(), "input")
	}
	if value != nil || err == nil {
		t.Fatalf("query=%v value=%+v err=%v", query, value, err)
	}
	return err
}

func assertReadOperation(t *testing.T, err error, wantCause error) {
	t.Helper()
	var operation *OperationError
	if !errors.As(err, &operation) || operation.Operation != "get" || operation.Resource != "read-items" || !reflect.DeepEqual(operation.Cause, wantCause) {
		t.Fatalf("operation=%+v error=%v wantCause=%v", operation, err, wantCause)
	}
}

func TestReadErrorsKeepOrdinaryNativeMissingAndExactOriginalEvidence(t *testing.T) {
	actual := nativeRead404()
	for _, tc := range []struct {
		name  string
		cause error
	}{
		{"native", actual}, {"fmt-wrapper", fmt.Errorf("service: %w", actual)},
		{"operation-wrapper", &OperationError{Operation: "nativeGet", Resource: "items", Cause: actual}},
		{"after-reauthentication", gophercloud.ErrErrorAfterReauthentication{ErrOriginal: actual}},
		{"logical-missing", &NotFoundError{Resource: "original", Reference: "input"}},
	} {
		for _, query := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/query=%v", tc.name, query), func(t *testing.T) {
				calls, lists := 0, 0
				c := readErrorCollection(tc.cause, &calls, &lists)
				err := getReadError(t, c, query)
				var operation *OperationError
				var missing *NotFoundError
				if !errors.Is(err, ErrNotFound) || !errors.As(err, &operation) || operation.Operation != "get" || !errors.As(err, &missing) || calls != 1 || lists != 0 {
					t.Fatalf("error=%v missing=%+v calls=%d lists=%d", err, missing, calls, lists)
				}
				if tc.name == "logical-missing" {
					assertReadOperation(t, err, tc.cause)
				} else {
					if missing.Resource != "read-items" || missing.Reference != "input" || !reflect.DeepEqual(missing.Cause, tc.cause) {
						t.Fatalf("missing=%+v original=%v", missing, tc.cause)
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != string(actual.Body) || native.ResponseHeader.Get("X-Request-Id") != "native-proof" || native.URL != actual.URL || native.Method != "GET" {
						t.Fatalf("native proof=%+v error=%v", native, err)
					}
				}
			})
		}
	}
}

func terminalReadCases() []struct {
	name  string
	cause error
} {
	native := nativeRead404()
	return []struct {
		name  string
		cause error
	}{
		{"url-transport", &url.Error{Op: "GET", URL: "http://original.example", Err: native}},
		{"accepted-response", &ResponseError{StatusCode: 200, Body: []byte(`{"accepted":true}`), Header: http.Header{"X-Accepted": {"proof"}}, Cause: native}},
		{"syntax", errors.Join(native, &json.SyntaxError{Offset: 3})},
		{"type", errors.Join(native, &json.UnmarshalTypeError{Value: "bool", Type: reflect.TypeFor[string]()})},
		{"invalid-unmarshal", errors.Join(native, &json.InvalidUnmarshalError{Type: reflect.TypeFor[string]()})},
		{"read-eof", errors.Join(native, io.EOF)},
		{"read-unexpected-eof", errors.Join(native, io.ErrUnexpectedEOF)},
		{"canceled", errors.Join(native, context.Canceled)},
		{"deadline", errors.Join(native, context.DeadlineExceeded)},
		{"invalid-option", errors.Join(native, ErrInvalidOption)},
		{"unsupported", errors.Join(native, ErrUnsupported)},
		{"callback", errors.Join(native, errors.New("untyped callback failed"))},
		{"nested callback", errors.Join(errors.Join(native, errors.New("nested callback failed")))},
	}
}

func TestReadErrorsNeverAddMissingToTerminalDirectOrQueryCauses(t *testing.T) {
	for _, tc := range terminalReadCases() {
		for _, query := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/query=%v", tc.name, query), func(t *testing.T) {
				calls, lists := 0, 0
				c := readErrorCollection(tc.cause, &calls, &lists)
				err := getReadError(t, c, query)
				if errors.Is(err, ErrNotFound) || calls != 1 || lists != 0 || !terminalReadError(err) {
					t.Fatalf("new false missing error=%v calls=%d lists=%d", err, calls, lists)
				}
				assertReadOperation(t, err, tc.cause)
				// No normalization or reconstruction of the original error object.
				operation := err.(*OperationError)
				if operation.Cause != tc.cause {
					t.Fatal("terminal cause pointer changed")
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"error":"actual missing"}` || native.ResponseHeader.Get("X-Request-Id") != "native-proof" {
					t.Fatalf("native cause lost: %v", err)
				}
				if tc.name == "accepted-response" {
					var proof *ResponseError
					if !errors.As(err, &proof) || proof != tc.cause || proof.StatusCode != 200 || string(proof.Body) != `{"accepted":true}` || proof.Header.Get("X-Accepted") != "proof" {
						t.Fatalf("accepted evidence=%+v err=%v", proof, err)
					}
				}
			})
		}
	}
}

func TestReadErrorsDefendIgnoreMissingAndWaitAgainstAdapterPrewrappedTerminalCauses(t *testing.T) {
	for _, tc := range terminalReadCases() {
		t.Run(tc.name, func(t *testing.T) {
			original := &NotFoundError{Resource: "adapter-owned", Reference: "input", Cause: tc.cause}
			calls, lists := 0, 0
			c := readErrorCollection(original, &calls, &lists)
			value, err := c.Find(context.Background(), ID("input"), WithIgnoreMissing())
			if value != nil || err == nil || !errors.Is(err, ErrNotFound) || calls != 1 || lists != 0 {
				t.Fatalf("find swallowed original error: value=%v err=%v calls=%d lists=%d", value, err, calls, lists)
			}
			assertReadOperation(t, err, original)
			if err.(*OperationError).Cause != original {
				t.Fatal("adapter missing was stripped or rebuilt")
			}
			calls, lists = 0, 0
			err = c.WaitDeleted(context.Background(), ID("input"), WithPollInterval(time.Millisecond))
			if err == nil || !errors.Is(err, ErrNotFound) || calls != 1 || lists != 0 {
				t.Fatalf("wait falsely succeeded: err=%v calls=%d lists=%d", err, calls, lists)
			}
			var wait *OperationError
			if !errors.As(err, &wait) || wait.Operation != "wait deleted" || wait.Resource != "read-items" {
				t.Fatalf("wait operation=%+v", wait)
			}
			get := wait.Cause.(*OperationError)
			if get.Operation != "get" || get.Cause != original {
				t.Fatalf("original chain changed: %+v", get)
			}
			for _, query := range []bool{false, true} {
				for _, policy := range []FindFallbackPolicy{FindFallbackCompatible, FindFallbackNotFoundOnly, FindFallbackNever} {
					calls, lists = 0, 0
					options := []IdentityFindOption{WithIdentityFindIgnoreMissing(true), WithIdentityFindFallback(policy)}
					if query {
						options = append(options, WithIdentityFindQuery("domain_id", "owned"))
					}
					value, err = c.FindIdentity(context.Background(), "input", options...)
					if value != nil || err == nil || calls != 1 || lists != 0 || !errors.Is(err, ErrNotFound) {
						t.Fatalf("query=%v policy=%v value=%v err=%v calls=%d lists=%d", query, policy, value, err, calls, lists)
					}
					var identity *OperationError
					if !errors.As(err, &identity) || identity.Operation != "find_identity" {
						t.Fatalf("identity error=%v", err)
					}
					if identity.Cause.(*OperationError).Cause != original {
						t.Fatal("identity replaced original terminal cause")
					}
				}
			}
		})
	}
}

func TestReadErrorsRetainOrdinaryMissingConsumersAndSuccessfulGetContextPolicy(t *testing.T) {
	for _, cause := range []error{nativeRead404(), fmt.Errorf("native wrapper: %w", nativeRead404()), &NotFoundError{Resource: "logical", Reference: "input"}} {
		calls, lists := 0, 0
		c := readErrorCollection(cause, &calls, &lists)
		if value, err := c.Find(context.Background(), ID("input"), WithIgnoreMissing()); value != nil || err != nil || calls != 1 {
			t.Fatalf("ordinary find=%v err=%v calls=%d", value, err, calls)
		}
		calls = 0
		if err := c.WaitDeleted(context.Background(), ID("input")); err != nil || calls != 1 {
			t.Fatalf("ordinary wait error=%v calls=%d", err, calls)
		}
	}
	for _, query := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		value := &terminalReadItem{ID: "input", Name: "input", Status: "ready"}
		get := func(context.Context, string) (*terminalReadItem, error) { cancel(); return value, nil }
		c := NewCollection(Adapter[terminalReadItem]{Kind: "read-items", Get: get, GetIdentityQuery: func(ctx context.Context, id string, _ url.Values) (*terminalReadItem, error) { return get(ctx, id) }})
		var actual *terminalReadItem
		var err error
		if query {
			actual, err = c.getIdentity(ctx, "input", url.Values{"domain_id": {"owned"}})
		} else {
			actual, err = c.Get(ctx, "input")
		}
		cancel()
		if err != nil || actual != value {
			t.Fatalf("query=%v successful Get policy changed: value=%v err=%v", query, actual, err)
		}
	}
	// Other native HTTP status failures are unchanged and cannot become missing.
	for _, code := range []int{400, 403, 409, 500} {
		cause := gophercloud.ErrUnexpectedResponseCode{Actual: code, Body: []byte("native")}
		calls, lists := 0, 0
		err := getReadError(t, readErrorCollection(cause, &calls, &lists), false)
		assertReadOperation(t, err, cause)
		if errors.Is(err, ErrNotFound) || terminalReadError(err) {
			t.Fatalf("ordinary native status classification changed: %v", err)
		}
	}
}
