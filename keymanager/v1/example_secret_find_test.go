package v1_test

import (
	"context"
	"flag"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_secretFind() {
	identity := flag.String("identity", "", "literal secret name or ID")
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	strict := flag.Bool("strict", false, "report logical absence")
	flag.Parse()
	if *identity == "" {
		log.Fatal("-identity is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	keyManager, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	secret, err := keyManager.Secrets.FindRecord(ctx, *identity,
		request.IgnoreMissing(!*strict))
	if err != nil {
		// An optional returned receipt has Resource=nil on failure.
		// Inspect errors.As for native/accepted HTTP evidence as needed.
		log.Fatal(err)
	}
	if secret == nil {
		log.Print("secret absent")
		return
	}
	log.Printf("id=%s name=%s status=%s",
		secret.Resource.Body["id"], secret.Resource.Body["name"],
		secret.Resource.Body["status"])
}
