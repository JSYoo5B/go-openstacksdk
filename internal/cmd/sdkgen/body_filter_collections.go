package main

import (
	"fmt"
	"go/types"
	"reflect"
)

type bodyFilterCollectionField struct {
	key, member string
	aliases     []string
}

type bodyFilterCollectionSpec struct {
	path, model string
	fields      []bodyFilterCollectionField
}

// These body-only Python filters are audited against the pinned native models.
// Other body fields and native collections remain unsupported until reviewed.
var bodyFilterCollectionSpecs = []bodyFilterCollectionSpec{
	{path: "network/v2/extensions/qos/policies", model: "Policy", fields: []bodyFilterCollectionField{{key: "rules", member: "Rules"}}},
	{path: "network/v2/extensions/security/addressgroups", model: "AddressGroup", fields: []bodyFilterCollectionField{{key: "addresses", member: "Addresses"}}},
	{path: "network/v2/extensions/subnetpools", model: "SubnetPool", fields: []bodyFilterCollectionField{{key: "prefixes", member: "Prefixes"}}},
	{path: "network/v2/networks", model: "Network", fields: []bodyFilterCollectionField{{key: "subnets", member: "Subnets", aliases: []string{"subnet_ids"}}}},
}

func bodyFilterCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	if len(spec.fields) != 1 {
		return false
	}
	field := spec.fields[0]
	if spec.path != "network/v2/networks" && len(field.aliases) != 0 {
		return false
	}
	switch spec.path {
	case "network/v2/extensions/qos/policies":
		return spec.model == "Policy" && field.key == "rules" && field.member == "Rules"
	case "network/v2/extensions/security/addressgroups":
		return spec.model == "AddressGroup" && field.key == "addresses" && field.member == "Addresses"
	case "network/v2/extensions/subnetpools":
		return spec.model == "SubnetPool" && field.key == "prefixes" && field.member == "Prefixes"
	case "network/v2/networks":
		return spec.model == "Network" && field.key == "subnets" && field.member == "Subnets" && len(field.aliases) == 1 && field.aliases[0] == "subnet_ids"
	}
	return false
}

func bodyFilterCollectionNativeSchema(pkg *types.Package, plan *collectionPlan, spec bodyFilterCollectionSpec) bool {
	switch spec.path {
	case "network/v2/extensions/qos/policies", "network/v2/extensions/security/addressgroups":
		return identityQoSAddressSchema(pkg, plan)
	case "network/v2/extensions/subnetpools":
		if !identityPoolTrunkSchema(pkg, plan) {
			return false
		}
	case "network/v2/networks":
		if !identityNetworkListSchema(pkg, plan) {
			return false
		}
	default:
		return false
	}
	fields, ok := plan.model.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	selected := spec.fields[0]
	for i := 0; i < fields.NumFields(); i++ {
		field := fields.Field(i)
		if field.Name() == selected.member {
			return !field.Embedded() && types.Identical(field.Type(), types.NewSlice(types.Typ[types.String])) && reflect.StructTag(fields.Tag(i)).Get("json") == selected.key
		}
	}
	return false
}

func bodyFilterCollectionContract(pkg *types.Package, plan *collectionPlan, parents int) (bodyFilterCollectionSpec, bool) {
	if parents != 0 || !identityCollectionEnabled(pkg, plan, parents) {
		return bodyFilterCollectionSpec{}, false
	}
	for _, spec := range bodyFilterCollectionSpecs {
		if sdkPath(pkg.Path()) == spec.path && plan.modelName == spec.model && bodyFilterCollectionMetadataValid(spec) && bodyFilterCollectionNativeSchema(pkg, plan, spec) {
			return spec, true
		}
	}
	return bodyFilterCollectionSpec{}, false
}

func validateBodyFilterCollectionContracts(pkg *types.Package, plan *collectionPlan) error {
	for _, spec := range bodyFilterCollectionSpecs {
		if sdkPath(pkg.Path()) == spec.path {
			if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
				return fmt.Errorf("audited body filter collection %s.%s: native model, field, or list schema changed", spec.path, spec.model)
			}
		}
	}
	return nil
}

// A fresh canonical field list is also suitable for the resource inventory.
// Emission and inventory use the same audited native model/schema gate.
func bodyFilterCollectionFields(pkg *types.Package, plan *collectionPlan, parents int) []string {
	spec, ok := bodyFilterCollectionContract(pkg, plan, parents)
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(spec.fields))
	for _, field := range spec.fields {
		fields = append(fields, field.key)
	}
	return fields
}

func emitBodyFilterCollection(e *emitter, plan *collectionPlan, parents int) {
	spec, ok := bodyFilterCollectionContract(e.pkg, plan, parents)
	if !ok {
		return
	}
	e.use("encoding/json")
	e.printf("BodyFilterFields:map[string]string{")
	for _, field := range spec.fields {
		e.printf("%q:%q,", field.key, field.key)
		for _, alias := range field.aliases {
			e.printf("%q:%q,", alias, field.key)
		}
	}
	e.printf("},\nBodyFilterValue:func(v *%s,key string)(json.RawMessage,error){\n", plan.modelName)
	e.printf("if v==nil{return nil,fmt.Errorf(\"%%w: nil body filter resource\",resource.ErrInvalidOption)}\nswitch key{\n")
	for _, field := range spec.fields {
		e.printf("case %q:return json.Marshal(v.%s)\n", field.key, field.member)
	}
	e.printf("default:return nil,fmt.Errorf(\"%%w: unsupported body filter field %%q\",resource.ErrInvalidOption,key)\n}},\n")
}
