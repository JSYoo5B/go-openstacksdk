package identity_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
)

func printValueCloudUserRoleGet(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", encoded)
}
func Example_cloudUserRoleGet() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	user, role := os.Getenv("IDENTITY_USER_REF"), os.Getenv("IDENTITY_ROLE_REF")
	if user == "" || role == "" {
		log.Fatal("set IDENTITY_USER_REF and IDENTITY_ROLE_REF to readable IDs or names")
	}
	userOpts := []identity.UserGetOption{}
	roleOpts := []identity.RoleGetOption{}
	if domain, ok := os.LookupEnv("IDENTITY_DOMAIN_ID"); ok {
		userOpts = append(userOpts, identity.WithUserGetDomainID(domain))
		roleOpts = append(roleOpts, identity.WithRoleGetDomainID(domain))
	}
	directUser, err := conn.GetUserRecord(ctx, user, userOpts...)
	if err != nil {
		log.Fatal(err)
	}
	printValueCloudUserRoleGet(directUser.Value)
	filteredUser, err := conn.GetUserRecord(ctx, user, append(userOpts,
		identity.WithUserGetFilters(json.RawMessage(`{}`)))...)
	if err != nil {
		log.Fatal(err)
	}
	printValueCloudUserRoleGet(filteredUser.Value)
	pinnedUser, err := conn.GetUserRecord(ctx, user, append(userOpts,
		identity.WithUserGetExpression("`[null]`"))...)
	if err != nil {
		log.Fatal(err)
	}
	printValueCloudUserRoleGet(pinnedUser.Value)
	directRole, err := conn.GetRoleRecord(ctx, role, roleOpts...)
	if err != nil {
		log.Fatal(err)
	}
	printValueCloudUserRoleGet(directRole.Value)
	filteredRole, err := conn.GetRoleRecord(ctx, role, append(roleOpts,
		identity.WithRoleGetExpression("`[null]`"))...)
	if err != nil {
		log.Fatal(err)
	}
	printValueCloudUserRoleGet(filteredRole.Value)
}
