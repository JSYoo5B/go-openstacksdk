package compute_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_consoleSelection() {
	serverID := flag.String("server-id", "", "server ID")
	kind := flag.String("console-type", "novnc", "console type")
	protocol := flag.String("protocol", "", "optional console protocol")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runConsoleSelection(ctx, *serverID, *kind, *protocol); err != nil {
		log.Fatal(err)
	}
}

func runConsoleSelection(ctx context.Context, serverID, kind, protocol string) error {
	if serverID == "" {
		return fmt.Errorf("-server-id is required")
	}
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	var options []compute.ConsoleOption
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "protocol" {
			options = append(options, compute.ConsoleOption(func(c *request.Config[compute.ConsoleOpts]) error {
				c.Options.Protocol = request.Present(protocol)
				return nil
			}))
		}
	})
	console, err := service.CreateConsole(ctx, serverID, kind, options...)
	if err != nil {
		return err
	}
	fmt.Println(string(console))
	return nil
}
