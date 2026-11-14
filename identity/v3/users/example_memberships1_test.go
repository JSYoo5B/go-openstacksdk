package users_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
)

func Example_memberships1() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runMemberships1(ctx); err != nil {
		log.Fatal(err)
	}
}

func runMemberships1(ctx context.Context) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s USER_ID", os.Args[0])
	}
	userID := os.Args[1]
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	identity, err := conn.Identity(ctx)
	if err != nil {
		return err
	}

	for project, err := range identity.Users.ListProjects(ctx, userID) {
		if err != nil {
			return fmt.Errorf("list user projects: %w", err)
		}
		fmt.Printf("project: %s %s domain=%s\n", project.ID, project.Name, project.DomainID)
	}
	for group, err := range identity.Users.ListGroups(ctx, userID) {
		if err != nil {
			return fmt.Errorf("list user groups: %w", err)
		}
		fmt.Printf("group: %s %s domain=%s\n", group.ID, group.Name, group.DomainID)
	}
	return nil
}
