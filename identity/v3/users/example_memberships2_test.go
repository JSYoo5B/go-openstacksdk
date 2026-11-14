package users_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
)

func Example_memberships2() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runMemberships2(ctx); err != nil {
		log.Fatal(err)
	}
}

func runMemberships2(ctx context.Context) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s USER_ID", os.Args[0])
	}
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	identity, err := conn.Identity(ctx)
	if err != nil {
		return err
	}
	for record, err := range identity.Users.ListProjectRecords(ctx, os.Args[1],
		users.WithProjectListOptions(users.ListProjectRecordsOpts{Limit: 20, MaxItems: 50}),
		users.WithProjectListFilter("is_enabled", true),
	) {
		if err != nil {
			return fmt.Errorf("list user project records: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{
			"resource": record.Resource,
			"wire":     record.Wire,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	for record, err := range identity.Users.ListGroupRecords(ctx, os.Args[1]) {
		if err != nil {
			return fmt.Errorf("list user group records: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{
			"resource": record.Resource,
			"wire":     record.Wire,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}
