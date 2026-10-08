package cloudread

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/request"
)

func TestOwnReadConfigRetainsPresenceAndOwnsRequestCarriers(t *testing.T) {
	query := url.Values{"empty": nil, "many": {"one", "two"}}
	fields := map[string]json.RawMessage{"number": json.RawMessage(`9007199254740993`)}
	headers := map[string]string{"X-Test": "original"}
	arguments := map[string]any{"selector": "original"}
	config := request.Config[string]{Options: "typed", Query: query, Fields: fields, Headers: headers, Arguments: arguments}
	OwnReadConfig(&config)
	query["many"][0] = "changed"
	fields["number"][0] = '1'
	headers["X-Test"] = "changed"
	arguments["selector"] = "changed"
	if config.Options != "typed" || config.Query["many"][0] != "one" || string(config.Fields["number"]) != "9007199254740993" || config.Headers["X-Test"] != "original" || config.Arguments["selector"] != "original" {
		t.Fatalf("caller mutation changed owned carriers: %#v", config)
	}
	if _, present := config.Query["empty"]; !present || config.Query["empty"] != nil {
		t.Fatalf("lost null query presence: %#v", config.Query)
	}
	empty := request.Config[int]{}
	OwnReadConfig(&empty)
	empty.Query.Set("q", "value")
	empty.Fields["field"] = json.RawMessage(`null`)
	empty.Headers["X-Test"] = "value"
	empty.Arguments["arg"] = "value"
}
