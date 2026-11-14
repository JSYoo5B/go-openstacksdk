# Keypairs List Find

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
for keypair in conn.compute.keypairs(type="ssh"):
    print(keypair.to_dict())

found = conn.compute.find_keypair("workstation", ignore_missing=True)
if found is not None:
    print(found.to_dict())
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

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
)

func main() {
    name := flag.String("find", "", "keypair name to find instead of listing")
    owner := flag.String("owner-user-id", "", "optional owner")
    kind := flag.String("type", "", "optional local list filter, for example ssh")
    maximum := flag.Int("max-items", 0, "raw list row cap; zero is unlimited")
    strict := flag.Bool("strict", false, "return an error when find has no match")
    version := flag.String("microversion", "", "optional operation version; explicit empty is versionless")
    flag.Parse()
    versionSet := false
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "microversion" { versionSet = true }
    })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *name, *owner, *kind, *maximum, *strict, *version, versionSet); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, name, owner, kind string, maximum int, strict bool, version string, versionSet bool) error {
    conn, err := sdk.Connect(ctx)
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    api := service.API.KeyPairs
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    printRecord := func(record *keypairs.KeypairRecord) error {
        return encoder.Encode(map[string]any{
            "resource": record.Resource, "wire": record.Wire,
            "envelope": string(record.Envelope),
            "status_code": record.StatusCode, "header": record.Header,
        })
    }
    if name != "" {
        options := []keypairs.KeypairFindOption{
            request.Custom(func(c *request.Config[keypairs.KeypairFindOpts]) error { c.Options.UserID = owner; return nil }),
            request.IgnoreMissing(!strict),
        }
        if versionSet { options = append(options, request.Microversion(version)) }
        record, err := api.FindKeypair(ctx, name, options...)
        if record != nil {
            if printErr := printRecord(record); printErr != nil { return errors.Join(err, printErr) }
        }
        if err != nil { return fmt.Errorf("find keypair: %w", err) }
        if record == nil { fmt.Println("keypair not found") }
        return nil
    }
    options := []keypairs.KeypairListOption{
        request.Custom(func(c *request.Config[keypairs.KeypairListOpts]) error { c.Options.UserID = owner; return nil }),
        request.MaxItems(maximum),
    }
    if kind != "" { options = append(options, keypairs.WithKeypairListFilter("type", kind)) }
    if versionSet { options = append(options, request.Microversion(version)) }
    for record, err := range api.ListRecords(ctx, options...) {
        if err != nil { return fmt.Errorf("list keypairs: %w", err) }
        if err := printRecord(record); err != nil { return err }
    }
    return nil
}
```

## References

- [Nova policy](https://docs.openstack.org/nova/latest/configuration/policy.html)
- [Compute](https://docs.openstack.org/api-ref/compute/#list-keypairs)
- [Console Selection](console-selection.md)
- [Proxy keypairs/find_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L910-L978)
- [Keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/keypair.py)
- [Keypairs Console](keypairs-console.md)
- [Keypair leaf API](v2/keypairs/README.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
