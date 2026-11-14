package portbindings_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_portBindings() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	port := flag.String("port", "", "Port ID")
	host := flag.String("host", "", "Binding host")
	mode := flag.String("mode", "", "create, list, activate or delete")
	flag.Parse()
	if *port == "" || *host == "" {
		log.Fatal("port and host are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := network.PortBindings
	var value any
	switch *mode {
	case "create":
		value, err = api.Create(ctx, *port,
			portbindings.WithAttribute("host", *host),
			portbindings.WithAttribute("profile", map[string]any{"capabilities": []string{"switchdev"}}))
	case "list":
		value, err = api.All(ctx, *port, request.Query("host", *host))
	case "activate":
		value, err = api.Activate(ctx, *port, *host)
	case "delete":
		value, err = api.Delete(ctx, *port, *host)
	default:
		log.Fatal("mode must be create, list, activate or delete")
	}
	if err != nil {
		log.Fatal(err)
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}
