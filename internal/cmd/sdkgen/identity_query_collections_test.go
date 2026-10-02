package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func identityQueryFixtureSource(spec identityCollectionSpec) string {
	getArgs, listArgs := "id string", ",opts ListOptsBuilder"
	if spec.parents != 0 {
		getArgs = "parentID string," + getArgs
		listArgs = ",parentID string,opts ListOptsBuilder"
	}
	if spec.rawListIterator != "" {
		listArgs = ",opts ListOpts"
	}
	opts := "type ListOpts struct{Name string `q:\"name\"`}\ntype ListOptsBuilder interface{ToListQuery()(string,error)}\nfunc(ListOpts)ToListQuery()(string,error){return \"\",nil}"
	source := controlledFixtureSource(spec.model, "ID string;Name string", spec.getter, getArgs, spec.lister, listArgs, opts)
	if spec.path == "network/v2/networks" {
		source = strings.Replace(source, "type Network struct{ID string;Name string}", "type Network struct{ID string;Name string;Subnets []string `json:\"subnets\"`}", 1)
		source = strings.ReplaceAll(source, "ToListQuery", "ToNetworkListQuery")
		source = strings.ReplaceAll(source, "ModelPage", "NetworkPage")
		source = strings.Replace(source, "type NetworkPage struct{}", "type NetworkPage struct{pagination.LinkedPageBase}", 1)
		source = strings.ReplaceAll(source, "ExtractModels", "ExtractNetworks")
		source += "\nfunc(*Network)UnmarshalJSON([]byte)error{return nil}\nfunc(NetworkPage)IsEmpty()(bool,error){return false,nil}\nfunc(NetworkPage)NextPageURL()(string,error){return \"\",nil}\nfunc(NetworkPage)ResourceKey()string{return \"networks\"}\nfunc ExtractNetworksInto(r pagination.Page,v any)error{return nil}\nfunc(GetResult)ExtractInto(v any)error{return nil}\n"
	}
	if spec.path == "network/v2/subnets" {
		fields := "ID string `json:\"id\"`;NetworkID string `json:\"network_id\"`;Name string `json:\"name\"`;Description string `json:\"description\"`;IPVersion int `json:\"ip_version\"`;CIDR string `json:\"cidr\"`;GatewayIP string `json:\"gateway_ip\"`;DNSNameservers []string `json:\"dns_nameservers\"`;DNSPublishFixedIP bool `json:\"dns_publish_fixed_ip\"`;ServiceTypes []string `json:\"service_types\"`;AllocationPools []AllocationPool `json:\"allocation_pools\"`;HostRoutes []HostRoute `json:\"host_routes\"`;EnableDHCP bool `json:\"enable_dhcp\"`;TenantID string `json:\"tenant_id\"`;ProjectID string `json:\"project_id\"`;IPv6AddressMode string `json:\"ipv6_address_mode\"`;IPv6RAMode string `json:\"ipv6_ra_mode\"`;SubnetPoolID string `json:\"subnetpool_id\"`;Tags []string `json:\"tags\"`;RevisionNumber int `json:\"revision_number\"`;SegmentID string `json:\"segment_id\"`;CreatedAt time.Time `json:\"-\"`;UpdatedAt time.Time `json:\"-\"`"
		source = strings.Replace(source, "type Subnet struct{ID string;Name string}", "type AllocationPool struct{Start string `json:\"start\"`;End string `json:\"end\"`}\ntype HostRoute struct{DestinationCIDR string `json:\"destination\"`;NextHop string `json:\"nexthop\"`}\ntype Subnet struct{"+fields+"}", 1)
		source = strings.Replace(source, "import \"context\"", "import \"context\"\nimport \"time\"", 1)
		source = strings.ReplaceAll(source, "ToListQuery", "ToSubnetListQuery")
		source = strings.ReplaceAll(source, "ModelPage", "SubnetPage")
		source = strings.Replace(source, "type SubnetPage struct{}", "type SubnetPage struct{pagination.LinkedPageBase}", 1)
		source = strings.ReplaceAll(source, "ExtractModels", "ExtractSubnets")
		source += "\nfunc(*Subnet)UnmarshalJSON([]byte)error{return nil}\nfunc(SubnetPage)IsEmpty()(bool,error){return false,nil}\nfunc(SubnetPage)NextPageURL()(string,error){return \"\",nil}\n"
	}
	if spec.path == "image/v2/images" {
		source = strings.Replace(source, "type Image struct{ID string;Name string}", "type Image struct{ID string;Name string;Hidden bool `json:\"os_hidden\"`}", 1)
		source = strings.Replace(source, "type ListOpts struct{Name string `q:\"name\"`}", "type ListOpts struct{Name string `q:\"name\"`;Hidden bool `q:\"os_hidden\"`}", 1)
		source = strings.ReplaceAll(source, "ModelPage", "ImagePage")
		source = strings.Replace(source, "type ImagePage struct{}", "type ImagePage struct{serviceURL string;pagination.LinkedPageBase}", 1)
		source = strings.ReplaceAll(source, "ExtractModels", "ExtractImages")
		source += "\nfunc(ImagePage)IsEmpty()(bool,error){return false,nil}\nfunc(ImagePage)NextPageURL()(string,error){return \"\",nil}\n"
	}
	if spec.path == "compute/v2/flavors" {
		source = strings.Replace(source, "type Flavor struct{ID string;Name string}", "type Flavor struct{ID string;Name string;ExtraSpecs map[string]string `json:\"extra_specs\"`}", 1)
		source = strings.Replace(source, "type ListOpts struct{Name string `q:\"name\"`}", "type AccessType string\ntype ListOpts struct{AccessType AccessType `q:\"is_public\"`}", 1)
		source = strings.ReplaceAll(source, "ModelPage", "FlavorPage")
		source = strings.Replace(source, "type FlavorPage struct{}", "type FlavorPage struct{pagination.LinkedPageBase}", 1)
		source = strings.ReplaceAll(source, "ExtractModels", "ExtractFlavors")
		source += "\nfunc(FlavorPage)IsEmpty()(bool,error){return false,nil}\nfunc(FlavorPage)NextPageURL()(string,error){return \"\",nil}\n"
		source += "type ListExtraSpecsResult struct{Body any;Header http.Header;Err error}\nfunc(ListExtraSpecsResult)Extract()(map[string]string,error){return nil,nil}\nfunc ListExtraSpecs(ctx context.Context,client *gophercloud.ServiceClient,id string)ListExtraSpecsResult{return ListExtraSpecsResult{}}\n"
	}

	if spec.path == "network/v2/extensions/layer3/routers" || spec.path == "network/v2/extensions/security/groups" {
		page, extract := "RouterPage", "ExtractRouters"
		if spec.rawListIterator != "" {
			page, extract = "SecGroupPage", "ExtractGroups"
		}
		source = strings.ReplaceAll(source, "ModelPage", page)
		source = strings.Replace(source, "type "+page+" struct{}", "type "+page+" struct{pagination.LinkedPageBase}", 1)
		source = strings.ReplaceAll(source, "ExtractModels", extract)
		source += "\nfunc(" + page + ")IsEmpty()(bool,error){return false,nil}\nfunc(" + page + ")NextPageURL()(string,error){return \"\",nil}\n"
		if spec.rawListIterator == "" {
			source += "func ExtractRoutersInto(r pagination.Page,v any)error{return nil}\n"
		}
	}

	if spec.path == "network/v2/extensions/subnetpools" || spec.path == "network/v2/extensions/trunks" {
		page, extract := "SubnetPoolPage", "ExtractSubnetPools"
		if spec.model == "Trunk" {
			page, extract = "TrunkPage", "ExtractTrunks"
		} else {
			source = strings.Replace(source, "type SubnetPool struct{ID string;Name string}", "type SubnetPool struct{ID string `json:\"id\"`;Name string `json:\"name\"`;DefaultQuota int `json:\"default_quota\"`;TenantID string `json:\"tenant_id\"`;ProjectID string `json:\"project_id\"`;CreatedAt time.Time `json:\"-\"`;UpdatedAt time.Time `json:\"-\"`;Prefixes []string `json:\"prefixes\"`;DefaultPrefixLen int `json:\"-\"`;MinPrefixLen int `json:\"-\"`;MaxPrefixLen int `json:\"-\"`;AddressScopeID string `json:\"address_scope_id\"`;IPversion int `json:\"ip_version\"`;Shared bool `json:\"shared\"`;Description string `json:\"description\"`;IsDefault bool `json:\"is_default\"`;RevisionNumber int `json:\"revision_number\"`;Tags []string `json:\"tags\"`}", 1)
			source = strings.Replace(source, "import \"context\"", "import \"context\"\nimport \"time\"", 1)
			source = strings.Replace(source, "type ListOpts struct{Name string `q:\"name\"`}", "type ListOpts struct{ID string `q:\"id\"`;Name string `q:\"name\"`;DefaultQuota int `q:\"default_quota\"`;TenantID string `q:\"tenant_id\"`;ProjectID string `q:\"project_id\"`;DefaultPrefixLen int `q:\"default_prefixlen\"`;MinPrefixLen int `q:\"min_prefixlen\"`;MaxPrefixLen int `q:\"max_prefixlen\"`;AddressScopeID string `q:\"address_scope_id\"`;IPVersion int `q:\"ip_version\"`;Shared *bool `q:\"shared\"`;Description string `q:\"description\"`;IsDefault *bool `q:\"is_default\"`;Limit int `q:\"limit\"`;Marker string `q:\"marker\"`;SortKey string `q:\"sort_key\"`;SortDir string `q:\"sort_dir\"`;Tags string `q:\"tags\"`;TagsAny string `q:\"tags-any\"`;NotTags string `q:\"not-tags\"`;NotTagsAny string `q:\"not-tags-any\"`;RevisionNumber int `q:\"revision_number\"`}", 1)
			source = strings.ReplaceAll(source, "ToListQuery", "ToSubnetPoolListQuery")
			source = strings.Replace(source, "type GetResult struct{Body any;Header http.Header;Err error}", "type commonResult struct{gophercloud.Result}\ntype GetResult struct{commonResult}", 1)
			source = strings.Replace(source, "func(GetResult)Extract()(*SubnetPool,error)", "func(commonResult)Extract()(*SubnetPool,error)", 1)
			source = strings.Replace(source, "import \"net/http\"\n", "", 1)
			source += "\nfunc(*SubnetPool)UnmarshalJSON([]byte)error{return nil}\n"
		}
		source = strings.ReplaceAll(source, "ModelPage", page)
		source = strings.Replace(source, "type "+page+" struct{}", "type "+page+" struct{pagination.LinkedPageBase}", 1)
		source = strings.ReplaceAll(source, "ExtractModels", extract)
		source += "\nfunc(" + page + ")IsEmpty()(bool,error){return false,nil}\n"
		if spec.model != "Trunk" {
			source += "func(" + page + ")NextPageURL()(string,error){return \"\",nil}\n"
		}
	}
	if spec.path == "network/v2/extensions/qos/policies" || spec.path == "network/v2/extensions/security/addressgroups" {
		page, extract := "AddressGroupPage", "ExtractGroups"
		fields := "ID string `json:\"id\"`;Name string `json:\"name\"`;Description string `json:\"description\"`;ProjectID string `json:\"project_id\"`;Addresses []string `json:\"addresses\"`"
		if spec.model == "Policy" {
			page, extract = "PolicyPage", "ExtractPolicies"
			fields = "ID string `json:\"id\"`;Name string `json:\"name\"`;TenantID string `json:\"tenant_id\"`;ProjectID string `json:\"project_id\"`;CreatedAt time.Time `json:\"created_at\"`;UpdatedAt time.Time `json:\"updated_at\"`;IsDefault bool `json:\"is_default\"`;Description string `json:\"description\"`;Shared bool `json:\"shared\"`;RevisionNumber int `json:\"revision_number\"`;Rules []map[string]any `json:\"rules\"`;Tags []string `json:\"tags\"`"
			source = strings.Replace(source, "import \"context\"", "import \"context\"\nimport \"time\"", 1)
			source = strings.ReplaceAll(source, "ListOptsBuilder", "PolicyListOptsBuilder")
			source = strings.ReplaceAll(source, "ToListQuery", "ToPolicyListQuery")
			source = strings.Replace(source, "type ListOpts struct{Name string `q:\"name\"`}", "type ListOpts struct{ID string `q:\"id\"`;TenantID string `q:\"tenant_id\"`;ProjectID string `q:\"project_id\"`;Name string `q:\"name\"`;Description string `q:\"description\"`;IsDefault *bool `q:\"is_default\"`;Shared *bool `q:\"shared\"`;Limit int `q:\"limit\"`;Marker string `q:\"marker\"`;SortKey string `q:\"sort_key\"`;SortDir string `q:\"sort_dir\"`;Tags string `q:\"tags\"`;TagsAny string `q:\"tags-any\"`;NotTags string `q:\"not-tags\"`;NotTagsAny string `q:\"not-tags-any\"`;RevisionNumber *int `q:\"revision_number\"`}", 1)
			source += "\nfunc ExtractPolicysInto(r pagination.Page,v any)error{return nil}\n"
		} else {
			source = strings.Replace(source, "type ListOpts struct{Name string `q:\"name\"`}", "type ListOpts struct{ID string `q:\"id\"`;Name string `q:\"name\"`;Description string `q:\"description\"`;ProjectID string `q:\"project_id\"`;Addresses []string `q:\"addresses\"`;Limit int `q:\"limit\"`;Marker string `q:\"marker\"`;SortKey string `q:\"sort_key\"`;SortDir string `q:\"sort_dir\"`}", 1)
			source = strings.ReplaceAll(source, "ToListQuery", "ToAddressGroupListQuery")
		}
		source = strings.Replace(source, "type "+spec.model+" struct{ID string;Name string}", "type "+spec.model+" struct{"+fields+"}", 1)
		source = strings.ReplaceAll(source, "ModelPage", page)
		source = strings.Replace(source, "type "+page+" struct{}", "type "+page+" struct{pagination.LinkedPageBase}", 1)
		source = strings.ReplaceAll(source, "ExtractModels", extract)
		source += "\nfunc(" + page + ")IsEmpty()(bool,error){return false,nil}\nfunc(" + page + ")NextPageURL()(string,error){return \"\",nil}\n"
	}

	// Member uses GetMemberResult upstream; use its actual signature instead of
	// allowing the emitter to assume every native getter returns GetResult.
	source = strings.ReplaceAll(source, "GetResult", spec.getter+"Result")
	for name, value := range identityNativeURLConstants[spec.path] {
		source += fmt.Sprintf("\nconst %s=%q\n", name, value)
	}
	return source
}

