package cloudread

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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

func TestApplyReadOptionsOwnsCallbackBoundariesAndRetainsTerminalCauses(t *testing.T) {
	type value struct{ Number *int }
	own := func(config *request.Config[value]) {
		if config.Options.Number != nil {
			copied := *config.Options.Number
			config.Options.Number = &copied
		}
		OwnReadConfig(config)
	}
	t.Run("retained option inputs cannot change the next callback", func(t *testing.T) {
		number := 7
		headers := map[string]string{"X-Input": "owned"}
		fields := map[string]json.RawMessage{"raw": json.RawMessage(`9007199254740993`)}
		query := url.Values{"empty": nil, "multi": {"owned", "two"}}
		var observed bool
		config, err := ApplyReadOptions(context.Background(), value{}, []request.Option[value]{
			func(c *request.Config[value]) error {
				c.Options.Number = &number
				c.Headers = headers
				c.Fields = fields
				c.Query = query
				return nil
			},
			func(c *request.Config[value]) error {
				number = 99
				headers["X-Input"] = "changed"
				fields["raw"][0] = '1'
				query["multi"][0] = "changed"
				observed = *c.Options.Number == 7 && c.Headers["X-Input"] == "owned" && string(c.Fields["raw"]) == `9007199254740993` && c.Query["multi"][0] == "owned"
				_, present := c.Query["empty"]
				observed = observed && present && c.Query["empty"] == nil
				return nil
			},
		}, own, nil)
		if err != nil || !observed || *config.Options.Number != 7 {
			t.Fatal("retained callback inputs changed owned config", config, observed, err)
		}
	})
	for _, phase := range []string{"nil option", "callback cause", "cancelled callback", "guard drift", "empty cancelled operation"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cause := errors.New("callback failure")
			drift := errors.New("binding changed")
			callbacks := 0
			changed := false
			guard := func(context.Context) error {
				if changed {
					return drift
				}
				return nil
			}
			first := request.Option[value](func(*request.Config[value]) error {
				callbacks++
				switch phase {
				case "callback cause":
					return cause
				case "cancelled callback":
					cancel()
					return cause
				case "guard drift":
					changed = true
				}
				return nil
			})
			options := []request.Option[value]{first, func(*request.Config[value]) error { callbacks++; return nil }}
			if phase == "nil option" {
				options[0] = nil
			}
			if phase == "empty cancelled operation" {
				options = nil
				cancel()
			}
			_, err := ApplyReadOptions(ctx, value{}, options, own, guard)
			switch phase {
			case "nil option":
				if !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
					t.Fatal(err, callbacks)
				}
			case "callback cause":
				if !errors.Is(err, cause) || callbacks != 1 {
					t.Fatal(err, callbacks)
				}
			case "cancelled callback":
				if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || callbacks != 1 {
					t.Fatal(err, callbacks)
				}
			case "guard drift":
				if !errors.Is(err, drift) || callbacks != 1 {
					t.Fatal(err, callbacks)
				}
			case "empty cancelled operation":
				if !errors.Is(err, context.Canceled) || callbacks != 0 {
					t.Fatal(err, callbacks)
				}
			}
		})
	}
}
