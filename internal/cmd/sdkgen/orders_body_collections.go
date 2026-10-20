package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
)

func orderBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{{key: "created_at"}, {key: "creator_id"}, {key: "id"}, {key: "meta"}, {key: "name"}, {key: "order_id"}, {key: "order_ref"}, {key: "secret_id"}, {key: "secret_ref"}, {key: "status"}, {key: "sub_status"}, {key: "sub_status_message"}, {key: "type"}, {key: "updated_at"}}
}

func orderBodyCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	return spec.path == "keymanager/v1/orders" && spec.model == "Order" && spec.rawRecord && reflect.DeepEqual(spec.fields, orderBodyCollectionFields())
}

func orderBodyNativeSchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "Order" || !plan.listQueryBuilder || plan.nameQuery != "" || plan.statusQuery != "" || plan.id != "OrderRef" || !plan.idIsURL || plan.name != "" || plan.status != "Status" || plan.getter == nil || plan.getter.Name() != "Get" || plan.lister == nil || plan.lister.Name() != "List" || !types.Identical(plan.getIDType, types.Typ[types.String]) {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok || !orderBodyNativeDecoder(model) || !rawBodyFieldsMatch(model, "json", map[string][2]string{
		"ContainerRef": {"string", "container_ref"}, "Created": {"time.Time", "-"}, "CreatorID": {"string", "creator_id"}, "ErrorReason": {"string", "error_reason"}, "ErrorStatusCode": {"string", "error_status_code"}, "OrderRef": {"string", "order_ref"},
		"Meta": {pkg.Path() + ".Meta", "meta"}, "SecretRef": {"string", "secret_ref"}, "Status": {"string", "status"}, "SubStatus": {"string", "sub_status"}, "SubStatusMessage": {"string", "sub_status_message"}, "Type": {"string", "type"}, "Updated": {"time.Time", "-"},
	}) || !rawBodyFieldsMatch(plan.listInput, "q", map[string][2]string{"Limit": {"int", "limit"}, "Offset": {"int", "offset"}}) {
		return false
	}
	object := pkg.Scope().Lookup("Meta")
	if object == nil {
		return false
	}
	meta, ok := object.Type().(*types.Named)
	if !ok || !orderBodyNativeDecoder(meta) || !rawBodyFieldsMatch(meta, "json", map[string][2]string{"Algorithm": {"string", "algorithm"}, "BitLength": {"int", "bit_length"}, "Expiration": {"time.Time", "-"}, "Mode": {"string", "mode"}, "Name": {"string", "name"}, "PayloadContentType": {"string", "payload_content_type"}}) {
		return false
	}
	list, ok := plan.listInput.(*types.Named)
	if !ok || list.NumMethods() != 1 || list.Method(0).Name() != "ToOrderListQuery" {
		return false
	}
	query := list.Method(0).Type().(*types.Signature)
	if !types.Identical(query.Recv().Type(), list) || query.Variadic() || query.Params().Len() != 0 || query.Results().Len() != 2 || !types.Identical(query.Results().At(0).Type(), types.Typ[types.String]) || !isError(query.Results().At(1).Type()) {
		return false
	}
	return rawBodyNativePage(pkg, plan.model, "OrderPage", "ExtractOrders")
}

func orderBodyNativeDecoder(model *types.Named) bool {
	return rawBodyNativeDecoder(model) && types.Identical(model.Method(0).Type().(*types.Signature).Recv().Type(), types.NewPointer(model))
}

var orderBodyNativeDeclarations = map[string]string{
	"ListOpts.ToOrderListQuery":         "58aff7f7689a1aaa3377a7e9e918e3fcd95e477c7ef605ecb505a67bdd6f65d2",
	"List":                              "889dc60a3ca160fcdaf846a32ce47ad0885426424f63af6dc04dae97de144301",
	"listURL":                           "15015d9b03c5f17054a1d4db99bbeaf24f99f7b7911c56caa5562953a58b014d",
	"Order.UnmarshalJSON":               "ce076a0872c8e5e9883afc8505ec1dab90f039eaa389da144ec55512861e6c70",
	"Meta.UnmarshalJSON":                "2cea2fc25483049595ed51a90686214a58386ab89d7a3ed91513d89055c73ee8",
	"OrderPage.IsEmpty":                 "0c9ad6ac4ddf66c1c33075b132cb88dc148179680bf789b0ab60031e913a96a3",
	"OrderPage.NextPageURL":             "f4d1c583c0340b1d49716d8c42a3f76b4bbdb1b558bc87ced5ba7bcd3a28061a",
	"ExtractOrders":                     "35ce66ade5a97f44bec93798b22fc9d69cddcdc69476b89dabce2a9097a36b15",
	"Get":                               "aa6bf83ccd6b2db8e3c0d09e73ca2cd3d7d4e69d9ff8d604080177e38eb40232",
	"getURL":                            "623449c17e6fc3b3c6527ff68d9b03a3b6d33ec424da0c7128c98beebfd838b3",
	"commonResult.Extract":              "664afc2ab7a1b28443e72f4af5de4e7f4ad11da0e75b97cda1a1a2472894ff87",
	"pagination.PageResultFrom":         "74ab15dabe2872e7a66623f3d6d17952e69a8abed23d6612350687126b0501e9",
	"pagination.PageResultFromParsed":   "3731e7529e8f52b6a07678931dc55380b3e04bde25aa39ef5f7420842ab49089",
	"pagination.LinkedPageBase.GetBody": "56322265078df9600b40f7136c0280ac2db3894f7142154069c2ef7d47173284",
}

func validateOrderBodyNativeDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl, plan *collectionPlan) error {
	if sdkPath(pkg.Path()) != "keymanager/v1/orders" {
		return nil
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 0); !ok {
		return fmt.Errorf("audited Order body collection: native model/list schema changed")
	}
	for name, want := range orderBodyNativeDeclarations {
		got, err := requestDeclarationHash(decls[name])
		if err != nil || got != want {
			return fmt.Errorf("audited Order body collection: pinned native declaration %s changed", name)
		}
	}
	return nil
}
