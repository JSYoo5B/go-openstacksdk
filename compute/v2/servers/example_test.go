package servers_test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
)

func computeV2(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleWithGetPasswordPrivateKey() {
	ctx := context.Background()
	service := computeV2(ctx)
	encrypted, err := service.Servers.GetPassword(ctx, "server-id")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(encrypted)
	data, err := os.ReadFile("id_rsa")
	if err != nil {
		log.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		log.Fatal("id_rsa is not PEM")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}
	password, err := service.Servers.GetPassword(ctx, "server-id", servers.WithGetPasswordPrivateKey(privateKey))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(password)
}
