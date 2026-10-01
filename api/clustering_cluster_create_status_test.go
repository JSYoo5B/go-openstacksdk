package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

// Senlin's published API uses 201, while its 16.0.0 router configures
// cluster_create with success=202. Both retain the full cluster response;
// an accepted 202 must also identify the asynchronous action.
func TestClusteringClusterCreatePublishedAndControllerSuccessCodes(t *testing.T) {
	for _, test := range []struct {
		name     string
		code     int
		location string
	}{
		{"published-without-location", http.StatusCreated, ""},
		{"published-with-location", http.StatusCreated, "/reverse/senlin/v1/actions/create-action"},
		{"controller-accepted", http.StatusAccepted, "/reverse/senlin/v1/actions/create-action"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := `{"cluster":{"id":"returned-cluster","name":"returned-name","profile_id":"profile-id","profile_name":null,"project":"project-id","user":"user-id","domain":null,"min_size":0,"max_size":-1,"desired_capacity":2,"timeout":3600,"init_at":null,"created_at":"2026-10-01T00:00:00","updated_at":null,"config":{"enabled":false},"metadata":{"large":9007199254740993},"data":{"ratio":1.0000000000000001},"dependents":{},"nodes":[],"policies":[],"status":"CREATING","status_reason":"Initializing","future":null}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/reverse/senlin/v1/clusters" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error("create followed Location or changed request", r.Method, r.URL, r.Header)
				}
				var requestBody map[string]map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil || len(requestBody) != 1 || len(requestBody["cluster"]) != 2 || string(requestBody["cluster"]["name"]) != `"requested-name"` || string(requestBody["cluster"]["profile_id"]) != `"profile-id"` {
					t.Error(requestBody, err)
				}
				w.Header().Set("X-Request-ID", "create-status-evidence")
				if test.location != "" {
					w.Header().Set("Location", test.location)
				}
				testcloud.JSON(w, test.code, body)
			})
			value, err := clusters.New(cloud.Client("clustering", "/reverse/senlin/v1")).Create(context.Background(), clusters.CreateOpts{Name: "requested-name", ProfileID: "profile-id"})
			if err != nil || value == nil {
				t.Fatal(value, err)
			}
			if value.ID != "returned-cluster" || value.Name != "returned-name" || value.Status != "CREATING" || value.StatusCode != test.code || value.Header.Get("X-Request-ID") != "create-status-evidence" || string(value.UserMetadata["large"]) != "9007199254740993" || string(value.Data["ratio"]) != "1.0000000000000001" || string(value.Body["future"]) != "null" || calls.Load() != 1 {
				t.Fatal("cluster response evidence changed", value, calls.Load())
			}
			if test.location == "" {
				if value.Operation != nil {
					t.Fatal("201 invented an action", value.Operation)
				}
			} else if value.Operation == nil || value.Operation.ActionID != "create-action" || value.Operation.Location != test.location || value.Operation.StatusCode != test.code || value.Operation.Header.Get("X-Request-ID") != "create-status-evidence" || string(value.Operation.Body) != body {
				t.Fatal("accepted action evidence changed", value.Operation)
			}
		})
	}
}

func TestClusteringClusterCreateAcceptedRequiresValidActionLocation(t *testing.T) {
	for _, test := range []struct {
		name     string
		location []string
	}{
		{"missing", nil},
		{"empty", []string{""}},
		{"foreign-origin", []string{"https://foreign.invalid/reverse/senlin/v1/actions/create-action"}},
		{"wrong-collection", []string{"/reverse/senlin/v1/clusters/create-action"}},
		{"query", []string{"/reverse/senlin/v1/actions/create-action?follow=1"}},
		{"multiple", []string{"/reverse/senlin/v1/actions/first", "/reverse/senlin/v1/actions/second"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := `{"cluster":{"id":"accepted-cluster","name":"cluster","profile_id":"profile","status":"CREATING","metadata":{"large":9007199254740993},"future":false}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/reverse/senlin/v1/clusters" {
					t.Error("accepted mutation resent or followed Location", r.Method, r.URL)
				}
				w.Header().Set("X-Request-ID", "accepted-create-location")
				for _, location := range test.location {
					w.Header().Add("Location", location)
				}
				testcloud.JSON(w, http.StatusAccepted, body)
			})
			value, err := clusters.New(cloud.Client("clustering", "/reverse/senlin/v1")).Create(context.Background(), clusters.CreateOpts{Name: "cluster", ProfileID: "profile"})
			var evidence *resource.ResponseError
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &evidence) || evidence.StatusCode != http.StatusAccepted || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "accepted-create-location" || len(evidence.Header.Values("Location")) != len(test.location) || calls.Load() != 1 {
				t.Fatal("accepted error lost evidence or was resent", value, err, evidence, calls.Load())
			}
			for index, location := range test.location {
				if evidence.Header.Values("Location")[index] != location {
					t.Fatal("Location evidence changed", evidence.Header)
				}
			}
		})
	}
}

func TestClusteringClusterCreateUndeclaredStatusAndMalformedAcceptedBody(t *testing.T) {
	for _, test := range []struct {
		name string
		code int
		body string
	}{
		{"unexpected-200", http.StatusOK, `{"cluster":{"id":"unexpected"}}`},
		{"malformed-202", http.StatusAccepted, `{"cluster":null}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Location", "/senlin/v1/actions/create-action")
				w.Header().Set("X-Request-ID", "create-error-evidence")
				testcloud.JSON(w, test.code, test.body)
			})
			value, err := clusters.New(cloud.Client("clustering", "/senlin/v1")).Create(context.Background(), clusters.CreateOpts{Name: "cluster", ProfileID: "profile"})
			var evidence *resource.ResponseError
			if test.code == http.StatusOK {
				if !gophercloud.ResponseCodeIs(err, http.StatusOK) || errors.As(err, &evidence) {
					t.Fatal("unexpected HTTP code lost native error", err)
				}
			} else if !errors.As(err, &evidence) || evidence.StatusCode != test.code || string(evidence.Body) != test.body || evidence.Header.Get("X-Request-ID") != "create-error-evidence" || evidence.Header.Get("Location") != "/senlin/v1/actions/create-action" {
				t.Fatal("accepted decode error lost evidence", err, evidence)
			}
			if value != nil || calls.Load() != 1 {
				t.Fatal("create error resent HTTP or fabricated resource", value, calls.Load())
			}
		})
	}
}
