# Cloud Server Console Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--server", default="vm")
    parser.add_argument("--server-json", default="")
    parser.add_argument("--length-json", default="")
    args = parser.parse_args()
    server = json.loads(args.server_json) if args.server_json else args.server
    length = json.loads(args.length_json) if args.length_json else None
    with openstack.connect(cloud=args.cloud) as conn:
        output = conn.get_server_console(server, length=length)
        print(json.dumps(output, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
)

func run(ctx context.Context, cloud, server, suppliedServer, length string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    var options []compute.ServerConsoleOption
    if suppliedServer != "" {
        options = append(options, compute.WithServerConsoleServerJSON(json.RawMessage(suppliedServer)))
    }
    if length != "" {
        options = append(options, compute.WithServerConsoleLengthJSON(json.RawMessage(length)))
    }
    result, err := conn.GetServerConsole(ctx, server, options...)
    if err != nil { return err }
    if result == nil { return fmt.Errorf("console result is missing") }
    return json.NewEncoder(os.Stdout).Encode(struct {
        Output json.RawMessage `json:"output"`
        HasOutput bool `json:"has_output"`
        Unsupported bool `json:"unsupported"`
        StatusCode int `json:"status_code"`
    }{result.Value, result.HasOutput, result.Unsupported, result.Action.StatusCode})
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    server := flag.String("server", "vm", "literal name or ID; ignored when server-json is supplied")
    suppliedServer := flag.String("server-json", "", "owned server dictionary, for example {\"id\":\"s1\"}")
    length := flag.String("length-json", "", "optional literal JSON length; empty flag omits it")
    timeout := flag.Duration("timeout", time.Minute, "whole lookup and action deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *server, *suppliedServer, *length); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

```sh
python main.py --cloud devstack --server vm --length-json 50
go run main.go --cloud devstack --server vm --length-json 50
python main.py --cloud devstack --server-json '{"id":"s1"}' --length-json 0
go run main.go --cloud devstack --server-json '{"id":"s1"}' --length-json 0
```

```go
service, err := conn.Compute(ctx)
if err != nil { return err }
result, err := service.GetServerConsole(ctx, "vm", compute.WithServerConsoleLength(50))
```
