# Cloud QoS Rule Types Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import types
import openstack

def print_value(value):
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, ensure_ascii=False, indent=2))

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--filters", default="null")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    for value in conn.list_qos_rule_types(filters=json.loads(args.filters)):
        if isinstance(value, types.GeneratorType):
            for record in value:
                print_value(record)
        else:
            print_value(value)

if __name__ == "__main__":
    main()
```

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
)

func printJSON(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}

func main() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	filters := flag.String("filters", "null", "original Source cloud filters JSON")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, err := service.ListQoSRuleTypeValues(ctx, network.WithQoSRuleTypeListFilters(json.RawMessage(*filters)))
	if err != nil {
		log.Fatal(err)
	}
	if result.JSON != nil {
		var decoded any
		decoder := json.NewDecoder(bytes.NewReader(result.JSON))
		decoder.UseNumber()
		if err := decoder.Decode(&decoded); err != nil {
			log.Fatal(err)
		}
		printJSON(decoded)
		return
	}
	for _, value := range result.Values {
		if generator, ok := value.(*ruletypes.ProxyListGenerator); ok {
			for record, err := range generator.Records {
				if err != nil {
					log.Fatal(err)
				}
				printJSON(record.Resource)
			}
		} else {
			printJSON(value)
		}
	}
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [QoS Plugin](https://github.com/openstack/neutron/blob/27.0.0/neutron/services/qos/qos_plugin.py)
- [RuleTypes record reader](v2/qos-rule-types-python.md)
- [Install](../docs/install.md)
- [ Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L281-L332)
- [network proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5270)
- [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
