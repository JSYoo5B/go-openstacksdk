package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_metadataFetch() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	kind := flag.String("kind", "container", "container or order")
	id := flag.String("id", "", "explicit resource ID")
	timeout := flag.Duration("timeout", time.Minute, "authentication and metadata query timeout")
	flag.Parse()
	if err := runMetadataFetch(*cloud, *kind, *id, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runMetadataFetch(cloud, kind, id string, timeout time.Duration) error {
	if timeout <= 0 || id == "" || (kind != "container" && kind != "order") {
		return fmt.Errorf("positive timeout, explicit id and kind container/order are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		return err
	}
	var output map[string]any
	switch kind {
	case "container":
		value, fetchErr := service.Containers.Fetch(ctx, resource.ID(id))
		err = fetchErr
		if value != nil {
			output = map[string]any{
				"request_id":   value.RequestID,
				"container_id": value.ContainerID,
				"resource":     value.Resource,
				"wire":         value.Wire,
				"envelope":     string(value.Envelope),
				"header":       value.Header,
				"status_code":  value.StatusCode,
			}
		}
	case "order":
		value, fetchErr := service.Orders.Fetch(ctx, resource.ID(id))
		err = fetchErr
		if value != nil {
			output = map[string]any{
				"request_id":  value.RequestID,
				"order_id":    value.OrderID,
				"secret_id":   value.SecretID,
				"resource":    value.Resource,
				"wire":        value.Wire,
				"envelope":    string(value.Envelope),
				"header":      value.Header,
				"status_code": value.StatusCode,
			}
		}
	}
	if output != nil {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		err = errors.Join(err, encoder.Encode(output))
	}
	return err
}
