package senlin_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy(t *testing.T) {
	config, err := request.Apply(struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	body, err := senlin.CommandBody(config, "check")
	if err != nil || string(body) != `{"check":{}}` {
		t.Fatal(string(body), err)
	}
	if _, err := senlin.Body(config, "node"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("ordinary mutation accepted no fields", err)
	}
	type recoverOptions struct {
		Check request.Optional[bool] `json:"check,omitzero"`
	}
	value := map[string]any{"off": false, "big": json.Number("9007199254740993")}
	extension := request.WithField[recoverOptions]("vendor", value)
	value["off"] = true
	options, err := request.Apply(recoverOptions{Check: request.Present(false)}, extension)
	if err != nil {
		t.Fatal(err)
	}
	body, err = senlin.CommandBody(options, "recover")
	if err != nil || string(body) != `{"recover":{"check":false,"vendor":{"big":9007199254740993,"off":false}}}` {
		t.Fatal(string(body), err)
	}
	for _, option := range []request.Option[recoverOptions]{request.WithField[recoverOptions]("check", true), request.WithField[recoverOptions]("id", "changed"), request.WithHeader[recoverOptions]("x-auth-token", "changed"), request.WithQuery[recoverOptions]("vendor", "ignored")} {
		config, err := request.Apply(recoverOptions{}, option)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := senlin.CommandBody(config, "recover", "id"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("invalid command extension accepted", err)
		}
	}
}

func TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/reverse/v1")
	for _, body := range []string{`{"action":"accepted","vendor":9007199254740993}`, `{}`, `{"action":null}`, `{"action":123}`, `{"action":{"id":"accepted"}}`, `{"action":"wrong"}`, `{"action":"bad/id"}`, `[]`, `null`, `{`} {
		for _, location := range []string{"actions/accepted", ""} {
			t.Run(body+location, func(t *testing.T) {
				response := &rest.Response{Body: json.RawMessage(body), Header: http.Header{"location": {location}, "X-Request-Id": {"accepted-request"}}, StatusCode: 202}
				identity, err := senlin.CommandActionID(client, response)
				if body == `{"action":"accepted","vendor":9007199254740993}` && location != "" {
					if err != nil || identity != "accepted" {
						t.Fatal(identity, err)
					}
					return
				}
				var evidence *resource.ResponseError
				if identity != "" || !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != body || evidence.Header.Get("X-Request-Id") != "accepted-request" {
					t.Fatal(identity, err, evidence)
				}
				response.Body[0] = '!'
				response.Header.Set("X-Request-Id", "changed")
				if string(evidence.Body) != body || evidence.Header.Get("X-Request-Id") != "accepted-request" {
					t.Fatal("response evidence was shared", evidence)
				}
			})
		}
	}
	if _, err := senlin.CommandActionID(client, nil); err == nil {
		t.Fatal("missing response accepted")
	}
}
