package rest_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestListAcceptedNoContentTerminatesWithoutDecodingOrContinuation(t *testing.T) {
	for _, mode := range []string{"empty", "misleading link", "close error", "read error", "cancel", "custom policy", "unaccepted"} {
		t.Run(mode, func(t *testing.T) {
			requests := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cause := errors.New("accepted body failed")
			body := &limitsGuardedBody{Reader: strings.NewReader("")}
			if mode == "close error" {
				body.closeErr = cause
			}
			if mode == "read error" {
				body.Reader = &limitsGuardedReader{cause: cause}
			}
			if mode == "cancel" {
				body.onClose = cancel
			}
			client := limitsGuardedClient(func(*http.Request) (*http.Response, error) {
				requests++
				response := limitsGuardedWire(204, body)
				if mode == "misleading link" {
					response.Header.Set("Link", "<https://foreign.invalid/items>; rel=\"next\"")
				}
				return response, nil
			})
			spec := sourceGuardSpec(client, nil)
			if mode == "custom policy" {
				spec.ValidateResponse = func(*rest.Response) error { return resource.ErrUnsupported }
			}
			if mode != "unaccepted" {
				spec.ListCodes = []int{200, 204}
			}
			rows := 0
			var failure error
			for row, err := range rest.List(ctx, spec, nil) {
				if row != nil {
					rows++
				}
				if err != nil {
					failure = err
				}
			}
			if requests != 1 || rows != 0 || body.closes.Load() != 1 {
				t.Fatal(requests, rows, body.closes.Load())
			}
			switch mode {
			case "empty", "misleading link":
				if failure != nil {
					t.Fatal(failure)
				}
			case "custom policy":
				if !errors.Is(failure, resource.ErrUnsupported) {
					t.Fatal(failure)
				}
			case "unaccepted":
				if failure == nil || errors.Is(failure, io.EOF) {
					t.Fatal(failure)
				}
			default:
				var proof *resource.ResponseError
				if !errors.As(failure, &proof) || proof.StatusCode != 204 {
					t.Fatal(failure, proof)
				}
				if mode == "cancel" {
					if !errors.Is(failure, context.Canceled) {
						t.Fatal(failure)
					}
				} else if !errors.Is(failure, cause) {
					t.Fatal(failure)
				}
			}
		})
	}
}