func identityQueryFixture(t *testing.T, spec identityCollectionSpec, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	path := spec.path
	if strings.HasPrefix(path, "network/") {
		path = "networking/" + strings.TrimPrefix(path, "network/")
	}
	pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/"+path, source)
	plan := identifyNamedCollection(pkg, decls, extractorsByPage(pkg, decls), spec.getter, []string{spec.lister}, "Delete", spec.parents)
	return pkg, plan
}

func identityAuditPlans(spec identityCollectionSpec, plan *collectionPlan) (*collectionPlan, []scopePlan) {
	if spec.parents == 0 {
		return plan, nil
	}
	return nil, []scopePlan{{collection: plan}}
}

func TestIdentityGetQueryUsesAuditedNativeRoutesCodesAndResult(t *testing.T) {
	routes := map[string]string{
		"compute/v2/servers":                           `[]string{"servers", id}, q, []int{200, 203}`,
		"compute/v2/flavors":                           `[]string{"flavors", id}, q, []int{200}`,
		"network/v2/extensions/layer3/routers":         `[]string{"routers", id}, q, []int{200}`,
		"network/v2/extensions/security/groups":        `[]string{"security-groups", id}, q, []int{200}`,
		"network/v2/extensions/subnetpools":            `[]string{"subnetpools", id}, q, []int{200}`,
		"network/v2/extensions/trunks":                 `[]string{"trunks", id}, q, []int{200}`,
		"network/v2/extensions/qos/policies":           `[]string{"qos", "policies", id}, q, []int{200}`,
		"network/v2/extensions/security/addressgroups": `[]string{"address-groups", id}, q, []int{200}`,
		"blockstorage/v3/volumes":                      `[]string{"volumes", id}, q, []int{200}`,
		"network/v2/ports":                             `[]string{"ports", id}, q, []int{200}`,
		"network/v2/networks":                          `[]string{"networks", id}, q, []int{200}`,
		"network/v2/subnets":                           `[]string{"subnets", id}, q, []int{200}`,
		"identity/v3/projects":                         `[]string{"projects", id}, q, []int{200}`,
		"identity/v3/users":                            `[]string{"users", id}, q, []int{200}`,
		"identity/v3/groups":                           `[]string{"groups", id}, q, []int{200}`,
		"identity/v3/domains":                          `[]string{"domains", id}, q, []int{200}`,
		"identity/v3/roles":                            `[]string{"roles", id}, q, []int{200}`,
		"dns/v2/recordsets":                            `[]string{"zones", s.parentID, "recordsets", id}, q, []int{200}`,
		"loadbalancer/v2/pools":                        `[]string{"lbaas", "pools", s.parentID, "members", id}, q, []int{200}`,
		"image/v2/images":                              `[]string{"images", id}, q, []int{200}`,
	}
	if len(identityCollectionSpecs) != 20 || len(identityNativeDeclarations) != 20 {
		t.Fatal("identity opt-in inventory must remain explicit", len(identityCollectionSpecs), len(identityNativeDeclarations))
	}
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path, func(t *testing.T) {
			pkg, plan := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
			if !identityCollectionEnabled(pkg, plan, spec.parents) {
				t.Fatal("native-shaped fixture unexpectedly disabled", spec)
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			receiver, parents := "a", []string(nil)
			if spec.parents != 0 {
				receiver, parents = "s.api", []string{"s.parentID"}
			}
			e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
			emitCollectionAdapter(&e, plan, receiver, parents)
			e.printf("}\n")
			source, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			body := string(source)
			wants := []string{
				"GetIdentityQuery: func(ctx context.Context, id string, q url.Values)",
				"var result upstream." + spec.getter + "Result",
				"result.Header, result.Err = nativefind.Get(ctx, " + receiver + ".RawClient(), " + routes[spec.path] + ", &result.Body)",
				"return result.Extract()",
			}
			if spec.rawListIterator != "" {
				wants = append(wants, "return nativefind."+spec.rawListIterator+"(ctx, "+receiver+".RawClient(), q, control)")
			} else {
				wants = append(wants, "config.Query[key] = append([]string(nil), values...)")
			}
			for _, want := range wants {
				if !strings.Contains(body, want) {
					t.Fatalf("lost native result/routing/query contract %q:\n%s", want, body)
				}
			}
			if strings.Contains(body, "WithListQuery(") || strings.Contains(body, "WithListMembersQuery(") || strings.Contains(body, "WithListByZoneQuery(") {
				t.Fatal("audited query map collapsed to scalar options", body)
			}
			get := controlledEmittedMethod(t, source, "newResources")
			if spec.parents == 0 {
				requireControlledCalls(t, get, "a.Get(ctx, string(id))")
			} else {
				requireControlledCalls(t, get, "s.api."+spec.getter+"(ctx, s.parentID, string(id))")
			}
		})
	}
}

