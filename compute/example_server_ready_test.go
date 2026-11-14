package compute_test

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

func Example_serverReady() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	file := flag.String("server-json", "server.json", "supplied Nova server JSON object")
	mode := flag.String("mode", "active", "active or wait")
	activeWait := flag.Bool("active-wait", false, "wait for GetActiveServer IP readiness and Nova observation")
	external := flag.String("floating-network", "", "exact allocation network name; empty uses roles/router")
	enabled := flag.Bool("auto-ip", true, "enable conditional floating IPv4 assignment")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := runServerReady(ctx, *cloud, *file, *mode, *external, *enabled, *activeWait); err != nil {
		log.Fatal(err)
	}
}

func runServerReady(ctx context.Context, cloud, file, mode, external string, enabled, activeWait bool) error {
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
