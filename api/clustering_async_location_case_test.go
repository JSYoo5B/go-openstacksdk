package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestClusteringClusterMutationLocationHeaderCaseAndAmbiguity(t *testing.T) {
	for _, operation := range []string{"Create", "Update", "Delete"} {
		for _, mode := range []string{"lowercase", "ambiguous", "empty"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", "/reverse/senlin/v1")
				var calls atomic.Int32
				body := `{"cluster":{"id":"accepted-cluster"}}`
				code, method, path := http.StatusAccepted, http.MethodPatch, "/reverse/senlin/v1/clusters/selected"
				if operation == "Create" {
					code, method, path = http.StatusCreated, http.MethodPost, "/reverse/senlin/v1/clusters"
				}
				if operation == "Delete" {
					method = http.MethodDelete
				}
				client.ProviderClient.HTTPClient.Transport = clusterRoundTrip(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Method != method || r.URL.Path != path || r.Header.Get("X-Auth-Token") != "test-token" {
						t.Error(r.Method, r.URL, r.Header)
					}
					header := http.Header{"location": {"actions/accepted-action"}, "X-Request-Id": {"case-evidence"}}
					if mode == "ambiguous" {
						header["Location"] = []string{"actions/another-action"}
					}
					if mode == "empty" {
						header["location"] = []string{""}
					}
					return &http.Response{StatusCode: code, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				})
				api := clusters.New(client)
				var submission *actions.Submission
				var err error
				switch operation {
				case "Create":
					var value *clusters.Cluster
					value, err = api.Create(context.Background(), clusters.CreateOpts{Name: "selected", ProfileID: "profile"})
					if value != nil {
						submission = value.Operation
					}
				case "Update":
					var value *clusters.Cluster
					value, err = api.Update(context.Background(), resource.ID("selected"), clusters.UpdateOpts{}, clusters.WithUpdateMetadata(map[string]any{}))
					if value != nil {
						submission = value.Operation
					}
				case "Delete":
					submission, err = api.Delete(context.Background(), resource.ID("selected"))
				}
				if calls.Load() != 1 {
					t.Fatal("accepted mutation fetched or resent", calls.Load())
				}
				if mode == "lowercase" {
					if err != nil || submission == nil || submission.ActionID != "accepted-action" || submission.Location != "actions/accepted-action" || submission.StatusCode != code || string(submission.Body) != body || submission.Header["location"][0] != "actions/accepted-action" {
						t.Fatal(submission, err)
					}
				} else {
					var evidence *resource.ResponseError
					if submission != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &evidence) || evidence.StatusCode != code || string(evidence.Body) != body || evidence.Header.Get("X-Request-Id") != "case-evidence" {
						t.Fatal(submission, err, evidence)
					}
				}
			})
		}
	}
}