func TestIdentityGetQueryRejectsNativeResultAndSignatureDrift(t *testing.T) {
	spec := identityCollectionSpecs[0]
	base := identityQueryFixtureSource(spec)
	mutations := map[string]func(string) string{
		"missing body":   func(s string) string { return strings.Replace(s, "Body any;", "", 1) },
		"typed body":     func(s string) string { return strings.Replace(s, "Body any;", "Body string;", 1) },
		"missing header": func(s string) string { return strings.Replace(s, "Header http.Header;", "", 1) },
		"different header": func(s string) string {
			return strings.Replace(s, "Header http.Header;", "Header map[string]string;", 1)
		},
		"different error":    func(s string) string { return strings.Replace(s, "Err error", "Err string", 1) },
		"nonexported result": func(s string) string { return strings.ReplaceAll(s, "GetResult", "getResult") },
		"named extractor only": func(s string) string {
			return strings.Replace(s, "func(GetResult)Extract()", "func(GetResult)ExtractServer()", 1)
		},
		"new getter option": func(s string) string {
			return strings.Replace(s, "id string)GetResult", "id string,opts ListOptsBuilder)GetResult", 1)
		},
		"client/context reorder": func(s string) string {
			return strings.Replace(s, "ctx context.Context,client *gophercloud.ServiceClient,id", "client *gophercloud.ServiceClient,ctx context.Context,id", 1)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			source := mutate(base)
			// Keep imports used after deleting/changing the header field.
			source += "\nvar _ http.Header\n"
			pkg, plan := identityQueryFixture(t, spec, source)
			if identityCollectionEnabled(pkg, plan, spec.parents) {
				t.Fatal("GET query hook survived native signature/result drift", name)
			}
			if err := validateIdentityCollectionContracts(pkg, nil, plan, nil, nil); err == nil || !strings.Contains(err.Error(), "schema changed") {
				t.Fatal("audited schema drift silently removed support", err)
			}
		})
	}
	// Actual upstream result fields are promoted through embedded wrappers.
	source := strings.Replace(base, "type GetResult struct{Body any;Header http.Header;Err error}", "type NativeResult struct{Body any;Header http.Header;Err error}\ntype GetResult struct{NativeResult}", 1)
	pkg, plan := identityQueryFixture(t, spec, source)
	if !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal("valid promoted native result fields were rejected")
	}
}

