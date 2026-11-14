package secretstores_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
)

func Example_string() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	name := flag.String("name", "", "server name query")
	id := flag.String("id", "", "local secret_store_id condition")
	timeout := flag.Duration("timeout", time.Minute, "authentication and list timeout")
	flag.Parse()
	if err := runSecretstores3(*cloud, *name, *id, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSecretstores3(cloud, name, id string, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
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
	options := []secretstores.ListOption{
		secretstores.WithListOptions(secretstores.ListOpts{Limit: 10}),
	}
	if name != "" {
		options = append(options, secretstores.WithListFilter("name", name))
	}
	if id != "" {
		options = append(options, secretstores.WithListFilter("secret_store_id", id))
	}
	stores, err := service.SecretStores.All(ctx, options...)
	if err != nil {
		return err
	}
	type item struct {
		ID         string                     `json:"id"`
		StatusCode int                        `json:"status_code"`
		Header     http.Header                `json:"header"`
		Body       map[string]json.RawMessage `json:"body"`
	}
	output := make([]item, 0, len(stores))
	for _, store := range stores {
		output = append(output, item{
			ID:         store.ID,
			StatusCode: store.StatusCode,
			Header:     store.Header,
			Body:       store.Body,
		})
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
