package main

import (
	"go/types"
	"reflect"
)

// Subnet's nine declared non-query Body fields require original row values.
// The native model has no prefix length and normalizes timestamps and nested
// route/pool structs, so this binding must not project a typed model as JSON.
func subnetBodyCollectionFields() []bodyFilterCollectionField {
	return []bodyFilterCollectionField{
		{key: "allocation_pools", member: "AllocationPools"},
		{key: "created_at", member: "CreatedAt"},
		{key: "dns_nameservers", member: "DNSNameservers"},
		{key: "host_routes", member: "HostRoutes"},
		{key: "prefixlen", aliases: []string{"prefix_length"}},
		{key: "revision_number", member: "RevisionNumber"},
		{key: "service_types", member: "ServiceTypes"},
		{key: "tenant_id", member: "TenantID"},
		{key: "updated_at", member: "UpdatedAt"},
	}
}

func subnetBodyCollectionMetadataValid(spec bodyFilterCollectionSpec) bool {
	return spec.path == "network/v2/subnets" && spec.model == "Subnet" && spec.rawRecord && reflect.DeepEqual(spec.fields, subnetBodyCollectionFields())
}

func identitySubnetBodySchema(pkg *types.Package, plan *collectionPlan) bool {
	if plan == nil || plan.modelName != "Subnet" || !plan.listQueryBuilder || plan.nameQuery != "name" || plan.statusQuery != "" {
		return false
	}
	model, ok := plan.model.(*types.Named)
	if !ok {
		return false
	}
	fields, ok := model.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	wanted := map[string]struct{ typ, tag string }{
		"ID": {"string", "id"}, "NetworkID": {"string", "network_id"}, "Name": {"string", "name"}, "Description": {"string", "description"},
		"IPVersion": {"int", "ip_version"}, "CIDR": {"string", "cidr"}, "GatewayIP": {"string", "gateway_ip"},
		"DNSNameservers": {"[]string", "dns_nameservers"}, "DNSPublishFixedIP": {"bool", "dns_publish_fixed_ip"}, "ServiceTypes": {"[]string", "service_types"},
		"AllocationPools": {"[]" + pkg.Path() + ".AllocationPool", "allocation_pools"}, "HostRoutes": {"[]" + pkg.Path() + ".HostRoute", "host_routes"},
		"EnableDHCP": {"bool", "enable_dhcp"}, "TenantID": {"string", "tenant_id"}, "ProjectID": {"string", "project_id"},
		"IPv6AddressMode": {"string", "ipv6_address_mode"}, "IPv6RAMode": {"string", "ipv6_ra_mode"}, "SubnetPoolID": {"string", "subnetpool_id"},
		"Tags": {"[]string", "tags"}, "RevisionNumber": {"int", "revision_number"}, "SegmentID": {"string", "segment_id"},
		"CreatedAt": {"time.Time", "-"}, "UpdatedAt": {"time.Time", "-"},
	}
	if fields.NumFields() != len(wanted) {
		return false
	}
	for i := 0; i < fields.NumFields(); i++ {
		field := fields.Field(i)
		want, known := wanted[field.Name()]
		if !known || field.Embedded() || types.TypeString(field.Type(), func(p *types.Package) string { return p.Path() }) != want.typ || reflect.StructTag(fields.Tag(i)).Get("json") != want.tag {
			return false
		}
	}
	for name, members := range map[string]map[string]string{
		"AllocationPool": {"Start": "start", "End": "end"},
		"HostRoute":      {"DestinationCIDR": "destination", "NextHop": "nexthop"},
	} {
		object := pkg.Scope().Lookup(name)
		if object == nil {
			return false
		}
		nested, ok := object.Type().Underlying().(*types.Struct)
		if !ok || nested.NumFields() != len(members) {
			return false
		}
		for i := 0; i < nested.NumFields(); i++ {
			member := nested.Field(i)
			tag, known := members[member.Name()]
			if !known || member.Embedded() || !types.Identical(member.Type(), types.Typ[types.String]) || reflect.StructTag(nested.Tag(i)).Get("json") != tag {
				return false
			}
		}
	}
	decoder, _, _ := types.LookupFieldOrMethod(types.NewPointer(model), true, nil, "UnmarshalJSON")
	decode, ok := decoder.(*types.Func)
	if !ok {
		return false
	}
	sig := decode.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 1 || !types.Identical(sig.Params().At(0).Type(), types.NewSlice(types.Typ[types.Uint8])) || sig.Results().Len() != 1 || !isError(sig.Results().At(0).Type()) {
		return false
	}
	pageObject := pkg.Scope().Lookup("SubnetPage")
	if pageObject == nil {
		return false
	}
	page, ok := pageObject.Type().(*types.Named)
	if !ok {
		return false
	}
	pageFields, ok := page.Underlying().(*types.Struct)
	if !ok || pageFields.NumFields() != 1 || !pageFields.Field(0).Embedded() || types.TypeString(pageFields.Field(0).Type(), func(p *types.Package) string { return p.Path() }) != upstreamModule+"/pagination.LinkedPageBase" || page.NumMethods() != 2 {
		return false
	}
	for name, result := range map[string]types.Type{"IsEmpty": types.Typ[types.Bool], "NextPageURL": types.Typ[types.String]} {
		method := extractionMethod(page, name)
		own := false
		for i := 0; i < page.NumMethods(); i++ {
			own = own || page.Method(i).Name() == name
		}
		if !own || method == nil || !types.Identical(method.Results().At(0).Type(), result) {
			return false
		}
	}
	extractor, ok := pkg.Scope().Lookup("ExtractSubnets").(*types.Func)
	if !ok {
		return false
	}
	sig = extractor.Type().(*types.Signature)
	return !sig.Variadic() && sig.Params().Len() == 1 && types.TypeString(sig.Params().At(0).Type(), func(p *types.Package) string { return p.Path() }) == upstreamModule+"/pagination.Page" && sig.Results().Len() == 2 && types.Identical(sig.Results().At(0).Type(), types.NewSlice(plan.model)) && isError(sig.Results().At(1).Type())
}

