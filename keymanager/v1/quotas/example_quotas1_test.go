package quotas_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
)

func Example_string() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	timeout := flag.Duration("timeout", time.Minute, "authentication and quota query timeout")
	flag.Parse()
	if err := runQuotas1(*cloud, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runQuotas1(cloud string, timeout time.Duration) error {
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
	quota, err := service.Quotas.Get(ctx)
	if err != nil {
		return err
	}

	output := struct {
		StatusCode int                        `json:"status_code"`
		Header     http.Header                `json:"header"`
		Quotas     map[string]json.RawMessage `json:"quotas"`
	}{
		StatusCode: quota.StatusCode,
		Header:     quota.Header,
		Quotas:     quota.Body,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
