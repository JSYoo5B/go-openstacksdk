package neutronfind_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestSourcePatchOwnsMemberPayloadSeedAndPhysicalProof(t *testing.T) {
	bodyRaw := json.RawMessage(`{"nested":[false,9007199254740993]}`)
	seedRaw := json.RawMessage(`{"saved":true}`)
	body := map[string]json.RawMessage{"shared": json.RawMessage(`false`), "patch_only": bodyRaw}
	seed := map[string]json.RawMessage{"id": json.RawMessage(`"seed-id"`), "name": json.RawMessage(`"seed-name"`), "tenant_id": json.RawMessage(`"owner"`), "seed_only": seedRaw}
	const wire = `{"record":{"id":"actual","shared":true,"revision":12.9},"receipt":"whole"}`
	calls := 0
	engine, client := mutationEngine(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/neutron/v2.0/records/folder%2Fname%25" || req.URL.RawQuery != "" || req.Header.Get("OpenStack-API-Version") != "network 2.1" || req.Header.Get("If-Match") != "revision-one" || req.Header.Get("Content-Type") != "application/json" {
			t.Fatal(req.Method, req.URL, req.Header)
		}
		if got := mutationRequestBody(t, req); got != `{"record":{"patch_only":{"nested":[false,9007199254740993]},"shared":false}}` {
			t.Fatal(got)
		}
		seedRaw[1] = 'X'
		bodyRaw[1] = 'X'
		seed["late"] = json.RawMessage(`true`)
		return mutationResponse(203, wire), nil
	})
	version := "2.1"
	row, err := engine.Mutate(context.Background(), http.MethodPatch, "folder/name%", body, seed, neutronfind.Options{Microversion: &version, Headers: map[string]string{"If-Match": "revision-one"}})
	if err != nil || row == nil || row.Resource == nil || row.Wire == nil || calls != 1 {
		t.Fatal(row, err, calls)
	}
	for key, want := range map[string]string{"id": `"actual"`, "name": `"seed-name"`, "project_id": `"owner"`, "is_shared": `true`, "revision_number": `12`, "tags": `[]`, "seed_only": `{"saved":true}`} {
		if string(row.Resource.Body[key]) != want {
			t.Fatal(key, row.Resource.Body)
		}
	}
	if row.Resource.Body["late"] != nil || row.Wire.Body["seed_only"] != nil || string(row.Wire.Body["revision"]) != "12.9" || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "mutation-actual" || string(row.Envelope) != wire || client.Microversion != "" {
		t.Fatal(row, client)
	}
	row.Resource.Body["id"][1] = 'X'
	row.Header.Set("X-Request-Id", "consumer")
	if string(row.Wire.Body["id"]) != `"actual"` || string(row.Envelope) != wire || row.Resource.Header.Get("X-Request-Id") != "mutation-actual" {
		t.Fatal(row)
	}
}

func TestSourcePatchPreflightAndRejectedRepresentationKeepBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		body     map[string]json.RawMessage
		policy   neutronfind.Options
	}{
		{name: "empty identity"},
		{name: "malformed body", id: "member", body: map[string]json.RawMessage{"name": json.RawMessage(`{bad`)}},
		{name: "read query", id: "member", policy: neutronfind.Options{Query: map[string]json.RawMessage{"name": json.RawMessage(`"value"`)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			engine, _ := mutationEngine(t, func(*http.Request) (*http.Response, error) { calls++; return mutationResponse(200, `{}`), nil })
			row, err := engine.Mutate(context.Background(), http.MethodPatch, tc.id, tc.body, nil, tc.policy)
			if row != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(row, err, calls)
			}
		})
	}
	for _, code := range []int{202, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			const raw = `{"record":[]}`
			calls := 0
			engine, _ := mutationEngine(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPatch {
					t.Fatal(req.Method)
				}
				return mutationResponse(code, raw), nil
			})
			row, err := engine.Mutate(context.Background(), http.MethodPatch, "member", nil, nil, neutronfind.Options{})
			if err == nil || calls != 1 {
				t.Fatal(row, err, calls)
			}
			if code < 400 {
				mutationProof(t, err, code, raw)
				if row == nil || row.Resource != nil || string(row.Envelope) != raw {
					t.Fatal(row)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodPatch || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "mutation-actual" {
					t.Fatal(row, err, native)
				}
			}
		})
	}
}

func TestSourcePatchPhysicalFaultIsTerminalAndKeepsAllCauses(t *testing.T) {
	for _, code := range []int{200, 401, 404, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			const raw = `{"record":{"id":"actual"}}`
			readCause, closeCause, cancelCause := errors.New("PATCH read"), errors.New("PATCH close"), errors.New("PATCH cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries, reauth := 0, 0, 0
			done := false
			var client *gophercloud.ServiceClient
			original := ""
			physical := &mutationFaultBody{closeCause: closeCause}
			physical.Reader = mutationFaultReader(func(p []byte) (int, error) {
				if done {
					return 0, io.EOF
				}
				done = true
				client.ResourceBase = "https://foreign.invalid/v2/"
				cancel(cancelCause)
				return copy(p, raw), readCause
			})
			physical.onClose = func() { client.ResourceBase = original }
			engine, selected := mutationEngine(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPatch {
					t.Fatal(req.Method)
				}
				response := mutationResponse(code, raw)
				response.Body = physical
				return response, nil
			})
			client = selected
			original = client.ResourceBase
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			client.ReauthFunc = func(context.Context) error { reauth++; return nil }
			row, err := engine.Mutate(ctx, http.MethodPatch, "member", map[string]json.RawMessage{"name": json.RawMessage(`"new"`)}, nil, neutronfind.Options{})
			if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || physical.closed != 1 || retries != 0 || reauth != 0 {
				t.Fatal(row, err, calls, physical.closed, retries, reauth)
			}
			if code < 400 {
				mutationProof(t, err, code, raw)
				if row == nil || row.Resource != nil || row.Wire != nil || string(row.Envelope) != raw {
					t.Fatal(row)
				}
			} else {
				receipt := err
				if code == 401 {
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &auth) || auth == nil || !errors.Is(auth.ErrReauth, readCause) || !errors.Is(auth.ErrReauth, closeCause) {
						t.Fatal(err, auth)
					}
					receipt = auth.ErrOriginal
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.As(receipt, &native) || native.Method != http.MethodPatch || native.Actual != code || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "mutation-actual" {
					t.Fatal(row, err, native)
				}
			}
		})
	}
}
