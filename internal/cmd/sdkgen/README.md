# sdkgen

sdkgen generates the SDK's wrappers around pinned Gophercloud v2.15.0.

Run it from the repository root:

```sh
make generate
```

`make generate` runs `generate.sh`, which:

1. requires Gophercloud v2.15.0 in `go.mod`;
2. reads Gophercloud's package and type data with `go list -deps -export -json`;
3. reads the pinned openstacksdk checkout as Python AST data, never importing it
   (set `SDKGEN_OPENSTACKSDK_SOURCE` to an existing checkout, otherwise the script
   fetches the pinned commit); the AST reader needs Python 3.14 as `python3`;
4. runs `go run ./internal/cmd/sdkgen` against both inputs.

It writes the `*_generated.go` files (API wrappers, collections, scopes, service
registries, Connection accessors and package docs where no `doc.go` exists), the
generated service READMEs and `api/gophercloud_inventory.json`. Edit the
generator, not its output; a second run must leave the tree unchanged.
