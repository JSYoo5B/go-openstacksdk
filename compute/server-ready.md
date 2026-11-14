# Server Ready

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None:
	raise RuntimeError("server not found")

# Provided status is inspected. Non-ACTIVE returns None; default wait=False.
active = conn.get_active_server(server, auto_ip=True, reuse=True, wait=False, timeout=180)

# Only the supplied ID chooses the target; current state is fetched and polled.
ready = conn.wait_for_server(server, auto_ip=True, reuse=True, timeout=180)
print(ready.id, ready.interface_ip)
```

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	file := flag.String("server-json", "server.json", "supplied Nova server JSON object")
	mode := flag.String("mode", "active", "active or wait")
	activeWait := flag.Bool("active-wait", false, "wait for GetActiveServer IP readiness and Nova observation")
	external := flag.String("floating-network", "", "exact allocation network name; empty uses roles/router")
	enabled := flag.Bool("auto-ip", true, "enable conditional floating IPv4 assignment")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := run(ctx, *cloud, *file, *mode, *external, *enabled, *activeWait); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, file, mode, external string, enabled, activeWait bool) error {
	if mode != "active" && mode != "wait" {
		return fmt.Errorf("mode must be active or wait")
	}
	wire, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var server compute.Server
	if err := json.Unmarshal(wire, &server); err != nil {
		return fmt.Errorf("supplied server: %w", err)
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	input := compute.AutomaticFloatingIPRequest{Server: &server}
	if external != "" {
		input.Network = resource.Name(external)
	}
	options := []compute.ServerReadyOption{
		compute.WithServerReadyAutomaticIPOptions(
			compute.WithAutomaticIPEnabled(enabled),
			compute.WithAutomaticIPWait(resource.WithTimeout(180*time.Second), resource.WithPollInterval(time.Second)),
			compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
			compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(true)),
		),
		compute.WithServerReadyWaitOptions(resource.WithPollInterval(time.Second)),
		compute.WithActiveServerWait(activeWait),
	}
	var result *compute.AutomaticServerIPResult
	if mode == "wait" {
		result, err = conn.WaitForServer(ctx, input, options...)
	} else {
		result, err = conn.GetActiveServer(ctx, input, options...)
	}
	if result == nil && err == nil {
		fmt.Println("supplied server is not ACTIVE")
		return nil
	}
	if result != nil {
		if result.Server != nil {
			fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
		}
		if decision := result.Decision; decision != nil {
			fmt.Printf("needed=%t reason=%s backend=%s\n", decision.Needed, decision.Reason, decision.Backend)
		}
		if assignment := result.Assignment; assignment != nil && assignment.FloatingIP != nil {
			fmt.Printf("floating4=%s status=%s reused=%t allocated=%t\n", assignment.FloatingIP.FloatingIP, assignment.FloatingIP.Status, assignment.Reused, assignment.Allocated)
		}
		fmt.Printf("observed=%t\n", result.Observed)
		// WaitForServer only: the get_server interface expansion of the
		// ACTIVE poll result that entered the IP stage.
		if view := result.Interfaces; view != nil {
			fmt.Printf("interface=%s public4=%s private4=%s\n", view.InterfaceIP, view.PublicIPv4, view.PrivateIPv4)
		}
	}
	if err != nil {
		if errors.Is(err, compute.ErrServerAddressesUnavailable) {
			log.Printf("ACTIVE server has no usable address rows; resources were retained")
		}
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("response status=%d", response.StatusCode)
		}
		return fmt.Errorf("server readiness: %w", err)
	}
	return nil
}
```

## References

- [AddIPsToServer·AddIPList](server-ip-helpers.md)
- [wait_for_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1415)
- [get_active_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1417-L1481)
- [attach helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1109-L1123)
- [IP dispatch](server-ip-dispatch.md)
- [Legacy Nova backend](server-nova-floating-ip.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
