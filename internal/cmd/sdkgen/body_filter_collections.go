package main

import (
	"fmt"
	"go/types"
)

type bodyFilterCollectionField struct {
	key, member string
	aliases     []string
}

type bodyFilterCollectionSpec struct {
	path, model string
	fields      []bodyFilterCollectionField
	rawRecord   bool
}

// These body-only Python filters are audited against the pinned native models.
// Other body fields and native collections remain unsupported until reviewed.
var bodyFilterCollectionSpecs = []bodyFilterCollectionSpec{
	{path: qosPolicySDKPath, model: "Policy", rawRecord: true, fields: qosPolicyBodyCollectionFields()},
	{path: "network/v2/extensions/security/addressgroups", model: "AddressGroup", rawRecord: true, fields: addressGroupBodyCollectionFields()},
	{path: subnetPoolSDKPath, model: "SubnetPool", rawRecord: true, fields: subnetPoolBodyCollectionFields()},
	{path: networkSDKPath, model: "Network", rawRecord: true, fields: networkBodyCollectionFields()},
	{path: "network/v2/subnets", model: "Subnet", rawRecord: true, fields: subnetBodyCollectionFields()},
	{path: "keymanager/v1/secrets", model: "Secret", rawRecord: true, fields: secretBodyCollectionFields()},
	{path: "keymanager/v1/containers", model: "Container", rawRecord: true, fields: containerBodyCollectionFields()},
	{path: "keymanager/v1/orders", model: "Order", rawRecord: true, fields: orderBodyCollectionFields()},
}

func bodyFilterCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	if spec.path == networkSDKPath {
		return networkBodyCollectionMetadataValid(spec)
	}
	if spec.path == subnetPoolSDKPath {
		return subnetPoolBodyCollectionMetadataValid(spec)
	}
	if spec.path == qosPolicySDKPath {
		return qosPolicyBodyCollectionMetadataValid(spec)
	}
	if spec.path == addressGroupSDKPath {
		return addressGroupBodyCollectionMetadataValid(spec)
	}
	if spec.path == "keymanager/v1/orders" {
		return orderBodyCollectionMetadataValid(spec)
	}
	if spec.path == "keymanager/v1/containers" {
		return containerBodyCollectionMetadataValid(spec)
	}
	if spec.path == "keymanager/v1/secrets" {
		return secretBodyCollectionMetadataValid(spec)
	}
	if spec.path == "network/v2/subnets" {
		return subnetBodyCollectionMetadataValid(spec)
	}
	if spec.rawRecord {
		return false
	}
	if len(spec.fields) != 1 {
		return false
	}
	field := spec.fields[0]
	if spec.path != "network/v2/networks" && len(field.aliases) != 0 {
		return false
	}
	switch spec.path {
	case "network/v2/extensions/subnetpools":
		return spec.model == "SubnetPool" && field.key == "prefixes" && field.member == "Prefixes"
	case "network/v2/networks":
		return spec.model == "Network" && field.key == "subnets" && field.member == "Subnets" && len(field.aliases) == 1 && field.aliases[0] == "subnet_ids"
	}
	return false
}

func bodyFilterCollectionNativeSchema(pkg *types.Package, plan *collectionPlan, spec bodyFilterCollectionSpec) bool {
	switch spec.path {
	case "keymanager/v1/orders":
		return orderBodyNativeSchema(pkg, plan)
	case "keymanager/v1/containers":
		return containerBodyNativeSchema(pkg, plan)
	case "keymanager/v1/secrets":
		return secretBodyNativeSchema(pkg, plan)
	case "network/v2/subnets":
		return identitySubnetBodySchema(pkg, plan)
	case "network/v2/extensions/security/addressgroups":
		return addressGroupBodyNativeSchema(pkg, plan)
	case "network/v2/extensions/qos/policies":
		return qosPolicyBodyNativeSchema(pkg, plan)
	case subnetPoolSDKPath:
		return subnetPoolBodyNativeSchema(pkg, plan)
	case networkSDKPath:
		return networkBodyNativeSchema(pkg, plan)
	default:
		return false
	}
}

func bodyFilterCollectionContract(pkg *types.Package, plan *collectionPlan, parents int) (bodyFilterCollectionSpec, bool) {
	if plan == nil || parents != 0 || (sdkPath(pkg.Path()) != "keymanager/v1/secrets" && sdkPath(pkg.Path()) != "keymanager/v1/containers" && sdkPath(pkg.Path()) != "keymanager/v1/orders" && !identityCollectionEnabled(pkg, plan, parents)) {
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
	if spec.rawRecord {
		if spec.path == networkSDKPath {
			emitNetworkBodyRecordAdapter(e, plan)
		} else if spec.path == subnetPoolSDKPath {
			emitSubnetPoolBodyRecordAdapter(e, plan)
		} else if spec.path == qosPolicySDKPath {
			emitQoSPolicyBodyRecordAdapter(e, plan)
		} else if spec.path == addressGroupSDKPath {
			emitAddressGroupBodyRecordAdapter(e, plan)
		} else if spec.path == "keymanager/v1/orders" {
			emitKeyManagerBodyRecordAdapter(e, plan, "orderBodyFilterValue")
		} else if spec.path == "keymanager/v1/containers" {
			emitKeyManagerBodyRecordAdapter(e, plan, "containerBodyFilterValue")
		} else if spec.path == "keymanager/v1/secrets" {
			emitSecretBodyRecordAdapter(e, plan)
		} else {
			emitSubnetBodyRecordAdapter(e, plan)
		}
		return
	}
	e.printf("},\nBodyFilterValue:func(v *%s,key string)(json.RawMessage,error){\n", plan.modelName)
	e.printf("if v==nil{return nil,fmt.Errorf(\"%%w: nil body filter resource\",resource.ErrInvalidOption)}\nswitch key{\n")
	for _, field := range spec.fields {
		e.printf("case %q:return json.Marshal(v.%s)\n", field.key, field.member)
	}
	e.printf("default:return nil,fmt.Errorf(\"%%w: unsupported body filter field %%q\",resource.ErrInvalidOption,key)\n}},\n")
}
