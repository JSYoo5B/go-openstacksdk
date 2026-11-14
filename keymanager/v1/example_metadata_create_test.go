package v1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
)

func Example_metadataCreate() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	secret, err := service.Secrets.CreateRecord(ctx, secrets.WithCreateRecordAttributes(map[string]any{
		"name": "stored-demo", "secret_type": secrets.OpaqueSecret,
		"payload": "demo", "payload_content_type": "text/plain",
	}))
	if err != nil {
		log.Fatal(err)
	}
	var secretRef string
	if err := json.Unmarshal(secret.Resource.Body["secret_ref"], &secretRef); err != nil || secretRef == "" {
		log.Fatalf("created secret has no usable reference: %v", err)
	}
	container, err := service.Containers.CreateRecord(ctx,
		containers.WithCreateRecordAttribute("type", containers.GenericContainer),
		containers.WithCreateRecordAttribute("name", "container-demo"),
		containers.WithCreateRecordAttribute("secret_refs", []map[string]any{{"name": "stored", "secret_ref": secretRef}}),
	)
	if err != nil {
		log.Fatal(err)
	}
	order, err := service.Orders.CreateRecord(ctx, orders.WithCreateRecordAttributes(map[string]any{
		"type": orders.KeyOrder, "meta": map[string]any{"algorithm": "aes", "bit_length": 256, "mode": "cbc"},
	}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("secret=%s container=%s order=%s\n", secret.Resource.Body["id"], container.Resource.Body["id"], order.Resource.Body["id"])
}
