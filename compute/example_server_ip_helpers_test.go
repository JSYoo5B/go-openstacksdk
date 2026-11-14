package compute_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_serverIpHelpers() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	id := flag.String("server-id", "", "existing Nova server ID")
	mode := flag.String("mode", "dispatch", "dispatch or list")
	pool := flag.String("pool", "", "exact external network name")
	ips := flag.String("ips", "", "comma-separated existing IPv4 addresses")
	wait := flag.Bool("wait", false, "observe each address in raw Nova")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := runServerIpHelpers(ctx, *cloud, *id, *mode, *pool, *ips, *wait); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServerIpHelpers(ctx context.Context, cloud, id, mode, pool, ips string, wait bool) error {
	if id == "" || (mode != "dispatch" && mode != "list") {
		return fmt.Errorf("provide -server-id and -mode dispatch or list")
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	server, err := service.Servers.Get(ctx, id)
	if err != nil {
		return err
	}
	if server == nil || server.ID != id {
		return fmt.Errorf("Nova returned a different or missing server")
	}
	var addresses []string
	if ips != "" {
		addresses = strings.Split(ips, ",")
		for i := range addresses {
			addresses[i] = strings.TrimSpace(addresses[i])
		}
	}
	var automatic []compute.AutomaticFloatingIPOption
	if len(addresses) != 0 {
		automatic = append(automatic, compute.WithFloatingIPAddresses(addresses...))
	}
	if pool != "" {
		automatic = append(automatic, compute.WithFloatingIPPool(resource.Name(pool)))
	}
	options := []compute.ServerIPOption{
		compute.WithServerIPAutomaticOptions(automatic...),
		compute.WithServerIPWait(wait),
	}
	var result *compute.AutomaticServerIPResult
	if mode == "list" {
		result, err = conn.AddIPList(ctx, server, addresses, options...)
	} else {
		result, err = conn.AddIPsToServer(ctx,
			compute.AutomaticFloatingIPRequest{Server: server}, options...)
	}
	if result != nil {
		fmt.Printf("mode=%s observed=%t\n", result.Mode, result.Observed)
		if result.Assignment != nil && result.Assignment.FloatingIP != nil {
			ip := result.Assignment.FloatingIP
			fmt.Printf("ip=%s address=%s status=%s\n", ip.ID, ip.FloatingIP, ip.Status)
		}
		for _, attempt := range result.Attempts {
			// ShortCircuited: the server already showed this address.
			// AttachmentNotFound: Neutron returned 404 after selection and
			// the item continued, so no association is confirmed.
			fmt.Printf("step=%d requested=%s completed=%t short=%t observed=%t notfound=%v error=%v\n",
				attempt.Index, attempt.RequestedAddress, attempt.Completed,
				attempt.ShortCircuited, attempt.Observed,
				attempt.AttachmentNotFound, attempt.Error)
		}
	}
	return err
}