func pinnedIdentityDeclarations(t *testing.T, spec identityCollectionSpec) map[string]*ast.FuncDecl {
	t.Helper()
	defaultGet := `func Get(ctx context.Context, client *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := client.Get(ctx, getURL(client, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`
	request := defaultGet
	var urls string
	switch spec.path {
	case "network/v2/networks":
		return pinnedNetworkBodyIdentityDeclarations(t)
	case "network/v2/subnets":
		return pinnedSubnetBodyIdentityDeclarations(t)
	case "image/v2/images":
		return pinnedImageIdentityDeclarations(t)
	case "compute/v2/flavors":
		return pinnedFlavorIdentityDeclarations(t)
	case "network/v2/extensions/layer3/routers", "network/v2/extensions/security/groups":
		return pinnedNeutronExtensionIdentityDeclarations(t, spec)
	case "network/v2/extensions/subnetpools", "network/v2/extensions/trunks":
		return pinnedPoolTrunkIdentityDeclarations(t, spec)
	case "network/v2/extensions/qos/policies", "network/v2/extensions/security/addressgroups":
		return pinnedQoSAddressIdentityDeclarations(t, spec)
	case "compute/v2/servers":
		request = strings.Replace(defaultGet, "&r.Body, nil", "&r.Body, &gophercloud.RequestOpts{\n\t\tOkCodes: []int{200, 203},\n\t}", 1)
		urls = `func getURL(client *gophercloud.ServiceClient, id string) string { return deleteURL(client,id) }
func deleteURL(client *gophercloud.ServiceClient, id string) string { return client.ServiceURL("servers",id) }`
	case "blockstorage/v3/volumes":
		urls = `func getURL(c *gophercloud.ServiceClient, id string) string { return deleteURL(c,id) }
func deleteURL(c *gophercloud.ServiceClient, id string) string { return c.ServiceURL("volumes",id) }`
	case "network/v2/ports":
		request = strings.ReplaceAll(defaultGet, "client", "c")
		urls = fmt.Sprintf(`func getURL(c *gophercloud.ServiceClient, id string) string { return resourceURL(c,id) }
func resourceURL(c *gophercloud.ServiceClient, id string) string { return c.ServiceURL(%q,id) }`, spec.getSegments[0])
	case "identity/v3/projects", "identity/v3/users", "identity/v3/groups", "identity/v3/domains":
		id := strings.ToLower(spec.model) + "ID"
		urls = fmt.Sprintf(`func getURL(client *gophercloud.ServiceClient, %s string) string { return client.ServiceURL(%q,%s) }`, id, spec.getSegments[0], id)
	case "identity/v3/roles":
		urls = `func getURL(client *gophercloud.ServiceClient, roleID string) string { return client.ServiceURL(rolePath,roleID) }`
	case "dns/v2/recordsets":
		request = `func Get(ctx context.Context, client *gophercloud.ServiceClient, zoneID string, rrsetID string) (r GetResult) {
	resp, err := client.Get(ctx, rrsetURL(client, zoneID, rrsetID), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`
		urls = `func rrsetURL(c *gophercloud.ServiceClient, zoneID string, rrsetID string) string { return c.ServiceURL("zones",zoneID,"recordsets",rrsetID) }`
	case "loadbalancer/v2/pools":
		request = `func GetMember(ctx context.Context, c *gophercloud.ServiceClient, poolID string, memberID string) (r GetMemberResult) {
	resp, err := c.Get(ctx, memberResourceURL(c, poolID, memberID), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}`
		urls = `func memberResourceURL(c *gophercloud.ServiceClient, poolID string, memberID string) string { return c.ServiceURL(rootPath,resourcePath,poolID,memberPath,memberID) }`
	default:
		t.Fatal("missing pinned request fixture", spec)
	}
	urls = strings.ReplaceAll(urls, " { return", " {\n\treturn")
	urls = strings.ReplaceAll(urls, " }", "\n}")
	file, err := parser.ParseFile(token.NewFileSet(), "native.go", "package fixture\n"+request+"\n"+urls, 0)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}
	return decls
}

