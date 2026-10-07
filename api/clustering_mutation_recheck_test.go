package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/policies"
	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/profiles"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestClusteringMutationsRecheckConfiguredSourceAfterOptions(t *testing.T) {
	for _, operation := range []string{"profile_create", "profile_validate", "policy_validate"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				testcloud.JSON(w, 200, `{}`)
			})
			client := cloud.Client("clustering", "/senlin/v1")
			client.Microversion = "1.2"
			var err error
			if operation == "profile_create" {
				_, err = profiles.New(client).Create(context.Background(), profiles.CreateOpts{Name: "template", Spec: json.RawMessage(`{}`)},
					func(config *request.Config[profiles.CreateOpts]) error { client.Type = "compute"; return nil })
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("changed service type accepted: %v", err)
				}
			} else {
				if operation == "profile_validate" {
					_, err = profiles.New(client).Validate(context.Background(), profiles.ValidateOpts{Spec: json.RawMessage(`{}`)},
						func(config *request.Config[profiles.ValidateOpts]) error { client.Microversion = "1.1"; return nil })
				} else {
					_, err = policies.New(client).Validate(context.Background(), policies.ValidateOpts{Spec: json.RawMessage(`{}`)},
						func(config *request.Config[policies.ValidateOpts]) error { client.Microversion = "1.1"; return nil })
				}
				if !errors.Is(err, resource.ErrUnsupported) {
					t.Fatalf("demoted validation version accepted: %v", err)
				}
			}
			if calls != 0 {
				t.Fatalf("changed source sent %d requests", calls)
			}
		})
	}
}