func emitSubnetBodyRecordAdapter(e *emitter, plan *collectionPlan) {
	e.use("gophercloudsdk/request")
	e.printf("},\nBodyFilterRecordValue:func(record *resource.BodyRecord[%s],key string)(json.RawMessage,error){\n", plan.modelName)
	e.printf("if record==nil{return nil,fmt.Errorf(\"%%w: nil body filter record\",resource.ErrInvalidOption)}\nswitch key{\n")
	e.printf("case \"revision_number\":return resource.BodyRecordField(record.Fields,key,resource.BodyFieldInteger)\n")
	e.printf("case \"allocation_pools\",\"created_at\",\"dns_nameservers\",\"host_routes\",\"prefixlen\",\"service_types\",\"tenant_id\",\"updated_at\":return resource.BodyRecordField(record.Fields,key,resource.BodyFieldJSON)\n")
	e.printf("default:return nil,fmt.Errorf(\"%%w: unsupported body filter field %%q\",resource.ErrInvalidOption,key)\n}},\n")
	e.printf("IterateBodyControlled:func(ctx context.Context,q url.Values,control resource.ListControl)iter.Seq2[*resource.BodyRecord[%s],error]{\n", plan.modelName)
	e.printf("options:=[]ListOption{func(config *request.Config[ListOpts])error{config.Query=make(url.Values,len(q));for key,values:=range q{config.Query[key]=append([]string(nil),values...)};return nil}}\nreturn a.listBodyWithControl(ctx,control,options...)\n},\n")
}

// Called once after newResources is closed, so the raw-row helper remains a
// private SDK method and the native public List options/signature do not change.
func emitBodyFilterList(e *emitter, plan *collectionPlan) {
	if sdkPath(e.pkg.Path()) == securityGroupSDKPath {
		if _, ok := bodyFilterCollectionContract(e.pkg, plan, 0); ok {
			emitSecurityGroupBodyFilterList(e, plan)
		}
		return
	}
	if sdkPath(e.pkg.Path()) == routerSDKPath {
		if _, ok := bodyFilterCollectionContract(e.pkg, plan, 0); ok {
			emitRouterBodyFilterList(e, plan)
		}
		return
	}
	if sdkPath(e.pkg.Path()) == networkSDKPath {
		if _, ok := bodyFilterCollectionContract(e.pkg, plan, 0); ok {
			emitNetworkBodyFilterList(e, plan)
		}
		return
	}
	if sdkPath(e.pkg.Path()) == subnetPoolSDKPath {
		if _, ok := bodyFilterCollectionContract(e.pkg, plan, 0); ok {
			emitSubnetPoolBodyFilterList(e, plan)
		}
		return
	}
	if sdkPath(e.pkg.Path()) == qosPolicySDKPath {
		if _, ok := bodyFilterCollectionContract(e.pkg, plan, 0); ok {
			emitQoSPolicyBodyFilterList(e, plan)
		}
		return
	}
	spec, ok := bodyFilterCollectionContract(e.pkg, plan, 0)
	if !ok || !spec.rawRecord {
		return
	}
	if spec.path == addressGroupSDKPath {
		emitAddressGroupBodyFilterList(e, plan)
		return
	}
	if spec.path == "keymanager/v1/orders" {
		emitKeyManagerBodyFilterList(e, plan, "orders", "ExtractOrders")
		return
	}
	if spec.path == "keymanager/v1/containers" {
		emitKeyManagerBodyFilterList(e, plan, "containers", "ExtractContainers")
		return
	}
	if spec.path == "keymanager/v1/secrets" {
		emitSecretBodyFilterList(e, plan)
		return
	}
	e.use("gophercloudsdk/request")
	e.use(upstreamModule + "/pagination")
	e.use(e.pkg.Path())
	e.printf("func(a *API)listBodyWithControl(ctx context.Context,control resource.ListControl,options ...ListOption)iter.Seq2[*resource.BodyRecord[%s],error]{\nvar opts ListOpts\ncfg,err:=request.Apply(opts,options...)\n", plan.modelName)
	e.printf("if err==nil{err=request.ValidateCapabilities(cfg,false,true,false)}\nif err!=nil{err=request.Wrap(\"List\",\"subnets\",err);return func(yield func(*resource.BodyRecord[%s],error)bool){yield(nil,err)}}\n", plan.modelName)
	e.printf("_opts:=listOptsBuilder{base:cfg.Options,config:cfg}\nreturn resource.BodyStreamWithControl(ctx,upstream.List(a.client,_opts),func(page pagination.Page)([]%s,error){values,err:=upstream.ExtractSubnets(page);return []%s(values),err},\"subnets\",control)\n}\n", plan.modelName, plan.modelName)
}
