package network

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestQoSReadGateFreshCompleteInventoryAndOuterGuardDoesNotRecurse(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/v2.0/")
	service := New(client)
	extensionCalls, policyCalls := 0, 0
	cloud.Mux.HandleFunc("GET /v2.0/extensions", func(w http.ResponseWriter, r *http.Request) {
		extensionCalls++
		if r.URL.Query().Get("marker") == "tail" {
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"tail"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}],"next":"`+cloud.Server.URL+`/v2.0/extensions?marker=tail"}`)
	})
	cloud.Mux.HandleFunc("GET /v2.0/qos/policies/one", func(w http.ResponseWriter, r *http.Request) {
		policyCalls++
		testcloud.JSON(w, 200, `{"policy":{"id":"one"}}`)
	})
	for range 2 {
		state, err := qosReadStart(context.Background(), service, "qos")
		if err != nil || len(state.Extensions) != 2 {
			t.Fatal(state, err)
		}
		ctx := rest.WithOperationGuard(context.Background(), state.Check)
		policy, err := state.Policy(ctx, "one")
		if err != nil || policy == nil || string(policy.Resource.Body["id"]) != `"one"` {
			t.Fatal(policy, err)
		}
	}
	if extensionCalls != 4 || policyCalls != 2 {
		t.Fatal(extensionCalls, policyCalls)
	}
}

func TestQoSReadGateUnavailableAndLaterUnhashableAliasKeepActualProof(t *testing.T) {
	for _, payload := range []string{`{"extensions":[{"alias":"other"}]}`, `{"extensions":[{"alias":"qos"},{"alias":{}}]}`} {
		t.Run(payload, func(t *testing.T) {
			cloud := testcloud.New(t)
			service := New(cloud.Client("network", "/v2.0/"))
			calls := 0
			cloud.Mux.HandleFunc("GET /v2.0/extensions", func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("X-Request-ID", "extension-proof")
				testcloud.JSON(w, 200, payload)
			})
			state, err := qosReadStart(context.Background(), service, "qos")
			if err == nil || state == nil || len(state.Extensions) == 0 || calls != 1 {
				t.Fatal(state, err, calls)
			}
			if strings.Contains(payload, `"alias":{}`) {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != payload || proof.Header.Get("X-Request-ID") != "extension-proof" {
					t.Fatal(err, proof)
				}
			} else if !errors.Is(err, ErrUnavailableExtension) {
				t.Fatal(err)
			}
		})
	}
}

func TestQoSReadSelectedHeadersAreFrozenAndOriginalServiceMutationStopsNextPhase(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/v2.0/")
	client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
	service := New(client)
	calls := 0
	cloud.Mux.HandleFunc("GET /v2.0/extensions", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2.0/qos/policies/one", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"policy":{"id":"one"}}`)
	})
	state, err := qosReadStart(context.Background(), service, "qos")
	if err != nil {
		t.Fatal(err)
	}
	client.MoreHeaders["X-Late"] = "after"
	client.MoreHeaders["X-Snapshot"] = "after"
	if _, err := state.Policy(context.Background(), "one"); err != nil {
		t.Fatal(err)
	}
	client.Endpoint += "changed/"
	if _, err := state.Policy(context.Background(), "one"); !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
		t.Fatal(err, calls)
	}
}

func TestQoSReadContextAndSelectedAPIValidationBeforeHTTP(t *testing.T) {
	if _, err := qosReadStart(nil, nil, "qos"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cloud := testcloud.New(t)
	service := New(cloud.Client("network", "/v2.0/"))
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("cancel QoS read")
	cancel(cause)
	if _, err := qosReadStart(ctx, service, "qos"); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	state, err := qosReadStart(context.Background(), service)
	if err != nil {
		t.Fatal(err)
	}
	service.API = nil
	if err := state.Check(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
