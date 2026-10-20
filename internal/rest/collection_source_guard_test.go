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
	"github.com/gophercloud/gophercloud/v2"
)

type sourceGuardItem struct {
	ID       string            `json:"id"`
	Metadata resource.Metadata `json:"-"`
}

func sourceGuardSpec(client *gophercloud.ServiceClient, guard func(context.Context) error) rest.CollectionSpec[sourceGuardItem] {
	return rest.CollectionSpec[sourceGuardItem]{Client: client, Path: "items", Kind: "items", SingleKey: "item", PluralKey: "items",
		ID: func(row *sourceGuardItem) string { return row.ID }, Metadata: func(row *sourceGuardItem) *resource.Metadata { return &row.Metadata },
		Get: true, Delete: true, SourceGuard: guard}
}

func sourceGuardOperation(spec rest.CollectionSpec[sourceGuardItem], operation string) error {
	ctx := context.Background()
	switch operation {
	case "get":
		_, err := rest.Collection(spec).Get(ctx, "input")
		return err
	case "delete":
		return rest.Collection(spec).Delete(ctx, resource.ID("input"))
	default:
		for _, err := range rest.List(ctx, spec, nil) {
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func TestCollectionSourceGuardStopsRetryAcrossOperations(t *testing.T) {
	for _, operation := range []string{"get", "list", "delete"} {
		t.Run(operation, func(t *testing.T) {
			requests, callbacks := 0, 0
			client := limitsGuardedClient(func(*http.Request) (*http.Response, error) {
				requests++
				return limitsGuardedWire(503, io.NopCloser(strings.NewReader("original503"))), nil
			})
			cause := errors.New("source changed")
			guard := limitsGuardedSnapshot(client, cause)
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				callbacks++
				client.Endpoint = "https://replacement.test/v2/"
				return nil
			}
			err := sourceGuardOperation(sourceGuardSpec(client, guard), operation)
			if !errors.Is(err, cause) || !gophercloud.ResponseCodeIs(err, 503) || requests != 1 || callbacks != 1 {
				t.Fatal(err, requests, callbacks)
			}
		})
	}
}

func TestCollectionSourceGuardKeepsAcceptedResponseEvidence(t *testing.T) {
	for _, operation := range []string{"get", "list", "delete"} {
		t.Run(operation, func(t *testing.T) {
			var client *gophercloud.ServiceClient
			requests := 0
			code, payload := 200, `{"item":{"id":"input"},"items":[{"id":"input"}]}`
			if operation == "delete" {
				code, payload = 204, ""
			}
			client = limitsGuardedClient(func(*http.Request) (*http.Response, error) {
				requests++
				return limitsGuardedWire(code, &limitsGuardedBody{Reader: strings.NewReader(payload), onClose: func() { client.Endpoint = "https://replacement.test/v2/" }}), nil
			})
			cause := errors.New("accepted source changed")
			err := sourceGuardOperation(sourceGuardSpec(client, limitsGuardedSnapshot(client, cause)), operation)
			var proof *resource.ResponseError
			if !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != payload || requests != 1 {
				t.Fatal(err, proof, requests)
			}
		})
	}
}

func TestCollectionDeleteDoesNotIgnoreExpanded404StatusPolicy(t *testing.T) {
	requests, callbacks := 0, 0
	client := limitsGuardedClient(func(*http.Request) (*http.Response, error) {
		requests++
		code := 503
		if requests == 2 {
			code = 404
		}
		return limitsGuardedWire(code, io.NopCloser(strings.NewReader("actual response"))), nil
	})
	client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
		callbacks++
		options.OkCodes = append(options.OkCodes, 404)
		return nil
	}
	err := sourceGuardOperation(sourceGuardSpec(client, func(context.Context) error { return nil }), "delete")
	if err == nil || errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) || requests != 2 || callbacks != 1 {
		t.Fatal(err, requests, callbacks)
	}
}
