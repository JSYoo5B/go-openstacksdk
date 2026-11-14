# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
quota = conn.key_manager.get_quota()
print(quota.secrets)
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
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    timeout := flag.Duration("timeout", time.Minute, "authentication and quota query timeout")
    flag.Parse()
    if err := run(*cloud, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud string, timeout time.Duration) error {
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
        Quotas     map[string]json.RawMessage  `json:"quotas"`
    }{
        StatusCode: quota.StatusCode,
        Header: quota.Header,
        Quotas: quota.Body,
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}
```

```go
package quotaexample

import (
    "context"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Configure(ctx context.Context, client *gophercloud.ServiceClient, projectID string) error {
    api := quotas.New(client)
    effective, err := api.Get(ctx)
    if err != nil { return err }
    fmt.Println(string(effective.Secrets))

    project, err := api.InProject(ctx, resource.ID(projectID))
    if err != nil { return err }
    configured, err := project.Get(ctx)
    if err != nil { return err }
    fmt.Println(string(configured.Orders))

    accepted, err := project.Update(ctx, quotas.UpdateOpts{
        Secrets: request.Present[int64](100),
    }, request.Custom(func(c *request.Config[quotas.UpdateOpts]) error { c.Options.Orders = request.Present(int64(0)); return nil }), request.Custom(func(c *request.Config[quotas.UpdateOpts]) error { c.Options.Containers = request.Present(int64(-1)); return nil }),
        request.Header("X-Request-ID", "quota-change"))
    if err != nil { return err }
    fmt.Println(accepted.StatusCode, accepted.Header.Get("X-Request-ID"))

    return project.Delete(ctx, request.IgnoreMissing(false))
}
```

## References

- [Python Project Records](python-project-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L397)
- [Barbican quota API](https://docs.openstack.org/api-ref/key-manager/v1/#quotas)
- [ProjectQuotaValidator](https://github.com/openstack/barbican/blob/master/barbican/common/validators.py)
- [Keymanager Quotas Test](../../../api/keymanager_quotas_test.go)
- [public get_quota](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L457)
- [Quota fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/quota.py#L16)
- [Connection Keymanager SDK Owned Test](../../../connectiontest/connection_keymanager_sdk_owned_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
