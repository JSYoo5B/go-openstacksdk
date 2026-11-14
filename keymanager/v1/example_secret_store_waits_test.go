package v1_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
)

func Example_secretStoreWaits() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	id := flag.String("id", "", "literal SecretStore identity")
	status := flag.String("status", "ACTIVE", "desired state")
	deleting := flag.Bool("delete", false, "poll GET for absence/deleted")
	flag.Parse()
	if err := runSecretStoreWaits(*cloud, *id, *status, *deleting); err != nil {
		log.Fatal(err)
	}
}

func runSecretStoreWaits(cloud, id, status string, deleting bool) error {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		return err
	}
	api := waits.New(service.RawClient())
	target := waits.SecretStore(id)
	var value *waits.Record
	if deleting {
		value, err = api.WaitForDelete(ctx, target)
	} else {
		value, err = api.WaitForStatus(ctx, target, &status)
	}
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(value.Resource.Body, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}
