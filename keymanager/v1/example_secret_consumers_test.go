package v1_test

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretconsumers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_secretConsumers() {
	action := flag.String("action", "list", "create, delete, or list")
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	secret := flag.String("secret", "", "literal secret ID")
	service := flag.String("service", "image", "consumer service")
	kind := flag.String("resource-type", "image", "consumer resource type")
	id := flag.String("resource-id", "example-image-id", "consumer resource ID")
	strict := flag.Bool("strict", false, "report missing deletion")
	flag.Parse()
	if *secret == "" {
		log.Fatal("-secret is required")
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
	attrs := map[string]any{
		"service": *service, "resource_type": *kind, "resource_id": *id,
	}
	switch *action {
	case "create":
		result, err := keyManager.SecretConsumers.CreateRecord(ctx, *secret,
			secretconsumers.WithCreateRecordAttributes(attrs))
		if err != nil {
			log.Fatal(err)
		}
		view, err := json.MarshalIndent(result.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("POST status=%d resource=%s actual_response=%s",
			result.StatusCode, view, result.Envelope)
	case "delete":
		result, err := keyManager.SecretConsumers.DeleteRecord(ctx, *secret,
			secretconsumers.WithDeleteRecordAttributes(attrs),
			request.IgnoreMissing(!*strict))
		if err != nil {
			log.Fatal(err)
		}
		if result == nil {
			log.Print("association absent")
			return
		}
		log.Printf("DELETE status=%d actual_response=%s",
			result.StatusCode, result.Envelope)
	case "list":
		for consumer, err := range keyManager.SecretConsumers.ListRecords(ctx,
			*secret, secretconsumers.WithListRecordQuery(
				map[string]any{"service": *service})) {
			if err != nil {
				log.Fatal(err)
			}
			view, err := json.MarshalIndent(consumer.Resource, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("secret=%s resource=%s", consumer.SecretID, view)
		}
	default:
		log.Fatal("-action must be create, delete, or list")
	}
}
