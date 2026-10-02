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
	var accepted *ResponseError
	var transport *url.Error
	var syntax *json.SyntaxError
	var typed *json.UnmarshalTypeError
	var target *json.InvalidUnmarshalError
	return errors.As(err, &accepted) || errors.As(err, &transport) || errors.As(err, &syntax) || errors.As(err, &typed) || errors.As(err, &target) ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, ErrInvalidOption) || errors.Is(err, ErrUnsupported)
}
