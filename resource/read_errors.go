package resource

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
)

// terminalReadError distinguishes a native HTTP failure from an error in the
// transport or accepted-response pipeline, even when it contains an HTTP code.
// Such errors cannot establish absence or permit an identity list fallback.
func terminalReadError(err error) bool {
	if joinedResponseFailure(err) {
		return true
	}
	var terminal interface{ TerminalSDKFailure() bool }
	if errors.As(err, &terminal) && terminal.TerminalSDKFailure() {
		return true
	}
	var accepted *ResponseError
	var transport *url.Error
	var syntax *json.SyntaxError
	var typed *json.UnmarshalTypeError
	var target *json.InvalidUnmarshalError
	return errors.As(err, &accepted) || errors.As(err, &transport) || errors.As(err, &syntax) || errors.As(err, &typed) || errors.As(err, &target) ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, ErrInvalidOption) || errors.Is(err, ErrUnsupported)
}

// A joined deletion failure carries more than missing-resource evidence.
// Preserve callback errors even when they use no SDK sentinel or known type.
func terminalDeleteError(err error) bool {
	return terminalReadError(err)
}

func joinedResponseFailure(err error) bool {
	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) {
		return false
	}
	causes := joined.Unwrap()
	if len(causes) == 1 {
		return joinedResponseFailure(causes[0])
	}
	return len(causes) > 1
}
