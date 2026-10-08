package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestClusteringPolicyGetAndValidateRetainMalformedAcceptedResponseEvidence(t *testing.T) {
	for _, operation := range []string{"Get", "Validate"} {
		for _, body := range []string{`{}`, `{"policy":null}`, `{"policy":[]}`, `{"policy":{"spec":[]}}`, `{"policy":`} {
			t.Run(operation+"/"+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				calls := 0
				method, path := "GET", "/senlin/v1/policies/identity"
				if operation == "Validate" {
					method, path = "POST", "/senlin/v1/policies/validate"
				}
				cloud.Mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, r *http.Request) {
					calls++
					w.Header().Set("X-Request-Id", "accepted-evidence")
					testcloud.JSON(w, 200, body)
				})
				client := cloud.Client("clustering", "/senlin/v1")
				client.Microversion = "1.2"
				api := policies.New(client)
				var value *policies.Policy
				var err error
				if operation == "Get" {
					value, err = api.Get(context.Background(), "identity")
				} else {
					value, err = api.Validate(context.Background(), policies.ValidateOpts{Spec: json.RawMessage(`{}`)})
				}
				var evidence *resource.ResponseError
				if value != nil || !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != body || evidence.Header.Get("X-Request-Id") != "accepted-evidence" || calls != 1 {
					t.Fatalf("value=%v evidence=%+v calls=%d err=%v", value, evidence, calls, err)
				}
			})
		}
	}
}

func TestClusteringPolicyGetAndValidateRejectUndeclaredSuccessCodes(t *testing.T) {
	for _, operation := range []string{"Get", "Validate"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			method, path := "GET", "/senlin/v1/policies/identity"
			if operation == "Validate" {
				method, path = "POST", "/senlin/v1/policies/validate"
			}
			cloud.Mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 201, `{"policy":{"id":"unexpected-code"}}`)
			})
			client := cloud.Client("clustering", "/senlin/v1")
			client.Microversion = "1.2"
			api := policies.New(client)
			var err error
			if operation == "Get" {
				_, err = api.Get(context.Background(), "identity")
			} else {
				_, err = api.Validate(context.Background(), policies.ValidateOpts{Spec: json.RawMessage(`{}`)})
			}
			if !gophercloud.ResponseCodeIs(err, 201) {
				t.Fatalf("undeclared code accepted: %v", err)
			}
		})
	}
}
