# Secretstores

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
)

func ReadBackends(ctx context.Context, client *gophercloud.ServiceClient) (*secretstores.SecretStore, *secretstores.SecretStore, error) {
	api := secretstores.New(client)
	global, err := api.GetGlobalDefault(ctx)
	if err != nil {
		return nil, nil, err
	}
	preferred, err := api.GetPreferred(ctx)
	return global, preferred, err
}
```

```go
package example

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func ListNonDefaultBackends(ctx context.Context, client *gophercloud.ServiceClient) ([]*secretstores.SecretStore, error) {
	return secretstores.New(client).All(ctx,
		secretstores.WithListOptions(secretstores.ListOpts{Status: "ACTIVE", Limit: 10}),
		request.Custom(func(c *request.Config[secretstores.ListOpts]) error { v := false; c.Options.GlobalDefault = &v; return nil }),
		request.MaxItems(5),
		request.Paginated(false),
	)
}
```

```python
stores = conn.key_manager.secret_stores(
    name="backend-pattern", secret_store_id="store-id", limit=10,
)
```

```go
stores, err := service.SecretStores.All(ctx,
    secretstores.WithListFilter("name", "backend-pattern"),
    secretstores.WithListFilter("secret_store_id", "store-id"),
    secretstores.WithListOptions(secretstores.ListOpts{Limit: 10}),
)
```

```go
package main

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

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    name := flag.String("name", "", "server name query")
    id := flag.String("id", "", "local secret_store_id condition")
    timeout := flag.Duration("timeout", time.Minute, "authentication and list timeout")
    flag.Parse()
    if err := run(*cloud, *name, *id, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud, name, id string, timeout time.Duration) error {
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
            ID: store.ID,
            StatusCode: store.StatusCode,
            Header: store.Header,
            Body: store.Body,
        })
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}
```

## References

- [Keymanager Secretstores Test](../../../api/keymanager_secretstores_test.go)
- [Store Backends](https://docs.openstack.org/barbican/wallaby/api/reference/store_backends.html)
- [Policy](https://docs.openstack.org/barbican/2024.1/configuration/policy.html)
- [Policy](https://docs.openstack.org/barbican/train/configuration/policy.html)