func identityQuerySourceConstants(t *testing.T, source string) map[string]string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "constants.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	return identitySourceConstants(file)
}

func TestIdentityCollectionContractsRejectRouteCodeAndConstantDrift(t *testing.T) {
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path, func(t *testing.T) {
			base := identityQueryFixtureSource(spec)
			pkg, plan := identityQueryFixture(t, spec, base)
			global, scopes := identityAuditPlans(spec, plan)
			decls := pinnedIdentityDeclarations(t, spec)
			if err := validateIdentityCollectionContracts(pkg, decls, global, scopes, identityQuerySourceConstants(t, base)); err != nil {
				t.Fatal("pinned native declaration mismatch", err)
			}
			t.Run("different accepted GET code", func(t *testing.T) {
				fresh := pinnedIdentityDeclarations(t, spec)
				options, err := parser.ParseExpr("&gophercloud.RequestOpts{OkCodes:[]int{202}}")
				if err != nil {
					t.Fatal(err)
				}
				changed := false
				ast.Inspect(fresh[spec.getter], func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok || len(call.Args) != 4 {
						return true
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if ok && selector.Sel.Name == "Get" {
						call.Args[3] = options
						changed = true
					}
					return true
				})
				if !changed {
					t.Fatal("missing native GET options")
				}
				if err := validateIdentityCollectionContracts(pkg, fresh, global, scopes, identityQuerySourceConstants(t, base)); err == nil || !strings.Contains(err.Error(), "declaration "+spec.getter+" changed") {
					t.Fatal("native accepted-code drift ignored", err)
				}
			})
			for name := range identityNativeDeclarations[spec.path] {
				t.Run("missing "+name, func(t *testing.T) {
					fresh := pinnedIdentityDeclarations(t, spec)
					delete(fresh, name)
					if err := validateIdentityCollectionContracts(pkg, fresh, global, scopes, identityQuerySourceConstants(t, base)); err == nil || !strings.Contains(err.Error(), "declaration "+name+" changed") {
						t.Fatal("missing native declaration ignored", err)
					}
				})
				t.Run("changed "+name, func(t *testing.T) {
					fresh := pinnedIdentityDeclarations(t, spec)
					fresh[name].Body.List = append(fresh[name].Body.List, &ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("changedRouteOrCode")}})
					if err := validateIdentityCollectionContracts(pkg, fresh, global, scopes, identityQuerySourceConstants(t, base)); err == nil || !strings.Contains(err.Error(), "declaration "+name+" changed") {
						t.Fatal("native body/codes/URL drift ignored", err)
					}
				})
			}
			for name, value := range identityNativeURLConstants[spec.path] {
				t.Run("constant "+name, func(t *testing.T) {
					altered := strings.Replace(base, fmt.Sprintf("const %s=%q", name, value), fmt.Sprintf("const %s=%q", name, "changed"), 1)
					badPkg, badPlan := identityQueryFixture(t, spec, altered)
					badGlobal, badScopes := identityAuditPlans(spec, badPlan)
					if err := validateIdentityCollectionContracts(badPkg, decls, badGlobal, badScopes, identityQuerySourceConstants(t, altered)); err == nil || !strings.Contains(err.Error(), "URL constant "+name+" changed") {
						t.Fatal("native URL constant drift ignored", err)
					}
				})
			}
		})
	}
}
