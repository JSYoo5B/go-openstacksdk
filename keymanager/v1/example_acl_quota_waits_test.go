package v1_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_aclQuotaWaits() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kind := flag.String("kind", "acl", "acl/quota")
	secretID := flag.String("secret-id", "", "literal secret ID for ACL")
	operation := flag.String("operation", "status", "status/delete")
	attribute := flag.String("attribute", "acl_ref", "declared string/null attribute")
	state := flag.String("state", "", "target string; empty string is explicit")
	nullState := flag.Bool("null-state", false, "target is Source None")
	interval := flag.Duration("interval", 2*time.Second, "polling interval")
	timeout := flag.Duration("timeout", 0, "nonzero override; zero keeps Source default")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var target waits.Target
	switch *kind {
	case "acl":
		target = waits.SecretACL(*secretID)
	case "quota":
		target = waits.Quota()
	default:
		log.Fatal("unknown kind")
	}
	options := []waits.Option{
		waits.WithWait(resource.WithPollInterval(*interval)),
		waits.WithWait(resource.WithProgressCallback(func(progress int) error {
			fmt.Println("progress", progress)
			return nil
		})),
	}
	if *timeout != 0 {
		options = append(options, waits.WithWait(resource.WithTimeout(*timeout)))
	}
	var result *waits.Record
	switch *operation {
	case "delete":
		result, err = service.Waits.WaitForDelete(ctx, target, options...)
	case "status":
		options = append(options, waits.WithWait(resource.WithStatusAttribute(*attribute)))
		selected := state
		if *nullState {
			selected = nil
		}
		result, err = service.Waits.WaitForStatus(ctx, target, selected, options...)
	default:
		log.Fatal("unknown operation")
	}
	if err != nil {
		log.Fatal(err)
	}
	encoded, err := json.MarshalIndent(result.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}
