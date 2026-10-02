package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
)

func secretBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{{key: "bit_length"}, {key: "content_types"}, {key: "created_at"}, {key: "expires_at"}, {key: "id"}, {key: "payload"}, {key: "payload_content_encoding"}, {key: "payload_content_type"}, {key: "secret_id"}, {key: "secret_ref"}, {key: "status"}, {key: "updated_at"}}
}

func secretBodyCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	return spec.path == "keymanager/v1/secrets" && spec.model == "Secret" && spec.rawRecord && reflect.DeepEqual(spec.fields, secretBodyCollectionFields())
}

func secretBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "Secret" || !plan.listQueryBuilder || plan.nameQuery != "name" || plan.statusQuery != "" || plan.id != "SecretRef" || !plan.idIsURL || plan.name != "Name" || plan.status != "Status" || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !isString(plan.getIDType) {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || !rawBodyNativeDecoder(model) || !rawBodyFieldsMatch(model, "json", map[string][2]string{
		"BitLength": {"int", "bit_length"}, "Algorithm": {"string", "algorithm"}, "Expiration": {"time.Time", "-"}, "ContentTypes": {"map[string]string", "content_types"},
		"Created": {"time.Time", "-"}, "CreatorID": {"string", "creator_id"}, "Mode": {"string", "mode"}, "Name": {"string", "name"}, "SecretRef": {"string", "secret_ref"}, "SecretType": {"string", "secret_type"}, "Status": {"string", "status"}, "Updated": {"time.Time", "-"},
	}) {
		return false
	}
	if !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{
		"Offset": {"int", "offset"}, "Limit": {"int", "limit"}, "Name": {"string", "name"}, "Alg": {"string", "alg"}, "Mode": {"string", "mode"}, "Bits": {"int", "bits"}, "SecretType": {pkg.Path() + ".SecretType", "secret_type"}, "ACLOnly": {"*bool", "acl_only"},
		"CreatedQuery": {"*" + pkg.Path() + ".DateQuery", ""}, "UpdatedQuery": {"*" + pkg.Path() + ".DateQuery", ""}, "ExpirationQuery": {"*" + pkg.Path() + ".DateQuery", ""}, "Sort": {"string", "sort"},
	}) {
		return false
	}
	date := pkg.Scope().Lookup("DateQuery")
	if date == nil || !rawBodyFieldsMatch(date.Type(), "json", map[string][2]string{"Date": {"time.Time", ""}, "Filter": {pkg.Path() + ".DateFilter", ""}}) {
		return false
	}
	for _, name := range []string{"SecretType", "DateFilter"} {
		object := pkg.Scope().Lookup(name)
		if object == nil || !types.Identical(object.Type().Underlying(), types.Typ[types.String]) {
			return false
		}
	}
	return rawBodyNativePage(pkg, plan.model, "SecretPage", "ExtractSecrets")
}

var secretBodyNativeDeclarations = map[string]string{
	"ListOpts.ToSecretListQuery":        "8a6b2aa7faea2d3c150203302df3f2a9cef7729df7becde897241977ecbb7cf1",
	"List":                              "7000cbc30d3d157d38dc8d535c3d996783832023f387928bbb130c0dafbb4d83",
	"listURL":                           "a31cc211dea534f7635c0c47398b8a3e0e777da1a0464b20bed315e8496b4e37",
	"Secret.UnmarshalJSON":              "a0974f48972a2f60d8d9c7a02af04d3a92e4617298126f4ba3319eda0cf03d0d",
	"SecretPage.IsEmpty":                "31ff189ac3ded8b7c9ac1a55d0d83c2f344bffa87098763b5e34816b22bcda35",
	"SecretPage.NextPageURL":            "fd8d8df64e30aba8f83dd80ef5287992e1e622bd77fcaf2f386f7acff2eb96f2",
	"ExtractSecrets":                    "de67f034773365f1929813c46dbf3baf0733b9baea0fc6ea0eaf054cd45696b3",
	"Get":                               "aa6bf83ccd6b2db8e3c0d09e73ca2cd3d7d4e69d9ff8d604080177e38eb40232",
	"getURL":                            "a2eb1bb5c393023e42e4a2cc1f636a677e92d4322b666c41d8e39be529327d6d",
	"commonResult.Extract":              "af4577fe486fff2a3d68b3fda5da41aee5e9e4710ff124dc3c25d21da6e27de1",
	"pagination.PageResultFrom":         "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9",
	"pagination.PageResultFromParsed":   "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089",
	"pagination.LinkedPageBase.GetBody": "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284",
}

func validateSecretBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != "keymanager/v1/secrets" {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited Secret body collection: native model/list schema changed")
	}
	for name, want := range secretBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited Secret body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}

func emitSecretBodyRecordAdapter(e *emitter, plan *collectionPlan) {
	emitKeyManagerBodyRecordAdapter(e, plan, "secretBodyFilterValue")
}

func emitSecretBodyFilterList(e *emitter, plan *collectionPlan) {
	emitKeyManagerBodyFilterList(e, plan, "secrets", "ExtractSecrets")
}
