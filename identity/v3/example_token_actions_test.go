package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens"
)

func Example_tokenActions() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	token := flag.String("token", "", "subject token")
	allowExpired := flag.Bool("allow-expired", false, "allow expired subject token")
	revoke := flag.Bool("revoke", false, "revoke after validation and check")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	record, err := service.Tokens.ValidateToken(ctx, *token,
		func(c *tokens.ValidateTokenOpts) error { c.NoCatalog = true; return nil },
		func(c *tokens.ValidateTokenOpts) error { c.AllowExpired = *allowExpired; return nil })
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(record.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
	valid, err := service.Tokens.CheckToken(ctx, *token,
		func(c *tokens.CheckTokenOpts) error { c.AllowExpired = *allowExpired; return nil })
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("valid: %t\n", valid)
	if *revoke {
		if err := service.Tokens.RevokeToken(ctx, *token); err != nil {
			log.Fatal(err)
		}
	}
}
