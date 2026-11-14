package compute_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
)

func Example_consoleAuthToken() {
	token := flag.String("console-token", "", "console token to look up")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runConsoleAuthToken(ctx, *token); err != nil {
		log.Fatal(err)
	}
}

func runConsoleAuthToken(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("-console-token is required")
	}
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	record, err := service.ValidateConsoleAuthToken(ctx, token)
	if record != nil {
		data, marshalErr := json.MarshalIndent(map[string]any{
			"resource":    record.Resource,
			"wire":        record.Wire,
			"status_code": record.StatusCode,
			"header":      record.Header,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Println(string(data))
	}
	return err
}
